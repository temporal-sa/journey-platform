-- 000002_create_aggregate_views.down.sql
-- Drop ClickHouse aggregate views and tables

DROP VIEW IF EXISTS mv_conversions_aggregated;
DROP VIEW IF EXISTS mv_exposures_aggregated;
DROP TABLE IF EXISTS aggregated_experiment_metrics;
