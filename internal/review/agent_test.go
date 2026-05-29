package review_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	anyllm "github.com/mozilla-ai/any-llm-go"
	"review-view/internal/model"
	"review-view/internal/review"
)

// fakeProvider 模拟 CompletionProvider
type fakeProvider struct {
	last     anyllm.CompletionParams
	response *anyllm.ChatCompletion
	err      error
	calls    int
	// callErrors 控制每次调用的错误，index 对应第 N 次调用（从 0 开始）
	callErrors map[int]error
}

func (f *fakeProvider) Completion(_ context.Context, params anyllm.CompletionParams) (*anyllm.ChatCompletion, error) {
	f.last = params
	f.calls++
	if f.callErrors != nil {
		if err, ok := f.callErrors[f.calls-1]; ok {
			return nil, err
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

// fakeStreamingProvider 同时实现 CompletionProvider 和 StreamingProvider
type fakeStreamingProvider struct {
	fakeProvider
	chunks []anyllm.ChatCompletionChunk
}

func (f *fakeStreamingProvider) CompletionStream(_ context.Context, params anyllm.CompletionParams) (<-chan anyllm.ChatCompletionChunk, <-chan error) {
	f.last = params
	f.calls++
	stream := make(chan anyllm.ChatCompletionChunk, len(f.chunks))
	errCh := make(chan error, 1)
	for _, chunk := range f.chunks {
		stream <- chunk
	}
	close(stream)
	close(errCh)
	return stream, errCh
}

// fakeMultiTurnProvider 支持多次返回不同响应的 provider
type fakeMultiTurnProvider struct {
	fakeProvider
	responses []*anyllm.ChatCompletion
	index     int
}

func (f *fakeMultiTurnProvider) Completion(_ context.Context, params anyllm.CompletionParams) (*anyllm.ChatCompletion, error) {
	f.last = params
	f.calls++
	if f.index >= len(f.responses) {
		return nil, fmt.Errorf("no more responses")
	}
	resp := f.responses[f.index]
	f.index++
	return resp, nil
}

// fakeMultiTurnStreamingProvider 支持多轮流式响应
type fakeMultiTurnStreamingProvider struct {
	fakeProvider
	rounds [][]anyllm.ChatCompletionChunk
	index  int
}

func (f *fakeMultiTurnStreamingProvider) CompletionStream(_ context.Context, params anyllm.CompletionParams) (<-chan anyllm.ChatCompletionChunk, <-chan error) {
	f.last = params
	f.calls++
	if f.index >= len(f.rounds) {
		ch := make(chan anyllm.ChatCompletionChunk)
		errCh := make(chan error, 1)
		close(ch)
		errCh <- fmt.Errorf("no more rounds")
		close(errCh)
		return ch, errCh
	}
	round := f.rounds[f.index]
	f.index++
	stream := make(chan anyllm.ChatCompletionChunk, len(round))
	errCh := make(chan error, 1)
	for _, chunk := range round {
		stream <- chunk
	}
	close(stream)
	close(errCh)
	return stream, errCh
}

func makeToolCallJSON(name, args, id string) string {
	return fmt.Sprintf(`{"name":"%s","arguments":%q,"id":"%s"}`, name, args, id)
}

// --- 测试用例 ---

func TestReviewAgentDirectText(t *testing.T) {
	// 无 tool call，直接返回文本（兼容旧行为）
	fake := &fakeProvider{
		response: &anyllm.ChatCompletion{
			Choices: []anyllm.Choice{{Message: anyllm.Message{Content: "LGTM"}, FinishReason: "stop"}},
			Usage:   &anyllm.Usage{PromptTokens: 100, CompletionTokens: 20},
		},
	}
	agent := review.NewReviewAgent(func(*model.ModelConfig) (review.CompletionProvider, error) {
		return fake, nil
	})

	result, err := agent.Review(context.Background(), review.ReviewParams{
		Prompt:      "你是代码审查助手",
		DiffContent: "diff --git a/main.go b/main.go",
		ModelConfig: &model.ModelConfig{Type: model.ModelTypeOpenAI, Model: "gpt-4o-mini"},
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if result.Content != "LGTM" {
		t.Fatalf("unexpected content %q", result.Content)
	}
	if result.InputTokens != 100 || result.OutputTokens != 20 {
		t.Fatalf("unexpected usage: %+v", result)
	}
	// 确认传入了 tools
	if len(fake.last.Tools) == 0 {
		t.Fatal("expected tools to be passed")
	}
}

func TestReviewAgentEnablesMediumReasoning(t *testing.T) {
	fake := &fakeProvider{
		response: &anyllm.ChatCompletion{Choices: []anyllm.Choice{{Message: anyllm.Message{Content: "ok"}, FinishReason: "stop"}}},
	}
	agent := review.NewReviewAgent(func(*model.ModelConfig) (review.CompletionProvider, error) {
		return fake, nil
	})

	_, err := agent.Review(context.Background(), review.ReviewParams{
		Prompt:      "prompt",
		DiffContent: "diff",
		ModelConfig: &model.ModelConfig{Type: model.ModelTypeGemini, Model: "gemini-2.5-pro", EnableThinking: true},
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if fake.last.ReasoningEffort != anyllm.ReasoningEffortMedium {
		t.Fatalf("expected medium reasoning, got %q", fake.last.ReasoningEffort)
	}
}

func TestReviewAgentSingleToolCall(t *testing.T) {
	// 单轮 tool call + 最终回答
	mtp := &fakeMultiTurnProvider{
		responses: []*anyllm.ChatCompletion{
			{
				Choices: []anyllm.Choice{{
					Message: anyllm.Message{
						Content: "",
						ToolCalls: []anyllm.ToolCall{
							{
								ID:   "call_1",
								Type: "function",
								Function: anyllm.FunctionCall{
									Name:      "ls",
									Arguments: `{}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				}},
			},
			{
				Choices: []anyllm.Choice{{
					Message:      anyllm.Message{Content: "review done"},
					FinishReason: "stop",
				}},
				Usage: &anyllm.Usage{PromptTokens: 200, CompletionTokens: 30},
			},
		},
	}
	agent := review.NewReviewAgent(func(*model.ModelConfig) (review.CompletionProvider, error) {
		return mtp, nil
	})

	result, err := agent.Review(context.Background(), review.ReviewParams{
		Prompt:      "review",
		DiffContent: "diff",
		WorkDir:     ".", // 使用当前目录
		ModelConfig: &model.ModelConfig{Type: model.ModelTypeOpenAI, Model: "gpt-4o-mini"},
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if result.Content != "review done" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if mtp.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", mtp.calls)
	}
	// 验证 messages 中包含 tool 消息
	msgs := mtp.last.Messages
	hasToolMsg := false
	for _, m := range msgs {
		if m.Role == anyllm.RoleTool {
			hasToolMsg = true
			break
		}
	}
	if !hasToolMsg {
		t.Fatal("expected tool message in conversation")
	}
}

func TestReviewAgentMultiTurnToolCall(t *testing.T) {
	// 多轮 tool call + 最终回答
	mtp := &fakeMultiTurnProvider{
		responses: []*anyllm.ChatCompletion{
			{
				Choices: []anyllm.Choice{{
					Message: anyllm.Message{
						ToolCalls: []anyllm.ToolCall{
							{ID: "c1", Type: "function", Function: anyllm.FunctionCall{Name: "cat", Arguments: `{"path":"main.go"}`}},
						},
					},
					FinishReason: "tool_calls",
				}},
			},
			{
				Choices: []anyllm.Choice{{
					Message: anyllm.Message{
						ToolCalls: []anyllm.ToolCall{
							{ID: "c2", Type: "function", Function: anyllm.FunctionCall{Name: "git", Arguments: `{"args":"log --oneline -5"}`}},
						},
					},
					FinishReason: "tool_calls",
				}},
			},
			{
				Choices: []anyllm.Choice{{
					Message:      anyllm.Message{Content: "final review"},
					FinishReason: "stop",
				}},
			},
		},
	}
	agent := review.NewReviewAgent(func(*model.ModelConfig) (review.CompletionProvider, error) {
		return mtp, nil
	})

	result, err := agent.Review(context.Background(), review.ReviewParams{
		Prompt:      "review",
		DiffContent: "diff",
		WorkDir:     ".",
		ModelConfig: &model.ModelConfig{Type: model.ModelTypeOpenAI, Model: "gpt-4o-mini"},
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if result.Content != "final review" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if mtp.calls != 3 {
		t.Fatalf("expected 3 calls, got %d", mtp.calls)
	}
}

func TestReviewAgentRetriesOn429(t *testing.T) {
	fake := &fakeProvider{
		response: &anyllm.ChatCompletion{
			Choices: []anyllm.Choice{{Message: anyllm.Message{Content: "retried ok"}, FinishReason: "stop"}},
		},
		callErrors: map[int]error{
			0: errors.New("429 rate limit exceeded"),
		},
	}
	agent := review.NewReviewAgent(func(*model.ModelConfig) (review.CompletionProvider, error) {
		return fake, nil
	})

	result, err := agent.Review(context.Background(), review.ReviewParams{
		Prompt:      "prompt",
		DiffContent: "diff",
		ModelConfig: &model.ModelConfig{Type: model.ModelTypeOpenAI, Model: "gpt-4o-mini"},
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if result.Content != "retried ok" {
		t.Fatalf("expected retried content, got %q", result.Content)
	}
	if fake.calls != 2 {
		t.Fatalf("expected 2 calls (1 fail + 1 retry), got %d", fake.calls)
	}
}

func TestReviewAgentNoRetryOnNon429(t *testing.T) {
	fake := &fakeProvider{
		err: errors.New("500 internal server error"),
	}
	agent := review.NewReviewAgent(func(*model.ModelConfig) (review.CompletionProvider, error) {
		return fake, nil
	})

	_, err := agent.Review(context.Background(), review.ReviewParams{
		Prompt:      "prompt",
		DiffContent: "diff",
		ModelConfig: &model.ModelConfig{Type: model.ModelTypeOpenAI, Model: "gpt-4o-mini"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if fake.calls != 1 {
		t.Fatalf("expected 1 call (no retry), got %d", fake.calls)
	}
}

func TestReviewAgentStreamingDirectText(t *testing.T) {
	fp := &fakeStreamingProvider{
		chunks: []anyllm.ChatCompletionChunk{
			{Choices: []anyllm.ChunkChoice{{Delta: anyllm.ChunkDelta{Content: "发现 "}}}},
			{Choices: []anyllm.ChunkChoice{{Delta: anyllm.ChunkDelta{Content: "1 个问题"}}}},
			{Choices: []anyllm.ChunkChoice{
				{Delta: anyllm.ChunkDelta{Content: ""}, FinishReason: "stop"},
			}, Usage: &anyllm.Usage{PromptTokens: 50, CompletionTokens: 10}},
		},
	}
	agent := review.NewReviewAgent(func(*model.ModelConfig) (review.CompletionProvider, error) {
		return fp, nil
	})

	var chunks []string
	result, err := agent.Review(context.Background(), review.ReviewParams{
		Prompt:      "审查代码",
		DiffContent: "diff content",
		ModelConfig: &model.ModelConfig{Type: model.ModelTypeOpenAI, Model: "gpt-4o-mini"},
		OnChunk:     func(text string) { chunks = append(chunks, text) },
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if result.Content != "发现 1 个问题" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if result.InputTokens != 50 || result.OutputTokens != 10 {
		t.Fatalf("unexpected usage: %+v", result)
	}
	if len(chunks) != 2 || chunks[0] != "发现 " || chunks[1] != "1 个问题" {
		t.Fatalf("unexpected chunks: %v", chunks)
	}
}

func TestReviewAgentStreamingWithToolCall(t *testing.T) {
	// 流式 + tool call
	mtsp := &fakeMultiTurnStreamingProvider{
		rounds: [][]anyllm.ChatCompletionChunk{
			// 第一轮：tool call
			{
				{
					Choices: []anyllm.ChunkChoice{{
						Delta: anyllm.ChunkDelta{
							ToolCalls: []anyllm.ToolCall{
								{ID: "tc_1", Type: "function", Function: anyllm.FunctionCall{Name: "ls", Arguments: `{}`}},
							},
						},
					}},
				},
				{
					Choices: []anyllm.ChunkChoice{{
						Delta:        anyllm.ChunkDelta{},
						FinishReason: "tool_calls",
					}},
				},
			},
			// 第二轮：最终文本
			{
				{Choices: []anyllm.ChunkChoice{{Delta: anyllm.ChunkDelta{Content: "review "}}}},
				{Choices: []anyllm.ChunkChoice{{Delta: anyllm.ChunkDelta{Content: "complete"}}}},
				{
					Choices: []anyllm.ChunkChoice{{Delta: anyllm.ChunkDelta{}, FinishReason: "stop"}},
					Usage:   &anyllm.Usage{PromptTokens: 80, CompletionTokens: 15},
				},
			},
		},
	}
	agent := review.NewReviewAgent(func(*model.ModelConfig) (review.CompletionProvider, error) {
		return mtsp, nil
	})

	var chunks []string
	var logs []string
	result, err := agent.Review(context.Background(), review.ReviewParams{
		Prompt:      "review",
		DiffContent: "diff",
		WorkDir:     ".",
		ModelConfig: &model.ModelConfig{Type: model.ModelTypeOpenAI, Model: "gpt-4o-mini"},
		OnChunk:     func(text string) { chunks = append(chunks, text) },
		OnLog:       func(level, msg string) { logs = append(logs, msg) },
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if result.Content != "review complete" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d: %v", len(chunks), chunks)
	}
	if len(logs) == 0 {
		t.Fatal("expected tool call logs")
	}
	// 验证日志包含工具调用信息
	hasToolLog := false
	for _, l := range logs {
		if strings.Contains(l, "$ ls -la") {
			hasToolLog = true
			break
		}
	}
	if !hasToolLog {
		t.Fatalf("expected tool call log, got: %v", logs)
	}
}

func TestReviewAgentStreamingFallsBackWithoutStreamingProvider(t *testing.T) {
	fp := &fakeProvider{
		response: &anyllm.ChatCompletion{
			Choices: []anyllm.Choice{{Message: anyllm.Message{Content: "ok"}, FinishReason: "stop"}},
		},
	}
	agent := review.NewReviewAgent(func(*model.ModelConfig) (review.CompletionProvider, error) {
		return fp, nil
	})

	var called bool
	_, err := agent.Review(context.Background(), review.ReviewParams{
		Prompt:      "test",
		DiffContent: "diff",
		ModelConfig: &model.ModelConfig{Type: model.ModelTypeOpenAI, Model: "gpt-4o-mini"},
		OnChunk:     func(text string) { called = true },
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if called {
		t.Fatal("OnChunk should not be called when StreamingProvider is not implemented")
	}
}

func TestReviewAgentOnLogCallback(t *testing.T) {
	mtp := &fakeMultiTurnProvider{
		responses: []*anyllm.ChatCompletion{
			{
				Choices: []anyllm.Choice{{
					Message: anyllm.Message{
						ToolCalls: []anyllm.ToolCall{
							{ID: "c1", Type: "function", Function: anyllm.FunctionCall{Name: "git", Arguments: `{"args":"status"}`}},
						},
					},
					FinishReason: "tool_calls",
				}},
			},
			{
				Choices: []anyllm.Choice{{
					Message:      anyllm.Message{Content: "done"},
					FinishReason: "stop",
				}},
			},
		},
	}
	agent := review.NewReviewAgent(func(*model.ModelConfig) (review.CompletionProvider, error) {
		return mtp, nil
	})

	var logs []string
	_, err := agent.Review(context.Background(), review.ReviewParams{
		Prompt:      "review",
		DiffContent: "diff",
		WorkDir:     ".",
		ModelConfig: &model.ModelConfig{Type: model.ModelTypeOpenAI, Model: "gpt-4o-mini"},
		OnLog:       func(level, msg string) { logs = append(logs, level+":"+msg) },
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected logs from tool execution")
	}
}

func TestReviewAgentGuidePromptInMessages(t *testing.T) {
	fake := &fakeProvider{
		response: &anyllm.ChatCompletion{
			Choices: []anyllm.Choice{{Message: anyllm.Message{Content: "ok"}, FinishReason: "stop"}},
		},
	}
	agent := review.NewReviewAgent(func(*model.ModelConfig) (review.CompletionProvider, error) {
		return fake, nil
	})

	_, err := agent.Review(context.Background(), review.ReviewParams{
		Prompt:         "review",
		DiffContent:    "M\tmain.go\n",
		CommitMessages: "abc1234 fix\n",
		ModelConfig:    &model.ModelConfig{Type: model.ModelTypeOpenAI, Model: "gpt-4o-mini"},
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}

	msgs := fake.last.Messages
	found := false
	for _, m := range msgs {
		if m.Role == anyllm.RoleUser && strings.Contains(fmt.Sprint(m.Content), "git diff") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected tool guide prompt mentioning 'git diff' in messages")
	}
}

func TestBuildDiffSummary(t *testing.T) {
	got := review.BuildDiffSummary("abc1234 fix login\n", "M\tmain.go\nA\thelper.go\n")
	want := "## 提交记录\nabc1234 fix login\n\n\n## 变更文件\nM\tmain.go\nA\thelper.go\n"
	if got != want {
		t.Fatalf("unexpected output:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

