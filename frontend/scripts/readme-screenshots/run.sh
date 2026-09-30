#!/usr/bin/env bash
# Regenerates the README screenshots in docs/screenshots/.
#
#   frontend/scripts/readme-screenshots/run.sh
#
# Builds the frontend + single binary (skip with FENSTER_BIN=<path>), starts a
# throwaway server with an in-memory DB, seeds demo data, captures the PNGs and
# stops the server. Env: PORT (default 3471), FENSTER_BIN.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../../.." && pwd)"
PORT="${PORT:-3471}"
TMP="$(mktemp -d)"
PID=""
cleanup() {
	[ -n "$PID" ] && kill "$PID" 2>/dev/null || true
	rm -rf "$TMP"
}
trap cleanup EXIT

BIN="${FENSTER_BIN:-}"
if [ -z "$BIN" ]; then
	(cd "$ROOT/frontend" && pnpm build)
	(cd "$ROOT" && go build -o "$TMP/fenster" .)
	BIN="$TMP/fenster"
fi

VIKUNJA_SERVICE_INTERFACE="127.0.0.1:$PORT" \
VIKUNJA_SERVICE_PUBLICURL="http://127.0.0.1:$PORT/" \
VIKUNJA_SERVICE_ROOTPATH="$TMP" \
VIKUNJA_SERVICE_JWTSECRET=screenshots \
VIKUNJA_DATABASE_TYPE=sqlite \
VIKUNJA_DATABASE_PATH=memory \
VIKUNJA_FILES_BASEPATH="$TMP/files" \
VIKUNJA_LOG_LEVEL=ERROR \
VIKUNJA_MAILER_ENABLED=false \
VIKUNJA_RATELIMIT_NOAUTHLIMIT=10000 \
VIKUNJA_SERVICE_ENABLEREGISTRATION=true \
	"$BIN" web >"$TMP/server.log" 2>&1 &
PID=$!

for _ in $(seq 1 60); do
	curl -fs "http://127.0.0.1:$PORT/api/v1/info" >/dev/null && break
	sleep 1
done
curl -fs "http://127.0.0.1:$PORT/api/v1/info" >/dev/null || { cat "$TMP/server.log"; exit 1; }

export BASE_URL="http://127.0.0.1:$PORT" STATE_FILE="$TMP/state.json" OUT_DIR="$ROOT/docs/screenshots"
cd "$ROOT/frontend"
node scripts/readme-screenshots/seed.mjs
node scripts/readme-screenshots/capture.mjs

if command -v magick >/dev/null; then
	for f in "$OUT_DIR"/*.png; do magick "$f" -strip -define png:compression-level=9 "$f"; done
fi
echo "Screenshots written to $OUT_DIR"
