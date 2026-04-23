#!/usr/bin/env sh
set -eu

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

if [ ! -f compose.prod.yaml ]; then
  echo "missing compose.prod.yaml" >&2
  exit 1
fi
if [ ! -f deploy/caddy/Caddyfile ]; then
  echo "missing deploy/caddy/Caddyfile" >&2
  exit 1
fi

services="$(docker compose -f compose.yaml -f compose.prod.yaml config --services)"
printf '%s\n' "${services}" | grep -qx "caddy"

config_json="$(docker compose -f compose.yaml -f compose.prod.yaml config --format json)"

printf '%s\n' "${config_json}" | python3 -c '
import json
import sys

config = json.load(sys.stdin)
services = config["services"]

def published_ports(service_name):
    return [
        str(port.get("published", ""))
        for port in services.get(service_name, {}).get("ports", [])
    ]

assert "80" in published_ports("caddy"), published_ports("caddy")
assert "443" in published_ports("caddy"), published_ports("caddy")
assert published_ports("app") == [], published_ports("app")
assert published_ports("postgres") == [], published_ports("postgres")
assert published_ports("redis") == [], published_ports("redis")
assert "8080" in [str(item) for item in services["app"].get("expose", [])], services["app"].get("expose", [])
'

grep -q "reverse_proxy app:8080" deploy/caddy/Caddyfile
grep -q "X-Forwarded-Proto" deploy/caddy/Caddyfile

echo "production proxy config tests passed"
