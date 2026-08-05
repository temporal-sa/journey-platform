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
        docker compose exec -T postgres psql -U journey -d journeydb -c "
            CREATE TABLE IF NOT EXISTS catalogs (tenant_id VARCHAR(255) NOT NULL, record_id VARCHAR(255) NOT NULL, name VARCHAR(255) NOT NULL, component_type VARCHAR(64) NOT NULL, version VARCHAR(64) NOT NULL, description TEXT NOT NULL DEFAULT '', schema_definition JSONB NOT NULL DEFAULT '{}'::jsonb, content_hash VARCHAR(64) NOT NULL DEFAULT '', tags TEXT[] NOT NULL DEFAULT '{}', is_deprecated BOOLEAN NOT NULL DEFAULT FALSE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY (tenant_id, record_id));
            CREATE TABLE IF NOT EXISTS journey_drafts (tenant_id VARCHAR(255) NOT NULL, draft_id VARCHAR(255) NOT NULL, name VARCHAR(255) NOT NULL, description TEXT NOT NULL DEFAULT '', version INT NOT NULL DEFAULT 1, nodes JSONB NOT NULL DEFAULT '[]'::jsonb, edges JSONB NOT NULL DEFAULT '[]'::jsonb, content_hash VARCHAR(64) NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY (tenant_id, draft_id));
            CREATE TABLE IF NOT EXISTS journey_versions (tenant_id VARCHAR(255) NOT NULL, version_id VARCHAR(255) NOT NULL, draft_id VARCHAR(255) NOT NULL, version INT NOT NULL DEFAULT 1, entry_node_id VARCHAR(255) NOT NULL DEFAULT '', nodes JSONB NOT NULL DEFAULT '[]'::jsonb, edges JSONB NOT NULL DEFAULT '[]'::jsonb, content_hash VARCHAR(64) NOT NULL DEFAULT '', compiled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY (tenant_id, version_id));
            CREATE TABLE IF NOT EXISTS static_lists (tenant_id VARCHAR(255) NOT NULL, list_id VARCHAR(255) NOT NULL, name VARCHAR(255) NOT NULL, description TEXT NOT NULL DEFAULT '', item_count INT NOT NULL DEFAULT 0, data_classification VARCHAR(64) NOT NULL DEFAULT 'NonPII', items JSONB NOT NULL DEFAULT '[]'::jsonb, content_hash VARCHAR(64) NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY (tenant_id, list_id));
            CREATE TABLE IF NOT EXISTS test_runs (tenant_id VARCHAR(255) NOT NULL, test_run_id VARCHAR(255) NOT NULL, draft_id VARCHAR(255) NOT NULL, ir_id VARCHAR(255) NOT NULL DEFAULT '', static_list_id VARCHAR(255) NOT NULL DEFAULT '', status VARCHAR(64) NOT NULL DEFAULT 'running', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY (tenant_id, test_run_id));
            CREATE TABLE IF NOT EXISTS enrollments (tenant_id VARCHAR(255) NOT NULL, enrollment_id VARCHAR(255) NOT NULL, journey_version_id VARCHAR(255) NOT NULL, subject_id VARCHAR(255) NOT NULL, status VARCHAR(64) NOT NULL DEFAULT 'active', current_node_id VARCHAR(255) NOT NULL DEFAULT '', state_data JSONB NOT NULL DEFAULT '{}'::jsonb, enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), completed_at TIMESTAMPTZ, PRIMARY KEY (tenant_id, enrollment_id));
        " 2>/dev/null || true
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
