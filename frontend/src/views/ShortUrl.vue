<template>
  <div class="container mx-auto p-4 max-w-4xl">
    <el-card>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="text-xl font-bold">短链生成器</span>
          <el-switch
            v-model="showAdvanced"
            active-text="高级模式"
            inactive-text="快捷模式"
          />
        </div>
      </template>

      <!-- 快捷模式 -->
      <div v-if="!showAdvanced" class="space-y-4">
        <el-input
          v-model="originalUrl"
          placeholder="输入要缩短的 URL（例如：https://example.com）"
          size="large"
          clearable
          @keyup.enter="createShortUrl(720)"
        >
          <template #prefix>
            <el-icon><Link /></el-icon>
          </template>
        </el-input>

        <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
          <el-button
            type="primary"
            size="large"
            @click="createShortUrl(24)"
            :loading="loading"
            class="w-full"
          >
            1 天
          </el-button>
          <el-button
            type="primary"
            size="large"
            @click="createShortUrl(168)"
            :loading="loading"
            class="w-full"
          >
            7 天
          </el-button>
          <el-button
            type="primary"
            size="large"
            @click="createShortUrl(720)"
            :loading="loading"
            class="w-full"
          >
            30 天
          </el-button>
          <el-button
            type="primary"
            size="large"
            @click="createShortUrl(8760)"
            :loading="loading"
            class="w-full"
          >
            1 年
          </el-button>
        </div>
      </div>

      <!-- 高级模式 -->
      <div v-else class="space-y-4">
        <el-form :model="form" label-width="120px">
          <el-form-item label="原始 URL" required>
            <el-input
              v-model="originalUrl"
              placeholder="https://example.com"
              clearable
            >
              <template #prefix>
                <el-icon><Link /></el-icon>
              </template>
            </el-input>
          </el-form-item>

          <el-form-item label="管理密码">
            <el-input
              v-model="password"
              type="password"
              placeholder="如需自定义短链ID，请输入管理密码"
              clearable
              show-password
            >
              <template #prefix>
                <el-icon><Lock /></el-icon>
              </template>
            </el-input>
            <div class="hint-text">
              配置密码后可使用自定义短链ID
            </div>
          </el-form-item>

          <el-form-item label="自定义短链ID">
            <el-input
              v-model="customId"
              placeholder="如: 1, abc, my-link（需要密码）"
              clearable
              :disabled="!password"
            >
              <template #prefix>
                <el-icon><Edit /></el-icon>
              </template>
            </el-input>
            <div class="hint-text">
              只能包含字母、数字、下划线、短横线，最长32字符
            </div>
          </el-form-item>

          <el-form-item label="过期时间">
            <el-select v-model="expiresIn" placeholder="选择过期时间" class="w-full">
              <el-option label="1 小时" :value="1" />
              <el-option label="6 小时" :value="6" />
              <el-option label="12 小时" :value="12" />
              <el-option label="1 天" :value="24" />
              <el-option label="3 天" :value="72" />
              <el-option label="7 天" :value="168" />
              <el-option label="30 天（默认）" :value="720" />
              <el-option label="90 天" :value="2160" />
              <el-option label="180 天" :value="4320" />
              <el-option label="1 年" :value="8760" />
            </el-select>
          </el-form-item>

          <el-form-item label="最大点击次数">
            <el-input-number
              v-model="maxClicks"
              :min="1"
              :max="100000"
              :step="100"
              class="w-full"
            />
            <div class="hint-text">
              默认 1000 次，最多 100000 次
            </div>
          </el-form-item>

          <el-form-item>
            <el-button
              type="primary"
              @click="createShortUrl(expiresIn)"
              :loading="loading"
              size="large"
              class="w-full"
            >
              生成短链
            </el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 结果显示 -->
      <div v-if="shortUrl" class="mt-6 space-y-4">
        <el-divider />

        <div class="url-display-box">
          <div class="url-label">生成的短链：</div>
          <div class="flex items-center gap-2">
            <el-input
              v-model="shortUrl"
              readonly
              size="large"
            >
              <template #prefix>
                <el-icon><Link /></el-icon>
              </template>
            </el-input>
            <el-button
              type="primary"
              @click="copyToClipboard"
              :icon="DocumentCopy"
              size="large"
            >
              复制
            </el-button>
          </div>
        </div>

        <!-- 统计信息 -->
        <el-descriptions :column="2" border>
          <el-descriptions-item label="短链 ID">
            {{ result.id }}
          </el-descriptions-item>
          <el-descriptions-item label="过期时间">
            {{ formatDateTime(result.expires_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="最大点击数">
            {{ result.max_clicks }}
          </el-descriptions-item>
          <el-descriptions-item label="当前点击数">
            <el-tag type="info">{{ stats.clicks || 0 }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <!-- QR 码 -->
        <div class="text-center">
          <div class="qr-label">扫码访问</div>
          <div class="qr-container">
            <canvas ref="qrcodeCanvas"></canvas>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="flex gap-2 justify-center">
          <el-button @click="refreshStats" :icon="Refresh">
            刷新统计
          </el-button>
          <el-button @click="reset" :icon="Plus">
            生成新短链
          </el-button>
          <el-button @click="testRedirect" :icon="Right">
            测试访问
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- 查询统计 -->
    <el-card class="mt-4">
      <template #header>
        <span class="font-bold">查询短链统计</span>
      </template>
      <div class="flex gap-2">
        <el-input
          v-model="queryId"
          placeholder="输入短链ID查询统计"
          clearable
          @keyup.enter="queryStats"
        />
        <el-button type="primary" @click="queryStats" :loading="queryLoading">
          查询
        </el-button>
      </div>
      <div v-if="queryResult" class="mt-4">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="短链ID">{{ queryResult.id }}</el-descriptions-item>
          <el-descriptions-item label="点击次数">
            <el-tag>{{ queryResult.clicks }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="原始URL" :span="2">
            <a :href="queryResult.original_url" target="_blank" class="text-blue-500 break-all">
              {{ queryResult.original_url }}
            </a>
          </el-descriptions-item>
          <el-descriptions-item label="过期时间">{{ formatDateTime(queryResult.expires_at) }}</el-descriptions-item>
          <el-descriptions-item label="最大点击">{{ queryResult.max_clicks }}</el-descriptions-item>
        </el-descriptions>
      </div>
    </el-card>

    <!-- 短链列表 -->
    <el-card class="mt-4">
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-bold">短链列表</span>
          <div class="flex gap-2">
            <el-input
              v-model="listPassword"
              type="password"
              placeholder="管理密码"
              size="small"
              style="width: 150px"
              show-password
            />
            <el-button type="primary" size="small" @click="loadList" :loading="listLoading">
              加载列表
            </el-button>
          </div>
        </div>
      </template>
      <el-table v-if="urlList.length" :data="urlList" size="small" max-height="400">
        <el-table-column prop="id" label="ID" width="100" />
        <el-table-column prop="original_url" label="原始URL" show-overflow-tooltip />
        <el-table-column prop="clicks" label="点击" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small">{{ row.clicks }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="expires_at" label="过期时间" width="160">
          <template #default="{ row }">
            {{ formatDateTime(row.expires_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" align="center">
          <template #default="{ row }">
            <el-button link size="small" @click="copyLink(row.id)">复制</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-else-if="listLoaded" description="暂无数据" />
      <div v-else class="empty-text">输入密码后点击加载</div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Link, DocumentCopy, Refresh, Plus, Right, Lock, Edit } from '@element-plus/icons-vue'
import QRCode from 'qrcode'

// 状态
const originalUrl = ref('')
const expiresIn = ref(720) // 30 天
const maxClicks = ref(1000)
const password = ref('')
const customId = ref('')
const showAdvanced = ref(false)
const loading = ref(false)
const shortUrl = ref('')
const result = ref({})
const stats = ref({})
const qrcodeCanvas = ref(null)
const form = ref({})

// 查询统计
const queryId = ref('')
const queryLoading = ref(false)
const queryResult = ref(null)

// 短链列表
const listPassword = ref('')
const listLoading = ref(false)
const listLoaded = ref(false)
const urlList = ref([])

// 验证 URL 格式
const validateUrl = (url) => {
  try {
    const urlObj = new URL(url)
    if (!['http:', 'https:'].includes(urlObj.protocol)) {
      return false
    }
    return true
  } catch {
    return false
  }
}

// 创建短链
const createShortUrl = async (hours) => {
  if (!originalUrl.value.trim()) {
    ElMessage.warning('请输入要缩短的 URL')
    return
  }

  if (!validateUrl(originalUrl.value.trim())) {
    ElMessage.error('请输入有效的 URL（必须以 http:// 或 https:// 开头）')
    return
  }

  // 验证自定义ID格式
  if (customId.value) {
    if (customId.value.length > 32) {
      ElMessage.error('自定义ID长度不能超过32个字符')
      return
    }
    if (!/^[a-zA-Z0-9_-]+$/.test(customId.value)) {
      ElMessage.error('自定义ID只能包含字母、数字、下划线和短横线')
      return
    }
  }

  loading.value = true
  try {
    const requestBody = {
      original_url: originalUrl.value.trim(),
      expires_in: hours || expiresIn.value,
      max_clicks: maxClicks.value
    }

    // 高级模式下添加密码和自定义ID
    if (showAdvanced.value) {
      if (password.value) {
        requestBody.password = password.value
      }
      if (customId.value) {
        requestBody.custom_id = customId.value
      }
    }

    const response = await fetch('/api/shorturl', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(requestBody)
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '创建失败')
    }

    result.value = data
    shortUrl.value = data.short_url
    stats.value = { clicks: 0 }

    ElMessage.success('短链生成成功！')

    // 生成 QR 码
    await nextTick()
    if (qrcodeCanvas.value) {
      await QRCode.toCanvas(qrcodeCanvas.value, data.short_url, {
        width: 200,
        margin: 2
      })
    }

    // 自动复制到剪贴板
    copyToClipboard()
  } catch (error) {
    ElMessage.error(error.message || '创建短链失败')
  } finally {
    loading.value = false
  }
}

// 复制到剪贴板
const copyToClipboard = async () => {
  try {
    await navigator.clipboard.writeText(shortUrl.value)
    ElMessage.success('已复制到剪贴板')
  } catch {
    ElMessage.warning('复制失败，请手动复制')
  }
}

// 刷新统计
const refreshStats = async () => {
  if (!result.value.id) return

  try {
    const response = await fetch(`/api/shorturl/${result.value.id}/stats`)
    const data = await response.json()

    if (response.ok) {
      stats.value = data
      ElMessage.success('统计信息已更新')
    } else {
      throw new Error(data.error || '获取统计信息失败')
    }
  } catch (error) {
    ElMessage.error(error.message)
  }
}

// 重置表单
const reset = () => {
  originalUrl.value = ''
  shortUrl.value = ''
  result.value = {}
  stats.value = {}
  expiresIn.value = 720
  maxClicks.value = 1000
  customId.value = ''
  // 保留密码不清除，方便连续创建
}

// 测试重定向
const testRedirect = () => {
  if (shortUrl.value) {
    window.open(shortUrl.value, '_blank')
  }
}

// 格式化日期时间
const formatDateTime = (dateStr) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 查询单个短链统计
const queryStats = async () => {
  if (!queryId.value.trim()) {
    ElMessage.warning('请输入短链ID')
    return
  }

  queryLoading.value = true
  queryResult.value = null
  try {
    const response = await fetch(`/api/shorturl/${queryId.value.trim()}/stats`)
    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '查询失败')
    }

    queryResult.value = data
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    queryLoading.value = false
  }
}

// 加载短链列表
const loadList = async () => {
  if (!listPassword.value) {
    ElMessage.warning('请输入管理密码')
    return
  }

  listLoading.value = true
  try {
    const response = await fetch(`/api/shorturl/list?password=${encodeURIComponent(listPassword.value)}`)
    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '加载失败')
    }

    urlList.value = data.urls || []
    listLoaded.value = true
    ElMessage.success(`已加载 ${urlList.value.length} 条记录`)
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    listLoading.value = false
  }
}

// 复制短链
const copyLink = async (id) => {
  const url = `${window.location.origin}/s/${id}`
  try {
    await navigator.clipboard.writeText(url)
    ElMessage.success('已复制')
  } catch {
    ElMessage.warning('复制失败')
  }
}
</script>

<style scoped>
.container {
  min-height: calc(100vh - 120px);
}

:deep(.el-input-number) {
  width: 100%;
}

:deep(.el-input-number .el-input__inner) {
  text-align: left;
}

/* URL 显示区域 */
.url-display-box {
  background: var(--bg-secondary);
  padding: 16px;
  border-radius: var(--radius-md);
}

.url-label {
  margin-bottom: 8px;
  font-size: 14px;
  color: var(--text-tertiary);
}

/* QR 码样式 */
.qr-label {
  margin-bottom: 8px;
  font-size: 14px;
  color: var(--text-tertiary);
}

.qr-container {
  display: inline-block;
  padding: 16px;
  background: var(--qr-bg);
  border-radius: var(--radius-md);
}

/* 提示文本 */
.hint-text {
  font-size: 14px;
  color: var(--text-tertiary);
  margin-top: 4px;
}

/* 空状态文本 */
.empty-text {
  text-align: center;
  color: var(--text-quaternary);
  padding: 16px 0;
}
</style>
