<template>
  <div class="jobs-container">
    <div class="jobs-header">
      <span class="jobs-title">执行记录</span>
      <el-button size="small" @click="load" :loading="loading">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <el-empty v-if="!loading && jobs.length === 0" description="暂无执行记录" />

    <div v-for="job in jobs" :key="job.id" :class="['job-item', { 'job-running': job.id === runningJobId }]" :data-job-id="job.id" @click="toggleJob(job.id)">
      <div class="job-header">
        <div class="job-meta">
          <span :class="['pill', statusClass(job.status)]">{{ statusLabel(job.status) }}</span>
          <span class="job-time">{{ formatTime(job.triggered_at) }}</span>
          <span class="job-branches" v-if="job.status === 'completed'">
            {{ job.changed_branch_count }} / {{ job.branch_count }} 分支有改动
          </span>
        </div>
        <div class="job-actions">
          <span v-if="job.report_path" class="report-path" @click.stop>
            <el-icon><Document /></el-icon>
            <a :href="job.report_path" target="_blank" rel="noopener">{{ job.report_path }}</a>
          </span>
          <el-icon class="expand-icon" :class="{ rotated: expandedJobId === job.id }"><ArrowDown /></el-icon>
        </div>
      </div>

      <!-- 展开：分支结果 -->
      <div v-if="expandedJobId === job.id" class="job-detail" @click.stop>
        <div v-if="loadingDetail" class="loading-detail">加载中...</div>
        <template v-else>
          <div v-if="job.error_message" class="error-msg">
            <el-icon><Warning /></el-icon> {{ job.error_message }}
          </div>
          <div v-if="branchResults.length === 0" class="empty-branch">无有改动的分支</div>
          <div v-for="br in branchResults" :key="br.id" class="branch-item">
            <div class="branch-header">
              <span class="branch-name">{{ br.branch_name }}</span>
              <span :class="['risk-pill', riskClass(br.risk_level)]">{{ riskLabel(br.risk_level) }}</span>
              <span class="branch-commits">{{ br.commit_count }} commits</span>
              <span class="stage-tag">{{ stageLabel(br.analysis_stage) }}</span>
            </div>
            <div class="branch-result" v-html="renderMd(br.result)" />
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Document, ArrowDown, Warning } from '@element-plus/icons-vue'
import { listScanJobs, getScanJob } from '../../api/scan'

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
  loadingDetail.value = true
  branchResults.value = []
  try {
    const res = await getScanJob(jobId)
    branchResults.value = res.data.results || []
  } catch {
    ElMessage.error('加载详情失败')
  } finally {
    loadingDetail.value = false
  }
}

function renderMd(text) {
  if (!text) return ''
  // 简单 Markdown 渲染：换行、粗体、代码
  return text
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\n/g, '<br>')
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
  // 如果有 initialJobId，自动展开对应 job
  if (props.initialJobId) {
    await toggleJob(props.initialJobId)
    // 滚动到该 job
    await nextTick()
    const el = document.querySelector(`.job-item[data-job-id="${props.initialJobId}"]`)
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
})

// 暴露 load 给父组件轮询时调用
defineExpose({ load })
</script>

<style scoped>
.jobs-container { padding: 4px 0; }
.jobs-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 16px;
}
.jobs-title { font-size: 15px; font-weight: 600; color: #1e293b; }

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

.job-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 16px; background: #fff;
}
.job-meta { display: flex; align-items: center; gap: 12px; }
.job-time { font-size: 13px; color: #64748b; }
.job-branches { font-size: 12px; color: #475569; }
.job-actions { display: flex; align-items: center; gap: 10px; }
.report-path { font-size: 11px; color: #64748b; display: flex; align-items: center; gap: 4px; }
.report-path a { color: #2563eb; text-decoration: none; }
.report-path a:hover { text-decoration: underline; }
.expand-icon { color: #94a3b8; transition: transform 0.2s; }
.expand-icon.rotated { transform: rotate(180deg); }

.pill { display: inline-block; padding: 2px 8px; border-radius: 99px; font-size: 11px; font-weight: 600; }
.pill-done    { background: #dcfce7; color: #166534; }
.pill-running { background: #dbeafe; color: #1d4ed8; }
.pill-fail    { background: #fee2e2; color: #991b1b; }
.pill-pending { background: #fef9c3; color: #854d0e; }

.job-detail {
  background: #f8fafc; border-top: 1px solid #e2e8f0;
  padding: 16px;
}
.loading-detail { text-align: center; color: #94a3b8; padding: 12px 0; }
.error-msg { display: flex; align-items: center; gap: 6px; color: #dc2626; font-size: 13px; margin-bottom: 12px; }
.empty-branch { color: #94a3b8; font-size: 13px; text-align: center; padding: 12px 0; }

.branch-item {
  background: #fff; border: 1px solid #e2e8f0; border-radius: 8px;
  margin-bottom: 12px; overflow: hidden;
}
.branch-header {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 14px; background: #f8fafc; border-bottom: 1px solid #f1f5f9;
}
.branch-name { font-weight: 600; font-size: 13px; color: #1e293b; }
.branch-commits { font-size: 12px; color: #64748b; }
.stage-tag { font-size: 11px; color: #94a3b8; margin-left: auto; }
.branch-result {
  padding: 12px 14px; font-size: 13px; color: #374151; line-height: 1.7;
}
.branch-result :deep(code) {
  background: #f1f5f9; padding: 1px 5px; border-radius: 3px;
  font-family: monospace; font-size: 12px;
}

.risk-pill { display: inline-block; padding: 1px 8px; border-radius: 99px; font-size: 11px; font-weight: 600; }
.risk-high    { background: #fee2e2; color: #991b1b; }
.risk-medium  { background: #ffedd5; color: #9a3412; }
.risk-low     { background: #fef9c3; color: #854d0e; }
.risk-none    { background: #dcfce7; color: #166534; }
.risk-unknown { background: #f1f5f9; color: #64748b; }
</style>
