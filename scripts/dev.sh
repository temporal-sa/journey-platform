#!/usr/bin/env bash
set -euo pipefail

# Load environment configuration file if present
if [ -f "${ENV_FILE:-.env}" ]; then
    set -a
    source "${ENV_FILE:-.env}"
    set +a
elif [ -f ".env.example" ]; then
    set -a
    source .env.example
    set +a
fi

PIDS=()

cleanup() {
    echo ""
    echo "Stopping all dev processes..."
    trap - EXIT INT TERM
    if [ ${#PIDS[@]} -gt 0 ]; then
        for pid in "${PIDS[@]}"; do
            if kill -0 "$pid" 2>/dev/null; then
                kill "$pid" 2>/dev/null || true
            fi
        done
    fi
    kill 0 2>/dev/null || true
    wait 2>/dev/null || true
    echo "Dev environment stopped."
}

trap cleanup EXIT INT TERM

run_service() {
    local name="$1"
    shift
    echo "Starting service [${name}]..."
    ("$@") 2>&1 | awk -v prefix="[${name}]" '{print prefix " " $0; fflush()}' &
    PIDS+=($!)
}

# Start Docker Compose infrastructure containers if available
if command -v docker >/dev/null 2>&1; then
    if [ -f "compose.yaml" ] || [ -f "docker-compose.yml" ] || [ -f "docker-compose.yaml" ]; then
        echo "Starting Docker Compose infrastructure containers..."
        docker compose up -d
        echo "Waiting for PostgreSQL database container to be healthy..."
        for i in {1..30}; do
            if docker compose exec -T postgres pg_isready -U journey -d journeydb >/dev/null 2>&1; then
                echo "PostgreSQL database is ready."
                break
            fi
            sleep 1
        done
        echo "Applying PostgreSQL database schema migrations..."
        if [ -f "migrations/postgres/000001_create_postgres_tables.up.sql" ]; then
            docker compose exec -T postgres psql -U journey -d journeydb < migrations/postgres/000001_create_postgres_tables.up.sql 2>/dev/null || true
        fi
    fi
fi

# Start Go services from cmd/
if [ -d "cmd" ]; then
    for dir in cmd/*/; do
        if [ -d "$dir" ]; then
            service_name=$(basename "$dir")
            case "$service_name" in
                control-api)
                    PORT="${CONTROL_API_PORT:-8080}" run_service "$service_name" go run "./$dir"
                    ;;
                event-ingress)
                    PORT="${EVENT_INGRESS_PORT:-8084}" run_service "$service_name" go run "./$dir"
                    ;;
                outcome-ingress)
                    PORT="${OUTCOME_INGRESS_PORT:-8085}" run_service "$service_name" go run "./$dir"
                    ;;
                fake-provider)
                    PORT="${FAKE_PROVIDER_PORT:-8082}" run_service "$service_name" go run "./$dir"
                    ;;
                journey-worker)
                    echo "Skipping automatic start of journey-worker (managed via dev panel/API)"
                    ;;
                *)
                    run_service "$service_name" go run "./$dir"
                    ;;
            esac
        fi
    done
fi

# Start Web frontend
if [ -d "web" ]; then
    PORT="${FRONTEND_PORT:-3002}" run_service "web" npm --prefix web run dev -- --port "${FRONTEND_PORT:-3002}"
fi

echo "Dev environment started. Press Ctrl+C to stop."
wait
