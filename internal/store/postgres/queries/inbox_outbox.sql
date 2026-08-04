-- name: SaveKafkaInboxMessage :one
INSERT INTO kafka_inbox (
    tenant_id, message_id, event_id, topic, partition, offset_val, payload, status, processed_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetKafkaInboxMessage :one
SELECT * FROM kafka_inbox
WHERE tenant_id = $1 AND message_id = $2;

-- name: MarkKafkaInboxProcessed :one
UPDATE kafka_inbox
SET status = 'processed',
    processed_at = $3
WHERE tenant_id = $1 AND message_id = $2
RETURNING *;

-- name: CreateOutboxEvent :one
INSERT INTO outbox (
    tenant_id, id, aggregate_type, aggregate_id, event_type, payload, headers, status, retry_count, created_at, processed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetOutboxEvent :one
SELECT * FROM outbox
WHERE tenant_id = $1 AND id = $2;

-- name: ListPendingOutboxEvents :many
SELECT * FROM outbox
WHERE tenant_id = $1 AND status = 'pending'
ORDER BY created_at ASC
LIMIT $2;

-- name: MarkOutboxProcessed :one
UPDATE outbox
SET status = 'processed',
    processed_at = $3
WHERE tenant_id = $1 AND id = $2
RETURNING *;
