<template>
  <div class="page-wrap">
    <div class="page-hero">
      <div class="hero-content">
        <h1 class="hero-title">推送通知</h1>
        <p class="hero-sub">配置扫描完成后的消息推送方式</p>
      </div>
      <div class="deco-circles">
        <div class="deco c1"></div>
        <div class="deco c2"></div>
      </div>
    </div>

    <div class="notify-body" v-loading="loading">

      <!-- 全局开关 -->
      <div class="global-switch-card">
        <div class="switch-info">
          <div class="switch-title">启用推送通知</div>
          <div class="switch-desc">开启后，定时扫描完成时将通过以下已配置的渠道发送通知</div>
        </div>
        <el-switch v-model="form.notify_enabled" size="large" />
      </div>

      <!-- 邮件 -->
      <div class="channel-card" :class="{ disabled: !form.notify_enabled }">
        <div class="channel-header">
          <div class="channel-icon email-icon">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
              <polyline points="22,6 12,13 2,6"/>
            </svg>
          </div>
          <div class="channel-info">
            <div class="channel-name">邮件通知</div>
            <div class="channel-desc">发送带 HTML 正文和 .md 附件的审计报告邮件（需管理员配置 SMTP）</div>
          </div>
          <div class="channel-status" :class="emailActive ? 'active' : 'inactive'">
            {{ emailActive ? '已配置' : '未配置' }}
          </div>
        </div>
        <div class="channel-body">
          <el-form-item label="收件地址">
            <el-input
              v-model="form.notify_emails"
              :disabled="!form.notify_enabled"
              placeholder="支持多个邮箱，逗号分隔：a@co.com, b@co.com"
              clearable
            />
            <div class="field-tip">所有地址将同时收到邮件，适合团队共享审计结果</div>
          </el-form-item>
          <div class="email-test-bar">
            <el-button
              size="small"
              :loading="testing"
              :disabled="!form.notify_enabled || form.notify_emails.trim() === ''"
              @click="handleTestEmail"
            >发送测试邮件</el-button>
            <span class="field-tip">使用管理员配置的 SMTP，向上方收件地址发送一封测试邮件，确认是否可用</span>
          </div>
        </div>
      </div>

      <!-- 企业微信 -->
      <div class="channel-card" :class="{ disabled: !form.notify_enabled }">
        <div class="channel-header">
          <div class="channel-icon wecom-icon">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
            </svg>
          </div>
          <div class="channel-info">
            <div class="channel-name">企业微信机器人</div>
            <div class="channel-desc">在企业微信群中添加群机器人，扫描完成后自动推送 Markdown 摘要</div>
          </div>
          <div class="channel-status" :class="wecomActive ? 'active' : 'inactive'">
            {{ wecomActive ? '已配置' : '未配置' }}
          </div>
        </div>
        <div class="channel-body">
          <el-form-item label="Webhook 地址">
            <el-input
              v-model="form.notify_wecom_webhook"
              :disabled="!form.notify_enabled"
              placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..."
              clearable
            />
            <div class="field-tip">企业微信群 → 群设置 → 群机器人 → 添加机器人 → 复制 Webhook 地址</div>
          </el-form-item>
        </div>
      </div>

      <!-- OA（预留）-->
      <div class="channel-card disabled">
        <div class="channel-header">
          <div class="channel-icon oa-icon">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
              <line x1="8" y1="21" x2="16" y2="21"/>
              <line x1="12" y1="17" x2="12" y2="21"/>
            </svg>
          </div>
          <div class="channel-info">
            <div class="channel-name">OA 系统通知</div>
            <div class="channel-desc">对接企业内部 OA 系统，推送审计通知至工作流或待办</div>
          </div>
          <div class="channel-status coming">即将支持</div>
        </div>
        <div class="channel-body">
          <div class="coming-placeholder">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
            该渠道正在开发中，敬请期待
          </div>
        </div>
      </div>

      <!-- 保存 -->
      <div class="save-bar">
        <el-button type="primary" :loading="saving" @click="handleSave">保存配置</el-button>
      </div>

    </div>
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
.page-wrap { padding: 0; }

.page-hero {
  position: relative;
  background: linear-gradient(135deg, #0f172a, #1e3a8a, #2563eb);
  padding: 24px 36px;
  display: flex;
  align-items: center;
  overflow: hidden;
}
.hero-title { font-size: 22px; font-weight: 700; color: #fff; margin: 0 0 4px; }
.hero-sub { font-size: 13px; color: rgba(255,255,255,0.75); margin: 0; }
.deco-circles { position: absolute; right: 0; top: 0; bottom: 0; width: 200px; pointer-events: none; }
.deco { position: absolute; border-radius: 50%; background: rgba(255,255,255,0.08); }
.c1 { width: 180px; height: 180px; right: -40px; top: -60px; }
.c2 { width: 100px; height: 100px; right: 60px; bottom: -30px; }

.notify-body {
  padding: 24px 36px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 720px;
}

/* 全局开关卡片 */
.global-switch-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 18px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.switch-title { font-size: 14px; font-weight: 600; color: #1e293b; margin-bottom: 3px; }
.switch-desc { font-size: 12.5px; color: #94a3b8; }

/* 渠道卡片 */
.channel-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  overflow: hidden;
  transition: border-color 0.2s, opacity 0.2s;
}
.channel-card.disabled { opacity: 0.55; }

.channel-header {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 20px;
  border-bottom: 1px solid #f3f4f6;
  background: #fafafa;
}

.channel-icon {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.email-icon  { background: linear-gradient(135deg, #dbeafe, #bfdbfe); color: #2563eb; }
.wecom-icon  { background: linear-gradient(135deg, #d1fae5, #a7f3d0); color: #059669; }
.oa-icon     { background: linear-gradient(135deg, #f3f4f6, #e5e7eb); color: #6b7280; }

.channel-info { flex: 1; min-width: 0; }
.channel-name { font-size: 14px; font-weight: 600; color: #1e293b; margin-bottom: 2px; }
.channel-desc { font-size: 12px; color: #94a3b8; line-height: 1.5; }

.channel-status {
  font-size: 11.5px;
  font-weight: 600;
  padding: 3px 10px;
  border-radius: 20px;
  white-space: nowrap;
  flex-shrink: 0;
}
.channel-status.active   { background: #ecfdf5; color: #059669; }
.channel-status.inactive { background: #f8fafc; color: #94a3b8; }
.channel-status.coming   { background: #faf5ff; color: #7c3aed; }

.channel-body { padding: 16px 20px; }
.channel-body :deep(.el-form-item) { margin-bottom: 0; }
.email-test-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 12px;
}
.email-test-bar .field-tip { margin-top: 0; }
.channel-body :deep(.el-form-item__label) {
  font-size: 13px;
  font-weight: 500;
  color: #374151;
  padding-bottom: 6px;
}

.field-tip {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 5px;
  line-height: 1.5;
}

.coming-placeholder {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #9ca3af;
  padding: 4px 0;
}

/* 保存按钮 */
.save-bar { padding: 4px 0 8px; }
</style>
