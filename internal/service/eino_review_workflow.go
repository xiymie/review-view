package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"review-view/internal/model"
)

type reviewCallbackTaskIDKey struct{}

type einoReviewState struct {
	Input   ReviewInput
	Context *ReviewContext
}

var einoReviewNodeNames = map[string]struct{}{
	"prepare":        {},
	"sync_repo":      {},
	"checkout":       {},
	"load_skills":    {},
	"build_prompt":   {},
	"run":            {},
	"sensitive_scan": {},
	"persist_result": {},
	"finish":         {},
}

// einoReviewWorkflow is the placeholder for the future Eino-backed review flow.
//
// Version 1 wires a real Eino Chain with coarse stages:
// prepare -> sync_repo -> checkout -> build_prompt -> run -> sensitive_scan -> persist_result -> finish.
// The run stage delegates model execution to the legacy workflow internals,
// sensitive_scan handles report enrichment, and persist_result writes the final
// task/project state before finish only cleans up runtime cancellation state.
type einoReviewWorkflow struct {
	fallback ReviewWorkflow
	legacy   *legacyReviewWorkflow
	cache    *TaskCache
	skills   *ReviewSkillService
	runner   compose.Runnable[ReviewInput, ReviewInput]
	initErr  error
}

func newEinoReviewWorkflow(fallback ReviewWorkflow, opts ...any) ReviewWorkflow {
	workflow := &einoReviewWorkflow{fallback: fallback}
	if legacy, ok := fallback.(*legacyReviewWorkflow); ok {
		workflow.legacy = legacy
	}
	for _, opt := range opts {
		switch v := opt.(type) {
		case *TaskCache:
			workflow.cache = v
		case *ReviewSkillService:
			workflow.skills = v
		}
	}
	workflow.runner, workflow.initErr = workflow.buildReviewChain(context.Background())
	return workflow
}

func (w *einoReviewWorkflow) Run(ctx context.Context, input ReviewInput) error {
	if w.initErr != nil {
		return w.initErr
	}
	_, err := w.runner.Invoke(ctx, input, compose.WithCallbacks(w.taskLogCallback()))
	if w.cache != nil && input.TaskID != 0 {
		w.cache.Flush(input.TaskID)
	}
	return err
}

func (w *einoReviewWorkflow) buildReviewChain(ctx context.Context) (compose.Runnable[ReviewInput, ReviewInput], error) {
	chain := compose.NewChain[ReviewInput, ReviewInput]().
		AppendLambda(compose.InvokableLambda(w.prepare), compose.WithNodeName("prepare")).
		AppendLambda(compose.InvokableLambda(w.syncRepo), compose.WithNodeName("sync_repo")).
		AppendLambda(compose.InvokableLambda(w.checkout), compose.WithNodeName("checkout")).
		AppendLambda(compose.InvokableLambda(w.loadSkills), compose.WithNodeName("load_skills")).
		AppendLambda(compose.InvokableLambda(w.buildPrompt), compose.WithNodeName("build_prompt")).
		AppendLambda(compose.InvokableLambda(w.run), compose.WithNodeName("run")).
		AppendLambda(compose.InvokableLambda(w.sensitiveScan), compose.WithNodeName("sensitive_scan")).
		AppendLambda(compose.InvokableLambda(w.persistResult), compose.WithNodeName("persist_result")).
		AppendLambda(compose.InvokableLambda(w.finish), compose.WithNodeName("finish"))

	return chain.Compile(ctx, compose.WithGraphName("review_chain_v1"))
}

func (w *einoReviewWorkflow) taskLogCallback() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if !isEinoReviewNode(info) {
				return ctx
			}
			taskID := taskIDFromCallbackInput(input)
			if taskID == 0 {
				return ctx
			}
			w.appendNodeLog(taskID, model.TaskLogLevelInfo, fmt.Sprintf("Eino 节点 %s 开始", info.Name))
			return context.WithValue(ctx, reviewCallbackTaskIDKey{}, taskID)
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, _ callbacks.CallbackOutput) context.Context {
			if !isEinoReviewNode(info) {
				return ctx
			}
			if taskID, ok := ctx.Value(reviewCallbackTaskIDKey{}).(int64); ok {
				w.appendNodeLog(taskID, model.TaskLogLevelInfo, fmt.Sprintf("Eino 节点 %s 完成", info.Name))
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if !isEinoReviewNode(info) {
				return ctx
			}
			if taskID, ok := ctx.Value(reviewCallbackTaskIDKey{}).(int64); ok {
				w.appendNodeLog(taskID, model.TaskLogLevelError, fmt.Sprintf("Eino 节点 %s 失败: %v", info.Name, err))
			}
			return ctx
		}).
		Build()
}

func isEinoReviewNode(info *callbacks.RunInfo) bool {
	if info == nil {
		return false
	}
	_, ok := einoReviewNodeNames[info.Name]
	return ok
}

func taskIDFromCallbackInput(input callbacks.CallbackInput) int64 {
	switch v := input.(type) {
	case ReviewInput:
		return v.TaskID
	case *einoReviewState:
		return v.Input.TaskID
	case einoReviewState:
		return v.Input.TaskID
	case *ReviewContext:
		if v != nil && v.Task != nil {
			return v.Task.ID
		}
	case ReviewContext:
		if v.Task != nil {
			return v.Task.ID
		}
	}
	return 0
}

func (w *einoReviewWorkflow) appendNodeLog(taskID int64, level model.TaskLogLevel, message string) {
	if w.cache == nil || taskID == 0 {
		return
	}
	w.cache.AppendLog(taskID, level, message)
}

func (w *einoReviewWorkflow) prepare(ctx context.Context, input ReviewInput) (*einoReviewState, error) {
	state := &einoReviewState{Input: input}
	if w.legacy == nil {
		return state, nil
	}
	reviewCtx, err := w.legacy.loadReviewContext(input.TaskID)
	if err != nil {
		return state, err
	}
	if err := w.legacy.beginReviewContext(ctx, reviewCtx); err != nil {
		w.legacy.cleanupReviewContext(reviewCtx)
		return state, err
	}
	state.Context = reviewCtx
	return state, nil
}

func (w *einoReviewWorkflow) syncRepo(_ context.Context, state *einoReviewState) (*einoReviewState, error) {
	if state == nil {
		return state, fmt.Errorf("eino review workflow state is nil")
	}
	if w.legacy != nil && state.Context != nil {
		if err := w.legacy.syncRepo(state.Context); err != nil {
			w.legacy.cleanupReviewContext(state.Context)
			return state, err
		}
	}
	return state, nil
}

func (w *einoReviewWorkflow) checkout(_ context.Context, state *einoReviewState) (*einoReviewState, error) {
	if state == nil {
		return state, fmt.Errorf("eino review workflow state is nil")
	}
	if w.legacy != nil && state.Context != nil {
		if err := w.legacy.checkoutRepo(state.Context); err != nil {
			w.legacy.cleanupReviewContext(state.Context)
			return state, err
		}
	}
	return state, nil
}

func (w *einoReviewWorkflow) loadSkills(_ context.Context, state *einoReviewState) (*einoReviewState, error) {
	if state == nil {
		return state, fmt.Errorf("eino review workflow state is nil")
	}
	if state.Context != nil {
		if w.skills != nil {
			skillIDs := decodeReviewSkillIDs(state.Context.Task.ReviewSkillIDs)
			skillPrompt, err := w.skills.BuildPromptForSkillIDs(skillIDs)
			if err != nil {
				return state, err
			}
			state.Context.SkillPrompt = skillPrompt
		}
	}
	return state, nil
}

func (w *einoReviewWorkflow) buildPrompt(_ context.Context, state *einoReviewState) (*einoReviewState, error) {
	if state == nil {
		return state, fmt.Errorf("eino review workflow state is nil")
	}
	if w.legacy != nil && state.Context != nil {
		state.Context.Prompt = w.legacy.buildReviewPrompt(state.Context)
	}
	return state, nil
}

func (w *einoReviewWorkflow) run(ctx context.Context, state *einoReviewState) (*einoReviewState, error) {
	if state == nil {
		return state, fmt.Errorf("eino review workflow state is nil")
	}
	if w.legacy != nil && state.Context != nil {
		if err := w.legacy.executeModel(state.Context); err != nil {
			w.legacy.cleanupReviewContext(state.Context)
			return state, err
		}
		return state, nil
	}
	if w.fallback == nil {
		return state, fmt.Errorf("eino review workflow fallback is nil")
	}
	if err := w.fallback.Run(ctx, state.Input); err != nil {
		return state, err
	}
	return state, nil
}

func (w *einoReviewWorkflow) sensitiveScan(_ context.Context, state *einoReviewState) (*einoReviewState, error) {
	if state == nil {
		return state, fmt.Errorf("eino review workflow state is nil")
	}
	if w.legacy != nil && state.Context != nil && state.Context.ReviewResult != nil {
		if err := w.legacy.scanSensitiveWords(state.Context); err != nil {
			w.legacy.cleanupReviewContext(state.Context)
			return state, err
		}
	}
	return state, nil
}

func (w *einoReviewWorkflow) persistResult(_ context.Context, state *einoReviewState) (*einoReviewState, error) {
	if state == nil {
		return state, fmt.Errorf("eino review workflow state is nil")
	}
	if w.legacy != nil && state.Context != nil && state.Context.ReviewResult != nil {
		if err := w.legacy.persistReviewResult(state.Context); err != nil {
			w.legacy.cleanupReviewContext(state.Context)
			return state, err
		}
	}
	return state, nil
}

func (w *einoReviewWorkflow) finish(_ context.Context, state *einoReviewState) (ReviewInput, error) {
	if state == nil {
		return ReviewInput{}, fmt.Errorf("eino review workflow state is nil")
	}
	if w.legacy != nil && state.Context != nil {
		w.legacy.cleanupReviewContext(state.Context)
	}
	return state.Input, nil
}
