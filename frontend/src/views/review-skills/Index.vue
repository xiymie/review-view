<template>
  <div class="page-wrap skill-page">
    <div class="page-hero">
      <div class="bubble bubble-a"></div>
      <div class="bubble bubble-b"></div>
      <div class="bubble bubble-c"></div>
      <div class="hero-content">
        <div class="eyebrow">Review Intelligence</div>
        <h1 class="hero-title">Review Skill 管理</h1>
        <p class="hero-sub">配置 Eino Review Workflow 的审查技能，启用后会通过 load_skills 节点注入 Review Prompt</p>
      </div>
      <div class="hero-actions">
        <el-button type="primary" :icon="Plus" @click="openDialog()">新增 Skill</el-button>
      </div>
    </div>

    <div class="skill-shell" v-loading="loading">
      <div class="summary-row">
        <div class="summary-card">
          <span class="summary-label">总 Skill</span>
          <strong>{{ skills.length }}</strong>
        </div>
        <div class="summary-card enabled">
          <span class="summary-label">已启用</span>
          <strong>{{ enabledCount }}</strong>
        </div>
        <div class="summary-card built-in">
          <span class="summary-label">内置</span>
          <strong>{{ builtInCount }}</strong>
        </div>
      </div>

      <div v-if="skills.length" class="skill-grid">
        <article v-for="row in skills" :key="row.id" class="skill-card" :class="{ disabled: !row.enabled, builtin: row.built_in }">
          <div class="skill-glow"></div>
          <div class="skill-top">
            <div class="skill-icon">
              <el-icon><MagicStick /></el-icon>
            </div>
            <div class="skill-tags">
              <el-tag v-if="row.built_in" size="small" type="info">内置</el-tag>
              <span :class="['state-pill', row.enabled ? 'on' : 'off']">{{ row.enabled ? '启用' : '停用' }}</span>
            </div>
          </div>

          <div class="skill-main">
            <h3>{{ row.name }}</h3>
            <p>{{ row.description || '暂无说明' }}</p>

            <!-- Structural component chips -->
            <div class="struct-chips">
              <span v-if="hasContent(row.agent_xml)" class="chip chip-agent" title="Agent XML">
                <el-icon><Connection /></el-icon> Agent
              </span>
              <span v-if="hasContent(row.skill_registry_xml) || hasContent(row.tool_registry_xml)" class="chip chip-registry" title="Registry XML">
                <el-icon><Grid /></el-icon> Registry
              </span>
              <span v-if="hasContent(row.policy_md)" class="chip chip-policy" title="Policy">
                <el-icon><DocumentChecked /></el-icon> Policy
              </span>
              <span v-if="hasContent(row.workflow_md)" class="chip chip-workflow" title="Workflow">
                <el-icon><Share /></el-icon> Workflow
              </span>
              <span v-if="hasContent(row.context_schema_json)" class="chip chip-context" title="Context Schema">
                <el-icon><Box /></el-icon> Context
              </span>
              <span v-if="hasContent(row.memory_schema_json)" class="chip chip-memory" title="Memory Schema">
                <el-icon><Collection /></el-icon> Memory
              </span>
            </div>

            <!-- Active Skill Prompt -->
            <div class="prompt-section">
              <div class="prompt-label">
                <el-icon><EditPen /></el-icon>
                Active Skill Prompt
              </div>
              <div class="prompt-preview">{{ row.prompt || '（无 Prompt）' }}</div>
            </div>
          </div>

          <div class="skill-meta">
            <span>排序 {{ row.sort_order }}</span>
            <span>{{ row.updated_at }}</span>
          </div>

          <div class="skill-actions">
            <el-switch
              v-model="row.enabled"
              :loading="togglingId === row.id"
              @change="val => handleToggle(row, val)"
            />
            <div class="action-buttons">
              <el-button size="small" type="primary" plain @click="openDialog(row)">编辑</el-button>
              <el-popconfirm v-if="!row.built_in" title="确认删除该 Skill？" @confirm="remove(row.id)">
                <template #reference>
                  <el-button size="small" type="danger" plain>删除</el-button>
                </template>
              </el-popconfirm>
              <el-tooltip v-else content="内置 Skill 不能删除" placement="top">
                <el-button size="small" disabled>删除</el-button>
              </el-tooltip>
            </div>
          </div>
        </article>
      </div>

      <div v-if="!loading && skills.length === 0" class="empty-state">
        <div class="empty-icon"><el-icon><MagicStick /></el-icon></div>
        <h3>暂无 Review Skill</h3>
        <p>添加审查技能，让 Eino Review Workflow 拥有更明确的专项审查维度。</p>
        <el-button type="primary" @click="openDialog()">新增第一个 Skill</el-button>
      </div>
    </div>

    <!-- ======= Dialog ======= -->
    <el-dialog
      v-model="dialogVisible"
      :title="form.id ? '编辑 Review Skill' : '新增 Review Skill'"
      width="820px"
      class="skill-dialog"
      :close-on-click-modal="false"
      destroy-on-close
      align-center
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-position="top" class="skill-form">

        <!-- ── Basic ── -->
        <div class="form-section-title">基本信息</div>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="例如：performance-review" :disabled="form.built_in" />
        </el-form-item>
        <el-form-item label="说明" prop="description">
          <el-input v-model="form.description" placeholder="简要说明这个 Skill 的审查重点" />
        </el-form-item>
        <div class="form-inline">
          <el-form-item label="排序" prop="sort_order">
            <el-input-number v-model="form.sort_order" :min="0" :step="10" />
          </el-form-item>
          <el-form-item label="启用">
            <el-switch v-model="form.enabled" active-text="启用" inactive-text="停用" />
          </el-form-item>
        </div>

        <!-- ── Active Skill Prompt ── -->
        <div class="form-section-title prompt-section-title">
          <el-icon><EditPen /></el-icon>
          Active Skill Prompt
          <span class="section-badge">注入 Prompt</span>
        </div>
        <el-form-item prop="prompt">
          <el-input
            v-model="form.prompt"
            type="textarea"
            :rows="8"
            placeholder="输入会被注入 Review Prompt 的审查要求"
          />
        </el-form-item>

        <!-- ── Structural Sections (collapsible) ── -->
        <div class="form-section-title collapsible-group-title">官方结构字段（可选）</div>

        <div v-for="sec in structSections" :key="sec.key" class="collapsible-section">
          <div class="collapsible-header" @click="toggleSection(sec.key)">
            <span class="cs-left">
              <span :class="['cs-dot', sec.color]"></span>
              <span class="cs-label">{{ sec.label }}</span>
              <span class="cs-hint">{{ sec.hint }}</span>
            </span>
            <span class="cs-right">
              <span v-if="hasContent(form[sec.key])" class="cs-filled">已填写</span>
              <el-icon class="cs-arrow" :class="{ open: openSections[sec.key] }"><ArrowDown /></el-icon>
            </span>
          </div>
          <transition name="section-fade">
            <div v-show="openSections[sec.key]" class="collapsible-body">
              <el-input
                v-model="form[sec.key]"
                type="textarea"
                :rows="sec.rows || 6"
                :placeholder="sec.placeholder"
                class="mono-area"
              />
            </div>
          </transition>
        </div>

      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Plus, MagicStick, Connection, Grid, DocumentChecked,
  Share, Box, Collection, EditPen, ArrowDown,
} from '@element-plus/icons-vue'
import {
  listReviewSkills,
  createReviewSkill,
  updateReviewSkill,
  toggleReviewSkill,
  deleteReviewSkill,
} from '../../api/review-skills'

// ── struct section definitions ──
const structSections = [
  {
    key: 'agent_xml',
    label: 'Agent',
    hint: 'agent_xml',
    color: 'dot-agent',
    placeholder: '<agent>\n  <name>...</name>\n  <role>...</role>\n</agent>',
    rows: 6,
  },
  {
    key: 'skill_registry_xml',
    label: 'Skill Registry',
    hint: 'skill_registry_xml',
    color: 'dot-registry',
    placeholder: '<skill_registry>\n  <skill>...</skill>\n</skill_registry>',
    rows: 5,
  },
  {
    key: 'tool_registry_xml',
    label: 'Tool Registry',
    hint: 'tool_registry_xml',
    color: 'dot-registry',
    placeholder: '<tool_registry>\n  <tool>...</tool>\n</tool_registry>',
    rows: 5,
  },
  {
    key: 'policy_md',
    label: 'Policy',
    hint: 'policy_md',
    color: 'dot-policy',
    placeholder: '# Policy\n\n描述审查策略和规则...',
    rows: 6,
  },
  {
    key: 'workflow_md',
    label: 'Workflow',
    hint: 'workflow_md',
    color: 'dot-workflow',
    placeholder: '# Workflow\n\n描述审查工作流步骤...',
    rows: 6,
  },
  {
    key: 'context_schema_json',
    label: 'Context Schema',
    hint: 'context_schema_json',
    color: 'dot-context',
    placeholder: '{\n  "type": "object",\n  "properties": {}\n}',
    rows: 6,
  },
  {
    key: 'memory_schema_json',
    label: 'Memory Schema',
    hint: 'memory_schema_json',
    color: 'dot-memory',
    placeholder: '{\n  "type": "object",\n  "properties": {}\n}',
    rows: 6,
  },
  {
    key: 'metadata_json',
    label: 'Metadata',
    hint: 'metadata_json',
    color: 'dot-meta',
    placeholder: '{\n  "version": "1.0",\n  "tags": []\n}',
    rows: 4,
  },
]

const structKeys = structSections.map(s => s.key)

const openSections = reactive(Object.fromEntries(structKeys.map(k => [k, false])))

function toggleSection(key) {
  openSections[key] = !openSections[key]
}

function hasContent(val) {
  return typeof val === 'string' && val.trim().length > 0
}

// ── data ──
const skills = ref([])
const loading = ref(false)
const saving = ref(false)
const togglingId = ref(null)
const dialogVisible = ref(false)
const formRef = ref(null)

const defaultForm = () => ({
  id: null,
  name: '',
  description: '',
  prompt: '',
  enabled: true,
  built_in: false,
  sort_order: 100,
  agent_xml: '',
  skill_registry_xml: '',
  tool_registry_xml: '',
  policy_md: '',
  workflow_md: '',
  context_schema_json: '',
  memory_schema_json: '',
  metadata_json: '',
})

const form = ref(defaultForm())

const enabledCount = computed(() => skills.value.filter(s => s.enabled).length)
const builtInCount = computed(() => skills.value.filter(s => s.built_in).length)

const rules = {
  name: [{ required: true, message: '请输入 Skill 名称', trigger: 'blur' }],
  prompt: [{ required: true, message: '请输入 Skill Prompt', trigger: 'blur' }],
}

async function load() {
  loading.value = true
  try {
    const res = await listReviewSkills()
    skills.value = res.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载 Review Skill 失败')
  } finally {
    loading.value = false
  }
}

function openDialog(row = null) {
  // reset open sections
  structKeys.forEach(k => { openSections[k] = false })

  if (row) {
    form.value = {
      id: row.id,
      name: row.name,
      description: row.description || '',
      prompt: row.prompt || '',
      enabled: row.enabled,
      built_in: row.built_in,
      sort_order: row.sort_order || 0,
      agent_xml: row.agent_xml || '',
      skill_registry_xml: row.skill_registry_xml || '',
      tool_registry_xml: row.tool_registry_xml || '',
      policy_md: row.policy_md || '',
      workflow_md: row.workflow_md || '',
      context_schema_json: row.context_schema_json || '',
      memory_schema_json: row.memory_schema_json || '',
      metadata_json: row.metadata_json || '',
    }
    // auto-open sections that already have content
    structKeys.forEach(k => {
      if (hasContent(form.value[k])) openSections[k] = true
    })
  } else {
    form.value = defaultForm()
  }
  dialogVisible.value = true
}

async function save() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.value.name,
      description: form.value.description,
      prompt: form.value.prompt,
      enabled: form.value.enabled,
      sort_order: form.value.sort_order,
      agent_xml: form.value.agent_xml,
      skill_registry_xml: form.value.skill_registry_xml,
      tool_registry_xml: form.value.tool_registry_xml,
      policy_md: form.value.policy_md,
      workflow_md: form.value.workflow_md,
      context_schema_json: form.value.context_schema_json,
      memory_schema_json: form.value.memory_schema_json,
      metadata_json: form.value.metadata_json,
    }
    if (form.value.id) await updateReviewSkill(form.value.id, payload)
    else await createReviewSkill(payload)
    ElMessage.success('保存成功')
    dialogVisible.value = false
    await load()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleToggle(row, enabled) {
  togglingId.value = row.id
  try {
    await toggleReviewSkill(row.id, enabled)
    ElMessage.success(enabled ? '已启用' : '已停用')
  } catch (err) {
    row.enabled = !enabled
    ElMessage.error(err.response?.data?.message || '状态切换失败')
  } finally {
    togglingId.value = null
  }
}

async function remove(id) {
  try {
    await deleteReviewSkill(id)
    ElMessage.success('删除成功')
    await load()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '删除失败')
  }
}

onMounted(load)
</script>

<style scoped>
/* ── page shell ── */
.skill-page { min-height: 100vh; background: radial-gradient(circle at 18% 8%, rgba(37,99,235,.10), transparent 28%), radial-gradient(circle at 86% 18%, rgba(124,58,237,.10), transparent 28%), #f5f7fb; padding: 0; }
.page-hero { position: relative; background: linear-gradient(135deg, #0f766e, #2563eb 52%, #7c3aed); padding: 30px 36px 28px; display: flex; align-items: center; justify-content: space-between; overflow: hidden; }
.hero-content, .hero-actions { position: relative; z-index: 2; }
.eyebrow { color: rgba(255,255,255,.62); font-size: 12px; font-weight: 750; letter-spacing: .14em; text-transform: uppercase; margin-bottom: 6px; }
.hero-title { font-size: 24px; font-weight: 760; color: #fff; margin: 0 0 5px; letter-spacing: -.45px; }
.hero-sub { font-size: 13px; color: rgba(255,255,255,.78); margin: 0; }
.hero-actions :deep(.el-button--primary) { background: rgba(255,255,255,.18) !important; border-color: rgba(255,255,255,.34) !important; color: #fff !important; box-shadow: 0 10px 28px rgba(15,23,42,.18); }
.bubble { position: absolute; border-radius: 999px; background: rgba(255,255,255,.11); pointer-events: none; }
.bubble-a { width: 230px; height: 230px; right: -56px; top: -82px; }
.bubble-b { width: 126px; height: 126px; right: 190px; bottom: -60px; background: rgba(255,255,255,.08); }
.bubble-c { width: 58px; height: 58px; left: 44%; top: 18px; background: rgba(255,255,255,.10); }

/* ── shell / summary ── */
.skill-shell { width: 100%; padding: 24px 36px 38px; }
.summary-row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; margin-bottom: 18px; }
.summary-card { background: #fff; border: 1px solid #e2e8f0; border-radius: 18px; padding: 14px 18px; display: flex; justify-content: space-between; align-items: center; box-shadow: var(--sh-card); }
.summary-label { color: #64748b; font-size: 13px; }
.summary-card strong { color: #0f172a; font-size: 22px; }
.summary-card.enabled strong { color: #059669; }
.summary-card.built-in strong { color: #7c3aed; }

/* ── card grid ── */
.skill-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 18px; }
.skill-card { position: relative; background: #fff; border: 1px solid #e2e8f0; border-radius: 22px; padding: 20px; overflow: hidden; box-shadow: var(--sh-card); transition: transform .18s ease, box-shadow .18s ease, opacity .18s ease; }
.skill-card:hover { transform: translateY(-3px); box-shadow: 0 22px 48px rgba(15,23,42,.10); }
.skill-card.disabled { opacity: .68; }
.skill-glow { position: absolute; right: -34px; top: -40px; width: 128px; height: 128px; border-radius: 50%; background: #7c3aed; opacity: .12; }
.skill-top { position: relative; display: flex; justify-content: space-between; align-items: center; }
.skill-icon { width: 46px; height: 46px; border-radius: 15px; background: #f5f3ff; color: #7c3aed; display: flex; align-items: center; justify-content: center; font-size: 22px; }
.skill-tags { display: flex; align-items: center; gap: 8px; }
.state-pill { font-size: 11.5px; font-weight: 750; padding: 4px 10px; border-radius: 999px; }
.state-pill.on { background: #ecfdf5; color: #059669; }
.state-pill.off { background: #f8fafc; color: #94a3b8; }

/* ── card body ── */
.skill-main { margin-top: 18px; position: relative; }
.skill-main h3 { margin: 0 0 4px; color: #0f172a; font-size: 17px; letter-spacing: -.25px; }
.skill-main > p { margin: 0 0 10px; color: #64748b; font-size: 12.5px; line-height: 1.55; }

/* structural chips */
.struct-chips { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 12px; }
.chip { display: inline-flex; align-items: center; gap: 4px; font-size: 11px; font-weight: 650; padding: 3px 9px; border-radius: 999px; border: 1px solid transparent; }
.chip .el-icon { font-size: 11px; }
.chip-agent    { background: #eff6ff; color: #2563eb; border-color: #bfdbfe; }
.chip-registry { background: #f5f3ff; color: #7c3aed; border-color: #ddd6fe; }
.chip-policy   { background: #ecfdf5; color: #059669; border-color: #a7f3d0; }
.chip-workflow { background: #fff7ed; color: #c2410c; border-color: #fed7aa; }
.chip-context  { background: #fdf4ff; color: #a21caf; border-color: #f0abfc; }
.chip-memory   { background: #f0fdf4; color: #16a34a; border-color: #bbf7d0; }

/* active skill prompt in card */
.prompt-section { margin-top: 2px; }
.prompt-label { display: flex; align-items: center; gap: 5px; font-size: 11px; font-weight: 700; color: #7c3aed; letter-spacing: .04em; text-transform: uppercase; margin-bottom: 5px; }
.prompt-preview { background: #f8fafc; border: 1px solid #eef2f7; border-radius: 12px; padding: 10px 12px; min-height: 60px; max-height: 96px; overflow: hidden; white-space: pre-wrap; color: #475569; font-size: 12px; line-height: 1.55; }

.skill-meta { display: flex; justify-content: space-between; gap: 12px; margin-top: 14px; color: #94a3b8; font-size: 12px; }
.skill-actions { display: flex; justify-content: space-between; align-items: center; gap: 12px; border-top: 1px solid #f1f5f9; padding-top: 14px; margin-top: 14px; }
.action-buttons { display: flex; align-items: center; gap: 8px; }
.action-buttons :deep(.el-button--primary) { border-color: #7c3aed !important; color: #7c3aed !important; background: #fff !important; font-weight: 650; }
.action-buttons :deep(.el-button--primary:hover) { background: #f5f3ff !important; }
.action-buttons :deep(.el-button--danger) { border-color: #ef4444 !important; color: #ef4444 !important; background: #fff !important; font-weight: 650; }
.action-buttons :deep(.el-button--danger:hover) { background: #fef2f2 !important; }

/* ── empty ── */
.empty-state { background: #fff; border: 1px solid #e2e8f0; border-radius: 22px; padding: 54px; text-align: center; box-shadow: var(--sh-card); }
.empty-icon { width: 58px; height: 58px; margin: 0 auto 14px; border-radius: 18px; background: #f5f3ff; color: #7c3aed; display: flex; align-items: center; justify-content: center; font-size: 26px; }
.empty-state h3 { margin: 0 0 6px; color: #0f172a; }
.empty-state p { margin: 0 0 18px; color: #64748b; font-size: 13px; }

/* ── form ── */
.form-inline { display: flex; gap: 24px; align-items: center; }
.skill-form { max-height: 72vh; overflow-y: auto; padding-right: 4px; }
.form-section-title { font-size: 12px; font-weight: 750; letter-spacing: .08em; text-transform: uppercase; color: #64748b; margin: 18px 0 10px; padding-bottom: 6px; border-bottom: 1px solid #f1f5f9; }
.prompt-section-title { display: flex; align-items: center; gap: 6px; color: #7c3aed; border-color: #ede9fe; }
.section-badge { font-size: 10.5px; font-weight: 700; padding: 2px 8px; border-radius: 999px; background: #f5f3ff; color: #7c3aed; border: 1px solid #ddd6fe; text-transform: none; letter-spacing: 0; }
.collapsible-group-title { margin-top: 22px; }

/* collapsible sections */
.collapsible-section { border: 1px solid #e8edf4; border-radius: 12px; margin-bottom: 8px; overflow: hidden; }
.collapsible-header { display: flex; justify-content: space-between; align-items: center; padding: 10px 14px; cursor: pointer; user-select: none; background: #f8fafc; transition: background .14s; }
.collapsible-header:hover { background: #f1f5f9; }
.cs-left { display: flex; align-items: center; gap: 8px; }
.cs-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.dot-agent    { background: #2563eb; }
.dot-registry { background: #7c3aed; }
.dot-policy   { background: #059669; }
.dot-workflow { background: #c2410c; }
.dot-context  { background: #a21caf; }
.dot-memory   { background: #16a34a; }
.dot-meta     { background: #94a3b8; }
.cs-label { font-size: 13px; font-weight: 650; color: #1e293b; }
.cs-hint  { font-size: 11px; color: #94a3b8; font-family: monospace; }
.cs-right { display: flex; align-items: center; gap: 8px; }
.cs-filled { font-size: 11px; font-weight: 650; color: #059669; background: #ecfdf5; padding: 2px 8px; border-radius: 999px; }
.cs-arrow { font-size: 13px; color: #94a3b8; transition: transform .2s; }
.cs-arrow.open { transform: rotate(180deg); }
.collapsible-body { padding: 10px 14px 14px; background: #fff; }
:deep(.mono-area .el-textarea__inner) { font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace; font-size: 12px; line-height: 1.6; }

/* transition */
.section-fade-enter-active, .section-fade-leave-active { transition: opacity .18s, max-height .22s ease; max-height: 400px; overflow: hidden; }
.section-fade-enter-from, .section-fade-leave-to { opacity: 0; max-height: 0; }

:deep(.skill-dialog .el-dialog) { border-radius: 22px !important; overflow: hidden; }
</style>
