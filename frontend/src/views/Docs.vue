<template>
  <div class="docs-root">

    <!-- Content -->
    <main class="docs-main" ref="mainRef" @scroll="onScroll">

      <!-- Hero -->
      <header class="hero">
        <div class="hero-inner">
          <div class="hero-eyebrow">
            <span class="eyebrow-dot" />文档 · 代码审核平台
          </div>
          <h1>使用指南</h1>
          <p>AI 驱动的自托管代码审查平台。从接入模型到 Webhook 自动触发，全流程使用说明。</p>
          <div class="hero-meta">
            <span v-for="chip in heroChips" :key="chip" class="hero-chip">{{ chip }}</span>
          </div>
        </div>
        <div class="hero-deco">
          <div class="deco-ring r1" /><div class="deco-ring r2" /><div class="deco-ring r3" />
        </div>
      </header>

      <!-- 1 -->
      <section :id="sections[0].id" class="section">
        <div class="section-anchor">
          <span class="anchor-tag">01</span>
          <h2>平台简介</h2>
        </div>
        <p class="intro-text">代码审核平台是一款可私有化部署的 AI 代码审查工具，将 Git 仓库与大语言模型结合，在代码推送或手动触发时自动执行 Code Review，结果通过 Web 界面实时呈现。</p>

        <div class="feat-grid">
          <div class="feat-card" v-for="f in features" :key="f.title">
            <div class="feat-icon-wrap" v-html="f.icon" />
            <div class="feat-title">{{ f.title }}</div>
            <div class="feat-desc">{{ f.desc }}</div>
          </div>
        </div>

        <div class="callout callout-blue">
          <div class="callout-icon">⚡</div>
          <div>
            <strong>单二进制部署</strong> — 前端资源内嵌，依赖 SQLite，无需额外服务。
            环境变量 <code>APP_ADDR</code>（默认 <code>:18083</code>）控制监听地址，
            <code>DATABASE_DSN</code> 控制数据库路径。
          </div>
        </div>
      </section>

      <!-- 2 -->
      <section :id="sections[1].id" class="section">
        <div class="section-anchor">
          <span class="anchor-tag">02</span>
          <h2>账户与权限</h2>
        </div>

        <h3>初始账户</h3>
        <p>系统首次启动时自动创建超级管理员账户，初始密码请联系部署人员获取。登录后，点击左侧底部用户名，可在「个人资料」弹窗中修改密码、邮箱、手机号和岗位信息。推送通知配置请进入左侧「<strong>推送通知</strong>」菜单单独管理。</p>

        <h3>三级权限体系</h3>
        <div class="perm-table">
          <div class="perm-head">
            <div>角色</div><div>仪表盘 / 项目 / 任务</div><div>凭据 / 定时扫描</div><div>系统配置</div>
          </div>
          <div class="perm-row" v-for="r in roles" :key="r.name">
            <div><span class="role-pill" :class="r.cls">{{ r.name }}</span></div>
            <div>{{ r.c1 }}</div>
            <div>{{ r.c2 }}</div>
            <div>{{ r.c3 }}</div>
          </div>
        </div>
        <div class="callout callout-blue">
          <div class="callout-icon">💡</div>
          <div>凭据和定时扫描采用<strong>归属隔离</strong>：每条记录绑定创建人，普通用户只能看到和操作自己创建的资源；管理员可查看和管理所有人的资源。项目、任务同理。</div>
        </div>
        <div class="callout callout-amber">
          <div class="callout-icon">⚠️</div>
          <div>只有<strong>超级管理员</strong>可以添加、删除管理员账户并提升用户权限。管理员可管理普通用户，但无法修改其他管理员的账户。</div>
        </div>
      </section>

      <!-- 3 -->
      <section :id="sections[2].id" class="section">
        <div class="section-anchor">
          <span class="anchor-tag">03</span>
          <h2>接入 AI 模型</h2>
        </div>
        <p>进入「<strong>模型配置</strong>」菜单，点击「新建模型」，支持两种接入模式：</p>

        <div class="mode-row">
          <div class="mode-card">
            <div class="mode-head api">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
              API 模式
            </div>
            <p>通过 HTTP API 调用云端或本地大模型，支持 7 种平台：</p>
            <div class="provider-chips">
              <span v-for="p in providers" :key="p.name" class="p-chip">
                <span class="p-dot" :style="{background: p.color}"></span>{{ p.name }}
              </span>
            </div>
            <ul class="doc-ul">
              <li><strong>Base URL</strong> — API 服务地址，Ollama 等本地服务填内网地址</li>
              <li><strong>API Key</strong> — 平台密钥，Ollama 可留空</li>
              <li><strong>Model</strong> — 模型名称，如 <code>gpt-4o</code>、<code>claude-opus-4-5</code>、<code>deepseek-coder</code></li>
              <li><strong>Max Context</strong> — 最大上下文 Token 数，默认 32000</li>
              <li><strong>Thinking</strong> — 开启后模型先推理再回答，结果更精准但耗时更长</li>
            </ul>
          </div>
          <div class="mode-card">
            <div class="mode-head cli">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
              CLI 模式
            </div>
            <p>通过本地 <code>claude</code> CLI 工具执行审查，适合已安装 Claude Code 的服务器环境。</p>
            <ul class="doc-ul">
              <li><strong>CLI 路径</strong> — 可执行文件路径，默认 <code>claude</code></li>
              <li><strong>环境变量</strong> — JSON 格式，注入 <code>ANTHROPIC_API_KEY</code> 等变量</li>
              <li><strong>Max Turns</strong> — 对应 <code>--max-turns</code> 参数，可选</li>
            </ul>
            <div class="callout callout-blue" style="margin-top:12px">
              <div class="callout-icon">💡</div>
              <div>CLI 模式下平台自动追加 commit 范围指令，Claude 通过 <code>git diff</code> 自行读取代码变更，Prompt 中无需指定。</div>
            </div>
          </div>
        </div>

        <h3>Global Prompt 编写建议</h3>
        <div class="codeblock">
          <div class="cb-label">推荐 Prompt 结构</div>
          <pre>你是一位资深软件工程师，请对提供的 Git diff 进行 Code Review。
重点关注以下方面：
1. 逻辑错误与边界条件
2. 安全漏洞（SQL 注入、XSS、权限校验缺失等）
3. 性能问题（N+1 查询、不必要的内存分配）
4. 代码规范与可读性
5. 错误处理完整性

输出格式：使用 Markdown，按严重程度分类（🔴 严重 / 🟡 建议 / 🟢 优化）。</pre>
        </div>
      </section>

      <!-- 4 -->
      <section :id="sections[3].id" class="section">
        <div class="section-anchor">
          <span class="anchor-tag">04</span>
          <h2>配置仓库凭证</h2>
        </div>
        <p>私有仓库需要凭证才能 clone/fetch。进入「<strong>仓库凭据</strong>」菜单统一管理，项目可按需选择关联。每位用户只能看到自己创建的凭据，管理员可查看所有人的凭据。</p>

        <div class="steps">
          <div class="step" v-for="(s, i) in credSteps" :key="i">
            <div class="step-num">{{ i + 1 }}</div>
            <div class="step-body">
              <div class="step-title">{{ s.title }}</div>
              <div class="step-desc">{{ s.desc }}</div>
            </div>
          </div>
        </div>

        <div class="info-grid">
          <div class="callout callout-blue">
            <div class="callout-icon">🔐</div>
            <div>凭证通过 HTTPS URL userinfo 方式注入：<br><code>https://user:token@gitlab.com/org/repo.git</code><br>SSH 协议的仓库地址不支持此方式，请改用 HTTPS URL。</div>
          </div>
          <div class="callout callout-amber">
            <div class="callout-icon">⚠️</div>
            <div>被项目引用的凭据<strong>不允许删除</strong>，需先在项目中解除关联。修改凭据密码后，已 clone 的仓库 remote URL 会自动更新。</div>
          </div>
        </div>
      </section>

      <!-- 5 -->
      <section :id="sections[4].id" class="section">
        <div class="section-anchor">
          <span class="anchor-tag">05</span>
          <h2>创建与管理项目</h2>
        </div>
        <p>进入「<strong>项目</strong>」菜单，点击「新建项目」，每个项目对应一个 Git 仓库分支。</p>

        <div class="field-table">
          <div class="ft-row" v-for="f in projectFields" :key="f.name">
            <div class="ft-key">{{ f.name }}</div>
            <div class="ft-val">{{ f.desc }}</div>
          </div>
        </div>

        <h3>Prompt 叠加机制</h3>
        <p>项目的「自定义 Prompt」<strong>不会覆盖</strong>模型全局 Prompt，而是追加在后面，两者共同构成完整指令：</p>
        <div class="codeblock">
          <div class="cb-label">最终 Prompt 构成</div>
          <pre><span class="token-comment"># API 模式 System Message</span>
[模型全局 Prompt]
+ "\n\n"
+ [项目自定义 Prompt]

<span class="token-comment"># CLI 模式（额外自动追加 commit 范围指令）</span>
[模型全局 Prompt]
+ "\n\n"
+ [项目自定义 Prompt]
+ "\n\n请审查此仓库中 {fromCommit} 到 {toCommit} 之间的代码变更。"</pre>
        </div>

        <h3>溢出策略说明</h3>
        <div class="strategy-row">
          <div class="strategy-card">
            <div class="strategy-label queue">queue · 排队等待</div>
            <p>有任务运行时，新任务进入等待队列，按 FIFO 顺序依次执行。<br>适合：不希望遗漏任何提交的场景。</p>
          </div>
          <div class="strategy-card">
            <div class="strategy-label reject">reject · 直接拒绝</div>
            <p>有任务运行时，新触发的任务直接被拒绝，状态标记为 <code>rejected</code>。<br>适合：高频提交、只关注最新版本的场景。</p>
          </div>
        </div>
      </section>

      <!-- 6 -->
      <section :id="sections[5].id" class="section">
        <div class="section-anchor">
          <span class="anchor-tag">06</span>
          <h2>触发 Code Review</h2>
        </div>

        <div class="trigger-row">
          <div class="trigger-card">
            <div class="trigger-icon">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
            </div>
            <div class="trigger-title">手动触发</div>
            <p>进入项目详情页，点击「手动触发审核」按钮，系统对当前分支最新 commit 发起审查。适合临时需要或验证配置时使用。</p>
          </div>
          <div class="trigger-card">
            <div class="trigger-icon">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2A19.5 19.5 0 0 1 3.35 3.6a2 2 0 0 1 1.68-2.18h3a2 2 0 0 1 2 1.72c.13.96.36 1.9.7 2.81a2 2 0 0 1-.45 2.11L7.91 9a16 16 0 0 0 6 6z"/></svg>
            </div>
            <div class="trigger-title">Webhook 自动触发</div>
            <p>在 Git 平台配置 Webhook，代码推送时自动调用以下地址：</p>
            <div class="codeblock" style="margin-top:10px">
              <div class="cb-label">Webhook 地址</div>
              <pre>POST http://your-server:18083/webhook/{projectId}</pre>
            </div>
            <p style="margin-top:10px">可选传入 commit hash（不传则使用分支最新 HEAD）：</p>
            <div class="codeblock">
              <div class="cb-label">请求体（可选）</div>
              <pre>{
  "commit": "abc1234def5678..."
}</pre>
            </div>
          </div>
        </div>

        <h3>自动跳过规则</h3>
        <p>以下情况触发后不创建任务，避免重复审查：</p>
        <div class="skip-list">
          <div class="skip-item" v-for="s in skipRules" :key="s">
            <span class="skip-icon">⊘</span>{{ s }}
          </div>
        </div>
      </section>

      <!-- 7 -->
      <section :id="sections[6].id" class="section">
        <div class="section-anchor">
          <span class="anchor-tag">07</span>
          <h2>查看审核结果</h2>
        </div>

        <h3>任务状态</h3>
        <div class="status-grid">
          <div class="status-pill" v-for="s in taskStatuses" :key="s.name">
            <span class="s-dot" :style="{background: s.color}"></span>
            <span class="s-name">{{ s.name }}</span>
            <span class="s-desc">{{ s.desc }}</span>
          </div>
        </div>

        <h3>任务详情页内容</h3>
        <div class="tab-preview">
          <div class="tab-bar">
            <span class="tab active-tab">Review 结果</span>
            <span class="tab">Diff 内容</span>
            <span class="tab">执行日志</span>
          </div>
          <div class="tab-desc-list">
            <div class="tab-desc-item" v-for="t in tabDescs" :key="t.name">
              <strong>{{ t.name }}</strong><span>{{ t.desc }}</span>
            </div>
          </div>
        </div>

        <div class="callout callout-blue" style="margin-top:18px">
          <div class="callout-icon">⚡</div>
          <div><strong>实时流式展示</strong> — 运行中的任务通过 SSE 长连接推送日志与审查结果，边生成边展示，无需手动刷新。Token 消耗（输入/输出/缓存命中/缓存写入）在任务完成后显示在详情页顶部。</div>
        </div>
      </section>

      <!-- 8 push notify -->
      <section :id="sections[7].id" class="section">
        <div class="section-anchor">
          <span class="anchor-tag">08</span>
          <h2>扫描结果推送通知</h2>
        </div>
        <p>定时扫描完成后，系统可将结果自动推送至多个渠道。进入左侧「<strong>推送通知</strong>」菜单进行配置，每个账户独立管理，互不影响。</p>

        <h3>配置入口</h3>
        <div class="steps">
          <div class="step">
            <div class="step-num">1</div>
            <div class="step-body">
              <div class="step-title">打开推送通知页面</div>
              <div class="step-desc">在左侧「我的配置」分组下点击「推送通知」菜单项进入配置页面。</div>
            </div>
          </div>
          <div class="step">
            <div class="step-num">2</div>
            <div class="step-body">
              <div class="step-title">开启总开关</div>
              <div class="step-desc">页面顶部有全局开关，关闭时所有渠道均不发送通知；开启后按各渠道是否填写配置决定是否发送。</div>
            </div>
          </div>
          <div class="step">
            <div class="step-num">3</div>
            <div class="step-body">
              <div class="step-title">配置所需渠道</div>
              <div class="step-desc">当前支持邮件和企业微信两个渠道，OA 系统通知正在开发中。各渠道卡片右上角实时显示「已配置 / 未配置」状态。</div>
            </div>
          </div>
        </div>

        <h3>支持渠道</h3>
        <div class="mode-row">
          <div class="mode-card">
            <div class="mode-head api">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/><polyline points="22,6 12,13 2,6"/></svg>
              邮件通知
            </div>
            <ul class="doc-ul">
              <li>支持<strong>多个收件地址</strong>，英文逗号分隔</li>
              <li>邮件含 HTML 正文（Markdown 渲染）和 <code>.md</code> 格式附件</li>
              <li>报告头部自动注入：审计完成时间、审计人（账号名）、项目名、Commit 范围</li>
              <li>发送依赖管理员在「设置」页面配置的 SMTP 服务</li>
            </ul>
          </div>
          <div class="mode-card">
            <div class="mode-head cli" style="background:#ecfdf5;color:#059669">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
              企业微信机器人
            </div>
            <ul class="doc-ul">
              <li>在企业微信群中添加「群机器人」获取 Webhook 地址</li>
              <li>扫描完成后自动推送 Markdown 格式摘要</li>
              <li>包含项目名、任务状态、commit 范围及审查结果前 500 字</li>
            </ul>
          </div>
        </div>

        <div class="callout callout-purple" style="margin-top:4px">
          <div class="callout-icon">🖥️</div>
          <div><strong>OA 系统通知</strong> — 正在开发中，页面已预留入口。未来将支持对接企业内部 OA 系统，将审计结果推送至工作流或待办事项。</div>
        </div>

        <h3>邮件报告说明</h3>
        <div class="callout callout-blue">
          <div class="callout-icon">📧</div>
          <div>
            邮件支持<strong>多个收件地址</strong>，使用英文逗号分隔，例如：<br>
            <code>lead@company.com, dev1@company.com, dev2@company.com</code><br>
            邮件头部由系统自动生成，包含准确的<strong>审计完成时间</strong>和<strong>审计人账号名</strong>，不依赖 AI 输出。
          </div>
        </div>

        <h3>管理员 SMTP 配置</h3>
        <p>进入「<strong>设置</strong>」菜单，在 SMTP 配置区块填写邮件服务参数（仅管理员可配置）：</p>
        <div class="field-table">
          <div class="ft-row" v-for="f in smtpFields" :key="f.name">
            <div class="ft-key">{{ f.name }}</div>
            <div class="ft-val">{{ f.desc }}</div>
          </div>
        </div>
      </section>

      <!-- 9 -->
      <section :id="sections[8].id" class="section">
        <div class="section-anchor">
          <span class="anchor-tag">09</span>
          <h2>敏感词拦截</h2>
        </div>
        <p>进入「<strong>敏感词管理</strong>」配置需要屏蔽的关键词列表。触发审查时，系统在调用 LLM 之前扫描 diff 内容：</p>

        <div class="flow-inline">
          <div class="fi-step" v-for="(s, i) in sensitiveFlow" :key="i">
            <div class="fi-box">{{ s }}</div>
            <div class="fi-arrow" v-if="i < sensitiveFlow.length - 1">→</div>
          </div>
        </div>

        <div class="callout callout-purple">
          <div class="callout-icon">🛡️</div>
          <div>适合防止内部密钥、手机号、身份证号、数据库连接串等敏感数据通过代码审查流程流出到第三方 AI 服务。命中后任务状态标记为 <code>rejected</code>，不计入失败，可查看拒绝原因。</div>
        </div>
      </section>

      <!-- 10 -->
      <section :id="sections[9].id" class="section">
        <div class="section-anchor">
          <span class="anchor-tag">10</span>
          <h2>全局设置</h2>
        </div>
        <p>进入「<strong>设置</strong>」菜单配置系统级参数，对所有项目生效（项目级配置优先级更高）。SMTP 邮件服务配置也在此页面（仅管理员可见）。</p>

        <div class="field-table">
          <div class="ft-head">
            <div>配置项</div><div>默认值</div><div>说明</div>
          </div>
          <div class="ft-row3" v-for="s in globalSettings" :key="s.name">
            <div class="ft-key">{{ s.name }}</div>
            <div><code>{{ s.default }}</code></div>
            <div class="ft-val3">{{ s.desc }}</div>
          </div>
        </div>
      </section>

      <!-- 11 -->
      <section :id="sections[10].id" class="section last-section">
        <div class="section-anchor">
          <span class="anchor-tag">11</span>
          <h2>工作原理</h2>
        </div>

        <h3>完整审查流程</h3>
        <div class="pipeline">
          <div class="pipe-step" v-for="(s, i) in pipeline" :key="i">
            <div class="pipe-icon" v-html="s.icon" />
            <div class="pipe-label">{{ s.label }}</div>
            <div class="pipe-connector" v-if="i < pipeline.length - 1" />
          </div>
        </div>

        <h3>调度器机制</h3>
        <p>后台调度器每 5 秒轮询一次 <code>pending</code> 任务，按创建时间 <strong>FIFO</strong> 排序。使用信号量（semaphore）控制全局并发数，每个任务运行在独立 goroutine 中，支持超时保护和手动取消（context cancel）。</p>

        <h3>仓库本地管理</h3>
        <p>首次触发时 clone 到服务器本地（<code>./repos/{projectId}/</code>），后续触发仅执行 <code>git fetch</code>，无需重复 clone。</p>

        <div class="codeblock">
          <div class="cb-label">Git 操作逻辑</div>
          <pre><span class="token-comment"># 首次</span>
git clone --branch {branch} {url} ./repos/{projectId}/

<span class="token-comment"># 后续</span>
git fetch origin {branch}
toCommit = git rev-parse origin/{branch}

<span class="token-comment"># 生成 diff</span>
git show {toCommit}              <span class="token-comment"># 首次（fromCommit 为空）</span>
git diff {fromCommit}..{toCommit} <span class="token-comment"># 后续</span></pre>
        </div>

        <h3>任务超时优先级</h3>
        <div class="priority-row">
          <div class="priority-item" v-for="(p, i) in timeoutPriority" :key="i">
            <div class="priority-badge">P{{ i + 1 }}</div>
            <div>{{ p }}</div>
          </div>
        </div>

        <div class="callout callout-blue" style="margin-top:28px">
          <div class="callout-icon">📖</div>
          <div>如有更多问题或功能建议，请查阅项目 <code>docs/</code> 目录下的详细技术文档，或联系平台管理员。</div>
        </div>
      </section>

    </main>

    <!-- Right TOC -->
    <aside class="toc-panel">
      <div class="toc-label">本页目录</div>
      <div class="toc-track">
        <div class="toc-track-line" />
        <a v-for="s in sections" :key="s.id"
          class="toc-entry"
          :class="{ active: activeSection === s.id }"
          @click.prevent="scrollTo(s.id)">
          <span class="toc-dot" />
          <span class="toc-text">{{ s.label }}</span>
        </a>
      </div>
    </aside>

  </div>
</template>

<script setup>
import { ref } from 'vue'

const mainRef = ref(null)
const activeSection = ref('intro')

const sections = [
  { id: 'intro',       label: '平台简介' },
  { id: 'account',    label: '账户与权限' },
  { id: 'model',      label: '接入 AI 模型' },
  { id: 'credential', label: '配置仓库凭证' },
  { id: 'project',    label: '创建与管理项目' },
  { id: 'trigger',    label: '触发 Code Review' },
  { id: 'result',     label: '查看审核结果' },
  { id: 'notify',     label: '推送通知' },
  { id: 'sensitive',  label: '敏感词拦截' },
  { id: 'settings',   label: '全局设置' },
  { id: 'principle',  label: '工作原理' },
]

const heroChips = ['自托管', 'SQLite', 'Webhook', '多模型', '流式输出', '推送通知']

const features = [
  { title: 'Webhook 自动触发', desc: '代码推送即发起审查，零人工介入', icon: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2A19.5 19.5 0 0 1 3.35 3.6a2 2 0 0 1 1.68-2.18h3a2 2 0 0 1 2 1.72c.13.96.36 1.9.7 2.81a2 2 0 0 1-.45 2.11L7.91 9a16 16 0 0 0 6 6z"/></svg>' },
  { title: '多模型支持', desc: '7 种平台 API + Claude CLI 双模式', icon: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/></svg>' },
  { title: '流式实时展示', desc: '审查结果边生成边展示，SSE 推送', icon: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>' },
  { title: '敏感词拦截', desc: '扫描 diff，防止敏感数据流出', icon: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>' },
  { title: '任务调度控制', desc: '并发限制、超时保护、取消重试', icon: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>' },
  { title: '私有化部署', desc: '单二进制，数据完全自主，无外部依赖', icon: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>' },
]

const roles = [
  { name: '超级管理员', cls: 'super', c1: '✅ 所有用户数据', c2: '✅ 所有用户资源', c3: '✅ 全部' },
  { name: '管理员',     cls: 'admin', c1: '✅ 所有用户数据', c2: '✅ 所有用户资源', c3: '✅ 全部' },
  { name: '普通用户',   cls: 'user',  c1: '✅ 仅自己的数据', c2: '✅ 仅自己创建的', c3: '❌' },
]

const providers = [
  { name: 'OpenAI',           color: '#10a37f' },
  { name: 'Anthropic',        color: '#c96442' },
  { name: 'DeepSeek',         color: '#4d6bfe' },
  { name: 'Google Gemini',    color: '#4285f4' },
  { name: 'Ollama（本地）',   color: '#6b7280' },
  { name: 'Mistral',          color: '#fa520f' },
  { name: 'Claude CLI',       color: '#c96442' },
]

const smtpFields = [
  { name: 'SMTP 主机',      desc: '邮件服务器地址，如 smtp.exmail.qq.com、smtp.gmail.com。' },
  { name: 'SMTP 端口',      desc: '通常：TLS 加密用 465，STARTTLS 用 587，明文用 25。' },
  { name: '加密方式',        desc: 'TLS（推荐）或 None，对应端口需匹配。' },
  { name: '发件人账户',      desc: 'SMTP 登录用户名，通常与发件邮箱地址相同。' },
  { name: '发件人密码',      desc: 'SMTP 密码或应用专用密码（企业邮箱/Gmail 推荐使用应用密码）。修改时留空表示保留原密码。' },
  { name: '发件人地址',      desc: '邮件 From 地址，如 review@company.com。' },
  { name: '发件人显示名称',  desc: '邮件客户端中"发件人"栏显示的名称，如"代码审计"，留空则只显示邮箱地址。' },
]

const credSteps = [
  { title: '新建凭据', desc: '进入「仓库凭据」菜单，填写凭据名称（便于区分）、Git 用户名、密码或 Personal Access Token（推荐使用 Token，权限范围更可控）。' },
  { title: '关联到项目', desc: '新建或编辑项目时，在「仓库凭据」下拉框中选择对应凭据。公开仓库可留空（无需凭据）。' },
]

const projectFields = [
  { name: '项目名称',   desc: '用于列表展示和任务关联，建议与仓库名保持一致，便于识别。' },
  { name: '仓库 URL',  desc: 'HTTPS 格式的 Git 仓库地址，如 https://github.com/org/repo.git。SSH 协议不支持凭证注入，请使用 HTTPS。' },
  { name: '分支',       desc: '需要持续审查的目标分支，如 main、master、develop 等。' },
  { name: '模型配置',   desc: '选择已创建的模型配置，决定使用哪个 AI 模型进行审查。每个项目独立绑定。' },
  { name: '仓库凭据',   desc: '私有仓库选择对应凭据；公开仓库留空。' },
  { name: '溢出策略',   desc: '有任务执行中时，新任务的处理方式：queue（排队）或 reject（拒绝）。可覆盖全局默认值。' },
  { name: '任务超时',   desc: '单次审查的最长执行时间（分钟）。留空则继承全局设置（默认 30 分钟）。' },
  { name: '自定义 Prompt', desc: '追加在模型全局 Prompt 之后的项目专属指令，可描述技术栈、编码规范、重点关注方向等。' },
]

const skipRules = [
  '当前 commit 与上次审查 commit 相同，无新提交',
  '相同 commit 范围已存在 completed 状态的任务',
  'git diff 获取到的内容为空',
]

const taskStatuses = [
  { name: 'pending',   color: '#f59e0b', desc: '等待调度' },
  { name: 'running',   color: '#3b82f6', desc: '审查中' },
  { name: 'completed', color: '#10b981', desc: '已完成' },
  { name: 'failed',    color: '#ef4444', desc: '执行失败，可重试' },
  { name: 'cancelled', color: '#6b7280', desc: '已手动取消' },
  { name: 'rejected',  color: '#8b5cf6', desc: '被拦截（溢出/敏感词）' },
]

const tabDescs = [
  { name: 'Review 结果', desc: ' — AI 审查报告，Markdown 渲染展示。运行中实时流式更新，完成后固定。' },
  { name: 'Diff 内容',   desc: ' — 本次审查的完整 git diff，格式化展示，可核对审查范围。' },
  { name: '执行日志',    desc: ' — 任务全生命周期日志：仓库同步、diff 获取、LLM 调用、完成/失败/超时，含 info / warn / error 三级。' },
]

const sensitiveFlow = ['Webhook / 手动触发', '获取 git diff', '扫描敏感词', '命中 → rejected', '未命中 → 调用 LLM', '结果写入 DB']

const globalSettings = [
  { name: '最大并发任务数', default: '3',       desc: '全局同时执行的最大审查任务数，超出后按项目溢出策略处理。' },
  { name: '全局溢出策略',   default: 'queue',   desc: '项目未配置溢出策略时的默认行为，可被项目级配置覆盖。' },
  { name: '任务超时时间',   default: '30 分钟', desc: '单次审查最长执行时间，项目级超时配置优先级更高。' },
  { name: '仓库根目录',     default: './repos', desc: '克隆仓库的本地存储路径，修改后已有仓库不会自动迁移。' },
]

const pipeline = [
  { label: '触发请求', icon: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 16.92v3a2 2 0 0 1-2.18 2A19.5 19.5 0 0 1 3.35 3.6a2 2 0 0 1 1.68-2.18h3a2 2 0 0 1 2 1.72c.13.96.36 1.9.7 2.81a2 2 0 0 1-.45 2.11L7.91 9a16 16 0 0 0 6 6z"/></svg>' },
  { label: '触发保护检查', icon: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>' },
  { label: '创建 pending 任务', icon: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>' },
  { label: '调度器拉取', icon: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>' },
  { label: 'git fetch + diff', icon: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>' },
  { label: '敏感词扫描', icon: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>' },
  { label: 'LLM / CLI 审查', icon: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4"/></svg>' },
  { label: 'completed', icon: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>' },
]

const timeoutPriority = [
  '项目级「任务超时」配置（最高优先级）',
  '全局设置中的「任务超时时间」',
  '系统默认值：30 分钟',
]

function scrollTo(id) {
  const el = document.getElementById(id)
  if (el && mainRef.value) {
    mainRef.value.scrollTo({ top: el.offsetTop - 24, behavior: 'smooth' })
  }
}

function onScroll() {
  if (!mainRef.value) return
  const top = mainRef.value.scrollTop + 80
  let cur = sections[0].id
  for (const s of sections) {
    const el = document.getElementById(s.id)
    if (el && el.offsetTop <= top) cur = s.id
  }
  activeSection.value = cur
}
</script>

<style scoped>
/* ── Root ── */
.docs-root {
  display: flex;
  height: 100%;
  overflow: hidden;
  background: #f8fafc;
  font-size: 14px;
  color: #374151;
  line-height: 1.7;
}

/* ── Main ── */
.docs-main {
  flex: 1;
  overflow-y: auto;
  scroll-behavior: smooth;
}

/* ── TOC Panel (right) ── */
.toc-panel {
  width: 188px;
  flex-shrink: 0;
  background: transparent;
  border-left: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 24px 0 24px 0;
}
.toc-label {
  font-size: 11px;
  font-weight: 700;
  color: #9ca3af;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  padding: 0 16px 12px;
}

.toc-track {
  flex: 1;
  overflow-y: auto;
  padding: 0 0 0 24px;
  position: relative;
}
.toc-track-line {
  position: absolute;
  left: 15px; top: 0; bottom: 0;
  width: 1px;
  background: #e5e7eb;
}
.toc-entry {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 5px 12px 5px 0;
  cursor: pointer;
  text-decoration: none;
  position: relative;
  transition: color 0.15s;
  color: #6b7280;
  font-size: 12px;
}
.toc-entry:hover { color: #1e293b; }
.toc-entry.active { color: #2563eb; font-weight: 600; }

.toc-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  border: 2px solid #d1d5db;
  flex-shrink: 0;
  background: #f8fafc;
  transition: all 0.15s;
  margin-left: -3px;
  position: relative;
  z-index: 1;
}
.toc-entry.active .toc-dot {
  border-color: #2563eb;
  background: #2563eb;
  box-shadow: 0 0 0 3px rgba(37,99,235,0.15);
}
.toc-text { flex: 1; }

/* ── Hero ── */
.hero {
  position: relative;
  background: linear-gradient(135deg, #0f172a 0%, #1e3a8a 40%, #2563eb 70%, #7c3aed 100%);
  padding: 52px 56px 48px;
  overflow: hidden;
  color: #fff;
}
.hero-inner { position: relative; z-index: 1; max-width: 640px; }
.hero-eyebrow {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: rgba(255,255,255,0.6);
  margin-bottom: 16px;
}
.eyebrow-dot { width: 6px; height: 6px; border-radius: 50%; background: #60a5fa; }
.hero h1 {
  font-size: 34px;
  font-weight: 800;
  letter-spacing: -1px;
  margin: 0 0 14px;
  line-height: 1.15;
}
.hero p {
  font-size: 15px;
  color: rgba(255,255,255,0.72);
  margin: 0 0 22px;
  line-height: 1.7;
}
.hero-meta { display: flex; flex-wrap: wrap; gap: 8px; }
.hero-chip {
  background: rgba(255,255,255,0.12);
  border: 1px solid rgba(255,255,255,0.2);
  color: rgba(255,255,255,0.85);
  font-size: 11.5px;
  font-weight: 500;
  padding: 3px 12px;
  border-radius: 20px;
}
.hero-deco { position: absolute; right: -60px; top: -60px; pointer-events: none; }
.deco-ring {
  position: absolute;
  border-radius: 50%;
  border: 1px solid rgba(255,255,255,0.08);
}
.r1 { width: 320px; height: 320px; top: -40px; right: -40px; }
.r2 { width: 240px; height: 240px; top: 0; right: 0; }
.r3 { width: 160px; height: 160px; top: 40px; right: 40px; }

/* ── Sections ── */
.section {
  padding: 44px 56px;
  border-bottom: 1px solid #f1f5f9;
  background: #fff;
  margin-bottom: 8px;
}
.last-section { border-bottom: none; margin-bottom: 0; }

.section-anchor {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 20px;
}
.anchor-tag {
  font-size: 10.5px;
  font-weight: 800;
  letter-spacing: 0.05em;
  color: #fff;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  padding: 3px 8px;
  border-radius: 6px;
  flex-shrink: 0;
}
.section-anchor h2 {
  font-size: 21px;
  font-weight: 700;
  color: #0f172a;
  margin: 0;
  letter-spacing: -0.3px;
}
h3 {
  font-size: 14px;
  font-weight: 700;
  color: #1e293b;
  margin: 28px 0 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f1f5f9;
}
p { margin: 0 0 12px; color: #475569; }
.intro-text { font-size: 15px; line-height: 1.8; color: #374151; margin-bottom: 24px; }

/* ── Feature grid ── */
.feat-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin-bottom: 20px;
}
.feat-card {
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 16px;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.feat-card:hover { border-color: #bfdbfe; box-shadow: 0 2px 8px rgba(37,99,235,0.07); }
.feat-icon-wrap {
  width: 32px; height: 32px;
  background: linear-gradient(135deg, rgba(37,99,235,0.1), rgba(124,58,237,0.1));
  border-radius: 8px;
  display: flex; align-items: center; justify-content: center;
  color: #6366f1;
  margin-bottom: 10px;
}
.feat-icon-wrap :deep(svg) { width: 16px; height: 16px; }
.feat-title { font-size: 13px; font-weight: 600; color: #1e293b; margin-bottom: 4px; }
.feat-desc { font-size: 12px; color: #94a3b8; line-height: 1.5; }

/* ── Callout boxes ── */
.callout {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  border-radius: 10px;
  padding: 14px 16px;
  font-size: 13.5px;
  line-height: 1.65;
  margin: 16px 0;
}
.callout-icon { font-size: 16px; flex-shrink: 0; margin-top: 1px; }
.callout-blue   { background: #eff6ff; border: 1px solid #bfdbfe; color: #1e40af; }
.callout-amber  { background: #fffbeb; border: 1px solid #fde68a; color: #92400e; }
.callout-purple { background: #faf5ff; border: 1px solid #e9d5ff; color: #6b21a8; }
.callout strong { font-weight: 600; }

/* ── Permission table ── */
.perm-table { border: 1px solid #e5e7eb; border-radius: 10px; overflow: hidden; margin: 14px 0; }
.perm-head, .perm-row {
  display: grid;
  grid-template-columns: 140px 1fr 1fr 1.4fr;
}
.perm-head {
  background: #f8fafc;
  font-size: 11.5px;
  font-weight: 600;
  color: #6b7280;
  letter-spacing: 0.02em;
}
.perm-head > div, .perm-row > div {
  padding: 11px 16px;
  border-bottom: 1px solid #f3f4f6;
  font-size: 13px;
}
.perm-row:last-child > div { border-bottom: none; }
.role-pill {
  display: inline-block;
  font-size: 11.5px;
  font-weight: 600;
  padding: 2px 10px;
  border-radius: 20px;
}
.role-pill.super { background: #fee2e2; color: #dc2626; }
.role-pill.admin { background: #fef3c7; color: #b45309; }
.role-pill.user  { background: #f0f9ff; color: #0369a1; }

/* ── Mode cards ── */
.mode-row { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; margin: 14px 0; }
.mode-card {
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 20px;
}
.mode-head {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 13px;
  font-weight: 700;
  border-radius: 6px;
  padding: 6px 12px;
  margin-bottom: 14px;
  width: fit-content;
}
.mode-head.api { background: #eff6ff; color: #2563eb; }
.mode-head.cli { background: #f0fdf4; color: #16a34a; }

.provider-chips { display: flex; flex-wrap: wrap; gap: 6px; margin: 10px 0; }
.p-chip {
  display: flex;
  align-items: center;
  gap: 5px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 3px 10px;
  font-size: 12px;
  color: #374151;
  font-weight: 500;
}
.p-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }

/* ── Code blocks ── */
.codeblock {
  background: #0f172a;
  border-radius: 10px;
  overflow: hidden;
  margin: 12px 0;
  box-shadow: 0 4px 12px rgba(0,0,0,0.12);
}
.cb-label {
  background: rgba(255,255,255,0.05);
  border-bottom: 1px solid rgba(255,255,255,0.06);
  padding: 8px 16px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.06em;
  color: #64748b;
  text-transform: uppercase;
}
.codeblock pre {
  margin: 0;
  padding: 16px;
  font-size: 12.5px;
  color: #e2e8f0;
  font-family: 'SF Mono', 'Fira Code', Cascadia Code, Consolas, monospace;
  line-height: 1.75;
  white-space: pre-wrap;
  word-break: break-all;
}
.token-comment { color: #64748b; }

code {
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  padding: 1px 6px;
  font-size: 12.5px;
  color: #7c3aed;
  font-family: 'SF Mono', Consolas, monospace;
}

/* ── Doc list ── */
.doc-ul {
  margin: 10px 0 0;
  padding-left: 18px;
  color: #475569;
  font-size: 13.5px;
  line-height: 1.85;
}
.doc-ul li::marker { color: #6366f1; }

/* ── Steps ── */
.steps { margin: 16px 0; display: flex; flex-direction: column; gap: 0; }
.step {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  padding: 16px 0;
  border-bottom: 1px dashed #f1f5f9;
}
.step:last-child { border-bottom: none; }
.step-num {
  width: 28px; height: 28px;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  color: #fff;
  font-size: 13px;
  font-weight: 700;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0; margin-top: 2px;
}
.step-title { font-size: 14px; font-weight: 600; color: #1e293b; margin-bottom: 5px; }
.step-desc { font-size: 13.5px; color: #6b7280; line-height: 1.65; }

/* ── Info grid ── */
.info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-top: 4px; }

/* ── Field table ── */
.field-table {
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  overflow: hidden;
  margin: 14px 0;
}
.ft-head {
  display: grid;
  grid-template-columns: 140px 100px 1fr;
  background: #f8fafc;
  font-size: 11.5px;
  font-weight: 600;
  color: #6b7280;
  letter-spacing: 0.02em;
}
.ft-head > div { padding: 10px 14px; border-bottom: 1px solid #e5e7eb; }
.ft-row {
  display: flex;
  border-bottom: 1px solid #f3f4f6;
}
.ft-row:last-child { border-bottom: none; }
.ft-row3 {
  display: grid;
  grid-template-columns: 140px 100px 1fr;
  border-bottom: 1px solid #f3f4f6;
  font-size: 13px;
}
.ft-row3:last-child { border-bottom: none; }
.ft-row3 > div { padding: 11px 14px; }
.ft-key {
  padding: 11px 14px;
  font-size: 13px;
  font-weight: 600;
  color: #374151;
  background: #fafafa;
  border-right: 1px solid #f3f4f6;
  width: 140px;
  flex-shrink: 0;
}
.ft-val { padding: 11px 14px; font-size: 13px; color: #6b7280; line-height: 1.6; flex: 1; }
.ft-val3 { color: #6b7280; line-height: 1.6; }

/* ── Strategy cards ── */
.strategy-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin: 12px 0; }
.strategy-card {
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 16px;
}
.strategy-label {
  font-size: 13px;
  font-weight: 700;
  padding: 4px 12px;
  border-radius: 6px;
  margin-bottom: 10px;
  width: fit-content;
}
.strategy-label.queue { background: #eff6ff; color: #2563eb; }
.strategy-label.reject { background: #fef2f2; color: #dc2626; }

/* ── Trigger cards ── */
.trigger-row { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; margin: 14px 0; }
.trigger-card {
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 20px;
}
.trigger-icon {
  width: 36px; height: 36px;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  color: #fff;
  margin-bottom: 12px;
}
.trigger-title { font-size: 14px; font-weight: 700; color: #1e293b; margin-bottom: 8px; }

/* ── Skip rules ── */
.skip-list { display: flex; flex-direction: column; gap: 8px; margin: 12px 0; }
.skip-item {
  display: flex;
  align-items: center;
  gap: 10px;
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 13.5px;
  color: #475569;
}
.skip-icon { color: #94a3b8; font-size: 15px; flex-shrink: 0; }

/* ── Status grid ── */
.status-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; margin: 12px 0; }
.status-pill {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 13px;
}
.s-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.s-name { font-weight: 600; color: #1e293b; }
.s-desc { color: #9ca3af; font-size: 12px; }

/* ── Tab preview ── */
.tab-preview {
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  overflow: hidden;
  margin: 12px 0;
}
.tab-bar {
  display: flex;
  gap: 0;
  border-bottom: 1px solid #e5e7eb;
  background: #fff;
  padding: 0 14px;
}
.tab {
  padding: 10px 16px;
  font-size: 13px;
  color: #9ca3af;
  cursor: default;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}
.active-tab { color: #2563eb; font-weight: 600; border-bottom-color: #2563eb; }
.tab-desc-list { padding: 12px 16px; display: flex; flex-direction: column; gap: 8px; }
.tab-desc-item { font-size: 13px; color: #475569; line-height: 1.65; }
.tab-desc-item strong { color: #1e293b; }

/* ── Sensitive flow ── */
.flow-inline { display: flex; align-items: center; flex-wrap: wrap; gap: 0; margin: 16px 0; }
.fi-step { display: flex; align-items: center; }
.fi-box {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 8px 14px;
  font-size: 13px;
  color: #374151;
  font-weight: 500;
  white-space: nowrap;
}
.fi-arrow { color: #d1d5db; font-size: 16px; padding: 0 6px; }

/* ── Pipeline ── */
.pipeline {
  display: flex;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 0;
  margin: 14px 0 24px;
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 20px 24px;
}
.pipe-step {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  flex: 1;
  min-width: 80px;
}
.pipe-icon {
  width: 38px; height: 38px;
  background: linear-gradient(135deg, rgba(37,99,235,0.1), rgba(124,58,237,0.1));
  border: 1px solid rgba(99,102,241,0.2);
  border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  color: #6366f1;
  margin-bottom: 8px;
}
.pipe-icon :deep(svg) { width: 16px; height: 16px; }
.pipe-label {
  font-size: 11.5px;
  color: #6b7280;
  font-weight: 500;
  text-align: center;
  line-height: 1.4;
}
.pipe-connector {
  position: absolute;
  top: 19px;
  right: -50%;
  width: 100%;
  height: 1px;
  background: linear-gradient(90deg, #e5e7eb, #c7d2fe);
  z-index: 0;
}

/* ── Priority ── */
.priority-row { display: flex; gap: 10px; margin: 12px 0; }
.priority-item {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 12px 14px;
  font-size: 13px;
  color: #475569;
}
.priority-badge {
  width: 24px; height: 24px;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
</style>
