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

    <div class="summary-row">
      <div class="summary-card"><span>项目总数</span><strong>{{ projects.length }}</strong></div>
      <div class="summary-card ready"><span>正常项目</span><strong>{{ readyCount }}</strong></div>
      <div class="summary-card active"><span>初始化中</span><strong>{{ initializingCount }}</strong></div>
      <div class="summary-card failed"><span>初始化失败</span><strong>{{ failedCount }}</strong></div>
    </div>

    <div class="filter-bar">
      <el-input v-model="keyword" placeholder="搜索项目名称 / 仓库地址 / 分支" clearable style="width: 320px" />
      <el-select v-model="filterStatus" placeholder="全部状态" clearable style="width: 150px">
        <el-option label="正常" value="ready" />
        <el-option label="初始化中" value="initializing" />
        <el-option label="初始化失败" value="init_failed" />
      </el-select>
      <el-button v-if="keyword || filterStatus" text @click="keyword='';filterStatus=''">清除</el-button>
      <span class="filter-count">{{ filteredProjects.length }} 条</span>
    </div>

    <div class="table-wrap">
      <el-table :data="filteredProjects" v-loading="loading" style="width:100%">
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
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <div class="action-cell">
              <el-button class="op-btn detail-btn" type="primary" plain size="small" @click="router.push(`/projects/${row.id}`)">详情</el-button>
              <el-button class="op-btn trigger-btn" type="success" plain size="small" :disabled="row.status !== 'ready'" @click="openTriggerDialog(row)">触发审核</el-button>
              <el-button class="op-btn" plain size="small" @click="router.push(`/projects/new?clone_from=${row.id}`)">克隆</el-button>
              <el-button class="op-btn" type="danger" plain size="small" @click="handleDelete(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!loading && filteredProjects.length === 0" class="empty-state">
        <div class="empty-icon">📁</div>
        <p>{{ projects.length ? '没有符合筛选条件的项目' : '暂无项目' }}</p>
        <el-button type="primary" size="small" @click="router.push('/projects/new')">新建第一个项目</el-button>
      </div>
    </div>

    <el-dialog
      v-model="triggerDialogVisible"
      :title="`触发审核 · ${triggerProjectRow?.name || ''}`"
      width="680px"
      class="trigger-dialog"
      :close-on-click-modal="false"
      destroy-on-close
      align-center
    >
      <div class="trigger-content" v-loading="commitsLoading || skillsLoading">
        <p class="trigger-desc">选择本次审核的 Commit 范围：</p>
        <el-form label-position="top">
          <el-form-item label="起始 Commit（From）">
            <el-select v-model="triggerForm.from_commit" placeholder="选择起始 Commit" style="width:100%" :loading="commitsLoading">
              <el-option v-for="c in commits" :key="c.sha" :value="c.sha" :label="`${c.sha.slice(0,7)} · ${c.message}`" />
            </el-select>
          </el-form-item>
          <el-form-item label="结束 Commit（To）">
            <el-select v-model="triggerForm.to_commit" placeholder="选择结束 Commit" style="width:100%" :loading="commitsLoading">
              <el-option v-for="c in commits" :key="c.sha" :value="c.sha" :label="`${c.sha.slice(0,7)} · ${c.message}`" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="allSkills.length" label="Review Skills">
            <div class="skills-check-group">
              <el-checkbox
                v-for="skill in allSkills"
                :key="skill.id"
                :label="skill.name"
                :model-value="triggerForm.skill_ids.includes(skill.id)"
                @change="val => { if (val) { triggerForm.skill_ids.push(skill.id) } else { triggerForm.skill_ids = triggerForm.skill_ids.filter(id => id !== skill.id) } }"
              >{{ skill.name }}</el-checkbox>
            </div>
          </el-form-item>
        </el-form>
        <div class="commit-list" v-if="commits.length">
          <div class="commit-list-title">最近提交</div>
          <div v-for="c in commits" :key="c.sha" class="commit-item">
            <span class="commit-hash-inline">{{ c.sha.slice(0, 7) }}</span>
            <span class="commit-msg">{{ c.message }}</span>
            <span class="commit-author">{{ c.author }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="triggerDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="triggering" @click="handleTrigger">触发审核</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import { listProjects, deleteProject, getCommits, triggerProject, listReviewSkills, getProjectSkills } from '../../api/projects'

const router = useRouter()
const projects = ref([])
const loading = ref(false)
const keyword = ref('')
const filterStatus = ref('')
const triggerDialogVisible = ref(false)
const triggering = ref(false)
const commitsLoading = ref(false)
const triggerProjectRow = ref(null)
const commits = ref([])
const triggerForm = ref({ from_commit: '', to_commit: '', skill_ids: [] })
const allSkills = ref([])
const skillsLoading = ref(false)

const isAdmin = computed(() => {
  const role = localStorage.getItem('role') || ''
  return role === 'admin' || role === 'super_admin'
})
const readyCount = computed(() => projects.value.filter(p => p.status === 'ready').length)
const initializingCount = computed(() => projects.value.filter(p => p.status === 'initializing').length)
const failedCount = computed(() => projects.value.filter(p => p.status === 'init_failed').length)
const filteredProjects = computed(() => projects.value.filter(p => {
  const q = keyword.value.trim().toLowerCase()
  if (filterStatus.value && p.status !== filterStatus.value) return false
  if (q && ![p.name, p.repo_url, p.branch].some(v => String(v || '').toLowerCase().includes(q))) return false
  return true
}))

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

async function openTriggerDialog(row) {
  triggerProjectRow.value = row
  triggerForm.value = { from_commit: '', to_commit: '', skill_ids: [] }
  commits.value = []
  allSkills.value = []
  triggerDialogVisible.value = true
  commitsLoading.value = true
  skillsLoading.value = true
  try {
    const [commitsRes, skillsRes, projectSkillsRes] = await Promise.all([
      getCommits(row.id),
      listReviewSkills(),
      getProjectSkills(row.id),
    ])
    commits.value = commitsRes.data || []
    allSkills.value = (skillsRes.data || []).filter(s => s.enabled)
    triggerForm.value.skill_ids = projectSkillsRes.data?.skill_ids || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载数据失败')
  } finally {
    commitsLoading.value = false
    skillsLoading.value = false
  }
}

async function handleTrigger() {
  if (!triggerProjectRow.value) return
  if (!triggerForm.value.from_commit || !triggerForm.value.to_commit) {
    ElMessage.warning('请选择完整的 Commit 范围')
    return
  }
  triggering.value = true
  try {
    const res = await triggerProject(triggerProjectRow.value.id, {
      from_commit: triggerForm.value.from_commit,
      to_commit: triggerForm.value.to_commit,
      skill_ids: triggerForm.value.skill_ids,
    })
    if (res.data.skipped) {
      ElMessage.warning('任务已在队列中')
    } else {
      triggerDialogVisible.value = false
      router.push('/tasks/' + res.data.task_id)
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '触发审核失败')
  } finally {
    triggering.value = false
  }
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

.summary-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin: 20px 36px 14px; }
.summary-card { background: #fff; border: 1px solid #e2e8f0; border-radius: 18px; padding: 14px 18px; display: flex; justify-content: space-between; align-items: center; box-shadow: var(--sh-card); }
.summary-card span { color: #64748b; font-size: 13px; }
.summary-card strong { color: #0f172a; font-size: 22px; font-variant-numeric: tabular-nums; }
.summary-card.ready strong { color: #059669; }
.summary-card.active strong { color: #d97706; }
.summary-card.failed strong { color: #dc2626; }
.filter-bar { display: flex; align-items: center; gap: 10px; margin: 0 36px 14px; background: #fff; border: 1px solid #e2e8f0; border-radius: 16px; padding: 12px 14px; box-shadow: var(--sh-card); }
.filter-count { margin-left: auto; color: #94a3b8; font-size: 13px; }
.table-wrap {
  background: #fff;
  border: 1px solid #e8edf4;
  border-radius: 16px;
  overflow: hidden;
  margin: 0 36px 24px;
  box-shadow: var(--sh-card);
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

.action-cell { display: flex; align-items: center; gap: 8px; }
.action-cell :deep(.op-btn) { font-weight: 650; opacity: 1 !important; visibility: visible !important; }
.action-cell :deep(.detail-btn) { color: #2563eb !important; border-color: #2563eb !important; background: #fff !important; }
.action-cell :deep(.detail-btn:hover) { background: #eff6ff !important; }
.action-cell :deep(.trigger-btn) { color: #059669 !important; border-color: #10b981 !important; background: #fff !important; }
.action-cell :deep(.trigger-btn:hover) { background: #ecfdf5 !important; }
.sep { color: #e2e8f0; font-size: 12px; }
.trigger-content { padding: 2px 4px 0; }
.trigger-desc { margin: 0 0 14px; color: #64748b; font-size: 14px; }
.commit-list { margin-top: 16px; border: 1px solid #e2e8f0; border-radius: 10px; overflow: hidden; }
.commit-list-title { padding: 8px 12px; background: #f8fafc; color: #64748b; font-size: 12px; font-weight: 700; }
.commit-item { display: flex; align-items: center; gap: 10px; padding: 9px 12px; border-top: 1px solid #f1f5f9; font-size: 13px; }
.commit-hash-inline { font-family: monospace; color: #475569; background: #f1f5f9; border-radius: 5px; padding: 2px 7px; }
.commit-msg { flex: 1; color: #1e293b; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.commit-author { color: #94a3b8; font-size: 12px; }

.empty-state { padding: 48px; text-align: center; }
.empty-icon  { font-size: 36px; margin-bottom: 10px; }
.empty-state p { margin: 0 0 12px; font-size: 14px; color: #94a3b8; }
.skills-check-group { display: flex; flex-wrap: wrap; gap: 10px 20px; }
</style>
