<template>
  <div class="page">
    <!-- Hero -->
    <div class="hero">
      <div class="hero-content">
        <div class="hero-title">定时扫描</div>
        <div class="hero-subtitle">为每个项目独立配置 Cron 表达式，到期自动触发 Code Review 任务</div>
      </div>
      <el-button @click="loadProjects" :loading="loading" size="small" class="refresh-btn">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <!-- Table -->
    <div class="table-card">
      <el-table :data="projects" v-loading="loading" row-key="id" stripe>
        <el-table-column label="项目名称" min-width="160">
          <template #default="{ row }">
            <router-link :to="`/projects/${row.id}`" class="project-link">{{ row.name }}</router-link>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <span v-if="row.cron_enabled" class="pill pill-enabled">已启用</span>
            <span v-else class="pill pill-disabled">未配置</span>
          </template>
        </el-table-column>

        <el-table-column label="Cron 表达式" min-width="150">
          <template #default="{ row }">
            <code v-if="row.cron_expression" class="cron-code">{{ row.cron_expression }}</code>
            <span v-else class="empty-val">—</span>
          </template>
        </el-table-column>

        <el-table-column label="下次执行时间" min-width="170">
          <template #default="{ row }">
            <span v-if="row.next_run_at && row.cron_enabled" class="next-run">
              {{ formatTime(row.next_run_at) }}
            </span>
            <span v-else class="empty-val">—</span>
          </template>
        </el-table-column>

        <el-table-column label="项目状态" width="110">
          <template #default="{ row }">
            <span :class="['pill', statusClass(row.status)]">{{ row.status }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openDrawer(row)">配置</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Config Drawer -->
    <el-drawer
      v-model="drawerVisible"
      :title="`定时扫描配置 · ${editingProject?.name}`"
      direction="rtl"
      size="420px"
      :close-on-click-modal="false"
    >
      <div class="drawer-body" v-if="editingProject">
        <!-- Enable switch -->
        <div class="field-group">
          <div class="field-label">启用定时扫描</div>
          <el-switch v-model="form.cron_enabled" active-text="开启" inactive-text="关闭" />
        </div>

        <template v-if="form.cron_enabled">
          <!-- Presets -->
          <div class="field-group">
            <div class="field-label">快捷预设</div>
            <div class="preset-tags">
              <span
                v-for="p in presets"
                :key="p.label"
                :class="['preset-tag', { active: form.cron_expression === p.expr && !isCustom }]"
                @click="applyPreset(p.expr)"
              >{{ p.label }}</span>
              <span
                :class="['preset-tag', { active: isCustom }]"
                @click="setCustom"
              >自定义</span>
            </div>
          </div>

          <!-- Cron expression input -->
          <div class="field-group">
            <div class="field-label">
              Cron 表达式
              <span class="field-hint">5 位标准格式：分 时 日 月 周</span>
            </div>
            <el-input
              v-model="form.cron_expression"
              placeholder="例：0 2 * * *"
              clearable
              @input="isCustom = true"
            />
            <div class="cron-desc" v-if="form.cron_expression">{{ cronDesc }}</div>
          </div>

          <!-- Next run preview -->
          <div class="next-run-preview" v-if="editingProject.next_run_at && !isDirty">
            <el-icon><Timer /></el-icon>
            当前下次执行：{{ formatTime(editingProject.next_run_at) }}
          </div>
          <div class="next-run-preview dirty" v-else-if="isDirty">
            <el-icon><InfoFilled /></el-icon>
            保存后将重新计算下次执行时间
          </div>
        </template>

        <div v-if="!form.cron_enabled" class="disable-hint">
          关闭后，该项目将不再自动触发扫描任务
        </div>
      </div>

      <template #footer>
        <div class="drawer-footer">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="save">保存</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Timer, InfoFilled } from '@element-plus/icons-vue'
import { listProjects, updateProjectSchedule } from '../../api/projects'

const projects = ref([])
const loading = ref(false)
const drawerVisible = ref(false)
const saving = ref(false)
const editingProject = ref(null)
const isCustom = ref(false)

const form = ref({
  cron_enabled: false,
  cron_expression: '',
})

const presets = [
  { label: '每小时',       expr: '0 * * * *'   },
  { label: '每天 02:00',   expr: '0 2 * * *'   },
  { label: '每6小时',      expr: '0 */6 * * *' },
  { label: '每周一 02:00', expr: '0 2 * * 1'   },
]

const isDirty = computed(() => {
  if (!editingProject.value) return false
  return (
    form.value.cron_expression !== editingProject.value.cron_expression ||
    form.value.cron_enabled !== editingProject.value.cron_enabled
  )
})

const cronDesc = computed(() => {
  const expr = form.value.cron_expression.trim()
  if (!expr) return ''
  const parts = expr.split(/\s+/)
  if (parts.length !== 5) return '格式不正确，需要 5 位（分 时 日 月 周）'
  const [min, hour, dom, month, dow] = parts
  if (dom === '*' && month === '*' && dow === '*') {
    if (min === '0' && hour === '*') return '每小时整点触发'
    if (min === '0' && hour !== '*') {
      const h = hour.startsWith('*/') ? `每 ${hour.slice(2)} 小时` : `每天 ${hour.padStart(2,'0')}:00`
      return h + ' 触发'
    }
  }
  if (dom === '*' && month === '*' && dow !== '*') {
    const days = ['周日','周一','周二','周三','周四','周五','周六']
    const d = days[parseInt(dow)] || `星期${dow}`
    return `每${d} ${hour.padStart(2,'0')}:${min.padStart(2,'0')} 触发`
  }
  return expr
})

async function loadProjects() {
  loading.value = true
  try {
    const res = await listProjects()
    projects.value = res.data
  } catch {
    ElMessage.error('加载项目列表失败')
  } finally {
    loading.value = false
  }
}

function openDrawer(project) {
  editingProject.value = project
  form.value = {
    cron_enabled: project.cron_enabled || false,
    cron_expression: project.cron_expression || '',
  }
  isCustom.value = !presets.some(p => p.expr === form.value.cron_expression)
  drawerVisible.value = true
}

function applyPreset(expr) {
  form.value.cron_expression = expr
  isCustom.value = false
}

function setCustom() {
  isCustom.value = true
}

async function save() {
  if (form.value.cron_enabled && !form.value.cron_expression.trim()) {
    ElMessage.warning('请填写 Cron 表达式')
    return
  }
  saving.value = true
  try {
    const res = await updateProjectSchedule(editingProject.value.id, {
      cron_expression: form.value.cron_expression.trim(),
      cron_enabled: form.value.cron_enabled,
    })
    // 更新本地列表中对应项目的 cron 数据
    const idx = projects.value.findIndex(p => p.id === editingProject.value.id)
    if (idx !== -1) {
      projects.value[idx] = {
        ...projects.value[idx],
        cron_expression: res.data.cron_expression,
        cron_enabled: res.data.cron_enabled,
        next_run_at: res.data.next_run_at,
      }
      editingProject.value = projects.value[idx]
    }
    ElMessage.success('保存成功')
    drawerVisible.value = false
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function formatTime(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function statusClass(status) {
  const map = {
    ready: 'pill-ready',
    initializing: 'pill-initializing',
    init_failed: 'pill-init-failed',
  }
  return map[status] || 'pill-default'
}

onMounted(loadProjects)
</script>

<style scoped>
.page {
  min-height: 100vh;
  background: #f8fafc;
}

/* Hero */
.hero {
  background: linear-gradient(135deg, #059669 0%, #0891b2 100%);
  padding: 28px 36px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.hero-title {
  font-size: 22px;
  font-weight: 700;
  color: #fff;
  margin-bottom: 4px;
}
.hero-subtitle {
  font-size: 13px;
  color: rgba(255,255,255,0.75);
}
.refresh-btn {
  background: rgba(255,255,255,0.15);
  border: 1px solid rgba(255,255,255,0.3);
  color: #fff;
}
.refresh-btn:hover {
  background: rgba(255,255,255,0.25);
  border-color: rgba(255,255,255,0.5);
  color: #fff;
}

/* Table card */
.table-card {
  margin: 24px 36px;
  background: #fff;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  overflow: hidden;
}

.project-link {
  color: #2563eb;
  text-decoration: none;
  font-weight: 500;
}
.project-link:hover { text-decoration: underline; }

.cron-code {
  font-family: 'SFMono-Regular', 'Menlo', monospace;
  font-size: 12px;
  background: #f1f5f9;
  color: #334155;
  padding: 2px 7px;
  border-radius: 4px;
}

.empty-val { color: #94a3b8; }

.next-run { color: #0f766e; font-size: 13px; }

/* Pills */
.pill {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 99px;
  font-size: 12px;
  font-weight: 500;
}
.pill-enabled    { background: #dcfce7; color: #166534; }
.pill-disabled   { background: #f1f5f9; color: #64748b; }
.pill-ready      { background: #dbeafe; color: #1d4ed8; }
.pill-initializing { background: #fef9c3; color: #854d0e; }
.pill-init-failed  { background: #fee2e2; color: #991b1b; }
.pill-default    { background: #f1f5f9; color: #64748b; }

/* Drawer */
.drawer-body {
  padding: 8px 4px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field-label {
  font-size: 13.5px;
  font-weight: 600;
  color: #374151;
  display: flex;
  align-items: center;
  gap: 8px;
}

.field-hint {
  font-size: 12px;
  font-weight: 400;
  color: #94a3b8;
}

.preset-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.preset-tag {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 99px;
  font-size: 12.5px;
  border: 1px solid #e2e8f0;
  color: #64748b;
  cursor: pointer;
  transition: all 0.15s;
  user-select: none;
}
.preset-tag:hover {
  border-color: #2563eb;
  color: #2563eb;
  background: #eff6ff;
}
.preset-tag.active {
  background: #2563eb;
  border-color: #2563eb;
  color: #fff;
}

.cron-desc {
  font-size: 12px;
  color: #0f766e;
  margin-top: 2px;
}

.next-run-preview {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #0f766e;
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 8px;
  padding: 10px 14px;
}
.next-run-preview.dirty {
  color: #92400e;
  background: #fffbeb;
  border-color: #fde68a;
}

.disable-hint {
  font-size: 13px;
  color: #64748b;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px 14px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
