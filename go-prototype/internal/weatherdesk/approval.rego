package weatherdesk.approval

import rego.v1

# Illustrative local policy. Official emergency instructions never pass this gate.
default allow := false
default hold := false

warning if input.event in {"Tornado Warning", "Severe Thunderstorm Warning"}

ready if {
 input.mode == "REPLAY"
 input.actor_role in {"fixture-manager", "manager"}
 input.actor_facility == input.facility_id
 input.fresh == true
 input.applicability == "applicable"
 input.lifecycle == "active"
 input.status == "Actual"
 warning
}

allow if { ready; input.action == "approve" }
allow if { ready; input.action == "simulate"; input.approval_bound == true }

hold if { input.mode == "REPLAY"; input.previous_hold == true }
hold if { input.mode == "REPLAY"; input.queued_notice == true; input.fresh == false }
hold if { input.mode == "REPLAY"; input.queued_notice == true; input.applicability == "unknown" }
hold if { input.mode == "REPLAY"; input.queued_notice == true; input.applicability == "applicable"; input.lifecycle == "active"; warning }

deny contains "Operational approvals are limited to the recorded replay." if input.mode != "REPLAY"
deny contains "A facility-scoped manager is required." if not input.actor_role in {"fixture-manager", "manager"}
deny contains "The actor's facility does not match the package." if input.actor_facility != input.facility_id
deny contains "Evidence is stale or missing." if input.fresh != true
deny contains "Facility applicability is not established." if input.applicability != "applicable"
deny contains "The warning is not active; review is required, not reopening." if input.lifecycle != "active"
deny contains "Only Actual warning evidence is eligible." if input.status != "Actual"
deny contains "No eligible warning triggers this illustrative correction policy." if not warning
deny contains "The exact current package lacks manager approval." if { input.action == "simulate"; input.approval_bound != true }
deny contains "Unsupported policy action." if not input.action in {"approve", "simulate"}

decision := {"allow": allow, "hold": hold, "reasons": sort(deny), "version": "illustrative-v1"}
