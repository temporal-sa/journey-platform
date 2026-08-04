-- name: CreateJourneyDraft :one
INSERT INTO journey_drafts (
    tenant_id, draft_id, name, description, version, nodes, edges, content_hash, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetJourneyDraft :one
SELECT * FROM journey_drafts
WHERE tenant_id = $1 AND draft_id = $2;

-- name: ListJourneyDrafts :many
SELECT * FROM journey_drafts
WHERE tenant_id = $1
ORDER BY updated_at DESC;

-- name: UpdateJourneyDraft :one
UPDATE journey_drafts
SET name = $3,
    description = $4,
    version = $5,
    nodes = $6,
    edges = $7,
    content_hash = $8,
    updated_at = $9
WHERE tenant_id = $1 AND draft_id = $2
RETURNING *;

-- name: DeleteJourneyDraft :exec
DELETE FROM journey_drafts
WHERE tenant_id = $1 AND draft_id = $2;

-- name: CreateJourneyVersion :one
INSERT INTO journey_versions (
    tenant_id, version_id, draft_id, version, entry_node_id, nodes, edges, content_hash, compiled_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetJourneyVersion :one
SELECT * FROM journey_versions
WHERE tenant_id = $1 AND version_id = $2;

-- name: GetJourneyVersionByDraftAndVersion :one
SELECT * FROM journey_versions
WHERE tenant_id = $1 AND draft_id = $2 AND version = $3;

-- name: ListJourneyVersionsByDraft :many
SELECT * FROM journey_versions
WHERE tenant_id = $1 AND draft_id = $2
ORDER BY version DESC;
