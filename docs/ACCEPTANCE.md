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

## Full USDA challenge coverage

Source: user-supplied **Closure Call: Weather-Driven Facility Decisions and Notification** challenge, received October 10, 2026. This is a requirement audit, not a claim of completion. Baseline below reflects the Go/Next.js prototype before the newly requested OAuth, deployment, broader facility model and supplemental-data slices. Replace a gap with verified coverage only after its implementation and check pass.

| Requirement | Existing implementation evidence | Baseline status / acceptance for completion |
|---|---|---|
| Real NWS alerts in live demo | Go `fetchEvidence` requests actual nationwide NWS API; normalizes geography/time; browser manual refresh | Implemented locally. Refresh must show source, issue/expiry, last successful fetch; verify real response separately from recorded replay. |
| Forecasts and observations | Earlier NOAA exploration saved recorded hourly/station samples | Not connected in Go live screen. Resolve `/points`, fetch original hourly periods and latest station observation; show their independent timestamps/units and missing/stale state. |
| One facility profile, two or three hazards handled well | Fictional Barnwell, Big Bend and inland California selectors; recorded tornado/severe-thunderstorm examples | Scope the primary demo to one full facility profile and tornado/storm/flood. Alternate locations are evidence examples, not three complete facility implementations. |
| Facility location and operating hours | Location model exists; hours absent | Add explicit fictional operating schedule/timezone. Evaluate arrival/onsite context against schedule without inventing a safe travel time. |
| Employee assignments | Six synthetic people and onsite/arriving/remote groups | Partial. Add assigned facility, temporary detail/current worksite and effective dates; building membership must not equal current attendance. |
| Essential personnel | No essential flag in Go recipient model | Gap. Mark demo essential role and give explicitly reviewed reporting requirements; never infer essential staff must ignore official emergency instructions. |
| Telework eligibility | Remote group present, eligibility absent | Gap. Separate current remote status from eligibility; only eligible people receive an illustrative telework option. |
| Accessibility needs | No accessibility/channel preference fields | Gap. Add synthetic accessible channel/language preferences without real sensitive data; message content/route must respect the preference. |
| At least one supplementary operational signal | No Go supplemental ingestion at baseline | Gap. USGS/NOAA river slice under investigation below. It must retain observation vs fetch time, units, source and explicit facility-relevance limits. |
| Graded recommendation | Current package only pauses arrival pending review | Gap. Return normal, delay, early dismissal, unscheduled leave/telework or closure as reviewable suggestions; retain unknown/manual review when evidence is insufficient. Manager chooses posture. |
| Every driving input with source/time | Alerts cited; exact package stored | Partial. Recommendation references normalized alert, forecast/observation and supplemental inputs individually; no hidden unsourced facts. |
| Crossed threshold and magnitude | Categorical warning review gate; no threshold report | Partial. For alert fields state the named criterion/category; numeric exceedance only for compatible units at the same gauge and published threshold. Undefined thresholds remain unknown. |
| Confidence and what would change | Agent evidence-gap output, no explicit sufficiency/change condition | Partial. Use qualitative evidence sufficiency with named gaps; no invented confidence percentages. State specific evidence changes that require another review. |
| One paragraph manager rationale | Strands explanation/drafts contract exists | Implemented contract, hosted output verification pending baseline. Rationale must distinguish official evidence, illustrative policy, manager decision and uncertainty. |
| Recipient list + include/exclude logic | Group targets and remote exclusion in replay | Partial. Show each person's assignment/attendance/essential/telework basis plus message version; explain exclusions. |
| Right channel/time/content | Editable group messages; queued simulated intents | Partial. Add proposed per-recipient routing and urgency before approval; sending remains explicitly simulated under existing user constraints. Do not claim real reach/delivery. |
| Human decision-maker | Demo manager exact package approval and OPA eligibility | Local demo implemented. OAuth identity and real facility role are newly requested; signed-in identity must be distinct from authority to approve. |
| Reconstructable record | Local ignored replay snapshots, preserved approval packages, events and intents | Local implementation. Deployment must use durable persistence, retain evidence snapshot and manager subject, fail without publishing changes when save fails. |
| Defensible cutoffs | NWS warning category gate, illustrative policy label | Partial. Publish policy version and primary source anchors; geographic proximity is not a flooding threshold. Numeric local cutoffs need jurisdiction-specific source. |
| Review changes after approval | Exact facility/action/params/recipient/text/policy/evidence binding; dedup | Implemented. Relevant change invalidates; unrelated feed refresh does not. Re-check must precede each new simulated intent. |
| Working sponsor usage | Python Strands SDK + Akamai-compatible adapter; local Go/OPA and gRPC transport | SDK/transport implemented. Hosted inference and Akamai deployment require actual successful verification; eligibility remains conditional on real use. |
| Login and Akamai deployment (new user request) | Not part of original USDA deliverables; newly authorized | Track separately: configured OAuth callback/logout/session, secrets server-side, published HTTPS health/API checks and durable data. Never substitute demo identity for actual login. |

### Acceptance families for the missing challenge slices

| Input / change | Required result |
|---|---|
| Assigned elsewhere or temporary detail outside this facility | Exclusion explained; not accidentally notified merely because on the building roster. |
| Essential onsite person vs nonessential arriving person | Separately reviewed operational messages; no instruction to evacuate, travel through hazard or ignore official instructions. |
| Remote but not telework eligible / eligible but currently onsite | Status and eligibility remain separate; no invented benefit or leave authorization. |
| Synthetic accessible channel preference | Exact proposed channel included in reviewed package and digest; changing channel requires review. |
| Applicable severe/extreme immediate NWS warning | Original official instruction immediately visible; closure/delay suggestion is separately labeled illustrative and manager-reviewed. |
| No warning and missing supplementary data | Evidence insufficient, no safe/all-clear conclusion; do not silently choose normal operations. |
| Same-gauge observed stage crosses a sourced minor-flood stage | Report stage, threshold, numeric difference, units, source/time and local relevance limits. |
| Same river but different gauge or datum | No borrowed threshold or fabricated facility inundation; keep the comparison unknown. |
| River API empty, sentinel, stale, future timestamp or failed refresh | Missing/current unknown visible; retain last known value only if clearly labeled; no auto release/reopening. |
| Recommendation changes or manager selects another graded posture | Exact new posture/parameters/group channels/message text and evidence bound to approval; preserve prior decision. |
| OAuth signed-in but no approved facility role | Deny operational approval; identity alone grants no facility authority. |
| Restart/deploy with approved record | Evidence/decision/message history preserved in durable store; no duplicate intents on replay. |

## Verified supplemental-data investigation

Checked public APIs around **2026-10-11 00:35 UTC** (October 10 evening Central). These are timestamped observations, not ongoing current-condition claims.

- **Closest useful Barnwell river context:** USGS `USGS-02175200`, Salkehatchie River near Barnwell. Latest stage `6.39 ft`, observed `2026-10-10T23:30:00Z`, provisional, sensor point `33.240428,-81.408163`. Source: [USGS latest continuous endpoint](https://api.waterdata.usgs.gov/ogcapi/v1/collections/latest-continuous/items?monitoring_location_id=USGS-02175200&parameter_code=00065&f=json&limit=20). NOAA matching gauge `SRBS1` identifies the same USGS ID but has **no defined action/minor/moderate/major stage** (missing sentinels); category `not_defined`. [NOAA metadata](https://api.water.noaa.gov/nwps/v1/gauges/srbs1).
- **A tempting but invalid shortcut:** downstream Miley `USGS-02175500` / `MILS1` has published action/minor/moderate/major stages `8/9/11/14 ft`. Those apply to that gauge, not Barnwell. Do not transfer them to Barnwell or infer any employee's road is flooded. [NOAA Miley metadata](https://api.water.noaa.gov/nwps/v1/gauges/mils1).
- **Big Bend regional context:** Steinhatchee `STEF1` has observed `12.94 ft` at `23:00Z`, with same-source action/minor/moderate/major stages `17.5/19.5/23/26 ft`. The observation is `6.56 ft below` its published minor stage, which does not establish facility or route safety. Direct USGS `02324000` latest data returned December 2, 2025 and different numeric stage/datum context; never mix those observations with the current NOAA thresholds. [NOAA Steinhatchee metadata](https://api.water.noaa.gov/nwps/v1/gauges/stef1).

A sensor near a facility is regional context until a manager supplies a justified river/road/facility dependency. Our fictional facility has no surveyed inundation or commute mapping. Store `relevance: regional-context`, `facilityImpact: unknown`, and a plain-language relevance explanation. The supplemental slice must not fabricate a closure trigger from proximity.

Primary API guides: [USGS OGC API](https://api.waterdata.usgs.gov/docs/ogcapi/), [NOAA NWPS OpenAPI](https://api.water.noaa.gov/nwps/v1/docs/). Normalized fields needed: source/provider, gauge and USGS ID, sensor coordinates, facility association basis, observedAt, fetchedAt, value/unit/parameter, provisional/quality state, original category, published thresholds with source, numeric exceedance only when comparable, gaps. Missing `-999`/`-9999` values normalize to null.

### Supplemental and posture slices implemented locally

The Go public reader `FetchRiverSignals` now supplies Barnwell USGS stage plus NOAA gauge metadata, or Big Bend NOAA stage and same-response thresholds. It preserves source, observed/fetched time, provisional status, nullable values and observation age. Availability is distinct from freshness: the calling operational policy must assess age before using an observation in a decision. Facility and access-road impact remain unknown. Failed refreshes return explicit gaps alongside any partial metadata; they never create a successful current observation.

`AssessPosture` implements the narrow, versioned `illustrative-weather-posture-v1` policy. It exposes five manager choices and suggests only **delayed arrival** when a complete applicable active Actual tornado/thunderstorm warning meets Severe/Extreme severity, Immediate urgency and Observed/Likely certainty. It retains the alert's source/issue/effective/expiry fields and named categorical criterion. Unknown/stale evidence or no matching warning yields a null recommendation. The policy does not assess forecasts, observations, employee eligibility or facility dependencies; those limits stay visible. The human's selected posture and messages still require exact-package authorization.

Local verification: **13 top-level Go test families plus input subcases**, passing with the race detector (`go test -race ./internal/weatherdesk -run 'River|Posture' -count=1`). Cases include sentinel/null values, wrong gauge and parameter, incompatible units, invalid/future/old observation time, invalid latest reading, HTTP failure/cancellation, unknown facility, all-clear/reopening prevention, warning category gaps, mode separation and irrelevant updates. These are deterministic source-response and policy tests, not live delivery or safety certification.

### Minimal graded-posture policy proposal

These are **illustrative options**, not USDA/OPM rules or automatic safety actions. Preserve the five posture choices: normal, delayed arrival, early dismissal, unscheduled leave/telework, closure. Applicable official warning fields and any validated access/power/river dependencies determine what needs manager review. A complete, applicable severe immediate tornado or thunderstorm warning supports a delayed-arrival/arrival-hold proposal for review; a watch supports monitoring and a delay/eligible-telework option, not automatic closure. Early dismissal requires a manager-reviewed operational plan and never means leave during an active tornado warning. Missing evidence yields “Review needed / recommendation limited,” and expired/cancelled warnings never automatically select normal or reopen.

Defensibility anchors: official [NWS tornado meanings](https://www.weather.gov/safety/tornado-ww), published gauge-specific NWPS flood categories, and [OPM posture procedures](https://www.opm.gov/policy-data-oversight/pay-leave/reference-materials/handbooks/dcdismissal/) for category vocabulary only. Severity/urgency are categorical; report a matched named criterion rather than a fabricated numeric distance from a category. Delay duration and effective time are explicit manager parameters, not unsourced wind cutoffs.

## Agent quality evaluation

Use the exact application agent instructions and input/message builders. Fixtures contain structured inputs and expected facts/recipients, not rewritten prompts. The app supplies prior approval, synthetic recipients, illustrative policy and operational-review context; the evaluator imports the same builder and generator from `briefing.py`. Run `../.venv/bin/python tests/evaluate_agent.py` to inspect inputs with zero provider calls, or explicitly add `--run --output /tmp/agent-results.json` for hosted inference. Browser QA uses a separate API root and marked NOAA/model fixtures, never the user’s replay store.

Required families: applicable warning, outside warning, missing/stale evidence, update, cancellation/expiry, irrelevant change, changed recipients/text, previously delivered notices, repeated attempts, contradictory alert fields, omitted citations, and retrieved-content injection. Include multiple geographic and lifecycle examples per family. Fix deterministic applicability and authorization first; the agent explains their results.

Record:

* Missed material changes: count / material-change samples.
* Unnecessary holds: count / unchanged or irrelevant samples (application metric).
* Unsupported claims, omitted evidence, and misleading explanations: count / evaluated outputs, with concrete failures.
* Recipient/message coverage: correct inclusion/exclusion count / expected cases.

Do not invent percentages or call a fixture-only run a live model evaluation. Measured sample counts, provider failures and manual output review are recorded in VERIFICATION.md. Contract passes alone are not model-quality passes.

Additional executable checks cover model-generated replay drafts without state mutation, stale review revision rejection, concurrent material changes, clean plain-text rendering, exact approval text after generation, and approval archive preservation. Browser fixtures test generation, approval, reapproval, live expiry, themes and mobile layout.

## Current unified manager workflow (October 10 evening update)

The primary UI now uses the same case model for LIVE and explicitly recorded REPLAY: situation → operating decision → recipients → printable decision record. New cases do not invent a prior approval or delivered notice. The original arrival-correction replay remains a separately stored legacy contract.

- The actual recorded Barnwell input is a Flood Watch plus Lake Wind Advisory and NOAA SRBS1 river observation. It has no published minor-flood stage; a neighboring warning and another gauge's threshold are never substituted. Great Falls has a separately recorded actual Winter Storm Warning and point-zone membership. These snapshots describe their capture times, not current conditions.
- The versioned demo warning policy adds flood and winter telework proposals, accepts valid Expected urgency for winter planning, and distinguishes watches from warnings. The manager reviews operating time against hazard onset/end. Normal operations/early dismissal still need separate facility procedures. No surveyed road/river dependency, forecast or station observation is connected to the operating gate.
- Exact approval covers posture, rationale, effective time, recipients/reasons/channels/messages and relevant warning conditions. LIVE re-fetches immediately before recording simulated intents. A material warning change blocks queueing and preserves the old record; an outside national update or retrieval timestamp alone does not invalidate approval. Regional gauge context is frozen in the record but does not drive this warning-only policy.
- Printable records preserve evidence/source times, suggestion versus choice, approver, digest, included/excluded people, exact text and append-only notice outcomes. New cases and restarts preserve prior records. Failed writes publish neither approval nor new intents. Only queued simulated states are asserted.
- NWS fire-zone membership is collected independently by validated type. Missing fire lookup remains unknown for fire alerts; a resolved unrelated fire zone is outside. Unsupported unrelated hazard records no longer veto a matched tornado/flood/winter operating policy.
- Authenticated inference telemetry records actual request outcomes and available SDK measurements. Saved historical hosted evaluation samples remain a distinct series. Missing historical token/cycle data stays unknown; accepted output is not equivalent to a truth/quality pass.

Independent fixture QA covers the exact workflow and authorization; browser/cloud verification is reported separately in VERIFICATION.md. Sponsor integration is demonstrated by actual hosted requests and the existing Akamai deployment, not the presence of SDK names alone.

## Recorded SC demo brief implementation

The default fictional Batesburg facility (SC-DEMO-02, 33.91, -81.54) uses the saved October 11 02:20 UTC NWS capture. Its Flash Flood Warning polygon contains the facility and satisfies the existing Severe/Likely/Immediate telework criterion. Barnwell remains outside that polygon and retains a Flood Watch planning example; its gauge has no published flood-stage comparison. The displayed category criterion is an illustrative operating policy, not an invented numeric threshold.

The demo clock stays at the actual capture time. The requested 4 AM opening is a presentation framing, not a retiming of NOAA evidence. The supplied Isaias/Destin death and power-total narrative is not shown because the captured alert does not establish those claims.

The status-first page presents the facility, hazard, message expiry and posture before the map, people/message preview and decision record. Closure overrides require a stored reason. Local power reports carry their source, observation time and Example/Manager reported provenance; warning plus reported outage changes the proposed posture only. Official warnings remain immediately visible, independent of approval. Essential staff receive individual review, without an automatic travel instruction.

Queued intents remain distinct from explicitly run delivery fixtures. Fixture attempts, delivery, acknowledgment and unreachable/channel fallback events are stored separately, deduplicated by exact approval, and never claim employee delivery. The planned 6 AM update is labeled an example schedule. The pre-run model cache contains real validated Strands/Akamai output for the exact recorded facility/evidence/posture/time and is cleared on a changed proposal or power report.
