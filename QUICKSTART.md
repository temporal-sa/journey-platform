# Quick Start & Demo Guide

Get up and running with the **Event-Driven Journey Platform** and demo Temporal's core resilience value propositions in under 5 minutes.

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
- **Jaeger Distributed Tracing**: [http://localhost:16686](http://localhost:16686)
- **Mailpit Email Preview**: [http://localhost:8025](http://localhost:8025)

---

## 2. Temporal Value Proposition Demos

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

## 3. Teardown

To cleanly stop all background services and containers:

```bash
make down
```
