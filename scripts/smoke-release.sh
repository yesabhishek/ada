#!/usr/bin/env bash
set -euo pipefail

BINARY="${1:-./bin/ada}"
case "${BINARY}" in
  /*) ;;
  *) BINARY="$(cd "$(dirname "${BINARY}")" && pwd)/$(basename "${BINARY}")" ;;
esac
TMP_DIR="$(mktemp -d)"
PORT="${ADA_SMOKE_PORT:-4197}"
UI_PID=""

cleanup() {
  if [ -n "${UI_PID}" ]; then
    kill "${UI_PID}" >/dev/null 2>&1 || true
  fi
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

REPO_DIR="${TMP_DIR}/repo"
mkdir -p "${REPO_DIR}"
cd "${REPO_DIR}"

git init >/dev/null
git config user.name "Ada Smoke"
git config user.email "ada-smoke@example.com"

cat > main.go <<'EOF'
package main

func add(a int, b int) int {
	return a + b
}
EOF

git add main.go
git commit -m "initial" >/dev/null

"${BINARY}" version --json | grep -q '"version"'
"${BINARY}" doctor --json | grep -q '"overall"'
"${BINARY}" start . >/dev/null
"${BINARY}" sync >/dev/null
"${BINARY}" status --json | grep -q '"workspace"'

cat > main.go <<'EOF'
package main

func add(a int, b int) int {
	return a - b
}
EOF

"${BINARY}" diff --semantic --json | grep -q '"mode": "semantic"'
"${BINARY}" diff --text --json | grep -q '"mode": "text"'
"${BINARY}" eval --json | grep -q '"results"'

"${BINARY}" ui --host 127.0.0.1 --port "${PORT}" >/tmp/ada-ui-smoke.log 2>&1 &
UI_PID="$!"
sleep 2

curl -fsS "http://127.0.0.1:${PORT}/" >/dev/null
curl -fsS "http://127.0.0.1:${PORT}/api/dashboard" | grep -q '"workspace"'

echo "Smoke test passed"
