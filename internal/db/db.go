package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

func Init(dbPath string) error {
	if dbPath == "" {
		return fmt.Errorf("db path is empty")
	}

	if err := os.MkdirAll(dirOf(dbPath), 0o755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return fmt.Errorf("set WAL mode: %w", err)
	}

	schemaBytes, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	if _, err := db.Exec(string(schemaBytes)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	return nil
}

func Open(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("db path is empty")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	return db, nil
}

type Workspace struct {
	WorkspaceID   string
	Branch        string
	WorktreePath  string
	CurrentTaskID sql.NullInt64
	AgentState    sql.NullString
	LastCommit    sql.NullString
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Task struct {
	ID          int64
	Description string
	Status      string
	TestCommand sql.NullString
	CreatedAt   time.Time
	CompletedAt sql.NullTime
}

type Trace struct {
	ID         int64
	TaskID     int64
	Command    sql.NullString
	ExitCode   sql.NullInt64
	Output     sql.NullString
	DurationMs sql.NullInt64
	RunAt      time.Time
}

type Checkpoint struct {
	ID          int64
	WorkspaceID string
	CommitHash  string
	AgentState  sql.NullString
	Message     sql.NullString
	CreatedAt   time.Time
}

func EnsureWorkspace(db *sql.DB, workspaceID, branch, worktreePath, lastCommit string) (Workspace, error) {
	if workspaceID == "" || branch == "" {
		return Workspace{}, fmt.Errorf("workspace_id and branch are required")
	}

	if _, err := db.Exec(
		`INSERT OR IGNORE INTO workspaces (workspace_id, branch, worktree_path, last_commit)
         VALUES (?, ?, ?, ?)`,
		workspaceID, branch, worktreePath, lastCommit,
	); err != nil {
		return Workspace{}, fmt.Errorf("insert workspace: %w", err)
	}

	if _, err := db.Exec(
		`UPDATE workspaces
         SET branch = ?, worktree_path = ?, last_commit = ?, updated_at = CURRENT_TIMESTAMP
         WHERE workspace_id = ?`,
		branch, worktreePath, lastCommit, workspaceID,
	); err != nil {
		return Workspace{}, fmt.Errorf("update workspace: %w", err)
	}

	return GetWorkspace(db, workspaceID)
}

func GetWorkspace(db *sql.DB, workspaceID string) (Workspace, error) {
	var ws Workspace
	row := db.QueryRow(
		`SELECT workspace_id, branch, worktree_path, current_task_id, agent_state, last_commit, created_at, updated_at
         FROM workspaces WHERE workspace_id = ?`,
		workspaceID,
	)
	if err := row.Scan(
		&ws.WorkspaceID,
		&ws.Branch,
		&ws.WorktreePath,
		&ws.CurrentTaskID,
		&ws.AgentState,
		&ws.LastCommit,
		&ws.CreatedAt,
		&ws.UpdatedAt,
	); err != nil {
		return Workspace{}, err
	}
	return ws, nil
}

func CreateTask(db *sql.DB, workspaceID, description, testCommand string) (int64, error) {
	if description == "" {
		return 0, fmt.Errorf("description is required")
	}
	res, err := db.Exec(
		`INSERT INTO tasks (workspace_id, description, status, test_command)
         VALUES (?, ?, 'pending', ?)`,
		workspaceID, description, nullIfEmpty(testCommand),
	)
	if err != nil {
		return 0, fmt.Errorf("insert task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("task id: %w", err)
	}
	return id, nil
}

func GetTask(db *sql.DB, taskID int64) (Task, error) {
	var t Task
	row := db.QueryRow(
		`SELECT id, description, status, test_command, created_at, completed_at
         FROM tasks WHERE id = ?`,
		taskID,
	)
	if err := row.Scan(&t.ID, &t.Description, &t.Status, &t.TestCommand, &t.CreatedAt, &t.CompletedAt); err != nil {
		return Task{}, err
	}
	return t, nil
}

func ListTasks(db *sql.DB, workspaceID string, limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Query(
		`SELECT id, description, status, test_command, created_at, completed_at
         FROM tasks WHERE workspace_id = ? ORDER BY id DESC LIMIT ?`,
		workspaceID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Description, &t.Status, &t.TestCommand, &t.CreatedAt, &t.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

func CountTasksByStatus(db *sql.DB, workspaceID string) (map[string]int, error) {
	rows, err := db.Query(
		`SELECT status, COUNT(*) FROM tasks WHERE workspace_id = ? GROUP BY status`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("count tasks: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan counts: %w", err)
		}
		counts[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("count tasks: %w", err)
	}
	return counts, nil
}

func UpdateTaskStatus(db *sql.DB, taskID int64, status string) error {
	if status == "" {
		return fmt.Errorf("status is required")
	}
	completedAt := "NULL"
	switch status {
	case "done", "failed", "blocked", "skipped":
		completedAt = "CURRENT_TIMESTAMP"
	}
	_, err := db.Exec(
		`UPDATE tasks SET status = ?, completed_at = `+completedAt+` WHERE id = ?`,
		status, taskID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

func SetWorkspaceCurrentTask(db *sql.DB, workspaceID string, taskID int64) error {
	_, err := db.Exec(
		`UPDATE workspaces SET current_task_id = ?, updated_at = CURRENT_TIMESTAMP WHERE workspace_id = ?`,
		taskID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("set current task: %w", err)
	}
	return nil
}

func ClearWorkspaceCurrentTask(db *sql.DB, workspaceID string) error {
	_, err := db.Exec(
		`UPDATE workspaces SET current_task_id = NULL, updated_at = CURRENT_TIMESTAMP WHERE workspace_id = ?`,
		workspaceID,
	)
	if err != nil {
		return fmt.Errorf("clear current task: %w", err)
	}
	return nil
}

func RecordTrace(db *sql.DB, taskID int64, command string, exitCode int, output string, durationMs int64) error {
	_, err := db.Exec(
		`INSERT INTO traces (task_id, command, exit_code, output, duration_ms)
         VALUES (?, ?, ?, ?, ?)`,
		taskID, command, exitCode, output, durationMs,
	)
	if err != nil {
		return fmt.Errorf("record trace: %w", err)
	}
	return nil
}

func CreateCheckpoint(db *sql.DB, workspaceID, commitHash, agentState, message string) error {
	if workspaceID == "" || commitHash == "" {
		return fmt.Errorf("workspace_id and commit_hash are required")
	}
	_, err := db.Exec(
		`INSERT INTO checkpoints (workspace_id, commit_hash, agent_state, message) VALUES (?, ?, ?, ?)`,
		workspaceID, commitHash, nullIfEmpty(agentState), nullIfEmpty(message),
	)
	if err != nil {
		return fmt.Errorf("create checkpoint: %w", err)
	}
	return nil
}

func GetLatestCheckpoint(db *sql.DB, workspaceID string) (Checkpoint, error) {
	var cp Checkpoint
	row := db.QueryRow(
		`SELECT id, workspace_id, commit_hash, agent_state, message, created_at
         FROM checkpoints WHERE workspace_id = ? ORDER BY id DESC LIMIT 1`,
		workspaceID,
	)
	if err := row.Scan(&cp.ID, &cp.WorkspaceID, &cp.CommitHash, &cp.AgentState, &cp.Message, &cp.CreatedAt); err != nil {
		return Checkpoint{}, err
	}
	return cp, nil
}

func ListCheckpoints(db *sql.DB, workspaceID string, limit int) ([]Checkpoint, error) {
	if limit <= 0 {
		limit = 0
	}
	query := `SELECT id, workspace_id, commit_hash, agent_state, message, created_at
         FROM checkpoints WHERE workspace_id = ? ORDER BY id DESC`
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = db.Query(query+` LIMIT ?`, workspaceID, limit)
	} else {
		rows, err = db.Query(query, workspaceID)
	}
	if err != nil {
		return nil, fmt.Errorf("list checkpoints: %w", err)
	}
	defer rows.Close()

	var cps []Checkpoint
	for rows.Next() {
		var cp Checkpoint
		if err := rows.Scan(&cp.ID, &cp.WorkspaceID, &cp.CommitHash, &cp.AgentState, &cp.Message, &cp.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan checkpoint: %w", err)
		}
		cps = append(cps, cp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list checkpoints: %w", err)
	}
	return cps, nil
}

func DeleteCheckpointsByID(db *sql.DB, workspaceID string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, workspaceID)
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	query := `DELETE FROM checkpoints WHERE workspace_id = ? AND id IN (` + strings.Join(placeholders, ",") + `)`
	if _, err := db.Exec(query, args...); err != nil {
		return fmt.Errorf("delete checkpoints: %w", err)
	}
	return nil
}

func UpdateWorkspaceLastCommit(db *sql.DB, workspaceID, commit string) error {
	_, err := db.Exec(
		`UPDATE workspaces SET last_commit = ?, updated_at = CURRENT_TIMESTAMP WHERE workspace_id = ?`,
		commit, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("update last commit: %w", err)
	}
	return nil
}

func UpdateWorkspaceAgentState(db *sql.DB, workspaceID, stateJSON string) error {
	_, err := db.Exec(
		`UPDATE workspaces SET agent_state = ?, updated_at = CURRENT_TIMESTAMP WHERE workspace_id = ?`,
		nullIfEmpty(stateJSON), workspaceID,
	)
	if err != nil {
		return fmt.Errorf("update agent_state: %w", err)
	}
	return nil
}

func ListTraces(db *sql.DB, taskID int64, limit int) ([]Trace, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := db.Query(
		`SELECT id, task_id, command, exit_code, output, duration_ms, run_at
         FROM traces WHERE task_id = ? ORDER BY id DESC LIMIT ?`,
		taskID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}
	defer rows.Close()

	var traces []Trace
	for rows.Next() {
		var t Trace
		if err := rows.Scan(&t.ID, &t.TaskID, &t.Command, &t.ExitCode, &t.Output, &t.DurationMs, &t.RunAt); err != nil {
			return nil, fmt.Errorf("scan trace: %w", err)
		}
		traces = append(traces, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}
	return traces, nil
}

func nullIfEmpty(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "."
}
