package weatherdesk

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"

	"github.com/open-policy-agent/opa/v1/rego"
)

//go:embed approval.rego operations.rego
var policyFiles embed.FS

type PolicyDecision struct {
	Allow   bool     `json:"allow"`
	Hold    bool     `json:"hold"`
	Reasons []string `json:"reasons"`
	Version string   `json:"version"`
}

var policySource, _ = policyFiles.ReadFile("approval.rego")
var policyDigest = digest(string(policySource))
var preparedPolicy, preparePolicyError = rego.New(rego.Query("data.weatherdesk.approval.decision"), rego.Module("approval.rego", string(policySource))).PrepareForEval(context.Background())

func evaluatePolicy(ctx context.Context, input map[string]any) (PolicyDecision, error) {
	if preparePolicyError != nil {
		return PolicyDecision{}, preparePolicyError
	}
	results, err := preparedPolicy.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return PolicyDecision{}, err
	}
	if len(results) != 1 || len(results[0].Expressions) != 1 {
		return PolicyDecision{}, fmt.Errorf("undefined approval policy")
	}
	raw, err := json.Marshal(results[0].Expressions[0].Value)
	if err != nil {
		return PolicyDecision{}, err
	}
	var decision PolicyDecision
	err = json.Unmarshal(raw, &decision)
	return decision, err
}
func policyInput(r *Replay, action string) map[string]any {
	bound := r.Approval != nil && r.Approval.Binding == digest(packageFor(r))
	return map[string]any{"mode": r.Mode, "actor_role": "fixture-manager", "actor_facility": r.Facility.ID, "facility_id": r.Facility.ID, "action": action, "fresh": r.Fresh, "applicability": r.Alert.Applicability, "lifecycle": r.Alert.Lifecycle, "status": r.Alert.Status, "event": r.Alert.Event, "approval_bound": bound, "queued_notice": true, "previous_hold": len(r.Held) > 0}
}
func (r *Replay) refreshPolicy() error {
	d, err := evaluatePolicy(context.Background(), policyInput(r, "approve"))
	if err != nil {
		r.PolicyDecision = PolicyDecision{}
		return err
	}
	r.PolicyDecision = d
	if d.Hold {
		for _, person := range r.Roster {
			if person.Original != "queued" {
				continue
			}
			seen := false
			for _, id := range r.Held {
				if id == person.ID {
					seen = true
				}
			}
			if !seen {
				r.Held = append(r.Held, person.ID)
			}
		}
	}
	return nil
}
