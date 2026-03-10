package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"graphiti/internal/adapters/driven/auth"
	"graphiti/internal/adapters/driven/filesystem"
	"graphiti/internal/adapters/driven/memory"
	"graphiti/internal/adapters/driven/sqlite"
	"graphiti/internal/adapters/driven/webhook"
	"graphiti/internal/adapters/driving/cli"
	httpAdapter "graphiti/internal/adapters/driving/http"
	"graphiti/internal/app"
	"graphiti/internal/ports/driven"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := cli.Load("config/app.yaml", "config/auth.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Driven adapters: storage
	var workflowRepo driven.WorkflowRepository
	var userRepo driven.UserRepository
	var execRepo driven.ExecutionRepository

	switch cfg.Storage.Adapter {
	case "sqlite":
		os.MkdirAll("data", 0o755)
		db, err := sqlite.NewDB(cfg.Storage.SQLite.Path, "migrations")
		if err != nil {
			slog.Error("failed to open database", "error", err)
			os.Exit(1)
		}
		defer db.Close()
		workflowRepo = sqlite.NewWorkflowRepository(db)
		userRepo = sqlite.NewUserRepository(db)
		execRepo = sqlite.NewExecutionRepository(db)
	default:
		slog.Info("using in-memory storage")
		workflowRepo = memory.NewWorkflowRepo()
		userRepo = memory.NewUserRepo()
		execRepo = memory.NewExecutionRepository()
	}

	// Driven adapters: node definitions
	nodeLoader := filesystem.NewNodeLoader(cfg.NodeDefinitions.Path, nil)

	// Driven adapters: auth
	var authAdapter driven.AuthProvider
	switch cfg.Auth.Provider {
	case "github":
		authAdapter = auth.NewGitHubAuth(auth.GitHubConfig{
			ClientID:     cfg.Auth.GitHub.ClientID,
			ClientSecret: cfg.Auth.GitHub.ClientSecret,
			Scopes:       cfg.Auth.GitHub.Scopes,
			AllowedOrgs:  cfg.Auth.GitHub.AllowedOrgs,
		})
		slog.Info("using GitHub OAuth2 authentication")
	default:
		authAdapter = auth.NewFakeAuth()
		slog.Info("using fake authentication (dev mode)")
	}

	// App services
	nodeRegistry := app.NewNodeRegistry(nodeLoader)
	if err := nodeRegistry.Load(context.Background()); err != nil {
		slog.Error("failed to load node definitions", "error", err)
		os.Exit(1)
	}

	workflowSvc := app.NewWorkflowService(workflowRepo, nodeRegistry, cfg.CommandHistory.MaxUndoDepth)
	execSvc := app.NewExecutionService(execRepo)

	// WebSocket hub
	wsHub := httpAdapter.NewWebSocketHub()

	// Wire WebSocket notifier to execution service
	execSvc.SetNotifier(func(workflowID string, cb app.ExecutionCallback) {
		wsHub.BroadcastToWorkflow(workflowID, httpAdapter.WebSocketMessage{
			Type:   "node_status",
			RunID:  cb.RunID,
			NodeID: cb.NodeID,
			Status: cb.Status,
		})
	})

	// Wire deploy targets from config
	if cfg.Deploy.DefaultTarget != "" {
		targetCfg, ok := cfg.Deploy.Targets[cfg.Deploy.DefaultTarget]
		if ok && targetCfg.WebhookURL != "" {
			deployTarget := webhook.NewWebhookDeployTarget(webhook.Config{
				URL:        targetCfg.WebhookURL,
				HMACSecret: cfg.Deploy.HMACSecret,
				Timeout:    targetCfg.Timeout,
				MaxRetries: targetCfg.Retries,
			})
			workflowSvc.SetDeployTarget(deployTarget)
			slog.Info("deploy target configured", "target", cfg.Deploy.DefaultTarget, "url", targetCfg.WebhookURL)
		}
	}

	// Session store
	sessionMaxAge := time.Duration(cfg.Auth.Session.MaxAge) * time.Second
	sessionStore := httpAdapter.NewSessionStore(cfg.Auth.Session.Secret, sessionMaxAge, cfg.Auth.Session.Secure)

	// HTTP router
	router := httpAdapter.NewRouter(httpAdapter.RouterDeps{
		WorkflowSvc:  workflowSvc,
		ExecutionSvc: execSvc,
		NodeRegistry: nodeRegistry,
		AuthProvider: authAdapter,
		UserRepo:     userRepo,
		SessionStore: sessionStore,
		WSHub:        wsHub,
		HMACSecret:   cfg.Deploy.HMACSecret,
	})

	// Start server
	server := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		slog.Info("starting server", "addr", cfg.Server.Addr())
		fmt.Printf("\n  Graphiti is running at http://localhost:%d\n\n", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}
