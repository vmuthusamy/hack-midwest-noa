package weatherdesk

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// This version identifies a demo operating policy, not a USDA safety standard.
const posturePolicyVersion = "illustrative-weather-posture-v1"

type PostureOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type PostureAnchor struct {
	Label  string `json:"label"`
	Source string `json:"source"`
	Use    string `json:"use"`
}

type PostureTrigger struct {
	AlertID   string `json:"alertId"`
	Source    string `json:"source"`
	Issued    string `json:"issued"`
	Effective string `json:"effective"`
	Expires   string `json:"expires"`
	Event     string `json:"event"`
	Severity  string `json:"severity"`
	Certainty string `json:"certainty"`
	Urgency   string `json:"urgency"`
	Criterion string `json:"criterion"`
	Matched   bool   `json:"matched"`
}

type PostureAssessment struct {
	Mode               string           `json:"mode"`
	FacilityID         string           `json:"facilityId"`
	PolicyVersion      string           `json:"policyVersion"`
	Illustrative       bool             `json:"illustrative"`
	RecommendedPosture *string          `json:"recommendedPosture"`
	Status             string           `json:"status"`
	Options            []PostureOption  `json:"options"`
	Triggers           []PostureTrigger `json:"triggers"`
	Anchors            []PostureAnchor  `json:"anchors"`
	EvidenceAdequacy   string           `json:"evidenceAdequacy"`
	Gaps               []string         `json:"gaps"`
	WhatWouldChange    []string         `json:"whatWouldChange"`
	Rationale          string           `json:"rationale"`
}

func postureOptions() []PostureOption {
	return []PostureOption{
		{"normal", "Normal operations", "Manager-selected operations require review of current conditions and facility policy."},
		{"delay", "Delayed arrival", "Manager sets the revised reporting time and reviews affected recipients."},
		{"early-dismissal", "Early dismissal", "Requires a separately reviewed departure plan; an emergency warning never becomes an instruction to leave."},
		{"telework", "Unscheduled leave / telework", "Manager reviews eligibility, essential reporting requirements and applicable leave policy."},
		{"closure", "Full closure", "Manager reviews operating instructions for each group; closure does not direct occupants to leave."},
	}
}

const warningPostureCriterion = "Actual, active, facility-applicable Tornado Warning or Severe Thunderstorm Warning; Severe or Extreme severity; Immediate urgency; Observed or Likely certainty."

func trustedPostureSource(source string) bool {
	u, err := url.Parse(source)
	return err == nil && u.Scheme == "https" && u.Host == "api.weather.gov" && strings.HasPrefix(u.Path, "/alerts/") && u.User == nil
}

func validPostureTimes(a Alert) bool {
	issued, errIssued := time.Parse(time.RFC3339, a.Issued)
	effective, errEffective := time.Parse(time.RFC3339, a.Effective)
	expires, errExpires := time.Parse(time.RFC3339, a.Expires)
	return errIssued == nil && errEffective == nil && errExpires == nil && expires.After(effective) && expires.After(issued)
}

// AssessPosture consumes deterministic applicability/lifecycle results. The caller
// evaluates them at the live clock or explicit replay clock before each assessment.
// A suggestion grants no approval and never sends, reopens, or changes official instructions.
func AssessPosture(mode string, facility Facility, alerts []Alert, fresh bool) PostureAssessment {
	p := PostureAssessment{
		Mode: mode, FacilityID: facility.ID, PolicyVersion: posturePolicyVersion,
		Illustrative: true, Status: "insufficient-evidence", Options: postureOptions(),
		Triggers: []PostureTrigger{}, EvidenceAdequacy: "limited",
		Gaps:            []string{"Forecast, observation, access-road and facility operating dependencies are not assessed by this warning-only policy.", "Manager must review operating time, essential personnel and telework eligibility."},
		WhatWouldChange: []string{"A change in warning applicability, lifecycle, severity, urgency or certainty requires reassessment.", "Missing or stale evidence requires another successful refresh and review.", "A verified facility access or operational dependency may require a different manager-selected posture.", "Cancellation or expiry requires review; it never automatically authorizes reopening."},
		Anchors: []PostureAnchor{
			{"NWS tornado watch and warning meanings", "https://www.weather.gov/safety/tornado-ww", "Warning meaning; delayed arrival is this illustrative facility policy, not an NWS closure rule."},
			{"NWS severe thunderstorm watch and warning meanings", "https://www.weather.gov/safety/thunderstorm-ww", "Warning meaning; named severity, certainty and urgency remain the original NWS categories."},
			{"OPM operating status procedures", "https://www.opm.gov/policy-data-oversight/pay-leave/reference-materials/handbooks/dcdismissal/", "Operating posture vocabulary only; eligibility and procedures require the facility's own approved policy."},
		},
		Rationale: "Current evidence does not support a posture suggestion under the illustrative warning policy. The manager should review the evidence and facility procedures. Missing alerts, cancellation or expiration do not establish safe conditions or authorize reopening.",
	}
	if mode != "LIVE" && mode != "REPLAY" {
		p.Gaps = append(p.Gaps, "Evidence mode is unknown; live and replay evidence must remain separate.")
		return p
	}
	if !fresh || facility.ID == "" {
		p.Gaps = append(p.Gaps, "Current facility evidence is missing or stale.")
		return p
	}
	matched, unknown := false, false
	for _, a := range alerts {
		if a.Geography == "outside" || a.Applicability == "not-applicable" || a.Lifecycle == "expired" || a.Lifecycle == "cancelled" {
			continue
		}
		if a.Applicability != "applicable" || a.Geography != "applicable" || a.Lifecycle != "active" {
			unknown = true
			p.Gaps = append(p.Gaps, fmt.Sprintf("Alert %s has unresolved facility applicability or timing.", a.ID))
			continue
		}
		candidate := a.Event == "Tornado Warning" || a.Event == "Severe Thunderstorm Warning"
		if !candidate {
			p.Gaps = append(p.Gaps, fmt.Sprintf("%s requires manager review beyond this tornado/thunderstorm warning policy.", a.Event))
			continue
		}
		ok := trustedPostureSource(a.Source) && a.ID != "" && validPostureTimes(a) && a.Status == "Actual" && (a.MessageType == "Alert" || a.MessageType == "Update") && (a.Severity == "Severe" || a.Severity == "Extreme") && a.Urgency == "Immediate" && (a.Certainty == "Observed" || a.Certainty == "Likely")
		p.Triggers = append(p.Triggers, PostureTrigger{a.ID, a.Source, a.Issued, a.Effective, a.Expires, a.Event, a.Severity, a.Certainty, a.Urgency, warningPostureCriterion, ok})
		if !ok {
			unknown = true
			p.Gaps = append(p.Gaps, fmt.Sprintf("Alert %s does not satisfy the complete sourced warning criterion; no posture is inferred.", a.ID))
		} else {
			matched = true
		}
	}
	if !matched || unknown {
		if !matched {
			p.Gaps = append(p.Gaps, "No complete applicable warning matched this policy; this is not an all-clear.")
		}
		return p
	}
	delay := "delay"
	p.RecommendedPosture, p.Status = &delay, "proposed-for-review"
	context := "Live NWS evidence"
	if mode == "REPLAY" {
		context = "Recorded replay evidence"
	}
	name := facility.Name
	if name == "" {
		name = "the selected facility"
	}
	p.Rationale = fmt.Sprintf("%s contains an active, applicable tornado or severe thunderstorm warning for %s that meets the named NWS category criterion. The illustrative facility policy proposes delaying arrivals while the manager reviews reporting time and group-specific operating instructions. Official emergency instructions apply immediately and separately. This proposal does not assess individual travel, authorize departure or establish when operations can resume.", context, name)
	return p
}
