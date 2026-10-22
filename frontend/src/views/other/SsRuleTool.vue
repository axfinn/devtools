<template>
  <div class="container mx-auto p-4 max-w-4xl">
    <!-- 密码门 -->
    <el-card v-if="!isAuthenticated" class="max-w-md mx-auto mt-10">
      <template #header>
        <span class="text-lg font-bold">Shadowsocks 规则文件</span>
      </template>
      <div class="space-y-4">
        <el-input
          v-model="passwordInput"
          type="password"
          placeholder="请输入管理密码"
          show-password
          @keyup.enter="login"
        />
        <el-button type="primary" class="w-full" :loading="loggingIn" @click="login">
          进入
        </el-button>
        <p class="text-xs text-gray-400 leading-relaxed">
          若确认密码无误仍提示错误，请检查后端 config.yaml 里的
          <code>ssrules.admin_password</code>：留空时本模块整体禁用。
        </p>
      </div>
    </el-card>

    <template v-else>
      <el-card>
        <template #header>
          <div class="flex items-center justify-between">
            <span class="text-lg font-bold">Shadowsocks 规则文件</span>
            <el-button link type="info" @click="logout">退出</el-button>
          </div>
        </template>

        <div class="space-y-4">
          <div>
            <div class="text-sm text-gray-600 mb-2">
              每行一条，支持裸域名、通配域名、完整 URL、IP / CIDR：
            </div>
            <el-input
              v-model="source"
              type="textarea"
              :rows="12"
              spellcheck="false"
              placeholder="google.com
youtube.com
*.githubusercontent.com
https://www.wikipedia.org/wiki/DevTools
1.1.1.1
10.0.0.0/8"
            />
          </div>

          <div class="flex items-center gap-3">
            <el-button type="primary" :loading="saving" @click="save">生成并保存</el-button>
            <span v-if="updatedAt" class="text-xs text-gray-400">
              最后更新：{{ formattedUpdatedAt }}
            </span>
          </div>

          <el-alert
            v-if="skipped.length"
            type="warning"
            :closable="false"
            show-icon
            title="以下行被跳过了，未写入规则"
          >
            <ul class="text-xs mt-1 space-y-0.5">
              <li v-for="(s, i) in skipped" :key="i">{{ s }}</li>
            </ul>
          </el-alert>
        </div>
      </el-card>

      <!-- 生成结果 -->
      <el-card v-if="content" class="mt-4">
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-bold">生成的规则文件</span>
            <div class="flex gap-2">
              <el-button size="small" @click="copy(content)">
                <el-icon><CopyDocument /></el-icon>
                <span class="ml-1">复制内容</span>
              </el-button>
              <el-button size="small" type="warning" plain @click="regenerateToken">
                <el-icon><Refresh /></el-icon>
                <span class="ml-1">换发链接</span>
              </el-button>
            </div>
          </div>
        </template>

        <pre class="bg-gray-50 border border-gray-200 rounded p-3 text-xs overflow-auto max-h-64">{{ content }}</pre>

        <el-divider content-position="left">
          <span class="text-xs text-gray-500">导入链接（公开，客户端直接填这个）</span>
        </el-divider>

        <div class="space-y-2">
          <div
            v-for="url in urls"
            :key="url"
            class="flex items-center gap-2"
          >
            <el-input :model-value="url" readonly size="small" />
            <el-button size="small" @click="copy(url)">复制</el-button>
          </div>
        </div>

        <p class="text-xs text-gray-400 mt-3 leading-relaxed">
          同一份内容提供多个后缀：不同客户端对规则文件 URL 的后缀要求不一样，内容本身是纯文本。
          链接公开可访问（客户端拉规则时不支持输密码），安全性靠这串随机 token，
          外泄时点「换发链接」即可让旧的立即失效。
        </p>
      </el-card>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CopyDocument, Refresh } from '@element-plus/icons-vue'
import { useAdminAuth } from '../../composables/useAdminAuth'
import { API_BASE } from '../../api'

const {
  isAuthenticated,
  passwordInput,
  loggingIn,
  login,
  logout,
  tryStored,
  authHeader,
} = useAdminAuth({
  storageKey: 'ssrules_admin_password',
  verifyEndpoint: `${API_BASE}/api/ssrules/verify`,
  credentialMode: 'header',
  credentialField: 'X-Super-Admin-Password',
})

const source = ref('')
const content = ref('')
const paths = ref([])
const skipped = ref([])
const updatedAt = ref('')
const saving = ref(false)

// 服务端不知道用户从哪个域名访问（可能走 CF、可能走内网），
// 所以只返回相对路径，origin 由浏览器补。
const urls = computed(() => paths.value.map((p) => `${window.location.origin}${p}`))

const formattedUpdatedAt = computed(() => {
  if (!updatedAt.value) return ''
  const d = new Date(updatedAt.value)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString()
})

function applyView(data) {
  source.value = data.source ?? ''
  content.value = data.content ?? ''
  paths.value = data.paths ?? []
  skipped.value = data.skipped ?? []
  updatedAt.value = data.updated_at ?? ''
}

async function load() {
  try {
    const res = await fetch(`${API_BASE}/api/ssrules`, { headers: authHeader() })
    if (res.status === 401) {
      logout()
      return
    }
    if (!res.ok) throw new Error()
    applyView(await res.json())
  } catch {
    ElMessage.error('读取规则文件失败')
  }
}

async function save() {
  saving.value = true
  try {
    const res = await fetch(`${API_BASE}/api/ssrules`, {
      method: 'POST',
      headers: authHeader({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ source: source.value }),
    })
    if (res.status === 401) {
      logout()
      return
    }
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      ElMessage.error(data.error || '保存失败')
      return
    }
    applyView(data)
    ElMessage.success('已保存')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function regenerateToken() {
  try {
    await ElMessageBox.confirm(
      '换发后所有旧链接立即失效，已导入的客户端需要重新填写新链接。确认继续？',
      '换发链接',
      { type: 'warning', confirmButtonText: '确认换发', cancelButtonText: '取消' },
    )
  } catch {
    return // 用户取消
  }

  try {
    const res = await fetch(`${API_BASE}/api/ssrules/regenerate-token`, {
      method: 'POST',
      headers: authHeader(),
    })
    if (res.status === 401) {
      logout()
      return
    }
    if (!res.ok) throw new Error()
    applyView(await res.json())
    ElMessage.success('已换发，请更新客户端里的链接')
  } catch {
    ElMessage.error('换发失败')
  }
}

function copy(text) {
  navigator.clipboard.writeText(text).then(() => ElMessage.success('已复制'))
}

onMounted(async () => {
  // 本地有凭据就直接进，后端会再验一次；失败自动回滚到登录页
  const ok = await tryStored()
  if (ok) await load()
})
</script>
