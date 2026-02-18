# AGIT

AGIT (Agent Git) is a Go MCP server that stores agent workflow state in a Git repository’s `.git/agit.db`. It provides Git-native persistence for AI coding workflows: tasks, checkpoints, and test traces, tied to a workspace.

## Why

Agent workflows need durable state that stays with the repo and is version-control aware. AGIT keeps that state *inside* `.git/` and exposes it via MCP tools to any agent client.

## Core Concepts

- `Workspace`: Stable workspace ID + branch + agent context.
- `Task`: Atomic unit of work, typically with a test.
- `Checkpoint`: Git commit + snapshot of agent state.
- `Trace`: Test run history.

## Status

This repository is bootstrapped for incremental development. The CLI includes `init`, `status`, `task`, `trace`, and `hook` commands.

## Quick Start

```bash
# from a git repo
agit init
agit status --last-output --max-output 2000
agit task new "Add schema migrations" --test "go test ./..."
agit task list
agit task start 1
agit task done 1 --print --max-output 2000
agit checkpoint "before refactor"
agit checkpoint list
agit restore --force
agit restore --dry-run
agit restore --keep
agit workspace set-state --json '{"focus":"refactor","notes":"..." }'
agit workspace clear-state
agit hook install
agit trace 1 --last --max-output 2000
agit trace 1 --json
```

## Repository Layout

```
cmd/
  agit/        # CLI
  agit-mcp/    # MCP server (stub)
internal/
  app/         # app logic + git helpers
  db/          # DB schema + init
```

## Roadmap (Short)

1. Checkpoints (commit + state snapshot).
2. Restore safety and pruning policies.
3. Agent state schema + migrations.

## Design Notes

- State lives in `.git/agit.db` only.
- Schema versioning is built-in via a `meta` table.
- Workspace identity is stored separately from branch name to allow future multi-worktree support.

## License

TBD
