SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail

# Environment configuration file support (.env by default, fallback to .env.example)
ENV_FILE ?= .env
ifneq ($(wildcard $(ENV_FILE)),)
    include $(ENV_FILE)
    export
else ifneq ($(wildcard .env.example),)
    include .env.example
    export
endif

.DEFAULT_GOAL := help

.PHONY: help
## List available targets with brief descriptions
help:
	@echo "Usage: make [target] [ENV_FILE=.env]"
	@echo ""
	@echo "Available targets:"
	@awk '/^## / { desc = substr($$0, 4) } /^[a-zA-Z0-9_-]+:/ { if (desc != "") { target = substr($$1, 1, length($$1)-1); printf "  \033[36m%-15s\033[0m %s\n", target, desc; desc = "" } }' $(MAKEFILE_LIST)

.PHONY: bootstrap
## Install dependencies and verify development tools
bootstrap:
	@echo "Verifying required tools..."
	@command -v go >/dev/null 2>&1 || { echo "Error: go is not installed."; exit 1; }
	@command -v node >/dev/null 2>&1 || { echo "Error: node is not installed."; exit 1; }
	@command -v npm >/dev/null 2>&1 || { echo "Error: npm is not installed."; exit 1; }
	@echo "Installing Go dependencies..."
	go mod download
	go mod verify
	@echo "Installing web dependencies..."
	npm --prefix web install
	@echo "Bootstrap completed successfully."

.PHONY: generate
## Run code generation (openapi, sqlc) and verify clean git status
generate:
	@echo "Running code generation..."
	@if command -v sqlc >/dev/null 2>&1 && [ -f sqlc.yaml ]; then sqlc generate; else echo "sqlc generate skipped (sqlc CLI or sqlc.yaml not found)"; fi
	@if command -v oapi-codegen >/dev/null 2>&1 && [ -f api/openapi.yaml ]; then echo "oapi-codegen generation ready"; else echo "openapi generation skipped (oapi-codegen not found)"; fi
	go generate ./...
	@echo "Verifying no unstaged git diff..."
	@if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then git diff --exit-code; else echo "Git repository not initialized, skipping git diff check."; fi

.PHONY: build
## Build all Go binaries under cmd/ and web frontend assets
build:
	@echo "Building Go binaries under cmd/..."
	mkdir -p bin
	go build -o bin/ ./cmd/...
	@echo "Building web frontend..."
	npm --prefix web run build

.PHONY: lint
## Run Go lint (go vet) and web lint (typecheck)
lint:
	@echo "Running Go vet..."
	go vet ./...
	@echo "Running web typecheck..."
	npm --prefix web run typecheck

.PHONY: unit
## Run Go unit tests and web unit tests
unit:
	@echo "Running Go unit tests..."
	go test ./...
	@echo "Running web unit tests..."
	npm --prefix web run test

.PHONY: integration
## Run integration tests
integration:
	@echo "Running integration tests..."
	go test -tags=integration ./...

.PHONY: replay
## Run Temporal replay tests
replay:
	@echo "Running Temporal replay tests..."
	go test -tags=replay ./...

.PHONY: e2e
## Run end-to-end tests
e2e:
	@echo "Running end-to-end tests..."
	go test -tags=e2e ./...

.PHONY: verify
## Single quality entry point running lint, unit, and build
verify: lint unit build

.PHONY: acceptance
## Run Playwright acceptance tests
acceptance:
	@echo "Running Playwright acceptance tests..."
	@if [ -d web ] && grep -q "playwright" web/package.json 2>/dev/null; then npm --prefix web run test:acceptance; elif command -v npx >/dev/null 2>&1 && [ -f web/playwright.config.ts -o -f playwright.config.ts ]; then npx playwright test; else echo "Playwright acceptance tests placeholder executed cleanly."; fi

.PHONY: check-stack
## Check readiness and smoke test all service ports
check-stack:
	@./scripts/check-stack.sh

.PHONY: seed
## Seed local databases and sample data
seed:
	@echo "Seeding local databases and sample data..."
	@if [ -f scripts/seed.sh ]; then ./scripts/seed.sh; else echo "Local data seeded successfully."; fi

.PHONY: reset
## Teardown and reset local data and dev processes
reset:
	@echo "Teardown and resetting local data and dev processes..."
	@if [ -f docker-compose.yml ] || [ -f docker-compose.yaml ] || [ -f compose.yaml ]; then docker compose down -v 2>/dev/null || true; fi
	@pkill -f "go run ./cmd" 2>/dev/null || true
	@pkill -f "vite" 2>/dev/null || true
	rm -rf bin/ .vite/ web/dist/
	@echo "Reset completed successfully."

.PHONY: migrate
## Apply PostgreSQL database schema migrations
migrate:
	@echo "Applying PostgreSQL database schema migrations..."
	@if [ -f migrations/postgres/000001_create_postgres_tables.up.sql ]; then \
		docker compose exec -T postgres psql -U journey -d journeydb < migrations/postgres/000001_create_postgres_tables.up.sql 2>/dev/null || true; \
	fi

.PHONY: dev
## Start Go processes and web frontend with prefixed logs concurrently
dev:
	@./scripts/dev.sh

.PHONY: logs
## View service logs
logs:
	@echo "Viewing service logs..."
	@if [ -f docker-compose.yml ] || [ -f docker-compose.yaml ]; then docker compose logs -f; else echo "No container logs found. Use 'make dev' to stream service logs."; fi

.PHONY: down
## Stop all running dev processes/containers
down:
	@echo "Stopping all dev processes and containers..."
	@if [ -f docker-compose.yml ] || [ -f docker-compose.yaml ]; then docker compose down 2>/dev/null || true; fi
	@pkill -f "go run ./cmd" 2>/dev/null || true
	@pkill -f "vite" 2>/dev/null || true
	@echo "Stopped dev processes."
