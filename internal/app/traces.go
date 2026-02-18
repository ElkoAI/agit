package app

import "github.com/agentgit/agit/internal/db"

type Trace = db.Trace

func ListTraces(ctx *Context, taskID int64, limit int) ([]Trace, error) {
	return db.ListTraces(ctx.DB, taskID, limit)
}
