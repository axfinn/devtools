<template>
  <div class="planner-shell" :class="`mode-${activeKind}`">
    <div class="planner-backdrop planner-backdrop-a"></div>
    <div class="planner-backdrop planner-backdrop-b"></div>

    <section v-if="!profileId" class="entry-layout">
      <div class="entry-hero">
        <p class="eyebrow">Planner Archive</p>
        <h1>工作和生活完全分开，记录、安排、提醒都在一个档案里。</h1>
        <p class="entry-copy">
          支持时间线、收件箱、事件、放一放、语音输入、日历导出，以及只给建议不落库的 AI 总结 / 规划。
        </p>
        <div class="entry-tips">
          <span>工作日 09:00-18:00 默认工作模式</span>
          <span>下班后默认生活模式</span>
          <span>手机上也能顺手记录和切换</span>
        </div>
      </div>

      <div class="entry-grid">
        <el-card shadow="never" class="entry-card">
          <template #header>
            <div class="entry-card-header">
              <span>创建档案</span>
              <el-tag size="small" type="success">新建</el-tag>
            </div>
          </template>
          <el-input v-model="createForm.name" placeholder="档案名称，例如：Finn 的事项档案" class="entry-input" />
          <el-input v-model="createForm.notifyEmail" placeholder="默认提醒邮箱（可选）" class="entry-input" />
          <el-input
            v-model="createForm.password"
            type="password"
            show-password
            placeholder="设置密码，至少 4 位"
            class="entry-input"
            @keyup.enter="createProfile"
          />
          <el-button type="primary" class="entry-btn" :loading="creating" @click="createProfile">
            创建并进入
          </el-button>
        </el-card>

        <el-card shadow="never" class="entry-card">
          <template #header>
            <div class="entry-card-header">
              <span>登录已有档案</span>
              <el-tag size="small">继续</el-tag>
            </div>
          </template>
          <el-input
            v-model="loginForm.password"
            type="password"
            show-password
            placeholder="输入档案密码"
            class="entry-input"
            @keyup.enter="loginProfile"
          />
          <el-button type="primary" plain class="entry-btn" :loading="loggingIn" @click="loginProfile">
            登录档案
          </el-button>
          <el-button text class="entry-link" @click="adminDialogVisible = true">
            超级管理员入口
          </el-button>
        </el-card>
      </div>
    </section>

    <section v-else class="planner-page">
      <header class="planner-topbar">
        <div class="topbar-copy">
          <p class="eyebrow">{{ activeKind === 'work' ? 'Work Zone' : 'Life Zone' }}</p>
          <h2>{{ profile.name || '事项档案' }}</h2>
          <p class="subtle">{{ modeHint }}</p>
        </div>
        <div class="topbar-actions">
          <el-tag effect="dark" :type="activeKind === 'work' ? 'primary' : 'success'">
            当前 {{ activeKind === 'work' ? '工作模式' : '生活模式' }}
          </el-tag>
          <el-button circle @click="settingsVisible = true">
            <el-icon><Setting /></el-icon>
          </el-button>
          <el-button circle @click="refreshAll">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </header>

      <section v-if="showInstallEntry" class="install-strip">
        <div>
          <strong>支持加入手机桌面</strong>
          <p>加入桌面后可像 App 一样快速进入事项页，更适合手机随手记录。</p>
        </div>
        <div class="install-actions">
          <el-button type="primary" plain @click="triggerInstallPrompt">
            {{ installPromptAvailable ? '添加到桌面' : '查看添加方式' }}
          </el-button>
        </div>
      </section>

      <section class="mode-switcher">
        <button class="mode-tab" :class="{ active: activeKind === 'work' }" @click="switchKind('work')">
          <span class="mode-tab-title">工作模式</span>
          <span class="mode-tab-sub">推进、交付、避免分心</span>
        </button>
        <button class="mode-tab" :class="{ active: activeKind === 'life' }" @click="switchKind('life')">
          <span class="mode-tab-title">生活模式</span>
          <span class="mode-tab-sub">照顾自己，也照顾节奏</span>
        </button>
      </section>

      <section class="hero-grid">
        <article class="hero-card hero-main">
          <div>
            <p class="hero-label">{{ activeKind === 'work' ? '今日推进感' : '今日生活感' }}</p>
            <h3>{{ currentTip.title }}</h3>
            <p>{{ currentTip.body }}</p>
          </div>
          <div class="hero-note">
            <strong>{{ board.focus.message }}</strong>
            <span>{{ board.recovery.message }}</span>
          </div>
        </article>

        <article class="hero-card hero-metrics">
          <div class="metric-card">
            <span>未完成</span>
            <strong>{{ openCount }}</strong>
          </div>
          <div class="metric-card">
            <span>收件箱</span>
            <strong>{{ board.inbox_items.length }}</strong>
          </div>
          <div class="metric-card">
            <span>事件</span>
            <strong>{{ board.counts.event_open || 0 }}</strong>
          </div>
          <div class="metric-card">
            <span>顺延</span>
            <strong>{{ board.counts.rolled_over || 0 }}</strong>
          </div>
        </article>
      </section>

      <section class="summary-strip">
        <div class="summary-chip">
          <span>聚焦上限</span>
          <strong>{{ board.focus.today_primary_count }}/{{ board.focus.today_primary_limit }}</strong>
        </div>
        <div class="summary-chip">
          <span>今日完成</span>
          <strong>{{ board.recovery.done_today }}</strong>
        </div>
        <div class="summary-chip">
          <span>今日取消</span>
          <strong>{{ board.recovery.cancelled_today }}</strong>
        </div>
        <div class="summary-chip">
          <span>最近完成</span>
          <strong>{{ board.recent_items.length }}</strong>
        </div>
      </section>

      <section class="focus-grid">
        <article class="hero-card focus-card">
          <div class="panel-heading">
            <div>
              <p class="hero-label">今天最重要的事</p>
              <h3>{{ primaryFocusTask ? primaryFocusTask.title : '今天先给自己留一个主焦点' }}</h3>
              <p>{{ primaryFocusTask?.detail || '把今天真正想推进的一件事放到最前面。' }}</p>
            </div>
            <el-button v-if="primaryFocusTask" type="primary" @click="openTask(primaryFocusTask)">打开</el-button>
          </div>
          <div v-if="primaryFocusTask" class="focus-meta">
            <span>{{ priorityLabel(primaryFocusTask.priority) }}</span>
            <span>{{ statusLabel(primaryFocusTask.status) }}</span>
            <span>{{ primaryFocusTask.time_hint || primaryFocusTask.display_label }}</span>
          </div>
          <div v-if="secondaryFocusTasks.length > 0" class="focus-secondary">
            <button v-for="task in secondaryFocusTasks" :key="task.id" class="focus-secondary-item" @click="openTask(task)">
              <strong>{{ task.title }}</strong>
              <span>{{ task.time_hint || task.display_label }}</span>
            </button>
          </div>
        </article>

        <article class="hero-card focus-card event-card-hero">
          <div class="panel-heading">
            <div>
              <p class="hero-label">下一场事件</p>
              <h3>{{ nextEventTask ? nextEventTask.title : '最近没有需要特别盯住的事件' }}</h3>
              <p>{{ nextEventTask?.detail || '会议、预约、出发这类事情，应该带着时间感被看见。' }}</p>
            </div>
            <el-button v-if="nextEventTask" plain @click="openTask(nextEventTask)">查看</el-button>
          </div>
          <div v-if="nextEventTask" class="focus-meta">
            <span>{{ eventPhaseLabel(nextEventTask.event_phase) }}</span>
            <span>{{ nextEventTask.time_hint || '已安排' }}</span>
            <span>{{ nextEventTask.remind_at ? formatDateTime(nextEventTask.remind_at) : nextEventTask.planned_for }}</span>
          </div>
          <div v-if="nextEventTask" class="task-actions compact-actions">
            <el-button size="small" plain @click="setTaskStatus(nextEventTask, 'in_progress')">开始准备</el-button>
            <el-button size="small" type="primary" plain @click="setTaskStatus(nextEventTask, 'done')">已完成</el-button>
            <el-button size="small" plain @click="cancelTask(nextEventTask)">取消</el-button>
          </div>
        </article>
      </section>

      <section class="quick-panel">
        <div class="panel-heading">
          <div>
            <h3>{{ activeKind === 'work' ? '快速记录工作事项' : '快速记录生活事项' }}</h3>
            <p>{{ activeKind === 'work' ? '先落地，再排序。' : '把脑中的小事卸下来，节奏会轻很多。' }}</p>
          </div>
          <div class="panel-heading-actions">
            <el-button plain @click="openVoice('quick')">
              <el-icon><Microphone /></el-icon>
              语音输入
            </el-button>
            <el-button plain @click="openAdviceDialog">
              <el-icon><MagicStick /></el-icon>
              AI 助手
            </el-button>
            <el-button plain @click="reviewDialogVisible = true">
              <el-icon><Calendar /></el-icon>
              周/月/年回顾
            </el-button>
            <el-button type="primary" plain @click="aiDialogVisible = true">
              <el-icon><EditPen /></el-icon>
              AI 整理入库
            </el-button>
            <el-button plain @click="openMeetingDialog">
              <el-icon><Document /></el-icon>
              会议纪要
            </el-button>
          </div>
        </div>

        <div class="quick-presets">
          <button class="preset-btn" @click="applyQuickPreset('today')">今天任务</button>
          <button class="preset-btn" @click="applyQuickPreset('inbox')">收件箱</button>
          <button class="preset-btn" @click="applyQuickPreset('event')">事件安排</button>
        </div>

        <div class="quick-grid quick-grid-core">
          <el-input
            v-model="quickForm.title"
            placeholder="比如：确认联调结果 / 买水果 / 预约复诊"
            maxlength="80"
            show-word-limit
          />
          <el-input v-model="quickForm.detail" type="textarea" :rows="3" placeholder="补充上下文、边界条件、备注" />
          <div class="quick-row">
            <div class="quick-actions quick-actions-primary">
              <el-button :loading="savingQuick" type="primary" @click="createQuickTask">
                保存
              </el-button>
              <el-button @click="fillTodayDefaults">恢复默认</el-button>
              <el-button v-if="isMobile" plain @click="quickAdvancedVisible = !quickAdvancedVisible">
                {{ quickAdvancedVisible ? '收起更多设置' : '更多设置' }}
              </el-button>
            </div>
          </div>
        </div>

        <div v-if="!isMobile || quickAdvancedVisible" class="quick-grid quick-grid-advanced">
          <div class="quick-row">
            <el-select v-model="quickForm.entryType" placeholder="条目类型">
              <el-option label="任务" value="task" />
              <el-option label="事件" value="event" />
            </el-select>
            <el-select v-model="quickForm.bucket" placeholder="放到哪里" :disabled="quickForm.entryType === 'event'">
              <el-option label="计划中" value="planned" />
              <el-option label="收件箱" value="inbox" />
              <el-option label="放一放" value="someday" />
            </el-select>
            <el-select v-model="quickForm.priority" placeholder="优先级">
              <el-option label="高优先级" value="high" />
              <el-option label="中优先级" value="medium" />
              <el-option label="低优先级" value="low" />
            </el-select>
          </div>
          <div class="quick-row">
            <el-date-picker v-model="quickForm.plannedFor" type="date" value-format="YYYY-MM-DD" placeholder="计划日期" />
            <el-date-picker
              v-model="quickForm.remindAt"
              type="datetime"
              value-format="YYYY-MM-DDTHH:mm"
              placeholder="提醒时间"
            />
          </div>
          <div class="quick-row">
            <el-select v-model="quickForm.repeatType" :disabled="!quickForm.remindAt" placeholder="重复提醒">
              <el-option label="不重复" value="none" />
              <el-option label="每天" value="daily" />
              <el-option label="工作日" value="weekdays" />
              <el-option label="每周" value="weekly" />
              <el-option label="每月" value="monthly" />
            </el-select>
            <el-input-number
              v-if="['daily', 'weekly', 'monthly'].includes(quickForm.repeatType)"
              v-model="quickForm.repeatInterval"
              :min="1"
              :max="30"
              placeholder="间隔"
            />
            <el-date-picker
              v-model="quickForm.repeatUntil"
              type="datetime"
              value-format="YYYY-MM-DDTHH:mm"
              placeholder="结束时间（可选）"
              :disabled="quickForm.repeatType === 'none'"
            />
          </div>
          <el-input v-model="quickForm.notifyEmail" placeholder="提醒邮箱，默认用档案邮箱" />
        </div>
      </section>

      <section class="board-switcher">
        <button
          v-for="view in viewOptions"
          :key="view.key"
          class="view-tab"
          :class="{ active: activeView === view.key }"
          @click="switchView(view.key)"
        >
          <span>{{ view.label }}</span>
          <strong>{{ view.count }}</strong>
        </button>
      </section>

      <section class="board-panel">
        <div class="timeline-header">
          <div>
            <h3>{{ currentViewMeta.title }}</h3>
            <p>{{ currentViewMeta.body }}</p>
          </div>
          <el-button plain @click="refreshBoard">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>

        <div v-if="timelineLoading" class="timeline-empty">正在加载看板...</div>

        <template v-else>
          <div v-if="activeView === 'timeline'">
            <div v-if="board.groups.length === 0" class="timeline-empty">
              <strong>还没有时间线事项</strong>
              <span>先把今天要推进的一件事写下来。</span>
            </div>
            <div v-else class="timeline-groups">
              <div v-for="group in board.groups" :key="group.date" class="timeline-group">
                <div class="timeline-group-head">
                  <div>
                    <h4>{{ group.label }}</h4>
                    <p>{{ group.date }}</p>
                  </div>
                  <el-tag size="small">{{ group.items.length }} 项</el-tag>
                </div>
                <div class="task-list">
                  <article
                    v-for="task in group.items"
                    :key="task.id"
                    class="task-card"
                    :class="[`status-${task.status}`, `priority-${task.priority}`]"
                  >
                    <div class="task-card-top">
                      <div class="task-copy" @click="openTask(task)">
                        <h5>{{ task.title }}</h5>
                        <p>{{ task.detail || '点击补充详情、提醒、备注和评论' }}</p>
                      </div>
                      <div class="task-tags">
                        <el-tag size="small">{{ entryTypeLabel(task.entry_type) }}</el-tag>
                        <el-tag size="small">{{ bucketLabel(task.bucket) }}</el-tag>
                        <el-tag size="small" :type="priorityTagType(task.priority)">{{ priorityLabel(task.priority) }}</el-tag>
                        <el-tag size="small" :type="statusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
                      </div>
                    </div>
                    <div class="task-meta">
                      <span>计划 {{ task.planned_for }}</span>
                      <span v-if="task.remind_at">提醒 {{ formatDateTime(task.remind_at) }}</span>
                      <span v-if="repeatSummary(task)">重复 {{ repeatSummary(task) }}</span>
                      <span v-if="task.is_rolled_over" class="rolled-tag">已顺延到今天</span>
                      <span v-if="task.last_postpone_reason">顺延原因：{{ task.last_postpone_reason }}</span>
                    </div>
                    <div v-if="task.comment_count || task.last_comment_preview" class="task-context" @click="openCommentDrawer(task)">
                      <span class="task-context-count">进展 {{ task.comment_count || 0 }}</span>
                      <span class="task-context-text">{{ task.last_comment_preview || '已存在评论' }}</span>
                      <span v-if="task.last_comment_at">{{ formatDateTime(task.last_comment_at) }}</span>
                    </div>
                    <div class="task-actions">
                      <el-button size="small" type="primary" plain @click="setTaskStatus(task, task.status === 'done' ? 'open' : 'done')">
                        {{ task.status === 'done' ? '重新打开' : '完成' }}
                      </el-button>
                      <el-button size="small" plain @click="setTaskStatus(task, 'in_progress')">进行中</el-button>
                      <el-button size="small" plain @click="postponeTask(task)">顺延</el-button>
                      <el-button size="small" plain @click="openCommentDrawer(task)">
                        <el-icon><ChatDotRound /></el-icon>
                        评论
                      </el-button>
                      <el-button size="small" plain @click="importCalendar(task)">
                        <el-icon><Calendar /></el-icon>
                        日历
                      </el-button>
                      <el-button size="small" plain @click="openTask(task)">编辑</el-button>
                    </div>
                  </article>
                </div>
              </div>
            </div>
          </div>

          <div v-else-if="activeView === 'events'">
            <div v-if="board.event_groups.length === 0" class="timeline-empty">
              <strong>还没有事件安排</strong>
              <span>会议、预约、复诊、出发时间这些更适合放这里。</span>
            </div>
            <div v-else class="timeline-groups">
              <div v-for="group in board.event_groups" :key="group.date" class="timeline-group">
                <div class="timeline-group-head">
                  <div>
                    <h4>{{ group.label }}</h4>
                    <p>{{ group.date }}</p>
                  </div>
                  <el-tag size="small" type="success">{{ group.items.length }} 个事件</el-tag>
                </div>
                <div class="task-list">
                  <article
                    v-for="task in group.items"
                    :key="task.id"
                    class="task-card"
                    :class="[`status-${task.status}`, `priority-${task.priority}`]"
                  >
                    <div class="task-card-top">
                      <div class="task-copy" @click="openTask(task)">
                        <h5>{{ task.title }}</h5>
                        <p>{{ task.detail || '补充地点、联系人、准备事项' }}</p>
                      </div>
                      <div class="task-tags">
                        <el-tag size="small" type="success">事件</el-tag>
                        <el-tag size="small" :type="priorityTagType(task.priority)">{{ priorityLabel(task.priority) }}</el-tag>
                        <el-tag size="small" :type="statusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
                      </div>
                    </div>
                    <div class="task-meta">
                      <span>日期 {{ task.planned_for }}</span>
                      <span v-if="task.remind_at">提醒 {{ formatDateTime(task.remind_at) }}</span>
                      <span v-if="repeatSummary(task)">重复 {{ repeatSummary(task) }}</span>
                      <span v-if="task.time_hint">{{ task.time_hint }}</span>
                      <span v-if="task.event_phase">{{ eventPhaseLabel(task.event_phase) }}</span>
                      <span v-if="task.needs_closure" class="rolled-tag">待收尾</span>
                      <span v-if="task.notes">备注：{{ task.notes }}</span>
                    </div>
                    <div v-if="task.comment_count || task.last_comment_preview" class="task-context" @click="openCommentDrawer(task)">
                      <span class="task-context-count">进展 {{ task.comment_count || 0 }}</span>
                      <span class="task-context-text">{{ task.last_comment_preview || '已存在评论' }}</span>
                      <span v-if="task.last_comment_at">{{ formatDateTime(task.last_comment_at) }}</span>
                    </div>
                    <div class="task-actions">
                      <el-button size="small" type="primary" plain @click="setTaskStatus(task, task.status === 'done' ? 'open' : 'done')">
                        {{ task.status === 'done' ? '重新打开' : '完成' }}
                      </el-button>
                      <el-button size="small" plain @click="setTaskStatus(task, 'in_progress')">开始准备</el-button>
                      <el-button size="small" plain @click="openCommentDrawer(task)">评论</el-button>
                      <el-button size="small" plain @click="importCalendar(task)">日历</el-button>
                      <el-button size="small" plain @click="openTask(task)">编辑</el-button>
                    </div>
                  </article>
                </div>
              </div>
            </div>
          </div>

          <div v-else-if="activeView === 'inbox'">
            <div v-if="board.inbox_items.length === 0" class="timeline-empty">
              <strong>收件箱是空的</strong>
              <span>这是好事，说明脑子里悬着的想法不多。</span>
            </div>
            <div v-else class="task-list">
              <article
                v-for="task in board.inbox_items"
                :key="task.id"
                class="task-card"
                :class="[`status-${task.status}`, `priority-${task.priority}`]"
              >
                <div class="task-card-top">
                  <div class="task-copy" @click="openTask(task)">
                    <h5>{{ task.title }}</h5>
                    <p>{{ task.detail || '先记在收件箱，回头再分流。' }}</p>
                  </div>
                  <div class="task-tags">
                    <el-tag size="small" type="info">收件箱</el-tag>
                    <el-tag size="small" :type="priorityTagType(task.priority)">{{ priorityLabel(task.priority) }}</el-tag>
                  </div>
                </div>
                <div class="task-meta">
                  <span>创建于 {{ formatDateTime(task.created_at) }}</span>
                </div>
                <div v-if="task.comment_count || task.last_comment_preview" class="task-context" @click="openCommentDrawer(task)">
                  <span class="task-context-count">进展 {{ task.comment_count || 0 }}</span>
                  <span class="task-context-text">{{ task.last_comment_preview || '已存在评论' }}</span>
                  <span v-if="task.last_comment_at">{{ formatDateTime(task.last_comment_at) }}</span>
                </div>
                <div class="task-actions">
                  <el-button size="small" type="primary" plain @click="scheduleInboxTask(task)">安排到今天</el-button>
                  <el-button size="small" plain @click="moveTaskBucket(task, 'someday')">放一放</el-button>
                  <el-button size="small" plain @click="openCommentDrawer(task)">评论</el-button>
                  <el-button size="small" plain @click="openTask(task)">编辑</el-button>
                </div>
              </article>
            </div>
          </div>

          <div v-else-if="activeView === 'someday'">
            <div v-if="board.someday_items.length === 0" class="timeline-empty">
              <strong>暂时没有放一放事项</strong>
              <span>需要以后再处理的事，可以先丢这里。</span>
            </div>
            <div v-else class="task-list">
              <article
                v-for="task in board.someday_items"
                :key="task.id"
                class="task-card"
                :class="[`status-${task.status}`, `priority-${task.priority}`]"
              >
                <div class="task-card-top">
                  <div class="task-copy" @click="openTask(task)">
                    <h5>{{ task.title }}</h5>
                    <p>{{ task.detail || '未来再说，但不丢。' }}</p>
                  </div>
                  <div class="task-tags">
                    <el-tag size="small" type="warning">放一放</el-tag>
                    <el-tag size="small" :type="priorityTagType(task.priority)">{{ priorityLabel(task.priority) }}</el-tag>
                  </div>
                </div>
                <div class="task-meta">
                  <span>计划日期 {{ task.planned_for }}</span>
                </div>
                <div v-if="task.comment_count || task.last_comment_preview" class="task-context" @click="openCommentDrawer(task)">
                  <span class="task-context-count">进展 {{ task.comment_count || 0 }}</span>
                  <span class="task-context-text">{{ task.last_comment_preview || '已存在评论' }}</span>
                  <span v-if="task.last_comment_at">{{ formatDateTime(task.last_comment_at) }}</span>
                </div>
                <div class="task-actions">
                  <el-button size="small" type="primary" plain @click="moveTaskBucket(task, 'planned')">排入计划</el-button>
                  <el-button size="small" plain @click="openCommentDrawer(task)">评论</el-button>
                  <el-button size="small" plain @click="openTask(task)">编辑</el-button>
                </div>
              </article>
            </div>
          </div>

          <div v-else-if="activeView === 'minutes'">
            <div class="timeline-header-inner">
              <div class="timeline-header-actions">
                <el-button type="primary" @click="openMeetingDialog">
                  <el-icon><Plus /></el-icon>
                  新建会议纪要
                </el-button>
              </div>
            </div>
            <div v-if="meetings.length === 0" class="timeline-empty">
              <strong>还没有会议纪要</strong>
              <span>记录会议内容和待办，支持录音和 AI 总结。</span>
            </div>
            <div v-else class="task-list">
              <article
                v-for="m in meetings"
                :key="m.id"
                class="meeting-card"
                :class="m.status === 'finalized' ? 'meeting-done' : 'meeting-draft'"
              >
                <!-- Header: status + tags -->
                <div class="meeting-card-head">
                  <div class="meeting-card-badges">
                    <el-tag v-if="m.status === 'finalized'" size="small" type="success" effect="dark" round>已定稿</el-tag>
                    <el-tag v-else size="small" type="info" round>草稿</el-tag>
                    <el-tag
                      v-for="tag in parseJsonArray(m.tags)"
                      :key="tag"
                      size="small"
                      type="warning"
                      round
                    >{{ tag }}</el-tag>
                  </div>
                </div>

                <!-- Title -->
                <h4 class="meeting-card-title" @click="openMeetingEdit(m)">{{ m.title }}</h4>

                <!-- Meta: date, time, duration, participants -->
                <div class="meeting-card-meta">
                  <span class="meeting-card-meta-item">
                    <el-icon><Calendar /></el-icon>
                    {{ m.meeting_date }}<template v-if="m.meeting_time"> {{ m.meeting_time }}</template>
                  </span>
                  <span v-if="m.duration_minutes" class="meeting-card-meta-item">
                    <el-icon><Clock /></el-icon>
                    {{ formatDuration(m.duration_minutes) }}
                  </span>
                  <span v-if="m.participants && m.participants !== '[]'" class="meeting-card-meta-item">
                    <el-icon><UserFilled /></el-icon>
                    {{ parseJsonArray(m.participants).join('、') }}
                  </span>
                </div>

                <!-- Content preview -->
                <div v-if="m.content" class="meeting-card-body" @click="openMeetingEdit(m)">
                  <p>{{ m.content.length > 120 ? m.content.slice(0, 120) + '...' : m.content }}</p>
                </div>

                <!-- Summary + action items row -->
                <div v-if="m.summary || (m.action_items && m.action_items !== '[]')" class="meeting-card-extras">
                  <div v-if="m.summary" class="meeting-card-summary" @click="openMeetingEdit(m)">
                    <el-tag size="small" type="primary">AI 摘要</el-tag>
                    <span>{{ m.summary.length > 80 ? m.summary.slice(0, 80) + '...' : m.summary }}</span>
                  </div>
                  <div v-if="m.action_items && m.action_items !== '[]'" class="meeting-card-todos" @click="openMeetingEdit(m)">
                    <span class="meeting-card-todo-badge">{{ parseJsonArray(m.action_items).length }} 项待办</span>
                  </div>
                </div>

                <!-- Recording player inline -->
                <div v-if="m.recording_url" class="meeting-card-recording" @click.stop>
                  <audio :src="m.recording_url" controls class="meeting-card-audio" preload="metadata"></audio>
                </div>

                <!-- Actions -->
                <div class="meeting-card-actions">
                  <el-button size="small" type="primary" plain @click="openMeetingEdit(m)">编辑</el-button>
                  <el-button v-if="m.content && !m.summary" size="small" plain :loading="summarizingId === m.id" @click="summarizeMeeting(m)">AI 总结</el-button>
                  <el-button size="small" plain @click="deleteMeetingConfirm(m)">删除</el-button>
                </div>
              </article>
            </div>
          </div>

          <div v-else>
            <div v-if="board.recent_items.length === 0" class="timeline-empty">
              <strong>最近还没有完成或取消记录</strong>
              <span>完成一件也值得被看见。</span>
            </div>
            <div v-else class="task-list">
              <article
                v-for="task in board.recent_items"
                :key="task.id"
                class="task-card"
                :class="[`status-${task.status}`, `priority-${task.priority}`]"
              >
                <div class="task-card-top">
                  <div class="task-copy" @click="openTask(task)">
                    <h5>{{ task.title }}</h5>
                    <p>{{ task.detail || '这条记录已经收尾。' }}</p>
                  </div>
                  <div class="task-tags">
                    <el-tag size="small" :type="statusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
                    <el-tag size="small">{{ entryTypeLabel(task.entry_type) }}</el-tag>
                  </div>
                </div>
                <div class="task-meta">
                  <span>完成/关闭于 {{ formatDateTime(task.completed_at || task.updated_at) }}</span>
                  <span v-if="task.cancel_reason">原因：{{ task.cancel_reason }}</span>
                </div>
                <div v-if="task.comment_count || task.last_comment_preview" class="task-context" @click="openCommentDrawer(task)">
                  <span class="task-context-count">进展 {{ task.comment_count || 0 }}</span>
                  <span class="task-context-text">{{ task.last_comment_preview || '已存在评论' }}</span>
                  <span v-if="task.last_comment_at">{{ formatDateTime(task.last_comment_at) }}</span>
                </div>
                <div class="task-actions">
                  <el-button size="small" type="primary" plain @click="setTaskStatus(task, 'open')">重新打开</el-button>
                  <el-button size="small" plain @click="openCommentDrawer(task)">评论</el-button>
                  <el-button size="small" plain @click="openTask(task)">查看</el-button>
                </div>
              </article>
            </div>
          </div>
        </template>


      </section>
    </section>

    <el-drawer v-model="settingsVisible" title="档案设置" :size="drawerSize">
      <div class="drawer-stack">
        <el-input v-model="profileForm.name" placeholder="档案名称" />
        <el-input v-model="profileForm.notifyEmail" placeholder="默认提醒邮箱" />
        <el-input-number v-model="profileForm.extendDays" :min="1" :max="3650" placeholder="需要延期时再填写" />
        <div class="soft-note">
          {{ hasCreatorPrivileges ? '当前设备持有创建者密钥，可修改档案设置和删除档案。' : '当前仅为访问模式，可管理事项，但档案设置和删除需要创建者密钥。' }}
        </div>
        <div class="drawer-actions">
          <el-button type="primary" :loading="savingProfile" :disabled="!hasCreatorPrivileges" @click="saveProfile">保存档案</el-button>
          <el-button type="danger" plain :disabled="!hasCreatorPrivileges" @click="deleteProfile">删除档案</el-button>
          <el-button @click="logout">退出</el-button>
        </div>
        <div class="soft-note">
          到期时间：{{ profile.expires_at ? formatDateTime(profile.expires_at) : '长期' }}
        </div>
      </div>
    </el-drawer>

    <el-drawer v-model="taskDrawerVisible" :title="taskForm.id ? '编辑事项' : '事项详情'" :size="detailDrawerSize">
      <div class="drawer-stack">
        <el-input v-model="taskForm.title" placeholder="事项标题" />
        <div class="quick-row">
          <el-select v-model="taskForm.kind">
            <el-option label="工作事项" value="work" />
            <el-option label="生活事项" value="life" />
          </el-select>
          <el-select v-model="taskForm.entryType">
            <el-option label="任务" value="task" />
            <el-option label="事件" value="event" />
          </el-select>
          <el-select v-model="taskForm.bucket" :disabled="taskForm.entryType === 'event'">
            <el-option label="计划中" value="planned" />
            <el-option label="收件箱" value="inbox" />
            <el-option label="放一放" value="someday" />
          </el-select>
        </div>
        <div class="quick-row">
          <el-select v-model="taskForm.status">
            <el-option label="未开始" value="open" />
            <el-option label="进行中" value="in_progress" />
            <el-option label="已完成" value="done" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
          <el-select v-model="taskForm.priority">
            <el-option label="高优先级" value="high" />
            <el-option label="中优先级" value="medium" />
            <el-option label="低优先级" value="low" />
          </el-select>
        </div>
        <div class="quick-row">
          <el-date-picker v-model="taskForm.plannedFor" type="date" value-format="YYYY-MM-DD" placeholder="计划日期" />
          <el-date-picker
            v-model="taskForm.remindAt"
            type="datetime"
            value-format="YYYY-MM-DDTHH:mm"
            placeholder="提醒时间"
          />
        </div>
        <div class="quick-row">
          <el-select v-model="taskForm.repeatType" :disabled="!taskForm.remindAt" placeholder="重复提醒">
            <el-option label="不重复" value="none" />
            <el-option label="每天" value="daily" />
            <el-option label="工作日" value="weekdays" />
            <el-option label="每周" value="weekly" />
            <el-option label="每月" value="monthly" />
          </el-select>
          <el-input-number
            v-if="['daily', 'weekly', 'monthly'].includes(taskForm.repeatType)"
            v-model="taskForm.repeatInterval"
            :min="1"
            :max="30"
            placeholder="间隔"
          />
          <el-date-picker
            v-model="taskForm.repeatUntil"
            type="datetime"
            value-format="YYYY-MM-DDTHH:mm"
            placeholder="结束时间（可选）"
            :disabled="taskForm.repeatType === 'none'"
          />
        </div>
        <el-input v-model="taskForm.notifyEmail" placeholder="提醒邮箱" />
        <el-input v-model="taskForm.detail" type="textarea" :rows="4" placeholder="详情" />
        <el-input v-model="taskForm.notes" type="textarea" :rows="4" placeholder="备注 / 评论摘要 / 拆解点" />
        <el-input
          v-model="taskForm.postponeReason"
          type="textarea"
          :rows="2"
          placeholder="如果顺延过，可以在这里写最近一次顺延原因"
        />
        <el-input
          v-model="taskForm.cancelReason"
          type="textarea"
          :rows="2"
          placeholder="如果取消，请写原因"
        />
        <div v-if="taskForm.id" class="detail-comments">
          <div class="detail-comments-head">
            <strong>进展 / 备注流</strong>
            <el-button plain size="small" @click="openCommentDrawer({ id: taskForm.id, title: taskForm.title })">
              查看全部 / 添加
            </el-button>
          </div>
        <div v-if="taskCommentsPreview.length > 0" class="comment-timeline">
            <div v-for="comment in taskCommentsPreview" :key="comment.id" class="comment-item">
              <div class="comment-item-top">
                <strong>{{ comment.author }}</strong>
                <span>{{ formatDateTime(comment.created_at) }}</span>
              </div>
              <p>{{ comment.content }}</p>
            </div>
        </div>
        <div v-else class="timeline-empty">还没有进展记录，可以补一句上下文。</div>
        </div>
        <div v-if="taskForm.id" class="detail-comments">
          <div class="detail-comments-head">
            <strong>生命周期流水</strong>
            <el-button plain size="small" @click="openActivityDrawer({ id: taskForm.id, title: taskForm.title })">
              查看完整流水
            </el-button>
          </div>
          <div v-if="taskActivitiesPreview.length > 0" class="activity-timeline">
            <div v-for="item in taskActivitiesPreview" :key="item.id" class="activity-item">
              <div class="comment-item-top">
                <strong>{{ item.title }}</strong>
                <span>{{ formatDateTime(item.created_at) }}</span>
              </div>
              <p>{{ item.content || '系统记录' }}</p>
            </div>
          </div>
          <div v-else class="timeline-empty">还没有生命周期流水。</div>
        </div>
        <div class="drawer-actions">
          <el-button type="primary" :loading="savingTask" @click="saveTask">保存</el-button>
          <el-button plain @click="taskDrawerVisible = false">关闭</el-button>
          <el-button v-if="taskForm.id" type="danger" plain @click="removeTask">删除</el-button>
        </div>
      </div>
    </el-drawer>

    <el-drawer v-model="commentDrawerVisible" title="事项评论" :size="drawerSize">
      <div class="drawer-stack">
        <div class="comment-timeline">
          <div v-for="comment in comments" :key="comment.id" class="comment-item">
            <div class="comment-item-top">
              <strong>{{ comment.author }}</strong>
              <span>{{ formatDateTime(comment.created_at) }}</span>
            </div>
            <p>{{ comment.content }}</p>
          </div>
          <div v-if="comments.length === 0" class="timeline-empty">
            还没有评论，补一句上下文也行。
          </div>
        </div>
        <el-input v-model="commentForm.content" type="textarea" :rows="4" placeholder="记录进展、补充想法、留一点上下文" />
        <div class="drawer-actions">
          <el-button type="primary" :loading="savingComment" @click="submitComment">添加评论</el-button>
          <el-button @click="commentDrawerVisible = false">关闭</el-button>
        </div>
      </div>
    </el-drawer>

    <el-drawer v-model="activityDrawerVisible" title="事项生命周期" :size="drawerSize">
      <div class="drawer-stack">
        <div class="activity-timeline">
          <div v-for="item in taskActivities" :key="item.id" class="activity-item">
            <div class="comment-item-top">
              <strong>{{ item.title }}</strong>
              <span>{{ formatDateTime(item.created_at) }}</span>
            </div>
            <p>{{ item.content || '系统记录' }}</p>
          </div>
          <div v-if="taskActivities.length === 0" class="timeline-empty">
            还没有生命周期流水。
          </div>
        </div>
      </div>
    </el-drawer>

    <el-dialog v-model="aiDialogVisible" title="AI 整理事项" :width="dialogWidth" :fullscreen="isMobile">
      <div class="drawer-stack">
        <div class="panel-heading-actions">
          <el-button plain @click="openVoice('ai')">
            <el-icon><Microphone /></el-icon>
            语音转文字
          </el-button>
          <span class="soft-note">{{ activeKind === 'work' ? '例如：今天要确认接口、补测试、晚上发周报。' : '例如：下班后买水果，周末整理衣柜，提醒妈妈复诊。' }}</span>
        </div>
        <el-input v-model="aiText" type="textarea" :rows="7" placeholder="把脑子里的事项直接写下来，AI 负责拆成结构化事项。" />
        <div class="drawer-actions">
          <el-button type="primary" :loading="parsingAI" @click="parseAI">开始整理</el-button>
          <el-button :disabled="aiSuggestions.length === 0" :loading="applyingAI" @click="applyAISuggestions">全部写入</el-button>
        </div>
        <div v-if="aiSuggestions.length > 0" class="ai-suggestions">
          <article v-for="(item, index) in aiSuggestions" :key="`${item.title}-${index}`" class="ai-card">
            <div>
              <strong>{{ item.title }}</strong>
              <p>{{ item.detail || '无补充描述' }}</p>
            </div>
            <div class="task-tags">
              <el-tag size="small">{{ item.kind === 'work' ? '工作' : '生活' }}</el-tag>
              <el-tag size="small">{{ entryTypeLabel(item.entry_type) }}</el-tag>
              <el-tag size="small">{{ bucketLabel(item.bucket) }}</el-tag>
              <el-tag size="small" :type="priorityTagType(item.priority)">{{ priorityLabel(item.priority) }}</el-tag>
            </div>
          </article>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="adviceDialogVisible" title="AI 总结 / 规划" :width="dialogWidth" :fullscreen="isMobile">
      <div class="drawer-stack">
        <div class="quick-row">
          <el-select v-model="adviceMode">
            <el-option label="AI 总结" value="summary" />
            <el-option label="AI 规划" value="plan" />
          </el-select>
          <el-input v-model="adviceText" placeholder="可选：补充你现在的困扰或想重点看的方向" />
        </div>
        <div class="drawer-actions">
          <el-button type="primary" :loading="adviceLoading" @click="requestAdvice">生成建议</el-button>
        </div>
        <div v-if="adviceResult.summary" class="advice-card">
          <div class="advice-header">
            <strong>{{ adviceMode === 'summary' ? 'AI 总结' : 'AI 规划' }}</strong>
            <el-tag size="small">{{ adviceProvider }}</el-tag>
          </div>
          <p class="advice-summary">{{ adviceResult.summary }}</p>
          <div class="advice-section">
            <h4>观察</h4>
            <ul>
              <li v-for="(item, index) in adviceResult.insights" :key="`insight-${index}`">{{ item }}</li>
            </ul>
          </div>
          <div class="advice-section">
            <h4>建议</h4>
            <ul>
              <li v-for="(item, index) in adviceResult.suggestions" :key="`suggest-${index}`">{{ item }}</li>
            </ul>
          </div>
          <div class="soft-note">这个模块只给建议，不会自动修改事项。</div>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="reviewDialogVisible" title="阶段回顾" :width="dialogWidth" :fullscreen="isMobile" @open="loadReview">
      <div class="drawer-stack">
        <div class="quick-row">
          <el-select v-model="reviewPeriod" @change="loadReview">
            <el-option label="周回顾" value="week" />
            <el-option label="月回顾" value="month" />
            <el-option label="年回顾" value="year" />
          </el-select>
          <el-button plain :loading="reviewLoading" @click="loadReview">刷新回顾</el-button>
        </div>
        <div v-if="review.summary" class="advice-card">
          <div class="advice-header">
            <strong>{{ review.label }}</strong>
            <el-tag size="small">{{ activeKind === 'work' ? '工作' : '生活' }}</el-tag>
          </div>
          <p class="advice-summary">{{ review.summary }}</p>
          <div class="summary-strip review-strip">
            <div class="summary-chip">
              <span>新增</span>
              <strong>{{ review.stats.created || 0 }}</strong>
            </div>
            <div class="summary-chip">
              <span>完成</span>
              <strong>{{ review.stats.done || 0 }}</strong>
            </div>
            <div class="summary-chip">
              <span>未收尾</span>
              <strong>{{ review.stats.open || 0 }}</strong>
            </div>
            <div class="summary-chip">
              <span>事件</span>
              <strong>{{ review.stats.events || 0 }}</strong>
            </div>
          </div>
          <div class="advice-section">
            <h4>做得好的地方</h4>
            <ul>
              <li v-for="(item, index) in review.wins" :key="`win-${index}`">{{ item }}</li>
            </ul>
          </div>
          <div class="advice-section" v-if="review.drifts.length > 0">
            <h4>容易漂掉的地方</h4>
            <ul>
              <li v-for="(item, index) in review.drifts" :key="`drift-${index}`">{{ item }}</li>
            </ul>
          </div>
          <div class="advice-section">
            <h4>下一阶段建议</h4>
            <ul>
              <li v-for="(item, index) in review.suggestions" :key="`review-suggest-${index}`">{{ item }}</li>
            </ul>
          </div>
          <div class="advice-section" v-if="review.highlights.length > 0">
            <h4>值得记住的记录</h4>
            <div class="focus-secondary">
              <button v-for="item in review.highlights" :key="item.id" class="focus-secondary-item" @click="openTask(item)">
                <strong>{{ item.title }}</strong>
                <span>{{ item.completed_at ? formatDateTime(item.completed_at) : item.display_date }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="voiceDialogVisible" title="语音录入" :width="voiceDialogWidth" :fullscreen="isMobile" @closed="resetVoiceDraft">
      <div class="drawer-stack">
        <div class="voice-panel-card">
          <div class="voice-panel-head">
            <div>
              <strong>{{ voiceTarget === 'quick' ? '写入快速记录' : '追加到 AI 整理内容' }}</strong>
              <p class="soft-note">支持录音备份；浏览器支持时会同步转写文字。</p>
            </div>
            <el-tag :type="voiceState === 'recording' ? 'danger' : voiceState === 'ready' ? 'success' : 'info'">
              {{ voiceState === 'recording' ? `录音中 ${formatVoiceDuration(voiceDuration)}` : voiceState === 'ready' ? '可写入' : '待开始' }}
            </el-tag>
          </div>
          <div class="voice-panel-actions">
            <el-button v-if="voiceState !== 'recording'" type="primary" @click="startVoiceCapture">
              开始录音
            </el-button>
            <el-button v-else type="danger" plain @click="stopVoiceCapture({ keepDraft: true })">
              结束录音
            </el-button>
            <el-button plain @click="resetVoiceDraft">重置</el-button>
            <el-button type="success" :disabled="![voiceTranscript, voiceInterimTranscript].join('').trim()" @click="applyVoiceDraft">
              写入内容
            </el-button>
          </div>
          <div class="voice-capability">
            <span>{{ browserSpeechSupported ? '浏览器转写已启用' : '当前浏览器不支持原生转写，仅保留录音' }}</span>
            <span>{{ mediaRecorderSupported ? '录音备份可用' : '当前浏览器不支持录音备份' }}</span>
          </div>
          <el-input
            v-model="voiceTranscript"
            type="textarea"
            :rows="6"
            placeholder="转写结果会显示在这里，也可以手动修改后再写入。"
          />
          <div v-if="voiceInterimTranscript" class="soft-note">识别中：{{ voiceInterimTranscript }}</div>
          <div v-if="voiceError" class="voice-error">{{ voiceError }}</div>
          <audio v-if="voiceAudioURL" class="voice-audio" :src="voiceAudioURL" controls />
        </div>
      </div>
      <template #footer>
        <div class="drawer-actions">
          <el-button @click="closeVoiceDialog">关闭</el-button>
          <el-button type="primary" :disabled="![voiceTranscript, voiceInterimTranscript].join('').trim()" @click="applyVoiceDraft">
            写入
          </el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog v-model="adminDialogVisible" title="超级管理员" :width="dialogWidth" :fullscreen="isMobile">
      <div class="drawer-stack">
        <div class="quick-row">
          <el-input v-model="adminPassword" type="password" show-password placeholder="输入超管密码" />
          <el-input v-model="adminKeyword" placeholder="按档案名、邮箱或 ID 搜索" />
        </div>
        <div class="drawer-actions">
          <el-button type="primary" :loading="loadingAdmin" @click="loadAdminProfiles">查询档案</el-button>
        </div>
        <div class="admin-list">
          <article v-for="item in adminItems" :key="item.id" class="admin-card">
            <div>
              <strong>{{ item.name }}</strong>
              <p>{{ item.id }} · {{ item.notify_email || '未设置邮箱' }}</p>
            </div>
            <div class="task-tags">
              <el-tag size="small">总 {{ item.task_count }}</el-tag>
              <el-tag size="small" type="warning">开 {{ item.open_count }}</el-tag>
              <el-button size="small" plain @click="openAdminProfile(item)">查看</el-button>
              <el-button size="small" type="danger" plain @click="deleteAdminProfile(item)">删除</el-button>
            </div>
          </article>
          <div v-if="adminItems.length === 0" class="timeline-empty">暂无数据</div>
        </div>
      </div>
    </el-dialog>

    <el-drawer v-model="adminDetailVisible" title="超管查看档案" :size="detailDrawerSize">
      <div class="drawer-stack">
        <div class="soft-note">{{ adminDetail.profile.id || '' }}</div>
        <el-input v-model="adminDetail.profile.name" placeholder="档案名称" />
        <el-input v-model="adminDetail.profile.notify_email" placeholder="默认提醒邮箱" />
        <el-input-number v-model="adminDetail.extendDays" :min="1" :max="3650" placeholder="延期天数" />
        <div class="drawer-actions">
          <el-button type="primary" :loading="savingAdminProfile" @click="saveAdminProfile">保存</el-button>
          <el-button plain @click="adminDetailVisible = false">关闭</el-button>
        </div>
        <div class="comment-timeline">
          <div v-for="task in adminDetail.tasks" :key="task.id" class="comment-item">
            <div class="comment-item-top">
              <strong>{{ task.title }}</strong>
              <span>{{ task.kind === 'work' ? '工作' : '生活' }}</span>
            </div>
            <p>{{ statusLabel(task.status) }} · {{ bucketLabel(task.bucket) }} · {{ task.planned_for }}</p>
          </div>
          <div v-if="adminDetail.tasks.length === 0" class="timeline-empty">当前档案还没有事项。</div>
        </div>
      </div>
    </el-drawer>

    <el-drawer v-model="meetingDialogVisible" title="会议纪要" :size="meetingDrawerSize" @closed="resetMeetingForm">
      <div class="drawer-stack">
        <div class="quick-row">
          <el-input v-model="meetingForm.title" placeholder="会议标题（必填）" maxlength="80" show-word-limit style="flex:1" />
          <el-select v-model="meetingForm.status" style="width:120px">
            <el-option label="草稿" value="draft" />
            <el-option label="已定稿" value="finalized" />
          </el-select>
        </div>
        <div class="quick-row">
          <el-date-picker v-model="meetingForm.meetingDate" type="date" value-format="YYYY-MM-DD" placeholder="会议日期" style="flex:1" />
          <el-time-picker v-model="meetingForm.meetingTime" value-format="HH:mm" placeholder="时间" style="flex:1" />
          <el-input-number v-model="meetingForm.durationMinutes" :min="0" :max="480" placeholder="时长(分)" style="flex:1" />
        </div>
        <el-input v-model="meetingForm.participantsText" placeholder="参与人，用逗号分隔" />
        <el-input v-model="meetingForm.tagsText" placeholder="标签，用逗号分隔（如：产品评审、需求）" />
        <div v-if="meetingVoiceState === 'recording' && meetingInterimText" class="meeting-interim">
          <el-tag size="small" type="danger" effect="dark" class="recording-badge">录音中</el-tag>
          <span class="interim-text">{{ meetingInterimText }}</span>
        </div>
        <el-input
          v-model="meetingForm.content"
          type="textarea"
          :rows="14"
          class="meeting-content-input"
          placeholder="会议内容记录…&#10;支持录音实时转写"
        />
        <div v-if="meetingAudioProcessing || meetingAudioError || (!browserSpeechSupported && mediaRecorderSupported)" class="meeting-summary-section">
          <div class="panel-heading">
            <h4>语音转写</h4>
          </div>
          <p v-if="meetingAudioProcessing" class="meeting-summary-text">录音已上传，正在调用内部转写服务处理，请稍等后再保存。</p>
          <p v-else-if="meetingAudioError" class="meeting-summary-text">{{ meetingAudioError }}</p>
          <p v-else class="meeting-summary-text">当前浏览器不支持内置语音识别，停止录音后会自动调用服务端开源 ASR 转文字。</p>
        </div>
        <div v-if="meetingLocalAudioUrl || meetingForm.recordingUrl" class="voice-audio-wrapper">
          <audio :src="meetingLocalAudioUrl || meetingForm.recordingUrl" controls class="voice-audio" />
          <el-button size="small" plain @click="clearMeetingRecording">移除录音</el-button>
        </div>
        <div v-if="meetingForm.summary || meetingCanSummarize || meetingForm.content" class="meeting-summary-section">
          <div class="panel-heading">
            <h4>AI 摘要</h4>
            <el-button v-if="meetingCanSummarize" size="small" plain :loading="summarizingId === meetingForm.id" @click="summarizeMeeting(meetingForm)">
              <el-icon><MagicStick /></el-icon>{{ meetingForm.summary ? '重新总结' : '生成总结' }}
            </el-button>
          </div>
          <p v-if="meetingForm.summary" class="meeting-summary-text">{{ meetingForm.summary }}</p>
          <p v-else-if="meetingForm.id && meetingForm.content" class="meeting-summary-text">纪要内容已就绪，可以直接生成总结和待办。</p>
          <p v-else-if="meetingForm.content" class="meeting-summary-text">先保存纪要，再生成总结和待办。</p>
        </div>
        <div v-if="meetingForm.actionItems && meetingForm.actionItems !== '[]'" class="meeting-summary-section">
          <div class="panel-heading"><h4>待办事项</h4></div>
          <div v-for="(item, idx) in parsedActionItems" :key="idx" class="meeting-action-item">
            <el-tag size="small" type="primary" class="meeting-action-tag">{{ item.assignee || '未指定' }}</el-tag>
            <span>{{ item.task }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <div class="drawer-actions">
          <el-button v-if="meetingForm.id" plain @click="deleteMeeting">删除此纪要</el-button>
          <div>
            <el-button @click="meetingDialogVisible = false">关闭</el-button>
            <el-button plain @click="startMeetingVoice">
              <el-icon><Microphone /></el-icon>{{ meetingVoiceState === 'recording' ? '停止录音' : '录音' }}
            </el-button>
            <el-button type="primary" :loading="savingMeeting || meetingAudioProcessing" @click="saveMeeting">保存</el-button>
          </div>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Calendar, ChatDotRound, Clock, Document, EditPen, MagicStick, Microphone, Plus, Refresh, Setting, UserFilled } from '@element-plus/icons-vue'

const API_BASE = '/api/planner'

function createEmptyBoard() {
  return {
    kind: 'life',
    profile_name: '',
    groups: [],
    event_groups: [],
    inbox_items: [],
    someday_items: [],
    recent_items: [],
    counts: {
      open: 0,
      in_progress: 0,
      done: 0,
      cancelled: 0,
      rolled_over: 0,
      inbox_open: 0,
      event_open: 0,
      someday_open: 0
    },
    focus: {
      today_primary_limit: 3,
      today_primary_count: 0,
      needs_trim: false,
      message: '还没有今天的聚焦建议。',
      primary: null,
      secondary: [],
      next_event: null
    },
    recovery: {
      done_today: 0,
      cancelled_today: 0,
      inbox_open: 0,
      message: '还没有恢复信息。'
    },
    mode_default: 'life',
    mode_hint: '默认进入生活模式'
  }
}

const CREATOR_KEY_STORAGE_KEY = 'planner_creator_keys'

const profileId = ref('')
const password = ref('')
const creatorKey = ref('')

const profile = reactive({
  id: '',
  name: '',
  notify_email: '',
  expires_at: '',
  mode_default: 'life'
})

const board = ref(createEmptyBoard())
const activeKind = ref('life')
const activeView = ref(localStorage.getItem('planner_active_view') || 'timeline')
const modeHint = ref('当前是下班或休息时段，默认进入生活模式')
const isMobile = ref(false)
const quickAdvancedVisible = ref(false)
const installPromptAvailable = ref(false)
const isStandaloneMode = ref(false)

const createForm = reactive({
  name: '',
  password: '',
  notifyEmail: ''
})

const loginForm = reactive({
  password: ''
})

const profileForm = reactive({
  name: '',
  notifyEmail: '',
  extendDays: null
})

const quickForm = reactive({
  title: '',
  detail: '',
  entryType: 'task',
  bucket: 'planned',
  plannedFor: '',
  remindAt: '',
  repeatType: 'none',
  repeatInterval: 1,
  repeatUntil: '',
  priority: 'medium',
  notifyEmail: ''
})

const meetingDialogVisible = ref(false)
const meetings = ref([])
const loadingMeetings = ref(false)
const savingMeeting = ref(false)
const summarizingId = ref('')
const meetingVoiceState = ref('idle')
const meetingLocalAudioUrl = ref('')
const meetingAudioProcessing = ref(false)
const meetingAudioError = ref('')
const meetingForm = reactive({
  id: '',
  title: '',
  content: '',
  summary: '',
  actionItems: '[]',
  participantsText: '',
  tagsText: '',
  meetingDate: '',
  meetingTime: '',
  durationMinutes: 0,
  recordingUrl: '',
  status: 'draft'
})
const meetingCanSummarize = computed(() => Boolean(meetingForm.id && meetingForm.content.trim()))

let meetingAudioUploadPromise = null

function revokeMeetingLocalAudio() {
  if (meetingLocalAudioUrl.value) {
    URL.revokeObjectURL(meetingLocalAudioUrl.value)
    meetingLocalAudioUrl.value = ''
  }
}

function clearMeetingRecording() {
  meetingForm.recordingUrl = ''
  revokeMeetingLocalAudio()
}

function applyMeetingToForm(meeting, options = {}) {
  const { preserveLocalAudio = false } = options
  if (!preserveLocalAudio) {
    revokeMeetingLocalAudio()
  }
  meetingForm.id = meeting?.id || ''
  meetingForm.title = meeting?.title || ''
  meetingForm.content = meeting?.content || ''
  meetingForm.summary = meeting?.summary || ''
  meetingForm.actionItems = meeting?.action_items || '[]'
  meetingForm.meetingDate = meeting?.meeting_date || ''
  meetingForm.meetingTime = meeting?.meeting_time || ''
  meetingForm.durationMinutes = meeting?.duration_minutes || 0
  meetingForm.recordingUrl = meeting?.recording_url || ''
  meetingForm.status = meeting?.status || 'draft'
  try {
    const participants = JSON.parse(meeting?.participants || '[]')
    meetingForm.participantsText = Array.isArray(participants) ? participants.join(', ') : ''
  } catch {
    meetingForm.participantsText = ''
  }
  try {
    const tags = JSON.parse(meeting?.tags || '[]')
    meetingForm.tagsText = Array.isArray(tags) ? tags.join(', ') : ''
  } catch {
    meetingForm.tagsText = ''
  }
  meetingAudioError.value = ''
  meetingAudioProcessing.value = false
  meetingAudioUploadPromise = null
}

function buildMeetingSnapshot(participants, tags) {
  return {
    id: meetingForm.id,
    title: meetingForm.title,
    content: meetingForm.content,
    summary: meetingForm.summary,
    action_items: meetingForm.actionItems,
    participants: JSON.stringify(participants),
    tags: JSON.stringify(tags),
    meeting_date: meetingForm.meetingDate || new Date().toISOString().slice(0, 10),
    meeting_time: meetingForm.meetingTime,
    duration_minutes: meetingForm.durationMinutes,
    recording_url: meetingForm.recordingUrl,
    status: meetingForm.status || 'draft'
  }
}

function mergeMeetingData(baseMeeting, incomingMeeting) {
  return {
    ...baseMeeting,
    ...(incomingMeeting || {}),
    id: incomingMeeting?.id || baseMeeting.id,
    title: incomingMeeting?.title ?? baseMeeting.title,
    content: incomingMeeting?.content ?? baseMeeting.content,
    summary: incomingMeeting?.summary ?? baseMeeting.summary,
    action_items: incomingMeeting?.action_items ?? baseMeeting.action_items,
    participants: incomingMeeting?.participants ?? baseMeeting.participants,
    tags: incomingMeeting?.tags ?? baseMeeting.tags,
    meeting_date: incomingMeeting?.meeting_date ?? baseMeeting.meeting_date,
    meeting_time: incomingMeeting?.meeting_time ?? baseMeeting.meeting_time,
    duration_minutes: incomingMeeting?.duration_minutes ?? baseMeeting.duration_minutes,
    recording_url: incomingMeeting?.recording_url ?? baseMeeting.recording_url,
    status: incomingMeeting?.status ?? baseMeeting.status
  }
}

function upsertMeeting(meeting) {
  if (!meeting?.id) return
  const index = meetings.value.findIndex(item => item.id === meeting.id)
  if (index >= 0) {
    meetings.value[index] = { ...meetings.value[index], ...meeting }
    return
  }
  meetings.value = [meeting, ...meetings.value]
}

function stopMeetingRecording() {
  if (meetingRecognition) {
    try { meetingRecognition.stop() } catch {}
    meetingRecognition = null
  }
  if (meetingMediaRecorder && meetingMediaRecorder.state === 'recording') {
    meetingMediaRecorder.stop()
  }
  if (meetingMediaStream) {
    meetingMediaStream.getTracks().forEach(t => t.stop())
    meetingMediaStream = null
  }
  meetingVoiceState.value = 'idle'
}

function resetMeetingForm() {
  stopMeetingRecording()
  revokeMeetingLocalAudio()
  meetingForm.id = ''
  meetingForm.title = ''
  meetingForm.content = ''
  meetingForm.summary = ''
  meetingForm.actionItems = '[]'
  meetingForm.participantsText = ''
  meetingForm.tagsText = ''
  meetingForm.meetingDate = ''
  meetingForm.meetingTime = ''
  meetingForm.durationMinutes = 0
  meetingForm.recordingUrl = ''
  meetingForm.status = 'draft'
  meetingVoiceState.value = 'idle'
  meetingAudioProcessing.value = false
  meetingAudioError.value = ''
  meetingAudioUploadPromise = null
}

const parsedActionItems = computed(() => {
  try {
    const items = JSON.parse(meetingForm.actionItems || '[]')
    return Array.isArray(items) ? items : []
  } catch { return [] }
})

function parseJsonArray(str) {
  try {
    const arr = JSON.parse(str || '[]')
    return Array.isArray(arr) ? arr : []
  } catch { return [] }
}

function formatDuration(minutes) {
  if (!minutes || minutes <= 0) return ''
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  if (h > 0 && m > 0) return `${h}小时${m}分`
  if (h > 0) return `${h}小时`
  return `${m}分钟`
}

const taskForm = reactive({
  id: '',
  kind: 'work',
  entryType: 'task',
  bucket: 'planned',
  title: '',
  detail: '',
  notes: '',
  plannedFor: '',
  remindAt: '',
  repeatType: 'none',
  repeatInterval: 1,
  repeatUntil: '',
  priority: 'medium',
  status: 'open',
  notifyEmail: '',
  cancelReason: '',
  postponeReason: ''
})

const commentForm = reactive({
  content: ''
})

const adviceResult = reactive({
  summary: '',
  insights: [],
  suggestions: []
})

const review = reactive({
  period: 'week',
  label: '',
  summary: '',
  stats: {},
  wins: [],
  drifts: [],
  suggestions: [],
  highlights: []
})

const creating = ref(false)
const loggingIn = ref(false)
const timelineLoading = ref(false)
const savingQuick = ref(false)
const savingTask = ref(false)
const savingProfile = ref(false)
const savingComment = ref(false)
const loadingAdmin = ref(false)
const savingAdminProfile = ref(false)
const applyingAI = ref(false)
const settingsVisible = ref(false)
const taskDrawerVisible = ref(false)
const commentDrawerVisible = ref(false)
const activityDrawerVisible = ref(false)
const aiDialogVisible = ref(false)
const adviceDialogVisible = ref(false)
const adminDialogVisible = ref(false)
const adminDetailVisible = ref(false)
const parsingAI = ref(false)
const adviceLoading = ref(false)
const reviewLoading = ref(false)
const reviewDialogVisible = ref(false)

const comments = ref([])
const taskCommentsPreview = ref([])
const taskActivities = ref([])
const taskActivitiesPreview = ref([])
const aiText = ref('')
const aiSuggestions = ref([])
const adminItems = ref([])
const adminPassword = ref('')
const adminKeyword = ref('')
const currentCommentTask = ref(null)
const adviceMode = ref('plan')
const adviceText = ref('')
const adviceProvider = ref('fallback')
const reviewPeriod = ref('week')
const voiceDialogVisible = ref(false)
const voiceTarget = ref('quick')
const voiceState = ref('idle')
const voiceTranscript = ref('')
const voiceInterimTranscript = ref('')
const voiceError = ref('')
const voiceDuration = ref(0)
const voiceAudioURL = ref('')
const browserSpeechSupported = ref(false)
const mediaRecorderSupported = ref(false)
const adminDetail = reactive({
  profile: {
    id: '',
    name: '',
    notify_email: ''
  },
  tasks: [],
  extendDays: null
})

const workTips = [
  { title: '今天先推最关键的一步。', body: '别同时追五件事，先把最影响结果的那一件往前推。' },
  { title: '完成比完美更稀缺。', body: '把事项拆小一点，先结束一个闭环，情绪会稳定很多。' },
  { title: '把注意力留给高价值动作。', body: '能写成事项的，就不要继续在脑子里循环。' }
]

const lifeTips = [
  { title: '生活事项也值得被认真对待。', body: '把照顾自己写进时间线，不是拖延，是在给自己留位置。' },
  { title: '别把小事一直挂在心上。', body: '记下来，安排好，今晚就能轻一点。' },
  { title: '生活感来自可完成的小动作。', body: '先做一件能让你舒服一点的小事，节奏会回来。' }
]

const currentTip = computed(() => {
  const source = activeKind.value === 'work' ? workTips : lifeTips
  const index = new Date().getDate() % source.length
  return source[index]
})

const openCount = computed(() => (board.value.counts.open || 0) + (board.value.counts.in_progress || 0))
const hasCreatorPrivileges = computed(() => Boolean(creatorKey.value))
const showInstallEntry = computed(() => !isStandaloneMode.value)
const primaryFocusTask = computed(() => board.value.focus?.primary || null)
const secondaryFocusTasks = computed(() => board.value.focus?.secondary || [])
const nextEventTask = computed(() => board.value.focus?.next_event || null)

const viewOptions = computed(() => [
  { key: 'timeline', label: '时间线', count: board.value.groups.reduce((sum, group) => sum + group.items.length, 0) },
  { key: 'events', label: '事件', count: board.value.event_groups.reduce((sum, group) => sum + group.items.length, 0) },
  { key: 'inbox', label: '收件箱', count: board.value.inbox_items.length },
  { key: 'someday', label: '放一放', count: board.value.someday_items.length },
  { key: 'minutes', label: '会议纪要', count: meetings.value.length },
  { key: 'recent', label: '最近', count: board.value.recent_items.length }
])

const currentViewMeta = computed(() => {
  return {
    timeline: {
      title: activeKind.value === 'work' ? '主时间线' : '生活时间线',
      body: activeKind.value === 'work' ? '把真正要推进的任务压到少量关键动作上。' : '生活事项也值得被温柔安排。'
    },
    events: {
      title: '事件安排',
      body: '会议、预约、复诊、出发等明确时间发生的事，单独看更不容易漏。'
    },
    inbox: {
      title: '收件箱',
      body: '这里只负责先接住，不要求立刻执行。'
    },
    someday: {
      title: '放一放',
      body: '以后再做的事先放这里，避免占满今天。'
    },
    recent: {
      title: '最近完成',
      body: '完成和取消都应该留下痕迹，帮助你复盘节奏。'
    },
    minutes: {
      title: '会议纪要',
      body: '记录会议内容和待办，支持录音和 AI 总结。'
    }
  }[activeView.value]
})

const drawerSize = computed(() => (isMobile.value ? '100%' : '420px'))
const detailDrawerSize = computed(() => (isMobile.value ? '100%' : '480px'))
const meetingDrawerSize = computed(() => (isMobile.value ? '100%' : 'min(1080px, 92vw)'))
const dialogWidth = computed(() => (isMobile.value ? '100%' : '720px'))
const voiceDialogWidth = computed(() => (isMobile.value ? '100%' : '560px'))

let deferredInstallPrompt = null
let recognition = null
let mediaRecorder = null
let mediaStream = null
let voiceChunks = []
let voiceTimer = null
let voiceCaptureToken = 0

function defaultModeByTime() {
  const now = new Date()
  const day = now.getDay()
  const hour = now.getHours()
  if (day >= 1 && day <= 5 && hour >= 9 && hour < 18) {
    return 'work'
  }
  return 'life'
}

function plannerModeCopy() {
  return activeKind.value === 'work' ? '当前聚焦工作事项。' : '当前聚焦生活事项。'
}

function loadCreatorKeyMap() {
  try {
    return JSON.parse(localStorage.getItem(CREATOR_KEY_STORAGE_KEY) || '{}')
  } catch {
    return {}
  }
}

function getCreatorKeyForProfile(id) {
  if (!id) return ''
  const map = loadCreatorKeyMap()
  return map[id] || ''
}

function setCreatorKeyForProfile(id, key) {
  if (!id || !key) return
  const map = loadCreatorKeyMap()
  map[id] = key
  localStorage.setItem(CREATOR_KEY_STORAGE_KEY, JSON.stringify(map))
}

function syncViewportFlags() {
  isMobile.value = window.innerWidth <= 768
  isStandaloneMode.value = window.matchMedia('(display-mode: standalone)').matches || window.navigator.standalone === true
  browserSpeechSupported.value = Boolean(window.SpeechRecognition || window.webkitSpeechRecognition)
  mediaRecorderSupported.value = Boolean(window.MediaRecorder && navigator.mediaDevices?.getUserMedia)
}

function fillTodayDefaults() {
  quickForm.title = ''
  quickForm.detail = ''
  quickForm.entryType = 'task'
  quickForm.bucket = 'planned'
  quickForm.plannedFor = new Date().toISOString().slice(0, 10)
  quickForm.remindAt = ''
  quickForm.repeatType = 'none'
  quickForm.repeatInterval = 1
  quickForm.repeatUntil = ''
  quickForm.priority = 'medium'
  quickForm.notifyEmail = profile.notify_email || ''
  if (isMobile.value) {
    quickAdvancedVisible.value = false
  }
}

function applyQuickPreset(type) {
  fillTodayDefaults()
  if (type === 'inbox') {
    quickForm.bucket = 'inbox'
  } else if (type === 'event') {
    quickForm.entryType = 'event'
  }
}

function resetTaskForm() {
  taskForm.id = ''
  taskForm.kind = activeKind.value
  taskForm.entryType = 'task'
  taskForm.bucket = 'planned'
  taskForm.title = ''
  taskForm.detail = ''
  taskForm.notes = ''
  taskForm.plannedFor = new Date().toISOString().slice(0, 10)
  taskForm.remindAt = ''
  taskForm.repeatType = 'none'
  taskForm.repeatInterval = 1
  taskForm.repeatUntil = ''
  taskForm.priority = 'medium'
  taskForm.status = 'open'
  taskForm.notifyEmail = profile.notify_email || ''
  taskForm.cancelReason = ''
  taskForm.postponeReason = ''
  taskCommentsPreview.value = []
  taskActivities.value = []
  taskActivitiesPreview.value = []
}

async function plannerFetch(url, options = {}) {
  const headers = { ...(options.headers || {}) }
  if (password.value) {
    headers['X-Password'] = password.value
  }
  if (creatorKey.value) {
    headers['X-Creator-Key'] = creatorKey.value
  }
  if (options.body && !(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }
  const response = await fetch(url, {
    ...options,
    headers
  })
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}))
    throw new Error(payload.error || '请求失败')
  }
  return response
}

function persistSession() {
  localStorage.setItem('planner_profile_id', profileId.value)
  localStorage.setItem('planner_password', password.value)
  localStorage.setItem('planner_active_kind', activeKind.value)
  localStorage.setItem('planner_active_view', activeView.value)
  if (profileId.value && creatorKey.value) {
    setCreatorKeyForProfile(profileId.value, creatorKey.value)
  }
}

function restoreSession() {
  const savedProfileId = localStorage.getItem('planner_profile_id')
  const savedPassword = localStorage.getItem('planner_password')
  if (!savedProfileId || !savedPassword) {
    activeKind.value = localStorage.getItem('planner_active_kind') || defaultModeByTime()
    return
  }
  profileId.value = savedProfileId
  password.value = savedPassword
  const legacyCreatorKey = localStorage.getItem('planner_creator_key') || ''
  if (legacyCreatorKey && !getCreatorKeyForProfile(savedProfileId)) {
    setCreatorKeyForProfile(savedProfileId, legacyCreatorKey)
  }
  creatorKey.value = getCreatorKeyForProfile(savedProfileId)
  activeKind.value = localStorage.getItem('planner_active_kind') || defaultModeByTime()
}

function clearSession() {
  profileId.value = ''
  password.value = ''
  creatorKey.value = ''
  localStorage.removeItem('planner_profile_id')
  localStorage.removeItem('planner_password')
  localStorage.removeItem('planner_creator_key')
}

async function createProfile() {
  if (createForm.password.length < 4) {
    ElMessage.warning('密码至少 4 位')
    return
  }
  creating.value = true
  try {
    const response = await fetch(`${API_BASE}/profile`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: createForm.name,
        password: createForm.password,
        notify_email: createForm.notifyEmail
      })
    })
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '创建失败')
    }
    profileId.value = data.id
    password.value = createForm.password
    creatorKey.value = data.creator_key || ''
    setCreatorKeyForProfile(data.id, creatorKey.value)
    persistSession()
    await loadProfile()
    fillTodayDefaults()
    resetTaskForm()
    await refreshBoard()
    ElMessage.success('档案已创建')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    creating.value = false
  }
}

async function loginProfile() {
  if (!loginForm.password) {
    ElMessage.warning('请输入密码')
    return
  }
  loggingIn.value = true
  try {
    const response = await fetch(`${API_BASE}/profile/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: loginForm.password })
    })
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '登录失败')
    }
    profileId.value = data.id
    password.value = loginForm.password
    creatorKey.value = getCreatorKeyForProfile(data.id)
    profile.id = data.id
    profile.name = data.name
    profile.notify_email = data.notify_email || ''
    profile.expires_at = data.expires_at || ''
    profile.mode_default = data.mode_default || defaultModeByTime()
    modeHint.value = data.mode_hint || '模式已准备好，可以随时切换。'
    activeKind.value = localStorage.getItem('planner_active_kind') || profile.mode_default || defaultModeByTime()
    persistSession()
    profileForm.name = profile.name
    profileForm.notifyEmail = profile.notify_email
    profileForm.extendDays = null
    fillTodayDefaults()
    resetTaskForm()
    await refreshBoard()
    ElMessage.success('已进入档案')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    loggingIn.value = false
  }
}

async function loadProfile() {
  if (!profileId.value) return
  const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}`)
  const data = await response.json()
  const payload = data.profile || {}
  profile.id = payload.id || profileId.value
  profile.name = payload.name || '事项档案'
  profile.notify_email = payload.notify_email || ''
  profile.expires_at = payload.expires_at || ''
  profile.mode_default = payload.mode_default || defaultModeByTime()
  modeHint.value = payload.meta?.mode_hint || plannerModeCopy()
  profileForm.name = profile.name
  profileForm.notifyEmail = profile.notify_email
  profileForm.extendDays = null
  if (!localStorage.getItem('planner_active_kind')) {
    activeKind.value = profile.mode_default
  }
}

function switchKind(kind) {
  activeKind.value = kind
  persistSession()
}

function switchView(view) {
  activeView.value = view
  localStorage.setItem('planner_active_view', view)
  if (view === 'minutes' && meetings.value.length === 0) {
    loadMeetings()
  }
}

async function refreshBoard() {
  if (!profileId.value) return
  timelineLoading.value = true
  try {
    const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/timeline?kind=${activeKind.value}`)
    const data = await response.json()
    board.value = { ...createEmptyBoard(), ...(data.board || {}) }
    modeHint.value = board.value.mode_hint || plannerModeCopy()
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    timelineLoading.value = false
  }
  loadMeetings()
}

async function refreshAll() {
  await loadProfile()
  await refreshBoard()
  loadMeetings()
}

async function createQuickTask() {
  if (!quickForm.title.trim()) {
    ElMessage.warning('先写一个事项标题')
    return
  }
  savingQuick.value = true
  try {
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks`, {
      method: 'POST',
      body: JSON.stringify({
        kind: activeKind.value,
        entry_type: quickForm.entryType,
        bucket: quickForm.entryType === 'event' ? 'planned' : quickForm.bucket,
        title: quickForm.title,
        detail: quickForm.detail,
        planned_for: quickForm.plannedFor,
        remind_at: quickForm.remindAt,
        repeat_type: quickForm.repeatType,
        repeat_interval: quickForm.repeatInterval,
        repeat_until: quickForm.repeatUntil,
        priority: quickForm.priority,
        notify_email: quickForm.notifyEmail
      })
    })
    ElMessage.success(activeKind.value === 'work' ? '工作事项已记录' : '生活事项已记录')
    fillTodayDefaults()
    await refreshBoard()
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    savingQuick.value = false
  }
}

function openTask(task) {
  taskForm.id = task.id
  taskForm.kind = task.kind
  taskForm.entryType = task.entry_type || 'task'
  taskForm.bucket = task.bucket || 'planned'
  taskForm.title = task.title
  taskForm.detail = task.detail || ''
  taskForm.notes = task.notes || ''
  taskForm.priority = task.priority || 'medium'
  taskForm.status = task.status || 'open'
  taskForm.plannedFor = task.planned_for || ''
  taskForm.remindAt = task.remind_at ? task.remind_at.slice(0, 16) : ''
  taskForm.repeatType = task.repeat_type || 'none'
  taskForm.repeatInterval = task.repeat_interval || 1
  taskForm.repeatUntil = task.repeat_until ? task.repeat_until.slice(0, 16) : ''
  taskForm.notifyEmail = task.notify_email || ''
  taskForm.cancelReason = task.cancel_reason || ''
  taskForm.postponeReason = task.last_postpone_reason || ''
  loadTaskCommentsPreview(task.id)
  loadTaskActivities(task.id)
  taskDrawerVisible.value = true
}

async function loadTaskCommentsPreview(taskId) {
  if (!taskId) {
    taskCommentsPreview.value = []
    return
  }
  try {
    const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskId}/comments`)
    const data = await response.json()
    const list = data.comments || []
    taskCommentsPreview.value = list.slice(-3).reverse()
  } catch {
    taskCommentsPreview.value = []
  }
}

async function loadTaskActivities(taskId) {
  if (!taskId) {
    taskActivities.value = []
    taskActivitiesPreview.value = []
    return
  }
  try {
    const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskId}/activities`)
    const data = await response.json()
    const list = data.activities || []
    taskActivities.value = list.slice().reverse()
    taskActivitiesPreview.value = taskActivities.value.slice(0, 5)
  } catch {
    taskActivities.value = []
    taskActivitiesPreview.value = []
  }
}

async function saveTask() {
  if (!taskForm.id || !taskForm.title.trim()) {
    ElMessage.warning('标题不能为空')
    return
  }
  savingTask.value = true
  try {
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskForm.id}`, {
      method: 'PUT',
      body: JSON.stringify({
        kind: taskForm.kind,
        entry_type: taskForm.entryType,
        bucket: taskForm.entryType === 'event' ? 'planned' : taskForm.bucket,
        title: taskForm.title,
        detail: taskForm.detail,
        notes: taskForm.notes,
        planned_for: taskForm.plannedFor,
        remind_at: taskForm.remindAt,
        repeat_type: taskForm.repeatType,
        repeat_interval: taskForm.repeatInterval,
        repeat_until: taskForm.repeatUntil,
        priority: taskForm.priority,
        status: taskForm.status,
        notify_email: taskForm.notifyEmail,
        cancel_reason: taskForm.cancelReason,
        postpone_reason: taskForm.postponeReason
      })
    })
    taskDrawerVisible.value = false
    ElMessage.success('事项已更新')
    await refreshBoard()
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    savingTask.value = false
  }
}

async function removeTask() {
  if (!taskForm.id) return
  try {
    await ElMessageBox.confirm('删除后该事项和评论都会清空，确认继续？', '删除确认', {
      type: 'warning'
    })
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskForm.id}`, {
      method: 'DELETE'
    })
    taskDrawerVisible.value = false
    ElMessage.success('事项已删除')
    await refreshBoard()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

async function updateTask(task, payload, successMessage = '') {
  try {
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${task.id}`, {
      method: 'PUT',
      body: JSON.stringify(payload)
    })
    if (successMessage) {
      ElMessage.success(successMessage)
    }
    await refreshBoard()
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function setTaskStatus(task, status) {
  if (status === 'cancelled') {
    await cancelTask(task)
    return
  }
  await updateTask(task, { status }, status === 'done' ? '已完成' : '状态已更新')
}

async function cancelTask(task) {
  try {
    const { value } = await ElMessageBox.prompt('填写取消原因', '取消事项', {
      inputPlaceholder: '例如：不再需要 / 条件变化 / 不是现在要做'
    })
    await updateTask(task, { status: 'cancelled', cancel_reason: value || '用户取消' }, '已取消')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '取消失败')
    }
  }
}

async function postponeTask(task) {
  const base = task.display_date || task.planned_for || new Date().toISOString().slice(0, 10)
  const date = new Date(`${base}T00:00:00`)
  date.setDate(date.getDate() + 1)
  const next = date.toISOString().slice(0, 10)
  try {
    const { value } = await ElMessageBox.prompt('填写顺延原因（可选）', '顺延事项', {
      inputPlaceholder: '例如：今天优先级下调 / 等待他人反馈'
    })
    await updateTask(task, { planned_for: next, status: 'open', postpone_reason: value || '手动顺延' }, '已顺延一天')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '顺延失败')
    }
  }
}

async function moveTaskBucket(task, bucket) {
  const payload = { bucket }
  if (bucket === 'planned') {
    payload.planned_for = new Date().toISOString().slice(0, 10)
  }
  await updateTask(task, payload, '已更新分区')
}

async function scheduleInboxTask(task) {
  await updateTask(task, {
    bucket: 'planned',
    planned_for: new Date().toISOString().slice(0, 10)
  }, '已安排到今天')
}

async function openCommentDrawer(task) {
  currentCommentTask.value = task
  commentDrawerVisible.value = true
  commentForm.content = ''
  await loadComments(task.id)
}

async function openActivityDrawer(task) {
  if (!task?.id) return
  await loadTaskActivities(task.id)
  activityDrawerVisible.value = true
}

async function loadComments(taskId) {
  const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskId}/comments`)
  const data = await response.json()
  comments.value = data.comments || []
  if (taskForm.id === taskId) {
    taskCommentsPreview.value = [...comments.value].slice(-3).reverse()
  }
}

async function submitComment() {
  if (!currentCommentTask.value || !commentForm.content.trim()) {
    ElMessage.warning('请输入评论内容')
    return
  }
  savingComment.value = true
  try {
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${currentCommentTask.value.id}/comments`, {
      method: 'POST',
      body: JSON.stringify({ content: commentForm.content })
    })
    commentForm.content = ''
    await loadComments(currentCommentTask.value.id)
    await loadTaskActivities(currentCommentTask.value.id)
    await refreshBoard()
    ElMessage.success('评论已保存')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    savingComment.value = false
  }
}

function importCalendar(task) {
  const url = `${API_BASE}/profile/${profileId.value}/tasks/${task.id}/calendar?password=${encodeURIComponent(password.value)}`
  window.open(url, '_blank')
}

async function saveProfile() {
  savingProfile.value = true
  try {
    const payload = {
      name: profileForm.name,
      notify_email: profileForm.notifyEmail
    }
    if (profileForm.extendDays) {
      payload.expires_in = profileForm.extendDays
    }
    await plannerFetch(`${API_BASE}/profile/${profileId.value}`, {
      method: 'PUT',
      body: JSON.stringify(payload)
    })
    settingsVisible.value = false
    profileForm.extendDays = null
    await loadProfile()
    ElMessage.success('档案设置已更新')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    savingProfile.value = false
  }
}

async function deleteProfile() {
  try {
    await ElMessageBox.confirm('删除档案后事项和评论都会被清空，确认继续？', '删除确认', {
      type: 'warning'
    })
    await plannerFetch(`${API_BASE}/profile/${profileId.value}`, {
      method: 'DELETE'
    })
    logout()
    settingsVisible.value = false
    ElMessage.success('档案已删除')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}


async function loadMeetings() {
  if (!profileId.value) return
  loadingMeetings.value = true
  try {
    const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/meetings`)
    const data = await response.json()
    meetings.value = data.meetings || []
  } catch (error) {
    ElMessage.error('\u52a0\u8f7d\u4f1a\u8bae\u7eaa\u8981\u5931\u8d25\uff1a' + error.message)
  } finally {
    loadingMeetings.value = false
  }
}

function openMeetingDialog() {
  resetMeetingForm()
  meetingForm.meetingDate = new Date().toISOString().slice(0, 10)
  meetingDialogVisible.value = true
}

function openMeetingEdit(m) {
  applyMeetingToForm(m)
  meetingDialogVisible.value = true
}

async function saveMeeting() {
  if (!meetingForm.title.trim()) {
    ElMessage.warning('\u8bf7\u586b\u5199\u4f1a\u8bae\u6807\u9898')
    return
  }
  if (meetingAudioUploadPromise) {
    ElMessage.info('等待录音上传和转写完成…')
    try {
      await meetingAudioUploadPromise
    } catch {
      // 上传错误已经在录音流程里提示，这里继续允许手动保存文本纪要
    }
  }
  savingMeeting.value = true
  const participants = meetingForm.participantsText
    .split(/[,\uff0c\u3001]/)
    .map(s => s.trim())
    .filter(Boolean)
  const tags = meetingForm.tagsText
    .split(/[,\uff0c\u3001]/)
    .map(s => s.trim())
    .filter(Boolean)
  const payload = buildMeetingSnapshot(participants, tags)
  const isCreating = !meetingForm.id
  try {
    let savedMeeting = null
    if (meetingForm.id) {
      const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/meetings/${meetingForm.id}`, {
        method: 'PUT',
        body: JSON.stringify(payload)
      })
      const data = await response.json()
      savedMeeting = data.meeting || {}
      ElMessage.success('会议纪要已保存，可继续编辑')
    } else {
      const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/meetings`, {
        method: 'POST',
        body: JSON.stringify(payload)
      })
      const data = await response.json()
      savedMeeting = data.meeting || {}
      ElMessage.success('会议纪要已创建并保存，可继续补充内容')
    }
    const mergedMeeting = mergeMeetingData(payload, savedMeeting)
    applyMeetingToForm(mergedMeeting, { preserveLocalAudio: true })
    upsertMeeting(mergedMeeting)
    const shouldAutoSummarize = Boolean(isCreating && mergedMeeting.id && mergedMeeting.content && !mergedMeeting.summary)
    await loadMeetings()
    const refreshedMeeting = meetings.value.find(item => item.id === mergedMeeting.id)
    if (refreshedMeeting) {
      applyMeetingToForm(refreshedMeeting, { preserveLocalAudio: true })
    }
    if (shouldAutoSummarize) {
      const target = meetings.value.find(item => item.id === mergedMeeting.id) || mergedMeeting
      await summarizeMeeting(target)
    }
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    savingMeeting.value = false
  }
}

async function deleteMeeting() {
  if (!meetingForm.id) return
  try {
    await ElMessageBox.confirm('\u786e\u5b9a\u5220\u9664\u8fd9\u4efd\u4f1a\u8bae\u7eaa\u8981\uff1f', '\u5220\u9664\u786e\u8ba4', { type: 'warning' })
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/meetings/${meetingForm.id}`, {
      method: 'DELETE'
    })
    meetingDialogVisible.value = false
    ElMessage.success('\u5df2\u5220\u9664')
    await loadMeetings()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.message || '\u5220\u9664\u5931\u8d25')
  }
}

async function deleteMeetingConfirm(m) {
  try {
    await ElMessageBox.confirm(`\u786e\u5b9a\u5220\u9664\u300c${m.title}\u300d\uff1f`, '\u5220\u9664\u786e\u8ba4', { type: 'warning' })
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/meetings/${m.id}`, {
      method: 'DELETE'
    })
    ElMessage.success('\u5df2\u5220\u9664')
    await loadMeetings()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.message || '\u5220\u9664\u5931\u8d25')
  }
}

async function summarizeMeeting(m) {
  const targetId = m.id || meetingForm.id
  if (!targetId || !(m.content || meetingForm.content)) return
  summarizingId.value = targetId
  try {
    const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/meetings/${targetId}/summarize`, {
      method: 'POST'
    })
    const data = await response.json()
    if (data.code === 0) {
      meetingForm.summary = data.summary || ''
      meetingForm.actionItems = data.action_items || '[]'
      if (m.id) {
        const found = meetings.value.find(x => x.id === m.id)
        if (found) {
          found.summary = data.summary || ''
          found.action_items = data.action_items || '[]'
        }
      }
      ElMessage.success('AI \u603b\u7ed3\u5b8c\u6210')
    } else {
      ElMessage.error(data.error || '\u603b\u7ed3\u5931\u8d25')
    }
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    summarizingId.value = ''
  }
}

let meetingRecognition = null
let meetingMediaRecorder = null
let meetingMediaStream = null
let meetingVoiceChunks = []
let meetingTranscriptText = ''
const meetingInterimText = ref('')

function appendTranscriptToMeetingContent(text) {
  const normalized = (text || '').trim()
  if (!normalized) return
  const exists = meetingForm.content.includes(normalized)
  if (exists) return
  const prefix = meetingForm.content ? meetingForm.content + '\n\n' : ''
  meetingForm.content = prefix + normalized
}

async function uploadMeetingRecording(blob) {
  const formData = new FormData()
  formData.append('file', blob, `meeting_${Date.now()}.webm`)
  meetingAudioProcessing.value = true
  meetingAudioError.value = ''
  try {
    const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/recordings`, {
      method: 'POST',
      body: formData
    })
    const data = await response.json()
    if (data.recording_url) {
      meetingForm.recordingUrl = data.recording_url
    }
    if (data.transcript) {
      appendTranscriptToMeetingContent(data.transcript)
      meetingTranscriptText = data.transcript
      ElMessage.success('录音已转写为文字')
    } else if (data.transcript_error) {
      meetingAudioError.value = data.transcript_error
      ElMessage.warning(data.transcript_error)
    } else if (data.transcript_status === 'skipped') {
      meetingAudioError.value = '录音已保存，但未启用语音转文字服务。'
    }
  } catch (error) {
    meetingAudioError.value = error.message || '录音上传或转写失败'
    ElMessage.error(meetingAudioError.value)
    throw error
  } finally {
    meetingAudioProcessing.value = false
    meetingAudioUploadPromise = null
  }
}

async function startMeetingVoice() {
  if (meetingVoiceState.value === 'recording') {
    // --- STOP ---
    if (meetingRecognition) {
      try { meetingRecognition.stop() } catch {}
      meetingRecognition = null
    }
    if (meetingMediaRecorder && meetingMediaRecorder.state === 'recording') {
      meetingMediaRecorder.stop()
    }
    if (meetingMediaStream) {
      meetingMediaStream.getTracks().forEach(t => t.stop())
      meetingMediaStream = null
    }
    meetingVoiceState.value = 'idle'

    // Append transcript to meeting content
    appendTranscriptToMeetingContent(meetingTranscriptText)

    // Create local blob URL + upload to backend
    if (meetingVoiceChunks.length > 0) {
      const blob = new Blob(meetingVoiceChunks, { type: meetingMediaRecorder?.mimeType || 'audio/webm' })
      // Local URL for instant playback
      revokeMeetingLocalAudio()
      meetingLocalAudioUrl.value = URL.createObjectURL(blob)
      meetingAudioUploadPromise = uploadMeetingRecording(blob)
      await meetingAudioUploadPromise.catch(() => {})
    }

    if (meetingTranscriptText.trim() || meetingInterimText.value.trim()) {
      ElMessage.success('\u8f6c\u5199\u6587\u5b57\u5df2\u5199\u5165\u4f1a\u8bae\u5185\u5bb9')
    }
    return
  }

  // --- START ---
  const SpeechAPI = window.SpeechRecognition || window.webkitSpeechRecognition
  if (!SpeechAPI && !navigator.mediaDevices?.getUserMedia) {
    ElMessage.warning('\u5f53\u524d\u6d4f\u89c8\u5668\u4e0d\u652f\u6301\u5f55\u97f3\u548c\u8bed\u97f3\u8f6c\u5199')
    return
  }

  meetingTranscriptText = ''
  meetingInterimText.value = ''
  meetingVoiceChunks = []
  meetingAudioError.value = ''

  // Start MediaRecorder (audio backup)
  if (navigator.mediaDevices?.getUserMedia) {
    navigator.mediaDevices.getUserMedia({ audio: true }).then(stream => {
      meetingMediaStream = stream
      const mimeType = MediaRecorder.isTypeSupported('audio/webm;codecs=opus')
        ? 'audio/webm;codecs=opus'
        : 'audio/webm'
      meetingMediaRecorder = new MediaRecorder(stream, { mimeType })
      meetingMediaRecorder.ondataavailable = e => {
        if (e.data.size > 0) meetingVoiceChunks.push(e.data)
      }
      meetingMediaRecorder.start()
    }).catch(() => {})
  }

  // Start SpeechRecognition (ASR)
  if (SpeechAPI) {
    try {
      meetingRecognition = new SpeechAPI()
      meetingRecognition.lang = 'zh-CN'
      meetingRecognition.continuous = true
      meetingRecognition.interimResults = true
      meetingRecognition.onresult = (event) => {
        let finalText = ''
        let interimText = ''
        for (let i = 0; i < event.results.length; i += 1) {
          const text = event.results[i][0]?.transcript || ''
          if (event.results[i].isFinal) {
            finalText += text
          } else {
            interimText += text
          }
        }
        if (finalText) {
          meetingTranscriptText = [meetingTranscriptText, finalText].filter(Boolean).join('')
        }
        meetingInterimText.value = interimText
      }
      meetingRecognition.onerror = (event) => {
        if (event?.error !== 'no-speech' && event?.error !== 'aborted') {
          console.warn('Meeting ASR error:', event?.error)
        }
      }
      meetingRecognition.start()
    } catch (e) {
      console.warn('Meeting ASR start failed:', e)
    }
  }

  meetingVoiceState.value = 'recording'
  if (meetingInterimText.value || meetingTranscriptText) {
    ElMessage.info('\u5f00\u59cb\u5f55\u97f3\uff0c\u8bf4\u8bdd\u5185\u5bb9\u5c06\u5b9e\u65f6\u8f6c\u5199')
  } else {
    ElMessage.info('\u5f00\u59cb\u5f55\u97f3')
  }
}

async function parseAI() {
  if (!aiText.value.trim()) {
    ElMessage.warning('先输入要整理的内容')
    return
  }
  parsingAI.value = true
  try {
    const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/ai/parse`, {
      method: 'POST',
      body: JSON.stringify({
        text: aiText.value,
        default_kind: activeKind.value
      })
    })
    const data = await response.json()
    aiSuggestions.value = data.tasks || []
    ElMessage.success(data.provider === 'minimax' ? 'AI 已整理完成' : '已按本地规则整理')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    parsingAI.value = false
  }
}

async function applyAISuggestions() {
  if (aiSuggestions.value.length === 0) return
  applyingAI.value = true
  try {
    const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/batch`, {
      method: 'POST',
      body: JSON.stringify({
        tasks: aiSuggestions.value.map(item => ({
          kind: item.kind,
          entry_type: item.entry_type,
          bucket: item.bucket,
          title: item.title,
          detail: item.detail,
          notes: item.notes,
          priority: item.priority,
          status: item.status,
          planned_for: item.planned_for,
          remind_at: item.remind_at,
          cancel_reason: item.cancel_reason
        }))
      })
    })
    const data = await response.json()
    aiDialogVisible.value = false
    aiSuggestions.value = []
    aiText.value = ''
    await refreshBoard()
    ElMessage.success(`AI 整理出的 ${data.created_count || 0} 条事项已写入`)
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    applyingAI.value = false
  }
}

function openAdviceDialog() {
  adviceDialogVisible.value = true
}

async function triggerInstallPrompt() {
  if (deferredInstallPrompt) {
    deferredInstallPrompt.prompt()
    try {
      await deferredInstallPrompt.userChoice
    } catch {
      // ignore
    }
    deferredInstallPrompt = null
    installPromptAvailable.value = false
    return
  }
  await ElMessageBox.alert(
    '安卓 Chrome：打开浏览器菜单，选择“添加到主屏幕”。<br>iPhone Safari：点击底部“分享”，再选择“添加到主屏幕”。',
    '加入手机桌面',
    {
      confirmButtonText: '知道了',
      dangerouslyUseHTMLString: true
    }
  )
}

async function requestAdvice() {
  adviceLoading.value = true
  try {
    const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/ai/advise`, {
      method: 'POST',
      body: JSON.stringify({
        kind: activeKind.value,
        mode: adviceMode.value,
        text: adviceText.value
      })
    })
    const data = await response.json()
    adviceProvider.value = data.provider || 'fallback'
    adviceResult.summary = data.advice?.summary || ''
    adviceResult.insights = data.advice?.insights || []
    adviceResult.suggestions = data.advice?.suggestions || []
    ElMessage.success(adviceMode.value === 'summary' ? '总结已生成' : '规划建议已生成')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    adviceLoading.value = false
  }
}

async function loadReview() {
  if (!profileId.value) return
  reviewLoading.value = true
  try {
    const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/review?kind=${activeKind.value}&period=${reviewPeriod.value}`)
    const data = await response.json()
    review.period = data.review?.period || reviewPeriod.value
    review.label = data.review?.label || ''
    review.summary = data.review?.summary || ''
    review.stats = data.review?.stats || {}
    review.wins = data.review?.wins || []
    review.drifts = data.review?.drifts || []
    review.suggestions = data.review?.suggestions || []
    review.highlights = data.review?.highlights || []
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    reviewLoading.value = false
  }
}

async function loadAdminProfiles() {
  if (!adminPassword.value) {
    ElMessage.warning('请输入超管密码')
    return
  }
  loadingAdmin.value = true
  try {
    const url = new URL(`${window.location.origin}${API_BASE}/admin/list`)
    url.searchParams.set('admin_password', adminPassword.value)
    if (adminKeyword.value.trim()) {
      url.searchParams.set('keyword', adminKeyword.value.trim())
    }
    const response = await fetch(url.toString())
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '查询失败')
    }
    adminItems.value = data.items || []
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    loadingAdmin.value = false
  }
}

async function openAdminProfile(item) {
  if (!adminPassword.value) {
    ElMessage.warning('请输入超管密码')
    return
  }
  try {
    const response = await fetch(`${API_BASE}/admin/${item.id}?admin_password=${encodeURIComponent(adminPassword.value)}`)
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '加载失败')
    }
    adminDetail.profile.id = data.profile?.id || item.id
    adminDetail.profile.name = data.profile?.name || ''
    adminDetail.profile.notify_email = data.profile?.notify_email || ''
    adminDetail.tasks = data.tasks || []
    adminDetail.extendDays = null
    adminDetailVisible.value = true
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function saveAdminProfile() {
  if (!adminPassword.value || !adminDetail.profile.id) return
  savingAdminProfile.value = true
  try {
    const payload = {
      name: adminDetail.profile.name,
      notify_email: adminDetail.profile.notify_email
    }
    if (adminDetail.extendDays) {
      payload.expires_in = adminDetail.extendDays
    }
    const response = await fetch(`${API_BASE}/admin/${adminDetail.profile.id}?admin_password=${encodeURIComponent(adminPassword.value)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '保存失败')
    }
    adminDetail.extendDays = null
    await loadAdminProfiles()
    ElMessage.success('超管更新成功')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    savingAdminProfile.value = false
  }
}

async function deleteAdminProfile(item) {
  if (!adminPassword.value) {
    ElMessage.warning('请输入超管密码')
    return
  }
  try {
    await ElMessageBox.confirm(`确认删除档案「${item.name}」？`, '超管删除确认', { type: 'warning' })
    const response = await fetch(`${API_BASE}/admin/${item.id}?admin_password=${encodeURIComponent(adminPassword.value)}`, {
      method: 'DELETE'
    })
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data.error || '删除失败')
    }
    if (adminDetail.profile.id === item.id) {
      adminDetailVisible.value = false
    }
    await loadAdminProfiles()
    ElMessage.success('档案已删除')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

let beforeInstallPromptHandler = null
let appInstalledHandler = null

function openVoice(target) {
  if (!browserSpeechSupported.value && !mediaRecorderSupported.value) {
    ElMessage.warning('当前浏览器不支持语音录入')
    return
  }
  voiceTarget.value = target
  voiceDialogVisible.value = true
  resetVoiceDraft()
  startVoiceCapture()
}

function repeatTypeLabel(type) {
  return {
    none: '不重复',
    daily: '每天',
    weekdays: '工作日',
    weekly: '每周',
    monthly: '每月'
  }[type] || '不重复'
}

function formatTimeOnly(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }
  return date.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit'
  })
}

function weekdayLabelFromValue(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }
  return ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][date.getDay()]
}

function repeatSummary(task) {
  if (!task?.remind_at) return ''
  const repeatType = task.repeat_type || 'none'
  if (repeatType === 'none') return ''
  const interval = Number(task.repeat_interval || 1)
  const timePart = formatTimeOnly(task.remind_at)
  const untilPart = task.repeat_until ? `，截止 ${formatDateTime(task.repeat_until)}` : ''
  switch (repeatType) {
    case 'daily':
      return interval > 1 ? `每 ${interval} 天 ${timePart}${untilPart}` : `每天 ${timePart}${untilPart}`
    case 'weekdays':
      return `工作日 ${timePart}${untilPart}`
    case 'weekly':
      return interval > 1
        ? `每 ${interval} 周${weekdayLabelFromValue(task.remind_at)} ${timePart}${untilPart}`
        : `${weekdayLabelFromValue(task.remind_at)} ${timePart}${untilPart}`
    case 'monthly':
      return interval > 1
        ? `每 ${interval} 月 ${new Date(task.remind_at).getDate()} 日 ${timePart}${untilPart}`
        : `每月 ${new Date(task.remind_at).getDate()} 日 ${timePart}${untilPart}`
    default:
      return ''
  }
}

function formatVoiceDuration(totalSeconds) {
  const minutes = String(Math.floor(totalSeconds / 60)).padStart(2, '0')
  const seconds = String(totalSeconds % 60).padStart(2, '0')
  return `${minutes}:${seconds}`
}

async function startVoiceCapture() {
  stopVoiceCapture({ keepDraft: true })
  voiceCaptureToken += 1
  const captureToken = voiceCaptureToken
  voiceError.value = ''
  voiceTranscript.value = ''
  voiceInterimTranscript.value = ''
  voiceDuration.value = 0
  voiceState.value = 'idle'
  voiceChunks = []
  let startedRecorder = false
  let startedRecognition = false

  if (voiceAudioURL.value) {
    URL.revokeObjectURL(voiceAudioURL.value)
    voiceAudioURL.value = ''
  }

  if (mediaRecorderSupported.value) {
    try {
      mediaStream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const preferredTypes = ['audio/webm;codecs=opus', 'audio/webm', 'audio/mp4', 'audio/ogg']
      const mimeType = preferredTypes.find(type => window.MediaRecorder?.isTypeSupported?.(type)) || ''
      mediaRecorder = mimeType ? new MediaRecorder(mediaStream, { mimeType }) : new MediaRecorder(mediaStream)
      mediaRecorder.ondataavailable = (event) => {
        if (event.data?.size) {
          voiceChunks.push(event.data)
        }
      }
      mediaRecorder.onstop = () => {
        if (captureToken !== voiceCaptureToken) {
          return
        }
        if (voiceChunks.length > 0) {
          const blob = new Blob(voiceChunks, { type: mediaRecorder?.mimeType || 'audio/webm' })
          voiceAudioURL.value = URL.createObjectURL(blob)
        }
      }
      mediaRecorder.start()
      startedRecorder = true
    } catch (error) {
      voiceError.value = error?.message || '麦克风权限获取失败'
    }
  }

  if (browserSpeechSupported.value) {
    try {
      const API = window.SpeechRecognition || window.webkitSpeechRecognition
      recognition = new API()
      recognition.lang = 'zh-CN'
      recognition.continuous = true
      recognition.interimResults = true
      recognition.onresult = (event) => {
        let finalText = ''
        let interimText = ''
        for (let i = 0; i < event.results.length; i += 1) {
          const text = event.results[i][0]?.transcript || ''
          if (event.results[i].isFinal) {
            finalText += text
          } else {
            interimText += text
          }
        }
        if (finalText) {
          voiceTranscript.value = [voiceTranscript.value, finalText].filter(Boolean).join('')
        }
        voiceInterimTranscript.value = interimText
      }
      recognition.onerror = (event) => {
        if (event?.error !== 'no-speech' && event?.error !== 'aborted') {
          voiceError.value = `转写失败: ${event?.error || '未知错误'}`
        }
      }
      recognition.onend = () => {
        recognition = null
        if (voiceState.value === 'recording' && (!mediaRecorder || mediaRecorder.state === 'inactive')) {
          voiceState.value = voiceTranscript.value.trim() ? 'ready' : 'idle'
        }
      }
      recognition.start()
      startedRecognition = true
    } catch (error) {
      voiceError.value = voiceError.value || error?.message || '启动语音转写失败'
    }
  }

  if (!startedRecorder && !startedRecognition) {
    voiceState.value = voiceTranscript.value.trim() ? 'ready' : 'idle'
    if (!voiceError.value) {
      voiceError.value = '当前浏览器未成功启动语音录入，请检查麦克风权限。'
    }
    return
  }

  voiceState.value = 'recording'
  if (voiceTimer) {
    clearInterval(voiceTimer)
  }
  voiceTimer = setInterval(() => {
    voiceDuration.value += 1
  }, 1000)
}

function stopMediaStream() {
  if (mediaStream) {
    mediaStream.getTracks().forEach(track => track.stop())
    mediaStream = null
  }
}

function stopVoiceCapture(options = {}) {
  const { keepDraft = false } = options
  if (recognition) {
    try {
      recognition.stop()
    } catch {
      // ignore
    }
    recognition = null
  }
  if (mediaRecorder) {
    if (mediaRecorder.state !== 'inactive') {
      mediaRecorder.stop()
    }
    mediaRecorder = null
  }
  if (voiceTimer) {
    clearInterval(voiceTimer)
    voiceTimer = null
  }
  stopMediaStream()
  voiceInterimTranscript.value = ''
  voiceState.value = keepDraft && (voiceTranscript.value.trim() || voiceAudioURL.value) ? 'ready' : 'idle'
}

function resetVoiceDraft() {
  voiceCaptureToken += 1
  stopVoiceCapture({ keepDraft: false })
  voiceTranscript.value = ''
  voiceInterimTranscript.value = ''
  voiceError.value = ''
  voiceDuration.value = 0
  voiceChunks = []
  if (voiceAudioURL.value) {
    URL.revokeObjectURL(voiceAudioURL.value)
    voiceAudioURL.value = ''
  }
}

function closeVoiceDialog() {
  resetVoiceDraft()
  voiceDialogVisible.value = false
}

function applyVoiceDraft() {
  const finalText = [voiceTranscript.value, voiceInterimTranscript.value].filter(Boolean).join('').trim()
  if (!finalText) {
    ElMessage.warning('还没有可用的转写内容')
    return
  }
  if (voiceTarget.value === 'quick') {
    if (!quickForm.title.trim()) {
      const segments = finalText.split(/[。！!？?\n]/).map(item => item.trim()).filter(Boolean)
      quickForm.title = segments[0] || finalText.slice(0, 40)
      if (!quickForm.detail.trim()) {
        quickForm.detail = segments.length > 1 ? segments.slice(1).join('\n') : finalText
      }
    } else {
      quickForm.detail = [quickForm.detail, finalText].filter(Boolean).join('\n')
    }
  } else {
    aiText.value = [aiText.value, finalText].filter(Boolean).join('\n')
  }
  voiceDialogVisible.value = false
  resetVoiceDraft()
  ElMessage.success('语音内容已写入')
}

function priorityLabel(priority) {
  return { high: '高优先级', medium: '中优先级', low: '低优先级' }[priority] || '中优先级'
}

function priorityTagType(priority) {
  return { high: 'danger', medium: 'warning', low: 'info' }[priority] || ''
}

function statusLabel(status) {
  return {
    open: '未开始',
    in_progress: '进行中',
    done: '已完成',
    cancelled: '已取消'
  }[status] || '未开始'
}

function statusTagType(status) {
  return {
    open: '',
    in_progress: 'warning',
    done: 'success',
    cancelled: 'info'
  }[status] || ''
}

function entryTypeLabel(entryType) {
  return entryType === 'event' ? '事件' : '任务'
}

function eventPhaseLabel(phase) {
  return {
    soon: '即将开始',
    today: '今日安排',
    in_window: '进行中',
    awaiting_closure: '待收尾',
    planned: '已安排'
  }[phase] || '已安排'
}

function bucketLabel(bucket) {
  return {
    inbox: '收件箱',
    planned: '计划中',
    someday: '放一放'
  }[bucket] || '计划中'
}

function formatDateTime(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

watch(activeKind, () => {
  if (profileId.value) {
    refreshBoard()
    if (reviewDialogVisible.value) {
      loadReview()
    }
  }
})

watch(
  () => quickForm.entryType,
  (value) => {
    if (value === 'event') {
      quickForm.bucket = 'planned'
    }
  }
)

watch(
  () => taskForm.entryType,
  (value) => {
    if (value === 'event') {
      taskForm.bucket = 'planned'
    }
  }
)

watch(
  () => quickForm.remindAt,
  (value) => {
    if (!value) {
      quickForm.repeatType = 'none'
      quickForm.repeatInterval = 1
      quickForm.repeatUntil = ''
    }
  }
)

watch(
  () => quickForm.repeatType,
  (value) => {
    if (value === 'none') {
      quickForm.repeatInterval = 1
      quickForm.repeatUntil = ''
    }
  }
)

watch(
  () => taskForm.remindAt,
  (value) => {
    if (!value) {
      taskForm.repeatType = 'none'
      taskForm.repeatInterval = 1
      taskForm.repeatUntil = ''
    }
  }
)

watch(
  () => taskForm.repeatType,
  (value) => {
    if (value === 'none') {
      taskForm.repeatInterval = 1
      taskForm.repeatUntil = ''
    }
  }
)

watch(
  () => profile.notify_email,
  (value, oldValue) => {
    if (!quickForm.notifyEmail || quickForm.notifyEmail === oldValue) {
      quickForm.notifyEmail = value || ''
    }
    if (!taskForm.notifyEmail || taskForm.notifyEmail === oldValue) {
      taskForm.notifyEmail = value || ''
    }
  }
)

onMounted(async () => {
  beforeInstallPromptHandler = (event) => {
    event.preventDefault()
    deferredInstallPrompt = event
    installPromptAvailable.value = true
  }
  appInstalledHandler = () => {
    deferredInstallPrompt = null
    installPromptAvailable.value = false
    syncViewportFlags()
  }
  syncViewportFlags()
  window.addEventListener('resize', syncViewportFlags, { passive: true })
  window.addEventListener('beforeinstallprompt', beforeInstallPromptHandler)
  window.addEventListener('appinstalled', appInstalledHandler)
  restoreSession()
  fillTodayDefaults()
  resetTaskForm()
  if (profileId.value) {
    try {
      await loadProfile()
      await refreshBoard()
    } catch (error) {
      clearSession()
    }
  }
})

onBeforeUnmount(() => {
  resetVoiceDraft()
  stopMeetingRecording()
  revokeMeetingLocalAudio()
  window.removeEventListener('resize', syncViewportFlags)
  if (beforeInstallPromptHandler) {
    window.removeEventListener('beforeinstallprompt', beforeInstallPromptHandler)
  }
  if (appInstalledHandler) {
    window.removeEventListener('appinstalled', appInstalledHandler)
  }
})
</script>

<style scoped>
.planner-shell {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  padding: 18px 14px 80px;
  color: #132238;
  transition: background 0.4s ease, color 0.4s ease;
}

.mode-work {
  --planner-accent: #2563eb;
  --planner-accent-soft: rgba(37, 99, 235, 0.14);
  --planner-card: rgba(250, 252, 255, 0.9);
  --planner-strong: #0f3f98;
  background:
    radial-gradient(circle at top left, rgba(59, 130, 246, 0.22), transparent 32%),
    linear-gradient(180deg, #edf4ff 0%, #f7fbff 48%, #f3f7ff 100%);
}

.mode-life {
  --planner-accent: #ec6548;
  --planner-accent-soft: rgba(236, 101, 72, 0.14);
  --planner-card: rgba(255, 252, 248, 0.9);
  --planner-strong: #9c3a24;
  background:
    radial-gradient(circle at top right, rgba(255, 179, 102, 0.22), transparent 28%),
    linear-gradient(180deg, #fff8f1 0%, #fffdf8 45%, #f8fff8 100%);
}

.planner-backdrop {
  position: absolute;
  border-radius: 999px;
  filter: blur(20px);
  pointer-events: none;
  opacity: 0.65;
}

.planner-backdrop-a {
  width: 220px;
  height: 220px;
  top: 24px;
  right: -30px;
  background: rgba(255, 255, 255, 0.55);
  animation: floatOrb 9s ease-in-out infinite;
}

.planner-backdrop-b {
  width: 170px;
  height: 170px;
  bottom: 32px;
  left: -20px;
  background: var(--planner-accent-soft);
  animation: floatOrb 12s ease-in-out infinite reverse;
}

.entry-layout,
.planner-page {
  position: relative;
  z-index: 1;
  max-width: 1180px;
  margin: 0 auto;
}

.entry-layout {
  display: grid;
  gap: 18px;
}

.entry-hero,
.entry-card,
.planner-topbar,
.install-strip,
.mode-switcher,
.hero-card,
.summary-strip,
.quick-panel,
.board-switcher,
.board-panel,
.admin-card,
.comment-item,
.activity-item,
.ai-card,
.advice-card {
  border: 1px solid rgba(255, 255, 255, 0.62);
  background: var(--planner-card);
  backdrop-filter: blur(18px);
  box-shadow: 0 20px 40px rgba(15, 23, 42, 0.08);
}

.entry-hero,
.planner-topbar,
.install-strip,
.hero-card,
.quick-panel,
.board-panel {
  border-radius: 26px;
  padding: 22px;
}

.entry-card,
.board-switcher,
.summary-strip,
.advice-card,
.admin-card,
.comment-item,
.activity-item,
.ai-card {
  border-radius: 22px;
}

.entry-grid,
.hero-grid,
.quick-grid,
.drawer-stack,
.timeline-groups,
.timeline-group,
.comment-timeline,
.activity-timeline,
.admin-list,
.ai-suggestions {
  display: grid;
  gap: 14px;
}

.entry-card-header,
.panel-heading,
.timeline-header,
.planner-topbar,
.install-strip,
.comment-item-top,
.timeline-group-head,
.task-card-top,
.drawer-actions,
.quick-row,
.topbar-actions,
.panel-heading-actions,
.advice-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.entry-card-header > *,
.panel-heading > *,
.timeline-header > *,
.planner-topbar > *,
.install-strip > *,
.comment-item-top > *,
.timeline-group-head > *,
.task-card-top > *,
.drawer-actions > *,
.quick-row > *,
.topbar-actions > *,
.panel-heading-actions > *,
.advice-header > * {
  min-width: 0;
}

.entry-input,
.entry-btn {
  margin-top: 12px;
}

.entry-link {
  margin-top: 10px;
}

.entry-copy,
.subtle,
.hero-card p,
.quick-panel p,
.timeline-header p,
.soft-note,
.task-card p,
.entry-tips span,
.task-meta,
.advice-summary,
.advice-section li {
  color: #5f6f84;
}

.eyebrow {
  margin: 0 0 8px;
  color: var(--planner-accent);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.entry-hero h1,
.planner-topbar h2,
.hero-card h3 {
  margin: 0;
  line-height: 1.15;
}

.entry-tips,
.quick-presets,
.task-tags,
.task-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.entry-tips {
  margin-top: 16px;
}

.entry-tips span,
.soft-note,
.preset-btn,
.summary-chip,
.view-tab {
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.64);
  font-size: 12px;
}

.preset-btn,
.view-tab,
.mode-tab {
  border: none;
  cursor: pointer;
}

.planner-topbar {
  display: flex;
  justify-content: space-between;
}

.install-strip {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  margin-top: 14px;
  padding: 16px 18px;
  border-radius: 22px;
}

.install-strip p {
  margin: 6px 0 0;
}

.install-actions {
  display: flex;
  align-items: center;
}

.topbar-copy {
  max-width: 680px;
}

.topbar-actions {
  flex-wrap: wrap;
}

.mode-switcher {
  position: sticky;
  top: 10px;
  z-index: 10;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  padding: 8px;
  margin: 16px 0;
}

.mode-tab {
  border-radius: 18px;
  background: transparent;
  padding: 14px 16px;
  text-align: left;
  transition: transform 0.25s ease, background 0.25s ease, box-shadow 0.25s ease;
}

.mode-tab.active {
  background: linear-gradient(135deg, var(--planner-accent), var(--planner-strong));
  color: #fff;
  transform: translateY(-2px);
  box-shadow: 0 14px 30px rgba(15, 23, 42, 0.14);
}

.mode-tab-title {
  display: block;
  font-size: 15px;
  font-weight: 700;
}

.mode-tab-sub {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  opacity: 0.84;
}

.hero-grid {
  grid-template-columns: 1.2fr 0.9fr;
}

.hero-main {
  display: grid;
  gap: 18px;
}

.hero-metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.hero-label {
  margin: 0 0 8px;
  font-size: 13px;
  font-weight: 700;
  color: var(--planner-accent);
}

.hero-note {
  display: grid;
  gap: 8px;
  padding: 16px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.68);
}

.hero-note strong {
  font-size: 16px;
  color: #21324c;
}

.hero-note span {
  color: #68798f;
}

.metric-card {
  padding: 16px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.68);
}

.metric-card span {
  display: block;
  font-size: 12px;
  color: #66788d;
}

.metric-card strong {
  display: block;
  margin-top: 10px;
  font-size: 30px;
}

.summary-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-top: 16px;
  padding: 10px;
}

.summary-chip {
  display: grid;
  gap: 4px;
  text-align: center;
}

.summary-chip strong {
  font-size: 20px;
  color: #1c2f49;
}

.focus-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-top: 16px;
}

.focus-card {
  display: grid;
  gap: 14px;
}

.focus-meta,
.focus-secondary {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.focus-meta span {
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.7);
  color: #5f6f84;
  font-size: 12px;
}

.focus-secondary-item {
  border: none;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.72);
  padding: 12px 14px;
  text-align: left;
  cursor: pointer;
  display: grid;
  gap: 6px;
  min-width: 180px;
  flex: 1 1 180px;
}

.focus-secondary-item strong {
  color: #21324c;
}

.focus-secondary-item span {
  color: #6c7d91;
  font-size: 12px;
}

.compact-actions {
  margin-top: 0;
}

.review-strip {
  margin-top: 12px;
}

.quick-panel,
.board-panel {
  margin-top: 16px;
}

.quick-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  width: 100%;
}

.board-switcher {
  position: sticky;
  top: 92px;
  z-index: 9;
  display: flex;
  gap: 10px;
  margin-top: 16px;
  padding: 10px;
  overflow-x: auto;
}

.view-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
  color: #4b617a;
  transition: background 0.2s ease, color 0.2s ease, transform 0.2s ease;
}

.view-tab strong {
  font-size: 13px;
}

.view-tab.active {
  background: linear-gradient(135deg, var(--planner-accent), var(--planner-strong));
  color: #fff;
  transform: translateY(-1px);
}

.timeline-empty {
  padding: 28px 20px;
  text-align: center;
  color: #75859b;
  display: grid;
  gap: 6px;
}

.timeline-group-head h4,
.task-card h5,
.advice-section h4 {
  margin: 0;
}

.timeline-group-head p {
  margin: 4px 0 0;
}

.task-list {
  display: grid;
  gap: 12px;
}

.task-card {
  border-radius: 22px;
  padding: 16px;
  background: rgba(255, 255, 255, 0.82);
  border: 1px solid rgba(255, 255, 255, 0.75);
  transition: transform 0.24s ease, box-shadow 0.24s ease;
}

.task-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 16px 28px rgba(15, 23, 42, 0.09);
}

.task-copy {
  cursor: pointer;
  min-width: 0;
  flex: 1 1 auto;
}

.task-copy h5 {
  font-size: 16px;
  line-height: 1.35;
  word-break: break-word;
}

.task-copy p,
.admin-card p,
.comment-item p,
.activity-item p,
.ai-card p {
  margin: 8px 0 0;
}

.task-copy p {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}

.priority-high {
  box-shadow: inset 4px 0 0 #ef4444;
}

.priority-medium {
  box-shadow: inset 4px 0 0 #f59e0b;
}

.priority-low {
  box-shadow: inset 4px 0 0 #94a3b8;
}

.task-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
  font-size: 12px;
}

.task-actions {
  margin-top: 12px;
}

.task-context {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
  padding: 10px 12px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.7);
  color: #597089;
  font-size: 12px;
  cursor: pointer;
}

.task-context-count {
  color: var(--planner-accent);
  font-weight: 700;
}

.task-context-text {
  flex: 1 1 180px;
  min-width: 0;
}

.detail-comments {
  display: grid;
  gap: 12px;
}

.detail-comments-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.activity-item {
  padding: 14px;
  background: rgba(255, 255, 255, 0.72);
}

.task-actions .el-button,
.panel-heading-actions .el-button,
.quick-actions .el-button {
  min-width: 0;
}

.rolled-tag {
  color: var(--planner-accent);
  font-weight: 600;
}

.advice-card {
  padding: 18px;
}

.advice-summary {
  margin: 14px 0 0;
  line-height: 1.7;
}

.advice-section ul {
  margin: 10px 0 0;
  padding-left: 18px;
}

.advice-section li + li {
  margin-top: 8px;
}

.voice-panel-card {
  display: grid;
  gap: 14px;
  padding: 18px;
  border-radius: 22px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(255, 255, 255, 0.72);
}

.voice-panel-head,
.voice-panel-actions,
.voice-capability {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
}

.voice-capability {
  color: #5f6f84;
  font-size: 12px;
}

.voice-error {
  padding: 10px 12px;
  border-radius: 14px;
  background: rgba(220, 38, 38, 0.08);
  color: #b42318;
  font-size: 13px;
}

.voice-audio {
  width: 100%;
}

@keyframes floatOrb {
  0%,
  100% {
    transform: translate3d(0, 0, 0) scale(1);
  }
  50% {
    transform: translate3d(0, -16px, 0) scale(1.04);
  }
}

.meeting-summary-section {
  margin-top: 16px;
  padding: 12px;
  background: rgba(255,255,255,0.5);
  border-radius: 8px;
}

.meeting-summary-text {
  font-size: 0.9em;
  line-height: 1.6;
  color: var(--text-secondary, #555);
  margin: 4px 0 0;
}

.meeting-action-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  font-size: 0.9em;
}

.meeting-action-tag {
  flex-shrink: 0;
}

.meeting-interim {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 12px;
  background: rgba(255, 87, 87, 0.06);
  border-radius: 8px;
  margin-bottom: 8px;
}

.recording-badge {
  flex-shrink: 0;
  animation: pulse-badge 1.2s ease-in-out infinite;
}

.interim-text {
  font-size: 0.9em;
  color: var(--text-secondary, #666);
  line-height: 1.5;
}

@keyframes pulse-badge {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.voice-audio-wrapper {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.meeting-content-input :deep(.el-textarea__inner) {
  min-height: 320px;
  line-height: 1.7;
}

.timeline-header-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.timeline-header-actions {
  display: flex;
  gap: 8px;
}

@media (min-width: 900px) {
  .entry-layout {
    grid-template-columns: 1.05fr 1fr;
    align-items: start;
  }

  .entry-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .hero-grid {
    grid-template-columns: 1fr;
  }

  .focus-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .planner-shell {
    padding: 12px 8px 72px;
  }

  .planner-topbar,
  .install-strip,
  .panel-heading,
  .timeline-header,
  .quick-row,
  .drawer-actions {
    display: grid;
    grid-template-columns: 1fr;
  }

  .entry-hero,
  .planner-topbar,
  .install-strip,
  .hero-card,
  .quick-panel,
  .board-panel {
    border-radius: 22px;
    padding: 16px;
  }

  .entry-card,
  .board-switcher,
  .summary-strip,
  .advice-card,
  .admin-card,
  .comment-item,
  .ai-card,
  .task-card {
    border-radius: 18px;
  }

  .entry-hero h1,
  .planner-topbar h2,
  .hero-card h3 {
    font-size: 24px;
  }

  .summary-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
    padding: 8px;
  }

  .hero-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .metric-card {
    padding: 12px;
  }

  .metric-card strong,
  .summary-chip strong {
    font-size: 22px;
  }

  .board-switcher {
    top: 78px;
    gap: 8px;
    padding: 8px;
    margin-top: 12px;
  }

  .view-tab {
    padding: 8px 10px;
  }

  .task-card-top,
  .timeline-group-head {
    align-items: flex-start;
  }

  .task-tags {
    gap: 6px;
  }

  .task-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .task-actions .el-button,
  .panel-heading-actions .el-button,
  .quick-actions .el-button {
    width: 100%;
    margin: 0;
  }

  .quick-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .panel-heading-actions,
  .topbar-actions,
  .install-actions,
  .voice-panel-head,
  .voice-panel-actions,
  .voice-capability {
    display: grid;
    grid-template-columns: 1fr;
    justify-content: stretch;
    width: 100%;
  }

  .panel-heading-actions :deep(.el-button),
  .topbar-actions :deep(.el-button),
  .topbar-actions :deep(.el-tag) {
    width: 100%;
  }

  .topbar-actions :deep(.el-tag) {
    justify-content: center;
    margin: 0;
  }

  .timeline-empty {
    padding: 22px 16px;
  }
}

@media (max-width: 560px) {
  .mode-switcher {
    grid-template-columns: 1fr;
    padding: 6px;
  }

  .hero-metrics,
  .summary-strip,
  .quick-actions,
  .task-actions {
    grid-template-columns: 1fr;
  }

  .board-switcher {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    overflow: visible;
  }

  .view-tab {
    justify-content: space-between;
  }

  .task-card {
    padding: 14px;
  }

  .task-card-top {
    display: grid;
    grid-template-columns: 1fr;
  }

  .task-tags {
    justify-content: flex-start;
  }
}

/* ── Meeting Card ── */
.meeting-card {
  background: var(--card-bg, #fff);
  border-radius: 14px;
  padding: 18px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
  border: 1px solid var(--border-color, #eee);
  transition: box-shadow 0.2s;
}
.meeting-card:hover {
  box-shadow: 0 2px 10px rgba(0,0,0,0.08);
}
.meeting-card + .meeting-card {
  margin-top: 12px;
}
.meeting-done {
  border-left: 3px solid var(--el-color-success, #67c23a);
}
.meeting-draft {
  border-left: 3px solid var(--el-color-info, #909399);
}

.meeting-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.meeting-card-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.meeting-card-title {
  font-size: 1.05em;
  font-weight: 600;
  margin: 0 0 8px;
  cursor: pointer;
  color: var(--text-primary, #333);
}
.meeting-card-title:hover {
  color: var(--el-color-primary, #409eff);
}

.meeting-card-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 10px;
  font-size: 0.85em;
  color: var(--text-secondary, #666);
}
.meeting-card-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}
.meeting-card-meta-item .el-icon {
  font-size: 14px;
}

.meeting-card-body {
  margin-bottom: 10px;
  cursor: pointer;
}
.meeting-card-body p {
  margin: 0;
  font-size: 0.88em;
  line-height: 1.6;
  color: var(--text-secondary, #555);
}

.meeting-card-extras {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}
.meeting-card-summary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.85em;
  color: var(--text-secondary, #555);
  cursor: pointer;
  padding: 4px 10px;
  background: rgba(64,158,255,0.06);
  border-radius: 6px;
}
.meeting-card-todos {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.85em;
  color: var(--text-secondary, #555);
  cursor: pointer;
  padding: 4px 10px;
  background: rgba(103,194,58,0.06);
  border-radius: 6px;
}
.meeting-card-todo-badge {
  background: var(--el-color-success, #67c23a);
  color: #fff;
  padding: 1px 8px;
  border-radius: 10px;
  font-size: 0.8em;
  font-weight: 500;
}

.meeting-card-recording {
  margin-bottom: 10px;
}
.meeting-card-audio {
  width: 100%;
  height: 36px;
}

.meeting-card-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

@media (max-width: 768px) {
  .meeting-card {
    padding: 14px;
  }
  .meeting-card-meta {
    flex-direction: column;
    gap: 4px;
  }
  .meeting-card-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .meeting-card-actions .el-button {
    width: 100%;
    margin: 0;
  }
}
</style>
