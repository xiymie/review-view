package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestReviewAgentListsStaticToolInfos(t *testing.T) {
	agent, err := NewReviewAgentFromWorkflow(ReviewWorkflowFunc(func(context.Context, ReviewInput) error { return nil }))
	if err != nil {
		t.Fatalf("new review agent: %v", err)
	}
	infos, err := agent.ToolInfos(context.Background())
	if err != nil {
		t.Fatalf("tool infos: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 tool info, got %d", len(infos))
	}
	if infos[0].Name != ReviewExecuteTaskToolName {
		t.Fatalf("unexpected tool name %q", infos[0].Name)
	}
	if infos[0].ParamsOneOf == nil {
		t.Fatal("expected params schema")
	}
}

func TestReviewAgentInvokesStaticTool(t *testing.T) {
	var gotInput ReviewInput
	agent, err := NewReviewAgentFromWorkflow(ReviewWorkflowFunc(func(_ context.Context, input ReviewInput) error {
		gotInput = input
		return nil
	}))
	if err != nil {
		t.Fatalf("new review agent: %v", err)
	}
	out, err := agent.InvokeTool(context.Background(), ReviewExecuteTaskToolName, `{"task_id":77}`)
	if err != nil {
		t.Fatalf("invoke review agent tool: %v", err)
	}
	if gotInput.TaskID != 77 {
		t.Fatalf("expected task 77, got %d", gotInput.TaskID)
	}
	var result reviewTaskToolOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal tool output: %v", err)
	}
	if result.TaskID != 77 || result.Status != "completed" {
		t.Fatalf("unexpected output: %+v", result)
	}
}

func TestReviewAgentRejectsUnknownTool(t *testing.T) {
	agent, err := NewReviewAgentFromWorkflow(ReviewWorkflowFunc(func(context.Context, ReviewInput) error { return nil }))
	if err != nil {
		t.Fatalf("new review agent: %v", err)
	}
	if _, err := agent.InvokeTool(context.Background(), "missing_tool", `{}`); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing tool error, got %v", err)
	}
}
