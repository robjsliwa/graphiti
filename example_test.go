package graphiti_test

import (
	"context"
	"fmt"
	"log"
	"net/http/httptest"
	"time"

	"graphiti"
	"graphiti/internal/adapters/driven/auth"
	"graphiti/internal/adapters/driven/inprocess"
	"graphiti/internal/adapters/driven/memory"
	"graphiti/internal/domain"
)

func ExampleNew() {
	app, err := graphiti.New(graphiti.Config{}, graphiti.Deps{
		WorkflowRepo: memory.NewWorkflowRepo(),
		UserRepo:     memory.NewUserRepo(),
		AuthProvider: auth.NewFakeAuth(),
	})
	if err != nil {
		log.Fatal(err)
	}

	// The handler serves the full Graphiti UI and API.
	// Unauthenticated requests redirect to login.
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	fmt.Println(w.Code)
	// Output: 302
}

func ExampleApp_Services() {
	app, err := graphiti.New(graphiti.Config{}, graphiti.Deps{
		WorkflowRepo: memory.NewWorkflowRepo(),
		UserRepo:     memory.NewUserRepo(),
		AuthProvider: auth.NewFakeAuth(),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Use the services API to create a workflow programmatically
	ctx := context.Background()
	wf, err := app.Services().Workflows.CreateWorkflow(ctx, "My Pipeline", "A test workflow", "user-1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(wf.Name)
	// Output: My Pipeline
}

func ExampleNew_withInProcessDeploy() {
	deployTarget := inprocess.NewDeployTarget(
		func(ctx context.Context, payload domain.DeployPayload) (*domain.DeployResult, error) {
			fmt.Printf("Deployed: %s v%d\n", payload.Workflow.Name, payload.Workflow.Version)
			return &domain.DeployResult{Success: true, Message: "registered"}, nil
		},
	)

	_, err := graphiti.New(graphiti.Config{}, graphiti.Deps{
		WorkflowRepo: memory.NewWorkflowRepo(),
		UserRepo:     memory.NewUserRepo(),
		AuthProvider: auth.NewFakeAuth(),
		DeployTarget: deployTarget,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("ok")
	// Output: ok
}

func ExampleApp_ReportNodeStatus() {
	execRepo := memory.NewExecutionRepository()
	execRepo.Create(context.Background(), &domain.ExecutionRun{
		ID:           "run-001",
		WorkflowID:   "wf-001",
		Status:       domain.ExecStatusRunning,
		StartedAt:    time.Now(),
		NodeStatuses: make(map[string]*domain.NodeExecutionStatus),
	})

	app, err := graphiti.New(graphiti.Config{}, graphiti.Deps{
		WorkflowRepo:  memory.NewWorkflowRepo(),
		UserRepo:      memory.NewUserRepo(),
		AuthProvider:  auth.NewFakeAuth(),
		ExecutionRepo: execRepo,
	})
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	err = app.ReportNodeStatus(context.Background(), graphiti.ExecutionUpdate{
		RunID:       "run-001",
		WorkflowID:  "wf-001",
		NodeID:      "node-001",
		Status:      "completed",
		CompletedAt: &now,
	})

	fmt.Println(err)
	// Output: <nil>
}
