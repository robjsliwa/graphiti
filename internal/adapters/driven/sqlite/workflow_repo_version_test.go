package sqlite_test

import (
	"context"
	"testing"
	"time"

	sqliteadapter "graphiti/internal/adapters/driven/sqlite"
	"graphiti/internal/domain"
)

func TestWorkflowRepo_CreateVersion(t *testing.T) {
	db := newTestDB(t)
	userRepo := sqliteadapter.NewUserRepository(db)
	repo := sqliteadapter.NewWorkflowRepository(db)
	ctx := context.Background()

	user := &domain.User{
		ID: "user-1", Username: "testuser",
		AuthProvider: "fake", AuthProviderID: "fake-1",
		CreatedAt: time.Now().UTC(), LastLoginAt: time.Now().UTC(),
	}
	userRepo.Upsert(ctx, user)

	wf := makeWorkflow("wf-ver-1", "Versioned WF", "user-1")
	if err := repo.Create(ctx, wf); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	// Create version 1
	v1 := &domain.WorkflowVersion{
		ID:         "ver-1",
		WorkflowID: "wf-ver-1",
		Version:    1,
		Definition: []byte(`{"nodes":[],"edges":[]}`),
		DeployedAt: time.Now().UTC(),
		DeployedBy: "user-1",
		CreatedAt:  time.Now().UTC(),
	}
	if err := repo.CreateVersion(ctx, v1); err != nil {
		t.Fatalf("create version: %v", err)
	}

	// Create version 2
	v2 := &domain.WorkflowVersion{
		ID:         "ver-2",
		WorkflowID: "wf-ver-1",
		Version:    2,
		Definition: []byte(`{"nodes":[{"id":"n1"}],"edges":[]}`),
		DeployedAt: time.Now().UTC(),
		DeployedBy: "user-1",
		CreatedAt:  time.Now().UTC(),
	}
	if err := repo.CreateVersion(ctx, v2); err != nil {
		t.Fatalf("create version 2: %v", err)
	}

	// Get version history
	versions, err := repo.GetVersionHistory(ctx, "wf-ver-1")
	if err != nil {
		t.Fatalf("get version history: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	// Versions should be ordered newest first
	if versions[0].Version != 2 {
		t.Errorf("first version should be 2, got %d", versions[0].Version)
	}
	if versions[1].Version != 1 {
		t.Errorf("second version should be 1, got %d", versions[1].Version)
	}
	if versions[0].DeployedBy != "user-1" {
		t.Errorf("deployed_by: got %q, want user-1", versions[0].DeployedBy)
	}
}

func TestWorkflowRepo_CreateVersion_CascadeDelete(t *testing.T) {
	db := newTestDB(t)
	userRepo := sqliteadapter.NewUserRepository(db)
	repo := sqliteadapter.NewWorkflowRepository(db)
	ctx := context.Background()

	user := &domain.User{
		ID: "user-1", Username: "testuser",
		AuthProvider: "fake", AuthProviderID: "fake-1",
		CreatedAt: time.Now().UTC(), LastLoginAt: time.Now().UTC(),
	}
	userRepo.Upsert(ctx, user)

	wf := makeWorkflow("wf-cascade-v", "Cascade Test", "user-1")
	repo.Create(ctx, wf)

	repo.CreateVersion(ctx, &domain.WorkflowVersion{
		ID: "ver-c1", WorkflowID: "wf-cascade-v", Version: 1,
		Definition: []byte(`{}`), DeployedAt: time.Now().UTC(), DeployedBy: "user-1", CreatedAt: time.Now().UTC(),
	})

	// Delete the workflow
	if err := repo.Delete(ctx, "wf-cascade-v"); err != nil {
		t.Fatalf("delete workflow: %v", err)
	}

	// Versions should be cascade deleted
	versions, err := repo.GetVersionHistory(ctx, "wf-cascade-v")
	if err != nil {
		t.Fatalf("get version history: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("expected 0 versions after cascade delete, got %d", len(versions))
	}
}
