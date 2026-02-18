package app

import (
	"encoding/json"
	"fmt"
)

// Checkpoint captures a git commit plus the current workspace snapshot.
func Checkpoint(ctx *Context, message string) (string, error) {
	if message == "" {
		return "", fmt.Errorf("message is required")
	}

	if err := gitAddAll(); err != nil {
		return "", err
	}
	if err := gitCommit(message); err != nil {
		return "", err
	}

	commit, err := GitOutput("rev-parse", "HEAD")
	if err != nil {
		return "", err
	}

	state := "{}"
	if ctx.Workspace.AgentState.Valid && ctx.Workspace.AgentState.String != "" {
		state = ctx.Workspace.AgentState.String
	} else {
		if b, err := json.Marshal(map[string]any{}); err == nil {
			state = string(b)
		}
	}
	if err := dbCreateCheckpoint(ctx, commit, state, message); err != nil {
		return "", err
	}

	if err := dbSetWorkspaceLastCommit(ctx, commit); err != nil {
		return "", err
	}

	return commit, nil
}

// Restore resets the repo to the latest checkpoint commit.
// Options:
// - force: allow restore when dirty
// - dryRun: do not modify repo, just return target commit
// - keep: use `git reset --keep` to preserve local changes
func Restore(ctx *Context, force, dryRun, keep bool) (string, error) {
	if !force {
		dirty, err := gitIsDirty()
		if err != nil {
			return "", err
		}
		if dirty {
			return "", fmt.Errorf("working tree is dirty; use --force to restore")
		}
	}

	cp, err := dbLatestCheckpoint(ctx)
	if err != nil {
		return "", err
	}

	if dryRun {
		return cp.CommitHash, nil
	}

	if keep {
		if err := gitResetKeep(cp.CommitHash); err != nil {
			return "", err
		}
	} else {
		if err := gitResetHard(cp.CommitHash); err != nil {
			return "", err
		}
	}

	if err := dbSetWorkspaceLastCommit(ctx, cp.CommitHash); err != nil {
		return "", err
	}

	return cp.CommitHash, nil
}

func ListCheckpoints(ctx *Context, limit int) ([]CheckpointRecord, error) {
	return dbListCheckpoints(ctx, limit)
}
