package app

import (
	"encoding/json"
	"fmt"

	"github.com/agentgit/agit/internal/db"
)

func SetAgentState(ctx *Context, stateJSON string) error {
	if !json.Valid([]byte(stateJSON)) {
		return fmt.Errorf("agent_state is not valid JSON")
	}
	return db.UpdateWorkspaceAgentState(ctx.DB, ctx.WorkspaceID, stateJSON)
}

func ClearAgentState(ctx *Context) error {
	return db.UpdateWorkspaceAgentState(ctx.DB, ctx.WorkspaceID, "")
}
