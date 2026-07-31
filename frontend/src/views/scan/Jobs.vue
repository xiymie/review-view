<template>
  <div class="jobs-container">
    <div class="jobs-header">
      <span class="jobs-title">执行记录</span>
      <el-button size="small" @click="load" :loading="loading">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <el-empty v-if="!loading && jobs.length === 0" description="暂无执行记录" />

    <div v-for="job in jobs" :key="job.id"
         :class="['job-item', { 'job-running': job.id === runningJobId }]"
         :data-job-id="job.id" @click="toggleJob(job.id)">

      <!-- ── Job summary row ── -->
      <div class="job-header">
        <div class="job-meta">
          <span :class="['pill', statusClass(job.status)]">{{ statusLabel(job.status) }}</span>
          <span class="job-time">{{ formatTime(job.triggered_at) }}</span>
          <span class="job-branches" v-if="job.status === 'completed'">
            <strong>{{ job.changed_branch_count }}</strong> / {{ job.branch_count }} 分支有改动
          </span>
        </div>
        <div class="job-actions">
          <span v-if="job.report_path" class="report-path" @click.stop>
            <el-icon><Document /></el-icon>
            <a :href="job.report_path" target="_blank" rel="noopener">{{ job.report_path }}</a>
          </span>
          <el-button link type="danger" size="small" class="del-btn"
            @click.stop="handleDelete(job)" :disabled="job.status === 'running'">
            <el-icon><Delete /></el-icon>
          </el-button>
          <el-icon class="expand-icon" :class="{ rotated: expandedJobId === job.id }"><ArrowDown /></el-icon>
        </div>
      </div>

      <!-- ── Expanded detail ── -->
      <div v-if="expandedJobId === job.id" class="job-detail" @click.stop>
        <div v-if="loadingDetail" class="loading-detail">
          <el-icon class="is-loading"><Loading /></el-icon> 加载中…
        </div>
        <template v-else>
          <div v-if="job.error_message" class="error-msg">
            <el-icon><Warning /></el-icon> {{ job.error_message }}
          </div>

          <!-- Workflow log panel with level filter -->
          <div v-if="jobLogs.length" class="workflow-log-panel">
            <div class="workflow-log-header">
              <span class="workflow-log-title">Workflow 节点日志</span>
              <div class="log-filters">
                <button v-for="lvl in logLevels" :key="lvl.value"
                  :class="['log-filter-btn', { active: logFilter === lvl.value }, `filter-${lvl.value}`]"
                  @click.stop="logFilter = lvl.value">
                  {{ lvl.label }}
                  <span class="filter-count">{{ logLevelCount(lvl.value) }}</span>
                </button>
              </div>
            </div>
            <div class="workflow-log-list">
              <div v-if="filteredLogs.length === 0" class="log-empty">无匹配日志</div>
              <div v-for="log in filteredLogs" :key="log.id"
                   class="workflow-log-line" :class="`workflow-log-${log.level}`">
                <span class="workflow-log-time">{{ log.created_at?.slice(11, 19) || '—' }}</span>
                <span :class="['workflow-log-badge', `badge-${log.level}`]">{{ (log.level || 'info').toUpperCase() }}</span>
                <span class="workflow-log-msg">{{ log.message }}</span>
              </div>
            </div>
          </div>

          <!-- Branch result cards -->
          <div v-if="branchResults.length === 0 && !job.error_message" class="empty-branch">
            无有改动的分支
          </div>
          <div v-for="br in branchResults" :key="br.id"
               :class="['branch-item', `branch-risk-${br.risk_level}`]">
            <div class="branch-header">
              <div class="branch-primary">
                <span :class="['risk-pill', riskClass(br.risk_level)]">{{ riskLabel(br.risk_level) }}</span>
                <span class="branch-name">{{ br.branch_name }}</span>
              </div>
              <div class="branch-secondary">
                <span class="branch-commits">{{ br.commit_count }} commits</span>
                <span class="stage-tag">{{ stageLabel(br.analysis_stage) }}</span>
              </div>
            </div>
            <div class="branch-result" v-html="renderMd(br.result)" />
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Document, ArrowDown, Warning, Delete, Loading } from '@element-plus/icons-vue'
import { listScanJobs, getScanJob, deleteScanJob } from '../../api/scan'

const props = defineProps({
  scheduleId: { type: Number, required: true },
  runningJobId: { type: Number, default: null },
  initialJobId: { type: Number, default: null },
})

const jobs = ref([])
const loading = ref(false)
const expandedJobId = ref(null)
const loadingDetail = ref(false)
const branchResults = ref([])
const jobLogs = ref([])
const logFilter = ref('all')

const logLevels = [
  { value: 'all',   label: '全部' },
  { value: 'info',  label: 'INFO' },
  { value: 'warn',  label: 'WARN' },
  { value: 'error', label: 'ERROR' },
]

const filteredLogs = computed(() => {
  if (logFilter.value === 'all') return jobLogs.value
  return jobLogs.value.filter(l => (l.level || 'info') === logFilter.value)
})

function logLevelCount(level) {
  if (level === 'all') return jobLogs.value.length
  return jobLogs.value.filter(l => (l.level || 'info') === level).length
}

async function load() {
  loading.value = true
  try {
    const res = await listScanJobs(props.scheduleId)
    jobs.value = res.data
  } catch {
    ElMessage.error('加载执行记录失败')
  } finally {
    loading.value = false
  }
}

async function toggleJob(jobId) {
  if (expandedJobId.value === jobId) {
    expandedJobId.value = null
    return
  }
  expandedJobId.value = jobId
  logFilter.value = 'all'
  loadingDetail.value = true
  branchResults.value = []
  jobLogs.value = []
  try {
    const res = await getScanJob(jobId)
    branchResults.value = res.data.results || []
    jobLogs.value = res.data.logs || []
  } catch {
    ElMessage.error('加载详情失败')
  } finally {
    loadingDetail.value = false
  }
}

function renderMd(text) {
  if (!text) return ''
  const escaped = text
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

  const lines = escaped.split('\n')
  const out = []
  let inList = false

  for (const raw of lines) {
    let line = raw
      .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
      .replace(/`([^`]+)`/g, '<code>$1</code>')

    if (/^### (.+)/.test(line)) {
      if (inList) { out.push('</ul>'); inList = false }
      out.push(`<h5>${line.replace(/^### /, '')}</h5>`)
    } else if (/^## (.+)/.test(line)) {
      if (inList) { out.push('</ul>'); inList = false }
      out.push(`<h4>${line.replace(/^## /, '')}</h4>`)
    } else if (/^# (.+)/.test(line)) {
      if (inList) { out.push('</ul>'); inList = false }
      out.push(`<h3>${line.replace(/^# /, '')}</h3>`)
    } else if (/^[-*] (.+)/.test(line)) {
      if (!inList) { out.push('<ul>'); inList = true }
      out.push(`<li>${line.replace(/^[-*] /, '')}</li>`)
    } else if (/^\d+\. (.+)/.test(line)) {
      if (inList) { out.push('</ul>'); inList = false }
      out.push(`<li>${line.replace(/^\d+\. /, '')}</li>`)
    } else if (line.trim() === '') {
      if (inList) { out.push('</ul>'); inList = false }
      out.push('<br>')
    } else {
      if (inList) { out.push('</ul>'); inList = false }
      out.push(`<p>${line}</p>`)
    }
  }
  if (inList) out.push('</ul>')
  return out.join('')
}

function statusClass(s) {
  const map = { completed: 'pill-done', running: 'pill-running', failed: 'pill-fail', pending: 'pill-pending' }
  return map[s] || 'pill-pending'
}
function statusLabel(s) {
  const map = { completed: '完成', running: '运行中', failed: '失败', pending: '等待' }
  return map[s] || s
}
function riskClass(r) {
  const map = { high: 'risk-high', medium: 'risk-medium', low: 'risk-low', none: 'risk-none', unknown: 'risk-unknown' }
  return map[r] || 'risk-unknown'
}
function riskLabel(r) {
  const map = { high: '高风险', medium: '中风险', low: '低风险', none: '无风险', unknown: '未知' }
  return map[r] || r
}
function stageLabel(s) {
  return s === 'with_diff' ? '含 diff 分析' : 'message 分析'
}
function formatTime(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  const p = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth()+1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

watch(() => props.scheduleId, load)
onMounted(async () => {
  await load()
  if (props.initialJobId) {
    await toggleJob(props.initialJobId)
    await nextTick()
    const el = document.querySelector(`.job-item[data-job-id="${props.initialJobId}"]`)
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
})

defineExpose({ load })

async function handleDelete(job) {
  try {
    await ElMessageBox.confirm(
      `确认删除该条执行记录（${formatTime(job.triggered_at)}）？删除后不可恢复。`,
      '删除确认',
      { type: 'warning', confirmButtonText: '删除', confirmButtonClass: 'el-button--danger' }
    )
    await deleteScanJob(job.id)
    ElMessage.success('已删除')
    await load()
  } catch (err) {
    if (err === 'cancel' || err?.type === 'cancel') return
    ElMessage.error(err.response?.data?.error || '删除失败')
  }
}
</script>

<style scoped>
/* ── Container ── */
.jobs-container { padding: 4px 0; }
.jobs-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 16px;
}
.jobs-title { font-size: 15px; font-weight: 600; color: #1e293b; }

/* ── Job card ── */
.job-item {
  border: 1px solid #e2e8f0; border-radius: 10px;
  margin-bottom: 10px; overflow: hidden; cursor: pointer;
  transition: box-shadow 0.15s;
}
.job-item:hover { box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
.job-item.job-running {
  border-color: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59,130,246,0.15);
}

/* ── Job header (summary row) ── */
.job-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 16px; background: #fff;
}
.job-meta { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.job-time { font-size: 13px; color: #64748b; }
.job-branches { font-size: 12px; color: #475569; }
.job-branches strong { color: #1e293b; }
.job-actions { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }
.report-path { font-size: 11px; color: #64748b; display: flex; align-items: center; gap: 4px; }
.report-path a { color: #2563eb; text-decoration: none; }
.report-path a:hover { text-decoration: underline; }
.expand-icon { color: #94a3b8; transition: transform 0.2s; }
.expand-icon.rotated { transform: rotate(180deg); }
.del-btn { color: #94a3b8 !important; font-size: 14px; }
.del-btn:hover { color: #ef4444 !important; }

/* ── Status pills ── */
.pill { display: inline-block; padding: 2px 8px; border-radius: 99px; font-size: 11px; font-weight: 600; }
.pill-done    { background: #dcfce7; color: #166534; }
.pill-running { background: #dbeafe; color: #1d4ed8; }
.pill-fail    { background: #fee2e2; color: #991b1b; }
.pill-pending { background: #fef9c3; color: #854d0e; }

/* ── Detail panel ── */
.job-detail {
  background: #f8fafc; border-top: 1px solid #e2e8f0;
  padding: 20px 18px 16px;
}
.loading-detail {
  display: flex; align-items: center; justify-content: center; gap: 6px;
  color: #94a3b8; padding: 16px 0; font-size: 13px;
}
.error-msg {
  display: flex; align-items: flex-start; gap: 6px;
  color: #dc2626; font-size: 13px; margin-bottom: 14px;
  background: #fff5f5; border: 1px solid #fecaca; border-radius: 6px;
  padding: 10px 12px;
}

/* ── Workflow log panel ── */
.workflow-log-panel {
  background: #fff; border: 1px solid #e2e8f0; border-radius: 8px;
  margin-bottom: 18px; overflow: hidden;
}
.workflow-log-header {
  display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px;
  padding: 10px 14px; background: #f8fafc; border-bottom: 1px solid #e2e8f0;
}
.workflow-log-title { font-size: 13px; font-weight: 700; color: #334155; }
.log-filters { display: flex; gap: 4px; }
.log-filter-btn {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 2px 8px; border-radius: 99px; font-size: 11px; font-weight: 600;
  border: 1px solid #e2e8f0; background: #fff; color: #64748b;
  cursor: pointer; transition: all 0.15s;
}
.log-filter-btn:hover { border-color: #94a3b8; color: #334155; }
.log-filter-btn.active { background: #1e293b; color: #fff; border-color: #1e293b; }
.log-filter-btn.filter-error.active { background: #dc2626; border-color: #dc2626; }
.log-filter-btn.filter-warn.active  { background: #d97706; border-color: #d97706; }
.log-filter-btn.filter-info.active  { background: #2563eb; border-color: #2563eb; }
.filter-count { opacity: 0.7; }
.workflow-log-list {
  max-height: 260px; overflow-y: auto; padding: 6px 0;
  font-family: ui-monospace, "SFMono-Regular", Menlo, monospace;
}
.log-empty { text-align: center; color: #94a3b8; font-size: 12px; padding: 12px 0; font-family: inherit; }
.workflow-log-line {
  display: grid; grid-template-columns: 70px 52px 1fr; gap: 8px;
  align-items: baseline; padding: 4px 14px; border-radius: 0; font-size: 12px; line-height: 1.6;
  border-bottom: 1px solid transparent;
}
.workflow-log-line:last-child { border-bottom: none; }
.workflow-log-line:hover { background: #f8fafc; }
.workflow-log-error { background: #fff5f5 !important; }
.workflow-log-error:hover { background: #fee2e2 !important; }
.workflow-log-time { color: #94a3b8; }
.workflow-log-badge {
  padding: 1px 5px; border-radius: 3px; font-size: 10px; font-weight: 700; text-align: center;
  background: #f1f5f9; color: #64748b;
}
.badge-error { background: #fee2e2; color: #991b1b; }
.badge-warn  { background: #ffedd5; color: #9a3412; }
.badge-info  { background: #dbeafe; color: #1d4ed8; }
.workflow-log-msg { color: #334155; word-break: break-word; white-space: pre-wrap; }
.workflow-log-error .workflow-log-msg { color: #b91c1c; }

/* ── Branch cards ── */
.empty-branch { color: #94a3b8; font-size: 13px; text-align: center; padding: 16px 0; }

.branch-item {
  background: #fff; border: 1px solid #e2e8f0; border-radius: 8px;
  margin-bottom: 12px; overflow: hidden;
}
/* risk-level left accent */
.branch-risk-high   { border-left: 3px solid #ef4444; }
.branch-risk-medium { border-left: 3px solid #f97316; }
.branch-risk-low    { border-left: 3px solid #eab308; }
.branch-risk-none   { border-left: 3px solid #22c55e; }

.branch-header {
  display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px;
  padding: 10px 16px; background: #f8fafc; border-bottom: 1px solid #f1f5f9;
}
.branch-primary { display: flex; align-items: center; gap: 8px; }
.branch-secondary { display: flex; align-items: center; gap: 10px; }
.branch-name { font-weight: 700; font-size: 13px; color: #1e293b; }
.branch-commits { font-size: 12px; color: #64748b; }
.stage-tag {
  font-size: 11px; color: #94a3b8;
  background: #f1f5f9; padding: 1px 6px; border-radius: 4px;
}

/* ── Result content ── */
.branch-result {
  padding: 14px 18px; font-size: 13.5px; color: #374151; line-height: 1.75;
}
.branch-result :deep(h3) { font-size: 14px; font-weight: 700; color: #1e293b; margin: 12px 0 6px; }
.branch-result :deep(h4) { font-size: 13px; font-weight: 700; color: #1e293b; margin: 10px 0 4px; }
.branch-result :deep(h5) { font-size: 12px; font-weight: 700; color: #334155; margin: 8px 0 4px; }
.branch-result :deep(p)  { margin: 0 0 6px; }
.branch-result :deep(ul) { margin: 4px 0 8px; padding-left: 18px; }
.branch-result :deep(li) { margin-bottom: 3px; }
.branch-result :deep(strong) { color: #0f172a; font-weight: 600; }
.branch-result :deep(code) {
  background: #f1f5f9; padding: 1px 5px; border-radius: 3px;
  font-family: ui-monospace, Menlo, monospace; font-size: 12px; color: #be185d;
}
.branch-result :deep(br) { display: block; content: ''; margin: 4px 0; }

/* ── Risk pills ── */
.risk-pill { display: inline-block; padding: 2px 9px; border-radius: 99px; font-size: 11px; font-weight: 700; }
.risk-high    { background: #fee2e2; color: #991b1b; }
.risk-medium  { background: #ffedd5; color: #9a3412; }
.risk-low     { background: #fef9c3; color: #854d0e; }
.risk-none    { background: #dcfce7; color: #166534; }
.risk-unknown { background: #f1f5f9; color: #64748b; }
</style>
