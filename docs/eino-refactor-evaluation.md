# review-view 引入 CloudWeGo Eino 重构后端逻辑评估

> 评估对象：`~/program/review-view` 当前后端逻辑  
> 参考项目：CloudWeGo Eino  
> 参考资料：Eino README、CloudWeGo 官方文档、EinoExt README、Eino Examples 文档  
> 生成时间：2026-07-29

---

## 1. 结论先说

我的判断：**可以嵌入，值得改，但不建议“一把梭全量重构”。**

更合适的路线是：

```text
第一阶段：保留现有 Gin / GORM / Scheduler / Task / SSE 外壳
        ↓
第二阶段：把 LLM Review 执行逻辑抽成 Eino Graph / Chain
        ↓
第三阶段：把巡检 Scan 逻辑也改成 Eino Workflow / Graph
        ↓
第四阶段：再引入 Agent / Tool / Skill / Multi-Agent 能力
```

也就是说，**Eino 最适合先替换 review-view 里面“LLM 业务编排层”，而不是一开始就替换整个后端架构。**

当前 review-view 的问题不是功能不能用，而是：

- Git 操作、Prompt 拼接、LLM 调用、敏感词扫描、报告生成、通知、日志这些逻辑都混在 `ExecuteTask()` / `RunSchedule()` 里。
- 业务步骤靠手写顺序代码串起来，未来加功能会越来越乱。
- 没有统一的节点级观测、回调、重试、工具化、Agent 化能力。
- Skill / Tool / Agent 这种扩展能力，如果继续在现有代码上硬塞，会很快变成一堆 if/else。

Eino 的优势正好是解决这类问题：

- 用 Graph / Chain / Workflow 表达 AI 任务流程。
- 用 Component 抽象 LLM、Tool、Retriever、Prompt、Lambda。
- 用 Callback 做日志、追踪、指标、Token、SSE 上屏。
- 用 Agent / Tool / Skill 支持未来的开放式扩展。
- 用 GraphTool 把确定性流程包装成 Agent 能力。

所以，**Eino 很适合作为 review-view 后端 AI 执行层的重构基础。**

---

## 2. Eino 是什么

Eino 是 CloudWeGo 生态下的 Go 语言 LLM 应用开发框架，定位类似 Go 版本的 LangChain / LangGraph / ADK。

它主要提供几类能力：

### 2.1 Component 组件抽象

Eino 定义了一批标准组件：

- `ChatModel`
- `Tool`
- `Retriever`
- `Embedding`
- `ChatTemplate`
- `Document Loader`
- `Document Transformer`
- `Indexer`
- `Lambda`

EinoExt 提供官方实现，比如：

| 类型 | 官方实现示例 |
|---|---|
| ChatModel | OpenAI、Claude、Gemini、Ark、Ollama |
| Tool | Google Search、DuckDuckGo 等 |
| Retriever | ElasticSearch、VikingDB 等 |
| Embedding | OpenAI、Ark 等 |
| Document Loader | WebURL、S3、File 等 |
| Callback | Langfuse tracing 等 |

对 review-view 来说，最重要的是：

```text
ChatModel + Tool + Graph + Callback + Agent + Skill
```

### 2.2 Compose 编排

Eino 支持：

- Chain：顺序串联
- Graph：图编排，支持分支、并行、循环
- Workflow：更业务化的数据流编排

这很适合把现在的手写流程：

```text
拉代码 -> 算 diff -> 构造 prompt -> 调 LLM -> 解析结果 -> 敏感词扫描 -> 保存结果 -> 通知
```

改造成可视化、可维护、可扩展的执行图。

### 2.3 ADK / Agent

Eino ADK 支持：

- ChatModelAgent
- ReAct Loop
- Tool Call
- Agent as Tool
- DeepAgent
- Multi-Agent
- 中断 / 恢复
- Session / Memory
- Middleware
- Cancel / TurnLoop

这对 review-view 的未来扩展很关键，因为你提到的 “skill 需求” 本质上更接近 Agent 能力，而不是简单 LLM Completion。

### 2.4 Skill Middleware

Eino Examples 里有 Skill Middleware 示例。

Skill 的定位是：

```text
Tool = 能做什么，是函数/接口/动作
Skill = 怎么做，是可复用知识包/操作手册
```

Skill 一般是：

```text
SKILL.md + references/*.md
```

Agent 可以在运行时按需加载 Skill。

这和你现在想做的方向非常匹配：以后代码审查不只是一个固定 prompt，而是可以根据项目类型、语言、框架、规则、场景动态加载不同 Skill。

---

## 3. review-view 当前后端现状

当前 review-view 后端核心分成两套执行逻辑。

### 3.1 普通 Code Review 任务

主要文件：

```text
internal/service/task_service.go
internal/service/scheduler.go
internal/review/reviewer.go
internal/review/provider_factory.go
internal/review/cli_reviewer.go
internal/review/git_repo.go
internal/service/task_cache.go
```

现在流程是：

```text
用户手动 / Webhook / Cron 触发
        ↓
TaskService.Trigger()
        ↓
创建 pending task
        ↓
Scheduler 每 5 秒轮询 pending task
        ↓
ClaimPending 抢占任务
        ↓
ExecuteTask()
        ↓
EnsureRepo / Checkout
        ↓
拼 Prompt
        ↓
Reviewer.Review()
        ↓
LLM 返回结果
        ↓
敏感词扫描
        ↓
写 DB / 写 .review 文件 / 通知
```

目前执行逻辑主要集中在：

```text
internal/service/scheduler.go: ExecuteTask()
```

这个方法职责偏重。

### 3.2 定时巡检 Scan

主要文件：

```text
internal/service/scan_scheduler.go
internal/service/scan_service.go
internal/model/scan.go
internal/store/gorm/scan_store.go
internal/handler/scan_handler.go
```

现在流程是：

```text
ScanScheduler 每分钟检查 enabled schedule
        ↓
HH:MM 匹配 scan_time
        ↓
RunSchedule()
        ↓
clone/fetch 仓库
        ↓
列出所有远程分支
        ↓
读取 BranchCheckpoints
        ↓
每个分支判断是否有新 commit
        ↓
有变更就拉 commit + diff
        ↓
analyzeBranch() 调 LLM
        ↓
保存 ScanBranchResult
        ↓
生成报告上传 NAS
        ↓
更新 checkpoint
```

这里核心逻辑集中在：

```text
internal/service/scan_service.go: RunSchedule()
internal/service/scan_service.go: analyzeBranch()
```

也属于“能跑，但未来继续加功能会越来越重”。

---

## 4. Eino 和 review-view 的适配度

### 4.1 语言和依赖适配度：高

review-view 是 Go 后端。

Eino 也是 Go 框架，go.mod 要求 Go 1.18+。

当前 review-view：

```text
go 1.25
```

所以版本不是问题。

当前 review-view 已经用：

```text
Gin + GORM + SQLite + any-llm-go + robfig/cron
```

Eino 可以嵌入到现有 service 层，不要求你换掉 Gin / GORM / SQLite。

### 4.2 架构适配度：中高

review-view 当前已经有比较清晰的分层：

```text
handler -> service -> review/store/model
```

Eino 最适合放在：

```text
internal/ai/
internal/workflow/
internal/eino/
```

或者逐步替换：

```text
internal/review/
```

建议新增一层：

```text
internal/ai/
  model_factory.go
  review_graph.go
  scan_graph.go
  tools/
  callbacks/
  skills/
```

不要一开始就把 Eino 代码塞进 `scheduler.go` 或 `scan_service.go`，否则只是换一种方式继续混乱。

### 4.3 业务形态适配度：高

Eino 官方文档有一个重要观点：

```text
确定性的业务流程用 Graph。
开放式、自主规划、多轮交互用 Agent。
两者最佳结合点是 Graph as Tool。
```

review-view 现在的大部分核心流程其实是确定性的：

```text
拉代码 -> 分析变更 -> 调模型 -> 生成结果 -> 入库
```

所以第一阶段应该用 Graph / Chain，而不是直接 DeepAgent。

未来如果要支持 Skill、工具调用、多 Agent 审查，可以再把 Graph 包装成 Tool 给 Agent 使用。

---

## 5. 哪些地方适合改成 Eino

### 5.1 最适合先改：普通 Code Review 执行链路

当前：

```text
Scheduler.ExecuteTask()
```

建议拆成 Eino Graph：

```text
ReviewInput
  ↓
LoadTaskNode
  ↓
LoadProjectNode
  ↓
EnsureRepoNode
  ↓
CheckoutNode
  ↓
BuildReviewContextNode
  ↓
BuildPromptNode
  ↓
ChatModelNode / ClaudeCLINode
  ↓
SensitiveScanNode
  ↓
PersistResultNode
  ↓
NotifyNode
ReviewOutput
```

对应 Go 概念：

- Git 操作：Lambda Node / Tool
- Prompt 构造：Lambda Node / ChatTemplate
- LLM 调用：ChatModel Component
- 敏感词扫描：Lambda Node
- 保存结果：Lambda Node
- 通知：Lambda Node
- 日志 / token / SSE：Callback

这样改的好处：

- 每一步职责更清楚。
- 每个节点可以单独测试。
- 以后要加“二次审查”“风险分级”“结果结构化解析”“失败重试”更容易。
- 可以通过 Callback 统一记录每个节点开始、结束、失败。
- 可以画出 Mermaid 流程图做可视化调试。

### 5.2 很适合改：Scan 巡检分析链路

当前：

```text
RunSchedule() + analyzeBranch()
```

可以拆成两个层次。

#### Schedule 级 Workflow

```text
LoadSchedule
  ↓
BuildScanConfig
  ↓
EnsureRepo
  ↓
ListRemoteBranches
  ↓
LoadCheckpoints
  ↓
ForEachBranch
  ↓
SaveCheckpoints
  ↓
BuildReport
  ↓
UploadNAS
  ↓
CleanupOldReports
```

#### Branch 级 Graph

```text
GetBranchHead
  ↓
CompareCheckpoint
  ↓
CollectCommits
  ↓
CollectDiff
  ↓
BuildScanPrompt
  ↓
ChatModel
  ↓
ParseRiskLevel
  ↓
SaveBranchResult
```

这个非常适合 Eino Graph / Workflow，因为 scan 本质上是多分支批处理。

以后可以扩展：

- 分支并发分析。
- 高风险分支二次分析。
- 根据语言选择不同 Skill。
- 根据风险类型走不同 Agent。
- 报告生成独立成 Graph。
- NAS / S3 / 飞书 / 邮件报告输出做成 Tool。

### 5.3 适合改：模型调用层

当前模型层有两套：

```text
API 模式：any-llm-go
CLI 模式：claude -p
```

EinoExt 里已经有 OpenAI / Claude / Gemini / Ollama 等 ChatModel 实现。

理论上可以把 API 模式逐步迁到 Eino ChatModel。

但是这里要注意：**不要立刻删掉 any-llm-go。**

原因：

- 当前 review-view 已经基于 any-llm-go 做了多 provider 统一调用。
- 现有模型配置、thinking、token 统计、base URL 配置都已经围绕 any-llm-go 实现。
- EinoExt 的 Provider 行为和 any-llm-go 不完全一致，需要逐个验证。
- Claude CLI 模式不是标准 ChatModel，需要自己适配成 Eino Component 或 Tool。

推荐做法：

```text
先写 Adapter，把现有 Reviewer 包成 Eino Lambda / ChatModel 风格
稳定后，再逐步把 API Provider 换成 EinoExt ChatModel
```

也就是先用 Eino 管流程，不急着换掉模型底层。

### 5.4 适合改：日志、SSE、Token 统计

当前日志是手动写：

```go
s.appendLog(task.ID, level, message)
s.cache.AppendResultChunk(task.ID, text)
s.cache.UpdateTokens(task.ID, ...)
```

Eino 有 Callback 机制，天然适合做：

- OnStart：节点开始，写日志
- OnEnd：节点结束，写日志和指标
- OnError：节点失败，写错误
- OnEndWithStreamOutput：处理流式输出
- ChatModel Callback：记录模型输入、输出、Token
- Tool Callback：记录工具调用参数和结果

这样可以把现在散落的日志逻辑集中起来。

建议做：

```text
internal/ai/callbacks/task_callback.go
```

专门把 Eino callback 映射到 review-view 的 TaskCache / DB / SSE。

### 5.5 适合改：Prompt 和 Skill

当前 Prompt 是：

```text
modelConfig.Prompt + project.CustomPrompt
```

未来如果要支持 Skill，可以设计成：

```text
全局 Review Policy
项目 Custom Prompt
语言 Skill
框架 Skill
安全规则 Skill
历史问题 Skill
团队规范 Skill
```

比如：

```text
skills/
  go-review/SKILL.md
  vue-review/SKILL.md
  android-review/SKILL.md
  sql-risk/SKILL.md
  config-risk/SKILL.md
  payment-security/SKILL.md
  tmm-business-rule/SKILL.md
```

Eino Skill Middleware 可以让 Agent 按需加载 Skill。

但普通 Graph 流程里，也可以不走 Agent，直接由后端根据项目类型加载 Skill 文档拼 Prompt。

建议分两步：

```text
第一步：后端确定性加载 skill 文档，拼进 prompt
第二步：Agent 自主判断要加载哪些 skill
```

不要一开始就让 Agent 自主选择所有 Skill，否则不可控。

---

## 6. 哪些地方不建议马上改

### 6.1 不建议替换 Gin / Handler 层

Gin 现在没问题。

Eino 不是 Web 框架，不应该替代 Gin。

保留：

```text
internal/handler
internal/app/router.go
```

### 6.2 不建议替换 GORM / Store 层

Eino 不负责业务数据持久化。

现有：

```text
TaskStore
ProjectStore
ScanScheduleStore
ScanJobStore
```

都可以继续用。

### 6.3 不建议马上替换 Scheduler

当前普通任务 Scheduler 虽然简单，但有几个关键能力已经可用：

- pending 队列
- FIFO
- ClaimPending 原子抢占
- 全局 semaphore 并发控制
- 任务取消
- 任务超时
- 重启恢复 running task

Eino 主要解决“任务内部怎么执行”，不是任务队列。

所以第一阶段保留 Scheduler，只把：

```text
ExecuteTask() 内部逻辑
```

替换成：

```text
ReviewGraph.Invoke()/Stream()
```

### 6.4 不建议一开始就上 DeepAgent

DeepAgent 很强，但也更不可控：

- 会自己规划。
- 会调用工具。
- 会产生更多模型请求。
- token 成本高。
- 过程复杂，调试难度大。
- 对模型能力要求更高。

review-view 的核心 Code Review 应该先保持确定性。

更稳的方式：

```text
核心审查链路：Graph
开放式辅助能力：Agent
复杂工作流封装：Graph as Tool
```

---

## 7. 推荐目标架构

建议最终架构是：

```text
Gin Handler
  ↓
Service 层
  ↓
Task Scheduler / Scan Scheduler
  ↓
Eino Workflow Runtime
  ↓
Graph / Chain / Agent / Tool / Skill
  ↓
Store / Git / Model / Notification / NAS
```

目录可以这样设计：

```text
internal/
  ai/
    model/
      factory.go              # 从 ModelConfig 创建 Eino ChatModel 或 Adapter
      anyllm_adapter.go        # 兼容现有 any-llm-go
      claude_cli_adapter.go    # 兼容 claude -p

    workflow/
      review_graph.go          # 普通 Code Review 图
      scan_schedule_graph.go   # 巡检总流程图
      scan_branch_graph.go     # 单分支分析图

    node/
      git_nodes.go             # EnsureRepo / Checkout / Diff / CommitLog
      prompt_nodes.go          # Prompt / Skill 拼装
      persist_nodes.go         # 保存 Task / ScanJob / BranchResult
      notify_nodes.go          # 通知
      sensitive_nodes.go       # 敏感词扫描

    tool/
      git_tool.go
      repo_file_tool.go
      nas_tool.go
      project_context_tool.go

    skill/
      loader.go
      selector.go
      middleware.go

    callback/
      task_callback.go         # Eino Callback -> TaskCache/SSE/DB
      scan_callback.go
      metrics_callback.go

    agent/
      review_agent.go          # 未来引入
      scan_agent.go
      risk_agent.go
```

现有 Service 变成编排入口：

```go
func (s *Scheduler) ExecuteTask(ctx context.Context, taskID int64) error {
    input := ai.ReviewInput{TaskID: taskID}
    output, err := s.reviewWorkflow.Run(ctx, input)
    // 只处理最终状态兜底
}
```

---

## 8. 分阶段改造方案

### 阶段 0：不动功能，补抽象边界

目标：给 Eino 迁移做准备。

动作：

1. 把 `ExecuteTask()` 拆成更小的私有方法。
2. 定义明确输入输出结构：

```go
type ReviewInput struct {
    TaskID int64
}

type ReviewContext struct {
    Task *model.Task
    Project *model.Project
    ModelConfig *model.ModelConfig
    RepoDir string
    Prompt string
}

type ReviewOutput struct {
    Result string
    InputTokens int64
    OutputTokens int64
}
```

3. 保留原逻辑，先不引入 Eino。

价值：降低后续迁移风险。

### 阶段 1：把普通 Review 改成 Eino Chain / Graph

目标：先让核心 Code Review 走 Eino 编排。

推荐 Graph：

```text
load_task
  -> load_project
  -> load_model
  -> ensure_repo
  -> checkout
  -> build_prompt
  -> run_review_model
  -> sensitive_scan
  -> persist_result
  -> notify
```

这一阶段可以继续使用现有 Reviewer：

```text
run_review_model 节点内部调用现有 reviewerFactory(modelConfig).Review()
```

这样风险最低。

### 阶段 2：接入 Eino Callback

目标：把日志、SSE、token、节点耗时统一治理。

动作：

- 写 `TaskCallbackHandler`
- Graph 运行时挂载 callback
- 节点开始/结束/错误自动写 TaskCache
- ChatModel 输出流映射到 SSE

完成后，`ExecuteTask()` 里面大量手写日志可以删掉。

### 阶段 3：把 Scan 巡检改成 Graph / Workflow

目标：拆掉 `RunSchedule()` 的大函数。

建议先改 branch 分析：

```text
ScanBranchGraph
```

再改 schedule 总流程：

```text
ScanScheduleWorkflow
```

原因：branch 分析最像 AI workflow，改起来收益最大。

### 阶段 4：Skill 确定性接入

目标：让不同项目/语言/风险类型有不同审查规则。

先不要让 Agent 自主选。

先做：

```text
Project.Language / Project.Tags / Project.SkillIDs
        ↓
后端加载对应 SKILL.md
        ↓
拼到 prompt
```

可以新增表：

```text
skills
project_skills
```

或者先用文件目录配置。

### 阶段 5：Agent 化扩展

目标：让 review-view 从“固定审查器”升级成“可扩展 AI 审查平台”。

可以加入：

- ReviewAgent：针对复杂代码变更自主调用工具
- RiskAgent：专门判断高风险变更
- SecurityAgent：专门查安全问题
- DBMigrationAgent：专门查数据库变更
- ConfigAgent：专门查配置、密钥、地址、端口变更

但是核心建议仍然是：

```text
确定性主流程 = Graph
开放式分析能力 = Agent
Agent 调用 GraphTool / Tools
```

---

## 9. 未来扩展性优势

### 9.1 Skill 扩展会更自然

现在如果加 Skill，大概率只能做：

```text
if 项目类型 == Go then prompt += Go规则
if 项目类型 == Android then prompt += Android规则
if 风险类型 == DB then prompt += DB规则
```

以后会越来越乱。

Eino 后可以演进成：

```text
Skill Backend
  ↓
Skill Middleware
  ↓
Agent 按需加载
  ↓
或者 Graph 确定性加载
```

这让 Skill 成为一等能力，而不是 Prompt 字符串拼接。

### 9.2 Tool 能力会更容易加

未来 Review 不一定只看 diff。

可能需要工具：

- 读仓库文件
- 查调用链
- 查配置文件
- 查数据库 migration
- 查依赖漏洞
- 查 Jira / 禅道 / GitLab MR
- 查历史审查记录
- 查线上日志
- 查接口文档

Eino Tool 能把这些能力标准化。

Agent 就可以按需调用。

### 9.3 多阶段审查更容易

比如未来想做：

```text
第一阶段：快速风险分类
第二阶段：高风险才拉完整 diff
第三阶段：专项 Agent 深挖
第四阶段：生成结构化报告
第五阶段：推送企业微信
```

现在手写会很乱。

Eino Graph 很适合表达这种流程。

### 9.4 可观测性更好

Eino Callback 可以天然记录：

- 哪个节点执行了
- 节点耗时
- 节点输入输出类型
- 模型调用次数
- Tool 调用次数
- Token 使用
- 错误位置
- 流式输出

这比现在手写 `appendLog()` 更体系化。

### 9.5 Graph 可视化和调试更好

Eino Examples 里有 Graph / Chain / Workflow 渲染 Mermaid 的能力。

未来可以在 review-view 前端展示：

```text
本次 Review 执行图
每个节点状态
每个节点耗时
失败节点
重试节点
```

这对运维排障很有价值。

### 9.6 更容易做“审查策略市场”

如果后面要产品化，可以做：

- 内置 Skill
- 用户自定义 Skill
- 项目绑定 Skill
- 团队共享 Skill
- 高风险规则模板
- 不同语言规则模板
- 不同业务线规则模板

Eino 的 Skill / Tool / Agent 模型更适合这个方向。

---

## 10. 风险和坑

### 10.1 引入复杂度会上升

Eino 是框架，不是一个简单 SDK。

引入后会多出这些概念：

- Component
- Graph
- Chain
- Workflow
- Callback
- Tool
- Agent
- Middleware
- Skill
- Runner
- Event

如果一开始边界没设计好，会比现在更乱。

所以必须先定原则：

```text
Eino 只管 AI 执行编排，不管 Web / DB / 权限 / 基础调度。
```

### 10.2 不要把所有逻辑 Agent 化

Code Review 平台需要稳定性。

Agent 自主性太强，容易出现：

- 结果不稳定
- 工具调用不可控
- 成本不可控
- 审查耗时不可控
- Debug 难

建议：

```text
主流程 Graph 化
增强能力 Agent 化
```

### 10.3 模型 Provider 迁移有成本

现在 any-llm-go 已经支持多 Provider。

EinoExt 也支持多 Provider，但配置结构、Token 统计、thinking、baseURL、流式行为不一定完全一致。

建议短期保留 any-llm-go，通过 Adapter 接进 Eino。

### 10.4 Claude CLI 要自定义适配

`claude -p` 不是标准 ChatModel API。

如果要接 Eino，有几种方式：

1. 包成 Lambda Node。
2. 包成自定义 ChatModel Component。
3. 包成 Tool。

短期建议包成 Lambda Node，简单稳定。

### 10.5 Skill 自主加载要做权限控制

以后如果 Skill 可以访问文件、工具、命令，需要注意：

- 哪些项目能用哪些 Skill
- Skill 能不能触发 Shell
- Tool 有没有白名单
- Agent 能不能读仓库外文件
- 是否需要人工审批

Eino 有 Human-in-the-loop / interrupt/resume 思路，但产品层还要自己做权限。

### 10.6 Scan 并发问题仍然要业务层解决

Eino 不会自动解决你现在 scan 手动触发可能重复跑的问题。

这个仍然要在业务层加：

```text
同一个 schedule 同一时间只能有一个 running job
```

可以在 `ScanJobStore` 增加 active job 检查，或者用 DB 约束 / 分布式锁。

---

## 11. 可以怎么具体改

### 11.1 新增 Eino 入口接口

建议先定义一个独立接口：

```go
type ReviewWorkflow interface {
    Run(ctx context.Context, input ReviewInput) (*ReviewOutput, error)
}
```

Scheduler 只依赖这个接口。

这样未来可以在配置里切换：

```text
legacy reviewer
Eino graph reviewer
Eino agent reviewer
```

### 11.2 第一版 ReviewGraph 不要太复杂

第一版节点可以粗一点：

```text
prepare_context
run_reviewer
post_process
```

不要一开始拆 20 个节点。

等跑稳后再细拆。

### 11.3 Callback 先只做日志和错误

第一版 Callback 做：

- node start
- node end
- node error

Token 和流式输出可以第二步再做。

### 11.4 Skill 先做静态加载

第一版：项目配置绑定 skill 文件。

比如：

```text
skills/go-review/SKILL.md
skills/security-review/SKILL.md
```

项目配置：

```json
{
  "skills": ["go-review", "security-review"]
}
```

后端读取后拼入 prompt。

后续再接 Eino Skill Middleware。

---

## 12. 建议优先级

### P0：先做架构边界

- 定义 `ReviewWorkflow` 接口
- 定义 `ReviewInput / ReviewOutput / ReviewContext`
- 把 `ExecuteTask()` 内部逻辑拆出来

### P1：普通 Review Graph 化

- 引入 Eino compose
- 用 Graph/Chain 包现有 Reviewer
- Scheduler 保持不变
- 测试普通手动 review / webhook / cron / retry / cancel

### P2：Callback 接入 TaskCache

- Graph 节点日志自动进入 TaskCache
- 错误统一记录
- SSE 继续复用现有接口

### P3：Scan Branch Graph 化

- 先改 `analyzeBranch()`
- 再改 `RunSchedule()`

### P4：Skill 静态接入

- 文件型 Skill
- 项目绑定 Skill
- Prompt 构造节点加载 Skill

### P5：Agent / Tool 扩展

- Repo 文件读取工具
- Git 工具
- Risk 专项 Agent
- Graph as Tool
- Skill Middleware

---

## 13. 最终建议

如果目标只是“现在功能能跑”，不用 Eino 也可以。

但如果你的目标是：

- 以后支持 Skill
- 支持复杂审查策略
- 支持工具调用
- 支持多 Agent
- 支持更细日志和可观测性
- 支持不同语言/项目/业务线的审查规则
- 把 review-view 从小工具升级成可扩展平台

那 Eino 是值得引入的。

我建议的核心原则是：

```text
不要用 Eino 重写整个后端。
用 Eino 重构 AI 执行层。

不要一开始就 Agent 化。
先 Graph 化，再 Tool 化，最后 Agent 化。

不要马上替换 any-llm-go。
先 Adapter 兼容，后续再迁移 Provider。

不要让 Skill 一开始完全自主。
先静态绑定，后续再 Skill Middleware。
```

最终目标架构应该是：

```text
review-view = 任务平台 + 项目管理 + 调度 + 展示
Eino = AI 工作流编排 + Tool/Skill/Agent 扩展层
```

这个组合是比较合理的。

---

## 14. 一句话判断

**Eino 很适合嵌入 review-view，但应该作为“AI 工作流内核”逐步接入，不应该替代现有 Web/DB/调度框架；最先改普通 Review 执行链路，随后改 Scan，最后再上 Skill 和 Agent。**
