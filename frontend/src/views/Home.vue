<template>
  <div class="dashboard">

    <!-- 欢迎横幅 -->
    <div class="hero">
      <div class="hero-left">
        <div class="greeting">{{ greeting }}，<span class="username">{{ username }}</span></div>
        <div class="hero-date">{{ dateLabel }} · 代码审核平台</div>
      </div>
      <div class="hero-right">
          <div class="hero-right-col">
            <div class="hero-badge" v-if="stats.running_count > 0">
              <span class="badge-dot blink"></span>
              {{ stats.running_count }} 个任务运行中
            </div>
            <div class="hero-badge hero-badge--idle" v-else>
              <span class="badge-dot idle"></span>
              系统空闲
            </div>
            <!-- 定时刷新选择器 -->
            <div class="refresh-selector">
              <el-icon class="refresh-icon" :class="{ spinning: autoRefreshInterval > 0 }"><Refresh /></el-icon>
              <el-select
                v-model="autoRefreshInterval"
                size="small"
                class="refresh-select"
                @change="onRefreshChange"
              >
                <el-option label="不自动刷新" :value="0" />
                <el-option label="每 5 秒" :value="5" />
                <el-option label="每 10 秒" :value="10" />
                <el-option label="每 30 秒" :value="30" />
              </el-select>
            </div>
          </div>
      </div>
      <div class="hero-bg">
        <div class="hb hb1"></div>
        <div class="hb hb2"></div>
        <div class="hb hb3"></div>
      </div>
    </div>

    <!-- 状态总览条 -->
    <div class="stat-bar" v-loading="loading">
      <div
        v-for="(s, i) in statItems"
        :key="s.label"
        class="stat-item"
        :class="{ clickable: s.route }"
        :style="{ '--c': s.color }"
        @click="s.route && $router.push(s.route)"
      >
        <div class="stat-accent"></div>
        <div class="stat-body">
          <div class="stat-num">{{ s.value }}</div>
          <div class="stat-label">{{ s.label }}</div>
        </div>
        <el-icon class="stat-icon"><component :is="s.icon" /></el-icon>
      </div>
    </div>

    <!-- 主内容区 -->
    <div class="main-grid">

      <!-- 左：动态时间轴 -->
      <div class="panel timeline-panel">
        <div class="panel-head">
          <span class="panel-title">最近动态</span>
          <span class="panel-sub">过去 7 天</span>
        </div>
        <div class="timeline" v-loading="loading">
          <div v-if="activities.length === 0" class="empty">暂无动态</div>
          <div
            v-for="act in activities"
            :key="act.kind + act.id"
            class="tl-item"
            :class="{ 'tl-clickable': true }"
            @click="goActivity(act)"
          >
            <div class="tl-line-wrap">
              <div class="tl-dot" :class="`tl-dot--${act.kind === 'scan' ? 'scan' : act.status}`"></div>
              <div class="tl-line"></div>
            </div>
            <div class="tl-content">
              <div class="tl-top">
                <span class="tl-kind" :class="`kind--${act.kind}`">
                  {{ act.kind === 'scan' ? '巡检' : 'Review' }}
                </span>
                <span class="tl-title">{{ act.title }}</span>
                <span
                  v-if="act.kind === 'scan' && act.has_risk"
                  class="tl-risk"
                >⚠ {{ act.risk_count }} 高风险</span>
                <span class="tl-status" :class="`sts--${act.status}`">{{ statusLabel(act.status) }}</span>
              </div>
              <div class="tl-bottom">
                <code class="tl-sub">{{ act.sub_title }}</code>
                <span class="tl-time">{{ act.time_ago }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧两块 -->
      <div class="right-col">

        <!-- 系统资产 -->
        <div class="panel asset-panel">
          <div class="panel-head">
            <span class="panel-title">系统资产</span>
          </div>
          <div class="asset-grid">
            <div class="asset-item" @click="$router.push('/models')" style="--ac:#6366f1">
              <el-icon class="asset-icon"><Cpu /></el-icon>
              <div class="asset-num">{{ stats.model_count }}</div>
              <div class="asset-label">模型配置</div>
            </div>
            <div class="asset-item" @click="$router.push('/credentials')" style="--ac:#0891b2">
              <el-icon class="asset-icon"><Key /></el-icon>
              <div class="asset-num">{{ stats.credential_count }}</div>
              <div class="asset-label">仓库凭据</div>
            </div>
            <div class="asset-item" @click="$router.push('/scan')" style="--ac:#7c3aed">
              <el-icon class="asset-icon"><Search /></el-icon>
              <div class="asset-num">{{ stats.scan_enabled_count }}</div>
              <div class="asset-label">巡检启用</div>
            </div>
          </div>
        </div>

        <!-- 7日热力图 -->
        <div class="panel heatmap-panel">
          <div class="panel-head">
            <span class="panel-title">本周任务热力</span>
            <span class="panel-sub">近 7 天</span>
          </div>
          <div class="heatmap">
            <div
              v-for="day in heatmap"
              :key="day.date"
              class="hm-day"
              :class="heatLevel(day.completed)"
              :title="`${day.date}  完成 ${day.completed}  失败 ${day.failed}`"
            >
              <div class="hm-bar">
                <div
                  class="hm-completed"
                  :style="{ height: barHeight(day.completed) + '%' }"
                ></div>
                <div
                  class="hm-failed"
                  :style="{ height: barHeight(day.failed) + '%', marginTop: '2px' }"
                ></div>
              </div>
              <div class="hm-date">{{ shortDate(day.date) }}</div>
              <div class="hm-count" v-if="day.completed + day.failed > 0">
                {{ day.completed + day.failed }}
              </div>
            </div>
          </div>
          <div class="hm-legend">
            <span class="leg-dot leg-completed"></span><span>完成</span>
            <span class="leg-dot leg-failed"></span><span>失败</span>
          </div>
        </div>

      </div>
    </div>

  </div>

  <!-- 动态详情弹窗 -->
  <el-dialog
    v-model="detailVisible"
    :title="detailKind === 'task' ? 'Review 任务详情' : '巡检任务详情'"
    width="660px"
    :close-on-click-modal="true"
    destroy-on-close
  >
    <div v-loading="detailLoading" style="min-height:80px">

      <!-- Task 详情 -->
      <template v-if="detailKind === 'task' && taskDetail">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="项目">{{ taskDetail.project_name }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <span :class="`sts--${taskDetail.task.status}`" style="padding:1px 8px;border-radius:99px;font-size:12px">
              {{ statusLabel(taskDetail.task.status) }}
            </span>
          </el-descriptions-item>
          <el-descriptions-item label="触发方式">{{ taskDetail.task.triggered_by }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ taskDetail.task.created_at }}</el-descriptions-item>
          <el-descriptions-item label="开始时间">{{ taskDetail.task.started_at || '—' }}</el-descriptions-item>
          <el-descriptions-item label="完成时间">{{ taskDetail.task.finished_at || '—' }}</el-descriptions-item>
          <el-descriptions-item label="From Commit" :span="2">
            <code style="font-size:11px">{{ taskDetail.task.from_commit || '—' }}</code>
            <span v-if="taskDetail.task.from_subject" style="color:#64748b;font-size:12px;margin-left:8px">{{ taskDetail.task.from_subject }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="To Commit" :span="2">
            <code style="font-size:11px">{{ taskDetail.task.to_commit || '—' }}</code>
            <span v-if="taskDetail.task.to_subject" style="color:#64748b;font-size:12px;margin-left:8px">{{ taskDetail.task.to_subject }}</span>
          </el-descriptions-item>
          <el-descriptions-item v-if="taskDetail.task.error_message" label="错误信息" :span="2">
            <span style="color:#dc2626;font-size:12px">{{ taskDetail.task.error_message }}</span>
          </el-descriptions-item>
          <el-descriptions-item v-if="taskDetail.task.input_tokens || taskDetail.task.output_tokens" label="Token 消耗" :span="2">
            输入 {{ taskDetail.task.input_tokens }} / 输出 {{ taskDetail.task.output_tokens }}
          </el-descriptions-item>
        </el-descriptions>
      </template>

      <!-- Scan 详情 -->
      <template v-if="detailKind === 'scan' && scanDetail">
        <el-descriptions :column="2" border size="small" style="margin-bottom:14px">
          <el-descriptions-item label="状态">
            <span :class="`sts--${scanDetail.job.status}`" style="padding:1px 8px;border-radius:99px;font-size:12px">
              {{ statusLabel(scanDetail.job.status) }}
            </span>
          </el-descriptions-item>
          <el-descriptions-item label="触发时间">{{ scanDetail.job.triggered_at }}</el-descriptions-item>
          <el-descriptions-item label="完成时间">{{ scanDetail.job.finished_at || '—' }}</el-descriptions-item>
          <el-descriptions-item label="分支情况">
            {{ scanDetail.job.changed_branch_count }} / {{ scanDetail.job.branch_count }} 分支有改动
          </el-descriptions-item>
          <el-descriptions-item v-if="scanDetail.job.error_message" label="错误信息" :span="2">
            <span style="color:#dc2626;font-size:12px">{{ scanDetail.job.error_message }}</span>
          </el-descriptions-item>
          <el-descriptions-item v-if="scanDetail.job.report_path" label="报告路径" :span="2">
            <a :href="scanDetail.job.report_path" target="_blank" style="color:#2563eb;font-size:12px">{{ scanDetail.job.report_path }}</a>
          </el-descriptions-item>
        </el-descriptions>

        <div v-if="scanDetail.results && scanDetail.results.length > 0" style="max-height:360px;overflow-y:auto">
          <div v-for="br in scanDetail.results" :key="br.id" class="dlg-branch-item">
            <div class="dlg-branch-header">
              <span class="branch-name">{{ br.branch_name }}</span>
              <span :class="['risk-pill', riskClass(br.risk_level)]">{{ riskLabel(br.risk_level) }}</span>
              <span style="font-size:12px;color:#64748b">{{ br.commit_count }} commits</span>
              <span style="font-size:11px;color:#94a3b8;margin-left:auto">{{ stageLabel(br.analysis_stage) }}</span>
            </div>
            <div class="dlg-branch-result" v-html="renderMd(br.result)" />
          </div>
        </div>
        <div v-else-if="!detailLoading" style="text-align:center;color:#94a3b8;font-size:13px;padding:16px 0">
          无有改动的分支
        </div>
      </template>

    </div>

    <template #footer>
      <el-button @click="detailVisible = false">关闭</el-button>
      <el-button type="primary" @click="goDetailPage">查看完整详情</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  FolderOpened, Loading, CircleCheck, CircleClose,
  WarningFilled, Cpu, Key, Search, Refresh, User,
} from '@element-plus/icons-vue'
import { getDashboard } from '../api/dashboard'
import { getTask } from '../api/tasks'
import { getScanJob } from '../api/scan'

const router = useRouter()

// ---- 动态详情弹窗 ----
const detailVisible = ref(false)
const detailLoading = ref(false)
const detailKind = ref('')   // 'task' | 'scan'
const taskDetail = ref(null)
const scanDetail = ref(null)  // { job, results }

function renderMd(text) {
  if (!text) return ''
  return text
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\n/g, '<br>')
}

async function goActivity(act) {
  detailKind.value = act.kind
  detailVisible.value = true
  detailLoading.value = true
  taskDetail.value = null
  scanDetail.value = null
  try {
    if (act.kind === 'task') {
      const res = await getTask(act.id)
      taskDetail.value = res.data
    } else if (act.kind === 'scan') {
      const res = await getScanJob(act.id)
      scanDetail.value = res.data
    }
  } catch {
    ElMessage.error('加载详情失败')
    detailVisible.value = false
  } finally {
    detailLoading.value = false
  }
}

function goDetailPage() {
  detailVisible.value = false
  if (detailKind.value === 'task' && taskDetail.value) {
    router.push(`/tasks/${taskDetail.value.task.id}`)
  } else if (detailKind.value === 'scan' && scanDetail.value) {
    router.push(`/scan?job_id=${scanDetail.value.job.id}`)
  }
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
const loading = ref(false)
const username = localStorage.getItem('username') || 'admin'
const dateLabel = ref('')
const stats = ref({
  project_count: 0, running_count: 0,
  today_completed_count: 0, failed_count: 0,
  week_failed_count: 0, week_risk_count: 0,
  model_count: 0, credential_count: 0, scan_enabled_count: 0,
  user_count: 0, sensitive_word_count: 0, scan_total_count: 0,
})
const activities = ref([])
const heatmap = ref([])

const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 6)  return '夜深了'
  if (h < 12) return '早上好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
})

const statItems = computed(() => [
  { label: '扫描项目总数', value: stats.value.project_count,       color: '#2563eb', icon: FolderOpened,  route: '/projects' },
  { label: '巡检项目总数', value: stats.value.scan_total_count,    color: '#7c3aed', icon: Search,        route: '/scan' },
  { label: '用户数量',     value: stats.value.user_count,          color: '#0891b2', icon: User,          route: '/users' },
  { label: '敏感词数量',   value: stats.value.sensitive_word_count, color: '#d97706', icon: WarningFilled, route: '/sensitive-words' },
  { label: '高风险分支',   value: stats.value.week_risk_count,      color: '#dc2626', icon: CircleClose,   route: '/scan' },
])

const maxHeat = computed(() => Math.max(...heatmap.value.map(d => d.completed + d.failed), 1))

function heatLevel(n) {
  if (n === 0) return 'lv0'
  if (n <= 2)  return 'lv1'
  if (n <= 5)  return 'lv2'
  return 'lv3'
}

function barHeight(n) {
  return Math.round((n / maxHeat.value) * 100)
}

function shortDate(s) {
  const d = new Date(s)
  return `${d.getMonth()+1}/${d.getDate()}`
}

const statusLabel = (s) => ({
  completed: '完成', running: '运行中', pending: '等待',
  failed: '失败', cancelled: '取消',
}[s] || s)

// ---- 自动刷新 ----
const autoRefreshInterval = ref(30)
let autoRefreshTimer = null

function onRefreshChange(val) {
  if (autoRefreshTimer) { clearInterval(autoRefreshTimer); autoRefreshTimer = null }
  if (val > 0) {
    autoRefreshTimer = setInterval(fetchData, val * 1000)
  }
}

let timer = null
const hasRunning = computed(() => stats.value.running_count > 0)
watch(hasRunning, active => {
  if (active && !timer) timer = setInterval(fetchData, 6000)
  else if (!active && timer) { clearInterval(timer); timer = null }
})

async function fetchData() {
  loading.value = true
  try {
    const { data } = await getDashboard()
    stats.value    = data.stats
    activities.value = data.activities || []
    heatmap.value  = data.heatmap || []
    dateLabel.value = data.date
  } catch {
    ElMessage.error('数据加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchData()
  onRefreshChange(autoRefreshInterval.value)
})
onUnmounted(() => { clearInterval(timer); clearInterval(autoRefreshTimer) })
</script>

<style scoped>
.dashboard { min-height: 100vh; background: #f1f5f9; }

/* ── 欢迎横幅 ── */
.hero {
  position: relative;
  overflow: hidden;
  background: linear-gradient(135deg, #0f172a 0%, #1e3a8a 50%, #4c1d95 100%);
  padding: 28px 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.hero-bg { position: absolute; inset: 0; pointer-events: none; }
.hb {
  position: absolute;
  border-radius: 50%;
  background: rgba(255,255,255,0.06);
}
.hb1 { width: 320px; height: 320px; right: -80px; top: -120px; }
.hb2 { width: 180px; height: 180px; right: 120px; bottom: -80px; }
.hb3 { width: 90px;  height: 90px;  right: 280px; top: 10px; background: rgba(255,255,255,0.04); }

.greeting {
  font-size: 22px; font-weight: 700; color: #fff; margin-bottom: 6px;
  position: relative; z-index: 1;
}
.username { color: #93c5fd; }
.hero-date { font-size: 13px; color: rgba(255,255,255,0.55); position: relative; z-index: 1; }

.hero-right { position: relative; z-index: 1; }
.hero-right-col { display: flex; flex-direction: column; align-items: flex-end; gap: 10px; }
.hero-badge {
  display: flex; align-items: center; gap: 8px;
  background: rgba(255,255,255,0.12);
  border: 1px solid rgba(255,255,255,0.2);
  border-radius: 20px;
  padding: 6px 14px;
  font-size: 13px; color: #fff;
}
.hero-badge--idle { opacity: 0.6; }
.badge-dot {
  width: 8px; height: 8px; border-radius: 50%;
}
.badge-dot.blink { background: #34d399; animation: pulse 1.5s infinite; }
.badge-dot.idle  { background: #94a3b8; }
@keyframes pulse {
  0%,100% { opacity: 1; transform: scale(1); }
  50%      { opacity: 0.5; transform: scale(1.3); }
}

.refresh-selector {
  display: flex; align-items: center; gap: 6px;
  background: rgba(255,255,255,0.1);
  border: 1px solid rgba(255,255,255,0.2);
  border-radius: 20px;
  padding: 4px 10px 4px 12px;
}
.refresh-icon { color: rgba(255,255,255,0.75); font-size: 14px; flex-shrink: 0; }
.refresh-icon.spinning { animation: spin 2s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
.refresh-select { width: 100px; }
.refresh-selector :deep(.el-select .el-input__wrapper) {
  background: transparent !important;
  box-shadow: none !important;
  padding: 0 4px;
}
.refresh-selector :deep(.el-select .el-input__inner) {
  color: rgba(255,255,255,0.85) !important;
  font-size: 12px;
}
.refresh-selector :deep(.el-select .el-input__suffix) { color: rgba(255,255,255,0.6); }

/* ── 状态总览条 ── */
.stat-bar {
  display: flex;
  gap: 0;
  margin: 20px 40px;
  background: #fff;
  border-radius: 14px;
  border: 1px solid #e2e8f0;
  overflow: hidden;
  box-shadow: 0 1px 4px rgba(0,0,0,0.05);
}
.stat-item {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 18px 20px;
  border-right: 1px solid #f1f5f9;
  position: relative;
  transition: background 0.15s;
}
.stat-item:last-child { border-right: none; }
.stat-item.clickable { cursor: pointer; }
.stat-item.clickable:hover { background: #f8fafc; }

.stat-accent {
  position: absolute; left: 0; top: 20%; bottom: 20%;
  width: 3px; border-radius: 0 3px 3px 0;
  background: var(--c);
}
.stat-body { flex: 1; padding-left: 4px; }
.stat-num {
  font-size: 26px; font-weight: 800; color: var(--c);
  line-height: 1; margin-bottom: 3px;
  font-variant-numeric: tabular-nums;
}
.stat-label { font-size: 12px; color: #64748b; font-weight: 500; }
.stat-icon { font-size: 20px; color: var(--c); opacity: 0.25; flex-shrink: 0; }

/* ── 主网格 ── */
.main-grid {
  display: grid;
  grid-template-columns: 1fr 340px;
  gap: 18px;
  margin: 0 40px 32px;
}

/* ── 通用 panel ── */
.panel {
  background: #fff;
  border-radius: 14px;
  border: 1px solid #e2e8f0;
  overflow: hidden;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
}
.panel-head {
  display: flex; align-items: baseline; gap: 8px;
  padding: 16px 20px 12px;
  border-bottom: 1px solid #f1f5f9;
}
.panel-title { font-size: 14px; font-weight: 700; color: #0f172a; }
.panel-sub   { font-size: 12px; color: #94a3b8; }

/* ── 时间轴 ── */
.timeline-panel { min-height: 480px; }
.timeline { padding: 8px 0; }
.empty { padding: 48px; text-align: center; color: #94a3b8; font-size: 14px; }

.tl-item {
  display: flex;
  gap: 0;
  padding: 0 20px;
  cursor: pointer;
}
.tl-item:hover .tl-content { background: #f8fafc; border-radius: 8px; }

.tl-line-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 28px;
  flex-shrink: 0;
  padding-top: 14px;
}
.tl-dot {
  width: 10px; height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
  border: 2px solid #fff;
  box-shadow: 0 0 0 2px currentColor;
}
.tl-dot--completed { color: #10b981; background: #10b981; }
.tl-dot--running   { color: #f59e0b; background: #f59e0b; }
.tl-dot--pending   { color: #3b82f6; background: #3b82f6; }
.tl-dot--failed    { color: #ef4444; background: #ef4444; }
.tl-dot--cancelled { color: #94a3b8; background: #94a3b8; }
.tl-dot--scan      { color: #8b5cf6; background: #8b5cf6; }

.tl-line {
  flex: 1;
  width: 1px;
  background: #e2e8f0;
  min-height: 16px;
  margin-top: 4px;
}
.tl-item:last-child .tl-line { display: none; }

.tl-content {
  flex: 1;
  padding: 10px 10px 10px 8px;
  min-width: 0;
}
.tl-top {
  display: flex; align-items: center; gap: 6px;
  flex-wrap: wrap; margin-bottom: 4px;
}
.tl-kind {
  font-size: 10px; font-weight: 700; padding: 1px 6px;
  border-radius: 4px; letter-spacing: 0.05em;
}
.kind--task { background: #dbeafe; color: #1d4ed8; }
.kind--scan { background: #ede9fe; color: #7c3aed; }

.tl-title { font-size: 13px; font-weight: 600; color: #1e293b; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tl-risk  { font-size: 11px; color: #dc2626; font-weight: 600; white-space: nowrap; }
.tl-status { font-size: 11px; padding: 1px 7px; border-radius: 99px; white-space: nowrap; }
.sts--completed { background: #dcfce7; color: #166534; }
.sts--running   { background: #fef3c7; color: #92400e; }
.sts--pending   { background: #dbeafe; color: #1e40af; }
.sts--failed    { background: #fee2e2; color: #991b1b; }
.sts--cancelled { background: #f1f5f9; color: #64748b; }

.tl-bottom { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.tl-sub { font-family: monospace; font-size: 11px; color: #94a3b8; background: #f8fafc; padding: 1px 5px; border-radius: 3px; }
.tl-time { font-size: 11px; color: #cbd5e1; white-space: nowrap; }

/* ── 右侧栏 ── */
.right-col { display: flex; flex-direction: column; gap: 18px; }

/* ── 系统资产 ── */
.asset-grid {
  display: grid; grid-template-columns: repeat(3, 1fr);
  gap: 1px; background: #f1f5f9;
}
.asset-item {
  display: flex; flex-direction: column; align-items: center;
  gap: 6px; padding: 20px 12px;
  background: #fff;
  cursor: pointer; transition: background 0.15s;
}
.asset-item:hover { background: #f8fafc; }
.asset-icon { font-size: 22px; color: var(--ac); opacity: 0.7; }
.asset-num  { font-size: 28px; font-weight: 800; color: var(--ac); line-height: 1; }
.asset-label { font-size: 11px; color: #94a3b8; font-weight: 500; }

/* ── 热力图 ── */
.heatmap {
  display: flex;
  justify-content: space-around;
  align-items: flex-end;
  padding: 12px 16px 4px;
  gap: 6px;
}
.hm-day {
  flex: 1;
  display: flex; flex-direction: column; align-items: center;
  gap: 4px; position: relative;
}
.hm-bar {
  width: 100%; height: 64px;
  display: flex; flex-direction: column; justify-content: flex-end;
  background: #f1f5f9; border-radius: 6px; overflow: hidden;
  padding: 2px;
}
.hm-completed {
  width: 100%; background: #10b981; border-radius: 4px 4px 0 0;
  min-height: 2px; transition: height 0.4s ease;
}
.hm-failed {
  width: 100%; background: #f87171; border-radius: 4px 4px 0 0;
  min-height: 0; transition: height 0.4s ease;
}
.hm-date  { font-size: 10px; color: #94a3b8; }
.hm-count { font-size: 11px; color: #475569; font-weight: 600; }

.hm-legend {
  display: flex; align-items: center; gap: 12px;
  padding: 4px 16px 12px;
  font-size: 11px; color: #94a3b8;
}
.leg-dot { width: 8px; height: 8px; border-radius: 2px; display: inline-block; margin-right: 3px; }
.leg-completed { background: #10b981; }
.leg-failed    { background: #f87171; }

/* ── 弹窗：巡检分支结果 ── */
.dlg-branch-item {
  background: #fff; border: 1px solid #e2e8f0; border-radius: 8px;
  margin-bottom: 10px; overflow: hidden;
}
.dlg-branch-header {
  display: flex; align-items: center; gap: 10px;
  padding: 9px 14px; background: #f8fafc; border-bottom: 1px solid #f1f5f9;
}
.dlg-branch-result {
  padding: 10px 14px; font-size: 13px; color: #374151; line-height: 1.7;
}
.dlg-branch-result :deep(code) {
  background: #f1f5f9; padding: 1px 5px; border-radius: 3px;
  font-family: monospace; font-size: 12px;
}
</style>
