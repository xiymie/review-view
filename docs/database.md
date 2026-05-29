# 数据库设计文档

数据库：SQLite，通过 GORM AutoMigrate 自动建表。

## projects 表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | int64 | PK, auto | 主键 |
| name | string | NOT NULL | 项目名称 |
| repo_url | string | NOT NULL | 仓库地址 |
| branch | string | NOT NULL | 要 review 的分支 |
| model_config_id | int64 | NOT NULL, INDEX | 关联的模型配置 ID |
| custom_prompt | string | | 覆盖模型配置中的公共 prompt，为空则使用模型配置的 prompt |
| last_reviewed_commit | string | | 上次 review 到的 commit hash |
| overflow_strategy | string | NOT NULL | `queue` / `reject` |
| task_timeout | int | nullable | 任务超时（分钟），为空使用全局配置 |
| repo_credential_id | int64 | nullable, INDEX | 关联的仓库凭据 ID，为空表示公开仓库 |
| created_at | timestamp | | 创建时间 |
| updated_at | timestamp | | 更新时间 |

## model_configs 表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | int64 | PK, auto | 主键 |
| name | string | NOT NULL | 配置名称 |
| type | string | NOT NULL, INDEX | 平台类型：openai / anthropic / ollama / deepseek / gemini / mistral / claude_cli |
| base_url | string | | API 地址（claude_cli 不使用） |
| api_key | string | | API Key（claude_cli 不使用，通过 env_vars 注入） |
| model | string | | 模型名称（claude_cli 不使用） |
| prompt | string | NOT NULL | 公共 review prompt |
| max_context | int | | 最大上下文 token 数（claude_cli 不使用） |
| enable_thinking | bool | | 是否启用 thinking（claude_cli 不使用） |
| extra_config | string | | JSON 格式，存储各类型特有参数 |
| created_at | timestamp | | 创建时间 |
| updated_at | timestamp | | 更新时间 |

### extra_config JSON 结构（claude_cli 类型）

```json
{
  "cli_path": "claude",
  "env_vars": { "KEY": "value" },
  "max_turns": 10
}
```

- `cli_path`：可执行文件路径，默认 `claude`
- `env_vars`：注入的环境变量
- `max_turns`：`--max-turns` 参数，可选

## tasks 表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | int64 | PK, auto | 主键 |
| project_id | int64 | NOT NULL, INDEX | 关联项目 ID |
| status | string | NOT NULL, INDEX | pending / running / completed / failed / cancelled / rejected |
| from_commit | string | | 起始 commit（首次为空） |
| to_commit | string | NOT NULL, INDEX | 结束 commit |
| diff_content | string | | 获取到的 diff 内容 |
| result | string | | LLM 返回的 review 结果（Markdown） |
| error_message | string | | 失败原因 |
| triggered_by | string | NOT NULL | manual / webhook |
| input_tokens | int64 | | 输入 token 数 |
| output_tokens | int64 | | 输出 token 数 |
| cache_creation_tokens | int64 | | 缓存写入 token 数 |
| cache_read_tokens | int64 | | 缓存命中 token 数 |
| created_at | timestamp | | 创建时间 |
| started_at | timestamp | nullable | 开始执行时间 |
| finished_at | timestamp | nullable | 完成时间 |

## task_logs 表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | int64 | PK, auto | 主键 |
| task_id | int64 | NOT NULL, INDEX | 关联任务 ID |
| level | string | NOT NULL | 日志级别：info / warn / error |
| message | string | NOT NULL | 日志消息 |
| created_at | timestamp | | 创建时间 |

## global_configs 表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| key | string | PK | 配置键 |
| value | string | NOT NULL | 配置值 |
| created_at | timestamp | | 创建时间 |
| updated_at | timestamp | | 更新时间 |

### 预设配置项

| key | 默认值 | 说明 |
|-----|--------|------|
| max_concurrent_tasks | 3 | 全局最大并发任务数 |
| global_overflow_strategy | queue | 全局溢出策略 |
| repo_base_dir | ./repos | 仓库 clone 根目录 |
| task_timeout | 30 | 全局默认任务超时（分钟） |

## repo_credentials 表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | int64 | PK, auto | 主键 |
| name | string | NOT NULL | 凭据名称，如 "公司 GitLab" |
| username | string | NOT NULL | 用户名 |
| password | string | NOT NULL | 密码或 Token |
| created_at | timestamp | | 创建时间 |
| updated_at | timestamp | | 更新时间 |
