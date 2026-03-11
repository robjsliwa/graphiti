package sqlite_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	sqliteadapter "graphiti/internal/adapters/driven/sqlite"
	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

func TestParseTimeStr(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // expected RFC3339 output
	}{
		{"RFC3339 with Z", "2025-03-11T14:30:00Z", "2025-03-11T14:30:00Z"},
		{"RFC3339 with offset", "2025-03-11T14:30:00+05:00", "2025-03-11T14:30:00+05:00"},
		{"space separator with offset", "2025-03-11 14:30:00+00:00", "2025-03-11T14:30:00Z"},
		{"space separator no offset", "2025-03-11 14:30:00", "2025-03-11T14:30:00Z"},
		{"empty string", "", "0001-01-01T00:00:00Z"},
		{"garbage", "not-a-time", "0001-01-01T00:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sqliteadapter.ParseTimeStr(tt.input)
			if got.Format(time.RFC3339) != tt.want {
				t.Errorf("ParseTimeStr(%q) = %s, want %s", tt.input, got.Format(time.RFC3339), tt.want)
			}
		})
	}
}

func TestWorkflowRepo_TimestampRoundTrip(t *testing.T) {
	db := newTestDB(t)
	userRepo := sqliteadapter.NewUserRepository(db)
	repo := sqliteadapter.NewWorkflowRepository(db)
	ctx := context.Background()

	user := &domain.User{
		ID: "user-1", Username: "testuser",
		AuthProvider: "fake", AuthProviderID: "fake-1",
		CreatedAt: time.Now().UTC(), LastLoginAt: time.Now().UTC(),
	}
	if err := userRepo.Upsert(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	wf := makeWorkflow("wf-ts", "Timestamp Test", "user-1")

	if err := repo.Create(ctx, wf); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	got, err := repo.GetByID(ctx, "wf-ts")
	if err != nil {
		t.Fatalf("get workflow: %v", err)
	}

	// Timestamps should survive the round-trip (truncated to seconds).
	wantCreated := wf.CreatedAt.Truncate(time.Second)
	gotCreated := got.CreatedAt.Truncate(time.Second)
	if !gotCreated.Equal(wantCreated) {
		t.Errorf("CreatedAt: got %v, want %v", gotCreated, wantCreated)
	}

	wantUpdated := wf.UpdatedAt.Truncate(time.Second)
	gotUpdated := got.UpdatedAt.Truncate(time.Second)
	if !gotUpdated.Equal(wantUpdated) {
		t.Errorf("UpdatedAt: got %v, want %v", gotUpdated, wantUpdated)
	}

	// Verify they're not zero.
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero time")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero time")
	}
}

func TestWorkflowRepo_ListTimestampRoundTrip(t *testing.T) {
	db := newTestDB(t)
	userRepo := sqliteadapter.NewUserRepository(db)
	repo := sqliteadapter.NewWorkflowRepository(db)
	ctx := context.Background()

	user := &domain.User{
		ID: "user-1", Username: "testuser",
		AuthProvider: "fake", AuthProviderID: "fake-1",
		CreatedAt: time.Now().UTC(), LastLoginAt: time.Now().UTC(),
	}
	if err := userRepo.Upsert(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	wf := makeWorkflow("wf-ts-list", "List Timestamp Test", "user-1")
	if err := repo.Create(ctx, wf); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	summaries, err := repo.List(ctx, driven.WorkflowFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(summaries) == 0 {
		t.Fatal("expected at least one summary")
	}

	wantUpdated := wf.UpdatedAt.Truncate(time.Second)
	gotUpdated := summaries[0].UpdatedAt.Truncate(time.Second)
	if !gotUpdated.Equal(wantUpdated) {
		t.Errorf("List UpdatedAt: got %v, want %v", gotUpdated, wantUpdated)
	}
	if summaries[0].UpdatedAt.IsZero() {
		t.Error("List UpdatedAt is zero time")
	}
}

func makeWorkflow(id, name, createdBy string) *domain.Workflow {
	now := time.Now().Truncate(time.Second).UTC()
	return &domain.Workflow{
		ID:          id,
		Name:        name,
		Description: "test workflow",
		Status:      domain.WorkflowStatusDraft,
		Version:     1,
		Nodes: []domain.NodeInstance{
			{
				ID:           "node-1",
				DefinitionID: "def-http",
				Label:        "HTTP Request",
				X:            100,
				Y:            200,
				AttributeValues: map[string]any{
					"url":    "https://example.com",
					"method": "GET",
				},
			},
		},
		Edges: []domain.Edge{},
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestWorkflowRepo_CreateAndGetByID(t *testing.T) {
	db := newTestDB(t)
	userRepo := sqliteadapter.NewUserRepository(db)
	repo := sqliteadapter.NewWorkflowRepository(db)
	ctx := context.Background()

	// Create a user first (foreign key).
	user := &domain.User{
		ID:             "user-1",
		Username:       "testuser",
		AuthProvider:   "fake",
		AuthProviderID: "fake-1",
		CreatedAt:      time.Now().UTC(),
		LastLoginAt:    time.Now().UTC(),
	}
	if err := userRepo.Upsert(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	wf := makeWorkflow("wf-1", "My Workflow", "user-1")
	if err := repo.Create(ctx, wf); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	got, err := repo.GetByID(ctx, "wf-1")
	if err != nil {
		t.Fatalf("get workflow: %v", err)
	}

	if got.ID != wf.ID {
		t.Errorf("ID: got %s, want %s", got.ID, wf.ID)
	}
	if got.Name != wf.Name {
		t.Errorf("Name: got %s, want %s", got.Name, wf.Name)
	}
	if got.Description != wf.Description {
		t.Errorf("Description: got %s, want %s", got.Description, wf.Description)
	}
	if got.Status != wf.Status {
		t.Errorf("Status: got %s, want %s", got.Status, wf.Status)
	}
	if got.Version != wf.Version {
		t.Errorf("Version: got %d, want %d", got.Version, wf.Version)
	}
	if len(got.Nodes) != 1 {
		t.Fatalf("Nodes: got %d, want 1", len(got.Nodes))
	}
	if got.Nodes[0].ID != "node-1" {
		t.Errorf("Node ID: got %s, want node-1", got.Nodes[0].ID)
	}
	if got.Nodes[0].X != 100 {
		t.Errorf("Node X: got %f, want 100", got.Nodes[0].X)
	}
	if got.Nodes[0].AttributeValues["url"] != "https://example.com" {
		t.Errorf("Node attr url: got %v, want https://example.com", got.Nodes[0].AttributeValues["url"])
	}
}

func TestWorkflowRepo_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := sqliteadapter.NewWorkflowRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent workflow")
	}
}

func TestWorkflowRepo_List(t *testing.T) {
	db := newTestDB(t)
	userRepo := sqliteadapter.NewUserRepository(db)
	repo := sqliteadapter.NewWorkflowRepository(db)
	ctx := context.Background()

	user := &domain.User{
		ID: "user-1", Username: "testuser",
		AuthProvider: "fake", AuthProviderID: "fake-1",
		CreatedAt: time.Now().UTC(), LastLoginAt: time.Now().UTC(),
	}
	if err := userRepo.Upsert(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	user2 := &domain.User{
		ID: "user-2", Username: "other",
		AuthProvider: "fake", AuthProviderID: "fake-2",
		CreatedAt: time.Now().UTC(), LastLoginAt: time.Now().UTC(),
	}
	if err := userRepo.Upsert(ctx, user2); err != nil {
		t.Fatalf("create user2: %v", err)
	}

	wf1 := makeWorkflow("wf-1", "Workflow 1", "user-1")
	wf2 := makeWorkflow("wf-2", "Workflow 2", "user-1")
	wf2.Status = domain.WorkflowStatusDeployed
	wf3 := makeWorkflow("wf-3", "Workflow 3", "user-2")

	for _, wf := range []*domain.Workflow{wf1, wf2, wf3} {
		if err := repo.Create(ctx, wf); err != nil {
			t.Fatalf("create workflow %s: %v", wf.ID, err)
		}
	}

	// List all.
	all, err := repo.List(ctx, driven.WorkflowFilter{})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("list all: got %d, want 3", len(all))
	}

	// Filter by user.
	byUser, err := repo.List(ctx, driven.WorkflowFilter{UserID: "user-1"})
	if err != nil {
		t.Fatalf("list by user: %v", err)
	}
	if len(byUser) != 2 {
		t.Errorf("list by user: got %d, want 2", len(byUser))
	}

	// Filter by status.
	byStatus, err := repo.List(ctx, driven.WorkflowFilter{Status: domain.WorkflowStatusDeployed})
	if err != nil {
		t.Fatalf("list by status: %v", err)
	}
	if len(byStatus) != 1 {
		t.Errorf("list by status: got %d, want 1", len(byStatus))
	}
	if byStatus[0].ID != "wf-2" {
		t.Errorf("list by status ID: got %s, want wf-2", byStatus[0].ID)
	}
}

func TestWorkflowRepo_Update(t *testing.T) {
	db := newTestDB(t)
	userRepo := sqliteadapter.NewUserRepository(db)
	repo := sqliteadapter.NewWorkflowRepository(db)
	ctx := context.Background()

	user := &domain.User{
		ID: "user-1", Username: "testuser",
		AuthProvider: "fake", AuthProviderID: "fake-1",
		CreatedAt: time.Now().UTC(), LastLoginAt: time.Now().UTC(),
	}
	if err := userRepo.Upsert(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	wf := makeWorkflow("wf-1", "Original", "user-1")
	if err := repo.Create(ctx, wf); err != nil {
		t.Fatalf("create: %v", err)
	}

	wf.Name = "Updated"
	wf.Version = 2
	wf.Nodes = append(wf.Nodes, domain.NodeInstance{
		ID:           "node-2",
		DefinitionID: "def-transform",
		Label:        "Transform",
		X:            300,
		Y:            200,
	})
	wf.Edges = append(wf.Edges, domain.Edge{
		ID:           "edge-1",
		SourceNodeID: "node-1",
		SourcePortID: "out",
		TargetNodeID: "node-2",
		TargetPortID: "in",
	})

	if err := repo.Update(ctx, wf); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := repo.GetByID(ctx, "wf-1")
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if got.Name != "Updated" {
		t.Errorf("Name: got %s, want Updated", got.Name)
	}
	if got.Version != 2 {
		t.Errorf("Version: got %d, want 2", got.Version)
	}
	if len(got.Nodes) != 2 {
		t.Errorf("Nodes: got %d, want 2", len(got.Nodes))
	}
	if len(got.Edges) != 1 {
		t.Errorf("Edges: got %d, want 1", len(got.Edges))
	}
}

func TestWorkflowRepo_Update_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := sqliteadapter.NewWorkflowRepository(db)
	ctx := context.Background()

	wf := makeWorkflow("nonexistent", "Ghost", "")
	err := repo.Update(ctx, wf)
	if err == nil {
		t.Fatal("expected error for updating nonexistent workflow")
	}
}

func TestWorkflowRepo_Delete(t *testing.T) {
	db := newTestDB(t)
	userRepo := sqliteadapter.NewUserRepository(db)
	repo := sqliteadapter.NewWorkflowRepository(db)
	ctx := context.Background()

	user := &domain.User{
		ID: "user-1", Username: "testuser",
		AuthProvider: "fake", AuthProviderID: "fake-1",
		CreatedAt: time.Now().UTC(), LastLoginAt: time.Now().UTC(),
	}
	if err := userRepo.Upsert(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	wf := makeWorkflow("wf-1", "To Delete", "user-1")
	if err := repo.Create(ctx, wf); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Insert a version to test cascade.
	_, err := db.ExecContext(ctx,
		`INSERT INTO workflow_versions (id, workflow_id, version, definition, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		"ver-1", "wf-1", 1, `{"nodes":[],"edges":[]}`, time.Now().UTC())
	if err != nil {
		t.Fatalf("insert version: %v", err)
	}

	if err := repo.Delete(ctx, "wf-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = repo.GetByID(ctx, "wf-1")
	if err == nil {
		t.Fatal("expected error after delete")
	}

	// Verify version was cascade-deleted.
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM workflow_versions WHERE workflow_id = ?", "wf-1").Scan(&count); err != nil {
		t.Fatalf("count versions: %v", err)
	}
	if count != 0 {
		t.Errorf("versions after cascade delete: got %d, want 0", count)
	}
}

func TestWorkflowRepo_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := sqliteadapter.NewWorkflowRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for deleting nonexistent workflow")
	}
}

func TestWorkflowRepo_GetVersionHistory(t *testing.T) {
	db := newTestDB(t)
	userRepo := sqliteadapter.NewUserRepository(db)
	repo := sqliteadapter.NewWorkflowRepository(db)
	ctx := context.Background()

	user := &domain.User{
		ID: "user-1", Username: "testuser",
		AuthProvider: "fake", AuthProviderID: "fake-1",
		CreatedAt: time.Now().UTC(), LastLoginAt: time.Now().UTC(),
	}
	if err := userRepo.Upsert(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	wf := makeWorkflow("wf-1", "Versioned", "user-1")
	if err := repo.Create(ctx, wf); err != nil {
		t.Fatalf("create: %v", err)
	}

	now := time.Now().UTC()
	for i := 1; i <= 3; i++ {
		_, err := db.ExecContext(ctx,
			`INSERT INTO workflow_versions (id, workflow_id, version, definition, deployed_at, deployed_by, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			fmt.Sprintf("ver-%d", i), "wf-1", i, `{"nodes":[],"edges":[]}`,
			now, "user-1", now)
		if err != nil {
			t.Fatalf("insert version %d: %v", i, err)
		}
	}

	versions, err := repo.GetVersionHistory(ctx, "wf-1")
	if err != nil {
		t.Fatalf("get version history: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("versions: got %d, want 3", len(versions))
	}
	// Should be ordered DESC by version.
	if versions[0].Version != 3 {
		t.Errorf("first version: got %d, want 3", versions[0].Version)
	}
	if versions[2].Version != 1 {
		t.Errorf("last version: got %d, want 1", versions[2].Version)
	}
}
