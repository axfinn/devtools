<template>
  <div class="askit-invite">
    <el-card class="gen-card">
      <template #header>
        <div class="card-header">
          <span>AskIt 云同步 — 邀请码生成</span>
          <el-button size="small" text @click="forgetPassword" v-if="remembered">忘记密码</el-button>
        </div>
      </template>

      <el-form label-width="96px" @submit.prevent="generate">
        <el-form-item label="管理员密码">
          <el-input v-model="adminPassword" type="password" show-password
            placeholder="config.yaml 里的 askit_sync.admin_password"
            @keyup.enter="generate" />
        </el-form-item>
        <el-form-item label="生成数量">
          <el-input-number v-model="count" :min="1" :max="100" :controls="true" style="width:140px" />
          <span class="hint">单次 1–100 个</span>
        </el-form-item>
        <el-form-item label="有效期">
          <el-input-number v-model="expiresDays" :min="0" :max="3650" :controls="true" style="width:140px" />
          <span class="hint">天数，0 表示永不过期</span>
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="remember">记住密码（本机 localStorage）</el-checkbox>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="generate">生成邀请码</el-button>
        </el-form-item>
      </el-form>

      <div class="tip">
        仅在 <code>registration_mode</code> 为 <b>invite</b> 时邀请码才有意义。
        把生成的码发给用户，用户在扩展弹窗「邀请码」栏填入后用邮箱验证码登录即可自动注册。
      </div>
    </el-card>

    <el-card class="users-card">
      <template #header>
        <div class="card-header">
          <span>用户备份概览</span>
          <el-button size="small" :loading="usersLoading" @click="loadUsers">刷新</el-button>
        </div>
      </template>
      <el-table :data="users" size="small" v-loading="usersLoading" empty-text="点「刷新」加载（需管理员密码）">
        <el-table-column label="邮箱" prop="email" min-width="160" show-overflow-tooltip />
        <el-table-column label="注册时间" width="150">
          <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="记录数" prop="recordCount" width="80" align="center" />
        <el-table-column label="最近备份" width="150">
          <template #default="{ row }">{{ fmtTime(row.lastBackupAt) }}</template>
        </el-table-column>
        <el-table-column label="密钥备份" width="160">
          <template #default="{ row }">
            <el-tag v-if="row.hasSecrets" type="success" size="small">已备份</el-tag>
            <el-tag v-else type="info" size="small">无</el-tag>
            <span v-if="row.hasSecrets" class="secrets-time">{{ fmtTime(row.secretsUpdated) }}</span>
          </template>
        </el-table-column>
      </el-table>
      <div class="tip">
        仅展示「有没有备份 / 密钥是否备份」等元数据，不读取任何数据内容。
        密钥为端到端加密密文，服务器也无法解密。
      </div>
    </el-card>

    <el-card v-if="codes.length" class="result-card">
      <template #header>
        <div class="card-header">
          <span>已生成 {{ codes.length }} 个邀请码</span>
          <el-button size="small" @click="copyAll">复制全部</el-button>
        </div>
      </template>
      <el-table :data="rows" size="small">
        <el-table-column label="#" type="index" width="50" />
        <el-table-column label="邀请码" prop="code">
          <template #default="{ row }">
            <span class="code">{{ row.code }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button size="small" text @click="copyOne(row.code)">复制</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="expiry" v-if="lastExpiresDays > 0">有效期 {{ lastExpiresDays }} 天</div>
      <div class="expiry" v-else>永不过期</div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

const SESSION_KEY = 'askit_invite_admin_password'

const adminPassword = ref('')
const count = ref(5)
const expiresDays = ref(7)
const remember = ref(false)
const remembered = ref(false)
const loading = ref(false)
const codes = ref([])
const lastExpiresDays = ref(0)
const users = ref([])
const usersLoading = ref(false)

const rows = computed(() => codes.value.map(code => ({ code })))

function forgetPassword() {
  localStorage.removeItem(SESSION_KEY)
  adminPassword.value = ''
  remember.value = false
  remembered.value = false
}

async function generate() {
  if (!adminPassword.value) { ElMessage.warning('请输入管理员密码'); return }
  loading.value = true
  try {
    const r = await fetch('/api/askit/v1/admin/invites', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Admin-Password': adminPassword.value,
      },
      body: JSON.stringify({ count: count.value, expiresDays: expiresDays.value }),
    })
    const data = await r.json().catch(() => ({}))
    if (!r.ok) {
      const map = {
        admin_disabled: '后端未配置 admin_password，邀请码接口已禁用',
        unauthorized: '管理员密码错误',
      }
      ElMessage.error(map[data.error] || `生成失败（${data.error || r.status}）`)
      return
    }
    codes.value = data.codes || []
    lastExpiresDays.value = expiresDays.value
    if (remember.value) {
      localStorage.setItem(SESSION_KEY, adminPassword.value)
      remembered.value = true
    }
    ElMessage.success(`已生成 ${codes.value.length} 个邀请码`)
  } catch {
    ElMessage.error('请求失败')
  } finally {
    loading.value = false
  }
}

async function copyText(text, okMsg) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(okMsg)
  } catch {
    ElMessage.error('复制失败，请手动选择')
  }
}

function copyOne(code) { copyText(code, '已复制') }
function copyAll() { copyText(codes.value.join('\n'), '已复制全部邀请码') }

function fmtTime(ms) {
  if (!ms) return '—'
  const d = new Date(ms)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

async function loadUsers() {
  if (!adminPassword.value) { ElMessage.warning('请输入管理员密码'); return }
  usersLoading.value = true
  try {
    const r = await fetch('/api/askit/v1/admin/users', {
      headers: { 'X-Admin-Password': adminPassword.value },
    })
    const data = await r.json().catch(() => ({}))
    if (!r.ok) {
      const map = {
        admin_disabled: '后端未配置 admin_password，接口已禁用',
        unauthorized: '管理员密码错误',
      }
      ElMessage.error(map[data.error] || `加载失败（${data.error || r.status}）`)
      return
    }
    users.value = data.users || []
  } catch {
    ElMessage.error('请求失败')
  } finally {
    usersLoading.value = false
  }
}

onMounted(() => {
  const saved = localStorage.getItem(SESSION_KEY)
  if (saved) {
    adminPassword.value = saved
    remember.value = true
    remembered.value = true
    loadUsers()
  }
})
</script>

<style scoped>
.askit-invite { max-width: 720px; margin: 0 auto; padding: 16px; display: flex; flex-direction: column; gap: 16px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.hint { font-size: 12px; color: var(--el-text-color-secondary); margin-left: 10px; }
.tip { font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.6; margin-top: 4px; }
.tip code { background: var(--el-fill-color-light); padding: 1px 5px; border-radius: 4px; }
.code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: 0.5px; }
.expiry { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 8px; }
.secrets-time { font-size: 11px; color: var(--el-text-color-secondary); margin-left: 6px; }
</style>
