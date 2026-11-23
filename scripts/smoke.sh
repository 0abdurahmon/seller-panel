#!/bin/sh
set -eu

backend="${BACKEND_URL:-http://localhost:8080}"
frontend="${FRONTEND_URL:-http://localhost:3000}"
docs_user="${SWAGGER_USER:-alifco}"
docs_password="${SWAGGER_PASSWORD:-alifco26}"
email="smoke-$(date +%s)-$$@example.com"
json="Content-Type: application/json"
failures=0

status() {
  curl -s -o /dev/null -w '%{http_code}' "$@"
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
check "register" 201 "$(status -X POST "$frontend/api/auth/register" -H "$json" -d "$register_body")"
check "register with the same email" 409 "$(status -X POST "$frontend/api/auth/register" -H "$json" -d "$register_body")"
check "login" 200 "$(status -X POST "$frontend/api/auth/login" -H "$json" -d "$login_body")"
check "login with a wrong password" 401 "$(status -X POST "$frontend/api/auth/login" -H "$json" -d "$wrong_login_body")"
check "docs without credentials" 401 "$(status "$frontend/docs/")"
check "docs with a wrong password" 401 "$(status -u "$docs_user:wrong-password" "$frontend/docs/")"
check "docs with credentials" 200 "$(status -u "$docs_user:$docs_password" "$frontend/docs/")"

if [ "$failures" -ne 0 ]; then
  printf '%s check(s) failed\n' "$failures" >&2
  exit 1
fi
printf 'all checks passed\n'
