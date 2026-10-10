# Experiment verification — October 10, 2026

* 13 named Go tests passed with race instrumentation; `go vet ./...` passed. Coverage includes geographic/time applicability, policy scope/default denial, exact bindings, sticky holds, immutable approved snapshots, duplicate prevention, HTTP revision checks, failed-persistence isolation and shared RPC deadlines/error mapping.
* 18 Python tests passed. Three exercise real loopback gRPC with a stub generator: typed evidence identity, unconfigured-provider rejection, and deadline cancellation/worker-lock release. Other tests cover normalized NOAA evidence, geographic holes/boundaries/multipolygons and shared agent contracts/source references.
* 17 Node checks passed for the earlier design experiment. Its deterministic fixture report: missed material changes **0/5**; unnecessary new holds or invalidations **0/2**. These small samples do not measure model quality or operational effectiveness.
* Next.js production build passed for `/`, `/gallery` and `/boundaries`.
* Final browser checks passed: demo identity, dark/light, 390px layout without horizontal overflow, gallery controls, interactive diagram lens, real NOAA refresh, disabled unconfigured inference and browser-only stale/expiry fixtures. No page script errors. Desktop/mobile screenshots were visually checked. Final checks did not mutate the user's replay history.
* Real NOAA refresh at **2026-10-10T21:45:52Z** returned 515 nationwide records. For the fictional Barnwell point, deterministic checks returned 3 applicable records and 4 with unknown applicability. This timestamped observation is not a current-warning claim or an all-clear.
* Astra high review confirmed the latest gRPC cancellation, retry and persistence fixes. No remaining blocker was found in that reviewed scope.
* The user-supplied Akamai Cloud token authenticated to the official Linode profile endpoint (HTTP 200). No profile information was logged and no cloud resources were created. Its ignored local configuration is separate from the workshop model key.
* Actual hosted Strands inference and model-quality evaluation: **not run, 0 samples**. The organizer inference team key is still required. Transport tests and successful cloud API authentication do not verify model compatibility.

Authenticated manager roles, production durable storage/transactional sending, full update/cancellation lineage, supplementary local signals and cloud deployment remain future slices. The running prototype uses a demo manager, ignored local replay snapshots and simulated queued intents; it sends no employee notices.

Reproduce: `go test -race ./...` and `go vet ./...` in `go-prototype`; `npm run build` in `frontend`; `../.venv/bin/python -m unittest discover -s tests`, `npm test` and `npm run report` at project root. App and agent tests import the shared `briefing.py`; transport fixtures substitute generation and make no provider call.
