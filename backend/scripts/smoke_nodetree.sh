#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@fpg.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-Admin123!}"
SMOKE_SUFFIX="${SMOKE_SUFFIX:-$(date +%s)-$$}"
MEMBER_EMAIL="${MEMBER_EMAIL:-member+${SMOKE_SUFFIX}@test.com}"
MEMBER_PASSWORD="${MEMBER_PASSWORD:-Member123!}"
MEMBER_NAME="${MEMBER_NAME:-MemberSmoke}"

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

json_check() {
  local json="$1"
  local expr="$2"
  JSON_INPUT="$json" JSON_EXPR="$expr" node -e '
const input = process.env.JSON_INPUT;
const expr = process.env.JSON_EXPR;
if (!input || !expr) {
  process.exit(1);
}

const root = JSON.parse(input);
const data = root.data;
const error = root.error;
const success = root.success;
let result = false;

try {
  result = Function("root", "data", "error", "success", `return (${expr});`)(root, data, error, success);
} catch {
  process.exit(1);
}

if (!result) {
  process.exit(1);
}
'
}

assert_status_and_check() {
  local label="$1"
  local expected_status="$2"
  local actual_status="$3"
  local body="$4"
  local expr="$5"

  if [[ "$actual_status" == "$expected_status" ]] && json_check "$body" "$expr"; then
    PASS=$((PASS + 1))
    echo "[PASS] $label"
  else
    FAIL=$((FAIL + 1))
    echo "[FAIL] $label -> status=$actual_status expr=$expr"
  fi
}

echo "=== NodeTree Smoke Test ==="

PING=$(request GET "/ping")
split_response "$PING"
if [[ "$RESPONSE_STATUS" != "200" ]]; then
  echo "Ping failed: expected 200 got $RESPONSE_STATUS"
  exit 1
fi
echo "[READY] GET /ping -> 200"

ADMIN_LOGIN=$(request POST "/api/auth/login" "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")
split_response "$ADMIN_LOGIN"
assert_status "admin login" 200 "$RESPONSE_STATUS"
ADMIN_ACCESS=$(extract_json "$RESPONSE_BODY" '.data.access_token')

MEMBER_REGISTER=$(request POST "/api/auth/register" "{\"email\":\"$MEMBER_EMAIL\",\"password\":\"$MEMBER_PASSWORD\",\"display_name\":\"$MEMBER_NAME\"}")
split_response "$MEMBER_REGISTER"
assert_status "register member" 201 "$RESPONSE_STATUS"
MEMBER_ACCESS=$(extract_json "$RESPONSE_BODY" '.data.access_token')

MEMBER_ME=$(request GET "/api/me" "" "$MEMBER_ACCESS")
split_response "$MEMBER_ME"
assert_status_and_check \
  "get member me" \
  200 \
  "$RESPONSE_STATUS" \
  "$RESPONSE_BODY" \
  "data && data.email === '$MEMBER_EMAIL' && data.display_name === '$MEMBER_NAME' && data.role === 'member'"
MEMBER_ID=$(extract_json "$RESPONSE_BODY" '.data.id')

ROOT_FOLDER_TITLE="Product Docs ${SMOKE_SUFFIX}"
ROOT_FOLDER_CREATE=$(request POST "/api/nodes" "{\"kind\":\"folder\",\"title\":\"$ROOT_FOLDER_TITLE\"}" "$ADMIN_ACCESS")
split_response "$ROOT_FOLDER_CREATE"
assert_status "T01 create root folder" 201 "$RESPONSE_STATUS"
ROOT_FOLDER_ID=$(extract_json "$RESPONSE_BODY" '.data.id')

DOC_TITLE="Requirements ${SMOKE_SUFFIX}"
DOC_CREATE=$(request POST "/api/nodes" "{\"parent_id\":\"$ROOT_FOLDER_ID\",\"kind\":\"doc\",\"title\":\"$DOC_TITLE\"}" "$ADMIN_ACCESS")
split_response "$DOC_CREATE"
assert_status "T02 create child doc" 201 "$RESPONSE_STATUS"
DOC_ID=$(extract_json "$RESPONSE_BODY" '.data.id')

ROOT_LIST=$(request GET "/api/nodes?parent=null" "" "$ADMIN_ACCESS")
split_response "$ROOT_LIST"
assert_status_and_check \
  "T03 list root children" \
  200 \
  "$RESPONSE_STATUS" \
  "$RESPONSE_BODY" \
  "Array.isArray(data) && data.some((item) => item.id === '$ROOT_FOLDER_ID' && item.kind === 'folder')"

FOLDER_GET=$(request GET "/api/nodes/$ROOT_FOLDER_ID" "" "$ADMIN_ACCESS")
split_response "$FOLDER_GET"
assert_status_and_check \
  "T04 get node" \
  200 \
  "$RESPONSE_STATUS" \
  "$RESPONSE_BODY" \
  "data && data.id === '$ROOT_FOLDER_ID' && data.title === '$ROOT_FOLDER_TITLE' && data.permission === 'manage'"

RENAMED_DOC_TITLE="PRD ${SMOKE_SUFFIX}"
DOC_RENAME=$(request PATCH "/api/nodes/$DOC_ID" "{\"title\":\"$RENAMED_DOC_TITLE\"}" "$ADMIN_ACCESS")
split_response "$DOC_RENAME"
assert_status_and_check \
  "T05 rename node" \
  200 \
  "$RESPONSE_STATUS" \
  "$RESPONSE_BODY" \
  "data && data.id === '$DOC_ID' && data.title === '$RENAMED_DOC_TITLE'"

SECOND_FOLDER_CREATE=$(request POST "/api/nodes" "{\"kind\":\"folder\",\"title\":\"Archive ${SMOKE_SUFFIX}\"}" "$ADMIN_ACCESS")
split_response "$SECOND_FOLDER_CREATE"
assert_status "T06a create second folder" 201 "$RESPONSE_STATUS"
SECOND_FOLDER_ID=$(extract_json "$RESPONSE_BODY" '.data.id')

DOC_MOVE=$(request PATCH "/api/nodes/$DOC_ID" "{\"parent_id\":\"$SECOND_FOLDER_ID\"}" "$ADMIN_ACCESS")
split_response "$DOC_MOVE"
assert_status_and_check \
  "T06b move doc to second folder" \
  200 \
  "$RESPONSE_STATUS" \
  "$RESPONSE_BODY" \
  "data && data.id === '$DOC_ID' && data.parent_id === '$SECOND_FOLDER_ID'"

SET_READABLE_PERMS=$(request PUT "/api/nodes/$ROOT_FOLDER_ID/permissions" "{\"permissions\":[{\"subject_type\":\"user\",\"subject_id\":\"$MEMBER_ID\",\"level\":\"readable\"}]}" "$ADMIN_ACCESS")
split_response "$SET_READABLE_PERMS"
assert_status "T07 set permissions (readable)" 200 "$RESPONSE_STATUS"

GET_ROOT_PERMS=$(request GET "/api/nodes/$ROOT_FOLDER_ID/permissions" "" "$ADMIN_ACCESS")
split_response "$GET_ROOT_PERMS"
assert_status_and_check \
  "T08 get permissions" \
  200 \
  "$RESPONSE_STATUS" \
  "$RESPONSE_BODY" \
  "data && data.node_id === '$ROOT_FOLDER_ID' && Array.isArray(data.permissions) && data.permissions.some((item) => item.subject_type === 'user' && item.subject_id === '$MEMBER_ID' && item.level === 'readable')"

SET_NONE_PERMS=$(request PUT "/api/nodes/$SECOND_FOLDER_ID/permissions" "{\"permissions\":[{\"subject_type\":\"user\",\"subject_id\":\"$MEMBER_ID\",\"level\":\"none\"}]}" "$ADMIN_ACCESS")
split_response "$SET_NONE_PERMS"
assert_status "T09 set permissions (none)" 200 "$RESPONSE_STATUS"

MEMBER_GET_NONE_FOLDER=$(request GET "/api/nodes/$SECOND_FOLDER_ID" "" "$MEMBER_ACCESS")
split_response "$MEMBER_GET_NONE_FOLDER"
assert_status "T10 none user gets 404" 404 "$RESPONSE_STATUS"

MEMBER_GET_READABLE_FOLDER=$(request GET "/api/nodes/$ROOT_FOLDER_ID" "" "$MEMBER_ACCESS")
split_response "$MEMBER_GET_READABLE_FOLDER"
assert_status_and_check \
  "T11a readable user can view" \
  200 \
  "$RESPONSE_STATUS" \
  "$RESPONSE_BODY" \
  "data && data.id === '$ROOT_FOLDER_ID' && data.permission === 'readable'"

MEMBER_CREATE_CHILD=$(request POST "/api/nodes" "{\"parent_id\":\"$ROOT_FOLDER_ID\",\"kind\":\"doc\",\"title\":\"Should Fail ${SMOKE_SUFFIX}\"}" "$MEMBER_ACCESS")
split_response "$MEMBER_CREATE_CHILD"
assert_status "T11b readable user cannot create child" 403 "$RESPONSE_STATUS"

ROOT_FOLDER_DELETE=$(request DELETE "/api/nodes/$ROOT_FOLDER_ID" "" "$ADMIN_ACCESS")
split_response "$ROOT_FOLDER_DELETE"
assert_status "T12 soft delete folder" 200 "$RESPONSE_STATUS"

GET_DELETED_FOLDER=$(request GET "/api/nodes/$ROOT_FOLDER_ID" "" "$ADMIN_ACCESS")
split_response "$GET_DELETED_FOLDER"
assert_status "T12b deleted node returns 404" 404 "$RESPONSE_STATUS"

RESTORE_FOLDER=$(request POST "/api/nodes/$ROOT_FOLDER_ID/restore" "" "$ADMIN_ACCESS")
split_response "$RESTORE_FOLDER"
assert_status "T13 restore folder" 200 "$RESPONSE_STATUS"

GET_RESTORED_FOLDER=$(request GET "/api/nodes/$ROOT_FOLDER_ID" "" "$ADMIN_ACCESS")
split_response "$GET_RESTORED_FOLDER"
assert_status_and_check \
  "T13b restored node returns 200" \
  200 \
  "$RESPONSE_STATUS" \
  "$RESPONSE_BODY" \
  "data && data.id === '$ROOT_FOLDER_ID' && data.title === '$ROOT_FOLDER_TITLE'"

ADMIN_ME=$(request GET "/api/me" "" "$ADMIN_ACCESS")
split_response "$ADMIN_ME"
if [[ "$RESPONSE_STATUS" != "200" ]]; then
  echo "Admin me lookup failed: expected 200 got $RESPONSE_STATUS"
  exit 1
fi
ADMIN_ID=$(extract_json "$RESPONSE_BODY" '.data.id')

GROUP_NAME="design-group-${SMOKE_SUFFIX}"
GROUP_CREATE=$(request POST "/api/admin/groups" "{\"name\":\"$GROUP_NAME\",\"leader_id\":\"$ADMIN_ID\"}" "$ADMIN_ACCESS")
split_response "$GROUP_CREATE"
assert_status "T14 create group" 201 "$RESPONSE_STATUS"
GROUP_ID=$(extract_json "$RESPONSE_BODY" '.data.id')

GROUP_UPDATE=$(request PATCH "/api/admin/groups/$GROUP_ID" "{\"name\":\"design-team-${SMOKE_SUFFIX}\"}" "$ADMIN_ACCESS")
split_response "$GROUP_UPDATE"
assert_status_and_check \
  "T15 rename group" \
  200 \
  "$RESPONSE_STATUS" \
  "$RESPONSE_BODY" \
  "data && data.id === '$GROUP_ID' && data.name === 'design-team-${SMOKE_SUFFIX}'"

GROUP_LIST_MEMBERS=$(request GET "/api/admin/groups/$GROUP_ID/members" "" "$ADMIN_ACCESS")
split_response "$GROUP_LIST_MEMBERS"
assert_status_and_check \
  "T16 list group members" \
  200 \
  "$RESPONSE_STATUS" \
  "$RESPONSE_BODY" \
  "Array.isArray(data) && data.some((item) => item.user_id === '$ADMIN_ID')"

GROUP_ADD_MEMBER=$(request POST "/api/admin/groups/$GROUP_ID/members" "{\"user_id\":\"$MEMBER_ID\"}" "$ADMIN_ACCESS")
split_response "$GROUP_ADD_MEMBER"
assert_status "T17 add member to group" 200 "$RESPONSE_STATUS"

GROUP_REMOVE_MEMBER=$(request DELETE "/api/admin/groups/$GROUP_ID/members/$MEMBER_ID" "" "$ADMIN_ACCESS")
split_response "$GROUP_REMOVE_MEMBER"
assert_status "T18 remove member from group" 200 "$RESPONSE_STATUS"

GROUP_DELETE=$(request DELETE "/api/admin/groups/$GROUP_ID" "" "$ADMIN_ACCESS")
split_response "$GROUP_DELETE"
assert_status "T19 delete group" 200 "$RESPONSE_STATUS"

PROMOTE_MEMBER=$(request PATCH "/api/admin/users/$MEMBER_ID/role" "{\"role\":\"manager\"}" "$ADMIN_ACCESS")
split_response "$PROMOTE_MEMBER"
assert_status_and_check \
  "T20 promote to manager" \
  200 \
  "$RESPONSE_STATUS" \
  "$RESPONSE_BODY" \
  "data && data.id === '$MEMBER_ID' && data.role === 'manager'"

DEMOTE_MEMBER=$(request PATCH "/api/admin/users/$MEMBER_ID/role" "{\"role\":\"member\"}" "$ADMIN_ACCESS")
split_response "$DEMOTE_MEMBER"
assert_status_and_check \
  "T21 demote to member" \
  200 \
  "$RESPONSE_STATUS" \
  "$RESPONSE_BODY" \
  "data && data.id === '$MEMBER_ID' && data.role === 'member'"

echo ""
echo "=== RESULT: $PASS passed, $FAIL failed ==="
if [[ "$FAIL" -eq 0 && "$PASS" -eq 28 ]]; then
  exit 0
fi
exit 1
