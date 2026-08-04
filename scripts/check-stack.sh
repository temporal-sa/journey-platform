#!/usr/bin/env bash

# Smoke test and health check script for the hermetic local service stack

set -u

# Load environment variables if available
if [ -f "${ENV_FILE:-.env}" ]; then
    set -a
    source "${ENV_FILE:-.env}"
    set +a
elif [ -f ".env.example" ]; then
    set -a
    source .env.example
    set +a
fi

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

FAILED=0
CHECK_COUNT=0
PASSED_COUNT=0

check_http() {
  local name="$1"
  local url="$2"
  local expected_code="${3:-200}"

  CHECK_COUNT=$((CHECK_COUNT + 1))
  printf "Checking %-25s (%s)... " "${name}" "${url}"

  local status_code
  status_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "${url}" 2>/dev/null || echo "000")

  if [ "${status_code}" -eq "${expected_code}" ] || ([ "${expected_code}" -eq 200 ] && [ "${status_code}" -ge 200 ] && [ "${status_code}" -lt 400 ]); then
    echo -e "[\033[0;32mOK\033[0m] (HTTP ${status_code})"
    PASSED_COUNT=$((PASSED_COUNT + 1))
  else
    echo -e "[\033[0;31mFAILED\033[0m] (Got HTTP ${status_code}, expected ${expected_code})"
    FAILED=1
  fi
}

check_tcp() {
  local name="$1"
  local host="$2"
  local port="$3"

  CHECK_COUNT=$((CHECK_COUNT + 1))
  printf "Checking %-25s (%s:%s)... " "${name}" "${host}" "${port}"

  if nc -z -w 5 "${host}" "${port}" 2>/dev/null; then
    echo -e "[\033[0;32mOK\033[0m]"
    PASSED_COUNT=$((PASSED_COUNT + 1))
  else
    echo -e "[\033[0;31mFAILED\033[0m] (Could not connect)"
    FAILED=1
  fi
}

echo "=========================================================="
echo "      Journey Platform Stack Readiness & Smoke Test       "
echo "=========================================================="

POSTGRES_HOST="${POSTGRES_HOST:-127.0.0.1}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
KAFKA_PORT="${KAFKA_PORT:-9092}"
SCHEMA_REGISTRY_PORT="${SCHEMA_REGISTRY_PORT:-8081}"
CLICKHOUSE_HOST="${CLICKHOUSE_HOST:-127.0.0.1}"
CLICKHOUSE_HTTP_PORT="${CLICKHOUSE_HTTP_PORT:-8123}"
CLICKHOUSE_NATIVE_PORT="${CLICKHOUSE_NATIVE_PORT:-9009}"
MINIO_PORT="${MINIO_PORT:-9000}"
MINIO_CONSOLE_PORT="${MINIO_CONSOLE_PORT:-9001}"
MAILPIT_UI_PORT="${MAILPIT_UI_PORT:-8025}"
MAILPIT_SMTP_PORT="${MAILPIT_SMTP_PORT:-1025}"
TEMPORAL_PORT="${TEMPORAL_PORT:-7233}"
TEMPORAL_UI_PORT="${TEMPORAL_UI_PORT:-8233}"
FAKE_PROVIDER_PORT="${FAKE_PROVIDER_PORT:-8082}"
JAEGER_UI_PORT="${JAEGER_UI_PORT:-16686}"

# Check services using configured ports
check_tcp "PostgreSQL" "${POSTGRES_HOST}" "${POSTGRES_PORT}"
check_tcp "Kafka KRaft" "127.0.0.1" "${KAFKA_PORT}"
check_http "Schema Registry" "http://127.0.0.1:${SCHEMA_REGISTRY_PORT}/subjects"
check_http "ClickHouse HTTP" "http://${CLICKHOUSE_HOST}:${CLICKHOUSE_HTTP_PORT}/ping"
check_tcp "ClickHouse Native" "${CLICKHOUSE_HOST}" "${CLICKHOUSE_NATIVE_PORT}"
check_http "MinIO API" "http://127.0.0.1:${MINIO_PORT}/minio/health/live"
check_http "MinIO Console" "http://127.0.0.1:${MINIO_CONSOLE_PORT}/"
check_http "Mailpit UI" "http://127.0.0.1:${MAILPIT_UI_PORT}/api/v1/info"
check_tcp "Mailpit SMTP" "127.0.0.1" "${MAILPIT_SMTP_PORT}"
check_tcp "Temporal Server gRPC" "127.0.0.1" "${TEMPORAL_PORT}"
check_http "Temporal Web UI" "http://127.0.0.1:${TEMPORAL_UI_PORT}/"
check_http "Fake Provider" "http://127.0.0.1:${FAKE_PROVIDER_PORT}/health"
check_http "Jaeger UI" "http://127.0.0.1:${JAEGER_UI_PORT}/"
echo "----------------------------------------------------------"
echo "Summary: ${PASSED_COUNT}/${CHECK_COUNT} checks passed."

if [ "${FAILED}" -ne 0 ]; then
  echo -e "\033[0;31mStack smoke test FAILED!\033[0m Not all services are reachable or healthy."
  exit 1
else
  echo -e "\033[0;32mStack smoke test PASSED!\033[0m All hermetic services are healthy."
  exit 0
fi
