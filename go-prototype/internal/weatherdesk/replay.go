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
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Group            string   `json:"group"`
	Original         string   `json:"original"`
	Assignment       string   `json:"assignment"`
	Essential        bool     `json:"essential"`
	TeleworkEligible bool     `json:"teleworkEligible"`
	Accessibility    string   `json:"accessibility"`
	Channels         []string `json:"channels"`
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
	Binding    string  `json:"binding"`
	Package    Package `json:"package"`
	State      string  `json:"state"`
	ActorID    string  `json:"actorId,omitempty"`
	ActorName  string  `json:"actorName,omitempty"`
	ApprovedAt string  `json:"approvedAt,omitempty"`
}
type Intent struct {
	Channels        []string `json:"channels"`
	Key             string   `json:"key"`
	Recipient       string   `json:"recipient"`
	Text            string   `json:"text"`
	State           string   `json:"state"`
	Simulated       bool     `json:"simulated"`
	ApprovalBinding string   `json:"approvalBinding"`
}
type Event struct {
	Time   string `json:"time"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
}
type Replay struct {
	OperatingPosture string         `json:"operatingPosture"`
	OperatingHours   string         `json:"operatingHours"`
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
	r.OperatingPosture = "delay"
	r.OperatingHours = "08:00–17:00 facility local time (synthetic)"
	r.Roster = []Recipient{
		{ID: "S01", Name: "Alex", Group: "onsite", Original: "delivered", Assignment: "facility", TeleworkEligible: true, Accessibility: "plain-text email", Channels: []string{"in-app", "email"}},
		{ID: "S02", Name: "Jordan", Group: "onsite", Original: "queued", Assignment: "essential facility operations", Essential: true, Accessibility: "plain-text email", Channels: []string{"in-app", "sms"}},
		{ID: "S03", Name: "Morgan", Group: "arriving", Original: "delivered", Assignment: "facility", TeleworkEligible: true, Channels: []string{"email", "sms"}},
		{ID: "S04", Name: "Casey", Group: "arriving", Original: "delivered", Assignment: "essential facility operations", Essential: true, Channels: []string{"email", "sms"}},
		{ID: "S05", Name: "Taylor", Group: "arriving", Original: "queued", Assignment: "temporary detail at this facility", TeleworkEligible: true, Accessibility: "plain-text email", Channels: []string{"email", "sms"}},
		{ID: "S06", Name: "Sam", Group: "remote", Original: "not targeted", Assignment: "remote assignment", TeleworkEligible: true, Channels: []string{"in-app", "email"}},
	}

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
	return Package{r.Facility, "revise-arrival-instruction", map[string]string{"previous": "Report at 10:00", "replacement": "Arrival instruction paused pending manager update", "operatingPosture": r.OperatingPosture, "recipientRules": recipientRules(r), "essentialStaff": "Manager must review essential assignments individually; designation never overrides official warning instructions"}, r.Policy, policyDigest, conditions(r), messages}
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
	return r.approveAs(Principal{Sub: "demo:fixture-manager", Name: "Demo facility manager", Role: "fixture-manager", Facilities: []string{r.Facility.ID}})
}
func (r *Replay) approveAs(actor Principal) error {
	decision, err := evaluatePolicy(context.Background(), policyInputFor(r, "approve", actor))
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
	r.Approval = &Approval{Binding: b, Package: p, State: "approved", ActorID: actor.Sub, ActorName: actor.Name, ApprovedAt: time.Now().UTC().Format(time.RFC3339)}
	r.ApprovedPackages = append(r.ApprovedPackages, *r.Approval)
	r.event("Manager approved correction", "Exact facility, action, parameters, recipients, messages, policy, and relevant evidence conditions are bound to approval "+b[:12]+".")
	return nil
}
func (r *Replay) simulate() (int, error) {
	return r.simulateAs(Principal{Role: "fixture-manager", Facilities: []string{r.Facility.ID}})
}
func (r *Replay) simulateAs(actor Principal) (int, error) {
	decision, err := evaluatePolicy(context.Background(), policyInputFor(r, "simulate", actor))
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
				channels := []string{}
				for _, person := range r.Roster {
					if person.ID == recipient {
						channels = append(channels, person.Channels...)
					}
				}
				r.Intents = append(r.Intents, Intent{Channels: channels, Key: key, Recipient: recipient, Text: m.Text, State: "queued", Simulated: true, ApprovalBinding: r.Approval.Binding})
				count++
			}
		}
	}
	r.event("T+7 · Simulated send re-check", fmt.Sprintf("%d new queued intents. No attempted, delivered, or acknowledged correction states asserted.", count))
	return count, nil
}

// Routing and roster decisions are part of the exact package; transport remains simulated.
func recipientRules(r *Replay) string {
	raw, _ := json.Marshal(r.Roster)
	return string(raw)
}
func (r *Replay) selectPosture(posture string) error {
	switch posture {
	case "normal", "delay", "early-dismissal", "telework", "closure":
	default:
		return errors.New("unknown operating posture")
	}
	if posture == "normal" {
		return errors.New("normal operations cannot be authorized by this warning correction policy; separate review required")
	}
	if r.OperatingPosture != posture {
		r.OperatingPosture = posture
		r.Approval = nil
		r.event("Operating posture requires review", "The proposed operating posture changed. Exact messages and essential assignments require manager review.")
	}
	return nil
}
