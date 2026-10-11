package weatherdesk

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/vmuthusamy/hack-midwest-noa/go-prototype/pkg/agentclient"
	"google.golang.org/protobuf/types/known/structpb"
)

const operationsPolicyVersion = "demo-facility-operations-v4"

type CaseRecipient struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Group            string   `json:"group"`
	Essential        bool     `json:"essential"`
	TeleworkEligible bool     `json:"teleworkEligible"`
	Channels         []string `json:"channels"`
	Included         bool     `json:"included"`
	Reason           string   `json:"reason"`
	Text             string   `json:"text"`
}
type OperationsCase struct {
	OperatingHours   string              `json:"operatingHours,omitempty"`
	CachedBriefing   *CachedCaseBriefing `json:"cachedBriefing,omitempty"`
	OverrideReason   string              `json:"overrideReason"`
	NextUpdateAt     string              `json:"nextUpdateAt"`
	DeliveryEvents   []DeliveryEvent     `json:"deliveryEvents"`
	ID               string              `json:"id"`
	Mode             string              `json:"mode"`
	Facility         Facility            `json:"facility"`
	Evidence         Evidence            `json:"evidence"`
	Roster           []Recipient         `json:"roster"`
	Recipients       []CaseRecipient     `json:"recipients"`
	OperatingPosture string              `json:"operatingPosture"`
	Rationale        string              `json:"rationale"`
	EffectiveAt      string              `json:"effectiveAt"`
	Revision         string              `json:"revision"`
	Approval         *Approval           `json:"approval"`
	History          []Event             `json:"history"`
	Intents          []Intent            `json:"intents"`
	RecordIDs        []string            `json:"recordIds"`
	PolicyVersion    string              `json:"policyVersion"`
	PolicyDecision   PolicyDecision      `json:"policyDecision"`
}
type DecisionRecord struct {
	OverrideReason   string          `json:"overrideReason"`
	NextUpdateAt     string          `json:"nextUpdateAt"`
	DeliveryEvents   []DeliveryEvent `json:"deliveryEvents"`
	ID               string          `json:"id"`
	CaseID           string          `json:"caseId"`
	Mode             string          `json:"mode"`
	Facility         Facility        `json:"facility"`
	Evidence         Evidence        `json:"evidence"`
	SuggestedPosture *string         `json:"suggestedPosture"`
	ChosenPosture    string          `json:"chosenPosture"`
	Rationale        string          `json:"rationale"`
	EffectiveAt      string          `json:"effectiveAt"`
	PolicyVersion    string          `json:"policyVersion"`
	Approval         Approval        `json:"approval"`
	Recipients       []CaseRecipient `json:"recipients"`
	CreatedAt        string          `json:"createdAt"`
	Intents          []Intent        `json:"intents"`
	Rechecks         []Event         `json:"rechecks"`
}

// This gate authorizes an exact operating update. Official emergency instructions
// never pass through it; essential designation never grants travel authority.
var operationsPolicySource, _ = policyFiles.ReadFile("operations.rego")
var operationsRego = string(operationsPolicySource)

var operationsPolicy, operationsPolicyError = rego.New(rego.Query("data.weatherdesk.operations.decision"), rego.Module("operations.rego", operationsRego)).PrepareForEval(context.Background())
var operationsPolicyDigest = digest(operationsRego)

func newCaseID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return digest(time.Now().UnixNano())[:24]
	}
	return hex.EncodeToString(b)
}
func cloneOperations(c *OperationsCase) *OperationsCase {
	if c == nil {
		return nil
	}
	raw, _ := json.Marshal(c)
	var out OperationsCase
	json.Unmarshal(raw, &out)
	return &out
}
func (c *OperationsCase) event(label, detail string) {
	c.History = append(c.History, Event{Time: time.Now().UTC().Format(time.RFC3339), Label: label, Detail: detail})
}

// Explicit demo routing rules, not model judgments about employee safety.
func caseRecipients(roster []Recipient, posture, effectiveAt string, nextUpdate ...string) []CaseRecipient {
	out := []CaseRecipient{}
	for _, p := range roster {
		channels := append([]string{}, p.Channels...)
		if strings.Contains(strings.ToLower(p.Accessibility), "email") {
			found := false
			for _, channel := range channels {
				if channel == "email" {
					found = true
				}
			}
			if !found {
				channels = append(channels, "email")
			}
		}
		r := CaseRecipient{ID: p.ID, Name: p.Name, Group: p.Group, Essential: p.Essential, TeleworkEligible: p.TeleworkEligible, Channels: channels, Included: p.Group != "remote"}
		switch {
		case p.Group == "remote":
			r.Reason = "Already assigned remotely; no change to reporting arrangements. No notification queued."
		case p.Essential:
			r.Reason = "Essential assignment affected; receive an individual reporting review, not an instruction to travel."
		case p.Group == "onsite":
			r.Reason = "Currently onsite; needs the operating update, without a departure instruction."
		case posture == "telework" && p.TeleworkEligible:
			r.Reason = "Assigned here and telework eligible; receive the approved telework arrangement."
		default:
			r.Reason = "Assigned to arrive at this facility; the reporting arrangement changes."
		}
		if r.Included {
			heading := ""
			switch posture {
			case "closure":
				heading = "Facility closed."
			case "telework":
				heading = "Eligible nonessential staff: telework."
			case "delay":
				heading = "Arrivals delayed."
			default:
				heading = "Reporting arrangements need review."
			}
			heading += " Effective " + humanEffective(effectiveAt, nextUpdate) + "."
			instruction := "Wait for your manager’s reporting update."
			switch {
			case p.Essential:
				instruction = "Your essential assignment needs individual manager review; essential status does not authorize travel."
			case p.Group == "onsite":
				instruction = "This operating update does not direct you to leave the building."
			case posture == "telework" && p.TeleworkEligible:
				instruction = "Use your approved telework arrangement."
			}
			r.Text = heading + " " + instruction + " Follow the NWS instructions."
			if len(nextUpdate) > 0 && nextUpdate[0] != "" {
				r.Text += " Next update " + humanUpdate(nextUpdate[0]) + " (example schedule)."
			}
		}
		out = append(out, r)
	}
	return out
}
func operatingHazard(event string) bool {
	switch event {
	case "Tornado Warning", "Severe Thunderstorm Warning", "Flash Flood Warning", "Flood Warning", "Coastal Flood Warning", "Flood Watch", "Winter Storm Warning", "Winter Storm Watch":
		return true
	}
	return false
}
func localCaseAlerts(e Evidence) []Alert {
	out := []Alert{}
	for _, a := range e.Alerts {
		if a.Geography == "applicable" || (a.Geography == "unknown" && operatingHazard(a.Event)) {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func caseConditions(e Evidence) string {
	// Retrieval clock and nationwide feed order are irrelevant. Regional gauge
	// readings are frozen in the record but do not drive this warning-only policy.
	return digest(struct {
		Mode     string
		Facility Facility
		Alerts   []Alert
		Policy   string
	}{e.Mode, e.Facility, localCaseAlerts(e), posturePolicyVersion})
}
func operationsPackage(c *OperationsCase) Package {
	messages := []Message{}
	for _, p := range c.Recipients {
		if p.Included {
			messages = append(messages, Message{p.ID, p.Text, []string{p.ID}})
		}
	}
	roster, _ := json.Marshal(c.Recipients)
	return Package{Facility: c.Facility, Action: "set-operating-posture", Parameters: map[string]string{"operatingPosture": c.OperatingPosture, "rationale": c.Rationale, "effectiveAt": c.EffectiveAt, "overrideReason": c.OverrideReason, "nextUpdateAt": c.NextUpdateAt, "powerSignal": string(mustJSON(c.Evidence.PowerSignal)), "recipients": string(roster)}, Policy: c.PolicyVersion, PolicyDigest: operationsPolicyDigest, Conditions: caseConditions(c.Evidence), Messages: messages}
}
func caseEvidenceReady(c *OperationsCase, now time.Time) (bool, bool) {
	t, err := time.Parse(time.RFC3339, c.Evidence.LastRefresh)
	fresh := err == nil && c.Evidence.Facility == c.Facility && c.Evidence.Mode == c.Mode
	if c.Mode == "LIVE" {
		fresh = fresh && !t.After(now.Add(time.Minute)) && now.Sub(t) <= 5*time.Minute
	} else if c.Mode != "REPLAY" {
		fresh = false
	}
	resolved := true
	for _, a := range localCaseAlerts(c.Evidence) {
		if a.Geography == "unknown" || a.Lifecycle == "unknown" {
			resolved = false
		}
		if c.Mode == "LIVE" && a.Lifecycle == "active" {
			end, err := time.Parse(time.RFC3339, a.Expires)
			if err != nil || !now.Before(end) {
				fresh = false
			}
		}
	}
	return fresh, resolved
}
func (c *OperationsCase) evaluate(action string, p Principal) (PolicyDecision, error) {
	if operationsPolicyError != nil {
		return PolicyDecision{}, operationsPolicyError
	}
	fresh, resolved := caseEvidenceReady(c, time.Now().UTC())
	scoped := false
	for _, id := range p.Facilities {
		if id == c.Facility.ID {
			scoped = true
		}
	}
	bound := c.Approval != nil && c.Approval.Binding == digest(operationsPackage(c))
	overrideRequired := c.Evidence.PostureAssessment.RecommendedPosture != nil && c.OperatingPosture != *c.Evidence.PostureAssessment.RecommendedPosture
	overrideReviewed := len(strings.TrimSpace(c.OverrideReason)) >= 8
	result, err := operationsPolicy.Eval(context.Background(), rego.EvalInput(map[string]any{"role": p.Role, "facility_scoped": scoped, "fresh": fresh, "resolved": resolved, "action": action, "posture": c.OperatingPosture, "bound": bound, "override_required": overrideRequired, "override_reviewed": overrideReviewed}))
	if err != nil || len(result) != 1 {
		return PolicyDecision{}, errors.New("Operating policy unavailable")
	}
	raw, _ := json.Marshal(result[0].Expressions[0].Value)
	var d PolicyDecision
	err = json.Unmarshal(raw, &d)
	return d, err
}
func (c *OperationsCase) revise(p Principal) {
	c.Revision = digest(operationsPackage(c))
	c.PolicyDecision, _ = c.evaluate("approve", p)
}
func (c *OperationsCase) choose(posture, rationale, effective string, texts map[string]string, p Principal) error {
	if posture == "" {
		posture = c.OperatingPosture
	}
	switch posture {
	case "delay", "telework", "closure":
	default:
		return errors.New("Choose delayed arrival, telework, or closure for this operating review")
	}
	if rationale == "" {
		rationale = c.Rationale
	}
	if len(strings.TrimSpace(rationale)) < 8 || len(rationale) > 4000 {
		return errors.New("A short manager rationale is required")
	}
	if effective == "" {
		effective = c.EffectiveAt
	}
	if _, err := time.Parse(time.RFC3339, effective); err != nil {
		return errors.New("A valid operating effective time is required")
	}
	if posture != c.OperatingPosture || effective != c.EffectiveAt {
		c.Recipients = caseRecipients(c.Roster, posture, effective, c.NextUpdateAt)
	}
	for id, text := range texts {
		found := false
		for i := range c.Recipients {
			if c.Recipients[i].ID == id && c.Recipients[i].Included {
				found = true
				if strings.TrimSpace(text) == "" || len(text) > 4000 {
					return errors.New("Each included person needs a reviewed message")
				}
				c.Recipients[i].Text = text
			}
		}
		if !found {
			return errors.New("Message recipient is not included in this case")
		}
	}
	if suggestion := c.Evidence.PostureAssessment.RecommendedPosture; suggestion != nil && posture != *suggestion {
		if c.OverrideReason == "" && rationale != c.Rationale {
			c.OverrideReason = strings.TrimSpace(rationale)
		}
		if len(strings.TrimSpace(c.OverrideReason)) < 8 {
			return errors.New("Explain why the chosen posture differs from the recommendation")
		}
	} else {
		c.OverrideReason = ""
	}
	if c.CachedBriefing != nil && (posture != c.CachedBriefing.ProposedPosture || effective != c.CachedBriefing.EffectiveAt) {
		c.CachedBriefing = nil
	}
	c.OperatingPosture, c.Rationale, c.EffectiveAt = posture, strings.TrimSpace(rationale), effective
	if c.Approval != nil && c.Approval.Binding != digest(operationsPackage(c)) {
		c.Approval = nil
		c.event("Decision changed", "The previous approval remains in its record; review this exact revised package.")
	}
	c.revise(p)
	return nil
}
func (c *OperationsCase) approve(p Principal) (*DecisionRecord, error) {
	decision, err := c.evaluate("approve", p)
	c.PolicyDecision = decision
	if err != nil || !decision.Allow {
		return nil, errors.New("Approval blocked: " + strings.Join(decision.Reasons, " "))
	}
	pkg := operationsPackage(c)
	binding := digest(pkg)
	if c.Approval != nil && c.Approval.Binding == binding {
		return nil, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	c.Approval = &Approval{Binding: binding, Package: pkg, State: "approved", ActorID: p.Sub, ActorName: p.Name, ApprovedAt: now}
	record := DecisionRecord{OverrideReason: c.OverrideReason, NextUpdateAt: c.NextUpdateAt, DeliveryEvents: []DeliveryEvent{}, ID: newCaseID(), CaseID: c.ID, Mode: c.Mode, Facility: c.Facility, Evidence: c.Evidence, SuggestedPosture: c.Evidence.PostureAssessment.RecommendedPosture, ChosenPosture: c.OperatingPosture, Rationale: c.Rationale, EffectiveAt: c.EffectiveAt, PolicyVersion: c.PolicyVersion, Approval: *c.Approval, Recipients: append([]CaseRecipient{}, c.Recipients...), CreatedAt: now, Intents: []Intent{}, Rechecks: []Event{}}
	raw, _ := json.Marshal(record)
	json.Unmarshal(raw, &record)
	c.RecordIDs = append(c.RecordIDs, record.ID)
	c.event("Operating decision approved", "Exact messages, recipients, channels, effective time and evidence approved. Decision record saved.")
	c.revise(p)
	return &record, nil
}
func (c *OperationsCase) queue(p Principal) (int, error) {
	d, err := c.evaluate("simulate", p)
	c.PolicyDecision = d
	if err != nil || !d.Allow {
		return 0, errors.New("Queue blocked: " + strings.Join(d.Reasons, " "))
	}
	count := 0
	for _, recipient := range c.Recipients {
		if !recipient.Included {
			continue
		}
		key := digest(struct{ Binding, Recipient string }{c.Approval.Binding, recipient.ID})
		found := false
		for _, i := range c.Intents {
			if i.Key == key {
				found = true
			}
		}
		if !found {
			c.Intents = append(c.Intents, Intent{Key: key, Recipient: recipient.ID, Text: recipient.Text, Channels: append([]string{}, recipient.Channels...), State: "queued", Simulated: true, ApprovalBinding: c.Approval.Binding})
			count++
		}
	}
	c.event("Notification simulation recorded", fmt.Sprintf("Evidence and exact approval rechecked at %s. %d new simulated queued notices. No delivery or acknowledgment claimed.", c.Evidence.LastRefresh, count))
	c.History[len(c.History)-1].ApprovalBinding = c.Approval.Binding
	return count, nil
}

type operationsStore struct {
	Cases   map[string]*OperationsCase `json:"cases"`
	Records map[string]DecisionRecord  `json:"records"`
}

func (a *App) saveOperations() error {
	dir := filepath.Join(a.root, ".cache")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.Marshal(operationsStore{a.cases, a.records})
	if err != nil {
		return err
	}
	temp := filepath.Join(dir, "operations.tmp")
	if err = os.WriteFile(temp, data, 0600); err != nil {
		return err
	}
	return os.Rename(temp, filepath.Join(dir, "operations.json"))
}
func (a *App) restoreOperations() error {
	data, err := os.ReadFile(filepath.Join(a.root, ".cache", "operations.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var store operationsStore
	if err = json.Unmarshal(data, &store); err != nil {
		return fmt.Errorf("Invalid operating history: %w", err)
	}
	if store.Cases != nil {
		a.cases = store.Cases
	}
	if store.Records != nil {
		a.records = store.Records
	}
	for scenario, c := range a.cases {
		baseline, ok := a.replays[scenario]
		if !ok || c == nil || c.Facility != baseline.Facility {
			delete(a.cases, scenario)
			continue
		}
		if c.OperatingHours == "" {
			c.OperatingHours = baseline.OperatingHours
		}
		c.PolicyVersion = operationsPolicyVersion
		if c.Approval != nil && c.Approval.Binding != digest(operationsPackage(c)) {
			c.Approval = nil
			c.event("Approval requires review", "Policy or package changed since restart; original record preserved.")
		}
		c.revise(Principal{Role: "manager", Facilities: []string{c.Facility.ID}})
	}
	return nil
}
func (a *App) publishCase(scenario string, c *OperationsCase, record *DecisionRecord) error {
	previous := a.cases[scenario]
	previousRecords := make(map[string]DecisionRecord, len(a.records))
	for id, saved := range a.records {
		previousRecords[id] = saved
	}
	a.cases[scenario] = c
	if record != nil {
		a.records[record.ID] = *record
	}
	// Records keep immutable decision/evidence fields and append-only notice outcomes.
	// They remain usable when a new active case replaces this one.
	for _, id := range c.RecordIDs {
		saved, ok := a.records[id]
		if !ok {
			continue
		}
		saved.Intents = append([]Intent{}, saved.Intents...)
		for _, intent := range c.Intents {
			if intent.ApprovalBinding != saved.Approval.Binding {
				continue
			}
			seen := false
			for _, old := range saved.Intents {
				if old.Key == intent.Key {
					seen = true
				}
			}
			if !seen {
				saved.Intents = append(saved.Intents, intent)
			}
		}
		saved.Rechecks = append([]Event{}, saved.Rechecks...)
		for _, event := range c.History {
			if event.Label != "Notification simulation recorded" && event.Label != "Evidence changed before queueing" {
				continue
			}
			seen := false
			for _, old := range saved.Rechecks {
				if old == event {
					seen = true
				}
			}
			if !seen && event.ApprovalBinding == saved.Approval.Binding {
				saved.Rechecks = append(saved.Rechecks, event)
			}
		}
		for _, event := range c.DeliveryEvents {
			if event.ApprovalBinding != saved.Approval.Binding {
				continue
			}
			seen := false
			for _, old := range saved.DeliveryEvents {
				if old.ID == event.ID {
					seen = true
				}
			}
			if !seen {
				saved.DeliveryEvents = append(saved.DeliveryEvents, event)
			}
		}
		a.records[id] = saved
	}
	if err := a.saveOperations(); err != nil {
		if previous == nil {
			delete(a.cases, scenario)
		} else {
			a.cases[scenario] = previous
		}
		a.records = previousRecords
		return err
	}
	return nil
}
func (a *App) recordFor(id, facility string) (DecisionRecord, bool) {
	record, ok := a.records[id]
	if !ok || record.Facility.ID != facility {
		return DecisionRecord{}, false
	}
	return record, true
}
func (a *App) serveOperations(w http.ResponseWriter, r *http.Request, scenario string) bool {
	path := r.URL.Path
	if path != "/api/case" && path != "/api/case/action" && path != "/api/case/briefing" && path != "/api/records" && !strings.HasPrefix(path, "/api/records/") {
		return false
	}
	principal := requestPrincipal(r)
	a.mu.Lock()
	baseline := a.replays[scenario]
	a.mu.Unlock()
	if path == "/api/records" || strings.HasPrefix(path, "/api/records/") {
		if r.Method != "GET" {
			failure(w, 405, "GET required")
			return true
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		if path == "/api/records" {
			records := []DecisionRecord{}
			for id, record := range a.records {
				if record.Facility.ID == baseline.Facility.ID {
					record, _ = a.recordFor(id, baseline.Facility.ID)
					records = append(records, record)
				}
			}
			sort.Slice(records, func(i, j int) bool { return records[i].CreatedAt > records[j].CreatedAt })
			respond(w, 200, map[string]any{"records": records})
		} else {
			record, ok := a.recordFor(strings.TrimPrefix(path, "/api/records/"), baseline.Facility.ID)
			if !ok {
				failure(w, 404, "Decision record not found")
			} else {
				respond(w, 200, map[string]any{"record": record})
			}
		}
		return true
	}
	if path == "/api/case" && r.Method == "GET" {
		a.mu.Lock()
		defer a.mu.Unlock()
		respond(w, 200, map[string]any{"case": a.cases[scenario]})
		return true
	}
	if r.Method != "POST" {
		failure(w, 405, "POST required")
		return true
	}
	if principal.Role != "manager" && principal.Role != "fixture-manager" {
		failure(w, 403, "Manager access is required")
		return true
	}
	if path == "/api/case" {
		var input struct {
			Mode string `json:"mode"`
		}
		if !decode(w, r, &input) {
			return true
		}
		var evidence Evidence
		if input.Mode == "LIVE" {
			var err error
			evidence, err = a.caseFetch(r.Context(), baseline.Facility)
			if err != nil {
				failure(w, 502, "NOAA refresh failed; no new decision case was opened.")
				return true
			}
		} else if input.Mode == "REPLAY" {
			var err error
			evidence, err = recordedCaseEvidence(scenario)
			if err != nil {
				failure(w, 500, "Recording unavailable")
				return true
			}

		} else {
			failure(w, 400, "Choose LIVE or REPLAY")
			return true
		}
		evidence.Alerts = localCaseAlerts(evidence)
		sort.SliceStable(evidence.Alerts, func(i, j int) bool { return alertRank(evidence.Alerts[i]) < alertRank(evidence.Alerts[j]) })
		posture := "telework"
		if evidence.PostureAssessment.RecommendedPosture != nil {
			posture = *evidence.PostureAssessment.RecommendedPosture
		}
		effective := time.Now().UTC().Format(time.RFC3339)
		if input.Mode == "REPLAY" {
			effective = evidence.LastRefresh
		}
		c := &OperationsCase{OperatingHours: baseline.OperatingHours, ID: newCaseID(), Mode: input.Mode, Facility: evidence.Facility, Evidence: evidence, Roster: append([]Recipient{}, baseline.Roster...), OperatingPosture: posture, Rationale: evidence.PostureAssessment.Rationale, EffectiveAt: effective, PolicyVersion: operationsPolicyVersion, Intents: []Intent{}, History: []Event{}, RecordIDs: []string{}}
		attachCachedBriefing(c, scenario)
		c.NextUpdateAt = exampleNextUpdate(c.Facility, effective)
		c.DeliveryEvents = []DeliveryEvent{}
		c.Recipients = caseRecipients(c.Roster, posture, effective, c.NextUpdateAt)
		c.event("Decision case opened", "Evidence captured for review. Notifications use a synthetic roster and simulated delivery.")
		c.revise(principal)
		a.mu.Lock()
		defer a.mu.Unlock()
		if err := a.publishCase(scenario, c, nil); err != nil {
			failure(w, 500, "Unable to save decision case")
			return true
		}
		a.live[scenario] = evidence
		respond(w, 200, map[string]any{"case": c})
		return true
	}
	var input struct {
		Action           string            `json:"action"`
		OverrideReason   string            `json:"overrideReason"`
		PowerSignal      *PowerSignal      `json:"powerSignal"`
		Revision         string            `json:"revision"`
		OperatingPosture string            `json:"operatingPosture"`
		Rationale        string            `json:"rationale"`
		EffectiveAt      string            `json:"effectiveAt"`
		Messages         map[string]string `json:"messages"`
	}
	if !decode(w, r, &input) {
		return true
	}
	a.mu.Lock()
	current := a.cases[scenario]
	if current == nil {
		a.mu.Unlock()
		failure(w, 409, "Open a decision case first")
		return true
	}
	if input.Revision != current.Revision {
		latest := cloneOperations(current)
		a.mu.Unlock()
		respond(w, 409, map[string]any{"error": "This case changed in another session. Review the latest package.", "conflict": "revision", "case": latest})
		return true
	}
	c := cloneOperations(current)
	a.mu.Unlock()
	if path == "/api/case/briefing" {
		// Explain the manager's unsaved proposal without publishing a new decision.
		if input.OperatingPosture != "" {
			switch input.OperatingPosture {
			case "delay", "telework", "closure":
				c.OperatingPosture = input.OperatingPosture
			default:
				failure(w, 400, "Unknown proposed operating posture")
				return true
			}
		}
		if input.EffectiveAt != "" {
			if _, err := time.Parse(time.RFC3339, input.EffectiveAt); err != nil {
				failure(w, 400, "Invalid proposed effective time")
				return true
			}
			c.EffectiveAt = input.EffectiveAt
		}
		a.caseBriefing(w, r, c, input.Revision, scenario)
		return true
	}
	if input.Action == "simulate" && c.Mode == "LIVE" {
		evidence, err := a.caseFetch(r.Context(), c.Facility)
		if err != nil {
			failure(w, 502, "NOAA recheck failed. No simulated notices queued; refresh and review again.")
			return true
		}
		evidence.PowerSignal = c.Evidence.PowerSignal
		applyPowerAssessment(&evidence)
		evidence.Alerts = localCaseAlerts(evidence)
		sort.SliceStable(evidence.Alerts, func(i, j int) bool { return alertRank(evidence.Alerts[i]) < alertRank(evidence.Alerts[j]) })
		if caseConditions(c.Evidence) != caseConditions(evidence) {
			c.Evidence = evidence
			priorBinding := ""
			if c.Approval != nil {
				priorBinding = c.Approval.Binding
			}
			c.Approval = nil
			c.event("Evidence changed before queueing", "A relevant warning changed. Previous approval and decision record are preserved; review the updated case.")
			c.History[len(c.History)-1].ApprovalBinding = priorBinding
			c.revise(principal)
			a.mu.Lock()
			defer a.mu.Unlock()
			if a.cases[scenario].ID != c.ID || a.cases[scenario].Revision != input.Revision {
				respond(w, 409, map[string]any{"error": "Case changed during evidence recheck", "conflict": "revision", "case": cloneOperations(a.cases[scenario])})
				return true
			}
			if err := a.publishCase(scenario, c, nil); err != nil {
				failure(w, 500, "Unable to save the evidence change")
				return true
			}
			respond(w, 409, map[string]any{"error": "Evidence changed; review and approve again.", "case": c})
			return true
		}
		c.Evidence = evidence
		c.revise(principal)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cases[scenario].ID != c.ID || a.cases[scenario].Revision != input.Revision {
		respond(w, 409, map[string]any{"error": "The case changed during review", "conflict": "revision", "case": cloneOperations(a.cases[scenario])})
		return true
	}
	var record *DecisionRecord
	var err error
	added := 0
	switch input.Action {
	case "power":
		err = c.setPower(input.PowerSignal, principal)
	case "delivery-fixture":
		added, err = c.deliveryFixture(principal)
	case "choose":
		if input.OverrideReason != "" {
			if len(input.OverrideReason) > 4000 {
				failure(w, 400, "Override reason too long")
				return true
			}
			c.OverrideReason = strings.TrimSpace(input.OverrideReason)
		}
		err = c.choose(input.OperatingPosture, input.Rationale, input.EffectiveAt, input.Messages, principal)
	case "approve":
		if input.OperatingPosture != "" || input.Rationale != "" || input.EffectiveAt != "" || input.Messages != nil || input.OverrideReason != "" || input.PowerSignal != nil {
			err = errors.New("Save and review draft changes before approving")
		} else {
			record, err = c.approve(principal)
		}
	case "simulate":
		added, err = c.queue(principal)
	default:
		err = errors.New("Unknown decision action")
	}
	if err != nil {
		failure(w, 409, err.Error())
		return true
	}
	if err = a.publishCase(scenario, c, record); err != nil {
		failure(w, 500, "Unable to save decision; no changes published")
		return true
	}
	respond(w, 200, map[string]any{"case": c, "added": added})
	return true
}
func (a *App) caseBriefing(w http.ResponseWriter, r *http.Request, c *OperationsCase, revision, scenario string) {
	fresh, _ := caseEvidenceReady(c, time.Now().UTC())
	if !fresh {
		failure(w, 409, "Refresh current evidence before AI explanation")
		return
	}
	if !a.agentMu.TryLock() {
		failure(w, 409, "An explanation is already running")
		return
	}
	defer a.agentMu.Unlock()
	e := c.Evidence
	e.Alerts = localCaseAlerts(e)
	e.Gaps = append(append([]string{}, e.Gaps...), "This is an operating-review case. No earlier arrival instruction is supplied; do not invent a before/after change or a correction draft.")
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	request := explainRequest(e)
	request.EvidenceOnly = true
	var suggestion any
	if e.PostureAssessment.RecommendedPosture != nil {
		suggestion = *e.PostureAssessment.RecommendedPosture
	}
	request.CaseContext, _ = structpb.NewStruct(map[string]any{"proposed_posture": c.OperatingPosture, "effective_at": c.EffectiveAt, "policy_version": c.PolicyVersion, "policy_suggestion": suggestion, "policy_rationale": e.PostureAssessment.Rationale, "evidence_adequacy": e.PostureAssessment.EvidenceAdequacy})
	response, err := a.agent.Explain(ctx, request)
	if err != nil {
		failure(w, 502, "AI explanation unavailable. The reviewed decision was not changed.")
		return
	}
	if response.FacilityId != c.Facility.ID || response.EvidenceTimestamp != e.LastRefresh || response.RequestId == "" {
		failure(w, 502, "AI evidence identity mismatch")
		return
	}
	raw, err := agentclient.OutputJSON(response.Briefing)
	var result any
	if err != nil || json.Unmarshal(raw, &result) != nil {
		failure(w, 502, "Invalid AI explanation")
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	current := a.cases[scenario]
	if current == nil || current.ID != c.ID || current.Revision != revision {
		respond(w, 409, map[string]any{"error": "The case changed while AI was explaining it", "conflict": "revision", "case": cloneOperations(current)})
		return
	}
	a.lastInference = time.Now().UTC().Format(time.RFC3339)
	respond(w, 200, map[string]any{"briefing": result, "evidenceTimestamp": e.LastRefresh, "facilityId": c.Facility.ID, "mode": c.Mode, "reviewRevision": revision})
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
