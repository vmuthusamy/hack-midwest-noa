# Live regional weather monitoring

Weather Desk should help a manager notice changes around an office before opening a decision case. The new `/incidents` page provides regional alerts, office applicability and observed history without changing the existing decision screen.

Status: implemented locally; Go and Next.js compilation completed and a local preview displayed live NWS data. Not deployed. Flow acceptance and QA remain pending. Source: [NWS API documentation](https://www.weather.gov/documentation/services-web-api).

## Refresh and cost

| Work | Frequency | Scope |
| --- | --- | --- |
| National active alert collection | Every 60 seconds under normal conditions | One collector for the deployment, covering all states and territories |
| Browser refresh | Every 15 seconds while visible | Reads our cache, never calls NWS directly |
| Office zone lookup | At startup and every 6 hours | Five configured fictional facilities; unavailable or over-24-hour-old membership stays unknown |
| Supplementary river query | Explicit live case refresh | Selected facility only |
| Strands inference | Explicit manager request | One case with its evidence and review revision |
| Evaluation runs | Explicit developer command | Saved fixtures, separate from monitoring |

The collector uses ETag and Last-Modified conditional requests when supplied by NWS. Failed checks back off to 2, 4, then 5 minutes. Requests time out after 25 seconds and do not overlap. Normal cadence means about 1,440 national feed requests per day, regardless of how many people view the dashboard. Bytes transferred depend on feed size and conditional-cache support; this is not a measured capacity or cost guarantee.

A newly published alert normally appears after the next collector poll and browser refresh: up to roughly 75 seconds plus network and NWS publication/cache delays. This is polling, not instantaneous emergency delivery. NWS does not publish a numeric rate-limit entitlement. Official emergency warnings do not wait for AI or approval.

## Area and history semantics

State/territory membership comes from NWS UGC codes or affected-zone prefixes. Marine or unmapped records have their own filter. An alert may appear in more than one region. Office applicability uses the existing server-side polygon check, falling back to matching the appropriate NWS zone type. Being in the same state or a named nearby town does not establish facility coverage.

The page separates NWS issuance and message expiry from the service's last successful refresh and change-observation time. Expired/cancelled messages leave the current list; unknown timing and upcoming records remain labeled. Non-Actual test records are excluded. If checking fails or the last success is older than 90 seconds, the snapshot is overdue and current conditions are unknown. A new live decision cannot use an overdue snapshot.

History records the initial snapshot, first observations, NWS update/cancellation messages when present, same-ID changes and records leaving the active feed. A disappeared record is labeled **Left active feed**, not **Cancelled** or **Safe**. Different alert IDs retain separate entries; a complete reference-linked cancellation lineage is not implemented.

History is stored atomically in the service's private persistent data directory, alongside current feed state. Retention is seven days or 5,000 national events, whichever bound removes an event first. The UI reports removals and shows the newest 100 retained regional entries. It starts when collection begins; it cannot recover every event between polls or during an outage. Recorded Isaias fixtures remain REPLAY and are never injected into live history.

## Acceptance walkthrough

1. Start with monitoring enabled; open `/incidents`. Watch last successful refresh advance without requesting an AI draft.
2. Select South Carolina, then Montana. Each uses the same source-refresh timestamp and its own alerts. Inspect an office match against the source geometry/zone evidence.
3. Select a state without records. The screen reports no alerts found, not safe conditions.
4. Inspect a history entry. NWS issuance differs from the time our app first observed it. Initial snapshot is not labeled a newly issued incident.
5. Open the manager case from the chosen office. Loading the link does not approve a posture or create a case. Explicitly select LIVE and check weather to create a fresh review.
6. When the source is unavailable, confirm the cached data is marked overdue and a new decision waits for a successful check. A failed refresh must not remove active records from history.
7. Restart using the same private data directory. Confirm observed history survives and the collector resumes. The existing recorded case still runs independently.

These steps are pending user acceptance, not a claim of completed QA. Build checks do not establish operational reliability.

## Beyond one instance

The deployment currently has one API process and therefore one collector. Before adding replicas, move ingestion to a dedicated worker or use a distributed lease, shared database and fan-out notifications. Add per-tenant facility subscriptions, spatial indexing, monitored source gaps, backups and retention governance. Add a bounded historical reconciliation job against NWS's seven-day alerts endpoint to recover some outages and explicit cancellations; respect pagination and maintain observed-versus-issued timestamps. None of that should start an agent for every national alert.
