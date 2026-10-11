package weatherdesk

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"
)

type PowerSignal struct {
	Status     string `json:"status"`
	Source     string `json:"source"`
	ObservedAt string `json:"observedAt"`
	Mode       string `json:"mode"`
}
type DeliveryEvent struct {
	ID              string `json:"id"`
	IntentKey       string `json:"intentKey"`
	Recipient       string `json:"recipient"`
	ApprovalBinding string `json:"approvalBinding"`
	Time            string `json:"time"`
	State           string `json:"state"`
	Channel         string `json:"channel"`
	Detail          string `json:"detail"`
	Simulated       bool   `json:"simulated"`
}

func facilityTimezone(f Facility) *time.Location {
	name := "America/New_York"
	if f.ID == "CA-DEMO-01" {
		name = "America/Los_Angeles"
	}
	if f.ID == "MT-DEMO-01" {
		name = "America/Denver"
	}
	zone, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return zone
}
func exampleNextUpdate(f Facility, clock string) string {
	t, err := time.Parse(time.RFC3339, clock)
	if err != nil {
		return ""
	}
	t = t.In(facilityTimezone(f))
	next := time.Date(t.Year(), t.Month(), t.Day(), 6, 0, 0, 0, t.Location())
	if !next.After(t) {
		next = next.AddDate(0, 0, 1)
	}
	return next.Format(time.RFC3339)
}
func humanUpdate(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return ""
	}
	return t.Format("Jan 2, 3:04 PM") + " facility time"
}
func applyPowerAssessment(e *Evidence) {
	e.PostureAssessment = AssessPosture(e.Mode, e.Facility, e.Alerts, true)
	if e.PowerSignal == nil || e.PowerSignal.Status != "yes" || e.PostureAssessment.RecommendedPosture == nil {
		return
	}
	posture := "closure"
	e.PostureAssessment.RecommendedPosture = &posture
	e.PostureAssessment.Rationale = "Power is reported out while an applicable NWS warning is active. Propose closure; review each staff group’s arrangements."
	e.PostureAssessment.PolicyVersion = posturePolicyVersion + "+power-v1"
	e.PostureAssessment.Gaps = append(e.PostureAssessment.Gaps, "Power availability is manager reported, not independently verified by NOAA.")
}
func (c *OperationsCase) setPower(input *PowerSignal, p Principal) error {
	if input == nil {
		return errors.New("Supply power status, source and observation time")
	}
	if input.Status != "yes" && input.Status != "no" && input.Status != "unknown" {
		return errors.New("Power status must be yes, no or unknown")
	}
	if len(strings.TrimSpace(input.Source)) < 3 || len(input.Source) > 300 {
		return errors.New("Name the source of the power report")
	}
	observed, err := time.Parse(time.RFC3339, input.ObservedAt)
	if err != nil {
		return errors.New("Power report needs a valid observation time")
	}
	clock := time.Now().UTC()
	if c.Mode == "REPLAY" {
		clock, err = time.Parse(time.RFC3339, c.Evidence.LastRefresh)
		if err != nil {
			return err
		}
	}
	if observed.After(clock.Add(time.Minute)) {
		return errors.New("Power report cannot be in the future of the selected evidence clock")
	}
	signal := *input
	signal.Source = strings.TrimSpace(signal.Source)
	signal.Mode = "MANAGER_REPORTED"
	if c.Mode == "REPLAY" {
		signal.Mode = "EXAMPLE"
	}
	c.CachedBriefing = nil
	c.Evidence.PowerSignal = &signal
	applyPowerAssessment(&c.Evidence)
	if c.Approval != nil && c.Approval.Binding != digest(operationsPackage(c)) {
		c.Approval = nil
		c.event("Facility evidence changed", "Power report changed; original decision record preserved. Review the exact revised package.")
	}
	c.event("Power report recorded", fmt.Sprintf("%s power %s, observed %s; source %s.", signal.Mode, signal.Status, signal.ObservedAt, signal.Source))
	c.revise(p)
	return nil
}
func (c *OperationsCase) deliveryFixture(p Principal) (int, error) {
	d, err := c.evaluate("simulate", p)
	if err != nil || !d.Allow {
		return 0, errors.New("Delivery example requires the exact current approval")
	}
	if len(c.Intents) == 0 {
		return 0, errors.New("Queue approved simulated notices first")
	}
	added := 0
	matching := 0
	for _, intent := range c.Intents {
		if intent.ApprovalBinding != c.Approval.Binding {
			continue
		}
		matching++
		exists := false
		for _, event := range c.DeliveryEvents {
			if event.IntentKey == intent.Key {
				exists = true
			}
		}
		if exists {
			continue
		}
		channel := "in-app"
		if len(intent.Channels) > 0 {
			channel = intent.Channels[0]
		}
		states := []string{"attempted", "delivered", "acknowledged"}
		if intent.Recipient == "S02" {
			states = []string{"attempted", "unreachable"}
			if len(intent.Channels) > 1 {
				states = append(states, "next-channel", "attempted", "delivered")
			}
		}
		for i, state := range states {
			if state == "next-channel" && len(intent.Channels) > 1 {
				channel = intent.Channels[1]
			}
			c.DeliveryEvents = append(c.DeliveryEvents, DeliveryEvent{ID: digest(struct {
				Key, State string
				Index      int
			}{intent.Key, state, i}), IntentKey: intent.Key, Recipient: intent.Recipient, ApprovalBinding: intent.ApprovalBinding, Time: time.Now().UTC().Format(time.RFC3339), State: state, Channel: channel, Detail: "Explicit delivery fixture; no network delivery or employee acknowledgment occurred.", Simulated: true})
			added++
		}
	}
	if matching == 0 {
		return 0, errors.New("Queue notices for the current exact approval first")
	}
	c.event("Delivery example played", fmt.Sprintf("%d fixture events recorded separately from queued intents. No real messages sent.", added))
	return added, nil
}

// Only validated real model output for the exact recorded case is eligible.
type CachedCaseBriefing struct {
	GeneratedAt        string         `json:"generatedAt"`
	ModelID            string         `json:"modelId"`
	Mode               string         `json:"mode"`
	FacilityID         string         `json:"facilityId"`
	EvidenceTimestamp  string         `json:"evidenceTimestamp"`
	EvidenceConditions string         `json:"evidenceConditions"`
	ProposedPosture    string         `json:"proposedPosture"`
	EffectiveAt        string         `json:"effectiveAt"`
	Briefing           map[string]any `json:"briefing"`
}

func attachCachedBriefing(c *OperationsCase, scenario string) {
	if c.Mode != "REPLAY" || c.Evidence.PowerSignal != nil {
		return
	}
	raw, err := assets.ReadFile("case-recordings/" + scenario + "-briefing-cache.json")
	if err != nil {
		return
	}
	var cache CachedCaseBriefing
	if json.Unmarshal(raw, &cache) != nil {
		return
	}
	if cache.Mode != c.Mode || cache.FacilityID != c.Facility.ID || cache.EvidenceTimestamp != c.Evidence.LastRefresh || cache.EvidenceConditions != caseConditions(c.Evidence) || cache.ProposedPosture != c.OperatingPosture || cache.EffectiveAt != c.EffectiveAt || cache.Briefing == nil {
		return
	}
	c.CachedBriefing = &cache
}

func humanEffective(value string, next []string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	if len(next) > 0 {
		if local, err := time.Parse(time.RFC3339, next[0]); err == nil {
			return t.In(local.Location()).Format("Jan 2, 3:04 PM") + " facility time"
		}
	}
	return t.UTC().Format("Jan 2, 3:04 PM UTC")
}
