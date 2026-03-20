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

	"graphiti"
	"graphiti/internal/adapters/driven/auth"
	"graphiti/internal/adapters/driven/memory"
	"graphiti/internal/adapters/driven/sqlite"
	"graphiti/internal/adapters/driven/webhook"
	"graphiti/internal/adapters/driving/cli"
	"graphiti/internal/ports/driven"
	"graphiti/web/templates"
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

	// Driven adapters: deploy target
	var deployTarget driven.DeployTarget
	if cfg.Deploy.DefaultTarget != "" {
		targetCfg, ok := cfg.Deploy.Targets[cfg.Deploy.DefaultTarget]
		if ok && targetCfg.WebhookURL != "" {
			deployTarget = webhook.NewWebhookDeployTarget(webhook.Config{
				URL:           targetCfg.WebhookURL,
				HMACSecret:    cfg.Deploy.HMACSecret,
				Timeout:       targetCfg.Timeout,
				MaxRetries:    targetCfg.Retries,
				StatusBaseURL: targetCfg.StatusBaseURL,
			})
			slog.Info("deploy target configured", "target", cfg.Deploy.DefaultTarget, "url", targetCfg.WebhookURL)
		}
	}

	// Resolve branding
	branding := templates.Branding{
		AppName:          cfg.Branding.AppName,
		Tagline:          cfg.Branding.Tagline,
		TitleSuffix:      cfg.Branding.TitleSuffix,
		LogoSVG:          cfg.Branding.Logo.SVG,
		ThemeStorageKey:  cfg.Branding.ThemeStorageKey,
		AccentLight:      cfg.Branding.Colors.AccentLight,
		AccentHoverLight: cfg.Branding.Colors.AccentHoverLight,
		AccentDark:       cfg.Branding.Colors.AccentDark,
		AccentHoverDark:  cfg.Branding.Colors.AccentHoverDark,
	}
	if branding.LogoSVG == "" && cfg.Branding.Logo.Path != "" {
		logoData, err := os.ReadFile(cfg.Branding.Logo.Path)
		if err != nil {
			slog.Warn("failed to read logo file", "path", cfg.Branding.Logo.Path, "error", err)
		} else {
			branding.LogoSVG = string(logoData)
		}
	}
	if cfg.Branding.Favicon != "" {
		if info, err := os.Stat(cfg.Branding.Favicon); err != nil {
			slog.Warn("favicon file not found, ignoring", "path", cfg.Branding.Favicon, "error", err)
		} else if !info.IsDir() {
			branding.FaviconPath = "/favicon.ico"
		}
	}
	if cfg.Branding.CustomCSS != "" {
		if info, err := os.Stat(cfg.Branding.CustomCSS); err != nil {
			slog.Warn("custom CSS file not found, ignoring", "path", cfg.Branding.CustomCSS, "error", err)
		} else if !info.IsDir() {
			branding.CustomCSSPath = "/branding/custom.css"
		}
	}

	// Create the Graphiti app using the library API
	app, err := graphiti.New(graphiti.Config{
		BasePath:               "/",
		NodeDefinitionsPath:    cfg.NodeDefinitions.Path,
		CommandHistoryMaxDepth: cfg.CommandHistory.MaxUndoDepth,
		Branding:               branding,
	}, graphiti.Deps{
		WorkflowRepo:  workflowRepo,
		UserRepo:      userRepo,
		ExecutionRepo: execRepo,
		AuthProvider:  authAdapter,
		DeployTarget:  deployTarget,
		HMACSecret:    cfg.Deploy.HMACSecret,
		SessionSecret: cfg.Auth.Session.Secret,
		SessionMaxAge: time.Duration(cfg.Auth.Session.MaxAge) * time.Second,
		SessionSecure: cfg.Auth.Session.Secure,
	})
	if err != nil {
		slog.Error("failed to initialize graphiti", "error", err)
		os.Exit(1)
	}

	// Start server
	server := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      app.Handler(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		slog.Info("starting server", "addr", cfg.Server.Addr())
		fmt.Printf("\n  %s is running at http://localhost:%d\n\n", branding.AppName, cfg.Server.Port)
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
