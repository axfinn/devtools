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
              AI 总结/规划
            </el-button>
            <el-button type="primary" plain @click="aiDialogVisible = true">
              <el-icon><EditPen /></el-icon>
              AI 整理入库
            </el-button>
          </div>
        </div>

        <div class="quick-presets">
          <button class="preset-btn" @click="applyQuickPreset('today')">今天任务</button>
          <button class="preset-btn" @click="applyQuickPreset('inbox')">收件箱</button>
          <button class="preset-btn" @click="applyQuickPreset('event')">事件安排</button>
        </div>

        <div class="quick-grid">
          <el-input
            v-model="quickForm.title"
            placeholder="比如：确认联调结果 / 买水果 / 预约复诊"
            maxlength="80"
            show-word-limit
          />
          <el-input v-model="quickForm.detail" type="textarea" :rows="3" placeholder="补充上下文、边界条件、备注" />
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
            <el-input v-model="quickForm.notifyEmail" placeholder="提醒邮箱，默认用档案邮箱" />
            <div class="quick-actions">
              <el-button :loading="savingQuick" type="primary" @click="createQuickTask">
                保存
              </el-button>
              <el-button @click="fillTodayDefaults">恢复默认</el-button>
            </div>
          </div>
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
                      <span v-if="task.is_rolled_over" class="rolled-tag">已顺延到今天</span>
                      <span v-if="task.last_postpone_reason">顺延原因：{{ task.last_postpone_reason }}</span>
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
                      <span v-if="task.notes">备注：{{ task.notes }}</span>
                    </div>
                    <div class="task-actions">
                      <el-button size="small" type="primary" plain @click="setTaskStatus(task, task.status === 'done' ? 'open' : 'done')">
                        {{ task.status === 'done' ? '重新打开' : '完成' }}
                      </el-button>
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
                <div class="task-actions">
                  <el-button size="small" type="primary" plain @click="moveTaskBucket(task, 'planned')">排入计划</el-button>
                  <el-button size="small" plain @click="openCommentDrawer(task)">评论</el-button>
                  <el-button size="small" plain @click="openTask(task)">编辑</el-button>
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

    <el-drawer v-model="settingsVisible" title="档案设置" size="420px">
      <div class="drawer-stack">
        <el-input v-model="profileForm.name" placeholder="档案名称" />
        <el-input v-model="profileForm.notifyEmail" placeholder="默认提醒邮箱" />
        <el-input-number v-model="profileForm.extendDays" :min="1" :max="3650" placeholder="需要延期时再填写" />
        <div class="drawer-actions">
          <el-button type="primary" :loading="savingProfile" @click="saveProfile">保存档案</el-button>
          <el-button type="danger" plain @click="deleteProfile">删除档案</el-button>
          <el-button @click="logout">退出</el-button>
        </div>
        <div class="soft-note">
          到期时间：{{ profile.expires_at ? formatDateTime(profile.expires_at) : '长期' }}
        </div>
      </div>
    </el-drawer>

    <el-drawer v-model="taskDrawerVisible" :title="taskForm.id ? '编辑事项' : '事项详情'" size="480px">
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
        <div class="drawer-actions">
          <el-button type="primary" :loading="savingTask" @click="saveTask">保存</el-button>
          <el-button plain @click="taskDrawerVisible = false">关闭</el-button>
          <el-button v-if="taskForm.id" type="danger" plain @click="removeTask">删除</el-button>
        </div>
      </div>
    </el-drawer>

    <el-drawer v-model="commentDrawerVisible" title="事项评论" size="420px">
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

    <el-dialog v-model="aiDialogVisible" title="AI 整理事项" width="720px">
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
          <el-button :disabled="aiSuggestions.length === 0" @click="applyAISuggestions">全部写入</el-button>
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

    <el-dialog v-model="adviceDialogVisible" title="AI 总结 / 规划" width="720px">
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

    <el-dialog v-model="adminDialogVisible" title="超级管理员" width="720px">
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

    <el-drawer v-model="adminDetailVisible" title="超管查看档案" size="460px">
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
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Calendar, ChatDotRound, EditPen, MagicStick, Microphone, Refresh, Setting } from '@element-plus/icons-vue'

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
      message: '还没有今天的聚焦建议。'
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
  priority: 'medium',
  notifyEmail: ''
})

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

const creating = ref(false)
const loggingIn = ref(false)
const timelineLoading = ref(false)
const savingQuick = ref(false)
const savingTask = ref(false)
const savingProfile = ref(false)
const savingComment = ref(false)
const loadingAdmin = ref(false)
const savingAdminProfile = ref(false)
const settingsVisible = ref(false)
const taskDrawerVisible = ref(false)
const commentDrawerVisible = ref(false)
const aiDialogVisible = ref(false)
const adviceDialogVisible = ref(false)
const adminDialogVisible = ref(false)
const adminDetailVisible = ref(false)
const parsingAI = ref(false)
const adviceLoading = ref(false)

const comments = ref([])
const aiText = ref('')
const aiSuggestions = ref([])
const adminItems = ref([])
const adminPassword = ref('')
const adminKeyword = ref('')
const currentCommentTask = ref(null)
const adviceMode = ref('plan')
const adviceText = ref('')
const adviceProvider = ref('fallback')
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

const viewOptions = computed(() => [
  { key: 'timeline', label: '时间线', count: board.value.groups.reduce((sum, group) => sum + group.items.length, 0) },
  { key: 'events', label: '事件', count: board.value.event_groups.reduce((sum, group) => sum + group.items.length, 0) },
  { key: 'inbox', label: '收件箱', count: board.value.inbox_items.length },
  { key: 'someday', label: '放一放', count: board.value.someday_items.length },
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
    }
  }[activeView.value]
})

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

function fillTodayDefaults() {
  quickForm.title = ''
  quickForm.detail = ''
  quickForm.entryType = 'task'
  quickForm.bucket = 'planned'
  quickForm.plannedFor = new Date().toISOString().slice(0, 10)
  quickForm.remindAt = ''
  quickForm.priority = 'medium'
  quickForm.notifyEmail = profile.notify_email || ''
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
  taskForm.priority = 'medium'
  taskForm.status = 'open'
  taskForm.notifyEmail = profile.notify_email || ''
  taskForm.cancelReason = ''
  taskForm.postponeReason = ''
}

async function plannerFetch(url, options = {}) {
  const headers = { ...(options.headers || {}) }
  if (password.value) {
    headers['X-Password'] = password.value
  }
  if (creatorKey.value) {
    headers['X-Creator-Key'] = creatorKey.value
  }
  if (options.body) {
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
  localStorage.setItem('planner_creator_key', creatorKey.value)
  localStorage.setItem('planner_active_kind', activeKind.value)
  localStorage.setItem('planner_active_view', activeView.value)
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
  creatorKey.value = localStorage.getItem('planner_creator_key') || ''
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
    creatorKey.value = localStorage.getItem('planner_creator_key') || ''
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
}

async function refreshAll() {
  await loadProfile()
  await refreshBoard()
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
  taskForm.notifyEmail = task.notify_email || ''
  taskForm.cancelReason = task.cancel_reason || ''
  taskForm.postponeReason = task.last_postpone_reason || ''
  taskDrawerVisible.value = true
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

async function loadComments(taskId) {
  const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskId}/comments`)
  const data = await response.json()
  comments.value = data.comments || []
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

function logout() {
  clearSession()
  board.value = createEmptyBoard()
  comments.value = []
  aiSuggestions.value = []
  profile.name = ''
  profile.notify_email = ''
  profile.expires_at = ''
  modeHint.value = '当前是下班或休息时段，默认进入生活模式'
  fillTodayDefaults()
  resetTaskForm()
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
  try {
    for (const item of aiSuggestions.value) {
      await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks`, {
        method: 'POST',
        body: JSON.stringify({
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
        })
      })
    }
    aiDialogVisible.value = false
    aiSuggestions.value = []
    aiText.value = ''
    await refreshBoard()
    ElMessage.success('AI 整理出的事项已写入')
  } catch (error) {
    ElMessage.error(error.message)
  }
}

function openAdviceDialog() {
  adviceDialogVisible.value = true
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

let recognition = null

function openVoice(target) {
  const API = window.SpeechRecognition || window.webkitSpeechRecognition
  if (!API) {
    ElMessage.warning('当前浏览器不支持语音识别')
    return
  }
  recognition = new API()
  recognition.lang = 'zh-CN'
  recognition.interimResults = false
  recognition.onresult = (event) => {
    let text = ''
    for (let i = 0; i < event.results.length; i += 1) {
      text += event.results[i][0].transcript
    }
    if (target === 'quick') {
      quickForm.title = text
    } else {
      aiText.value = [aiText.value, text].filter(Boolean).join('\n')
    }
  }
  recognition.onerror = () => {
    ElMessage.warning('语音识别失败，请重试')
  }
  recognition.start()
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

onMounted(async () => {
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
.mode-switcher,
.hero-card,
.summary-strip,
.quick-panel,
.board-switcher,
.board-panel,
.admin-card,
.comment-item,
.ai-card,
.advice-card {
  border: 1px solid rgba(255, 255, 255, 0.62);
  background: var(--planner-card);
  backdrop-filter: blur(18px);
  box-shadow: 0 20px 40px rgba(15, 23, 42, 0.08);
}

.entry-hero,
.planner-topbar,
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
.admin-list,
.ai-suggestions {
  display: grid;
  gap: 14px;
}

.entry-card-header,
.panel-heading,
.timeline-header,
.planner-topbar,
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

.topbar-copy {
  max-width: 680px;
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
}

.task-copy p,
.admin-card p,
.comment-item p,
.ai-card p {
  margin: 8px 0 0;
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

@keyframes floatOrb {
  0%,
  100% {
    transform: translate3d(0, 0, 0) scale(1);
  }
  50% {
    transform: translate3d(0, -16px, 0) scale(1.04);
  }
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
}

@media (max-width: 768px) {
  .planner-shell {
    padding: 14px 10px 72px;
  }

  .planner-topbar,
  .panel-heading,
  .timeline-header,
  .quick-row,
  .drawer-actions {
    display: grid;
    grid-template-columns: 1fr;
  }

  .summary-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .hero-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .board-switcher {
    top: 78px;
  }

  .task-actions .el-button {
    flex: 1 1 calc(50% - 5px);
    min-width: 0;
  }

  .panel-heading-actions,
  .topbar-actions {
    justify-content: flex-start;
    flex-wrap: wrap;
  }
}
</style>
