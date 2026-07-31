<template>
  <div class="page-wrap">
    <!-- 面包屑 -->
    <el-breadcrumb separator=">" class="breadcrumb">
      <el-breadcrumb-item :to="{ path: '/tasks' }">任务</el-breadcrumb-item>
      <el-breadcrumb-item>#{{ route.params.id }}</el-breadcrumb-item>
    </el-breadcrumb>

    <template v-if="task">
      <!-- 头部 hero -->
      <div class="page-hero">
        <div class="hero-content">
          <h2 class="hero-title">任务 #{{ task.id }}</h2>
          <p class="hero-sub">{{ projectName }} · {{ task.triggered_by }}</p>
        </div>
        <div class="hero-right">
          <span class="status-pill" :class="`pill-${task.status}`">{{ statusLabel(task.status) }}</span>
          <!-- action buttons in hero -->
          <div class="hero-actions">
            <el-button v-if="task.status === 'running'" size="small" type="warning" plain @click="handleCancel">取消任务</el-button>
            <el-button v-else-if="task.status === 'failed' || task.status === 'cancelled'" size="small" type="primary" plain @click="handleRetry">重试</el-button>
            <el-button v-if="task.status === 'completed'" size="small" type="primary" plain @click="handleRescan">再次扫描</el-button>
            <el-button v-if="!['running','pending'].includes(task.status)" size="small" type="danger" plain @click="handleDelete">删除</el-button>
          </div>
        </div>
        <div class="deco-circles">
          <div class="deco c1"></div>
          <div class="deco c2"></div>
        </div>
      </div>

      <!-- 信息摘要 -->
      <div class="summary-row">
        <div class="summary-card"><span>Commit 范围</span><strong>{{ commitRange }}</strong></div>
        <div class="summary-card"><span>输入 Token</span><strong>{{ task.input_tokens ?? '—' }}</strong></div>
        <div class="summary-card"><span>输出 Token</span><strong>{{ task.output_tokens ?? '—' }}</strong></div>
        <div class="summary-card">
          <span>耗时</span>
          <strong>
            <span v-if="task.finished_at || !isActive">—</span>
            <span v-else class="elapsed-timer">{{ elapsedText }}</span>
          </strong>
        </div>
      </div>

      <!-- 两列主体布局 -->
      <div class="main-columns">
        <!-- 左列：Review 结果 + diff/commits tabs -->
        <div class="col-main">
          <el-tabs v-model="activeTab" class="result-tabs">
            <el-tab-pane label="Review 结果" name="review-result">
              <div class="result-toolbar">
                <el-button text :icon="FullScreen" size="small" @click="fullscreen = true">全屏</el-button>
              </div>
              <el-scrollbar class="result-scrollbar fill-scrollbar" always>
                <div v-if="task.result" class="markdown-body" v-html="renderedResult" />
                <div v-else class="tab-empty">暂无结果</div>
              </el-scrollbar>
            </el-tab-pane>

            <el-tab-pane label="变更文件" name="diff">
              <div v-if="diffFiles.length > 1" class="diff-nav">
                <span
                  v-for="f in diffFiles"
                  :key="f.lineIdx"
                  class="diff-nav-item"
                  @click="scrollToDiffFile(f.lineIdx)"
                >{{ f.name }}</span>
              </div>
              <el-scrollbar ref="diffScrollbar" class="fill-scrollbar" always>
                <pre class="diff-block"><template v-for="(line, idx) in diffLines" :key="idx"><span :class="line.cls" :data-line="idx">{{ line.text }}</span>
</template></pre>
              </el-scrollbar>
            </el-tab-pane>

            <el-tab-pane v-if="task.commit_messages" label="提交记录" name="commits">
              <el-scrollbar class="fill-scrollbar" always>
                <pre class="commit-block">{{ task.commit_messages }}</pre>
              </el-scrollbar>
            </el-tab-pane>
          </el-tabs>

          <!-- commit info bar -->
          <div v-if="task.to_subject || task.from_subject" class="commit-info-bar">
            <span class="mono">{{ commitRange }}</span>
            <span class="commit-subjects">
              <template v-if="task.from_commit">{{ task.from_subject || '—' }} <span class="commit-arrow">→</span> </template>{{ task.to_subject || '—' }}
            </span>
          </div>
        </div>

        <!-- 右列：执行日志 sticky panel -->
        <div class="col-log">
          <div class="log-panel">
            <div class="log-panel-header">
              <span class="log-panel-title">执行日志</span>
              <div class="log-panel-actions">
                <span v-if="isActive" class="live-dot"><span class="blink-dot"></span>实时</span>
                <el-button text :icon="FullScreen" size="small" @click="logFullscreen = true">全屏</el-button>
              </div>
            </div>
            <el-scrollbar ref="logScrollbar" class="log-scrollbar" always>
              <div class="log-list" ref="logList">
                <template v-if="logs.length">
                  <div v-for="log in logs" :key="log.id" class="log-line" :class="`log-line-${log.level}`">
                    <span class="log-time" :title="log.created_at">{{ log.created_at.slice(11) }}</span>
                    <span class="log-badge" :class="`log-badge-${log.level}`">{{ log.level }}</span>
                    <span class="log-msg">{{ log.message }}</span>
                  </div>
                </template>
                <div v-else class="log-empty">暂无日志</div>
              </div>
            </el-scrollbar>
            <div class="log-panel-footer">
              <span class="log-count">{{ logs.length }} 条</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 全屏 Dialog -->
      <el-dialog v-model="fullscreen" title="Review 结果" fullscreen>
        <el-scrollbar>
          <div class="markdown-body markdown-body-dialog" v-html="renderedResult" />
        </el-scrollbar>
      </el-dialog>

      <!-- 日志全屏 Dialog -->
      <el-dialog v-model="logFullscreen" title="执行日志" fullscreen>
        <el-scrollbar ref="logFullscreenScrollbar" class="log-fullscreen-scrollbar" always>
          <div class="log-list log-list-dialog">
            <template v-if="logs.length">
              <div v-for="log in logs" :key="log.id" class="log-line" :class="`log-line-${log.level}`">
                <span class="log-time" :title="log.created_at">{{ log.created_at.slice(11, 19) }}</span>
                <span class="log-badge" :class="`log-badge-${log.level}`">{{ log.level }}</span>
                <span class="log-msg">{{ log.message }}</span>
              </div>
            </template>
            <div v-else class="log-empty">暂无日志</div>
          </div>
        </el-scrollbar>
      </el-dialog>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { FullScreen } from '@element-plus/icons-vue'
import { marked } from 'marked'
import { getTask, cancelTask as apiCancelTask, retryTask as apiRetryTask, deleteTask as apiDeleteTask } from '../../api/tasks'

const route = useRoute()
const router = useRouter()

const task = ref(null)
const projectName = ref('')
const logs = ref([])
const activeTab = ref('review-result')
const fullscreen = ref(false)
const logFullscreen = ref(false)
const logScrollbar = ref(null)
const logFullscreenScrollbar = ref(null)

const scrollElementToBottom = (el) => {
  if (!el) return
  const wrap = el.wrapRef || el.$el?.querySelector?.('.el-scrollbar__wrap') || el.querySelector?.('.el-scrollbar__wrap')
  if (!wrap) {
    el.setScrollTop?.(99999999)
    return
  }
  wrap.scrollTop = wrap.scrollHeight
}
const scrollLogsToBottom = () => {
  const doScroll = () => {
    scrollElementToBottom(logScrollbar.value)
    if (logFullscreen.value) scrollElementToBottom(logFullscreenScrollbar.value)
  }
  nextTick(() => {
    doScroll()
    requestAnimationFrame(doScroll)
    setTimeout(doScroll, 80)
  })
}
watch(() => logs.value.length, scrollLogsToBottom, { flush: 'post' })
watch(logFullscreen, (open) => { if (open) scrollLogsToBottom() })

const isActive = computed(() => ['running', 'pending'].includes(task.value?.status))

const elapsedSeconds = ref(0)
let elapsedTimer = null
const elapsedText = computed(() => {
  const s = elapsedSeconds.value
  if (s < 60) return `${s}s`
  return `${Math.floor(s/60)}m ${s%60}s`
})
watch(isActive, (active) => {
  if (active) {
    elapsedSeconds.value = 0
    elapsedTimer = setInterval(() => elapsedSeconds.value++, 1000)
  } else {
    clearInterval(elapsedTimer)
    elapsedTimer = null
  }
}, { immediate: true })

const renderedResult = computed(() => {
  if (!task.value?.result) return ''
  return marked.parse(task.value.result)
})

const commitRange = computed(() => {
  if (!task.value) return ''
  if (task.value.from_commit) return `${task.value.from_commit.slice(0,7)}..${task.value.to_commit.slice(0,7)}`
  return task.value.to_commit?.slice(0, 7)
})

const diffLines = computed(() => {
  if (!task.value?.diff_content) return []
  return task.value.diff_content.split('\n').map(line => {
    if (line.startsWith('+')) return { text: line, cls: 'diff-add' }
    if (line.startsWith('-')) return { text: line, cls: 'diff-del' }
    return { text: line, cls: '' }
  })
})

const diffFiles = computed(() => {
  const files = []
  diffLines.value.forEach((line, idx) => {
    if (line.text.startsWith('diff --git ')) {
      const m = line.text.match(/diff --git a\/.+ b\/(.+)/)
      files.push({ name: m ? m[1] : line.text.slice(11), lineIdx: idx })
    }
  })
  return files
})

const diffScrollbar = ref(null)
const scrollToDiffFile = (lineIdx) => {
  diffScrollbar.value?.setScrollTop(lineIdx * 20)
}

const statusLabel = (status) => {
  const map = { completed: '已完成', running: '运行中', pending: '等待中', failed: '失败', cancelled: '已取消' }
  return map[status] ?? status
}

const fmtTime = (iso) => {
  const d = new Date(iso)
  const p = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth()+1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

let sseAbort = null
const startSSE = () => {
  const token = localStorage.getItem('token')
  const ctrl = new AbortController()
  sseAbort = ctrl
  const run = async () => {
    try {
      const res = await fetch(`/api/tasks/${route.params.id}/stream`, {
        headers: { Authorization: `Bearer ${token}` },
        signal: ctrl.signal,
      })
      const ct = res.headers.get('content-type') || ''
      if (ct.includes('application/json')) {
        const data = await res.json()
        if (data.done) {
          task.value.status = data.status
          task.value.result = data.result
          task.value.input_tokens = data.input_tokens
          task.value.output_tokens = data.output_tokens
          try {
            const { data: final } = await getTask(route.params.id)
            task.value = final.task
            projectName.value = final.project_name
            logs.value = final.logs
          } catch {}
        }
        return
      }
      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buf = ''
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buf += decoder.decode(value, { stream: true })
        const parts = buf.split('\n\n')
        buf = parts.pop()
        for (const part of parts) {
          const lines = part.split('\n')
          let eventType = 'message', dataStr = ''
          for (const line of lines) {
            if (line.startsWith('event: ')) eventType = line.slice(7).trim()
            if (line.startsWith('data: ')) dataStr = line.slice(6)
          }
          if (!dataStr) continue
          const data = JSON.parse(dataStr)
          if (eventType === 'log') {
            logs.value.push({ id: data.id, level: data.level, message: data.message, created_at: fmtTime(data.createdAt) })
          } else if (eventType === 'token') {
            task.value.input_tokens = data.input_tokens
            task.value.output_tokens = data.output_tokens
          } else if (eventType === 'result') {
            task.value.result = data.content
          } else if (eventType === 'done') {
            task.value.status = data.status
            task.value.result = data.result
            task.value.input_tokens = data.input_tokens
            task.value.output_tokens = data.output_tokens
            try {
              const { data: final } = await getTask(route.params.id)
              task.value = final.task
              projectName.value = final.project_name
              logs.value = final.logs
            } catch {}
            return
          }
        }
      }
    } catch (e) {
      if (e.name !== 'AbortError') ElMessage.error('实时连接中断')
    }
  }
  run()
}

onMounted(async () => {
  try {
    const { data } = await getTask(route.params.id)
    task.value = data.task
    projectName.value = data.project_name
    logs.value = data.logs || []
    scrollLogsToBottom()
    if (isActive.value) {
      startSSE()
    }
  } catch (e) {
    ElMessage.error('加载任务详情失败')
  }
})

onUnmounted(() => {
  sseAbort?.abort()
  clearInterval(elapsedTimer)
})

const handleCancel = async () => {
  try {
    await ElMessageBox.confirm('确认取消该任务？', '提示', { confirmButtonText: '确认', cancelButtonText: '取消', type: 'warning' })
    await apiCancelTask(route.params.id)
    router.push('/tasks')
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('取消任务失败')
  }
}
const handleRetry = async () => {
  try {
    const { data } = await apiRetryTask(route.params.id)
    router.push('/tasks/' + data.task_id)
  } catch (e) {
    ElMessage.error('重试任务失败')
  }
}
const handleRescan = async () => {
  try {
    await ElMessageBox.confirm('确认对相同 commit 范围再次发起扫描？', '再次扫描', { confirmButtonText: '确认', cancelButtonText: '取消', type: 'warning' })
    const { data } = await apiRetryTask(route.params.id)
    router.push('/tasks/' + data.task_id)
  } catch (e) {
    if (e !== 'cancel' && e?.type !== 'cancel') ElMessage.error(e.response?.data?.message || '再次扫描失败')
  }
}
const handleDelete = async () => {
  try {
    await ElMessageBox.confirm(`确认删除任务 #${task.value.id}？此操作不可恢复。`, '删除确认', { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' })
    await apiDeleteTask(route.params.id)
    ElMessage.success('删除成功')
    router.push('/tasks')
  } catch (e) {
    if (e !== 'cancel' && e?.type !== 'cancel') ElMessage.error(e.response?.data?.message || '删除失败')
  }
}
</script>

<style scoped>
.page-wrap { padding: 32px 36px; }
.breadcrumb { margin-bottom: 16px; }

/* Hero */
.page-hero {
  position: relative;
  background: linear-gradient(135deg, #7c3aed, #6d28d9, #2563eb);
  padding: 20px 28px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  overflow: hidden;
  margin-bottom: 20px;
}
.hero-title { margin: 0 0 4px; font-size: 22px; font-weight: 700; color: #fff; letter-spacing: -0.3px; }
.hero-sub { margin: 0; font-size: 13px; color: rgba(255,255,255,0.8); }
.hero-right {
  display: flex; flex-direction: column; align-items: flex-end; gap: 10px;
  position: relative; z-index: 2;
}
.hero-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }
.hero-actions :deep(.el-button) {
  font-weight: 600;
  background: rgba(255,255,255,0.15) !important;
  border-color: rgba(255,255,255,0.35) !important;
  color: #fff !important;
}
.hero-actions :deep(.el-button:hover) { background: rgba(255,255,255,0.28) !important; }
.deco-circles { position: absolute; right: 0; top: 0; bottom: 0; width: 200px; pointer-events: none; }
.deco { position: absolute; border-radius: 50%; background: rgba(255,255,255,0.08); }
.c1 { width: 180px; height: 180px; right: -40px; top: -60px; }
.c2 { width: 100px; height: 100px; right: 60px; bottom: -30px; }

/* Status pill */
.status-pill {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 4px 12px; border-radius: 20px; font-size: 13px; font-weight: 500; flex-shrink: 0;
  background: rgba(255,255,255,0.2); color: #fff; border: 1px solid rgba(255,255,255,0.3);
}

/* Summary row */
.summary-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin-bottom: 20px; }
.summary-card {
  background: #fff; border: 1px solid #e2e8f0; border-radius: 14px;
  padding: 14px 18px; display: flex; justify-content: space-between; align-items: center;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
}
.summary-card span { color: #64748b; font-size: 13px; }
.summary-card strong { color: #0f172a; font-size: 18px; max-width: 62%; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }

.elapsed-timer { color: #d97706; font-size: 14px; font-family: monospace; font-weight: 700; }

/* Two-column layout */
.main-columns {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 420px;
  gap: 20px;
  align-items: stretch;
  height: calc(100vh - 260px);
  min-height: 680px;
}

/* Left: tabs */
.col-main { min-width: 0; min-height: 0; display: flex; flex-direction: column; }
.result-tabs {
  background: #fff; border: 1px solid #e2e8f0; border-radius: 16px;
  padding: 12px 16px 16px; box-shadow: 0 1px 4px rgba(0,0,0,0.04);
  flex: 1; min-height: 0; display: flex; flex-direction: column;
}
.result-tabs :deep(.el-tabs__content) { flex: 1; min-height: 0; display: flex; }
.result-tabs :deep(.el-tab-pane) { flex: 1; min-height: 0; flex-direction: column; }
.result-tabs :deep(.el-tab-pane[style*="visibility: visible"]),
.result-tabs :deep(.el-tab-pane:not([style*="display: none"])) { display: flex; }
.result-toolbar { display: flex; justify-content: flex-end; padding: 2px 4px 4px; flex-shrink: 0; }
.fill-scrollbar { flex: 1; min-height: 0; height: 100%; }
.result-scrollbar {
  background: #fafbfc; border-radius: 10px;
  border: 1px solid #e2e8f0; box-shadow: 0 1px 4px rgba(0,0,0,0.03);
}
.tab-empty { color: #94a3b8; font-size: 13px; padding: 16px 4px; }

.commit-info-bar {
  margin-top: 12px;
  background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 10px;
  padding: 10px 14px; display: flex; align-items: center; gap: 12px; flex-wrap: wrap;
}
.commit-subjects { font-size: 12.5px; color: #64748b; }
.commit-arrow { color: #94a3b8; }

/* Right: log panel */
.col-log { min-height: 0; }
.log-panel {
  position: sticky;
  top: 16px;
  background: #0f172a;
  border-radius: 14px;
  border: 1px solid #1e293b;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0,0,0,0.18);
  display: flex;
  flex-direction: column;
  height: 100%;
  max-height: calc(100vh - 120px);
}
.log-panel-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 14px; border-bottom: 1px solid #1e293b; flex-shrink: 0;
}
.log-panel-title { font-size: 13px; font-weight: 600; color: #94a3b8; letter-spacing: 0.5px; text-transform: uppercase; }
.log-panel-actions { display: flex; align-items: center; gap: 6px; }
.log-panel-actions :deep(.el-button) { color: #64748b !important; }
.log-panel-actions :deep(.el-button:hover) { color: #94a3b8 !important; }

.live-dot {
  display: inline-flex; align-items: center; gap: 5px;
  font-size: 11px; color: #4ade80; font-weight: 600;
}
.blink-dot {
  width: 6px; height: 6px; border-radius: 50%; background: #4ade80;
  animation: blink-dot 1.2s ease-in-out infinite;
}
@keyframes blink-dot { 0%,100% { opacity: 1 } 50% { opacity: 0.2 } }

.log-scrollbar {
  flex: 1;
  min-height: 0;
  height: 100%;
}
.log-fullscreen-scrollbar { height: calc(100vh - 84px); }
.log-panel-footer {
  padding: 6px 14px; border-top: 1px solid #1e293b;
  flex-shrink: 0;
}
.log-count { font-size: 11px; color: #475569; }

/* Log lines — dark theme inside panel */
.log-list { padding: 4px 0; }
.log-line {
  display: grid;
  grid-template-columns: 68px 42px minmax(0, 1fr);
  align-items: baseline;
  column-gap: 7px;
  padding: 3px 10px 3px 8px;
  font-size: 12px;
  line-height: 1.6;
  border-left: 2px solid transparent;
  transition: background 0.1s;
}
.log-line + .log-line { margin-top: 1px; }
.log-line:hover { background: rgba(148,163,184,0.06); }

.log-line-error {
  background: linear-gradient(90deg, rgba(220,38,38,0.12) 0%, transparent 80%);
  border-left-color: #ef4444;
}
.log-line-warn {
  background: linear-gradient(90deg, rgba(234,179,8,0.10) 0%, transparent 80%);
  border-left-color: #eab308;
}
.log-line-error:hover { background: rgba(220,38,38,0.18); }
.log-line-warn:hover  { background: rgba(234,179,8,0.15); }

.log-empty { color: #475569; font-size: 13px; padding: 12px 10px; }
.log-msg { color: #94a3b8; word-break: break-word; white-space: pre-wrap; }
.log-line-error .log-msg { color: #fca5a5; }
.log-line-warn  .log-msg { color: #fde68a; }

.log-time {
  font-family: monospace; font-size: 10.5px; color: #475569;
  white-space: nowrap; user-select: none; padding-top: 1px;
}
.log-badge {
  display: inline-flex; align-items: center; justify-content: center;
  width: 34px; height: 16px; border-radius: 3px;
  font-size: 9.5px; font-weight: 700; letter-spacing: 0.4px;
  text-transform: uppercase; white-space: nowrap; flex-shrink: 0; margin-top: 2px;
}
.log-badge-info  { background: #1e293b; color: #64748b; }
.log-badge-warn  { background: rgba(234,179,8,0.2); color: #fbbf24; }
.log-badge-error { background: rgba(239,68,68,0.2); color: #f87171; }

/* fullscreen log dialog uses light theme */
.log-list-dialog { padding: 8px 16px; background: #fff; }
.log-list-dialog .log-line { border-left-color: transparent; }
.log-list-dialog .log-line-error { background: linear-gradient(90deg,#fff5f5,transparent 80%); border-left-color: #fca5a5; }
.log-list-dialog .log-line-warn  { background: linear-gradient(90deg,#fefce8,transparent 80%); border-left-color: #fde68a; }
.log-list-dialog .log-msg { color: #334155; }
.log-list-dialog .log-line-error .log-msg { color: #b91c1c; }
.log-list-dialog .log-line-warn  .log-msg { color: #92400e; }
.log-list-dialog .log-time { color: #94a3b8; }
.log-list-dialog .log-badge-info  { background: #f1f5f9; color: #64748b; }
.log-list-dialog .log-badge-warn  { background: #fef3c7; color: #b45309; }
.log-list-dialog .log-badge-error { background: #fee2e2; color: #dc2626; }

/* Markdown */
.markdown-body { padding: 20px 24px; font-size: 14.5px; line-height: 1.85; color: #1e293b; }
.markdown-body :deep(h1) {
  font-size: 1.45em; font-weight: 700; color: #0f172a;
  margin: 0 0 20px; padding-bottom: 12px;
  border-bottom: 2px solid #e2e8f0; letter-spacing: -0.3px;
}
.markdown-body :deep(h2) {
  font-size: 1.18em; font-weight: 700; color: #1e293b;
  margin: 28px 0 12px; padding: 9px 14px;
  background: linear-gradient(90deg, #eff6ff 0%, #f8fafc 100%);
  border-left: 3px solid #2563eb; border-radius: 0 6px 6px 0;
}
.markdown-body :deep(h3) {
  font-size: 1.04em; font-weight: 600; color: #374151;
  margin: 22px 0 8px; padding-left: 10px; border-left: 2px solid #93c5fd;
}
.markdown-body :deep(h4),.markdown-body :deep(h5),.markdown-body :deep(h6) {
  font-size: 0.96em; font-weight: 600; color: #4b5563; margin: 16px 0 6px;
}
.markdown-body :deep(p) { margin: 10px 0; }
.markdown-body :deep(ul),.markdown-body :deep(ol) { padding-left: 26px; margin: 10px 0; }
.markdown-body :deep(li) { margin: 6px 0; line-height: 1.75; }
.markdown-body :deep(ul > li)::marker,.markdown-body :deep(ol > li)::marker { color: #6366f1; font-weight: 600; }
.markdown-body :deep(code) {
  background: #f1f5f9; border: 1px solid #e2e8f0; border-radius: 4px;
  padding: 1px 6px; font-family: 'JetBrains Mono','Fira Code',monospace;
  font-size: 12.5px; color: #e11d48;
}
.markdown-body :deep(pre) {
  background: #0f172a; border-radius: 8px; padding: 16px 20px;
  overflow-x: auto; margin: 14px 0; border: 1px solid #1e293b;
  box-shadow: 0 2px 10px rgba(0,0,0,0.14);
}
.markdown-body :deep(pre code) {
  background: none; border: none; padding: 0;
  color: #e2e8f0; font-size: 13px;
  font-family: 'JetBrains Mono','Fira Code',monospace; line-height: 1.65;
}
.markdown-body :deep(blockquote) {
  border-left: 4px solid #6366f1;
  background: linear-gradient(90deg,#f5f3ff 0%,rgba(245,243,255,0.3) 60%,transparent 100%);
  color: #4b5563; margin: 14px 0; padding: 10px 16px;
  border-radius: 0 8px 8px 0; font-style: italic;
}
.markdown-body :deep(table) {
  border-collapse: collapse; width: 100%; margin: 14px 0;
  border: 1px solid #e2e8f0; border-radius: 8px; overflow: hidden;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
}
.markdown-body :deep(th) {
  background: linear-gradient(135deg,#f8fafc,#f1f5f9); font-weight: 600;
  color: #374151; padding: 10px 14px; font-size: 13px;
  border-bottom: 2px solid #e2e8f0; text-align: left; white-space: nowrap;
}
.markdown-body :deep(td) {
  padding: 9px 14px; font-size: 13px; border-bottom: 1px solid #f1f5f9;
  color: #4b5563; vertical-align: top;
}
.markdown-body :deep(tr:last-child td) { border-bottom: none; }
.markdown-body :deep(tr:nth-child(even) td) { background: #fafbfc; }
.markdown-body :deep(hr) {
  border: none; height: 2px;
  background: linear-gradient(90deg,#e2e8f0,rgba(226,232,240,0.2));
  margin: 22px 0; border-radius: 1px;
}
.markdown-body :deep(strong) { color: #0f172a; font-weight: 700; }
.markdown-body :deep(em) { color: #475569; }
.markdown-body :deep(a) { color: #2563eb; text-decoration: none; border-bottom: 1px solid #93c5fd; transition: color 0.15s,border-color 0.15s; }
.markdown-body :deep(a:hover) { color: #1d4ed8; border-bottom-color: #2563eb; }
.markdown-body-dialog { max-width: 900px; margin: 0 auto; padding: 24px 32px; }

/* Diff */
.diff-block {
  font-family: monospace; font-size: 12.5px; line-height: 1.6;
  background: #0f172a; color: #cbd5e1; border-radius: 8px;
  padding: 16px; margin: 0; white-space: pre;
}
.diff-nav {
  display: flex; flex-wrap: wrap; gap: 6px;
  padding: 8px 4px; border-bottom: 1px solid #e2e8f0; margin-bottom: 4px;
}
.diff-nav-item {
  font-family: monospace; font-size: 12px; color: #2563eb; background: #eff6ff;
  padding: 2px 8px; border-radius: 4px; cursor: pointer;
  white-space: nowrap; max-width: 240px; overflow: hidden; text-overflow: ellipsis;
}
.diff-nav-item:hover { background: #dbeafe; }
:deep(.diff-add) { color: #4ade80; }
:deep(.diff-del) { color: #f87171; }

.commit-block {
  font-family: monospace; font-size: 13px; line-height: 1.8;
  color: #475569; white-space: pre-wrap; margin: 0; padding: 4px 2px;
}

.mono {
  font-family: monospace; font-size: 12.5px;
  background: #f1f5f9; padding: 2px 7px; border-radius: 5px; color: #475569;
}

/* Responsive: collapse to single column on narrow screens */
@media (max-width: 960px) {
  .main-columns { grid-template-columns: 1fr; height: auto; min-height: 0; }
  .log-panel { position: static; height: 420px; max-height: 420px; }
  .log-scrollbar { height: 100% !important; }
  .fill-scrollbar { height: 520px; }
}
</style>
