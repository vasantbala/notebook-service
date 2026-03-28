# Notebook Service

Backend service for notebook-centric RAG workflows.

This service provides:
- Notebook CRUD
- Conversation and message APIs
- Source upload and ingestion tracking
- Streaming chat endpoint with retrieval + citations
- OpenAPI/Swagger docs generated from handler annotations

## Features

- REST API with chi router
- JWT auth middleware with JWKS validation
- Redis-backed rate limiting and message cache
- PostgreSQL persistence for notebooks, conversations, messages, sources, citations
- RAG retrieval delegation to rag-anything
- OpenAI-compatible LLM client integration
- SSE chat streaming endpoint
- Swagger UI endpoint at /swagger/index.html
- Auto-regenerated OpenAPI docs in dev via Air pre-build hook

## Service Flow

### Notebook and Conversation Flow

1. Client authenticates with Bearer token.
2. API middleware validates token and sets user context.
3. Handlers call service layer.
4. Service layer enforces business rules and calls repositories.
5. Repository layer persists and loads data from PostgreSQL.

### Chat Flow

1. Client calls conversation chat endpoint.
2. Service loads message history (Redis cache first, then DB fallback).
3. Service loads ready source document IDs for the notebook.
4. Service calls rag-anything retrieve endpoint for top chunks.
5. Prompt is built from system instructions, retrieved chunks, history, and user query.
6. LLM stream is forwarded to client as SSE tokens.
7. User and assistant messages are persisted; assistant citations are stored.
8. Conversation cache is invalidated.

### Source Ingestion Flow

1. Client uploads file via multipart form.
2. Source row is created in PostgreSQL with pending/processing status.
3. Service posts file to rag-anything upload endpoint.
4. Returned rag doc ID is saved on source record.
5. Source status is updated as ingestion progresses.

## API Docs

- Swagger UI: http://localhost:8080/swagger/index.html
- OpenAPI document: http://localhost:8080/swagger/doc.json

OpenAPI docs are generated from annotations in handler files and main metadata comments.

## Environment Variables

Use `.env.sample` as the template for local configuration.

1. Copy the sample file:

	cp .env.sample .env

2. Update values in `.env` for your environment.

Required values to set:
- JWKS_ENDPOINT
- OPENAI__APIKEY

Common values (see `.env.sample`):
- PORT (default: 8080)
- DATABASE_URL
- REDIS_URL
- RAG_ANYTHING_BASE_URL
- LLM_PROVIDER (openai)
- OPENAI__BASEURL
- OPENAI__MODEL

## Development Setup

Development uses Docker Compose with live rebuild and Air.

Prerequisites:
- Docker and Docker Compose

1. Create a local env file in the project root:

	cp .env.sample .env

2. Update `.env` values as needed (at minimum `JWKS_ENDPOINT` and `OPENAI__APIKEY`).

3. Start dev stack:

	docker compose -f docker-compose.dev.yml up --build

4. Optional: use Compose watch mode:

	docker compose -f docker-compose.dev.yml up --watch

Dev behavior:
- notebook-service container runs Air
- On Go file changes, Air runs swag init, rebuilds, and restarts the app
- Swagger docs update automatically

## Production Setup

Production uses a multi-stage Docker build with a small runtime image.

Prerequisites:
- Docker and Docker Compose

1. Prepare env values from the sample file:

	cp .env.sample .env

2. Set production-safe values in `.env` (especially `JWKS_ENDPOINT`, `OPENAI__APIKEY`, and DB endpoints).

3. Build and run:

	docker compose up --build -d

4. Verify endpoints:

	curl http://localhost:8080/health

Notes:
- App starts on port 8080
- Postgres and Redis are included in docker-compose.yml
- Containers use restart unless-stopped

## Useful Commands

Run tests:

	go test ./...

Run app locally without Docker:

	go run ./cmd/notebook-service/...

Regenerate OpenAPI docs manually:

	swag init -g cmd/notebook-service/main.go -o internal/docs

## Project Layout

- cmd/notebook-service: application entrypoint
- internal/api: router, middleware, handlers
- internal/service: domain logic and integrations
- internal/db: repository interfaces and postgres implementations
- internal/model: domain models and typed errors
- internal/cache: redis cache and rate limiter
- internal/config: environment config loading
- migrations: SQL migrations

