-- Digital Egiz Database Initialization Script

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "timescaledb";

-- Create database for MLflow if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'mlflow') THEN
        CREATE DATABASE mlflow;
    END IF;
END
$$;

-- Create schemas
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS metadata;
CREATE SCHEMA IF NOT EXISTS history;

-- Create tables

-- Users and authentication
CREATE TABLE IF NOT EXISTS auth.users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth.refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

-- Projects
CREATE TABLE IF NOT EXISTS metadata.projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by UUID NOT NULL REFERENCES auth.users(id)
);

CREATE TABLE IF NOT EXISTS metadata.project_members (
    project_id UUID NOT NULL REFERENCES metadata.projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);

-- Twin types
CREATE TABLE IF NOT EXISTS metadata.twin_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    schema JSONB NOT NULL, -- JSON Schema for twin properties
    project_id UUID NOT NULL REFERENCES metadata.projects(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by UUID NOT NULL REFERENCES auth.users(id),
    CONSTRAINT unique_type_name_per_project UNIQUE (name, project_id)
);

-- Twin instances
CREATE TABLE IF NOT EXISTS metadata.twins (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    thing_id VARCHAR(255) NOT NULL UNIQUE, -- Ditto thing ID
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type_id UUID NOT NULL REFERENCES metadata.twin_types(id),
    project_id UUID NOT NULL REFERENCES metadata.projects(id) ON DELETE CASCADE,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by UUID NOT NULL REFERENCES auth.users(id),
    CONSTRAINT unique_twin_name_per_project UNIQUE (name, project_id)
);

-- 3D Model bindings
CREATE TABLE IF NOT EXISTS metadata.model_3d (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    twin_id UUID NOT NULL REFERENCES metadata.twins(id) ON DELETE CASCADE,
    model_path VARCHAR(255) NOT NULL, -- Path to the glTF model file
    scale NUMERIC NOT NULL DEFAULT 1.0,
    rotation JSONB, -- Rotation in x, y, z
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS metadata.data_bindings_3d (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    model_id UUID NOT NULL REFERENCES metadata.model_3d(id) ON DELETE CASCADE,
    object_name VARCHAR(100) NOT NULL, -- Name of the object in the 3D model
    ditto_path VARCHAR(255) NOT NULL, -- Path to the property in Ditto
    binding_type VARCHAR(50) NOT NULL, -- color, visibility, text, rotation, position, etc.
    mapping_function VARCHAR(255), -- Optional JavaScript function for mapping values
    min_value NUMERIC, -- For range mappings
    max_value NUMERIC, -- For range mappings
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unique_binding_per_object_path UNIQUE (model_id, object_name, ditto_path, binding_type)
);

-- ML Tasks
CREATE TABLE IF NOT EXISTS metadata.ml_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    twin_type_id UUID NOT NULL REFERENCES metadata.twin_types(id),
    input_schema JSONB NOT NULL, -- Schema for input data
    output_schema JSONB NOT NULL, -- Schema for output data
    endpoint VARCHAR(255) NOT NULL, -- REST endpoint for ML service
    kafka_input_topic VARCHAR(255), -- Kafka topic for streaming input
    kafka_output_topic VARCHAR(255), -- Kafka topic for streaming output
    config JSONB, -- Additional configuration
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by UUID NOT NULL REFERENCES auth.users(id),
    CONSTRAINT unique_task_name_per_type UNIQUE (name, twin_type_id)
);

-- Create hypertables for time-series data
CREATE TABLE IF NOT EXISTS history.twin_states (
    time TIMESTAMPTZ NOT NULL,
    twin_id UUID NOT NULL,
    thing_id VARCHAR(255) NOT NULL,
    feature_id VARCHAR(255) NOT NULL,
    property_path VARCHAR(255) NOT NULL,
    value_string TEXT,
    value_number DOUBLE PRECISION,
    value_boolean BOOLEAN,
    value_json JSONB,
    CONSTRAINT fk_twin_states_twin FOREIGN KEY (twin_id) REFERENCES metadata.twins(id) ON DELETE CASCADE
);

-- Convert to hypertable
SELECT create_hypertable('history.twin_states', 'time');

-- Create indexes
CREATE INDEX idx_twin_states_twin_id ON history.twin_states (twin_id, time DESC);
CREATE INDEX idx_twin_states_thing_id ON history.twin_states (thing_id, time DESC);
CREATE INDEX idx_twin_states_feature_property ON history.twin_states (feature_id, property_path, time DESC);

-- Create admin user (password: admin)
INSERT INTO auth.users (username, email, password_hash, first_name, last_name, role)
VALUES ('admin', 'admin@example.com', '$2a$10$jRvbMrY3xQQKX0TUqZ/lSuVK0QIlkl5GpJQ83BzUxpbbM45Ax6OMC', 'Admin', 'User', 'admin')
ON CONFLICT (username) DO NOTHING;

-- Create test project
DO $$
DECLARE
    admin_id UUID;
    project_id UUID;
BEGIN
    SELECT id INTO admin_id FROM auth.users WHERE username = 'admin';
    
    INSERT INTO metadata.projects (name, description, created_by)
    VALUES ('Example Project', 'Default example project', admin_id)
    ON CONFLICT DO NOTHING
    RETURNING id INTO project_id;
    
    IF project_id IS NOT NULL THEN
        INSERT INTO metadata.project_members (project_id, user_id, role)
        VALUES (project_id, admin_id, 'owner')
        ON CONFLICT DO NOTHING;
    END IF;
END
$$; 