package app

import (
	"path/filepath"
	"testing"

	"github.com/agentgit/agit/internal/db"
)

func newTestContext(t *testing.T) *Context {
	t.Helper()
	root := t.TempDir()
	dbPath := filepath.Join(root, "agit.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Init: %v", err)
	}
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	ws, err := db.EnsureWorkspace(conn, "ws1", "main", root, "abc123")
	if err != nil {
		conn.Close()
		t.Fatalf("EnsureWorkspace: %v", err)
	}
	return &Context{DB: conn, WorkspaceID: ws.WorkspaceID, Workspace: ws}
}

func TestDoneTask_NoTestRequiresFlag(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Close()

	id, err := db.CreateTask(ctx.DB, ctx.WorkspaceID, "no test", "")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if _, err := DoneTask(ctx, id, false); err == nil {
		t.Fatal("expected error without --no-test")
	}
}

func TestDoneTask_NoTestWithFlag(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Close()

	id, err := db.CreateTask(ctx.DB, ctx.WorkspaceID, "no test", "")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	exit, err := DoneTask(ctx, id, true)
	if err != nil {
		t.Fatalf("DoneTask: %v", err)
	}
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}

	task, err := db.GetTask(ctx.DB, id)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.Status != "done" {
		t.Fatalf("expected status done, got %s", task.Status)
	}
}

func TestDoneTask_WithTestSuccess(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Close()

	id, err := db.CreateTask(ctx.DB, ctx.WorkspaceID, "test ok", "printf 'ok'")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	exit, err := DoneTask(ctx, id, false)
	if err != nil {
		t.Fatalf("DoneTask: %v", err)
	}
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}

	traces, err := db.ListTraces(ctx.DB, id, 10)
	if err != nil {
		t.Fatalf("ListTraces: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
}

func TestDoneTask_WithTestFailure(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Close()

	id, err := db.CreateTask(ctx.DB, ctx.WorkspaceID, "test fail", "false")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	exit, err := DoneTask(ctx, id, false)
	if err != nil {
		t.Fatalf("DoneTask: %v", err)
	}
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}

	task, err := db.GetTask(ctx.DB, id)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.Status != "failed" {
		t.Fatalf("expected status failed, got %s", task.Status)
	}
}

func TestGetLastTrace(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Close()

	id, err := db.CreateTask(ctx.DB, ctx.WorkspaceID, "trace", "printf 'ok'")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if _, err := DoneTask(ctx, id, false); err != nil {
		t.Fatalf("DoneTask: %v", err)
	}

	trace, err := GetLastTrace(ctx, id)
	if err != nil {
		t.Fatalf("GetLastTrace: %v", err)
	}
	if trace == nil {
		t.Fatal("expected trace")
	}
}
