<template>
  <div class="page-wrap">
    <div class="page-hero">
      <div class="hero-content">
        <h1 class="hero-title">审核任务</h1>
        <p class="hero-sub">所有代码审查任务的运行记录</p>
      </div>
      <div class="hero-actions">
        <el-button @click="loadTasks" :loading="loading">
          <el-icon><Refresh /></el-icon>&nbsp;刷新
        </el-button>
      </div>
      <div class="deco-circles">
        <div class="deco c1"></div>
        <div class="deco c2"></div>
      </div>
    </div>

    <!-- 统计 -->
    <div class="stat-row" :class="{ 'stat-in': mounted }">
      <div class="stat-item" v-for="(s, i) in stats" :key="s.label" :style="{'--dot': s.color, '--delay': `${i * 60}ms`}">
        <span class="stat-num" :style="{ color: s.color }">{{ displayStats[i] }}</span>
        <span class="stat-lbl">{{ s.label }}</span>
      </div>
    </div>

    <!-- 筛选 -->
    <div class="filter-bar">
      <el-select v-model="filterStatus" placeholder="全部状态" clearable size="small" style="width:130px">
        <el-option label="运行中" value="running" />
        <el-option label="等待中" value="pending" />
        <el-option label="已完成" value="completed" />
        <el-option label="失败" value="failed" />
        <el-option label="已取消" value="cancelled" />
      </el-select>
      <el-select v-model="filterProject" placeholder="全部项目" clearable size="small" style="width:160px">
        <el-option v-for="p in projectOptions" :key="p" :label="p" :value="p" />
      </el-select>
      <el-button v-if="filterStatus || filterProject" size="small" text @click="filterStatus='';filterProject=''">清除</el-button>
      <span class="filter-count">{{ filteredTasks.length }} 条</span>
    </div>

    <!-- 表格 -->
    <div class="table-wrap">
      <el-table :data="filteredTasks" v-loading="loading" style="width:100%">
        <el-table-column label="#" width="64">
          <template #default="{ row }">
            <span class="task-id">#{{ row.id }}</span>
          </template>
        </el-table-column>

        <el-table-column label="项目" min-width="130">
          <template #default="{ row }">
            <span class="proj-link" @click="router.push(`/projects/${row.project_id}`)">{{ row.project_name }}</span>
          </template>
        </el-table-column>

        <el-table-column v-if="isAdmin" label="所属用户" width="100">
          <template #default="{ row }">
            <span class="owner-tag">{{ row.owner_username || '—' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Commit 范围" min-width="180">
          <template #default="{ row }">
            <code class="code-tag">{{ commitRange(row) }}</code>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="106">
          <template #default="{ row }">
            <span class="status-pill" :class="`pill-${row.status}`">
              <i class="status-dot"></i>{{ statusLabel(row.status) }}
            </span>
          </template>
        </el-table-column>

        <el-table-column label="触发" width="80">
          <template #default="{ row }">
            <span class="meta-text">{{ row.triggered_by }}</span>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" width="155" prop="created_at" />

        <el-table-column label="操作" width="160">
          <template #default="{ row }">
            <div class="action-cell">
              <el-button link type="primary" size="small" @click="router.push(`/tasks/${row.id}`)">查看</el-button>
              <span class="sep">·</span>
              <el-button v-if="row.status === 'running'" link type="warning" size="small" @click="cancelTask(row)">取消</el-button>
              <el-button v-if="row.status === 'failed' || row.status === 'cancelled'" link type="primary" size="small" @click="retryTask(row)">重试</el-button>
              <template v-if="!['running','pending'].includes(row.status)">
                <span class="sep">·</span>
                <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
              </template>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!loading && filteredTasks.length === 0" class="empty-state">
        <div class="empty-icon">📋</div>
        <p>{{ tasks.length ? '没有符合筛选条件的任务' : '暂无任务记录' }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { listTasks, cancelTask as apiCancelTask, retryTask as apiRetryTask, deleteTask as apiDeleteTask } from '../../api/tasks'

const router = useRouter()
const tasks = ref([])
const loading = ref(false)
const filterStatus = ref('')
const filterProject = ref('')
const mounted = ref(false)
const displayStats = ref([0, 0, 0, 0])
let autoRefreshTimer = null

const isAdmin = computed(() => {
  const role = localStorage.getItem('role') || ''
  return role === 'admin' || role === 'super_admin'
})

const hasActiveTasks = computed(() => tasks.value.some(t => t.status === 'running' || t.status === 'pending'))

watch(hasActiveTasks, (active) => {
  if (active && !autoRefreshTimer) {
    autoRefreshTimer = setInterval(loadTasks, 5000)
  } else if (!active && autoRefreshTimer) {
    clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
})

const stats = computed(() => [
  { label: '全部',   value: tasks.value.length,                                     color: '#94a3b8' },
  { label: '运行中', value: tasks.value.filter(t => t.status === 'running').length,  color: '#d97706' },
  { label: '已完成', value: tasks.value.filter(t => t.status === 'completed').length,color: '#059669' },
  { label: '失败',   value: tasks.value.filter(t => t.status === 'failed').length,   color: '#dc2626' },
])

function animateCount(targets) {
  const duration = 600
  const start = Date.now()
  const from = [...displayStats.value]
  const tick = () => {
    const elapsed = Date.now() - start
    const progress = Math.min(elapsed / duration, 1)
    const ease = 1 - Math.pow(1 - progress, 3)
    displayStats.value = targets.map((t, i) => Math.round(from[i] + (t - from[i]) * ease))
    if (progress < 1) requestAnimationFrame(tick)
  }
  requestAnimationFrame(tick)
}

const projectOptions = computed(() => [...new Set(tasks.value.map(t => t.project_name).filter(Boolean))])

const filteredTasks = computed(() => tasks.value.filter(t => {
  if (filterStatus.value && t.status !== filterStatus.value) return false
  if (filterProject.value && t.project_name !== filterProject.value) return false
  return true
}))

const loadTasks = async () => {
  loading.value = true
  try {
    const { data } = await listTasks()
    tasks.value = data
    animateCount([data.length, data.filter(t => t.status === 'running').length, data.filter(t => t.status === 'completed').length, data.filter(t => t.status === 'failed').length])
  } catch (e) {
    ElMessage.error('加载任务列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadTasks()
  setTimeout(() => { mounted.value = true }, 50)
})
onUnmounted(() => clearInterval(autoRefreshTimer))

const commitRange = (row) => {
  if (row.from_commit) return `${row.from_commit.slice(0,7)}..${row.to_commit.slice(0,7)}`
  return row.to_commit?.slice(0, 7)
}

const statusLabel = (status) => {
  const map = { completed: '已完成', running: '运行中', pending: '等待中', failed: '失败', cancelled: '已取消' }
  return map[status] ?? status
}

const cancelTask = async (row) => {
  try {
    await ElMessageBox.confirm('确认取消该任务？', '提示', { confirmButtonText: '确认', cancelButtonText: '取消', type: 'warning' })
    await apiCancelTask(row.id)
    await loadTasks()
  } catch (e) {
    if (e !== 'cancel' && e?.type !== 'cancel') ElMessage.error('取消任务失败')
  }
}

const retryTask = async (row) => {
  try {
    const { data } = await apiRetryTask(row.id)
    router.push('/tasks/' + data.task_id)
  } catch (e) {
    ElMessage.error('重试任务失败')
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除任务 #${row.id}？`, '删除确认', { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' })
    await apiDeleteTask(row.id)
    ElMessage.success('删除成功')
    await loadTasks()
  } catch (e) {
    if (e !== 'cancel' && e?.type !== 'cancel') ElMessage.error(e.response?.data?.message || '删除失败')
  }
}
</script>

<style scoped>
.page-wrap { padding: 0; }

.page-hero {
  position: relative;
  background: linear-gradient(135deg, #7c3aed, #6d28d9, #2563eb);
  padding: 24px 36px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  overflow: hidden;
}
.hero-title { font-size: 22px; font-weight: 700; color: #fff; margin: 0 0 4px; }
.hero-sub { font-size: 13px; color: rgba(255,255,255,0.75); margin: 0; }
.hero-actions { position: relative; z-index: 2; }
.hero-actions :deep(.el-button) { background: rgba(255,255,255,0.2) !important; border-color: rgba(255,255,255,0.3) !important; color: #fff !important; }
.hero-actions :deep(.el-button:hover) { background: rgba(255,255,255,0.3) !important; }
.deco-circles { position: absolute; right: 0; top: 0; bottom: 0; width: 200px; pointer-events: none; }
.deco { position: absolute; border-radius: 50%; background: rgba(255,255,255,0.08); }
.c1 { width: 180px; height: 180px; right: -40px; top: -60px; }
.c2 { width: 100px; height: 100px; right: 60px; bottom: -30px; }

/* 统计行 */
.stat-row {
  display: flex;
  gap: 28px;
  margin: 20px 36px 12px;
  opacity: 0;
  transform: translateY(8px);
  transition: opacity 0.4s ease, transform 0.4s ease;
}
.stat-row.stat-in {
  opacity: 1;
  transform: translateY(0);
}
.stat-item {
  display: flex;
  align-items: baseline;
  gap: 7px;
}
.stat-item::before {
  content: '';
  display: inline-block;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--dot);
  flex-shrink: 0;
  margin-bottom: 1px;
}
.stat-num  { font-size: 26px; font-weight: 800; font-variant-numeric: tabular-nums; }
.stat-lbl  { font-size: 13px; color: #64748b; }

/* 筛选栏 */
.filter-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0 36px 14px;
}
.filter-count { font-size: 13px; color: #94a3b8; margin-left: auto; }

/* 表格容器 */
.table-wrap {
  background: #fff;
  border: 1px solid #e8edf4;
  border-radius: 10px;
  overflow: hidden;
  margin: 0 36px 24px;
}

/* 行内元素 */
.task-id { font-family: monospace; font-size: 12px; color: #94a3b8; font-weight: 600; }

.proj-link {
  font-size: 13px;
  font-weight: 500;
  color: #2563eb;
  cursor: pointer;
}
.proj-link:hover { text-decoration: underline; }

.code-tag {
  font-family: monospace;
  font-size: 12px;
  background: #f1f5f9;
  color: #475569;
  padding: 2px 7px;
  border-radius: 5px;
}

/* 状态 pill */
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 9px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
}
.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  flex-shrink: 0;
}
.pill-completed { background: #ecfdf5; color: #059669; }
.pill-running   { background: #fffbeb; color: #d97706; }
.pill-pending   { background: #eff6ff; color: #2563eb; }
.pill-failed    { background: #fef2f2; color: #dc2626; }
.pill-cancelled { background: #f8fafc; color: #64748b; }
.pill-rejected  { background: #fff7ed; color: #ea580c; }

.meta-text { font-size: 12.5px; color: #94a3b8; }
.owner-tag { font-size: 12.5px; color: #6366f1; font-weight: 500; }

/* 操作列 */
.action-cell { display: flex; align-items: center; gap: 3px; white-space: nowrap; }
.sep { color: #e2e8f0; font-size: 12px; }

/* 空状态 */
.empty-state { padding: 48px; text-align: center; color: #94a3b8; }
.empty-icon  { font-size: 36px; margin-bottom: 10px; }
.empty-state p { margin: 0; font-size: 14px; }
</style>
