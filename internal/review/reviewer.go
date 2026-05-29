package review

import (
	"context"

	anyllm "github.com/mozilla-ai/any-llm-go"
	"review-view/internal/model"
)

type Reviewer interface {
	Review(ctx context.Context, params ReviewParams) (*ReviewResult, error)
}

type ReviewParams struct {
	Prompt         string
	WorkDir        string
	FromCommit     string
	ToCommit       string
	DiffContent    string
	CommitMessages string
	ModelConfig    *model.ModelConfig
	OnChunk        func(text string)                  // 可选：流式输出回调，每收到一段文本就调用
	OnLog          func(level string, message string) // 可选：agent loop 日志回调
	Replace        func(string) string                // 可选：发送前替换敏感词
	Restore        func(string) string                // 可选：收到响应后还原敏感词
}

type ReviewResult struct {
	Content             string
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	DurationMs          int64
}

type CompletionProvider interface {
	Completion(ctx context.Context, params anyllm.CompletionParams) (*anyllm.ChatCompletion, error)
}

// StreamingProvider 扩展 CompletionProvider，支持流式调用。
// ReviewAgent.OnChunk 非空时会尝试将 provider 断言为该接口，
// 成功则走流式路径，否则退化为普通 Completion。
type StreamingProvider interface {
	CompletionProvider
	CompletionStream(ctx context.Context, params anyllm.CompletionParams) (<-chan anyllm.ChatCompletionChunk, <-chan error)
}
