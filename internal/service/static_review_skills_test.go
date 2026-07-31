package service

import (
	"strings"
	"testing"
)

func TestStaticReviewSkillsBuildPrompt(t *testing.T) {
	skills := LoadStaticReviewSkills()
	if len(skills) != 13 {
		t.Fatalf("expected 13 static review skills, got %d", len(skills))
	}
	defaults := LoadDefaultEnabledStaticReviewSkills()
	if len(defaults) != 0 {
		t.Fatalf("expected 0 default-enabled static review skills, got %d", len(defaults))
	}
	prompt := BuildStaticReviewSkillPrompt(skills)
	for _, want := range []string{"## 静态 Review Skills", "性能优化专家", "前端体验工程师", "security-audit", "performance-optimizer", "Measure first"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected skill prompt to contain %q, got:\n%s", want, prompt)
		}
	}
}

// TestStaticReviewSkillsBuildPromptIgnoresDefaultEnabled verifies that
// BuildStaticReviewSkillPrompt output is driven solely by the slice passed in,
// not by the DefaultEnabled flag — that field governs seeding, not rendering.
func TestStaticReviewSkillsBuildPromptIgnoresDefaultEnabled(t *testing.T) {
	// Pass two skills: one with DefaultEnabled true, one false.
	skills := []StaticReviewSkill{
		{Name: "enabled-skill", Description: "desc-e", Prompt: "prompt-e", DefaultEnabled: true},
		{Name: "disabled-skill", Description: "desc-d", Prompt: "prompt-d", DefaultEnabled: false},
	}
	prompt := BuildStaticReviewSkillPrompt(skills)
	for _, want := range []string{"prompt-e", "prompt-d"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("expected %q in output (DefaultEnabled should not filter), got:\n%s", want, prompt)
		}
	}
}

func TestStaticReviewSkillsBuildPromptSkipsEmptyPrompt(t *testing.T) {
	prompt := BuildStaticReviewSkillPrompt([]StaticReviewSkill{
		{Name: "empty", Description: "empty desc"},
		{Name: "real", Description: "real desc", Prompt: "real prompt"},
	})
	if strings.Contains(prompt, "empty desc") {
		t.Fatalf("expected empty skill prompt to be skipped, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "real") || !strings.Contains(prompt, "real prompt") {
		t.Fatalf("expected real skill prompt, got:\n%s", prompt)
	}
}
