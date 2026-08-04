-- 000001_create_postgres_tables.down.sql
-- Drop PostgreSQL tables in reverse order of dependency

DROP TABLE IF EXISTS tombstones;
DROP TABLE IF EXISTS lifecycle_events;
DROP TABLE IF EXISTS test_runs;
DROP TABLE IF EXISTS static_lists;
DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS exposures;
DROP TABLE IF EXISTS assignments;
DROP TABLE IF EXISTS experiment_definitions;
DROP TABLE IF EXISTS action_ledger;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS enrollments;
DROP TABLE IF EXISTS dispatch_ledger;
DROP TABLE IF EXISTS target_manifests;
DROP TABLE IF EXISTS kafka_inbox;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS journey_versions;
DROP TABLE IF EXISTS journey_drafts;
DROP TABLE IF EXISTS catalogs;
