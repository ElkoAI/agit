package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/agentgit/agit/internal/app"
	"github.com/agentgit/agit/internal/db"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "init":
		if err := runInit(); err != nil {
			fmt.Fprintln(os.Stderr, "agit init failed:", err)
			os.Exit(1)
		}
	case "status":
		if err := runStatus(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "agit status failed:", err)
			os.Exit(1)
		}
	case "task":
		if err := runTask(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "agit task failed:", err)
			os.Exit(1)
		}
	case "checkpoint":
		if err := runCheckpoint(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "agit checkpoint failed:", err)
			os.Exit(1)
		}
	case "restore":
		if err := runRestore(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "agit restore failed:", err)
			os.Exit(1)
		}
	case "trace":
		if err := runTrace(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "agit trace failed:", err)
			os.Exit(1)
		}
	case "workspace":
		if err := runWorkspace(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "agit workspace failed:", err)
			os.Exit(1)
		}
	case "hook":
		if err := runHook(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "agit hook failed:", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("agit - Agent Git CLI")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  agit init             Initialize .git/agit.db in current repo")
	fmt.Println("  agit status           Show workspace status")
	fmt.Println("  agit task new         Create a new task")
	fmt.Println("  agit task list        List tasks")
	fmt.Println("  agit task start       Mark task as active")
	fmt.Println("  agit task done        Run test and mark done")
	fmt.Println("  agit checkpoint       Create a checkpoint (git commit + state)")
	fmt.Println("  agit checkpoint list  List checkpoints")
	fmt.Println("  agit restore          Restore to latest checkpoint")
	fmt.Println("  agit trace            Show recent test traces")
	fmt.Println("  agit workspace        Manage workspace state")
	fmt.Println("  agit hook install     Install git hooks")
	fmt.Println("  agit hook uninstall   Remove AGIT git hooks")
	fmt.Println("  agit hook status      Show AGIT hook status")
}

func runInit() error {
	gitDir, err := app.FindGitDir()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(gitDir, "agit.db")
	if _, err := os.Stat(dbPath); err == nil {
		return fmt.Errorf("db already exists at %s", dbPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := db.Init(dbPath); err != nil {
		return err
	}

	fmt.Println("initialized", dbPath)
	return nil
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("agit status", flag.ContinueOnError)
	lastOutput := fs.Bool("last-output", false, "print last trace output")
	maxOutput := fs.Int("max-output", 0, "max output bytes to print (0 = no limit)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, err := app.LoadContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	counts, err := db.CountTasksByStatus(ctx.DB, ctx.WorkspaceID)
	if err != nil {
		return err
	}

	fmt.Println("Workspace:", ctx.Workspace.Branch)
	fmt.Println("Workspace ID:", ctx.WorkspaceID)
	if ctx.Workspace.LastCommit.Valid {
		fmt.Println("Last commit:", ctx.Workspace.LastCommit.String)
	}
	if ctx.Workspace.CurrentTaskID.Valid {
		fmt.Println("Current task ID:", ctx.Workspace.CurrentTaskID.Int64)
	}

	if ctx.Workspace.CurrentTaskID.Valid {
		if trace, err := app.GetLastTrace(ctx, ctx.Workspace.CurrentTaskID.Int64); err == nil && trace != nil {
			fmt.Println("Last trace:")
			printTraceSummary(*trace)
			if *lastOutput && trace.Output.Valid && strings.TrimSpace(trace.Output.String) != "" {
				fmt.Println(trimOutput(trace.Output.String, *maxOutput))
			}
		}
	}

	if len(counts) == 0 {
		fmt.Println("Tasks: none")
		return nil
	}

	fmt.Println("Tasks:")
	for _, key := range []string{"active", "pending", "done", "failed", "blocked", "skipped"} {
		if v, ok := counts[key]; ok {
			fmt.Printf("  %s: %d\n", key, v)
		}
	}
	return nil
}

func runTask(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("task subcommand required (new, list, start, done)")
	}

	switch args[0] {
	case "new":
		return runTaskNew(args[1:])
	case "list":
		return runTaskList(args[1:])
	case "start":
		return runTaskStart(args[1:])
	case "done":
		return runTaskDone(args[1:])
	default:
		return fmt.Errorf("unknown task subcommand: %s", args[0])
	}
}

func runHook(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("hook subcommand required (install, uninstall, status)")
	}
	switch args[0] {
	case "install":
		return runHookInstall()
	case "uninstall":
		return runHookUninstall()
	case "status":
		return runHookStatus()
	default:
		return fmt.Errorf("unknown hook subcommand: %s", args[0])
	}
}

func runHookInstall() error {
	gitDir, err := app.FindGitDir()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}

	hookBody := `#!/bin/sh
# AGIT hook: refresh workspace state
agit status >/dev/null 2>&1 || true
`

	for _, name := range []string{"post-checkout", "post-merge", "post-rewrite"} {
		path := filepath.Join(hooksDir, name)
		if err := writeHook(path, hookBody); err != nil {
			return err
		}
	}

	fmt.Println("installed hooks in", hooksDir)
	return nil
}

func runHookUninstall() error {
	gitDir, err := app.FindGitDir()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	for _, name := range []string{"post-checkout", "post-merge", "post-rewrite"} {
		path := filepath.Join(hooksDir, name)
		if data, err := os.ReadFile(path); err == nil {
			if strings.Contains(string(data), "AGIT hook") {
				if err := os.Remove(path); err != nil {
					return err
				}
			}
		}
	}

	fmt.Println("removed AGIT hooks from", hooksDir)
	return nil
}

func runHookStatus() error {
	gitDir, err := app.FindGitDir()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	for _, name := range []string{"post-checkout", "post-merge", "post-rewrite"} {
		path := filepath.Join(hooksDir, name)
		status := "missing"
		if data, err := os.ReadFile(path); err == nil {
			if strings.Contains(string(data), "AGIT hook") {
				status = "installed"
			} else {
				status = "custom"
			}
		}
		fmt.Printf("%s: %s\n", name, status)
	}
	return nil
}

func writeHook(path, body string) error {
	if data, err := os.ReadFile(path); err == nil {
		if !strings.Contains(string(data), "AGIT hook") {
			return fmt.Errorf("hook exists and is not managed by AGIT: %s", path)
		}
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		return err
	}
	return nil
}

func runTaskNew(args []string) error {
	fs := flag.NewFlagSet("agit task new", flag.ContinueOnError)
	testCmd := fs.String("test", "", "test command")
	if err := fs.Parse(args); err != nil {
		return err
	}

	desc := strings.Join(fs.Args(), " ")
	if strings.TrimSpace(desc) == "" {
		return fmt.Errorf("description is required")
	}

	ctx, err := app.LoadContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	id, err := db.CreateTask(ctx.DB, ctx.WorkspaceID, desc, *testCmd)
	if err != nil {
		return err
	}

	fmt.Println("created task", id)
	return nil
}

func runTaskList(args []string) error {
	fs := flag.NewFlagSet("agit task list", flag.ContinueOnError)
	limit := fs.Int("limit", 20, "max tasks")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, err := app.LoadContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	tasks, err := db.ListTasks(ctx.DB, ctx.WorkspaceID, *limit)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		fmt.Println("no tasks")
		return nil
	}

	for _, t := range tasks {
		testNote := ""
		if t.TestCommand.Valid {
			testNote = " [test]"
		}
		fmt.Printf("%d  %s  %s%s\n", t.ID, t.Status, t.Description, testNote)
	}
	return nil
}

func runTaskStart(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("task id required")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid task id: %w", err)
	}

	ctx, err := app.LoadContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	if err := app.StartTask(ctx, id); err != nil {
		return err
	}

	fmt.Println("started task", id)
	return nil
}

func runTaskDone(args []string) error {
	fs := flag.NewFlagSet("agit task done", flag.ContinueOnError)
	noTest := fs.Bool("no-test", false, "mark done without running tests")
	printOnly := fs.Bool("print", false, "print only the last trace output")
	maxOutput := fs.Int("max-output", 0, "max trace output bytes to print (0 = no limit)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) < 1 {
		return fmt.Errorf("task id required")
	}
	id, err := strconv.ParseInt(fs.Args()[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid task id: %w", err)
	}

	ctx, err := app.LoadContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	exitCode, err := app.DoneTask(ctx, id, *noTest)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		if !*printOnly {
			fmt.Println("task failed with exit code", exitCode)
		}
		printLastTrace(ctx, id, *maxOutput, *printOnly)
		return nil
	}

	if !*printOnly {
		fmt.Println("completed task", id)
	}
	printLastTrace(ctx, id, *maxOutput, *printOnly)
	return nil
}

func runCheckpoint(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("checkpoint subcommand required (list or message)")
	}
	if args[0] == "list" {
		return runCheckpointList(args[1:])
	}

	msg := strings.Join(args, " ")
	if strings.TrimSpace(msg) == "" {
		return fmt.Errorf("checkpoint message is required")
	}

	ctx, err := app.LoadContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	commit, err := app.Checkpoint(ctx, msg)
	if err != nil {
		return err
	}

	fmt.Println("checkpointed", commit)
	return nil
}

func runCheckpointList(args []string) error {
	fs := flag.NewFlagSet("agit checkpoint list", flag.ContinueOnError)
	limit := fs.Int("limit", 10, "max checkpoints")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, err := app.LoadContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	cps, err := app.ListCheckpoints(ctx, *limit)
	if err != nil {
		return err
	}
	if len(cps) == 0 {
		fmt.Println("no checkpoints")
		return nil
	}

	for _, cp := range cps {
		fmt.Printf("%d  %s  %s\n", cp.ID, cp.CommitHash, cp.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}

func runRestore(args []string) error {
	fs := flag.NewFlagSet("agit restore", flag.ContinueOnError)
	force := fs.Bool("force", false, "restore even if working tree is dirty")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, err := app.LoadContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	commit, err := app.Restore(ctx, *force)
	if err != nil {
		return err
	}

	fmt.Println("restored to", commit)
	return nil
}

func runTrace(args []string) error {
	fs := flag.NewFlagSet("agit trace", flag.ContinueOnError)
	limit := fs.Int("limit", 5, "max traces")
	last := fs.Bool("last", false, "show only the most recent trace")
	maxOutput := fs.Int("max-output", 0, "max trace output bytes to print (0 = no limit)")
	jsonOut := fs.Bool("json", false, "print JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) < 1 {
		return fmt.Errorf("task id required")
	}
	id, err := strconv.ParseInt(fs.Args()[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid task id: %w", err)
	}

	ctx, err := app.LoadContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	if *last {
		*limit = 1
	}
	traces, err := app.ListTraces(ctx, id, *limit)
	if err != nil {
		return err
	}
	if len(traces) == 0 {
		if *jsonOut {
			fmt.Println("[]")
			return nil
		}
		fmt.Println("no traces")
		return nil
	}

	if *jsonOut {
		return printTracesJSON(traces, *maxOutput)
	}

	for _, t := range traces {
		exit := "-"
		if t.ExitCode.Valid {
			exit = fmt.Sprintf("%d", t.ExitCode.Int64)
		}
		dur := "-"
		if t.DurationMs.Valid {
			dur = fmt.Sprintf("%dms", t.DurationMs.Int64)
		}
		cmd := ""
		if t.Command.Valid {
			cmd = t.Command.String
		}
		fmt.Printf("#%d  exit=%s  duration=%s  %s\n", t.ID, exit, dur, cmd)
		if t.Output.Valid && strings.TrimSpace(t.Output.String) != "" {
			fmt.Println(trimOutput(t.Output.String, *maxOutput))
		}
	}
	return nil
}

func runWorkspace(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("workspace subcommand required (set-state, clear-state)")
	}
	switch args[0] {
	case "set-state":
		return runWorkspaceSetState(args[1:])
	case "clear-state":
		return runWorkspaceClearState()
	default:
		return fmt.Errorf("unknown workspace subcommand: %s", args[0])
	}
}

func runWorkspaceSetState(args []string) error {
	fs := flag.NewFlagSet("agit workspace set-state", flag.ContinueOnError)
	jsonStr := fs.String("json", "", "agent_state JSON string")
	file := fs.String("file", "", "path to JSON file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	state := strings.TrimSpace(*jsonStr)
	if *file != "" {
		data, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		state = strings.TrimSpace(string(data))
	}
	if state == "" {
		return fmt.Errorf("agent_state JSON is required via --json or --file")
	}

	ctx, err := app.LoadContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	if err := app.SetAgentState(ctx, state); err != nil {
		return err
	}

	fmt.Println("updated agent_state")
	return nil
}

func runWorkspaceClearState() error {
	ctx, err := app.LoadContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	if err := app.ClearAgentState(ctx); err != nil {
		return err
	}

	fmt.Println("cleared agent_state")
	return nil
}

func printLastTrace(ctx *app.Context, taskID int64, maxOutput int, printOnly bool) {
	traces, err := app.ListTraces(ctx, taskID, 1)
	if err != nil || len(traces) == 0 {
		return
	}
	t := traces[0]
	if t.Output.Valid && strings.TrimSpace(t.Output.String) != "" {
		out := trimOutput(t.Output.String, maxOutput)
		fmt.Println(out)
	}
}

func trimOutput(out string, max int) string {
	if max <= 0 || len(out) <= max {
		return out
	}
	return out[:max] + "\n... (truncated)"
}

func printTraceSummary(t app.Trace) {
	exit := "-"
	if t.ExitCode.Valid {
		exit = fmt.Sprintf("%d", t.ExitCode.Int64)
	}
	dur := "-"
	if t.DurationMs.Valid {
		dur = fmt.Sprintf("%dms", t.DurationMs.Int64)
	}
	cmd := ""
	if t.Command.Valid {
		cmd = t.Command.String
	}
	fmt.Printf("  exit=%s  duration=%s  %s\n", exit, dur, cmd)
}

func printTracesJSON(traces []app.Trace, maxOutput int) error {
	type traceOut struct {
		ID         int64  `json:"id"`
		TaskID     int64  `json:"task_id"`
		Command    string `json:"command,omitempty"`
		ExitCode   *int64 `json:"exit_code,omitempty"`
		Output     string `json:"output,omitempty"`
		DurationMs *int64 `json:"duration_ms,omitempty"`
		RunAt      string `json:"run_at"`
	}

	out := make([]traceOut, 0, len(traces))
	for _, t := range traces {
		item := traceOut{
			ID:     t.ID,
			TaskID: t.TaskID,
			RunAt:  t.RunAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if t.Command.Valid {
			item.Command = t.Command.String
		}
		if t.ExitCode.Valid {
			v := t.ExitCode.Int64
			item.ExitCode = &v
		}
		if t.DurationMs.Valid {
			v := t.DurationMs.Int64
			item.DurationMs = &v
		}
		if t.Output.Valid && strings.TrimSpace(t.Output.String) != "" {
			item.Output = trimOutput(t.Output.String, maxOutput)
		}
		out = append(out, item)
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
