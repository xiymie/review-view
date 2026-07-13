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

    <!-- Table -->
    <div class="table-card">
      <el-table :data="list" v-loading="loading" row-key="id" stripe>
        <el-table-column label="名称" min-width="140">
          <template #default="{ row }">
            <span class="name-link" @click="openJobs(row)">{{ row.name }}</span>
          </template>
        </el-table-column>

        <el-table-column label="仓库" min-width="200">
          <template #default="{ row }">
            <span class="repo-url">{{ row.repo_url }}</span>
          </template>
        </el-table-column>

        <el-table-column label="巡检时间" width="110">
          <template #default="{ row }">
            <code class="time-code">{{ row.scan_time || '全局默认' }}</code>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <span :class="['pill', row.enabled ? 'pill-on' : 'pill-off']">
              {{ row.enabled ? '启用' : '停用' }}
            </span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openJobs(row)">记录</el-button>
            <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-button link size="small" @click="handleTrigger(row)" :loading="triggeringId === row.id">
              立即执行
            </el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
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
import { Plus } from '@element-plus/icons-vue'
import {
  listScanSchedules,
  createScanSchedule,
  updateScanSchedule,
  deleteScanSchedule,
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

// 触发后轮询
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
  // 模型和凭据独立加载，不因主列表失败而中断
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

    // 打开执行记录面板
    currentSchedule.value = row
    jobsDrawerVisible.value = true

    // 等一秒让后端创建 job，然后开始轮询最新 job 状态
    setTimeout(() => startPolling(row.id), 1200)
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '触发失败')
  } finally {
    triggeringId.value = null
  }
}

async function startPolling(scheduleId) {
  stopPolling()
  // 先拿最新的 job id
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
      // 通知 Jobs 子组件刷新
      scanJobsRef.value?.load()
      if (job.status === 'completed') {
        ElMessage.success(`巡检完成，${job.changed_branch_count} 个分支有改动`)
      } else {
        ElMessage.error(`巡检失败: ${job.error_message || '未知错误'}`)
      }
    } else {
      // 还在跑，通知子组件刷新列表
      scanJobsRef.value?.load()
    }
  } catch { /* ignore */ }
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
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
  // 处理从首页最近动态跳转过来的 job_id
  const jobIdParam = route.query.job_id
  if (jobIdParam) {
    const jobId = parseInt(jobIdParam)
    // 需要找到该 job 属于哪个 schedule，用 API 查
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
    } catch { /* ignore, just don't open */ }
  }
})
onUnmounted(stopPolling)
</script>

<style scoped>
.page { min-height: 100vh; background: #f8fafc; }

.hero {
  background: linear-gradient(135deg, #7c3aed 0%, #2563eb 100%);
  padding: 28px 36px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.hero-title { font-size: 22px; font-weight: 700; color: #fff; margin-bottom: 4px; }
.hero-subtitle { font-size: 13px; color: rgba(255,255,255,0.75); }
.new-btn { background: rgba(255,255,255,0.15); border: 1px solid rgba(255,255,255,0.3); color: #fff; }
.new-btn:hover { background: rgba(255,255,255,0.25); border-color: rgba(255,255,255,0.5); color: #fff; }

.table-card {
  margin: 24px 36px;
  background: #fff;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  overflow: hidden;
}

.name-link { color: #2563eb; cursor: pointer; font-weight: 500; }
.name-link:hover { text-decoration: underline; }
.repo-url { font-size: 12px; color: #64748b; font-family: monospace; }
.time-code {
  font-family: monospace; font-size: 12px;
  background: #f1f5f9; color: #334155;
  padding: 2px 7px; border-radius: 4px;
}

.pill { display: inline-block; padding: 2px 10px; border-radius: 99px; font-size: 12px; font-weight: 500; }
.pill-on  { background: #dcfce7; color: #166534; }
.pill-off { background: #f1f5f9; color: #64748b; }

.form-body { padding: 8px 4px; }
.drawer-footer { display: flex; justify-content: flex-end; gap: 8px; }

.nas-test-row { display: flex; align-items: center; gap: 10px; }
.nas-test-result { font-size: 13px; }
.nas-test-result.ok   { color: #16a34a; }
.nas-test-result.fail { color: #dc2626; }
</style>
