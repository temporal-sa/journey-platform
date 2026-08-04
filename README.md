# Event-Driven Journey Platform

A locally testable journey-authoring and execution platform implemented in Go and TypeScript. Kafka serves as the initial event emitter, Temporal executes compiled journey workflows, `@xyflow/react` powers the authoring canvas, marketers can run percentage-based experiments and inspect aggregate outcomes, and candidate journeys can be tested against immutable static contact lists.

---

## Repository Layout

```
.
├── api/                   # OpenAPI 3.0 specification & JSON schemas
│   ├── openapi.yaml
│   └── schemas/
├── cmd/                   # Go binary entrypoints
│   ├── control-api/       # REST API control plane
│   ├── event-ingress/     # Event ingress REST emitter & capture consumer
│   ├── target-dispatcher/ # Kafka partition fencer & Temporal dispatcher
│   ├── journey-worker/    # Temporal worker daemon
│   ├── outcome-ingress/   # Webhook & outcome callback receiver
│   ├── report-materializer/# Analytical report materializer
│   └── fake-provider/     # Channel fake provider service
├── internal/              # Core domain logic & storage packages
│   ├── activities/        # Temporal Activities & action gateway
│   ├── catalog/           # Event/action/metric catalogs
│   ├── compiler/          # Graph canonicalizer, validator, IR, simulator
│   ├── domain/            # Versioned cross-component domain contracts
│   ├── experiments/       # HMAC bucketing, sticky assignment, reports
│   ├── ingress/           # Ingress codecs, capture, dispatcher
│   ├── security/          # Tenant guards, structured redaction, tombstones
│   ├── store/             # PostgreSQL, MinIO, ClickHouse repositories
│   ├── testaudience/      # Static-list CSV parser, test-run coordinator
│   ├── testkit/           # Test helpers & version pinning assertions
│   └── workflows/         # Temporal Journey Workflow interpreter
├── migrations/            # SQL migration scripts (postgres & clickhouse)
├── web/                   # React + TypeScript xyflow frontend
├── test/                  # Contract, integration, e2e, and load test suites
└── Makefile               # Repeatable developer command surface
```

---

## Local Development Prerequisites

- **Go**: 1.22+ (pinned in `.tool-versions`)
- **Node.js**: 20+ (pinned in `.nvmrc` & `.tool-versions`)
- **Docker & Docker Compose**: Required for local infrastructure stack
- **Make**: Standard build automation tool

---

## Developer Command Surface

Execute all operations via `make`:

```bash
make help        # List all available targets and descriptions
make bootstrap   # Verify environment and install Go/Node dependencies
make dev         # Start all local Go microservices and frontend concurrently
make seed        # Seed local catalogs, sample journeys, and verified contact lists
make build       # Compile Go binaries and Vite web assets
make lint        # Run go vet and TypeScript typecheck
make unit        # Run Go unit tests and Vitest component suite
make verify      # Single quality entry point (lint + unit + build)
make acceptance  # Execute Playwright end-to-end browser test suite
make check-stack # Health check and smoke test all configured service ports
make reset       # Teardown and reset local databases and artifacts
make down        # Stop all running containers and dev background processes
```

---

## Configuring Custom Ports via `.env`

If any default ports (e.g. 8080, 5432, 3000, 9092, 7233) conflict with services already running on your system:

1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```
2. Open `.env` and customize any of the port environment variables (`CONTROL_API_PORT`, `FRONTEND_PORT`, `POSTGRES_PORT`, `KAFKA_PORT`, `TEMPORAL_PORT`, `FAKE_PROVIDER_PORT`, etc.).
3. All `make` commands (`make dev`, `make seed`, `make check-stack`, `make down`) automatically read `.env` and propagate the port overrides across Docker Compose, Go binaries, shell scripts, and the frontend dev server.

---

## Local Infrastructure Stack

Local services run in hermetic Docker containers exposed on loopback `127.0.0.1` (configurable via `.env`):

| Service | Local URL / Endpoint | Default Port | Description |
| :--- | :--- | :--- | :--- |
| **Frontend UI** | [http://localhost:3000](http://localhost:3000) | `FRONTEND_PORT` (3000) | xyflow journey authoring & reporting canvas |
| **Control API** | [http://localhost:8080/healthz](http://localhost:8080/healthz) | `CONTROL_API_PORT` (8080) | REST Control API |
| **Temporal UI** | [http://localhost:8233](http://localhost:8233) | `TEMPORAL_UI_PORT` (8233) | Temporal Workflow Execution UI |
| **Mailpit UI** | [http://localhost:8025](http://localhost:8025) | `MAILPIT_UI_PORT` (8025) | Local email capture sandbox |
| **MinIO Console** | [http://localhost:9001](http://localhost:9001) | `MINIO_CONSOLE_PORT` (9001) | Object store console (User: `minioadmin` / Pass: `minioadmin`) |
| **Fake Provider** | [http://localhost:8082/health](http://localhost:8082/health) | `FAKE_PROVIDER_PORT` (8082) | Channel fake HTTP provider |

---

## Step-by-Step Local Golden Path

Follow these steps to run the complete journey lifecycle locally:

### 1. Bootstrap & Start Services
```bash
make bootstrap
make dev
```

### 2. Seed Initial Data
```bash
make seed
```
This provisions default catalogs, verified static lists, and the sample **Onboarding Journey** (`draft_onboarding_journey`).

### 3. Emit a Test Kafka Event
Emit a `signup` event for a test subject using curl:
```bash
curl -X POST http://localhost:8080/api/v1/events/emit \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant_default" \
  -d '{
    "event_type": "user_signup",
    "subject_ref": "user_1",
    "payload": {
      "locale": "en-US",
      "tier": "free"
    }
  }'
```

### 4. Inspect Workflow Execution
Open [Temporal UI at http://localhost:8233](http://localhost:8233) to trace `CompiledJourneyWorkflow` execution and node visits.

### 5. Inspect Fake Channel Outputs
Open [Mailpit at http://localhost:8025](http://localhost:8025) or inspect the fake provider request ledger:
```bash
curl http://localhost:8082/api/v1/fake-provider/ledger
```

### 6. Submit Outcome Callbacks & View Experiment Report
Ingest a conversion click event:
```bash
curl -X POST http://localhost:8080/api/v1/ingest \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant_default" \
  -d '{
    "event_type": "click",
    "subject_ref": "user_1",
    "action_id": "act_email_welcome",
    "experiment_id": "exp_onboarding_split",
    "variant_key": "treatment"
  }'
```
Inspect aggregate experiment metrics in the UI at [http://localhost:3000](http://localhost:3000) or via API:
```bash
curl http://localhost:8080/api/v1/reports/experiments/exp_onboarding_split
```

---

## Verification & Quality Assurance

Run the quality gate:
```bash
make verify
```

Notice: Local adapters send **no real external emails, SMS, or webhooks**. Production infrastructure administration is explicitly excluded.
