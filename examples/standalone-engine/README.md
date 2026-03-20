# Standalone Engine Example

This example demonstrates running Graphiti as a separate service with an independent execution engine. The engine receives deployed workflows via HTTP webhook and reports execution status back via HTTP callbacks.

## Architecture

Three separate services communicate over HTTP:

```
Graphiti (:8080) --webhook--> Engine (:9090) --> PostgreSQL (:5432)
                 <--callback--
```

## Running with Docker Compose

```bash
docker compose up --build
```

## Running Standalone

```bash
# Terminal 1: Start Graphiti
cd ../../
task run

# Terminal 2: Start PostgreSQL
# (use your preferred method)

# Terminal 3: Start the engine
cd examples/standalone-engine
export DATABASE_URL=postgres://user:pass@localhost:5432/graphiti?sslmode=disable
export GRAPHITI_CALLBACK_URL=http://localhost:8080/api/callbacks/execution
export DEPLOY_HMAC_SECRET=your-shared-secret
go run .
```

## See Also

- `../embedded-engine/` — Same functionality, single binary
- `../../README.md` — Full documentation including the ToDo API tutorial
