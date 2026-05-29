# any-llm-go 使用文档

any-llm-go 是 Mozilla AI 推出的 Go 语言 LLM 统一调用库，提供统一的接口访问 OpenAI、Anthropic、Gemini、Ollama、DeepSeek、Mistral 等 LLM 服务。

- 模块路径: `github.com/mozilla-ai/any-llm-go`
- 版本: v0.9.0
- 协议: Apache-2.0
- Go 版本要求: 1.25+

## 安装

```bash
go get github.com/mozilla-ai/any-llm-go@v0.9.0
```

## Provider 配置

所有 Provider 共享相同的构造函数签名:

```go
func New(opts ...config.Option) (*Provider, error)
```

### 导入路径

```go
import (
    anyllm "github.com/mozilla-ai/any-llm-go"
    "github.com/mozilla-ai/any-llm-go/providers/openai"
    "github.com/mozilla-ai/any-llm-go/providers/anthropic"
    "github.com/mozilla-ai/any-llm-go/providers/ollama"
    "github.com/mozilla-ai/any-llm-go/providers/deepseek"
    "github.com/mozilla-ai/any-llm-go/providers/gemini"
    "github.com/mozilla-ai/any-llm-go/providers/mistral"
)
```

### 通用配置选项

```go
anyllm.WithAPIKey("sk-xxx")           // 显式设置 API Key
anyllm.WithBaseURL("https://...")     // 自定义 Base URL
anyllm.WithHTTPClient(client)         // 自定义 HTTP 客户端（用于代理、TLS 等）
anyllm.WithTimeout(30 * time.Second)  // 请求超时
```

### 各 Provider 详情

#### OpenAI

```go
provider, err := openai.New()
// 默认读取 OPENAI_API_KEY 环境变量
// 默认 Base URL: https://api.openai.com/v1
```

#### Anthropic

```go
provider, err := anthropic.New()
// 默认读取 ANTHROPIC_API_KEY 环境变量
// 使用 anthropic-sdk-go 直接调用，非 OpenAI 兼容接口
```

**注意**: Anthropic 的 `ReasoningEffort` 会自动映射为 thinking budget，并调整 `MaxTokens`:
- Low = 1024 tokens
- Medium = 4096 tokens
- High = 16384 tokens

#### Ollama

```go
provider, err := ollama.New()
// 默认地址: http://localhost:11434
// 默认读取 OLLAMA_HOST 环境变量
// 无需 API Key
// 使用 ollama/api 直接调用
```

**注意**: Ollama 的 `ReasoningEffort` 仅支持布尔开关，不支持 budget 级别。

#### DeepSeek

```go
provider, err := deepseek.New()
// 默认读取 DEEPSEEK_API_KEY 环境变量
// 默认 Base URL: https://api.deepseek.com
// 内部使用 OpenAI 兼容接口
```

#### Gemini

```go
provider, err := gemini.New()
// 默认读取 GEMINI_API_KEY（备选 GOOGLE_API_KEY）
// 使用 google.golang.org/genai 直接调用
```

**注意**: Gemini 的 `ReasoningEffort` 映射为 thinking budget:
- Low = 1024 tokens
- Medium = 8192 tokens
- High = 24576 tokens

#### Mistral

```go
provider, err := mistral.New()
// 默认读取 MISTRAL_API_KEY 环境变量
// 默认 Base URL: https://api.mistral.ai/v1/
// 内部使用 OpenAI 兼容接口
```

**注意**: Mistral 不支持 `ReasoningEffort`，请求会被 `transformRequest()` 自动清除。

## Completion API

### Provider 接口

```go
type Provider interface {
    Name() string
    Completion(ctx context.Context, params CompletionParams) (*ChatCompletion, error)
    CompletionStream(ctx context.Context, params CompletionParams) (<-chan ChatCompletionChunk, <-chan error)
}
```

### 请求参数 CompletionParams

```go
type CompletionParams struct {
    Model             string            // 模型名称，如 "gpt-4o-mini"、"claude-sonnet-4-20250514"
    Messages          []Message         // 对话消息列表
    Temperature       *float64          // 采样温度
    TopP              *float64          // Top-P 采样
    MaxTokens         *int              // 最大输出 token 数
    Stop              []string          // 停止词列表
    Stream            bool              // 是否流式输出
    Tools             []Tool            // 函数工具定义
    ToolChoice        any               // 工具选择策略
    ParallelToolCalls *bool             // 是否并行调用工具
    ResponseFormat    *ResponseFormat   // 响应格式（如 JSON mode）
    ReasoningEffort   ReasoningEffort   // 推理力度
    Seed              *int              // 随机种子（可复现输出）
    User              string            // 用户标识（用于审计）
    Extra             map[string]any    // 额外参数（Provider 特有）
}
```

### 消息结构 Message

```go
type Message struct {
    Role       string      // "system"、"user"、"assistant"、"tool"
    Content    any         // string 或 []ContentPart（多模态内容）
    Name       string      // 参与者名称
    ToolCalls  []ToolCall  // 模型返回的工具调用
    ToolCallID string      // 工具调用的回复标识
    Reasoning  *Reasoning  // 推理内容（仅部分 Provider 支持）
}
```

常用角色常量:

```go
anyllm.RoleSystem    // "system"
anyllm.RoleUser      // "user"
anyllm.RoleAssistant // "assistant"
anyllm.RoleTool      // "tool"
```

### 响应结构 ChatCompletion

```go
type ChatCompletion struct {
    ID                string    // 响应 ID
    Object            string    // 对象类型
    Created           int64     // 创建时间戳
    Model             string    // 实际使用的模型名
    Choices           []Choice  // 响应选项列表
    Usage             *Usage    // Token 用量（可能为 nil）
    SystemFingerprint string    // 系统指纹
}

type Choice struct {
    Index        int       // 选项索引
    Message      Message   // 消息内容
    FinishReason string    // 结束原因："stop"、"length"、"tool_calls"
}
```

### 基本调用示例

```go
package main

import (
    "context"
    "fmt"
    "log"

    anyllm "github.com/mozilla-ai/any-llm-go"
    "github.com/mozilla-ai/any-llm-go/providers/openai"
)

func main() {
    ctx := context.Background()

    // 创建 Provider（默认读取 OPENAI_API_KEY 环境变量）
    provider, err := openai.New()
    if err != nil {
        log.Fatal(err)
    }

    // 发送请求
    response, err := provider.Completion(ctx, anyllm.CompletionParams{
        Model: "gpt-4o-mini",
        Messages: []anyllm.Message{
            {Role: anyllm.RoleSystem, Content: "你是一个有帮助的助手。"},
            {Role: anyllm.RoleUser, Content: "用三句话介绍 Go 语言。"},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // 输出回复内容
    fmt.Println(response.Choices[0].Message.Content)

    // 输出 Token 用量
    if response.Usage != nil {
        fmt.Printf("Tokens - 输入: %d, 输出: %d, 总计: %d\n",
            response.Usage.PromptTokens,
            response.Usage.CompletionTokens,
            response.Usage.TotalTokens)
    }
}
```

### 使用系统提示词

```go
response, err := provider.Completion(ctx, anyllm.CompletionParams{
    Model: "gpt-4o-mini",
    Messages: []anyllm.Message{
        {
            Role:    anyllm.RoleSystem,
            Content: "你是一个专业的 Go 代码审查助手。只给出简洁的建议。",
        },
        {
            Role:    anyllm.RoleUser,
            Content: "请审查这段代码: func add(a, b int) int { return a + b }",
        },
    },
    Temperature: ptrFloat64(0.3),  // 低温度，输出更确定
    MaxTokens:   ptrInt(500),       // 限制输出长度
})
```

辅助函数:

```go
func ptrFloat64(v float64) *float64 { return &v }
func ptrInt(v int) *int             { return &v }
```

## Usage（Token 用量）

### Usage 结构

```go
type Usage struct {
    PromptTokens     int  // 输入 token 数
    CompletionTokens int  // 输出 token 数
    TotalTokens      int  // 总 token 数
    ReasoningTokens  int  // 推理 token 数（仅 Gemini 填充）
}
```

### 访问方式

```go
if response.Usage != nil {
    fmt.Printf("输入: %d tokens\n", response.Usage.PromptTokens)
    fmt.Printf("输出: %d tokens\n", response.Usage.CompletionTokens)
    fmt.Printf("总计: %d tokens\n", response.Usage.TotalTokens)
}
```

**注意事项**:
- `Usage` 字段可能为 `nil`，使用前必须判空
- 没有 cache token 字段，Anthropic 的 cache 信息不通过此结构暴露
- 流式模式下 usage 出现在最后一个 chunk（`FinishReason` 被设置时）
- `ReasoningTokens` 仅 Gemini 填充，其他 Provider 为 0

## ReasoningEffort（推理力度）

用于控制模型的推理深度，适用于支持思考模式的模型。

### 可选值

```go
anyllm.ReasoningEffortAuto   // "auto"  - 自动决定
anyllm.ReasoningEffortHigh   // "high"  - 深度推理
anyllm.ReasoningEffortMedium // "medium"- 中等推理
anyllm.ReasoningEffortLow    // "low"   - 轻度推理
anyllm.ReasoningEffortNone   // "none"  - 不推理
```

### 使用示例

```go
response, err := provider.Completion(ctx, anyllm.CompletionParams{
    Model: "gpt-4o-mini",
    Messages: []anyllm.Message{
        {Role: anyllm.RoleUser, Content: "证明根号2是无理数。"},
    },
    ReasoningEffort: anyllm.ReasoningEffortHigh,
})

// 获取推理过程
if response.Choices[0].Message.Reasoning != nil {
    fmt.Println("推理过程:", response.Choices[0].Message.Reasoning.Content)
}
```

### 各 Provider 行为差异

| Provider | 行为 |
|----------|------|
| OpenAI / DeepSeek | 直接转发 `reasoning_effort` 字段给 API |
| Anthropic | 映射为 thinking budget: Low=1024, Medium=4096, High=16384，并自动调整 `MaxTokens` |
| Gemini | 映射为 thinking budget: Low=1024, Medium=8192, High=24576 |
| Ollama | 仅布尔开关，不支持 budget 级别 |
| Mistral | `transformRequest()` 会清除该字段，不支持推理 |

## 流式调用

使用 `CompletionStream` 方法实现流式输出，返回两个 channel: 数据 channel 和错误 channel。

### 基本流式调用

```go
package main

import (
    "context"
    "fmt"
    "log"

    anyllm "github.com/mozilla-ai/any-llm-go"
    "github.com/mozilla-ai/any-llm-go/providers/anthropic"
)

func main() {
    ctx := context.Background()

    provider, err := anthropic.New()
    if err != nil {
        log.Fatal(err)
    }

    // 流式请求
    stream, errCh := provider.CompletionStream(ctx, anyllm.CompletionParams{
        Model: "claude-sonnet-4-20250514",
        Messages: []anyllm.Message{
            {Role: anyllm.RoleUser, Content: "用 Go 写一个快速排序算法，附带中文注释。"},
        },
    })

    // 消费流式响应
    for {
        select {
        case chunk, ok := <-stream:
            if !ok {
                // 流结束
                fmt.Println("\n--- 完成 ---")
                return
            }
            // 输出每个 chunk 的内容
            for _, choice := range chunk.Choices {
                if choice.Delta.Content != nil {
                    content, _ := choice.Delta.Content.(string)
                    fmt.Print(content)
                }
                // 最后一个 chunk 包含 FinishReason 和 Usage
                if choice.FinishReason != "" {
                    fmt.Printf("\n结束原因: %s\n", choice.FinishReason)
                }
            }
        case err := <-errCh:
            if err != nil {
                log.Fatal("流式错误: ", err)
            }
            return
        }
    }
}
```

### 带 Token 统计的流式调用

```go
stream, errCh := provider.CompletionStream(ctx, anyllm.CompletionParams{
    Model: "gpt-4o-mini",
    Messages: []anyllm.Message{
        {Role: anyllm.RoleUser, Content: "解释 Go 的 goroutine 调度器。"},
    },
})

for chunk := range stream {
    for _, choice := range chunk.Choices {
        if choice.FinishReason != "" {
            // 流结束时 Usage 信息在最后一个 chunk 中
            if chunk.Usage != nil {
                fmt.Printf("\nToken 用量 - 输入: %d, 输出: %d\n",
                    chunk.Usage.PromptTokens,
                    chunk.Usage.CompletionTokens)
            }
        }
    }
}

// 检查错误 channel
for err := range errCh {
    if err != nil {
        log.Printf("流式错误: %v", err)
    }
}
```

## 多轮对话

```go
package main

import (
    "context"
    "fmt"
    "log"

    anyllm "github.com/mozilla-ai/any-llm-go"
    "github.com/mozilla-ai/any-llm-go/providers/openai"
)

func main() {
    ctx := context.Background()

    provider, err := openai.New()
    if err != nil {
        log.Fatal(err)
    }

    // 维护对话历史
    messages := []anyllm.Message{
        {Role: anyllm.RoleSystem, Content: "你是一个 Go 语言专家。"},
        {Role: anyllm.RoleUser, Content: "Go 里的 defer 是什么?"},
    }

    // 第一轮
    resp1, err := provider.Completion(ctx, anyllm.CompletionParams{
        Model:    "gpt-4o-mini",
        Messages: messages,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("助手:", resp1.Choices[0].Message.Content)

    // 将助手回复加入历史
    messages = append(messages, resp1.Choices[0].Message)

    // 第二轮 - 基于上下文继续对话
    messages = append(messages, anyllm.Message{
        Role:    anyllm.RoleUser,
        Content: "能给一个实际的使用场景吗?",
    })

    resp2, err := provider.Completion(ctx, anyllm.CompletionParams{
        Model:    "gpt-4o-mini",
        Messages: messages,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("助手:", resp2.Choices[0].Message.Content)
}
```

## 切换 Provider

any-llm-go 的核心优势是统一的 `Provider` 接口，切换 LLM 只需更换 Provider 实例，业务代码无需改动。

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    anyllm "github.com/mozilla-ai/any-llm-go"
    "github.com/mozilla-ai/any-llm-go/providers/anthropic"
    "github.com/mozilla-ai/any-llm-go/providers/deepseek"
    "github.com/mozilla-ai/any-llm-go/providers/gemini"
    "github.com/mozilla-ai/any-llm-go/providers/ollama"
    "github.com/mozilla-ai/any-llm-go/providers/openai"
)

// 创建 Provider 实例，根据环境变量决定使用哪个
func createProvider() (anyllm.Provider, error) {
    name := os.Getenv("LLM_PROVIDER")
    switch name {
    case "anthropic":
        return anthropic.New()
    case "gemini":
        return gemini.New()
    case "deepseek":
        return deepseek.New()
    case "ollama":
        return ollama.New()
    default:
        return openai.New()
    }
}

func main() {
    ctx := context.Background()

    provider, err := createProvider()
    if err != nil {
        log.Fatal(err)
    }

    // 统一调用方式，与具体 Provider 无关
    response, err := provider.Completion(ctx, anyllm.CompletionParams{
        Model: os.Getenv("LLM_MODEL"), // 如 "gpt-4o-mini"、"claude-sonnet-4-20250514"
        Messages: []anyllm.Message{
            {Role: anyllm.RoleUser, Content: "Hello!"},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("[%s] %s\n", provider.Name(), response.Choices[0].Message.Content)
}
```
