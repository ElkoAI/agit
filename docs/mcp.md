# AGIT MCP Tools

This document describes the MCP tools exposed by `agit-mcp`, their arguments, and example responses.

## workspace_status
Get current workspace state.

Arguments:
- `include_output` (bool, optional): include last trace output if available
- `include_state` (bool, optional): include `agent_state` JSON string if available
- `max_output` (number, optional): max output bytes to return (0 = no limit)

Response (example):
```json
{
  "workspace_id": "<id>",
  "branch": "main",
  "tasks": {"pending": 2, "done": 1},
  "last_commit": "<hash>",
  "current_task_id": 3,
  "last_trace": {
    "id": 12,
    "task_id": 3,
    "run_at": "2026-02-18T12:34:56-05:00",
    "command": "go test ./...",
    "exit_code": 0,
    "duration_ms": 412,
    "output": "..."
  },
  "agent_state": "{\"focus\":\"refactor\"}"
}
```

## task_new
Create a new task.

Arguments:
- `description` (string, required)
- `test_command` (string, optional)

Response:
```json
{"task_id": 1}
```

## task_list
List tasks for the current workspace.

Arguments:
- `limit` (number, optional)

Response:
```json
[
  {"id": 1, "description": "Add feature", "status": "pending", "test_command": "go test ./..."}
]
```

## task_start
Mark a task as active.

Arguments:
- `task_id` (number, required)

Response:
```
"task started"
```

## task_done
Run test and mark task done.

Arguments:
- `task_id` (number, required)
- `no_test` (bool, optional): mark done without running tests

Response:
```json
{"exit_code": 0}
```

## trace
Show recent test traces for a task.

Arguments:
- `task_id` (number, required)
- `limit` (number, optional)
- `last` (bool, optional)
- `max_output` (number, optional)

Response:
```json
[
  {
    "id": 5,
    "task_id": 1,
    "run_at": "2026-02-18T12:34:56-05:00",
    "command": "go test ./...",
    "exit_code": 0,
    "duration_ms": 412,
    "output": "..."
  }
]
```

## checkpoint
Create a checkpoint (git commit + state).

Arguments:
- `message` (string, required)

Response:
```json
{"commit": "<hash>", "message": "before refactor"}
```

## checkpoint_list
List checkpoints.

Arguments:
- `limit` (number, optional)

Response:
```json
[
  {"id": 1, "commit": "<hash>", "created_at": "2026-02-18T12:34:56-05:00", "message": "before refactor"}
]
```

## restore
Restore to latest checkpoint.

Arguments:
- `force` (bool, optional)
- `dry_run` (bool, optional)
- `keep` (bool, optional)

Response:
```json
{"commit": "<hash>", "dry_run": false, "keep": false}
```

## workspace_update
Update workspace `agent_state` JSON.

Arguments:
- `agent_state` (string, required)

Response:
```
"agent_state updated"
```

## workspace_clear_state
Clear workspace `agent_state`.

Arguments:
- none

Response:
```
"agent_state cleared"
```
