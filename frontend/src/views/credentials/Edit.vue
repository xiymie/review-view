<template>
  <div class="page-wrap">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">编辑凭据</h2>
        <p class="page-subtitle">修改仓库访问凭据</p>
      </div>
    </div>

    <el-form :model="form" :rules="rules" ref="formRef" label-position="top" style="max-width: 640px">
      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" placeholder="请输入凭据名称" />
      </el-form-item>

      <el-form-item label="用户名" prop="username">
        <el-input v-model="form.username" placeholder="请输入用户名" />
      </el-form-item>

      <el-form-item label="密码 / Token">
        <el-input
          v-model="form.password"
          type="password"
          show-password
          placeholder="留空则不修改"
        />
      </el-form-item>

      <el-form-item>
        <div class="form-actions">
          <el-button @click="router.push('/credentials')">取消</el-button>
          <el-button type="primary" @click="handleSubmit">保存修改</el-button>
        </div>
      </el-form-item>
    </el-form>

    <!-- 危险区 -->
    <div style="max-width: 640px; margin-top: 40px">
      <el-card class="danger-zone">
        <template #header>
          <span class="danger-title">危险区</span>
        </template>
        <div class="danger-body">
          <div>
            <p class="danger-label">删除凭据</p>
            <p class="danger-desc">此操作不可撤销，删除后相关项目将无法访问仓库。</p>
          </div>
          <el-button type="danger" plain @click="handleDelete">删除凭据</el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getCredential, updateCredential, deleteCredential } from '../../api/credentials'

const router = useRouter()
const route = useRoute()
const formRef = ref()

const id = route.params.id
const form = ref({ name: '', username: '', password: '' })

const rules = {
  name: [{ required: true, message: '请输入凭据名称', trigger: 'blur' }],
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
}

onMounted(async () => {
  try {
    const res = await getCredential(id)
    form.value.name = res.data.name || ''
    form.value.username = res.data.username || ''
    form.value.password = ''
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  }
})

const handleSubmit = async () => {
  await formRef.value.validate()
  try {
    const payload = { name: form.value.name, username: form.value.username }
    if (form.value.password) payload.password = form.value.password
    await updateCredential(id, payload)
    router.push('/credentials')
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  }
}

const handleDelete = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要删除凭据 "${form.value.name}" 吗？此操作不可撤销。`,
      '删除确认',
      {
        confirmButtonText: '确认删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger',
      }
    )
    await deleteCredential(id)
    ElMessage.success('删除成功')
    router.push('/credentials')
  } catch (err) {
    if (err !== 'cancel') {
      ElMessage.error(err.response?.data?.message || '操作失败')
    }
  }
}
</script>

<style scoped>
.page-wrap { padding: 32px 36px; }

.page-header { margin-bottom: 24px; }

.page-title { margin: 0 0 4px; font-size: 22px; font-weight: 700; color: #1e293b; letter-spacing: -0.3px; }
.page-subtitle { margin: 0; font-size: 14px; color: #64748b; }

.form-actions { display: flex; justify-content: flex-end; gap: 8px; width: 100%; }

.danger-zone { border-color: #fecaca !important; }
.danger-zone :deep(.el-card__header) { border-bottom-color: #fecaca; background: #fff5f5; }
.danger-title { color: #ef4444; font-weight: 600; }

.danger-body { display: flex; align-items: center; justify-content: space-between; }
.danger-label { margin: 0 0 4px; font-weight: 500; color: #1e293b; }
.danger-desc { margin: 0; font-size: 13px; color: #64748b; }
</style>
