package weatherdesk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const policyVersion = "illustrative-v1"

type Recipient struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Group    string `json:"group"`
	Original string `json:"original"`
}
type Message struct {
	Group      string   `json:"group"`
	Text       string   `json:"text"`
	Recipients []string `json:"recipients"`
}
type Package struct {
	Facility     Facility          `json:"facility"`
	Action       string            `json:"action"`
	Parameters   map[string]string `json:"parameters"`
	Policy       string            `json:"policyVersion"`
	PolicyDigest string            `json:"policyDigest"`
	Conditions   string            `json:"evidenceConditions"`
	Messages     []Message         `json:"messages"`
}
type Approval struct {
	Binding string  `json:"binding"`
	Package Package `json:"package"`
	State   string  `json:"state"`
}
type Intent struct {
	Key             string `json:"key"`
	Recipient       string `json:"recipient"`
	Text            string `json:"text"`
	State           string `json:"state"`
	Simulated       bool   `json:"simulated"`
	ApprovalBinding string `json:"approvalBinding"`
}
type Event struct {
	Time   string `json:"time"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
}
type Replay struct {
	Mode             string         `json:"mode"`
	Scenario         string         `json:"scenario"`
	CapturedAt       string         `json:"capturedAt"`
	Facility         Facility       `json:"facility"`
	Alert            Alert          `json:"alert"`
	Case             string         `json:"case"`
	MaterialVersion  string         `json:"materialVersion"`
	Fresh            bool           `json:"fresh"`
	Policy           string         `json:"policyVersion"`
	Roster           []Recipient    `json:"roster"`
	Messages         []Message      `json:"messages"`
	Approval         *Approval      `json:"approval"`
	ApprovedPackages []Approval     `json:"approvedPackages"`
	Intents          []Intent       `json:"intents"`
	History          []Event        `json:"history"`
	Gate             string         `json:"gate"`
	Held             []string       `json:"heldRecipients"`
	PolicyDecision   PolicyDecision `json:"policyDecision"`
}
type Recording struct {
	CapturedAt string   `json:"capturedAt"`
	Feature    Feature  `json:"feature"`
	Facility   Facility `json:"facility"`
}

func newReplay(scenario string, previous []Event) (*Replay, error) {
	data, err := assets.ReadFile("fixtures/" + scenario + ".json")
	if err != nil {
		return nil, err
	}
	var recording Recording
	if err = json.Unmarshal(data, &recording); err != nil {
		return nil, err
	}
	captured, err := time.Parse(time.RFC3339Nano, recording.CapturedAt)
	if err != nil {
		return nil, err
	}
	r := &Replay{Mode: "REPLAY", Scenario: scenario, CapturedAt: recording.CapturedAt, Facility: recording.Facility, Alert: normalize(recording.Feature, recording.Facility, nil, captured), Case: "applicable", MaterialVersion: "recorded-v1", Fresh: true, Policy: policyVersion, Intents: []Intent{}, History: append([]Event{}, previous...)}
	r.Roster = []Recipient{{"S01", "Alex", "onsite", "delivered"}, {"S02", "Jordan", "onsite", "queued"}, {"S03", "Morgan", "arriving", "delivered"}, {"S04", "Casey", "arriving", "delivered"}, {"S05", "Taylor", "arriving", "queued"}, {"S06", "Sam", "remote", "not targeted"}}
	r.Messages = []Message{{"onsite", "The earlier “Report at 10:00” instruction is superseded. Your onsite reporting status is under review. This message does not direct you to leave the building. Official NWS instructions are displayed separately.", []string{"S01", "S02"}}, {"arriving", "The earlier “Report at 10:00” instruction is superseded. The facility manager has paused the arrival instruction pending a further operational update. Official NWS instructions are displayed separately.", []string{"S03", "S04", "S05"}}}
	r.Gate = gate(r)
	r.Held = []string{}
	if err := r.refreshPolicy(); err != nil {
		return nil, err
	}
	r.event("T+0 · Original instruction approved", "Synthetic timeline: Report at 10:00. Notice fixtures: 3 delivered, 2 queued, 1 remote excluded.")
	r.event("T+5 · Recorded NOAA evidence introduced", "Original NOAA dates are preserved. Replay uses a synthetic clock. Illustrative policy holds 2 queued notices for operational review.")
	return r, nil
}
func (r *Replay) event(label, detail string) {
	r.History = append(r.History, Event{time.Now().UTC().Format(time.RFC3339), label, detail})
}
func digest(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func conditions(r *Replay) string {
	// Refresh timestamp and unrelated statewide alerts are intentionally not material.
	return digest(struct {
		Mode    string
		Alert   Alert
		Fresh   bool
		Version string
	}{r.Mode, r.Alert, r.Fresh, r.MaterialVersion})
}
func packageFor(r *Replay) Package {
	// Copy the exact messages and recipients so later draft edits cannot mutate approval history.
	raw, _ := json.Marshal(r.Messages)
	var messages []Message
	json.Unmarshal(raw, &messages)
	return Package{r.Facility, "revise-arrival-instruction", map[string]string{"previous": "Report at 10:00", "replacement": "Arrival instruction paused pending manager update"}, r.Policy, policyDigest, conditions(r), messages}
}
func gate(r *Replay) string {
	if !r.Fresh || r.Alert.Lifecycle == "unknown" || r.Alert.Geography == "unknown" {
		return "unknown"
	}
	if r.Alert.Lifecycle != "active" {
		return "review"
	}
	if r.Alert.Applicability != "applicable" {
		return "unaffected"
	}
	if r.Alert.Event == "Tornado Warning" || r.Alert.Event == "Severe Thunderstorm Warning" {
		return "review"
	}
	return "unaffected"
}
func (r *Replay) setCase(kind string) error {
	if kind == "irrelevant" {
		r.event("Unrelated statewide update", "Approval remains bound to the same facility evidence and messages.")
		return nil
	}
	if kind != "applicable" && kind != "outside" && kind != "missing" && kind != "stale" && kind != "update" && kind != "cancel" && kind != "expired" {
		return errors.New("unknown replay case")
	}
	baseline, err := newReplay(r.Scenario, nil)
	if err != nil {
		return err
	}
	r.Alert = baseline.Alert
	r.Fresh = true
	r.MaterialVersion = "recorded-v1"
	r.Case = kind
	switch kind {
	case "outside":
		r.Alert.Geography = "outside"
		r.Alert.Applicability = "not-applicable"
		r.Alert.Geometry = nil
		r.Alert.MatchMethod = "synthetic outside-facility fixture"
	case "missing":
		r.Alert.Geography = "unknown"
		r.Alert.Applicability = "unknown"
		r.Alert.Geometry = nil
		r.Alert.MatchMethod = "synthetic missing-data fixture"
	case "stale":
		r.Fresh = false
	case "update":
		r.MaterialVersion = "synthetic-update-v2" // Keep the original official instructions intact.
	case "cancel":
		r.Alert.Lifecycle = "cancelled"
		r.Alert.Applicability = "not-applicable"
		r.MaterialVersion = "synthetic-cancel"
	case "expired":
		r.Alert.Lifecycle = "expired"
		r.Alert.Applicability = "not-applicable"
		r.MaterialVersion = "synthetic-expiry"
	}
	r.Gate = gate(r)
	if err := r.refreshPolicy(); err != nil {
		return err
	}
	if r.Approval != nil && r.Approval.Binding != digest(packageFor(r)) {
		r.Approval = nil
		r.event("Approval requires review", "Relevant replay evidence changed. Original approvals and notice states remain in history.")
	}
	r.event("Replay case: "+kind, "Synthetic test condition; it does not alter the recorded NOAA source.")
	return nil
}
func (r *Replay) updateMessages(texts map[string]string) error {
	if len(texts) != 2 {
		return errors.New("exactly two reviewed group messages required")
	}
	for _, m := range r.Messages {
		t, ok := texts[m.Group]
		if !ok || strings.TrimSpace(t) == "" || len(t) > 4000 {
			return errors.New("invalid group message")
		}
	}
	old := digest(packageFor(r))
	for i := range r.Messages {
		r.Messages[i].Text = texts[r.Messages[i].Group]
	}
	if old != digest(packageFor(r)) && r.Approval != nil {
		r.Approval = nil
		r.event("Message edit requires review", "Approval was invalidated because the exact operational text changed.")
	}
	return nil
}
func (r *Replay) approve() error {
	decision, err := evaluatePolicy(context.Background(), policyInput(r, "approve"))
	if err != nil || !decision.Allow {
		return errors.New("OPA denied approval: review current applicability, freshness, and policy conditions")
	}
	if r.Mode != "REPLAY" || !r.Fresh || r.Gate != "review" || r.Alert.Lifecycle != "active" || r.Alert.Applicability != "applicable" {
		return errors.New("fresh, applicable replay warning required; review evidence first")
	}
	p := packageFor(r)
	b := digest(p)
	if r.Approval != nil && r.Approval.Binding == b {
		return nil
	}
	r.Approval = &Approval{b, p, "approved"}
	r.ApprovedPackages = append(r.ApprovedPackages, *r.Approval)
	r.event("T+6 · Fixture manager approved correction", "Exact facility, action, parameters, recipients, messages, policy, and relevant evidence conditions are bound to approval "+b[:12]+".")
	return nil
}
func (r *Replay) simulate() (int, error) {
	decision, err := evaluatePolicy(context.Background(), policyInput(r, "simulate"))
	if err != nil || !decision.Allow {
		return 0, errors.New("OPA denied simulated sending: a current exact approval and eligible evidence are required")
	}
	if r.Approval == nil {
		return 0, errors.New("review and approval required")
	}
	if r.Approval.Binding != digest(packageFor(r)) || !r.Fresh || r.Alert.Lifecycle != "active" || r.Alert.Applicability != "applicable" {
		return 0, errors.New("approved conditions changed; review again")
	}
	count := 0
	for _, m := range r.Approval.Package.Messages {
		for _, recipient := range m.Recipients {
			key := digest(struct{ Binding, Recipient string }{r.Approval.Binding, recipient})
			found := false
			for _, intent := range r.Intents {
				if intent.Key == key {
					found = true
				}
			}
			if !found {
				r.Intents = append(r.Intents, Intent{key, recipient, m.Text, "queued", true, r.Approval.Binding})
				count++
			}
		}
	}
	r.event("T+7 · Simulated send re-check", fmt.Sprintf("%d new queued intents. No attempted, delivered, or acknowledged correction states asserted.", count))
	return count, nil
}
