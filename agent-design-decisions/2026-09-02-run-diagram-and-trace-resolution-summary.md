# 2026-09-02 Execution Graph Trace, Sub-Run Scoping & Replay Determinism Architecture Summary

## Executive Overview

This release resolves critical issues across execution graph trace visualization, sub-run lifecycle state scoping, timeline resolution, and Temporal replay determinism. Key improvements include strict ID-based graph node/edge traversal without heuristic or index-based fallbacks, removal of synthetic linear DAG derivations, automatic first sub-run selection on test run redirect, elimination of cross-run cache and sub-run ID leakage, enhanced backend timeline resolution across audience sub-runs, and deterministic Temporal workflow replay execution.

---

## 1. Strict Node & Edge Traversal with Branch Isolation (`ExecutionGraphView.tsx`)

- **Prefix-Aware Node ID Normalization**:
  - `visitSteps` emitted by the execution engine use internal intermediate representation IDs prefixed with `ir-` (e.g. `ir-node-start`, `ir-node-welcome-email`), whereas canvas draft definitions use non-prefixed IDs (e.g. `node-start`, `node-welcome-email`).
  - Added bidirectional sanitized ID mapping in `traversedEdgePairs` and `flowEdges`:
    ```tsx
    const isTraversed =
      traversedEdgePairs.has(`${edge.source}->${edge.target}`) ||
      traversedEdgePairs.has(`${sanitize(edge.source)}->${sanitize(edge.target)}`);
    ```
- **Eliminated False `isCompleted` All-Edge Illumination**:
  - Removed legacy fallback logic (`if (step || isCompleted)`) that previously marked all canvas nodes and edges as visited upon workflow completion.
  - Traversed edges (`#4cd7f6` cyan stroke) are now calculated strictly from sequential pairs in `visitSteps`.
- **Removed Heuristic & Index-Based Fallback Matching**:
  - Removed `nameOrTypeMatch` fallback that caused all nodes sharing a type (e.g., all `Email` nodes across true/false branches) to illuminate.
  - Removed `indexMatch` fallback that matched canvas nodes by raw array index position (`s.stepIndex === index + 1`).
  - Implemented strict direct sanitized ID matching: only nodes and edges genuinely executed during the run evaluate to `isVisited = true` / `isTraversed = true`. Untaken decision branches (e.g. unchosen A/B variants, false condition branches, un-triggered timeouts) remain dimmed (`opacity-45`, muted stroke `#464554`).

---

## 2. Removal of Synthetic Linear Graph Fallbacks (`RunDetailPage.tsx`)

- **Explicit Missing Draft State**:
  - Removed the legacy synthetic fallback in `RunDetailPage.tsx` that synthesized dummy linear Email nodes (`derivedNodes = timelineEvents.map(...)`) when a draft failed to load.
  - Introduced an explicit error state card (`graph-missing-draft-state`):
    ```tsx
    <div data-testid="graph-missing-draft-state">
      <span className="material-symbols-outlined text-4xl text-amber-400">warning</span>
      <h3>Journey Canvas Draft Not Available</h3>
      <p>Could not load canvas graph layout for workflow "{parameters.workflow_id}". Switch to Timeline List view to inspect recorded events.</p>
    </div>
    ```

---

## 3. Sub-Run Scoping & Stale State Cleanup (`App.tsx`, `RunDetailPage.tsx`, `public_api.go`)

- **Auto-Selection of First Sub-Run**:
  - When redirecting to a newly launched test execution run (`tr-xxxx`), `RunDetailPage` automatically selects the first sub-run (`subRunsData.sub_runs[0].sub_run_id`) as soon as the directory resolves, preventing the graph from remaining on an unselected placeholder state.
- **Cross-Run State Isolation in `App.tsx`**:
  - Updated `handleStartTestRun` and `handleHashChange` to reset `selectedSubRunId` to `undefined` when navigating to a new run, preventing stale sub-run IDs from previous executions from leaking into new runs.
- **Run-Scoped `placeholderData` in TanStack Query**:
  - Scoped query `placeholderData` to retain cached graphs only when navigating between sub-runs **within the same `activeRunId`**, immediately dropping old timeline data when switching to a completely new run.
- **Backend Sub-Run Validation (`public_api.go`)**:
  - In `GetRunTimeline`, enforced that `targetEntityID` must match or share the `runID` prefix (`strings.HasPrefix(targetEntityID, runID)`), preventing cross-run database event contamination if a client supplies a stale sub-run ID.
  - Enhanced fallback resolution to search across all contact rows (`runID-row-1` through `runID-row-N`) when querying timeline by parent `runID`.

---

## 4. Deterministic Temporal Workflow Search Attributes & Event IDs (`journey_workflow.go`, `wait_for_event.go`)

- **Replay-Safe Search Attributes**:
  - Removed `!workflow.IsReplaying(ctx)` guard from `workflow.UpsertSearchAttributes`. Search attributes emit the `EVENT_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES` event deterministically on both initial runs and history replays, preventing non-determinism panics during worker restart.
- **Deterministic Signal Event IDs**:
  - Replaced non-deterministic `time.Now().UnixNano()` in `convertToEventSignal` with deterministic event generation:
    ```go
    es.EventID = fmt.Sprintf("evt-%s-%d", es.EventType, es.Generation)
    ```

---

## 5. Automated Verification & Test Coverage

- **Frontend Vitest Suite (`web/`)**:
  - `ExecutionGraphView.test.tsx`:
    - `renders traversed edge with cyan stroke and unvisited edge with muted stroke`
    - `accurately resolves ir- prefixed visit steps and only marks taken branch edge as traversed on completed run`
    - `strictly isolates unvisited nodes sharing the same type or array index`
  - `RunDetailPage.test.tsx`:
    - `renders run detail header, parameters, node visits timeline, and action executions`
    - `executes replay / retry trigger mutation on button click`
    - `calls onBackToList when Back to Run List button is clicked`
    - `renders view segment toggle and toggles between Visual Graph Trace and Timeline List`
  - Total: **48/48** frontend component and page unit tests passed.
- **Backend Go Test Suite (`internal/`)**:
  - `internal/activities`: passed.
  - `internal/api/handlers`: passed.
  - `internal/workflows` (including all Temporal replay tests): passed.
  - `internal/compiler`: passed.
  - `internal/domain`: passed.
- **Stack Verification (`check-stack.sh`)**:
  - **13/13** hermetic service health checks passed.
  - Verified live workflow progression through 25s delay timers to final completion with `journey-worker`.
