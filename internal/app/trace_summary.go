package app

import "github.com/agentgit/agit/internal/db"

type TraceSummary struct {
	ID         int64  `json:"id"`
	TaskID     int64  `json:"task_id"`
	ExitCode   *int64 `json:"exit_code,omitempty"`
	DurationMs *int64 `json:"duration_ms,omitempty"`
	Command    string `json:"command,omitempty"`
	RunAt      string `json:"run_at"`
}

func GetLastTrace(ctx *Context, taskID int64) (*db.Trace, error) {
	traces, err := db.ListTraces(ctx.DB, taskID, 1)
	if err != nil || len(traces) == 0 {
		return nil, err
	}
	return &traces[0], nil
}

func ToTraceSummary(t db.Trace) TraceSummary {
	var exitCode *int64
	if t.ExitCode.Valid {
		v := t.ExitCode.Int64
		exitCode = &v
	}
	var duration *int64
	if t.DurationMs.Valid {
		v := t.DurationMs.Int64
		duration = &v
	}
	cmd := ""
	if t.Command.Valid {
		cmd = t.Command.String
	}
	return TraceSummary{
		ID:         t.ID,
		TaskID:     t.TaskID,
		ExitCode:   exitCode,
		DurationMs: duration,
		Command:    cmd,
		RunAt:      t.RunAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
