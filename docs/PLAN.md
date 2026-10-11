# Weather Desk: design and implementation plan

## User and core question

A USDA-style facility manager needs to understand: **“We approved an arrival instruction. New evidence makes it outdated. Who needs a correction?”** The manager needs authoritative weather evidence, facility applicability, staff notice states, and a clear approval record. Uniqueness remains a hypothesis; public desk research is not a usability study.

Use the original `problem-map.html` diagram as the design baseline. Its eight nodes describe NOAA, facility/people, one supplementary local signal, deterministic checks, one drafting agent, a facility manager, recipient-specific packages, and history. **OPA refines the deterministic-checks node; it does not replace the manager or gate emergency instructions.**

## Evidence and scenario

Default recorded replay: Barnwell, SC warning captured at **2026-10-10T21:07:32.473769Z**, original alert ID `urn:oid:2.49.0.1.840.0.5474757533998c4aafaa881b6d3f43912b26da4a.001.1`. Original issue 17:06 EDT and expiry 17:15 EDT remain unchanged. NWS described a radar-confirmed tornado; source statements are not independently verified observations by this app.

Fictional Barnwell facility: 33.28, -81.35, inside the original polygon. Additional fictional Florida and California scenarios use separately recorded NOAA warning features. Each recording preserves source text, geometry, references, original dates, and retrieval time. Recorded does not mean currently active.

Synthetic sequence:

- T+0: Manager-approved “Report at 10:00”; onsite Alex delivered/Jordan queued; arriving Morgan and Casey delivered/Taylor queued; remote Sam not targeted. These delivery states are explicit fixtures.
- T+5: Recorded applicable warning introduced. Official instructions display immediately. Illustrative policy holds two queued originals and requests review.
- T+6: Review exact onsite/arriving correction text and recipients; remote remains excluded. Manager approves a bound package.
- T+7: Re-evaluate Rego and exact package/evidence under a Go lock; record five simulated queued intents. Repeat adds zero duplicates.

Original notice states and approved package snapshots remain in local replay snapshots. Holds are sticky after cancellation, expiry or outside-facility changes; no release/reopening decision is inferred. There are no real delivery routes or invented shelter/evacuation instructions.

## Live NOAA evidence

Go fetches the nationwide active alert feed with an identifying User-Agent. It separately resolves the facility’s NWS point metadata into forecast zone and county. A valid warning polygon takes precedence over zone membership. Missing polygon geometry can match exact affected forecast/county zones; malformed geometry or failed location evidence remains unknown. Current time/effective/expiry and Actual/Cancel state are checked deterministically.

LIVE contains no replay approval or notice history. The UI advances the clock to mark expiry and staleness; it preserves received official text with explicit historical/stale labels. Missing/failed/expired data is never an all-clear. Five-minute freshness is an illustrative implementation limit to validate. No silent live/replay mixing.

## Clickable design and gallery

Dark command center with a light option. Top: mode, fictional facility, authoritative warning text/source/dates. Middle left: official polygon + facility extent plot; middle right: previously approved → changed facts → next action. Below: staff group/notice state comparison, decision history, expandable source and policy trace. Native modal dialog supports exact text review and keyboard focus. Tailwind and shared React Panel/Button/Badge/SourceLink components provide consistent treatment.

The [UX investigation](UX-RESEARCH.md) maps seven gallery patterns to product requirements: authoritative warning, geography variants, material-change diff, recipient matrix, event timeline, sourced discrete forecast bars, and policy explanation. The gallery offers research annotations and dark/light variants. Forecast bars use recorded inputs with their own dates; they do not imply a connected live forecast integration.

Validate with 3–5 facility or continuity practitioners: identify the official instruction, explain applicability, find who received the old message, edit/approve exact correction, distinguish queued from delivered, and identify unknown evidence. Measure errors and completion with sample counts. No interviews have been performed.

## Architecture and policy

Next.js App Router frontend on 127.0.0.1:8088 proxies `/api/*` to Go on 127.0.0.1:8087. Go uses `cmd/weatherdesk` for lifecycle and `internal/weatherdesk` for HTTP, evidence, replay and policy boundaries. Request contexts, upstream timeouts, bounded bodies, explicit JSON contracts, synchronized state, graceful shutdown, formatting, tests with race instrumentation, and go vet support a maintainable small prototype. Dependencies are pinned in Go/Node lockfiles.

Embedded OPA/Rego (`approval.rego`, illustrative-v1) evaluates allow/deny and queue holds. Server-derived input includes the fixture actor/facility, freshness, geographic/time applicability, alert status/event, action and approval binding. It denies LIVE approvals, wrong role/facility, missing/stale/non-applicable/non-active/non-Actual evidence and sending without exact approval. Undefined/evaluation failure denies. Go binds exact facility/action/parameters/recipients/text/policy version/source digest/material evidence and deduplicates recipient intents. Policies can later be signed/distributed bundles after authenticated real roles and durable storage exist.

Go calls the Python worker through a versioned Protobuf/gRPC contract. Common client libraries bound deadlines, propagate request IDs, validate response identity, and map transport errors. Cancellation reaches async inference, and automatic retries are disabled. Local replay snapshots survive restarts; authenticated durable production storage remains a later slice.

One Python Strands agent receives relevant normalized facility evidence and explicit unresolved gaps. Go narrows input to avoid unrelated nationwide alerts dominating the briefing. The shared `briefing.py` instructions/schema/tool remain authoritative for app and evals. Python reads only supplied source references; model output cannot approve, hold, send, reopen or provision. LIVE has no prior operating review, so operational drafts are rejected.

The workshop endpoint, model alias and private organizer team key are configured locally. Real Strands structured-output calls through Akamai have executed. The model drafts exactly two correction groups for an eligible replay and has no operational authority. Shared-provider failures return recoverable errors; they do not alter approval or history. Sponsor eligibility still depends on the organizers’ review. No cloud provisioning in this slice.

## Small slices and design review

Delivered for review: clickable Next dashboard, separate live NOAA ingestion, recorded warning replay, exact reviewed corrections and simulated intents, embedded Rego, typed Strands gRPC worker, seven-pattern UX gallery and original boundary diagram.

Next after design review:

1. Resolve practitioner/user feedback on operating postures, notice audiences, approval wording, and evidence hierarchy. Choose gallery patterns and density before expanding.
2. Use the completed inference and shared-instruction evaluation results to improve source coverage, output quality and latency. Expand evaluation fixtures and practitioner testing; see VERIFICATION.md for measured sample counts and limitations.
3. Add one real supplementary local signal and source-aware forecast/observation ingestion, with missing/stale semantics and traceable thresholds.
4. Add authenticated facility-scoped roles, append-only persistent decision/package/notice store and transactional intent dedup. Keep official warnings independent.
5. Follow update/cancel reference lineage and resilient caching/refresh. Then scope an authorized Akamai Cloud deployment.

Tests cover applicable/outside/missing/stale/update/cancel/expired/irrelevant evidence, edited recipients/message/facility/policy, immutable history, already-delivered originals, repeat sends, Rego role/scope/deny cases, and HTTP revisions. No invented confidence percentages or safety certification. App and evaluator import the same Strands instructions, builder and output contract. Eight recorded/synthetic model cases and deterministic workflow checks are reported separately in VERIFICATION.md; provider errors are counted as errors, never quality passes.
