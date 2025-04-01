-- Drop time-series tables
DROP TABLE IF EXISTS ml_prediction_data;
DROP TABLE IF EXISTS alert_data;
DROP TABLE IF EXISTS aggregated_data;
DROP TABLE IF EXISTS timeseries_data;

-- Drop ML-related tables
DROP TABLE IF EXISTS ml_task_bindings;
DROP TABLE IF EXISTS ml_model_metadata;
DROP TABLE IF EXISTS ml_tasks;

-- Drop twin-related tables
DROP TABLE IF EXISTS data_bindings_3d;
DROP TABLE IF EXISTS twin_models_3d;
DROP TABLE IF EXISTS twins;
DROP TABLE IF EXISTS twin_types;

-- Drop project-related tables
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS projects;

-- Drop user-related tables
DROP TABLE IF EXISTS users;

-- Drop extensions
DROP EXTENSION IF EXISTS "uuid-ossp";
DROP EXTENSION IF EXISTS timescaledb; 