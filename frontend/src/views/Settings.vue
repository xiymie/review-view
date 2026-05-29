<template>
  <div class="page-wrap">
    <div class="page-hero">
      <div class="hero-content">
        <h1 class="hero-title">全局设置</h1>
        <p class="hero-sub">配置系统全局运行参数</p>
      </div>
      <div class="deco-circles">
        <div class="deco c1"></div>
        <div class="deco c2"></div>
      </div>
    </div>

    <el-card style="max-width: 640px" class="settings-card">
      <el-form :model="form" label-position="top" v-loading="loading">
        <el-form-item label="最大并发任务数">
          <el-input-number v-model="form.max_concurrent_tasks" :min="1" style="width: 100%" />
        </el-form-item>

        <el-form-item label="全局溢出策略">
          <el-radio-group v-model="form.overflow_strategy">
            <el-radio value="queue">排队等待</el-radio>
            <el-radio value="reject">拒绝</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="任务超时时间（分钟）">
          <el-input-number v-model="form.task_timeout" :min="1" style="width: 100%" />
        </el-form-item>

        <el-form-item label="仓库根目录">
          <el-input v-model="form.repo_base_dir" placeholder="/data/repos" />
        </el-form-item>

        <el-divider>扫描行为</el-divider>

        <el-form-item label="定时扫描无新提交时">
          <el-switch
            v-model="form.scheduled_scan_unchanged"
            active-text="继续扫描"
            inactive-text="跳过"
          />
          <div class="field-tip">关闭时，定时任务遇到无新提交将直接跳过，不发起 Review 也不发邮件</div>
        </el-form-item>

        <el-form-item label="手动扫描无新提交时">
          <el-switch
            v-model="form.manual_scan_unchanged"
            active-text="继续扫描"
            inactive-text="跳过"
          />
          <div class="field-tip">关闭时，手动触发遇到无新提交将直接跳过，不发起 Review 也不发邮件</div>
        </el-form-item>

        <el-divider>邮件推送 SMTP 配置</el-divider>

        <el-form-item label="SMTP 服务器地址">
          <el-input v-model="form.smtp_host" placeholder="smtp.example.com" clearable />
        </el-form-item>

        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="SMTP 端口">
              <el-input v-model="form.smtp_port" placeholder="465 / 587" clearable />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="加密方式">
              <el-select v-model="form.smtp_tls" style="width: 100%">
                <el-option label="SSL/TLS（端口 465）" value="true" />
                <el-option label="STARTTLS（端口 587）" value="false" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="SMTP 账号">
          <el-input v-model="form.smtp_username" placeholder="no-reply@example.com" clearable />
        </el-form-item>

        <el-form-item label="SMTP 密码">
          <el-input v-model="form.smtp_password" type="password" show-password placeholder="留空则不更新密码" clearable />
        </el-form-item>

        <el-form-item label="发件人地址（From）">
          <el-input v-model="form.smtp_from" placeholder="no-reply@example.com" clearable />
        </el-form-item>

        <el-form-item label="发件人显示名称">
          <el-input v-model="form.smtp_from_name" placeholder="代码审计" clearable />
          <div class="field-tip">邮件客户端中"发件人"栏显示的名称，留空则只显示邮箱地址</div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSave">保存设置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSettings, updateSettings } from '../api/settings'

const loading = ref(false)

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
})

onMounted(async () => {
  loading.value = true
  try {
    const res = await getSettings()
    Object.assign(form.value, res.data)
    form.value.smtp_password = ''
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  } finally {
    loading.value = false
  }
})

const handleSave = async () => {
  loading.value = true
  try {
    await updateSettings(form.value)
    ElMessage.success('保存成功')
    form.value.smtp_password = ''
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.page-wrap { padding: 0; }
.settings-card { margin: 24px 36px; }
.field-tip {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 4px;
  line-height: 1.4;
}

.page-hero {
  position: relative;
  background: linear-gradient(135deg, #0f172a, #1e1b4b);
  padding: 24px 36px;
  display: flex;
  align-items: center;
  overflow: hidden;
}
.hero-title { font-size: 22px; font-weight: 700; color: #fff; margin: 0 0 4px; }
.hero-sub { font-size: 13px; color: rgba(255,255,255,0.6); margin: 0; }
.deco-circles { position: absolute; right: 0; top: 0; bottom: 0; width: 200px; pointer-events: none; }
.deco { position: absolute; border-radius: 50%; background: rgba(255,255,255,0.05); }
.c1 { width: 180px; height: 180px; right: -40px; top: -60px; }
.c2 { width: 100px; height: 100px; right: 60px; bottom: -30px; }
</style>
