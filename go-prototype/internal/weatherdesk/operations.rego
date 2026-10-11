package weatherdesk.operations
import rego.v1
default allow := false
manager if { input.role in {"manager", "fixture-manager"}; input.facility_scoped == true }
override_ok if input.override_required != true
override_ok if input.override_reviewed == true
ready if { manager; override_ok; input.fresh == true; input.resolved == true; input.action in {"approve", "simulate"}; input.posture in {"delay", "telework", "closure"} }
allow if { ready; input.action == "approve" }
allow if { ready; input.action == "simulate"; input.bound == true }
deny contains "A facility-scoped manager is required." if not manager
deny contains "Current evidence is missing or stale; refresh before approval." if input.fresh != true
deny contains "A relevant alert has unresolved applicability or timing." if input.resolved != true
deny contains "Normal operations and departure decisions require a separate facility procedure." if not input.posture in {"delay", "telework", "closure"}
deny contains "The exact reviewed package is not approved." if { input.action == "simulate"; input.bound != true }
deny contains "Explain why the selected posture differs from the current recommendation." if { input.override_required == true; input.override_reviewed != true }
decision := {"allow": allow, "hold": false, "reasons": sort(deny), "version": "demo-facility-operations-v3"}
