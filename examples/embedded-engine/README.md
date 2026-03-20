# Embedded Engine Example

This example demonstrates running Graphiti as an embedded Go library inside a host application. A single binary serves both the workflow builder UI and a simple echo execution engine on the same port.

## How It Works

- Graphiti is initialized via `graphiti.New()` with in-memory storage and fake auth
- The in-process deploy target receives workflow definitions directly (no HTTP webhooks)
- Execution status is reported back via `app.ReportNodeStatus()` (no HTTP callbacks)
- A single `http.ServeMux` serves both the Graphiti UI and the engine's API

## Running

```bash
cd examples/embedded-engine
go run .
```

Open http://localhost:8080 in your browser. You'll be auto-authenticated in dev mode.

## What Happens When You Deploy

1. Build a workflow in the Graphiti UI
2. Click **Deploy**
3. The deploy function is called directly in-process (no webhook)
4. The engine simulates executing each node, reporting status back to Graphiti
5. Switch to **Execution mode** to see nodes light up in real-time

## Differences from Standalone Mode

| Aspect | Standalone | Embedded |
|--------|-----------|----------|
| Processes | 2+ (Graphiti + Engine) | 1 binary |
| Deploy mechanism | HTTP webhook + HMAC | Function call |
| Status reporting | HTTP callbacks | `app.ReportNodeStatus()` |
| Auth | Independent | Shared |
| Configuration | Webhook URLs, secrets | None needed |
