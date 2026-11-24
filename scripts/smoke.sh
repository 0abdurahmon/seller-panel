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
check "register" 201 "$(status -X POST "$frontend/api/auth/register" -H "$json" -d "$register_body")"
check "register with the same email" 409 "$(status -X POST "$frontend/api/auth/register" -H "$json" -d "$register_body")"
login_response="$(curl -s -X POST "$frontend/api/auth/login" -H "$json" -d "$login_body")"
check "login returns a token" 1 "$(printf '%s' "$login_response" | field token | grep -c .)"
token="$(printf '%s' "$login_response" | field token)"
auth="Authorization: Bearer $token"
check "login with a wrong password" 401 "$(status -X POST "$frontend/api/auth/login" -H "$json" -d "$wrong_login_body")"
sku="SMOKE-$(date +%s)-$$"
category="$(curl -s "$frontend/api/categories" -H "$auth" | field id | head -n 1)"
product_body="{\"category_id\":${category:-0},\"title\":\"Smoke phone\",\"price\":1000000,\"sku\":\"$sku\",\"stock\":3,\"status\":\"active\"}"

check "categories without a token" 401 "$(status "$frontend/api/categories")"
check "categories with a token" 200 "$(status "$frontend/api/categories" -H "$auth")"
check "seeded category exists" 1 "$(printf '%s' "$category" | grep -c '[0-9]')"
check "products without a token" 401 "$(status "$frontend/api/products")"
check "create product" 201 "$(status -X POST "$frontend/api/products" -H "$auth" -H "$json" -d "$product_body")"
check "create product with the same sku" 409 "$(status -X POST "$frontend/api/products" -H "$auth" -H "$json" -d "$product_body")"
check "create product with a bad price" 400 "$(status -X POST "$frontend/api/products" -H "$auth" -H "$json" -d "{\"category_id\":${category:-0},\"title\":\"x\",\"price\":0,\"sku\":\"$sku-bad\"}")"
list_response="$(curl -s "$frontend/api/products?q=$sku" -H "$auth")"
product_id="$(printf '%s' "$list_response" | field id | head -n 1)"
check "list finds the product by sku" 1 "$(printf '%s' "$list_response" | grep -c "\"stock_label\":\"low_stock\"")"
check "get product" 200 "$(status "$frontend/api/products/${product_id:-0}" -H "$auth")"
check "archive product" 200 "$(status -X POST "$frontend/api/products/${product_id:-0}/archive" -H "$auth")"
check "missing product" 404 "$(status "$frontend/api/products/999999999" -H "$auth")"
check "docs without credentials" 401 "$(status "$frontend/docs/")"
check "docs with a wrong password" 401 "$(status -u "$docs_user:wrong-password" "$frontend/docs/")"
check "docs with credentials" 200 "$(status -u "$docs_user:$docs_password" "$frontend/docs/")"

if [ "$failures" -ne 0 ]; then
  printf '%s check(s) failed\n' "$failures" >&2
  exit 1
fi
printf 'all checks passed\n'
