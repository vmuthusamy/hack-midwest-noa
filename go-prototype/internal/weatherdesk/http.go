package weatherdesk

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vmuthusamy/hack-midwest-noa/go-prototype/pkg/agentclient"
	pb "github.com/vmuthusamy/hack-midwest-noa/go-prototype/pkg/gen/weatherdeskv1"
	"github.com/vmuthusamy/hack-midwest-noa/go-prototype/pkg/rpcpattern"
)

type App struct {
	monitor       *incidentMonitor
	auth          *authorization
	mu            sync.Mutex
	replays       map[string]*Replay
	live          map[string]Evidence
	root          string
	agent         *agentclient.Client
	agentMu       sync.Mutex
	lastInference string
	cases         map[string]*OperationsCase
	records       map[string]DecisionRecord
	caseFetch     func(context.Context, Facility) (Evidence, error)
}

func (a *App) Close() error {
	if a.monitor != nil {
		a.monitor.cancel()
	}
	return a.agent.Close()
}
func explainRequest(e Evidence) *pb.ExplainRequest {
	f := e.Facility
	request := &pb.ExplainRequest{Mode: e.Mode, Facility: &pb.Facility{Id: f.ID, Name: f.Name, Latitude: f.Latitude, Longitude: f.Longitude, Fictional: f.Fictional}, EvidenceTimestamp: e.LastRefresh, EvidenceGaps: e.Gaps}
	for _, a := range e.Alerts {
		request.Alerts = append(request.Alerts, &pb.Alert{Id: a.ID, Source: a.Source, Event: a.Event, Description: a.Description, Instruction: a.Instruction, Area: a.Area, Issuer: a.Issuer, Issued: a.Issued, Effective: a.Effective, Expires: a.Expires, Onset: a.Onset, Ends: a.Ends, Status: a.Status, MessageType: a.MessageType, Severity: a.Severity, Certainty: a.Certainty, Urgency: a.Urgency, Geography: a.Geography, Lifecycle: a.Lifecycle, Applicability: a.Applicability, MatchMethod: a.MatchMethod})
	}
	return request
}
func respond(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}
func failure(w http.ResponseWriter, code int, message string) {
	respond(w, code, map[string]string{"error": message})
}
func decode(w http.ResponseWriter, r *http.Request, input any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(input) != nil {
		failure(w, 400, "Invalid request")
		return false
	}
	return true
}
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		origin := r.Header.Get("Origin")
		if origin != "" && origin != "http://127.0.0.1:8088" && origin != "http://localhost:8088" && origin != "http://127.0.0.1:8087" && (a.auth == nil || origin != a.auth.origin) {
			failure(w, 403, "Origin not allowed")
			return
		}
	}
	scenario := r.URL.Query().Get("facility")
	if scenario == "" {
		scenario = "barnwell"
	}
	a.mu.Lock()
	replay, exists := a.replays[scenario]
	a.mu.Unlock()
	if !exists {
		failure(w, 404, "Unknown fictional facility")
		return
	}
	if exists && a.auth != nil {
		principal, err := a.auth.verify(r, replay.Facility.ID)
		if err != nil {
			failure(w, 403, err.Error())
			return
		}
		r = withPrincipal(r, principal)
	}
	if r.URL.Path == "/api/status" && r.Method == "GET" {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		status, err := a.agent.Status(ctx)
		configured, installed := false, false
		if err == nil {
			configured = status.Configured
			installed = status.Installed
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		respond(w, 200, map[string]any{"go": true, "strandsInstalled": installed, "agentConfigured": configured, "inferenceVerified": a.lastInference != "", "lastSuccessfulInference": a.lastInference, "delivery": "simulation only", "history": "local replay snapshots", "transport": "gRPC v1", "session": requestPrincipal(r).Name})
		return
	}
	if !exists {
		failure(w, 404, "Unknown fictional facility")
		return
	}
	if r.URL.Path == "/api/incidents" {
		if r.Method != "GET" {
			failure(w, 405, "GET required")
			return
		}
		if a.monitor == nil {
			failure(w, 503, "Live monitoring is not enabled")
			return
		}
		region := r.URL.Query().Get("region")
		if region != "" && region != "ALL" && region != "OTHER" && (len(region) != 2 || !strings.Contains(" "+stateCodes+" ", " "+region+" ")) {
			failure(w, 400, "Unknown region")
			return
		}
		respond(w, 200, a.monitor.view(region, replay.Facility))
		return
	}
	if r.URL.Path == "/api/metrics" {
		a.serveMetrics(w, r)
		return
	}
	if a.serveOperations(w, r, scenario) {
		return
	}
	switch r.URL.Path {
	case "/api/replay":
		if r.Method != "GET" {
			failure(w, 405, "GET required")
			return
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		replay = a.replays[scenario]
		respond(w, 200, map[string]any{"replay": replay, "postureAssessment": AssessPosture(replay.Mode, replay.Facility, []Alert{replay.Alert}, replay.Fresh), "revision": digest(packageFor(replay))})
	case "/api/replay/action":
		if r.Method != "POST" {
			failure(w, 405, "POST required")
			return
		}
		var input struct {
			OperatingPosture string            `json:"operatingPosture"`
			Action           string            `json:"action"`
			Case             string            `json:"case"`
			Messages         map[string]string `json:"messages"`
			Revision         string            `json:"revision"`
			Binding          string            `json:"binding"`
		}
		if !decode(w, r, &input) {
			return
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		previous := a.replays[scenario]
		raw, _ := json.Marshal(previous)
		replay = &Replay{}
		if err := json.Unmarshal(raw, replay); err != nil {
			failure(w, 500, "Unable to prepare the review")
			return
		}
		if input.Revision != digest(packageFor(replay)) {
			failure(w, 409, "The review changed. Reload the current package.")
			return
		}
		var err error
		added := 0
		if input.OperatingPosture != "" {
			if input.Action != "edit" && input.Action != "approve" {
				failure(w, 400, "Posture changes require review")
				return
			}
			if err := replay.selectPosture(input.OperatingPosture); err != nil {
				failure(w, 409, err.Error())
				return
			}
		}
		switch input.Action {
		case "case":
			err = replay.setCase(input.Case)
		case "edit":
			err = replay.updateMessages(input.Messages)
		case "approve":
			if input.Messages != nil {
				err = replay.updateMessages(input.Messages)
			}
			if err == nil {
				err = replay.approveAs(requestPrincipal(r))
			}
		case "simulate":
			if replay.Approval == nil || input.Binding != replay.Approval.Binding {
				err = fmt.Errorf("exact approval binding required")
			} else {
				added, err = replay.simulateAs(requestPrincipal(r))
			}
		default:
			err = fmt.Errorf("unknown action")
		}
		if err != nil {
			failure(w, 409, err.Error())
			return
		}
		a.replays[scenario] = replay
		if err := a.saveReplay(); err != nil {
			a.replays[scenario] = previous
			failure(w, 500, "Unable to save the local replay record")
			return
		}
		respond(w, 200, map[string]any{"replay": replay, "postureAssessment": AssessPosture(replay.Mode, replay.Facility, []Alert{replay.Alert}, replay.Fresh), "revision": digest(packageFor(replay)), "added": added})
	case "/api/live":
		if r.Method != "GET" {
			failure(w, 405, "GET required")
			return
		}
		evidence, err := a.caseFetch(r.Context(), replay.Facility)
		a.mu.Lock()
		defer a.mu.Unlock()
		if err != nil {
			delete(a.live, scenario)
			failure(w, 502, "NOAA refresh failed. Current conditions are unknown. Try refreshing again.")
			return
		}
		a.live[scenario] = evidence
		respond(w, 200, evidence)
	case "/api/briefing":
		if r.Method != "POST" {
			failure(w, 405, "POST required")
			return
		}
		if !a.agentMu.TryLock() {
			failure(w, 409, "A briefing is already running")
			return
		}
		defer a.agentMu.Unlock()
		var input struct {
			Mode     string `json:"mode"`
			Revision string `json:"revision"`
		}
		if !decode(w, r, &input) {
			return
		}
		if input.Mode == "" {
			input.Mode = "LIVE"
		}
		if input.Mode != "LIVE" && input.Mode != "REPLAY" {
			failure(w, 400, "Unknown evidence mode")
			return
		}
		var e Evidence
		var request *pb.ExplainRequest
		if input.Mode == "REPLAY" {
			a.mu.Lock()
			current := a.replays[scenario]
			if input.Revision != digest(packageFor(current)) {
				a.mu.Unlock()
				failure(w, 409, "The review changed. Reload the current package.")
				return
			}
			e = Evidence{Mode: "REPLAY", Facility: current.Facility, LastRefresh: current.CapturedAt, Alerts: []Alert{current.Alert}, Gaps: []string{"Recorded NOAA evidence and synthetic recipients; not live conditions. Illustrative order is T+0 approval, T+5 warning. History timestamps record local interactions, not the NOAA issue chronology."}}
			if !current.Fresh {
				e.Gaps = append(e.Gaps, "Synthetic stale evidence; approval is blocked.")
			}
			request = explainRequest(e)
			recipients := []string{}
			for _, p := range current.Roster {
				request.Recipients = append(request.Recipients, &pb.Recipient{Id: p.ID, Name: p.Name, Group: p.Group, OriginalNoticeState: p.Original})
				if p.Original != "not targeted" {
					recipients = append(recipients, p.ID)
				}
			}
			approvedAt := ""
			if len(current.History) > 0 {
				approvedAt = current.History[0].Time
			}
			request.PriorApproval = &pb.PriorApproval{Instruction: "Report at 10:00", ApprovedAt: approvedAt, RecipientIds: recipients, CorrectionPackageApproved: current.Approval != nil}
			if current.Approval != nil {
				request.PriorApproval.CorrectionBinding = current.Approval.Binding
			}
			request.ReviewPolicy = &pb.ReviewPolicy{Version: current.Policy, Digest: policyDigest, Eligible: current.PolicyDecision.Allow, Action: "revise-arrival-instruction"}
			request.ReviewRevision = input.Revision
			request.ReplayCondition = current.Case
			a.mu.Unlock()
		} else {
			a.mu.Lock()
			var ok bool
			e, ok = a.live[scenario]
			a.mu.Unlock()
			e.Alerts = append([]Alert{}, e.Alerts...)
			last, err := time.Parse(time.RFC3339, e.LastRefresh)
			if !ok || err != nil || time.Since(last) > 5*time.Minute {
				failure(w, 409, "Refresh live NOAA evidence before requesting a briefing")
				return
			}
			// Re-evaluate validity at inference time; cache freshness cannot prolong a warning.
			now := time.Now()
			for i := range e.Alerts {
				end, err := time.Parse(time.RFC3339, e.Alerts[i].Expires)
				if err == nil && !now.Before(end) {
					e.Alerts[i].Lifecycle = "expired"
					e.Alerts[i].Applicability = "not-applicable"
				}
			}
			// The model gets facility evidence, not every unrelated alert in the nationwide feed.
			local := []Alert{}
			excludedUnknown := 0
			for _, alert := range e.Alerts {
				if alert.Applicability == "applicable" || (alert.Geography == "applicable" && alert.Applicability == "unknown") {
					local = append(local, alert)
				} else if alert.Applicability == "unknown" {
					excludedUnknown++
				}
			}
			e.Alerts = local
			if excludedUnknown > 0 {
				e.Gaps = append(e.Gaps, fmt.Sprintf("%d nationwide records have unresolved applicability and are excluded from this facility briefing. Their absence is not an all-clear.", excludedUnknown))
			}
			request = explainRequest(e)
		}
		ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
		defer cancel()
		response, err := a.agent.Explain(ctx, request)
		if err != nil {
			message := "Inference failed. The reviewed messages and notice history were not changed."
			switch status.Code(err) {
			case codes.DeadlineExceeded:
				message = "The shared inference service timed out. You can retry or review the editable fixture draft."
			case codes.FailedPrecondition:
				message = "The agent response did not satisfy the evidence contract. Review the fixture draft or try again."
			case codes.ResourceExhausted:
				message = "Another briefing is running. Wait for it to finish."
			case codes.Unavailable:
				message = "The local agent worker is unavailable. Start it with scripts/dev.sh."
			}
			failure(w, rpcpattern.HTTPStatus(err), message)
			return
		}
		if response.EvidenceTimestamp != e.LastRefresh || response.FacilityId != e.Facility.ID || response.RequestId == "" {
			failure(w, 502, "Briefing evidence identity mismatch")
			return
		}
		raw, err := agentclient.OutputJSON(response.Briefing)
		var result any
		if err != nil || json.Unmarshal(raw, &result) != nil {
			failure(w, 502, "Invalid agent output")
			return
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		if input.Mode == "REPLAY" && digest(packageFor(a.replays[scenario])) != input.Revision {
			failure(w, 409, "The review changed while the draft was generated. Review again.")
			return
		}
		a.lastInference = time.Now().UTC().Format(time.RFC3339)
		respond(w, 200, map[string]any{"briefing": result, "evidenceTimestamp": e.LastRefresh, "facilityId": e.Facility.ID, "mode": input.Mode, "reviewRevision": input.Revision})
	default:
		failure(w, 404, "Route not found")
	}
}

// New validates the policy and fixtures before accepting HTTP requests.
func New(root, agentAddress string) (*App, error) {
	if preparePolicyError != nil {
		return nil, preparePolicyError
	}
	client, err := agentclient.New(agentAddress)
	if err != nil {
		return nil, err
	}
	auth, err := newAuthorization()
	if err != nil {
		client.Close()
		return nil, err
	}
	app := &App{auth: auth, replays: map[string]*Replay{}, live: map[string]Evidence{}, root: root, agent: client, cases: map[string]*OperationsCase{}, records: map[string]DecisionRecord{}, caseFetch: fetchEvidence}
	for _, key := range []string{"barnwell", "florida", "california", "montana", "batesburg"} {
		r, err := newReplay(key, nil)
		if err != nil {
			return nil, err
		}
		app.replays[key] = r
	}
	saved, err := os.ReadFile(filepath.Join(root, ".cache", "replay-sessions.json"))
	if err == nil {
		var restored map[string]*Replay
		if err = json.Unmarshal(saved, &restored); err != nil {
			client.Close()
			return nil, fmt.Errorf("invalid local replay history")
		}
		for key, record := range restored {
			if original, ok := app.replays[key]; ok && record != nil && record.Mode == "REPLAY" && record.Facility == original.Facility {
				if record.Approval != nil && record.Approval.Binding != digest(packageFor(record)) {
					record.Approval = nil
					record.event("Approval requires review", "Policy or package changed across local restart.")
				}
				if record.OperatingPosture == "" {
					record.OperatingPosture = "delay"
					record.OperatingHours = original.OperatingHours
				}
				for i := range record.Roster {
					for _, person := range original.Roster {
						if record.Roster[i].ID == person.ID {
							record.Roster[i].Assignment = person.Assignment
							record.Roster[i].Essential = person.Essential
							record.Roster[i].TeleworkEligible = person.TeleworkEligible
							record.Roster[i].Accessibility = person.Accessibility
							record.Roster[i].Channels = person.Channels
						}
					}
				}
				if err := record.refreshPolicy(); err != nil {
					client.Close()
					return nil, err
				}
				app.replays[key] = record
			}
		}
	} else if !os.IsNotExist(err) {
		client.Close()
		return nil, err
	}
	if err := app.restoreOperations(); err != nil {
		client.Close()
		return nil, err
	}
	if os.Getenv("WEATHER_DESK_MONITOR_ENABLED") == "true" {
		monitor, err := newIncidentMonitor(root)
		if err != nil {
			client.Close()
			return nil, err
		}
		app.monitor = monitor
		app.caseFetch = func(ctx context.Context, f Facility) (Evidence, error) { return fetchEvidenceUsing(ctx, f, monitor) }
		go monitor.run()
		facilities := []Facility{}
		for _, replay := range app.replays {
			facilities = append(facilities, replay.Facility)
		}
		go monitor.refreshZones(facilities)
	}
	return app, nil
}
func (a *App) saveReplay() error {
	directory := filepath.Join(a.root, ".cache")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	data, err := json.Marshal(a.replays)
	if err != nil {
		return err
	}
	temp := filepath.Join(directory, "replay-sessions.tmp")
	if err = os.WriteFile(temp, data, 0600); err != nil {
		return err
	}
	return os.Rename(temp, filepath.Join(directory, "replay-sessions.json"))
}
