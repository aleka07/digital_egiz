# Zonesun ZS-XYZ Filling Machine Example

This example demonstrates how to use the Digital Egiz framework with a simulated Zonesun ZS-XYZ Filling Machine. It includes a data simulator, pre-defined twin type, 3D model with data bindings, and ML integration for anomaly detection.

## Components

- **Data Simulator**: Generates realistic machine data for testing and demonstration
- **Twin Type Definition**: Pre-configured digital twin type specific to filling machines
- **3D Model**: Interactive 3D visualization of the filling machine with data bindings
- **ML Integration**: Example anomaly detection model for predictive maintenance

## Setup

1. Ensure the main Digital Egiz services are running:
   ```
   cd ../../docker
   docker-compose up -d
   ```

2. Start the Zonesun example services:
   ```
   cd ../examples/zonesun-filling-machine
   docker-compose -f docker-compose.override.yml up -d
   ```

3. Access the UI at http://localhost:3000 and log in with:
   - Username: admin
   - Password: admin

4. Navigate to the "Example Project" where the Zonesun Filling Machine twin will already be configured.

## Data Simulation

The simulator generates the following data:

- **Temperature**: Simulated temperature readings from various machine components (20-80°C)
- **Pressure**: Simulated pressure readings from the filling system (0.5-5.0 bar)
- **Fill Level**: Current level of liquid in the reservoir (0-100%)
- **Vibration**: Vibration levels indicating machine health (0.01-2.0 g)
- **Flow Rate**: Rate of liquid flow during filling operations (0-5.0 L/min)
- **Motor Speed**: RPM of main drive motors (0-3000 RPM)
- **Bottle Counter**: Number of bottles processed
- **Bottle Rejected**: Count of bottles rejected for quality issues
- **Status**: Operational status (idle, running, error, maintenance)
- **Error Codes**: Any active error codes

## 3D Model Interactions

The 3D model visualization includes:

- **Color-coded components**: Machine parts change color based on temperature
- **Moving parts**: Visualize machine motion during operation
- **Indicators**: Status lights reflect current machine state
- **Interactive elements**: Click on components to view detailed information
- **Real-time updates**: Model updates based on live data

## ML Integration

The example demonstrates machine learning integration for:

- **Anomaly Detection**: Identifies unusual patterns in sensor data
- **Predictive Maintenance**: Estimates remaining useful life of components
- **Quality Control**: Predicts potential quality issues based on machine parameters

## Customization

You can modify the simulator behavior by editing the configuration files in the `config` directory:

- `machine_config.json`: Machine parameters and constraints
- `simulation_config.json`: Simulation behavior and data generation settings

## API Access

The example exposes the following API endpoints:

- Digital Twin API: http://localhost:8080/api/2/things/zonesun:filling-machine-01
- ML Predictions: http://localhost:8000/predict/anomaly_detection

## Developing Your Own Example

This example serves as a template for creating your own digital twin implementations. Key steps:

1. Define your twin type in the database
2. Create a data simulator or connector to real device
3. Configure the 3D model and data bindings
4. Set up ML models relevant to your application

For more information, see the main Digital Egiz documentation.
