<template>
  <div class="page-wrap">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">编辑配置</h2>
        <p class="page-subtitle">修改模型配置信息</p>
      </div>
    </div>

    <el-form :model="form" :rules="rules" ref="formRef" label-position="top" class="edit-form">
      <el-form-item label="配置名称" prop="name">
        <el-input v-model="form.name" placeholder="请输入配置名称" />
      </el-form-item>

      <el-form-item label="平台类型" prop="type">
        <el-select v-model="form.type" placeholder="请选择平台" style="width: 100%">
          <el-option label="OpenAI" value="openai" />
          <el-option label="Anthropic" value="anthropic" />
          <el-option label="Ollama" value="ollama" />
          <el-option label="DeepSeek" value="deepseek" />
          <el-option label="Gemini" value="gemini" />
          <el-option label="Mistral" value="mistral" />
          <el-option label="Claude CLI" value="claude_cli" />
        </el-select>
      </el-form-item>

      <!-- 非 claude_cli 字段 -->
      <template v-if="!isCliMode">
        <el-form-item label="API 地址" prop="base_url">
          <el-input v-model="form.base_url" placeholder="https://api.openai.com/v1" />
        </el-form-item>

        <el-form-item label="API Key" prop="api_key">
          <el-input v-model="form.api_key" type="password" show-password placeholder="请输入 API Key" />
        </el-form-item>

        <el-form-item label="模型名称" prop="model">
          <el-input v-model="form.model" placeholder="例如 gpt-4o" />
        </el-form-item>

        <el-form-item label="最大上下文 Token" prop="max_context">
          <el-input-number v-model="form.max_context" :min="1" style="width: 100%" />
        </el-form-item>

        <el-form-item label="启用 Thinking">
          <el-switch v-model="form.enable_thinking" />
        </el-form-item>
      </template>

      <!-- claude_cli 字段 -->
      <template v-else>
        <el-form-item label="CLI 路径" prop="cli_path">
          <el-input v-model="form.cli_path" placeholder="claude" />
        </el-form-item>

        <el-form-item label="环境变量 JSON" prop="env_vars_json">
          <el-input
            v-model="form.env_vars_json"
            type="textarea"
            :rows="4"
            placeholder='{"ANTHROPIC_API_KEY": "sk-..."}'
          />
        </el-form-item>

        <el-form-item label="Max Turns" prop="max_turns">
          <el-input-number v-model="form.max_turns" :min="1" style="width: 100%" />
        </el-form-item>
      </template>

      <el-form-item prop="prompt">
        <template #label>
          <div class="prompt-label-row">
            <span>Review Prompt</span>
            <div class="prompt-tabs">
              <span :class="['prompt-tab', promptTab === 'edit' && 'active']" @click="promptTab = 'edit'">编辑</span>
              <span :class="['prompt-tab', promptTab === 'preview' && 'active']" @click="promptTab = 'preview'">预览</span>
            </div>
            <el-button text :icon="FullScreen" size="small" class="prompt-fs-btn" @click="promptFullscreen = true">全屏编辑</el-button>
          </div>
        </template>
        <el-input
          v-if="promptTab === 'edit'"
          v-model="form.prompt"
          type="textarea"
          :rows="16"
          class="prompt-textarea"
          placeholder="请输入 Review Prompt"
        />
        <div v-else class="prompt-preview" v-html="renderedPrompt || '<span class=\'preview-empty\'>暂无内容</span>'" />
      </el-form-item>

      <el-form-item>
        <div class="form-actions">
          <el-button :loading="testLoading" @click="handleTestConnection">测试连接</el-button>
          <div class="action-right">
            <el-button @click="router.push('/models')">取消</el-button>
            <el-button type="primary" @click="handleSubmit">保存修改</el-button>
          </div>
        </div>
      </el-form-item>
    </el-form>

    <!-- 危险区 -->
    <div class="danger-wrap">
      <el-card class="danger-zone">
        <template #header>
          <span class="danger-title">危险区</span>
        </template>
        <div class="danger-row">
          <p class="danger-tip">删除后不可恢复，请确保无项目使用该配置。</p>
          <el-button type="danger" plain @click="handleDelete">删除配置</el-button>
        </div>
      </el-card>
    </div>

    <!-- Prompt 全屏编辑 -->
    <el-dialog v-model="promptFullscreen" title="Review Prompt" fullscreen>
      <div class="prompt-dialog-body">
        <div class="fs-tab-bar">
          <span :class="['prompt-tab', fsTab === 'preview' && 'active']" @click="fsTab = 'preview'">预览</span>
          <span :class="['prompt-tab', fsTab === 'edit' && 'active']" @click="fsTab = 'edit'">编辑</span>
        </div>
        <el-input
          v-if="fsTab === 'edit'"
          v-model="form.prompt"
          type="textarea"
          class="prompt-dialog-input"
          placeholder="请输入 Review Prompt"
        />
        <div v-else class="prompt-dialog-preview" v-html="renderedPrompt || '<span class=\'preview-empty\'>暂无内容</span>'" />
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { FullScreen } from '@element-plus/icons-vue'
import { marked } from 'marked'
import { getModel, updateModel, testModel, deleteModel } from '../../api/models'

const router = useRouter()
const route = useRoute()
const formRef = ref()
const promptFullscreen = ref(false)
const promptTab = ref('preview')
const fsTab = ref('preview')
const renderedPrompt = computed(() => form.value.prompt ? marked.parse(form.value.prompt) : '')

const id = route.params.id

const form = ref({
  name: '',
  type: 'openai',
  base_url: '',
  api_key: '',
  model: '',
  max_context: 32000,
  enable_thinking: false,
  cli_path: 'claude',
  env_vars_json: '{}',
  max_turns: null,
  prompt: '',
})

const rules = {
  name: [{ required: true, message: '请输入配置名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择平台类型', trigger: 'change' }],
  prompt: [{ required: true, message: '请输入 Review Prompt', trigger: 'blur' }],
}

const isCliMode = computed(() => form.value.type === 'claude_cli')

const testLoading = ref(false)

onMounted(async () => {
  try {
    const res = await getModel(id)
    const data = res.data
    form.value = {
      name: data.name || '',
      type: data.type || 'openai',
      base_url: data.base_url || '',
      api_key: data.api_key || '',
      model: data.model || '',
      max_context: data.max_context || 32000,
      enable_thinking: data.enable_thinking || false,
      cli_path: data.cli_path || 'claude',
      env_vars_json: data.env_vars_json || '{}',
      max_turns: data.max_turns || null,
      prompt: data.prompt || '',
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  }
})

const handleTestConnection = async () => {
  testLoading.value = true
  try {
    await testModel({
      name: form.value.name,
      type: form.value.type,
      base_url: form.value.base_url,
      api_key: form.value.api_key,
      model: form.value.model,
      max_context: form.value.max_context,
      enable_thinking: form.value.enable_thinking,
      cli_path: form.value.cli_path,
      env_vars_json: form.value.env_vars_json,
      max_turns: form.value.max_turns,
      prompt: form.value.prompt,
    })
    ElMessage.success('连接测试成功')
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  } finally {
    testLoading.value = false
  }
}

const handleDelete = async () => {
  try {
    await ElMessageBox.confirm(
      `确认删除模型配置 "${form.value.name}"？此操作不可恢复。`,
      '删除确认',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
    await deleteModel(id)
    ElMessage.success('删除成功')
    router.push('/models')
  } catch (err) {
    if (err !== 'cancel' && err?.type !== 'cancel') {
      ElMessage.error(err.response?.data?.message || '删除失败')
    }
  }
}

const handleSubmit = async () => {
  await formRef.value.validate()
  try {
    await updateModel(id, form.value)
    router.push('/models')
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  }
}
</script>

<style scoped>
.page-wrap { padding: 32px 36px; }

.page-header { margin-bottom: 24px; }

.page-title { margin: 0 0 4px; font-size: 22px; font-weight: 700; color: #1e293b; letter-spacing: -0.3px; }
.page-subtitle { margin: 0; font-size: 14px; color: #64748b; }

.edit-form { max-width: 860px; }

.form-actions { display: flex; justify-content: space-between; width: 100%; }
.action-right { display: flex; gap: 8px; }

.prompt-label-row {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.prompt-fs-btn { margin-left: auto; color: #6b7280; font-size: 12px; }
.prompt-fs-btn:hover { color: #2563eb; }

.prompt-tabs {
  display: flex;
  background: #f1f5f9;
  border-radius: 6px;
  padding: 2px;
  gap: 2px;
}
.prompt-tab {
  padding: 3px 14px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  color: #6b7280;
  cursor: pointer;
  user-select: none;
  transition: all 0.15s;
}
.prompt-tab.active {
  background: #fff;
  color: #2563eb;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}
.prompt-tab:hover:not(.active) { color: #374151; }

/* 让 el-form-item 的 label 容纳 flex 内容且星号不换行 */
.edit-form :deep(.el-form-item__label) {
  display: inline-flex;
  align-items: center;
  width: 100%;
  height: auto;
  white-space: nowrap;
}

.prompt-preview {
  width: 100%;
  height: 340px;
  overflow-y: auto;
  padding: 12px 16px;
  background: #0f172a;
  border: 1px solid #1e293b;
  border-radius: 8px;
  font-size: 13px;
  line-height: 1.6;
  color: #e2e8f0;
  box-sizing: border-box;
}
.prompt-preview :deep(h1),
.prompt-preview :deep(h2),
.prompt-preview :deep(h3) { color: #f1f5f9; margin: 10px 0 6px; font-weight: 600; }
.prompt-preview :deep(h1) { font-size: 1.2em; border-bottom: 1px solid #1e293b; padding-bottom: 6px; }
.prompt-preview :deep(h2) { font-size: 1.05em; }
.prompt-preview :deep(h3) { font-size: 0.95em; }
.prompt-preview :deep(p) { margin: 6px 0; }
.prompt-preview :deep(ul), .prompt-preview :deep(ol) { padding-left: 20px; margin: 6px 0; }
.prompt-preview :deep(li) { margin: 3px 0; }
.prompt-preview :deep(code) {
  background: #1e293b; border-radius: 3px; padding: 1px 5px;
  color: #7dd3fc; font-size: 12.5px;
}
.prompt-preview :deep(pre) {
  background: #020817; border-radius: 6px;
  padding: 10px 14px; overflow-x: auto; margin: 8px 0;
  border: 1px solid #1e293b;
}
.prompt-preview :deep(pre code) { background: none; color: #e2e8f0; padding: 0; }
.prompt-preview :deep(blockquote) {
  border-left: 3px solid #6366f1; padding: 6px 12px;
  margin: 8px 0; color: #94a3b8;
}
.prompt-preview :deep(strong) { color: #f1f5f9; }
.prompt-preview :deep(.preview-empty) { color: #475569; }

.prompt-textarea :deep(textarea) {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
  line-height: 1.6;
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 8px;
  border-color: #1e293b;
  resize: vertical;
}
.prompt-textarea :deep(textarea)::placeholder { color: #475569; }

.prompt-dialog-body {
  padding: 0 24px 24px;
  height: calc(100vh - 108px);
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.fs-tab-bar {
  display: flex;
  background: #1e293b;
  border-radius: 6px;
  padding: 2px;
  gap: 2px;
  align-self: flex-start;
}
.fs-tab-bar .prompt-tab { color: #94a3b8; }
.fs-tab-bar .prompt-tab.active { background: #334155; color: #e2e8f0; box-shadow: none; }
.prompt-dialog-input { flex: 1; display: flex; flex-direction: column; }
.prompt-dialog-input :deep(.el-textarea) { flex: 1; display: flex; flex-direction: column; }
.prompt-dialog-input :deep(textarea) {
  flex: 1;
  resize: none;
  height: 100% !important;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
  line-height: 1.6;
  background: #0f172a;
  color: #e2e8f0;
  border-color: #1e293b;
}
.prompt-dialog-input :deep(textarea)::placeholder { color: #475569; }
.prompt-dialog-preview {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  background: #0f172a;
  border: 1px solid #1e293b;
  border-radius: 8px;
  font-size: 13.5px;
  line-height: 1.7;
  color: #e2e8f0;
}
.prompt-dialog-preview :deep(h1),
.prompt-dialog-preview :deep(h2),
.prompt-dialog-preview :deep(h3) { color: #f1f5f9; margin: 12px 0 8px; font-weight: 600; }
.prompt-dialog-preview :deep(h1) { font-size: 1.3em; border-bottom: 1px solid #1e293b; padding-bottom: 8px; }
.prompt-dialog-preview :deep(h2) { font-size: 1.1em; }
.prompt-dialog-preview :deep(h3) { font-size: 1em; }
.prompt-dialog-preview :deep(p) { margin: 8px 0; }
.prompt-dialog-preview :deep(ul), .prompt-dialog-preview :deep(ol) { padding-left: 22px; margin: 8px 0; }
.prompt-dialog-preview :deep(li) { margin: 4px 0; }
.prompt-dialog-preview :deep(code) {
  background: #1e293b; border-radius: 3px; padding: 1px 6px; color: #7dd3fc;
}
.prompt-dialog-preview :deep(pre) {
  background: #020817; border-radius: 6px; padding: 12px 16px;
  overflow-x: auto; margin: 10px 0; border: 1px solid #1e293b;
}
.prompt-dialog-preview :deep(pre code) { background: none; color: #e2e8f0; padding: 0; }
.prompt-dialog-preview :deep(blockquote) {
  border-left: 3px solid #6366f1; padding: 8px 14px; margin: 10px 0; color: #94a3b8;
}
.prompt-dialog-preview :deep(strong) { color: #f1f5f9; }
.prompt-dialog-preview :deep(.preview-empty) { color: #475569; }

.danger-wrap { max-width: 860px; margin-top: 40px; }
.danger-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.danger-zone { border-color: #fecaca !important; }
.danger-zone :deep(.el-card__header) { border-bottom-color: #fecaca; background: #fff5f5; }
.danger-title { color: #ef4444; font-weight: 600; }
.danger-tip { margin: 0; font-size: 13px; color: #64748b; }
</style>
