<template>
  <!-- ===== Password Gate ===== -->
  <div v-if="!authenticated" class="min-h-screen flex items-center justify-center p-4"
    :class="isDark ? 'bg-gradient-to-br from-slate-900 via-purple-950 to-slate-900' : 'bg-gradient-to-br from-slate-100 via-purple-50 to-slate-100'">
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
  <div v-else class="autodev-workspace" :class="{ 'theme-light': !isDark }">

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
        <el-button size="small" @click="claudeDrawerVisible = true" class="topbar-btn">
          <el-icon><SetUp /></el-icon>
          <span class="hidden sm:inline ml-1">Claude 管理</span>
        </el-button>
        <el-tooltip :content="themeMode === 'auto' ? '跟随系统主题' : isDark ? '切换浅色模式' : '切换深色模式'" placement="bottom">
          <el-button size="small" @click="toggleTheme" class="topbar-btn">
            <el-icon><component :is="isDark ? Sunny : MoonNight" /></el-icon>
          </el-button>
        </el-tooltip>
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

          <!-- Mode Tabs (only show when not resuming) -->
          <div v-if="!resumeTaskId" class="mode-tabs mb-3">
            <button
              v-for="mode in ['develop', 'ask', 'extend', 'export']"
              :key="mode"
              class="mode-tab"
              :class="{ 'mode-tab--active': activeMode === mode }"
              @click="activeMode = mode"
            >
              {{ mode === 'develop' ? '开发' : mode === 'ask' ? '问答' : mode === 'extend' ? '扩展' : '导出' }}
            </button>
          </div>

          <!-- Ask Mode: Simple input -->
          <div v-if="activeMode === 'ask' && !resumeTaskId" class="mb-3">
            <el-input
              v-model="newTask.description"
              type="textarea"
              :rows="3"
              placeholder="输入你的问题，例如：如何用 Go 实现并发爬虫？"
              maxlength="500"
              show-word-limit
              class="mb-2"
            />
            <el-input
              v-model="newTask.workDir"
              placeholder="项目目录（已有项目）"
              class="mb-2"
            >
              <template #prefix><el-icon class="text-slate-400"><Folder /></el-icon></template>
            </el-input>
            <el-select
              v-model="newTask.workDir"
              placeholder="从历史项目选择"
              clearable
              filterable
              class="w-full mb-2"
            >
              <el-option
                v-for="proj in projects"
                :key="proj"
                :label="proj"
                :value="proj"
              />
            </el-select>
            <div class="output-hint">
              <el-icon class="text-slate-500 text-xs shrink-0"><InfoFilled /></el-icon>
              <span>AI 将基于当前项目上下文回答问题，结果保存到 <code class="hint-code">process/qa.md</code></span>
            </div>
          </div>

          <!-- Extend Mode: Extend existing project -->
          <div v-if="activeMode === 'extend' && !resumeTaskId" class="mb-3">
            <el-input
              v-model="newTask.description"
              type="textarea"
              :rows="3"
              placeholder="输入要追加的新需求，例如：添加 OAuth2 登录功能"
              maxlength="500"
              show-word-limit
              class="mb-2"
            />
            <el-input
              v-model="newTask.workDir"
              placeholder="已有项目目录（必填）"
              class="mb-2"
            >
              <template #prefix><el-icon class="text-slate-400"><Folder /></el-icon></template>
            </el-input>
            <el-select
              v-model="newTask.workDir"
              placeholder="从历史项目选择"
              clearable
              filterable
              class="w-full mb-2"
            >
              <el-option
                v-for="proj in projects"
                :key="proj"
                :label="proj"
                :value="proj"
              />
            </el-select>
            <div class="output-hint">
              <el-icon class="text-slate-500 text-xs shrink-0"><InfoFilled /></el-icon>
              <span>在已有项目上追加新需求，自动走 DESIGN→DO→REVIEW→DELIVER 流程</span>
            </div>
          </div>

          <!-- Export Mode -->
          <div v-if="activeMode === 'export' && !resumeTaskId" class="mb-3">
            <el-input
              v-model="newTask.description"
              type="textarea"
              :rows="2"
              placeholder="要导出的任务描述或目录名"
              maxlength="200"
              class="mb-2"
            />
            <el-select v-model="newTask.exportFormat" placeholder="导出格式" size="small" class="w-full mb-2">
              <el-option label="ZIP 压缩包" value="zip" />
              <el-option label="TAR.GZ 压缩包" value="tar" />
            </el-select>
            <div class="output-hint">
              <el-icon class="text-slate-500 text-xs shrink-0"><InfoFilled /></el-icon>
              <span>导出任务目录为压缩包，支持 <code class="hint-code">zip</code> 或 <code class="hint-code">tar</code> 格式</span>
            </div>
          </div>

          <!-- Develop Mode -->
          <template v-if="activeMode === 'develop' && !resumeTaskId">
          <el-input
            v-model="newTask.description"
            type="textarea"
            :rows="4"
            :placeholder="taskPlaceholder"
            maxlength="500"
            show-word-limit
            class="mb-3"
          />
          <!-- Output location hint -->
          <div class="output-hint">
            <el-icon class="text-slate-500 text-xs shrink-0"><InfoFilled /></el-icon>
            <span>结果将输出到 <code class="hint-code">RESULT.md</code>，代码文件保存在任务目录根路径，过程文档在 <code class="hint-code">process/</code></span>
          </div>
          <div v-if="resumeTaskId" class="flex items-center gap-2 mb-3">
            <span class="text-xs text-slate-400 shrink-0">从阶段</span>
            <el-input-number v-model="newTask.resumeFrom" :min="1" :max="6" size="small" class="flex-1" />
            <span class="text-xs text-purple-400 shrink-0">{{ phaseLabel(newTask.resumeFrom) }}</span>
          </div>
          <div v-if="activeMode === 'develop'" class="flex gap-2 mb-3 flex-wrap">
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
          </template>

          <!-- Submit Button (visible for all modes) -->
          <el-button
            v-if="!resumeTaskId"
            type="primary"
            :loading="submitting"
            :disabled="submitDisabled"
            @click="submitTask"
            class="w-full submit-btn"
          >
            <el-icon v-if="!submitting">
              <VideoPlay v-if="activeMode === 'develop'" />
              <ChatLineRound v-else-if="activeMode === 'ask'" />
              <Plus v-else-if="activeMode === 'extend'" />
              <Download v-else-if="activeMode === 'export'" />
            </el-icon>
            <span class="ml-1">开始执行</span>
          </el-button>
        </div>

        <!-- Task List -->
        <div class="sidebar-card flex-1 overflow-hidden flex flex-col">
          <div class="sidebar-card-header">
            <el-icon class="text-slate-400 text-sm"><List /></el-icon>
            <span>任务列表</span>
            <span v-if="taskTotal > 0" class="text-xs text-slate-500 ml-1">({{ taskTotal }})</span>
            <el-button size="small" :loading="loadingList" @click="loadTasks(true)" circle text class="ml-auto !text-slate-400">
              <el-icon><Refresh /></el-icon>
            </el-button>
          </div>
          <!-- Filter pills -->
          <div class="filter-bar">
            <button class="filter-pill" :class="{ 'filter-pill--active': listFilter.status === '' }" @click="setStatusFilter('')">全部</button>
            <button class="filter-pill" :class="{ 'filter-pill--active': listFilter.status === 'running' }" @click="setStatusFilter('running')">运行中</button>
            <button class="filter-pill" :class="{ 'filter-pill--active': listFilter.status === 'completed' }" @click="setStatusFilter('completed')">已完成</button>
            <button class="filter-pill" :class="{ 'filter-pill--active': listFilter.status === 'failed' }" @click="setStatusFilter('failed')">失败</button>
          </div>
          <div v-if="tasks.length === 0 && !loadingList" class="text-center text-slate-500 py-8">
            <el-icon class="text-3xl mb-2 text-slate-600"><DocumentRemove /></el-icon>
            <p class="text-sm">{{ listFilter.status ? '无匹配任务' : '暂无任务' }}</p>
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
                  <div class="flex items-center gap-1.5 mt-0.5">
                    <span class="task-type-tag" :class="`task-type-tag--${task.type || 'develop'}`">{{ typeLabel(task.type) }}</span>
                    <p class="text-xs text-slate-500">{{ formatTime(task.created_at) }}</p>
                  </div>
                  <!-- Running phase -->
                  <div v-if="task.status === 'running' && task.autodev_state" class="mt-1.5">
                    <div class="text-xs text-amber-400 flex items-center gap-1 mb-1">
                      <span class="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse inline-block"></span>
                      {{ task.autodev_state.phase_label || '执行中' }}
                    </div>
                    <div v-if="task.type === 'ask' || task.type === 'extend'" class="flex gap-0.5">
                      <div class="flex-1 h-1 rounded-full bg-amber-400 animate-pulse" />
                    </div>
                    <div v-else class="flex gap-0.5">
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
            <!-- Load more -->
            <div v-if="tasks.length < taskTotal" class="py-2 text-center">
              <el-button size="small" text :loading="loadingMore" @click="loadMore" class="!text-slate-400 !text-xs">
                加载更多（{{ tasks.length }}/{{ taskTotal }}）
              </el-button>
            </div>
            <div v-else-if="tasks.length > 0" class="py-1 text-center text-xs text-slate-600">全部 {{ taskTotal }} 条已加载</div>
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
            <!-- Row 1: status + title -->
            <div class="content-header-title-row">
              <div class="flex items-center gap-2 shrink-0">
                <el-tag :type="statusType(selectedTask.status)" effect="dark" size="small">
                  <el-icon v-if="selectedTask.status === 'running'" class="is-loading mr-1"><Loading /></el-icon>
                  {{ statusLabel(selectedTask.status) }}
                </el-tag>
                <span class="task-type-tag" :class="`task-type-tag--${selectedTask.type || 'develop'}`">{{ typeLabel(selectedTask.type) }}</span>
                <span class="text-xs text-slate-500 font-mono">#{{ selectedTask.id }}</span>
              </div>
              <h2 class="content-header-title" :title="selectedTask.description">{{ selectedTask.description }}</h2>
            </div>
            <!-- Row 2: action buttons (always on own line, never wrapped off-screen) -->
            <div class="content-header-actions">
              <el-button size="small" :loading="loadingDetail" @click="refreshDetail" class="action-btn" plain>
                <el-icon><Refresh /></el-icon>
              </el-button>
              <el-button size="small" type="primary" plain :loading="downloading" @click="downloadTask" class="action-btn">
                <el-icon><Download /></el-icon> 下载
              </el-button>
              <el-tooltip content="在此项目中追问 / 执行小任务" placement="bottom">
                <el-button size="small" plain @click="quickAsk(selectedTask)" class="action-btn"
                  :disabled="selectedTask.status === 'running'">
                  <el-icon><ChatLineRound /></el-icon> 问答
                </el-button>
              </el-tooltip>
              <el-tooltip content="在此项目上追加新需求（迭代开发）" placement="bottom">
                <el-button size="small" plain @click="quickExtend(selectedTask)" class="action-btn"
                  :disabled="selectedTask.status === 'running'">
                  <el-icon><Plus /></el-icon> 扩展
                </el-button>
              </el-tooltip>
              <el-tooltip content="生成 CLAUDE.md，为 ask/extend 提供冷启动上下文" placement="bottom">
                <el-button size="small" plain @click="initProject(selectedTask)" class="action-btn"
                  :loading="initingProject"
                  :disabled="selectedTask.status === 'running'">
                  <el-icon v-if="!initingProject"><DocumentChecked /></el-icon> 初始化
                </el-button>
              </el-tooltip>
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

          <!-- Phase Steps: develop tasks with state.json -->
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
          <!-- Simple progress for ask/extend tasks -->
          <div v-else-if="selectedTask.type === 'ask' || selectedTask.type === 'extend'" class="simple-task-progress">
            <div class="simple-task-dot"
              :class="selectedTask.status === 'running' ? 'simple-task-dot--active' : selectedTask.status === 'completed' ? 'simple-task-dot--done' : 'simple-task-dot--pending'">
              <el-icon v-if="selectedTask.status === 'completed'" class="text-xs"><Check /></el-icon>
              <el-icon v-else-if="selectedTask.status === 'running'" class="text-xs is-loading"><Loading /></el-icon>
              <el-icon v-else class="text-xs"><WarningFilled /></el-icon>
            </div>
            <span class="simple-task-label">
              {{ selectedTask.type === 'ask' ? 'ASK 问答执行' : 'EXTEND 迭代扩展' }}
            </span>
            <span v-if="selectedTask.status === 'running'" class="ml-2 text-xs text-amber-400 flex items-center gap-1">
              <span class="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse inline-block"></span>
              运行中
            </span>
            <span v-else-if="selectedTask.status === 'completed'" class="ml-2 text-xs text-green-400">完成</span>
            <span v-else class="ml-2 text-xs text-red-400">失败</span>
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
                {{ selectedTask.type === 'ask' ? '问答记录' : selectedTask.type === 'extend' ? '迭代报告' : '结果文档' }}
              </button>
              <button
                class="tab-btn"
                :class="{ 'tab-btn--active': activeTab === 'files' }"
                @click="switchToFiles()"
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
                <p class="text-slate-400">{{ resultEmptyHint }}</p>
                <p v-if="selectedTask.status === 'running'" class="text-slate-500 text-sm mt-1">执行完成后自动显示</p>
              </div>
              <div v-else-if="loadingDetail" class="empty-content">
                <el-icon class="text-3xl text-purple-400 is-loading mb-2"><Loading /></el-icon>
                <p class="text-slate-400 text-sm">加载中…</p>
              </div>
              <div v-else class="result-content">
                <!-- Toolbar -->
                <div class="result-toolbar">
                  <span class="text-xs text-slate-500">{{ resultDocTitle }}</span>
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
                <!-- Grouped File Tree -->
                <div class="file-tree">
                  <!-- Search box -->
                  <div class="file-search-wrap">
                    <el-input
                      v-model="fileSearch"
                      size="small"
                      placeholder="搜索文件…"
                      clearable
                      :prefix-icon="Search"
                      class="file-search-input"
                    />
                  </div>
                  <template v-for="group in filteredFileGroups" :key="group.category">
                    <div class="file-group-header" @click="toggleGroupCollapse(group.category)" style="cursor:pointer">
                      <span class="file-group-icon">{{ group.icon }}</span>
                      <span class="file-group-label">{{ group.label }}</span>
                      <span class="file-group-count">{{ group.files.length }}</span>
                      <el-icon class="ml-auto text-slate-500 text-xs transition-transform" :class="collapsedGroups.has(group.category) ? 'rotate-0' : 'rotate-90'"><ArrowRight /></el-icon>
                    </div>
                    <template v-if="!collapsedGroups.has(group.category)">
                    <div
                      v-for="file in group.files"
                      :key="file.path"
                      class="file-item"
                      :class="{ 'file-item--active': activeFilePath === file.path }"
                      @click="loadFile(file.path)"
                      :title="file.path"
                    >
                      <span class="file-icon">{{ fileIcon(file.name) }}</span>
                      <div class="file-info">
                        <span class="file-name">{{ file.name }}</span>
                        <span class="file-path-hint" v-if="file.path.includes('/')">{{ file.path.substring(0, file.path.lastIndexOf('/')) }}/</span>
                      </div>
                    </div>
                    </template>
                  </template>
                </div>
                <!-- File Content -->
                <div class="file-content-area">
                  <div v-if="!activeFilePath" class="empty-content">
                    <div class="text-center">
                      <p class="text-slate-500 text-sm mb-3">点击左侧文件查看内容</p>
                      <div class="text-xs text-slate-600 space-y-1 text-left inline-block">
                        <p>📄 <strong class="text-slate-500">RESULT.md</strong> — 任务最终结果</p>
                        <p>💻 <strong class="text-slate-500">项目代码</strong> — autodev 生成的代码文件</p>
                        <p>📋 <strong class="text-slate-500">process/</strong> — 各阶段过程文档</p>
                      </div>
                    </div>
                  </div>
                  <div v-else-if="loadingFile" class="empty-content">
                    <el-icon class="text-2xl text-purple-400 is-loading"><Loading /></el-icon>
                  </div>
                  <div v-else class="h-full flex flex-col">
                    <div class="result-toolbar">
                      <div class="flex items-center gap-2 min-w-0">
                        <span class="file-category-badge" :class="`badge--${activeCategoryName}`">{{ activeCategoryLabel }}</span>
                        <span class="text-xs text-slate-500 font-mono truncate">{{ activeFilePath }}</span>
                      </div>
                      <div class="flex gap-2 shrink-0">
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

    <!-- ===== Init 初始化 Dialog ===== -->
    <el-dialog v-model="initDialogVisible" title="初始化项目上下文" width="560px" :close-on-click-modal="!initingProject" destroy-on-close>
      <div class="init-dialog-body">
        <p class="text-sm text-slate-400 mb-3">
          扫描项目目录，生成 <code class="bg-slate-700 px-1 rounded text-xs">CLAUDE.md</code>，为后续 ask/extend 提供冷启动上下文（避免重复安装依赖、重复调研）。
        </p>
        <div v-if="initingProject || initLogs.length" class="init-log-box" ref="initLogEl">
          <div v-if="initLogs.length === 0" class="text-slate-500 text-xs">正在初始化…</div>
          <div v-for="(line, i) in initLogs" :key="i" class="init-log-line">{{ line }}</div>
        </div>
        <div v-if="initResult" class="mt-3 flex items-center gap-2" :class="initResult.ok ? 'text-green-400' : 'text-red-400'">
          <el-icon><component :is="initResult.ok ? Check : WarningFilled" /></el-icon>
          <span class="text-sm">{{ initResult.ok ? 'CLAUDE.md 已生成，后续 ask/extend 将自动读取上下文' : '初始化失败: ' + initResult.error }}</span>
        </div>
      </div>
      <template #footer>
        <el-button v-if="!initingProject" @click="initDialogVisible = false">关闭</el-button>
        <el-button v-else type="danger" plain @click="initDialogVisible = false">后台运行</el-button>
      </template>
    </el-dialog>

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

        <!-- clawtest Card -->
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="font-semibold text-sm">AutoDev 引擎 (clawtest)</span>
              <el-button size="small" :loading="loadingClawtestInfo" @click="loadClawtestVersion" circle plain>
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>
          </template>
          <div v-if="loadingClawtestInfo" class="text-center py-4"><el-icon class="is-loading text-purple-500 text-xl"><Loading /></el-icon></div>
          <div v-else-if="clawtestInfo" class="space-y-2 mb-3">
            <div class="flex items-center justify-between py-1.5 border-b">
              <span class="text-sm text-gray-500">状态</span>
              <el-tag :type="clawtestInfo.available ? 'success' : 'danger'" size="small" effect="dark">{{ clawtestInfo.available ? '已安装' : '未安装' }}</el-tag>
            </div>
            <div class="flex items-center justify-between py-1.5 border-b">
              <span class="text-sm text-gray-500">当前 commit</span>
              <span class="text-sm font-mono font-semibold text-purple-600">{{ clawtestInfo.commit_short || '—' }}</span>
            </div>
            <div class="flex items-center justify-between py-1.5 border-b">
              <span class="text-sm text-gray-500">提交时间</span>
              <span class="text-xs font-mono text-gray-600">{{ clawtestInfo.commit_date || '—' }}</span>
            </div>
            <div class="flex items-center justify-between py-1.5 border-b">
              <span class="text-sm text-gray-500">分支</span>
              <span class="text-xs font-mono text-gray-600">{{ clawtestInfo.branch || '—' }}</span>
            </div>
            <div class="flex items-center justify-between py-1.5">
              <span class="text-sm text-gray-500">路径</span>
              <span class="text-xs font-mono text-gray-600 truncate max-w-[240px]">{{ clawtestInfo.path || '—' }}</span>
            </div>
          </div>
          <div v-else class="text-center text-gray-400 py-4 text-sm mb-3">点击刷新按钮获取版本信息</div>
          <div class="space-y-3">
            <div class="bg-gray-50 rounded p-3 text-xs text-gray-600">
              <p>执行命令：</p>
              <code class="font-mono text-purple-700">git pull ({{ clawtestInfo?.path || '/opt/clawtest' }})</code>
            </div>
            <el-button type="primary" class="w-full" :loading="updatingClawtest" :disabled="updatingClawtest" @click="startClawtestUpdate">
              <el-icon><Upload /></el-icon>
              <span class="ml-1">{{ updatingClawtest ? '更新中…' : '更新到最新版本' }}</span>
            </el-button>
            <div v-if="clawtestUpdateLogs.length">
              <div class="flex items-center justify-between mb-1">
                <span class="text-xs text-gray-500">更新输出</span>
                <el-button size="small" text @click="clawtestUpdateLogs = []">清空</el-button>
              </div>
              <pre ref="clawtestUpdateLogEl" class="text-xs bg-gray-900 text-green-400 p-3 rounded overflow-auto max-h-[300px] whitespace-pre-wrap break-all font-mono leading-5">{{ clawtestUpdateLogs.join('\n') }}</pre>
            </div>
            <el-result v-if="clawtestUpdateResult" :icon="clawtestUpdateResult.success ? 'success' : 'error'" :title="clawtestUpdateResult.success ? '更新成功' : '更新失败'" :sub-title="clawtestUpdateResult.message">
              <template #extra>
                <el-button size="small" @click="loadClawtestVersion">刷新版本信息</el-button>
              </template>
            </el-result>
          </div>
        </el-card>

        <!-- SSH Key Card -->
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <el-icon class="text-green-500"><Key /></el-icon>
                <span class="font-semibold text-sm">GitHub SSH 密钥</span>
              </div>
              <div class="flex items-center gap-2">
                <el-button size="small" :loading="loadingSSHKey" @click="loadSSHKey" circle plain>
                  <el-icon><Refresh /></el-icon>
                </el-button>
                <el-popconfirm title="重新生成将使旧密钥失效，需重新添加到 GitHub，确认吗？" confirm-button-text="确认重新生成" cancel-button-text="取消" @confirm="regenerateSSHKey">
                  <template #reference>
                    <el-button size="small" type="danger" plain :loading="regeneratingSSHKey">
                      <el-icon><RefreshRight /></el-icon> 重新生成
                    </el-button>
                  </template>
                </el-popconfirm>
              </div>
            </div>
          </template>
          <div v-if="loadingSSHKey || regeneratingSSHKey" class="text-center py-4">
            <el-icon class="is-loading text-green-500 text-xl"><Loading /></el-icon>
            <p class="text-xs text-gray-400 mt-2">{{ regeneratingSSHKey ? '正在重新生成密钥…' : '正在获取密钥…' }}</p>
          </div>
          <div v-else-if="sshKeyInfo" class="space-y-3">
            <div class="flex items-center justify-between py-1 border-b">
              <span class="text-sm text-gray-500">类型</span>
              <el-tag type="success" size="small" effect="dark">{{ sshKeyInfo.key_type || 'ed25519' }}</el-tag>
            </div>
            <div class="py-1.5">
              <div class="flex items-center justify-between mb-2">
                <span class="text-sm text-gray-500">公钥（添加到 GitHub）</span>
                <el-button size="small" type="primary" plain @click="copySSHKey">
                  <el-icon><CopyDocument /></el-icon> 复制
                </el-button>
              </div>
              <div class="relative">
                <pre class="text-xs bg-gray-900 text-green-300 p-3 rounded overflow-x-auto whitespace-pre-wrap break-all font-mono leading-5 select-all">{{ sshKeyInfo.public_key }}</pre>
              </div>
            </div>
            <el-alert type="info" :closable="false" show-icon>
              <template #default>
                <p class="text-xs leading-5">
                  将上方公钥添加到
                  <a href="https://github.com/settings/keys" target="_blank" class="text-blue-500 underline">GitHub → Settings → SSH keys</a>
                  后，autodev 任务即可通过 SSH 访问你的私有仓库。
                  <br/>密钥保存在数据 Volume 中，重建容器后无需重新配置。
                </p>
              </template>
            </el-alert>
          </div>
          <div v-else class="text-center text-gray-400 py-4 text-sm">
            <p>点击刷新按钮获取 SSH 公钥</p>
            <p class="text-xs mt-1 text-gray-300">首次获取时会自动生成密钥对</p>
          </div>
        </el-card>
      </div>
    </el-drawer>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useTheme } from '../composables/useTheme'
import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js'
import {
  Lock, MagicStick, VideoPlay, VideoPause, Refresh, RefreshRight,
  Delete, Download, DocumentRemove, Pointer, Loading, Document,
  FolderOpened, Bottom, SetUp, Upload, Monitor, Connection,
  SwitchButton, List, Check, FullScreen, CopyDocument, InfoFilled,
  Sunny, MoonNight, Key, ChatLineRound, Plus, Search, ArrowRight, WarningFilled,
  DocumentChecked
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

// ---- theme: sync with global useTheme composable ----
const { currentTheme, themeMode, toggleTheme } = useTheme()
const isDark = computed(() => currentTheme.value === 'dark')

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
const taskTotal = ref(0)
const loadingList = ref(false)
const loadingMore = ref(false)
const submitting = ref(false)
const resumeTaskId = ref('')
const activeMode = ref('develop') // 'develop' | 'ask' | 'extend' | 'export'
const newTask = ref({ description: '', publish: false, build: false, push: false, resumeFrom: 1, workDir: '', exportFormat: 'zip' })
const runningCount = computed(() => tasks.value.filter(t => t.status === 'running').length)

// ---- list filter & pagination ----
const LIST_PAGE_SIZE = 20
const listFilter = ref({ status: '', type: '' })

function setStatusFilter(status) {
  listFilter.value.status = status
  loadTasks(true)
}

function typeLabel(type) {
  const map = { develop: '开发', ask: '问答', extend: '扩展', export: '导出', init: '初始化' }
  return map[type] || type || '开发'
}

// ---- projects list ----
const projects = ref([])

async function loadProjects() {
  try {
    const res = await fetch(`${API_BASE}/projects?password=${encodeURIComponent(savedPassword)}`)
    if (res.ok) {
      const data = await res.json()
      projects.value = data.projects || []
    }
  } catch (e) {
    console.error('Failed to load projects:', e)
  }
}

// Button disabled condition
const submitDisabled = computed(() => {
  if (!newTask.value.description.trim()) return true
  // For ask and extend modes, workDir is required
  if ((activeMode.value === 'ask' || activeMode.value === 'extend') && !newTask.value.workDir.trim()) {
    return true
  }
  return false
})

// Button text based on mode
const submitButtonText = computed(() => {
  if (resumeTaskId.value) return '恢复执行'
  switch (activeMode.value) {
    case 'ask': return '提问'
    case 'extend': return '扩展'
    case 'export': return '导出'
    default: return '开始执行'
  }
})

async function loadTasks(reset = false) {
  loadingList.value = true
  try {
    const { status, type } = listFilter.value
    const params = new URLSearchParams({
      password: savedPassword,
      limit: LIST_PAGE_SIZE,
      offset: 0,
    })
    if (status) params.set('status', status)
    if (type) params.set('type', type)
    const res = await fetch(`${API_BASE}/tasks?${params}`)
    if (res.ok) {
      const data = await res.json()
      tasks.value = data.tasks || []
      taskTotal.value = data.total || 0
      // Sync selected task status from list (lightweight state update)
      if (selectedTask.value) {
        const found = tasks.value.find(t => t.id === selectedTask.value.id)
        if (found) {
          // Update status/autodev_state without replacing the whole object (avoids re-render flicker)
          selectedTask.value.status = found.status
          selectedTask.value.exit_code = found.exit_code
          if (found.autodev_state) taskState.value = found.autodev_state
        }
      }
    }
  } finally { loadingList.value = false }
}

async function loadMore() {
  if (loadingMore.value || tasks.value.length >= taskTotal.value) return
  loadingMore.value = true
  try {
    const { status, type } = listFilter.value
    const params = new URLSearchParams({
      password: savedPassword,
      limit: LIST_PAGE_SIZE,
      offset: tasks.value.length,
    })
    if (status) params.set('status', status)
    if (type) params.set('type', type)
    const res = await fetch(`${API_BASE}/tasks?${params}`)
    if (res.ok) {
      const data = await res.json()
      const newTasks = data.tasks || []
      // Deduplicate by id before appending
      const existingIds = new Set(tasks.value.map(t => t.id))
      tasks.value.push(...newTasks.filter(t => !existingIds.has(t.id)))
      taskTotal.value = data.total || taskTotal.value
    }
  } finally { loadingMore.value = false }
}

async function submitTask() {
  if (!newTask.value.description.trim()) return
  // For ask and extend, workDir is required
  if ((activeMode.value === 'ask' || activeMode.value === 'extend') && !newTask.value.workDir.trim()) {
    ElMessage.error('请输入项目目录')
    return
  }
  submitting.value = true
  try {
    let res, data

    if (activeMode.value === 'ask') {
      // Ask mode: use dedicated ask API
      res = await fetch(`${API_BASE}/ask`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          password: savedPassword,
          description: newTask.value.description.trim(),
          work_dir: newTask.value.workDir.trim()
        })
      })
      data = await res.json()
    } else if (activeMode.value === 'extend') {
      // Extend mode: use dedicated extend API
      res = await fetch(`${API_BASE}/extend`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          password: savedPassword,
          description: newTask.value.description.trim(),
          work_dir: newTask.value.workDir.trim()
        })
      })
      data = await res.json()
    } else {
      // Build request body based on mode
      const body = {
        password: savedPassword,
      }

      if (resumeTaskId.value) {
        // Resume mode
        body.description = newTask.value.description.trim()
        body.type = 'develop'
        body.resume_from = newTask.value.resumeFrom
        body.work_dir = newTask.value.workDir
      } else {
        // Normal mode - include type based on activeMode
        body.description = newTask.value.description.trim()
        body.type = activeMode.value

        if (activeMode.value === 'develop') {
          body.publish = newTask.value.publish
          body.build = newTask.value.build
          body.push = newTask.value.push
        } else if (activeMode.value === 'export') {
          body.export_format = newTask.value.exportFormat
        }
      }

      res = await fetch(`${API_BASE}/tasks`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      })
      data = await res.json()
    }

    if (res.ok) {
      const successMsg = resumeTaskId.value ? '已恢复执行' :
        activeMode.value === 'ask' ? '问题已提交，AI 正在思考…' :
        activeMode.value === 'extend' ? '扩展任务已提交，正在执行…' :
        activeMode.value === 'export' ? '导出任务已提交…' : '任务已提交，正在执行…'
      ElMessage.success(successMsg)
      resumeTaskId.value = ''
      newTask.value = { description: '', publish: false, build: false, push: false, resumeFrom: 1, workDir: '', exportFormat: 'zip' }
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
const initingProject = ref(false)
const initDialogVisible = ref(false)
const initLogs = ref([])
const initResult = ref(null)
const initLogEl = ref(null)

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
  // Reset all content — lazy loaded per tab
  activeTab.value = 'result'
  activeFilePath.value = null
  activeFileContent.value = ''
  logContent.value = ''
  taskFiles.value = []
  taskHasSite.value = false
  resultContent.value = ''
  availableLogPhases.value = ['driver']
  activeLogPhase.value = 'driver'
  // Fetch task detail (lightweight: no file walk)
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
    // Load result content for the active tab (lazy: only if result tab is visible)
    if (activeTab.value === 'result') await loadResultForTask()
    else if (activeTab.value === 'files') await loadFiles()
  } finally { loadingDetail.value = false }
}

// loadResultForTask: loads the primary result file for the selected task type
async function loadResultForTask() {
  if (!selectedTask.value) return
  // Need file list to find the right file path
  const res = await fetch(`${API_BASE}/tasks/${selectedTask.value.id}/files?password=${encodeURIComponent(savedPassword)}`)
  if (!res.ok) return
  const data = await res.json()
  taskFiles.value = data.files || []
  taskHasSite.value = !!data.has_site

  const type = selectedTask.value.type
  if (type === 'ask') {
    const qa = taskFiles.value.find(f => f.path === 'process/qa.md')
    if (qa) await loadResultFile(qa.path)
  } else if (type === 'extend') {
    const result = taskFiles.value.find(f => f.name === 'RESULT.md')
    if (result) await loadResultFile(result.path)
    else {
      const iterFiles = taskFiles.value
        .filter(f => /process\/iter-\d+\/result\.md/.test(f.path))
        .sort((a, b) => b.path.localeCompare(a.path))
      if (iterFiles.length) await loadResultFile(iterFiles[0].path)
    }
  } else {
    const result = taskFiles.value.find(f => f.name === 'RESULT.md')
    if (result) await loadResultFile(result.path)
  }
}

async function loadFiles() {
  if (!selectedTask.value) return
  const res = await fetch(`${API_BASE}/tasks/${selectedTask.value.id}/files?password=${encodeURIComponent(savedPassword)}`)
  if (res.ok) {
    const data = await res.json()
    taskFiles.value = data.files || []
    taskHasSite.value = !!data.has_site
  }
}

// ---- computed labels for result tab ----
const resultDocTitle = computed(() => {
  const type = selectedTask.value?.type
  if (type === 'ask') return 'process/qa.md'
  if (type === 'extend') return 'RESULT.md（含所有迭代记录）'
  return 'RESULT.md'
})

const resultEmptyHint = computed(() => {
  const type = selectedTask.value?.type
  const running = selectedTask.value?.status === 'running'
  if (type === 'ask') return running ? 'AI 正在回答问题…' : '暂无问答记录（process/qa.md）'
  if (type === 'extend') return running ? 'AI 正在执行迭代开发…' : '暂无迭代报告（RESULT.md）'
  return running ? 'AI 正在生成结果文档…' : '暂无结果文档'
})

// ---- quick ask / extend from selected task ----
function quickAsk(task) {
  activeMode.value = 'ask'
  newTask.value = { ...newTask.value, description: '', workDir: task.work_dir }
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function quickExtend(task) {
  activeMode.value = 'extend'
  newTask.value = { ...newTask.value, description: '', workDir: task.work_dir }
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function initProject(task) {
  if (!task?.work_dir) return
  // Open dialog and start SSE stream
  initDialogVisible.value = true
  initLogs.value = []
  initResult.value = null
  initingProject.value = true

  const params = new URLSearchParams({ password: savedPassword, work_dir: task.work_dir })
  const es = new EventSource(`${API_BASE}/init/stream?${params}`)

  es.addEventListener('log', (e) => {
    try {
      const data = JSON.parse(e.data)
      if (data.line !== undefined) {
        initLogs.value.push(data.line)
        nextTick(() => {
          if (initLogEl.value) initLogEl.value.scrollTop = initLogEl.value.scrollHeight
        })
      }
    } catch {}
  })

  es.addEventListener('done', (e) => {
    es.close()
    initingProject.value = false
    try {
      const data = JSON.parse(e.data)
      initResult.value = data.ok ? { ok: true } : { ok: false, error: data.error || '未知错误' }
    } catch {
      initResult.value = { ok: false, error: '解析响应失败' }
    }
  })

  es.onerror = () => {
    es.close()
    initingProject.value = false
    if (!initResult.value) initResult.value = { ok: false, error: 'SSE 连接中断' }
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

function switchToFiles() {
  activeTab.value = 'files'
  if (!taskFiles.value.length) loadFiles()
}

function switchToLogs() {
  activeTab.value = 'logs'
  loadLogs()
}

// Lazy-load content when switching tabs
watch(activeTab, tab => {
  if (!selectedTask.value) return
  if (tab === 'logs') loadLogs()
  else if (tab === 'files' && taskFiles.value.length === 0) loadFiles()
  else if (tab === 'result' && !resultContent.value) loadResultForTask()
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

// ---- file groups (computed from taskFiles by category) ----
const FILE_GROUP_DEFS = [
  { category: 'result',  icon: '📄', label: '结果文档' },
  { category: 'code',    icon: '💻', label: '项目代码' },
  { category: 'process', icon: '📋', label: '过程文档' },
  { category: 'docs',    icon: '📚', label: '文档配置' },
  { category: 'log',     icon: '📝', label: '执行日志' },
  { category: 'state',   icon: '⚙️', label: '运行状态' },
]

const fileSearch = ref('')
const collapsedGroups = ref(new Set())

function toggleGroupCollapse(category) {
  const s = new Set(collapsedGroups.value)
  if (s.has(category)) s.delete(category)
  else s.add(category)
  collapsedGroups.value = s
}

const fileGroups = computed(() => {
  const map = {}
  for (const file of taskFiles.value) {
    const cat = file.category || 'code'
    if (!map[cat]) map[cat] = []
    map[cat].push(file)
  }
  return FILE_GROUP_DEFS.filter(g => map[g.category]?.length).map(g => ({
    ...g,
    files: map[g.category],
  }))
})

const filteredFileGroups = computed(() => {
  const q = fileSearch.value.trim().toLowerCase()
  if (!q) return fileGroups.value
  return fileGroups.value.map(g => ({
    ...g,
    files: g.files.filter(f => f.name.toLowerCase().includes(q) || f.path.toLowerCase().includes(q)),
  })).filter(g => g.files.length > 0)
})

const activeCategoryName = computed(() => {
  if (!activeFilePath.value) return ''
  const f = taskFiles.value.find(f => f.path === activeFilePath.value)
  return f?.category || 'code'
})

const activeCategoryLabel = computed(() => {
  const def = FILE_GROUP_DEFS.find(g => g.category === activeCategoryName.value)
  return def ? `${def.icon} ${def.label}` : ''
})

// Example task placeholders to guide output expectations
const TASK_EXAMPLES = [
  '用 C++ 写一个 hello world，结果放到 RESULT.md',
  '写一份 Redis 集群最佳实践文档，保存到 RESULT.md',
  '实现一个 Python 快速排序，代码和说明都输出到 RESULT.md',
  '分析 Go 并发模型并写教程，保存到 RESULT.md',
]
const taskPlaceholder = computed(() => {
  const idx = Math.floor(Date.now() / 10000) % TASK_EXAMPLES.length
  return `例如：${TASK_EXAMPLES[idx]}`
})

// ---- file helpers ----
function fileIcon(name) {
  const ext = name.split('.').pop()?.toLowerCase()
  const map = {
    md: '📄', txt: '📄',
    json: '📋', yaml: '📋', yml: '📋', toml: '📋',
    js: '📜', ts: '📜', jsx: '📜', tsx: '📜',
    py: '🐍', rb: '💎', php: '🐘',
    go: '🔵', rs: '🦀', java: '☕',
    c: '⚡', cpp: '⚡', h: '⚡', hpp: '⚡',
    sh: '⚙️', bash: '⚙️', zsh: '⚙️',
    html: '🌐', css: '🎨', scss: '🎨',
    log: '📝',
    sql: '🗄️', db: '🗄️',
    png: '🖼️', jpg: '🖼️', svg: '🖼️',
  }
  return map[ext] || '📄'
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

// ---- SSH Key ----
const sshKeyInfo = ref(null)
const loadingSSHKey = ref(false)
const regeneratingSSHKey = ref(false)

async function loadSSHKey() {
  loadingSSHKey.value = true
  try {
    const res = await fetch(`${API_BASE}/sshkey?password=${encodeURIComponent(savedPassword)}`)
    if (res.ok) sshKeyInfo.value = await res.json()
    else ElMessage.error('获取 SSH 密钥失败')
  } catch { ElMessage.error('网络错误') } finally { loadingSSHKey.value = false }
}

async function regenerateSSHKey() {
  regeneratingSSHKey.value = true
  try {
    const res = await fetch(`${API_BASE}/sshkey/regenerate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: savedPassword })
    })
    if (res.ok) {
      sshKeyInfo.value = await res.json()
      ElMessage.success('密钥已重新生成，请将新公钥添加到 GitHub')
    } else { ElMessage.error('重新生成失败') }
  } catch { ElMessage.error('网络错误') } finally { regeneratingSSHKey.value = false }
}

async function copySSHKey() {
  if (!sshKeyInfo.value?.public_key) return
  try {
    await navigator.clipboard.writeText(sshKeyInfo.value.public_key)
    ElMessage.success('公钥已复制到剪贴板')
  } catch { ElMessage.error('复制失败，请手动选择复制') }
}

// ---- clawtest ----
const clawtestInfo = ref(null)
const loadingClawtestInfo = ref(false)
const updatingClawtest = ref(false)
const clawtestUpdateLogs = ref([])
const clawtestUpdateResult = ref(null)
const clawtestUpdateLogEl = ref(null)

async function loadClawtestVersion() {
  loadingClawtestInfo.value = true
  try {
    const res = await fetch(`${API_BASE}/clawtest/version?password=${encodeURIComponent(savedPassword)}`)
    if (res.ok) clawtestInfo.value = await res.json()
  } catch { ElMessage.error('获取 clawtest 版本失败') }
  finally { loadingClawtestInfo.value = false }
}

function startClawtestUpdate() {
  if (updatingClawtest.value) return
  updatingClawtest.value = true
  clawtestUpdateLogs.value = []
  clawtestUpdateResult.value = null
  const url = `${API_BASE}/clawtest/update/stream?password=${encodeURIComponent(savedPassword)}`
  const es = new EventSource(url)
  es.addEventListener('log', (e) => {
    try {
      const data = JSON.parse(e.data)
      clawtestUpdateLogs.value.push(data.line)
      nextTick(() => { if (clawtestUpdateLogEl.value) clawtestUpdateLogEl.value.scrollTop = clawtestUpdateLogEl.value.scrollHeight })
    } catch {}
  })
  es.addEventListener('done', (e) => {
    es.close(); updatingClawtest.value = false
    try {
      const data = JSON.parse(e.data)
      if (data.error) { clawtestUpdateResult.value = { success: false, message: data.error } }
      else {
        const msg = data.new_commit && data.new_commit !== data.old_commit
          ? `${data.old_commit} → ${data.new_commit}` : `已是最新: ${data.new_commit || '未知'}`
        clawtestUpdateResult.value = { success: true, message: msg }
        clawtestInfo.value = null
      }
    } catch { clawtestUpdateResult.value = { success: true, message: '完成' } }
  })
  es.onerror = () => {
    es.close(); updatingClawtest.value = false
    if (!clawtestUpdateResult.value) clawtestUpdateResult.value = { success: false, message: '连接中断' }
  }
}

function onClaudeDrawerOpen() {
  if (!claudeInfo.value) loadClaudeVersion()
  if (!clawtestInfo.value) loadClawtestVersion()
  if (!sshKeyInfo.value) loadSSHKey()
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
// Strategy:
//   - Every 4s: If selected task is running, refresh only that task's state (1 API call)
//               + refresh logs if logs tab is active
//   - Every 15s: Reload the task list (to pick up status changes of non-selected tasks)
let refreshStateTimer = null
let refreshListTimer = null

async function refreshSelectedTaskState() {
  const task = selectedTask.value
  if (!task || task.status !== 'running') return
  try {
    const res = await fetch(`${API_BASE}/tasks/${task.id}/state?password=${encodeURIComponent(savedPassword)}`)
    if (res.ok) {
      const state = await res.json()
      taskState.value = state
      // Sync status from state.json (running tasks may complete)
      if (state.status === 'finished' || state.status === 'completed') {
        selectedTask.value.status = 'completed'
      } else if (state.status === 'failed') {
        selectedTask.value.status = 'failed'
      }
    }
  } catch {}
  if (activeTab.value === 'logs') await loadLogs()
  // Auto-reload result when task finishes
  if (selectedTask.value?.status !== 'running') {
    await loadResultForTask()
    loadTasks(true)
  }
}

function startAutoRefresh() {
  // Fast: refresh selected running task state every 4s
  refreshStateTimer = setInterval(refreshSelectedTaskState, 4000)
  // Slow: refresh full task list every 15s (catches status changes for non-selected tasks)
  refreshListTimer = setInterval(() => {
    if (tasks.value.some(t => t.status === 'running')) loadTasks()
  }, 15000)
}

onMounted(() => {
  const pw = getPassword()
  if (pw) { savedPassword = pw; authenticated.value = true; loadTasks(); loadProjects() }
  startAutoRefresh()
})
onUnmounted(() => {
  clearInterval(refreshStateTimer)
  clearInterval(refreshListTimer)
})
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

/* ===== Filter Bar ===== */
.filter-bar {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
  margin-bottom: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid #1e293b;
}
.filter-pill {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 12px;
  border: 1px solid #334155;
  background: transparent;
  color: #64748b;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}
.filter-pill:hover { border-color: #7c3aed; color: #a78bfa; }
.filter-pill--active { background: #4c1d95; border-color: #7c3aed; color: #c4b5fd; }

/* ===== Task Type Tag ===== */
.task-type-tag {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 4px;
  font-weight: 500;
  flex-shrink: 0;
}
.task-type-tag--develop { background: #1e3a5f; color: #60a5fa; }
.task-type-tag--ask     { background: #1a3a2a; color: #34d399; }
.task-type-tag--extend  { background: #3b1f5e; color: #c084fc; }
.task-type-tag--export  { background: #3a2b00; color: #fbbf24; }
.task-type-tag--init    { background: #1a2f3a; color: #38bdf8; }

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
  flex-direction: column;
  gap: 8px;
  padding: 10px 16px;
  background: #1e293b;
  border-bottom: 1px solid #334155;
  flex-shrink: 0;
}

/* Row 1: status tags + title */
.content-header-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.content-header-title {
  font-size: 14px;
  font-weight: 600;
  color: #f1f5f9;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}

/* Row 2: action buttons — scrollable on very narrow screens */
.content-header-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
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

/* Simple progress row for ask/extend tasks */
.simple-task-progress {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px 10px;
  border-bottom: 1px solid #1e293b;
}
.simple-task-dot {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  flex-shrink: 0;
}
.simple-task-dot--done { background: #16a34a; color: white; }
.simple-task-dot--active { background: #d97706; color: white; box-shadow: 0 0 0 4px rgba(217,119,6,0.2); }
.simple-task-dot--pending { background: #334155; color: #64748b; }
.simple-task-label { font-size: 12px; font-weight: 600; color: #94a3b8; letter-spacing: 0.04em; }

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
.file-info { display: flex; flex-direction: column; min-width: 0; }
.file-name { font-size: 12px; color: #94a3b8; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-path-hint { font-size: 10px; color: #475569; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-item--active .file-name { color: #a78bfa; font-weight: 500; }
.file-item--active .file-path-hint { color: #7c3aed; }

/* File group headers */
.file-group-header {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 8px 8px 4px;
  margin-top: 4px;
  border-top: 1px solid #1e293b;
}
.file-group-header:first-child { border-top: none; margin-top: 0; }
.file-group-icon { font-size: 11px; }
.file-group-label { font-size: 11px; font-weight: 700; color: #64748b; text-transform: uppercase; letter-spacing: 0.06em; flex: 1; }
.file-group-count { font-size: 10px; background: #334155; color: #64748b; border-radius: 8px; padding: 0 5px; }
.file-group-header:hover { background: #1e293b; border-radius: 6px; }

/* File search */
.file-search-wrap { padding: 6px 4px 8px; }
.file-search-input :deep(.el-input__wrapper) { background: #1e293b; box-shadow: none; border: 1px solid #334155; }
.file-search-input :deep(.el-input__inner) { color: #94a3b8; font-size: 12px; }
.file-search-input :deep(.el-input__prefix) { color: #475569; }

/* Category badge in file content toolbar */
.file-category-badge {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 7px;
  border-radius: 10px;
  white-space: nowrap;
  flex-shrink: 0;
}
.badge--result  { background: #1e1b4b; color: #a78bfa; }
.badge--code    { background: #052e16; color: #4ade80; }
.badge--process { background: #172554; color: #60a5fa; }
.badge--log     { background: #1c1917; color: #a8a29e; }
.badge--docs    { background: #1c1917; color: #fbbf24; }
.badge--state   { background: #1c1917; color: #94a3b8; }

/* Mode tabs in submit form */
.mode-tabs {
  display: flex;
  gap: 4px;
  background: #0f172a;
  padding: 3px;
  border-radius: 8px;
  margin-bottom: 12px;
}
.mode-tab {
  flex: 1;
  padding: 6px 12px;
  border: none;
  background: transparent;
  color: #94a3b8;
  font-size: 12px;
  font-weight: 500;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}
.mode-tab:hover {
  color: #e2e8f0;
  background: #1e293b;
}
.mode-tab--active {
  background: #7c3aed;
  color: #fff;
}
.mode-tab--active:hover {
  background: #6d28d9;
}

/* Output hint in submit form */
.output-hint {
  display: flex;
  align-items: flex-start;
  gap: 5px;
  background: #0f172a;
  border: 1px solid #1e293b;
  border-radius: 6px;
  padding: 6px 10px;
  margin-bottom: 10px;
  font-size: 11px;
  color: #64748b;
  line-height: 1.5;
}
.hint-code {
  background: #1e293b;
  color: #a78bfa;
  padding: 0 4px;
  border-radius: 3px;
  font-family: monospace;
  font-size: 11px;
}

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

/* ===== Init Dialog ===== */
.init-dialog-body { padding: 4px 0; }
.init-log-box {
  background: #0f172a;
  border: 1px solid #334155;
  border-radius: 8px;
  padding: 10px 12px;
  max-height: 300px;
  overflow-y: auto;
  font-family: monospace;
  font-size: 12px;
  line-height: 1.6;
}
.init-log-line { color: #94a3b8; white-space: pre-wrap; word-break: break-all; }

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

/* =====================================================
   Light theme overrides — applied when .theme-light
   ===================================================== */
.theme-light {
  background: #f8fafc;
  color: #1e293b;
}

.theme-light .topbar {
  background: #ffffff;
  border-bottom-color: #e2e8f0;
}
.theme-light .topbar-btn {
  border-color: #cbd5e1 !important;
  color: #475569 !important;
}
.theme-light .topbar-btn:hover {
  border-color: #7c3aed !important;
  color: #6d28d9 !important;
}

.theme-light .sidebar {
  background: #f1f5f9;
  border-right-color: #e2e8f0;
}
.theme-light .sidebar-card {
  background: #ffffff;
  border-color: #e2e8f0;
}
.theme-light .sidebar-card-header { color: #64748b; }

.theme-light .task-item:hover { background: #f8fafc; border-color: #cbd5e1; }
.theme-light .task-item--active { background: #f5f3ff; border-color: #7c3aed !important; }
.theme-light .task-item p.text-slate-200 { color: #1e293b !important; }
.theme-light .task-item p.text-slate-500 { color: #94a3b8 !important; }
.theme-light .filter-bar { border-bottom-color: #e2e8f0; }
.theme-light .filter-pill { border-color: #e2e8f0; color: #94a3b8; }
.theme-light .filter-pill:hover { border-color: #7c3aed; color: #7c3aed; }
.theme-light .filter-pill--active { background: #ede9fe; border-color: #7c3aed; color: #6d28d9; }

.theme-light .content-panel { background: #ffffff; }
.theme-light .content-header {
  background: #f8fafc;
  border-bottom-color: #e2e8f0;
}
.theme-light .content-header-title { color: #0f172a; }
.theme-light .action-btn {
  border-color: #cbd5e1 !important;
  color: #64748b !important;
}
.theme-light .action-btn:hover {
  border-color: #7c3aed !important;
  color: #6d28d9 !important;
}

.theme-light .phase-steps {
  background: #f8fafc;
  border-bottom-color: #e2e8f0;
}
.theme-light .phase-step-dot--pending { background: #e2e8f0; color: #94a3b8; }
.theme-light .phase-connector { background: #e2e8f0; }

.theme-light .tab-bar {
  background: #f8fafc;
  border-bottom-color: #e2e8f0;
}
.theme-light .tab-btn { color: #94a3b8; }
.theme-light .tab-btn:hover { color: #475569; background: rgba(0,0,0,0.03); }
.theme-light .tab-btn--active { color: #7c3aed; border-bottom-color: #7c3aed; }
.theme-light .tab-badge { background: #e2e8f0; color: #64748b; }

.theme-light .result-toolbar {
  background: #f8fafc;
  border-bottom-color: #e2e8f0;
}

/* Light markdown */
.theme-light .markdown-view { color: #1e293b; }
.theme-light .markdown-view :deep(h1) { color: #0f172a; border-bottom-color: #e2e8f0; }
.theme-light .markdown-view :deep(h2) { color: #1e293b; border-bottom-color: #e2e8f0; }
.theme-light .markdown-view :deep(h3) { color: #334155; }
.theme-light .markdown-view :deep(h4),
.theme-light .markdown-view :deep(h5),
.theme-light .markdown-view :deep(h6) { color: #475569; }
.theme-light .markdown-view :deep(p) { color: #334155; }
.theme-light .markdown-view :deep(a) { color: #6d28d9; }
.theme-light .markdown-view :deep(strong) { color: #0f172a; }
.theme-light .markdown-view :deep(em) { color: #64748b; }
.theme-light .markdown-view :deep(code) {
  background: #f1f5f9;
  border-color: #e2e8f0;
  color: #be185d;
}
.theme-light .markdown-view :deep(.hljs-block) { border-color: #e2e8f0; }
.theme-light .markdown-view :deep(.hljs-block code) { background: #f8fafc !important; }
.theme-light .markdown-view :deep(pre) { background: #f8fafc; border-color: #e2e8f0; }
.theme-light .markdown-view :deep(pre code) { color: #1e293b; }
.theme-light .markdown-view :deep(li) { color: #334155; }
.theme-light .markdown-view :deep(blockquote) { background: #f5f3ff; border-left-color: #7c3aed; color: #64748b; }
.theme-light .markdown-view :deep(hr) { border-top-color: #e2e8f0; }
.theme-light .markdown-view :deep(th) { background: #f1f5f9; color: #64748b; border-color: #e2e8f0; }
.theme-light .markdown-view :deep(td) { border-color: #e2e8f0; color: #334155; }
.theme-light .markdown-view :deep(tr:nth-child(even)) { background: #f8fafc; }

/* Light file tree */
.theme-light .file-tree {
  background: #f8fafc;
  border-right-color: #e2e8f0;
}
.theme-light .file-item:hover { background: #f1f5f9; }
.theme-light .file-item--active { background: #ede9fe !important; }
.theme-light .file-name { color: #475569; }
.theme-light .file-path-hint { color: #94a3b8; }
.theme-light .file-item--active .file-name { color: #6d28d9; }
.theme-light .file-group-header { border-top-color: #e2e8f0; }
.theme-light .file-group-label { color: #94a3b8; }
.theme-light .file-group-count { background: #e2e8f0; color: #94a3b8; }

.theme-light .file-content-area { background: #ffffff; }
.theme-light .code-view { background: #f8fafc; color: #1e293b; }

/* Light log terminal */
.theme-light .log-toolbar {
  background: #f8fafc;
  border-bottom-color: #e2e8f0;
}
.theme-light .log-terminal {
  background: #1e293b;   /* keep terminal dark even in light mode — readability */
  color: #4ade80;
}

/* Light empty states */
.theme-light .empty-content { color: #94a3b8; }

/* Light output hint */
.theme-light .output-hint {
  background: #f8fafc;
  border-color: #e2e8f0;
  color: #94a3b8;
}
.theme-light .hint-code {
  background: #e2e8f0;
  color: #6d28d9;
}

/* Light category badges */
.theme-light .badge--result  { background: #ede9fe; color: #6d28d9; }
.theme-light .badge--code    { background: #dcfce7; color: #16a34a; }
.theme-light .badge--process { background: #dbeafe; color: #2563eb; }
.theme-light .badge--log     { background: #f1f5f9; color: #64748b; }
.theme-light .badge--docs    { background: #fef9c3; color: #92400e; }
.theme-light .badge--state   { background: #f1f5f9; color: #64748b; }
</style>
