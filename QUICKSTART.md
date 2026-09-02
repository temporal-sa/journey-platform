# Quick Start & Demo Guide

Get up and running with the **Event-Driven Journey Platform**, author test workflows via **WebMCP**, execute static audience test runs, and demo Temporal's core resilience value propositions in under 5 minutes.

---

## 1. Prerequisites & Boot Stack

Ensure **Docker Desktop** is running, then start the entire environment:

```bash
# 1. Start Docker containers, Go services, and Vite frontend
make dev

# 2. Verify all 13 services are healthy
./scripts/check-stack.sh

# 3. Start the Temporal Journey Worker daemon
curl -X POST http://localhost:8087/api/v1/worker/start
```

### Core Web Interfaces
- **Journey Canvas & Control Center**: [http://localhost:3002](http://localhost:3002)
- **Temporal Web UI**: [http://localhost:8233](http://localhost:8233)
- **WebMCP JSON-RPC Interface**: [http://localhost:3002/mcp](http://localhost:3002/mcp) (or port `8087`)
- **Jaeger Distributed Tracing**: [http://localhost:16686](http://localhost:16686)
- **Mailpit Email Preview**: [http://localhost:8025](http://localhost:8025)

---

## 2. Author a Test Case / Journey via WebMCP

The platform provides a **WebMCP standard interface** (`POST /mcp` JSON-RPC 2.0) allowing AI agents and developers to programmatically create, inspect, and test journey DAGs.

### Create a Journey Draft (`create_journey_draft`)
```bash
curl -s -X POST http://localhost:3002/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "create_journey_draft",
      "arguments": {
        "name": "Onboarding VIP Flow",
        "description": "Welcome email and VIP condition check",
        "nodes": [
          {"id": "node-start", "name": "User Signup", "type": "trigger", "position": {"x": 100, "y": 100}},
          {"id": "node-email", "name": "Welcome Email", "type": "Email", "config": {"template_id": "tmpl_welcome", "subject": "Welcome aboard!"}, "position": {"x": 100, "y": 300}},
          {"id": "node-exit", "name": "Exit", "type": "Exit", "position": {"x": 100, "y": 500}}
        ],
        "edges": [
          {"id": "e1", "source": "node-start", "target": "node-email"},
          {"id": "e2", "source": "node-email", "target": "node-exit"}
        ]
      }
    }
  }' | jq .
```

### Inspect Available WebMCP Tools (`tools/list`)
```bash
curl -s -X POST http://localhost:3002/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}}' | jq .
```

---

## 3. Start a Static Audience Test Run

Static test runs simulate candidate journeys against immutable contact CSV datasets without emitting external emails or SMS.

### Option A: Via Web UI
1. Navigate to **Journeys Directory** ([http://localhost:3002/#/journeys](http://localhost:3002/#/journeys)) and click on any journey.
2. Click **Test Run** in the canvas header.
3. Select an **Audience Source** (e.g. `Test Audience List` or upload a custom CSV).
4. Choose **Execution Mode** (`Realistic` or `Forced Variant Coverage`) and click **Launch Test Run**.
5. You are redirected to **Run Details** ([http://localhost:3002/#/run-detail](http://localhost:3002/#/run-detail)), where the first contact sub-run is automatically selected, and the execution trace graph auto-refreshes in real time.

### Option B: Via WebMCP / API
```bash
# Trigger execution via WebMCP
curl -s -X POST http://localhost:3002/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "trigger_test_run",
      "arguments": {
        "draft_id": "draft-signup-welcome-purchase-nudge",
        "static_list_id": "list-static-001"
      }
    }
  }' | jq .

# Or directly via REST API
curl -s -X POST http://localhost:8087/api/v1/test-runs \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: default" \
  -d '{"draft_id": "draft-signup-welcome-purchase-nudge", "static_list_id": "list-static-001"}' | jq .
```

---

## 4. Temporal Value Proposition Demos

Open the **Developer Panel** in the bottom bar of [http://localhost:3002](http://localhost:3002) (or press `Ctrl+Shift+D`).

---

### Demo A: Activity Failure & Self-Healing Retries
*Demonstrates Temporal's ability to withstand downstream API/provider outages with exponential backoff and resume seamlessly once recovered.*

1. **Inject Failure**: In the Developer Panel, toggle **Simulated Activity Failure** to **ON** (or run `curl -X POST http://localhost:8087/api/v1/simulation/activity-failure -H "Content-Type: application/json" -d '{"simulated_activity_failure": true}'`).
2. **Launch Workflow**: Go to **Journeys** $\rightarrow$ open `Signup Welcome & Purchase Nudge Journey` $\rightarrow$ click **Test Run** $\rightarrow$ **Launch Test Run**.
3. **Inspect in Temporal UI**: Open [http://localhost:8233](http://localhost:8233) and click the active workflow. Observe the activity (`ExecuteActionGateway`) failing and retrying with exponential backoff (`Attempt 1`, `Attempt 2`, ...). The workflow state is preserved durably.
4. **Self-Heal**: Toggle **Simulated Activity Failure** to **OFF**.
5. **Result**: On the next retry attempt, Temporal immediately succeeds and the workflow execution progresses to completion without dropping state or restarting from scratch.

---

### Demo B: Worker Crash & Process Restart Resilience
*Demonstrates zero state loss when application worker processes or containers crash mid-execution.*

1. **Launch Workflow**: Start a test run on a workflow with a delay timer (e.g. `Signup Welcome & Purchase Nudge Journey`, which contains a 25s delay).
2. **Kill Worker Process**: In the Developer Panel, click **Stop Worker** (or run `curl -X POST http://localhost:8087/api/v1/worker/stop`).
3. **Verify Worker is Offline**:
   ```bash
   curl -s http://localhost:8087/api/v1/worker/status | jq .
   # Output: {"status": "stopped", "uptime_seconds": 0}
   ```
4. **Inspect Temporal UI**: Open [http://localhost:8233](http://localhost:8233). Notice the workflow execution timer (`TimerStarted`) continues ticking safely on Temporal Server.
5. **Restart Worker**: Click **Start Worker** (or run `curl -X POST http://localhost:8087/api/v1/worker/start`).
6. **Result**: The revived worker automatically polls the task queue and resumes the workflow at the exact next step without missing a beat.

---

### Demo C: Latency Injection & Asynchronous Non-Blocking Execution
*Demonstrates Temporal executing long-running activities asynchronously without blocking HTTP threads or timing out API clients.*

1. **Inject Latency**: In the Developer Panel, set **Activity Latency** to **1000ms** or **2000ms** (or run `curl -X POST http://localhost:8087/api/v1/simulation/activity-latency -H "Content-Type: application/json" -d '{"latency_ms": 1500}'`).
2. **Launch Multi-Contact Run**: Run a test execution using a multi-member static list (`Test Audience List`).
3. **Observe Live Auto-Refresh**: On the **Run Detail** page ([http://localhost:3002/#/run-detail](http://localhost:3002/#/run-detail)), watch the trace graph and sub-runs directory update smoothly in real time as each activity resolves.
4. **Reset Latency**: Set latency back to `0ms`.

---

## 5. Teardown

To cleanly stop all background services and containers:

```bash
make down
```
