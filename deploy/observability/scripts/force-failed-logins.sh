#!/usr/bin/env bash
# Forced test of the alerts (`make obs-alert-test`): sends a burst of failed
# logins to the local API and waits until Grafana fires "Pico de logins
# fallidos". Needs the API (`make run`) and the stack (`make obs-up`) running.
#
# It first sends one failure and waits for its series in Prometheus: increase()
# ignores the first sample of a new series, so a burst on a fresh API process
# would not count.
set -euo pipefail

API=${API:-http://localhost:8080}
PROMETHEUS=${PROMETHEUS:-http://localhost:9090}
GRAFANA=${GRAFANA:-http://localhost:3000}
ATTEMPTS=${ATTEMPTS:-40}
RULE_UID=cuadra-failed-logins-spike
TIMEOUT=${TIMEOUT:-300}

command -v jq >/dev/null || { echo "Falta jq." >&2; exit 1; }

fail_login() {
  curl -s -o /dev/null -w '%{http_code}' -X POST "$API/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d '{"email":"alerta-forzada@cuadra.invalid","password":"no-es-la-clave"}'
}

wait_for() { # wait_for <description> <command...>
  local what=$1 start=$SECONDS
  shift
  until "$@"; do
    if (( SECONDS - start > TIMEOUT )); then
      echo "Pasaron ${TIMEOUT}s sin $what." >&2
      exit 1
    fi
    sleep 5
  done
}

failures_series_exists() {
  curl -sf -G "$PROMETHEUS/api/v1/query" \
    --data-urlencode 'query=cuadra_identity_logins_total{result="failure"}' |
    jq -e '.data.result | length > 0' >/dev/null
}

alert_firing() {
  curl -sf "$GRAFANA/api/prometheus/grafana/api/v1/rules" |
    jq -e --arg uid "$RULE_UID" '[.data.groups[].rules[] | select(.uid == $uid and .state == "firing")] | length > 0' >/dev/null
}

curl -sf "$API/healthz" >/dev/null || { echo "La API no responde en $API (make run)." >&2; exit 1; }

echo "1/3 Login fallido inicial (status $(fail_login)); esperando la serie en Prometheus..."
wait_for "la métrica de logins fallidos en Prometheus" failures_series_exists

echo "2/3 Enviando $ATTEMPTS logins fallidos..."
for _ in $(seq "$ATTEMPTS"); do fail_login >/dev/null; done

echo "3/3 Esperando a que la alerta se dispare (la métrica se exporta cada 60 s)..."
wait_for "que se disparara la alerta $RULE_UID" alert_firing
echo "La alerta «Pico de logins fallidos» está disparada: $GRAFANA/alerting/list"
