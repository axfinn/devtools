<template>
  <div class="console-tool">
    <!-- 登录 -->
    <div v-if="!isAdmin" class="login-card">
      <el-card>
        <template #header><span>控制台 — 登录</span></template>
        <el-form @submit.prevent="login">
          <el-form-item label="管理员密码">
            <el-input v-model="passwordInput" type="password" show-password
              placeholder="请输入管理员密码" @keyup.enter="login" />
          </el-form-item>
          <el-button type="primary" @click="login" :loading="loginLoading">登录</el-button>
        </el-form>
      </el-card>
    </div>

    <!-- 主界面 -->
    <div v-else class="main-content">
      <el-card>
        <template #header>
          <div class="card-header">
            <span>导航模块显隐</span>
            <div>
              <el-button type="primary" size="small" :loading="saving" @click="save">保存</el-button>
              <el-button size="small" @click="logout">退出</el-button>
            </div>
          </div>
        </template>
        <div class="hint">勾选后该模块将从侧边栏隐藏（路由仍可直接访问）。控制台本身不可隐藏。</div>
        <div v-for="(routes, category) in groupedRoutes" :key="category" class="category-group">
          <div class="category-title">{{ categoryNames[category] || category }}</div>
          <div v-for="route in routes" :key="route.path" class="route-item">
            <el-checkbox v-model="hiddenSet[route.path]">
              {{ route.meta.title }}
              <span class="route-path">{{ route.path }}</span>
            </el-checkbox>
          </div>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

const SESSION_KEY = 'console_admin_pass'

const router = useRouter()
const passwordInput = ref('')
const isAdmin = ref(false)
const loginLoading = ref(false)
const saving = ref(false)
const hiddenSet = reactive({})

const categoryNames = {
  dev: '开发工具', convert: '转换工具', draw: '绘图/图表',
  collab: '协作分享', life: '生活工具', other: '其他工具'
}

// 所有可隐藏的路由（排除 /console 和带参数的路由）
const allRoutes = router.options.routes.filter(r =>
  r.meta?.title && !r.path.includes(':') && r.meta?.category && r.meta.category !== 'home' && r.path !== '/console'
)

const groupedRoutes = allRoutes.reduce((acc, r) => {
  const cat = r.meta.category || 'other'
  if (!acc[cat]) acc[cat] = []
  acc[cat].push(r)
  return acc
}, {})

async function login() {
  if (!passwordInput.value) { ElMessage.warning('请输入密码'); return }
  loginLoading.value = true
  try {
    const res = await fetch(`/api/console/verify?admin_password=${encodeURIComponent(passwordInput.value)}`, {
      method: 'POST'
    })
    if (res.status === 401) {
      ElMessage.error('密码错误')
      return
    }
    sessionStorage.setItem(SESSION_KEY, passwordInput.value)
    isAdmin.value = true
    await loadSettings()
  } catch {
    ElMessage.error('请求失败')
  } finally {
    loginLoading.value = false
  }
}

async function loadSettings() {
  const res = await fetch('/api/console/settings')
  const data = await res.json()
  const hidden = data.hidden_routes || []
  allRoutes.forEach(r => { hiddenSet[r.path] = hidden.includes(r.path) })
}

async function save() {
  saving.value = true
  const pass = sessionStorage.getItem(SESSION_KEY)
  const hidden = Object.keys(hiddenSet).filter(k => hiddenSet[k])
  try {
    const res = await fetch(`/api/console/settings?admin_password=${encodeURIComponent(pass)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ hidden_routes: hidden })
    })
    if (!res.ok) { ElMessage.error('保存失败'); return }
    ElMessage.success('已保存，刷新页面生效')
  } catch {
    ElMessage.error('请求失败')
  } finally {
    saving.value = false
  }
}

function logout() {
  sessionStorage.removeItem(SESSION_KEY)
  isAdmin.value = false
  passwordInput.value = ''
}

onMounted(() => {
  const saved = sessionStorage.getItem(SESSION_KEY)
  if (saved) {
    passwordInput.value = saved
    login()
  }
})
</script>

<style scoped>
.console-tool { padding: 20px; max-width: 700px; margin: 0 auto; }
.login-card { max-width: 400px; margin: 60px auto; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.hint { font-size: 12px; color: #999; margin-bottom: 16px; }
.category-group { margin-bottom: 16px; }
.category-title { font-weight: bold; color: #606266; margin-bottom: 8px; font-size: 13px; }
.route-item { padding: 4px 0; }
.route-path { font-size: 11px; color: #aaa; margin-left: 6px; }
</style>
