<template>
  <div class="container mx-auto p-4 max-w-6xl">
    <!-- Password Gate -->
    <div v-if="!authenticated" class="flex justify-center items-center min-h-[60vh]">
      <el-card class="w-full max-w-md shadow-lg">
        <template #header>
          <div class="flex items-center gap-2">
            <el-icon class="text-purple-500 text-xl"><MagicStick /></el-icon>
            <span class="text-xl font-bold">AutoDev AI 任务助手</span>
          </div>
        </template>
        <div class="py-4">
          <p class="text-gray-500 mb-6 text-sm">此工具需要管理员密码才能使用。</p>
          <el-form @submit.prevent="login">
            <el-form-item>
              <el-input
                v-model="passwordInput"
                type="password"
                placeholder="请输入访问密码"
                show-password
                size="large"
                @keyup.enter="login"
              >
                <template #prefix><el-icon><Lock /></el-icon></template>
              </el-input>
            </el-form-item>
            <el-button type="primary" size="large" class="w-full" :loading="loggingIn" @click="login">
              进入
            </el-button>
          </el-form>
        </div>
      </el-card>
    </div>

    <!-- Main UI -->
    <div v-else>
      <div class="flex items-center justify-between mb-5">
        <div class="flex items-center gap-2">
          <el-icon class="text-purple-500 text-2xl"><MagicStick /></el-icon>
          <h1 class="text-2xl font-bold">AutoDev AI 任务助手</h1>
        </div>
        <div class="flex items-center gap-2">
          <!-- Claude 版本管理入口 -->
          <el-button size="small" @click="claudeDrawerVisible = true" plain>
            <el-icon><SetUp /></el-icon>
            Claude 管理
          </el-button>
          <el-button size="small" @click="logout" plain>退出登录</el-button>
        </div>
      </div>

      <!-- Claude 版本管理抽屉 -->
      <el-drawer
        v-model="claudeDrawerVisible"
        title="Claude CLI 版本管理"
        direction="rtl"
        size="480px"
        @open="onClaudeDrawerOpen"
      >
        <div class="space-y-4 p-2">
          <!-- 版本信息 -->
          <el-card shadow="never">
            <template #header>
              <div class="flex items-center justify-between">
                <span class="font-semibold text-sm">当前环境</span>
                <el-button size="small" :loading="loadingClaudeInfo" @click="loadClaudeVersion" circle plain>
                  <el-icon><Refresh /></el-icon>
                </el-button>
              </div>
            </template>
            <div v-if="loadingClaudeInfo" class="text-center py-4">
              <el-icon class="is-loading text-purple-500 text-xl"><Loading /></el-icon>
            </div>
            <div v-else-if="claudeInfo" class="space-y-2">
              <div class="flex items-center justify-between py-1.5 border-b">
                <span class="text-sm text-gray-500">Claude CLI</span>
                <div class="flex items-center gap-2">
                  <el-tag v-if="claudeInfo.available" type="success" size="small" effect="dark">已安装</el-tag>
                  <el-tag v-else type="danger" size="small" effect="dark">未安装</el-tag>
                  <span class="text-sm font-mono font-semibold text-purple-600">{{ claudeInfo.version || '—' }}</span>
                </div>
              </div>
              <div class="flex items-center justify-between py-1.5 border-b">
                <span class="text-sm text-gray-500">安装路径</span>
                <span class="text-xs font-mono text-gray-600 truncate max-w-[240px]">{{ claudeInfo.path || '—' }}</span>
              </div>
              <div class="flex items-center justify-between py-1.5 border-b">
                <span class="text-sm text-gray-500">Node.js</span>
                <span class="text-sm font-mono">{{ claudeInfo.node_version || '—' }}</span>
              </div>
              <div class="flex items-center justify-between py-1.5">
                <span class="text-sm text-gray-500">npm</span>
                <span class="text-sm font-mono">{{ claudeInfo.npm_version || '—' }}</span>
              </div>
            </div>
            <div v-else class="text-center text-gray-400 py-4 text-sm">
              点击刷新按钮获取版本信息
            </div>
          </el-card>

          <!-- 模型健康检测 -->
          <el-card shadow="never">
            <template #header>
              <div class="flex items-center justify-between">
                <span class="font-semibold text-sm">模型连通性测试</span>
                <el-button size="small" :loading="testingModel" @click="testModel" plain>
                  <el-icon><Connection /></el-icon> 测试
                </el-button>
              </div>
            </template>
            <div v-if="!modelHealth && !testingModel" class="text-center text-gray-400 py-4 text-sm">
              点击测试按钮检查 API 连通性
            </div>
            <div v-else-if="testingModel" class="text-center py-4">
              <el-icon class="is-loading text-purple-500 text-xl"><Loading /></el-icon>
              <p class="text-xs text-gray-400 mt-2">正在发送测试请求…</p>
            </div>
            <div v-else-if="modelHealth" class="space-y-2">
              <div class="flex items-center justify-between py-1.5 border-b">
                <span class="text-sm text-gray-500">状态</span>
                <el-tag :type="modelHealth.ok ? 'success' : 'danger'" size="small" effect="dark">
                  {{ modelHealth.ok ? '✅ 连通' : '❌ 不可用' }}
                </el-tag>
              </div>
              <div class="flex items-center justify-between py-1.5 border-b">
                <span class="text-sm text-gray-500">模型</span>
                <span class="text-xs font-mono text-purple-600">{{ modelHealth.model }}</span>
              </div>
              <div class="flex items-center justify-between py-1.5 border-b">
                <span class="text-sm text-gray-500">API 地址</span>
                <span class="text-xs font-mono text-gray-600 truncate max-w-[220px]" :title="modelHealth.base_url">{{ modelHealth.base_url }}</span>
              </div>
              <div class="flex items-center justify-between py-1.5 border-b">
                <span class="text-sm text-gray-500">Token</span>
                <el-tag :type="modelHealth.has_token ? 'success' : 'danger'" size="small">
                  {{ modelHealth.has_token ? '已配置' : '未配置' }}
                </el-tag>
              </div>
              <div v-if="modelHealth.ok" class="flex items-center justify-between py-1.5 border-b">
                <span class="text-sm text-gray-500">响应延迟</span>
                <span class="text-sm font-mono" :class="modelHealth.latency_ms < 3000 ? 'text-green-600' : 'text-yellow-600'">
                  {{ modelHealth.latency_ms }} ms
                </span>
              </div>
              <div v-if="modelHealth.response" class="py-1.5 border-b">
                <span class="text-xs text-gray-500 block mb-1">模型回复</span>
                <span class="text-xs font-mono text-green-700 bg-green-50 px-2 py-1 rounded block">{{ modelHealth.response }}</span>
              </div>
              <div v-if="modelHealth.error" class="py-1.5">
                <span class="text-xs text-gray-500 block mb-1">错误信息</span>
                <span class="text-xs text-red-600 bg-red-50 px-2 py-1 rounded block break-all">{{ modelHealth.error }}</span>
              </div>
            </div>
          </el-card>

          <!-- 更新操作 -->
          <el-card shadow="never">
            <template #header>
              <span class="font-semibold text-sm">更新 Claude Code CLI</span>
            </template>
            <div class="space-y-3">
              <div class="bg-gray-50 rounded p-3 text-xs text-gray-600">
                <p>执行命令：</p>
                <code class="font-mono text-purple-700">npm install -g @anthropic-ai/claude-code@latest</code>
              </div>
              <el-button
                type="primary"
                class="w-full"
                :loading="updating"
                :disabled="updating"
                @click="startUpdate"
              >
                <el-icon><Upload /></el-icon>
                <span class="ml-1">{{ updating ? '更新中…' : '立即更新到最新版本' }}</span>
              </el-button>

              <!-- 更新输出日志 -->
              <div v-if="updateLogs.length">
                <div class="flex items-center justify-between mb-1">
                  <span class="text-xs text-gray-500">更新输出</span>
                  <el-button size="small" text @click="updateLogs = []">清空</el-button>
                </div>
                <pre
                  ref="updateLogEl"
                  class="text-xs bg-gray-900 text-green-400 p-3 rounded overflow-auto max-h-[300px] whitespace-pre-wrap break-all font-mono leading-5"
                >{{ updateLogs.join('\n') }}</pre>
              </div>

              <!-- 更新结果 -->
              <el-result
                v-if="updateResult"
                :icon="updateResult.success ? 'success' : 'error'"
                :title="updateResult.success ? '更新成功' : '更新失败'"
                :sub-title="updateResult.message"
              >
                <template #extra>
                  <el-button size="small" @click="loadClaudeVersion">刷新版本信息</el-button>
                </template>
              </el-result>
            </div>
          </el-card>
        </div>
      </el-drawer>

      <el-row :gutter="16">
        <!-- Left Panel: Submit + Task List -->
        <el-col :xs="24" :md="9">
          <!-- Submit Task -->
          <el-card class="mb-4">
            <template #header>
              <span class="font-semibold">{{ resumeTaskId ? '断点恢复任务' : '提交新任务' }}</span>
              <el-button v-if="resumeTaskId" size="small" class="ml-2" @click="cancelResume" plain>取消恢复</el-button>
            </template>
            <el-form label-position="top">
              <el-form-item label="任务描述">
                <el-input
                  v-model="newTask.description"
                  type="textarea"
                  :rows="3"
                  placeholder="例如：写一份 Redis 集群最佳实践文档"
                  maxlength="500"
                  show-word-limit
                  :disabled="!!resumeTaskId"
                />
              </el-form-item>
              <el-form-item v-if="resumeTaskId" label="从第几阶段恢复">
                <el-input-number v-model="newTask.resumeFrom" :min="1" :max="6" />
                <span class="text-xs text-gray-400 ml-2">{{ phaseLabel(newTask.resumeFrom) }}</span>
              </el-form-item>
              <el-form-item label="执行选项">
                <div class="flex flex-wrap gap-2">
                  <el-checkbox v-model="newTask.publish" border size="small">
                    <span>--publish<br/><span class="text-xs text-gray-400">生成文档站</span></span>
                  </el-checkbox>
                  <el-checkbox v-model="newTask.build" border size="small">
                    <span>--build<br/><span class="text-xs text-gray-400">编译构建</span></span>
                  </el-checkbox>
                  <el-checkbox v-model="newTask.push" border size="small">
                    <span>--push<br/><span class="text-xs text-gray-400">推送远端</span></span>
                  </el-checkbox>
                </div>
              </el-form-item>
              <el-button
                type="primary"
                :loading="submitting"
                :disabled="!newTask.description.trim()"
                @click="submitTask"
                class="w-full"
              >
                <el-icon><VideoPlay /></el-icon>
                <span class="ml-1">{{ resumeTaskId ? '恢复执行' : '开始执行' }}</span>
              </el-button>
            </el-form>
          </el-card>

          <!-- Task List -->
          <el-card>
            <template #header>
              <div class="flex items-center justify-between">
                <span class="font-semibold">任务列表 <el-badge :value="runningCount" type="warning" v-if="runningCount" class="ml-1" /></span>
                <el-button size="small" :loading="loadingList" @click="loadTasks" circle>
                  <el-icon><Refresh /></el-icon>
                </el-button>
              </div>
            </template>
            <div v-if="tasks.length === 0" class="text-center text-gray-400 py-8">
              <el-icon class="text-3xl mb-2"><DocumentRemove /></el-icon>
              <p class="text-sm">暂无任务</p>
            </div>
            <div v-else class="space-y-2 max-h-[460px] overflow-y-auto pr-1">
              <div
                v-for="task in tasks"
                :key="task.id"
                class="border rounded-lg p-3 cursor-pointer transition-all hover:border-purple-400"
                :class="{ 'border-purple-500 bg-purple-50': selectedTask?.id === task.id }"
                @click="selectTask(task)"
              >
                <div class="flex items-start gap-2">
                  <div class="flex-1 min-w-0">
                    <p class="text-sm font-medium truncate" :title="task.description">{{ task.description }}</p>
                    <p class="text-xs text-gray-400 mt-0.5">{{ formatTime(task.created_at) }}</p>
                    <!-- Phase progress bar if running -->
                    <div v-if="task.status === 'running' && task.autodev_state" class="mt-1">
                      <div class="flex items-center gap-1">
                        <el-icon class="is-loading text-orange-400 text-xs"><Loading /></el-icon>
                        <span class="text-xs text-orange-500">{{ task.autodev_state.phase_label || '执行中' }}</span>
                      </div>
                      <el-progress
                        :percentage="phaseProgress(task.autodev_state)"
                        :stroke-width="4"
                        :show-text="false"
                        status="warning"
                        class="mt-1"
                      />
                    </div>
                  </div>
                  <div class="flex flex-col items-end gap-1 shrink-0">
                    <el-tag :type="statusType(task.status)" size="small" effect="plain">
                      {{ statusLabel(task.status) }}
                    </el-tag>
                    <div class="flex gap-1">
                      <el-tooltip v-if="task.status === 'running'" content="发送中断信号" placement="top">
                        <el-button size="small" type="warning" circle plain @click.stop="stopTask(task)">
                          <el-icon><VideoPause /></el-icon>
                        </el-button>
                      </el-tooltip>
                      <el-tooltip v-if="task.status === 'failed'" content="断点恢复" placement="top">
                        <el-button size="small" type="success" circle plain @click.stop="startResume(task)">
                          <el-icon><RefreshRight /></el-icon>
                        </el-button>
                      </el-tooltip>
                      <el-tooltip content="删除任务" placement="top">
                        <el-button size="small" type="danger" circle plain @click.stop="deleteTask(task)">
                          <el-icon><Delete /></el-icon>
                        </el-button>
                      </el-tooltip>
                    </div>
                  </div>
                </div>
                <div class="flex gap-1 mt-1 flex-wrap">
                  <el-tag v-if="parseOptions(task.options).publish" size="small" type="info" effect="plain">publish</el-tag>
                  <el-tag v-if="parseOptions(task.options).build" size="small" type="info" effect="plain">build</el-tag>
                  <el-tag v-if="parseOptions(task.options).push" size="small" type="info" effect="plain">push</el-tag>
                </div>
              </div>
            </div>
          </el-card>
        </el-col>

        <!-- Right Panel: Detail Viewer -->
        <el-col :xs="24" :md="15">
          <div v-if="!selectedTask" class="flex items-center justify-center min-h-[400px] text-gray-400">
            <div class="text-center">
              <el-icon class="text-4xl mb-3"><Pointer /></el-icon>
              <p>点击左侧任务查看详情</p>
            </div>
          </div>

          <div v-else class="space-y-3">
            <!-- Task Info Bar -->
            <el-card body-style="padding: 12px 16px">
              <div class="flex items-center justify-between flex-wrap gap-2">
                <div class="flex-1 min-w-0">
                  <p class="font-semibold truncate">{{ selectedTask.description }}</p>
                  <p class="text-xs text-gray-400">
                    ID: {{ selectedTask.id }} ·
                    工作目录: <span class="font-mono">{{ selectedTask.work_dir }}</span>
                  </p>
                </div>
                <div class="flex gap-2 flex-wrap">
                  <el-tag :type="statusType(selectedTask.status)" effect="dark" size="small">
                    <el-icon v-if="selectedTask.status === 'running'" class="is-loading mr-1"><Loading /></el-icon>
                    {{ statusLabel(selectedTask.status) }}
                  </el-tag>
                  <el-button size="small" :loading="loadingDetail" @click="refreshDetail" plain>
                    <el-icon><Refresh /></el-icon>
                  </el-button>
                  <el-button size="small" type="primary" plain :loading="downloading" @click="downloadTask">
                    <el-icon><Download /></el-icon> 打包下载
                  </el-button>
                  <el-button
                    v-if="hasSite"
                    size="small" type="success" plain @click="previewSite"
                  >
                    <el-icon><Monitor /></el-icon> 预览站点
                  </el-button>
                  <el-button
                    v-if="selectedTask.status === 'running'"
                    size="small" type="warning" plain @click="stopTask(selectedTask)"
                  >
                    <el-icon><VideoPause /></el-icon> 中断
                  </el-button>
                  <el-button
                    v-if="selectedTask.status === 'failed'"
                    size="small" type="success" plain @click="startResume(selectedTask)"
                  >
                    <el-icon><RefreshRight /></el-icon> 断点恢复
                  </el-button>
                </div>
              </div>

              <!-- Phase progress detail -->
              <div v-if="taskState" class="mt-3 pt-3 border-t">
                <div class="flex items-center gap-3 mb-2">
                  <span class="text-xs text-gray-500">执行进度</span>
                  <el-tag size="small" :type="taskState.status === 'finished' ? 'success' : 'warning'" effect="plain">
                    {{ taskState.phase_label || taskState.status || '未知' }}
                  </el-tag>
                </div>
                <div class="flex gap-1">
                  <div
                    v-for="(ph, i) in phaseNames"
                    :key="i"
                    class="flex-1 text-center"
                  >
                    <div
                      class="h-2 rounded-full mb-1 transition-colors"
                      :class="phaseBarClass(i, taskState)"
                    />
                    <span class="text-xs text-gray-400 leading-none">{{ ph.short }}</span>
                  </div>
                </div>
              </div>
            </el-card>

            <!-- Tabs: Files / Logs -->
            <el-card>
              <template #header>
                <el-tabs v-model="activeTab" class="-mb-4">
                  <el-tab-pane label="结果文档" name="files" />
                  <el-tab-pane name="logs">
                    <template #label>
                      <span>执行日志</span>
                      <el-badge v-if="selectedTask.status === 'running'" is-dot type="warning" class="ml-1" />
                    </template>
                  </el-tab-pane>
                </el-tabs>
              </template>

              <!-- Files Tab -->
              <div v-show="activeTab === 'files'">
                <div v-if="!taskFiles.length" class="text-center text-gray-400 py-10">
                  <el-icon class="text-3xl mb-2"><FolderOpened /></el-icon>
                  <p class="text-sm">{{ selectedTask.status === 'running' ? '任务执行中，文件生成后自动显示…' : '暂无文件' }}</p>
                </div>
                <div v-else class="flex gap-3 min-h-[400px]">
                  <!-- File Tree -->
                  <div class="w-44 shrink-0 border-r pr-2 overflow-y-auto max-h-[480px]">
                    <p class="text-xs text-gray-400 mb-2 uppercase font-medium">文件</p>
                    <div
                      v-for="file in taskFiles"
                      :key="file.path"
                      class="flex items-center gap-1.5 py-1.5 px-2 rounded cursor-pointer text-sm hover:bg-gray-100 transition-colors truncate"
                      :class="{ 'bg-purple-100 text-purple-700 font-medium': activeFilePath === file.path }"
                      @click="loadFile(file.path)"
                      :title="file.path"
                    >
                      <el-icon class="shrink-0 text-xs text-gray-400">
                        <Document />
                      </el-icon>
                      <span class="truncate text-xs">{{ file.name }}</span>
                    </div>
                  </div>
                  <!-- File Content -->
                  <div class="flex-1 min-w-0 overflow-hidden">
                    <div v-if="!activeFilePath" class="text-center text-gray-400 py-10 text-sm">点击左侧文件查看</div>
                    <div v-else-if="loadingFile" class="flex items-center justify-center py-10">
                      <el-icon class="is-loading text-2xl text-purple-500"><Loading /></el-icon>
                    </div>
                    <div v-else>
                      <div
                        v-if="activeFilePath.endsWith('.md')"
                        class="markdown-body overflow-auto max-h-[480px] pr-1"
                        v-html="renderedMarkdown"
                      />
                      <pre
                        v-else
                        class="text-xs bg-gray-50 p-3 rounded overflow-auto max-h-[480px] whitespace-pre-wrap break-all leading-5"
                      >{{ activeFileContent }}</pre>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Logs Tab -->
              <div v-show="activeTab === 'logs'">
                <div class="flex items-center gap-2 mb-2 flex-wrap">
                  <el-select v-model="activeLogPhase" size="small" class="w-36" @change="loadLogs">
                    <el-option
                      v-for="ph in availableLogPhases"
                      :key="ph"
                      :label="ph"
                      :value="ph"
                    />
                  </el-select>
                  <span class="text-xs text-gray-400">{{ logLineCount }} 行</span>
                  <el-button size="small" :loading="loadingLogs" @click="loadLogs" plain class="ml-auto">
                    <el-icon><Refresh /></el-icon> 刷新
                  </el-button>
                  <el-button size="small" plain @click="scrollLogsToBottom">
                    <el-icon><Bottom /></el-icon> 跳到底部
                  </el-button>
                </div>
                <div v-if="!logContent" class="text-center text-gray-400 py-8 text-sm">
                  {{ selectedTask.status === 'running' ? '任务执行中，日志生成后显示…' : '暂无日志' }}
                </div>
                <pre
                  v-else
                  ref="logEl"
                  class="text-xs bg-gray-900 text-green-400 p-4 rounded overflow-auto max-h-[480px] whitespace-pre-wrap break-all font-mono leading-5"
                >{{ logContent }}</pre>
              </div>
            </el-card>
          </div>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Lock, MagicStick, VideoPlay, VideoPause, Refresh, RefreshRight,
  Delete, Download, DocumentRemove, Pointer, Loading, Document,
  FolderOpened, Bottom, SetUp, Upload, Monitor, Connection
} from '@element-plus/icons-vue'

const SESSION_KEY = 'autodev_password'
const API_BASE = '/api/autodev'

const PHASE_NAMES = [
  { label: 'DISCOVER', short: 'DIS' },
  { label: 'DEFINE',   short: 'DEF' },
  { label: 'DESIGN',   short: 'DSN' },
  { label: 'DO',       short: 'DO'  },
  { label: 'REVIEW',   short: 'REV' },
  { label: 'DELIVER',  short: 'DEL' },
]
const phaseNames = PHASE_NAMES

function phaseLabel(n) {
  return PHASE_NAMES[n - 1]?.label || `阶段 ${n}`
}

// ---- auth ----
const authenticated = ref(false)
const passwordInput = ref('')
const loggingIn = ref(false)
let savedPassword = ''

function getPassword() { return sessionStorage.getItem(SESSION_KEY) || '' }

async function login() {
  if (!passwordInput.value.trim()) return
  loggingIn.value = true
  try {
    const res = await fetch(`${API_BASE}/verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: passwordInput.value })
    })
    if (res.ok) {
      sessionStorage.setItem(SESSION_KEY, passwordInput.value)
      savedPassword = passwordInput.value
      authenticated.value = true
      loadTasks()
    } else {
      ElMessage.error('密码错误')
    }
  } catch { ElMessage.error('连接失败') }
  finally { loggingIn.value = false }
}

function logout() {
  sessionStorage.removeItem(SESSION_KEY)
  authenticated.value = false
  passwordInput.value = ''
  savedPassword = ''
  tasks.value = []
  selectedTask.value = null
}

// ---- tasks ----
const tasks = ref([])
const loadingList = ref(false)
const submitting = ref(false)
const resumeTaskId = ref('')
const newTask = ref({ description: '', publish: false, build: false, push: false, resumeFrom: 1, workDir: '' })

const runningCount = computed(() => tasks.value.filter(t => t.status === 'running').length)

async function loadTasks() {
  loadingList.value = true
  try {
    const res = await fetch(`${API_BASE}/tasks?password=${encodeURIComponent(savedPassword)}`)
    if (res.ok) {
      const data = await res.json()
      tasks.value = data.tasks || []
      if (selectedTask.value) {
        const found = tasks.value.find(t => t.id === selectedTask.value.id)
        if (found) {
          selectedTask.value = found
          taskState.value = found.autodev_state || null
        }
      }
    }
  } finally { loadingList.value = false }
}

async function submitTask() {
  if (!newTask.value.description.trim()) return
  submitting.value = true
  try {
    const body = {
      description: newTask.value.description.trim(),
      password: savedPassword,
      publish: newTask.value.publish,
      build: newTask.value.build,
      push: newTask.value.push,
    }
    if (resumeTaskId.value) {
      body.resume_from = newTask.value.resumeFrom
      body.work_dir = newTask.value.workDir
    }
    const res = await fetch(`${API_BASE}/tasks`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    const data = await res.json()
    if (res.ok) {
      ElMessage.success(resumeTaskId.value ? '已恢复执行' : '任务已提交，正在执行…')
      resumeTaskId.value = ''
      newTask.value = { description: '', publish: false, build: false, push: false, resumeFrom: 1, workDir: '' }
      await loadTasks()
      selectTask(data)
    } else {
      ElMessage.error(data.error || '提交失败')
    }
  } catch { ElMessage.error('网络错误') }
  finally { submitting.value = false }
}

async function stopTask(task) {
  try {
    const res = await fetch(`${API_BASE}/tasks/${task.id}/stop`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: savedPassword })
    })
    const data = await res.json()
    if (res.ok) {
      ElMessage.warning(`已发送中断信号。可从阶段 ${data.resume_from} 断点恢复。`)
      await loadTasks()
    } else {
      ElMessage.error(data.error || '操作失败')
    }
  } catch { ElMessage.error('操作失败') }
}

function startResume(task) {
  const state = task.autodev_state
  let resumeFrom = 1
  if (state?.last_completed != null) {
    resumeFrom = Math.floor(state.last_completed) + 2
  }
  resumeTaskId.value = task.id
  newTask.value = {
    description: task.description,
    publish: parseOptions(task.options).publish || false,
    build: parseOptions(task.options).build || false,
    push: parseOptions(task.options).push || false,
    resumeFrom: resumeFrom > 0 ? resumeFrom : 1,
    workDir: task.work_dir,
  }
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function cancelResume() {
  resumeTaskId.value = ''
  newTask.value = { description: '', publish: false, build: false, push: false, resumeFrom: 1, workDir: '' }
}

async function deleteTask(task) {
  try {
    await ElMessageBox.confirm(
      `确认删除任务「${task.description.slice(0, 40)}」及其所有文件？`,
      '删除确认', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch { return }
  try {
    await fetch(`${API_BASE}/tasks/${task.id}?password=${encodeURIComponent(savedPassword)}`, { method: 'DELETE' })
    ElMessage.success('已删除')
    if (selectedTask.value?.id === task.id) selectedTask.value = null
    await loadTasks()
  } catch { ElMessage.error('删除失败') }
}

// ---- detail ----
const selectedTask = ref(null)
const taskState = ref(null)
const activeTab = ref('files')
const taskFiles = ref([])
const taskHasSite = ref(false)
const activeFilePath = ref(null)
const activeFileContent = ref('')
const loadingFile = ref(false)
const loadingDetail = ref(false)
const logContent = ref('')
const availableLogPhases = ref(['driver'])
const activeLogPhase = ref('driver')
const loadingLogs = ref(false)
const logEl = ref(null)
const downloading = ref(false)

async function selectTask(task) {
  selectedTask.value = task
  taskState.value = task.autodev_state || null
  activeTab.value = 'files'
  activeFilePath.value = null
  activeFileContent.value = ''
  logContent.value = ''
  taskFiles.value = []
  taskHasSite.value = false
  availableLogPhases.value = ['driver']
  activeLogPhase.value = 'driver'
  await refreshDetail()
}

async function refreshDetail() {
  if (!selectedTask.value) return
  loadingDetail.value = true
  try {
    const res = await fetch(`${API_BASE}/tasks/${selectedTask.value.id}?password=${encodeURIComponent(savedPassword)}`)
    if (res.ok) {
      const data = await res.json()
      selectedTask.value = data
      taskState.value = data.autodev_state || null
    }
    await loadFiles()
  } finally { loadingDetail.value = false }
}

async function loadFiles() {
  if (!selectedTask.value) return
  const res = await fetch(`${API_BASE}/tasks/${selectedTask.value.id}/files?password=${encodeURIComponent(savedPassword)}`)
  if (res.ok) {
    const data = await res.json()
    taskFiles.value = data.files || []
    taskHasSite.value = !!data.has_site
    if (!activeFilePath.value) {
      const result = taskFiles.value.find(f => f.name === 'RESULT.md')
      if (result) loadFile(result.path)
    }
  }
}

async function loadFile(path) {
  activeFilePath.value = path
  activeFileContent.value = ''
  loadingFile.value = true
  try {
    const res = await fetch(
      `${API_BASE}/tasks/${selectedTask.value.id}/file?password=${encodeURIComponent(savedPassword)}&path=${encodeURIComponent(path)}`
    )
    if (res.ok) {
      const data = await res.json()
      activeFileContent.value = data.content || ''
    }
  } finally { loadingFile.value = false }
}

async function loadLogs() {
  if (!selectedTask.value) return
  loadingLogs.value = true
  try {
    const res = await fetch(
      `${API_BASE}/tasks/${selectedTask.value.id}/logs?password=${encodeURIComponent(savedPassword)}&phase=${encodeURIComponent(activeLogPhase.value)}`
    )
    if (res.ok) {
      const data = await res.json()
      logContent.value = data.logs || ''
      if (data.available_phases?.length) {
        availableLogPhases.value = data.available_phases
        if (!availableLogPhases.value.includes(activeLogPhase.value)) {
          activeLogPhase.value = availableLogPhases.value[0]
        }
      }
      await nextTick()
      scrollLogsToBottom()
    }
  } finally { loadingLogs.value = false }
}

function scrollLogsToBottom() {
  if (logEl.value) logEl.value.scrollTop = logEl.value.scrollHeight
}

const logLineCount = computed(() => logContent.value ? logContent.value.split('\n').length : 0)

watch(activeTab, tab => { if (tab === 'logs') loadLogs() })

async function downloadTask() {
  if (!selectedTask.value) return
  downloading.value = true
  try {
    const url = `${API_BASE}/tasks/${selectedTask.value.id}/download?password=${encodeURIComponent(savedPassword)}`
    const a = document.createElement('a')
    a.href = url
    a.download = ''
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    ElMessage.success('开始下载')
  } catch { ElMessage.error('下载失败') }
  finally { setTimeout(() => { downloading.value = false }, 1000) }
}

const hasSite = computed(() => taskHasSite.value)

function previewSite() {
  if (!selectedTask.value) return
  const url = `${API_BASE}/tasks/${selectedTask.value.id}/site/index.html?password=${encodeURIComponent(savedPassword)}`
  window.open(url, '_blank')
}

// ---- phase progress helpers ----
function phaseProgress(state) {
  if (!state) return 0
  if (state.status === 'finished') return 100
  const cur = state.current_phase ?? -1
  return Math.min(Math.round((cur / PHASE_NAMES.length) * 100), 95)
}

function phaseBarClass(index, state) {
  if (!state) return 'bg-gray-200'
  const last = state.last_completed ?? -1
  const cur = state.current_phase ?? -1
  if (index <= last) return 'bg-green-400'
  if (index === cur) return 'bg-orange-400 animate-pulse'
  return 'bg-gray-200'
}

// ---- markdown renderer ----
const renderedMarkdown = computed(() => {
  if (!activeFileContent.value) return ''
  return renderMarkdown(activeFileContent.value)
})

function renderMarkdown(md) {
  // Escape HTML first
  let html = md
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  // Code blocks (must come before inline code)
  html = html.replace(/```[\w]*\n([\s\S]*?)```/g, (_, code) =>
    `<pre><code>${code}</code></pre>`)

  // Headings
  html = html
    .replace(/^#{6} (.+)$/gm, '<h6>$1</h6>')
    .replace(/^#{5} (.+)$/gm, '<h5>$1</h5>')
    .replace(/^#{4} (.+)$/gm, '<h4>$1</h4>')
    .replace(/^#{3} (.+)$/gm, '<h3>$1</h3>')
    .replace(/^## (.+)$/gm, '<h2>$1</h2>')
    .replace(/^# (.+)$/gm, '<h1>$1</h1>')

  // Inline
  html = html
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.+?)\*/g, '<em>$1</em>')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>')
    .replace(/^> (.+)$/gm, '<blockquote>$1</blockquote>')
    .replace(/^---$/gm, '<hr/>')

  // Lists
  html = html.replace(/^[-*] (.+)$/gm, '<li>$1</li>')
  html = html.replace(/^(\d+)\. (.+)$/gm, '<li>$2</li>')
  html = html.replace(/(<li>[\s\S]*?<\/li>)/g, '<ul>$1</ul>')

  // Tables
  html = html.replace(/^\|(.+)\|$/gm, (row) => {
    const cells = row.split('|').filter(c => c.trim() && !/^[-: ]+$/.test(c.trim()))
    if (!cells.length) return row
    return '<tr>' + cells.map(c => `<td>${c.trim()}</td>`).join('') + '</tr>'
  })
  html = html.replace(/(<tr>[\s\S]*?<\/tr>)/g, '<table>$1</table>')

  // Paragraphs
  html = html.replace(/\n\n+/g, '</p><p>')
  html = '<p>' + html + '</p>'
  html = html.replace(/<p>(<h[1-6]>)/g, '$1')
  html = html.replace(/(<\/h[1-6]>)<\/p>/g, '$1')
  html = html.replace(/<p>(<pre>)/g, '$1')
  html = html.replace(/(<\/pre>)<\/p>/g, '$1')
  html = html.replace(/<p>(<hr\/>)/g, '$1')
  html = html.replace(/<p>(<ul>)/g, '$1')
  html = html.replace(/(<\/ul>)<\/p>/g, '$1')
  html = html.replace(/<p>(<table>)/g, '$1')
  html = html.replace(/(<\/table>)<\/p>/g, '$1')
  html = html.replace(/<p><\/p>/g, '')

  return html
}

// ---- helpers ----
function statusType(s) {
  return { pending: 'info', running: 'warning', completed: 'success', failed: 'danger' }[s] || 'info'
}
function statusLabel(s) {
  return { pending: '等待中', running: '执行中', completed: '已完成', failed: '已中断' }[s] || s
}
function parseOptions(str) {
  try { return JSON.parse(str || '{}') } catch { return {} }
}
function formatTime(t) {
  if (!t) return ''
  return new Date(t).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

// ---- Claude version management ----
const claudeDrawerVisible = ref(false)
const claudeInfo = ref(null)
const loadingClaudeInfo = ref(false)
const modelHealth = ref(null)
const testingModel = ref(false)
const updating = ref(false)
const updateLogs = ref([])
const updateResult = ref(null)
const updateLogEl = ref(null)

function onClaudeDrawerOpen() {
  if (!claudeInfo.value) loadClaudeVersion()
}

async function loadClaudeVersion() {
  loadingClaudeInfo.value = true
  try {
    const res = await fetch(`${API_BASE}/claude/version?password=${encodeURIComponent(savedPassword)}`)
    if (res.ok) claudeInfo.value = await res.json()
  } catch { ElMessage.error('获取版本信息失败') }
  finally { loadingClaudeInfo.value = false }
}

async function testModel() {
  testingModel.value = true
  modelHealth.value = null
  try {
    const res = await fetch(`${API_BASE}/claude/test?password=${encodeURIComponent(savedPassword)}`)
    if (res.ok) modelHealth.value = await res.json()
    else ElMessage.error('测试请求失败')
  } catch { ElMessage.error('测试请求失败') }
  finally { testingModel.value = false }
}

function startUpdate() {
  if (updating.value) return
  updating.value = true
  updateLogs.value = []
  updateResult.value = null

  const url = `${API_BASE}/claude/update/stream?password=${encodeURIComponent(savedPassword)}`
  const es = new EventSource(url)

  es.addEventListener('log', (e) => {
    try {
      const data = JSON.parse(e.data)
      updateLogs.value.push(data.line)
      nextTick(() => {
        if (updateLogEl.value) updateLogEl.value.scrollTop = updateLogEl.value.scrollHeight
      })
    } catch { /* ignore */ }
  })

  es.addEventListener('done', (e) => {
    es.close()
    updating.value = false
    try {
      const data = JSON.parse(e.data)
      if (data.error) {
        updateResult.value = { success: false, message: data.error }
      } else {
        const msg = data.new_version && data.new_version !== data.old_version
          ? `${data.old_version} → ${data.new_version}`
          : `当前版本: ${data.new_version || '未知'}`
        updateResult.value = { success: true, message: msg }
        claudeInfo.value = null // force reload
      }
    } catch { updateResult.value = { success: true, message: '完成' } }
  })

  es.onerror = () => {
    es.close()
    updating.value = false
    if (!updateResult.value) {
      updateResult.value = { success: false, message: '连接中断' }
    }
  }
}

// ---- auto-refresh ----
let refreshTimer = null
function startAutoRefresh() {
  refreshTimer = setInterval(async () => {
    if (tasks.value.some(t => t.status === 'running')) {
      await loadTasks()
      if (selectedTask.value?.status === 'running') {
        await loadFiles()
        if (activeTab.value === 'logs') await loadLogs()
      }
    }
  }, 4000)
}

onMounted(() => {
  const pw = getPassword()
  if (pw) { savedPassword = pw; authenticated.value = true; loadTasks() }
  startAutoRefresh()
})
onUnmounted(() => clearInterval(refreshTimer))
</script>

<style scoped>
.markdown-body :deep(h1) { font-size: 1.4rem; font-weight: 700; margin: 1rem 0 0.5rem; border-bottom: 2px solid #e5e7eb; padding-bottom: 0.3rem; }
.markdown-body :deep(h2) { font-size: 1.2rem; font-weight: 600; margin: 0.9rem 0 0.4rem; border-bottom: 1px solid #e5e7eb; padding-bottom: 0.2rem; }
.markdown-body :deep(h3) { font-size: 1.05rem; font-weight: 600; margin: 0.7rem 0 0.25rem; }
.markdown-body :deep(h4), .markdown-body :deep(h5), .markdown-body :deep(h6) { font-weight: 600; margin: 0.5rem 0 0.2rem; }
.markdown-body :deep(p) { margin: 0.4rem 0; line-height: 1.65; font-size: 0.875rem; }
.markdown-body :deep(code) { background: #f3f4f6; padding: 0.1rem 0.3rem; border-radius: 3px; font-family: monospace; font-size: 0.82em; color: #be185d; }
.markdown-body :deep(pre) { background: #1e293b; color: #e2e8f0; padding: 0.8rem; border-radius: 6px; overflow-x: auto; margin: 0.6rem 0; }
.markdown-body :deep(pre) code { background: none; padding: 0; color: inherit; font-size: 0.8rem; }
.markdown-body :deep(ul), .markdown-body :deep(ol) { padding-left: 1.5rem; margin: 0.4rem 0; }
.markdown-body :deep(li) { margin: 0.2rem 0; line-height: 1.5; font-size: 0.875rem; }
.markdown-body :deep(blockquote) { border-left: 3px solid #8b5cf6; padding-left: 0.75rem; color: #6b7280; margin: 0.4rem 0; font-style: italic; }
.markdown-body :deep(hr) { border: none; border-top: 1px solid #e5e7eb; margin: 0.8rem 0; }
.markdown-body :deep(a) { color: #7c3aed; text-decoration: underline; }
.markdown-body :deep(strong) { font-weight: 700; }
.markdown-body :deep(em) { font-style: italic; }
.markdown-body :deep(table) { border-collapse: collapse; width: 100%; margin: 0.5rem 0; font-size: 0.85rem; }
.markdown-body :deep(td), .markdown-body :deep(th) { border: 1px solid #e5e7eb; padding: 0.35rem 0.6rem; }
.markdown-body :deep(tr:nth-child(even)) { background: #f9fafb; }
</style>
