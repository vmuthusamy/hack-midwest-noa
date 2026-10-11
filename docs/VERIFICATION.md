# Weather Desk verification

The tested application separates live NOAA evidence, recorded replay, manager approval and simulated notices. The original USDA challenge coverage is tracked in ACCEPTANCE.md; the application is not a safety certification or a complete production notification service.

## Current unified workflow (2026-10-11)

The manager case now shows facility status and evidence beside the operating decision, followed by people and the decision record. New cases do not invent an earlier approval or employee notice. Barnwell's recorded source contains a Flood Watch and Lake Wind Advisory, not a Flash Flood Warning; SRBS1 has no published stage threshold in that response. The Montana recording contains a Winter Storm Warning. Live mode always performs its own refresh.

- Go: 66 named tests pass with race instrumentation; one opt-in browser-fixture server test is skipped in ordinary runs. `go vet ./...` passes.
- Python: 67 tests pass, including metrics persistence/sanitization, worker outcomes, bounded proposed operating context and evidence-only model input.
- Node: 76 tests pass; Next production build passes.
- Independent unified-case QA: nine Go tests with 22 subcases; seven gateway tests; ten browser assertion groups plus the final AI-input handling check. The browser used actual Auth.js decoding, signed forwarding and an isolated Go persistence root, with synthetic identities and recorded/retimed NOAA fixtures. No hosted calls are claimed by these fixture checks.
- Full browser flow checked exact review/approval, five simulated queued intents, deduplication, immutable decision snapshots, print output, mode separation, expiry gates, unknown usage and 320/390px dark/light layouts. No page or console errors were observed. [QA report](evaluation-results/operations-flow-qa-summary.json).
- The proposed posture and effective time sent to AI are unapproved context. Mismatched facility, stale timestamp and changed local choice responses are discarded. Deterministic recipient messages and exact approval remain separate from inference.

Telemetry uses an explicit rolling 24-hour completion window for application totals, 24 hourly outcome buckets, six latency bins, p50/p95 and recent calls. All complete calls in the window drive charts even though the recent table is capped at 20. Four independent Python arithmetic tests, three Node helper tests and six chart browser assertion groups pass. [Chart QA report](evaluation-results/telemetry-chart-qa-summary.json).

Telemetry persists allowlisted request outcomes, measured latency, model ID and available Strands cycles/tools/tokens in the worker's private JSONL journal. GetMetrics exposes aggregates through authenticated Go and Next routes; no prompts, generated text, credentials or raw provider errors are stored. Historical ten hosted evaluation samples remain separately labeled; eight returned outputs and two validator catches do not imply general truth or safety scores. Deployment and a fresh operating-context call are recorded below after verification.

## Earlier independent checks (prior UI and contract)

- Python: 33 tests passed including the two lifecycle-output regression cases, including real loopback gRPC identity, cancellation and shared model contract checks.
- Go: all 44 named tests passed with race instrumentation; go vet passed. Coverage includes the original approval/evidence/RPC behavior, nine authorization tests, thirteen river/posture tests and three profile/posture binding regressions.
- Frontend: 18 independent Google authorization tests passed. Isolated browser checks used actual Auth.js encrypted session decoding with synthetic keys: missing and tampered sessions denied; viewer writes and inference denied; manager allowlist recomputed server-side; 320px and 390px layouts checked.
- Next production build passed. The isolated replay browser flow checked exact approval, five queued simulated intents, duplicate prevention, immutable prior approvals, edited messages, evidence changes, inference cancellation and dark/light mobile layout. No real employee notice was sent.
- Deterministic Go fixtures: missed approval-invalidating changes 0/6; unnecessary invalidations 0/1; changed exact packages erroneously accepted 0/4; repeated send duplicates 0/1; automatic release of held original notices 0/3. These are fixture outcomes, not model-quality metrics.

## Actual hosted inference

Eight final JSON-schema inference samples were attempted through the same Strands instructions and message builder as the app. Seven structured outputs were accepted; an injection-case output was rejected. Independent review of the seven accepted summaries found one cancellation described as expiration and one omitted update marker. Missing/stale evidence also needed greater prominence in two summaries. New application guards reject both identified lifecycle defects. A fresh two-case hosted evaluation accepted an explicitly marked Update summary and rejected the cancellation output for still asserting expiration. Provider behavior remains imperfect; rejection protects the displayed review without asserting general model correctness.

Actual app requests traversed Go, gRPC, Strands and Akamai: a LIVE request returned HTTP 200 at 2026-10-11T00:03:01Z; a REPLAY request returned HTTP 200 in 48.9 seconds at 00:26:49Z. These earlier outputs precede subsequent validator and authentication changes. An accepted draft does not imply authorization or delivery.

## OAuth and cloud verification boundaries

OAuth initially used GitHub via Auth.js; Google is now the selected provider. The Google migration passes 18 provider/gateway checks, including unverified-email, mismatched-subject, old-provider-cookie and role-grant rejection. Independent Google QA used synthetic encrypted Auth.js sessions; it verified legacy-provider rejection, verified-email requirements, viewer restrictions and manager email/subject grants. The preserved report is [google-oauth-qa-summary.json](evaluation-results/google-oauth-qa-summary.json). The actual Google account chooser, basic identity consent and callback subsequently succeeded on the HTTPS deployment; the coordinating agent witnessed the app showing the intended manager identity. Independent QA reviewed the successful-login screenshot and verified cookie-free HTTP responses: root redirects to /signin, the Google sign-in page returns 200, and /api/status and /api/live return uncached 401. The provider endpoint lists only Google and an unauthenticated session is null. The separate [actual callback report](evaluation-results/google-live-oauth-qa-summary.json) distinguishes the coordinating agent’s browser evidence from independent HTTP and screenshot checks; QA could not attach to the signed-in browser. Actual viewer-provider login and a post-login mutation flow were not repeated in this read-only pass. The Go API independently verifies a signed short-lived principal bound to method, path, exact body, facility scope and write nonce; Rego receives the verified manager identity. Cloud manifests require authentication and expose only HTTPS through Caddy; API and gRPC stay on a private container network. Secrets are excluded from Git and build context, and the cloud-management token stays local.

Akamai instance weather-desk-hack26 was created in Chicago on the 4 GB plan, listed at $0.036/hour and $24/month cap. All four cloud services are running. HTTPS is verified: root redirects to sign-in, favicon and sign-in return HTTP 200, and workspace API fails closed without OAuth configuration. At 2026-10-11T00:50:41Z, private cloud integration using an explicitly synthetic signed QA principal returned HTTP 200 for worker status, actual NOAA refresh (560 national records and one regional river signal), and real Strands/Akamai LIVE inference with two source references and zero operational drafts. Viewer inference was denied. This test does not verify a provider callback. The account owner approved creation of the Google Cloud project and OAuth branding; the OAuth client is now configured and the real Google callback succeeded. The brand remains Testing and requests only openid, email and profile. Google’s [basic identity exception](https://support.google.com/cloud/answer/15549945?hl=en) means those sign-in scopes do not require a trusted test-user entry. No saved test-user registration or production brand verification is claimed. The authenticated screenshot keeps recorded replay explicitly separate from LIVE; runtime UI status was additionally read by the coordinating agent. These observations do not repeat the earlier model integration test. Local development remains explicitly labeled demo mode; production rejects demo access.

## Reproduction

Run go test -race ./... and go vet ./... in go-prototype; npm run build in frontend; ../.venv/bin/python -m unittest discover -s tests and node --test tests/auth-qa.test.mjs at project root. Browser checks are in tests/browser-qa.cjs and tests/auth-browser-qa.cjs. Inference evaluation uses tests/evaluate_agent.py and fixtures as inputs and expected outcomes, never copied prompts.

## Current real-call observations

After the initial workflow deployment, the signed-in Google manager browser opened fresh Great Falls NOAA evidence and requested a closure explanation through Next → Go → gRPC → Strands → Akamai. The worker journal records one accepted LIVE request, 41.516 seconds, one SDK cycle, 2,188 input and 482 output tokens (2,670 total), and zero tool calls. The private worker journal is persistent; the result did not approve a decision.

The final recorded Batesburg pre-run uses the exact Go-normalized capture and the same agent instructions/input builder as the application. It returned a validated two-sentence, 49-word rationale in 18.716 seconds, one SDK cycle and 3,778 reported tokens (3,278 input / 500 output), with the exact Flash Flood Warning source ID. This local hosted pre-run is stored as a timestamped recorded cache and a separate ignored metrics journal, not invented as a cloud application request. The prior pre-run was also recorded locally; it was superseded after concise-copy instruction changes. The ten earlier evaluation calls remain a separately labeled historical series.

The recorded cache is exact-bound to mode, facility, evidence timestamp/conditions, proposed posture and effective time; changes to posture, effective time or power clear it. The final deployed-browser verification is recorded below.

## Final recorded-demo QA

Independent QA passed seven focused Go tests, nine named subcases, five invalid power-input checks and ten distinct browser groups (nine workflow groups plus the final map-worker regression). The desktop approval action ends at 877.84px in a 900px viewport; the 390px phone view shows the posture before scrolling. The final same-origin MapLibre worker loads with HTTP 200; a controlled valid style renders the outside-warning map, and a failed style preserves the offline polygon. Final map checks observed zero page and console errors. The test session and roster were synthetic; no hosted calls or real staff messages were sent. [Recorded-demo QA report](evaluation-results/recorded-demo-qa-summary.json).

## Review recovery, notifications and alert timeline

The stale-revision response now includes the latest authorized case. The UI preserves the same-facility, same-mode operating draft and keeps old message edits as unapplied notes; it never retries approval or queueing automatically. Recipient preparation remains available with stale weather, while approval and simulation still require current evidence. The new Go regression exercises stale choose, approve and simulate requests without changing server state. Full Go race checks pass.

Both recipient tables now separate notification channels from status. An included person starts as Not attempted; only exact-approval simulated intents/events show Queued, Attempted, Delivered or Acknowledged. Excluded recipients remain Not targeted.

The supplied alert timeline is integrated below the map. It uses the facility IANA timezone, recorded capture clock in REPLAY, and existing synthetic work hours. Message expiry and expected hazard end are distinct; expired/cancelled local messages remain visible, outside geography is excluded and missing timing remains unknown. Ten timeline tests cover source fixtures, DST, overnight hours and lifecycle semantics. All 76 Node tests and the integrated Next production build pass.

Independent follow-up browser QA passed seven focused groups using synthetic Google sessions and isolated Go state: stale-revision closure recovery and explicit retry, preserved unapplied message notes, stale-evidence preview with approval blocked, exact-bound simulated notification states, initial desktop action visibility, and recorded timelines at 320/390px in dark/light themes. No page errors or provider calls occurred. [Follow-up QA report](evaluation-results/review-notifications-timeline-qa-summary.json).

## Final deployed-browser verification · 2026-10-11 03:03 UTC

The existing Akamai deployment rebuilt successfully and restarted the agent, API and web containers. The signed-in Google manager browser verified the recorded Batesburg timeline, facility-local EDT labels, fixed Recorded time marker and official NWS source link. Selecting Close facility with an explicit demo override opened the exact review for five included staff and one excluded remote worker. Included rows showed Not attempted, the excluded row showed Not targeted, and notification channels were separate. This deployed check left an unapproved fictional closure draft; no notification was queued or sent.

The telemetry page retained two accepted LIVE application calls across the container restart (41.5 seconds / 2,670 tokens and 16.5 seconds / 3,194 tokens). Ten historical evaluation samples and their two validator catches remained separately labeled. The browser check did not make a new inference request.

## Phone flow acceptance revision

At the user's request, further QA is paused until flow acceptance. The earlier official-alert/icon report covers the initial structured-card version only; it does not certify the subsequent compact phone layout. The phone revision uses the supplied compact NWS card, keeps full source text expandable, collapses map/timeline and rationale detail by default, and places the review action immediately below the operating choice. Existing approvals headline the approved posture. Deployment builds are required to publish the preview; no new acceptance or regression pass is claimed.
