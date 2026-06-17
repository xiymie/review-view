<template>
  <el-container class="app-layout">
    <el-aside width="216px" class="sidebar">
      <!-- Brand -->
      <div class="brand">
        <div class="brand-icon">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
          </svg>
        </div>
        <span class="brand-name">代码审核平台</span>
      </div>

      <!-- Nav -->
      <nav class="nav">
        <router-link class="nav-item" :class="{ active: activeMenu === '/home' }" to="/home">
          <el-icon><DataBoard /></el-icon><span>仪表盘</span>
        </router-link>
        <router-link class="nav-item" :class="{ active: activeMenu === '/projects' }" to="/projects">
          <el-icon><FolderOpened /></el-icon><span>项目</span>
        </router-link>
        <router-link class="nav-item" :class="{ active: activeMenu === '/tasks' }" to="/tasks">
          <el-icon><List /></el-icon><span>任务</span>
        </router-link>
        <!-- 所有用户可见 -->
        <div class="nav-group-label">我的配置</div>
        <router-link class="nav-item" :class="{ active: activeMenu === '/scan-schedules' }" to="/scan-schedules">
          <el-icon><Timer /></el-icon><span>定时扫描</span>
        </router-link>
        <router-link class="nav-item" :class="{ active: activeMenu === '/credentials' }" to="/credentials">
          <el-icon><Key /></el-icon><span>仓库凭据</span>
        </router-link>
        <router-link class="nav-item" :class="{ active: activeMenu === '/notify' }" to="/notify">
          <el-icon><Bell /></el-icon><span>推送通知</span>
        </router-link>

        <!-- 仅管理员可见 -->
        <template v-if="isAdmin">
          <div class="nav-group-label">管理</div>
          <router-link class="nav-item" :class="{ active: activeMenu === '/users' }" to="/users">
            <el-icon><UserFilled /></el-icon><span>用户管理</span>
          </router-link>
          <router-link class="nav-item" :class="{ active: activeMenu === '/models' }" to="/models">
            <el-icon><Cpu /></el-icon><span>模型配置</span>
          </router-link>
          <router-link class="nav-item" :class="{ active: activeMenu === '/sensitive-words' }" to="/sensitive-words">
            <el-icon><Filter /></el-icon><span>敏感词管理</span>
          </router-link>
          <router-link class="nav-item" :class="{ active: activeMenu === '/settings' }" to="/settings">
            <el-icon><Setting /></el-icon><span>设置</span>
          </router-link>
        </template>
      </nav>

      <!-- Docs link -->
      <router-link class="docs-link" :class="{ active: activeMenu === '/docs' }" to="/docs">
        <el-icon><Document /></el-icon><span>使用文档</span>
      </router-link>

      <!-- Footer -->
      <div class="sidebar-footer">
        <el-avatar :size="28" class="avatar">{{ username[0]?.toUpperCase() }}</el-avatar>
        <el-tooltip content="个人资料" placement="top">
          <span class="username username-clickable" @click="openProfile">{{ username }}</span>
        </el-tooltip>
        <el-tooltip content="退出登录" placement="top">
          <el-icon class="logout-icon" @click="logout"><SwitchButton /></el-icon>
        </el-tooltip>
      </div>
    </el-aside>

    <el-main class="main-content">
      <router-view />
    </el-main>
  </el-container>

  <!-- Profile dialog -->
  <el-dialog v-model="profileVisible" title="个人资料" width="520px" :close-on-click-modal="false">
    <el-form :model="profileForm" :rules="profileRules" ref="profileFormRef" label-position="top">
      <el-form-item label="用户名">
        <el-input :value="username" disabled />
      </el-form-item>
      <el-form-item label="邮箱" prop="email">
        <el-input v-model="profileForm.email" placeholder="example@company.com" clearable />
      </el-form-item>
      <el-form-item label="手机号" prop="phone">
        <el-input v-model="profileForm.phone" placeholder="13800138000" clearable />
      </el-form-item>
      <el-form-item label="岗位" prop="position">
        <el-input v-model="profileForm.position" placeholder="例如：后端工程师" clearable />
      </el-form-item>
      <el-form-item label="备注" prop="remark">
        <el-input v-model="profileForm.remark" placeholder="备注信息" clearable />
      </el-form-item>

      <el-divider>修改密码（可选）</el-divider>

      <el-form-item label="原密码" prop="old_password">
        <el-input v-model="profileForm.old_password" type="password" show-password placeholder="留空则不修改密码" clearable />
      </el-form-item>
      <el-form-item label="新密码" prop="new_password">
        <el-input v-model="profileForm.new_password" type="password" show-password placeholder="留空则不修改密码" clearable />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="profileVisible = false">取消</el-button>
      <el-button type="primary" :loading="profileSaving" @click="saveProfile">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  DataBoard, FolderOpened, Cpu, Key,
  List, Setting, SwitchButton, Filter, UserFilled, Document, Timer, Bell,
} from '@element-plus/icons-vue'
import { getMe, updateMe } from '../api/users'

const route = useRoute()
const router = useRouter()
const activeMenu = computed(() => '/' + route.path.split('/')[1])
const username = localStorage.getItem('username') || 'admin'
const isAdmin = computed(() => {
  const role = localStorage.getItem('role') || ''
  return role === 'admin' || role === 'super_admin'
})

const logout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  localStorage.removeItem('role')
  router.push('/login')
}

// Profile dialog
const profileVisible = ref(false)
const profileSaving = ref(false)
const profileFormRef = ref(null)
const profileForm = ref({
  email: '', phone: '', position: '', remark: '',
  old_password: '', new_password: '',
})

const profileRules = {
  new_password: [
    {
      validator: (_, val, cb) => {
        if (val && !profileForm.value.old_password) {
          cb(new Error('请输入原密码'))
        } else {
          cb()
        }
      },
      trigger: 'blur',
    },
  ],
}

async function openProfile() {
  try {
    const res = await getMe()
    const u = res.data
    profileForm.value = {
      email: u.email || '',
      phone: u.phone || '',
      position: u.position || '',
      remark: u.remark || '',
      old_password: '',
      new_password: '',
    }
    profileVisible.value = true
  } catch {
    profileVisible.value = true
  }
}

async function saveProfile() {
  const valid = await profileFormRef.value?.validate().catch(() => false)
  if (!valid) return
  profileSaving.value = true
  try {
    await updateMe(profileForm.value)
    ElMessage.success('保存成功')
    profileVisible.value = false
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  } finally {
    profileSaving.value = false
  }
}
</script>

<style scoped>
.app-layout {
  height: 100vh;
  background: #f8fafc;
}

/* ── Sidebar ── */
.sidebar {
  background: linear-gradient(180deg, #0f172a 0%, #1e1b4b 100%);
  border-right: none;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  overflow: hidden;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 18px 16px 16px;
  border-bottom: 1px solid rgba(255,255,255,0.06);
}

.brand-icon {
  width: 30px;
  height: 30px;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(37,99,235,0.4);
}

.brand-name {
  font-size: 14.5px;
  font-weight: 700;
  color: #f1f5f9;
  letter-spacing: -0.2px;
}

/* ── Nav ── */
.nav {
  flex: 1;
  padding: 8px 8px;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 0 10px;
  height: 38px;
  border-radius: 7px;
  font-size: 13.5px;
  color: rgba(148,163,184,0.9);
  text-decoration: none;
  transition: background 0.15s, color 0.15s;
  margin-bottom: 1px;
  position: relative;
}

.nav-item:hover {
  background: rgba(255,255,255,0.08);
  color: #fff;
}

.nav-item.active {
  background: rgba(37,99,235,0.25);
  color: #fff;
  font-weight: 500;
}

.nav-item.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 7px;
  bottom: 7px;
  width: 3px;
  background: linear-gradient(180deg, #2563eb, #7c3aed);
  border-radius: 0 3px 3px 0;
}

.nav-item .el-icon {
  font-size: 15px;
  flex-shrink: 0;
}

.nav-group-label {
  font-size: 11px;
  font-weight: 600;
  color: rgba(71,85,105,0.8);
  letter-spacing: 0.08em;
  text-transform: uppercase;
  padding: 14px 10px 5px;
}

/* ── Docs link ── */
.docs-link {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 0 18px;
  height: 38px;
  font-size: 13.5px;
  color: rgba(148,163,184,0.9);
  text-decoration: none;
  border-top: 1px solid rgba(255,255,255,0.06);
  flex-shrink: 0;
  transition: background 0.15s, color 0.15s;
}
.docs-link:hover { background: rgba(255,255,255,0.08); color: #fff; }
.docs-link.active { color: #fff; background: rgba(37,99,235,0.2); }
.docs-link .el-icon { font-size: 15px; flex-shrink: 0; }

/* ── Footer ── */
.sidebar-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 14px;
  border-top: 1px solid rgba(255,255,255,0.08);
}

.avatar {
  background: #2563eb;
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  flex-shrink: 0;
}

.username {
  flex: 1;
  color: #94a3b8;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.username-clickable {
  cursor: pointer;
  transition: color 0.2s;
}
.username-clickable:hover { color: #e2e8f0; }

.logout-icon {
  color: #475569;
  font-size: 16px;
  cursor: pointer;
  transition: color 0.2s;
}
.logout-icon:hover { color: #ef4444; }

/* ── Main ── */
.main-content {
  overflow-y: auto;
  padding: 0;
  background: #f8fafc;
}
</style>
