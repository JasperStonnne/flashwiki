#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
SMOKE_EMAIL="${SMOKE_EMAIL:-smoke+$(date +%s)@test.com}"
SMOKE_PASSWORD="${SMOKE_PASSWORD:-Smoke123!}"
SMOKE_NEW_PASSWORD="${SMOKE_NEW_PASSWORD:-NewSmoke1!}"
SMOKE_NAME="${SMOKE_NAME:-Smoke}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@fpg.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-Admin123!}"

PASS=0
FAIL=0
RESPONSE_BODY=""
RESPONSE_STATUS=""

split_response() {
  local response="$1"
  RESPONSE_STATUS="${response##*$'\n'}"
  RESPONSE_BODY="${response%$'\n'*}"
}

extract_json() {
  local json="$1"
  local expr="$2"
  if command -v jq >/dev/null 2>&1; then
    jq -er "$expr" <<<"$json"
    return
  fi

  JSON_INPUT="$json" JSON_EXPR="$expr" node -e '
const input = process.env.JSON_INPUT;
const expr = process.env.JSON_EXPR;
if (!input || !expr || !expr.startsWith(".")) {
  process.exit(1);
}

let current = JSON.parse(input);
for (const rawPart of expr.slice(1).split(".")) {
  const part = rawPart.trim();
  if (!part) continue;
  if (current === null || current === undefined || !(part in current)) {
    process.exit(1);
  }
  current = current[part];
}

if (current === null || current === undefined) {
  process.exit(1);
}
process.stdout.write(String(current));
'
}

assert_status() {
  local label="$1"
  local expected="$2"
  local actual="$3"
  if [[ "$actual" == "$expected" ]]; then
    PASS=$((PASS + 1))
    echo "[PASS] $label -> $actual"
  else
    FAIL=$((FAIL + 1))
    echo "[FAIL] $label -> expected $expected got $actual"
  fi
}

request() {
  local method="$1"
  local path="$2"
  local body="${3-}"
  local token="${4-}"
  local args=(-sS -w $'\n%{http_code}' -X "$method" "$BASE$path")

  if [[ -n "$token" ]]; then
    args+=(-H "Authorization: Bearer $token")
  fi
  if [[ -n "$body" ]]; then
    args+=(-H "Content-Type: application/json" -d "$body")
  fi

  curl "${args[@]}"
}

echo "=== Auth Smoke Test ==="

# readiness check (not counted in PASS/FAIL summary)
PING=$(request GET "/ping")
split_response "$PING"
if [[ "$RESPONSE_STATUS" != "200" ]]; then
  echo "Ping failed: expected 200 got $RESPONSE_STATUS"
  exit 1
fi
echo "[READY] GET /ping -> 200"

REG=$(request POST "/api/auth/register" "{\"email\":\"$SMOKE_EMAIL\",\"password\":\"$SMOKE_PASSWORD\",\"display_name\":\"$SMOKE_NAME\"}")
split_response "$REG"
assert_status "register" 201 "$RESPONSE_STATUS"
ACCESS=$(extract_json "$RESPONSE_BODY" '.data.access_token')
REFRESH=$(extract_json "$RESPONSE_BODY" '.data.refresh_token')

ME=$(request GET "/api/me" "" "$ACCESS")
split_response "$ME"
assert_status "get me with access" 200 "$RESPONSE_STATUS"

DUP=$(request POST "/api/auth/register" "{\"email\":\"$SMOKE_EMAIL\",\"password\":\"$SMOKE_PASSWORD\",\"display_name\":\"$SMOKE_NAME\"}")
split_response "$DUP"
assert_status "duplicate register" 409 "$RESPONSE_STATUS"

REF=$(request POST "/api/auth/refresh" "{\"refresh_token\":\"$REFRESH\"}")
split_response "$REF"
assert_status "refresh" 200 "$RESPONSE_STATUS"
ACCESS2=$(extract_json "$RESPONSE_BODY" '.data.access_token')
REFRESH2=$(extract_json "$RESPONSE_BODY" '.data.refresh_token')

REUSE=$(request POST "/api/auth/refresh" "{\"refresh_token\":\"$REFRESH\"}")
split_response "$REUSE"
assert_status "reuse old refresh" 401 "$RESPONSE_STATUS"

LOGIN=$(request POST "/api/auth/login" "{\"email\":\"$SMOKE_EMAIL\",\"password\":\"$SMOKE_PASSWORD\"}")
split_response "$LOGIN"
assert_status "login after reuse" 200 "$RESPONSE_STATUS"
ACCESS3=$(extract_json "$RESPONSE_BODY" '.data.access_token')
REFRESH3=$(extract_json "$RESPONSE_BODY" '.data.refresh_token')
if ! SMOKE_USER_ID=$(extract_json "$RESPONSE_BODY" '.data.user.id' 2>/dev/null); then
  SMOKE_USER_ID=$(extract_json "$RESPONSE_BODY" '.data.user.ID')
fi

CHPW=$(request POST "/api/me/password" "{\"old_password\":\"$SMOKE_PASSWORD\",\"new_password\":\"$SMOKE_NEW_PASSWORD\"}" "$ACCESS3")
split_response "$CHPW"
assert_status "change password" 200 "$RESPONSE_STATUS"
ACCESS4=$(extract_json "$RESPONSE_BODY" '.data.access_token')
REFRESH4=$(extract_json "$RESPONSE_BODY" '.data.refresh_token')

OLD_ME=$(request GET "/api/me" "" "$ACCESS3")
split_response "$OLD_ME"
assert_status "old access invalid after change password" 401 "$RESPONSE_STATUS"

ADMIN_LOGIN=$(request POST "/api/auth/login" "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")
split_response "$ADMIN_LOGIN"
assert_status "admin login" 200 "$RESPONSE_STATUS"
ADMIN_ACCESS=$(extract_json "$RESPONSE_BODY" '.data.access_token')

KICK=$(request POST "/api/admin/users/$SMOKE_USER_ID/force-logout" "" "$ADMIN_ACCESS")
split_response "$KICK"
assert_status "force logout target user" 200 "$RESPONSE_STATUS"

KICKED_ME=$(request GET "/api/me" "" "$ACCESS4")
split_response "$KICKED_ME"
assert_status "kicked access invalid" 401 "$RESPONSE_STATUS"

KICKED_REF=$(request POST "/api/auth/refresh" "{\"refresh_token\":\"$REFRESH4\"}")
split_response "$KICKED_REF"
assert_status "kicked refresh invalid" 401 "$RESPONSE_STATUS"

RELOGIN=$(request POST "/api/auth/login" "{\"email\":\"$SMOKE_EMAIL\",\"password\":\"$SMOKE_NEW_PASSWORD\"}")
split_response "$RELOGIN"
assert_status "re-login with new password" 200 "$RESPONSE_STATUS"
FINAL_ACCESS=$(extract_json "$RESPONSE_BODY" '.data.access_token')
FINAL_REFRESH=$(extract_json "$RESPONSE_BODY" '.data.refresh_token')

LOGOUT=$(request POST "/api/auth/logout" "{\"refresh_token\":\"$FINAL_REFRESH\"}" "$FINAL_ACCESS")
split_response "$LOGOUT"
assert_status "logout" 200 "$RESPONSE_STATUS"

POST_LOGOUT_REF=$(request POST "/api/auth/refresh" "{\"refresh_token\":\"$FINAL_REFRESH\"}")
split_response "$POST_LOGOUT_REF"
assert_status "refresh after logout invalid" 401 "$RESPONSE_STATUS"

echo ""
echo "=== RESULT: $PASS passed, $FAIL failed ==="
if [[ "$FAIL" -eq 0 && "$PASS" -eq 15 ]]; then
  exit 0
fi
exit 1
