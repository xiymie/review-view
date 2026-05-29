<template>
  <div class="login-bg" ref="bgRef">
    <canvas ref="canvasRef" class="particles-canvas" />

    <!-- Left decorative panel -->
    <div class="deco-panel">
      <div class="deco-overlay" />
      <div class="light-blob blob1" />
      <div class="light-blob blob2" />
      <div class="light-blob blob3" />

      <div class="deco-content">
        <div class="brand-block">
          <div class="brand-icon">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
            </svg>
          </div>
          <span class="brand-name">Review View</span>
        </div>

        <div class="headline-block">
          <div class="eyebrow">AI-POWERED CODE REVIEW</div>
          <h1 class="headline">让每一行代码<br /><span class="headline-gradient">都经得起审视</span></h1>
          <p class="sub-headline">智能分析代码质量、安全隐患与最佳实践，<br />帮助团队持续交付高质量软件。</p>
        </div>

        <div class="stats-row">
          <div class="stat-item" v-for="(s, i) in stats" :key="i" :style="`--i:${i}`">
            <div class="stat-num">{{ s.num }}</div>
            <div class="stat-label">{{ s.label }}</div>
          </div>
        </div>

        <div class="divider-line" />

        <div class="feature-grid">
          <div class="fg-item" v-for="(f, i) in features" :key="i" :style="`--i:${i}`">
            <div class="fg-icon">
              <svg v-html="f.icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
            </div>
            <div>
              <div class="fg-title">{{ f.title }}</div>
              <div class="fg-desc">{{ f.desc }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Right login panel -->
    <div class="form-panel">
      <!-- Shimmer streaks -->
      <div class="shimmer s1" />
      <div class="shimmer s2" />
      <div class="shimmer s3" />
      <div class="shimmer-orb o1" />
      <div class="shimmer-orb o2" />

      <div class="login-card">
        <div class="login-header">
          <div class="logo-icon">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
            </svg>
          </div>
          <div>
            <div class="logo-title">欢迎回来</div>
            <div class="logo-sub">请登录您的账号继续操作</div>
          </div>
        </div>

        <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="login-form" @submit.prevent @keyup.enter="handleLogin">
          <el-form-item prop="username">
            <div class="input-wrap">
              <el-icon class="input-icon"><User /></el-icon>
              <input v-model="form.username" class="custom-input" placeholder="用户名" autocomplete="off" />
            </div>
          </el-form-item>
          <el-form-item prop="password">
            <div class="input-wrap">
              <el-icon class="input-icon"><Lock /></el-icon>
              <input v-model="form.password" :type="showPwd ? 'text' : 'password'" class="custom-input" placeholder="密码" autocomplete="off" />
              <el-icon class="input-icon-right" @click="showPwd = !showPwd">
                <View v-if="!showPwd" /><Hide v-else />
              </el-icon>
            </div>
          </el-form-item>
          <button type="button" class="login-btn" :class="{ loading }" @click="handleLogin">
            <span class="btn-shine" />
            <span v-if="!loading" class="btn-text">登 录</span>
            <span v-else class="btn-dots"><i /><i /><i /></span>
          </button>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { User, Lock, View, Hide } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const router = useRouter()
const formRef = ref()
const canvasRef = ref()
const loading = ref(false)
const showPwd = ref(false)
const year = new Date().getFullYear()
const form = reactive({ username: '', password: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

const stats = [
  { num: '10x', label: '审查效率提升' },
  { num: '99%', label: '问题识别率' },
  { num: '24/7', label: '持续监控' },
]

const features = [
  {
    title: 'Webhook 自动触发',
    desc: '推送代码即自动发起审查',
    icon: '<path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07A19.5 19.5 0 0 1 4.69 12a19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 3.6 1.28h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L7.91 9a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0 1 22 16.92z"/>',
  },
  {
    title: '多维度分析',
    desc: '安全、性能、规范全面覆盖',
    icon: '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>',
  },
  {
    title: '多模型支持',
    desc: 'OpenAI、Claude、Ollama 等',
    icon: '<circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/>',
  },
  {
    title: '敏感词拦截',
    desc: '自定义规则保护代码安全',
    icon: '<rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>',
  },
]

let animId = null

function initParticles() {
  const canvas = canvasRef.value
  const ctx = canvas.getContext('2d')
  const resize = () => { canvas.width = window.innerWidth; canvas.height = window.innerHeight }
  resize()
  window.addEventListener('resize', resize)

  const particles = Array.from({ length: 80 }, () => ({
    x: Math.random() * window.innerWidth,
    y: Math.random() * window.innerHeight,
    r: Math.random() * 1.8 + 0.4,
    dx: (Math.random() - 0.5) * 0.3,
    dy: (Math.random() - 0.5) * 0.3,
    alpha: Math.random() * 0.45 + 0.1,
  }))

  const draw = () => {
    ctx.clearRect(0, 0, canvas.width, canvas.height)

    // Connecting lines
    for (let i = 0; i < particles.length; i++) {
      for (let j = i + 1; j < particles.length; j++) {
        const a = particles[i], b = particles[j]
        const dist = Math.hypot(a.x - b.x, a.y - b.y)
        if (dist < 120) {
          ctx.beginPath()
          ctx.strokeStyle = `rgba(99,130,240,${0.1 * (1 - dist / 120)})`
          ctx.lineWidth = 0.5
          ctx.moveTo(a.x, a.y); ctx.lineTo(b.x, b.y); ctx.stroke()
        }
      }
    }

    // Particles
    particles.forEach(p => {
      ctx.beginPath()
      ctx.arc(p.x, p.y, p.r, 0, Math.PI * 2)
      ctx.fillStyle = `rgba(59,130,246,${p.alpha})`
      ctx.fill()
      p.x += p.dx; p.y += p.dy
      if (p.x < 0 || p.x > canvas.width) p.dx *= -1
      if (p.y < 0 || p.y > canvas.height) p.dy *= -1
    })

    animId = requestAnimationFrame(draw)
  }
  draw()
}

const handleLogin = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: form.username, password: form.password }),
      })
      const data = await res.json()
      if (!res.ok) { ElMessage.error(data.message || '登录失败'); return }
      localStorage.setItem('token', data.token)
      localStorage.setItem('username', data.username)
      localStorage.setItem('role', data.role || 'user')
      router.push('/home')
    } catch { ElMessage.error('服务器连接失败') }
    finally { loading.value = false }
  })
}

onMounted(initParticles)
onUnmounted(() => cancelAnimationFrame(animId))
</script>

<style scoped>
/* ─── Base ─── */
.login-bg {
  min-height: 100vh;
  background: linear-gradient(140deg, #eef2ff 0%, #f0e9ff 30%, #e0f2fe 60%, #f5f3ff 100%);
  display: flex;
  overflow: hidden;
  position: relative;
}

.particles-canvas {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 0;
}

/* ─── Left deco panel ─── */
.deco-panel {
  flex: 0 0 64%;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1;
  overflow: hidden;
}

.deco-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg,
    rgba(37,99,235,0.04) 0%,
    rgba(124,58,237,0.03) 60%,
    transparent 100%);
}

/* CSS light blobs */
.light-blob {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  pointer-events: none;
}
.blob1 {
  width: 500px; height: 500px;
  background: radial-gradient(circle, rgba(99,102,241,0.22) 0%, transparent 70%);
  top: -160px; left: -60px;
  animation: drift 14s ease-in-out infinite;
}
.blob2 {
  width: 380px; height: 380px;
  background: radial-gradient(circle, rgba(37,99,235,0.18) 0%, transparent 70%);
  bottom: -80px; left: 30%;
  animation: drift 18s ease-in-out infinite reverse;
}
.blob3 {
  width: 300px; height: 300px;
  background: radial-gradient(circle, rgba(168,85,247,0.16) 0%, transparent 70%);
  top: 40%; left: 52%;
  animation: drift 12s ease-in-out infinite;
  animation-delay: -5s;
}

@keyframes drift {
  0%, 100% { transform: translate(0, 0) scale(1); }
  33% { transform: translate(40px, -50px) scale(1.08); }
  66% { transform: translate(-30px, 30px) scale(0.94); }
}

.deco-content {
  position: relative;
  z-index: 2;
  padding: 0 48px;
  max-width: 540px;
  width: 100%;
  text-align: left;
}

/* Brand row */
.brand-block {
  display: flex; align-items: center; gap: 10px;
  margin-bottom: 48px;
  animation: fade-up 0.6s ease both;
}
.brand-icon {
  width: 40px; height: 40px;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  box-shadow: 0 4px 16px rgba(37,99,235,0.3);
}
.brand-name { font-size: 15px; font-weight: 700; color: #1e293b; letter-spacing: 0.3px; }

/* Headline */
.headline-block { margin-bottom: 40px; }

.eyebrow {
  font-size: 11px; font-weight: 700; letter-spacing: 2.5px;
  color: #6366f1; margin-bottom: 14px;
  animation: fade-up 0.6s 0.05s ease both;
}

.headline {
  font-size: 48px; font-weight: 800; line-height: 1.18;
  color: #0f172a; margin: 0 0 18px;
  letter-spacing: -1.5px;
  animation: fade-up 0.65s 0.12s ease both;
}

.headline-gradient {
  background: linear-gradient(135deg, #2563eb 0%, #7c3aed 55%, #06b6d4 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.sub-headline {
  font-size: 15.5px; line-height: 1.75; color: #475569; margin: 0;
  animation: fade-up 0.65s 0.2s ease both;
}

/* Stats */
.stats-row {
  display: flex; gap: 0; margin-bottom: 36px;
  animation: fade-up 0.65s 0.28s ease both;
}
.stat-item {
  flex: 1;
  padding: 16px 0;
  border-right: 1px solid rgba(226,232,240,0.8);
}
.stat-item:last-child { border-right: none; }
.stat-num {
  font-size: 32px; font-weight: 800; color: #0f172a;
  letter-spacing: -1px; line-height: 1;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text;
}
.stat-label { font-size: 12px; color: #94a3b8; margin-top: 5px; font-weight: 500; }

.divider-line {
  height: 1px;
  background: linear-gradient(90deg, rgba(99,102,241,0.25) 0%, rgba(226,232,240,0.4) 100%);
  margin-bottom: 28px;
  animation: fade-up 0.6s 0.35s ease both;
}

/* Feature grid */
.feature-grid {
  display: grid; grid-template-columns: 1fr 1fr; gap: 18px;
}
.fg-item {
  display: flex; align-items: flex-start; gap: 12px;
  animation: fade-up 0.6s calc(0.4s + var(--i) * 0.07s) ease both;
}
.fg-icon {
  width: 34px; height: 34px; flex-shrink: 0;
  border-radius: 9px;
  background: linear-gradient(135deg, rgba(37,99,235,0.1), rgba(124,58,237,0.1));
  border: 1px solid rgba(99,102,241,0.15);
  display: flex; align-items: center; justify-content: center;
  color: #6366f1;
}
.fg-title { font-size: 13.5px; font-weight: 600; color: #1e293b; margin-bottom: 2px; }
.fg-desc { font-size: 12px; color: #94a3b8; line-height: 1.5; }

/* ─── Right form panel ─── */
.form-panel {
  flex: 0 0 36%;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  z-index: 2;
  overflow: hidden;
  background: rgba(255,255,255,0.62);
  backdrop-filter: blur(28px);
  border-left: 1px solid rgba(226,232,240,0.5);
}

/* Shimmer streaks */
.shimmer {
  position: absolute;
  width: 1.5px;
  background: linear-gradient(180deg,
    transparent 0%,
    rgba(99,102,241,0.5) 30%,
    rgba(37,99,235,0.7) 50%,
    rgba(99,102,241,0.5) 70%,
    transparent 100%);
  border-radius: 2px;
  animation: shimmer-fall linear infinite;
  pointer-events: none;
}
.s1 { height: 180px; left: 20%; animation-duration: 4s; animation-delay: 0s; }
.s2 { height: 120px; left: 55%; animation-duration: 5.5s; animation-delay: -2s; }
.s3 { height: 200px; left: 80%; animation-duration: 3.8s; animation-delay: -1.2s; }

@keyframes shimmer-fall {
  0%   { top: -220px; opacity: 0; }
  10%  { opacity: 1; }
  90%  { opacity: 1; }
  100% { top: 110%; opacity: 0; }
}

/* Glowing orbs on the right panel */
.shimmer-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(50px);
  pointer-events: none;
  animation: orb-pulse 6s ease-in-out infinite;
}
.o1 {
  width: 220px; height: 220px;
  background: radial-gradient(circle, rgba(99,102,241,0.2) 0%, transparent 70%);
  top: -60px; right: -40px;
  animation-delay: 0s;
}
.o2 {
  width: 180px; height: 180px;
  background: radial-gradient(circle, rgba(14,165,233,0.18) 0%, transparent 70%);
  bottom: -40px; left: -30px;
  animation-delay: -3s;
}

@keyframes orb-pulse {
  0%, 100% { transform: scale(1); opacity: 0.8; }
  50% { transform: scale(1.2); opacity: 1; }
}

/* Login card */
.login-card {
  position: relative;
  z-index: 2;
  width: 100%;
  max-width: 360px;
  padding: 48px 44px;
  animation: fade-up 0.7s 0.2s ease both;
}

.login-header { display: flex; align-items: center; gap: 12px; margin-bottom: 32px; }

.logo-icon {
  width: 44px; height: 44px;
  background: linear-gradient(135deg, #2563eb, #4f46e5);
  border-radius: 12px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  box-shadow: 0 4px 14px rgba(37,99,235,0.35);
}
.logo-title { font-size: 17px; font-weight: 700; color: #0f172a; letter-spacing: -0.3px; }
.logo-sub { font-size: 12px; color: #94a3b8; margin-top: 3px; }

.login-form :deep(.el-form-item) { margin-bottom: 16px; }
.login-form :deep(.el-form-item__error) { color: #ef4444; padding-top: 4px; }

.input-wrap { position: relative; width: 100%; display: flex; align-items: center; }

.input-icon {
  position: absolute; left: 12px; color: #94a3b8;
  font-size: 15px; pointer-events: none; z-index: 1;
}
.input-icon-right {
  position: absolute; right: 12px; color: #94a3b8;
  font-size: 15px; cursor: pointer; z-index: 1;
}

.custom-input {
  width: 100%; height: 46px; padding: 0 40px;
  background: rgba(248,250,252,0.9);
  border: 1px solid #e2e8f0; border-radius: 10px;
  color: #1e293b; font-size: 14px; outline: none;
  transition: border-color 0.2s, box-shadow 0.2s, background 0.2s;
}
.custom-input::placeholder { color: #cbd5e1; }
.custom-input:focus {
  background: #fff;
  border-color: #6366f1;
  box-shadow: 0 0 0 3px rgba(99,102,241,0.12);
}

/* Login button with shine sweep */
.login-btn {
  position: relative;
  overflow: hidden;
  width: 100%; height: 46px; margin-top: 10px;
  border: none; border-radius: 10px;
  background: linear-gradient(135deg, #2563eb 0%, #6366f1 50%, #7c3aed 100%);
  color: #fff; font-size: 14px; font-weight: 600; letter-spacing: 3px;
  cursor: pointer;
  transition: opacity 0.2s, box-shadow 0.2s, transform 0.1s;
  box-shadow: 0 4px 18px rgba(99,102,241,0.42);
}
.login-btn:hover { opacity: 0.93; box-shadow: 0 6px 24px rgba(99,102,241,0.55); }
.login-btn:active { transform: scale(0.99); }
.login-btn.loading { cursor: not-allowed; opacity: 0.7; }

.btn-text { position: relative; z-index: 1; }

/* Shine sweep on the button */
.btn-shine {
  position: absolute;
  top: 0; left: -100%;
  width: 60%; height: 100%;
  background: linear-gradient(105deg, transparent 40%, rgba(255,255,255,0.28) 50%, transparent 60%);
  animation: btn-shine 3s ease-in-out infinite;
  z-index: 2;
}
@keyframes btn-shine {
  0%   { left: -100%; }
  30%  { left: 130%; }
  100% { left: 130%; }
}

.btn-dots { display: flex; align-items: center; justify-content: center; gap: 5px; position: relative; z-index: 1; }
.btn-dots i {
  display: inline-block; width: 6px; height: 6px; border-radius: 50%;
  background: #fff; animation: dot-bounce 0.9s infinite;
}
.btn-dots i:nth-child(2) { animation-delay: 0.15s; }
.btn-dots i:nth-child(3) { animation-delay: 0.3s; }

@keyframes dot-bounce {
  0%, 80%, 100% { transform: scale(0.7); opacity: 0.5; }
  40% { transform: scale(1.1); opacity: 1; }
}

.footer-text {
  margin-top: 24px; text-align: center;
  color: #cbd5e1; font-size: 12px; letter-spacing: 0.5px;
}

@keyframes fade-up {
  from { opacity: 0; transform: translateY(20px); }
  to   { opacity: 1; transform: translateY(0); }
}
</style>
