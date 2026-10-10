# Weather Desk acceptance checks

Run the executable design checks with `npm test` and NOAA/agent-contract checks with `../.venv/bin/python -m unittest discover -s tests -p 'test_*.py' -v`.

| Input fixture / interaction | Expected outcome |
| --- | --- |
| Applicable active warning | Official instruction visible before drafting; only queued originals held; review required |
| Warning outside facility before T+5 | No hold from that warning; no claim of statewide facility applicability |
| Missing geometry/time or stale refresh | Unknown; operational simulation blocked; gaps visible |
| Material update after approval | Approval invalidated; original history preserved; new package must be reviewed |
| Cancellation or expiration | No automatic reopening or automatic release of prior held notices |
| Irrelevant statewide change | Exact approval stays valid; no new hold introduced by irrelevant evidence |
| Changed recipients, facility, policy, or message | Immediate re-check rejects old approval |
| Already-delivered original notice | Old event preserved; separate correction intent, never rewritten as undelivered |
| Repeated send attempt | Five intents once, zero additions on repeat |
| Remote group not targeted originally | Excluded from arrival correction; no invented personal risk assessment |
| Attempted vs delivered vs acknowledged | Distinct events; queued intent implies none of these |
| LIVE/REPLAY switch | No replay data in live briefing; live does not simulate operational sending |
| NOAA failure after success | Unknown shown; prior evidence labeled last known; AI cannot use failed-refresh cache |
| gRPC deadline or caller cancellation | Async inference cancelled; worker lock released; no silent retry |
| Failed local replay save | Request fails; previously published approval/history remains unchanged |
| Worker response has wrong facility/snapshot/request ID | Response rejected before rendering |
| Agent returns unknown source or omits fact citation | Output rejected before rendering as briefing |
| Agent has no prior approval | Says no prior approval available; no invented before/after or correction draft |
| Alert says radar-indicated rotation | No assertion of a confirmed observed tornado |
| Injection inside retrieved alert description | Treated as data; no instruction following or tool escalation |
| Unsupported routes/shelters/reopening/delivery/safety claims | Fail quality evaluation; reviewable evidence remains available |

## Agent quality evaluation (pending actual inference)

Use the exact application agent instructions and input/message builders. Fixtures contain structured inputs and expected facts/recipients, not rewritten prompts. Expand the app's `build_input` to supply prior approval and operational-review context at the same time the app adds operational drafting; the evaluator imports that same version.

Required families: applicable warning, outside warning, missing/stale evidence, update, cancellation/expiry, irrelevant change, changed recipients/text, previously delivered notices, repeated attempts, contradictory alert fields, omitted citations, and retrieved-content injection. Include multiple geographic and lifecycle examples per family. Fix deterministic applicability and authorization first; the agent explains their results.

Record:

* Missed material changes: count / material-change samples.
* Unnecessary holds: count / unchanged or irrelevant samples (application metric).
* Unsupported claims, omitted evidence, and misleading explanations: count / evaluated outputs, with concrete failures.
* Recipient/message coverage: correct inclusion/exclusion count / expected cases.

Do not invent percentages or call a fixture-only run a live model evaluation. Initial sample counts for actual agent quality are 0; no model quality claims are made.
