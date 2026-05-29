<template>
  <div class="page-wrap">
    <!-- 页头 -->
    <div class="page-header">
      <el-button link @click="router.push('/models')" class="back-btn">
        <el-icon><ArrowLeft /></el-icon> 返回
      </el-button>
      <div class="header-content">
        <h1>新建模型配置</h1>
        <p>接入 AI Provider，用于自动代码审查</p>
      </div>
    </div>

    <div class="form-body">
      <!-- 左侧主表单 -->
      <el-form :model="form" :rules="rules" ref="formRef" label-position="top" class="main-form">

        <!-- ① 基本信息 -->
        <section class="form-section">
          <div class="section-label"><span class="section-num">01</span> 基本信息</div>
          <el-form-item label="配置名称" prop="name">
            <el-input v-model="form.name" placeholder="例如：GPT-4o Production" size="large" clearable>
              <template #prefix><el-icon color="#9ca3af"><EditPen /></el-icon></template>
            </el-input>
          </el-form-item>
        </section>

        <!-- ② 平台选择 -->
        <section class="form-section">
          <div class="section-label"><span class="section-num">02</span> 选择平台</div>
          <div class="platform-grid">
            <div
              v-for="p in platforms"
              :key="p.value"
              class="platform-card"
              :class="{ active: form.type === p.value }"
              @click="form.type = p.value"
            >
              <span class="platform-icon">{{ p.icon }}</span>
              <span class="platform-name">{{ p.label }}</span>
              <el-icon v-if="form.type === p.value" class="check-icon" color="#2563eb"><Select /></el-icon>
            </div>
          </div>
        </section>

        <!-- ③ API 模式字段 -->
        <transition name="fade-slide">
          <section class="form-section" v-if="!isCliMode" key="api">
            <div class="section-label"><span class="section-num">03</span> 接口配置</div>
            <div class="field-grid">
              <el-form-item label="API 地址" prop="base_url" class="span-full">
                <el-input v-model="form.base_url" placeholder="留空使用默认地址（如 https://api.openai.com/v1）" clearable>
                  <template #prefix><el-icon color="#9ca3af"><Link /></el-icon></template>
                </el-input>
              </el-form-item>
              <el-form-item label="API Key" prop="api_key" class="span-full">
                <el-input v-model="form.api_key" type="password" show-password placeholder="sk-..." clearable>
                  <template #prefix><el-icon color="#9ca3af"><Key /></el-icon></template>
                </el-input>
              </el-form-item>
              <el-form-item label="模型名称" prop="model">
                <el-input v-model="form.model" placeholder="gpt-4o / claude-3-5-sonnet" clearable />
              </el-form-item>
              <el-form-item label="最大上下文 Token" prop="max_context">
                <el-input-number v-model="form.max_context" :min="1000" :step="1000" style="width:100%" />
              </el-form-item>
              <el-form-item label="启用 Thinking 模式" class="span-full">
                <div class="switch-row">
                  <el-switch v-model="form.enable_thinking" active-color="#2563eb" />
                  <span class="switch-desc">开启后模型将在回答前进行深度推理（仅 claude-3-7 以上支持）</span>
                </div>
              </el-form-item>
            </div>
          </section>
        </transition>

        <!-- ③ CLI 模式字段 -->
        <transition name="fade-slide">
          <section class="form-section" v-if="isCliMode" key="cli">
            <div class="section-label"><span class="section-num">03</span> CLI 配置</div>
            <div class="field-grid">
              <el-form-item label="CLI 可执行文件路径" prop="cli_path" class="span-full">
                <el-input v-model="form.cli_path" placeholder="claude">
                  <template #prefix><el-icon color="#9ca3af"><Monitor /></el-icon></template>
                </el-input>
              </el-form-item>
              <el-form-item label="Max Turns" prop="max_turns">
                <el-input-number v-model="form.max_turns" :min="1" style="width:100%" placeholder="不限" />
              </el-form-item>
              <el-form-item label="环境变量（JSON）" prop="env_vars_json" class="span-full">
                <el-input
                  v-model="form.env_vars_json"
                  type="textarea" :rows="4"
                  class="code-textarea"
                  placeholder='{"ANTHROPIC_API_KEY": "sk-ant-..."}'
                />
              </el-form-item>
            </div>
          </section>
        </transition>

        <!-- ④ Prompt -->
        <section class="form-section">
          <div class="section-label">
            <span class="section-num">04</span> Review Prompt
            <div class="prompt-tabs">
              <span :class="['prompt-tab', promptTab === 'edit' && 'active']" @click="promptTab = 'edit'">编辑</span>
              <span :class="['prompt-tab', promptTab === 'preview' && 'active']" @click="promptTab = 'preview'">预览</span>
            </div>
            <el-button text :icon="FullScreen" size="small" class="prompt-fs-btn" @click="promptFullscreen = true">全屏编辑</el-button>
          </div>
          <el-form-item prop="prompt">
            <el-input
              v-if="promptTab === 'edit'"
              v-model="form.prompt"
              type="textarea" :rows="14"
              class="code-textarea prompt-input"
              placeholder="请输入 Review Prompt，告诉模型如何审查代码..."
            />
            <div v-else class="prompt-preview markdown-body" v-html="renderedPrompt || '<span class=\'preview-empty\'>暂无内容</span>'" />
            <div class="prompt-tip">提示：可以在 Prompt 中说明重点关注的内容，如安全漏洞、代码规范、性能等。</div>
          </el-form-item>
        </section>

        <!-- 操作区 -->
        <div class="form-actions">
          <el-button size="large" :loading="testLoading" @click="handleTest" class="test-btn">
            <el-icon><Connection /></el-icon> 测试连接
          </el-button>
          <div class="right-actions">
            <el-button size="large" @click="router.push('/models')">取消</el-button>
            <el-button size="large" type="primary" @click="handleSubmit" :loading="submitLoading" class="submit-btn">
              创建配置
            </el-button>
          </div>
        </div>
      </el-form>

      <!-- 右侧提示卡 -->
      <aside class="side-tips">
        <div class="tip-card">
          <div class="tip-icon">💡</div>
          <h4>配置说明</h4>
          <ul>
            <li><strong>API 模式</strong>：直接调用 Provider 接口，支持 OpenAI / Anthropic / Ollama 等</li>
            <li><strong>CLI 模式</strong>：通过本地 Claude CLI 工具执行审查，适合已有 Claude 授权的环境</li>
          </ul>
        </div>
        <div class="tip-card">
          <div class="tip-icon">🔒</div>
          <h4>API Key 安全</h4>
          <p>API Key 加密存储，不会在日志中明文输出。</p>
        </div>
        <div class="tip-card">
          <div class="tip-icon">⚡</div>
          <h4>Prompt 建议</h4>
          <p>Prompt 应清晰描述审查目标，例如："你是一名资深工程师，请对以下代码变更进行安全和性能审查..."</p>
        </div>
      </aside>
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
          class="code-textarea prompt-dialog-input"
          placeholder="请输入 Review Prompt，告诉模型如何审查代码..."
        />
        <div v-else class="prompt-dialog-preview" v-html="renderedPrompt || '<span class=\'preview-empty\'>暂无内容</span>'" />
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  ArrowLeft, EditPen, Link, Key, Select, Connection, Monitor, FullScreen,
} from '@element-plus/icons-vue'
import { marked } from 'marked'
import { createModel, testModel } from '../../api/models'

const router = useRouter()
const formRef = ref()
const testLoading = ref(false)
const submitLoading = ref(false)
const promptFullscreen = ref(false)
const promptTab = ref('preview')
const fsTab = ref('preview')
const renderedPrompt = computed(() => form.value.prompt ? marked.parse(form.value.prompt) : '')

const platforms = [
  { value: 'openai',     label: 'OpenAI',    icon: '🤖' },
  { value: 'anthropic',  label: 'Anthropic',  icon: '🧠' },
  { value: 'deepseek',   label: 'DeepSeek',   icon: '🔍' },
  { value: 'gemini',     label: 'Gemini',     icon: '✨' },
  { value: 'mistral',    label: 'Mistral',    icon: '💨' },
  { value: 'ollama',     label: 'Ollama',     icon: '🦙' },
  { value: 'claude_cli', label: 'Claude CLI', icon: '⌨️' },
]

const form = ref({
  name: '', type: 'openai',
  base_url: '', api_key: '', model: '',
  max_context: 32000, enable_thinking: false,
  cli_path: 'claude', env_vars_json: '{}', max_turns: null,
  prompt: '',
})

const rules = {
  name:   [{ required: true, message: '请输入配置名称', trigger: 'blur' }],
  prompt: [{ required: true, message: '请输入 Review Prompt', trigger: 'blur' }],
}

const isCliMode = computed(() => form.value.type === 'claude_cli')

const handleTest = async () => {
  testLoading.value = true
  try {
    const { data } = await testModel({ ...form.value })
    ElMessage({ message: data.message || '连接测试成功', type: 'success', duration: 4000 })
  } catch (err) {
    ElMessage.error(err.response?.data?.message || err.response?.data?.error || '连接失败')
  } finally {
    testLoading.value = false
  }
}

const handleSubmit = async () => {
  await formRef.value.validate()
  submitLoading.value = true
  try {
    await createModel({ ...form.value })
    ElMessage.success('创建成功')
    router.push('/models')
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '创建失败')
  } finally {
    submitLoading.value = false
  }
}
</script>

<style scoped>
.page-wrap { padding: 32px 36px; min-height: 100%; }

/* 页头 */
.back-btn { color: #6b7280; margin-bottom: 16px; padding: 0; font-size: 13px; }
.back-btn:hover { color: #2563eb; }
.header-content h1 { font-size: 24px; font-weight: 700; color: #111827; margin: 0 0 6px; }
.header-content p  { font-size: 14px; color: #6b7280; margin: 0 0 28px; }

/* 布局 */
.form-body { display: flex; gap: 32px; align-items: flex-start; }
.main-form { flex: 1; min-width: 0; }
.side-tips { width: 260px; flex-shrink: 0; position: sticky; top: 24px; display: flex; flex-direction: column; gap: 14px; }

/* 分区 */
.form-section {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 16px;
  border: 1px solid #e2e8f0;
}

.section-label {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  font-weight: 600;
  color: #374151;
  margin-bottom: 20px;
}

.section-num {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px; height: 22px;
  border-radius: 50%;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  flex-shrink: 0;
}

/* 平台卡片网格 */
.platform-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
}

.platform-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 14px 10px;
  border-radius: 10px;
  border: 2px solid #f0f0f0;
  background: #fafafa;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.platform-card:hover { border-color: #93c5fd; background: #eff6ff; }
.platform-card.active {
  border-color: #2563eb;
  background: linear-gradient(135deg, #eff6ff, #f0f0ff);
  box-shadow: 0 0 0 3px rgba(37,99,235,0.1);
}

.platform-icon { font-size: 22px; line-height: 1; }
.platform-name { font-size: 12px; font-weight: 500; color: #374151; }
.check-icon { position: absolute; top: 6px; right: 6px; font-size: 13px; }

/* 字段网格 */
.field-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0 20px; }
.field-grid :deep(.el-form-item) { margin-bottom: 18px; }
.span-full { grid-column: 1 / -1; }

/* switch 行 */
.switch-row { display: flex; align-items: center; gap: 12px; }
.switch-desc { font-size: 12px; color: #6b7280; }

/* 代码风格 textarea */
.code-textarea :deep(textarea) {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
  line-height: 1.6;
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 8px;
  border-color: #1e293b;
  resize: vertical;
}
.code-textarea :deep(textarea)::placeholder { color: #475569; }

.prompt-input :deep(textarea) { min-height: 260px; }
.prompt-tip { font-size: 12px; color: #9ca3af; margin-top: 6px; }

.prompt-fs-btn { margin-left: auto; color: #6b7280; font-size: 12px; }
.prompt-fs-btn:hover { color: #2563eb; }

.prompt-tabs {
  display: flex;
  background: #f1f5f9;
  border-radius: 6px;
  padding: 2px;
  gap: 2px;
  margin-left: 12px;
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

.prompt-preview {
  width: 100%;
  height: 300px;
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
  min-height: unset;
}
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

/* 操作栏 */
.form-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #fff;
  border-radius: 12px;
  padding: 16px 24px;
  border: 1px solid #e2e8f0;
}

.right-actions { display: flex; gap: 10px; }

.test-btn { color: #374151; border-color: #d1d5db; }
.test-btn:hover { color: #2563eb; border-color: #2563eb; background: #eff6ff; }

.submit-btn {
  background: linear-gradient(90deg, #2563eb, #7c3aed);
  border: none;
  padding: 0 28px;
}
.submit-btn:hover { opacity: 0.9; }

/* 右侧提示卡 */
.tip-card {
  background: #fff;
  border-radius: 12px;
  padding: 18px;
  border: 1px solid #e2e8f0;
}

.tip-icon { font-size: 20px; margin-bottom: 8px; }
.tip-card h4 { font-size: 13px; font-weight: 600; color: #374151; margin: 0 0 8px; }
.tip-card p, .tip-card li { font-size: 12px; color: #6b7280; line-height: 1.6; margin: 0; }
.tip-card ul { padding-left: 16px; margin: 0; display: flex; flex-direction: column; gap: 6px; }

/* 过渡动画 */
.fade-slide-enter-active, .fade-slide-leave-active { transition: all 0.25s ease; }
.fade-slide-enter-from { opacity: 0; transform: translateY(-6px); }
.fade-slide-leave-to   { opacity: 0; transform: translateY(6px); }
</style>
