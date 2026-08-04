-- name: CreateCatalog :one
INSERT INTO catalogs (
    tenant_id, record_id, name, component_type, version, description, schema_definition, content_hash, tags, is_deprecated, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: GetCatalog :one
SELECT * FROM catalogs
WHERE tenant_id = $1 AND record_id = $2;

-- name: ListCatalogs :many
SELECT * FROM catalogs
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: UpdateCatalog :one
UPDATE catalogs
SET name = $3,
    component_type = $4,
    version = $5,
    description = $6,
    schema_definition = $7,
    content_hash = $8,
    tags = $9,
    is_deprecated = $10,
    updated_at = $11
WHERE tenant_id = $1 AND record_id = $2
RETURNING *;

-- name: DeleteCatalog :exec
DELETE FROM catalogs
WHERE tenant_id = $1 AND record_id = $2;
