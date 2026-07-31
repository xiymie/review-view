package service

import (
	"context"
	"strings"
	"testing"
)

func TestSpecialistReviewAgentBuildsPrompt(t *testing.T) {
	agent, err := NewSpecialistReviewAgent(StaticReviewSkill{
		Name:        "security-review",
		Description: "security desc",
		Prompt:      "check auth and token leaks",
	})
	if err != nil {
		t.Fatalf("new specialist agent: %v", err)
	}

	result, err := agent.Review(context.Background(), SpecialistReviewInput{
		TaskID:         101,
		BasePrompt:     "base review prompt",
		DiffContent:    "diff --git a/main.go b/main.go",
		CommitMessages: "fix auth bug",
	})
	if err != nil {
		t.Fatalf("specialist review: %v", err)
	}
	if result.AgentName != "security-review" || result.Status != "ready" {
		t.Fatalf("unexpected specialist result: %+v", result)
	}
	for _, want := range []string{"base review prompt", "专项审查 Agent：security-review", "security desc", "check auth and token leaks", "fix auth bug", "diff --git"} {
		if !strings.Contains(result.Prompt, want) {
			t.Fatalf("expected specialist prompt to contain %q, got:\n%s", want, result.Prompt)
		}
	}
}

func TestMultiReviewAgentListsAndRunsDefaultAgents(t *testing.T) {
	agent, err := NewDefaultMultiReviewAgent()
	if err != nil {
		t.Fatalf("new default multi review agent: %v", err)
	}
	names := agent.Agents()
	if len(names) != 0 {
		t.Fatalf("expected no default specialist agents when built-in skills are disabled by default, got %d: %v", len(names), names)
	}

	result, err := agent.ReviewAll(context.Background(), SpecialistReviewInput{
		TaskID:      202,
		BasePrompt:  "base prompt",
		DiffContent: "diff content",
	})
	if err != nil {
		t.Fatalf("multi review all: %v", err)
	}
	if result.TaskID != 202 || len(result.Results) != 0 || result.Summary != "未配置专项审查 Agent。" {
		t.Fatalf("unexpected empty default multi review result: %+v", result)
	}
}

func TestMultiReviewAgentSkipsEmptySkillPrompt(t *testing.T) {
	agent, err := NewMultiReviewAgentFromSkills([]StaticReviewSkill{
		{Name: "empty", Description: "empty"},
		{Name: "logic-review", Description: "logic", Prompt: "logic prompt"},
	})
	if err != nil {
		t.Fatalf("new multi review agent: %v", err)
	}
	names := agent.Agents()
	if len(names) != 1 || names[0] != "logic-review" {
		t.Fatalf("expected only logic-review, got %v", names)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
