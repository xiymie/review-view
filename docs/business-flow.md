# 业务流程文档

## 架构分层

```
cmd/server/main.go           入口，加载配置 + 启动服务
internal/
  config/                    环境变量配置
  model/                     GORM 领域模型
  store/                     Store 接口定义
  store/gorm/                GORM + SQLite 实现
  service/                   业务逻辑
  review/                    Review 核心（git diff + LLM/CLI 调用）
  handler/                   Gin HTTP handler
  render/                    Markdown -> HTML（goldmark）
web/
  static/                    CSS/JS
  templates/                 Go HTML 模板
```

依赖注入在 `internal/app/router.go` 中手动构造：DB -> Stores -> Services -> Handlers -> Router。

## 任务触发流程

```
触发请求（手动 / Webhook）
    |
    v
获取项目信息，EnsureRepo（clone 或 fetch）
    |
    v
解析 toCommit（手动取 HEAD，Webhook 可指定）
    |
    v
fromCommit = project.LastReviewedCommit
    |
    v
fromCommit == toCommit? ── 是 ──> 跳过，不创建任务
    |
    v
已存在相同范围的 completed 任务? ── 是 ──> 跳过
    |
    v
创建 Task（status=pending）
    |
    v
项目有 running 任务 + strategy=reject? ── 是 ──> status=rejected
    |
    v
返回 task（等待调度）
```

## 任务执行流程

```
调度器轮询 pending 任务（间隔可配，默认 5s）
    |
    v
全局并发已满? ── 是 ──> 保持 pending，下次再试
    |
    v
获取 semaphore，启动 goroutine 执行
    |
    v
status=running，注册 context cancel
    |
    v
设置超时（project.task_timeout > global.task_timeout > 30min）
    |
    v
EnsureRepo（clone 或 fetch）
    |
    v
┌─────────────────── modelConfig.type ───────────────────┐
│                                                        │
│  claude_cli                          其他（API 模式）    │
│  ┌─────────────────┐                ┌────────────────┐  │
│  │ BuildCLIPrompt  │                │ BuildDiff      │  │
│  │ 注入 commit 信息 │                │ git show/diff  │  │
│  │                 │                │                │  │
│  │ （CLI 自行读    │                │ diff 为空?     │  │
│  │  取 diff 和     │                │ ── 是 ──> fail │  │
│  │  文件上下文）    │                │                │  │
│  └────────┬────────┘                └───────┬────────┘  │
│           │                                 │           │
└───────────┴─────────────────────────────────┴───────────┘
            |
            v
调用 Reviewer.Review()
            |
    ┌───────┴────────┐
    |                |
  成功             失败/超时/取消
    |                |
    v                v
存储 result       记录 error_message
+ token 数据      status=failed
status=completed  /cancelled
    |
    v
更新 project.LastReviewedCommit = toCommit
    |
    v
释放 semaphore
```

## 任务日志与流式结果

执行过程中通过 `TaskCache` 内存缓冲区管理日志和流式结果，每 5 秒批量写入 DB，任务结束时立即刷盘：

- **日志**：flush 后从内存删除（已消费），新日志重新追加
- **结果**：LLM 流式输出通过 `OnChunk` 回调实时累积到内存，flush 只写 DB 不删除（SSE 需要完整快照），任务完成后调用 `RemoveResult` 清理
|------|-------|---------|
| 任务开始执行 | info | 任务开始执行 |
| 仓库同步完成 | info | 代码仓库同步完成 |
| 仓库同步失败 | error | 代码仓库同步失败: {err} |
| Diff 获取完成 | info | Diff 获取完成 ({n} 字符) |
| Diff 为空 | warn | commit 范围无 diff 内容 |
| Diff 获取失败 | error | Diff 获取失败: {err} |
| 开始 LLM 调用 | info | 开始调用 {type} |
| Review 完成 | info | Review 完成，耗时 {ms}ms |
| Review 失败 | error | Review 调用失败: {err} |
| 任务超时 | error | 任务超时 ({n} 分钟) |
| 任务取消 | info | 任务被取消 |

运行中的任务日志和流式结果可通过 SSE 端点 `GET /api/tasks/:id/stream` 实时获取。

## 项目删除

```
用户点击"删除项目"
    |
    v
检查是否有 running 任务? ── 是 ──> 拒绝，返回 400
    |
    v
事务内：
  1. 删除 task_logs（通过 task_id 关联）
  2. 删除 tasks（project_id 匹配）
  3. 删除 project
    |
    v
303 重定向到 /projects
```

## 调度器

- 后台 goroutine，按 `interval` 间隔轮询
- 按创建时间排序，FIFO
- 使用 `semaphore.Weighted` 控制全局并发数
- 每个任务在独立 goroutine 中执行
- 支持通过 `context.WithCancel` 取消正在执行的任务

## Git 操作

- 仓库 clone 到 `{repo_base_dir}/{project_id}/`
- 首次触发：`git clone --branch <branch> <url> <dir>`
- 后续触发：`git fetch origin <branch>`
- 解析 HEAD：`git rev-parse origin/<branch>`

### 私有仓库凭据

项目可关联 `RepoCredential`（用户名+密码/Token），clone/fetch 时自动注入 HTTPS URL userinfo：
- 原始 URL：`https://gitlab.com/org/repo.git`
- 注入后：`https://alice:token@gitlab.com/org/repo.git`
- 已有仓库在凭据变更时通过 `git remote set-url origin` 更新
- SSH URL 不处理凭据注入
- 构建 diff：
  - `fromCommit` 为空（首次）：`git show <toCommit>`
  - 有 `fromCommit`：`git diff <from>..<to>`

## LLM 调用

### API 模式（LLMReviewer）

基于 `any-llm-go` 库，根据 `ModelConfig.type` 选择 provider：

| type | provider |
|------|----------|
| openai | providers/openai |
| anthropic | providers/anthropic |
| ollama | providers/ollama |
| deepseek | providers/deepseek |
| gemini | providers/gemini |
| mistral | providers/mistral |

调用方式：
- System message = prompt（project.CustomPrompt 覆盖 modelConfig.Prompt）
- User message = diff 内容
- `enable_thinking=true` 时设置 `ReasoningEffortMedium`

### CLI 模式（CLIReviewer）

通过 `claude -p` 无交互模式执行：
1. 构建 prompt（注入 commit 范围信息）
2. 执行 `claude -p "{prompt}" --output-format json [--max-turns N]`
3. 工作目录 = 项目本地仓库路径
4. 环境变量合并 `os.Environ()` + `extra_config.env_vars`
5. 解析 JSON 输出，提取 result 和 usage 数据

### Prompt 覆盖优先级

`project.CustomPrompt` > `modelConfig.Prompt`

CLI 模式的 prompt 会在基础 prompt 后追加 commit 范围指令：
- 有 from_commit：`请审查此仓库中 {from} 到 {to} 之间的代码变更。使用 git diff {from}..{to} 查看变更内容。`
- 无 from_commit（首次）：`请审查此仓库中最新一次提交 ({to}) 的代码变更。使用 git show {to} 查看变更内容。`
