package app

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/agentgit/agit/internal/db"
)

type CheckpointRecord = db.Checkpoint

func dbCreateCheckpoint(ctx *Context, commitHash, agentState, message string) error {
	return db.CreateCheckpoint(ctx.DB, ctx.WorkspaceID, commitHash, agentState, message)
}

func dbLatestCheckpoint(ctx *Context) (db.Checkpoint, error) {
	return db.GetLatestCheckpoint(ctx.DB, ctx.WorkspaceID)
}

func dbListCheckpoints(ctx *Context, limit int) ([]db.Checkpoint, error) {
	return db.ListCheckpoints(ctx.DB, ctx.WorkspaceID, limit)
}

func dbSetWorkspaceLastCommit(ctx *Context, commit string) error {
	return db.UpdateWorkspaceLastCommit(ctx.DB, ctx.WorkspaceID, commit)
}

func gitAddAll() error {
	cmd := exec.Command("git", "add", "-A")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add -A: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func gitCommit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func gitResetHard(commit string) error {
	cmd := exec.Command("git", "reset", "--hard", commit)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git reset --hard %s: %v: %s", commit, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func gitResetKeep(commit string) error {
	cmd := exec.Command("git", "reset", "--keep", commit)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git reset --keep %s: %v: %s", commit, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func gitIsDirty() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status --porcelain: %w", err)
	}
	return strings.TrimSpace(string(out)) != "", nil
}
