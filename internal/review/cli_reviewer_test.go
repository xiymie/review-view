package review_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"review-view/internal/model"
	"review-view/internal/review"
)

func TestBuildCLIPromptForCommitRange(t *testing.T) {
	got := review.BuildCLIPrompt("基础提示", "abc", "def")
	if !strings.Contains(got, "git diff abc..def") {
		t.Fatalf("expected git diff instruction, got %q", got)
	}
}

func TestBuildCLIPromptForFirstReview(t *testing.T) {
	got := review.BuildCLIPrompt("基础提示", "", "def")
	if !strings.Contains(got, "git show def") {
		t.Fatalf("expected git show instruction, got %q", got)
	}
}

func TestCLIReviewerRunsClaudeWithMergedEnv(t *testing.T) {
	commander := &fakeCommander{
		stdout: `{"type":"system","subtype":"init"}
{"type":"result","result":"发现 1 个问题","is_error":false,"usage":{"input_tokens":120,"output_tokens":45,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`,
	}
	reviewer := review.NewCLIReviewer(commander)

	result, err := reviewer.Review(context.Background(), review.ReviewParams{
		Prompt: "请审查最近变更",
		WorkDir: "/tmp/repos/42",
		ModelConfig: &model.ModelConfig{
			Type:        model.ModelTypeClaudeCLI,
			ExtraConfig: `{"cli_path":"claude","env_vars":{"ANTHROPIC_API_KEY":"sk-123"},"max_turns":5}`,
		},
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if result.OutputTokens != 45 {
		t.Fatalf("unexpected tokens: %+v", result)
	}
	if commander.workDir != "/tmp/repos/42" {
		t.Fatalf("unexpected workdir %q", commander.workDir)
	}
	if !strings.Contains(strings.Join(commander.env, " "), "ANTHROPIC_API_KEY=sk-123") {
		t.Fatalf("expected merged env, got %+v", commander.env)
	}
}

type fakeCommander struct {
	workDir string
	env     []string
	stdout  string
	err     error
}

func (f *fakeCommander) Run(_ context.Context, workDir string, env []string, _ string, _ ...string) ([]byte, error) {
	f.workDir = workDir
	f.env = env
	if f.err != nil {
		return nil, f.err
	}
	return []byte(f.stdout), nil
}

// fakeStreamingCommander 同时实现 Commander 和 StreamCommander
type fakeStreamingCommander struct {
	fakeCommander
	onLineCalls [][]byte
}

func (f *fakeStreamingCommander) RunStream(_ context.Context, workDir string, env []string, onLine func(line []byte), _ string, _ ...string) ([]byte, error) {
	f.workDir = workDir
	f.env = env
	if f.err != nil {
		return nil, f.err
	}
	for _, line := range bytes.Split([]byte(f.stdout), []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if onLine != nil {
			cp := make([]byte, len(line))
			copy(cp, line)
			f.onLineCalls = append(f.onLineCalls, cp)
			onLine(cp)
		}
	}
	return []byte(f.stdout), nil
}

func TestCLIReviewerStreamingCallsOnChunk(t *testing.T) {
	output := `{"type":"system","subtype":"init"}
{"type":"assistant","message":{"content":[{"type":"text","text":"发现 "}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"1 个问题"}]}}
{"type":"result","result":"发现 1 个问题","is_error":false,"usage":{"input_tokens":100,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`

	commander := &fakeStreamingCommander{}
	commander.stdout = output

	var chunks []string
	reviewer := review.NewCLIReviewer(commander)

	result, err := reviewer.Review(context.Background(), review.ReviewParams{
		Prompt:  "审查代码",
		WorkDir: "/tmp/repo",
		ModelConfig: &model.ModelConfig{
			Type:        model.ModelTypeClaudeCLI,
			ExtraConfig: `{}`,
		},
		OnChunk: func(text string) {
			chunks = append(chunks, text)
		},
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if result.Content != "发现 1 个问题" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if len(chunks) != 2 || chunks[0] != "发现 " || chunks[1] != "1 个问题" {
		t.Fatalf("unexpected chunks: %v", chunks)
	}
}

func TestCLIReviewerStreamingFallsBackWithoutStreamCommander(t *testing.T) {
	// 普通 fakeCommander 不实现 StreamCommander，即使设置了 OnChunk 也应退化为普通 Run
	commander := &fakeCommander{
		stdout: `{"type":"result","result":"ok","is_error":false,"usage":{"input_tokens":10,"output_tokens":5}}`,
	}
	var called bool
	reviewer := review.NewCLIReviewer(commander)
	_, err := reviewer.Review(context.Background(), review.ReviewParams{
		Prompt:  "test",
		WorkDir: "/tmp",
		ModelConfig: &model.ModelConfig{
			Type:        model.ModelTypeClaudeCLI,
			ExtraConfig: `{}`,
		},
		OnChunk: func(text string) { called = true },
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if called {
		t.Fatal("OnChunk should not be called when StreamCommander is not implemented")
	}
}

func TestCLIReviewerStreamingToolUseOnLog(t *testing.T) {
	output := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"call_1","name":"Read","input":{"file_path":"/tmp/repo/main.go"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"call_1","content":"package main\n"}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"已审查"}]}}
{"type":"result","result":"已审查","is_error":false,"usage":{"input_tokens":100,"output_tokens":20}}`

	commander := &fakeStreamingCommander{}
	commander.stdout = output

	var chunks []string
	var logs []string
	reviewer := review.NewCLIReviewer(commander)

	_, err := reviewer.Review(context.Background(), review.ReviewParams{
		Prompt:  "审查代码",
		WorkDir: "/tmp/repo",
		ModelConfig: &model.ModelConfig{
			Type:        model.ModelTypeClaudeCLI,
			ExtraConfig: `{}`,
		},
		OnChunk: func(text string) { chunks = append(chunks, text) },
		OnLog:   func(level, msg string) { logs = append(logs, level+":"+msg) },
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}

	// OnChunk 只收到 text 内容
	if len(chunks) != 1 || chunks[0] != "已审查" {
		t.Fatalf("unexpected chunks: %v", chunks)
	}

	// OnLog 收到 tool_use 和 tool_result
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d: %v", len(logs), logs)
	}
	if !strings.Contains(logs[0], "$ Read") {
		t.Fatalf("expected tool_use log, got %q", logs[0])
	}
	if !strings.Contains(logs[1], "工具返回") {
		t.Fatalf("expected tool_result log, got %q", logs[1])
	}
}
