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

	state, _ := json.Marshal(map[string]any{})
	if err := dbCreateCheckpoint(ctx, commit, string(state)); err != nil {
		return "", err
	}

	if err := dbSetWorkspaceLastCommit(ctx, commit); err != nil {
		return "", err
	}

	return commit, nil
}

// Restore resets the repo to the latest checkpoint commit.
func Restore(ctx *Context, force bool) (string, error) {
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

	if err := gitResetHard(cp.CommitHash); err != nil {
		return "", err
	}

	if err := dbSetWorkspaceLastCommit(ctx, cp.CommitHash); err != nil {
		return "", err
	}

	return cp.CommitHash, nil
}

func ListCheckpoints(ctx *Context, limit int) ([]CheckpointRecord, error) {
	return dbListCheckpoints(ctx, limit)
}
