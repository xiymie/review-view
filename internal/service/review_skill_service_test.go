package service

import (
	"strings"
	"testing"

	"review-view/internal/model"
	gormstore "review-view/internal/store/gorm"
)

func TestReviewSkillServiceEnsuresBuiltInsAndBuildsPrompt(t *testing.T) {
	stores := newReviewSkillStores(t)
	svc := NewReviewSkillService(stores.ReviewSkills)

	if err := svc.EnsureBuiltIns(); err != nil {
		t.Fatalf("ensure built-ins: %v", err)
	}
	if err := svc.EnsureBuiltIns(); err != nil {
		t.Fatalf("ensure built-ins idempotent: %v", err)
	}

	skills, err := svc.List()
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if len(skills) != 13 {
		t.Fatalf("expected 13 built-in skills, got %d", len(skills))
	}
	for _, skill := range skills {
		if !skill.BuiltIn {
			t.Fatalf("expected built-in skill, got %+v", skill)
		}
	}
	enabled, err := svc.ListEnabled()
	if err != nil {
		t.Fatalf("list enabled skills: %v", err)
	}
	if len(enabled) != 0 {
		t.Fatalf("expected built-ins to be disabled by default, got %d: %+v", len(enabled), enabled)
	}

	prompt, err := svc.BuildEnabledPrompt()
	if err != nil {
		t.Fatalf("build enabled prompt: %v", err)
	}
	for _, want := range []string{"智能代码审查", "架构审查", "性能优化专家"} {
		if strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to be empty before enabling skills, got %q in:\n%s", want, prompt)
		}
	}
	perf, err := stores.ReviewSkills.GetByName("性能优化专家")
	if err != nil {
		t.Fatalf("get performance-optimizer: %v", err)
	}
	if perf.Enabled {
		t.Fatalf("expected 性能优化专家 to be disabled by default")
	}
	if _, err := svc.SetEnabled(perf.ID, true); err != nil {
		t.Fatalf("enable performance-optimizer: %v", err)
	}
	prompt, err = svc.BuildEnabledPrompt()
	if err != nil {
		t.Fatalf("build prompt after enabling performance-optimizer: %v", err)
	}
	if !strings.Contains(prompt, "性能优化专家") || !strings.Contains(prompt, "performance-optimizer") || !strings.Contains(prompt, "Measure first") {
		t.Fatalf("expected enabled performance-optimizer prompt, got:\n%s", prompt)
	}
}

func TestReviewSkillServiceCRUDAndBuiltInDeleteGuard(t *testing.T) {
	stores := newReviewSkillStores(t)
	svc := NewReviewSkillService(stores.ReviewSkills)

	custom, err := svc.Create(ReviewSkillInput{
		Name:        "custom-skill",
		Description: "custom desc",
		Prompt:      "custom prompt",
		Enabled:     true,
		SortOrder:   99,
	})
	if err != nil {
		t.Fatalf("create skill: %v", err)
	}

	updated, err := svc.Update(custom.ID, ReviewSkillInput{
		Name:        "custom-skill-renamed",
		Description: "new desc",
		Prompt:      "new prompt",
		Enabled:     false,
		SortOrder:   100,
	})
	if err != nil {
		t.Fatalf("update skill: %v", err)
	}
	if updated.Name != "custom-skill-renamed" || updated.Enabled {
		t.Fatalf("unexpected updated skill: %+v", updated)
	}

	if _, err := svc.SetEnabled(custom.ID, true); err != nil {
		t.Fatalf("enable skill: %v", err)
	}
	enabled, err := svc.ListEnabled()
	if err != nil {
		t.Fatalf("list enabled: %v", err)
	}
	if len(enabled) != 1 || enabled[0].Name != "custom-skill-renamed" {
		t.Fatalf("unexpected enabled skills: %+v", enabled)
	}

	if err := svc.Delete(custom.ID); err != nil {
		t.Fatalf("delete custom skill: %v", err)
	}

	builtIn := &model.ReviewSkill{Name: "built-in", Prompt: "prompt", Enabled: true, BuiltIn: true}
	if err := stores.ReviewSkills.Create(builtIn); err != nil {
		t.Fatalf("create built-in: %v", err)
	}
	if err := svc.Delete(builtIn.ID); err == nil || !strings.Contains(err.Error(), "内置") {
		t.Fatalf("expected built-in delete guard, got %v", err)
	}
}

// TestReviewSkillServiceFieldsRoundtrip verifies that Description, Prompt,
// SortOrder, and BuiltIn survive a full create→get→update cycle, and that
// Update cannot flip BuiltIn (it is system-owned).
func TestReviewSkillServiceFieldsRoundtrip(t *testing.T) {
	stores := newReviewSkillStores(t)
	svc := NewReviewSkillService(stores.ReviewSkills)

	created, err := svc.Create(ReviewSkillInput{
		Name:        "roundtrip-skill",
		Description: "original description",
		Prompt:      "original prompt",
		Enabled:     false,
		BuiltIn:     true,
		SortOrder:   42,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Description != "original description" {
		t.Errorf("Description create: got %q", created.Description)
	}
	if created.Prompt != "original prompt" {
		t.Errorf("Prompt create: got %q", created.Prompt)
	}
	if created.SortOrder != 42 {
		t.Errorf("SortOrder create: got %d", created.SortOrder)
	}
	if !created.BuiltIn {
		t.Errorf("BuiltIn create: got false")
	}

	updated, err := svc.Update(created.ID, ReviewSkillInput{
		Name:        "roundtrip-skill",
		Description: "updated description",
		Prompt:      "updated prompt",
		Enabled:     true,
		SortOrder:   99,
		// BuiltIn omitted — Update must not flip the flag
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Description != "updated description" {
		t.Errorf("Description update: got %q", updated.Description)
	}
	if updated.Prompt != "updated prompt" {
		t.Errorf("Prompt update: got %q", updated.Prompt)
	}
	if updated.SortOrder != 99 {
		t.Errorf("SortOrder update: got %d", updated.SortOrder)
	}
	if !updated.Enabled {
		t.Errorf("Enabled update: got false")
	}
	if !updated.BuiltIn {
		t.Errorf("BuiltIn must not be flipped by Update: got false")
	}

	// Confirm persistence via a fresh Get.
	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if got.Prompt != "updated prompt" || got.Description != "updated description" || got.SortOrder != 99 {
		t.Errorf("persisted fields mismatch: %+v", got)
	}
}

// TestReviewSkillServiceEnsureBuiltInsDoesNotAlterPrompt verifies that a
// re-seed via EnsureBuiltIns never overwrites a Prompt that has been
// customised since the initial seed.
func TestReviewSkillServiceEnsureBuiltInsDoesNotAlterPrompt(t *testing.T) {
	stores := newReviewSkillStores(t)
	svc := NewReviewSkillService(stores.ReviewSkills)

	if err := svc.EnsureBuiltIns(); err != nil {
		t.Fatalf("initial EnsureBuiltIns: %v", err)
	}

	const targetName = "性能优化专家"
	skill, err := stores.ReviewSkills.GetByName(targetName)
	if err != nil {
		t.Fatalf("get built-in: %v", err)
	}

	skill.Prompt = "customised prompt"
	if err := stores.ReviewSkills.Update(skill); err != nil {
		t.Fatalf("manual prompt customisation: %v", err)
	}

	// Re-seed must leave the customised Prompt alone.
	if err := svc.EnsureBuiltIns(); err != nil {
		t.Fatalf("re-seed EnsureBuiltIns: %v", err)
	}

	after, err := stores.ReviewSkills.GetByName(targetName)
	if err != nil {
		t.Fatalf("get after re-seed: %v", err)
	}
	if after.Prompt != "customised prompt" {
		t.Errorf("EnsureBuiltIns must not overwrite Prompt: got %q", after.Prompt)
	}
}

// TestReviewSkillServiceBuildPromptForSkillIDs verifies that:
//   - only the requested enabled skills appear in the output;
//   - disabled skills are excluded even when their ID is in the request set;
//   - nil / empty ID list returns an empty string;
//   - output is prompt-only — structural fields (SortOrder, BuiltIn) do not
//     appear in the rendered text.
func TestReviewSkillServiceBuildPromptForSkillIDs(t *testing.T) {
	stores := newReviewSkillStores(t)
	svc := NewReviewSkillService(stores.ReviewSkills)

	a, err := svc.Create(ReviewSkillInput{Name: "skill-a", Prompt: "prompt content A", Enabled: true, SortOrder: 10})
	if err != nil {
		t.Fatalf("create a: %v", err)
	}
	b, err := svc.Create(ReviewSkillInput{Name: "skill-b", Prompt: "prompt content B", Enabled: true, SortOrder: 20})
	if err != nil {
		t.Fatalf("create b: %v", err)
	}
	c, err := svc.Create(ReviewSkillInput{Name: "skill-c", Prompt: "prompt content C", Enabled: false, SortOrder: 30})
	if err != nil {
		t.Fatalf("create c: %v", err)
	}

	// Only skill-a requested; skill-b must be excluded.
	prompt, err := svc.BuildPromptForSkillIDs([]int64{a.ID})
	if err != nil {
		t.Fatalf("BuildPromptForSkillIDs single: %v", err)
	}
	if !strings.Contains(prompt, "prompt content A") {
		t.Errorf("expected prompt A, got:\n%s", prompt)
	}
	if strings.Contains(prompt, "prompt content B") {
		t.Errorf("prompt B must be excluded, got:\n%s", prompt)
	}

	// Disabled skill-c must not appear even when its ID is explicitly requested.
	promptC, err := svc.BuildPromptForSkillIDs([]int64{c.ID})
	if err != nil {
		t.Fatalf("BuildPromptForSkillIDs disabled: %v", err)
	}
	if strings.Contains(promptC, "prompt content C") {
		t.Errorf("disabled skill must not appear in output, got:\n%s", promptC)
	}
	if promptC != "" {
		t.Errorf("expected empty string for disabled-only request, got %q", promptC)
	}

	// Nil and empty ID lists → empty string.
	for _, ids := range [][]int64{nil, {}} {
		empty, err := svc.BuildPromptForSkillIDs(ids)
		if err != nil {
			t.Fatalf("BuildPromptForSkillIDs empty/nil: %v", err)
		}
		if empty != "" {
			t.Errorf("expected empty string for %v IDs, got %q", ids, empty)
		}
	}

	// Output is prompt-only: structural field names must not leak into rendered text.
	combined, err := svc.BuildPromptForSkillIDs([]int64{a.ID, b.ID})
	if err != nil {
		t.Fatalf("BuildPromptForSkillIDs combined: %v", err)
	}
	for _, forbidden := range []string{"SortOrder", "sort_order", "BuiltIn", "built_in"} {
		if strings.Contains(combined, forbidden) {
			t.Errorf("structural field %q must not appear in prompt output, got:\n%s", forbidden, combined)
		}
	}
}

func newReviewSkillStores(t *testing.T) gormstore.Stores {
	t.Helper()
	db, err := gormstore.Open("file:" + t.TempDir() + "/review-skill.db?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return gormstore.New(db)
}
