---
name: boot-app-make-dev
description: Guides Antigravity agents on how to boot the event-driven journey engine application using 'make dev', managing environment configuration, git worktree symlinks, background daemons, and readiness checks.
---

# Boot Application via `make dev`

This skill provides step-by-step instructions for booting the full event-driven workflow engine application stack using `make dev` in Google Antigravity.

---

## Prerequisites & Worktree Preparation

### 1. Git Worktree Symlink Check
If executing within a git worktree (e.g. `../<worktree-name>`), ensure the `cmd/` directory is accessible in the working directory. If missing, create a symlink to the main repository:

```bash
ln -s ../event-driven-workflow-engine-validated-pattern/cmd ./cmd
```

### 2. Environment File (.env)
The `Makefile` and `scripts/dev.sh` automatically source the `.env` file (with fallback to `.env.example`). Key variables include:
- `CONTROL_API_PORT=8080`
- `EVENT_INGRESS_PORT=8084`
- `OUTCOME_INGRESS_PORT=8085`
- `FAKE_PROVIDER_PORT=8082`
- `FRONTEND_PORT=3002`
- `TEMPORAL_HOST_PORT=127.0.0.1:7233`
- `TEMPORAL_NAMESPACE=default`
- `POSTGRES_DSN=postgres://journey:journey_dev_pass@127.0.0.1:5432/journeydb?sslmode=disable`

---

## Execution Workflow

### 1. Launch Stack using `run_command`
To start the application stack concurrently in the background, invoke `run_command`:

- **CommandLine**: `make dev`
- **Cwd**: `<workspace-path>`
- **IsDaemon**: `true`
- **BypassSandbox**: `true`
- **WaitMsBeforeAsync**: `5000`

### 2. Services Started by `make dev`
When `make dev` executes `scripts/dev.sh`, it starts:
1. **Docker Infrastructure**: PostgreSQL, Temporal Server, Temporal Web UI, Kafka, ClickHouse, MinIO, Mailpit, Jaeger, Fake Provider via `docker compose up -d`.
2. **Go Services**:
   - `[control-api]` (port 8080): Core REST API server.
   - `[event-ingress]` (port 8084): Event ingestion gateway.
   - `[outcome-ingress]` (port 8085): Outcome tracking service.
   - `[fake-provider]` (port 8082): Mock third-party messaging providers.
   - `[target-dispatcher]`: Dispatcher for Kafka/queue target lanes.
   - `[report-materializer]`: Analytics materializer service.
   - `[journey-worker]`: Temporal worker processing `CompiledJourneyWorkflow` on task queue `journey-engine-task-queue`.
3. **Web Frontend**: Vite dev server on port 3002 (`npm --prefix web run dev`).

---

## Verification & Health Check

### 1. Stack Readiness Script
Run `./scripts/check-stack.sh` with `BypassSandbox: true`:

```bash
./scripts/check-stack.sh
```

All 13 health checks (PostgreSQL, Kafka, Schema Registry, ClickHouse, MinIO, Mailpit, Temporal, Temporal UI, Jaeger, Fake Provider) should report `[OK]`.

### 2. REST API Verification
Verify `control-api` is responding:

```bash
curl -s http://localhost:8080/api/v1/journeys/drafts
```

---

## Shutdown Instructions

To stop the running application:
1. Use `manage_task` with `Action: "kill"` on the `make dev` task ID.
2. Run `make down` to cleanly stop Docker containers and dev processes:

```bash
make down
```
