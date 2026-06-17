# 需求文档

## 产品定位

自托管的 Code Review 工具，单二进制部署。通过项目管理 Git 仓库的自动/手动 review 流程，调用 LLM 对代码变更进行智能审查，结果在 Web 页面展示。

## 核心功能

### 项目管理

- 创建/编辑/查看/删除项目，每个项目关联一个 Git 仓库和分支
- 项目关联一个模型配置，支持自定义 Prompt 覆盖模型配置中的公共 Prompt
- 支持溢出策略（排队等待 / 拒绝），控制项目有进行中任务时的行为
- 支持任务超时配置（分钟），可覆盖全局默认值
- 项目详情页展示配置信息、历史任务列表、手动触发审核
- 删除项目时级联删除所有关联任务，有运行中任务时不允许删除
- 项目列表显示最近任务状态

### 仓库凭据管理

- 创建/编辑/删除仓库凭据（名称、用户名、密码/Token）
- 凭据作为全局资源管理，项目通过下拉框选择是否关联
- 公开仓库无需配置凭据
- 被项目引用的凭据不允许删除
- 凭据通过 HTTPS URL userinfo 方式注入到 git clone/fetch 命令
- 已 clone 的仓库在凭据变更后通过 `git remote set-url` 更新 remote URL

### 模型配置管理

- 创建/编辑模型配置，支持 API 和 CLI 两种模式
- API 模式支持 6 种平台：openai / anthropic / ollama / deepseek / gemini / mistral
- CLI 模式通过 `claude -p` 无交互模式执行代码审查
- 支持测试模型连接，创建前可验证配置是否可用
- 支持 Thinking 模式开关

### 任务调度

- 后台调度器轮询 pending 任务，按创建时间 FIFO 执行
- 全局并发控制（可配置最大并发数）
- 任务支持取消和重试
- 任务超时保护（项目级覆盖 > 全局配置 > 默认 30 分钟）

### 触发方式

- **手动触发**：项目详情页点击"手动触发审核"按钮
- **Webhook 触发**：POST `/webhook/{projectId}`，可选传入 commit hash

### 触发保护

- `fromCommit == toCommit`（无新提交）时自动跳过，不创建任务
- 已存在相同 commit 范围的 completed 任务时自动跳过
- diff 为空时任务标记为 failed 并记录原因

### 审核结果展示

- 任务详情页通过 tab 切换展示：Review 结果 / Diff 内容 / 执行日志
- 执行日志记录任务生命周期事件：仓库同步、diff 获取、LLM 调用、完成/失败等
- 日志分为 info / warn / error 三个级别
- 展示 token 消耗统计（输入/输出/缓存写入/缓存命中）

### 列表操作

- 所有数据表格末尾有"操作"列，提供详情/编辑/删除等按钮
- 任务列表根据状态动态显示"取消"（running）和"重试"（failed/cancelled）按钮
- 状态显示使用彩色药丸标签区分不同状态

### 全局设置

- 最大并发任务数（默认 3）
- 全局溢出策略（默认 queue）
- 任务超时时间（默认 30 分钟）
- 仓库根目录（默认 ./repos）

## 技术约束

- 单二进制，前端（Vue 3 + Element Plus）构建产物通过 `go:embed` 打包
- 无认证，默认内网使用
- 数据库使用 SQLite，GORM AutoMigrate 建表
- Git 操作通过 `os/exec` 调用本地 `git` 命令
- LLM API 通过 `any-llm-go` 统一调用，文档[any-llm-go-usage.md](./any-llm-go-usage.md)
- 环境变量配置：`APP_ADDR`（默认 :8080）、`DATABASE_DSN`（默认 SQLite 文件）
