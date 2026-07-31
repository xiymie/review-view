package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
)

// ReviewAgent is the minimal backend prototype for the future review agent.
//
// It does not call an LLM yet. The current scope is intentionally small: hold a
// static Eino tool registry, expose tool metadata, and provide a direct tool
// invocation entry. This makes the Tool layer testable before wiring ReAct or
// another agent loop.
type ReviewAgent struct {
	registry *ReviewToolRegistry
}

func NewReviewAgent(registry *ReviewToolRegistry) (*ReviewAgent, error) {
	if registry == nil {
		return nil, fmt.Errorf("review tool registry is nil")
	}
	return &ReviewAgent{registry: registry}, nil
}

func NewReviewAgentFromWorkflow(workflow ReviewWorkflow) (*ReviewAgent, error) {
	registry, err := NewStaticReviewToolRegistry(workflow)
	if err != nil {
		return nil, err
	}
	return NewReviewAgent(registry)
}

func (a *ReviewAgent) ToolInfos(ctx context.Context) ([]*schema.ToolInfo, error) {
	if a == nil || a.registry == nil {
		return nil, fmt.Errorf("review agent is not initialized")
	}
	tools := a.registry.List()
	infos := make([]*schema.ToolInfo, 0, len(tools))
	for _, item := range tools {
		info, err := item.Info(ctx)
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (a *ReviewAgent) InvokeTool(ctx context.Context, name string, argumentsInJSON string) (string, error) {
	if a == nil || a.registry == nil {
		return "", fmt.Errorf("review agent is not initialized")
	}
	item, ok := a.registry.Get(name)
	if !ok {
		return "", fmt.Errorf("review agent tool %q not found", name)
	}
	return item.InvokableRun(ctx, argumentsInJSON)
}
