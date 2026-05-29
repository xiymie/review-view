<template>
  <div class="page-wrap">
    <el-button link @click="router.push('/projects')" class="back-btn">
      <el-icon><ArrowLeft /></el-icon> 返回
    </el-button>

    <div class="page-header">
      <div class="header-icon">
        <el-icon :size="26" color="#fff"><FolderAdd /></el-icon>
      </div>
      <div>
        <h1>{{ isClone ? '克隆项目' : '新建项目' }}</h1>
        <p>{{ isClone ? `基于项目 #${cloneFrom} 创建副本` : '配置仓库地址、模型与触发策略' }}</p>
      </div>
    </div>

    <div class="form-body" v-loading="initLoading">
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="main-form">

        <!-- 01 仓库信息 -->
        <section class="form-section">
          <div class="section-label"><span class="section-num">01</span>仓库信息</div>
          <div class="field-grid">
            <el-form-item label="项目名称" prop="name">
              <el-input v-model="form.name" placeholder="例如：backend-api" clearable size="large">
                <template #prefix><el-icon color="#9ca3af"><EditPen /></el-icon></template>
              </el-input>
            </el-form-item>
            <el-form-item label="监听分支" prop="branch">
              <el-input v-model="form.branch" placeholder="main" clearable size="large">
                <template #prefix><el-icon color="#9ca3af"><Share /></el-icon></template>
              </el-input>
            </el-form-item>
            <el-form-item label="仓库地址" prop="repo_url" class="span-full">
              <el-input v-model="form.repo_url" placeholder="https://github.com/example/repo.git" clearable size="large">
                <template #prefix><el-icon color="#9ca3af"><Link /></el-icon></template>
              </el-input>
            </el-form-item>
            <el-form-item label="仓库凭据" prop="repo_credential_id" class="span-full">
              <el-select v-model="form.repo_credential_id" placeholder="无（公开仓库）" style="width:100%" size="large" clearable>
                <template #prefix><el-icon color="#9ca3af"><Key /></el-icon></template>
                <el-option :value="null" label="无（公开仓库）">
                  <span style="color:#9ca3af">无（公开仓库）</span>
                </el-option>
                <el-option v-for="c in credentials" :key="c.id" :value="c.id" :label="c.name">
                  <div class="cred-option">
                    <el-icon><Key /></el-icon>
                    <span>{{ c.name }}</span>
                    <span class="cred-user">{{ c.username }}</span>
                  </div>
                </el-option>
              </el-select>
              <div class="field-hint">
                私有仓库需要选择对应凭据，
                <el-button link type="primary" size="small" @click="router.push('/credentials/new')">去新建凭据</el-button>
              </div>
            </el-form-item>
          </div>
        </section>

        <!-- 02 模型配置 -->
        <section class="form-section">
          <div class="section-label"><span class="section-num">02</span>模型配置</div>
          <el-form-item prop="model_config_id">
            <div class="model-grid" v-if="models.length">
              <div
                v-for="m in models"
                :key="m.id"
                class="model-card"
                :class="{ active: form.model_config_id === m.id }"
                @click="form.model_config_id = m.id"
              >
                <div class="model-card-inner">
                  <span class="model-type-badge">{{ m.type }}</span>
                  <span class="model-name">{{ m.name }}</span>
                  <span class="model-model">{{ m.model || 'CLI' }}</span>
                </div>
                <el-icon v-if="form.model_config_id === m.id" class="model-check" color="#2563eb"><Select /></el-icon>
              </div>
            </div>
            <el-empty v-else description="暂无模型配置" :image-size="60">
              <el-button type="primary" size="small" @click="router.push('/models/new')">去新建模型</el-button>
            </el-empty>
          </el-form-item>
        </section>

        <!-- 03 触发策略 -->
        <section class="form-section">
          <div class="section-label"><span class="section-num">03</span>触发策略</div>
          <div class="field-grid">
            <el-form-item label="溢出策略" prop="overflow_strategy" class="span-full">
              <div class="strategy-cards">
                <div
                  class="strategy-card"
                  :class="{ active: form.overflow_strategy === 'queue' }"
                  @click="form.overflow_strategy = 'queue'"
                >
                  <div class="strategy-icon">⏳</div>
                  <div class="strategy-info">
                    <strong>排队等待</strong>
                    <span>任务满时进入队列，等待空闲后执行</span>
                  </div>
                  <el-icon v-if="form.overflow_strategy === 'queue'" color="#2563eb"><Select /></el-icon>
                </div>
                <div
                  class="strategy-card"
                  :class="{ active: form.overflow_strategy === 'reject' }"
                  @click="form.overflow_strategy = 'reject'"
                >
                  <div class="strategy-icon">🚫</div>
                  <div class="strategy-info">
                    <strong>直接拒绝</strong>
                    <span>任务满时拒绝新任务，返回错误</span>
                  </div>
                  <el-icon v-if="form.overflow_strategy === 'reject'" color="#2563eb"><Select /></el-icon>
                </div>
              </div>
            </el-form-item>
            <el-form-item label="任务超时（分钟）" prop="task_timeout">
              <div class="timeout-row">
                <el-input-number v-model="form.task_timeout" :min="1" :max="1440" size="large" style="width:160px" />
                <div class="timeout-presets">
                  <el-tag
                    v-for="t in [15, 30, 60, 120]"
                    :key="t"
                    :effect="form.task_timeout === t ? 'dark' : 'plain'"
                    :type="form.task_timeout === t ? '' : 'info'"
                    class="preset-tag"
                    @click="form.task_timeout = t"
                  >{{ t }}m</el-tag>
                </div>
              </div>
            </el-form-item>
          </div>
        </section>

        <!-- 04 自定义 Prompt（可折叠） -->
        <section class="form-section">
          <div class="section-label collapsible" @click="showPrompt = !showPrompt">
            <span class="section-num">04</span>自定义 Prompt
            <el-tag size="small" type="info" effect="plain" style="margin-left:8px">可选</el-tag>
            <div class="prompt-tabs" v-if="showPrompt" @click.stop>
              <span :class="['prompt-tab', promptTab === 'edit' && 'active']" @click="promptTab = 'edit'">编辑</span>
              <span :class="['prompt-tab', promptTab === 'preview' && 'active']" @click="promptTab = 'preview'">预览</span>
            </div>
            <el-button v-if="showPrompt" text :icon="FullScreen" size="small" class="prompt-fs-btn" @click.stop="promptFullscreen = true">全屏</el-button>
            <el-icon class="collapse-icon" :style="{ transform: showPrompt ? 'rotate(180deg)' : '' }">
              <ArrowDown />
            </el-icon>
          </div>
          <transition name="fade-slide">
            <el-form-item prop="custom_prompt" v-if="showPrompt">
              <el-input
                v-if="promptTab === 'edit'"
                v-model="form.custom_prompt"
                type="textarea"
                :rows="10"
                class="code-textarea"
                placeholder="描述项目背景、技术栈、审查重点等补充信息，例如：这是一个 Go 后端服务，重点关注并发安全和数据库操作..."
              />
              <div v-else class="prompt-preview" v-html="renderedPrompt || '<span class=\'preview-empty\'>暂无内容</span>'" />
              <div class="field-hint">将追加到模型全局 Prompt 之后，作为本项目的补充说明。</div>
            </el-form-item>
          </transition>
        </section>

        <!-- 操作栏 -->
        <div class="form-actions">
          <el-button size="large" @click="router.push('/projects')">取消</el-button>
          <el-button size="large" type="primary" :loading="submitting" @click="handleSubmit" class="submit-btn">
            {{ isClone ? '克隆项目' : '创建项目' }}
          </el-button>
        </div>
      </el-form>

      <!-- 右侧提示 -->
      <aside class="side-tips">
        <div class="tip-card">
          <div class="tip-icon">📦</div>
          <h4>仓库要求</h4>
          <ul>
            <li>支持 HTTP(S) 协议的 Git 仓库</li>
            <li>私有仓库需配置凭据</li>
            <li>首次创建后自动克隆仓库到本地</li>
          </ul>
        </div>
        <div class="tip-card">
          <div class="tip-icon">🔁</div>
          <h4>触发方式</h4>
          <ul>
            <li><strong>Webhook</strong>：Push 后自动触发审查</li>
            <li><strong>手动</strong>：在项目详情页选择 commit 范围触发</li>
          </ul>
        </div>
        <div class="tip-card accent">
          <div class="tip-icon">✨</div>
          <h4>Prompt 叠加机制</h4>
          <p>模型全局 Prompt + 项目自定义 Prompt 共同生效</p>
          <p style="margin-top:6px">全局 Prompt 定义通用审查规则，项目 Prompt 补充项目背景和专项要求。</p>
        </div>
      </aside>
    </div>
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
import { ElMessage } from 'element-plus'
import { ArrowLeft, ArrowDown, EditPen, Link, Key, Share, Select, FolderAdd, FullScreen } from '@element-plus/icons-vue'
import { marked } from 'marked'
import { listModels, listCredentials, getProject, createProject } from '../../api/projects'

const router = useRouter()
const route = useRoute()

const cloneFrom = computed(() => route.query.clone_from || null)
const isClone = computed(() => !!cloneFrom.value)

const formRef = ref(null)
const submitting = ref(false)
const initLoading = ref(false)
const showPrompt = ref(false)
const models = ref([])
const credentials = ref([])
const promptTab = ref('preview')
const fsTab = ref('preview')
const promptFullscreen = ref(false)
const renderedPrompt = computed(() => form.value.custom_prompt ? marked.parse(form.value.custom_prompt) : '')

const form = ref({
  name: '', repo_url: '', repo_credential_id: null,
  branch: 'main', model_config_id: null,
  custom_prompt: '', overflow_strategy: 'queue', task_timeout: 30,
})

const rules = {
  name:            [{ required: true, message: '请输入项目名称', trigger: 'blur' }],
  repo_url:        [{ required: true, message: '请输入仓库地址', trigger: 'blur' }],
  branch:          [{ required: true, message: '请输入分支名',   trigger: 'blur' }],
  model_config_id: [{ required: true, message: '请选择模型配置', trigger: 'change' }],
}

onMounted(async () => {
  initLoading.value = true
  try {
    const [modelsRes, credsRes] = await Promise.all([listModels(), listCredentials()])
    models.value = modelsRes.data
    credentials.value = credsRes.data
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载选项失败')
  } finally {
    initLoading.value = false
  }

  if (cloneFrom.value) {
    try {
      const res = await getProject(cloneFrom.value)
      const p = res.data.project
      form.value = {
        name: p.name + '-copy',
        repo_url: p.repo_url,
        repo_credential_id: p.repo_credential_id,
        branch: p.branch,
        model_config_id: p.model_config_id,
        custom_prompt: p.custom_prompt,
        overflow_strategy: p.overflow_strategy,
        task_timeout: p.task_timeout,
      }
      if (p.custom_prompt) showPrompt.value = true
    } catch (err) {
      ElMessage.error(err.response?.data?.message || '加载克隆源失败')
    }
  }
})

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    await createProject(form.value)
    ElMessage.success(isClone.value ? '克隆成功' : '项目创建成功')
    router.push('/projects')
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.page-wrap { padding: 32px 36px; min-height: 100%; }

.back-btn { color: #6b7280; margin-bottom: 20px; padding: 0; font-size: 13px; }
.back-btn:hover { color: #2563eb; }

/* 页头 */
.page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 28px;
}

.header-icon {
  width: 52px; height: 52px;
  border-radius: 14px;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 4px 16px rgba(37,99,235,0.25);
}

.page-header h1 { font-size: 22px; font-weight: 700; color: #111827; margin: 0 0 4px; }
.page-header p  { font-size: 13px; color: #6b7280; margin: 0; }

/* 布局 */
.form-body { display: flex; gap: 28px; align-items: flex-start; }
.main-form { flex: 1; min-width: 0; }
.side-tips { width: 240px; flex-shrink: 0; position: sticky; top: 24px; display: flex; flex-direction: column; gap: 12px; }

/* 分区卡片 */
.form-section {
  background: #fff;
  border-radius: 12px;
  padding: 22px 24px;
  margin-bottom: 14px;
  border: 1px solid #e2e8f0;
}

.section-label {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  font-weight: 600;
  color: #374151;
  margin-bottom: 18px;
}

.section-label.collapsible { cursor: pointer; margin-bottom: 0; user-select: none; }
.section-label.collapsible:hover { color: #2563eb; }

.collapse-icon { margin-left: auto; transition: transform 0.25s; color: #9ca3af; }

.section-num {
  width: 22px; height: 22px;
  border-radius: 50%;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

/* 字段网格 */
.field-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0 20px; }
.field-grid :deep(.el-form-item) { margin-bottom: 16px; }
.span-full { grid-column: 1 / -1; }

.field-hint { font-size: 12px; color: #9ca3af; margin-top: 5px; display: flex; align-items: center; gap: 2px; }

/* 凭据下拉选项 */
.cred-option { display: flex; align-items: center; gap: 8px; }
.cred-user { margin-left: auto; font-size: 12px; color: #9ca3af; }

/* 模型卡片网格 */
.model-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 10px; }

.model-card {
  position: relative;
  padding: 14px 16px;
  border-radius: 10px;
  border: 2px solid #f0f0f0;
  background: #fafafa;
  cursor: pointer;
  transition: all 0.18s;
}

.model-card:hover { border-color: #93c5fd; background: #eff6ff; }
.model-card.active {
  border-color: #2563eb;
  background: linear-gradient(135deg, #eff6ff, #f0f0ff);
  box-shadow: 0 0 0 3px rgba(37,99,235,0.1);
}

.model-card-inner { display: flex; flex-direction: column; gap: 4px; }
.model-type-badge {
  display: inline-block;
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  background: #e5e7eb;
  color: #6b7280;
  font-weight: 600;
  width: fit-content;
  text-transform: uppercase;
}
.model-name { font-size: 13px; font-weight: 600; color: #111827; }
.model-model { font-size: 11px; color: #9ca3af; font-family: monospace; }
.model-check { position: absolute; top: 8px; right: 8px; }

/* 策略卡片 */
.strategy-cards { display: flex; gap: 12px; }

.strategy-card {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border-radius: 10px;
  border: 2px solid #f0f0f0;
  background: #fafafa;
  cursor: pointer;
  transition: all 0.18s;
}

.strategy-card:hover { border-color: #93c5fd; background: #eff6ff; }
.strategy-card.active {
  border-color: #2563eb;
  background: linear-gradient(135deg, #eff6ff, #f0f0ff);
  box-shadow: 0 0 0 3px rgba(37,99,235,0.1);
}

.strategy-icon { font-size: 22px; flex-shrink: 0; }
.strategy-info { flex: 1; }
.strategy-info strong { display: block; font-size: 13px; color: #111827; margin-bottom: 2px; }
.strategy-info span  { font-size: 12px; color: #6b7280; }

/* 超时行 */
.timeout-row { display: flex; align-items: center; gap: 14px; }
.timeout-presets { display: flex; gap: 6px; }
.preset-tag { cursor: pointer; user-select: none; }

/* 代码 textarea */
.code-textarea :deep(textarea) {
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 8px;
  border-color: #1e293b;
  resize: vertical;
}
.code-textarea :deep(textarea)::placeholder { color: #475569; }

/* 操作栏 */
.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  background: #fff;
  border-radius: 12px;
  padding: 16px 24px;
  border: 1px solid #e2e8f0;
}

.submit-btn {
  background: linear-gradient(90deg, #2563eb, #7c3aed) !important;
  border: none !important;
  padding: 0 28px;
}
.submit-btn:hover { opacity: 0.9; }

/* 右侧提示卡 */
.tip-card {
  background: #fff;
  border-radius: 12px;
  padding: 16px 18px;
  border: 1px solid #e2e8f0;
}

.tip-card.accent {
  background: linear-gradient(135deg, #eff6ff, #f5f3ff);
  border: 1px solid #dbeafe;
}

.tip-icon { font-size: 18px; margin-bottom: 7px; }
.tip-card h4 { font-size: 13px; font-weight: 600; color: #374151; margin: 0 0 8px; }
.tip-card p, .tip-card li { font-size: 12px; color: #6b7280; line-height: 1.6; margin: 0; }
.tip-card ul { padding-left: 15px; margin: 0; display: flex; flex-direction: column; gap: 5px; }

/* Prompt tabs */
.prompt-tabs {
  display: flex;
  background: #f1f5f9;
  border-radius: 6px;
  padding: 2px;
  gap: 2px;
  margin-left: 8px;
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

.prompt-fs-btn { margin-left: 4px; color: #6b7280; font-size: 12px; }
.prompt-fs-btn:hover { color: #2563eb; }

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


.fade-slide-enter-active, .fade-slide-leave-active { transition: all 0.22s ease; overflow: hidden; }
.fade-slide-enter-from { opacity: 0; transform: translateY(-6px); }
.fade-slide-leave-to   { opacity: 0; transform: translateY(-4px); }
</style>
