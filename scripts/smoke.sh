#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
JWT_SECRET="${JWT_SECRET:-smoke_dev_secret}"
PING_URL="${PING_URL:-http://127.0.0.1:8080/ping}"
DOC_ID="${DOC_ID:-11111111-1111-1111-1111-111111111111}"
WS_URL="${WS_URL:-ws://127.0.0.1:8080/api/ws?doc=${DOC_ID}}"

started=0

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "[smoke] missing required command: $1" >&2
    exit 1
  fi
}

cleanup() {
  if [[ "$started" -eq 1 ]]; then
    echo "[smoke] docker compose down"
    (cd "${ROOT_DIR}" && JWT_SECRET="${JWT_SECRET}" docker compose down)
  fi
}

trap cleanup EXIT

require_cmd docker
require_cmd curl
require_cmd node

cd "${ROOT_DIR}"

echo "[smoke] docker compose up --build -d"
JWT_SECRET="${JWT_SECRET}" docker compose up --build -d
started=1

echo "[smoke] waiting for /ping ..."
ready=0
for _ in $(seq 1 60); do
  if curl -fsS "${PING_URL}" | grep -q '"success":true'; then
    ready=1
    break
  fi
  sleep 2
done

if [[ "${ready}" -ne 1 ]]; then
  echo "[smoke] /ping is not ready in time" >&2
  exit 1
fi

echo "[smoke] verifying websocket placeholder response ..."
node - "${WS_URL}" <<'NODE'
const wsUrl = process.argv[2];
const ws = new WebSocket(wsUrl, ["access.dummy"]);

let done = false;
const timeout = setTimeout(() => fail("timeout"), 10000);

function finish(code, message) {
  if (done) return;
  done = true;
  clearTimeout(timeout);
  try { ws.close(1000, "done"); } catch {}
  if (code === 0) {
    console.log(`[smoke] ${message}`);
  } else {
    console.error(`[smoke] ${message}`);
  }
  process.exit(code);
}

function fail(reason) {
  finish(1, `websocket failed: ${reason}`);
}

ws.onerror = () => fail("connection error");

ws.onopen = () => {
  ws.send(JSON.stringify({ type: "probe" }));
};

ws.onmessage = (event) => {
  let payload;
  try {
    payload = JSON.parse(String(event.data));
  } catch {
    fail("non-json payload");
    return;
  }

  if (payload && payload.type === "not_implemented") {
    finish(0, "websocket received not_implemented");
    return;
  }

  fail(`unexpected payload: ${JSON.stringify(payload)}`);
};

ws.onclose = (event) => {
  if (!done) {
    fail(`closed early (code=${event.code})`);
  }
};
NODE

echo "[smoke] done"
