---
name: investigate-system-issues
description: Investigates runtime behavior, condition evaluations, data flows, Jaeger traces, service logs, and workflow execution anomalies across APIs, OpenTelemetry, Temporal gRPC event histories, and PostgreSQL databases through hypothesis formulation and empirical verification without code changes.
---

# System Issue Investigation & Empirical Verification Skill

Use this skill when investigating bugs, workflow branch routing mismatches, condition evaluation discrepancies, or data flow anomalies across the event-driven journey engine stack.

> [!IMPORTANT]
> **Core Objective**: Formulate empirical hypotheses and verify them using live APIs, database queries, Jaeger distributed traces, service logs, and Temporal gRPC history inspection **WITHOUT changing existing production code** until requested.

---

## Workflow Steps

```mermaid
flowchart TD
    A["User Issue Report / Symptom"] --> B["1. Formulate Hypotheses"]
    B --> C["2. Query PostgreSQL Database State"]
    C --> D["3. Query Jaeger Traces & Spans"]
    D --> E["4. Inspect Structured Service Logs"]
    E --> F["5. Inspect Temporal gRPC Workflow Histories"]
    F --> G["6. Trace End-to-End Payload Flow"]
    G --> H["7. Synthesize Empirical Findings & Evidence"]
```

---

## 1. Formulate Diagnostic Hypotheses
Before editing code or jumping to conclusions, enumerate potential root causes across layers:
- **UI / API Payload Layer**: Was `static_list_id` or proper input context included in the HTTP request payload, or did it fall back to default mock inputs?
- **Database Layer**: Do the target static list rows, journey draft nodes, or test run records in PostgreSQL contain the expected JSON attributes?
- **Temporal Workflow Layer**: Did Temporal launch 1 workflow per row, or single fallback workflows? Was `EvaluateCondition` passed the flattened context?
- **Branch Evaluation Layer**: Did `EvaluateCondition` evaluate to `true` or `false`, and did the engine route to the correct `sourceHandle` branch (`true` vs `false`)?

---

## 2. Query Live Database State (PostgreSQL & Control API)
Inspect active draft nodes, edges, static list audience items, and test run records in PostgreSQL.

### A. Query REST API Endpoints
```bash
# Check journey drafts and node configurations
curl -s http://localhost:8080/api/v1/journeys/drafts

# Check uploaded static lists
curl -s http://localhost:8080/api/v1/static-lists

# Check test runs
curl -s http://localhost:8080/api/v1/journeys/runs
```

### B. Query PostgreSQL Directly via Docker
```bash
# Inspect static lists items & attributes
docker compose exec -T postgres psql -U journey -d journeydb -c "SELECT list_id, name, item_count, items FROM static_lists;"

# Inspect test run records
docker compose exec -T postgres psql -U journey -d journeydb -c "SELECT test_run_id, draft_id, status, mock_inputs FROM test_runs ORDER BY created_at DESC LIMIT 5;"
```
---

## 3. Query Jaeger Distributed Traces & OpenTelemetry Spans
Use the Jaeger API to inspect HTTP request headers, SQL statements, span tags, and status codes associated with specific `trace_id` or `request_id` values.

### A. Fetch Trace Details by Trace ID
```bash
# Query exact trace details and format with jq
curl -s http://localhost:16686/api/traces/<trace_id> | jq .
```

### B. Search Recent Traces by Service Name
```bash
# Search recent spans for a target service to find operations and HTTP status codes
curl -s "http://localhost:16686/api/traces?service=<service_name>&limit=50" | jq '.data[] | {traceID: .traceID, op: .spans[0].operationName, tags: .spans[0].tags}'
```

---

## 4. Inspect Structured Service Logs
Search runtime service log output streams to correlate `request_id`, `trace_id`, service startup configurations, and error messages.

```bash
# Search log output for specific trace ID or request ID
grep "<trace_id_or_request_id>" /path/to/service.log

# Inspect recent error or warning entries in service logs
grep -E '"level":"(error|warn)"' /path/to/service.log
```

---

## 5. Inspect Temporal gRPC Workflow Event Histories

### Temporal History Inspection Pattern
Create a scratch script in `<artifacts_dir>/scratch/inspect_workflows.go` or `test/inspect_workflows.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

func main() {
	ctx := context.Background()

	c, err := client.Dial(client.Options{
		HostPort:  "127.0.0.1:7233",
		Namespace: "default",
	})
	if err != nil {
		log.Fatalf("Failed to connect Temporal SDK client: %v", err)
	}
	defer c.Close()

	resp, err := c.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: "default",
	})
	if err != nil {
		log.Fatalf("Failed to list workflows: %v", err)
	}

	for idx, wf := range resp.Executions {
		fmt.Printf("\n=== [%d] Workflow ID: %s | Run ID: %s ===\n", idx+1, wf.Execution.WorkflowId, wf.Execution.RunId)
		iter := c.GetWorkflowHistory(ctx, wf.Execution.WorkflowId, wf.Execution.RunId, false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)

		for iter.HasNext() {
			event, err := iter.Next()
			if err != nil {
				break
			}
			switch event.GetEventType() {
			case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:
				attr := event.GetWorkflowExecutionStartedEventAttributes()
				if attr.Input != nil && len(attr.Input.Payloads) > 0 {
					fmt.Printf("  - WORKFLOW_STARTED Input: %s\n", string(attr.Input.Payloads[0].Data))
				}

			case enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
				attr := event.GetActivityTaskScheduledEventAttributes()
				actName := attr.GetActivityType().GetName()
				if actName == "EvaluateCondition" || actName == "ExecuteActionGateway" {
					inputData := ""
					if attr.Input != nil && len(attr.Input.Payloads) > 0 {
						inputData = string(attr.Input.Payloads[0].Data)
					}
					fmt.Printf("  - SCHEDULED [%s]: %s\n", actName, inputData)
				}

			case enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
				attr := event.GetActivityTaskCompletedEventAttributes()
				if attr.Result != nil && len(attr.Result.Payloads) > 0 {
					res := string(attr.Result.Payloads[0].Data)
					if res == "true" || res == "false" || len(res) > 20 {
						fmt.Printf("  - COMPLETED Result: %s\n", res)
					}
				}
			}
		}
	}
}
```

Execute the research script:
```bash
go run ./test/inspect_workflows.go
```

---

## 6. Cross-Layer Verification Checklist

- [ ] **Payload Integrity**: Does the `WORKFLOW_STARTED` input contain the audience member attributes (`tier`, `score`, `region`) or only generic testpack metadata?
- [ ] **Context Flattening**: Does `EvaluateCondition` receive flattened top-level keys (`"tier": "gold"`) alongside nested objects?
- [ ] **Condition Activity Result**: Did `EvaluateCondition` return `true` or `false`?
- [ ] **Node Visit & Action Gateway**: Did `true` route to the node on handle `true` (e.g. Email), and `false` route to the node on handle `false` (e.g. SMS)?
- [ ] **Run Identity Matching**: Are the workflow IDs being compared from the **same test run ID** (e.g. `tr-369711-row-1` vs `tr-369711-row-2`), rather than comparing an older run against a newer run?
- [ ] **Trace & Span Correlation**: Was the Jaeger `trace_id` or `request_id` queried to verify request headers (such as `X-Tenant-ID`), HTTP status codes, and DB query statements?
- [ ] **Log Alignment**: Do the service logs confirm the service port, tenant context, and runtime error messages match the trace findings?

---

## 7. Synthesize Findings
Present findings in a structured report detailing:
1. **Empirical Evidence**: Raw event payloads from `WORKFLOW_STARTED`, `EvaluateCondition`, and `ExecuteActionGateway`.
2. **Confirmed Root Cause**: Precise explanation of why the observed behavior occurred.
3. **Hypothesis Verification**: Confirmation of which hypothesis was proven by the empirical data.
4. **Actionable Recommendations**: Next steps or suggested fixes (without making unrequested code changes).
