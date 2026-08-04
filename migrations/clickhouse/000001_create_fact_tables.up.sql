-- 000001_create_fact_tables.up.sql
-- Create append-only ClickHouse fact tables with test mode partitioning (is_test)

CREATE TABLE IF NOT EXISTS fact_assignments (
    assignment_id String,
    tenant_id String,
    experiment_id String,
    experiment_version UInt32 DEFAULT 1,
    journey_id String,
    journey_version UInt32 DEFAULT 1,
    variant_id String,
    subject_id String,
    filter_version String DEFAULT 'v1',
    is_test Bool DEFAULT false,
    assigned_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now64(3),
    context String DEFAULT '{}'
) ENGINE = MergeTree()
PARTITION BY is_test
ORDER BY (tenant_id, experiment_id, experiment_version, variant_id, assigned_at, subject_id);

CREATE TABLE IF NOT EXISTS fact_exposures (
    exposure_id String,
    assignment_id String,
    tenant_id String,
    experiment_id String,
    experiment_version UInt32 DEFAULT 1,
    journey_id String,
    journey_version UInt32 DEFAULT 1,
    variant_id String,
    subject_id String,
    filter_version String DEFAULT 'v1',
    is_test Bool DEFAULT false,
    exposed_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now64(3),
    context String DEFAULT '{}'
) ENGINE = MergeTree()
PARTITION BY is_test
ORDER BY (tenant_id, experiment_id, experiment_version, variant_id, exposed_at, subject_id);

CREATE TABLE IF NOT EXISTS fact_action_attempts (
    attempt_id String,
    run_id String,
    node_id String,
    activity_type String,
    tenant_id String,
    experiment_id String,
    experiment_version UInt32 DEFAULT 1,
    journey_id String,
    journey_version UInt32 DEFAULT 1,
    variant_id String,
    subject_id String,
    attempt_number UInt32 DEFAULT 1,
    status String DEFAULT 'success',
    filter_version String DEFAULT 'v1',
    is_test Bool DEFAULT false,
    attempted_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now64(3),
    metadata String DEFAULT '{}'
) ENGINE = MergeTree()
PARTITION BY is_test
ORDER BY (tenant_id, journey_id, journey_version, attempted_at, run_id);

CREATE TABLE IF NOT EXISTS fact_accepted_sends (
    send_id String,
    attempt_id String,
    run_id String,
    node_id String,
    channel String DEFAULT 'email',
    tenant_id String,
    experiment_id String,
    experiment_version UInt32 DEFAULT 1,
    journey_id String,
    journey_version UInt32 DEFAULT 1,
    variant_id String,
    subject_id String,
    filter_version String DEFAULT 'v1',
    is_test Bool DEFAULT false,
    accepted_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now64(3),
    metadata String DEFAULT '{}'
) ENGINE = MergeTree()
PARTITION BY is_test
ORDER BY (tenant_id, experiment_id, experiment_version, variant_id, accepted_at, subject_id);

CREATE TABLE IF NOT EXISTS fact_deliveries (
    delivery_id String,
    send_id String,
    run_id String,
    node_id String,
    channel String DEFAULT 'email',
    tenant_id String,
    experiment_id String,
    experiment_version UInt32 DEFAULT 1,
    journey_id String,
    journey_version UInt32 DEFAULT 1,
    variant_id String,
    subject_id String,
    delivery_status String DEFAULT 'delivered',
    filter_version String DEFAULT 'v1',
    is_test Bool DEFAULT false,
    delivered_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now64(3),
    metadata String DEFAULT '{}'
) ENGINE = MergeTree()
PARTITION BY is_test
ORDER BY (tenant_id, experiment_id, experiment_version, variant_id, delivered_at, subject_id);

CREATE TABLE IF NOT EXISTS fact_opens (
    open_id String,
    delivery_id String,
    run_id String,
    tenant_id String,
    experiment_id String,
    experiment_version UInt32 DEFAULT 1,
    journey_id String,
    journey_version UInt32 DEFAULT 1,
    variant_id String,
    subject_id String,
    user_agent String DEFAULT '',
    filter_version String DEFAULT 'v1',
    is_test Bool DEFAULT false,
    opened_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now64(3),
    metadata String DEFAULT '{}'
) ENGINE = MergeTree()
PARTITION BY is_test
ORDER BY (tenant_id, experiment_id, experiment_version, variant_id, opened_at, subject_id);

CREATE TABLE IF NOT EXISTS fact_clicks (
    click_id String,
    delivery_id String,
    run_id String,
    tenant_id String,
    experiment_id String,
    experiment_version UInt32 DEFAULT 1,
    journey_id String,
    journey_version UInt32 DEFAULT 1,
    variant_id String,
    subject_id String,
    target_url String DEFAULT '',
    filter_version String DEFAULT 'v1',
    is_test Bool DEFAULT false,
    clicked_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now64(3),
    metadata String DEFAULT '{}'
) ENGINE = MergeTree()
PARTITION BY is_test
ORDER BY (tenant_id, experiment_id, experiment_version, variant_id, clicked_at, subject_id);

CREATE TABLE IF NOT EXISTS fact_conversions (
    conversion_id String,
    outcome_id String,
    run_id String,
    tenant_id String,
    experiment_id String,
    experiment_version UInt32 DEFAULT 1,
    journey_id String,
    journey_version UInt32 DEFAULT 1,
    variant_id String,
    subject_id String,
    metric_name String,
    value Float64 DEFAULT 1.0,
    unit String DEFAULT 'count',
    filter_version String DEFAULT 'v1',
    is_test Bool DEFAULT false,
    converted_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now64(3),
    metadata String DEFAULT '{}'
) ENGINE = MergeTree()
PARTITION BY is_test
ORDER BY (tenant_id, experiment_id, experiment_version, metric_name, variant_id, converted_at, subject_id);

CREATE TABLE IF NOT EXISTS fact_data_quality (
    quality_event_id String,
    tenant_id String,
    experiment_id String,
    experiment_version UInt32 DEFAULT 1,
    journey_id String,
    journey_version UInt32 DEFAULT 1,
    entity_type String,
    entity_id String,
    rule_id String,
    issue_type String,
    severity String DEFAULT 'error',
    filter_version String DEFAULT 'v1',
    is_test Bool DEFAULT false,
    occurred_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now64(3),
    details String DEFAULT '{}'
) ENGINE = MergeTree()
PARTITION BY is_test
ORDER BY (tenant_id, issue_type, occurred_at);
