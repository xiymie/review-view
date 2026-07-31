# Review View

自托管的 AI 代码审查 & 定时巡检平台，单二进制部署，支持多种 LLM 提供商。

## 特性

- 🚀 **权限分离**：每位开发者拥有独立项目空间，自行管理仓库凭据，仅能操作自己创建的资源
- 🤖 **多 LLM 支持**：OpenAI / Anthropic / Ollama / DeepSeek / Gemini / Mistral / Claude CLI，以及兼容 OpenAI API 的服务
- 🧠 **Eino Workflow**：Review 与 Scan 已抽象为可观测工作流节点，支持 legacy / Eino 渐进式切换
- 🧩 **Review Skill Registry**：内置 13 个 Grok 风格审查 Skill，支持全局启用、项目绑定、本次审核选择，并预留 Agent / Registry / Tool / Policy / Workflow / Context / Memory 结构字段
- 📦 **项目管理**：支持模型全局 Prompt、项目 Prompt、项目 CLAUDE.md、Skill Prompt 分层叠加
- 🔄 **灵活触发**：项目页手动触发、指定 commit 范围触发、Webhook 自动触发
- 🔍 **定时巡检**：定期扫描仓库所有分支，基于 checkpoint 分析新增 commit 和 diff 风险，报告上传 NAS
- 📊 **仪表盘**：扫描项目总数 / 巡检项目总数 / 用户数量 / 敏感词数量 / 高风险分支，支持定时自动刷新
- 📣 **结果推送**：扫描结果可配置推送邮件或企业微信机器人
- 🔐 **敏感词管理**：支持检测类敏感词和替换类敏感词；提交 AI 前自动替换，返回后还原，并在巡检日志中记录命中的词、文件和行号
- 📈 **详细日志**：任务执行日志、工作流节点日志、工具调用日志、Token 消耗统计（含缓存命中）
- 🎨 **现代化管理界面**：全宽自适应后台、卡片化配置页、任务/巡检详情双栏日志阅读体验

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

# 构建前端
cd frontend && npm install && npm run build && cd ..

# 编译后端（前端资源自动内嵌）
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

# Review workflow 模式：legacy（默认）或 eino
export REVIEW_WORKFLOW_MODE=legacy
```

### 使用流程

1. 访问 `http://localhost:18083`
2. 在「模型配置」中添加 LLM 配置（API Key 或 CLI 模式）
3. 在「仓库凭据」中添加私有仓库认证（可选）
4. 在「Review Skill」中启用可用 Skill，并在项目中选择默认 Skill
5. 在「项目」中创建项目，关联 Git 仓库、模型、项目 Prompt 和 Skill
6. 手动触发审核时可再次选择本次使用的 Skill，也可配置 Webhook 自动触发
7. （可选）在「巡检配置」中设置定时扫描仓库分支

## 架构

```
review-view/
├── cmd/server/          # 主程序入口
├── internal/
│   ├── app/            # 应用初始化、路由
│   ├── config/         # 配置管理
│   ├── handler/        # HTTP 处理器
│   ├── model/          # 数据模型
│   ├── notify/         # 通知服务（邮件、企业微信）
│   ├── review/         # Git 操作、LLM 调用
│   ├── service/        # 业务逻辑（含巡检调度器）
│   └── store/          # 数据库访问（GORM + SQLite）
├── frontend/           # Vue 3 + Element Plus 前端
├── web/                # 前端构建产物（embed）
└── docs/               # 文档
```

## 核心功能

### 仪表盘

首页仪表盘展示 5 个核心数据卡片：

- **扫描项目总数** — 已接入代码审查的项目数
- **巡检项目总数** — 已配置定时巡检的仓库数
- **用户数量** — 平台账户总数
- **敏感词数量** — 已配置的敏感词条数
- **高风险分支** — 本周巡检发现的高风险分支数

右上角提供**定时自动刷新**开关（5 秒 / 10 秒 / 30 秒），无需手动点刷新即可实时感知任务状态。

「最近动态」时间轴中，点击任意条目直接跳转到对应审核结果或巡检详情。

### 代码审查（Code Review）

- 支持多项目、多分支、指定 commit 范围审核
- 支持 `REVIEW_WORKFLOW_MODE=legacy|eino`，默认保持 legacy，Eino 模式使用节点化 Review Workflow
- Prompt 分层顺序：本次任务选择的 Skill Prompt → 模型全局 Prompt → 项目自定义 Prompt → 项目 `CLAUDE.md` → commit / diff 上下文
- Review Skill 是“全局可用池 + 项目默认选择 + 本次任务选择”，未选择则不注入
- 支持运行中流式输出、右侧 sticky 执行日志、Diff / Commit / Review 结果分栏查看
- 溢出策略：排队等待 / 拒绝新任务
- 任务超时配置（项目级 > 全局）

### Review Skill Registry

- 内置 13 个 Grok 风格审查 Skill，Prompt 保持英文原文，展示名称和简介本地化
- Skill 默认关闭，启用后进入全局候选池；项目可绑定默认 Skill，触发审核时可临时增减本次 Skill
- 当前执行逻辑保持简单可靠：多个 Skill 合并为一个 Skill Prompt，一次模型调用输出一份结果
- 数据结构已预留官方式扩展字段，便于后续实现 Skill Selector / Multi-Agent / 图模式执行：
  - `agent_xml`
  - `skill_registry_xml`
  - `tool_registry_xml`
  - `policy_md`
  - `workflow_md`
  - `context_schema_json`
  - `memory_schema_json`
  - `metadata_json`

### 定时巡检（Scan）

- 定时扫描仓库所有活跃分支，按增量 checkpoint 判断是否有新 commit
- 单次 LLM 调用始终结合 commit message + diff + CLAUDE.md，避免只看提交信息造成误判
- Prompt 分层顺序：项目 Skill Prompt → 全局巡检 Prompt → 项目自定义 Prompt → 巡检配置 Prompt → `CLAUDE.md` → commit 信息 → diff 代码
- 高风险判定要求结合调用链、上游校验、配置和项目约定复核，减少单函数误报
- 分支风险评级（高 / 中 / 低 / 无风险），并将风险数量汇总到 ScanJob，避免仪表盘 N+1 查询
- Workflow 日志记录 Skill、Prompt、CLAUDE.md、diff、AI 调用、风险等级、敏感词替换命中情况
- 报告自动上传到 NAS（WebDAV），按保留天数自动清理旧报告
- 支持巡检完成后推送通知

### 模型配置

**API 模式**：支持 7 种平台

- OpenAI（GPT-4o 等）
- Anthropic（Claude）
- Ollama（本地部署）
- DeepSeek
- Google Gemini
- Mistral AI
- 其他兼容 OpenAI API 的服务

**CLI 模式**：通过 `claude -p` 无交互执行，适合已安装 Claude Code 的服务器

### 任务调度

- FIFO 队列，按创建时间执行
- 全局并发控制（可配置，默认 3）
- 超时保护（默认 30 分钟）
- 支持取消运行中任务
- 支持重试/再次扫描失败任务

### 触发方式

**手动触发**：项目详情页点击按钮

**Webhook 触发**：

```bash
# 审查最新提交
curl -X POST http://localhost:18083/webhook/{projectId}

# 审查指定提交范围
curl -X POST http://localhost:18083/webhook/{projectId} \
  -H "Content-Type: application/json" \
  -d '{"commit": "abc123"}'
```

### 智能跳过

- 无新提交时自动跳过（`fromCommit == toCommit`）
- 相同 commit 范围已审查过时跳过
- Diff 为空时标记失败

### 审核结果

任务详情页采用左右双栏布局：

- **左侧结果区**：Review 结果、Diff、Commit 记录撑满页面可用高度，支持独立滚动
- **右侧执行日志**：sticky 日志面板，运行中自动滚动到底，历史任务可手动滚动查看
- **日志内容**：工作流节点、仓库同步、Skill 注入、Prompt 构建、CLAUDE.md、AI 调用、敏感词命中、Token 消耗统计

## 开发

### 本地运行

```bash
# 后端
go run ./cmd/server

# 前端（开发模式，代理到后端 18083）
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
cd frontend && npm run build

# 构建后端（自动内嵌前端资源）
cd .. && go build -o review-view ./cmd/server
```

## 配置示例

### 系统设置

在「设置」页面配置：

- **最大并发任务数**：同时运行的最大任务数（默认 3）
- **全局溢出策略**：`queue`（排队）或 `reject`（拒绝）
- **任务超时时间**：全局默认超时（分钟，默认 30）
- **仓库根目录**：Git 仓库克隆位置（默认 `./repos`）
- **巡检默认提示词**：所有巡检仓库的默认 LLM 提示词
- **SMTP 配置**：邮件推送所需的邮件服务器参数

### 模型配置示例

**OpenAI API**

```
名称: GPT-4o
类型: API
平台: openai
API Key: sk-...
Base URL: https://api.openai.com/v1
模型: gpt-4o
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
模型配置: GPT-4o
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
EXPOSE 18083
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
        # SSE 流式输出需要关闭缓冲
        proxy_buffering off;
        proxy_read_timeout 600s;
    }
}
```

### macOS LaunchAgent

项目内置的 `deploy.sh` 可用于部署到 macOS LaunchAgent 环境。注意：

- `go-sqlite3` 需要 `CGO_ENABLED=1`
- macOS Ventura+ 推荐使用 `launchctl bootout/bootstrap`，不要再使用旧的 `unload/load`
- plist 路径必须使用远程用户的绝对路径，例如 `/Users/user/Library/LaunchAgents/com.app.review_app.plist`，避免本机 `$HOME` 提前展开导致 `Bootstrap failed: 5`

## 常见问题

**Q: 支持 GitHub/GitLab 集成吗？**

A: 通过 Webhook 触发，在 Git 平台的项目设置中配置 Webhook URL 即可。

**Q: 可以审查本地仓库吗？**

A: 可以，仓库 URL 填写本地路径（如 `file:///path/to/repo`）。

**Q: 如何保护敏感信息？**

A: 建议内网部署，或通过反向代理添加认证（Basic Auth / OAuth）。API Key 等凭据加密存储在本地 SQLite 中，不向外传输。

**Q: 支持哪些 Git 托管平台？**

A: 支持所有标准 HTTPS Git 协议，包括 GitHub、GitLab、Gitea、Bitbucket、Gogs 等。SSH 协议不支持凭证注入，请使用 HTTPS URL。

**Q: LLM 调用失败怎么办？**

A: 查看任务详情页的「日志」Tab，会显示详细错误信息。常见原因：API Key 错误、网络问题、模型名称不对。

**Q: 定时巡检和 Code Review 有什么区别？**

A: Code Review 针对具体 commit 范围进行一次性深度审查；定时巡检按计划（如每天 09:00）扫描仓库所有分支的新增 commit，聚焦于高风险变更检测，结果上传 NAS 存档。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License

## 致谢

- [any-llm-go](https://github.com/your-username/any-llm-go) — 统一 LLM API 调用
- [Element Plus](https://element-plus.org/) — Vue 3 组件库
- [GORM](https://gorm.io/) — Go ORM 框架
