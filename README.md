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

## What to click

Start in **Recorded replay** with the captured Barnwell, South Carolina Tornado Warning. Its original NOAA text and timestamps remain intact; it is not a current warning. A synthetic T+0 instruction, “Report at 10:00,” already has three delivered notice fixtures and two queued fixtures.

1. Read the recorded official warning and its geographic footprint.
2. Choose **Review correction**, edit the two group messages, and approve the exact package as the fixture manager.
3. Choose **Re-check & simulate sending**. Five intents become queued. Repeat and zero duplicates are added.
4. Open **Source metadata & policy trace** to test stale/missing/outside/update/cancel/expired evidence or an unrelated update. Relevant changes invalidate approval. Held originals stay held until a separate operating decision releases them; none exists in this prototype.

**Live NOAA** is separate. Manually refresh the nationwide NWS feed and the selected fictional facility’s NWS point-to-zone metadata. Official instructions for applicable alerts appear without AI or operational approval. Missing data stays unknown; expiration and cancellation never authorize reopening. The UI updates expiry and freshness labels as time passes. The five-minute freshness ceiling is an illustrative technical setting, not a weather safety standard.

The [UX research](docs/UX-RESEARCH.md) and gallery compare seven patterns against the [product plan](docs/PLAN.md) and the original [boundary diagram](docs/problem-map.html). Public desk research is not user validation.

## Go and Rego boundaries

```text
go-prototype/
  cmd/weatherdesk/main.go         # HTTP lifecycle and shutdown
  internal/weatherdesk/
    evidence.go                  # NWS ingestion, polygon/zone/time checks
    http.go                      # HTTP routes and synchronized state
    replay.go                    # exact-package binding, history, dedup
    policy.go + approval.rego     # embedded OPA eligibility checks
    fixtures/                    # public recorded NOAA evidence
frontend/                        # Next App Router + reusable UI components
briefing.py                      # shared Strands instructions/schema/tool
```

Rego starts with deny. It allows approval or simulation only for an Actual, active, applicable, fresh warning in REPLAY and a facility-scoped fixture manager. Simulation also requires an exact bound approval. Policy evaluation errors deny the action. Go binds the facility, action, parameters, recipients, text, evidence conditions, policy version, and policy source digest; Go rechecks under a lock and deduplicates intents. Original notice states and immutable approved package snapshots are preserved in a local, ignored replay snapshot. Failed writes leave the prior state intact. The actor is explicitly a fixture, not production authentication.

## Strands + Akamai

SDK installation is complete. No inference credentials are in the repository. The official workshop defaults are configured: `https://ws.akamai-workshops.com/v1` and model `hackathon-chat`. Enter the organizer team key privately with `python3 scripts/configure-inference.py`; `.env` is ignored and restricted to your user. A Linode personal access token is for cloud management and is separate from this inference key. Keys stay in Python’s server environment, never the browser. This adapter needs streaming OpenAI-compatible chat, tool calling, and forced tool selection for the installed Strands structured output loop. Validate actual endpoint/model compatibility before claiming integration.

Go calls a local Python gRPC worker on 127.0.0.1:8089 using the typed contract in `contracts/proto/weatherdesk/v1/briefing.proto`. Shared Go packages provide deadlines, correlation IDs, error mapping and response identity checks. Cancellation stops async inference; requests have no automatic retries. See [reusable gRPC patterns](docs/GRPC-PATTERNS.md). One Strands agent receives applicable facility evidence and explicit gaps, with one read-only source lookup tool. The model has no approval, sending, filesystem, or provisioning tools. Current LIVE input has no prior operational review context, so agent-generated operational drafts are rejected. Replay drafts are labeled fixtures. Actual Akamai inference/model-quality evaluation: **not run / 0 samples**.

The [Akamai workshop](https://github.com/akamai-developers/akamai-workshop-hackmidwest) supplies the inference setup path; sponsor eligibility remains conditional on working implementation. No cloud resources have been created.

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

This local prototype has local fixture JSON snapshots, a synthetic manager/roster, simulated queued intents, manual refresh, and no live operational approval. No actual staff messages, delivery, or acknowledgment are claimed. Road/river/power inputs and forecast ingestion into the operational screen remain future slices; the gallery includes separately labeled recorded forecast data. Refreshing does not fetch a full historical cancellation lineage. Deterministic tests are not evidence of model quality or safety certification.

Earlier Python/buildless experiments remain in `server.py` and `prototype/` for reference and run separately on port 8086.
