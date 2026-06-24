#!/usr/bin/env sh
set -eu

API_BASE_URL=${API_BASE_URL:-http://localhost:8080}
FRONTEND_URL=${FRONTEND_URL:-http://localhost:8081}
ACCOUNT=${ACCOUNT:-${BOOTSTRAP_ADMIN_ACCOUNT:-admin}}
PASSWORD=${PASSWORD:-${BOOTSTRAP_ADMIN_PASSWORD:-admin123}}
TIMEOUT_SEC=${TIMEOUT_SEC:-10}
RETRY_COUNT=${RETRY_COUNT:-30}
RETRY_DELAY_SEC=${RETRY_DELAY_SEC:-2}
EXERCISE_WRITES=${EXERCISE_WRITES:-false}

API_BASE_URL=${API_BASE_URL%/}
FRONTEND_URL=${FRONTEND_URL%/}

fail() {
  echo "smoke failed: $*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

retry() {
  name=$1
  shift
  attempt=1
  while :; do
    output=$("$@" 2>&1) && {
      printf '%s' "$output"
      return 0
    }
    if [ "$attempt" -ge "$RETRY_COUNT" ]; then
      fail "$name failed after $attempt attempt(s): $output"
    fi
    echo "Waiting for $name (attempt $attempt/$RETRY_COUNT)..." >&2
    sleep "$RETRY_DELAY_SEC"
    attempt=$((attempt + 1))
  done
}

api_get() {
  path=$1
  token=${2:-}
  if [ -n "$token" ]; then
    curl -fsS --max-time "$TIMEOUT_SEC" -H "Authorization: $token" "$API_BASE_URL$path"
  else
    curl -fsS --max-time "$TIMEOUT_SEC" "$API_BASE_URL$path"
  fi
}

api_post() {
  path=$1
  body=$2
  token=${3:-}
  if [ -n "$token" ]; then
    curl -fsS --max-time "$TIMEOUT_SEC" -X POST -H "Authorization: $token" -H "Content-Type: application/json" -d "$body" "$API_BASE_URL$path"
  else
    curl -fsS --max-time "$TIMEOUT_SEC" -X POST -H "Content-Type: application/json" -d "$body" "$API_BASE_URL$path"
  fi
}

api_delete() {
  path=$1
  token=${2:-}
  if [ -n "$token" ]; then
    curl -fsS --max-time "$TIMEOUT_SEC" -X DELETE -H "Authorization: $token" "$API_BASE_URL$path"
  else
    curl -fsS --max-time "$TIMEOUT_SEC" -X DELETE "$API_BASE_URL$path"
  fi
}

assert_api_ok() {
  name=$1
  payload=$2
  echo "$payload" | grep -q '"code"[[:space:]]*:[[:space:]]*0' || fail "$name did not return code=0: $payload"
}

json_string() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

json_field() {
  field=$1
  payload=$2
  printf '%s' "$payload" | sed -n 's/.*"'"$field"'"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}

json_number() {
  field=$1
  payload=$2
  printf '%s' "$payload" | sed -n 's/.*"'"$field"'"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p' | head -n 1
}

assert_has_field() {
  name=$1
  field=$2
  payload=$3
  echo "$payload" | grep -q '"'"$field"'"[[:space:]]*:' || fail "$name missing field: $field"
}

assert_page() {
  name=$1
  payload=$2
  assert_has_field "$name" "items" "$payload"
  assert_has_field "$name" "total" "$payload"
  assert_has_field "$name" "page" "$payload"
  assert_has_field "$name" "perPage" "$payload"
}

require_command curl
require_command date
require_command grep
require_command sed
require_command sleep

echo "==> Backend liveness"
health=$(retry "backend liveness" api_get "/healthz")
assert_api_ok "healthz" "$health"

echo "==> Backend readiness"
ready=$(retry "backend readiness" api_get "/readyz")
assert_api_ok "readyz" "$ready"

echo "==> Auth login"
login_body='{"account":"'"$(json_string "$ACCOUNT")"'","password":"'"$(json_string "$PASSWORD")"'"}'
login=$(retry "auth login" api_post "/api/v1/auth/login" "$login_body")
assert_api_ok "login" "$login"
token=$(json_field "token" "$login")
[ -n "$token" ] || fail "login did not return a token"
auth_token="Bearer $token"

echo "==> Auth session"
me=$(api_get "/api/v1/auth/me" "$auth_token")
assert_api_ok "me" "$me"
account=$(json_field "account" "$me")
role=$(json_field "role" "$me")
[ "$account" = "$ACCOUNT" ] || fail "authenticated account mismatch: expected $ACCOUNT got $account"
[ "$role" = "admin" ] || fail "smoke account must be admin, got role=$role"

echo "==> Dashboard"
dashboard=$(api_get "/api/v1/dashboard" "$auth_token")
assert_api_ok "dashboard" "$dashboard"
for field in users classes orders pending flashOrders flashPending queueOrders queueRefreshes queueSubmit queueSubmitFlash queueRefresh queueRefreshFlash activeUsers onlineClasses; do
  assert_has_field "dashboard" "$field" "$dashboard"
done

echo "==> Settings"
settings=$(api_get "/api/v1/settings" "$auth_token")
assert_api_ok "settings" "$settings"

echo "==> Orders and logs"
orders=$(api_get "/api/v1/orders?perPage=5" "$auth_token")
assert_api_ok "orders" "$orders"
assert_page "orders" "$orders"
flash_orders=$(api_get "/api/v1/orders?flashMode=true&perPage=5" "$auth_token")
assert_api_ok "flash orders" "$flash_orders"
assert_page "flash orders" "$flash_orders"
normal_orders=$(api_get "/api/v1/orders?flashMode=false&perPage=5" "$auth_token")
assert_api_ok "normal orders" "$normal_orders"
assert_page "normal orders" "$normal_orders"
logs=$(api_get "/api/v1/logs?perPage=5" "$auth_token")
assert_api_ok "logs" "$logs"
assert_page "logs" "$logs"

if [ "$EXERCISE_WRITES" = "true" ]; then
  echo "==> Flash order write path"
  suffix=$(date +%s)
  connector_id=
  order_id=

  cleanup_write_check() {
    if [ -n "$order_id" ]; then
      api_delete "/api/v1/orders/$order_id" "$auth_token" >/dev/null 2>&1 || echo "warning: failed to delete smoke order $order_id" >&2
    fi
    if [ -n "$connector_id" ]; then
      api_delete "/api/v1/connectors/$connector_id" "$auth_token" >/dev/null 2>&1 || echo "warning: failed to delete smoke connector $connector_id" >&2
    fi
  }
  trap cleanup_write_check EXIT

  connector_body='{"name":"smoke-flash-'"$suffix"'","baseUrl":"http://127.0.0.1:1","kind":"smoke","status":"active","timeoutMs":1000}'
  connector=$(api_post "/api/v1/connectors" "$connector_body" "$auth_token")
  assert_api_ok "create smoke connector" "$connector"
  connector_id=$(json_number "id" "$connector")
  [ -n "$connector_id" ] || fail "create smoke connector did not return id"

  order_body='{"connectorId":'"$connector_id"',"account":"smoke-'"$suffix"'","courseName":"Smoke Flash Queue","flashMode":true,"fee":0}'
  order=$(api_post "/api/v1/orders" "$order_body" "$auth_token")
  assert_api_ok "create smoke flash order" "$order"
  order_id=$(json_number "id" "$order")
  [ -n "$order_id" ] || fail "create smoke flash order did not return id"
  echo "$order" | grep -q '"flashMode"[[:space:]]*:[[:space:]]*true' || fail "smoke order was not created in flash mode: $order"
  echo "$order" | grep -q '"status"[[:space:]]*:[[:space:]]*"queued"' || fail "smoke order was not created as queued: $order"
  echo "$order" | grep -q '"dockingStatus"[[:space:]]*:[[:space:]]*"pending"' || fail "smoke order docking status was not pending: $order"

  cleanup_write_check
  trap - EXIT
fi

echo "==> Frontend entry"
frontend=$(retry "frontend entry" curl -fsS --max-time "$TIMEOUT_SEC" "$FRONTEND_URL/")
echo "$frontend" | grep -q '<div id="app"></div>' || fail "frontend entry does not look like the Vue app shell"

echo "==> Frontend API proxy"
proxied_health=$(retry "frontend API proxy" curl -fsS --max-time "$TIMEOUT_SEC" "$FRONTEND_URL/api/v1/health")
assert_api_ok "frontend API proxy" "$proxied_health"

echo "Smoke checks completed."
