# Enterprise use and future in-app weather agents

Weather Desk's useful unit of work is a corrected operating decision: which office is affected, what changed since approval and which people need a revised message. A weather dashboard is an input to that work, not its outcome.

## Who would use it

A regional operations lead watches several locations. A local facility manager chooses an operating posture and approves exact messages. A staff member receives the instruction relevant to their assignment and can acknowledge it. A reviewer later reconstructs the evidence and authorization. The current app implements the manager workflow with fictional facilities and a synthetic roster; it does not yet provide real employee inboxes or enterprise tenant administration.

## A useful morning workflow

1. The regional lead sees recently issued or changed alerts, last successful source checks and offices whose coverage matches. A state-level alert does not automatically affect every office in that state.
2. The manager opens an affected office and compares current conditions with the prior instruction. Official source instructions are immediately available, even if the model is unavailable.
3. An optional agent explains the material change using the cited snapshot. Deterministic policy suggests a posture; the manager can override it with a reason.
4. The manager reviews recipients and exact text, then approves. The application rechecks the current evidence and exact approval before attempting notifications.
5. A real implementation would show provider-confirmed attempts and receipts, plus staff acknowledgments, with follow-up for people who were not reached. The hackathon app simulates those states and must keep that label.
6. The decision record preserves the original evidence, instruction and correction instead of rewriting history when new weather arrives.

The business hypothesis is reduced time from a material source change to an approved correction, with fewer missed recipients and less time reconstructing decisions. No measured savings or customer adoption are claimed.

## Future in-app agent

An in-app assistant is useful when it answers “what changed for my offices?” and links directly to the affected review. It should not generate a new paragraph for every national feed refresh.

A future event pipeline would be:

```text
Shared NWS collection → source change detection → facility match
→ relevant change since approval → deduplicate / batch / budget
→ one evidence-and-drafting agent → validated draft notification
→ manager's in-app inbox → exact approval → delivery adapter
```

A compact inbox item could show the office, changed warning, source time and “Review affected staff.” An agent can explain why the existing instruction needs review and prepare a draft. Delivery, acknowledgments and approvals remain application events. The assistant never invents shelter, evacuation or travel instructions.

Use event keys containing facility, source version and approved-package revision. Skip irrelevant updates, cache already explained evidence, cancel obsolete work and enforce per-facility and per-tenant rate/token budgets. If inference times out, show the deterministic change and source immediately. Keep external source text as data, not instructions. This is a future concept; proactive agent updates and push notifications are not implemented in the current release.

## What makes an operational pilot credible

| Area | Present implementation | Needed before an enterprise pilot |
| --- | --- | --- |
| Identity | Google login, manager allowlist, signed facility-scoped gateway principal | Tenant separation, enterprise SSO, role lifecycle, delegations and break-glass procedures |
| Evidence | Live NWS retrieval, deterministic geometry/zone checks, recorded replay, new local shared monitor | Redundant collection, historical reconciliation, regional source ownership, availability targets and outage alerts |
| Decisions | Exact-package approval, policy versions, persistent original records | Reviewed facility policy, governance for changes, backups and audit retention controls |
| Notifications | Explicit simulated intents and outcome fixtures | Real providers, verified contacts, accessibility, receipt reconciliation, retries and escalation ownership |
| AI | One Strands evidence/drafting agent, validation and persisted call metrics | Tenant budgets, representative independent evals, model-change controls and monitored drift |
| Application | Go, Next.js, private Python gRPC worker on one Akamai host | Shared durable storage, collector leadership, recovery drills, load validation and deployment controls |

## Pilot acceptance measures

Measure source-change-to-manager-visible latency separately from model latency, the share of material changes correctly matched to offices, unnecessary holds, time to approve a correction, recipients not reached, duplicate attempts and the ability to reconstruct an earlier decision. Evaluate source outages and stale evidence as first-class cases. Have facility practitioners review labels and workflow before making performance claims.

The current project demonstrates an implementation approach and a testable value hypothesis. It should be presented as a working proof of concept, not an enterprise-ready crisis communications service.
