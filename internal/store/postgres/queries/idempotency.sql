-- name: CreateIdempotencyKey :one
INSERT INTO idempotency_keys (
    tenant_id, key, scope, status, response_payload, expires_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetIdempotencyKey :one
SELECT * FROM idempotency_keys
WHERE tenant_id = $1 AND scope = $2 AND key = $3;

-- name: UpdateIdempotencyKey :one
UPDATE idempotency_keys
SET status = $4,
    response_payload = $5,
    updated_at = $6
WHERE tenant_id = $1 AND scope = $2 AND key = $3
RETURNING *;

-- name: DeleteExpiredIdempotencyKeys :exec
DELETE FROM idempotency_keys
WHERE expires_at < $1;
