package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestStaticReviewToolRegistryRegistersReviewExecuteTaskTool(t *testing.T) {
	workflow := ReviewWorkflowFunc(func(context.Context, ReviewInput) error { return nil })
	registry, err := NewStaticReviewToolRegistry(workflow)
	if err != nil {
		t.Fatalf("new static review tool registry: %v", err)
	}
	tools := registry.List()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	item, ok := registry.Get(ReviewExecuteTaskToolName)
	if !ok {
		t.Fatalf("expected tool %s", ReviewExecuteTaskToolName)
	}
	info, err := item.Info(context.Background())
	if err != nil {
		t.Fatalf("tool info: %v", err)
	}
	if info.Name != ReviewExecuteTaskToolName {
		t.Fatalf("unexpected tool name %q", info.Name)
	}
	if !strings.Contains(info.Desc, "review workflow") {
		t.Fatalf("unexpected tool desc %q", info.Desc)
	}
	if info.ParamsOneOf == nil {
		t.Fatal("expected tool params schema")
	}
}

func TestReviewExecuteTaskGraphToolInvokesWorkflow(t *testing.T) {
	var gotInput ReviewInput
	workflow := ReviewWorkflowFunc(func(_ context.Context, input ReviewInput) error {
		gotInput = input
		return nil
	})
	registry, err := NewStaticReviewToolRegistry(workflow)
	if err != nil {
		t.Fatalf("new static review tool registry: %v", err)
	}
	item, ok := registry.Get(ReviewExecuteTaskToolName)
	if !ok {
		t.Fatalf("expected tool %s", ReviewExecuteTaskToolName)
	}

	out, err := item.InvokableRun(context.Background(), `{"task_id":42}`)
	if err != nil {
		t.Fatalf("invoke review tool: %v", err)
	}
	if gotInput.TaskID != 42 {
		t.Fatalf("expected workflow task 42, got %d", gotInput.TaskID)
	}
	var result reviewTaskToolOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal tool output: %v", err)
	}
	if result.TaskID != 42 || result.Status != "completed" {
		t.Fatalf("unexpected tool output: %+v", result)
	}
}

func TestReviewExecuteTaskGraphToolValidatesInput(t *testing.T) {
	workflow := ReviewWorkflowFunc(func(context.Context, ReviewInput) error { return nil })
	registry, err := NewStaticReviewToolRegistry(workflow)
	if err != nil {
		t.Fatalf("new static review tool registry: %v", err)
	}
	item, _ := registry.Get(ReviewExecuteTaskToolName)
	if _, err := item.InvokableRun(context.Background(), `{"task_id":0}`); err == nil || !strings.Contains(err.Error(), "task_id") {
		t.Fatalf("expected task_id validation error, got %v", err)
	}
}
