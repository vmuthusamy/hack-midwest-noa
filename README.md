# Weather Desk

NOAA-first facility operations experiment for Hack Midwest. Primary user: a USDA-style facility manager reviewing an operating instruction after weather evidence changes. Fictional facilities and synthetic people only.

The new prototype uses **Next.js 16.4 / React 19.3 / Tailwind 4**, a **Go backend**, an embedded **OPA/Rego approval policy**, and the existing **Python Strands Agents 1.59 adapter** for an Akamai OpenAI-compatible inference endpoint.

## Run locally

```bash
cd ~/code/hack-midwest-ai/weather-desk
./scripts/dev.sh
```

Open [the prototype](http://127.0.0.1:8088), [the UX pattern gallery](http://127.0.0.1:8088/gallery), or [the system boundaries](http://127.0.0.1:8088/boundaries).

Go was installed from the checksum-verified official Go release into `../.tools/go` (Go 1.27.2). Python SDKs are in `../.venv`. Node.js is already installed. The launch script uses the local toolchain without changing global shell settings.

On another machine: install Go 1.26 or newer and Node.js 20.9 or newer, create `../.venv`, install `requirements.txt`, and run `npm ci` in `frontend`. Set `WEATHER_DESK_PYTHON` if your Python interpreter is elsewhere. `scripts/dev.sh` accepts an installed Go binary as a fallback.

## The manager workflow

Open a **LIVE** case from fresh NOAA evidence or an explicitly labeled **REPLAY** case from recorded public source responses. Both use one flow:

1. **Situation:** facility status and suggested posture appear together with official NWS instructions and a map. The default recorded Batesburg case is inside a saved Flash Flood Warning polygon. Barnwell is outside that polygon and has a Flood Watch; its regional gauge has no published flood-stage threshold. Great Falls demonstrates a recorded Winter Storm Warning.
2. **Operating decision:** review the versioned policy suggestion, choose delay, telework, or closure, enter an effective time and rationale. **Suggest rationale with AI** calls the actual Strands/Akamai agent against the case's exact evidence. Inference failure leaves the editable manager review available.
3. **Who is notified:** review every synthetic person's inclusion/exclusion reason, planned accessible channels and exact editable operating message. Essential status never authorizes travel. Remote staff whose arrangements do not change are excluded with a reason.
4. **Decision record:** approval saves the immutable evidence and exact package. **Re-check & simulate notices** refreshes LIVE evidence immediately; a material change requires a new review, unrelated national changes do not, and repeated attempts add no duplicates. Open the printable record to see what was known, who approved it, and the separate queued-notice state.

Normal operations and early dismissal are visible vocabulary but require separate facility procedures; this warning-review demo does not authorize reopening or departures. A watch supports planning review, not a claim that flooding has occurred. Missing/current evidence stays unknown. Recorded data never becomes LIVE implicitly. Official emergency instructions do not wait for AI or manager approval.

The separate **Inference telemetry** page shows persisted application calls, accepted/rejected/timeout/cancelled/provider outcomes, latency samples, measured tokens and actual SDK loop/tool timings. Historical hosted evaluation runs are labeled separately; missing measurements remain unknown. The two recorded guardrail catches are traceable to evaluation files rather than fabricated examples. Prompts, generated text, raw provider errors and credentials are not logged.

The [UX research](docs/UX-RESEARCH.md) and gallery compare design patterns against the product plan and boundary diagram. Public desk research is not user validation.

## Go and Rego boundaries

```text
go-prototype/
  cmd/weatherdesk/main.go         # HTTP lifecycle and shutdown
  internal/weatherdesk/
    evidence.go                  # NWS ingestion, polygon/zone/time checks
    http.go                      # HTTP routes and synchronized state
    replay.go                    # exact-package binding, history, dedup
    policy.go + *.rego            # embedded OPA eligibility checks
    operations.go                # one live/replay case and durable decision records
    case_recording.go             # explicitly recorded source snapshots
    metrics.go                   # authenticated worker measurement reader
    fixtures/                    # public recorded NOAA evidence
frontend/                        # Next App Router + reusable UI components
briefing.py                      # shared Strands instructions/schema/tool
```

Rego starts with deny. The current operations policy authorizes a facility-scoped manager's exact reviewed delay, telework or closure package with fresh, resolved evidence in either mode. LIVE uses the current clock and a fresh pre-queue source check; REPLAY uses its labeled capture clock. Go binds facility, action, posture, rationale, effective time, recipients, inclusion rules, channels, text, relevant warning conditions, policy version and policy source digest. Decision snapshots are immutable; notice outcomes append separately and survive opening another case and restarting. Failed persistence leaves previous state intact. The original arrival-correction replay and its stricter Rego policy remain available as legacy test/reference endpoints; they are not mixed into a new case.

Production uses Google OAuth through Auth.js. Go verifies a signed principal bound to facility, path, method, exact body and expiry; write nonces prevent replay. Login identity alone does not grant manager authority. API and agent gRPC are private behind the Next gateway and HTTPS edge.

## Strands + Akamai

SDK installation is complete. No inference credentials are in the repository. The official workshop defaults are configured: `https://ws.akamai-workshops.com/v1` and model `hackathon-chat`. Enter the organizer team key privately with `python3 scripts/configure-inference.py`; `.env` is ignored and restricted to your user. A Linode personal access token is for cloud management and is separate from this inference key. Keys stay in Python’s server environment, never the browser. The configured adapter uses non-streaming OpenAI-compatible chat and a constrained JSON response schema through Strands. Thinking is disabled for the workshop model; outputs have a bounded token budget. Actual hosted calls have exercised this path. The shared endpoint can time out; errors preserve the existing decision and notice history.

Go calls a local Python gRPC worker on 127.0.0.1:8089 using the typed contract in `contracts/proto/weatherdesk/v1/briefing.proto`. Shared Go packages provide deadlines, correlation IDs, error mapping and response identity checks. Cancellation stops async inference; requests have no automatic retries. See [reusable gRPC patterns](docs/GRPC-PATTERNS.md). One Strands agent receives applicable facility evidence and explicit gaps, with one read-only source lookup tool. The model has no approval, sending, filesystem, or provisioning tools. Fresh case explanation requests explicitly have no prior approval, in LIVE or REPLAY, and return no invented corrections. The manager reviews deterministic recipient-specific drafts. Legacy REPLAY correction generation receives the recorded warning, prior instruction, existing correction approval, synthetic notice states, illustrative policy and exact review revision. Generated drafts are explicitly unapproved; manager approval remains a separate Go/OPA action. The app rejects unknown or omitted citations, missing affected groups, malformed tool markup and unsupported operational movement directives. These checks are not a truth or safety certification. See the timestamped [QA and model evaluation results](docs/VERIFICATION.md).

The [Akamai workshop](https://github.com/akamai-developers/akamai-workshop-hackmidwest) supplies the inference setup path; sponsor eligibility remains conditional on working implementation. The app is deployed on [Akamai HTTPS](https://172-236-96-42.sslip.io); Google sign-in has been verified with the intended manager account. See [deployment configuration](docs/DEPLOYMENT.md).

## Checks

```bash
cd go-prototype
../../.tools/go/bin/go test -race ./...
../../.tools/go/bin/go vet ./...
cd ../frontend
npm run build
cd ..
../.venv/bin/python -m unittest discover -s tests
node --test tests/*.test.mjs
```

Use `go` directly if it is on PATH. See [acceptance cases](docs/ACCEPTANCE.md) and [verification evidence](docs/VERIFICATION.md).

## Current limits

The app has fictional facilities and a synthetic roster, persistent decision snapshots, manual refresh, and approvals of simulated notice packages in LIVE and REPLAY modes. Cloud login is OAuth; local demo access is explicitly configured. Five operating postures are shown, with illustrative evidence-based delayed-arrival suggestions and exact selected-posture binding. Essential assignments, telework eligibility, accessibility preferences and simulated channel routing are recorded in the reviewed package. No actual staff messages, delivery, or acknowledgment are claimed. LIVE river context now uses matched USGS/NOAA gauge sources and preserves unknown flood thresholds. Manager-reported power inputs are included, and recorded inputs are labeled EXAMPLE. Road conditions and forecast ingestion into the operational screen remain future slices; the gallery includes separately labeled recorded forecast data. Refreshing does not fetch a full historical cancellation lineage. Deterministic tests are not evidence of model quality or safety certification.

Earlier Python/buildless experiments remain in `server.py` and `prototype/` for reference and run separately on port 8086.

## Login configuration

Cloud authentication is mandatory. Register Google with the callback in [DEPLOYMENT.md](docs/DEPLOYMENT.md), then set the private AUTH_GOOGLE_ID and AUTH_GOOGLE_SECRET. Local development can explicitly set WEATHER_DESK_AUTH_MODE=demo and AUTH_REQUIRED=false in ignored .env; production rejects demo mode. scripts/dev.sh loads configuration as data without printing credentials.
