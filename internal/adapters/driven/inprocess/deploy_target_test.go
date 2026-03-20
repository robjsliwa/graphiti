package inprocess

import (
	"context"
	"errors"
	"testing"

	"graphiti/internal/domain"
)

func TestDeployTarget_ImplementsInterface(t *testing.T) {
	// Verify at compile time that DeployTarget satisfies driven.DeployTarget.
	// The import is checked by the var _ line in deploy_target.go.
}

func TestDeployTarget_CallsFunctionWithPayload(t *testing.T) {
	var received domain.DeployPayload
	called := false

	target := NewDeployTarget(func(ctx context.Context, payload domain.DeployPayload) (*domain.DeployResult, error) {
		called = true
		received = payload
		return &domain.DeployResult{Success: true, Message: "deployed"}, nil
	})

	payload := domain.DeployPayload{
		APIVersion: "graphiti/v1",
		Event:      "workflow.deployed",
		Workflow: domain.WorkflowInfo{
			ID:      "wf-001",
			Name:    "Test Workflow",
			Version: 3,
		},
	}

	result, err := target.Deploy(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("deploy function was not called")
	}
	if !result.Success {
		t.Fatal("expected success")
	}
	if result.Message != "deployed" {
		t.Errorf("got message %q, want %q", result.Message, "deployed")
	}
	if received.Workflow.ID != "wf-001" {
		t.Errorf("got workflow ID %q, want %q", received.Workflow.ID, "wf-001")
	}
	if received.Workflow.Version != 3 {
		t.Errorf("got version %d, want 3", received.Workflow.Version)
	}
}

func TestDeployTarget_PropagatesErrors(t *testing.T) {
	target := NewDeployTarget(func(ctx context.Context, payload domain.DeployPayload) (*domain.DeployResult, error) {
		return nil, errors.New("engine unavailable")
	})

	_, err := target.Deploy(context.Background(), domain.DeployPayload{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "engine unavailable" {
		t.Errorf("got error %q, want %q", err.Error(), "engine unavailable")
	}
}

func TestDeployTarget_ContextPropagation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	target := NewDeployTarget(func(ctx context.Context, payload domain.DeployPayload) (*domain.DeployResult, error) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return &domain.DeployResult{Success: true}, nil
	})

	_, err := target.Deploy(ctx, domain.DeployPayload{})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestDeployTarget_NilFunction(t *testing.T) {
	target := NewDeployTarget(nil)
	_, err := target.Deploy(context.Background(), domain.DeployPayload{})
	if err == nil {
		t.Fatal("expected error for nil function")
	}
}
