<template>
  <div class="page-wrap model-page">
    <div class="page-hero">
      <div class="bubble bubble-a"></div>
      <div class="bubble bubble-b"></div>
      <div class="bubble bubble-c"></div>
      <div class="hero-content">
        <div class="eyebrow">Model Runtime</div>
        <h1 class="hero-title">模型配置</h1>
        <p class="hero-sub">管理用于 Review Workflow / Skill / Agent 的 AI 模型入口</p>
      </div>
      <div class="hero-actions">
        <el-button type="primary" @click="router.push('/models/new')">
          <el-icon><Plus /></el-icon> 新建配置
        </el-button>
      </div>
    </div>

    <div class="model-shell" v-loading="loading">
      <div v-if="models.length" class="summary-strip">
        <div class="summary-item">
          <span>模型总数</span>
          <strong>{{ models.length }}</strong>
        </div>
        <div class="summary-item">
          <span>平台类型</span>
          <strong>{{ providerCount }}</strong>
        </div>
        <div class="summary-item">
          <span>Thinking</span>
          <strong>{{ thinkingCount }}</strong>
        </div>
      </div>

      <div v-if="models.length" class="model-grid">
        <article v-for="row in models" :key="row.id" class="model-card" @click="router.push(`/models/${row.id}/edit`)" :style="providerStyle(row.type)">
          <div class="model-glow"></div>
          <div class="model-top">
            <div class="model-icon">
              <el-icon><Cpu /></el-icon>
            </div>
            <div class="model-status" :class="row.enable_thinking ? 'on' : 'off'">
              {{ row.enable_thinking ? 'Thinking' : 'Standard' }}
            </div>
          </div>
          <div class="model-main">
            <h3>{{ row.name }}</h3>
            <div class="model-code">{{ row.model || '未填写模型名' }}</div>
          </div>
          <div class="model-meta">
            <span class="type-pill">{{ row.type }}</span>
            <span>{{ row.created_at }}</span>
          </div>
          <div class="model-actions" @click.stop>
            <el-button size="small" type="primary" plain @click="router.push(`/models/${row.id}/edit`)">
              <el-icon><Edit /></el-icon> 编辑
            </el-button>
          </div>
        </article>
      </div>

      <div v-if="!loading && models.length === 0" class="empty-state">
        <div class="empty-icon"><el-icon><Cpu /></el-icon></div>
        <h3>暂无模型配置</h3>
        <p>先创建一个模型配置，再把它绑定到项目或巡检配置。</p>
        <el-button type="primary" @click="router.push('/models/new')">新建第一个配置</el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Cpu, Plus, Edit } from '@element-plus/icons-vue'
import { listModels } from '../../api/models'

const router = useRouter()
const models = ref([])
const loading = ref(false)

const providerCount = computed(() => new Set(models.value.map(m => m.type).filter(Boolean)).size)
const thinkingCount = computed(() => models.value.filter(m => m.enable_thinking).length)

function providerStyle(type) {
  const map = {
    openai:     { accent: '#10b981', bg: '#ecfdf5' },
    anthropic:  { accent: '#7c3aed', bg: '#f5f3ff' },
    claude_cli: { accent: '#2563eb', bg: '#eff6ff' },
    ollama:     { accent: '#f59e0b', bg: '#fffbeb' },
    gemini:     { accent: '#0891b2', bg: '#ecfeff' },
    deepseek:   { accent: '#dc2626', bg: '#fef2f2' },
    mistral:    { accent: '#6366f1', bg: '#eef2ff' },
  }
  const item = map[type] || { accent: '#2563eb', bg: '#eff6ff' }
  return { '--accent': item.accent, '--accent-bg': item.bg }
}

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
.model-page { min-height: 100vh; background: radial-gradient(circle at 18% 10%, rgba(14,165,233,0.10), transparent 28%), radial-gradient(circle at 86% 18%, rgba(124,58,237,0.10), transparent 28%), #f5f7fb; padding: 0; }
.page-hero { position: relative; background: linear-gradient(135deg, #0f766e, #0891b2 52%, #2563eb); padding: 30px 36px 28px; display: flex; align-items: center; justify-content: space-between; overflow: hidden; }
.hero-content, .hero-actions { position: relative; z-index: 2; }
.eyebrow { color: rgba(255,255,255,.62); font-size: 12px; font-weight: 750; letter-spacing: .14em; text-transform: uppercase; margin-bottom: 6px; }
.hero-title { font-size: 24px; font-weight: 760; color: #fff; margin: 0 0 5px; letter-spacing: -.45px; }
.hero-sub { font-size: 13px; color: rgba(255,255,255,.76); margin: 0; }
.hero-actions :deep(.el-button--primary) { background: rgba(255,255,255,.18) !important; border-color: rgba(255,255,255,.34) !important; color: #fff !important; box-shadow: 0 10px 28px rgba(15,23,42,.18); }
.bubble { position: absolute; border-radius: 999px; background: rgba(255,255,255,.11); pointer-events: none; }
.bubble-a { width: 230px; height: 230px; right: -56px; top: -82px; }
.bubble-b { width: 126px; height: 126px; right: 190px; bottom: -60px; background: rgba(255,255,255,.08); }
.bubble-c { width: 58px; height: 58px; left: 44%; top: 18px; background: rgba(255,255,255,.10); }
.model-shell { width: 100%; padding: 24px 36px 38px; }
.summary-strip { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; margin-bottom: 18px; }
.summary-item { background: rgba(255,255,255,.9); border: 1px solid #e2e8f0; border-radius: 18px; padding: 14px 18px; display: flex; justify-content: space-between; align-items: center; box-shadow: var(--sh-card); }
.summary-item span { color: #64748b; font-size: 13px; }
.summary-item strong { font-size: 22px; color: #0f172a; font-variant-numeric: tabular-nums; }
.model-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 18px; }
.model-card { position: relative; min-height: 210px; background: #fff; border: 1px solid #e2e8f0; border-radius: 22px; padding: 20px; overflow: hidden; cursor: pointer; box-shadow: var(--sh-card); transition: transform .18s ease, box-shadow .18s ease, border-color .18s ease; }
.model-card:hover { transform: translateY(-3px); border-color: var(--accent); box-shadow: 0 22px 48px rgba(15,23,42,.10); }
.model-glow { position: absolute; right: -34px; top: -40px; width: 128px; height: 128px; border-radius: 50%; background: var(--accent); opacity: .13; }
.model-top { position: relative; display: flex; align-items: center; justify-content: space-between; }
.model-icon { width: 46px; height: 46px; border-radius: 15px; background: var(--accent-bg); color: var(--accent); display: flex; align-items: center; justify-content: center; font-size: 22px; }
.model-status { font-size: 11.5px; font-weight: 750; padding: 4px 10px; border-radius: 999px; }
.model-status.on { background: #ecfdf5; color: #059669; }
.model-status.off { background: #f8fafc; color: #94a3b8; }
.model-main { position: relative; margin-top: 24px; }
.model-main h3 { margin: 0 0 8px; font-size: 17px; color: #0f172a; letter-spacing: -.25px; }
.model-code { font-family: 'SFMono-Regular', Menlo, monospace; font-size: 12.5px; color: #475569; background: #f8fafc; border: 1px solid #eef2f7; border-radius: 10px; padding: 8px 10px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.model-meta { margin-top: 18px; display: flex; align-items: center; justify-content: space-between; gap: 12px; color: #94a3b8; font-size: 12px; }
.type-pill { color: var(--accent); background: var(--accent-bg); border-radius: 999px; padding: 3px 9px; font-family: 'SFMono-Regular', Menlo, monospace; }
.model-actions { margin-top: 16px; display: flex; justify-content: flex-end; opacity: 1; }
.model-actions :deep(.el-button) { border-color: var(--accent) !important; color: var(--accent) !important; background: #fff !important; font-weight: 650; }
.model-actions :deep(.el-button:hover) { background: var(--accent-bg) !important; }
.empty-state { background: #fff; border: 1px solid #e2e8f0; border-radius: 22px; padding: 54px; text-align: center; box-shadow: var(--sh-card); }
.empty-icon { width: 58px; height: 58px; margin: 0 auto 14px; border-radius: 18px; background: #eff6ff; color: #2563eb; display: flex; align-items: center; justify-content: center; font-size: 26px; }
.empty-state h3 { margin: 0 0 6px; color: #0f172a; }
.empty-state p { margin: 0 0 18px; color: #64748b; font-size: 13px; }
</style>
