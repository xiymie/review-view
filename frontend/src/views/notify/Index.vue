<template>
  <div class="page-wrap notify-page">
    <div class="page-hero">
      <div class="bubble bubble-a"></div>
      <div class="bubble bubble-b"></div>
      <div class="bubble bubble-c"></div>
      <div class="hero-content">
        <div class="eyebrow">Notification Hub</div>
        <h1 class="hero-title">推送通知</h1>
        <p class="hero-sub">把 Review 与巡检结果推送到团队邮箱和协作群，让结果不只停留在页面里</p>
      </div>
      <div class="hero-status" :class="form.notify_enabled ? 'on' : 'off'">
        <span class="status-dot"></span>
        {{ form.notify_enabled ? '通知已启用' : '通知已关闭' }}
      </div>
    </div>

    <main class="notify-shell" v-loading="loading">
      <section class="control-panel">
        <div class="control-copy">
          <div class="control-kicker">全局开关</div>
          <h2>启用推送通知</h2>
          <p>开启后，定时扫描完成时会通过下方已配置渠道发送摘要和报告。关闭后保留配置但不发送。</p>
        </div>
        <el-switch v-model="form.notify_enabled" size="large" />
      </section>

      <section class="channel-grid">
        <article class="channel-card" :class="{ disabled: !form.notify_enabled }">
          <div class="channel-glow email-glow"></div>
          <div class="channel-header">
            <div class="channel-icon email-icon">
              <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
                <polyline points="22,6 12,13 2,6"/>
              </svg>
            </div>
            <div class="channel-info">
              <div class="channel-name">邮件通知</div>
              <div class="channel-desc">发送 HTML 正文和 .md 附件，适合归档和多人同步。</div>
            </div>
            <span class="channel-status" :class="emailActive ? 'active' : 'inactive'">
              {{ emailActive ? '已配置' : '未配置' }}
            </span>
          </div>
          <div class="channel-body">
            <el-form-item label="收件地址">
              <el-input
                v-model="form.notify_emails"
                :disabled="!form.notify_enabled"
                placeholder="支持多个邮箱，逗号分隔：a@co.com, b@co.com"
                clearable
              />
              <div class="field-tip">所有地址会同时收到邮件，适合团队共享审计结果。</div>
            </el-form-item>
            <div class="email-test-bar">
              <el-button
                size="small"
                :loading="testing"
                :disabled="!form.notify_enabled || form.notify_emails.trim() === ''"
                @click="handleTestEmail"
              >发送测试邮件</el-button>
              <span class="field-tip inline-tip">使用管理员配置的 SMTP 发送测试邮件。</span>
            </div>
          </div>
        </article>

        <article class="channel-card" :class="{ disabled: !form.notify_enabled }">
          <div class="channel-glow wecom-glow"></div>
          <div class="channel-header">
            <div class="channel-icon wecom-icon">
              <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
              </svg>
            </div>
            <div class="channel-info">
              <div class="channel-name">企业微信机器人</div>
              <div class="channel-desc">扫描完成后自动推送 Markdown 摘要到企业微信群。</div>
            </div>
            <span class="channel-status" :class="wecomActive ? 'active' : 'inactive'">
              {{ wecomActive ? '已配置' : '未配置' }}
            </span>
          </div>
          <div class="channel-body">
            <el-form-item label="Webhook 地址">
              <el-input
                v-model="form.notify_wecom_webhook"
                :disabled="!form.notify_enabled"
                placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..."
                clearable
              />
              <div class="field-tip">企业微信群 → 群设置 → 群机器人 → 添加机器人 → 复制 Webhook。</div>
            </el-form-item>
          </div>
        </article>
      </section>

      <section class="coming-panel">
        <div class="coming-icon">
          <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
        </div>
        <div>
          <div class="coming-title">OA 系统通知</div>
          <p>后续可对接企业内部 OA，推送审计通知至工作流或待办。当前先保留入口说明。</p>
        </div>
        <span class="channel-status coming">即将支持</span>
      </section>

      <div class="save-bar">
        <el-button type="primary" size="large" :loading="saving" @click="handleSave">保存通知配置</el-button>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getMe, updateMe, testMyEmail } from '../../api/users'

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)

const form = ref({
  notify_enabled: false,
  notify_emails: '',
  notify_wecom_webhook: '',
})

const emailActive = computed(() => form.value.notify_enabled && form.value.notify_emails.trim() !== '')
const wecomActive = computed(() => form.value.notify_enabled && form.value.notify_wecom_webhook.trim() !== '')

onMounted(async () => {
  loading.value = true
  try {
    const res = await getMe()
    const u = res.data
    form.value = {
      notify_enabled: u.notify_enabled || false,
      notify_emails: u.notify_emails || '',
      notify_wecom_webhook: u.notify_wecom_webhook || '',
    }
  } catch {
    ElMessage.error('加载配置失败')
  } finally {
    loading.value = false
  }
})

async function handleSave() {
  saving.value = true
  try {
    await updateMe(form.value)
    ElMessage.success('保存成功')
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleTestEmail() {
  if (form.value.notify_emails.trim() === '') {
    ElMessage.warning('请先填写收件地址')
    return
  }
  testing.value = true
  try {
    const res = await testMyEmail({ notify_emails: form.value.notify_emails })
    ElMessage.success(res.data.message || '测试邮件已发送')
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '发送失败')
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.notify-page { min-height: 100vh; background: radial-gradient(circle at 14% 8%, rgba(37,99,235,0.10), transparent 28%), radial-gradient(circle at 88% 18%, rgba(16,185,129,0.10), transparent 26%), #f5f7fb; padding: 0; }
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
.bubble { position: absolute; border-radius: 999px; background: rgba(255,255,255,0.11); pointer-events: none; }
.bubble-a { width: 230px; height: 230px; right: -56px; top: -82px; }
.bubble-b { width: 126px; height: 126px; right: 190px; bottom: -60px; background: rgba(255,255,255,0.08); }
.bubble-c { width: 58px; height: 58px; left: 44%; top: 18px; background: rgba(255,255,255,0.10); }
.hero-status { position: relative; z-index: 2; display: inline-flex; align-items: center; gap: 8px; padding: 9px 14px; border-radius: 999px; background: rgba(255,255,255,0.16); border: 1px solid rgba(255,255,255,0.28); color: #fff; font-size: 13px; font-weight: 650; }
.status-dot { width: 8px; height: 8px; border-radius: 50%; background: currentColor; box-shadow: 0 0 0 4px rgba(255,255,255,0.12); }
.hero-status.on { color: #bbf7d0; }
.hero-status.off { color: #cbd5e1; }
.notify-shell { width: 100%; padding: 24px 36px 36px; }
.control-panel { background: rgba(255,255,255,0.9); border: 1px solid rgba(226,232,240,0.95); border-radius: 22px; padding: 22px 24px; display: flex; align-items: center; justify-content: space-between; gap: 24px; box-shadow: 0 18px 42px rgba(15,23,42,0.07); }
.control-kicker { color: #2563eb; font-size: 12px; font-weight: 750; letter-spacing: .12em; text-transform: uppercase; margin-bottom: 6px; }
.control-copy h2 { margin: 0 0 6px; color: #0f172a; font-size: 20px; letter-spacing: -0.35px; }
.control-copy p { margin: 0; color: #64748b; font-size: 13px; line-height: 1.6; }
.channel-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 18px; margin-top: 18px; }
.channel-card { position: relative; background: #fff; border: 1px solid #e2e8f0; border-radius: 22px; overflow: hidden; box-shadow: 0 14px 34px rgba(15,23,42,0.06); transition: transform .18s ease, box-shadow .18s ease, opacity .18s ease; }
.channel-card:hover { transform: translateY(-2px); box-shadow: 0 20px 44px rgba(15,23,42,0.09); }
.channel-card.disabled { opacity: .62; }
.channel-glow { position: absolute; right: -34px; top: -38px; width: 118px; height: 118px; border-radius: 50%; opacity: .14; }
.email-glow { background: #2563eb; }
.wecom-glow { background: #059669; }
.channel-header { position: relative; display: flex; align-items: center; gap: 14px; padding: 20px 22px 16px; border-bottom: 1px solid #f1f5f9; }
.channel-icon { width: 42px; height: 42px; border-radius: 14px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.email-icon  { background: linear-gradient(135deg, #dbeafe, #bfdbfe); color: #2563eb; }
.wecom-icon  { background: linear-gradient(135deg, #d1fae5, #a7f3d0); color: #059669; }
.channel-info { flex: 1; min-width: 0; }
.channel-name { font-size: 15px; font-weight: 750; color: #0f172a; margin-bottom: 3px; letter-spacing: -0.18px; }
.channel-desc { font-size: 12.5px; color: #64748b; line-height: 1.55; }
.channel-status { font-size: 11.5px; font-weight: 700; padding: 4px 10px; border-radius: 999px; white-space: nowrap; flex-shrink: 0; }
.channel-status.active   { background: #ecfdf5; color: #059669; }
.channel-status.inactive { background: #f8fafc; color: #94a3b8; }
.channel-status.coming   { background: #faf5ff; color: #7c3aed; }
.channel-body { padding: 18px 22px 20px; }
.channel-body :deep(.el-form-item) { margin-bottom: 0; }
.channel-body :deep(.el-form-item__label) { font-size: 13px; font-weight: 650; color: #374151; padding-bottom: 7px; }
.email-test-bar { display: flex; align-items: center; gap: 12px; margin-top: 12px; }
.field-tip { font-size: 12px; color: #94a3b8; margin-top: 5px; line-height: 1.5; }
.inline-tip { margin-top: 0; }
.coming-panel { margin-top: 18px; background: rgba(255,255,255,0.78); border: 1px dashed #cbd5e1; border-radius: 18px; padding: 16px 18px; display: flex; align-items: center; gap: 14px; color: #64748b; }
.coming-icon { width: 40px; height: 40px; border-radius: 13px; display: flex; align-items: center; justify-content: center; color: #7c3aed; background: #faf5ff; flex-shrink: 0; }
.coming-title { font-size: 14px; font-weight: 750; color: #334155; }
.coming-panel p { margin: 4px 0 0; font-size: 12.5px; line-height: 1.5; }
.coming-panel .channel-status { margin-left: auto; }
.save-bar { display: flex; justify-content: center; padding: 22px 0 0; }
.save-bar :deep(.el-button) { min-width: 180px; height: 42px; border-radius: 999px !important; }
@media (max-width: 980px) {
  .channel-grid { grid-template-columns: 1fr; }
  .notify-shell { padding-left: 20px; padding-right: 20px; }
}
</style>
