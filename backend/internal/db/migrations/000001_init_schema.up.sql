-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- User-related tables
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    active BOOLEAN NOT NULL DEFAULT true,
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Project-related tables
CREATE TABLE projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE project_members (
    id SERIAL PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES projects(id),
    user_id INTEGER NOT NULL REFERENCES users(id),
    role VARCHAR(20) NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, user_id)
);

-- Twin-related tables
CREATE TABLE twin_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    version VARCHAR(50) NOT NULL,
    schema_json JSONB,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE twins (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type_id INTEGER NOT NULL REFERENCES twin_types(id),
    project_id INTEGER NOT NULL REFERENCES projects(id),
    ditto_thing_id VARCHAR(255) UNIQUE NOT NULL,
    metadata_json JSONB,
    has_3d_model BOOLEAN NOT NULL DEFAULT false,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE twin_models_3d (
    id SERIAL PRIMARY KEY,
    twin_id INTEGER UNIQUE NOT NULL REFERENCES twins(id),
    model_url VARCHAR(255) NOT NULL,
    model_format VARCHAR(20) NOT NULL DEFAULT 'gltf',
    version_tag VARCHAR(50),
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE data_bindings_3d (
    id SERIAL PRIMARY KEY,
    twin_id INTEGER NOT NULL REFERENCES twins(id),
    object_name VARCHAR(100) NOT NULL,
    ditto_path VARCHAR(255) NOT NULL,
    binding_type VARCHAR(50) NOT NULL,
    binding_value_map JSONB,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ML-related tables
CREATE TABLE ml_tasks (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    model_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    config_json JSONB,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE ml_task_bindings (
    id SERIAL PRIMARY KEY,
    task_id INTEGER NOT NULL REFERENCES ml_tasks(id),
    twin_id INTEGER NOT NULL REFERENCES twins(id),
    input_mapping_json JSONB,
    output_path_json JSONB,
    schedule_type VARCHAR(20) NOT NULL DEFAULT 'event',
    schedule_config JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE ml_model_metadata (
    id SERIAL PRIMARY KEY,
    model_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    version VARCHAR(50) NOT NULL,
    input_schema JSONB,
    output_schema JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Time-series tables
CREATE TABLE timeseries_data (
    time TIMESTAMP WITH TIME ZONE NOT NULL,
    twin_id VARCHAR(255) NOT NULL,
    feature_path VARCHAR(255) NOT NULL,
    value_type VARCHAR(50) NOT NULL,
    value_num DOUBLE PRECISION,
    value_bool BOOLEAN,
    value_str TEXT,
    value_json JSONB,
    source VARCHAR(255),
    PRIMARY KEY (time, twin_id, feature_path)
);

-- Convert to TimescaleDB hypertable
SELECT create_hypertable('timeseries_data', 'time');

CREATE TABLE aggregated_data (
    time_interval TIMESTAMP WITH TIME ZONE NOT NULL,
    twin_id VARCHAR(255) NOT NULL,
    feature_path VARCHAR(255) NOT NULL,
    interval_type VARCHAR(20) NOT NULL,
    min DOUBLE PRECISION,
    max DOUBLE PRECISION,
    avg DOUBLE PRECISION,
    sum DOUBLE PRECISION,
    count INTEGER,
    first_time TIMESTAMP WITH TIME ZONE,
    last_time TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (time_interval, twin_id, feature_path, interval_type)
);

-- Convert to TimescaleDB hypertable
SELECT create_hypertable('aggregated_data', 'time_interval');

CREATE TABLE alert_data (
    time TIMESTAMP WITH TIME ZONE NOT NULL,
    alert_id VARCHAR(255) NOT NULL,
    twin_id VARCHAR(255) NOT NULL,
    feature_path VARCHAR(255),
    severity VARCHAR(20) NOT NULL,
    message TEXT,
    value_json JSONB,
    source VARCHAR(255),
    acknowledged BOOLEAN NOT NULL DEFAULT false,
    ack_by VARCHAR(255),
    ack_time TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (time, alert_id)
);

-- Convert to TimescaleDB hypertable
SELECT create_hypertable('alert_data', 'time');

CREATE TABLE ml_prediction_data (
    time TIMESTAMP WITH TIME ZONE NOT NULL,
    twin_id VARCHAR(255) NOT NULL,
    task_id VARCHAR(255) NOT NULL,
    prediction_type VARCHAR(50) NOT NULL,
    score_num DOUBLE PRECISION,
    label_str VARCHAR(255),
    details_json JSONB,
    model_version VARCHAR(100),
    PRIMARY KEY (time, twin_id, task_id)
);

-- Convert to TimescaleDB hypertable
SELECT create_hypertable('ml_prediction_data', 'time');

-- Create indexes for time-series data
CREATE INDEX idx_timeseries_twin_feature ON timeseries_data(twin_id, feature_path);
CREATE INDEX idx_alert_twin ON alert_data(twin_id);
CREATE INDEX idx_ml_prediction_twin ON ml_prediction_data(twin_id);

-- Create retained_compression policy (optional, should be configured per deployment)
-- SELECT add_compression_policy('timeseries_data', INTERVAL '7 days');
-- SELECT add_compression_policy('ml_prediction_data', INTERVAL '30 days');
-- SELECT add_compression_policy('alert_data', INTERVAL '90 days');

-- Create initial admin user (password: admin)
INSERT INTO users (email, password, first_name, last_name, role, active)
VALUES ('admin@digital-egiz.com', '$2a$10$qxYI/R6IJdUAaKD1dqsWL.JgVBj7grYvv3XiGdSGmKV2Za8FQp7nS', 'Admin', 'User', 'admin', true); 