package app

import (
	"testing"

	"github.com/agentgit/agit/internal/db"
)

func TestSetAgentState_Validation(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Close()

	if err := SetAgentState(ctx, "{invalid"); err == nil {
		t.Fatal("expected invalid JSON error")
	}

	if err := SetAgentState(ctx, "{\"ok\":true}"); err != nil {
		t.Fatalf("SetAgentState: %v", err)
	}

	ws, err := db.GetWorkspace(ctx.DB, ctx.WorkspaceID)
	if err != nil {
		t.Fatalf("GetWorkspace: %v", err)
	}
	if !ws.AgentState.Valid || ws.AgentState.String == "" {
		t.Fatal("expected agent_state to be set")
	}
}
