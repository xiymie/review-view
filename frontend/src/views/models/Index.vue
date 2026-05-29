<template>
  <div class="page-wrap">
    <div class="page-hero">
      <div class="hero-content">
        <h1 class="hero-title">模型配置</h1>
        <p class="hero-sub">管理用于代码审查的 AI 模型配置</p>
      </div>
      <div class="hero-actions">
        <el-button type="primary" @click="router.push('/models/new')">新建配置</el-button>
      </div>
      <div class="deco-circles">
        <div class="deco c1"></div>
        <div class="deco c2"></div>
      </div>
    </div>

    <div class="table-wrap">
      <el-table v-loading="loading" :data="models" style="width:100%">
        <el-table-column label="名称" min-width="160">
          <template #default="{ row }">
            <span class="name-link" @click="router.push(`/models/${row.id}/edit`)">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="平台类型" width="140">
          <template #default="{ row }">
            <span class="type-tag">{{ row.type }}</span>
          </template>
        </el-table-column>
        <el-table-column label="模型名称" prop="model" min-width="180" />
        <el-table-column label="Thinking" width="110">
          <template #default="{ row }">
            <span class="status-pill" :class="row.enable_thinking ? 'pill-on' : 'pill-off'">
              {{ row.enable_thinking ? '已开启' : '未开启' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="created_at" width="160" />
        <el-table-column label="操作" width="80">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="router.push(`/models/${row.id}/edit`)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!loading && models.length === 0" class="empty-state">
        <div class="empty-icon">🤖</div>
        <p>暂无模型配置</p>
        <el-button type="primary" size="small" @click="router.push('/models/new')">新建第一个配置</el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { listModels } from '../../api/models'

const router = useRouter()
const models = ref([])
const loading = ref(false)

onMounted(async () => {
  loading.value = true
  try {
    const res = await listModels()
    models.value = res.data
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
  background: linear-gradient(135deg, #0f766e, #0891b2, #2563eb);
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

.table-wrap {
  background: #fff;
  border: 1px solid #e8edf4;
  border-radius: 10px;
  overflow: hidden;
  margin: 20px 36px;
}

.name-link {
  color: #2563eb;
  font-weight: 500;
  font-size: 13.5px;
  cursor: pointer;
}
.name-link:hover { text-decoration: underline; }

.type-tag {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 5px;
  font-size: 12px;
  font-family: monospace;
  background: #f1f5f9;
  color: #475569;
}

.status-pill {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
}
.pill-on  { background: #ecfdf5; color: #059669; }
.pill-off { background: #f1f5f9; color: #94a3b8; }

.empty-state { padding: 48px; text-align: center; }
.empty-icon  { font-size: 36px; margin-bottom: 10px; }
.empty-state p { margin: 0 0 12px; font-size: 14px; color: #94a3b8; }
</style>
