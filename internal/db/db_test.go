package db

import (
	"path/filepath"
	"testing"
	"time"
)

func TestInitAndWorkspaceAndTask(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "agit.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Init: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	ws, err := EnsureWorkspace(db, "ws1", "main", tmp, "abc123")
	if err != nil {
		t.Fatalf("EnsureWorkspace: %v", err)
	}
	if ws.WorkspaceID != "ws1" {
		t.Fatalf("workspace id mismatch: %s", ws.WorkspaceID)
	}

	id, err := CreateTask(db, "ws1", "do thing", "go test ./...")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if err := UpdateTaskStatus(db, id, "active"); err != nil {
		t.Fatalf("UpdateTaskStatus: %v", err)
	}

	if err := RecordTrace(db, id, "go test ./...", 0, "ok", 10); err != nil {
		t.Fatalf("RecordTrace: %v", err)
	}

	traces, err := ListTraces(db, id, 10)
	if err != nil {
		t.Fatalf("ListTraces: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	if err := UpdateTaskStatus(db, id, "done"); err != nil {
		t.Fatalf("UpdateTaskStatus done: %v", err)
	}

	if err := SetWorkspaceCurrentTask(db, "ws1", id); err != nil {
		t.Fatalf("SetWorkspaceCurrentTask: %v", err)
	}
	if err := ClearWorkspaceCurrentTask(db, "ws1"); err != nil {
		t.Fatalf("ClearWorkspaceCurrentTask: %v", err)
	}

	// sanity check timestamps are set
	task, err := GetTask(db, id)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.CreatedAt.IsZero() {
		t.Fatal("task created_at not set")
	}
	if task.CompletedAt.Valid {
		if task.CompletedAt.Time.After(time.Now().Add(1 * time.Minute)) {
			t.Fatal("task completed_at looks incorrect")
		}
	}
}
