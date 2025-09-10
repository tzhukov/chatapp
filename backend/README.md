# Backend Service

## API Documentation
See `openapi.yaml` for the full API specification.

## Environment Variables
- `DEX_ISSUER_URL`: Dex issuer URL
- `DEX_CLIENT_ID`: OIDC client ID for backend
- `DEX_AUDIENCE`: Expected audience claim in JWT
- `KAFKA_BROKER`: Kafka broker address
- `KAFKA_TOPIC`: Kafka topic name
- `API_PORT`: Port to run the API server

## Running Locally with Tilt
Tilt will automatically load environment variables and start the backend service. See the main project README for details.

## Kafka Setup
Ensure Kafka is running and accessible at the address specified in `KAFKA_BROKER`.

## DexIdp Setup
Ensure Dex is running and configured with the backend client and any required connectors.
