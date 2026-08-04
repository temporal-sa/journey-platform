-- name: RecordLifecycleEvent :one
INSERT INTO lifecycle_events (
    tenant_id, event_id, entity_type, entity_id, event_name, payload, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: ListLifecycleEventsByEntity :many
SELECT * FROM lifecycle_events
WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3
ORDER BY created_at ASC;

-- name: CreateTombstone :one
INSERT INTO tombstones (
    tenant_id, tombstone_id, entity_type, entity_id, deleted_by, reason, deleted_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetTombstone :one
SELECT * FROM tombstones
WHERE tenant_id = $1 AND tombstone_id = $2;

-- name: GetTombstoneByEntity :one
SELECT * FROM tombstones
WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3;
