package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// graphTool adapts an Eino Runnable into an InvokableTool.
//
// It is intentionally small: arguments are JSON-decoded into the graph input
// type, the graph is invoked, and the graph output is JSON-encoded for the tool
// result. This gives review-view a stable bridge from deterministic Eino graphs
// to future Agent tool calling.
type graphTool[I, O any] struct {
	info   *schema.ToolInfo
	runner compose.Runnable[I, O]
}

func newGraphTool[I, O any](info *schema.ToolInfo, runner compose.Runnable[I, O]) tool.InvokableTool {
	return &graphTool[I, O]{info: info, runner: runner}
}

func (t *graphTool[I, O]) Info(context.Context) (*schema.ToolInfo, error) {
	if t.info == nil {
		return nil, fmt.Errorf("graph tool info is nil")
	}
	return t.info, nil
}

func (t *graphTool[I, O]) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	if t.runner == nil {
		return "", fmt.Errorf("graph tool runner is nil")
	}
	var input I
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("decode graph tool arguments: %w", err)
	}
	output, err := t.runner.Invoke(ctx, input)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("encode graph tool output: %w", err)
	}
	return string(payload), nil
}
