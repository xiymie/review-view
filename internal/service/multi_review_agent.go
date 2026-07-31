package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
)

type SpecialistReviewInput struct {
	TaskID         int64  `json:"task_id"`
	BasePrompt     string `json:"base_prompt"`
	DiffContent    string `json:"diff_content"`
	CommitMessages string `json:"commit_messages"`
}

type SpecialistReviewResult struct {
	AgentName   string `json:"agent_name"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	Status      string `json:"status"`
}

type MultiReviewResult struct {
	TaskID  int64                    `json:"task_id"`
	Results []SpecialistReviewResult `json:"results"`
	Summary string                   `json:"summary"`
}

type SpecialistReviewAgent struct {
	skill   StaticReviewSkill
	runner  compose.Runnable[SpecialistReviewInput, SpecialistReviewResult]
	initErr error
}

func NewSpecialistReviewAgent(skill StaticReviewSkill) (*SpecialistReviewAgent, error) {
	if strings.TrimSpace(skill.Name) == "" {
		return nil, fmt.Errorf("specialist review agent name is required")
	}
	if strings.TrimSpace(skill.Prompt) == "" {
		return nil, fmt.Errorf("specialist review agent prompt is required")
	}
	agent := &SpecialistReviewAgent{skill: skill}
	agent.runner, agent.initErr = agent.build(context.Background())
	if agent.initErr != nil {
		return nil, agent.initErr
	}
	return agent, nil
}

func (a *SpecialistReviewAgent) Name() string {
	if a == nil {
		return ""
	}
	return a.skill.Name
}

func (a *SpecialistReviewAgent) Description() string {
	if a == nil {
		return ""
	}
	return a.skill.Description
}

func (a *SpecialistReviewAgent) Review(ctx context.Context, input SpecialistReviewInput) (SpecialistReviewResult, error) {
	if a == nil || a.runner == nil {
		return SpecialistReviewResult{}, fmt.Errorf("specialist review agent is not initialized")
	}
	return a.runner.Invoke(ctx, input)
}

func (a *SpecialistReviewAgent) build(ctx context.Context) (compose.Runnable[SpecialistReviewInput, SpecialistReviewResult], error) {
	chain := compose.NewChain[SpecialistReviewInput, SpecialistReviewResult]().
		AppendLambda(compose.InvokableLambda(a.buildSpecialistPrompt), compose.WithNodeName("build_specialist_prompt")).
		AppendLambda(compose.InvokableLambda(a.buildSpecialistResult), compose.WithNodeName("build_specialist_result"))
	return chain.Compile(ctx, compose.WithGraphName("specialist_review_agent_"+a.skill.Name))
}

func (a *SpecialistReviewAgent) buildSpecialistPrompt(_ context.Context, input SpecialistReviewInput) (SpecialistReviewResult, error) {
	var b strings.Builder
	if strings.TrimSpace(input.BasePrompt) != "" {
		b.WriteString(strings.TrimSpace(input.BasePrompt))
		b.WriteString("\n\n")
	}
	b.WriteString("## 专项审查 Agent：")
	b.WriteString(a.skill.Name)
	b.WriteString("\n\n")
	if a.skill.Description != "" {
		b.WriteString(a.skill.Description)
		b.WriteString("\n\n")
	}
	b.WriteString(strings.TrimSpace(a.skill.Prompt))
	if strings.TrimSpace(input.CommitMessages) != "" {
		b.WriteString("\n\n## Commit 信息\n\n")
		b.WriteString(strings.TrimSpace(input.CommitMessages))
	}
	if strings.TrimSpace(input.DiffContent) != "" {
		b.WriteString("\n\n## Diff 内容\n\n")
		b.WriteString(strings.TrimSpace(input.DiffContent))
	}
	return SpecialistReviewResult{
		AgentName:   a.skill.Name,
		Description: a.skill.Description,
		Prompt:      b.String(),
		Status:      "prepared",
	}, nil
}

func (a *SpecialistReviewAgent) buildSpecialistResult(_ context.Context, result SpecialistReviewResult) (SpecialistReviewResult, error) {
	result.Status = "ready"
	return result, nil
}

type MultiReviewAgent struct {
	agents []*SpecialistReviewAgent
}

func NewMultiReviewAgentFromSkills(skills []StaticReviewSkill) (*MultiReviewAgent, error) {
	agents := make([]*SpecialistReviewAgent, 0, len(skills))
	for _, skill := range skills {
		if strings.TrimSpace(skill.Prompt) == "" {
			continue
		}
		agent, err := NewSpecialistReviewAgent(skill)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return &MultiReviewAgent{agents: agents}, nil
}

func NewDefaultMultiReviewAgent() (*MultiReviewAgent, error) {
	return NewMultiReviewAgentFromSkills(LoadDefaultEnabledStaticReviewSkills())
}

func (a *MultiReviewAgent) Agents() []string {
	if a == nil {
		return nil
	}
	names := make([]string, 0, len(a.agents))
	for _, agent := range a.agents {
		names = append(names, agent.Name())
	}
	return names
}

func (a *MultiReviewAgent) ReviewAll(ctx context.Context, input SpecialistReviewInput) (MultiReviewResult, error) {
	if a == nil {
		return MultiReviewResult{}, fmt.Errorf("multi review agent is not initialized")
	}
	results := make([]SpecialistReviewResult, 0, len(a.agents))
	for _, agent := range a.agents {
		result, err := agent.Review(ctx, input)
		if err != nil {
			return MultiReviewResult{}, err
		}
		results = append(results, result)
	}
	return MultiReviewResult{
		TaskID:  input.TaskID,
		Results: results,
		Summary: buildMultiReviewSummary(results),
	}, nil
}

func buildMultiReviewSummary(results []SpecialistReviewResult) string {
	if len(results) == 0 {
		return "未配置专项审查 Agent。"
	}
	names := make([]string, 0, len(results))
	for _, result := range results {
		names = append(names, result.AgentName)
	}
	return fmt.Sprintf("已准备 %d 个专项审查 Agent：%s。", len(results), strings.Join(names, ", "))
}
