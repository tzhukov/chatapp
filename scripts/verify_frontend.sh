#!/usr/bin/env bash
set -euo pipefail

# Simple post-Tilt verification script for frontend + backend ingress.
# Requirements:
#  - ingress.local mapped to 127.0.0.1 in /etc/hosts
#  - Tilt environment running (tilt up)
#  - curl, jq present
#  - Using self-signed / custom CA -> we pass -k to curl; adjust if you have trusted certs

BASE_HOST="https://ingress.local"
API="$BASE_HOST/api"
DEX="$BASE_HOST/dex"
FRONTEND="$BASE_HOST"

pass() { echo -e "[PASS] $1"; }
fail() { echo -e "[FAIL] $1" >&2; FAILED=1; }

FAILED=0

echo "== Frontend / OIDC / API Smoke Verification =="

# 1. Frontend root
HTML=$(curl -ks -w '\n%{http_code}' "$FRONTEND/")
STATUS=${HTML##*$'\n'}
BODY=${HTML%$'\n'*}
if [[ "$STATUS" == "200" && "$BODY" == *"<div id=\"app\"></div>"* ]]; then
  pass "Frontend root served (200 + app div)"
else
  fail "Frontend root unexpected (status=$STATUS)"
fi

# 2. Runtime config
CFG=$(curl -ks -w '\n%{http_code}' "$FRONTEND/config.js")
CFG_STATUS=${CFG##*$'\n'}
CFG_BODY=${CFG%$'\n'*}
if [[ "$CFG_STATUS" == "200" && "$CFG_BODY" == *"VUE_APP_DEX_ISSUER_URL"* ]]; then
  pass "Runtime config.js present"
else
  fail "config.js missing or malformed (status=$CFG_STATUS)"
fi

# 3. Dex discovery (/.well-known/openid-configuration)
DISC=$(curl -ks -w '\n%{http_code}' "$DEX/.well-known/openid-configuration")
DISC_STATUS=${DISC##*$'\n'}
DISC_BODY=${DISC%$'\n'*}
if [[ "$DISC_STATUS" == "200" && "$DISC_BODY" == *"authorization_endpoint"* ]]; then
  pass "Dex discovery OK"
else
  fail "Dex discovery failed (status=$DISC_STATUS)"
fi

# 4. API unauthenticated (should be 401)
API_RESP_CODE=$(curl -k -s -o /dev/null -w '%{http_code}' "$API/messages")
if [[ "$API_RESP_CODE" == "401" ]]; then
  pass "API /messages rejects unauthenticated with 401"
else
  fail "API /messages expected 401 got $API_RESP_CODE"
fi

# 5. WebSocket endpoint reachability (handshake should fail auth -> 401 in upgrade response or close)
# We'll attempt a websocket handshake using curl -- no full ws client in script.
WS_CODE=$(curl -k -s -o /dev/null -w '%{http_code}' -H 'Connection: Upgrade' -H 'Upgrade: websocket' -H 'Sec-WebSocket-Key: dummykey1234567890==' -H 'Sec-WebSocket-Version: 13' "$BASE_HOST/api/ws") || true
if [[ "$WS_CODE" == "401" || "$WS_CODE" == "400" || "$WS_CODE" == "426" ]]; then
  pass "WebSocket endpoint (/api/ws) responds (code $WS_CODE) without token"
else
  fail "WebSocket unexpected code $WS_CODE for /api/ws"
fi

# Summary
if [[ $FAILED -eq 0 ]]; then
  echo "All checks passed."
else
  echo "One or more checks failed." >&2
  exit 1
fi
