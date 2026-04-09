<template>
  <div class="nps-tool">
    <!-- 登录 -->
    <div v-if="!isAdmin" class="login-card">
      <el-card>
        <template #header><span>NPS 端口映射 — 登录</span></template>
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
      <!-- 状态栏 -->
      <el-card class="status-card">
        <template #header>
          <div class="card-header">
            <span>客户端状态</span>
            <el-button size="small" @click="logout">退出</el-button>
          </div>
        </template>
        <div v-if="clientInfo" class="status-row">
          <el-tag :type="clientInfo.IsConnect ? 'success' : 'danger'" size="small">
            {{ clientInfo.IsConnect ? '在线' : '离线' }}
          </el-tag>
          <span class="status-label">{{ clientInfo.Remark || clientInfo.VerifyKey }}</span>
          <span v-if="portRangeStart > 0" class="port-range">
            自动端口区间：{{ portRangeStart }} – {{ portRangeEnd }}
          </span>
          <el-button
            v-if="tunnelPort"
            size="small"
            type="warning"
            :loading="addingProxy"
            @click="addProxyTunnel"
          >一键映射代理端口 ({{ tunnelPort }})</el-button>
          <el-button
            size="small"
            :type="npcRunning ? 'danger' : 'success'"
            :loading="npcLoading"
            @click="toggleNpc"
          >npc {{ npcRunning ? '停止' : '启动' }}</el-button>
        </div>
        <div v-else class="text-gray">加载中...</div>
      </el-card>

      <!-- 添加 tunnel -->
      <el-card class="add-card">
        <template #header><span>添加端口映射</span></template>
        <el-form :model="addForm" inline>
          <el-form-item label="类型">
            <el-select v-model="addForm.type" style="width:90px">
              <el-option label="TCP" value="tcp" />
              <el-option label="UDP" value="udp" />
            </el-select>
          </el-form-item>
          <el-form-item label="服务端端口">
            <el-input-number v-model="addForm.port" :min="1" :max="65535"
              placeholder="留空自动分配" :controls="false" style="width:130px"
              :value-on-clear="null" />
          </el-form-item>
          <el-form-item label="目标地址">
            <el-input v-model="addForm.target" placeholder="192.168.1.1:22" style="width:180px" />
          </el-form-item>
          <el-form-item label="备注">
            <el-input v-model="addForm.remark" placeholder="可选" style="width:120px" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="addTunnel" :loading="adding">添加</el-button>
          </el-form-item>
        </el-form>
        <div v-if="portRangeStart > 0" class="hint">
          端口留空时自动从 {{ portRangeStart }}–{{ portRangeEnd }} 区间分配
        </div>
      </el-card>

      <!-- tunnel 列表 -->
      <el-card class="tunnels-card">
        <template #header>
          <div class="card-header">
            <span>端口映射列表（{{ tunnels.length }} 条）</span>
            <el-button size="small" @click="loadTunnels" :loading="loadingTunnels">刷新</el-button>
          </div>
        </template>
        <el-table :data="tunnels" size="small" v-loading="loadingTunnels">
          <el-table-column label="ID" prop="Id" width="60" />
          <el-table-column label="类型" prop="Mode" width="70" />
          <el-table-column label="服务端端口" prop="Port" width="100" />
          <el-table-column label="目标地址" min-width="160">
            <template #default="{ row }">
              {{ row.Target?.TargetStr || row.TargetAddr || '—' }}
            </template>
          </el-table-column>
          <el-table-column label="备注" prop="Remark" min-width="120" show-overflow-tooltip />
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.RunStatus ? 'success' : 'info'" size="small">
                {{ row.RunStatus ? '运行中' : '已停止' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80">
            <template #default="{ row }">
              <el-button size="small" type="danger"
                @click="confirmDelete(row)" :loading="deletingId === row.Id">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const SESSION_KEY = 'nps_admin_password'

const passwordInput = ref('')
const isAdmin = ref(false)
const loginLoading = ref(false)

const clientInfo = ref(null)
const portRangeStart = ref(0)
const portRangeEnd = ref(0)
const tunnelPort = ref('')  // proxy.tunnel_port，用于一键映射代理端口
const npcRunning = ref(false)
const npcLoading = ref(false)

const tunnels = ref([])
const loadingTunnels = ref(false)
const adding = ref(false)
const addingProxy = ref(false)
const deletingId = ref(null)

const addForm = ref({ type: 'tcp', port: null, target: '', remark: '' })

function adminPassword() {
  return sessionStorage.getItem(SESSION_KEY) || ''
}

function logout() {
  sessionStorage.removeItem(SESSION_KEY)
  isAdmin.value = false
  passwordInput.value = ''
}

async function login() {
  if (!passwordInput.value) { ElMessage.warning('请输入密码'); return }
  loginLoading.value = true
  try {
    const r = await fetch(`/api/nps/status?admin_password=${encodeURIComponent(passwordInput.value)}`)
    const data = await r.json()
    if (data.error) {
      ElMessage.error('密码错误或 NPS 未配置')
    } else {
      sessionStorage.setItem(SESSION_KEY, passwordInput.value)
      isAdmin.value = true
      applyStatus(data)
      loadTunnels()
    }
  } catch { ElMessage.error('请求失败') }
  finally { loginLoading.value = false }
}

function applyStatus(data) {
  clientInfo.value = data.client?.client || data.client
  portRangeStart.value = data.port_range_start || 0
  portRangeEnd.value = data.port_range_end || 0
  tunnelPort.value = data.tunnel_port || ''
}

async function loadStatus() {
  try {
    const r = await fetch(`/api/nps/status?admin_password=${encodeURIComponent(adminPassword())}`)
    const data = await r.json()
    if (!data.error) applyStatus(data)
    // 同步 npc 状态
    const nr = await fetch(`/api/nps/npc/status?admin_password=${encodeURIComponent(adminPassword())}`)
    const nd = await nr.json()
    npcRunning.value = nd.running
  } catch {}
}

async function toggleNpc() {
  npcLoading.value = true
  try {
    const action = npcRunning.value ? 'stop' : 'start'
    await fetch(`/api/nps/npc/${action}?admin_password=${encodeURIComponent(adminPassword())}`, { method: 'POST' })
    await new Promise(r => setTimeout(r, 1000))
    const nr = await fetch(`/api/nps/npc/status?admin_password=${encodeURIComponent(adminPassword())}`)
    const nd = await nr.json()
    npcRunning.value = nd.running
  } catch { ElMessage.error('操作失败') }
  finally { npcLoading.value = false }
}

async function loadTunnels() {
  loadingTunnels.value = true
  try {
    const r = await fetch(`/api/nps/tunnels?admin_password=${encodeURIComponent(adminPassword())}`)
    const data = await r.json()
    if (data.error) { ElMessage.error(data.error); return }
    tunnels.value = data.rows || []
  } catch { ElMessage.error('加载失败') }
  finally { loadingTunnels.value = false }
}

async function addTunnel() {
  if (!addForm.value.target) { ElMessage.warning('请填写目标地址'); return }
  adding.value = true
  try {
    const body = {
      admin_password: adminPassword(),
      type: addForm.value.type,
      target: addForm.value.target,
      remark: addForm.value.remark,
    }
    if (addForm.value.port) body.port = addForm.value.port
    const r = await fetch('/api/nps/tunnels', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    const data = await r.json()
    if (data.error) { ElMessage.error(data.error); return }
    ElMessage.success(`添加成功，端口 ${data.port}`)
    addForm.value = { type: 'tcp', port: null, target: '', remark: '' }
    loadTunnels()
  } catch { ElMessage.error('添加失败') }
  finally { adding.value = false }
}

// 一键将 proxy.tunnel_port 映射到 NPS 公网
async function addProxyTunnel() {
  if (!tunnelPort.value) return
  addingProxy.value = true
  try {
    const body = {
      admin_password: adminPassword(),
      type: 'tcp',
      port: parseInt(tunnelPort.value),
      target: `127.0.0.1:${tunnelPort.value}`,
      remark: '代理端口（防探测）',
    }
    const r = await fetch('/api/nps/tunnels', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    const data = await r.json()
    if (data.error) { ElMessage.error(data.error); return }
    ElMessage.success(`代理端口已映射到公网端口 ${data.port}`)
    loadTunnels()
  } catch { ElMessage.error('映射失败') }
  finally { addingProxy.value = false }
}

async function confirmDelete(row) {
  try {
    await ElMessageBox.confirm(
      `确认删除端口 ${row.Port} → ${row.Target?.TargetStr || row.TargetAddr} 的映射？`,
      '删除确认', { type: 'warning' }
    )
  } catch { return }
  deletingId.value = row.Id
  try {
    const r = await fetch(`/api/nps/tunnels/${row.Id}?admin_password=${encodeURIComponent(adminPassword())}`, {
      method: 'DELETE'
    })
    const data = await r.json()
    if (data.status === 1 || data.msg === 'success') {
      ElMessage.success('已删除')
      loadTunnels()
    } else {
      ElMessage.error(data.msg || '删除失败')
    }
  } catch { ElMessage.error('删除失败') }
  finally { deletingId.value = null }
}

onMounted(() => {
  const saved = sessionStorage.getItem(SESSION_KEY)
  if (!saved) return
  fetch(`/api/nps/status?admin_password=${encodeURIComponent(saved)}`)
    .then(r => r.json())
    .then(data => {
      if (!data.error) {
        isAdmin.value = true
        applyStatus(data)
        loadTunnels()
      } else {
        sessionStorage.removeItem(SESSION_KEY)
      }
    })
    .catch(() => {})
})
</script>

<style scoped>
.nps-tool { max-width: 1000px; margin: 0 auto; padding: 16px; }
.login-card { max-width: 400px; margin: 60px auto; }
.main-content { display: flex; flex-direction: column; gap: 16px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.status-row { display: flex; align-items: center; gap: 12px; }
.status-label { font-size: 14px; color: var(--el-text-color-primary); }
.port-range { font-size: 12px; color: var(--el-text-color-secondary); margin-left: auto; }
.hint { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 6px; }
.text-gray { color: var(--el-text-color-secondary); font-size: 13px; }
</style>
