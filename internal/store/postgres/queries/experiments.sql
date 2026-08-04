-- name: CreateExperimentDefinition :one
INSERT INTO experiment_definitions (
    tenant_id, experiment_id, name, description, status, variants, target_audience, content_hash, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetExperimentDefinition :one
SELECT * FROM experiment_definitions
WHERE tenant_id = $1 AND experiment_id = $2;

-- name: UpdateExperimentDefinition :one
UPDATE experiment_definitions
SET name = $3,
    description = $4,
    status = $5,
    variants = $6,
    target_audience = $7,
    content_hash = $8,
    updated_at = $9
WHERE tenant_id = $1 AND experiment_id = $2
RETURNING *;

-- name: CreateAssignment :one
INSERT INTO assignments (
    tenant_id, assignment_id, experiment_id, subject_id, variant_id, weight_basis_points, assigned_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetAssignment :one
SELECT * FROM assignments
WHERE tenant_id = $1 AND assignment_id = $2;

-- name: GetAssignmentByExperimentSubject :one
SELECT * FROM assignments
WHERE tenant_id = $1 AND experiment_id = $2 AND subject_id = $3;

-- name: CreateExposure :one
INSERT INTO exposures (
    tenant_id, exposure_id, experiment_id, assignment_id, subject_id, variant_id, weight_basis_points, context, exposed_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetExposure :one
SELECT * FROM exposures
WHERE tenant_id = $1 AND exposure_id = $2;
