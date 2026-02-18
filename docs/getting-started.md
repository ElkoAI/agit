# Getting Started

This guide walks you through the basic AGIT workflow: initialize state, create tasks, run tests, checkpoint, and restore.

## Prerequisites

- Go 1.22+
- Git repository

## 1) Initialize

From the root of your Git repo:

```bash
agit init
```

This creates `.git/agit.db`.

## 2) Create and Run a Task

```bash
agit task new "Add feature X" --test "go test ./..."
agit task list
agit task start 1
# ... implement changes ...
agit task done 1
```

If the task has no test command:

```bash
agit task done 1 --no-test
```

## 3) View Trace History

```bash
agit trace 1 --last --max-output 2000
agit trace 1 --json
```

## 4) Checkpoint and Restore

```bash
agit checkpoint "before refactor"
agit checkpoint list
agit restore --dry-run
agit restore --force
```

To preserve local changes while restoring:

```bash
agit restore --keep
```

## 5) Manage Agent State

```bash
agit workspace set-state --json '{"focus":"refactor","notes":"..."}'
agit workspace clear-state
```

## 6) Install Git Hooks

```bash
agit hook install
agit hook status
agit hook uninstall
```

## MCP Usage

See `docs/mcp.md` for MCP tool names, arguments, and responses.
