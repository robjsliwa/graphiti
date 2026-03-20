// Embedded Engine Example
//
// This example demonstrates running Graphiti as an embedded Go library
// inside a host application. A single binary serves both the workflow
// builder UI and a simple echo execution engine on the same port.
//
// Run it:
//
//	cd examples/embedded-engine
//	go run .
//
// Then open http://localhost:8080 in your browser.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"graphiti"
	"graphiti/internal/adapters/driven/auth"
	"graphiti/internal/adapters/driven/inprocess"
	"graphiti/internal/adapters/driven/memory"
	"graphiti/internal/domain"

	"github.com/google/uuid"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	// In-memory repositories for Graphiti
	workflowRepo := memory.NewWorkflowRepo()
	userRepo := memory.NewUserRepo()
	execRepo := memory.NewExecutionRepository()

	// The engine tracks deployed workflows
	engine := &EchoEngine{
		workflows: make(map[string]*domain.DeployPayload),
	}

	// SetApp is called after graphiti.New() to inject the app reference
	// into the engine so it can report execution status. The deploy callback
	// below does not use graphitiApp until a deploy occurs, which can only
	// happen after the server is listening — well after SetApp is called.
	var mu sync.Mutex
	var graphitiApp *graphiti.App
	setApp := func(a *graphiti.App) {
		mu.Lock()
		defer mu.Unlock()
		graphitiApp = a
	}
	getApp := func() *graphiti.App {
		mu.Lock()
		defer mu.Unlock()
		return graphitiApp
	}

	// In-process deploy target: when the user clicks Deploy in the UI,
	// this function is called directly — no HTTP, no webhook.
	deployTarget := inprocess.NewDeployTarget(
		func(ctx context.Context, payload domain.DeployPayload) (*domain.DeployResult, error) {
			logger.Info("workflow deployed via in-process call",
				"workflow", payload.Workflow.Name,
				"version", payload.Workflow.Version,
			)
			engine.Register(payload)

			// Simulate a quick execution run to demonstrate status reporting
			go func() {
				time.Sleep(100 * time.Millisecond)
				engine.SimulateExecution(getApp(), payload)
			}()

			return &domain.DeployResult{
				Success: true,
				RunID:   uuid.New().String(),
				Message: "Workflow registered in echo engine",
			}, nil
		},
	)

	// Initialize Graphiti as a library
	app, err := graphiti.New(graphiti.Config{
		BasePath:               "/",
		CommandHistoryMaxDepth: 100,
		Logger:                 logger,
	}, graphiti.Deps{
		WorkflowRepo:  workflowRepo,
		UserRepo:      userRepo,
		ExecutionRepo: execRepo,
		AuthProvider:  auth.NewFakeAuth(),
		DeployTarget:  deployTarget,
	})
	if err != nil {
		logger.Error("failed to initialize graphiti", "error", err)
		os.Exit(1)
	}
	setApp(app)

	// Build the combined router
	mux := http.NewServeMux()

	// Mount Graphiti UI and API at root
	mux.Handle("/", app.Handler())

	// Mount the engine's echo endpoint
	mux.HandleFunc("POST /engine/echo", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"echo":      body,
			"timestamp": time.Now().Format(time.RFC3339),
			"engine":    "embedded",
		})
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		fmt.Print("\n  Embedded Graphiti + Engine running at http://localhost:8080\n\n")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// EchoEngine is a minimal execution engine that demonstrates in-process
// integration with Graphiti.
type EchoEngine struct {
	mu        sync.RWMutex
	workflows map[string]*domain.DeployPayload
}

// Register stores a deployed workflow definition.
func (e *EchoEngine) Register(payload domain.DeployPayload) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.workflows[payload.Workflow.ID] = &payload
}

// SimulateExecution walks through the workflow nodes and reports status
// back to Graphiti via the in-process ReportNodeStatus API.
func (e *EchoEngine) SimulateExecution(app *graphiti.App, payload domain.DeployPayload) {
	if app == nil {
		return
	}

	runID := uuid.New().String()
	ctx := context.Background()

	// Get workflow definition to extract node IDs
	defJSON, err := json.Marshal(payload.Workflow.Definition)
	if err != nil {
		slog.Error("failed to marshal definition", "error", err)
		return
	}

	var def struct {
		Nodes []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(defJSON, &def); err != nil {
		slog.Error("failed to unmarshal definition", "error", err)
		return
	}

	// Report each node as running then completed
	for _, node := range def.Nodes {
		now := time.Now()
		if err := app.ReportNodeStatus(ctx, graphiti.ExecutionUpdate{
			RunID:      runID,
			WorkflowID: payload.Workflow.ID,
			NodeID:     node.ID,
			Status:     "running",
			StartedAt:  &now,
		}); err != nil {
			slog.Error("failed to report node running status", "nodeID", node.ID, "error", err)
		}

		time.Sleep(200 * time.Millisecond) // Simulate processing

		completed := time.Now()
		if err := app.ReportNodeStatus(ctx, graphiti.ExecutionUpdate{
			RunID:       runID,
			WorkflowID:  payload.Workflow.ID,
			NodeID:      node.ID,
			Status:      "completed",
			CompletedAt: &completed,
			OutputSummary: map[string]any{
				"node":    node.Label,
				"message": "echo execution completed",
			},
		}); err != nil {
			slog.Error("failed to report node completed status", "nodeID", node.ID, "error", err)
		}
	}

	slog.Info("simulated execution completed", "runID", runID, "nodes", len(def.Nodes))
}
