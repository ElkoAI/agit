package app

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentgit/agit/internal/db"
)

type Context struct {
	DB          *sql.DB
	WorkspaceID string
	Workspace   db.Workspace
}

func (c *Context) Close() {
	if c == nil || c.DB == nil {
		return
	}
	_ = c.DB.Close()
}

func LoadContext() (*Context, error) {
	gitDir, err := FindGitDir()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(gitDir, "agit.db")
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("db not found, run: agit init")
		}
		return nil, err
	}

	repoRoot, err := GitOutput("rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	branch, err := GitOutput("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	lastCommit, _ := GitOutput("rev-parse", "HEAD")

	dbConn, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}

	workspaceID := workspaceHash(repoRoot, branch)
	ws, err := db.EnsureWorkspace(dbConn, workspaceID, branch, repoRoot, lastCommit)
	if err != nil {
		dbConn.Close()
		return nil, err
	}

	return &Context{
		DB:          dbConn,
		WorkspaceID: workspaceID,
		Workspace:   ws,
	}, nil
}

func workspaceHash(repoRoot, branch string) string {
	h := sha1.Sum([]byte(repoRoot + ":" + branch))
	return hex.EncodeToString(h[:])
}
