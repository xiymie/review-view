<template>
  <div class="page-wrap">
    <!-- 渐变欢迎横幅 -->
    <div class="hero-banner">
      <div class="hero-text">
        <h1 class="hero-title">仪表盘</h1>
        <p class="hero-sub">{{ date }} · 欢迎回来，<strong>{{ username }}</strong></p>
      </div>
      <div class="hero-deco">
        <div class="deco-circle c1"></div>
        <div class="deco-circle c2"></div>
        <div class="deco-circle c3"></div>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stat-grid" v-loading="loading">
      <div
        class="stat-card"
        v-for="(c, i) in statCards"
        :key="c.label"
        :style="{ '--card-color': c.color, '--card-bg': c.bg, '--delay': `${i * 80}ms` }"
        :class="{ 'card-in': mounted }"
      >
        <div class="stat-icon-wrap">
          <el-icon :size="20"><component :is="c.icon" /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ displayValues[i] }}</div>
          <div class="stat-label">{{ c.label }}</div>
        </div>
        <div class="stat-glow"></div>
      </div>
    </div>

    <!-- 最近任务 -->
    <div class="section-card" :class="{ 'card-in': mounted }" style="--delay: 320ms">
      <div class="section-header">
        <div class="section-title-wrap">
          <span class="section-dot"></span>
          <span class="section-title">最近任务</span>
        </div>
        <el-button link type="primary" size="small" @click="$router.push('/tasks')">查看全部 →</el-button>
      </div>
      <el-table :data="recentTasks" v-loading="loading" size="small">
        <el-table-column label="项目" min-width="120">
          <template #default="{ row }">
            <span class="link-text" @click="$router.push(`/projects/${row.project_id}`)">{{ row.project_name }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="isAdmin" label="所属用户" width="96">
          <template #default="{ row }">
            <span class="owner-tag">{{ row.owner_username || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="Commit 范围" min-width="150">
          <template #default="{ row }">
            <code class="code-tag">{{ commitRange(row) }}</code>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="96">
          <template #default="{ row }">
            <span class="status-pill" :class="`pill-${row.status}`">{{ statusLabel(row.status) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="触发" prop="triggered_by" width="80">
          <template #default="{ row }">
            <span class="trigger-tag">{{ row.triggered_by }}</span>
          </template>
        </el-table-column>
        <el-table-column label="时间" prop="created_at" min-width="148" />
        <el-table-column label="" width="72">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="$router.push(`/tasks/${row.id}`)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!loading && recentTasks.length === 0" class="empty-tip">暂无任务记录</div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Folder, Loading, Check, WarningFilled } from '@element-plus/icons-vue'
import { getDashboard } from '../api/dashboard'

const loading = ref(false)
const date = ref('')
const mounted = ref(false)
const username = localStorage.getItem('username') || 'admin'
const stats = ref({ project_count: 0, running_count: 0, today_completed_count: 0, failed_count: 0 })
const recentTasks = ref([])
const displayValues = ref([0, 0, 0, 0])

const isAdmin = computed(() => {
  const role = localStorage.getItem('role') || ''
  return role === 'admin' || role === 'super_admin'
})

const statCards = computed(() => [
  { label: '项目总数',  value: stats.value.project_count,        icon: Folder,        color: '#2563eb', bg: 'linear-gradient(135deg,#eff6ff,#dbeafe)' },
  { label: '运行中',   value: stats.value.running_count,         icon: Loading,       color: '#d97706', bg: 'linear-gradient(135deg,#fffbeb,#fef3c7)' },
  { label: '今日完成', value: stats.value.today_completed_count, icon: Check,         color: '#059669', bg: 'linear-gradient(135deg,#ecfdf5,#d1fae5)' },
  { label: '失败',     value: stats.value.failed_count,          icon: WarningFilled, color: '#dc2626', bg: 'linear-gradient(135deg,#fef2f2,#fee2e2)' },
])

// 数字滚动动效
function animateCount(targets) {
  const duration = 600
  const start = Date.now()
  const from = [...displayValues.value]
  const tick = () => {
    const elapsed = Date.now() - start
    const progress = Math.min(elapsed / duration, 1)
    const ease = 1 - Math.pow(1 - progress, 3)
    displayValues.value = targets.map((t, i) => Math.round(from[i] + (t - from[i]) * ease))
    if (progress < 1) requestAnimationFrame(tick)
  }
  requestAnimationFrame(tick)
}

const statusLabel = (s) => ({ completed: '已完成', running: '运行中', pending: '等待中', failed: '失败', cancelled: '已取消' }[s] || s)
const commitRange = (row) => {
  if (row.from_commit) return `${row.from_commit.slice(0,7)}..${row.to_commit.slice(0,7)}`
  return row.to_commit?.slice(0, 7)
}

let autoRefreshTimer = null
const hasRunning = computed(() => recentTasks.value.some(t => t.status === 'running' || t.status === 'pending'))
watch(hasRunning, (active) => {
  if (active && !autoRefreshTimer) {
    autoRefreshTimer = setInterval(fetchData, 5000)
  } else if (!active && autoRefreshTimer) {
    clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
})

const fetchData = async () => {
  loading.value = true
  try {
    const { data } = await getDashboard()
    stats.value = data.stats
    recentTasks.value = data.recent_tasks
    date.value = data.date
    animateCount([data.stats.project_count, data.stats.running_count, data.stats.today_completed_count, data.stats.failed_count])
  } catch { ElMessage.error('数据加载失败') }
  finally { loading.value = false }
}

onMounted(async () => {
  await fetchData()
  setTimeout(() => { mounted.value = true }, 50)
})
onUnmounted(() => clearInterval(autoRefreshTimer))
</script>

<style scoped>
.page-wrap { padding: 0; }

/* ── 欢迎横幅 ── */
.hero-banner {
  position: relative;
  background: linear-gradient(135deg, #1e3a8a 0%, #2563eb 45%, #7c3aed 100%);
  padding: 28px 36px 24px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.hero-text { position: relative; z-index: 2; }
.hero-title {
  margin: 0 0 4px;
  font-size: 24px;
  font-weight: 700;
  color: #fff;
  letter-spacing: -0.5px;
}
.hero-sub { margin: 0; font-size: 14px; color: rgba(255,255,255,0.75); }
.hero-sub strong { color: #fff; }

/* 装饰圆 */
.hero-deco { position: absolute; right: 0; top: 0; bottom: 0; width: 320px; }
.deco-circle {
  position: absolute;
  border-radius: 50%;
  opacity: 0.12;
  background: #fff;
}
.c1 { width: 200px; height: 200px; right: -40px; top: -60px; }
.c2 { width: 140px; height: 140px; right: 80px; bottom: -50px; }
.c3 { width: 80px;  height: 80px;  right: 180px; top: 10px; opacity: 0.08; }

/* 内容区 padding */
.stat-grid,
.section-card { margin: 0 36px; }

/* ── 统计卡片 ── */
.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  margin-top: 24px;
  margin-bottom: 20px;
}

/* 卡片入场动画 */
.stat-card, .section-card {
  opacity: 0;
  animation: none;
}
.card-in.stat-card { animation: slide-in 0.45s ease forwards; animation-delay: var(--delay, 0ms); }
.card-in.section-card { animation: slide-in 0.45s ease forwards; animation-delay: var(--delay, 0ms); }

@keyframes slide-in {
  from { opacity: 0; transform: translateY(12px); }
  to   { opacity: 1; transform: translateY(0); }
}

.stat-card {
  background: var(--card-bg);
  border: 1px solid rgba(255,255,255,0.6);
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  position: relative;
  overflow: hidden;
  cursor: default;
  transition: box-shadow 0.2s, transform 0.2s;
}

.stat-card:hover {
  box-shadow: 0 4px 20px rgba(0,0,0,0.08);
  transform: translateY(-2px);
}

.stat-glow {
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, transparent 60%, rgba(255,255,255,0.3));
  pointer-events: none;
}

.stat-icon-wrap {
  width: 46px; height: 46px;
  border-radius: 11px;
  background: white;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--card-color);
}

.stat-value {
  font-size: 30px;
  font-weight: 800;
  color: var(--card-color);
  line-height: 1;
  margin-bottom: 4px;
  font-variant-numeric: tabular-nums;
}

.stat-label {
  font-size: 12.5px;
  color: #64748b;
  font-weight: 500;
}

/* ── 任务列表卡片 ── */
.section-card {
  background: #fff;
  border: 1px solid #e8edf4;
  border-radius: 12px;
  overflow: hidden;
  margin-bottom: 24px;
  opacity: 0;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid #f1f5f9;
}

.section-title-wrap { display: flex; align-items: center; gap: 8px; }
.section-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
}
.section-title { font-size: 14px; font-weight: 600; color: #1e293b; }

/* 表格内元素 */
.link-text {
  color: #2563eb;
  font-weight: 500;
  cursor: pointer;
  font-size: 13px;
}
.link-text:hover { text-decoration: underline; }

.code-tag {
  font-family: monospace;
  font-size: 12px;
  background: #f1f5f9;
  color: #475569;
  padding: 2px 7px;
  border-radius: 5px;
}

.status-pill {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}
.pill-completed { background: #ecfdf5; color: #059669; }
.pill-running   { background: #fffbeb; color: #d97706; }
.pill-pending   { background: #eff6ff; color: #2563eb; }
.pill-failed    { background: #fef2f2; color: #dc2626; }
.pill-cancelled { background: #f8fafc; color: #64748b; }

.trigger-tag { font-size: 12px; color: #94a3b8; }
.owner-tag { font-size: 12px; color: #6366f1; font-weight: 500; }

.empty-tip {
  padding: 40px;
  text-align: center;
  color: #94a3b8;
  font-size: 14px;
}
</style>
