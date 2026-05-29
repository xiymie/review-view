<template>
  <div class="page-wrap">
    <el-button link @click="router.push('/credentials')" class="back-btn">
      <el-icon><ArrowLeft /></el-icon> 返回
    </el-button>

    <div class="center-layout">
      <!-- 顶部图标区 -->
      <div class="hero">
        <div class="hero-icon">
          <el-icon :size="32" color="#fff"><Key /></el-icon>
        </div>
        <h1>新建仓库凭据</h1>
        <p>用于克隆和访问私有 Git 仓库</p>
      </div>

      <!-- 表单卡片 -->
      <el-card class="form-card" shadow="never">
        <el-form :model="form" :rules="rules" ref="formRef" label-position="top">

          <el-form-item prop="name">
            <template #label>
              <span class="field-label">凭据名称</span>
            </template>
            <el-input
              v-model="form.name"
              placeholder="例如：GitHub Personal Token"
              size="large"
              clearable
            >
              <template #prefix><el-icon color="#9ca3af"><Tickets /></el-icon></template>
            </el-input>
          </el-form-item>

          <el-form-item prop="username">
            <template #label>
              <span class="field-label">用户名</span>
            </template>
            <el-input
              v-model="form.username"
              placeholder="Git 用户名（如 GitHub 账号）"
              size="large"
              clearable
            >
              <template #prefix><el-icon color="#9ca3af"><User /></el-icon></template>
            </el-input>
          </el-form-item>

          <el-form-item prop="password">
            <template #label>
              <div class="label-row">
                <span class="field-label">密码 / Access Token</span>
                <el-tag size="small" type="warning" effect="light">敏感信息</el-tag>
              </div>
            </template>
            <el-input
              v-model="form.password"
              type="password"
              show-password
              placeholder="密码或 Personal Access Token"
              size="large"
            >
              <template #prefix><el-icon color="#9ca3af"><Lock /></el-icon></template>
            </el-input>
            <div class="field-hint">Token 加密存储，不会在日志中明文展示</div>
          </el-form-item>

          <!-- 分隔线 -->
          <el-divider />

          <!-- 操作按钮 -->
          <div class="form-actions">
            <el-button size="large" @click="router.push('/credentials')" style="flex:1">取消</el-button>
            <el-button
              size="large"
              type="primary"
              @click="handleSubmit"
              :loading="loading"
              class="submit-btn"
              style="flex:2"
            >
              <el-icon v-if="!loading"><CircleCheck /></el-icon>
              创建凭据
            </el-button>
          </div>
        </el-form>
      </el-card>

      <!-- 底部提示 -->
      <div class="help-text">
        <el-icon color="#9ca3af"><InfoFilled /></el-icon>
        <span>GitHub 建议使用 <strong>Personal Access Token</strong>（Settings → Developer Settings → Tokens），
        权限勾选 <code>repo</code> 即可。</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Key, User, Lock, Tickets, CircleCheck, InfoFilled } from '@element-plus/icons-vue'
import { createCredential } from '../../api/credentials'

const router = useRouter()
const formRef = ref()
const loading = ref(false)

const form = ref({ name: '', username: '', password: '' })

const rules = {
  name:     [{ required: true, message: '请输入凭据名称', trigger: 'blur' }],
  username: [{ required: true, message: '请输入用户名',   trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码或 Token', trigger: 'blur' }],
}

const handleSubmit = async () => {
  await formRef.value.validate()
  loading.value = true
  try {
    await createCredential({ ...form.value })
    ElMessage.success('凭据创建成功')
    router.push('/credentials')
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '创建失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.page-wrap { padding: 32px 36px; min-height: 100%; }

.back-btn { color: #6b7280; margin-bottom: 20px; padding: 0; font-size: 13px; }
.back-btn:hover { color: #2563eb; }

/* 居中布局 */
.center-layout {
  max-width: 520px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
}

/* 顶部英雄区 */
.hero {
  text-align: center;
  padding-top: 12px;
}

.hero-icon {
  width: 68px; height: 68px;
  border-radius: 20px;
  background: linear-gradient(135deg, #2563eb 0%, #7c3aed 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 16px;
  box-shadow: 0 8px 24px rgba(37,99,235,0.3);
}

.hero h1 {
  font-size: 22px;
  font-weight: 700;
  color: #111827;
  margin: 0 0 6px;
}

.hero p {
  font-size: 14px;
  color: #6b7280;
  margin: 0;
}

/* 表单卡片 */
.form-card {
  width: 100%;
  border-radius: 12px !important;
  border: 1px solid #e2e8f0 !important;
  box-shadow: none !important;
}

.form-card :deep(.el-card__body) { padding: 28px; }
.form-card :deep(.el-form-item) { margin-bottom: 20px; }
.form-card :deep(.el-form-item:last-child) { margin-bottom: 0; }

/* 字段标签 */
.field-label { font-size: 13.5px; font-weight: 500; color: #374151; }
.label-row { display: flex; align-items: center; gap: 8px; }
.field-hint { font-size: 12px; color: #9ca3af; margin-top: 5px; }

/* 操作区 */
.form-actions { display: flex; gap: 10px; }

.submit-btn {
  background: linear-gradient(90deg, #2563eb, #7c3aed) !important;
  border: none !important;
  font-size: 15px;
}

.submit-btn:hover { opacity: 0.9; }

/* 底部帮助 */
.help-text {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 12.5px;
  color: #64748b;
  background: #f8fafc;
  border-radius: 10px;
  padding: 14px 16px;
  line-height: 1.6;
  width: 100%;
  box-sizing: border-box;
  border: 1px solid #e2e8f0;
}

.help-text code {
  background: #e5e7eb;
  padding: 1px 5px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 11.5px;
  color: #374151;
}

.help-text strong { color: #374151; }
</style>
