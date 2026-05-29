<template>
  <div class="page-wrap">
    <el-button link @click="router.push('/users')" class="back-btn">
      <el-icon><ArrowLeft /></el-icon> 返回
    </el-button>

    <div class="page-header">
      <div class="header-icon">
        <el-icon :size="24" color="#fff"><UserFilled /></el-icon>
      </div>
      <div>
        <h1>编辑用户 · {{ form.username }}</h1>
        <p>修改用户权限和个人信息</p>
      </div>
    </div>

    <div class="form-body" v-loading="initLoading">
      <el-form ref="formRef" :model="form" label-position="top" class="main-form">

        <section class="form-section">
          <div class="section-label"><span class="section-num">01</span>权限设置</div>
          <el-form-item prop="role">
            <div class="role-cards">
              <div class="role-card" :class="{ active: form.role === 'user' }" @click="isSuperAdmin && (form.role = 'user')">
                <div class="role-icon">👤</div>
                <div class="role-info">
                  <strong>普通用户</strong>
                  <span>可访问仪表盘、项目、任务</span>
                </div>
                <el-icon v-if="form.role === 'user'" color="#2563eb"><Select /></el-icon>
              </div>
              <div class="role-card" :class="{ active: form.role === 'admin' }" @click="isSuperAdmin && (form.role = 'admin')">
                <div class="role-icon">🛡️</div>
                <div class="role-info">
                  <strong>管理员</strong>
                  <span>拥有全部配置权限，可管理普通用户</span>
                </div>
                <el-icon v-if="form.role === 'admin'" color="#2563eb"><Select /></el-icon>
              </div>
              <div class="role-card" :class="{ active: form.role === 'super_admin' }" @click="isSuperAdmin && (form.role = 'super_admin')">
                <div class="role-icon">👑</div>
                <div class="role-info">
                  <strong>超级管理员</strong>
                  <span>拥有全部权限，可管理所有账户</span>
                </div>
                <el-icon v-if="form.role === 'super_admin'" color="#2563eb"><Select /></el-icon>
              </div>
            </div>
            <div v-if="!isSuperAdmin" style="font-size:12px;color:#9ca3af;margin-top:6px">只有超级管理员可以修改权限</div>
          </el-form-item>
        </section>

        <section class="form-section">
          <div class="section-label"><span class="section-num">02</span>个人信息</div>
          <div class="field-grid">
            <el-form-item label="邮箱">
              <el-input v-model="form.email" placeholder="example@company.com" clearable />
            </el-form-item>
            <el-form-item label="手机号">
              <el-input v-model="form.phone" placeholder="13800138000" clearable />
            </el-form-item>
            <el-form-item label="岗位">
              <el-input v-model="form.position" placeholder="例如：后端工程师" clearable />
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="form.remark" placeholder="备注信息" clearable />
            </el-form-item>
          </div>
        </section>

        <section class="form-section">
          <div class="section-label"><span class="section-num">03</span>重置密码 <el-tag size="small" type="info" effect="plain" style="margin-left:8px">可选</el-tag></div>
          <el-form-item label="新密码">
            <el-input v-model="form.password" type="password" show-password placeholder="留空则不修改密码" clearable style="max-width:360px">
              <template #prefix><el-icon color="#9ca3af"><Lock /></el-icon></template>
            </el-input>
          </el-form-item>
        </section>

        <div class="form-actions">
          <el-button size="large" @click="router.push('/users')">取消</el-button>
          <el-button size="large" type="primary" :loading="submitting" @click="handleSubmit" class="submit-btn">
            保存修改
          </el-button>
        </div>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Lock, Select, UserFilled } from '@element-plus/icons-vue'
import { getUser, updateUser } from '../../api/users'

const router = useRouter()
const route = useRoute()
const id = route.params.id
const formRef = ref(null)
const submitting = ref(false)
const initLoading = ref(false)
const isSuperAdmin = computed(() => localStorage.getItem('role') === 'super_admin')

const form = ref({
  username: '', role: 'user',
  email: '', phone: '', position: '', remark: '', password: '',
})

onMounted(async () => {
  initLoading.value = true
  try {
    const res = await getUser(id)
    const u = res.data
    form.value = {
      username: u.username, role: u.role,
      email: u.email || '', phone: u.phone || '',
      position: u.position || '', remark: u.remark || '',
      password: '',
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载失败')
  } finally {
    initLoading.value = false
  }
})

async function handleSubmit() {
  submitting.value = true
  try {
    await updateUser(id, form.value)
    ElMessage.success('保存成功')
    router.push('/users')
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.page-wrap { padding: 32px 36px; min-height: 100%; }
.back-btn { color: #6b7280; margin-bottom: 20px; padding: 0; font-size: 13px; }
.back-btn:hover { color: #2563eb; }
.page-header { display: flex; align-items: center; gap: 16px; margin-bottom: 28px; }
.header-icon {
  width: 48px; height: 48px; border-radius: 12px;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  box-shadow: 0 4px 14px rgba(37,99,235,0.25);
}
.page-header h1 { font-size: 20px; font-weight: 700; color: #111827; margin: 0 0 4px; }
.page-header p  { font-size: 13px; color: #6b7280; margin: 0; }

.form-body { max-width: 760px; }
.main-form { width: 100%; }

.form-section {
  background: #fff; border-radius: 12px;
  padding: 22px 24px; margin-bottom: 14px;
  border: 1px solid #e2e8f0;
}
.section-label {
  display: flex; align-items: center; gap: 10px;
  font-size: 14px; font-weight: 600; color: #374151; margin-bottom: 18px;
}
.section-num {
  width: 22px; height: 22px; border-radius: 50%;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  color: #fff; font-size: 11px; font-weight: 700;
  display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.field-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0 20px; }
.field-grid :deep(.el-form-item) { margin-bottom: 16px; }

.role-cards { display: flex; gap: 12px; }
.role-card {
  flex: 1; display: flex; align-items: center; gap: 12px;
  padding: 14px 16px; border-radius: 10px;
  border: 2px solid #f0f0f0; background: #fafafa;
  cursor: pointer; transition: all 0.18s;
}
.role-card:hover { border-color: #93c5fd; background: #eff6ff; }
.role-card.active {
  border-color: #2563eb;
  background: linear-gradient(135deg, #eff6ff, #f0f0ff);
  box-shadow: 0 0 0 3px rgba(37,99,235,0.1);
}
.role-icon { font-size: 22px; flex-shrink: 0; }
.role-info { flex: 1; }
.role-info strong { display: block; font-size: 13px; color: #111827; margin-bottom: 2px; }
.role-info span   { font-size: 12px; color: #6b7280; }

.form-actions {
  display: flex; justify-content: flex-end; gap: 10px;
  background: #fff; border-radius: 12px;
  padding: 16px 24px; border: 1px solid #e2e8f0;
}
.submit-btn {
  background: linear-gradient(90deg, #2563eb, #7c3aed) !important;
  border: none !important; padding: 0 28px;
}
.submit-btn:hover { opacity: 0.9; }
</style>
