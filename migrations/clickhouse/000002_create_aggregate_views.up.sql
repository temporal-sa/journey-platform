-- 000002_create_aggregate_views.up.sql
-- Create aggregated tables and materialized views for experiment and journey metrics

CREATE TABLE IF NOT EXISTS aggregated_experiment_metrics (
    tenant_id String,
    experiment_id String,
    experiment_version UInt32,
    journey_id String,
    journey_version UInt32,
    variant_id String,
    metric_name String,
    filter_version String,
    is_test Bool,
    window_start DateTime64(3, 'UTC'),
    window_end DateTime64(3, 'UTC'),
    assignments_count UInt64,
    exposures_count UInt64,
    sends_count UInt64,
    deliveries_count UInt64,
    opens_count UInt64,
    clicks_count UInt64,
    conversions_count UInt64,
    conversions_sum Float64,
    unique_subjects UInt64,
    watermark DateTime64(3, 'UTC'),
    max_ingested_at DateTime64(3, 'UTC'),
    updated_at DateTime64(3, 'UTC') DEFAULT now64(3)
) ENGINE = SummingMergeTree()
PARTITION BY is_test
ORDER BY (tenant_id, experiment_id, experiment_version, journey_id, journey_version, variant_id, metric_name, filter_version, window_start, window_end);

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_exposures_aggregated
TO aggregated_experiment_metrics
AS SELECT
    tenant_id,
    experiment_id,
    experiment_version,
    journey_id,
    journey_version,
    variant_id,
    'exposure' AS metric_name,
    filter_version,
    is_test,
    toStartOfHour(exposed_at) AS window_start,
    toStartOfHour(exposed_at) + INTERVAL 1 HOUR AS window_end,
    0 AS assignments_count,
    count() AS exposures_count,
    0 AS sends_count,
    0 AS deliveries_count,
    0 AS opens_count,
    0 AS clicks_count,
    0 AS conversions_count,
    0.0 AS conversions_sum,
    uniqExact(subject_id) AS unique_subjects,
    max(exposed_at) AS watermark,
    max(ingested_at) AS max_ingested_at,
    now64(3) AS updated_at
FROM fact_exposures
GROUP BY tenant_id, experiment_id, experiment_version, journey_id, journey_version, variant_id, filter_version, is_test, window_start, window_end;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_conversions_aggregated
TO aggregated_experiment_metrics
AS SELECT
    tenant_id,
    experiment_id,
    experiment_version,
    journey_id,
    journey_version,
    variant_id,
    metric_name,
    filter_version,
    is_test,
    toStartOfHour(converted_at) AS window_start,
    toStartOfHour(converted_at) + INTERVAL 1 HOUR AS window_end,
    0 AS assignments_count,
    0 AS exposures_count,
    0 AS sends_count,
    0 AS deliveries_count,
    0 AS opens_count,
    0 AS clicks_count,
    count() AS conversions_count,
    sum(value) AS conversions_sum,
    uniqExact(subject_id) AS unique_subjects,
    max(converted_at) AS watermark,
    max(ingested_at) AS max_ingested_at,
    now64(3) AS updated_at
FROM fact_conversions
GROUP BY tenant_id, experiment_id, experiment_version, journey_id, journey_version, variant_id, metric_name, filter_version, is_test, window_start, window_end;
