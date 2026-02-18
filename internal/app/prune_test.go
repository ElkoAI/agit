package app

import (
	"testing"
	"time"

	"github.com/agentgit/agit/internal/db"
)

func TestPruneCheckpoints_KeepLast(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Close()

	for i := 0; i < 3; i++ {
		if err := db.CreateCheckpoint(ctx.DB, ctx.WorkspaceID, "hash"+string(rune('a'+i)), "{}", "msg"); err != nil {
			t.Fatalf("CreateCheckpoint: %v", err)
		}
	}

	res, err := PruneCheckpoints(ctx, 2, 0, true)
	if err != nil {
		t.Fatalf("PruneCheckpoints: %v", err)
	}
	if res.Deleted != 1 {
		t.Fatalf("expected 1 delete, got %d", res.Deleted)
	}
}

func TestPruneCheckpoints_KeepDays(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Close()

	if err := db.CreateCheckpoint(ctx.DB, ctx.WorkspaceID, "hash1", "{}", "msg"); err != nil {
		t.Fatalf("CreateCheckpoint: %v", err)
	}
	if err := db.CreateCheckpoint(ctx.DB, ctx.WorkspaceID, "hash2", "{}", "msg"); err != nil {
		t.Fatalf("CreateCheckpoint: %v", err)
	}

	// set first checkpoint to be old
	_, err := ctx.DB.Exec(`UPDATE checkpoints SET created_at = ? WHERE commit_hash = ?`, time.Now().Add(-40*24*time.Hour), "hash1")
	if err != nil {
		t.Fatalf("update created_at: %v", err)
	}

	res, err := PruneCheckpoints(ctx, 0, 30, true)
	if err != nil {
		t.Fatalf("PruneCheckpoints: %v", err)
	}
	if res.Deleted != 1 {
		t.Fatalf("expected 1 delete, got %d", res.Deleted)
	}
}
