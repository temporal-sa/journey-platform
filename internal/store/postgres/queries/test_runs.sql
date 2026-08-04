-- name: CreateTestRun :one
INSERT INTO test_runs (
    tenant_id, test_run_id, draft_id, ir_id, status, mock_inputs, expected_outcomes, actual_outcomes, execution_time_ms, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetTestRun :one
SELECT * FROM test_runs
WHERE tenant_id = $1 AND test_run_id = $2;

-- name: UpdateTestRun :one
UPDATE test_runs
SET status = $3,
    actual_outcomes = $4,
    execution_time_ms = $5,
    updated_at = $6
WHERE tenant_id = $1 AND test_run_id = $2
RETURNING *;
