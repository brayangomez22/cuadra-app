#!/usr/bin/env bash
# DAST: an OWASP ZAP API scan of the API of the end-to-end stack (make e2e-up),
# driven by api/openapi.yaml, as the owner of a fresh business so it reaches
# the protected operations too. The rules (rules.tsv) fail on the serious
# findings; the HTML and JSON reports go to $REPORT_DIR. Against staging once
# it exists (T46): API=https://... .github/zap/scan.sh
set -euo pipefail

api="${API:-http://127.0.0.1:18080}"
report_dir="${REPORT_DIR:-zap-report}"
here="$(cd "$(dirname "$0")" && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

cp "$here/rules.tsv" "$here/../../api/openapi.yaml" "$work/"
chmod 777 "$work"

# The stack may still be starting: wait up to a minute for the API.
for _ in $(seq 60); do
  curl -fsS "$api/readyz" >/dev/null 2>&1 && break
  sleep 1
done
curl -fsS "$api/readyz" >/dev/null

email="dast-$(date +%s)@example.com"
token="$(curl -fsS "$api/api/v1/auth/signup" -H 'Content-Type: application/json' \
  -d '{"tenant_name":"Negocio de escaneo","nit":"900123456-8","name":"Escáner DAST","email":"'"$email"'","password":"escaneo-dast-con-frase-larga","business_template":"generica"}' |
  jq -r .access_token)"

status=0
docker run --rm --network host -v "$work:/zap/wrk:rw" \
  -e ZAP_AUTH_HEADER=Authorization -e ZAP_AUTH_HEADER_VALUE="Bearer $token" \
  ghcr.io/zaproxy/zaproxy:stable \
  zap-api-scan.py -t /zap/wrk/openapi.yaml -f openapi -O "$api" -c rules.tsv \
  -r report.html -J report.json -I || status=$?

mkdir -p "$report_dir"
cp "$work"/report.* "$report_dir/" 2>/dev/null || true
exit "$status"
