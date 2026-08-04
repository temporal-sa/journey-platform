-- name: CreateTargetManifest :one
INSERT INTO target_manifests (
    tenant_id, manifest_id, name, query_spec, total_count, content_hash, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetTargetManifest :one
SELECT * FROM target_manifests
WHERE tenant_id = $1 AND manifest_id = $2;

-- name: CreateDispatchLedger :one
INSERT INTO dispatch_ledger (
    tenant_id, dispatch_id, manifest_id, journey_version_id, subject_id, status, dispatched_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetDispatchLedger :one
SELECT * FROM dispatch_ledger
WHERE tenant_id = $1 AND dispatch_id = $2;

-- name: CreateEnrollment :one
INSERT INTO enrollments (
    tenant_id, enrollment_id, journey_version_id, subject_id, status, current_node_id, state_data, enrolled_at, updated_at, completed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetEnrollment :one
SELECT * FROM enrollments
WHERE tenant_id = $1 AND enrollment_id = $2;

-- name: UpdateEnrollmentStatus :one
UPDATE enrollments
SET status = $3,
    current_node_id = $4,
    state_data = $5,
    updated_at = $6,
    completed_at = $7
WHERE tenant_id = $1 AND enrollment_id = $2
RETURNING *;

-- name: CreateSubscription :one
INSERT INTO subscriptions (
    tenant_id, subscription_id, enrollment_id, event_type, condition_expr, status, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: ListSubscriptionsByEnrollment :many
SELECT * FROM subscriptions
WHERE tenant_id = $1 AND enrollment_id = $2 AND status = 'active';

-- name: CreateActionLedger :one
INSERT INTO action_ledger (
    tenant_id, action_id, enrollment_id, node_id, activity_type, status, output, error_message, execution_duration_ms, completed_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetActionLedger :one
SELECT * FROM action_ledger
WHERE tenant_id = $1 AND action_id = $2;

-- name: UpdateActionLedgerStatus :one
UPDATE action_ledger
SET status = $3,
    output = $4,
    error_message = $5,
    execution_duration_ms = $6,
    completed_at = $7
WHERE tenant_id = $1 AND action_id = $2
RETURNING *;
