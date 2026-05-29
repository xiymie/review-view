package review

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"review-view/internal/model"
)

type Commander interface {
	Run(ctx context.Context, workDir string, env []string, name string, args ...string) ([]byte, error)
}

// StreamCommander 扩展 Commander，支持逐行流式回调。
// CLIReviewer.OnChunk 非空时会尝试将 Commander 断言为该接口，
// 成功则走流式路径，否则退化为普通 Run。
type StreamCommander interface {
	Commander
	RunStream(ctx context.Context, workDir string, env []string, onLine func(line []byte), name string, args ...string) ([]byte, error)
}

type execCommander struct{}

func (execCommander) Run(ctx context.Context, workDir string, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.Wait()
	return cmd.CombinedOutput()
}

// RunStream 启动命令并通过 stdout pipe 逐行回调，同时收集全部输出用于最终解析。
func (execCommander) RunStream(ctx context.Context, workDir string, env []string, onLine func(line []byte), name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir
	cmd.Env = env

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// 后台收集 stderr
	var stderr bytes.Buffer
	go io.Copy(&stderr, stderrPipe)

	// 逐行读取 stdout 并回调
	var stdout bytes.Buffer
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		raw := scanner.Bytes()
		stdout.Write(raw)
		stdout.WriteByte('\n')
		if onLine != nil {
			line := make([]byte, len(raw))
			copy(line, raw)
			onLine(line)
		}
	}

	if err := cmd.Wait(); err != nil {
		return stdout.Bytes(), fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return stdout.Bytes(), nil
}

type CLIReviewer struct {
	commander Commander
}

func NewCLIReviewer(commander Commander) *CLIReviewer {
	if commander == nil {
		commander = execCommander{}
	}
	return &CLIReviewer{commander: commander}
}

// cliResultMessage 对应 Claude CLI --output-format json 输出的 result 类型消息
type cliResultMessage struct {
	Type       string `json:"type"`
	Result     string `json:"result"`
	IsError    bool   `json:"is_error"`
	DurationMs int64  `json:"duration_ms"`
	Usage      struct {
		InputTokens              int64 `json:"input_tokens"`
		OutputTokens             int64 `json:"output_tokens"`
		CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	} `json:"usage"`
}

// cliAssistantMessage 对应 stream-json 中 type=assistant 的消息，
// 用于流式提取文本内容和 tool_use 调用信息
type cliAssistantMessage struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type  string         `json:"type"`
			Text  string         `json:"text"`
			Name  string         `json:"name,omitempty"`
			Input map[string]any `json:"input,omitempty"`
		} `json:"content"`
	} `json:"message"`
}

// cliUserMessage 对应 stream-json 中 type=user 的消息，
// 用于提取 tool_result 工具返回结果
type cliUserMessage struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type      string `json:"type"`
			Content   string `json:"content"`
		} `json:"content"`
	} `json:"message"`
}

// extractStreamText 从单行 stream-json 中提取 assistant 文本内容。
// 非 assistant 消息或无文本内容时返回空字符串。
func extractStreamText(line []byte) string {
	var msg cliAssistantMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return ""
	}
	if msg.Type != "assistant" {
		return ""
	}
	var texts []string
	for _, c := range msg.Message.Content {
		if c.Type == "text" && c.Text != "" {
			texts = append(texts, c.Text)
		}
	}
	return strings.Join(texts, "")
}

const maxToolResultLogLen = 500

// extractToolUseLog 从单行 stream-json 中提取 tool_use 信息，格式化为可读日志。
// 返回空字符串表示无 tool_use 内容。
func extractToolUseLog(line []byte) string {
	var msg cliAssistantMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return ""
	}
	if msg.Type != "assistant" {
		return ""
	}
	var parts []string
	for _, c := range msg.Message.Content {
		if c.Type != "tool_use" {
			continue
		}
		var args []string
		for k, v := range c.Input {
			args = append(args, fmt.Sprintf("%s=%v", k, v))
		}
		if len(args) > 0 {
			parts = append(parts, fmt.Sprintf("$ %s %s", c.Name, strings.Join(args, " ")))
		} else {
			parts = append(parts, fmt.Sprintf("$ %s", c.Name))
		}
	}
	return strings.Join(parts, "\n")
}

// extractToolResultLog 从单行 stream-json 中提取 tool_result 信息，格式化为可读日志。
// 返回空字符串表示无 tool_result 内容。
func extractToolResultLog(line []byte) string {
	var msg cliUserMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return ""
	}
	if msg.Type != "user" {
		return ""
	}
	var parts []string
	for _, c := range msg.Message.Content {
		if c.Type != "tool_result" {
			continue
		}
		output := c.Content
		if len(output) > maxToolResultLogLen {
			output = output[:maxToolResultLogLen] + "\n... (输出已截断)"
		}
		lines := strings.Count(output, "\n")
		if output != "" && !strings.HasSuffix(output, "\n") {
			lines++
		}
		parts = append(parts, fmt.Sprintf("工具返回 (%d 行)\n%s", lines, output))
	}
	return strings.Join(parts, "\n")
}

// parseCLIResult 解析 Claude CLI 的 JSON 输出，支持 JSON 数组和 NDJSON 两种格式，
// 提取 type 为 "result" 的最终消息
func parseCLIResult(output []byte) (*cliResultMessage, error) {
	trimmed := bytes.TrimSpace(output)

	// 尝试 JSON 数组格式
	if bytes.HasPrefix(trimmed, []byte("[")) {
		var messages []cliResultMessage
		if err := json.Unmarshal(trimmed, &messages); err == nil {
			for i := len(messages) - 1; i >= 0; i-- {
				if messages[i].Type == "result" {
					return &messages[i], nil
				}
			}
		}
		return nil, fmt.Errorf("cli output: missing result message in JSON array")
	}

	// NDJSON 格式：逐行解析，取最后一条 result 消息
	var last *cliResultMessage
	scanner := bufio.NewScanner(bytes.NewReader(trimmed))
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var msg cliResultMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if msg.Type == "result" {
			last = &msg
		}
	}
	if last != nil {
		return last, nil
	}
	return nil, fmt.Errorf("cli output: missing result message in NDJSON stream")
}

func (r *CLIReviewer) Review(ctx context.Context, params ReviewParams) (*ReviewResult, error) {
	if params.ModelConfig == nil {
		return nil, fmt.Errorf("model config is required")
	}

	var extra model.ClaudeCLIExtraConfig
	if err := params.ModelConfig.DecodeExtraConfig(&extra); err != nil {
		return nil, err
	}
	if extra.CLIPath == "" {
		extra.CLIPath = "claude"
	}

	args := []string{"-p", params.Prompt, "--output-format", "stream-json"}
	if extra.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", extra.MaxTurns))
	}

	env := append([]string{}, os.Environ()...)
	for key, value := range extra.EnvVars {
		env = append(env, key+"="+value)
	}

	started := time.Now()
	var output []byte
	var err error
	if sc, ok := r.commander.(StreamCommander); ok && params.OnChunk != nil {
		output, err = sc.RunStream(ctx, params.WorkDir, env, func(line []byte) {
			if text := extractStreamText(line); text != "" {
				params.OnChunk(text)
			}
			if msg := extractToolUseLog(line); msg != "" {
				onLog(params.OnLog, "info", msg)
			}
			if msg := extractToolResultLog(line); msg != "" {
				onLog(params.OnLog, "info", msg)
			}
		}, extra.CLIPath, args...)
	} else {
		output, err = r.commander.Run(ctx, params.WorkDir, env, extra.CLIPath, args...)
	}
	if err != nil {
		return nil, err
	}

	response, err := parseCLIResult(output)
	if err != nil {
		return nil, err
	}
	if response.IsError {
		return nil, fmt.Errorf("cli review failed: %s", response.Result)
	}

	return &ReviewResult{
		Content:             response.Result,
		InputTokens:         response.Usage.InputTokens,
		OutputTokens:        response.Usage.OutputTokens,
		CacheCreationTokens: response.Usage.CacheCreationInputTokens,
		CacheReadTokens:     response.Usage.CacheReadInputTokens,
		DurationMs:          time.Since(started).Milliseconds(),
	}, nil
}
