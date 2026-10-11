#!/usr/bin/env bash
set -euo pipefail
TASK_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export WEATHER_DESK_ROOT="$TASK_ROOT"
export WEATHER_DESK_DEMO_ORIGIN="${WEATHER_DESK_DEMO_ORIGIN:-http://127.0.0.1:8088}"
export GOCACHE="$TASK_ROOT/.cache/go-build"
export GOPATH="$TASK_ROOT/../.tools/go-modules"
GO_BIN="$TASK_ROOT/../.tools/go/bin/go"
if [ ! -x "$GO_BIN" ]; then GO_BIN="$(command -v go)"; fi
mkdir -p "$TASK_ROOT/.cache"
PYTHON_BIN="${WEATHER_DESK_PYTHON:-$TASK_ROOT/../.venv/bin/python}"
ENV_LAUNCHER="$TASK_ROOT/scripts/run-with-env.py"
cd "$TASK_ROOT/go-prototype"
"$GO_BIN" build -o "$TASK_ROOT/.cache/weatherdesk" ./cmd/weatherdesk
cd "$TASK_ROOT"
"$PYTHON_BIN" "$ENV_LAUNCHER" "$PYTHON_BIN" -m agent_service.server &
AGENT_PID=$!
"$PYTHON_BIN" "$ENV_LAUNCHER" "$TASK_ROOT/.cache/weatherdesk" &
API_PID=$!
trap 'kill "$API_PID" "$AGENT_PID" 2>/dev/null || true' EXIT INT TERM
cd "$TASK_ROOT/frontend"
"$PYTHON_BIN" "$ENV_LAUNCHER" npm run dev
