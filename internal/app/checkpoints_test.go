package app

import (
	"testing"

	"github.com/agentgit/agit/internal/db"
)

func TestListCheckpointsEmpty(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Close()

	cps, err := ListCheckpoints(ctx, 10)
	if err != nil {
		t.Fatalf("ListCheckpoints: %v", err)
	}
	if len(cps) != 0 {
		t.Fatalf("expected 0 checkpoints, got %d", len(cps))
	}
}

func TestCheckpointCRUD(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Close()

	if err := db.CreateCheckpoint(ctx.DB, ctx.WorkspaceID, "abc123", "{}", "first"); err != nil {
		t.Fatalf("CreateCheckpoint: %v", err)
	}

	cp, err := db.GetLatestCheckpoint(ctx.DB, ctx.WorkspaceID)
	if err != nil {
		t.Fatalf("GetLatestCheckpoint: %v", err)
	}
	if cp.CommitHash != "abc123" {
		t.Fatalf("unexpected commit hash: %s", cp.CommitHash)
	}
	if !cp.Message.Valid || cp.Message.String != "first" {
		t.Fatalf("expected message 'first', got %+v", cp.Message)
	}

	list, err := db.ListCheckpoints(ctx.DB, ctx.WorkspaceID, 10)
	if err != nil {
		t.Fatalf("ListCheckpoints: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 checkpoint, got %d", len(list))
	}
}
