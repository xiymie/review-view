# Review View

自托管的 AI 代码审查平台，单二进制部署，支持多种 LLM 提供商。

## 特性

- 🚀 **权限分离**：每一个开发拥有自己的项目空间，配置自己的代码仓库token，仅能管理自己的代码审核
- 🤖 **多 LLM 支持**：OpenAI / Anthropic / Ollama / DeepSeek / Gemini / Mistral / Claude CLI
- 📦 **项目管理**：支持全局和自定义 Prompt
- 🔄 **灵活触发**：手动触发 + Webhook 自动触发
- 📊 **结果推送**：扫描结构可配置推送邮箱或企业微信
- 🔐 **敏感词管理**：支持敏感词替换，绕过模型网关的封禁。同时支持敏感词检测，避免出现真实敏感信息泄露
- 📈 **详细日志**：任务执行日志、Token 消耗统计
- 🎨 **现代 UI**：Vue 3 + Element Plus

## 快速开始

### 前置要求

- Go 1.21+（编译需要）
- Git（运行时需要）
- 可访问的 LLM API 或本地 Ollama

### 安装

**方式 1：从源码编译**

```bash
git clone https://github.com/xiymie/review-view.git
cd review-view

# 编译（前端资源会自动打包）
go build -o review-view ./cmd/server

# 运行
./review-view
```

**方式 2：下载预编译二进制**

从 [Releases](https://github.com/xiymie/review-view/releases) 下载对应平台的二进制文件。

### 配置

通过环境变量配置：

```bash
# 监听地址（默认 :18083）
export APP_ADDR=:18083

# 数据库路径（默认 SQLite）
export DATABASE_DSN="file:review-view.db?_foreign_keys=on"
```

### 使用流程

1. 访问 `http://localhost:5173`
2. 在"模型配置"中添加 LLM 配置（API Key 或 CLI 模式）
3. 在"仓库凭据"中添加私有仓库认证（可选）
4. 在"项目"中创建项目，关联 Git 仓库和模型
5. 手动触发审核或配置 Webhook 自动触发

## 架构

```
review-view/
├── cmd/server/          # 主程序入口
├── internal/
│   ├── app/            # 应用初始化、路由
│   ├── config/         # 配置管理
│   ├── handler/        # HTTP 处理器
│   ├── model/          # 数据模型
│   ├── notify/         # 通知服务（邮件等）
│   ├── review/         # Git 操作、LLM 调用
│   ├── service/        # 业务逻辑
│   └── store/          # 数据库访问
├── frontend/           # Vue 3 前端
├── web/                # 前端构建产物（embed）
└── docs/               # 文档
```

## 核心功能

### 项目管理

- 支持多项目、多分支
- 自定义 Review Prompt（可覆盖模型配置）
- 溢出策略：排队等待 / 拒绝新任务
- 任务超时配置（项目级 > 全局）

### 模型配置

**API 模式**：支持 6 种平台

- OpenAI（GPT-4、GPT-3.5）
- Anthropic（Claude）
- Ollama（本地部署）
- DeepSeek
- Google Gemini
- Mistral AI

**CLI 模式**：通过 `claude -p` 无交互执行

### 任务调度

- FIFO 队列，按创建时间执行
- 全局并发控制（可配置）
- 超时保护（默认 30 分钟）
- 支持取消运行中任务
- 支持重试失败任务

### 触发方式

**手动触发**：项目详情页点击按钮

**Webhook 触发**：

```bash
# 审查最新提交
curl -X POST http://localhost:5173/webhook/{projectId}

# 审查指定提交
curl -X POST http://localhost:5173/webhook/{projectId} \
  -H "Content-Type: application/json" \
  -d '{"commit": "abc123"}'
```

### 智能跳过

- 无新提交时自动跳过（`fromCommit == toCommit`）
- 相同 commit 范围已审查过时跳过
- Diff 为空时标记失败

### 审核结果

任务详情页提供三个 Tab：

- **Review**：LLM 审查结果（Markdown 渲染）
- **Diff**：代码变更内容
- **日志**：执行日志（仓库同步、LLM 调用、错误等）

显示 Token 消耗统计：

- Input Tokens
- Output Tokens
- Cache Write Tokens
- Cache Read Tokens

## 开发

### 本地运行

```bash
# 后端
go run ./cmd/server

# 前端（开发模式）
cd frontend
npm install
npm run dev
```

### 测试

```bash
go test ./...
```

### 构建

```bash
# 构建前端
cd frontend
npm run build

# 构建后端（会自动打包前端资源）
cd ..
go build -o review-view ./cmd/server
```

## 配置示例

### 系统设置

在"系统设置"页面配置：

- **最大并发任务数**：同时运行的最大任务数（默认 3）
- **全局溢出策略**：`queue`（排队）或 `reject`（拒绝）
- **任务超时时间**：全局默认超时（分钟，默认 30）
- **仓库根目录**：Git 仓库克隆位置（默认 `./repos`）

### 模型配置示例

**OpenAI API**

```
名称: GPT-4
类型: API
平台: openai
API Key: sk-...
Base URL: https://api.openai.com/v1
模型: gpt-4-turbo-preview
```

**Claude CLI**

```
名称: Claude Code
类型: CLI
命令: claude
```

### 项目配置示例

```
项目名称: my-backend
仓库 URL: https://github.com/user/repo.git
分支: main
模型配置: GPT-4
仓库凭据: （可选，私有仓库需要）
溢出策略: queue
任务超时: 45 分钟
自定义 Prompt: 请重点关注安全问题和性能优化
```

## 部署建议

### Docker（推荐）

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN apk add --no-cache git nodejs npm && \
    cd frontend && npm install && npm run build && cd .. && \
    go build -o review-view ./cmd/server

FROM alpine:latest
RUN apk add --no-cache git ca-certificates
COPY --from=builder /app/review-view /usr/local/bin/
EXPOSE 8080
CMD ["review-view"]
```

### Systemd

```ini
[Unit]
Description=Review View
After=network.target

[Service]
Type=simple
User=review
WorkingDirectory=/opt/review-view
Environment="APP_ADDR=:18083"
Environment="DATABASE_DSN=file:/var/lib/review-view/review-view.db?_foreign_keys=on"
ExecStart=/opt/review-view/review-view
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### 反向代理（Nginx）

```nginx
server {
    listen 80;
    server_name review.example.com;

    location / {
        proxy_pass http://127.0.0.1:18083;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 常见问题

**Q: 支持 GitHub/GitLab 集成吗？**

A: 目前通过 Webhook 触发，需要在 Git 平台配置 Webhook URL。未来计划支持 OAuth 和 PR 评论。

**Q: 可以审查本地仓库吗？**

A: 可以，仓库 URL 填写本地路径（如 `file:///path/to/repo`）。

**Q: 如何保护敏感信息？**

A: 建议内网部署，或通过反向代理添加认证（Basic Auth / OAuth）。

**Q: 支持哪些 Git 托管平台？**

A: 支持所有标准 Git 协议（HTTPS / SSH），包括 GitHub、GitLab、Gitea、Bitbucket 等。

**Q: LLM 调用失败怎么办？**

A: 查看任务详情页的"日志" Tab，会显示详细错误信息。常见原因：API Key 错误、网络问题、模型不存在。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License

## 致谢

- [any-llm-go](https://github.com/your-username/any-llm-go) - 统一 LLM API 调用
- [Element Plus](https://element-plus.org/) - Vue 3 组件库
- [GORM](https://gorm.io/) - Go ORM 框架
