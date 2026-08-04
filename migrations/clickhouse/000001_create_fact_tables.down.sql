-- 000001_create_fact_tables.down.sql
-- Drop ClickHouse fact tables

DROP TABLE IF EXISTS fact_data_quality;
DROP TABLE IF EXISTS fact_conversions;
DROP TABLE IF EXISTS fact_clicks;
DROP TABLE IF EXISTS fact_opens;
DROP TABLE IF EXISTS fact_deliveries;
DROP TABLE IF EXISTS fact_accepted_sends;
DROP TABLE IF EXISTS fact_action_attempts;
DROP TABLE IF EXISTS fact_exposures;
DROP TABLE IF EXISTS fact_assignments;
