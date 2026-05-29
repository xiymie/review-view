<template>
  <div class="page-wrap">
    <!-- 面包屑 -->
    <el-breadcrumb separator="/" class="breadcrumb">
      <el-breadcrumb-item :to="{ path: '/projects' }">项目</el-breadcrumb-item>
      <el-breadcrumb-item>{{ project.name }}</el-breadcrumb-item>
    </el-breadcrumb>

    <!-- 状态 Alert -->
    <el-alert
      v-if="project.status === 'initializing'"
      type="warning"
      title="项目初始化中"
      description="正在克隆仓库并初始化项目，请稍候…"
      show-icon
      :closable="false"
      class="status-alert"
    />
    <el-alert
      v-if="project.status === 'init_failed'"
      type="error"
      title="初始化失败"
      description="项目初始化过程中发生错误，请检查仓库地址和凭据后重试。"
      show-icon
      :closable="false"
      class="status-alert"
    >
      <template #default>
        <div class="alert-action">
          <span>项目初始化过程中发生错误，请检查仓库地址和凭据后重试。</span>
          <el-button size="small" type="danger" plain @click="handleReinit">重新初始化</el-button>
        </div>
      </template>
    </el-alert>

    <!-- 页头 hero -->
    <div class="page-hero">
      <div class="hero-content">
        <h2 class="hero-title">{{ project.name }}</h2>
        <p class="hero-sub">
          <span class="repo-url-hero">{{ project.repo_url }}</span>
          <span class="sep-hero">·</span>
          <span>{{ project.branch }}</span>
        </p>
      </div>
      <div class="header-actions hero-actions">
        <el-button @click="router.push(`/projects/${project.id}/edit`)">编辑</el-button>
        <el-button @click="router.push(`/projects/new?clone_from=${project.id}`)">克隆</el-button>
        <el-button type="primary" @click="openTriggerDrawer">手动触发审核</el-button>
        <el-button type="danger" plain @click="handleDelete">删除</el-button>
      </div>
      <div class="deco-circles">
        <div class="deco c1"></div>
        <div class="deco c2"></div>
      </div>
    </div>

    <!-- 信息条 -->
    <el-card class="info-card">
      <el-descriptions :column="3" border>
        <el-descriptions-item label="模型配置">{{ modelConfig?.name || '—' }}</el-descriptions-item>
        <el-descriptions-item label="溢出策略">
          {{ project.overflow_strategy === 'queue' ? '排队等待' : '拒绝' }}
        </el-descriptions-item>
        <el-descriptions-item label="最近 Review Commit">
          <span class="commit-hash">{{ project.last_reviewed_commit ? project.last_reviewed_commit.slice(0, 7) : '暂无' }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="Webhook 地址" :span="3">
          <div class="webhook-row">
            <code class="webhook-url">{{ webhookUrl }}</code>
            <el-button size="small" text @click="copyWebhook">复制</el-button>
          </div>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 自定义 Prompt -->
    <el-card v-if="project.custom_prompt" class="prompt-card">
      <template #header>
        <span class="card-title">自定义 Prompt</span>
      </template>
      <pre class="prompt-content">{{ project.custom_prompt }}</pre>
    </el-card>

    <!-- 任务列表 -->
    <el-card class="table-card">
      <template #header>
        <span class="card-title">审核任务</span>
      </template>
      <el-table :data="tasks" style="width:100%">
        <el-table-column label="ID" prop="id" width="80" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Commit 范围" min-width="200">
          <template #default="{ row }">
            <span class="commit-hash">{{ row.from_commit ? row.from_commit.slice(0,7) : '—' }}</span>
            <span class="arrow"> → </span>
            <span class="commit-hash">{{ row.to_commit ? row.to_commit.slice(0,7) : '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="触发方式" prop="trigger_type" width="120" />
        <el-table-column label="时间" prop="created_at" width="180" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="router.push(`/tasks/${row.id}`)">
              查看
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 手动触发 Drawer -->
    <el-drawer
      v-model="triggerDrawerVisible"
      title="手动触发审核"
      direction="rtl"
      size="420px"
    >
      <div class="drawer-content">
        <p class="drawer-desc">选择本次审核的 Commit 范围：</p>

        <el-form label-position="top">
          <el-form-item label="起始 Commit（From）">
            <el-select v-model="triggerForm.from_commit" placeholder="选择起始 Commit" style="width:100%" :loading="commitsLoading">
              <el-option
                v-for="c in commits"
                :key="c.sha"
                :value="c.sha"
                :label="`${c.sha.slice(0,7)} · ${c.message}`"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="结束 Commit（To）">
            <el-select v-model="triggerForm.to_commit" placeholder="选择结束 Commit" style="width:100%" :loading="commitsLoading">
              <el-option
                v-for="c in commits"
                :key="c.sha"
                :value="c.sha"
                :label="`${c.sha.slice(0,7)} · ${c.message}`"
              />
            </el-select>
          </el-form-item>
        </el-form>

        <div class="commit-list">
          <div class="commit-list-title">最近提交</div>
          <div v-for="c in commits" :key="c.sha" class="commit-item">
            <span class="commit-hash">{{ c.sha.slice(0, 7) }}</span>
            <span class="commit-msg">{{ c.message }}</span>
            <span class="commit-author">{{ c.author }}</span>
          </div>
        </div>

        <div class="drawer-footer">
          <el-button @click="triggerDrawerVisible = false">取消</el-button>
          <el-button type="primary" @click="handleTrigger">触发审核</el-button>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import { getProject, triggerProject, initProject, deleteProject, getCommits } from '../../api/projects'

const router = useRouter()
const route = useRoute()

const id = route.params.id

const project = ref({})
const modelConfig = ref(null)
const tasks = ref([])
const commits = ref([])
const triggerDrawerVisible = ref(false)
const triggerForm = ref({ from_commit: '', to_commit: '' })
const loading = ref(false)
const commitsLoading = ref(false)

const webhookUrl = computed(() => `${window.location.origin}/webhook/${id}`)

function copyWebhook() {
  navigator.clipboard.writeText(webhookUrl.value)
  ElMessage.success('已复制到剪贴板')
}

onMounted(async () => {
  loading.value = true
  try {
    const res = await getProject(id)
    project.value = res.data.project
    modelConfig.value = res.data.model_config
    tasks.value = res.data.tasks || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载失败')
  } finally {
    loading.value = false
  }
})

async function openTriggerDrawer() {
  triggerDrawerVisible.value = true
  commitsLoading.value = true
  try {
    const res = await getCommits(id)
    commits.value = res.data
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载 Commit 列表失败')
  } finally {
    commitsLoading.value = false
  }
}

function statusTagType(status) {
  const map = {
    completed: 'success', running: 'warning', pending: 'info',
    failed: 'danger', cancelled: '', ready: 'success',
    initializing: 'warning', init_failed: 'danger',
  }
  return map[status] ?? ''
}

function statusLabel(status) {
  const map = {
    completed: '已完成', running: '运行中', pending: '等待中',
    failed: '失败', cancelled: '已取消',
  }
  return map[status] ?? status
}

async function handleReinit() {
  try {
    await initProject(id)
    ElMessage.success('已发起重新初始化')
    const res = await getProject(id)
    project.value = res.data.project
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  }
}

async function handleDelete() {
  try {
    await ElMessageBox.confirm(
      `确定要删除项目 "${project.value.name}" 吗？此操作不可恢复。`,
      '删除确认',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
    await deleteProject(id)
    ElMessage.success('删除成功')
    router.push('/projects')
  } catch (err) {
    if (err !== 'cancel' && err?.type !== 'cancel') {
      if (err.response) ElMessage.error(err.response?.data?.message || '删除失败')
    }
  }
}

async function handleTrigger() {
  if (!triggerForm.value.from_commit || !triggerForm.value.to_commit) {
    ElMessage.warning('请选择完整的 Commit 范围')
    return
  }
  try {
    const res = await triggerProject(id, {
      from_commit: triggerForm.value.from_commit,
      to_commit: triggerForm.value.to_commit,
    })
    if (res.data.skipped) {
      ElMessage.warning('任务已在队列中')
    } else {
      triggerDrawerVisible.value = false
      triggerForm.value = { from_commit: '', to_commit: '' }
      router.push('/tasks/' + res.data.task_id)
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  }
}
</script>

<style scoped>
.page-wrap { padding: 32px 36px 32px; }

.breadcrumb { margin-bottom: 16px; }

.status-alert { margin-bottom: 16px; }

.alert-action {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
}

/* Hero page header */
.page-hero {
  position: relative;
  background: linear-gradient(135deg, #1e3a8a, #2563eb, #0891b2);
  padding: 24px 28px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  overflow: hidden;
  margin-bottom: 20px;
}
.hero-title { margin: 0 0 4px; font-size: 22px; font-weight: 700; color: #fff; letter-spacing: -0.3px; }
.hero-sub { margin: 0; font-size: 13px; color: rgba(255,255,255,0.8); }
.repo-url-hero { color: rgba(255,255,255,0.9); }
.sep-hero { margin: 0 6px; color: rgba(255,255,255,0.5); }
.hero-actions { display: flex; gap: 8px; flex-shrink: 0; position: relative; z-index: 2; }
.hero-actions :deep(.el-button) { background: rgba(255,255,255,0.15) !important; border-color: rgba(255,255,255,0.3) !important; color: #fff !important; }
.hero-actions :deep(.el-button:hover) { background: rgba(255,255,255,0.25) !important; }
.hero-actions :deep(.el-button--primary) { background: rgba(255,255,255,0.25) !important; }
.hero-actions :deep(.el-button--danger) { background: rgba(239,68,68,0.3) !important; border-color: rgba(239,68,68,0.5) !important; }
.deco-circles { position: absolute; right: 0; top: 0; bottom: 0; width: 200px; pointer-events: none; }
.deco { position: absolute; border-radius: 50%; background: rgba(255,255,255,0.08); }
.c1 { width: 180px; height: 180px; right: -40px; top: -60px; }
.c2 { width: 100px; height: 100px; right: 60px; bottom: -30px; }

.info-card { margin-bottom: 16px; }
.prompt-card { margin-bottom: 16px; }
.table-card {}

.prompt-content {
  margin: 0; font-size: 13px; color: #475569;
  white-space: pre-wrap; word-break: break-word; font-family: monospace;
}

.card-title { font-weight: 600; color: #1e293b; font-size: 14px; }

.commit-hash { font-family: monospace; font-size: 12.5px; color: #475569; }
.arrow { color: #94a3b8; }

/* 任务表格状态 pill */
.status-pill {
  display: inline-block; padding: 2px 9px; border-radius: 20px; font-size: 12px; font-weight: 500;
}
.pill-completed { background: #ecfdf5; color: #059669; }
.pill-running   { background: #fffbeb; color: #d97706; }
.pill-pending   { background: #eff6ff; color: #2563eb; }
.pill-failed    { background: #fef2f2; color: #dc2626; }
.pill-cancelled { background: #f8fafc; color: #64748b; }

.action-cell { display: flex; align-items: center; gap: 3px; }
.op-sep { color: #e2e8f0; font-size: 12px; }

/* Webhook */
.webhook-row { display: flex; align-items: center; gap: 8px; }
.webhook-url {
  font-family: monospace; font-size: 12.5px; color: #2563eb;
  background: #eff6ff; padding: 2px 8px; border-radius: 5px; word-break: break-all;
}

/* Drawer */
.drawer-content { padding: 0 4px; }
.drawer-desc { margin: 0 0 16px; color: #64748b; font-size: 14px; }

.commit-list { margin-top: 20px; border: 1px solid #e2e8f0; border-radius: 8px; overflow: hidden; }
.commit-list-title {
  padding: 8px 12px; background: #f8fafc;
  font-size: 11px; font-weight: 600; color: #64748b;
  text-transform: uppercase; letter-spacing: 0.04em;
}
.commit-item { display: flex; align-items: center; gap: 10px; padding: 9px 12px; border-top: 1px solid #f1f5f9; font-size: 13px; }
.commit-msg { flex: 1; color: #1e293b; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.commit-author { color: #94a3b8; font-size: 12px; }

.drawer-footer { display: flex; justify-content: flex-end; gap: 10px; margin-top: 24px; padding-top: 16px; border-top: 1px solid #f1f5f9; }
</style>
