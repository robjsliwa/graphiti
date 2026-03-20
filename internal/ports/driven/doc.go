// Package driven defines the interfaces that external adapters must implement
// to provide storage, authentication, and deployment capabilities to Graphiti.
//
// These are the "right side" of the hexagonal architecture: they represent
// what Graphiti needs from the outside world. The host application provides
// concrete implementations when calling graphiti.New.
//
// Reference implementations are available in the adapters subpackages:
//
//   - SQLite: graphiti/internal/adapters/driven/sqlite
//   - In-memory: graphiti/internal/adapters/driven/memory
//   - Filesystem (YAML loader): graphiti/internal/adapters/driven/filesystem
//   - Webhook deploy: graphiti/internal/adapters/driven/webhook
//   - In-process deploy: graphiti/internal/adapters/driven/inprocess
//   - Fake auth (dev): graphiti/internal/adapters/driven/auth
//   - GitHub OAuth2: graphiti/internal/adapters/driven/auth
package driven
