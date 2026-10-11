# Weather Desk verification

The tested application separates live NOAA evidence, recorded replay, manager approval and simulated notices. The original USDA challenge coverage is tracked in ACCEPTANCE.md; the application is not a safety certification or a complete production notification service.

## Independent checks

- Python: 33 tests passed including the two lifecycle-output regression cases, including real loopback gRPC identity, cancellation and shared model contract checks.
- Go: all 44 named tests passed with race instrumentation; go vet passed. Coverage includes the original approval/evidence/RPC behavior, nine authorization tests, thirteen river/posture tests and three profile/posture binding regressions.
- Frontend: 15 independent authorization tests passed. Isolated browser checks used actual Auth.js encrypted session decoding with synthetic keys: missing and tampered sessions denied; viewer writes and inference denied; manager allowlist recomputed server-side; 320px and 390px layouts checked.
- Next production build passed. The isolated replay browser flow checked exact approval, five queued simulated intents, duplicate prevention, immutable prior approvals, edited messages, evidence changes, inference cancellation and dark/light mobile layout. No real employee notice was sent.
- Deterministic Go fixtures: missed approval-invalidating changes 0/6; unnecessary invalidations 0/1; changed exact packages erroneously accepted 0/4; repeated send duplicates 0/1; automatic release of held original notices 0/3. These are fixture outcomes, not model-quality metrics.

## Actual hosted inference

Eight final JSON-schema inference samples were attempted through the same Strands instructions and message builder as the app. Seven structured outputs were accepted; an injection-case output was rejected. Independent review of the seven accepted summaries found one cancellation described as expiration and one omitted update marker. Missing/stale evidence also needed greater prominence in two summaries. New application guards reject both identified lifecycle defects. A fresh two-case hosted evaluation accepted an explicitly marked Update summary and rejected the cancellation output for still asserting expiration. Provider behavior remains imperfect; rejection protects the displayed review without asserting general model correctness.

Actual app requests traversed Go, gRPC, Strands and Akamai: a LIVE request returned HTTP 200 at 2026-10-11T00:03:01Z; a REPLAY request returned HTTP 200 in 48.9 seconds at 00:26:49Z. These earlier outputs precede subsequent validator and authentication changes. An accepted draft does not imply authorization or delivery.

## OAuth and cloud verification boundaries

OAuth initially used GitHub via Auth.js; Google is now the selected provider. The Google migration passes 18 provider/gateway checks, including unverified-email, mismatched-subject, old-provider-cookie and role-grant rejection. Actual Google callback verification still requires its registered client. The Go API independently verifies a signed short-lived principal bound to method, path, exact body, facility scope and write nonce; Rego receives the verified manager identity. Cloud manifests require authentication and expose only HTTPS through Caddy; API and gRPC stay on a private container network. Secrets are excluded from Git and build context, and the cloud-management token stays local.

Akamai instance weather-desk-hack26 was created in Chicago on the 4 GB plan, listed at $0.036/hour and $24/month cap. All four cloud services are running. HTTPS is verified: root redirects to sign-in, favicon and sign-in return HTTP 200, and workspace API fails closed without OAuth configuration. At 2026-10-11T00:50:41Z, private cloud integration using an explicitly synthetic signed QA principal returned HTTP 200 for worker status, actual NOAA refresh (560 national records and one regional river signal), and real Strands/Akamai LIVE inference with two source references and zero operational drafts. Viewer inference was denied. This test does not verify a GitHub callback. Real provider callback verification is pending Google Cloud client registration after the account owner signs in. Local development remains explicitly labeled demo mode; production rejects demo access.

## Reproduction

Run go test -race ./... and go vet ./... in go-prototype; npm run build in frontend; ../.venv/bin/python -m unittest discover -s tests and node --test tests/auth-qa.test.mjs at project root. Browser checks are in tests/browser-qa.cjs and tests/auth-browser-qa.cjs. Inference evaluation uses tests/evaluate_agent.py and fixtures as inputs and expected outcomes, never copied prompts.
