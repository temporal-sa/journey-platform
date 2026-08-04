-- 000001_create_postgres_tables.up.sql
-- PostgreSQL schemas for journey workflow engine domain tables

CREATE TABLE IF NOT EXISTS catalogs (
    tenant_id VARCHAR(255) NOT NULL,
    record_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    component_type VARCHAR(64) NOT NULL,
    version VARCHAR(64) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    schema_definition JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_hash VARCHAR(64) NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    is_deprecated BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, record_id)
);

CREATE INDEX IF NOT EXISTS idx_catalogs_component_type ON catalogs(tenant_id, component_type);

CREATE TABLE IF NOT EXISTS journey_drafts (
    tenant_id VARCHAR(255) NOT NULL,
    draft_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version INT NOT NULL DEFAULT 1,
    nodes JSONB NOT NULL DEFAULT '[]'::jsonb,
    edges JSONB NOT NULL DEFAULT '[]'::jsonb,
    content_hash VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, draft_id)
);

CREATE TABLE IF NOT EXISTS journey_versions (
    tenant_id VARCHAR(255) NOT NULL,
    version_id VARCHAR(255) NOT NULL,
    draft_id VARCHAR(255) NOT NULL,
    version INT NOT NULL DEFAULT 1,
    entry_node_id VARCHAR(255) NOT NULL DEFAULT '',
    nodes JSONB NOT NULL DEFAULT '[]'::jsonb,
    edges JSONB NOT NULL DEFAULT '[]'::jsonb,
    content_hash VARCHAR(64) NOT NULL DEFAULT '',
    compiled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, version_id),
    CONSTRAINT uk_journey_versions_draft_version UNIQUE (tenant_id, draft_id, version)
);

CREATE INDEX IF NOT EXISTS idx_journey_versions_draft ON journey_versions(tenant_id, draft_id);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    tenant_id VARCHAR(255) NOT NULL,
    key VARCHAR(255) NOT NULL,
    scope VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(64) NOT NULL DEFAULT 'pending',
    response_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, scope, key)
);

CREATE TABLE IF NOT EXISTS kafka_inbox (
    tenant_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    event_id VARCHAR(255) NOT NULL,
    topic VARCHAR(255) NOT NULL,
    partition INT NOT NULL DEFAULT 0,
    offset_val BIGINT NOT NULL DEFAULT 0,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(64) NOT NULL DEFAULT 'received',
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, message_id),
    CONSTRAINT uk_kafka_inbox_topic_partition_offset UNIQUE (tenant_id, topic, partition, offset_val)
);

CREATE INDEX IF NOT EXISTS idx_kafka_inbox_status ON kafka_inbox(tenant_id, status);

CREATE TABLE IF NOT EXISTS target_manifests (
    tenant_id VARCHAR(255) NOT NULL,
    manifest_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    query_spec JSONB NOT NULL DEFAULT '{}'::jsonb,
    total_count BIGINT NOT NULL DEFAULT 0,
    content_hash VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, manifest_id)
);

CREATE TABLE IF NOT EXISTS dispatch_ledger (
    tenant_id VARCHAR(255) NOT NULL,
    dispatch_id VARCHAR(255) NOT NULL,
    manifest_id VARCHAR(255) NOT NULL,
    journey_version_id VARCHAR(255) NOT NULL,
    subject_id VARCHAR(255) NOT NULL,
    status VARCHAR(64) NOT NULL DEFAULT 'dispatched',
    dispatched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, dispatch_id),
    CONSTRAINT uk_dispatch_ledger_boundary UNIQUE (tenant_id, manifest_id, subject_id, journey_version_id)
);

CREATE TABLE IF NOT EXISTS enrollments (
    tenant_id VARCHAR(255) NOT NULL,
    enrollment_id VARCHAR(255) NOT NULL,
    journey_version_id VARCHAR(255) NOT NULL,
    subject_id VARCHAR(255) NOT NULL,
    status VARCHAR(64) NOT NULL DEFAULT 'active',
    current_node_id VARCHAR(255) NOT NULL DEFAULT '',
    state_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, enrollment_id),
    CONSTRAINT uk_enrollments_journey_subject UNIQUE (tenant_id, journey_version_id, subject_id)
);

CREATE INDEX IF NOT EXISTS idx_enrollments_status ON enrollments(tenant_id, status);

CREATE TABLE IF NOT EXISTS subscriptions (
    tenant_id VARCHAR(255) NOT NULL,
    subscription_id VARCHAR(255) NOT NULL,
    enrollment_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    condition_expr TEXT NOT NULL DEFAULT '',
    status VARCHAR(64) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, subscription_id),
    CONSTRAINT uk_subscriptions_enrollment_event UNIQUE (tenant_id, enrollment_id, event_type)
);

CREATE TABLE IF NOT EXISTS action_ledger (
    tenant_id VARCHAR(255) NOT NULL,
    action_id VARCHAR(255) NOT NULL,
    enrollment_id VARCHAR(255) NOT NULL,
    node_id VARCHAR(255) NOT NULL,
    activity_type VARCHAR(128) NOT NULL,
    status VARCHAR(64) NOT NULL DEFAULT 'pending',
    output JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    execution_duration_ms BIGINT NOT NULL DEFAULT 0,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, action_id),
    CONSTRAINT uk_action_ledger_enrollment_node UNIQUE (tenant_id, enrollment_id, node_id)
);

CREATE TABLE IF NOT EXISTS experiment_definitions (
    tenant_id VARCHAR(255) NOT NULL,
    experiment_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(64) NOT NULL DEFAULT 'draft',
    variants JSONB NOT NULL DEFAULT '[]'::jsonb,
    target_audience TEXT NOT NULL DEFAULT '',
    content_hash VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, experiment_id)
);

CREATE TABLE IF NOT EXISTS assignments (
    tenant_id VARCHAR(255) NOT NULL,
    assignment_id VARCHAR(255) NOT NULL,
    experiment_id VARCHAR(255) NOT NULL,
    subject_id VARCHAR(255) NOT NULL,
    variant_id VARCHAR(255) NOT NULL,
    weight_basis_points INT NOT NULL DEFAULT 0,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, assignment_id),
    CONSTRAINT uk_assignments_experiment_subject UNIQUE (tenant_id, experiment_id, subject_id)
);

CREATE TABLE IF NOT EXISTS exposures (
    tenant_id VARCHAR(255) NOT NULL,
    exposure_id VARCHAR(255) NOT NULL,
    experiment_id VARCHAR(255) NOT NULL,
    assignment_id VARCHAR(255) NOT NULL,
    subject_id VARCHAR(255) NOT NULL,
    variant_id VARCHAR(255) NOT NULL,
    weight_basis_points INT NOT NULL DEFAULT 0,
    context JSONB NOT NULL DEFAULT '{}'::jsonb,
    exposed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, exposure_id),
    CONSTRAINT uk_exposures_exp_subj_assign UNIQUE (tenant_id, experiment_id, subject_id, assignment_id)
);

CREATE TABLE IF NOT EXISTS outbox (
    tenant_id VARCHAR(255) NOT NULL,
    id VARCHAR(255) NOT NULL,
    aggregate_type VARCHAR(128) NOT NULL,
    aggregate_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(64) NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_outbox_status ON outbox(tenant_id, status);

CREATE TABLE IF NOT EXISTS static_lists (
    tenant_id VARCHAR(255) NOT NULL,
    list_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    item_count INT NOT NULL DEFAULT 0,
    data_classification VARCHAR(64) NOT NULL DEFAULT 'NonPII',
    items JSONB NOT NULL DEFAULT '[]'::jsonb,
    content_hash VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, list_id)
);

CREATE TABLE IF NOT EXISTS test_runs (
    tenant_id VARCHAR(255) NOT NULL,
    test_run_id VARCHAR(255) NOT NULL,
    draft_id VARCHAR(255) NOT NULL,
    ir_id VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(64) NOT NULL DEFAULT 'pending',
    mock_inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    expected_outcomes JSONB NOT NULL DEFAULT '{}'::jsonb,
    actual_outcomes JSONB NOT NULL DEFAULT '{}'::jsonb,
    execution_time_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, test_run_id)
);

CREATE TABLE IF NOT EXISTS lifecycle_events (
    tenant_id VARCHAR(255) NOT NULL,
    event_id VARCHAR(255) NOT NULL,
    entity_type VARCHAR(128) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, event_id)
);

CREATE INDEX IF NOT EXISTS idx_lifecycle_events_entity ON lifecycle_events(tenant_id, entity_type, entity_id);

CREATE TABLE IF NOT EXISTS tombstones (
    tenant_id VARCHAR(255) NOT NULL,
    tombstone_id VARCHAR(255) NOT NULL,
    entity_type VARCHAR(128) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    deleted_by VARCHAR(255) NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    deleted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, tombstone_id),
    CONSTRAINT uk_tombstones_entity UNIQUE (tenant_id, entity_type, entity_id)
);
