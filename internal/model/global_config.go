package model

import "time"

const (
	GlobalConfigKeyMaxConcurrentTasks = "max_concurrent_tasks"
	GlobalConfigKeyOverflowStrategy   = "global_overflow_strategy"
	GlobalConfigKeyRepoBaseDir        = "repo_base_dir"
	GlobalConfigKeyTaskTimeout        = "task_timeout"

	GlobalConfigKeySMTPHost     = "smtp_host"
	GlobalConfigKeySMTPPort     = "smtp_port"
	GlobalConfigKeySMTPUsername = "smtp_username"
	GlobalConfigKeySMTPPassword = "smtp_password"
	GlobalConfigKeySMTPFrom     = "smtp_from"
	GlobalConfigKeySMTPFromName = "smtp_from_name"
	GlobalConfigKeySMTPTLS      = "smtp_tls"

	GlobalConfigKeyScheduledScanUnchanged = "scheduled_scan_unchanged" // 定时扫描无新提交是否扫描，默认 false（跳过）
	GlobalConfigKeyManualScanUnchanged    = "manual_scan_unchanged"    // 手动扫描无新提交是否扫描，默认 true（扫描）

	// 巡检全局配置
	GlobalConfigKeyScanTime        = "scan_default_time"   // 默认巡检时间 HH:MM，如 "09:00"
	GlobalConfigKeyScanPrompt      = "scan_default_prompt" // 默认巡检提示词
	GlobalConfigKeyScanNasURL      = "scan_nas_url"        // NAS WebDAV 地址
	GlobalConfigKeyScanNasUsername = "scan_nas_username"   // NAS 用户名
	GlobalConfigKeyScanNasPassword = "scan_nas_password"   // NAS 密码
	GlobalConfigKeyScanRetainDays  = "scan_retain_days"    // NAS 报告保留天数，0=永久保留
)

const LegacyDefaultScanPrompt = `仓库 %s，分支 %s，日期 %s，共 %d 个提交：

%s

以下是本次变更的代码 diff：

%s

请结合 commit 信息和代码变更完成以下两项分析：

**一、风险识别**
识别以下类型的风险（有则列出，无则不提）：
- DB：数据库结构变更（建表/删表/加减字段/索引）
- DEP：依赖版本升级（可能引入 breaking change）
- CFG：配置变更（环境变量/密钥/服务地址/端口）
- BIZ：核心业务逻辑改动（支付/权限/核心流程）
- SEC：安全相关（认证/鉴权/加密/敏感数据）

**二、逻辑漏洞检查**
仅关注逻辑层面的问题，忽略语法错误，检查：
- 边界条件未处理（空值/零值/越界）
- 并发/竞态条件
- 错误处理缺失或忽略错误返回值
- 条件判断不严谨导致意外分支
- 数据流中的遗漏校验或状态不一致

**输出格式：**
**风险等级**：无风险 / 低风险 / 中风险 / 高风险
**命中风险类型**：（命中的标签，无则填"—"）
**风险说明**：（简要说明触发原因）
**逻辑漏洞**：（列出发现的逻辑问题；若无则填"未发现"）`

const DefaultScanPrompt = `仓库 %[1]s，分支 %[2]s，日期 %[3]s，共 %[4]d 个提交。

请结合仓库上下文、项目约定、commit 信息和代码 diff 完成巡检。系统会在最终 Prompt 中按顺序附加：项目选择的 Skill、全局默认巡检提示词、巡检配置自定义提示词、CLAUDE.md、commit 信息和 diff 代码。

**一、风险识别**
识别以下类型的风险（有则列出，无则不提）：
- DB：数据库结构变更（建表/删表/加减字段/索引）
- DEP：依赖版本升级（可能引入 breaking change）
- CFG：配置变更（环境变量/密钥/服务地址/端口）
- BIZ：核心业务逻辑改动（支付/权限/核心流程）
- SEC：安全相关（认证/鉴权/加密/敏感数据）
- PERF：性能风险（慢查询/N+1/大循环/缓存失效/资源泄漏）
- ARCH：架构和边界风险（职责混乱/跨层调用/状态边界破坏）

**二、逻辑漏洞检查**
仅关注逻辑层面的问题，忽略语法错误，检查：
- 边界条件未处理（空值/零值/越界）
- 并发/竞态条件
- 错误处理缺失或忽略错误返回值
- 条件判断不严谨导致意外分支
- 数据流中的遗漏校验或状态不一致
- 调用方/上游/配置/中间件是否已经提供了约束或保护

**三、高风险判定前必须做上下文复核**
不要只因为“单看某个函数像有问题”就直接判高风险。由于代码已拉取到本地，发现疑似问题后必须结合相关调用方、上游校验、配置、中间件、事务/幂等保护和项目 CLAUDE.md 复核确认。
判定高风险前必须验证：
- 该问题在真实调用链中是否可达；
- 上游是否已经做过鉴权、参数校验、事务保护、幂等保护或边界过滤；
- 相关配置、CLAUDE.md 项目约定、commit 意图是否已经说明这是预期行为；
- 是否存在明确的数据损坏、权限绕过、资金/核心业务错误、安全漏洞或线上故障路径。
如果只有局部代码疑点，但结合整体上下文不可达、已有保护或影响有限，请降级为中/低风险，并在“高危验证”里说明为什么没有判高风险。

**输出格式：**
**风险等级**：无风险 / 低风险 / 中风险 / 高风险
**命中风险类型**：（命中的标签，无则填"—"）
**风险说明**：（简要说明触发原因）
**高危验证**：（若为高风险，说明真实调用链/上下文证据；若非高风险但存在局部疑点，说明为何降级）
**逻辑漏洞**：（列出发现的逻辑问题；若无则填"未发现"）`

type GlobalConfig struct {
	Key       string `gorm:"primaryKey"`
	Value     string `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
