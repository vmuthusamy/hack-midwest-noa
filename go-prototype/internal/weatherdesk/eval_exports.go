package weatherdesk

import "time"

// Thin exports for cmd/posture-eval. They call the production code paths unchanged.

func NormalizeForEval(f Feature, facility Facility, zones []string, now time.Time) Alert {
	return normalize(f, facility, zones, now)
}

func ApplyPowerForEval(e *Evidence) { applyPowerAssessment(e) }
