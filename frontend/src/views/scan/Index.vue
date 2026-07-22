<template>
  <div class="page">
    <!-- Hero -->
    <div class="hero">
      <div class="hero-content">
        <div class="hero-title">巡检配置</div>
        <div class="hero-subtitle">定时扫描仓库所有活跃分支，自动分析每日 commit 风险并上传报告到 NAS</div>
      </div>
      <el-button type="primary" @click="openCreate" class="new-btn">
        <el-icon><Plus /></el-icon> 新建巡检
      </el-button>
    </div>

    <!-- 卡片列表 -->
    <div class="list-wrap" v-loading="loading">
      <div v-if="!loading && list.length === 0" class="empty-state">
        <el-empty description="暂无巡检配置，点击右上角新建" />
      </div>

      <div v-for="row in list" :key="row.id" class="item-card">
        <!-- 左：名称 + 仓库 -->
        <div class="item-main">
          <div class="item-name-row">
            <span class="item-name" @click="openJobs(row)">{{ row.name }}</span>
            <span :class="['status-pill', row.enabled ? 'pill-on' : 'pill-off']">
              {{ row.enabled ? '启用' : '停用' }}
            </span>
          </div>
          <el-tooltip :content="row.repo_url" placement="top" :show-after="400">
            <div class="item-repo">
              <el-icon class="repo-icon"><Link /></el-icon>
              <span class="repo-url-text">{{ row.repo_url }}</span>
            </div>
          </el-tooltip>
        </div>

        <!-- 中：巡检时间 -->
        <div class="item-meta">
          <div class="meta-label">巡检时间</div>
          <code class="time-code">{{ row.scan_time || '全局默认' }}</code>
        </div>

        <!-- 右：操作 -->
        <div class="item-actions">
          <el-button size="small" @click="openJobs(row)">
            <el-icon><Document /></el-icon> 记录
          </el-button>
          <el-button
            size="small"
            type="primary"
            :loading="triggeringId === row.id"
            @click="handleTrigger(row)"
          >
            <el-icon><VideoPlay /></el-icon> 执行
          </el-button>
          <el-dropdown trigger="click" @command="cmd => handleCmd(cmd, row)">
            <el-button size="small">
              <el-icon><MoreFilled /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="edit">
                  <el-icon><Edit /></el-icon> 编辑
                </el-dropdown-item>
                <el-dropdown-item command="toggle">
                  <el-icon><VideoPause v-if="row.enabled" /><VideoPlay v-else /></el-icon>
                  {{ row.enabled ? '暂停' : '恢复' }}
                </el-dropdown-item>
                <el-dropdown-item command="delete" divided class="danger-item">
                  <el-icon><Delete /></el-icon> 删除
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>

    <!-- 执行记录侧滑 -->
    <el-drawer
      v-model="jobsDrawerVisible"
      :title="`执行记录 · ${currentSchedule?.name}`"
      direction="rtl"
      size="700px"
      @close="initialJobId = null"
    >
      <ScanJobs
        v-if="jobsDrawerVisible && currentSchedule"
        :schedule-id="currentSchedule.id"
        :running-job-id="runningJobId"
        :initial-job-id="initialJobId"
        ref="scanJobsRef"
      />
    </el-drawer>

    <!-- 新建/编辑 drawer -->
    <el-drawer
      v-model="formDrawerVisible"
      :title="editingItem ? '编辑巡检配置' : '新建巡检配置'"
      direction="rtl"
      size="520px"
      :close-on-click-modal="false"
    >
      <div class="form-body" v-if="formDrawerVisible">
        <el-form :model="form" label-position="top" ref="formRef">
          <el-form-item label="名称" required>
            <el-input v-model="form.name" placeholder="例：后端主仓库巡检" />
          </el-form-item>

          <el-form-item label="仓库地址" required>
            <el-input v-model="form.repo_url" placeholder="https://github.com/org/repo.git" />
          </el-form-item>

          <el-form-item label="仓库凭据">
            <el-select v-model="form.repo_credential_id" placeholder="无（公开仓库）" clearable style="width:100%">
              <el-option
                v-for="c in credentials"
                :key="c.id"
                :label="c.name"
                :value="c.id"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="模型配置" required>
            <el-select v-model="form.model_config_id" placeholder="请选择" style="width:100%">
              <el-option
                v-for="m in models"
                :key="m.id"
                :label="m.name"
                :value="m.id"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="巡检时间">
            <el-input v-model="form.scan_time" placeholder="留空使用全局默认，格式 HH:MM，如 09:00" />
          </el-form-item>

          <el-form-item label="自定义提示词">
            <el-input
              v-model="form.custom_prompt"
              type="textarea"
              :rows="4"
              placeholder="留空使用全局默认提示词"
            />
          </el-form-item>

          <el-divider>NAS 配置（留空使用全局默认）</el-divider>

          <el-form-item label="WebDAV 地址">
            <el-input v-model="form.nas_url" placeholder="http://192.168.1.57:5005" />
          </el-form-item>

          <el-form-item label="NAS 用户名">
            <el-input v-model="form.nas_username" />
          </el-form-item>

          <el-form-item label="NAS 密码">
            <el-input v-model="form.nas_password" type="password" show-password placeholder="编辑时留空不修改" />
          </el-form-item>

          <el-form-item label="上传子目录">
            <el-input v-model="form.nas_sub_dir" placeholder="留空使用仓库名" />
          </el-form-item>

          <!-- 测试 NAS 连接 -->
          <el-form-item label="测试 NAS 连接">
            <div class="nas-test-row">
              <el-button size="small" :loading="testingNas" @click="handleTestNas">测试连接</el-button>
              <span v-if="nasTestResult" :class="['nas-test-result', nasTestResult.ok ? 'ok' : 'fail']">
                {{ nasTestResult.ok ? '✓ ' + nasTestResult.message : '✗ ' + nasTestResult.error }}
              </span>
            </div>
          </el-form-item>

          <el-form-item label="状态">
            <el-switch v-model="form.enabled" active-text="启用" inactive-text="停用" />
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <div class="drawer-footer">
          <el-button @click="formDrawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="save">保存</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Document, VideoPlay, VideoPause, MoreFilled, Edit, Delete, Link } from '@element-plus/icons-vue'
import {
  listScanSchedules,
  createScanSchedule,
  updateScanSchedule,
  deleteScanSchedule,
  toggleScanSchedule,
  triggerScanSchedule,
  listScanModels,
  listScanCredentials,
  testScanNas,
  getScanJob,
  listScanJobs,
} from '../../api/scan'
import ScanJobs from './Jobs.vue'

const list = ref([])
const loading = ref(false)
const models = ref([])
const credentials = ref([])
const saving = ref(false)
const triggeringId = ref(null)

const route = useRoute()
const formDrawerVisible = ref(false)
const jobsDrawerVisible = ref(false)
const editingItem = ref(null)
const currentSchedule = ref(null)
const scanJobsRef = ref(null)
const initialJobId = ref(null)

const runningJobId = ref(null)
let pollTimer = null

const testingNas = ref(false)
const nasTestResult = ref(null)

const defaultForm = () => ({
  name: '',
  repo_url: '',
  repo_credential_id: null,
  model_config_id: null,
  scan_time: '',
  custom_prompt: '',
  nas_url: '',
  nas_username: '',
  nas_password: '',
  nas_sub_dir: '',
  enabled: true,
})
const form = ref(defaultForm())

async function load() {
  loading.value = true
  try {
    const r1 = await listScanSchedules()
    list.value = r1.data
  } catch {
    ElMessage.error('加载巡检列表失败')
  } finally {
    loading.value = false
  }
  try {
    const r2 = await listScanModels()
    models.value = r2.data
  } catch {
    ElMessage.error('加载模型配置失败，请检查权限')
  }
  try {
    const r3 = await listScanCredentials()
    credentials.value = r3.data
  } catch {
    ElMessage.error('加载仓库凭据失败，请检查权限')
  }
}

function openCreate() {
  editingItem.value = null
  form.value = defaultForm()
  nasTestResult.value = null
  formDrawerVisible.value = true
}

function openEdit(row) {
  editingItem.value = row
  form.value = {
    name: row.name,
    repo_url: row.repo_url,
    repo_credential_id: row.repo_credential_id || null,
    model_config_id: row.model_config_id,
    scan_time: row.scan_time || '',
    custom_prompt: row.custom_prompt || '',
    nas_url: row.nas_url || '',
    nas_username: row.nas_username || '',
    nas_password: '',
    nas_sub_dir: row.nas_sub_dir || '',
    enabled: row.enabled,
  }
  nasTestResult.value = null
  formDrawerVisible.value = true
}

function openJobs(row) {
  currentSchedule.value = row
  jobsDrawerVisible.value = true
}

function handleCmd(cmd, row) {
  if (cmd === 'edit') openEdit(row)
  if (cmd === 'toggle') handleToggle(row)
  if (cmd === 'delete') handleDelete(row)
}

async function handleToggle(row) {
  try {
    const res = await toggleScanSchedule(row.id)
    row.enabled = res.data.enabled
    ElMessage.success(row.enabled ? '已恢复' : '已暂停')
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '操作失败')
  }
}

async function save() {
  if (!form.value.name || !form.value.repo_url || !form.value.model_config_id) {
    ElMessage.warning('请填写名称、仓库地址和模型配置')
    return
  }
  saving.value = true
  try {
    if (editingItem.value) {
      await updateScanSchedule(editingItem.value.id, form.value)
    } else {
      await createScanSchedule(form.value)
    }
    ElMessage.success('保存成功')
    formDrawerVisible.value = false
    await load()
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleTestNas() {
  testingNas.value = true
  nasTestResult.value = null
  try {
    const res = await testScanNas({
      nas_url: form.value.nas_url,
      nas_username: form.value.nas_username,
      nas_password: form.value.nas_password,
    })
    nasTestResult.value = res.data
  } catch (err) {
    nasTestResult.value = { ok: false, error: err.response?.data?.error || '请求失败' }
  } finally {
    testingNas.value = false
  }
}

async function handleTrigger(row) {
  triggeringId.value = row.id
  try {
    await triggerScanSchedule(row.id)
    ElMessage.success('巡检已启动')
    currentSchedule.value = row
    jobsDrawerVisible.value = true
    setTimeout(() => startPolling(row.id), 1200)
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '触发失败')
  } finally {
    triggeringId.value = null
  }
}

async function startPolling(scheduleId) {
  stopPolling()
  try {
    const res = await listScanJobs(scheduleId)
    const jobs = res.data
    if (!jobs || jobs.length === 0) return
    const latest = jobs[0]
    if (latest.status === 'completed' || latest.status === 'failed') return
    runningJobId.value = latest.id
    pollTimer = setInterval(() => pollJob(latest.id), 3000)
  } catch { /* ignore */ }
}

async function pollJob(jobId) {
  try {
    const res = await getScanJob(jobId)
    const job = res.data?.job
    if (!job) return
    if (job.status === 'completed' || job.status === 'failed') {
      stopPolling()
      runningJobId.value = null
      scanJobsRef.value?.load()
      if (job.status === 'completed') {
        ElMessage.success(`巡检完成，${job.changed_branch_count} 个分支有改动`)
      } else {
        ElMessage.error(`巡检失败: ${job.error_message || '未知错误'}`)
      }
    } else {
      scanJobsRef.value?.load()
    }
  } catch { /* ignore */ }
}

function stopPolling() {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
}

async function handleDelete(row) {
  await ElMessageBox.confirm(`确认删除巡检配置「${row.name}」？`, '删除确认', {
    type: 'warning',
    confirmButtonText: '删除',
    confirmButtonClass: 'el-button--danger',
  })
  try {
    await deleteScanSchedule(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '删除失败')
  }
}

onMounted(async () => {
  await load()
  const jobIdParam = route.query.job_id
  if (jobIdParam) {
    const jobId = parseInt(jobIdParam)
    try {
      const res = await getScanJob(jobId)
      const jobData = res.data?.job || res.data
      if (jobData && jobData.schedule_id) {
        const sched = list.value.find(s => s.id === jobData.schedule_id)
        if (sched) {
          initialJobId.value = jobId
          currentSchedule.value = sched
          jobsDrawerVisible.value = true
        }
      }
    } catch { /* ignore */ }
  }
})
onUnmounted(stopPolling)
</script>

<style scoped>
.page { min-height: 100vh; background: #f8fafc; }

/* Hero */
.hero {
  background: linear-gradient(135deg, #7c3aed 0%, #2563eb 100%);
  padding: 28px 36px 24px;
  display: flex; align-items: center; justify-content: space-between;
}
.hero-title { font-size: 22px; font-weight: 700; color: #fff; margin-bottom: 4px; }
.hero-subtitle { font-size: 13px; color: rgba(255,255,255,0.75); }
.new-btn {
  background: rgba(255,255,255,0.15);
  border: 1px solid rgba(255,255,255,0.3);
  color: #fff;
}
.new-btn:hover {
  background: rgba(255,255,255,0.25);
  border-color: rgba(255,255,255,0.5);
  color: #fff;
}

/* 卡片列表 */
.list-wrap {
  margin: 24px 36px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.empty-state {
  background: #fff;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  padding: 40px;
}

.item-card {
  background: #fff;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  padding: 16px 20px;
  display: flex;
  align-items: center;
  gap: 24px;
  transition: box-shadow 0.15s, border-color 0.15s;
}
.item-card:hover {
  box-shadow: 0 2px 12px rgba(0,0,0,0.07);
  border-color: #cbd5e1;
}
.item-card:nth-child(5n+1) { background: #f0f7ff; border-color: #dbeafe; }
.item-card:nth-child(5n+2) { background: #f0fdf4; border-color: #dcfce7; }
.item-card:nth-child(5n+3) { background: #fdf4ff; border-color: #f3e8ff; }
.item-card:nth-child(5n+4) { background: #fff7ed; border-color: #fed7aa; }
.item-card:nth-child(5n+5) { background: #f0fdfa; border-color: #ccfbf1; }
.item-card:nth-child(5n+1):hover { border-color: #93c5fd; }
.item-card:nth-child(5n+2):hover { border-color: #86efac; }
.item-card:nth-child(5n+3):hover { border-color: #d8b4fe; }
.item-card:nth-child(5n+4):hover { border-color: #fb923c; }
.item-card:nth-child(5n+5):hover { border-color: #5eead4; }

/* 左：名称+仓库 */
.item-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.item-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.item-name {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 260px;
}
.item-name:hover { color: #2563eb; }

.status-pill {
  display: inline-block;
  padding: 1px 9px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}
.pill-on  { background: #dcfce7; color: #166534; }
.pill-off { background: #f1f5f9; color: #64748b; }

.item-repo {
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
}
.repo-icon {
  font-size: 12px;
  color: #94a3b8;
  flex-shrink: 0;
}
.repo-url-text {
  font-size: 12px;
  color: #94a3b8;
  font-family: 'SFMono-Regular', 'Menlo', monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 中：巡检时间 */
.item-meta {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  min-width: 80px;
}
.meta-label {
  font-size: 11px;
  color: #94a3b8;
}
.time-code {
  font-family: 'SFMono-Regular', 'Menlo', monospace;
  font-size: 12px;
  background: #f1f5f9;
  color: #334155;
  padding: 2px 8px;
  border-radius: 4px;
  white-space: nowrap;
}

/* 右：操作 */
.item-actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}
.item-actions :deep(.el-button) {
  height: 34px;
  padding: 0 16px;
  font-size: 13px;
  font-weight: 500;
  border-radius: 8px;
}
.item-actions :deep(.el-button .el-icon) {
  font-size: 14px;
}
/* 执行按钮加渐变更醒目 */
.item-actions :deep(.el-button--primary) {
  background: linear-gradient(135deg, #6366f1, #2563eb);
  border-color: transparent;
}
.item-actions :deep(.el-button--primary:hover) {
  background: linear-gradient(135deg, #4f46e5, #1d4ed8);
  border-color: transparent;
}
/* 更多按钮 */
.item-actions :deep(.el-button:not(.el-button--primary)) {
  border-color: #e2e8f0;
  color: #475569;
}
.item-actions :deep(.el-button:not(.el-button--primary):hover) {
  border-color: #94a3b8;
  color: #1e293b;
  background: #f8fafc;
}

/* 下拉危险项 */
.danger-item { color: #dc2626 !important; }
.danger-item:hover { background: #fef2f2 !important; }

/* Form / Drawer */
.form-body { padding: 8px 4px; }
.drawer-footer { display: flex; justify-content: flex-end; gap: 8px; }
.nas-test-row { display: flex; align-items: center; gap: 10px; }
.nas-test-result { font-size: 13px; }
.nas-test-result.ok   { color: #16a34a; }
.nas-test-result.fail { color: #dc2626; }
</style>
