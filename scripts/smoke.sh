#!/bin/sh
set -eu

backend="${BACKEND_URL:-http://localhost:8080}"
frontend="${FRONTEND_URL:-http://localhost:3000}"
email="smoke-$(date +%s)-$$@example.com"
json="Content-Type: application/json"
failures=0

status() {
  curl -s -o /dev/null -w '%{http_code}' "$@"
}

field() {
  sed -n "s/.*\"$1\":\"\{0,1\}\([^\",}]*\).*/\1/p"
}

check() {
  name="$1"
  want="$2"
  got="$3"
  if [ "$got" = "$want" ]; then
    printf 'ok    %s\n' "$name"
  else
    printf 'FAIL  %s (want %s, got %s)\n' "$name" "$want" "$got"
    failures=$((failures + 1))
  fi
}

register_body="{\"email\":\"$email\",\"password\":\"correct-horse\",\"shop_name\":\"Smoke\"}"
login_body="{\"email\":\"$email\",\"password\":\"correct-horse\"}"
wrong_login_body="{\"email\":\"$email\",\"password\":\"wrong-password\"}"

check "backend health" 200 "$(status "$backend/health")"
check "frontend page" 200 "$(status "$frontend/")"
check "frontend deep link falls back to the app" 200 "$(status "$frontend/orders/1")"
check "frontend proxies /api to the backend" 200 "$(status "$frontend/api/health")"
check "security headers are sent" 1 "$(curl -sI "$frontend/" | grep -ic '^x-content-type-options: nosniff')"
check "register through nginx" 201 "$(status -X POST "$frontend/api/auth/register" -H "$json" -d "$register_body")"
login_response="$(curl -s -X POST "$frontend/api/auth/login" -H "$json" -d "$login_body")"
token="$(printf '%s' "$login_response" | field token)"
check "login through nginx returns a token" 1 "$(printf '%s' "$token" | grep -c .)"
check "protected route without a token" 401 "$(status "$frontend/api/shop")"
check "protected route with a token (Authorization passes the proxy)" 200 "$(status "$frontend/api/shop" -H "Authorization: Bearer $token")"
check "spec JSON under /api" 200 "$(status "$frontend/api/openapi.json")"

edge_limited=0
for attempt in 1 2 3 4 5 6 7 8 9 10 11 12; do
  code="$(status -X POST "$frontend/api/auth/login" -H "$json" -d "$wrong_login_body")"
  [ "$code" = 429 ] && edge_limited=$((edge_limited + 1))
done
check "nginx limits rapid requests to /api/auth/" 1 "$([ "$edge_limited" -ge 1 ] && echo 1 || echo 0)"
edge_body="$(curl -s -X POST "$frontend/api/auth/login" -H "$json" -d "$wrong_login_body")"
check "nginx 429 has the standard error body" 1 "$(printf '%s' "$edge_body" | grep -c '"code":"rate_limited"')"

global_limited=0
for attempt in $(seq 1 80); do
  code="$(status "$frontend/api/health")"
  [ "$code" = 429 ] && global_limited=$((global_limited + 1))
done
check "nginx limits bursts on /api/" 1 "$([ "$global_limited" -ge 1 ] && echo 1 || echo 0)"

check "docs open without a login" 200 "$(status "$frontend/docs/")"
check "spec YAML open without a login" 200 "$(status "$frontend/docs/spec/openapi.yaml")"
check "spec JSON at /swagger" 200 "$(status "$frontend/swagger")"
check "spec JSON next to the YAML" 200 "$(status "$frontend/docs/spec/openapi.json")"
check "spec JSON is the OpenAPI document" 1 "$(curl -s "$frontend/swagger" | grep -c '^  "openapi":')"

if [ "$failures" -ne 0 ]; then
  printf '%s check(s) failed\n' "$failures" >&2
  exit 1
fi
printf 'all checks passed\n'
