<template>
  <div class="page-wrap">
    <div class="page-hero">
      <div class="hero-content">
        <h1 class="hero-title">仓库凭据</h1>
        <p class="hero-sub">管理访问代码仓库所需的认证凭据</p>
      </div>
      <div class="hero-actions">
        <el-button type="primary" @click="router.push('/credentials/new')">新建凭据</el-button>
      </div>
      <div class="deco-circles">
        <div class="deco c1"></div>
        <div class="deco c2"></div>
      </div>
    </div>

    <div class="table-wrap">
      <el-table v-loading="loading" :data="credentials" style="width: 100%">
        <el-table-column label="名称" prop="name" min-width="180" />
        <el-table-column label="用户名" prop="username" min-width="160" />
        <el-table-column v-if="isAdmin" label="所属用户" width="110">
          <template #default="{ row }">
            <span class="owner-tag">{{ row.owner_username || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="created_at" width="180" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button link type="primary" @click="router.push(`/credentials/${row.id}/edit`)">
              编辑
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!loading && credentials.length === 0" class="empty-state">
        <div class="empty-icon">🔑</div>
        <p>暂无凭据</p>
        <el-button type="primary" size="small" @click="router.push('/credentials/new')">新建第一个凭据</el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { listCredentials } from '../../api/credentials'

const router = useRouter()

const credentials = ref([])
const loading = ref(false)

const isAdmin = computed(() => {
  const role = localStorage.getItem('role') || ''
  return role === 'admin' || role === 'super_admin'
})

onMounted(async () => {
  loading.value = true
  try {
    const res = await listCredentials()
    credentials.value = res.data
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.page-wrap { padding: 0; }

.page-hero {
  position: relative;
  background: linear-gradient(135deg, #dc2626, #db2777, #7c3aed);
  padding: 24px 36px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  overflow: hidden;
}
.hero-title { font-size: 22px; font-weight: 700; color: #fff; margin: 0 0 4px; }
.hero-sub { font-size: 13px; color: rgba(255,255,255,0.75); margin: 0; }
.hero-actions { position: relative; z-index: 2; }
.hero-actions :deep(.el-button--primary) { background: rgba(255,255,255,0.2) !important; border-color: rgba(255,255,255,0.3) !important; color: #fff !important; }
.hero-actions :deep(.el-button--primary:hover) { background: rgba(255,255,255,0.3) !important; }
.deco-circles { position: absolute; right: 0; top: 0; bottom: 0; width: 200px; pointer-events: none; }
.deco { position: absolute; border-radius: 50%; background: rgba(255,255,255,0.08); }
.c1 { width: 180px; height: 180px; right: -40px; top: -60px; }
.c2 { width: 100px; height: 100px; right: 60px; bottom: -30px; }

.table-wrap { background: #fff; border: 1px solid #e8edf4; border-radius: 10px; overflow: hidden; margin: 20px 36px; }
.owner-tag { font-size: 12.5px; color: #6366f1; font-weight: 500; }

.empty-state { padding: 48px; text-align: center; }
.empty-icon  { font-size: 36px; margin-bottom: 10px; }
.empty-state p { margin: 0 0 12px; font-size: 14px; color: #94a3b8; }
</style>
