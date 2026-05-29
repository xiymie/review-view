<template>
  <div class="page-wrap">
    <div class="page-header">
      <div>
        <h2 class="page-title">编辑项目 · {{ form.name }}</h2>
        <p class="page-subtitle">修改项目配置</p>
      </div>
    </div>

    <el-card class="form-card">
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        @submit.prevent="handleSubmit"
      >
        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="项目名称" prop="name">
              <el-input v-model="form.name" placeholder="例如：backend-api" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="仓库地址" prop="repo_url">
              <el-input v-model="form.repo_url" placeholder="https://github.com/example/repo" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="仓库凭据" prop="repo_credential_id">
              <el-select v-model="form.repo_credential_id" placeholder="选择凭据" style="width:100%">
                <el-option :value="null" label="无（公开仓库）" />
                <el-option
                  v-for="c in credentials"
                  :key="c.id"
                  :value="c.id"
                  :label="c.name"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="分支" prop="branch">
              <el-input v-model="form.branch" placeholder="main" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="模型配置" prop="model_config_id">
              <el-select v-model="form.model_config_id" placeholder="选择模型配置" style="width:100%">
                <el-option
                  v-for="m in models"
                  :key="m.id"
                  :value="m.id"
                  :label="m.name"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="溢出策略" prop="overflow_strategy">
              <el-radio-group v-model="form.overflow_strategy">
                <el-radio value="queue">排队等待</el-radio>
                <el-radio value="reject">拒绝</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item prop="custom_prompt">
          <template #label>
            <div class="prompt-label-row">
              <span>自定义 Prompt</span>
              <div class="prompt-tabs">
                <span :class="['prompt-tab', promptTab === 'edit' && 'active']" @click="promptTab = 'edit'">编辑</span>
                <span :class="['prompt-tab', promptTab === 'preview' && 'active']" @click="promptTab = 'preview'">预览</span>
              </div>
              <el-button text :icon="FullScreen" size="small" class="prompt-fs-btn" @click="promptFullscreen = true">全屏编辑</el-button>
            </div>
          </template>
          <el-input
            v-if="promptTab === 'edit'"
            v-model="form.custom_prompt"
            type="textarea"
            :rows="10"
            class="prompt-textarea"
            placeholder="描述项目背景、技术栈、审查重点等补充信息，例如：这是一个 Go 后端服务，重点关注并发安全和数据库操作..."
          />
          <div v-else class="prompt-preview" v-html="renderedPrompt || '<span class=\'preview-empty\'>暂无内容</span>'" />
          <div class="field-hint">将追加到模型全局 Prompt 之后，作为本项目的补充说明。</div>
        </el-form-item>

        <el-form-item label="任务超时（分钟）" prop="task_timeout">
          <el-input-number
            v-model="form.task_timeout"
            :min="1"
            :max="1440"
            style="width:200px"
          />
        </el-form-item>

        <div class="form-actions">
          <el-button @click="router.push(`/projects/${projectId}`)">取消</el-button>
          <el-button type="primary" native-type="submit" :loading="submitting">保存</el-button>
        </div>
      </el-form>
    </el-card>

    <!-- 危险区域 -->
    <el-card class="danger-card">
      <template #header>
        <span class="danger-title">危险区域</span>
      </template>
      <div class="danger-body">
        <div>
          <div class="danger-label">删除项目</div>
          <div class="danger-desc">删除后所有相关任务和数据将一并移除，不可恢复。</div>
        </div>
        <el-button type="danger" plain @click="handleDelete">删除项目</el-button>
      </div>
    </el-card>

    <!-- Prompt 全屏编辑 -->
    <el-dialog v-model="promptFullscreen" title="自定义 Prompt" fullscreen>
      <div class="prompt-dialog-body">
        <div class="fs-tab-bar">
          <span :class="['prompt-tab', fsTab === 'preview' && 'active']" @click="fsTab = 'preview'">预览</span>
          <span :class="['prompt-tab', fsTab === 'edit' && 'active']" @click="fsTab = 'edit'">编辑</span>
        </div>
        <el-input
          v-if="fsTab === 'edit'"
          v-model="form.custom_prompt"
          type="textarea"
          class="prompt-dialog-input"
          placeholder="描述项目背景、技术栈、审查重点等补充信息..."
        />
        <div v-else class="prompt-dialog-preview" v-html="renderedPrompt || '<span class=\'preview-empty\'>暂无内容</span>'" />
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import { FullScreen } from '@element-plus/icons-vue'
import { marked } from 'marked'
import { getProject, updateProject, deleteProject, listModels, listCredentials } from '../../api/projects'

const router = useRouter()
const route = useRoute()

const projectId = computed(() => route.params.id)

const formRef = ref(null)
const submitting = ref(false)
const models = ref([])
const credentials = ref([])
const promptTab = ref('preview')
const fsTab = ref('preview')
const promptFullscreen = ref(false)

const form = ref({
  name: '',
  repo_url: '',
  repo_credential_id: null,
  branch: 'main',
  model_config_id: null,
  custom_prompt: '',
  overflow_strategy: 'queue',
  task_timeout: 30,
})

const renderedPrompt = computed(() => form.value.custom_prompt ? marked.parse(form.value.custom_prompt) : '')

const rules = {
  name: [{ required: true, message: '请输入项目名称', trigger: 'blur' }],
  repo_url: [{ required: true, message: '请输入仓库地址', trigger: 'blur' }],
  branch: [{ required: true, message: '请输入分支名', trigger: 'blur' }],
  model_config_id: [{ required: true, message: '请选择模型配置', trigger: 'change' }],
}

onMounted(async () => {
  try {
    const [projectRes, modelsRes, credsRes] = await Promise.all([
      getProject(projectId.value),
      listModels(),
      listCredentials(),
    ])
    const p = projectRes.data.project
    models.value = modelsRes.data
    credentials.value = credsRes.data
    form.value = {
      name: p.name,
      repo_url: p.repo_url,
      repo_credential_id: p.repo_credential_id,
      branch: p.branch,
      model_config_id: p.model_config_id,
      custom_prompt: p.custom_prompt,
      overflow_strategy: p.overflow_strategy,
      task_timeout: p.task_timeout,
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载失败')
  }
})

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    await updateProject(projectId.value, form.value)
    ElMessage.success('保存成功')
    router.push(`/projects/${projectId.value}`)
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete() {
  try {
    await ElMessageBox.confirm(
      `确定要删除项目 "${form.value.name}" 吗？此操作不可恢复。`,
      '删除确认',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
    await deleteProject(projectId.value)
    ElMessage.success('删除成功')
    router.push('/projects')
  } catch (err) {
    if (err !== 'cancel' && err?.type !== 'cancel') {
      if (err.response) ElMessage.error(err.response?.data?.message || '删除失败')
    }
  }
}
</script>

<style scoped>
.page-wrap { padding: 32px 36px; }

.page-header { margin-bottom: 24px; }

.page-title { margin: 0 0 4px; font-size: 22px; font-weight: 700; color: #1e293b; letter-spacing: -0.3px; }
.page-subtitle { margin: 0; font-size: 14px; color: #64748b; }

.form-card {
  border-radius: 10px !important;
  border: 1px solid #e2e8f0 !important;
  box-shadow: none !important;
  max-width: 900px;
  margin-bottom: 24px;
}

.form-actions {
  display: flex; justify-content: flex-end; gap: 12px;
  margin-top: 8px; padding-top: 20px; border-top: 1px solid #f1f5f9;
}

.danger-card {
  border-radius: 10px !important;
  border: 1px solid #fecaca !important;
  box-shadow: none !important;
  max-width: 900px;
}

.danger-card :deep(.el-card__header) { border-bottom: 1px solid #fecaca; background: #fff5f5; }
.danger-title { font-weight: 600; color: #ef4444; }
.danger-body { display: flex; align-items: center; justify-content: space-between; }
.danger-label { font-weight: 500; color: #1e293b; margin-bottom: 4px; }
.danger-desc { font-size: 13px; color: #64748b; }

/* Prompt label row */
.prompt-label-row {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.form-card :deep(.el-form-item__label) {
  display: inline-flex;
  align-items: center;
  width: 100%;
  height: auto;
  white-space: nowrap;
}

.field-hint { font-size: 12px; color: #9ca3af; margin-top: 5px; }

/* Prompt tabs */
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

.prompt-fs-btn { margin-left: auto; color: #6b7280; font-size: 12px; }
.prompt-fs-btn:hover { color: #2563eb; }

/* Prompt textarea */
.prompt-textarea :deep(textarea) {
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 8px;
  border-color: #1e293b;
  resize: vertical;
}
.prompt-textarea :deep(textarea)::placeholder { color: #475569; }

/* Prompt preview */
.prompt-preview {
  width: 100%;
  height: 240px;
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

/* Fullscreen dialog */
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
  font-family: 'JetBrains Mono', monospace;
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
</style>
