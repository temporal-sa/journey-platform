-- Initial PostgreSQL schema for Journey Platform
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE SCHEMA IF NOT EXISTS journey;

-- Journey Definitions
CREATE TABLE IF NOT EXISTS journey.journeys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    version INT NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    definition JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_journey_name_version UNIQUE (name, version)
);

-- Journey Executions
CREATE TABLE IF NOT EXISTS journey.executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    journey_id UUID NOT NULL REFERENCES journey.journeys(id) ON DELETE CASCADE,
    workflow_id VARCHAR(255) NOT NULL UNIQUE,
    run_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'RUNNING',
    input_payload JSONB,
    state_data JSONB,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Journey Steps / Nodes Log
CREATE TABLE IF NOT EXISTS journey.execution_steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    execution_id UUID NOT NULL REFERENCES journey.executions(id) ON DELETE CASCADE,
    step_id VARCHAR(255) NOT NULL,
    step_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    output JSONB,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Index for fast lookup
CREATE INDEX IF NOT EXISTS idx_executions_journey_id ON journey.executions(journey_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON journey.executions(status);
CREATE INDEX IF NOT EXISTS idx_execution_steps_execution_id ON journey.execution_steps(execution_id);
