<template>
  <div class="page-wrap">
    <div class="page-hero">
      <div class="hero-content">
        <h1 class="hero-title">敏感词管理</h1>
        <p class="hero-sub">发送给上游 LLM 前自动替换，收到响应后自动还原</p>
      </div>
      <div class="hero-actions">
        <el-button type="primary" :icon="Plus" @click="openDialog()">新增</el-button>
      </div>
      <div class="deco-circles">
        <div class="deco c1"></div>
        <div class="deco c2"></div>
      </div>
    </div>

    <el-card shadow="never" class="table-card">
      <el-table :data="words" v-loading="loading" stripe>
        <el-table-column label="原始词" prop="Original" />
        <el-table-column label="替换词" prop="Replacement" />
        <el-table-column label="创建时间" prop="CreatedAt" width="180"
          :formatter="(r) => new Date(r.CreatedAt).toLocaleString()" />
        <el-table-column label="操作" width="140" align="right">
          <template #default="{ row }">
            <el-button size="small" text @click="openDialog(row)">编辑</el-button>
            <el-popconfirm title="确认删除？" @confirm="remove(row.ID)">
              <template #reference>
                <el-button size="small" text type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && words.length === 0" description="暂无敏感词" />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.ID ? '编辑敏感词' : '新增敏感词'" width="420px" @closed="resetForm">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="80px">
        <el-form-item label="原始词" prop="Original">
          <el-input v-model="form.Original" placeholder="发送前被替换的词" />
        </el-form-item>
        <el-form-item label="替换词" prop="Replacement">
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
import { ref, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
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
const form = ref({ ID: null, Original: '', Replacement: '' })

const rules = {
  Original: [{ required: true, message: '请输入原始词', trigger: 'blur' }],
  Replacement: [{ required: true, message: '请输入替换词', trigger: 'blur' }],
}

async function load() {
  loading.value = true
  try {
    const res = await listSensitiveWords()
    words.value = res.data || []
  } finally {
    loading.value = false
  }
}

function openDialog(row = null) {
  form.value = row
    ? { ID: row.ID, Original: row.Original, Replacement: row.Replacement }
    : { ID: null, Original: '', Replacement: '' }
  dialogVisible.value = true
}

function resetForm() {
  formRef.value?.resetFields()
}

async function save() {
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    if (form.value.ID) {
      await updateSensitiveWord(form.value.ID, {
        original: form.value.Original,
        replacement: form.value.Replacement,
      })
    } else {
      await createSensitiveWord({
        original: form.value.Original,
        replacement: form.value.Replacement,
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
  await deleteSensitiveWord(id)
  ElMessage.success('删除成功')
  load()
}

onMounted(load)
</script>

<style scoped>
.page-wrap { padding: 0; }

.page-hero {
  position: relative;
  background: linear-gradient(135deg, #d97706, #dc2626, #7c3aed);
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

.table-card { border-radius: 10px; margin: 20px 36px; }
</style>
