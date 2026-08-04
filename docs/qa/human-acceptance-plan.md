# Human Application Quality Assurance Acceptance Plan

This executable quality assurance plan defines the step-by-step human verification checklist for the **Journey Platform**. All tests run against the **full live stack with real dependencies** (PostgreSQL, Kafka, Temporal, ClickHouse, MinIO, Mailpit, Fake Provider, and Go microservices) with **zero dry-run or mock fallbacks**.

---

## 1. Environment Setup & Custom Port Configuration

### Service Port Customization
If default ports clash with existing local services on your host, copy `.env.example` to `.env` and customize ports:
```bash
cp .env.example .env
```
The `Makefile` and startup scripts automatically load `.env` and propagate custom ports across Docker Compose, Go microservices, Vite, and health scripts.

### Required Service Endpoints
- **Web Frontend**: `http://localhost:${FRONTEND_PORT:-3000}`
- **Control API Service**: `http://localhost:${CONTROL_API_PORT:-8080}`
- **Event Ingestion API**: `http://localhost:${EVENT_INGRESS_PORT:-8084}`
- **Outcome Ingestion API**: `http://localhost:${OUTCOME_INGRESS_PORT:-8085}`
- **Temporal Web UI**: `http://localhost:${TEMPORAL_UI_PORT:-8233}`
- **Mailpit Web UI (Email Inbox)**: `http://localhost:${MAILPIT_UI_PORT:-8025}`
- **Fake Provider Ledger API**: `http://localhost:${FAKE_PROVIDER_PORT:-8082}`
- **MinIO Storage Console**: `http://localhost:${MINIO_CONSOLE_PORT:-9001}`

---

## 2. Clean Teardown & Fresh Environment Reset

To ensure you are testing against a completely **fresh database** and clean message queue state with no stale artifacts:

### 1. Perform Clean Teardown
Stop all dev processes and purge Docker volumes (PostgreSQL data, Kafka logs, ClickHouse data, MinIO buckets):
```bash
make reset
```
*Alternatively, run:*
```bash
docker compose down -v
rm -rf bin/ .vite/ web/dist/
```

### 2. Launch Full Service Stack
Start the live infrastructure containers, Go microservice binaries, and Vite web frontend:
```bash
make dev
```
Verify stack readiness across all ports:
```bash
make check-stack
```

### 3. Seed Fresh PostgreSQL Database
Populate the fresh PostgreSQL database with golden-path fixture dataset (`draft-101`, `draft_onboarding_journey`, experiments, catalogs, and static contact lists):
```bash
make seed
```
Verify initial PostgreSQL contents:
```bash
docker exec -it journey-postgres psql -U journey -d journeydb -c \
"SELECT tenant_id, draft_id, name, version, content_hash FROM journey_drafts;"
```

---

## 3. Human Acceptance Test Cases

### Case HQA-001: Monorepo Startup, Database Seed & Draft Save Verification
- **Preconditions**: Clean teardown completed (`make reset`), stack running (`make dev`), database seeded (`make seed`).
- **Action**:
  1. Open `http://localhost:3000/journeys` in your browser.
  2. Click into **Welcome Journey Draft** (`draft-101`).
  3. Drag an Email node from the component palette onto the canvas, connect it to the graph, and click **Save Draft**.
  4. Verify real-time modal feedback transitions: `"Saving Journey Draft..."` -> `"Draft Saved!"`.
  5. Open terminal and query PostgreSQL directly:
     ```bash
     docker exec -it journey-postgres psql -U journey -d journeydb -c \
     "SELECT draft_id, version, content_hash, updated_at FROM journey_drafts WHERE draft_id = 'draft-101';"
     ```
  6. Confirm PostgreSQL output shows `version` incremented to `2` with a fresh `content_hash` ETag.
  7. Open a new browser window/tab, navigate to `http://localhost:3000/journeys`, and re-open `draft-101`.
- **Expected Result**: Canvas loads cleanly, Save modal confirms PostgreSQL write, PostgreSQL query returns updated row version 2, and opening the journey from any tab displays all newly saved nodes.
- **Evidence**: Screenshot of canvas editor state and terminal PostgreSQL query output in `artifacts/qa/human-execution-results/HQA-001.png`.

### Case HQA-002: Visual Canvas Graph Editing & Property Inspector
- **Action**: Open `draft_onboarding_journey` on the canvas. Drag an Email node onto canvas, connect to Condition node `true` branch, edit properties in Node Inspector (e.g. Subject Line = `"Welcome to Platform"`), and click **Save Draft**.
- **Expected Result**: Labeled edges connect smoothly, Node Inspector form updates node state, save writes changes to PostgreSQL, and re-querying `journey_drafts` in PostgreSQL displays updated nodes JSON.
- **Evidence**: Inspector screenshot and PostgreSQL nodes JSON snippet in `artifacts/qa/human-execution-results/HQA-002.png`.

### Case HQA-003: Database ETag Optimistic Concurrency Conflict Resolution
- **Action**:
  1. Open `http://localhost:3000/canvas` in **Tab 1** and **Tab 2** simultaneously.
  2. In **Tab 1**: Modify description, click **Save Draft** (PostgreSQL updates to version 2 with new ETag).
  3. In **Tab 2**: (which still holds version 1's ETag), make an edit and click **Save Draft**.
- **Expected Result**: Go `control-api` evaluates `If-Match` against PostgreSQL, detects the ETag mismatch, and returns HTTP 409 Conflict. **Tab 2** displays the **Conflict Resolution Dialog** (`ConflictDialog.tsx`) showing side-by-side local vs server diffs with options to *Keep Local*, *Accept Server*, or *Merge*.
- **Evidence**: Screenshot of ConflictDialog modal in `artifacts/qa/human-execution-results/HQA-003.png`.

### Case HQA-004: Live Event Ingestion & Temporal Workflow Execution
- **Action**:
  1. Post a user signup event via `curl` to the Event Ingestion microservice:
     ```bash
     curl -X POST http://localhost:8084/api/v1/events/emit \
       -H "Content-Type: application/json" \
       -H "X-Tenant-ID: default" \
       -d '{"event_type":"signup","subject_ref":"user_101","payload":{"email":"user101@example.com","locale":"en-US"}}'
     ```
  2. Open Temporal Web UI at `http://localhost:8233`.
- **Expected Result**: Temporal UI displays active `CompiledJourneyWorkflow` execution under `user_101`, showing step-by-step node visit progression, decision branching, and completed execution state.
- **Evidence**: Temporal Web UI execution trace screenshot in `artifacts/qa/human-execution-results/HQA-004.png`.

### Case HQA-005: Channel Fake Provider & Mailpit Delivery Verification
- **Action**:
  1. Open Mailpit Inbox at `http://localhost:8025`.
  2. Inspect the Fake Provider Request Ledger API:
     ```bash
     curl -s http://localhost:8082/api/v1/fake-provider/ledger | jq .
     ```
- **Expected Result**: Mailpit Inbox displays the generated welcome email with personalized attributes. Fake Provider ledger records `accepted` status with matching transaction correlation keys.
- **Evidence**: Mailpit inbox screenshot in `artifacts/qa/human-execution-results/HQA-005.png`.

### Case HQA-006: Experiment Basis-Point Allocation Validator
- **Action**: Open Experiment Inspector (`ExperimentInspector.tsx`) for `exp_onboarding_split`. Change variant weights to 7000 / 3000 basis points (70% / 30%).
- **Expected Result**: Total allocation badge displays `10,000 / 10,000 bps (100%)` in green, enabling the Save button. Setting total != 10,000 displays an error badge and blocks saving.
- **Evidence**: Experiment Inspector screenshot in `artifacts/qa/human-execution-results/HQA-006.png`.

### Case HQA-007: Static List Test Run & Production Data Isolation
- **Action**:
  1. Upload static contact list `list_verified_contacts.csv` via the Static List Modal.
  2. Trigger a Static List Test Run in Forced Variant Coverage mode.
  3. Query PostgreSQL to verify test mode isolation:
     ```bash
     docker exec -it journey-postgres psql -U journey -d journeydb -c \
     "SELECT count(*) FROM journey_drafts WHERE tenant_id = 'default';"
     ```
- **Expected Result**: Frontend displays per-member execution timeline marked with explicit **TEST MODE** badges. Production database tables and ClickHouse aggregate reports show zero test run records (`is_test = true` rows isolated).
- **Evidence**: Test-run detail view screenshot in `artifacts/qa/human-execution-results/HQA-007.png`.

### Case HQA-008: Subject Tombstone & Idempotent Privacy Erasure
- **Action**:
  1. Submit subject tombstone for `user_5`:
     ```bash
     curl -X POST http://localhost:8080/api/v1/tombstones \
       -H "Content-Type: application/json" \
       -H "X-Tenant-ID: default" \
       -d '{"subject_ref":"user_5","reason":"gdpr_erasure_request"}'
     ```
  2. Attempt to emit a signup event for `user_5` via `http://localhost:8084/api/v1/events/emit`.
- **Expected Result**: Ingestion instantly blocks workflow creation and provider dispatch. Action ledger records `suppressed` status with reason `tombstoned_subject`.
- **Evidence**: Tombstone API curl response log in `artifacts/qa/human-execution-results/HQA-008.png`.

---

## 4. Evidence Storage Convention

Save all human execution evidence artifacts (screenshots, curl outputs, SQL query logs) under:
`artifacts/qa/human-execution-results/`
