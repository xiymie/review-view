<template>
  <div class="page-wrap">
    <div class="page-header">
      <div class="header-left">
        <div class="header-icon">
          <el-icon :size="24" color="#fff"><UserFilled /></el-icon>
        </div>
        <div>
          <h1>用户管理</h1>
          <p>管理系统用户账号和权限</p>
        </div>
      </div>
      <el-button type="primary" class="add-btn" @click="router.push('/users/new')">
        <el-icon><Plus /></el-icon> 新建用户
      </el-button>
    </div>

    <div class="table-card" v-loading="loading">
      <el-table :data="users" style="width:100%" row-class-name="table-row">
        <el-table-column label="用户名" min-width="130">
          <template #default="{ row }">
            <div class="user-cell">
              <el-avatar :size="28" class="user-avatar">{{ row.username[0]?.toUpperCase() }}</el-avatar>
              <span class="user-name">{{ row.username }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="权限" width="110">
          <template #default="{ row }">
            <el-tag
              :type="row.role === 'super_admin' ? 'danger' : row.role === 'admin' ? 'warning' : 'info'"
              size="small" effect="plain"
            >
              {{ roleLabel(row.role) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="邮箱" prop="email" min-width="160" show-overflow-tooltip />
        <el-table-column label="手机" prop="phone" width="130" />
        <el-table-column label="岗位" prop="position" width="120" show-overflow-tooltip />
        <el-table-column label="备注" prop="remark" min-width="140" show-overflow-tooltip />
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="router.push(`/users/${row.id}/edit`)">编辑</el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { UserFilled, Plus } from '@element-plus/icons-vue'
import { listUsers, deleteUser } from '../../api/users'

const router = useRouter()
const loading = ref(false)
const users = ref([])

onMounted(load)

async function load() {
  loading.value = true
  try {
    const res = await listUsers()
    users.value = res.data
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`确认删除用户 "${row.username}"？`, '删除确认', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning',
    })
    await deleteUser(row.id)
    ElMessage.success('删除成功')
    load()
  } catch (err) {
    if (err !== 'cancel' && err?.type !== 'cancel') {
      ElMessage.error(err.response?.data?.message || '删除失败')
    }
  }
}

function roleLabel(role) {
  if (role === 'super_admin') return '超级管理员'
  if (role === 'admin') return '管理员'
  return '普通用户'
}

function formatDate(s) {
  if (!s) return '-'
  return new Date(s).toLocaleString('zh-CN', { hour12: false }).replace(/\//g, '-')
}
</script>

<style scoped>
.page-wrap { padding: 32px 36px; }

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}
.header-left { display: flex; align-items: center; gap: 16px; }
.header-icon {
  width: 48px; height: 48px;
  border-radius: 12px;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 4px 14px rgba(37,99,235,0.25);
}
.page-header h1 { font-size: 20px; font-weight: 700; color: #111827; margin: 0 0 4px; }
.page-header p  { font-size: 13px; color: #6b7280; margin: 0; }
.add-btn {
  background: linear-gradient(90deg, #2563eb, #7c3aed) !important;
  border: none !important;
}

.table-card {
  background: #fff;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  overflow: hidden;
}

.user-cell { display: flex; align-items: center; gap: 10px; }
.user-avatar { background: #2563eb; color: #fff; font-size: 12px; font-weight: 700; flex-shrink: 0; }
.user-name { font-weight: 500; color: #111827; }

:deep(.table-row td) { padding: 12px 0; }
:deep(.el-table th) { background: #f8fafc !important; color: #374151; font-weight: 600; font-size: 13px; }
</style>
