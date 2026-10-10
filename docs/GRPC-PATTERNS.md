# Reusable gRPC patterns

The browser calls Go over HTTP. Go calls the single Python Strands worker over loopback gRPC. Credentials remain in Python; the worker only explains evidence. Approval and simulation stay in Go and OPA.

| Shared component | Purpose |
| --- | --- |
| `contracts/proto/weatherdesk/v1/briefing.proto` | Versioned typed facility, alert, snapshot, draft and source contracts |
| `go-prototype/pkg/rpcpattern` | Parent cancellation, bounded deadlines, request ID metadata, HTTP error mapping |
| `go-prototype/pkg/agentclient` | Connection lifecycle, 2 MiB bounds, snapshot identity checks, no automatic retries |
| `go-prototype/pkg/gen/weatherdeskv1` | Generated Go client/server types |
| `agent_service/generated` | Generated Python message/service types |
| `agent_service/server.py` | Provider configuration, single inference lock, cancellation, sanitized failures |

`GetStatus` reports configuration without exposing secrets. `Explain` accepts LIVE evidence and returns structured facts, citations, gaps and explanation. Current LIVE inputs have no prior approval, so operational drafts are rejected. Neither RPC has approval, sending, provisioning or account authority.

Status requests have a three-second deadline; explanations have a ninety-second ceiling. The Python worker cancels its async SDK task when its caller disconnects or its deadline expires. Strands retries are disabled to avoid invisible repeated requests. Tests exercise real loopback gRPC with a stub generator; these verify transport behavior, not model quality.

The local worker binds 127.0.0.1:8089. Use TLS and authenticated service identities before remote deployment; the current prototype has a demo manager and no account login.

Generated stubs are committed. Regeneration uses the pinned Python `grpcio-tools` and the Go Protobuf generators installed in the local hackathon toolchain. Keep the contract and both generated clients in the same change.
