package weatherdesk

import (
	"context"
	"testing"
)

func TestRegoEligibility(t *testing.T) {
	r, _ := newReplay("barnwell", nil)
	baseline := policyInput(r, "approve")
	d, err := evaluatePolicy(context.Background(), baseline)
	if err != nil || !d.Allow || !d.Hold {
		t.Fatal(d, err)
	}
	cases := []struct {
		key   string
		value any
	}{{"mode", "LIVE"}, {"actor_role", "agent"}, {"actor_facility", "OTHER"}, {"fresh", false}, {"applicability", "unknown"}, {"lifecycle", "cancelled"}, {"status", "Test"}, {"action", "unsupported"}, {"event", "Flood Watch"}}
	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			input := policyInput(r, "approve")
			input[c.key] = c.value
			d, err := evaluatePolicy(context.Background(), input)
			if err != nil || d.Allow || len(d.Reasons) == 0 {
				t.Fatal(d, err)
			}
		})
	}
	d, err = evaluatePolicy(context.Background(), policyInput(r, "simulate"))
	if err != nil || d.Allow {
		t.Fatal(d, err)
	}
	r.approve()
	d, err = evaluatePolicy(context.Background(), policyInput(r, "simulate"))
	if err != nil || !d.Allow {
		t.Fatal(d, err)
	}
}
func TestHeldOriginalsAreNeverReleasedByEvidenceChange(t *testing.T) {
	for _, kind := range []string{"outside", "cancel", "expired"} {
		r, _ := newReplay("barnwell", nil)
		r.setCase(kind)
		if len(r.Held) != 2 {
			t.Fatalf("%s released a held notice", kind)
		}
	}
}
