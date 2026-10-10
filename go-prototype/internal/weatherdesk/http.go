package weatherdesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"weatherdesk.local/prototype/pkg/agentclient"
	pb "weatherdesk.local/prototype/pkg/gen/weatherdeskv1"
	"weatherdesk.local/prototype/pkg/rpcpattern"
)

type App struct {
	mu      sync.Mutex
	replays map[string]*Replay
	live    map[string]Evidence
	root    string
	agent   *agentclient.Client
	agentMu sync.Mutex
}

func (a *App) Close() error { return a.agent.Close() }
func explainRequest(e Evidence) *pb.ExplainRequest {
	f := e.Facility
	request := &pb.ExplainRequest{Mode: e.Mode, Facility: &pb.Facility{Id: f.ID, Name: f.Name, Latitude: f.Latitude, Longitude: f.Longitude, Fictional: f.Fictional}, EvidenceTimestamp: e.LastRefresh, EvidenceGaps: e.Gaps}
	for _, a := range e.Alerts {
		request.Alerts = append(request.Alerts, &pb.Alert{Id: a.ID, Source: a.Source, Event: a.Event, Description: a.Description, Instruction: a.Instruction, Area: a.Area, Issuer: a.Issuer, Issued: a.Issued, Effective: a.Effective, Expires: a.Expires, Status: a.Status, MessageType: a.MessageType, Severity: a.Severity, Certainty: a.Certainty, Urgency: a.Urgency, Geography: a.Geography, Lifecycle: a.Lifecycle, Applicability: a.Applicability, MatchMethod: a.MatchMethod})
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
		if origin != "" && origin != "http://127.0.0.1:8088" && origin != "http://localhost:8088" && origin != "http://127.0.0.1:8087" {
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
	if r.URL.Path == "/api/status" && r.Method == "GET" {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		status, err := a.agent.Status(ctx)
		configured, installed := false, false
		if err == nil {
			configured = status.Configured
			installed = status.Installed
		}
		respond(w, 200, map[string]any{"go": true, "strandsInstalled": installed, "agentConfigured": configured, "inferenceVerified": false, "delivery": "simulation only", "history": "local replay snapshots", "transport": "gRPC v1", "session": "demo; no account signed in"})
		return
	}
	if !exists {
		failure(w, 404, "Unknown fictional facility")
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
		respond(w, 200, map[string]any{"replay": replay, "revision": digest(packageFor(replay))})
	case "/api/replay/action":
		if r.Method != "POST" {
			failure(w, 405, "POST required")
			return
		}
		var input struct {
			Action   string            `json:"action"`
			Case     string            `json:"case"`
			Messages map[string]string `json:"messages"`
			Revision string            `json:"revision"`
			Binding  string            `json:"binding"`
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
				err = replay.approve()
			}
		case "simulate":
			if replay.Approval == nil || input.Binding != replay.Approval.Binding {
				err = fmt.Errorf("exact approval binding required")
			} else {
				added, err = replay.simulate()
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
		respond(w, 200, map[string]any{"replay": replay, "revision": digest(packageFor(replay)), "added": added})
	case "/api/live":
		if r.Method != "GET" {
			failure(w, 405, "GET required")
			return
		}
		evidence, err := fetchEvidence(r.Context(), replay.Facility)
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
		a.mu.Lock()
		e, ok := a.live[scenario]
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
		ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
		defer cancel()
		response, err := a.agent.Explain(ctx, explainRequest(e))
		if err != nil {
			failure(w, rpcpattern.HTTPStatus(err), "Strands inference unavailable. Check the local team key and provider connection.")
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
		respond(w, 200, map[string]any{"briefing": result, "evidenceTimestamp": e.LastRefresh, "facilityId": e.Facility.ID})
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
	app := &App{replays: map[string]*Replay{}, live: map[string]Evidence{}, root: root, agent: client}
	for _, key := range []string{"barnwell", "florida", "california"} {
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
