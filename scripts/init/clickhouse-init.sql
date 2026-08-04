-- Initial ClickHouse schema for Journey Analytics & Materialized Reports
CREATE DATABASE IF NOT EXISTS journeydb;

USE journeydb;

-- Journey Ingested Events
CREATE TABLE IF NOT EXISTS journeydb.journey_events (
    event_id String,
    journey_id String,
    execution_id String,
    event_type String,
    subject_id String,
    payload String,
    timestamp DateTime64(3, 'UTC') DEFAULT now64(3)
) ENGINE = MergeTree()
ORDER BY (timestamp, journey_id, event_type);

-- Journey Execution Outcomes
CREATE TABLE IF NOT EXISTS journeydb.journey_outcomes (
    outcome_id String,
    execution_id String,
    journey_id String,
    step_id String,
    status String,
    details String,
    timestamp DateTime64(3, 'UTC') DEFAULT now64(3)
) ENGINE = MergeTree()
ORDER BY (timestamp, journey_id, status);

-- Aggregated Execution Metrics
CREATE TABLE IF NOT EXISTS journeydb.execution_metrics (
    journey_id String,
    date Date DEFAULT toDate(timestamp),
    timestamp DateTime64(3, 'UTC') DEFAULT now64(3),
    total_executions UInt64,
    successful_executions UInt64,
    failed_executions UInt64,
    avg_duration_ms Float64
) ENGINE = MergeTree()
ORDER BY (date, journey_id);
