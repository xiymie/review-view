package review

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"review-view/internal/model"

	anyllm "github.com/mozilla-ai/any-llm-go"
)

// defaultMaxOutputTokens 默认输出 token 上限，覆盖 any-llm-go 的 4096 硬编码默认值
const defaultMaxOutputTokens = 16000

const maxToolOutputLen = 8 * 1024 // 8KB

// maxAgentRounds agent loop 最大轮数，防止 context 无限增长
const maxAgentRounds = 100

// ReviewAgent 实现带 agent loop + tool calling 的代码审查。
// 替代旧 LLMReviewer，让模型可以主动探索代码仓库。
type ReviewAgent struct {
	providers ProviderFactory
}

// NewReviewAgent 创建 ReviewAgent 实例。factory 为 nil 时使用默认 NewProvider。
func NewReviewAgent(factory ProviderFactory) *ReviewAgent {
	if factory == nil {
		factory = NewProvider
	}
	return &ReviewAgent{providers: factory}
}

// agentTools 定义模型可调用的 6 个 shell 工具
var agentTools = []anyllm.Tool{
	{
		Type: "function",
		Function: anyllm.Function{
			Name:        "git",
			Description: "在代码仓库中执行 git 命令",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"args": map[string]any{
						"type":        "string",
						"description": "git 命令参数，如 'log --oneline -10'",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "本次调用的意图说明",
					},
				},
				"required": []string{"args"},
			},
		},
	},
	{
		Type: "function",
		Function: anyllm.Function{
			Name:        "grep",
			Description: "搜索文件内容",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "搜索模式",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "搜索路径，默认为当前目录",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "本次调用的意图说明",
					},
				},
				"required": []string{"pattern"},
			},
		},
	},
	{
		Type: "function",
		Function: anyllm.Function{
			Name:        "ls",
			Description: "列出目录内容",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "目录路径，默认为当前目录",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "本次调用的意图说明",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: anyllm.Function{
			Name:        "cat",
			Description: "查看文件内容",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "文件路径",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "本次调用的意图说明",
					},
				},
				"required": []string{"path"},
			},
		},
	},
	{
		Type: "function",
		Function: anyllm.Function{
			Name:        "find",
			Description: "查找文件",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "文件名模式",
					},
					"type": map[string]any{
						"type":        "string",
						"description": "文件类型：f(文件) 或 d(目录)",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "本次调用的意图说明",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: anyllm.Function{
			Name:        "sed",
			Description: "文本替换或查看",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"expression": map[string]any{
						"type":        "string",
						"description": "sed 表达式",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "文件路径",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "本次调用的意图说明",
					},
				},
				"required": []string{"expression", "path"},
			},
		},
	},
	{
		Type: "function",
		Function: anyllm.Function{
			Name:        "glob",
			Description: "用通配符模式查找文件",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "glob 模式，如 '*.go', '*_test.go'",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "本次调用的意图说明",
					},
				},
				"required": []string{"pattern"},
			},
		},
	},
}

// toolGuidePrompt 追加到 messages 末尾的 tool 引导语
const toolGuidePrompt = "以上是本次变更的提交记录和文件列表。请使用 git diff 工具查看具体变更内容，完成代码审查。\n你可以使用 git、grep、cat 等工具探索代码仓库，理解变更上下文。"

// historyContextPrompt 当 .review/ 目录存在时追加的提示
const historyContextPrompt = "该项目之前有过代码审查记录，存放在 .review/ 目录下。你可以使用 ls .review/ 查看历史审查文件，使用 cat .review/xxx.md 查看具体内容。参考历史审查记录有助于提高审查的准确性和一致性。"

// BuildDiffSummary 将 commit 记录和变更文件列表构造为 LLM 可读的摘要
func BuildDiffSummary(commitMessages, diffContent string) string {
	var b strings.Builder
	b.WriteString("## 提交记录\n")
	b.WriteString(commitMessages)
	b.WriteString("\n\n## 变更文件\n")
	b.WriteString(diffContent)
	return b.String()
}

// Review 执行带 agent loop 的代码审查，实现 Reviewer 接口。
func (a *ReviewAgent) Review(ctx context.Context, params ReviewParams) (*ReviewResult, error) {
	if params.ModelConfig == nil {
		return nil, fmt.Errorf("model config is required")
	}

	provider, err := a.providers(params.ModelConfig)
	if err != nil {
		return nil, err
	}

	started := time.Now()

	messages := []anyllm.Message{
		{Role: anyllm.RoleSystem, Content: params.Prompt},
		{Role: anyllm.RoleUser, Content: BuildDiffSummary(params.CommitMessages, params.DiffContent)},
	}

	// 如果存在历史 review 记录目录，追加提示
	reviewDir := filepath.Join(params.WorkDir, ".review")
	if entries, err := os.ReadDir(reviewDir); err == nil && len(entries) > 0 {
		messages = append(messages, anyllm.Message{
			Role:    anyllm.RoleUser,
			Content: historyContextPrompt,
		})
	}

	messages = append(messages, anyllm.Message{
		Role:    anyllm.RoleUser,
		Content: toolGuidePrompt,
	})

	maxTokens := defaultMaxOutputTokens
	completion := anyllm.CompletionParams{
		Model:     params.ModelConfig.Model,
		Messages:  replaceInMessages(messages, params.Replace),
		Tools:     agentTools,
		MaxTokens: &maxTokens,
	}
	if params.ModelConfig.EnableThinking {
		completion.ReasoningEffort = anyllm.ReasoningEffortMedium
	}

	var totalInput, totalOutput int64

	// 包装 OnChunk，对流式 chunk 即时还原敏感词
	onChunk := params.OnChunk
	if onChunk != nil && params.Restore != nil {
		onChunk = func(text string) {
			params.OnChunk(params.Restore(text))
		}
	}

	for round := 0; round < maxAgentRounds; round++ {
		// 尝试流式路径；失败则降级为非流式（代理可能不完整支持 SSE）
		if onChunk != nil {
			if sp, ok := provider.(StreamingProvider); ok {
				text, finishReason, usage, toolCalls, err := a.streamRound(ctx, sp, completion, onChunk)
				if err == nil {
					if usage != nil {
						totalInput += int64(usage.PromptTokens)
						totalOutput += int64(usage.CompletionTokens)
					}
					if finishReason != anyllm.FinishReasonToolCalls {
						if params.Restore != nil {
							text = params.Restore(text)
						}
						return &ReviewResult{
							Content:      text,
							InputTokens:  totalInput,
							OutputTokens: totalOutput,
							DurationMs:   time.Since(started).Milliseconds(),
						}, nil
					}
					messages = append(messages, anyllm.Message{
						Role:      anyllm.RoleAssistant,
						Content:   text,
						ToolCalls: toolCalls,
					})
					messages = a.executeToolCalls(ctx, params, messages, toolCalls)
					completion.Messages = replaceInMessages(messages, params.Replace)
					continue
				}
				// 流式失败，降级到非流式重试本轮
				onLog(params.OnLog, "warn", fmt.Sprintf("streaming failed (round %d), retrying non-streaming: %s", round, err))
			}
		}

		// 非流式路径
		response, err := a.nonStreamRound(ctx, provider, completion)
		if err != nil {
			return nil, err
		}
		if response.Usage != nil {
			totalInput += int64(response.Usage.PromptTokens)
			totalOutput += int64(response.Usage.CompletionTokens)
		}

		choice := response.Choices[0]
		if choice.FinishReason != anyllm.FinishReasonToolCalls {
			content := fmt.Sprint(choice.Message.Content)
			if params.Restore != nil {
				content = params.Restore(content)
			}
			if onChunk != nil {
				onChunk(content)
			}
			return &ReviewResult{
				Content:      content,
				InputTokens:  totalInput,
				OutputTokens: totalOutput,
				DurationMs:   time.Since(started).Milliseconds(),
			}, nil
		}

		messages = append(messages, choice.Message)
		messages = a.executeToolCalls(ctx, params, messages, choice.Message.ToolCalls)
		completion.Messages = replaceInMessages(messages, params.Replace)
	}
	return nil, fmt.Errorf("agent loop exceeded %d rounds", maxAgentRounds)
}

// nonStreamRound 执行一轮非流式 Completion 调用（含 429 重试）
func (a *ReviewAgent) nonStreamRound(ctx context.Context, provider CompletionProvider, completion anyllm.CompletionParams) (*anyllm.ChatCompletion, error) {
	response, err := provider.Completion(ctx, completion)
	maxRetries := 3
	for retry := 0; retry < maxRetries && isRateLimitError(err); retry++ {
		backoff := time.Duration(1<<(retry+1)) * time.Second // 2s, 4s, 8s
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		response, err = provider.Completion(ctx, completion)
	}
	if err != nil {
		return nil, err
	}
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("empty llm response")
	}
	return response, nil
}

// streamRound 执行一轮流式 CompletionStream 调用。
// 返回累积文本、finishReason、usage 和 toolCalls。
func (a *ReviewAgent) streamRound(ctx context.Context, sp StreamingProvider, completion anyllm.CompletionParams, onChunk func(string)) (string, string, *anyllm.Usage, []anyllm.ToolCall, error) {
	stream, errCh := sp.CompletionStream(ctx, completion)

	var buf strings.Builder
	var usage *anyllm.Usage
	var finishReason string
	// Anthropic provider 每个 chunk 的 Arguments 是累积值（非 delta），直接覆盖而非拼接
	toolCallArgs := map[int]string{}
	toolCallMap := map[int]*anyllm.ToolCall{}

	for chunk := range stream {
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				buf.WriteString(choice.Delta.Content)
				onChunk(choice.Delta.Content)
			}
			for i, tc := range choice.Delta.ToolCalls {
				if _, ok := toolCallMap[i]; !ok {
					toolCallMap[i] = &anyllm.ToolCall{
						ID:       tc.ID,
						Type:     tc.Type,
						Function: anyllm.FunctionCall{Name: tc.Function.Name},
					}
				}
				// 覆盖而非追加：库已在内部累积，每次 chunk 发出的是完整字符串
				if tc.Function.Arguments != "" {
					toolCallArgs[i] = tc.Function.Arguments
				}
				if tc.ID != "" {
					toolCallMap[i].ID = tc.ID
				}
				if tc.Function.Name != "" {
					toolCallMap[i].Function.Name = tc.Function.Name
				}
			}
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
				if chunk.Usage != nil {
					usage = chunk.Usage
				}
			}
		}
	}
	for err := range errCh {
		if err != nil {
			return "", "", nil, nil, err
		}
	}

	// 组装 tool calls
	var toolCalls []anyllm.ToolCall
	if finishReason == anyllm.FinishReasonToolCalls {
		toolCalls = make([]anyllm.ToolCall, len(toolCallMap))
		for i := 0; i < len(toolCallMap); i++ {
			tc := toolCallMap[i]
			args := toolCallArgs[i]
			if args == "" || !json.Valid([]byte(args)) {
				args = "{}"
			}
			tc.Function.Arguments = args
			toolCalls[i] = *tc
		}
	}

	return buf.String(), finishReason, usage, toolCalls, nil
}

// executeToolCalls 逐个执行 tool call，将结果追加到 messages
func (a *ReviewAgent) executeToolCalls(ctx context.Context, params ReviewParams, messages []anyllm.Message, toolCalls []anyllm.ToolCall) []anyllm.Message {
	for _, tc := range toolCalls {
		name := tc.Function.Name
		args := tc.Function.Arguments

		var argsMap map[string]any
		_ = json.Unmarshal([]byte(args), &argsMap)

		output := executeToolCommand(ctx, params.WorkDir, name, args)

		// 截断输出
		if len(output) > maxToolOutputLen {
			output = output[:maxToolOutputLen] + "\n... (输出已截断)"
		}

		// 合并为单条日志，避免 cache flush 导致 display log 丢失
		onLog(params.OnLog, "info", formatToolDisplay(name, argsMap)+"\n"+formatToolResult(name, output))

		messages = append(messages, anyllm.Message{
			Role:       anyllm.RoleTool,
			Content:    output,
			ToolCallID: tc.ID,
		})
	}
	return messages
}

// executeToolCommand 根据 tool 名称和参数构造 shell 命令并执行
func executeToolCommand(ctx context.Context, workDir, name, argsJSON string) string {
	var argsMap map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &argsMap); err != nil {
		return fmt.Sprintf("error: invalid JSON arguments: %s\nraw: %s", err, argsJSON)
	}

	var cmd *exec.Cmd
	switch name {
	case "git":
		args := getStringArg(argsMap, "args")
		if args == "" {
			return "error: git tool requires 'args' field, e.g. {\"args\": \"log --oneline -10\"}"
		}
		cmd = exec.CommandContext(ctx, "git", strings.Fields(args)...)
	case "grep":
		pattern := getStringArg(argsMap, "pattern")
		path := getStringArg(argsMap, "path")
		if path == "" {
			path = "."
		}
		cmd = exec.CommandContext(ctx, "grep", "-rn", pattern, path)
	case "ls":
		path := getStringArg(argsMap, "path")
		if path == "" {
			path = "."
		}
		cmd = exec.CommandContext(ctx, "ls", "-la", path)
	case "cat":
		path := getStringArg(argsMap, "path")
		cmd = exec.CommandContext(ctx, "cat", path)
	case "find":
		findArgs := []string{"."}
		if n := getStringArg(argsMap, "name"); n != "" {
			findArgs = append(findArgs, "-name", n)
		}
		if t := getStringArg(argsMap, "type"); t != "" {
			findArgs = append(findArgs, "-type", t)
		}
		cmd = exec.CommandContext(ctx, "find", findArgs...)
	case "sed":
		expr := getStringArg(argsMap, "expression")
		path := getStringArg(argsMap, "path")
		cmd = exec.CommandContext(ctx, "sed", expr, path)
	case "glob":
		pattern := getStringArg(argsMap, "pattern")
		cmd = exec.CommandContext(ctx, "find", ".", "-name", pattern, "-type", "f")
	default:
		return fmt.Sprintf("unknown tool: %s", name)
	}

	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("error: %s\n%s", err, string(out))
	}
	return string(out)
}

// getStringArg 从解析后的 JSON 参数 map 中取字符串值
func getStringArg(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprint(v)
	}
	return s
}

// onLog 安全调用日志回调
func onLog(fn func(string, string), level, msg string) {
	if fn != nil {
		fn(level, msg)
	}
}

// isRateLimitError 判断是否为真正的 429 限流错误（排除携带 rate_limit 字样的验证错误）
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// 验证错误（如 tool_use.input 格式错误）不应重试
	if strings.Contains(msg, "input should be") ||
		strings.Contains(msg, "valid dictionary") ||
		strings.Contains(msg, "validation") {
		return false
	}
	return strings.Contains(msg, "429") || strings.Contains(msg, "rate_limit")
}

// formatToolDisplay 将工具调用格式化为可读的命令行展示。
// args 中的 "description" 字段为 AI 填写的调用意图描述，非空时展示在命令行之前。
func formatToolDisplay(name string, args map[string]any) string {
	var b strings.Builder
	if desc := getStringArg(args, "description"); desc != "" {
		b.WriteString(desc)
		b.WriteString("\n")
	}

	switch name {
	case "git":
		b.WriteString(fmt.Sprintf("$ git %s", getStringArg(args, "args")))
	case "grep":
		pattern := getStringArg(args, "pattern")
		path := getStringArg(args, "path")
		if path != "" {
			b.WriteString(fmt.Sprintf("$ grep -rn '%s' %s", pattern, path))
		} else {
			b.WriteString(fmt.Sprintf("$ grep -rn '%s'", pattern))
		}
	case "ls":
		path := getStringArg(args, "path")
		if path != "" {
			b.WriteString(fmt.Sprintf("$ ls -la %s", path))
		} else {
			b.WriteString("$ ls -la")
		}
	case "cat":
		b.WriteString(fmt.Sprintf("$ cat %s", getStringArg(args, "path")))
	case "find":
		parts := []string{"$ find ."}
		if n := getStringArg(args, "name"); n != "" {
			parts = append(parts, fmt.Sprintf("-name '%s'", n))
		}
		if t := getStringArg(args, "type"); t != "" {
			parts = append(parts, fmt.Sprintf("-type %s", t))
		}
		b.WriteString(strings.Join(parts, " "))
	case "sed":
		b.WriteString(fmt.Sprintf("$ sed '%s' %s", getStringArg(args, "expression"), getStringArg(args, "path")))
	case "glob":
		b.WriteString(fmt.Sprintf("$ glob '%s'", getStringArg(args, "pattern")))
	default:
		b.WriteString(fmt.Sprintf("$ %s", name))
	}
	return b.String()
}

// replaceInMessages 复制 messages 并对每条消息的 Content 应用 replace 函数
func replaceInMessages(messages []anyllm.Message, replace func(string) string) []anyllm.Message {
	if replace == nil {
		return messages
	}
	result := make([]anyllm.Message, len(messages))
	for i, m := range messages {
		m.Content = replace(fmt.Sprint(m.Content))
		result[i] = m
	}
	return result
}

// formatToolResult 格式化工具返回结果，包含行数统计和截断预览
func formatToolResult(name string, output string) string {
	lines := strings.Count(output, "\n")
	if output != "" && !strings.HasSuffix(output, "\n") {
		lines++
	}

	// 错误输出直接展示
	if strings.HasPrefix(output, "error:") {
		return fmt.Sprintf("执行失败 (%s)", strings.TrimSpace(output))
	}

	// 截断预览
	preview := output
	maxPreview := 200
	if len(preview) > maxPreview {
		preview = preview[:maxPreview] + "..."
	}

	return fmt.Sprintf("返回结果 (%d 行)\n%s", lines, preview)
}

// GetReviewer 根据模型配置返回对应的 Reviewer 实现
func GetReviewer(config *model.ModelConfig) Reviewer {
	if config != nil && config.Type == model.ModelTypeClaudeCLI {
		return NewCLIReviewer(nil)
	}
	return NewReviewAgent(nil)
}

// TestCompletionParams 构造一条最小化请求用于测试模型连接
func TestCompletionParams(config *model.ModelConfig) anyllm.CompletionParams {
	return anyllm.CompletionParams{
		Model: config.Model,
		Messages: []anyllm.Message{
			{Role: anyllm.RoleUser, Content: "Hi, reply with \"ok\"."},
		},
	}
}
