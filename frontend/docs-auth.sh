#!/bin/sh
set -eu

user="${SWAGGER_USER:?SWAGGER_USER is required}"
password="${SWAGGER_PASSWORD:?SWAGGER_PASSWORD is required}"

case "$user" in
  *:*) echo "SWAGGER_USER must not contain a colon" >&2; exit 1 ;;
esac

hash="$(printf '%s' "$password" | mkpasswd -m sha512 -s)"
printf '%s:%s\n' "$user" "$hash" > /tmp/docs.htpasswd
