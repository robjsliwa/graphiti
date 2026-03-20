// Package graphiti provides an embeddable visual workflow builder UI.
//
// Graphiti lets users design directed workflow graphs through a browser-based
// canvas, then deploy the resulting definitions to an execution engine.
// It can run as a standalone service or be embedded into a Go application
// as a library.
//
// # Quick Start (Standalone)
//
// Run the pre-built server:
//
//	go run graphiti/cmd/server
//
// # Quick Start (Embedded Library)
//
// Mount Graphiti into your own HTTP server:
//
//	app, err := graphiti.New(graphiti.Config{
//	    BasePath: "/",
//	}, graphiti.Deps{
//	    WorkflowRepo: myWorkflowRepo,
//	    UserRepo:     myUserRepo,
//	    AuthProvider: myAuthProvider,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	http.Handle("/", app.Handler())
//
// # Architecture
//
// Graphiti uses hexagonal architecture. The host application provides
// driven adapters (storage, auth, deploy targets) through the [Deps] struct.
// Graphiti provides the domain logic, UI, and HTTP handlers.
//
// All repository interfaces are defined in the internal/ports/driven package.
// Reference implementations for SQLite and in-memory storage are included
// in the internal/adapters/driven subpackages.
//
// # Node Definitions
//
// Workflow node types are defined in YAML configuration files, not in Go code.
// See the config/nodes/ directory for examples, or provide your own via
// [Config.NodeDefinitionsPath]. When no path is specified, Graphiti uses its
// embedded default set of 11 node definitions.
//
// # Execution Integration
//
// Graphiti communicates with execution engines in two ways:
//
//   - Standalone mode: versioned webhook payloads via [Deps.DeployTarget]
//   - Embedded mode: direct function calls via the inprocess adapter
//
// For live execution status updates, use [App.ReportNodeStatus] in embedded
// mode or POST to /api/callbacks/execution in standalone mode.
package graphiti
