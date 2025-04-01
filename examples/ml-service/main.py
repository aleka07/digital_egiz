"""
Digital Egiz - ML Service Placeholder

This is a placeholder ML service that demonstrates the integration
with the Digital Egiz framework. It provides a simple API for
making predictions using dummy models.
"""

import os
import json
import logging
import asyncio
from typing import Dict, List, Optional, Any, Union
import uuid
from datetime import datetime

import numpy as np
from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
import joblib
from aiokafka import AIOKafkaConsumer, AIOKafkaProducer

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger("ml-service")

# Create FastAPI app
app = FastAPI(
    title="Digital Egiz ML Service",
    description="API for ML model predictions in the Digital Egiz framework",
    version="0.1.0",
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Kafka configuration
KAFKA_BROKERS = os.getenv("KAFKA_BROKERS", "kafka:9092")
KAFKA_INPUT_TOPIC = "ml.input.v1"
KAFKA_OUTPUT_TOPIC = "ml.output.v1"

# Placeholder for models (in a real implementation, these would be loaded from files)
MODELS = {
    "anomaly_detection": {
        "model": None,  # Placeholder for actual model
        "predict": lambda data: {"anomaly_score": np.random.random(), "is_anomaly": np.random.random() > 0.8},
    },
    "predictive_maintenance": {
        "model": None,  # Placeholder for actual model
        "predict": lambda data: {
            "remaining_useful_life": int(np.random.normal(1000, 200)),
            "failure_probability": np.random.random(),
        },
    },
}

# Pydantic models for API
class PredictionInput(BaseModel):
    model_id: str
    features: Dict[str, Union[float, int, str, bool]]
    metadata: Optional[Dict[str, Any]] = None


class PredictionOutput(BaseModel):
    prediction_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    model_id: str
    predictions: Dict[str, Any]
    timestamp: str = Field(default_factory=lambda: datetime.now().isoformat())
    input_features: Dict[str, Union[float, int, str, bool]]
    metadata: Optional[Dict[str, Any]] = None


class HealthResponse(BaseModel):
    status: str
    version: str
    models: List[str]


# Kafka consumers and producers
kafka_producer = None
kafka_consumer_task = None


async def start_kafka():
    """Start Kafka producer and consumer."""
    global kafka_producer
    try:
        # Create Kafka producer
        kafka_producer = AIOKafkaProducer(bootstrap_servers=KAFKA_BROKERS)
        await kafka_producer.start()
        logger.info("Kafka producer started")

        # Start consumer in background
        asyncio.create_task(consume_kafka_messages())
    except Exception as e:
        logger.error(f"Failed to start Kafka: {e}")
        if kafka_producer:
            await kafka_producer.stop()
            kafka_producer = None


async def consume_kafka_messages():
    """Consume messages from Kafka input topic."""
    try:
        consumer = AIOKafkaConsumer(
            KAFKA_INPUT_TOPIC,
            bootstrap_servers=KAFKA_BROKERS,
            group_id="ml-service",
            auto_offset_reset="latest",
        )
        await consumer.start()
        logger.info(f"Kafka consumer started, listening on {KAFKA_INPUT_TOPIC}")

        try:
            async for msg in consumer:
                try:
                    # Parse message and make prediction
                    payload = json.loads(msg.value.decode())
                    logger.info(f"Received message: {payload}")

                    # Extract model_id and features
                    model_id = payload.get("model_id")
                    features = payload.get("features", {})
                    metadata = payload.get("metadata", {})

                    if not model_id or not features:
                        logger.error("Invalid message format")
                        continue

                    # Make prediction
                    prediction_result = await make_prediction(model_id, features, metadata)

                    # Send result to output topic
                    if kafka_producer:
                        await kafka_producer.send(
                            KAFKA_OUTPUT_TOPIC,
                            json.dumps(prediction_result.dict()).encode(),
                        )
                        logger.info(f"Sent prediction result to {KAFKA_OUTPUT_TOPIC}")
                except Exception as e:
                    logger.error(f"Error processing message: {e}")
        finally:
            await consumer.stop()
    except Exception as e:
        logger.error(f"Kafka consumer error: {e}")


async def make_prediction(model_id: str, features: Dict, metadata: Optional[Dict] = None) -> PredictionOutput:
    """Make a prediction using the specified model."""
    if model_id not in MODELS:
        raise HTTPException(status_code=404, detail=f"Model '{model_id}' not found")

    # Get model and make prediction
    model_info = MODELS[model_id]
    predictions = model_info["predict"](features)

    # Create response
    return PredictionOutput(
        model_id=model_id,
        predictions=predictions,
        input_features=features,
        metadata=metadata,
    )


@app.on_event("startup")
async def startup_event():
    """Initialize the application on startup."""
    await start_kafka()


@app.on_event("shutdown")
async def shutdown_event():
    """Clean up resources on shutdown."""
    if kafka_producer:
        await kafka_producer.stop()


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "version": "0.1.0",
        "models": list(MODELS.keys()),
    }


@app.post("/predict/{model_id}", response_model=PredictionOutput)
async def predict(model_id: str, data: PredictionInput):
    """Make a prediction using the specified model."""
    # Override model_id in the input with the one from the path
    data.model_id = model_id
    return await make_prediction(model_id, data.features, data.metadata)


@app.get("/models", response_model=List[str])
async def get_models():
    """Get a list of available models."""
    return list(MODELS.keys())


if __name__ == "__main__":
    import uvicorn

    uvicorn.run("main:app", host="0.0.0.0", port=8000, reload=True) 