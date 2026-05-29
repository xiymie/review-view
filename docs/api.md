# 接口设计文档

## 页面路由

### GET /

仪表盘页面。返回统计卡片 + 最近任务列表。

### GET /projects

项目列表页面。

### GET /projects/new

新建项目表单页面。

### GET /projects/:id

项目详情页面。展示项目配置、历史任务列表。

**路径参数：**
- `id`：项目 ID

### GET /projects/:id/edit

编辑项目表单页面，预填现有数据。

**路径参数：**
- `id`：项目 ID

### GET /models

模型配置列表页面。

### GET /models/new

新建模型配置表单页面。

### GET /models/:id/edit

编辑模型配置表单页面，预填现有数据。

**路径参数：**
- `id`：模型配置 ID

### GET /tasks

任务列表页面。

### GET /tasks/:id

任务详情页面。三段式布局：信息条（Commit 范围 + Token + 时间）→ 独立执行日志区域 → Review 结果/Diff 内容双 tab。运行中任务通过 SSE 实时更新日志和状态。

**路径参数：**
- `id`：任务 ID

### GET /settings

全局设置表单页面。

### GET /credentials

凭据列表页面。

### GET /credentials/new

新建凭据表单页面。

### GET /credentials/:id/edit

编辑凭据表单页面，预填现有数据（密码字段留空不修改）。

**路径参数：**
- `id`：凭据 ID

## 表单提交接口

### POST /projects

创建项目。成功后 303 重定向到 `/projects`。

**请求（form-data）：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 项目名称 |
| repo_url | string | 是 | 仓库地址 |
| branch | string | 是 | 分支 |
| model_config_id | int64 | 是 | 模型配置 ID |
| custom_prompt | string | 否 | 自定义 Prompt |
| overflow_strategy | string | 是 | queue / reject，默认 queue |
| task_timeout | int | 否 | 任务超时（分钟） |
| repo_credential_id | int64 | 否 | 仓库凭据 ID，不传或为空表示公开仓库 |

### POST /projects/:id

更新项目。成功后 303 重定向到 `/projects/:id`。

**路径参数：**
- `id`：项目 ID

**请求（form-data）：** 同创建项目。

### POST /projects/:id/trigger

手动触发审核。成功后 303 重定向到任务详情页 `/tasks/:taskId`。如果跳过（无新提交），重定向到项目详情页。

**路径参数：**
- `id`：项目 ID

### POST /projects/:id/delete

删除项目及其所有关联任务。成功后 303 重定向到 `/projects`。

**路径参数：**
- `id`：项目 ID

**前置检查：** 项目有运行中的任务时返回 400 错误。

### POST /models

创建模型配置。成功后 303 重定向到 `/models`。

**请求（form-data）：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 配置名称 |
| type | string | 是 | 平台类型 |
| base_url | string | 否 | API 地址 |
| api_key | string | 否 | API Key |
| model | string | 否 | 模型名称 |
| prompt | string | 是 | Review Prompt |
| max_context | int | 否 | 最大上下文 Token |
| enable_thinking | string | 否 | checkbox，值为 "on" |
| cli_path | string | 否 | CLI 路径 |
| env_vars_json | string | 否 | 环境变量 JSON |
| max_turns | int | 否 | Max Turns |

### POST /models/:id

更新模型配置。成功后 303 重定向到 `/models`。

**路径参数：**
- `id`：模型配置 ID

**请求（form-data）：** 同创建模型配置。

### POST /settings

更新全局设置。成功后 303 重定向到 `/settings`。

**请求（form-data）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| max_concurrent_tasks | int | 最大并发任务数 |
| global_overflow_strategy | string | queue / reject |
| task_timeout | int | 任务超时（分钟） |
| repo_base_dir | string | 仓库根目录 |

### POST /credentials

创建凭据。成功后 303 重定向到 `/credentials`。

**请求（form-data）：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 凭据名称 |
| username | string | 是 | 用户名 |
| password | string | 是 | 密码或 Token |

### POST /credentials/:id

更新凭据。成功后 303 重定向到 `/credentials`。

**路径参数：**
- `id`：凭据 ID

**请求（form-data）：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 凭据名称 |
| username | string | 是 | 用户名 |
| password | string | 否 | 密码或 Token，留空则不修改 |

### POST /credentials/:id/delete

删除凭据。被项目引用时返回 400 错误。成功后 303 重定向到 `/credentials`。

**路径参数：**
- `id`：凭据 ID

## API 接口

### POST /api/models/test

测试模型连接。提交当前表单数据，发送一条最小化请求验证 LLM 连接。

**请求（form-data）：** 同创建模型配置的字段。

**响应（JSON）：**

成功：
```json
{ "ok": true, "message": "连接成功: ..." }
```

失败：
```json
{ "ok": false, "error": "请求失败: ..." }
```

- 200：连接成功（API 模式返回 LLM 响应内容，CLI 模式返回"CLI 配置已就绪"）
- 400：配置不合法或创建 provider 失败
- 502：LLM 请求失败

### POST /api/tasks/:id/cancel

取消正在运行的任务。成功后 303 重定向到 `/tasks/:id`。

**路径参数：**
- `id`：任务 ID

**错误响应：** 400，任务未在运行状态。

### POST /api/tasks/:id/retry

重试失败/已取消的任务。创建一个新的 pending 任务。成功后 303 重定向到新任务详情页。

**路径参数：**
- `id`：原任务 ID

**错误响应：** 400，任务状态不允许重试（仅 failed/cancelled 可重试）。

### GET /api/tasks/:id/stream

SSE 端点，推送运行中任务的增量日志、流式结果和状态变更。

**路径参数：**
- `id`：任务 ID

**行为：**
- 已结束任务：直接返回 JSON 快照 `{ "status", "result", "done": true }`（result 为原始 Markdown）
- 运行中/等待中任务：建立 SSE 连接

**SSE 事件类型：**

| event | data 格式 | 说明 |
|-------|-----------|------|
| `log` | `{ "id", "level", "message", "createdAt" }` | 单条增量日志 |
| `result` | `{ "content" }` | 全量结果快照（累积的 Markdown），每次包含完整内容 |
| `done` | `{ "status", "result" }` | 任务结束，包含最终 Review 结果（Markdown） |

连接建立时先发送 DB 中已有的全量日志，之后推送内存缓冲区的新日志和流式结果快照。

### POST /webhook/:projectId

Webhook 触发审核。

**路径参数：**
- `projectId`：项目 ID

**请求（JSON，可选）：**

```json
{ "commit": "abc123" }
```

- `commit`：目标 commit hash，不传则使用分支最新 commit

**响应（JSON）：**

```json
{ "skipped": false, "task_id": 42 }
```

- `skipped`：是否跳过（无新提交或重复范围）
- `task_id`：创建的任务 ID，跳过时为 0

**状态码：**
- 202：成功创建任务或跳过
- 400：参数错误或触发失败
