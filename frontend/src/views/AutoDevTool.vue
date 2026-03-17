<template>
  <!-- ===== Password Gate ===== -->
  <div v-if="!authenticated" class="min-h-screen bg-gradient-to-br from-slate-900 via-purple-950 to-slate-900 flex items-center justify-center p-4">
    <div class="w-full max-w-sm">
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-purple-600/30 border border-purple-500/40 mb-4">
          <el-icon class="text-3xl text-purple-400"><MagicStick /></el-icon>
        </div>
        <h1 class="text-2xl font-bold text-white">AutoDev AI</h1>
        <p class="text-slate-400 text-sm mt-1">AI 驱动的自动化开发助手</p>
      </div>
      <div class="bg-slate-800/60 backdrop-blur border border-slate-700/50 rounded-2xl p-6 shadow-2xl">
        <el-input
          v-model="passwordInput"
          type="password"
          placeholder="访问密码"
          show-password
          size="large"
          class="mb-4"
          @keyup.enter="login"
        >
          <template #prefix><el-icon class="text-slate-400"><Lock /></el-icon></template>
        </el-input>
        <el-button type="primary" size="large" class="w-full" :loading="loggingIn" @click="login"
          style="background: linear-gradient(135deg, #7c3aed, #6d28d9); border: none; height: 44px; border-radius: 10px; font-size: 15px;">
          进入工作台
        </el-button>
      </div>
    </div>
  </div>

  <!-- ===== Main Workspace ===== -->
  <div v-else class="autodev-workspace">

    <!-- Top Bar -->
    <div class="topbar">
      <div class="flex items-center gap-3">
        <div class="flex items-center justify-center w-8 h-8 rounded-lg bg-purple-600/30 border border-purple-500/40">
          <el-icon class="text-purple-400 text-sm"><MagicStick /></el-icon>
        </div>
        <span class="font-bold text-white text-base">AutoDev AI</span>
        <el-badge v-if="runningCount" :value="runningCount" type="warning" class="ml-1">
          <span class="text-xs text-slate-400">任务运行中</span>
        </el-badge>
      </div>
      <div class="flex items-center gap-2">
        <el-button size="small" @click="claudeDrawerVisible = true"
          class="topbar-btn">
          <el-icon><SetUp /></el-icon>
          <span class="hidden sm:inline ml-1">Claude 管理</span>
        </el-button>
        <el-button size="small" @click="logout" class="topbar-btn">
          <el-icon><SwitchButton /></el-icon>
        </el-button>
      </div>
    </div>

    <!-- Main Layout -->
    <div class="main-layout">

      <!-- ===== Left Sidebar ===== -->
      <div class="sidebar">

        <!-- Submit Form -->
        <div class="sidebar-card mb-3">
          <div class="sidebar-card-header">
            <el-icon class="text-purple-400 text-sm"><VideoPlay /></el-icon>
            <span>{{ resumeTaskId ? '断点恢复' : '新建任务' }}</span>
            <el-button v-if="resumeTaskId" size="small" text class="ml-auto !text-slate-400 !text-xs" @click="cancelResume">取消</el-button>
          </div>
          <el-input
            v-model="newTask.description"
            type="textarea"
            :rows="3"
            placeholder="描述你的任务，例如：写一份 Redis 集群最佳实践文档"
            maxlength="500"
            show-word-limit
            :disabled="!!resumeTaskId"
            class="mb-3"
          />
          <div v-if="resumeTaskId" class="flex items-center gap-2 mb-3">
            <span class="text-xs text-slate-400 shrink-0">从阶段</span>
            <el-input-number v-model="newTask.resumeFrom" :min="1" :max="6" size="small" class="flex-1" />
            <span class="text-xs text-purple-400 shrink-0">{{ phaseLabel(newTask.resumeFrom) }}</span>
          </div>
          <div class="flex gap-2 mb-3 flex-wrap">
            <el-checkbox v-model="newTask.publish" size="small" border class="!text-xs">
              <span class="text-xs text-slate-300">publish</span>
            </el-checkbox>
            <el-checkbox v-model="newTask.build" size="small" border class="!text-xs">
              <span class="text-xs text-slate-300">build</span>
            </el-checkbox>
            <el-checkbox v-model="newTask.push" size="small" border class="!text-xs">
              <span class="text-xs text-slate-300">push</span>
            </el-checkbox>
          </div>
          <el-button
            type="primary"
            :loading="submitting"
            :disabled="!newTask.description.trim()"
            @click="submitTask"
            class="w-full submit-btn"
          >
            <el-icon v-if="!submitting"><VideoPlay /></el-icon>
            <span class="ml-1">{{ resumeTaskId ? '恢复执行' : '开始执行' }}</span>
          </el-button>
        </div>

        <!-- Task List -->
        <div class="sidebar-card flex-1 overflow-hidden flex flex-col">
          <div class="sidebar-card-header">
            <el-icon class="text-slate-400 text-sm"><List /></el-icon>
            <span>任务列表</span>
            <el-button size="small" :loading="loadingList" @click="loadTasks" circle text class="ml-auto !text-slate-400">
              <el-icon><Refresh /></el-icon>
            </el-button>
          </div>
          <div v-if="tasks.length === 0" class="text-center text-slate-500 py-8">
            <el-icon class="text-3xl mb-2 text-slate-600"><DocumentRemove /></el-icon>
            <p class="text-sm">暂无任务</p>
          </div>
          <div v-else class="task-list">
            <div
              v-for="task in tasks"
              :key="task.id"
              class="task-item"
              :class="{ 'task-item--active': selectedTask?.id === task.id }"
              @click="selectTask(task)"
            >
              <div class="flex items-start gap-2">
                <div class="task-status-dot" :class="`task-status-dot--${task.status}`" />
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium text-slate-200 truncate leading-tight" :title="task.description">
                    {{ task.description }}
                  </p>
                  <p class="text-xs text-slate-500 mt-0.5">{{ formatTime(task.created_at) }}</p>
                  <!-- Running phase -->
                  <div v-if="task.status === 'running' && task.autodev_state" class="mt-1.5">
                    <div class="text-xs text-amber-400 flex items-center gap-1 mb-1">
                      <span class="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse inline-block"></span>
                      {{ task.autodev_state.phase_label || '执行中' }}
                    </div>
                    <div class="flex gap-0.5">
                      <div v-for="(_, i) in PHASE_NAMES" :key="i"
                        class="flex-1 h-1 rounded-full transition-colors"
                        :class="miniPhaseClass(i, task.autodev_state)" />
                    </div>
                  </div>
                </div>
                <div class="flex flex-col items-end gap-1 shrink-0">
                  <el-tag :type="statusType(task.status)" size="small" effect="dark" class="!text-xs">
                    {{ statusLabel(task.status) }}
                  </el-tag>
                  <div class="flex gap-1">
                    <el-tooltip v-if="task.status === 'running'" content="中断" placement="top">
                      <el-button size="small" type="warning" circle plain @click.stop="stopTask(task)" class="!w-6 !h-6 !p-0">
                        <el-icon class="text-xs"><VideoPause /></el-icon>
                      </el-button>
                    </el-tooltip>
                    <el-tooltip v-if="task.status === 'failed'" content="断点恢复" placement="top">
                      <el-button size="small" type="success" circle plain @click.stop="startResume(task)" class="!w-6 !h-6 !p-0">
                        <el-icon class="text-xs"><RefreshRight /></el-icon>
                      </el-button>
                    </el-tooltip>
                    <el-tooltip content="删除" placement="top">
                      <el-button size="small" type="danger" circle plain @click.stop="deleteTask(task)" class="!w-6 !h-6 !p-0">
                        <el-icon class="text-xs"><Delete /></el-icon>
                      </el-button>
                    </el-tooltip>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- ===== Right Content Panel ===== -->
      <div class="content-panel">
        <!-- Empty State -->
        <div v-if="!selectedTask" class="flex items-center justify-center h-full text-slate-500">
          <div class="text-center">
            <div class="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-slate-800/50 border border-slate-700/50 mb-4">
              <el-icon class="text-4xl text-slate-600"><Pointer /></el-icon>
            </div>
            <p class="text-slate-400 font-medium">选择一个任务查看详情</p>
            <p class="text-slate-600 text-sm mt-1">或提交新任务开始执行</p>
          </div>
        </div>

        <div v-else class="h-full flex flex-col">

          <!-- Task Header -->
          <div class="content-header">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <el-tag :type="statusType(selectedTask.status)" effect="dark" size="small">
                  <el-icon v-if="selectedTask.status === 'running'" class="is-loading mr-1"><Loading /></el-icon>
                  {{ statusLabel(selectedTask.status) }}
                </el-tag>
                <span class="text-xs text-slate-500 font-mono">#{{ selectedTask.id }}</span>
              </div>
              <h2 class="text-base font-semibold text-white leading-tight truncate">{{ selectedTask.description }}</h2>
            </div>
            <div class="flex items-center gap-2 flex-wrap">
              <el-button size="small" :loading="loadingDetail" @click="refreshDetail" class="action-btn" plain>
                <el-icon><Refresh /></el-icon>
              </el-button>
              <el-button size="small" type="primary" plain :loading="downloading" @click="downloadTask" class="action-btn">
                <el-icon><Download /></el-icon> 下载
              </el-button>
              <el-button v-if="hasSite" size="small" type="success" plain @click="previewSite" class="action-btn">
                <el-icon><Monitor /></el-icon> 预览
              </el-button>
              <el-button v-if="selectedTask.status === 'running'" size="small" type="warning" plain @click="stopTask(selectedTask)" class="action-btn">
                <el-icon><VideoPause /></el-icon> 中断
              </el-button>
              <el-button v-if="selectedTask.status === 'failed'" size="small" type="success" plain @click="startResume(selectedTask)" class="action-btn">
                <el-icon><RefreshRight /></el-icon> 恢复
              </el-button>
            </div>
          </div>

          <!-- Phase Steps -->
          <div v-if="taskState" class="phase-steps">
            <div v-for="(ph, i) in PHASE_NAMES" :key="i" class="phase-step">
              <div class="phase-step-dot" :class="phaseStepClass(i, taskState)">
                <el-icon v-if="isPhaseCompleted(i, taskState)" class="text-xs"><Check /></el-icon>
                <el-icon v-else-if="isPhaseActive(i, taskState)" class="text-xs is-loading"><Loading /></el-icon>
                <span v-else class="text-xs">{{ i + 1 }}</span>
              </div>
              <span class="phase-step-label" :class="phaseStepLabelClass(i, taskState)">{{ ph.short }}</span>
              <div v-if="i < PHASE_NAMES.length - 1" class="phase-connector"
                :class="isPhaseCompleted(i, taskState) ? 'bg-green-500' : 'bg-slate-700'" />
            </div>
          </div>

          <!-- Content Tabs -->
          <div class="content-tabs">
            <div class="tab-bar">
              <button
                class="tab-btn"
                :class="{ 'tab-btn--active': activeTab === 'result' }"
                @click="activeTab = 'result'"
              >
                <el-icon><Document /></el-icon>
                结果文档
              </button>
              <button
                class="tab-btn"
                :class="{ 'tab-btn--active': activeTab === 'files' }"
                @click="activeTab = 'files'"
              >
                <el-icon><FolderOpened /></el-icon>
                全部文件
                <span v-if="taskFiles.length" class="tab-badge">{{ taskFiles.length }}</span>
              </button>
              <button
                class="tab-btn"
                :class="{ 'tab-btn--active': activeTab === 'logs' }"
                @click="switchToLogs"
              >
                <el-icon><Monitor /></el-icon>
                执行日志
                <span v-if="selectedTask.status === 'running'" class="ml-1 w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse inline-block" />
              </button>
            </div>

            <!-- ===== Result Tab ===== -->
            <div v-show="activeTab === 'result'" class="tab-content">
              <div v-if="!resultContent && !loadingDetail" class="empty-content">
                <el-icon class="text-4xl text-slate-600 mb-3"><Document /></el-icon>
                <p class="text-slate-400">{{ selectedTask.status === 'running' ? 'AI 正在生成结果文档…' : '暂无结果文档' }}</p>
                <p v-if="selectedTask.status === 'running'" class="text-slate-500 text-sm mt-1">执行完成后自动显示</p>
              </div>
              <div v-else-if="loadingDetail" class="empty-content">
                <el-icon class="text-3xl text-purple-400 is-loading mb-2"><Loading /></el-icon>
                <p class="text-slate-400 text-sm">加载中…</p>
              </div>
              <div v-else class="result-content">
                <!-- Toolbar -->
                <div class="result-toolbar">
                  <span class="text-xs text-slate-500">RESULT.md</span>
                  <div class="flex gap-2">
                    <el-button size="small" text class="!text-slate-400" @click="openFullscreen">
                      <el-icon><FullScreen /></el-icon> 全屏
                    </el-button>
                    <el-button size="small" text class="!text-slate-400" @click="copyResult">
                      <el-icon><CopyDocument /></el-icon> 复制
                    </el-button>
                  </div>
                </div>
                <div class="markdown-view" v-html="renderedResult" />
              </div>
            </div>

            <!-- ===== Files Tab ===== -->
            <div v-show="activeTab === 'files'" class="tab-content">
              <div v-if="taskFiles.length === 0" class="empty-content">
                <el-icon class="text-4xl text-slate-600 mb-3"><FolderOpened /></el-icon>
                <p class="text-slate-400">{{ selectedTask.status === 'running' ? '文件生成后自动显示…' : '暂无文件' }}</p>
              </div>
              <div v-else class="files-layout">
                <!-- File Tree -->
                <div class="file-tree">
                  <div
                    v-for="file in taskFiles"
                    :key="file.path"
                    class="file-item"
                    :class="{ 'file-item--active': activeFilePath === file.path }"
                    @click="loadFile(file.path)"
                    :title="file.path"
                  >
                    <span class="file-icon">{{ fileIcon(file.name) }}</span>
                    <span class="file-name">{{ file.name }}</span>
                  </div>
                </div>
                <!-- File Content -->
                <div class="file-content-area">
                  <div v-if="!activeFilePath" class="empty-content">
                    <p class="text-slate-500 text-sm">点击左侧文件查看内容</p>
                  </div>
                  <div v-else-if="loadingFile" class="empty-content">
                    <el-icon class="text-2xl text-purple-400 is-loading"><Loading /></el-icon>
                  </div>
                  <div v-else class="h-full flex flex-col">
                    <div class="result-toolbar">
                      <span class="text-xs text-slate-500 font-mono">{{ activeFilePath }}</span>
                      <div class="flex gap-2">
                        <el-button v-if="activeFilePath?.endsWith('.md')" size="small" text class="!text-slate-400" @click="openFullscreenFile">
                          <el-icon><FullScreen /></el-icon>
                        </el-button>
                        <el-button size="small" text class="!text-slate-400" @click="copyCurrentFile">
                          <el-icon><CopyDocument /></el-icon>
                        </el-button>
                      </div>
                    </div>
                    <div class="flex-1 overflow-auto">
                      <div v-if="activeFilePath?.endsWith('.md')"
                        class="markdown-view p-4"
                        v-html="renderedActiveFile"
                      />
                      <pre v-else class="code-view">{{ activeFileContent }}</pre>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- ===== Logs Tab ===== -->
            <div v-show="activeTab === 'logs'" class="tab-content">
              <div class="log-toolbar">
                <el-select v-model="activeLogPhase" size="small" class="w-36" @change="loadLogs">
                  <el-option v-for="ph in availableLogPhases" :key="ph" :label="ph" :value="ph" />
                </el-select>
                <span class="text-xs text-slate-500">{{ logLineCount }} 行</span>
                <div class="flex gap-2 ml-auto">
                  <el-button size="small" :loading="loadingLogs" @click="loadLogs" plain class="action-btn">
                    <el-icon><Refresh /></el-icon>
                  </el-button>
                  <el-button size="small" plain @click="scrollLogsToBottom" class="action-btn">
                    <el-icon><Bottom /></el-icon>
                  </el-button>
                </div>
              </div>
              <div v-if="!logContent" class="empty-content bg-slate-950">
                <el-icon class="text-3xl text-slate-700 mb-2"><Monitor /></el-icon>
                <p class="text-slate-500">{{ selectedTask.status === 'running' ? '等待日志输出…' : '暂无日志' }}</p>
              </div>
              <pre v-else ref="logEl" class="log-terminal">{{ logContent }}</pre>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== Fullscreen Document Dialog ===== -->
    <el-dialog
      v-model="fullscreenVisible"
      :title="fullscreenTitle"
      fullscreen
      class="fullscreen-dialog"
      destroy-on-close
    >
      <div class="fullscreen-content markdown-view" v-html="fullscreenHtml" />
    </el-dialog>

    <!-- ===== Claude 管理 Drawer ===== -->
    <el-drawer v-model="claudeDrawerVisible" title="Claude CLI 管理" direction="rtl" size="480px" @open="onClaudeDrawerOpen">
      <div class="space-y-4 p-2">
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="font-semibold text-sm">当前环境</span>
              <el-button size="small" :loading="loadingClaudeInfo" @click="loadClaudeVersion" circle plain>
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>
          </template>
          <div v-if="loadingClaudeInfo" class="text-center py-4"><el-icon class="is-loading text-purple-500 text-xl"><Loading /></el-icon></div>
          <div v-else-if="claudeInfo" class="space-y-2">
            <div class="flex items-center justify-between py-1.5 border-b">
              <span class="text-sm text-gray-500">Claude CLI</span>
              <div class="flex items-center gap-2">
                <el-tag :type="claudeInfo.available ? 'success' : 'danger'" size="small" effect="dark">{{ claudeInfo.available ? '已安装' : '未安装' }}</el-tag>
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
          <div v-else class="text-center text-gray-400 py-4 text-sm">点击刷新按钮获取版本信息</div>
        </el-card>

        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="font-semibold text-sm">模型连通性测试</span>
              <el-button size="small" :loading="testingModel" @click="testModel" plain>
                <el-icon><Connection /></el-icon> 测试
              </el-button>
            </div>
          </template>
          <div v-if="!modelHealth && !testingModel" class="text-center text-gray-400 py-4 text-sm">点击测试按钮检查 API 连通性</div>
          <div v-else-if="testingModel" class="text-center py-4">
            <el-icon class="is-loading text-purple-500 text-xl"><Loading /></el-icon>
            <p class="text-xs text-gray-400 mt-2">正在发送测试请求…</p>
          </div>
          <div v-else-if="modelHealth" class="space-y-2">
            <div class="flex items-center justify-between py-1.5 border-b">
              <span class="text-sm text-gray-500">状态</span>
              <el-tag :type="modelHealth.ok ? 'success' : 'danger'" size="small" effect="dark">{{ modelHealth.ok ? '✅ 连通' : '❌ 不可用' }}</el-tag>
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
              <el-tag :type="modelHealth.has_token ? 'success' : 'danger'" size="small">{{ modelHealth.has_token ? '已配置' : '未配置' }}</el-tag>
            </div>
            <div v-if="modelHealth.ok" class="flex items-center justify-between py-1.5 border-b">
              <span class="text-sm text-gray-500">响应延迟</span>
              <span class="text-sm font-mono" :class="modelHealth.latency_ms < 3000 ? 'text-green-600' : 'text-yellow-600'">{{ modelHealth.latency_ms }} ms</span>
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

        <el-card shadow="never">
          <template #header><span class="font-semibold text-sm">更新 Claude Code CLI</span></template>
          <div class="space-y-3">
            <div class="bg-gray-50 rounded p-3 text-xs text-gray-600">
              <p>执行命令：</p>
              <code class="font-mono text-purple-700">npm install -g @anthropic-ai/claude-code@latest</code>
            </div>
            <el-button type="primary" class="w-full" :loading="updating" :disabled="updating" @click="startUpdate">
              <el-icon><Upload /></el-icon>
              <span class="ml-1">{{ updating ? '更新中…' : '立即更新到最新版本' }}</span>
            </el-button>
            <div v-if="updateLogs.length">
              <div class="flex items-center justify-between mb-1">
                <span class="text-xs text-gray-500">更新输出</span>
                <el-button size="small" text @click="updateLogs = []">清空</el-button>
              </div>
              <pre ref="updateLogEl" class="text-xs bg-gray-900 text-green-400 p-3 rounded overflow-auto max-h-[300px] whitespace-pre-wrap break-all font-mono leading-5">{{ updateLogs.join('\n') }}</pre>
            </div>
            <el-result v-if="updateResult" :icon="updateResult.success ? 'success' : 'error'" :title="updateResult.success ? '更新成功' : '更新失败'" :sub-title="updateResult.message">
              <template #extra>
                <el-button size="small" @click="loadClaudeVersion">刷新版本信息</el-button>
              </template>
            </el-result>
          </div>
        </el-card>
      </div>
    </el-drawer>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js'
import {
  Lock, MagicStick, VideoPlay, VideoPause, Refresh, RefreshRight,
  Delete, Download, DocumentRemove, Pointer, Loading, Document,
  FolderOpened, Bottom, SetUp, Upload, Monitor, Connection,
  SwitchButton, List, Check, FullScreen, CopyDocument
} from '@element-plus/icons-vue'

// ---- markdown-it setup ----
const md = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
  highlight(str, lang) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return `<pre class="hljs-block"><code class="hljs language-${lang}">${hljs.highlight(str, { language: lang, ignoreIllegals: true }).value}</code></pre>`
      } catch {}
    }
    return `<pre class="hljs-block"><code class="hljs">${md.utils.escapeHtml(str)}</code></pre>`
  }
})

function renderMd(content) {
  if (!content) return ''
  return md.render(content)
}

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

function phaseLabel(n) { return PHASE_NAMES[n - 1]?.label || `阶段 ${n}` }

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
  if (state?.last_completed != null) resumeFrom = Math.floor(state.last_completed) + 2
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
const activeTab = ref('result')
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
const resultContent = ref('')

// fullscreen
const fullscreenVisible = ref(false)
const fullscreenTitle = ref('')
const fullscreenHtml = ref('')

const renderedResult = computed(() => renderMd(resultContent.value))
const renderedActiveFile = computed(() => renderMd(activeFileContent.value))

function openFullscreen() {
  fullscreenTitle.value = 'RESULT.md'
  fullscreenHtml.value = renderedResult.value
  fullscreenVisible.value = true
}

function openFullscreenFile() {
  fullscreenTitle.value = activeFilePath.value?.split('/').pop() || '文件'
  fullscreenHtml.value = renderedActiveFile.value
  fullscreenVisible.value = true
}

async function copyResult() {
  try {
    await navigator.clipboard.writeText(resultContent.value)
    ElMessage.success('已复制')
  } catch { ElMessage.error('复制失败') }
}

async function copyCurrentFile() {
  try {
    await navigator.clipboard.writeText(activeFileContent.value)
    ElMessage.success('已复制')
  } catch { ElMessage.error('复制失败') }
}

async function selectTask(task) {
  selectedTask.value = task
  taskState.value = task.autodev_state || null
  activeTab.value = 'result'
  activeFilePath.value = null
  activeFileContent.value = ''
  logContent.value = ''
  taskFiles.value = []
  taskHasSite.value = false
  resultContent.value = ''
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
    // Auto-load RESULT.md
    const result = taskFiles.value.find(f => f.name === 'RESULT.md')
    if (result) {
      await loadResultFile(result.path)
    } else if (!activeFilePath.value && taskFiles.value.length) {
      await loadFile(taskFiles.value[0].path)
    }
  }
}

async function loadResultFile(path) {
  try {
    const res = await fetch(
      `${API_BASE}/tasks/${selectedTask.value.id}/file?password=${encodeURIComponent(savedPassword)}&path=${encodeURIComponent(path)}`
    )
    if (res.ok) {
      const data = await res.json()
      resultContent.value = data.content || ''
    }
  } catch {}
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

function switchToLogs() {
  activeTab.value = 'logs'
  loadLogs()
}

watch(activeTab, tab => {
  if (tab === 'logs') loadLogs()
})

async function downloadTask() {
  if (!selectedTask.value) return
  downloading.value = true
  try {
    const url = `${API_BASE}/tasks/${selectedTask.value.id}/download?password=${encodeURIComponent(savedPassword)}`
    const a = document.createElement('a')
    a.href = url; a.download = ''
    document.body.appendChild(a); a.click(); document.body.removeChild(a)
    ElMessage.success('开始下载')
  } catch { ElMessage.error('下载失败') }
  finally { setTimeout(() => { downloading.value = false }, 1000) }
}

const hasSite = computed(() => taskHasSite.value)

function previewSite() {
  if (!selectedTask.value) return
  window.open(`${API_BASE}/tasks/${selectedTask.value.id}/site/index.html?password=${encodeURIComponent(savedPassword)}`, '_blank')
}

// ---- phase helpers ----
function isPhaseCompleted(i, state) {
  if (!state) return false
  if (state.status === 'finished') return true
  return i <= (state.last_completed ?? -1)
}
function isPhaseActive(i, state) {
  if (!state || state.status === 'finished') return false
  return i === (state.current_phase ?? -1)
}
function phaseStepClass(i, state) {
  if (!state) return 'phase-step-dot--pending'
  if (isPhaseCompleted(i, state)) return 'phase-step-dot--done'
  if (isPhaseActive(i, state)) return 'phase-step-dot--active'
  return 'phase-step-dot--pending'
}
function phaseStepLabelClass(i, state) {
  if (!state) return 'text-slate-600'
  if (isPhaseCompleted(i, state)) return 'text-green-400'
  if (isPhaseActive(i, state)) return 'text-amber-400'
  return 'text-slate-600'
}
function miniPhaseClass(i, state) {
  if (!state) return 'bg-slate-700'
  if (isPhaseCompleted(i, state)) return 'bg-green-500'
  if (isPhaseActive(i, state)) return 'bg-amber-400'
  return 'bg-slate-700'
}

// ---- file helpers ----
function fileIcon(name) {
  if (name.endsWith('.md')) return '📄'
  if (name.endsWith('.json')) return '📋'
  if (name.endsWith('.js') || name.endsWith('.ts')) return '📜'
  if (name.endsWith('.py')) return '🐍'
  if (name.endsWith('.go')) return '🔵'
  if (name.endsWith('.html')) return '🌐'
  if (name.endsWith('.css')) return '🎨'
  if (name.endsWith('.sh')) return '⚙️'
  if (name.endsWith('.log')) return '📝'
  return '📄'
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

function onClaudeDrawerOpen() { if (!claudeInfo.value) loadClaudeVersion() }

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
      nextTick(() => { if (updateLogEl.value) updateLogEl.value.scrollTop = updateLogEl.value.scrollHeight })
    } catch {}
  })
  es.addEventListener('done', (e) => {
    es.close(); updating.value = false
    try {
      const data = JSON.parse(e.data)
      if (data.error) { updateResult.value = { success: false, message: data.error } }
      else {
        const msg = data.new_version && data.new_version !== data.old_version
          ? `${data.old_version} → ${data.new_version}` : `当前版本: ${data.new_version || '未知'}`
        updateResult.value = { success: true, message: msg }
        claudeInfo.value = null
      }
    } catch { updateResult.value = { success: true, message: '完成' } }
  })
  es.onerror = () => {
    es.close(); updating.value = false
    if (!updateResult.value) updateResult.value = { success: false, message: '连接中断' }
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

<style>
/* Import highlight.js theme */
@import 'highlight.js/styles/github-dark.css';
</style>

<style scoped>
/* ===== Workspace Layout ===== */
.autodev-workspace {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #0f172a;
  color: #e2e8f0;
  overflow: hidden;
}

/* ===== Top Bar ===== */
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 52px;
  padding: 0 16px;
  background: #1e293b;
  border-bottom: 1px solid #334155;
  flex-shrink: 0;
  z-index: 10;
}
.topbar-btn {
  background: transparent !important;
  border-color: #475569 !important;
  color: #94a3b8 !important;
}
.topbar-btn:hover {
  border-color: #7c3aed !important;
  color: #a78bfa !important;
}

/* ===== Main Layout ===== */
.main-layout {
  display: flex;
  flex: 1;
  overflow: hidden;
}

/* ===== Sidebar ===== */
.sidebar {
  width: 320px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  padding: 12px;
  background: #1e293b;
  border-right: 1px solid #334155;
  overflow-y: auto;
  gap: 0;
}

.sidebar-card {
  background: #0f172a;
  border: 1px solid #334155;
  border-radius: 10px;
  padding: 12px;
  margin-bottom: 10px;
}

.sidebar-card-header {
  display: flex;
  align-items: center;
  gap-: 6px;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: #94a3b8;
  margin-bottom: 10px;
}

.submit-btn {
  background: linear-gradient(135deg, #7c3aed, #6d28d9) !important;
  border: none !important;
  border-radius: 8px !important;
  font-weight: 600 !important;
}

/* ===== Task List ===== */
.task-list {
  overflow-y: auto;
  flex: 1;
  margin: -4px;
  padding: 4px;
}

.task-item {
  padding: 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s;
  margin-bottom: 6px;
  border: 1px solid transparent;
}
.task-item:hover { background: #1e293b; border-color: #475569; }
.task-item--active { background: #1e1b4b; border-color: #7c3aed !important; }

.task-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-top: 5px;
  flex-shrink: 0;
}
.task-status-dot--running { background: #f59e0b; animation: pulse 1.5s infinite; }
.task-status-dot--completed { background: #22c55e; }
.task-status-dot--failed { background: #ef4444; }
.task-status-dot--pending { background: #64748b; }

/* ===== Content Panel ===== */
.content-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #0f172a;
}

.content-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #1e293b;
  border-bottom: 1px solid #334155;
  flex-shrink: 0;
}

.action-btn {
  background: transparent !important;
  border-color: #475569 !important;
  color: #94a3b8 !important;
  border-radius: 6px !important;
}
.action-btn:hover {
  border-color: #7c3aed !important;
  color: #a78bfa !important;
}

/* ===== Phase Steps ===== */
.phase-steps {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  background: #1e293b;
  border-bottom: 1px solid #334155;
  flex-shrink: 0;
}

.phase-step {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  flex: 1;
}

.phase-step-dot {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  z-index: 1;
  margin-bottom: 4px;
}
.phase-step-dot--done { background: #16a34a; color: white; }
.phase-step-dot--active { background: #d97706; color: white; box-shadow: 0 0 0 4px rgba(217, 119, 6, 0.2); }
.phase-step-dot--pending { background: #334155; color: #64748b; }

.phase-step-label {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.05em;
}

.phase-connector {
  position: absolute;
  top: 12px;
  left: calc(50% + 12px);
  right: calc(-50% + 12px);
  height: 2px;
}

/* ===== Content Tabs ===== */
.content-tabs {
  display: flex;
  flex-direction: column;
  flex: 1;
  overflow: hidden;
}

.tab-bar {
  display: flex;
  gap: 2px;
  padding: 8px 16px 0;
  background: #1e293b;
  border-bottom: 1px solid #334155;
  flex-shrink: 0;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 7px 14px;
  font-size: 13px;
  color: #64748b;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: all 0.15s;
  border-radius: 6px 6px 0 0;
  font-weight: 500;
}
.tab-btn:hover { color: #94a3b8; background: rgba(255,255,255,0.03); }
.tab-btn--active { color: #a78bfa; border-bottom-color: #7c3aed; }

.tab-badge {
  background: #334155;
  color: #94a3b8;
  border-radius: 10px;
  padding: 0 6px;
  font-size: 11px;
  font-weight: 600;
}

.tab-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* ===== Empty State ===== */
.empty-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 40px;
  color: #475569;
}

/* ===== Result View ===== */
.result-content {
  display: flex;
  flex-direction: column;
  flex: 1;
  overflow: hidden;
}

.result-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 16px;
  background: #1e293b;
  border-bottom: 1px solid #334155;
  flex-shrink: 0;
}

/* ===== Markdown View ===== */
.markdown-view {
  overflow-y: auto;
  padding: 24px 32px;
  flex: 1;
  color: #e2e8f0;
  font-size: 15px;
  line-height: 1.7;
}

.markdown-view :deep(h1) {
  font-size: 1.75rem;
  font-weight: 700;
  color: #f1f5f9;
  margin: 1.5rem 0 0.75rem;
  padding-bottom: 0.5rem;
  border-bottom: 2px solid #334155;
}
.markdown-view :deep(h2) {
  font-size: 1.4rem;
  font-weight: 600;
  color: #e2e8f0;
  margin: 1.25rem 0 0.6rem;
  padding-bottom: 0.3rem;
  border-bottom: 1px solid #334155;
}
.markdown-view :deep(h3) {
  font-size: 1.15rem;
  font-weight: 600;
  color: #cbd5e1;
  margin: 1rem 0 0.5rem;
}
.markdown-view :deep(h4), .markdown-view :deep(h5), .markdown-view :deep(h6) {
  font-weight: 600;
  color: #94a3b8;
  margin: 0.75rem 0 0.4rem;
}
.markdown-view :deep(p) {
  margin: 0.6rem 0;
  color: #cbd5e1;
}
.markdown-view :deep(a) { color: #a78bfa; text-decoration: underline; }
.markdown-view :deep(strong) { font-weight: 700; color: #f1f5f9; }
.markdown-view :deep(em) { font-style: italic; color: #94a3b8; }
.markdown-view :deep(code) {
  background: #1e293b;
  border: 1px solid #334155;
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.85em;
  color: #f472b6;
}
.markdown-view :deep(.hljs-block) {
  margin: 0.75rem 0;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid #334155;
}
.markdown-view :deep(.hljs-block code) {
  display: block;
  padding: 14px 16px;
  overflow-x: auto;
  background: #0d1117 !important;
  color: #e2e8f0;
  font-size: 0.85rem;
  line-height: 1.6;
  border: none;
  border-radius: 0;
}
.markdown-view :deep(pre) {
  background: #0d1117;
  border: 1px solid #334155;
  border-radius: 8px;
  padding: 14px 16px;
  overflow-x: auto;
  margin: 0.75rem 0;
}
.markdown-view :deep(pre code) {
  background: none;
  border: none;
  padding: 0;
  color: #e2e8f0;
  font-size: 0.85rem;
}
.markdown-view :deep(ul), .markdown-view :deep(ol) {
  padding-left: 1.75rem;
  margin: 0.5rem 0;
}
.markdown-view :deep(li) { margin: 0.3rem 0; color: #cbd5e1; }
.markdown-view :deep(blockquote) {
  border-left: 3px solid #7c3aed;
  padding: 8px 16px;
  margin: 0.75rem 0;
  background: #1e1b4b;
  border-radius: 0 6px 6px 0;
  color: #94a3b8;
}
.markdown-view :deep(hr) {
  border: none;
  border-top: 1px solid #334155;
  margin: 1.5rem 0;
}
.markdown-view :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 0.75rem 0;
  font-size: 0.9rem;
}
.markdown-view :deep(th) {
  background: #1e293b;
  color: #94a3b8;
  font-weight: 600;
  padding: 8px 12px;
  border: 1px solid #334155;
  text-align: left;
}
.markdown-view :deep(td) {
  padding: 8px 12px;
  border: 1px solid #334155;
  color: #cbd5e1;
}
.markdown-view :deep(tr:nth-child(even)) { background: #1e293b40; }
.markdown-view :deep(img) { max-width: 100%; border-radius: 6px; }

/* ===== Files Layout ===== */
.files-layout {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.file-tree {
  width: 200px;
  flex-shrink: 0;
  border-right: 1px solid #334155;
  overflow-y: auto;
  padding: 8px;
  background: #0f172a;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.1s;
  margin-bottom: 2px;
}
.file-item:hover { background: #1e293b; }
.file-item--active { background: #1e1b4b !important; }

.file-icon { font-size: 13px; flex-shrink: 0; }
.file-name { font-size: 12px; color: #94a3b8; truncate: true; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-item--active .file-name { color: #a78bfa; font-weight: 500; }

.file-content-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
}

.code-view {
  padding: 16px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #e2e8f0;
  white-space: pre-wrap;
  word-break: break-all;
  overflow-y: auto;
  flex: 1;
  background: #0f172a;
  margin: 0;
}

/* ===== Log Terminal ===== */
.log-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #1e293b;
  border-bottom: 1px solid #334155;
  flex-shrink: 0;
}

.log-terminal {
  flex: 1;
  overflow-y: auto;
  padding: 12px 16px;
  background: #020617;
  color: #4ade80;
  font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 12.5px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}

/* ===== Fullscreen Dialog ===== */
.fullscreen-dialog :deep(.el-dialog__body) {
  padding: 0;
  background: #0f172a;
  overflow-y: auto;
  height: calc(100vh - 55px);
}
.fullscreen-dialog :deep(.el-dialog__header) {
  background: #1e293b;
  border-bottom: 1px solid #334155;
  padding: 14px 20px;
  margin: 0;
}
.fullscreen-dialog :deep(.el-dialog__title) {
  color: #e2e8f0;
  font-weight: 600;
}
.fullscreen-content {
  max-width: 900px;
  margin: 0 auto;
  padding: 32px 40px;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

/* Responsive */
@media (max-width: 768px) {
  .sidebar { width: 100%; border-right: none; border-bottom: 1px solid #334155; max-height: 50vh; }
  .main-layout { flex-direction: column; }
  .content-panel { flex: 1; min-height: 50vh; }
  .markdown-view { padding: 16px; }
}
</style>
