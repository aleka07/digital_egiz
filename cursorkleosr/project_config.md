# Project Configuration (LTM)

This file contains the stable, long-term context for the **Digital Egiz** digital twin framework project. It defines the core goals, technology stack, architectural patterns, and key constraints. It should be updated infrequently, primarily when fundamental aspects of the framework change.

## Core Goal

To create a flexible, scalable, and self-hostable framework named **Digital Egiz** for developing, deploying, and managing digital twins (DTs), built upon Eclipse Ditto. The framework aims to provide:

*   Core digital twin state management and connectivity via Eclipse Ditto.
*   A robust Go (Gin) backend for custom APIs, business logic, historical data management, and orchestration.
*   A rich React-based web user interface for data visualization, twin management, and interactive 3D visualization using Three.js.
*   Integration capabilities for custom ML/AI models (PyTorch recommended) via a standardized REST API for advanced analytics like predictive maintenance and anomaly detection.
*   A Kafka-based data backbone for asynchronous communication and stream processing.
*   Easy deployment for end-users via Docker Compose.
*   An integrated, runnable example based on a Zonesun ZS-XYZ Filling Machine to demonstrate core features and provide a starting point for using **Digital Egiz**.

## Tech Stack

*   **Digital Twin Core:** Eclipse Ditto (State management, Thing interaction, Connectivity)
*   **Backend:**
    *   Language: Go (Golang)
    *   Framework: Gin
    *   Database: PostgreSQL (for metadata, users, configurations) + TimescaleDB extension (for historical time-series data)
    *   Messaging Client: Kafka Client for Go
*   **Frontend:**
    *   Library: React
    *   Language: JavaScript (or TypeScript - TBD by initial setup)
    *   3D Visualization: Three.js
    *   State Management: TBD (e.g., Redux Toolkit, Zustand)
    *   UI Kit: TBD (e.g., Material UI, Ant Design)
*   **ML/AI Integration:**
    *   Recommended Implementation: Python with PyTorch
    *   Deployment: Separate microservice exposing a REST API (FastAPI recommended).
    *   Model Management (Recommended): MLflow (runnable via Docker Compose)
*   **Messaging/Integration Backbone:** Apache Kafka (with Zookeeper)
*   **Deployment (Primary Method):** Docker Compose
*   **Deployment (Advanced Option):** Helm Charts for Kubernetes
*   **Testing:**
    *   Backend (Go): Go standard testing library, `testify` suite
    *   Frontend (React): Jest, React Testing Library (RTL), Playwright/Cypress (for E2E)
    *   ML (Python): PyTest
*   **Linting/Formatting:**
    *   Go: `gofmt`, `golangci-lint`
    *   JS/TS: ESLint, Prettier
    *   Python: `flake8`, `black`, `isort`

## Critical Patterns & Conventions

*   **Architecture:** Microservices (Ditto, Go Backend, React Frontend (served by Nginx), ML Service, Kafka, PostgreSQL/TimescaleDB, MLflow - all containerized).
*   **API Design:**
    *   Go Backend: RESTful API (defined via OpenAPI 3.0 spec as **Digital Egiz Framework API**) for frontend and external interactions. WebSocket support for real-time UI updates.
    *   ML Service: Standardized REST API (`POST /predict/{model_identifier}`) accepting batch instances and returning predictions (see detailed contract discussed).
    *   Ditto: Utilizes standard Ditto APIs (HTTP, WebSocket) for core twin interactions.
*   **Data Flow & Kafka:** Kafka acts as the central bus. Standardized topics (e.g., `ditto.twin.events.v1`, `history.timeseries.v1.<type>`, `ml.input.v1.<task>`, `ml.output.v1.<task>`, `framework.notifications.v1.<type>`) using JSON message format. Go backend consumes from Ditto/ML topics and produces to history/notification/command topics. Short retention policies in Kafka, long-term history in TimescaleDB.
*   **Database Usage:**
    *   PostgreSQL: Stores **Digital Egiz** framework metadata (users, projects, twin types, twin metadata, 3D bindings, ML task configs, audit logs). See defined table structures.
    *   TimescaleDB: Stores historical time-series data ingested from Kafka (likely via a dedicated Go consumer). Structure based on hypertable per data type. Go backend queries for history API.
*   **Authentication & Authorization:** JWT-based authentication handled by the Go backend. Users register/login against the local PostgreSQL DB. API endpoints are protected, requiring a valid Bearer token. Authorization based on user roles within projects (defined in `project_members`).
*   **3D Visualization:**
    *   Models: glTF format recommended.
    *   Data Binding: Configured in the Go backend's `data_bindings_3d` table, linking Ditto feature paths to Three.js object names and defining the mapping logic (color, visibility, text, etc.). Frontend fetches this config and applies updates dynamically.
    *   Optionality: The framework and UI should function without the 3D component if not configured or enabled.
*   **Configuration:** Primarily via configuration files (YAML/TOML) committed with service code, overridden by environment variables (essential for Docker Compose / Helm). Secrets (passwords, JWT secret key) MUST be provided via environment variables or secure mechanisms (e.g., Docker secrets, K8s secrets).
*   **Error Handling:** Standard HTTP error codes from API. Structured JSON error responses. Robust error handling in Go backend for external service calls (Ditto, ML, Kafka, DB) using timeouts, retries, and circuit breakers where appropriate. Kafka consumers use DLQs for unprocessable messages. Detailed structured logging.
*   **Extensibility:**
    *   New Twin Types: Defined via Go API -> `twin_types` table.
    *   New ML Models: User deploys ML service adhering to API contract, configures via Go API -> `ml_tasks` table.
    *   Custom Logic: Recommended via external Kafka consumers/producers interacting with the framework's Kafka topics.
*   **Commit Messages:** Follow Conventional Commits format.

## Key Constraints

*   **Self-Hosted:** The **Digital Egiz** framework is designed to be downloaded and run by users in their own environment using Docker Compose. External network security is the user's responsibility.
*   **Ditto Core Dependency:** Relies on Eclipse Ditto for fundamental twin operations. Ditto's capabilities and limitations must be considered.
*   **Kafka Backbone:** Kafka availability is critical for data flow and inter-service communication.
*   **User-Provided ML:** The framework provides integration points, but users must develop and deploy their own ML models fitting the defined API contract.
*   **Configuration Management:** Correct configuration via environment variables is crucial for deployment. Default passwords should be changed.

## Integrated Example: Zonesun Filling Machine

*   A pre-packaged example (`examples/zonesun-filling-machine/`) demonstrates core **Digital Egiz** functionality.
*   Includes:
    *   A data simulator (Python/Go script) publishing data to Kafka.
    *   Pre-defined Twin Type (`FillingMachine_Zonesun_ZSXYZ`) and instance metadata.
    *   Example 3D model (`.gltf`) and data bindings configuration.
    *   A placeholder ML task configuration for anomaly detection.
    *   Instructions (`README.md`, `docker-compose.override.yml`) for running the example.