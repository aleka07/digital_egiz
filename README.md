# Digital Egiz

Digital Egiz is a comprehensive framework for developing, deploying, and managing digital twins (DTs), built upon Eclipse Ditto. It provides a robust infrastructure for digital twin state management, connectivity, and visualization.

## Features

- **Core Digital Twin Management**: Built on Eclipse Ditto for state management and connectivity
- **Go Backend**: RESTful API, business logic, and historical data management using Gin framework
- **React Frontend**: Rich UI for twin management and interactive 3D visualization with Three.js
- **ML/AI Integration**: Standardized REST API for custom models (e.g., predictive maintenance)
- **Kafka Backbone**: Asynchronous communication and stream processing
- **Easy Deployment**: Docker Compose for simple self-hosting

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for development)
- Node.js 18+ and npm/yarn (for frontend development)

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/your-org/digital-egiz.git
   cd digital-egiz
   ```

2. Set up environment variables:
   ```
   cp docker/.env.example docker/.env
   ```

3. Start the services:
   ```
   cd docker
   docker-compose up -d
   ```

4. Access the application:
   - Frontend: http://localhost:3000
   - Ditto API: http://localhost:8080
   - Backend API: http://localhost:8088

## Architecture

Digital Egiz follows a microservices architecture with the following key components:

- **Eclipse Ditto**: Core digital twin engine
- **Go Backend**: Business logic and API services
- **PostgreSQL/TimescaleDB**: Metadata and time-series data storage
- **Kafka**: Messaging backbone
- **React Frontend**: User interface
- **Optional ML Service**: Custom ML model integration

For more details, see the [architecture documentation](./docs/architecture.md).

## Examples

The repository includes the Zonesun ZS-XYZ Filling Machine example to demonstrate core features:

- Preconfigured twin type and instance
- 3D model and data binding
- Data simulator
- Example ML task configuration

See the [example documentation](./examples/zonesun-filling-machine/README.md) for details.

## Documentation

- [Architecture](./docs/architecture.md)
- [API Specification](./docs/api-spec.yaml)
- [Deployment Guide](./docs/deployment.md)
- [Development Guide](./docs/development.md)
- [User Guide](./docs/user-guide.md)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 