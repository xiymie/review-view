package service

import (
	"strings"
	"testing"
)

// TestBuildPromptUnchangedByRegistryFields verifies that BuildPrompt output
// is identical whether or not the new registry fields are populated.
func TestBuildPromptUnchangedByRegistryFields(t *testing.T) {
	stores := newReviewSkillStores(t)
	svc := NewReviewSkillService(stores.ReviewSkills)

	// Create a skill without registry fields.
	plain, err := svc.Create(ReviewSkillInput{
		Name:      "plain-skill",
		Prompt:    "plain prompt body",
		Enabled:   true,
		SortOrder: 10,
	})
	if err != nil {
		t.Fatalf("create plain skill: %v", err)
	}

	promptPlain, err := svc.BuildEnabledPrompt()
	if err != nil {
		t.Fatalf("build plain prompt: %v", err)
	}

	// Update the same skill with all registry fields populated.
	_, err = svc.Update(plain.ID, ReviewSkillInput{
		Name:              "plain-skill",
		Prompt:            "plain prompt body",
		Enabled:           true,
		SortOrder:         10,
		AgentXML:          "<agent/>",
		SkillRegistryXML:  "<skill-registry/>",
		ToolRegistryXML:   "<tool-registry/>",
		PolicyMD:          "# policy",
		WorkflowMD:        "# workflow",
		ContextSchemaJSON: `{"type":"object"}`,
		MemorySchemaJSON:  `{"type":"array"}`,
		MetadataJSON:      `{"version":"1"}`,
	})
	if err != nil {
		t.Fatalf("update skill with registry fields: %v", err)
	}

	promptWithRegistry, err := svc.BuildEnabledPrompt()
	if err != nil {
		t.Fatalf("build prompt with registry fields: %v", err)
	}

	if promptPlain != promptWithRegistry {
		t.Fatalf("BuildPrompt changed when registry fields were added:\nbefore:\n%s\nafter:\n%s",
			promptPlain, promptWithRegistry)
	}
	if !strings.Contains(promptPlain, "plain prompt body") {
		t.Fatalf("expected prompt to contain skill body, got:\n%s", promptPlain)
	}
}

// TestRegistryFieldsRoundtrip verifies that all registry fields persist and
// are returned unchanged via Create → Get and Update → Get.
func TestRegistryFieldsRoundtrip(t *testing.T) {
	stores := newReviewSkillStores(t)
	svc := NewReviewSkillService(stores.ReviewSkills)

	input := ReviewSkillInput{
		Name:              "registry-skill",
		Prompt:            "some prompt",
		Enabled:           false,
		SortOrder:         5,
		AgentXML:          "<agent>foo</agent>",
		SkillRegistryXML:  "<skill-registry>bar</skill-registry>",
		ToolRegistryXML:   "<tool-registry>baz</tool-registry>",
		PolicyMD:          "# Policy\nDo things.",
		WorkflowMD:        "# Workflow\nStep 1.",
		ContextSchemaJSON: `{"type":"object","properties":{}}`,
		MemorySchemaJSON:  `{"type":"array","items":{}}`,
		MetadataJSON:      `{"version":"2","author":"test"}`,
	}

	created, err := svc.Create(input)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	assertRegistryFields(t, "after create", created.ID, input, svc)

	// Update with different values.
	updated := input
	updated.AgentXML = "<agent>updated</agent>"
	updated.MetadataJSON = `{"version":"3"}`
	if _, err := svc.Update(created.ID, updated); err != nil {
		t.Fatalf("update: %v", err)
	}

	assertRegistryFields(t, "after update", created.ID, updated, svc)
}

// TestBuiltInRegistryFieldsDefaultEmpty ensures built-in skills seeded by
// EnsureBuiltIns have all registry fields empty (zero value).
func TestBuiltInRegistryFieldsDefaultEmpty(t *testing.T) {
	stores := newReviewSkillStores(t)
	svc := NewReviewSkillService(stores.ReviewSkills)

	if err := svc.EnsureBuiltIns(); err != nil {
		t.Fatalf("ensure built-ins: %v", err)
	}

	skills, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, skill := range skills {
		if !skill.BuiltIn {
			continue
		}
		if skill.AgentXML != "" || skill.SkillRegistryXML != "" || skill.ToolRegistryXML != "" ||
			skill.PolicyMD != "" || skill.WorkflowMD != "" ||
			skill.ContextSchemaJSON != "" || skill.MemorySchemaJSON != "" || skill.MetadataJSON != "" {
			t.Errorf("built-in skill %q has non-empty registry field(s): AgentXML=%q SkillRegistryXML=%q ToolRegistryXML=%q PolicyMD=%q WorkflowMD=%q ContextSchemaJSON=%q MemorySchemaJSON=%q MetadataJSON=%q",
				skill.Name,
				skill.AgentXML, skill.SkillRegistryXML, skill.ToolRegistryXML,
				skill.PolicyMD, skill.WorkflowMD,
				skill.ContextSchemaJSON, skill.MemorySchemaJSON, skill.MetadataJSON)
		}
	}
}

func assertRegistryFields(t *testing.T, when string, id int64, want ReviewSkillInput, svc *ReviewSkillService) {
	t.Helper()
	got, err := svc.Get(id)
	if err != nil {
		t.Fatalf("%s get: %v", when, err)
	}
	type pair struct{ field, got, want string }
	checks := []pair{
		{"AgentXML", got.AgentXML, want.AgentXML},
		{"SkillRegistryXML", got.SkillRegistryXML, want.SkillRegistryXML},
		{"ToolRegistryXML", got.ToolRegistryXML, want.ToolRegistryXML},
		{"PolicyMD", got.PolicyMD, want.PolicyMD},
		{"WorkflowMD", got.WorkflowMD, want.WorkflowMD},
		{"ContextSchemaJSON", got.ContextSchemaJSON, want.ContextSchemaJSON},
		{"MemorySchemaJSON", got.MemorySchemaJSON, want.MemorySchemaJSON},
		{"MetadataJSON", got.MetadataJSON, want.MetadataJSON},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s %s: got %q, want %q", when, c.field, c.got, c.want)
		}
	}
}
