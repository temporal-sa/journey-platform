-- name: CreateStaticList :one
INSERT INTO static_lists (
    tenant_id, list_id, name, description, item_count, data_classification, items, content_hash, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetStaticList :one
SELECT * FROM static_lists
WHERE tenant_id = $1 AND list_id = $2;

-- name: UpdateStaticList :one
UPDATE static_lists
SET name = $3,
    description = $4,
    item_count = $5,
    data_classification = $6,
    items = $7,
    content_hash = $8,
    updated_at = $9
WHERE tenant_id = $1 AND list_id = $2
RETURNING *;

-- name: DeleteStaticList :exec
DELETE FROM static_lists
WHERE tenant_id = $1 AND list_id = $2;
