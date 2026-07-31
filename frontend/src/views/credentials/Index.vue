<template>
  <div class="page-wrap credential-page">
    <div class="page-hero">
      <div class="bubble bubble-a"></div>
      <div class="bubble bubble-b"></div>
      <div class="bubble bubble-c"></div>
      <div class="hero-content">
        <div class="eyebrow">Repository Access</div>
        <h1 class="hero-title">仓库凭据</h1>
        <p class="hero-sub">管理访问私有 Git 仓库所需的认证凭据，供 Review 和巡检流程复用</p>
      </div>
      <div class="hero-actions">
        <el-button type="primary" @click="router.push('/credentials/new')">
          <el-icon><Plus /></el-icon> 新建凭据
        </el-button>
      </div>
    </div>

    <div class="credential-shell" v-loading="loading">
      <div v-if="credentials.length" class="summary-strip">
        <div class="summary-item">
          <span>凭据总数</span>
          <strong>{{ credentials.length }}</strong>
        </div>
        <div class="summary-item">
          <span>当前视图</span>
          <strong>{{ isAdmin ? '全部' : '我的' }}</strong>
        </div>
      </div>

      <div v-if="credentials.length" class="credential-grid">
        <article v-for="row in credentials" :key="row.id" class="credential-card" @click="router.push(`/credentials/${row.id}/edit`)" >
          <div class="credential-glow"></div>
          <div class="credential-top">
            <div class="credential-icon"><el-icon><Key /></el-icon></div>
            <span v-if="isAdmin" class="owner-pill">{{ row.owner_username || '—' }}</span>
          </div>
          <div class="credential-main">
            <h3>{{ row.name }}</h3>
            <div class="credential-user">{{ row.username || '未填写用户名' }}</div>
          </div>
          <div class="credential-meta">
            <span>创建时间</span>
            <strong>{{ row.created_at }}</strong>
          </div>
          <div class="credential-actions" @click.stop>
            <el-button size="small" type="primary" plain @click="router.push(`/credentials/${row.id}/edit`)">
              <el-icon><Edit /></el-icon> 编辑
            </el-button>
          </div>
        </article>
      </div>

      <div v-if="!loading && credentials.length === 0" class="empty-state">
        <div class="empty-icon"><el-icon><Key /></el-icon></div>
        <h3>暂无仓库凭据</h3>
        <p>添加 SSH/账号凭据后，可在项目和巡检配置中复用。</p>
        <el-button type="primary" @click="router.push('/credentials/new')">新建第一个凭据</el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Key, Plus, Edit } from '@element-plus/icons-vue'
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
.credential-page { min-height: 100vh; background: radial-gradient(circle at 18% 10%, rgba(219,39,119,.10), transparent 28%), radial-gradient(circle at 86% 18%, rgba(124,58,237,.10), transparent 28%), #f5f7fb; padding: 0; }
.page-hero { position: relative; background: linear-gradient(135deg, #991b1b, #db2777 52%, #7c3aed); padding: 30px 36px 28px; display: flex; align-items: center; justify-content: space-between; overflow: hidden; }
.hero-content, .hero-actions { position: relative; z-index: 2; }
.eyebrow { color: rgba(255,255,255,.62); font-size: 12px; font-weight: 750; letter-spacing: .14em; text-transform: uppercase; margin-bottom: 6px; }
.hero-title { font-size: 24px; font-weight: 760; color: #fff; margin: 0 0 5px; letter-spacing: -.45px; }
.hero-sub { font-size: 13px; color: rgba(255,255,255,.76); margin: 0; }
.hero-actions :deep(.el-button--primary) { background: rgba(255,255,255,.18) !important; border-color: rgba(255,255,255,.34) !important; color: #fff !important; box-shadow: 0 10px 28px rgba(15,23,42,.18); }
.bubble { position: absolute; border-radius: 999px; background: rgba(255,255,255,.11); pointer-events: none; }
.bubble-a { width: 230px; height: 230px; right: -56px; top: -82px; }
.bubble-b { width: 126px; height: 126px; right: 190px; bottom: -60px; background: rgba(255,255,255,.08); }
.bubble-c { width: 58px; height: 58px; left: 44%; top: 18px; background: rgba(255,255,255,.10); }
.credential-shell { width: 100%; padding: 24px 36px 38px; }
.summary-strip { display: grid; grid-template-columns: repeat(2, minmax(220px, 1fr)); gap: 14px; margin-bottom: 18px; }
.summary-item { background: #fff; border: 1px solid #e2e8f0; border-radius: 18px; padding: 14px 18px; display: flex; justify-content: space-between; align-items: center; box-shadow: var(--sh-card); }
.summary-item span { color: #64748b; font-size: 13px; }
.summary-item strong { color: #0f172a; font-size: 22px; }
.credential-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 18px; }
.credential-card { position: relative; min-height: 190px; background: #fff; border: 1px solid #e2e8f0; border-radius: 22px; padding: 20px; overflow: hidden; cursor: pointer; box-shadow: var(--sh-card); transition: transform .18s ease, box-shadow .18s ease, border-color .18s ease; }
.credential-card:hover { transform: translateY(-3px); border-color: #db2777; box-shadow: 0 22px 48px rgba(15,23,42,.10); }
.credential-glow { position: absolute; right: -34px; top: -40px; width: 128px; height: 128px; border-radius: 50%; background: #db2777; opacity: .12; }
.credential-top { position: relative; display: flex; justify-content: space-between; align-items: center; }
.credential-icon { width: 46px; height: 46px; border-radius: 15px; background: #fdf2f8; color: #db2777; display: flex; align-items: center; justify-content: center; font-size: 22px; }
.owner-pill { font-size: 11.5px; font-weight: 750; padding: 4px 10px; border-radius: 999px; background: #f5f3ff; color: #7c3aed; }
.credential-main { position: relative; margin-top: 22px; }
.credential-main h3 { margin: 0 0 8px; font-size: 17px; color: #0f172a; letter-spacing: -.25px; }
.credential-user { font-family: 'SFMono-Regular', Menlo, monospace; color: #475569; background: #f8fafc; border: 1px solid #eef2f7; border-radius: 10px; padding: 8px 10px; font-size: 12.5px; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.credential-meta { margin-top: 16px; display: flex; justify-content: space-between; gap: 12px; color: #94a3b8; font-size: 12px; }
.credential-meta strong { color: #64748b; font-weight: 500; }
.credential-actions { margin-top: 16px; display: flex; justify-content: flex-end; }
.credential-actions :deep(.el-button) { border-color: #db2777 !important; color: #db2777 !important; background: #fff !important; font-weight: 650; }
.credential-actions :deep(.el-button:hover) { background: #fdf2f8 !important; }
.empty-state { background: #fff; border: 1px solid #e2e8f0; border-radius: 22px; padding: 54px; text-align: center; box-shadow: var(--sh-card); }
.empty-icon { width: 58px; height: 58px; margin: 0 auto 14px; border-radius: 18px; background: #fdf2f8; color: #db2777; display: flex; align-items: center; justify-content: center; font-size: 26px; }
.empty-state h3 { margin: 0 0 6px; color: #0f172a; }
.empty-state p { margin: 0 0 18px; color: #64748b; font-size: 13px; }
</style>
