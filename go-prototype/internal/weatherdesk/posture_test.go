package weatherdesk

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func postureWarning() Alert {
	return Alert{ID: "urn:example:nws-warning", Source: "https://api.weather.gov/alerts/urn:example:nws-warning", Event: "Tornado Warning", Issued: "2026-10-10T21:06:00Z", Effective: "2026-10-10T21:06:00Z", Expires: "2026-10-10T21:15:00Z", Status: "Actual", MessageType: "Update", Severity: "Extreme", Certainty: "Observed", Urgency: "Immediate", Geography: "applicable", Lifecycle: "active", Applicability: "applicable"}
}

func TestPostureProposesReviewedDelayFromCompleteApplicableWarning(t *testing.T) {
	for _, mode := range []string{"LIVE", "REPLAY"} {
		for _, event := range []string{"Tornado Warning", "Severe Thunderstorm Warning"} {
			t.Run(mode+"/"+event, func(t *testing.T) {
				a := postureWarning()
				a.Event, a.Certainty = event, "Likely"
				p := AssessPosture(mode, Facility{ID: "SC-DEMO-01", Name: "Barnwell Field Office"}, []Alert{a}, true)
				if p.RecommendedPosture == nil || *p.RecommendedPosture != "delay" || p.Status != "proposed-for-review" || !p.Illustrative || len(p.Options) != 5 || len(p.Triggers) != 1 || !p.Triggers[0].Matched {
					t.Fatalf("unexpected assessment: %+v", p)
				}
				trigger := p.Triggers[0]
				if trigger.Source != a.Source || trigger.Issued != a.Issued || trigger.Expires != a.Expires || trigger.Severity != a.Severity || p.PolicyVersion == "" || len(p.Anchors) != 3 || p.EvidenceAdequacy != "limited" {
					t.Fatalf("reasoning lost source criteria or uncertainty: %+v", p)
				}
				if !strings.Contains(p.Rationale, "Official emergency instructions apply immediately") || (mode == "REPLAY" && !strings.Contains(p.Rationale, "Recorded replay")) || len(p.WhatWouldChange) == 0 {
					t.Fatalf("rationale lost emergency/replay boundaries: %s", p.Rationale)
				}
			})
		}
	}
}

func TestPostureMissingOrIncompatibleEvidenceDoesNotInventRecommendation(t *testing.T) {
	mutations := map[string]func(*Alert){
		"outside":               func(a *Alert) { a.Geography, a.Applicability = "outside", "not-applicable" },
		"geography gap":         func(a *Alert) { a.Geography, a.Applicability = "unknown", "unknown" },
		"conflicting geography": func(a *Alert) { a.Geography = "unknown" },
		"expired":               func(a *Alert) { a.Lifecycle, a.Applicability = "expired", "not-applicable" },
		"cancelled":             func(a *Alert) { a.Lifecycle, a.MessageType = "cancelled", "Cancel" },
		"unknown timing":        func(a *Alert) { a.Lifecycle = "unknown" },
		"test alert":            func(a *Alert) { a.Status = "Test" },
		"missing source":        func(a *Alert) { a.Source = "" },
		"wrong source":          func(a *Alert) { a.Source = "https://api.weather.gov.evil.example/alerts/warning" },
		"missing issue":         func(a *Alert) { a.Issued = "" },
		"invalid expiry":        func(a *Alert) { a.Expires = "invalid" },
		"reversed times":        func(a *Alert) { a.Expires = "2026-10-10T20:00:00Z" },
		"category gap":          func(a *Alert) { a.Severity = "Unknown" },
		"certainty gap":         func(a *Alert) { a.Certainty = "Unknown" },
		"future urgency":        func(a *Alert) { a.Urgency = "Future" },
		"watch":                 func(a *Alert) { a.Event = "Tornado Watch" },
		"flood":                 func(a *Alert) { a.Event = "Flood Warning" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			a := postureWarning()
			mutate(&a)
			p := AssessPosture("LIVE", Facility{ID: "SC-DEMO-01"}, []Alert{a}, true)
			if p.RecommendedPosture != nil || p.Status != "insufficient-evidence" {
				t.Fatalf("unsupported automatic recommendation: %+v", p)
			}
		})
	}
	for _, p := range []PostureAssessment{
		AssessPosture("LIVE", Facility{ID: "SC-DEMO-01"}, nil, true),
		AssessPosture("LIVE", Facility{ID: "SC-DEMO-01"}, []Alert{postureWarning()}, false),
		AssessPosture("unknown", Facility{ID: "SC-DEMO-01"}, []Alert{postureWarning()}, true),
		AssessPosture("LIVE", Facility{}, []Alert{postureWarning()}, true),
	} {
		if p.RecommendedPosture != nil || p.Status != "insufficient-evidence" {
			t.Fatalf("missing evidence/mode/facility inferred operations: %+v", p)
		}
	}
}

func TestPostureUnrelatedWarningDoesNotCauseHoldOrChangeSuggestion(t *testing.T) {
	relevant := postureWarning()
	unrelated := postureWarning()
	unrelated.ID, unrelated.Geography, unrelated.Applicability = "elsewhere", "outside", "not-applicable"
	unrelated.Severity, unrelated.Lifecycle = "Unknown", "unknown"
	before := AssessPosture("LIVE", Facility{ID: "SC-DEMO-01"}, []Alert{relevant}, true)
	after := AssessPosture("LIVE", Facility{ID: "SC-DEMO-01"}, []Alert{relevant, unrelated}, true)
	x, _ := json.Marshal(before)
	y, _ := json.Marshal(after)
	if string(x) != string(y) {
		t.Fatalf("unrelated update changed the suggestion: before=%s after=%s", x, y)
	}
}

func TestPostureUnresolvedRelevantEvidenceRequiresReview(t *testing.T) {
	unknown := postureWarning()
	unknown.ID, unknown.Applicability = "unresolved-update", "unknown"
	p := AssessPosture("LIVE", Facility{ID: "SC-DEMO-01"}, []Alert{postureWarning(), unknown}, true)
	if p.RecommendedPosture != nil || len(p.Gaps) < 3 {
		t.Fatalf("unresolved material evidence was ignored: %+v", p)
	}
}

func TestPosturePreservesInputsAndSerializesUnknownAsNull(t *testing.T) {
	a := postureWarning()
	p := AssessPosture("LIVE", Facility{ID: "SC-DEMO-01"}, []Alert{a}, true)
	if a.Instruction != "" || !reflect.DeepEqual(a, postureWarning()) {
		t.Fatal("assessment changed official evidence")
	}
	p.Options[0].Value = "corrupted"
	next := AssessPosture("LIVE", Facility{ID: "SC-DEMO-01"}, nil, true)
	if next.Options[0].Value != "normal" {
		t.Fatal("caller modification leaked into a later assessment")
	}
	encoded, _ := json.Marshal(next)
	if !strings.Contains(string(encoded), `"recommendedPosture":null`) || strings.Contains(string(encoded), `"confidence"`) {
		t.Fatalf("unknown posture or unsupported confidence misrepresented: %s", encoded)
	}
}
