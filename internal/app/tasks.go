package app

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/agentgit/agit/internal/db"
)

func StartTask(ctx *Context, taskID int64) error {
	if err := db.UpdateTaskStatus(ctx.DB, taskID, "active"); err != nil {
		return err
	}
	return db.SetWorkspaceCurrentTask(ctx.DB, ctx.WorkspaceID, taskID)
}

func DoneTask(ctx *Context, taskID int64, noTest bool) (int, error) {
	task, err := db.GetTask(ctx.DB, taskID)
	if err != nil {
		return 0, err
	}

	if !task.TestCommand.Valid || strings.TrimSpace(task.TestCommand.String) == "" {
		if noTest {
			if err := db.UpdateTaskStatus(ctx.DB, taskID, "done"); err != nil {
				return 0, err
			}
			return 0, nil
		}
		return 0, fmt.Errorf("task has no test_command; use --no-test to mark done")
	}

	start := time.Now()
	exitCode, output, runErr := runCommand(task.TestCommand.String)
	duration := time.Since(start)

	if err := db.RecordTrace(ctx.DB, taskID, task.TestCommand.String, exitCode, output, duration.Milliseconds()); err != nil {
		return exitCode, err
	}

	status := "done"
	if exitCode != 0 || runErr != nil {
		status = "failed"
	}

	if err := db.UpdateTaskStatus(ctx.DB, taskID, status); err != nil {
		return exitCode, err
	}
	if err := db.ClearWorkspaceCurrentTask(ctx.DB, ctx.WorkspaceID); err != nil {
		return exitCode, err
	}

	if runErr != nil {
		if _, ok := runErr.(*exec.ExitError); ok {
			return exitCode, nil
		}
		return exitCode, runErr
	}

	return exitCode, nil
}

func runCommand(command string) (int, string, error) {
	cmd := exec.Command("sh", "-c", command)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return exitCode, string(out), err
}
