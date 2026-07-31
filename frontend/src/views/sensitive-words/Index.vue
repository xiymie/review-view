<template>
  <div class="page-wrap sensitive-page">
    <div class="page-hero">
      <div class="bubble bubble-a"></div>
      <div class="bubble bubble-b"></div>
      <div class="bubble bubble-c"></div>
      <div class="hero-content">
        <div class="eyebrow">Safety Rules</div>
        <h1 class="hero-title">敏感词管理</h1>
        <p class="hero-sub">替换类用于发送给 LLM 前脱敏；检测类用于扫描代码并写入审核报告</p>
      </div>
      <div class="hero-actions">
        <el-button type="primary" :icon="Plus" @click="openDialog()">新增规则</el-button>
      </div>
    </div>

    <div class="rule-shell" v-loading="loading">
      <div class="summary-strip">
        <div class="summary-item replace">
          <span>替换规则</span>
          <strong>{{ replaceCount }}</strong>
        </div>
        <div class="summary-item detect">
          <span>检测规则</span>
          <strong>{{ detectCount }}</strong>
        </div>
        <div class="summary-item total">
          <span>规则总数</span>
          <strong>{{ words.length }}</strong>
        </div>
      </div>

      <div v-if="words.length" class="rule-grid">
        <article v-for="row in words" :key="row.ID" class="rule-card" :class="row.Type === 'detect' ? 'detect-card' : 'replace-card'">
          <div class="rule-icon">
            <el-icon><component :is="row.Type === 'detect' ? Search : Lock" /></el-icon>
          </div>
          <div class="rule-main">
            <div class="rule-top">
              <span class="rule-type">{{ row.Type === 'detect' ? '检测' : '替换' }}</span>
              <span class="rule-time">{{ formatTime(row.CreatedAt) }}</span>
            </div>
            <h3>{{ row.Original }}</h3>
            <p v-if="row.Type === 'detect'">命中后会追加到敏感词检测报告。</p>
            <p v-else>发送前替换为 <code>{{ row.Replacement }}</code>，响应后再还原。</p>
          </div>
          <div class="rule-actions">
            <el-button size="small" text @click="openDialog(row)">编辑</el-button>
            <el-popconfirm title="确认删除？" @confirm="remove(row.ID)">
              <template #reference>
                <el-button size="small" text type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </div>
        </article>
      </div>

      <div v-if="!loading && words.length === 0" class="empty-state">
        <div class="empty-icon"><el-icon><Lock /></el-icon></div>
        <h3>暂无敏感词规则</h3>
        <p>添加替换或检测规则，保护 Prompt 和代码审查报告。</p>
        <el-button type="primary" @click="openDialog()">新增第一条规则</el-button>
      </div>
    </div>

    <el-dialog v-model="dialogVisible" :title="form.ID ? '编辑敏感词' : '新增敏感词'" width="520px" class="rule-dialog" @closed="resetForm" align-center destroy-on-close>
      <el-form :model="form" :rules="rules" ref="formRef" label-position="top">
        <el-form-item label="类型" prop="Type">
          <el-radio-group v-model="form.Type">
            <el-radio-button label="replace">替换</el-radio-button>
            <el-radio-button label="detect">检测</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="form.Type === 'detect' ? '检测词' : '原始词'" prop="Original">
          <el-input v-model="form.Original" :placeholder="form.Type === 'detect' ? '在代码中检索的词' : '发送前被替换的词'" />
        </el-form-item>
        <el-form-item v-if="form.Type !== 'detect'" label="替换词" prop="Replacement">
          <el-input v-model="form.Replacement" placeholder="实际发给上游的词" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { Plus, Lock, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import {
  listSensitiveWords, createSensitiveWord,
  updateSensitiveWord, deleteSensitiveWord,
} from '../../api/sensitive-words'

const words = ref([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const formRef = ref(null)
const form = ref({ ID: null, Type: 'replace', Original: '', Replacement: '' })

const replaceCount = computed(() => words.value.filter(w => w.Type !== 'detect').length)
const detectCount = computed(() => words.value.filter(w => w.Type === 'detect').length)

const rules = {
  Original: [{ required: true, message: '请输入原始词', trigger: 'blur' }],
  Replacement: [{
    validator: (rule, value, callback) => {
      if (form.value.Type !== 'detect' && !value) callback(new Error('请输入替换词'))
      else callback()
    },
    trigger: 'blur',
  }],
}

function formatTime(v) {
  if (!v) return '—'
  return new Date(v).toLocaleDateString('zh-CN').replaceAll('/', '-')
}

async function load() {
  loading.value = true
  try {
    const res = await listSensitiveWords()
    words.value = res.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.response?.data?.message || '加载敏感词失败')
  } finally {
    loading.value = false
  }
}

function openDialog(row = null) {
  form.value = row
    ? { ID: row.ID, Type: row.Type || 'replace', Original: row.Original, Replacement: row.Replacement }
    : { ID: null, Type: 'replace', Original: '', Replacement: '' }
  dialogVisible.value = true
}

function resetForm() {
  formRef.value?.resetFields()
}

async function save() {
  try { await formRef.value.validate() } catch { return }
  saving.value = true
  try {
    if (form.value.ID) {
      await updateSensitiveWord(form.value.ID, {
        type: form.value.Type,
        original: form.value.Original,
        replacement: form.value.Type === 'detect' ? '' : form.value.Replacement,
      })
    } else {
      await createSensitiveWord({
        type: form.value.Type,
        original: form.value.Original,
        replacement: form.value.Type === 'detect' ? '' : form.value.Replacement,
      })
    }
    dialogVisible.value = false
    ElMessage.success('保存成功')
    load()
  } catch (err) {
    ElMessage.error('保存失败: ' + (err.response?.data?.error || err.message))
  } finally {
    saving.value = false
  }
}

async function remove(id) {
  try {
    await deleteSensitiveWord(id)
    ElMessage.success('删除成功')
    await load()
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.response?.data?.message || '删除失败')
  }
}

onMounted(load)
</script>

<style scoped>
.sensitive-page { min-height: 100vh; background: radial-gradient(circle at 16% 8%, rgba(217,119,6,0.10), transparent 26%), radial-gradient(circle at 86% 18%, rgba(220,38,38,0.08), transparent 28%), #f5f7fb; padding: 0; }
.page-hero { position: relative; background: linear-gradient(135deg, #92400e, #dc2626 55%, #7c3aed); padding: 30px 36px 28px; display: flex; align-items: center; justify-content: space-between; overflow: hidden; }
.hero-content, .hero-actions { position: relative; z-index: 2; }
.eyebrow { color: rgba(255,255,255,.62); font-size: 12px; font-weight: 750; letter-spacing: .14em; text-transform: uppercase; margin-bottom: 6px; }
.hero-title { font-size: 24px; font-weight: 760; color: #fff; margin: 0 0 5px; letter-spacing: -.45px; }
.hero-sub { font-size: 13px; color: rgba(255,255,255,.76); margin: 0; }
.hero-actions :deep(.el-button--primary) { background: rgba(255,255,255,.18) !important; border-color: rgba(255,255,255,.34) !important; color: #fff !important; box-shadow: 0 10px 28px rgba(15,23,42,.18); }
.bubble { position: absolute; border-radius: 999px; background: rgba(255,255,255,.11); pointer-events: none; }
.bubble-a { width: 230px; height: 230px; right: -56px; top: -82px; }
.bubble-b { width: 126px; height: 126px; right: 190px; bottom: -60px; background: rgba(255,255,255,.08); }
.bubble-c { width: 58px; height: 58px; left: 44%; top: 18px; background: rgba(255,255,255,.10); }
.rule-shell { width: 100%; padding: 24px 36px 38px; }
.summary-strip { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; margin-bottom: 18px; }
.summary-item { background: #fff; border: 1px solid #e2e8f0; border-radius: 18px; padding: 14px 18px; display: flex; justify-content: space-between; align-items: center; box-shadow: var(--sh-card); }
.summary-item span { color: #64748b; font-size: 13px; }
.summary-item strong { color: #0f172a; font-size: 22px; }
.summary-item.replace strong { color: #2563eb; }
.summary-item.detect strong { color: #dc2626; }
.rule-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 18px; }
.rule-card { position: relative; background: #fff; border: 1px solid #e2e8f0; border-radius: 22px; padding: 18px; box-shadow: var(--sh-card); display: grid; grid-template-columns: 46px 1fr; gap: 14px; transition: transform .18s ease, box-shadow .18s ease; }
.rule-card:hover { transform: translateY(-3px); box-shadow: 0 22px 48px rgba(15,23,42,.10); }
.rule-icon { width: 46px; height: 46px; border-radius: 15px; display: flex; align-items: center; justify-content: center; font-size: 21px; }
.replace-card .rule-icon { color: #2563eb; background: #eff6ff; }
.detect-card .rule-icon { color: #dc2626; background: #fef2f2; }
.rule-main { min-width: 0; }
.rule-top { display: flex; justify-content: space-between; gap: 12px; align-items: center; }
.rule-type { font-size: 11.5px; font-weight: 750; padding: 3px 9px; border-radius: 999px; background: #f8fafc; color: #64748b; }
.detect-card .rule-type { background: #fef2f2; color: #dc2626; }
.replace-card .rule-type { background: #eff6ff; color: #2563eb; }
.rule-time { color: #94a3b8; font-size: 12px; }
.rule-main h3 { margin: 12px 0 8px; font-size: 16px; color: #0f172a; word-break: break-word; }
.rule-main p { margin: 0; color: #64748b; font-size: 12.5px; line-height: 1.55; }
.rule-main code { background: #f1f5f9; border-radius: 6px; padding: 1px 5px; color: #475569; }
.rule-actions { grid-column: 1 / -1; display: flex; justify-content: flex-end; gap: 6px; border-top: 1px solid #f1f5f9; padding-top: 12px; }
.empty-state { background: #fff; border: 1px solid #e2e8f0; border-radius: 22px; padding: 54px; text-align: center; box-shadow: var(--sh-card); }
.empty-icon { width: 58px; height: 58px; margin: 0 auto 14px; border-radius: 18px; background: #fef2f2; color: #dc2626; display: flex; align-items: center; justify-content: center; font-size: 26px; }
.empty-state h3 { margin: 0 0 6px; color: #0f172a; }
.empty-state p { margin: 0 0 18px; color: #64748b; font-size: 13px; }
:deep(.rule-dialog .el-dialog) { border-radius: 22px !important; overflow: hidden; }
</style>
