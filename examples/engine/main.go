package main

import (
	"crypto/hmac"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	port := getEnv("PORT", "9090")
	dbURL := getEnv("DATABASE_URL", "")
	callbackURL := getEnv("GRAPHITI_CALLBACK_URL", "http://localhost:8080/api/callbacks/execution")
	hmacSecret := getEnv("DEPLOY_HMAC_SECRET", "")

	var db *sql.DB
	if dbURL != "" {
		var err error
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			slog.Error("database ping failed", "error", err)
			os.Exit(1)
		}

		// Create todos table if it doesn't exist
		if err := initDB(db); err != nil {
			slog.Error("failed to initialize database", "error", err)
			os.Exit(1)
		}

		slog.Info("connected to PostgreSQL")
	} else {
		slog.Warn("DATABASE_URL not set - PostgreSQL nodes will fail at runtime")
	}

	runner := NewRunner(db, callbackURL, hmacSecret)

	mux := http.NewServeMux()

	// Deploy endpoint - receives workflow definitions from Graphiti
	mux.HandleFunc("POST /deploy", handleDeploy(runner, hmacSecret))

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"workflows": len(runner.workflows),
		})
	})

	// Deploy status endpoint - Graphiti checks if a workflow is still deployed
	mux.HandleFunc("GET /status/workflows/{id}", handleWorkflowStatus(runner))

	// Catch-all for workflow API routes
	mux.HandleFunc("/api/", handleWorkflowRequest(runner))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	fmt.Printf("\n  Graphiti Sample Engine running on :%s\n", port)
	fmt.Printf("  Deploy endpoint:  POST http://localhost:%s/deploy\n", port)
	fmt.Printf("  Status endpoint:  GET  http://localhost:%s/status/workflows/{id}\n", port)
	fmt.Printf("  Health check:     GET  http://localhost:%s/health\n", port)
	fmt.Printf("  Callback URL:     %s\n\n", callbackURL)

	if err := server.ListenAndServe(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

// handleDeploy receives workflow deploy payloads from Graphiti.
func handleDeploy(runner *Runner, hmacSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		// Verify HMAC if configured
		if hmacSecret != "" {
			sig := r.Header.Get("X-Graphiti-Signature")
			if sig == "" {
				http.Error(w, "missing signature", http.StatusUnauthorized)
				return
			}
			expectedSig := computeHMAC(body, hmacSecret)
			if !strings.HasPrefix(sig, "sha256=") || !hmac.Equal([]byte(sig[7:]), []byte(expectedSig)) {
				http.Error(w, "invalid signature", http.StatusUnauthorized)
				return
			}
		}

		var payload DeployPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if err := runner.Deploy(payload); err != nil {
			http.Error(w, "deploy failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		runID := generateID()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "deployed",
			"runId":  runID,
		})

		slog.Info("workflow deployed via webhook",
			"workflowID", payload.Workflow.ID,
			"name", payload.Workflow.Name,
			"version", payload.Workflow.Version,
		)
	}
}

// handleWorkflowStatus returns the deployment status of a specific workflow.
// HMAC signature is verified if the engine has a secret configured.
func handleWorkflowStatus(runner *Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.PathValue("id")

		// Verify HMAC signature if configured
		if runner.hmacSecret != "" {
			sig := r.Header.Get("X-Graphiti-Signature")
			if sig == "" {
				http.Error(w, "missing signature", http.StatusUnauthorized)
				return
			}
			expectedSig := computeHMAC([]byte(workflowID), runner.hmacSecret)
			if !strings.HasPrefix(sig, "sha256=") || !hmac.Equal([]byte(sig[7:]), []byte(expectedSig)) {
				http.Error(w, "invalid signature", http.StatusUnauthorized)
				return
			}
		}

		runner.mu.RLock()
		deployed, ok := runner.workflows[workflowID]
		runner.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"deployed":   false,
				"workflowId": workflowID,
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"deployed":   true,
			"workflowId": workflowID,
			"version":    deployed.Payload.Workflow.Version,
		})
	}
}

// handleWorkflowRequest processes incoming API requests against deployed workflows.
func handleWorkflowRequest(runner *Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entry, pathParams := runner.MatchRoute(r.Method, r.URL.Path)
		if entry == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "no workflow handles this route",
				"path":  r.URL.Path,
			})
			return
		}

		// Parse request body
		var body map[string]any
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				// If body isn't JSON, that's okay for GET/DELETE
				body = nil
			}
		}

		if pathParams == nil {
			pathParams = make(map[string]any)
		}

		result, err := runner.ExecuteWorkflow(r.Context(), entry, r, pathParams, body)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}

		// Write the response from the HTTP Response node
		statusCode := 200
		if sc, ok := result["statusCode"].(int); ok {
			statusCode = sc
		}
		contentType := "application/json"
		if ct, ok := result["contentType"].(string); ok {
			contentType = ct
		}

		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(statusCode)

		if responseBody, ok := result["body"]; ok {
			json.NewEncoder(w).Encode(responseBody)
		}
	}
}

// initDB creates the todos table for the sample workflow.
func initDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			completed BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func generateID() string {
	return uuid.New().String()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func timeNow() time.Time {
	return time.Now()
}

func timeZero() time.Time {
	return time.Time{}
}
