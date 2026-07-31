package service

import (
	"fmt"
	"strings"

	"review-view/internal/model"
	"review-view/internal/store"
)

type ReviewSkillInput struct {
	Name        string
	Description string
	Prompt      string
	Enabled     bool
	BuiltIn     bool
	SortOrder   int

	// Skill Registry 扩展字段：默认为空，审查注入逻辑不读取这些字段。
	AgentXML          string
	SkillRegistryXML  string
	ToolRegistryXML   string
	PolicyMD          string
	WorkflowMD        string
	ContextSchemaJSON string
	MemorySchemaJSON  string
	MetadataJSON      string
}

type ReviewSkillService struct {
	store store.ReviewSkillStore
}

func NewReviewSkillService(store store.ReviewSkillStore) *ReviewSkillService {
	return &ReviewSkillService{store: store}
}

func (s *ReviewSkillService) List() ([]model.ReviewSkill, error) {
	return s.store.List()
}

func (s *ReviewSkillService) ListEnabled() ([]model.ReviewSkill, error) {
	return s.store.ListEnabled()
}

func (s *ReviewSkillService) Get(id int64) (*model.ReviewSkill, error) {
	return s.store.GetByID(id)
}

func (s *ReviewSkillService) Create(input ReviewSkillInput) (*model.ReviewSkill, error) {
	if err := validateReviewSkillInput(input); err != nil {
		return nil, err
	}
	skill := &model.ReviewSkill{
		Name:              strings.TrimSpace(input.Name),
		Description:       strings.TrimSpace(input.Description),
		Prompt:            strings.TrimSpace(input.Prompt),
		Enabled:           input.Enabled,
		BuiltIn:           input.BuiltIn,
		SortOrder:         input.SortOrder,
		AgentXML:          input.AgentXML,
		SkillRegistryXML:  input.SkillRegistryXML,
		ToolRegistryXML:   input.ToolRegistryXML,
		PolicyMD:          input.PolicyMD,
		WorkflowMD:        input.WorkflowMD,
		ContextSchemaJSON: input.ContextSchemaJSON,
		MemorySchemaJSON:  input.MemorySchemaJSON,
		MetadataJSON:      input.MetadataJSON,
	}
	if err := s.store.Create(skill); err != nil {
		return nil, err
	}
	return skill, nil
}

func (s *ReviewSkillService) Update(id int64, input ReviewSkillInput) (*model.ReviewSkill, error) {
	if err := validateReviewSkillInput(input); err != nil {
		return nil, err
	}
	skill, err := s.store.GetByID(id)
	if err != nil {
		return nil, err
	}
	skill.Name = strings.TrimSpace(input.Name)
	skill.Description = strings.TrimSpace(input.Description)
	skill.Prompt = strings.TrimSpace(input.Prompt)
	skill.Enabled = input.Enabled
	skill.SortOrder = input.SortOrder
	skill.AgentXML = input.AgentXML
	skill.SkillRegistryXML = input.SkillRegistryXML
	skill.ToolRegistryXML = input.ToolRegistryXML
	skill.PolicyMD = input.PolicyMD
	skill.WorkflowMD = input.WorkflowMD
	skill.ContextSchemaJSON = input.ContextSchemaJSON
	skill.MemorySchemaJSON = input.MemorySchemaJSON
	skill.MetadataJSON = input.MetadataJSON
	// BuiltIn is system-owned and cannot be flipped by update.
	if err := s.store.Update(skill); err != nil {
		return nil, err
	}
	return skill, nil
}

func (s *ReviewSkillService) Delete(id int64) error {
	skill, err := s.store.GetByID(id)
	if err != nil {
		return err
	}
	if skill.BuiltIn {
		return fmt.Errorf("内置 Skill 不能删除")
	}
	return s.store.Delete(id)
}

func (s *ReviewSkillService) SetEnabled(id int64, enabled bool) (*model.ReviewSkill, error) {
	skill, err := s.store.GetByID(id)
	if err != nil {
		return nil, err
	}
	skill.Enabled = enabled
	if err := s.store.Update(skill); err != nil {
		return nil, err
	}
	return skill, nil
}

func (s *ReviewSkillService) EnsureBuiltIns() error {
	currentBuiltIns := make(map[string]struct{})
	for i, staticSkill := range LoadStaticReviewSkills() {
		currentBuiltIns[staticSkill.Name] = struct{}{}
		if strings.TrimSpace(staticSkill.Name) == "" || strings.TrimSpace(staticSkill.Prompt) == "" {
			continue
		}
		existing, err := s.store.GetByName(staticSkill.Name)
		if err == nil {
			changed := false
			if !existing.BuiltIn {
				existing.BuiltIn = true
				changed = true
			}
			if existing.SortOrder == 0 {
				existing.SortOrder = (i + 1) * 10
				changed = true
			}
			if changed {
				if err := s.store.Update(existing); err != nil {
					return err
				}
			}
			continue
		}
		_, err = s.Create(ReviewSkillInput{
			Name:        staticSkill.Name,
			Description: staticSkill.Description,
			Prompt:      staticSkill.Prompt,
			Enabled:     staticSkill.DefaultEnabled,
			BuiltIn:     true,
			SortOrder:   (i + 1) * 10,
		})
		if err != nil {
			return err
		}
	}
	return s.deleteStaleBuiltIns(currentBuiltIns)
}

func (s *ReviewSkillService) deleteStaleBuiltIns(current map[string]struct{}) error {
	skills, err := s.store.List()
	if err != nil {
		return err
	}
	for _, skill := range skills {
		if !skill.BuiltIn {
			continue
		}
		if _, ok := current[skill.Name]; ok {
			continue
		}
		if err := s.store.Delete(skill.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *ReviewSkillService) BuildEnabledPrompt() (string, error) {
	skills, err := s.store.ListEnabled()
	if err != nil {
		return "", err
	}
	return buildReviewSkillPromptFromModels(skills), nil
}

func (s *ReviewSkillService) BuildPromptForSkillIDs(skillIDs []int64) (string, error) {
	if len(skillIDs) == 0 {
		return "", nil
	}
	wanted := make(map[int64]struct{}, len(skillIDs))
	for _, id := range skillIDs {
		if id > 0 {
			wanted[id] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return "", nil
	}
	skills, err := s.store.ListEnabled()
	if err != nil {
		return "", err
	}
	selected := make([]model.ReviewSkill, 0, len(skills))
	for _, skill := range skills {
		if _, ok := wanted[skill.ID]; ok {
			selected = append(selected, skill)
		}
	}
	return buildReviewSkillPromptFromModels(selected), nil
}

func buildReviewSkillPromptFromModels(skills []model.ReviewSkill) string {
	if len(skills) == 0 {
		return ""
	}
	staticSkills := make([]StaticReviewSkill, 0, len(skills))
	for _, skill := range skills {
		staticSkills = append(staticSkills, StaticReviewSkill{
			Name:        skill.Name,
			Description: skill.Description,
			Prompt:      skill.Prompt,
		})
	}
	return BuildStaticReviewSkillPrompt(staticSkills)
}

func validateReviewSkillInput(input ReviewSkillInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("Skill 名称不能为空")
	}
	if strings.TrimSpace(input.Prompt) == "" {
		return fmt.Errorf("Skill Prompt 不能为空")
	}
	return nil
}
