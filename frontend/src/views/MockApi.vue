<template>
  <div class="container mx-auto p-4 max-w-6xl">
    <el-card>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="text-xl font-bold">Mock API 生成器</span>
          <el-switch
            v-model="showAdvanced"
            active-text="高级模式"
            inactive-text="快捷模式"
          />
        </div>
      </template>

      <!-- 快捷模式 -->
      <div v-if="!showAdvanced && !createdMock" class="space-y-4">
        <el-select
          v-model="method"
          placeholder="选择 HTTP 方法"
          size="large"
          class="w-full"
        >
          <el-option label="GET" value="GET" />
          <el-option label="POST" value="POST" />
          <el-option label="PUT" value="PUT" />
          <el-option label="DELETE" value="DELETE" />
          <el-option label="PATCH" value="PATCH" />
        </el-select>

        <el-input
          v-model="responseBody"
          type="textarea"
          :rows="8"
          placeholder='响应内容，例如: {"status": "ok", "data": {"id": 1}}'
          class="font-mono"
        />

        <el-select v-model="responseStatus" placeholder="响应状态码" size="large" class="w-full">
          <el-option label="200 OK" :value="200" />
          <el-option label="201 Created" :value="201" />
          <el-option label="400 Bad Request" :value="400" />
          <el-option label="404 Not Found" :value="404" />
          <el-option label="500 Internal Server Error" :value="500" />
        </el-select>

        <el-button
          type="primary"
          size="large"
          @click="createMock"
          :loading="loading"
          class="w-full"
        >
          <el-icon><Plus /></el-icon>
          创建 Mock API
        </el-button>

        <el-button size="small" text @click="showAdvanced = true" class="w-full">
          高级选项（自定义路径、延迟、过期时间等）
        </el-button>
      </div>

      <!-- 高级模式 -->
      <div v-if="showAdvanced && !createdMock" class="space-y-4">
        <el-form :model="form" label-width="130px">
          <el-form-item label="Mock 名称">
            <el-input
              v-model="name"
              placeholder="可选，方便识别"
              clearable
            />
          </el-form-item>

          <el-form-item label="HTTP 方法" required>
            <el-select v-model="method" class="w-full">
              <el-option label="GET" value="GET" />
              <el-option label="POST" value="POST" />
              <el-option label="PUT" value="PUT" />
              <el-option label="DELETE" value="DELETE" />
              <el-option label="PATCH" value="PATCH" />
              <el-option label="OPTIONS" value="OPTIONS" />
            </el-select>
          </el-form-item>

          <el-form-item label="自定义路径">
            <el-input
              v-model="customPath"
              placeholder="例如: my-api (可选，留空则随机生成)"
              clearable
            />
            <div class="text-sm text-gray-500 mt-1">
              只能包含字母、数字、下划线、短横线，最长32字符
            </div>
          </el-form-item>

          <el-form-item label="响应内容" required>
            <el-input
              v-model="responseBody"
              type="textarea"
              :rows="10"
              placeholder='{"status": "ok", "message": "Hello World"}'
              class="font-mono"
            />
            <div class="text-sm text-gray-500 mt-1">
              最大 100KB
            </div>
          </el-form-item>

          <el-form-item label="响应状态码">
            <el-select v-model="responseStatus" class="w-full">
              <el-option label="200 OK" :value="200" />
              <el-option label="201 Created" :value="201" />
              <el-option label="204 No Content" :value="204" />
              <el-option label="400 Bad Request" :value="400" />
              <el-option label="401 Unauthorized" :value="401" />
              <el-option label="403 Forbidden" :value="403" />
              <el-option label="404 Not Found" :value="404" />
              <el-option label="500 Internal Server Error" :value="500" />
              <el-option label="502 Bad Gateway" :value="502" />
              <el-option label="503 Service Unavailable" :value="503" />
            </el-select>
          </el-form-item>

          <el-form-item label="响应头">
            <div class="space-y-2">
              <div
                v-for="(header, index) in responseHeaders"
                :key="index"
                class="flex gap-2"
              >
                <el-input
                  v-model="header.key"
                  placeholder="Header Name"
                  style="width: 200px"
                />
                <el-input
                  v-model="header.value"
                  placeholder="Header Value"
                  class="flex-1"
                />
                <el-button
                  @click="removeHeader(index)"
                  :icon="Delete"
                  circle
                />
              </div>
              <el-button @click="addHeader" size="small" text>
                <el-icon><Plus /></el-icon>
                添加响应头
              </el-button>
            </div>
          </el-form-item>

          <el-form-item label="响应延迟">
            <div class="flex items-center gap-4 w-full">
              <el-slider
                v-model="responseDelay"
                :min="0"
                :max="30"
                show-input
                class="flex-1"
              />
              <span class="text-sm text-gray-500">秒</span>
            </div>
            <div class="text-sm text-gray-500 mt-1">
              模拟慢速 API，0-30 秒
            </div>
          </el-form-item>

          <el-form-item label="过期时间">
            <el-select v-model="expiresIn" class="w-full">
              <el-option label="1 小时" :value="1" />
              <el-option label="6 小时" :value="6" />
              <el-option label="24 小时（默认）" :value="24" />
              <el-option label="3 天" :value="72" />
              <el-option label="7 天" :value="168" />
            </el-select>
          </el-form-item>

          <el-form-item label="最大调用次数">
            <el-input-number
              v-model="maxCalls"
              :min="1"
              :max="100000"
              :step="100"
              class="w-full"
            />
            <div class="text-sm text-gray-500 mt-1">
              默认 1000 次，最多 100000 次
            </div>
          </el-form-item>

          <el-form-item label="访问密码">
            <el-input
              v-model="password"
              type="password"
              placeholder="可选，用于管理操作"
              clearable
              show-password
            />
            <div class="text-sm text-gray-500 mt-1">
              设置后更新、删除、查看日志需要密码
            </div>
          </el-form-item>

          <el-form-item>
            <div class="flex gap-2 w-full">
              <el-button
                type="primary"
                @click="createMock"
                :loading="loading"
                size="large"
                class="flex-1"
              >
                <el-icon><Plus /></el-icon>
                创建 Mock API
              </el-button>
              <el-button @click="showAdvanced = false" size="large">
                简洁模式
              </el-button>
            </div>
          </el-form-item>
        </el-form>
      </div>

      <!-- 结果显示 -->
      <div v-if="createdMock" class="space-y-6">
        <el-alert
          title="Mock API 创建成功！"
          type="success"
          :closable="false"
          show-icon
        />

        <!-- Mock URL -->
        <div class="url-display-box">
          <div class="url-label">Mock URL：</div>
          <div class="flex items-center gap-2">
            <el-input
              v-model="createdMock.mock_url"
              readonly
              size="large"
            >
              <template #prefix>
                <el-icon><Link /></el-icon>
              </template>
            </el-input>
            <el-button
              type="primary"
              @click="copyMockURL"
              :icon="DocumentCopy"
              size="large"
            >
              复制
            </el-button>
            <el-button
              @click="testInBrowser"
              :icon="Right"
              size="large"
            >
              测试
            </el-button>
          </div>
        </div>

        <!-- cURL 示例 -->
        <div>
          <div class="curl-label">cURL 命令：</div>
          <div class="relative">
            <pre class="curl-command">{{ curlCommand }}</pre>
            <el-button
              class="absolute top-2 right-2"
              size="small"
              @click="copyCurl"
              :icon="DocumentCopy"
            >
              复制
            </el-button>
          </div>
        </div>

        <!-- 配置信息 -->
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Mock ID">
            {{ createdMock.id }}
          </el-descriptions-item>
          <el-descriptions-item label="HTTP 方法">
            <el-tag>{{ method }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="响应状态">
            <el-tag>{{ responseStatus }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="响应延迟">
            {{ responseDelay }} 秒
          </el-descriptions-item>
          <el-descriptions-item label="过期时间">
            {{ formatDateTime(createdMock.expires_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="最大调用次数">
            {{ createdMock.max_calls }}
          </el-descriptions-item>
          <el-descriptions-item label="当前调用次数" :span="2">
            <el-tag type="info">{{ mockStats.call_count || 0 }}</el-tag>
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
        <div class="flex gap-2 justify-center flex-wrap">
          <el-button @click="refreshStats" :icon="Refresh">
            刷新统计
          </el-button>
          <el-button @click="updateMockDialog = true" :icon="Edit">
            更新配置
          </el-button>
          <el-button @click="deleteMockConfirm" type="danger" :icon="Delete">
            删除 Mock
          </el-button>
          <el-button @click="reset" :icon="Plus">
            创建新 Mock
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- 请求日志 -->
    <el-card v-if="createdMock" class="mt-4">
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-bold">请求日志</span>
          <div class="flex gap-2 items-center">
            <el-switch
              v-model="autoRefresh"
              active-text="自动刷新"
              size="small"
            />
            <el-button
              size="small"
              @click="loadLogs"
              :loading="logsLoading"
              :icon="Refresh"
            >
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <div v-if="logs.length === 0" class="text-center text-gray-400 py-8">
        暂无请求日志
      </div>
      <el-table v-else :data="logs" size="small" max-height="500">
        <el-table-column prop="created_at" label="时间" width="160">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="method" label="方法" width="80">
          <template #default="{ row }">
            <el-tag size="small">{{ row.method }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="client_ip" label="IP" width="140" />
        <el-table-column label="Headers" width="100">
          <template #default="{ row }">
            <el-button link size="small" @click="viewLogDetails(row, 'headers')">
              查看
            </el-button>
          </template>
        </el-table-column>
        <el-table-column label="Query" width="100">
          <template #default="{ row }">
            <el-button link size="small" @click="viewLogDetails(row, 'query')">
              查看
            </el-button>
          </template>
        </el-table-column>
        <el-table-column label="Body" width="100">
          <template #default="{ row }">
            <el-button link size="small" @click="viewLogDetails(row, 'body')">
              查看
            </el-button>
          </template>
        </el-table-column>
        <el-table-column prop="user_agent" label="User Agent" show-overflow-tooltip />
      </el-table>
    </el-card>

    <!-- 测试面板 -->
    <el-card v-if="createdMock" class="mt-4">
      <template #header>
        <span class="font-bold">测试面板</span>
      </template>

      <div class="space-y-4">
        <el-form label-width="100px">
          <el-form-item label="请求方法">
            <el-select v-model="testMethod" class="w-full">
              <el-option label="GET" value="GET" />
              <el-option label="POST" value="POST" />
              <el-option label="PUT" value="PUT" />
              <el-option label="DELETE" value="DELETE" />
              <el-option label="PATCH" value="PATCH" />
            </el-select>
          </el-form-item>

          <el-form-item label="请求头">
            <div class="space-y-2">
              <div
                v-for="(header, index) in testHeaders"
                :key="index"
                class="flex gap-2"
              >
                <el-input
                  v-model="header.key"
                  placeholder="Header Name"
                  style="width: 200px"
                  size="small"
                />
                <el-input
                  v-model="header.value"
                  placeholder="Header Value"
                  class="flex-1"
                  size="small"
                />
                <el-button
                  @click="testHeaders.splice(index, 1)"
                  :icon="Delete"
                  circle
                  size="small"
                />
              </div>
              <el-button @click="testHeaders.push({ key: '', value: '' })" size="small" text>
                <el-icon><Plus /></el-icon>
                添加请求头
              </el-button>
            </div>
          </el-form-item>

          <el-form-item label="Query 参数">
            <div class="space-y-2">
              <div
                v-for="(param, index) in testQueryParams"
                :key="index"
                class="flex gap-2"
              >
                <el-input
                  v-model="param.key"
                  placeholder="Key"
                  style="width: 200px"
                  size="small"
                />
                <el-input
                  v-model="param.value"
                  placeholder="Value"
                  class="flex-1"
                  size="small"
                />
                <el-button
                  @click="testQueryParams.splice(index, 1)"
                  :icon="Delete"
                  circle
                  size="small"
                />
              </div>
              <el-button @click="testQueryParams.push({ key: '', value: '' })" size="small" text>
                <el-icon><Plus /></el-icon>
                添加参数
              </el-button>
            </div>
          </el-form-item>

          <el-form-item label="请求 Body">
            <el-input
              v-model="testBody"
              type="textarea"
              :rows="5"
              placeholder='{"key": "value"}'
              class="font-mono"
            />
          </el-form-item>

          <el-form-item>
            <el-button
              type="primary"
              @click="sendTestRequest"
              :loading="testLoading"
              size="large"
            >
              <el-icon><Promotion /></el-icon>
              发送请求
            </el-button>
          </el-form-item>
        </el-form>

        <!-- 测试响应 -->
        <div v-if="testResponse">
          <el-divider>响应结果</el-divider>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="状态码">
              <el-tag>{{ testResponse.status }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="响应时间">
              {{ testResponse.duration }} ms
            </el-descriptions-item>
          </el-descriptions>

          <div class="mt-4">
            <div class="text-sm font-medium mb-2">响应头：</div>
            <pre class="response-content">{{ testResponse.headers }}</pre>
          </div>

          <div class="mt-4">
            <div class="text-sm font-medium mb-2">响应内容：</div>
            <pre class="response-content">{{ testResponse.body }}</pre>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 更新配置对话框 -->
    <el-dialog v-model="updateMockDialog" title="更新 Mock 配置" width="600px">
      <el-form label-width="120px">
        <el-form-item label="响应内容">
          <el-input
            v-model="updateForm.response_body"
            type="textarea"
            :rows="8"
            class="font-mono"
          />
        </el-form-item>
        <el-form-item label="响应状态码">
          <el-input-number v-model="updateForm.response_status" :min="100" :max="599" />
        </el-form-item>
        <el-form-item label="响应延迟">
          <el-slider
            v-model="updateForm.response_delay"
            :min="0"
            :max="30"
            show-input
          />
        </el-form-item>
        <el-form-item label="密码" v-if="password">
          <el-input
            v-model="updatePassword"
            type="password"
            placeholder="请输入密码"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="updateMockDialog = false">取消</el-button>
        <el-button type="primary" @click="updateMock" :loading="updating">
          更新
        </el-button>
      </template>
    </el-dialog>

    <!-- 日志详情对话框 -->
    <el-dialog v-model="logDetailDialog" title="日志详情" width="700px">
      <div v-if="currentLogDetail">
        <div class="text-sm font-medium mb-2">{{ currentLogDetailType }}：</div>
        <pre class="log-detail">{{ currentLogDetail }}</pre>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Link,
  DocumentCopy,
  Refresh,
  Right,
  Delete,
  Edit,
  Promotion
} from '@element-plus/icons-vue'
import QRCode from 'qrcode'

// 基本状态
const name = ref('')
const method = ref('GET')
const responseBody = ref('{"status": "ok", "message": "Hello from Mock API"}')
const responseStatus = ref(200)
const responseHeaders = ref([])
const responseDelay = ref(0)
const expiresIn = ref(24)
const maxCalls = ref(1000)
const customPath = ref('')
const password = ref('')
const showAdvanced = ref(false)
const loading = ref(false)
const form = ref({})

// 创建结果
const createdMock = ref(null)
const mockStats = ref({})
const qrcodeCanvas = ref(null)

// 日志
const logs = ref([])
const logsLoading = ref(false)
const autoRefresh = ref(false)
let refreshTimer = null

// 测试面板
const testMethod = ref('GET')
const testHeaders = ref([])
const testQueryParams = ref([])
const testBody = ref('')
const testLoading = ref(false)
const testResponse = ref(null)

// 更新对话框
const updateMockDialog = ref(false)
const updateForm = ref({})
const updatePassword = ref('')
const updating = ref(false)

// 日志详情对话框
const logDetailDialog = ref(false)
const currentLogDetail = ref('')
const currentLogDetailType = ref('')

// 计算 cURL 命令
const curlCommand = ref('')

// 添加响应头
const addHeader = () => {
  responseHeaders.value.push({ key: '', value: '' })
}

// 移除响应头
const removeHeader = (index) => {
  responseHeaders.value.splice(index, 1)
}

// 创建 Mock API
const createMock = async () => {
  if (!responseBody.value.trim()) {
    ElMessage.warning('请输入响应内容')
    return
  }

  // 验证自定义路径格式
  if (customPath.value) {
    if (customPath.value.length > 32) {
      ElMessage.error('自定义路径长度不能超过32个字符')
      return
    }
    if (!/^[a-zA-Z0-9_-]+$/.test(customPath.value)) {
      ElMessage.error('自定义路径只能包含字母、数字、下划线和短横线')
      return
    }
  }

  loading.value = true
  try {
    const requestBody = {
      name: name.value,
      method: method.value,
      response_body: responseBody.value,
      response_status: responseStatus.value,
      response_delay: responseDelay.value,
      expires_in: expiresIn.value,
      max_calls: maxCalls.value
    }

    // 添加响应头
    if (responseHeaders.value.length > 0) {
      const headers = {}
      responseHeaders.value.forEach(h => {
        if (h.key && h.value) {
          headers[h.key] = h.value
        }
      })
      requestBody.response_headers = headers
    }

    // 添加自定义路径
    if (customPath.value) {
      requestBody.custom_path = customPath.value
    }

    // 添加密码
    if (password.value) {
      requestBody.password = password.value
    }

    const response = await fetch('/api/mockapi', {
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

    createdMock.value = data
    mockStats.value = { call_count: 0 }

    // 生成 cURL 命令
    generateCurlCommand()

    ElMessage.success('Mock API 创建成功！')

    // 生成 QR 码
    await nextTick()
    if (qrcodeCanvas.value) {
      await QRCode.toCanvas(qrcodeCanvas.value, data.mock_url, {
        width: 200,
        margin: 2
      })
    }

    // 自动复制 URL
    copyMockURL()

    // 加载日志
    loadLogs()
  } catch (error) {
    ElMessage.error(error.message || '创建 Mock API 失败')
  } finally {
    loading.value = false
  }
}

// 生成 cURL 命令
const generateCurlCommand = () => {
  if (!createdMock.value) return

  let cmd = `curl -X ${method.value} '${createdMock.value.mock_url}'`

  if (responseHeaders.value.length > 0) {
    responseHeaders.value.forEach(h => {
      if (h.key && h.value) {
        cmd += ` \\\n  -H '${h.key}: ${h.value}'`
      }
    })
  }

  if (method.value !== 'GET' && responseBody.value) {
    cmd += ` \\\n  -d '${responseBody.value}'`
  }

  curlCommand.value = cmd
}

// 复制 Mock URL
const copyMockURL = async () => {
  try {
    await navigator.clipboard.writeText(createdMock.value.mock_url)
    ElMessage.success('Mock URL 已复制到剪贴板')
  } catch {
    ElMessage.warning('复制失败，请手动复制')
  }
}

// 复制 cURL 命令
const copyCurl = async () => {
  try {
    await navigator.clipboard.writeText(curlCommand.value)
    ElMessage.success('cURL 命令已复制到剪贴板')
  } catch {
    ElMessage.warning('复制失败，请手动复制')
  }
}

// 在浏览器中测试
const testInBrowser = () => {
  if (createdMock.value) {
    window.open(createdMock.value.mock_url, '_blank')
  }
}

// 刷新统计
const refreshStats = async () => {
  if (!createdMock.value) return

  try {
    const response = await fetch(`/api/mockapi/${createdMock.value.id}`)
    const data = await response.json()

    if (response.ok) {
      mockStats.value = data
      ElMessage.success('统计信息已更新')
    } else {
      throw new Error(data.error || '获取统计信息失败')
    }
  } catch (error) {
    ElMessage.error(error.message)
  }
}

// 加载日志
const loadLogs = async () => {
  if (!createdMock.value) return

  logsLoading.value = true
  try {
    let url = `/api/mockapi/${createdMock.value.id}/logs`
    if (password.value) {
      url += `?password=${encodeURIComponent(password.value)}`
    }

    const response = await fetch(url)
    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '获取日志失败')
    }

    logs.value = data.logs || []
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    logsLoading.value = false
  }
}

// 查看日志详情
const viewLogDetails = (log, type) => {
  currentLogDetailType.value = type
  let content = ''

  switch (type) {
    case 'headers':
      try {
        content = JSON.stringify(JSON.parse(log.headers), null, 2)
      } catch {
        content = log.headers
      }
      break
    case 'query':
      try {
        content = JSON.stringify(JSON.parse(log.query_params), null, 2)
      } catch {
        content = log.query_params
      }
      break
    case 'body':
      content = log.body || '(空)'
      break
  }

  currentLogDetail.value = content
  logDetailDialog.value = true
}

// 更新 Mock
const updateMock = async () => {
  if (!createdMock.value) return

  if (password.value && !updatePassword.value) {
    ElMessage.warning('请输入密码')
    return
  }

  updating.value = true
  try {
    const updates = {}

    if (updateForm.value.response_body !== undefined) {
      updates.response_body = updateForm.value.response_body
    }
    if (updateForm.value.response_status !== undefined) {
      updates.response_status = updateForm.value.response_status
    }
    if (updateForm.value.response_delay !== undefined) {
      updates.response_delay = updateForm.value.response_delay
    }

    if (password.value) {
      updates.password = updatePassword.value
    }

    const response = await fetch(`/api/mockapi/${createdMock.value.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(updates)
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '更新失败')
    }

    ElMessage.success('Mock API 更新成功')
    updateMockDialog.value = false
    refreshStats()
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    updating.value = false
  }
}

// 删除 Mock
const deleteMockConfirm = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要删除这个 Mock API 吗？此操作不可恢复。',
      '确认删除',
      {
        type: 'warning'
      }
    )

    let url = `/api/mockapi/${createdMock.value.id}`
    if (password.value) {
      url += `?password=${encodeURIComponent(password.value)}`
    }

    const response = await fetch(url, { method: 'DELETE' })
    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '删除失败')
    }

    ElMessage.success('Mock API 已删除')
    reset()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

// 发送测试请求
const sendTestRequest = async () => {
  if (!createdMock.value) return

  testLoading.value = true
  const startTime = Date.now()

  try {
    // 构建 URL with query params
    let url = createdMock.value.mock_url
    const queryParams = testQueryParams.value
      .filter(p => p.key)
      .map(p => `${encodeURIComponent(p.key)}=${encodeURIComponent(p.value)}`)
      .join('&')

    if (queryParams) {
      url += '?' + queryParams
    }

    // 构建 headers
    const headers = {}
    testHeaders.value.forEach(h => {
      if (h.key && h.value) {
        headers[h.key] = h.value
      }
    })

    const options = {
      method: testMethod.value,
      headers
    }

    if (testMethod.value !== 'GET' && testBody.value) {
      options.body = testBody.value
    }

    const response = await fetch(url, options)
    const duration = Date.now() - startTime

    // 读取响应
    const body = await response.text()

    // 格式化响应头
    const responseHeaders = {}
    response.headers.forEach((value, key) => {
      responseHeaders[key] = value
    })

    testResponse.value = {
      status: response.status,
      duration,
      headers: JSON.stringify(responseHeaders, null, 2),
      body: body
    }

    // 刷新日志
    loadLogs()
  } catch (error) {
    ElMessage.error('请求失败: ' + error.message)
  } finally {
    testLoading.value = false
  }
}

// 重置表单
const reset = () => {
  name.value = ''
  method.value = 'GET'
  responseBody.value = '{"status": "ok", "message": "Hello from Mock API"}'
  responseStatus.value = 200
  responseHeaders.value = []
  responseDelay.value = 0
  expiresIn.value = 24
  maxCalls.value = 1000
  customPath.value = ''
  password.value = ''
  createdMock.value = null
  mockStats.value = {}
  logs.value = []
  testResponse.value = null
  autoRefresh.value = false
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
    minute: '2-digit',
    second: '2-digit'
  })
}

// 监听自动刷新
watch(autoRefresh, (newVal) => {
  if (newVal && createdMock.value) {
    refreshTimer = setInterval(() => {
      loadLogs()
      refreshStats()
    }, 5000)
  } else {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
  }
})

// 监听更新对话框打开
watch(updateMockDialog, (newVal) => {
  if (newVal) {
    updateForm.value = {
      response_body: responseBody.value,
      response_status: responseStatus.value,
      response_delay: responseDelay.value
    }
    updatePassword.value = ''
  }
})
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

:deep(.el-slider) {
  padding: 0 12px;
}

pre {
  white-space: pre-wrap;
  word-wrap: break-word;
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
  font-weight: 500;
  color: var(--text-secondary);
}

/* cURL 命令样式 */
.curl-label {
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
}

.curl-command {
  background: var(--code-bg);
  color: var(--color-success);
  padding: 16px;
  border-radius: var(--radius-md);
  font-size: 14px;
  overflow-x: auto;
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

/* 响应内容样式 */
.response-content {
  background: var(--bg-secondary);
  padding: 12px;
  border-radius: var(--radius-md);
  font-size: 12px;
  overflow-x: auto;
  color: var(--text-primary);
}

/* 日志详情样式 */
.log-detail {
  background: var(--bg-secondary);
  padding: 16px;
  border-radius: var(--radius-md);
  font-size: 12px;
  overflow-x: auto;
  max-height: 384px;
  color: var(--text-primary);
}
</style>
