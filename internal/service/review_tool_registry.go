package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const ReviewExecuteTaskToolName = "review_execute_task"

type ReviewToolRegistry struct {
	tools  []tool.InvokableTool
	byName map[string]tool.InvokableTool
}

type reviewTaskToolInput struct {
	TaskID int64 `json:"task_id"`
}

type reviewTaskToolOutput struct {
	TaskID int64  `json:"task_id"`
	Status string `json:"status"`
}

func NewStaticReviewToolRegistry(reviewWorkflow ReviewWorkflow) (*ReviewToolRegistry, error) {
	if reviewWorkflow == nil {
		return nil, fmt.Errorf("review workflow is nil")
	}
	reviewTaskTool, err := newReviewExecuteTaskTool(reviewWorkflow)
	if err != nil {
		return nil, err
	}
	registry := &ReviewToolRegistry{
		tools:  []tool.InvokableTool{reviewTaskTool},
		byName: map[string]tool.InvokableTool{},
	}
	for _, item := range registry.tools {
		info, err := item.Info(context.Background())
		if err != nil {
			return nil, err
		}
		registry.byName[info.Name] = item
	}
	return registry, nil
}

func (r *ReviewToolRegistry) List() []tool.InvokableTool {
	out := make([]tool.InvokableTool, len(r.tools))
	copy(out, r.tools)
	return out
}

func (r *ReviewToolRegistry) Get(name string) (tool.InvokableTool, bool) {
	item, ok := r.byName[name]
	return item, ok
}

func newReviewExecuteTaskTool(reviewWorkflow ReviewWorkflow) (tool.InvokableTool, error) {
	chain := compose.NewChain[reviewTaskToolInput, reviewTaskToolOutput]().
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, input reviewTaskToolInput) (reviewTaskToolOutput, error) {
			if input.TaskID == 0 {
				return reviewTaskToolOutput{}, fmt.Errorf("task_id is required")
			}
			if err := reviewWorkflow.Run(ctx, ReviewInput{TaskID: input.TaskID}); err != nil {
				return reviewTaskToolOutput{TaskID: input.TaskID, Status: "failed"}, err
			}
			return reviewTaskToolOutput{TaskID: input.TaskID, Status: "completed"}, nil
		}), compose.WithNodeName("run_review_workflow"))
	runner, err := chain.Compile(context.Background(), compose.WithGraphName("review_execute_task_tool_graph_v1"))
	if err != nil {
		return nil, err
	}
	info := &schema.ToolInfo{
		Name: ReviewExecuteTaskToolName,
		Desc: "Run the review workflow for an existing review task by task_id. This tool has side effects: it executes the review and updates task status/logs/results.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"task_id": {
				Type:     schema.Integer,
				Desc:     "Existing review task ID to execute.",
				Required: true,
			},
		}),
	}
	return newGraphTool[reviewTaskToolInput, reviewTaskToolOutput](info, runner), nil
}
