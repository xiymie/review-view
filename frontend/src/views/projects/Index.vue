<template>
  <div class="page-wrap">
    <div class="page-hero">
      <div class="hero-content">
        <h1 class="hero-title">项目</h1>
        <p class="hero-sub">管理所有代码审核项目</p>
      </div>
      <div class="hero-actions">
        <el-button type="primary" @click="router.push('/projects/new')">新建项目</el-button>
      </div>
      <div class="deco-circles">
        <div class="deco c1"></div>
        <div class="deco c2"></div>
      </div>
    </div>

    <div class="table-wrap">
      <el-table :data="projects" v-loading="loading" style="width:100%">
        <el-table-column label="名称" min-width="160">
          <template #default="{ row }">
            <span class="proj-link" @click="router.push(`/projects/${row.id}`)">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="仓库地址" prop="repo_url" min-width="220" show-overflow-tooltip />
        <el-table-column label="分支" prop="branch" width="120" />
        <el-table-column v-if="isAdmin" label="所属用户" width="100">
          <template #default="{ row }">
            <span class="owner-tag">{{ row.owner_username || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="最近 Review" width="120">
          <template #default="{ row }">
            <code v-if="row.last_reviewed_commit" class="commit-code">{{ row.last_reviewed_commit.slice(0,7) }}</code>
            <span v-else class="none-text">—</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <span class="status-pill" :class="`pill-${row.status}`">{{ statusLabel(row.status) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <div class="action-cell">
              <el-button link type="primary" size="small" @click="router.push(`/projects/${row.id}`)">详情</el-button>
              <span class="sep">·</span>
              <el-button link size="small" @click="router.push(`/projects/new?clone_from=${row.id}`)">克隆</el-button>
              <span class="sep">·</span>
              <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!loading && projects.length === 0" class="empty-state">
        <div class="empty-icon">📁</div>
        <p>暂无项目</p>
        <el-button type="primary" size="small" @click="router.push('/projects/new')">新建第一个项目</el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import { listProjects, deleteProject } from '../../api/projects'

const router = useRouter()
const projects = ref([])
const loading = ref(false)

const isAdmin = computed(() => {
  const role = localStorage.getItem('role') || ''
  return role === 'admin' || role === 'super_admin'
})

async function loadProjects() {
  loading.value = true
  try {
    const res = await listProjects()
    projects.value = res.data
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadProjects)

function statusLabel(status) {
  const map = { ready: '正常', initializing: '初始化中', init_failed: '初始化失败', completed: '已完成', running: '运行中', pending: '等待中', failed: '失败', cancelled: '已取消' }
  return map[status] ?? status
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`确定要删除项目 "${row.name}" 吗？`, '删除确认', { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' })
    await deleteProject(row.id)
    ElMessage.success('删除成功')
    await loadProjects()
  } catch (err) {
    if (err !== 'cancel' && err?.type !== 'cancel') {
      if (err.response) ElMessage.error(err.response?.data?.message || '删除失败')
    }
  }
}
</script>

<style scoped>
.page-wrap { padding: 0; }

.page-hero {
  position: relative;
  background: linear-gradient(135deg, #1e3a8a, #2563eb, #0891b2);
  padding: 24px 36px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  overflow: hidden;
}
.hero-title { font-size: 22px; font-weight: 700; color: #fff; margin: 0 0 4px; }
.hero-sub { font-size: 13px; color: rgba(255,255,255,0.75); margin: 0; }
.hero-actions { position: relative; z-index: 2; }
.hero-actions :deep(.el-button--primary) { background: rgba(255,255,255,0.2) !important; border-color: rgba(255,255,255,0.3) !important; color: #fff !important; }
.hero-actions :deep(.el-button--primary:hover) { background: rgba(255,255,255,0.3) !important; }
.deco-circles { position: absolute; right: 0; top: 0; bottom: 0; width: 200px; pointer-events: none; }
.deco { position: absolute; border-radius: 50%; background: rgba(255,255,255,0.08); }
.c1 { width: 180px; height: 180px; right: -40px; top: -60px; }
.c2 { width: 100px; height: 100px; right: 60px; bottom: -30px; }

.table-wrap {
  background: #fff;
  border: 1px solid #e8edf4;
  border-radius: 10px;
  overflow: hidden;
  margin: 20px 36px;
}

.proj-link {
  color: #2563eb;
  font-weight: 500;
  font-size: 13.5px;
  cursor: pointer;
}
.proj-link:hover { text-decoration: underline; }

.commit-code {
  font-family: monospace;
  font-size: 12px;
  background: #f1f5f9;
  color: #475569;
  padding: 2px 7px;
  border-radius: 5px;
}

.none-text { color: #94a3b8; font-size: 13px; }
.owner-tag { font-size: 12.5px; color: #6366f1; font-weight: 500; }

.status-pill {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
}
.pill-ready         { background: #ecfdf5; color: #059669; }
.pill-initializing  { background: #fffbeb; color: #d97706; }
.pill-init_failed   { background: #fef2f2; color: #dc2626; }
.pill-completed     { background: #ecfdf5; color: #059669; }
.pill-running       { background: #fffbeb; color: #d97706; }
.pill-pending       { background: #eff6ff; color: #2563eb; }
.pill-failed        { background: #fef2f2; color: #dc2626; }
.pill-cancelled     { background: #f8fafc; color: #64748b; }

.action-cell { display: flex; align-items: center; gap: 3px; }
.sep { color: #e2e8f0; font-size: 12px; }

.empty-state { padding: 48px; text-align: center; }
.empty-icon  { font-size: 36px; margin-bottom: 10px; }
.empty-state p { margin: 0 0 12px; font-size: 14px; color: #94a3b8; }
</style>
