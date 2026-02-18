package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agentgit/agit/internal/app"
	"github.com/agentgit/agit/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer(
		"AGIT",
		"0.1.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	s.AddTool(workspaceStatusTool(), workspaceStatusHandler)
	s.AddTool(taskNewTool(), taskNewHandler)
	s.AddTool(taskListTool(), taskListHandler)
	s.AddTool(taskStartTool(), taskStartHandler)
	s.AddTool(taskDoneTool(), taskDoneHandler)
	s.AddTool(traceTool(), traceHandler)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func workspaceStatusTool() mcp.Tool {
	return mcp.NewTool("workspace_status",
		mcp.WithDescription("Get current workspace state"),
		mcp.WithBoolean("include_output",
			mcp.Description("Include last trace output"),
		),
		mcp.WithNumber("max_output",
			mcp.Description("Max output bytes to return (0 = no limit)"),
		),
	)
}

func taskNewTool() mcp.Tool {
	return mcp.NewTool("task_new",
		mcp.WithDescription("Create a new task"),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Task description"),
		),
		mcp.WithString("test_command",
			mcp.Description("Optional test command"),
		),
	)
}

func taskListTool() mcp.Tool {
	return mcp.NewTool("task_list",
		mcp.WithDescription("List tasks for the current workspace"),
		mcp.WithNumber("limit",
			mcp.Description("Max tasks to return"),
		),
	)
}

func taskStartTool() mcp.Tool {
	return mcp.NewTool("task_start",
		mcp.WithDescription("Mark task as active"),
		mcp.WithNumber("task_id",
			mcp.Required(),
			mcp.Description("Task ID"),
		),
	)
}

func taskDoneTool() mcp.Tool {
	return mcp.NewTool("task_done",
		mcp.WithDescription("Run test and mark task done"),
		mcp.WithNumber("task_id",
			mcp.Required(),
			mcp.Description("Task ID"),
		),
		mcp.WithBoolean("no_test",
			mcp.Description("Mark done without running tests"),
		),
	)
}

func traceTool() mcp.Tool {
	return mcp.NewTool("trace",
		mcp.WithDescription("Show recent test traces for a task"),
		mcp.WithNumber("task_id",
			mcp.Required(),
			mcp.Description("Task ID"),
		),
		mcp.WithBoolean("last",
			mcp.Description("Return only the most recent trace"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max traces to return"),
		),
		mcp.WithNumber("max_output",
			mcp.Description("Max output bytes to return (0 = no limit)"),
		),
	)
}

func workspaceStatusHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appCtx, err := app.LoadContext()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer appCtx.Close()

	counts, err := db.CountTasksByStatus(appCtx.DB, appCtx.WorkspaceID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := request.GetArguments()
	includeOutput := false
	maxOutput := 0
	if v, ok := args["include_output"].(bool); ok {
		includeOutput = v
	}
	if v, ok := args["max_output"].(float64); ok {
		if int(v) > 0 {
			maxOutput = int(v)
		}
	}

	resp := map[string]any{
		"workspace_id": appCtx.WorkspaceID,
		"branch":       appCtx.Workspace.Branch,
		"tasks":        counts,
	}
	if appCtx.Workspace.LastCommit.Valid {
		resp["last_commit"] = appCtx.Workspace.LastCommit.String
	}
	if appCtx.Workspace.CurrentTaskID.Valid {
		resp["current_task_id"] = appCtx.Workspace.CurrentTaskID.Int64
		if trace, err := app.GetLastTrace(appCtx, appCtx.Workspace.CurrentTaskID.Int64); err == nil && trace != nil {
			summary := app.ToTraceSummary(*trace)
			item := map[string]any{
				"id":          summary.ID,
				"task_id":     summary.TaskID,
				"run_at":      summary.RunAt,
				"command":     summary.Command,
				"exit_code":   summary.ExitCode,
				"duration_ms": summary.DurationMs,
			}
			if includeOutput && trace.Output.Valid {
				item["output"] = trimOutput(trace.Output.String, maxOutput)
			}
			resp["last_trace"] = item
		}
	}

	return jsonResult(resp)
}

func taskNewHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	desc, err := request.RequireString("description")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := request.GetArguments()
	testCommand, _ := args["test_command"].(string)

	appCtx, err := app.LoadContext()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer appCtx.Close()

	id, err := db.CreateTask(appCtx.DB, appCtx.WorkspaceID, desc, testCommand)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(map[string]any{"task_id": id})
}

func taskListHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appCtx, err := app.LoadContext()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer appCtx.Close()

	args := request.GetArguments()
	limit := 50
	if v, ok := args["limit"].(float64); ok {
		if int(v) > 0 {
			limit = int(v)
		}
	}

	tasks, err := db.ListTasks(appCtx.DB, appCtx.WorkspaceID, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	resp := make([]map[string]any, 0, len(tasks))
	for _, t := range tasks {
		item := map[string]any{
			"id":          t.ID,
			"description": t.Description,
			"status":      t.Status,
		}
		if t.TestCommand.Valid {
			item["test_command"] = t.TestCommand.String
		}
		resp = append(resp, item)
	}

	return jsonResult(resp)
}

func taskStartHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("task_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	id := int64(idFloat)

	appCtx, err := app.LoadContext()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer appCtx.Close()

	if err := app.StartTask(appCtx, id); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText("task started"), nil
}

func taskDoneHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("task_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	id := int64(idFloat)

	args := request.GetArguments()
	noTest, _ := args["no_test"].(bool)

	appCtx, err := app.LoadContext()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer appCtx.Close()

	exitCode, runErr := app.DoneTask(appCtx, id, noTest)
	if runErr != nil && exitCode == 0 {
		return mcp.NewToolResultError(runErr.Error()), nil
	}

	return jsonResult(map[string]any{
		"exit_code": exitCode,
	})
}

func traceHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("task_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	id := int64(idFloat)

	args := request.GetArguments()
	limit := 5
	last := false
	maxOutput := 0
	if v, ok := args["last"].(bool); ok {
		last = v
	}
	if v, ok := args["limit"].(float64); ok {
		if int(v) > 0 {
			limit = int(v)
		}
	}
	if v, ok := args["max_output"].(float64); ok {
		if int(v) > 0 {
			maxOutput = int(v)
		}
	}
	if last {
		limit = 1
	}

	appCtx, err := app.LoadContext()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer appCtx.Close()

	traces, err := app.ListTraces(appCtx, id, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	resp := make([]map[string]any, 0, len(traces))
	for _, t := range traces {
		item := map[string]any{
			"id":      t.ID,
			"task_id": t.TaskID,
			"run_at":  t.RunAt,
		}
		if t.Command.Valid {
			item["command"] = t.Command.String
		}
		if t.ExitCode.Valid {
			item["exit_code"] = t.ExitCode.Int64
		}
		if t.DurationMs.Valid {
			item["duration_ms"] = t.DurationMs.Int64
		}
		if t.Output.Valid {
			item["output"] = trimOutput(t.Output.String, maxOutput)
		}
		resp = append(resp, item)
	}

	return jsonResult(resp)
}

func trimOutput(out string, max int) string {
	if max <= 0 || len(out) <= max {
		return out
	}
	return out[:max] + "\n... (truncated)"
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}
