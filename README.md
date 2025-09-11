# Real-Time Chat Application

Modern real-time chat stack using:

* Vue 3 SPA frontend (runtime-injected OIDC config)
* Go 1.24 backend (modular packages: `api`, `kafka`, `store`, `oidc`, `metrics`, `models`, `config`)
* Apache Kafka (ingest-first event pipeline)
* MongoDB (idempotent persistence w/ indexes)
* Dex (OIDC Identity Provider)
* Helm + Tilt for local Kubernetes orchestration

For deeper architectural details see `chatapp.md`.

## Architecture Overview

Flow (Kafka‑first ingestion):
1. Frontend obtains tokens from Dex (OIDC Authorization Code Flow via `oidc-client`).
2. User submits a message (REST `POST /api/messages` or WebSocket send).
3. Backend validates JWT, generates server UUID + timestamp, enqueues to Kafka, returns `202 Accepted` (asynchronous persistence + broadcast).
4. Kafka consumer in backend reads message, broadcasts to all WebSocket clients, then persists to MongoDB (idempotent insert on `message_id`).
5. Metrics counters track ingestion, broadcast, websocket connections, and persistence outcomes.

### Components
| Component | Purpose |
|-----------|---------|
| Frontend (Vue 3) | UI, OIDC login, REST + WS client, dynamic runtime config via `config.js` ConfigMap. |
| Backend (Go) | REST API, WebSocket hub, Kafka producer + consumer, OIDC token verification, schema validation, metrics. |
| Kafka | Decouples ingress from broadcast/persist (single topic: `chat-messages`). |
| MongoDB | Durable store with unique index on `message_id`, secondary index on `timestamp`. |
| Dex | OpenID Connect provider issuing JWT access tokens. |

## Project Structure

```
backend/
    src/
        api/        # server, hub, schema validator
        kafka/      # producer/consumer, DLQ logic
        store/      # Mongo repository + adapter
        oidc/       # OIDC verifier + backoff/fallback
        metrics/    # counters + HTTP handler
        models/     # message structs
        config/     # env/runtime config
        main.go
    chart/        # Helm chart for backend
frontend/
    src/          # Vue components, services
    chart/        # Helm chart for frontend
dex/             # Dex Helm values
scripts/         # Test & smoke scripts
Tiltfile         # Local orchestration
```

## Runtime OIDC Configuration
Frontend does NOT bake secrets into the bundle. A `config.js` file (served via ConfigMap) injects:
```
window.__CHATAPP_CONFIG__ = {
    VUE_APP_DEX_ISSUER_URL,
    VUE_APP_DEX_CLIENT_ID,
    VUE_APP_DEX_REDIRECT_URI,
    VUE_APP_DEX_SCOPES,
    VUE_APP_API_BASE_URL,
    VUE_APP_WS_URL
};
```
`chatService.js` lazily creates the OIDC `UserManager` using these runtime values.

## Local Development with Tilt

### Prerequisites
* Docker
* A local Kubernetes cluster (Docker Desktop, Kind, Minikube, etc.)
* Helm
* Tilt
* (Optional) `kubectl` + `jq` for debugging

### Start Everything
```bash
tilt up
```
Tilt will:
* Create namespace `chatapp`.
* Deploy Kafka, MongoDB, Dex via Helm.
* Build & deploy backend + frontend images.
* Expose frontend via port-forward (default: 8081). (Adjust in `Tiltfile` if needed.)
* Register manual test local_resources (`backend-tests`, `frontend-tests`).

### Manual Test Triggers in Tilt
Tests are configured with `trigger_mode=manual` to avoid auto rebuild noise:
* Open the Tilt UI (CLI output shows URL, usually http://localhost:10350/).
* Click ▶ on `backend-tests` or `frontend-tests` to run their scripts.

### Test Scripts (Outside Tilt)
```bash
scripts/test_backend.sh       # Go unit tests
scripts/test_frontend.sh      # Jest unit/integration
scripts/verify_frontend.sh    # Post-deploy smoke (expects 401 on protected endpoints)
```

### Smoke Verification (after `tilt up`)
```bash
scripts/verify_frontend.sh
```
Checks:
* Frontend root served (200)
* `config.js` present
* Dex OIDC discovery reachable
* `GET /api/messages` returns 401 (unauthenticated)
* WebSocket endpoint denies w/out token (handshake response)

## Backend Highlights
* Go 1.24 module (`go.mod` directive `go 1.24`).
* JSON Schema validation via `gojsonschema` (`api/MessageValidator`).
* OIDC verifier with retry/backoff + optional CA injection & fallback issuer logic.
* Kafka write w/ retry (quadratic backoff) + DLQ writer stub.
* Metrics exposed via HTTP handler (counter increments for ingest/broadcast, websocket connections).
* Idempotent Mongo insert (duplicate suppression on `message_id`).
* REST returns `202 Accepted` to decouple client latency from persistence.

## Frontend Highlights
* Vue 3 component set: `ChatWindow`, `MessageInput`, `Sidebar`, `Topbar`.
* Jest + `@vue/test-utils` unit/integration tests (all green).
* OIDC via `oidc-client` with silent renew fallback.
* WebSocket connection includes `?token=` query parameter.

## Running Individual Pieces
Build images manually (optional):
```bash
docker build -t chatapp-backend ./backend
docker build -t chatapp-frontend ./frontend
```

Run only tests locally without Tilt:
```bash
scripts/test_backend.sh
scripts/test_frontend.sh
```

## Troubleshooting
| Issue | Cause | Fix |
|-------|-------|-----|
| `invalid go version '1.24.0'` | Patch version specified in `go.mod` | Use `go 1.24` (no patch) |
| Frontend OIDC error `authority not configured` | Missing `config.js` injection | Ensure ConfigMap + `window.__CHATAPP_CONFIG__` present |
| 404 vs 401 on `/api/messages` | Missing `/api` prefix routing | Confirm backend exposes both `/messages` & `/api/messages` |
| Jest `Vue is not defined` | Browser build of test-utils resolved | CJS mapper + added compiler dependency |

## Stopping
```bash
tilt down
```
or Ctrl+C in the Tilt session.

## Next Ideas
* Add coverage & CI pipeline.
* Add load testing harness (k6 or vegeta) via another Tilt local_resource.
* Expose Prometheus + Grafana stack for metrics visualization.
* Implement message editing / deletion events via new Kafka topics.

---
This README reflects the current Dex + Kafka-first architecture. See `chatapp.md` for the detailed design narrative.
