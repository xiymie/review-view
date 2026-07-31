<template>
  <div class="page-wrap settings-page">
    <div class="page-hero">
      <div class="bubble bubble-a"></div>
      <div class="bubble bubble-b"></div>
      <div class="bubble bubble-c"></div>
      <div class="hero-content">
        <div class="eyebrow">Global Control</div>
        <h1 class="hero-title">全局设置</h1>
        <p class="hero-sub">集中管理并发、仓库、巡检、NAS 与邮件推送参数</p>
      </div>
      <el-button type="primary" class="hero-btn" @click="openSettingsDialog">
        编辑全局设置
      </el-button>
    </div>

    <div class="settings-overview" v-loading="loading">
      <div class="overview-card accent-blue">
        <span class="overview-label">最大并发</span>
        <strong>{{ form.max_concurrent_tasks }}</strong>
        <span class="overview-sub">任务同时执行数</span>
      </div>
      <div class="overview-card accent-violet">
        <span class="overview-label">超时时间</span>
        <strong>{{ form.task_timeout }}m</strong>
        <span class="overview-sub">单任务保护阈值</span>
      </div>
      <div class="overview-card accent-teal">
        <span class="overview-label">默认巡检</span>
        <strong>{{ form.scan_default_time || '未设置' }}</strong>
        <span class="overview-sub">未配置时使用</span>
      </div>
      <div class="overview-card accent-amber">
        <span class="overview-label">报告保留</span>
        <strong>{{ form.scan_retain_days === 0 ? '永久' : `${form.scan_retain_days}天` }}</strong>
        <span class="overview-sub">NAS 清理策略</span>
      </div>
    </div>

    <div class="settings-panels">
      <section class="info-panel">
        <div class="panel-head">
          <div>
            <h3>运行策略</h3>
            <p>Review 队列和仓库工作目录配置</p>
          </div>
        </div>
        <div class="kv-list">
          <div class="kv-row"><span>溢出策略</span><strong>{{ strategyLabel(form.overflow_strategy) }}</strong></div>
          <div class="kv-row"><span>仓库根目录</span><strong>{{ form.repo_base_dir || '—' }}</strong></div>
          <div class="kv-row"><span>定时无新提交</span><strong>{{ form.scheduled_scan_unchanged ? '继续扫描' : '跳过' }}</strong></div>
          <div class="kv-row"><span>手动无新提交</span><strong>{{ form.manual_scan_unchanged ? '继续扫描' : '跳过' }}</strong></div>
        </div>
      </section>

      <section class="info-panel">
        <div class="panel-head">
          <div>
            <h3>巡检与通知</h3>
            <p>Scan workflow 报告上传和邮件推送</p>
          </div>
        </div>
        <div class="kv-list">
          <div class="kv-row"><span>NAS 地址</span><strong>{{ form.scan_nas_url || '—' }}</strong></div>
          <div class="kv-row"><span>NAS 用户</span><strong>{{ form.scan_nas_username || '—' }}</strong></div>
          <div class="kv-row"><span>SMTP 服务</span><strong>{{ form.smtp_host || '—' }}</strong></div>
          <div class="kv-row"><span>发件人</span><strong>{{ form.smtp_from || '—' }}</strong></div>
        </div>
      </section>
    </div>

    <el-dialog
      v-model="settingsVisible"
      title="编辑全局设置"
      width="820px"
      class="settings-dialog"
      :close-on-click-modal="false"
      destroy-on-close
      align-center
    >
      <el-form :model="form" label-position="top" v-loading="loading" class="settings-form">
        <div class="form-section">
          <div class="section-title">基础运行</div>
          <div class="form-grid two">
            <el-form-item label="最大并发任务数">
              <el-input-number v-model="form.max_concurrent_tasks" :min="1" style="width: 100%" />
            </el-form-item>
            <el-form-item label="任务超时时间（分钟）">
              <el-input-number v-model="form.task_timeout" :min="1" style="width: 100%" />
            </el-form-item>
          </div>
          <el-form-item label="全局溢出策略">
            <el-radio-group v-model="form.overflow_strategy">
              <el-radio value="queue">排队等待</el-radio>
              <el-radio value="reject">拒绝</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="仓库根目录">
            <el-input v-model="form.repo_base_dir" placeholder="/data/repos" />
          </el-form-item>
        </div>

        <div class="form-section muted-section">
          <div class="section-title">扫描行为</div>
          <div class="switch-grid">
            <div class="switch-card">
              <span>定时扫描无新提交时</span>
              <el-switch v-model="form.scheduled_scan_unchanged" active-text="继续扫描" inactive-text="跳过" />
              <p>关闭时，定时任务遇到无新提交将跳过，不发起 Review 也不发邮件。</p>
            </div>
            <div class="switch-card">
              <span>手动扫描无新提交时</span>
              <el-switch v-model="form.manual_scan_unchanged" active-text="继续扫描" inactive-text="跳过" />
              <p>关闭时，手动触发遇到无新提交将跳过，不发起 Review 也不发邮件。</p>
            </div>
          </div>
        </div>

        <div class="form-section">
          <div class="section-title">巡检全局配置</div>
          <el-form-item label="默认巡检时间">
            <el-input v-model="form.scan_default_time" placeholder="格式 HH:MM，如 09:00" clearable style="max-width:220px" />
            <div class="field-tip">各巡检配置未单独设置时间时使用此值。</div>
          </el-form-item>
          <el-form-item label="默认巡检提示词">
            <el-input
              v-model="form.scan_default_prompt"
              type="textarea"
              :rows="5"
              placeholder="留空使用内置默认提示词"
              clearable
            />
            <div class="field-tip">支持占位符：仓库URL、分支名、日期、commit数量、commit列表、diff 内容。</div>
          </el-form-item>
        </div>

        <div class="form-section muted-section">
          <div class="section-title">NAS WebDAV</div>
          <el-form-item label="NAS WebDAV 地址">
            <el-input v-model="form.scan_nas_url" placeholder="http://192.168.1.100:5005/reports" clearable />
          </el-form-item>
          <div class="form-grid two">
            <el-form-item label="NAS 用户名">
              <el-input v-model="form.scan_nas_username" clearable />
            </el-form-item>
            <el-form-item label="NAS 密码">
              <el-input v-model="form.scan_nas_password" type="password" show-password placeholder="留空则不更新密码" clearable />
            </el-form-item>
          </div>
          <el-form-item label="报告保留天数">
            <el-input-number v-model="form.scan_retain_days" :min="0" :step="1" style="width:160px" />
            <div class="field-tip">0 = 永久保留；设为 7 则自动删除 NAS 上 7 天前的报告。</div>
          </el-form-item>
        </div>

        <div class="form-section">
          <div class="section-title">邮件推送 SMTP</div>
          <el-form-item label="SMTP 服务器地址">
            <el-input v-model="form.smtp_host" placeholder="smtp.example.com" clearable />
          </el-form-item>
          <div class="form-grid two">
            <el-form-item label="SMTP 端口">
              <el-input v-model="form.smtp_port" placeholder="465 / 587" clearable />
            </el-form-item>
            <el-form-item label="加密方式">
              <el-select v-model="form.smtp_tls" style="width: 100%">
                <el-option label="SSL/TLS（端口 465）" value="true" />
                <el-option label="STARTTLS（端口 587）" value="false" />
              </el-select>
            </el-form-item>
          </div>
          <div class="form-grid two">
            <el-form-item label="SMTP 账号">
              <el-input v-model="form.smtp_username" placeholder="no-reply@example.com" clearable />
            </el-form-item>
            <el-form-item label="SMTP 密码">
              <el-input v-model="form.smtp_password" type="password" show-password placeholder="留空则不更新密码" clearable />
            </el-form-item>
          </div>
          <div class="form-grid two">
            <el-form-item label="发件人地址（From）">
              <el-input v-model="form.smtp_from" placeholder="no-reply@example.com" clearable />
            </el-form-item>
            <el-form-item label="发件人显示名称">
              <el-input v-model="form.smtp_from_name" placeholder="代码审计" clearable />
            </el-form-item>
          </div>
          <el-form-item label="发送测试邮件">
            <div class="test-row">
              <el-input v-model="testEmailTo" placeholder="收件地址" clearable />
              <el-button :loading="testLoading" @click="handleTestEmail">发送测试</el-button>
            </div>
            <div class="field-tip">使用当前弹窗里的 SMTP 配置发送测试邮件。</div>
          </el-form-item>
        </div>
      </el-form>

      <template #footer>
        <el-button @click="settingsVisible = false">取消</el-button>
        <el-button type="primary" :loading="loading" @click="handleSave">保存设置</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'
import { getSettings, updateSettings, testEmail } from '../api/settings'

const loading = ref(false)
const testLoading = ref(false)
const testEmailTo = ref('')
const settingsVisible = ref(false)

const form = ref({
  max_concurrent_tasks: 3,
  overflow_strategy: 'queue',
  task_timeout: 60,
  repo_base_dir: '',
  scheduled_scan_unchanged: false,
  manual_scan_unchanged: true,
  smtp_host: '',
  smtp_port: '465',
  smtp_username: '',
  smtp_password: '',
  smtp_from: '',
  smtp_from_name: '',
  smtp_tls: 'true',
  scan_default_time: '',
  scan_default_prompt: '',
  scan_nas_url: '',
  scan_nas_username: '',
  scan_nas_password: '',
  scan_retain_days: 0,
})

function strategyLabel(v) {
  return v === 'reject' ? '拒绝新任务' : '排队等待'
}

function openSettingsDialog() {
  settingsVisible.value = true
}

async function loadSettings() {
  loading.value = true
  try {
    const res = await getSettings()
    Object.assign(form.value, res.data)
    form.value.smtp_password = ''
    form.value.scan_nas_password = ''
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadSettings)

const handleSave = async () => {
  loading.value = true
  try {
    await updateSettings(form.value)
    ElMessage.success('保存成功')
    form.value.smtp_password = ''
    form.value.scan_nas_password = ''
    settingsVisible.value = false
    await loadSettings()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  } finally {
    loading.value = false
  }
}

const handleTestEmail = async () => {
  if (!testEmailTo.value) {
    ElMessage.warning('请先填写收件地址')
    return
  }
  testLoading.value = true
  try {
    const res = await testEmail({
      to:             testEmailTo.value,
      smtp_host:      form.value.smtp_host,
      smtp_port:      form.value.smtp_port,
      smtp_username:  form.value.smtp_username,
      smtp_password:  form.value.smtp_password,
      smtp_from:      form.value.smtp_from,
      smtp_from_name: form.value.smtp_from_name,
      smtp_tls:       form.value.smtp_tls,
    })
    ElNotification({ title: '发送成功', message: res.data.message, type: 'success', duration: 3000 })
  } catch (err) {
    const msg = err.response?.data?.message || '发送失败'
    ElNotification({ title: '发送失败', message: msg, type: 'error', duration: 3000 })
  } finally {
    testLoading.value = false
  }
}
</script>

<style scoped>
.settings-page { min-height: 100vh; background: radial-gradient(circle at 16% 12%, rgba(37,99,235,0.10), transparent 30%), radial-gradient(circle at 86% 8%, rgba(20,184,166,0.10), transparent 28%), #f5f7fb; }
.page-wrap { padding: 0; }
.page-hero {
  position: relative;
  background: linear-gradient(135deg, #0f172a, #1e40af 58%, #0f766e);
  padding: 30px 36px 28px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  overflow: hidden;
}
.hero-content { position: relative; z-index: 2; }
.eyebrow { color: rgba(255,255,255,0.62); font-size: 12px; font-weight: 700; letter-spacing: .14em; text-transform: uppercase; margin-bottom: 6px; }
.hero-title { font-size: 24px; font-weight: 750; color: #fff; margin: 0 0 5px; letter-spacing: -0.45px; }
.hero-sub { font-size: 13px; color: rgba(255,255,255,0.76); margin: 0; }
.hero-btn { position: relative; z-index: 2; background: rgba(255,255,255,0.18) !important; border-color: rgba(255,255,255,0.34) !important; color: #fff !important; box-shadow: 0 10px 28px rgba(15,23,42,0.18); }
.bubble { position: absolute; border-radius: 999px; background: rgba(255,255,255,0.11); pointer-events: none; }
.bubble-a { width: 230px; height: 230px; right: -56px; top: -82px; }
.bubble-b { width: 126px; height: 126px; right: 190px; bottom: -60px; background: rgba(255,255,255,0.08); }
.bubble-c { width: 58px; height: 58px; left: 44%; top: 18px; background: rgba(255,255,255,0.10); }
.settings-overview { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin: 22px 36px 0; }
.overview-card { position: relative; background: rgba(255,255,255,0.88); border: 1px solid rgba(226,232,240,0.92); border-radius: 18px; padding: 16px 18px; box-shadow: 0 14px 34px rgba(15,23,42,0.06); overflow: hidden; }
.overview-card::after { content: ''; position: absolute; right: -22px; top: -26px; width: 78px; height: 78px; border-radius: 50%; opacity: .16; }
.accent-blue::after { background: #2563eb; }
.accent-violet::after { background: #7c3aed; }
.accent-teal::after { background: #0f766e; }
.accent-amber::after { background: #d97706; }
.overview-label { display: block; color: #64748b; font-size: 12px; font-weight: 650; }
.overview-card strong { display: block; color: #0f172a; font-size: 24px; margin-top: 6px; font-variant-numeric: tabular-nums; }
.overview-sub { display: block; color: #94a3b8; font-size: 12px; margin-top: 4px; }
.settings-panels { display: grid; grid-template-columns: 1fr 1fr; gap: 18px; margin: 18px 36px 28px; }
.info-panel { background: #fff; border: 1px solid #e2e8f0; border-radius: 18px; box-shadow: 0 12px 28px rgba(15,23,42,0.05); overflow: hidden; }
.panel-head { padding: 18px 20px 12px; border-bottom: 1px solid #f1f5f9; }
.panel-head h3 { margin: 0; color: #0f172a; font-size: 15px; }
.panel-head p { margin: 4px 0 0; color: #94a3b8; font-size: 12px; }
.kv-list { padding: 6px 20px 12px; }
.kv-row { display: flex; justify-content: space-between; gap: 18px; padding: 11px 0; border-bottom: 1px solid #f1f5f9; font-size: 13px; }
.kv-row:last-child { border-bottom: none; }
.kv-row span { color: #64748b; }
.kv-row strong { color: #1e293b; text-align: right; max-width: 68%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.settings-form { max-height: 70vh; overflow-y: auto; padding: 2px 4px 0; }
.form-section { background: #fff; border: 1px solid #e5e7eb; border-radius: 14px; padding: 16px 16px 10px; margin-bottom: 14px; }
.muted-section { background: linear-gradient(180deg, #fff, #f8fafc); }
.section-title { font-size: 14px; font-weight: 750; color: #0f172a; margin-bottom: 12px; letter-spacing: -0.2px; }
.form-grid.two { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.switch-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.switch-card { background: #fff; border: 1px solid #e2e8f0; border-radius: 12px; padding: 12px; }
.switch-card span { display: block; font-weight: 650; color: #334155; margin-bottom: 10px; }
.switch-card p { margin: 8px 0 0; color: #94a3b8; font-size: 12px; line-height: 1.45; }
.field-tip { font-size: 12px; color: #94a3b8; margin-top: 4px; line-height: 1.4; }
.test-row { display: grid; grid-template-columns: 1fr auto; gap: 10px; width: 100%; max-width: 460px; }
:deep(.settings-dialog .el-dialog) { border-radius: 22px !important; overflow: hidden; }
:deep(.settings-dialog .el-dialog__header) { padding: 20px 24px 12px; }
:deep(.settings-dialog .el-dialog__body) { padding: 10px 24px 4px; background: #f8fafc; }
:deep(.settings-dialog .el-dialog__footer) { padding: 14px 24px 20px; background: #f8fafc; }
@media (max-width: 1100px) {
  .settings-overview { grid-template-columns: repeat(2, 1fr); }
  .settings-panels { grid-template-columns: 1fr; }
}
</style>
