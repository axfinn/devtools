<template>
  <div class="planner-shell" :class="`mode-${activeKind}`">
    <div class="planner-backdrop planner-backdrop-a"></div>
    <div class="planner-backdrop planner-backdrop-b"></div>

    <div class="celebration-layer" aria-hidden="true">
      <div
        v-for="burst in celebrationBursts"
        :key="burst.id"
        class="celebration-burst"
      >
        <span v-for="i in 8" :key="i" class="celebration-spark" :style="`--i:${i}`"></span>
      </div>
    </div>

    <transition name="focus-bar">
      <div v-if="focusSession" class="focus-bar" :class="{ 'focus-bar-paused': focusSession.isPaused }">
        <div class="focus-bar-pulse"></div>
        <div class="focus-bar-info">
          <span class="focus-bar-label">🎯 正在专注</span>
          <strong class="focus-bar-title">{{ focusSession.taskTitle }}</strong>
        </div>
        <div class="focus-bar-timer">
          <span class="focus-bar-time">{{ formatFocusDuration(focusElapsedMs) }}</span>
          <span class="focus-bar-state">{{ focusSession.isPaused ? '已暂停' : '进行中' }}</span>
        </div>
        <div class="focus-bar-actions">
          <el-button size="small" :type="focusSession.isPaused ? 'primary' : ''" plain @click="toggleFocusPause">
            {{ focusSession.isPaused ? '继续' : '暂停' }}
          </el-button>
          <el-button size="small" plain @click="stopFocus">停止</el-button>
          <el-button size="small" type="success" plain @click="completeFocusedTask">✓ 完成</el-button>
        </div>
      </div>
    </transition>

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
          <div class="global-search" :class="{ 'global-search-active': searchKeyword }">
            <el-icon class="global-search-icon"><Plus /></el-icon>
            <input
              ref="globalSearchInput"
              v-model="searchKeyword"
              type="text"
              placeholder="搜索所有事项（标题/详情/备注）"
              class="global-search-input"
              @keydown.down.prevent="searchMoveSelection(1)"
              @keydown.up.prevent="searchMoveSelection(-1)"
              @keydown.enter.prevent="searchOpenSelected"
              @keydown.esc="closeSearch"
              @focus="searchOpen = true"
            />
            <span v-if="searchKeyword" class="global-search-clear" @click="clearSearch">×</span>
            <kbd v-else class="global-search-kbd" title="按 / 聚焦搜索">/</kbd>
            <transition name="search-dropdown">
              <div v-if="searchOpen && searchKeyword" class="search-dropdown" @click.stop>
                <div v-if="searchResults.length === 0 && serverSearchResults.length === 0 && !serverSearchLoading" class="search-empty">
                  <span>没有匹配「{{ searchKeyword }}」的事项</span>
                  <el-button size="small" plain @click="searchCreateFromKeyword">快速创建</el-button>
                </div>
                <template v-else>
                  <ul v-if="searchResults.length > 0" class="search-results">
                    <li class="search-section-label">⚡ 任务字段</li>
                    <li
                      v-for="(item, idx) in searchResults.slice(0, 8)"
                      :key="item.id"
                      :class="{ 'search-result-active': searchSelectedIndex === idx }"
                      @mouseenter="searchSelectedIndex = idx"
                      @click="openSearchResult(item)"
                    >
                      <div class="search-result-main">
                        <span class="search-result-title" v-html="highlightMatch(item.title, searchKeyword)"></span>
                        <span class="search-result-meta">
                          <el-tag size="small" effect="plain" :type="priorityTagType(item.priority)">{{ priorityLabel(item.priority) }}</el-tag>
                          <el-tag size="small" effect="plain" :type="statusTagType(item.status)">{{ statusLabel(item.status) }}</el-tag>
                          <span class="search-result-date">{{ item.display_date || item.planned_for }}</span>
                        </span>
                      </div>
                      <p v-if="item.detail" class="search-result-detail" v-html="highlightMatch(truncateText(item.detail, 60), searchKeyword)"></p>
                    </li>
                  </ul>
                  <ul v-if="serverSearchResults.length > 0" class="search-results search-results-server">
                    <li class="search-section-label">
                      🔍 全文(评论/录音)
                      <span v-if="serverSearchLoading" class="search-section-loading">…</span>
                    </li>
                    <li
                      v-for="(hit, idx) in serverSearchResults.slice(0, 8)"
                      :key="`srv-${hit.task_id}-${hit.source_id || hit.match_kind}-${idx}`"
                      class="search-result-server"
                      @click="openServerSearchResult(hit)"
                    >
                      <div class="search-result-main">
                        <span class="search-result-title">{{ hit.task_title || '已删除任务' }}</span>
                        <span class="search-result-meta">
                          <el-tag size="small" effect="plain" :type="matchKindTagType(hit.match_kind)">{{ matchKindLabel(hit.match_kind) }}</el-tag>
                          <span v-if="hit.created_at" class="search-result-date">{{ formatDateTime(hit.created_at) }}</span>
                        </span>
                      </div>
                      <p v-if="hit.snippet" class="search-result-detail" v-html="highlightMatch(hit.snippet, searchKeyword)"></p>
                    </li>
                  </ul>
                </template>
                <div v-if="searchResults.length > 8" class="search-more">
                  还有 {{ searchResults.length - 8 }} 项,按 Enter 打开第 {{ searchSelectedIndex + 1 }} 项
                </div>
              </div>
            </transition>
          </div>
          <el-tag
            effect="dark"
            :type="activeKind === 'work' ? 'primary' : 'success'"
            class="topbar-mode-tag"
            :title="activeKind === 'work' ? '当前工作模式 · 点一下切到生活' : '当前生活模式 · 点一下切到工作'"
            @click="switchKind(activeKind === 'work' ? 'life' : 'work')"
            style="cursor: pointer;"
          >
            {{ activeKind === 'work' ? '💼' : '🏠' }} {{ activeKind === 'work' ? '工作模式' : '生活模式' }} · 切换
          </el-tag>
          <el-button type="primary" class="topbar-quick-add" title="快速新建（⌘/Ctrl+K）" @click="openGlobalQuickAdd">
            <el-icon><Plus /></el-icon>
            <span class="topbar-quick-add-label">快速记录</span>
          </el-button>
          <el-button circle title="档案设置" @click="settingsVisible = true">
            <el-icon><Setting /></el-icon>
          </el-button>
          <el-button circle title="刷新" @click="refreshAll">
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

      <section v-if="crossDayHintVisible && lastSessionMemory" class="crossday-hint-card">
        <div class="crossday-hint-content">
          <span class="crossday-hint-icon">{{ crossDayIcon }}</span>
          <div>
            <strong>{{ crossDayHeadline }}</strong>
            <p>{{ crossDayBody }}</p>
          </div>
        </div>
        <el-button size="small" type="primary" plain @click="dismissCrossDayHint">好的</el-button>
      </section>

      <!-- 阶段 6 减法:完成反思卡 — 在任务刚完成时出现,1 步打标"完成感受" -->
      <transition name="completion-feedback-fade">
        <section v-if="completionFeedbackVisible && completionFeedbackTask" class="completion-feedback-card">
          <div class="completion-feedback-head">
            <span class="completion-feedback-icon">✨</span>
            <div>
              <strong>「{{ completionFeedbackTask.title }}」做完了,感觉怎么样?</strong>
              <p>1 步打个标,留个底,以后回顾能看到自己的节奏。</p>
            </div>
          </div>
          <div class="completion-feedback-chips">
            <button
              v-for="f in COMPLETION_FEELINGS"
              :key="f.key"
              class="completion-feedback-chip"
              :title="f.desc"
              @click="setCompletionFeeling(f.key)"
            >
              <span class="chip-icon">{{ f.icon }}</span>
              <span class="chip-label">{{ f.label }}</span>
            </button>
            <button class="completion-feedback-skip" @click="dismissCompletionFeedback">跳过</button>
          </div>
        </section>
      </transition>

      <section v-if="rolloverInsightVisible" class="rollover-insight-card">
        <div class="rollover-insight-head">
          <div>
            <strong>🔁 顺延回顾</strong>
            <p>这些事已经滑过几次了,要不要重新决定一下?</p>
          </div>
          <el-button text size="small" @click="dismissRolloverInsightToday" title="今天内不再展示">今天先跳过</el-button>
        </div>
        <ul class="rollover-insight-list">
          <li v-for="task in topRolledOverTasks" :key="task.id" class="rollover-insight-row">
            <div class="rollover-insight-copy" @click="openTask(task)">
              <span class="rollover-insight-title">{{ task.title }}</span>
              <span class="rollover-insight-meta">
                已顺延 <strong>{{ task.rollover_count || 0 }}</strong> 次
                <span v-if="task.planned_for" class="rollover-insight-date">· {{ task.planned_for }}</span>
              </span>
            </div>
            <div class="rollover-insight-actions">
              <el-button size="small" type="primary" plain @click="rescheduleRolledOverToday(task)">📅 今天做</el-button>
              <!-- 阶段 5 减法 (iter 46):"🔁 顺延回顾" 1 步"↻ 顺延到明天"
                   补齐顺延决策第三选项(今天做 / 顺延 / 不再做了 = 3 选 1 决策矩阵)
                   场景:用户看到已顺延 3 次的事,"今天还是没空,先挪到明天"
                   原来:必须点 📅 今天做 → 后悔 → 改 task 日期(4-5 步摩擦)
                   现在:主页直接点 ↻ 顺延到明天 → 1 步完成 → 5s 撤销
                   视觉:蓝色调(与 focus.secondary 改天 / focus primary 改明天 色系一致)
                   与 focus.secondary 批量操作三色闭环(✓ 绿 / ↻ 蓝 / ✗ 棕)完全对称
                   位置:今天做(主操作)和不再做了(放弃)中间,平衡视觉重心 -->
              <el-button size="small" plain class="rollover-postpone-btn" :title="`1 步改到 ${tomorrowDateString()}`" @click="postponeRolledOverToTomorrow(task)">↻ 顺延明天</el-button>
              <el-button size="small" plain class="rollover-discard-btn" @click="cancelRolledOver(task)">不再做了</el-button>
            </div>
          </li>
        </ul>
      </section>

      <section class="hero-grid">
      <article class="hero-card hero-metrics">
          <!-- 阶段 5 减法 (iter 32):当 4 个 metric card 全 0 时,合并为 1 个"今天清爽"卡
               减法(4→1)+ 主页信息密度自适应,符合"主页只回答我现在该做什么" -->
          <div v-if="allMetricsZero" class="metric-card metric-card-calm" title="所有维度都清空了,留给自己">
            <span class="metric-card-calm-icon">✨</span>
            <span class="metric-card-calm-label">今天清爽,留给自己</span>
          </div>
          <button
            v-else
            class="metric-card metric-card-clickable"
            :title="openCount > 0 ? '点击查看未完成事项' : '当前没有未完成事项'"
            @click="goOpenTasks"
          >
            <span>未完成</span>
            <strong>{{ openCount }}</strong>
          </button>
          <button
            class="metric-card metric-card-clickable"
            :class="{ 'metric-card-alert': board.inbox_items.length > 0 }"
            :title="board.inbox_items.length > 0 ? '点击分类收件箱' : '收件箱已清空'"
            @click="goInboxOrTriage"
          >
            <span>收件箱</span>
            <strong>{{ board.inbox_items.length }}</strong>
          </button>
          <button
            class="metric-card metric-card-clickable"
            :title="(board.counts.event_open || 0) > 0 ? '点击查看事件' : '没有未来事件'"
            @click="activeView = 'events'"
          >
            <span>事件</span>
            <strong>{{ board.counts.event_open || 0 }}</strong>
          </button>
          <button
            class="metric-card metric-card-clickable"
            :class="{ 'metric-card-warn': (board.counts.rolled_over || 0) > 0 }"
            :title="(board.counts.rolled_over || 0) > 0 ? '点击查看顺延的事项' : '没有顺延事项'"
            @click="goRolledOver"
          >
            <span>顺延</span>
            <strong>{{ board.counts.rolled_over || 0 }}</strong>
          </button>
        </article>
        <article class="hero-card hero-hint">
          <p class="hero-hint-greeting">
            <span class="hero-hint-greeting-icon">{{ timeGreeting.icon }}</span>
            {{ timeGreeting.text }}
          </p>
          <p class="hero-hint-text">{{ board.focus.message }}</p>
          <p class="hero-hint-sub">{{ board.recovery.message }}</p>
          <div v-if="heroHintAction" class="hero-hint-actions">
            <el-button
              :type="heroHintAction.primary ? 'primary' : 'default'"
              :plain="!heroHintAction.primary"
              size="small"
              @click="heroHintAction.onClick"
            >
              <el-icon v-if="heroHintAction.icon"><component :is="heroHintAction.icon" /></el-icon>
              {{ heroHintAction.label }}
            </el-button>
          </div>
        </article>
      </section>

      <section class="summary-strip">
        <!-- 阶段 5 减法 (iter 39):主页 🚨 紧急 chip 1 步 = 时间线 + 紧急 filter
             复用已有 priorityFilter + activeView 系统,与 timeline filter-strip 入口对称
             消除"看到紧急 N 件 → 想看这 N 件必须滚动 / 切 view / 用搜索"的 2 步摩擦
             active 状态:正在查看紧急 filter 时高亮 + cursor pointer,无紧急时不可点 -->
        <div
          class="summary-chip summary-chip-urgent"
          :class="{
            'summary-chip-urgent-active': urgentOpenCount > 0 && priorityFilter === 'urgent' && activeView === 'timeline',
            'summary-chip-clickable': urgentOpenCount > 0
          }"
          :title="urgentOpenCount > 0
            ? (priorityFilter === 'urgent' && activeView === 'timeline'
              ? '正在查看紧急事项 · 切回全部查看时间线'
              : `1 步查看 ${urgentOpenCount} 件紧急事项`)
            : '没有紧急事项,放心推进其他事吧'"
          @click="jumpToUrgent"
        >
          <span>🚨 紧急</span>
          <strong>{{ urgentOpenCount }}</strong>
        </div>
        <div
          v-if="todayTotalCount > 0"
          class="summary-chip summary-chip-progress"
          :class="{
            'summary-chip-progress-full': todayOpenCount === 0,
            'summary-chip-progress-empty': board.recovery.done_today === 0
          }"
          :title="todayOpenCount === 0 ? '今天已收尾,给自己留点空' : `今天完成 ${board.recovery.done_today} / ${todayTotalCount},还差 ${todayOpenCount} 件 · 点一下批量收尾`"
          @click="finishTodayAll"
        >
          <span class="summary-chip-progress-label">{{ todayOpenCount === 0 ? '已收尾' : '今日' }}</span>
          <strong>{{ board.recovery.done_today }}<span class="summary-chip-progress-divider">/</span>{{ todayTotalCount }}</strong>
          <div class="summary-chip-progress-track">
            <div class="summary-chip-progress-fill" :style="{ width: todayProgressPercent + '%' }" />
          </div>
        </div>
        <div
          v-else
          class="summary-chip summary-chip-progress summary-chip-progress-empty"
          title="今天还没有事项,留给自己"
        >
          <span>✨</span>
          <span class="summary-chip-progress-label">今天清爽</span>
        </div>
        <div
          class="summary-chip summary-chip-tomorrow"
          :class="{ 'summary-chip-tomorrow-has': tomorrowCount > 0 }"
          :title="tomorrowCount > 0 ? `明天有 ${tomorrowCount} 件事,提前瞄一眼` : '明天还没有计划,今天的事可以顺延过来'"
          @click="activeView = 'tomorrow'"
        >
          <span>📅 明天</span>
          <strong>{{ tomorrowCount }}</strong>
          <span class="summary-chip-sub">件</span>
        </div>
        <div
          class="summary-chip summary-chip-energy"
          :class="{ 'summary-chip-energy-on': lowEnergyMode }"
          :title="lowEnergyMode ? '🪫 低能量模式已开启:不推高难度,推荐事务性' : '点一下切换到低能量模式,适合疲惫或心情低落时'"
          @click="toggleLowEnergyMode"
        >
          <span>{{ lowEnergyMode ? '🪫' : '🔋' }}</span>
          <span class="summary-chip-energy-label">{{ lowEnergyMode ? '低能量' : '正常' }}</span>
        </div>
        <div
          class="summary-chip summary-chip-streak"
          :class="{
            'summary-chip-streak-active': completionStreak > 0,
            'summary-chip-clickable': completionStreak > 0 || last7DaysTotal > 0,
            'summary-chip-streak-view-active': activeView === 'recent' && (completionStreak > 0 || last7DaysTotal > 0)
          }"
          :title="completionStreak > 0 || last7DaysTotal > 0
            ? (activeView === 'recent'
              ? '正在查看最近完成 · 切回时间线'
              : `1 步查看最近完成的 ${last7DaysTotal} 件事`)
            : '完成一件事开始累积连续天数'"
          @click="jumpToStreak"
        >
          <span>{{ completionStreak > 0 ? '🔥 连续' : '连续' }}</span>
          <strong>{{ completionStreak }}</strong>
          <span class="summary-chip-sub">天</span>
          <div class="week-bars" :title="last7DaysTitle">
            <span
              v-for="d in last7Days"
              :key="d.date"
              class="week-bar"
              :class="{ 'week-bar-today': d.isToday, 'week-bar-active': d.count > 0 }"
              :style="{ '--h': d.barHeight + '%' }"
            />
          </div>
        </div>
        <div
          class="summary-chip summary-chip-focus"
          :class="{ 'summary-chip-focus-on': focusMode }"
          :title="focusMode ? '🎯 专注模式已开启 · 时间线只显示最关键的 N 件' : '点一下开启专注模式,只看到今天最重要的事'"
          @click="toggleFocusMode"
        >
          <span>{{ focusMode ? '🎯' : '🎯' }}</span>
          <span class="summary-chip-focus-label">{{ focusMode ? '专注中' : '专注' }}</span>
        </div>
        <div
          class="summary-chip summary-chip-mode"
          :class="{ 'summary-chip-mode-work': activeKind === 'work', 'summary-chip-mode-life': activeKind === 'life' }"
          :title="activeKind === 'work' ? '当前工作模式 · 点一下切到生活' : '当前生活模式 · 点一下切到工作'"
          @click="switchKind(activeKind === 'work' ? 'life' : 'work')"
        >
          <span class="summary-chip-mode-icon">{{ activeKind === 'work' ? '💼' : '🏠' }}</span>
          <span class="summary-chip-mode-label">{{ activeKind === 'work' ? '工作' : '生活' }}</span>
          <span class="summary-chip-mode-hint">切换</span>
        </div>
      </section>

      <section class="focus-grid">
        <article class="hero-card focus-card" :class="{
          'focus-card-urgent': activePinnedFocus?.primary?.priority === 'urgent' || (!activePinnedFocus && primaryFocusTask?.priority === 'urgent'),
          'focus-card-pinned': activePinnedFocus,
          'focus-card-rest': lowEnergyMode && !activePinnedFocus,
          'focus-card-done': todayAllDone,
          'focus-card-in-progress': (activePinnedFocus?.primary?.status === 'in_progress') || (!activePinnedFocus && primaryFocusTask?.status === 'in_progress')
        }">
          <div class="panel-heading">
            <!-- 阶段 5 减法 (iter 31):focus card 主区域 1 步 = openTask
                 与 timeline 卡片"整卡可点"对称;开始/完成按钮独立在右侧
                 用户心智:看 focus → 想 detail 1 步可达,不用回 timeline 找 -->
            <div class="focus-card-content" :class="{ 'focus-card-content-clickable': !!currentFocusTask }" @click="currentFocusTask && openTask(currentFocusTask)">
              <p class="hero-label">
                <el-icon v-if="activePinnedFocus" class="focus-pin-icon"><StarFilled /></el-icon>
                <el-icon v-else-if="lowEnergyMode" class="focus-pin-icon"><Moon /></el-icon>
                <span v-else-if="todayAllDone" class="focus-pin-icon">🎉</span>
                {{ activePinnedFocus ? '今天我聚焦的' : (lowEnergyMode ? '🪫 今天轻松一点' : (todayAllDone ? '今天已经收尾' : (focusCandidateTasks.length > 0 ? '🎯 挑一件聚焦' : '今天最重要的事'))) }}
              </p>
              <h3 v-if="todayAllDone">做得很好</h3>
              <h3 v-else-if="activePinnedFocus || displayFocusTask">{{ activePinnedFocus ? activePinnedFocus.primary.title : displayFocusTask.title }}</h3>
              <h3 v-else-if="focusCandidateTasks.length > 0">点候选 1 步置入,或者回时间线找 ★</h3>
              <h3 v-else>点任意事项的 ★ 即可置入今天的聚焦</h3>
              <p v-if="todayAllDone">今天做完 {{ board.recovery.done_today }} 件{{ tomorrowCount > 0 ? `,明天还有 ${tomorrowCount} 件在等` : ',先给自己留点空吧' }}。</p>
              <p v-else-if="activePinnedFocus">{{ activePinnedFocus.primary.detail || '你决定今天先推进这一件。' }}</p>
              <p v-else-if="displayFocusTask">{{ displayFocusTask.detail || '聚焦点 = 你自己挑选 1-N 件事,系统不再替你做主。' }}</p>
              <p v-else-if="lowEnergyMode">今天没有事务性的简单事,做点别的吧,或者点掉低能量模式。</p>
              <p v-else-if="focusCandidateTasks.length > 0">下方 {{ focusCandidateTasks.length }} 件候选,点哪个就聚焦哪个,1 步完成。</p>
            </div>
            <div class="focus-card-actions">
              <el-button v-if="activePinnedFocus" type="success" plain @click="setTaskStatus(activePinnedFocus.primary, 'done')">✓ 完成</el-button>
              <el-button v-if="activePinnedFocus" plain @click="togglePin(activePinnedFocus.primary)">取消聚焦</el-button>
              <!-- 阶段 5 减法 (iter 30):displayFocusTask "打开" 2 步 → "▶ 开始" 1 步(进入 in_progress)
                   状态机更清晰:open → 开始(in_progress)→ 完成(done),每步用户主动 1 步选择
                   与 activePinnedFocus 的"✓ 完成"路径对称;想看详情可去 timeline 卡片 -->
              <el-button v-else-if="displayFocusTask && displayFocusTask.status === 'open'" type="primary" @click="setTaskStatus(displayFocusTask, 'in_progress')">▶ 开始</el-button>
              <el-button v-else-if="displayFocusTask && displayFocusTask.status === 'in_progress'" type="success" @click="setTaskStatus(displayFocusTask, 'done')">✓ 完成</el-button>
              <el-button v-else-if="displayFocusTask && displayFocusTask.status !== 'done'" plain @click="openTask(displayFocusTask)">查看</el-button>
              <el-button v-else-if="todayAllDone && tomorrowCount > 0" plain @click="activeView = 'tomorrow'">看明天 →</el-button>
              <!-- 阶段 5 减法 (iter 42):focus 主卡 1 步"⏸ 暂停" — 补齐状态机
                   之前:open → ▶ 开始 → in_progress → ✓ 完成 → done;中途被打断想暂停,只能点 查看 → 找状态改回 open(2+ 步)
                   现在:在 status === 'in_progress' 时显示"⏸ 暂停"按钮,1 步回到 open
                   状态机完整闭环:open ↔ in_progress ↔ done(任意方向 1 步可达)
                   适用:用户开始做 focus 任务,中途被叫走/会议/电话,想"暂停回到 open,稍后再做"
                   复用 setTaskStatus(task, 'open')(已有函数,无需新增)
                   视觉:plain 灰按钮(中性,与 ▶ 开始 蓝色 / ✓ 完成 绿色 形成动作光谱)
                   与 activePinnedFocus / displayFocusTask 两个 focus 来源都生效
                   位置:"✓ 完成"之后,操作光谱:完成(绿)/暂停(灰)/改天(蓝)/记一下(灰)/取消(灰) -->
              <el-button
                v-if="currentFocusTask && currentFocusTask.status === 'in_progress'"
                plain
                size="small"
                :title="'1 步暂停,回到 open 状态,稍后可继续'"
                @click="setTaskStatus(currentFocusTask, 'open')"
              >
                ⏸ 暂停
              </el-button>
              <!-- 阶段 5 减法 (iter 38):focus 主卡 1 步"📅 改明天"
                   与 focus.secondary iter 36 批量顺延对称 — 主卡也能 1 步改天
                   用户看 focus card 决定"这件今天做不动" → 1 步改到明天,5s 内可撤回
                   蓝色 = 冷思考/改期,与 iter 36 同色系(色温区分"完成"绿 vs "改天"蓝)
                   与 activePinnedFocus 的 ✓ 完成 / 取消聚焦、displayFocusTask 的 ▶ 开始 完全独立
                   任何状态(除 done/cancelled)都可推迟,给用户"想换日子"的 1 步出口 -->
              <el-button
                v-if="currentFocusTask && currentFocusTask.status !== 'done' && currentFocusTask.status !== 'cancelled'"
                plain
                size="small"
                class="focus-primary-postpone-btn"
                :title="`1 步改到 ${tomorrowDateString()},5s 内可撤销`"
                @click="postponeFocusPrimaryToTomorrow"
              >
                📅 改明天
              </el-button>
              <!-- 阶段 5 减法 (iter 40):focus card 1 步"📝 记一下"
                   与 event card iter 37 完全对称 — 两个卡都能 1 步记下进展
                   复用 QuickCommentPopover(buttonText/buttonIcon prop iter 37 已就绪)
                   场景:开始 focus 任务后想记一句话("卡在 X 处"/"决定先做 A 再做 B")
                   原来:点标题打开 detail (iter 31) → 找评论 tab → 写 → 保存(3 步)
                   现在:点"📝 记一下" → 1 步写完 Enter 提交,0 步离开主页
                   任何状态(除 done/cancelled)都可记,与 📅 改明天 并列形成"做/改/记"三选一
                   done 状态:不显示(完成 1 步撤回 iter 18 已处理,不需要记一下)
                   cancelled 状态:不显示(已取消,记下来没意义) -->
              <QuickCommentPopover
                v-if="currentFocusTask && currentFocusTask.status !== 'done' && currentFocusTask.status !== 'cancelled'"
                :task="currentFocusTask"
                button-text="📝 记一下"
                button-icon="EditPen"
                @posted="refreshBoard"
                @open-drawer="openCommentDrawer"
              />
              <!-- 阶段 5 减法 (iter 41):focus card 1 步"❌ 取消"
                   与 event card iter 35 取消完全对称 — 两个卡都能 1 步取消
                   复用 CancelReasonPopover(iter 13 1 步 popover,选 chip = 1 步,Enter 提交)
                   场景:用户决定 focus primary 今天不做了("时机不对"/"优先级变了"/"今天状态不适合")
                   原来:回 timeline 找卡 → 点"取消"按钮 → 选原因 (3 步摩擦,还容易分心)
                   现在:主页点"❌ 取消" → 选 chip 1 步完成,5s 撤销 snackbar
                   任何状态(除 done/cancelled)都可取消,与 event card 完全对称
                   视觉:plain 灰按钮(取消是 destructive 但非主要操作,de-emphasize 避免误点)
                   位置:最右,远离"✓ 完成"主操作,符合 destructive 按钮放边缘的 UX 共识 -->
              <CancelReasonPopover
                v-if="currentFocusTask && currentFocusTask.status !== 'done' && currentFocusTask.status !== 'cancelled'"
                button-text="❌ 取消"
                @cancel="(r) => cancelTaskWithReason(currentFocusTask, r)"
              />
            </div>
          </div>
          <div v-if="activePinnedFocus" class="focus-meta">
            <span>{{ priorityLabel(activePinnedFocus.primary.priority) }}</span>
            <span>{{ statusLabel(activePinnedFocus.primary.status) }}</span>
            <span>{{ activePinnedFocus.primary.time_hint || activePinnedFocus.primary.display_label }}</span>
            <span v-if="parseFamilyRound(activePinnedFocus.primary) > 0" class="focus-meta-family" :title="`这是你第 ${parseFamilyRound(activePinnedFocus.primary)} 次做这件重复任务`">🔁 第 {{ parseFamilyRound(activePinnedFocus.primary) }} 次</span>
            <span v-if="activePinnedFocus.extras > 0">+{{ activePinnedFocus.extras }} 项</span>
          </div>
          <div v-else-if="displayFocusTask" class="focus-meta">
            <span>{{ priorityLabel(displayFocusTask.priority) }}</span>
            <span>{{ statusLabel(displayFocusTask.status) }}</span>
            <span>{{ displayFocusTask.time_hint || displayFocusTask.display_label }}</span>
            <span v-if="parseFamilyRound(displayFocusTask) > 0" class="focus-meta-family" :title="`这是你第 ${parseFamilyRound(displayFocusTask)} 次做这件重复任务`">🔁 第 {{ parseFamilyRound(displayFocusTask) }} 次</span>
          </div>
          <div v-if="activePinnedFocus && activePinnedFocus.secondary.length > 0" class="focus-secondary">
            <!-- 阶段 5 减法 (iter 34 + iter 36):focus.secondary 1 步批量操作
                 iter 34: ✓ 收尾 N 个次要 (完成场景)
                 iter 36: ↻ 顺延 N 个到明天 (改天场景)
                 消除"N 个次要逐条点完成/改日期"的 N 步摩擦 -->
            <div v-if="activePinnedFocus.secondary.length >= 2" class="focus-secondary-bulk-row">
              <button
                class="focus-secondary-bulk focus-secondary-bulk-complete"
                :title="`1 步收尾这 ${activePinnedFocus.secondary.length} 个次要 · 5s 内可撤销`"
                @click="completeAllSecondary"
              >
                ✓ 收尾 {{ activePinnedFocus.secondary.length }} 个
              </button>
              <button
                class="focus-secondary-bulk focus-secondary-bulk-postpone"
                :title="`1 步顺延这 ${activePinnedFocus.secondary.length} 个到明天 · 5s 内可撤销`"
                @click="postponeAllSecondaryToTomorrow"
              >
                ↻ 顺延到明天
              </button>
              <!-- 阶段 5 减法 (iter 44):focus.secondary "✗ 全部取消" 1 步批量
                   与 iter 34 ✓ 收尾 + iter 36 ↻ 顺延 形成状态机 1 步批量完整闭环
                   覆盖"今天这堆都不做了/条件变化"场景,无需逐个取消
                   颜色用 cancel 棕色调,与收尾绿/顺延蓝 区分
                   5s 内可撤销(snapshot + onUndo 兜底) -->
              <button
                class="focus-secondary-bulk focus-secondary-bulk-cancel"
                :title="`1 步取消这 ${activePinnedFocus.secondary.length} 个次要 · 5s 内可撤销`"
                @click="cancelAllSecondary"
              >
                ✗ 取消 {{ activePinnedFocus.secondary.length }} 个
              </button>
            </div>
            <div
              v-for="task in activePinnedFocus.secondary"
              :key="task.id"
              class="focus-secondary-item-row"
            >
              <button
                class="focus-secondary-item"
                :title="`点击直接完成「${task.title}」· 5s 内可撤销`"
                @click="setTaskStatus(task, 'done')"
              >
                <strong>{{ task.title }}</strong>
                <span>{{ task.time_hint || task.display_label }}</span>
              </button>
              <!-- 阶段 5 减法 (iter 43):focus.secondary 单件 1 步"→ 明天"
                   与 iter 36 批量顺延形成单/批完整对称 — 改天维度 100% 覆盖
                   场景:用户想改 1 个次要到明天,但其他次要不变
                   原来:回 timeline 找该卡 → 点"→ 明天"按钮(3 步摩擦,还容易分心)
                   现在:主页直接点 secondary 卡右侧"→ 明天"→ 1 步完成
                   复用 postponeToTomorrow(已有函数,0 新增)
                   视觉:蓝色调与 iter 36/38 顺延色系一致(冷思考/改天)
                   位置:卡右侧,与点击完成区域(左)分离,@click.stop 防止冒泡 -->
              <button
                class="focus-secondary-item-postpone"
                :title="`1 步改到 ${tomorrowDateString()}`"
                @click.stop="postponeToTomorrow(task)"
              >→ 明天</button>
              <!-- 阶段 5 减法 (iter 44):focus.secondary 单件 1 步"✗ 取消" inline
                   与 iter 41 focus 主卡取消形成单/批完整对称
                   场景:用户想取消 1 个次要(条件变化/不再需要),但其他次要不变
                   原来:回 timeline 找该卡 → 点 ⋯ → 取消(3 步摩擦,还容易分心去看别的卡)
                   现在:主页直接点 secondary 卡右侧 ✗ → 选原因 chip → 1 步完成
                   复用 CancelReasonPopover(iter 13 + iter 41),仅加 iconOnly prop
                   视觉:与"→ 明天"同色系但用 cancel-btn 棕色调区分(取消/不再做)
                   位置:卡最右,与"→ 明天"并排(改天/取消 是用户的两个主要决策)
                   @click.stop 防止冒泡到点击完成区 -->
              <CancelReasonPopover
                icon-only
                @cancel="(r) => cancelTaskWithReason(task, r)"
              />
            </div>
          </div>
          <div v-else-if="!activePinnedFocus && secondaryFocusTasks.length > 0" class="focus-secondary">
            <!-- 阶段 5 减法 (iter 34 + iter 36):system-suggested focus.secondary 1 步批量 -->
            <div v-if="secondaryFocusTasks.length >= 2" class="focus-secondary-bulk-row">
              <button
                class="focus-secondary-bulk focus-secondary-bulk-complete"
                :title="`1 步收尾这 ${secondaryFocusTasks.length} 个次要 · 5s 内可撤销`"
                @click="completeAllSecondary"
              >
                ✓ 收尾 {{ secondaryFocusTasks.length }} 个
              </button>
              <button
                class="focus-secondary-bulk focus-secondary-bulk-postpone"
                :title="`1 步顺延这 ${secondaryFocusTasks.length} 个到明天 · 5s 内可撤销`"
                @click="postponeAllSecondaryToTomorrow"
              >
                ↻ 顺延到明天
              </button>
              <!-- 阶段 5 减法 (iter 44):同上(system-suggested focus.secondary 也加 1 步批量取消) -->
              <button
                class="focus-secondary-bulk focus-secondary-bulk-cancel"
                :title="`1 步取消这 ${secondaryFocusTasks.length} 个次要 · 5s 内可撤销`"
                @click="cancelAllSecondary"
              >
                ✗ 取消 {{ secondaryFocusTasks.length }} 个
              </button>
            </div>
            <div
              v-for="task in secondaryFocusTasks"
              :key="task.id"
              class="focus-secondary-item-row"
            >
              <button
                class="focus-secondary-item"
                :title="`点击直接完成「${task.title}」· 5s 内可撤销`"
                @click="setTaskStatus(task, 'done')"
              >
                <strong>{{ task.title }}</strong>
                <span>{{ task.time_hint || task.display_label }}</span>
              </button>
              <!-- 阶段 5 减法 (iter 43):同上(对 system-suggested focus.secondary 也加 1 步改天) -->
              <button
                class="focus-secondary-item-postpone"
                :title="`1 步改到 ${tomorrowDateString()}`"
                @click.stop="postponeToTomorrow(task)"
              >→ 明天</button>
              <!-- 阶段 5 减法 (iter 44):同上(system-suggested focus.secondary 也加 1 步取消) -->
              <CancelReasonPopover
                icon-only
                @cancel="(r) => cancelTaskWithReason(task, r)"
              />
            </div>
          </div>
          <!-- 阶段 5 减法 (iter 33):focus card 空状态 1 步置入候选
               消除"我得回 timeline 找 ★" 的导航摩擦,用户看 focus card 就能 1 步 pin -->
          <div v-else-if="focusCandidateTasks.length > 0" class="focus-candidates">
            <div
              v-for="task in focusCandidateTasks"
              :key="task.id"
              class="focus-candidate-row"
            >
              <button
                class="focus-candidate-chip"
                :class="`priority-${task.priority || 'medium'}`"
                :title="`点击置入今天聚焦「${task.title}」`"
                @click="togglePin(task)"
              >
                <span class="focus-candidate-pin">📌</span>
                <span class="focus-candidate-title">{{ task.title }}</span>
                <span class="focus-candidate-source">{{ task.bucket === 'inbox' ? '收件箱' : (task.time_hint || task.display_label || '今天') }}</span>
              </button>
              <!-- 阶段 5 减法 (iter 47):focus candidate chip "1 步完成" inline
                   与 focus.secondary 单件 → 明天 (iter 43) / ✗ (iter 44) 行内按钮模式完全对称
                   场景:用户看到候选 1 件(收件箱/简单任务),实际不需要置入聚焦,直接做掉就行
                   原来:点 chip → pin 置入 focus → 主页出现该 focus → 点"✓ 完成" = 2 步摩擦
                   现在:chip 行内直接点 ✓ → 1 步完成 → 5s 撤销
                   复用 setTaskStatus(task, 'done')(已有函数,0 新增)
                   视觉:绿色调(暖色/完成)与 focus.secondary 收尾色系一致
                   @click.stop 防止冒泡到 chip 主点击(pin 置入)
                   位置:chip 行内最右,与主点击区域(左)分离 -->
              <button
                class="focus-candidate-complete"
                :title="`1 步完成「${task.title}」· 5s 内可撤销`"
                @click.stop="setTaskStatus(task, 'done')"
              >✓</button>
            </div>
          </div>
        </article>

        <article class="hero-card focus-card event-card-hero">
          <div class="panel-heading">
            <!-- 阶段 5 减法 (iter 35):事件卡 body 1 步 = openTask
                 与 focus card iter 31 完全对称;状态按钮(开始准备/已完成/取消)独立在下方
                 用户心智:看事件 → 想 detail 1 步可达,不用专门找"查看"按钮 -->
            <div
              class="event-card-content"
              :class="{ 'event-card-content-clickable': !!nextEventTask }"
              @click="nextEventTask && openTask(nextEventTask)"
            >
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
            <!-- 阶段 5 减法 (iter 45):事件卡 1 步"⏸ 暂停" — 与 focus primary iter 42 完全对称
                 事件卡是主页第二显眼位置(下一场事件),状态机应与 focus primary 一致
                 场景:用户开始准备 3 点的会议,中途被打断(老板喊走/电话/急事),想"暂停回到待办"
                 原来:点 ⋯ → 标记为待办(2 步摩擦,且在 timeline 才有 ⋯)
                 现在:主页事件卡直接点"⏸ 暂停"→ 1 步回到 open 状态,5s 撤销兜底
                 复用 setTaskStatus(task, 'open')(已有函数,0 新增)
                 视觉:plain 灰按钮(中性,与 ▶ 开始准备 灰/ ✓ 已完成 蓝 形成动作光谱)
                 仅在 in_progress 时显示(状态机闭环:open ↔ in_progress ↔ done) -->
            <el-button
              v-if="nextEventTask.status === 'in_progress'"
              size="small"
              plain
              :title="'1 步暂停,回到 open 状态,稍后可继续准备'"
              @click="setTaskStatus(nextEventTask, 'open')"
            >
              ⏸ 暂停
            </el-button>
            <!-- 阶段 5 减法 (iter 45):事件卡 1 步"📅 改明天" — 与 focus primary iter 38 完全对称
                 事件卡是主页第二显眼位置,改天能力必须与 focus primary 一致
                 场景:用户看到 3 点的会要推迟(临时有事/改时间) → 必须点查看 → 找日期 → 改 → 保存(4 步)
                 现在:主页事件卡直接点"📅 改明天"→ 1 步完成 → 5s 撤销
                 复用 postponeToTomorrow(已有函数,0 新增)
                 视觉:蓝色调(冷思考/改期)与 focus primary 改天色系一致(色温区分"完成"绿/蓝 vs "改天"蓝)
                 任何状态(除 done/cancelled)都可推迟,给用户"想换日子"的 1 步出口 -->
            <el-button
              v-if="nextEventTask.status !== 'done' && nextEventTask.status !== 'cancelled'"
              size="small"
              plain
              class="focus-primary-postpone-btn"
              :title="`1 步改到 ${tomorrowDateString()},5s 内可撤销`"
              @click="postponeToTomorrow(nextEventTask)"
            >
              📅 改明天
            </el-button>
            <!-- 阶段 5 减法 (iter 37):事件卡"📝 记一下" 1 步顺势记录
                 复用 QuickCommentPopover(iter 22 的 1 步 popover),不离开主页 1 步写下会议/出发/约会的准备情况
                 与 focus card iter 31 标题 1 步 / iter 35 事件卡 body 1 步 完全对称:主页可"看"可"做"
                 按钮文案换成「📝 记一下」比"评论"更顺势(事件轻量场景,不是任务工作流) -->
            <QuickCommentPopover
              :task="nextEventTask"
              button-text="📝 记一下"
              button-icon="EditPen"
              @posted="refreshBoard"
              @open-drawer="openCommentDrawer"
            />
            <CancelReasonPopover
              button-text="取消"
              @cancel="(r) => cancelTaskWithReason(nextEventTask, r)"
            />
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
          <button class="preset-btn preset-btn-strong" @click="applyQuickPreset('tomorrow')">明天做</button>
          <button class="preset-btn" @click="applyQuickPreset('inbox')">收件箱</button>
          <button class="preset-btn" @click="applyQuickPreset('event')">事件安排</button>
        </div>

        <div v-if="quickTemplates.length > 0" class="quick-templates">
          <span class="quick-templates-label">📋 常用 · 点击 1 步创建</span>
          <button
            v-for="t in quickTemplates"
            :key="quickTemplateSignature(t)"
            class="quick-template-chip"
            :title="`点击直接创建「${t.title}」· 点 ✎ 编辑 · × 删除`"
            @click="applyQuickTemplateDirect(t)"
          >
            <span class="quick-template-title">{{ truncateText(t.title, 14) }}</span>
            <span class="quick-template-priority" :class="`qp-${t.priority}`">{{ priorityLabel(t.priority) }}</span>
            <span class="quick-template-edit" @click.stop="applyQuickTemplate(t)" title="填表编辑(自定义日期/描述)">✎</span>
            <span class="quick-template-remove" @click.stop="removeQuickTemplate(t)" title="删除模板">×</span>
          </button>
        </div>

        <div class="quick-grid quick-grid-core">
          <el-input
            v-model="quickForm.title"
            placeholder="比如：明天下午3点 给客户发合同 紧急 / 改天再约牙医 / 周末买牛奶  多个用 / 或换行分隔"
            maxlength="500"
            show-word-limit
            :ref="bindQuickAddInput"
            @input="onQuickFormTitleInput"
          />
          <el-input v-model="quickForm.detail" type="textarea" :rows="2" placeholder="补充说明（可选）" @input="onQuickFormTitleInput" />
          <div v-if="quickFormHints" class="quick-form-hints" :title="quickFormHints.hint">
            <el-icon><MagicStick /></el-icon>
            <span>已识别: <strong>{{ quickFormHints.hint }}</strong> · 保存时自动归类</span>
          </div>
          <!-- 阶段 5 减法 (iter 24):多任务预览 — 让 / 切分在保存前可见,避免误拆分 -->
          <div v-if="quickFormBatchCount >= 2" class="quick-batch-preview" :title="`将一次性记录 ${quickFormBatchCount} 条`">
            <el-icon><Files /></el-icon>
            <span class="quick-batch-label">将记录 {{ quickFormBatchCount }} 条:</span>
            <ol class="quick-batch-list">
              <li v-for="(t, i) in quickFormBatchTitles.slice(0, 5)" :key="i">
                <span class="quick-batch-num">{{ i + 1 }}.</span>
                <span class="quick-batch-title">{{ truncateText(t, 24) }}</span>
                <span v-if="extractHintsFromText(t).hint" class="quick-batch-hint">→ {{ extractHintsFromText(t).hint }}</span>
              </li>
              <li v-if="quickFormBatchCount > 5" class="quick-batch-more">… 还有 {{ quickFormBatchCount - 5 }} 条</li>
            </ol>
          </div>
          <div class="quick-row quick-repeat-row">
            <span class="quick-row-label">重复</span>
            <div class="quick-repeat-chips">
              <button
                v-for="opt in repeatQuickOptions"
                :key="opt.value"
                class="repeat-chip"
                :class="{ 'repeat-chip-active': quickForm.repeatType === opt.value }"
                :disabled="opt.value !== 'none' && !quickForm.remindAt"
                @click="applyQuickRepeat(opt.value)"
              >
                {{ opt.label }}
              </button>
            </div>
            <span v-if="quickForm.repeatType !== 'none' && !quickForm.remindAt" class="quick-repeat-hint">
              <el-icon><InfoFilled /></el-icon>
              重复需要先设置提醒时间
            </span>
          </div>
          <div class="quick-row">
            <div class="quick-actions quick-actions-primary">
              <el-button :loading="savingQuick" type="primary" @click="createQuickTask">
                {{ quickFormBatchLabel }}
              </el-button>
              <el-button @click="fillTodayDefaults">恢复默认</el-button>
              <el-button plain :disabled="!quickForm.title.trim()" @click="saveCurrentAsTemplate" title="把当前标题/优先级/分类保存为常用模板">
                <el-icon><Star /></el-icon>
                存为模板
              </el-button>
              <el-popover
                v-model:visible="routinePopoverVisible"
                placement="bottom-end"
                :width="320"
                trigger="click"
                popper-class="routine-popover"
              >
                <template #reference>
                  <el-button plain :title="routines.length ? `${routines.length} 个套路可用` : '管理多任务套路'">
                    <el-icon><Files /></el-icon>
                    套路 · {{ routines.length }}
                  </el-button>
                </template>
                <div class="routine-popover-content">
                  <div v-if="routines.length === 0" class="routine-empty">
                    <p>还没有套路。</p>
                    <p class="routine-empty-hint">套路 = 一次创建多个相关任务,例如「周一上午」「出差前夜」。</p>
                  </div>
                  <ul v-else class="routine-list">
                    <li v-for="r in routines" :key="r.id" class="routine-item">
                      <div class="routine-item-main">
                        <strong class="routine-item-name">{{ r.name }}</strong>
                        <span class="routine-item-count">{{ r.items.length }} 项</span>
                      </div>
                      <div class="routine-item-actions">
                        <el-button size="small" type="primary" plain @click="applyRoutine(r)">▶ 应用</el-button>
                        <el-button size="small" plain @click="openEditRoutineDialog(r)">编辑</el-button>
                        <el-button size="small" plain @click="deleteRoutine(r)">删除</el-button>
                      </div>
                    </li>
                  </ul>
                  <div class="routine-popover-footer">
                    <el-button size="small" type="primary" plain @click="openNewRoutineDialog">+ 新建套路</el-button>
                  </div>
                </div>
              </el-popover>
              <el-button plain @click="quickAdvancedVisible = !quickAdvancedVisible">
                <el-icon><Setting /></el-icon>
                {{ quickAdvancedVisible ? '收起高级' : '高级设置' }}
              </el-button>
            </div>
          </div>
        </div>

        <div v-if="quickAdvancedVisible" class="quick-grid quick-grid-advanced">
          <div class="quick-row">
            <el-select v-model="quickForm.entryType" placeholder="任务/事件">
              <el-option label="任务" value="task" />
              <el-option label="事件" value="event" />
            </el-select>
            <el-select v-model="quickForm.bucket" placeholder="放到哪里" :disabled="quickForm.entryType === 'event'">
              <el-option label="计划中" value="planned" />
              <el-option label="收件箱" value="inbox" />
              <el-option label="放一放" value="someday" />
            </el-select>
            <el-select v-model="quickForm.priority" placeholder="优先级">
              <el-option label="🚨 紧急" value="urgent" />
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
          <div class="quick-row quick-row-natural-date">
            <el-input
              v-model="quickNaturalDate"
              :placeholder="quickNaturalDateHint"
              clearable
              @keydown.enter.prevent="applyQuickNaturalDate"
              @input="onQuickNaturalDateInput"
            >
              <template #prefix><el-icon><MagicStick /></el-icon></template>
            </el-input>
            <el-button :disabled="!quickNaturalDate.trim()" plain @click="applyQuickNaturalDate">识别</el-button>
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
          <div class="quick-row">
            <el-input v-model="quickForm.intent" placeholder="意图：为什么要做？（可选）" maxlength="80" />
            <el-select v-model="quickForm.energy_level" placeholder="精力类型（可选）" clearable>
              <el-option label="需专注" value="deep" />
              <el-option label="事务性" value="shallow" />
              <el-option label="需外出" value="errand" />
              <el-option label="创意型" value="creative" />
            </el-select>
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

        <div v-if="activeView === 'timeline'" class="timeline-filter-strip">
          <button
            v-for="opt in priorityFilterOptions"
            :key="opt.key"
            class="filter-chip"
            :class="{ active: priorityFilter === opt.key, 'filter-chip-urgent': opt.key === 'urgent' && priorityFilter === 'urgent' }"
            @click="setPriorityFilter(opt.key)"
          >
            <span>{{ opt.label }}</span>
            <strong>{{ opt.count }}</strong>
          </button>
        </div>

        <div v-if="timelineLoading" class="timeline-empty">正在加载看板...</div>

        <template v-else>
          <div v-if="activeView === 'timeline'">
            <div v-if="(focusMode ? focusedTimelineGroups.length : displayedTimelineGroups.length) === 0" class="timeline-empty">
              <strong v-if="focusMode && displayedTimelineGroups.length > 0">🎯 专注中 · 当前时间线没有关键事项</strong>
              <strong v-else-if="priorityFilter === 'urgent' && board.groups.length > 0">没有紧急事项,放心推进其他事吧 ✨</strong>
              <strong v-else-if="timelineFolded && foldedTimelineItemCount > 0">{{ TIMELINE_FOLD_DAYS }} 天内暂无事项,更远的事已折叠</strong>
              <strong v-else>还没有时间线事项</strong>
              <span v-if="focusMode && displayedTimelineGroups.length > 0">只有非紧急的 {{ focusHiddenItemCount }} 件被收起着,点解除专注展开看。</span>
              <span v-else-if="priorityFilter === 'urgent' && board.groups.length > 0">所有紧急事项都已收尾。切换到「全部」看完整时间线。</span>
              <span v-else-if="timelineFolded && foldedTimelineItemCount > 0">先聚焦这一周,展开可看更远。</span>
              <span v-else>先把今天要推进的一件事写下来。</span>
            </div>
            <div v-else class="timeline-groups">
              <!-- 减法:7 天以外的折叠到一行,1 步展开 -->
              <div v-if="timelineFolded && foldedTimelineItemCount > 0" class="timeline-fold-banner">
                <span>还有 <strong>{{ foldedTimelineItemCount }}</strong> 件事排在 {{ TIMELINE_FOLD_DAYS }} 天后</span>
                <el-button size="small" plain @click="toggleTimelineFold">展开看看</el-button>
              </div>
              <!-- 专注模式:非关键件收起,显示"还有 N 件被收起"banner -->
              <div v-if="focusMode && focusHiddenItemCount > 0" class="timeline-fold-banner timeline-fold-banner-focus">
                <span>🎯 专注中 · 还有 <strong>{{ focusHiddenItemCount }}</strong> 件非紧急被收起</span>
                <el-button size="small" plain @click="toggleFocusMode">解除专注</el-button>
              </div>
              <div v-for="group in (focusMode ? focusedTimelineGroups : displayedTimelineGroups)" :key="group.date" class="timeline-group">
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
                    :data-task-id="task.id"
                    :class="[`status-${task.status}`, `priority-${task.priority}`, { 'task-card-celebrate': celebratedTaskIds.has(task.id), 'task-card-overdue': isTaskOverdue(task), 'task-card-bulk-selected': isSelected(task.id) }]"
                  >
                    <div class="task-card-top">
                      <div class="task-copy" @click="openTask(task)">
                        <el-checkbox
                          class="task-card-select"
                          :model-value="isSelected(task.id)"
                          @click.stop
                          @change="toggleSelected(task.id, $event)"
                        />
                        <h5>{{ task.title }}</h5>
                        <p>{{ task.detail || '点击补充详情、提醒、备注和评论' }}</p>
                      </div>
                      <div class="task-tags">
                        <button
                          class="pin-toggle"
                          :class="{ 'pin-toggle-active': isPinned(task.id) }"
                          :title="isPinned(task.id) ? '取消聚焦' : '置入今天聚焦'"
                          @click.stop="togglePin(task)"
                        >
                          <el-icon><component :is="isPinned(task.id) ? 'StarFilled' : 'Star'" /></el-icon>
                        </button>
                        <el-tag size="small">{{ entryTypeLabel(task.entry_type) }}</el-tag>
                        <el-dropdown trigger="click" size="small" @command="(b) => moveTaskBucket(task, b)">
                          <el-tag size="small" :type="bucketTagType(task.bucket)" class="clickable-priority">
                            {{ bucketLabel(task.bucket) }}
                          </el-tag>
                          <template #dropdown>
                            <el-dropdown-menu>
                              <el-dropdown-item command="planned">📅 计划中</el-dropdown-item>
                              <el-dropdown-item command="inbox">📥 收件箱</el-dropdown-item>
                              <el-dropdown-item command="someday">💤 放一放</el-dropdown-item>
                            </el-dropdown-menu>
                          </template>
                        </el-dropdown>
                        <el-dropdown trigger="click" size="small" @command="(p) => updateTask(task, { priority: p })">
                          <el-tag size="small" :type="priorityTagType(task.priority)" class="clickable-priority">
                            {{ priorityLabel(task.priority) }}
                          </el-tag>
                          <template #dropdown>
                            <el-dropdown-menu>
                              <el-dropdown-item command="urgent">🔴 紧急</el-dropdown-item>
                              <el-dropdown-item command="high">🟠 高</el-dropdown-item>
                              <el-dropdown-item command="medium">🔵 中</el-dropdown-item>
                              <el-dropdown-item command="low">⚪ 低</el-dropdown-item>
                            </el-dropdown-menu>
                          </template>
                        </el-dropdown>
                        <el-dropdown trigger="click" size="small" @command="(e) => updateTask(task, { energy_level: e || '' })">
                          <el-tag v-if="task.energy_level" size="small" :type="energyTagType(task.energy_level)" class="clickable-priority">
                            {{ energyIcon(task.energy_level) }} {{ energyLabel(task.energy_level) }}
                          </el-tag>
                          <el-tag v-else size="small" class="clickable-priority energy-hint-tag" title="点一下标精力类型">
                            🪫 标精力
                          </el-tag>
                          <template #dropdown>
                            <el-dropdown-menu>
                              <el-dropdown-item command="deep">🧠 需专注</el-dropdown-item>
                              <el-dropdown-item command="shallow">📋 事务性</el-dropdown-item>
                              <el-dropdown-item command="errand">🚶 需外出</el-dropdown-item>
                              <el-dropdown-item command="creative">💡 创意型</el-dropdown-item>
                              <el-dropdown-item v-if="task.energy_level" command="" divided>清空</el-dropdown-item>
                            </el-dropdown-menu>
                          </template>
                        </el-dropdown>
                        <span v-if="task.status !== 'cancelled'" class="task-status-cycle" :title="statusCycleHint(task)" @click.stop="cycleTaskStatus(task)">
                          <el-tag size="small" :type="statusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
                          <el-icon class="status-cycle-hint"><Refresh /></el-icon>
                        </span>
                        <el-tag v-else size="small" :type="statusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
                        <!-- 阶段 5 减法 (iter 20):重复任务接力标识 — 让"我正在做这件事第几次"可见 -->
                        <el-tag
                          v-if="parseFamilyRound(task) > 0"
                          size="small"
                          class="family-round-chip"
                          :title="`这是你第 ${parseFamilyRound(task)} 次做这件重复任务`"
                        >
                          🔁 第 {{ parseFamilyRound(task) }} 次
                        </el-tag>
                      </div>
                    </div>
                    <div class="task-meta">
                      <TaskDateChip
                        :task="task"
                        @update="(date, label) => postponeTo(task, date, label)"
                        @move-to-inbox="moveTaskBucket(task, 'inbox')"
                      />
                      <span v-if="task.remind_at">提醒 {{ formatDateTime(task.remind_at) }}</span>
                      <span v-if="repeatSummary(task)">重复 {{ repeatSummary(task) }}</span>
                      <span v-if="(task.completion_count || 0) >= 2" class="task-completion-badge" :title="`这件事你已经完成过 ${task.completion_count} 次`">已完成 {{ task.completion_count }} 次</span>
                      <HabitSeedPopover
                        v-if="habitSeedEligible(task)"
                        :task="task"
                        @set="(payload) => applyHabitSeed(task, payload)"
                      />
                      <span v-if="taskAgeDays(task) >= 3" class="task-age-badge" :title="`${taskAgeDays(task)} 天前创建,需要的话可以推迟或取消`">{{ taskAgeLabel(task) }}</span>
                      <span v-if="isTaskOverdue(task)" class="task-overdue-badge">
                        已超期 {{ taskOverdueDays(task) }} 天
                        <button class="overdue-snooze-btn" @click.stop="postponeToTomorrow(task)" title="推迟到明天">→ 明天</button>
                        <button class="overdue-snooze-btn" @click.stop="postponeToNextMonday(task)" title="推迟到下周一">→ 下周一</button>
                      </span>
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
                      <!-- 阶段 5 减法 (iter 49):timeline (planned) 1 步"⏸ 暂停" — 补齐状态机
                           之前:in_progress 任务只能 ⋯ 菜单 = 2 步摩擦(且 ⋯ 没这选项,需点"标记为待办"路径)
                           现在:在 status === 'in_progress' 时直接显示 ⏸ 按钮,1 步回到 open
                           复用 setTaskStatus(task, 'open')(已有函数,0 新增)
                           与 focus primary iter 42 / 主页事件卡 iter 45 / 事件视图 iter 48 完全对称
                           状态机完整闭环:open ↔ in_progress ↔ done(任意方向 1 步可达) -->
                      <el-button
                        v-if="task.status === 'in_progress'"
                        size="small"
                        plain
                        :title="'1 步暂停,回到 open 状态,稍后可继续'"
                        @click="setTaskStatus(task, 'open')"
                      >⏸ 暂停</el-button>
                      <el-button size="small" plain class="quick-postpone-btn" :title="`推迟到 ${tomorrowDateString()}`" @click="postponeToTomorrow(task)">→ 明天</el-button>
                      <el-button size="small" plain @click="postponeTask(task)">顺延…</el-button>
                      <el-button size="small" plain @click="openCommentDrawer(task)">
                        <el-icon><ChatDotRound /></el-icon>
                        评论
                      </el-button>
                      <el-button size="small" plain @click="importCalendar(task)">
                        <el-icon><Calendar /></el-icon>
                        日历
                      </el-button>
                      <el-button size="small" plain @click="openTask(task)">编辑</el-button>
                      <!-- 取消 1 步化(替代 ElMessageBox 模态):点按钮 → popover → 1 键 chip -->
                      <CancelReasonPopover
                        v-if="task.status !== 'done' && task.status !== 'cancelled'"
                        button-text="取消"
                        @cancel="(r) => cancelTaskWithReason(task, r)"
                      />
                      <!-- 减法:低频操作(专注/复制/进行中)收进 ⋯ 菜单 -->
                      <el-dropdown trigger="click" size="small" @command="(cmd) => handleTaskMore(task, cmd)">
                        <el-button size="small" plain class="task-more-btn" :title="'更多操作'">
                          <el-icon><MoreFilled /></el-icon>
                        </el-button>
                        <template #dropdown>
                          <el-dropdown-menu>
                            <el-dropdown-item
                              v-if="task.status !== 'done' && task.status !== 'cancelled' && task.status !== 'in_progress'"
                              command="in_progress"
                            >标记为进行中</el-dropdown-item>
                            <el-dropdown-item
                              v-if="task.status !== 'done' && task.status !== 'cancelled'"
                              command="focus"
                            >
                              <el-icon><VideoPlay /></el-icon>
                              {{ focusSession && focusSession.taskId === task.id ? '结束专注' : '开始专注' }}
                            </el-dropdown-item>
                            <el-dropdown-item command="copy">复制为新草稿</el-dropdown-item>
                            <el-dropdown-item command="delete" divided>
                              <span style="color: #d97757;">删除(5 秒内可撤销)</span>
                            </el-dropdown-item>
                          </el-dropdown-menu>
                        </template>
                      </el-dropdown>
                    </div>
                  </article>
                </div>
              </div>
            </div>
          </div>

          <div v-else-if="activeView === 'tomorrow'">
            <div v-if="tomorrowGroups.length === 0" class="timeline-empty">
              <strong>明天还没有计划</strong>
              <span>点上方"明天做"快速预设,把今天先不急的事排到明天。</span>
            </div>
            <div v-else class="timeline-groups">
              <div class="timeline-group">
                <div class="timeline-group-head">
                  <div>
                    <h4>明天 · {{ tomorrowDateString() }}</h4>
                    <p>共 {{ tomorrowGroups.reduce((s, g) => s + g.items.length, 0) }} 项</p>
                  </div>
                  <el-tag size="small" type="primary">明日待办</el-tag>
                </div>
                <div class="task-list">
                  <article
                    v-for="task in tomorrowGroups.flatMap(g => g.items)"
                    :key="task.id"
                    class="task-card"
                    :data-task-id="task.id"
                    :class="[`status-${task.status}`, `priority-${task.priority}`, { 'task-card-celebrate': celebratedTaskIds.has(task.id), 'task-card-overdue': isTaskOverdue(task), 'task-card-bulk-selected': isSelected(task.id) }]"
                  >
                    <div class="task-card-top">
                      <div class="task-copy" @click="openTask(task)">
                        <el-checkbox
                          class="task-card-select"
                          :model-value="isSelected(task.id)"
                          @click.stop
                          @change="toggleSelected(task.id, $event)"
                        />
                        <h5>{{ task.title }}</h5>
                        <p>{{ task.detail || '点击补充详情' }}</p>
                      </div>
                      <div class="task-tags">
                        <button
                          class="pin-toggle"
                          :class="{ 'pin-toggle-active': isPinned(task.id) }"
                          :title="isPinned(task.id) ? '取消聚焦' : '置入今天聚焦'"
                          @click.stop="togglePin(task)"
                        >
                          <el-icon><component :is="isPinned(task.id) ? 'StarFilled' : 'Star'" /></el-icon>
                        </button>
                        <el-tag size="small">{{ entryTypeLabel(task.entry_type) }}</el-tag>
                        <el-dropdown trigger="click" size="small" @command="(p) => updateTask(task, { priority: p })">
                          <el-tag size="small" :type="priorityTagType(task.priority)" class="clickable-priority">
                            {{ priorityLabel(task.priority) }}
                          </el-tag>
                          <template #dropdown>
                            <el-dropdown-menu>
                              <el-dropdown-item command="urgent">🔴 紧急</el-dropdown-item>
                              <el-dropdown-item command="high">🟠 高</el-dropdown-item>
                              <el-dropdown-item command="medium">🔵 中</el-dropdown-item>
                              <el-dropdown-item command="low">⚪ 低</el-dropdown-item>
                            </el-dropdown-menu>
                          </template>
                        </el-dropdown>
                        <el-dropdown trigger="click" size="small" @command="(e) => updateTask(task, { energy_level: e || '' })">
                          <el-tag v-if="task.energy_level" size="small" :type="energyTagType(task.energy_level)" class="clickable-priority">
                            {{ energyIcon(task.energy_level) }} {{ energyLabel(task.energy_level) }}
                          </el-tag>
                          <el-tag v-else size="small" class="clickable-priority energy-hint-tag" title="点一下标精力类型">
                            🪫 标精力
                          </el-tag>
                          <template #dropdown>
                            <el-dropdown-menu>
                              <el-dropdown-item command="deep">🧠 需专注</el-dropdown-item>
                              <el-dropdown-item command="shallow">📋 事务性</el-dropdown-item>
                              <el-dropdown-item command="errand">🚶 需外出</el-dropdown-item>
                              <el-dropdown-item command="creative">💡 创意型</el-dropdown-item>
                              <el-dropdown-item v-if="task.energy_level" command="" divided>清空</el-dropdown-item>
                            </el-dropdown-menu>
                          </template>
                        </el-dropdown>
                        <span v-if="task.status !== 'cancelled'" class="task-status-cycle" :title="statusCycleHint(task)" @click.stop="cycleTaskStatus(task)">
                          <el-tag size="small" :type="statusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
                          <el-icon class="status-cycle-hint"><Refresh /></el-icon>
                        </span>
                        <el-tag v-else size="small" :type="statusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
                        <!-- 阶段 5 减法 (iter 20):重复任务接力标识 — 让"我正在做这件事第几次"可见 -->
                        <el-tag
                          v-if="parseFamilyRound(task) > 0"
                          size="small"
                          class="family-round-chip"
                          :title="`这是你第 ${parseFamilyRound(task)} 次做这件重复任务`"
                        >
                          🔁 第 {{ parseFamilyRound(task) }} 次
                        </el-tag>
                      </div>
                    </div>
                    <div class="task-meta">
                      <span v-if="task.remind_at">提醒 {{ formatDateTime(task.remind_at) }}</span>
                      <span v-if="repeatSummary(task)">重复 {{ repeatSummary(task) }}</span>
                    </div>
                    <div class="task-actions">
                      <el-button size="small" type="primary" plain @click="setTaskStatus(task, 'done')">完成</el-button>
                      <el-button v-if="task.status === 'open'" size="small" plain @click="setTaskStatus(task, 'in_progress')">进行中</el-button>
                      <!-- 阶段 5 减法 (iter 49):tomorrow 1 步"⏸ 暂停" — 补齐状态机
                           之前:"进行中"按钮一直显示,in_progress 任务点它无效果(仍在 in_progress)
                           现在:open 时显示"进行中",in_progress 时显示"⏸ 暂停",状态机清晰闭环
                           与 timeline (planned) iter 49 + 主页事件卡 iter 45 + 事件视图 iter 48 完全对称
                           复用 setTaskStatus(task, 'open')(已有函数,0 新增) -->
                      <el-button
                        v-if="task.status === 'in_progress'"
                        size="small"
                        plain
                        :title="'1 步暂停,回到 open 状态,稍后可继续'"
                        @click="setTaskStatus(task, 'open')"
                      >⏸ 暂停</el-button>
                      <el-button size="small" plain @click="moveTaskBucket(task, 'someday')">放一放</el-button>
                      <el-button size="small" plain @click="openCommentDrawer(task)">
                        <el-icon><ChatDotRound /></el-icon>
                        评论
                      </el-button>
                      <el-button size="small" plain @click="openTask(task)">编辑</el-button>
                      <el-button size="small" plain @click="copyTask(task)" title="复制为新草稿,可继续修改后保存">复制</el-button>
                      <!-- 取消 1 步化:明天的事也可能临时不要做 -->
                      <CancelReasonPopover
                        v-if="task.status !== 'done' && task.status !== 'cancelled'"
                        button-text="取消"
                        @cancel="(r) => cancelTaskWithReason(task, r)"
                      />
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
              <!-- 减法:7 天外的事件也折叠 -->
              <div v-if="timelineFolded && foldedEventItemCount > 0" class="timeline-fold-banner">
                <span>还有 <strong>{{ foldedEventItemCount }}</strong> 个事件排在 {{ TIMELINE_FOLD_DAYS }} 天后</span>
                <el-button size="small" plain @click="toggleTimelineFold">展开看看</el-button>
              </div>
              <div v-for="group in displayedEventGroups" :key="group.date" class="timeline-group">
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
                    :data-task-id="task.id"
                    :class="[`status-${task.status}`, `priority-${task.priority}`, { 'task-card-celebrate': celebratedTaskIds.has(task.id), 'task-card-overdue': isTaskOverdue(task), 'task-card-bulk-selected': isSelected(task.id) }]"
                  >
                    <div class="task-card-top">
                      <div class="task-copy" @click="openTask(task)">
                        <el-checkbox
                          class="task-card-select"
                          :model-value="isSelected(task.id)"
                          @click.stop
                          @change="toggleSelected(task.id, $event)"
                        />
                        <h5>{{ task.title }}</h5>
                        <p>{{ task.detail || '补充地点、联系人、准备事项' }}</p>
                      </div>
                      <div class="task-tags">
                        <button
                          class="pin-toggle"
                          :class="{ 'pin-toggle-active': isPinned(task.id) }"
                          :title="isPinned(task.id) ? '取消聚焦' : '置入今天聚焦'"
                          @click.stop="togglePin(task)"
                        >
                          <el-icon><component :is="isPinned(task.id) ? 'StarFilled' : 'Star'" /></el-icon>
                        </button>
                        <el-tag size="small" type="success">事件</el-tag>
                        <el-dropdown trigger="click" size="small" @command="(p) => updateTask(task, { priority: p })">
                          <el-tag size="small" :type="priorityTagType(task.priority)" class="clickable-priority">
                            {{ priorityLabel(task.priority) }}
                          </el-tag>
                          <template #dropdown>
                            <el-dropdown-menu>
                              <el-dropdown-item command="urgent">🔴 紧急</el-dropdown-item>
                              <el-dropdown-item command="high">🟠 高</el-dropdown-item>
                              <el-dropdown-item command="medium">🔵 中</el-dropdown-item>
                              <el-dropdown-item command="low">⚪ 低</el-dropdown-item>
                            </el-dropdown-menu>
                          </template>
                        </el-dropdown>
                        <el-dropdown trigger="click" size="small" @command="(e) => updateTask(task, { energy_level: e || '' })">
                          <el-tag v-if="task.energy_level" size="small" :type="energyTagType(task.energy_level)" class="clickable-priority">
                            {{ energyIcon(task.energy_level) }} {{ energyLabel(task.energy_level) }}
                          </el-tag>
                          <el-tag v-else size="small" class="clickable-priority energy-hint-tag" title="点一下标精力类型">
                            🪫 标精力
                          </el-tag>
                          <template #dropdown>
                            <el-dropdown-menu>
                              <el-dropdown-item command="deep">🧠 需专注</el-dropdown-item>
                              <el-dropdown-item command="shallow">📋 事务性</el-dropdown-item>
                              <el-dropdown-item command="errand">🚶 需外出</el-dropdown-item>
                              <el-dropdown-item command="creative">💡 创意型</el-dropdown-item>
                              <el-dropdown-item v-if="task.energy_level" command="" divided>清空</el-dropdown-item>
                            </el-dropdown-menu>
                          </template>
                        </el-dropdown>
                        <span v-if="task.status !== 'cancelled'" class="task-status-cycle" :title="statusCycleHint(task)" @click.stop="cycleTaskStatus(task)">
                          <el-tag size="small" :type="statusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
                          <el-icon class="status-cycle-hint"><Refresh /></el-icon>
                        </span>
                        <el-tag v-else size="small" :type="statusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
                        <!-- 阶段 5 减法 (iter 20):重复任务接力标识 — 让"我正在做这件事第几次"可见 -->
                        <el-tag
                          v-if="parseFamilyRound(task) > 0"
                          size="small"
                          class="family-round-chip"
                          :title="`这是你第 ${parseFamilyRound(task)} 次做这件重复任务`"
                        >
                          🔁 第 {{ parseFamilyRound(task) }} 次
                        </el-tag>
                      </div>
                    </div>
                    <div class="task-meta">
                      <TaskDateChip
                        :task="task"
                        @update="(date, label) => postponeTo(task, date, label)"
                        @move-to-inbox="moveTaskBucket(task, 'inbox')"
                      />
                      <span v-if="task.remind_at">提醒 {{ formatDateTime(task.remind_at) }}</span>
                      <span v-if="repeatSummary(task)">重复 {{ repeatSummary(task) }}</span>
                      <span v-if="task.time_hint">{{ task.time_hint }}</span>
                      <span v-if="(task.completion_count || 0) >= 2" class="task-completion-badge" :title="`这件事你已经完成过 ${task.completion_count} 次`">已完成 {{ task.completion_count }} 次</span>
                      <HabitSeedPopover
                        v-if="habitSeedEligible(task)"
                        :task="task"
                        @set="(payload) => applyHabitSeed(task, payload)"
                      />
                      <span v-if="taskAgeDays(task) >= 3" class="task-age-badge" :title="`${taskAgeDays(task)} 天前创建,需要的话可以推迟或取消`">{{ taskAgeLabel(task) }}</span>
                      <span v-if="isTaskOverdue(task)" class="task-overdue-badge">
                        已超期 {{ taskOverdueDays(task) }} 天
                        <button class="overdue-snooze-btn" @click.stop="postponeToTomorrow(task)" title="推迟到明天">→ 明天</button>
                        <button class="overdue-snooze-btn" @click.stop="postponeToNextMonday(task)" title="推迟到下周一">→ 下周一</button>
                      </span>
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
                      <!-- 阶段 5 减法 (iter 48):事件视图 1 步"▶ 开始准备" — 与主页事件卡 iter 45 完全对称
                           之前:在 ⋯ 菜单的"标记为进行中"里 = 2 步摩擦
                           现在:open 状态直接显示 ▶ 按钮,1 步进入 in_progress
                           复用 setTaskStatus(task, 'in_progress')(已有函数,0 新增)
                           视觉:plain 灰按钮(中性,与 ✓ 完成 绿/▶ 开始准备 蓝 形成动作光谱)
                           状态机:open → in_progress → done(1 步可达,与 focus primary iter 42 对称) -->
                      <el-button
                        v-if="task.status === 'open'"
                        size="small"
                        plain
                        :title="'1 步开始准备,进入 in_progress 状态'"
                        @click="setTaskStatus(task, 'in_progress')"
                      >▶ 开始准备</el-button>
                      <!-- 阶段 5 减法 (iter 48):事件视图 1 步"⏸ 暂停" — 补齐状态机
                           与主页事件卡 iter 45 + focus primary iter 42 完全对称
                           之前:in_progress 状态无法暂停,只能 ⋯ 菜单 = 2 步摩擦(且 ⋯ 菜单没这选项)
                           现在:in_progress 时直接显示 ⏸ 按钮,1 步回到 open
                           复用 setTaskStatus(task, 'open')(已有函数,0 新增)
                           状态机完整闭环:open ↔ in_progress ↔ done(任意方向 1 步可达) -->
                      <el-button
                        v-if="task.status === 'in_progress'"
                        size="small"
                        plain
                        :title="'1 步暂停,回到 open 状态,稍后可继续准备'"
                        @click="setTaskStatus(task, 'open')"
                      >⏸ 暂停</el-button>
                      <!-- 阶段 5 减法 (iter 48):事件视图 1 步"📅 改明天" — 与 timeline 任务卡 + 主页事件卡 iter 45 完全对称
                           之前:事件视图完全没有 1 步改天能力,必须 ⋯ → 顺延…(模态,3 步摩擦)
                           现在:任何状态(除 done/cancelled)直接显示 📅 按钮,1 步改到明天,5s 撤销
                           复用 postponeToTomorrow(task)(已有函数,0 新增)
                           视觉:蓝色调(冷思考/改期)与 timeline → 明天 + focus primary 改明天色系一致
                           蓝调一致性:任何地方看到蓝色按钮 = 推迟/改期 -->
                      <el-button
                        v-if="task.status !== 'done' && task.status !== 'cancelled'"
                        size="small"
                        plain
                        class="focus-primary-postpone-btn"
                        :title="`1 步改到 ${tomorrowDateString()},5s 内可撤销`"
                        @click="postponeToTomorrow(task)"
                      >📅 改明天</el-button>
                      <el-button size="small" plain @click="openCommentDrawer(task)">
                        <el-icon><ChatDotRound /></el-icon>
                        评论
                      </el-button>
                      <el-button size="small" plain @click="importCalendar(task)">日历</el-button>
                      <el-button size="small" plain @click="openTask(task)">编辑</el-button>
                      <!-- 取消 1 步化:事件型任务也能直接取消 -->
                      <CancelReasonPopover
                        v-if="task.status !== 'done' && task.status !== 'cancelled'"
                        button-text="取消"
                        @cancel="(r) => cancelTaskWithReason(task, r)"
                      />
                      <el-dropdown trigger="click" size="small" @command="(cmd) => handleTaskMore(task, cmd)">
                        <el-button size="small" plain class="task-more-btn" :title="'更多操作'">
                          <el-icon><MoreFilled /></el-icon>
                        </el-button>
                        <template #dropdown>
                          <el-dropdown-menu>
                            <el-dropdown-item command="copy">复制为新草稿</el-dropdown-item>
                            <el-dropdown-item command="delete" divided>
                              <span style="color: #d97757;">删除(5 秒内可撤销)</span>
                            </el-dropdown-item>
                          </el-dropdown-menu>
                        </template>
                      </el-dropdown>
                    </div>
                  </article>
                </div>
              </div>
            </div>
          </div>

          <div v-else-if="activeView === 'inbox'">
            <div v-if="triageActive" class="triage-panel">
              <div class="triage-header">
                <strong>处理收件箱（{{ triageIndex + 1 }}/{{ board.inbox_items.length }}）</strong>
                <el-button text @click="triageActive = false">退出处理模式</el-button>
              </div>
              <div class="triage-card" v-if="triageCurrent">
                <h5>{{ triageCurrent.title }}</h5>
                <p>{{ triageCurrent.detail || '没有更多描述' }}</p>
                <p v-if="triageCurrent.raw_text && triageCurrent.raw_text !== triageCurrent.title" class="soft-note">原文：{{ triageCurrent.raw_text }}</p>
                <div class="triage-actions">
                  <el-button type="primary" @click="triageDecide('today')"><kbd class="kbd-hint">1</kbd> 今天做</el-button>
                  <el-button plain @click="triageDecide('planned')"><kbd class="kbd-hint">2</kbd> 改天做</el-button>
                  <el-button plain @click="triageDecide('someday')"><kbd class="kbd-hint">3</kbd> 放一放</el-button>
                  <el-button class="triage-discard-btn" plain @click="triageDecide('discard')"><kbd class="kbd-hint">4</kbd> 不做</el-button>
                </div>
                <p class="triage-kbd-hint">按 <kbd class="kbd-hint">1</kbd>/<kbd class="kbd-hint">2</kbd>/<kbd class="kbd-hint">3</kbd>/<kbd class="kbd-hint">4</kbd> 快速决定,<kbd class="kbd-hint">Esc</kbd> 退出处理</p>
              </div>
              <div v-else class="timeline-empty">
                <strong>收件箱已清空</strong>
                <span>所有事项已处理完毕。</span>
              </div>
            </div>
            <div v-else-if="triageCompleted" class="triage-summary-card">
              <div class="triage-summary-headline">
                <span class="triage-summary-icon">🎉</span>
                <div>
                  <strong>收件箱已清空</strong>
                  <p>你刚刚分流了 {{ triageStats.planned + triageStats.someday + triageStats.discarded }} 件脑子里的事。</p>
                </div>
              </div>
              <div class="triage-summary-stats">
                <div class="triage-summary-stat triage-summary-stat-today">
                  <span class="triage-summary-stat-label">今天/改天</span>
                  <strong>{{ triageStats.planned }}</strong>
                </div>
                <div class="triage-summary-stat triage-summary-stat-someday">
                  <span class="triage-summary-stat-label">放一放</span>
                  <strong>{{ triageStats.someday }}</strong>
                </div>
                <div class="triage-summary-stat triage-summary-stat-discard">
                  <span class="triage-summary-stat-label">不做</span>
                  <strong>{{ triageStats.discarded }}</strong>
                </div>
              </div>
              <p class="triage-summary-message">
                {{ triageSummaryMessage }}
              </p>
              <div class="triage-summary-actions">
                <el-button type="primary" plain @click="activeView = 'timeline'">看时间线 →</el-button>
                <el-button @click="triageCompleted = false; activeView = 'inbox'">留在收件箱</el-button>
              </div>
            </div>
            <template v-else>
              <div v-if="board.inbox_items.length === 0" class="timeline-empty">
                <strong>收件箱是空的</strong>
                <span>这是好事，说明脑子里悬着的想法不多。</span>
              </div>
              <div v-else>
                <div class="inbox-toolbar">
                  <el-button type="primary" plain @click="startTriage">逐条处理收件箱</el-button>
                  <el-checkbox
                    :model-value="allInboxSelected"
                    :indeterminate="inboxSelectedIds.size > 0 && !allInboxSelected"
                    @change="toggleSelectAllInbox"
                  >
                    全选 ({{ inboxSelectedIds.size }}/{{ board.inbox_items.length }})
                  </el-checkbox>
                  <span class="soft-note">{{ board.inbox_items.length }} 条等待分流</span>
                </div>
                <transition name="batch-bar">
                  <div v-if="inboxSelectedIds.size > 0" class="inbox-batch-bar">
                    <span>已选 {{ inboxSelectedIds.size }} 条</span>
                    <el-button size="small" type="primary" plain @click="batchUpdateTasks('move_to_today')">移到今天</el-button>
                    <el-button size="small" plain @click="batchUpdateTasks('move_to_someday')">移到以后</el-button>
                    <el-dropdown trigger="click" size="small" @command="(p) => batchUpdateTasks('set_priority', { priority: p })">
                      <el-button size="small" plain>
                        <el-icon><Flag /></el-icon>
                        改优先级
                      </el-button>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item command="urgent">🔴 紧急</el-dropdown-item>
                          <el-dropdown-item command="high">🟠 高</el-dropdown-item>
                          <el-dropdown-item command="medium">🔵 中</el-dropdown-item>
                          <el-dropdown-item command="low">⚪ 低</el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                    <el-button size="small" plain @click="batchUpdateTasks('mark_done')">标完成</el-button>
                    <el-button size="small" type="danger" plain @click="batchUpdateTasks('delete')">删除</el-button>
                    <el-button size="small" text @click="clearInboxSelection">取消</el-button>
                  </div>
                </transition>
                <div class="task-list">
              <article
                v-for="task in board.inbox_items"
                :key="task.id"
                class="task-card"
                :class="[`status-${task.status}`, `priority-${task.priority}`, { 'task-card-celebrate': celebratedTaskIds.has(task.id), 'task-card-selected': inboxSelectedIds.has(task.id), 'task-card-overdue': isTaskOverdue(task) }]"
              >
                <div class="task-card-top">
                  <div class="task-copy" @click="openTask(task)">
                    <el-checkbox
                      class="inbox-select"
                      :model-value="inboxSelectedIds.has(task.id)"
                      @click.stop
                      @change="toggleInboxSelected(task.id, $event)"
                    />
                    <h5>{{ task.title }}</h5>
                    <p>{{ task.detail || '先记在收件箱，回头再分流。' }}</p>
                  </div>
                  <div class="task-tags">
                    <button
                      class="pin-toggle"
                      :class="{ 'pin-toggle-active': isPinned(task.id) }"
                      :title="isPinned(task.id) ? '取消聚焦' : '置入今天聚焦'"
                      @click.stop="togglePin(task)"
                    >
                      <el-icon><component :is="isPinned(task.id) ? 'StarFilled' : 'Star'" /></el-icon>
                    </button>
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
                  <el-button v-if="task.status === 'open'" size="small" plain @click="setTaskStatus(task, 'in_progress')">进行中</el-button>
                  <!-- 阶段 5 减法 (iter 49):inbox 1 步"⏸ 暂停" — 补齐状态机
                       之前:inbox 视图完全没有"进行中/暂停"入口,用户标记收件箱里的小事为进行中得 ⋯ 菜单 = 2 步
                       现在:open 时显示"进行中",in_progress 时显示"⏸ 暂停",状态机清晰闭环
                       与 timeline (planned) iter 49 / tomorrow iter 49 / 主页事件卡 iter 45 / 事件视图 iter 48 完全对称
                       复用 setTaskStatus(已有函数,0 新增) -->
                  <el-button
                    v-if="task.status === 'in_progress'"
                    size="small"
                    plain
                    :title="'1 步暂停,回到 open 状态,稍后可继续'"
                    @click="setTaskStatus(task, 'open')"
                  >⏸ 暂停</el-button>
                  <el-button size="small" plain @click="moveTaskBucket(task, 'someday')">放一放</el-button>
                  <el-button size="small" plain @click="openCommentDrawer(task)">
                    <el-icon><ChatDotRound /></el-icon>
                    评论
                  </el-button>
                  <el-button size="small" plain @click="openTask(task)">编辑</el-button>
                  <el-button size="small" plain @click="copyTask(task)" title="复制为新草稿,可继续修改后保存">复制</el-button>
                </div>
              </article>
            </div>
          </div>
            </template>
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
                :data-task-id="task.id"
                :class="[`status-${task.status}`, `priority-${task.priority}`, { 'task-card-celebrate': celebratedTaskIds.has(task.id), 'task-card-overdue': isTaskOverdue(task), 'task-card-bulk-selected': isSelected(task.id) }]"
              >
                <div class="task-card-top">
                  <div class="task-copy" @click="openTask(task)">
                    <el-checkbox
                      class="task-card-select"
                      :model-value="isSelected(task.id)"
                      @click.stop
                      @change="toggleSelected(task.id, $event)"
                    />
                    <h5>{{ task.title }}</h5>
                    <p>{{ task.detail || '未来再说，但不丢。' }}</p>
                  </div>
                  <div class="task-tags">
                    <button
                      class="pin-toggle"
                      :class="{ 'pin-toggle-active': isPinned(task.id) }"
                      :title="isPinned(task.id) ? '取消聚焦' : '置入今天聚焦'"
                      @click.stop="togglePin(task)"
                    >
                      <el-icon><component :is="isPinned(task.id) ? 'StarFilled' : 'Star'" /></el-icon>
                    </button>
                    <el-tag size="small" type="warning">放一放</el-tag>
                    <el-tag size="small" :type="priorityTagType(task.priority)">{{ priorityLabel(task.priority) }}</el-tag>
                  </div>
                </div>
                <div class="task-meta">
                  <TaskDateChip
                    :task="task"
                    @update="(date, label) => postponeTo(task, date, label)"
                    @move-to-inbox="moveTaskBucket(task, 'inbox')"
                  />
                  <span v-if="(task.completion_count || 0) >= 2" class="task-completion-badge" :title="`这件事你已经完成过 ${task.completion_count} 次`">已完成 {{ task.completion_count }} 次</span>
                  <HabitSeedPopover
                    v-if="habitSeedEligible(task)"
                    :task="task"
                    @set="(payload) => applyHabitSeed(task, payload)"
                  />
                  <span v-if="taskAgeDays(task) >= 3" class="task-age-badge" :title="`${taskAgeDays(task)} 天前创建,需要的话可以推迟或取消`">{{ taskAgeLabel(task) }}</span>
                  <span v-if="isTaskOverdue(task)" class="task-overdue-badge">
                    已超期 {{ taskOverdueDays(task) }} 天
                    <button class="overdue-snooze-btn" @click.stop="postponeToTomorrow(task)" title="推迟到明天">→ 明天</button>
                    <button class="overdue-snooze-btn" @click.stop="postponeToNextMonday(task)" title="推迟到下周一">→ 下周一</button>
                  </span>
                </div>
                <div v-if="task.comment_count || task.last_comment_preview" class="task-context" @click="openCommentDrawer(task)">
                  <span class="task-context-count">进展 {{ task.comment_count || 0 }}</span>
                  <span class="task-context-text">{{ task.last_comment_preview || '已存在评论' }}</span>
                  <span v-if="task.last_comment_at">{{ formatDateTime(task.last_comment_at) }}</span>
                </div>
                <div class="task-actions">
                  <el-button size="small" type="primary" plain @click="moveTaskBucket(task, 'planned')">排入计划</el-button>
                  <el-button v-if="task.status === 'open'" size="small" plain @click="setTaskStatus(task, 'in_progress')">进行中</el-button>
                  <!-- 阶段 5 减法 (iter 49):someday 1 步"⏸ 暂停" — 补齐状态机
                       之前:someday 视图完全没有"进行中/暂停"入口
                       现在:open 时显示"进行中",in_progress 时显示"⏸ 暂停",状态机清晰闭环
                       与 timeline (planned) iter 49 / tomorrow iter 49 / inbox iter 49 / 主页事件卡 iter 45 / 事件视图 iter 48 完全对称
                       复用 setTaskStatus(已有函数,0 新增) -->
                  <el-button
                    v-if="task.status === 'in_progress'"
                    size="small"
                    plain
                    :title="'1 步暂停,回到 open 状态,稍后可继续'"
                    @click="setTaskStatus(task, 'open')"
                  >⏸ 暂停</el-button>
                  <el-button size="small" plain @click="openCommentDrawer(task)">
                    <el-icon><ChatDotRound /></el-icon>
                    评论
                  </el-button>
                  <el-button size="small" plain @click="openTask(task)">编辑</el-button>
                  <el-dropdown trigger="click" size="small" @command="(cmd) => handleTaskMore(task, cmd)">
                    <el-button size="small" plain class="task-more-btn" :title="'更多操作'">
                      <el-icon><MoreFilled /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="copy">复制为新草稿</el-dropdown-item>
                        <el-dropdown-item command="delete" divided>
                          <span style="color: #d97757;">删除(5 秒内可撤销)</span>
                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
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

            <!-- Search & Filter -->
            <div class="meeting-search-bar">
              <el-input
                v-model="meetingSearchQuery"
                placeholder="搜索会议标题或内容…"
                clearable
                class="meeting-search-input"
              />
              <el-date-picker
                v-model="meetingDateRange"
                type="daterange"
                value-format="YYYY-MM-DD"
                range-separator="至"
                start-placeholder="开始日期"
                end-placeholder="结束日期"
                class="meeting-date-filter"
              />
            </div>

            <!-- Loading skeleton -->
            <div v-if="loadingMeetings" class="task-list">
              <el-skeleton v-for="n in 3" :key="n" animated class="meeting-skeleton">
                <template #template>
                  <el-skeleton-item variant="h3" style="width:50%;margin-bottom:8px" />
                  <el-skeleton-item variant="text" style="margin-bottom:4px" />
                  <el-skeleton-item variant="text" style="width:60%" />
                </template>
              </el-skeleton>
            </div>

            <!-- Empty state -->
            <div v-else-if="groupedMeetings.length === 0" class="timeline-empty">
              <strong v-if="isMeetingFiltered">{{ meetingSearchQuery || meetingDateRange ? '没有找到匹配的会议纪要' : '还没有会议纪要' }}</strong>
              <strong v-else>还没有会议纪要</strong>
              <span>记录会议内容和待办，支持录音和 AI 总结。</span>
            </div>

            <!-- Grouped meetings -->
            <div v-else class="task-list">
              <template v-for="group in groupedMeetings" :key="group.month">
                <div class="meeting-month-divider">
                  <span class="meeting-month-label">{{ group.month }}</span>
                  <span class="meeting-month-count">{{ group.items.length }} 条</span>
                </div>
                <article
                  v-for="m in group.items"
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
                    <p>{{ truncateText(m.content, 120) }}</p>
                  </div>

                  <!-- Summary + action items row -->
                  <div v-if="m.summary || (m.action_items && m.action_items !== '[]')" class="meeting-card-extras">
                    <div v-if="m.summary" class="meeting-card-summary" @click="openMeetingEdit(m)">
                      <el-tag size="small" type="primary">AI 摘要</el-tag>
                      <span>{{ truncateText(m.summary, 80) }}</span>
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
                    <el-button v-if="m.content" size="small" plain :loading="summarizingId === m.id" @click="summarizeMeeting(m)">{{ m.summary ? '重新总结' : 'AI 总结' }}</el-button>
                    <el-button size="small" plain @click="exportMeetingMarkdown(m)">导出</el-button>
                    <el-button size="small" plain @click="deleteMeetingConfirm(m)">删除</el-button>
                  </div>
                </article>
              </template>
            </div>
          </div>

          <div v-else>
            <div v-if="board.recent_items.length === 0" class="timeline-empty">
              <strong>最近还没有完成或取消记录</strong>
              <span>完成一件也值得被看见。</span>
            </div>
            <div v-else class="task-list">
              <article
                v-for="task in displayedRecent"
                :key="task.id"
                class="task-card"
                :data-task-id="task.id"
                :class="[`status-${task.status}`, `priority-${task.priority}`, { 'task-card-celebrate': celebratedTaskIds.has(task.id), 'task-card-overdue': isTaskOverdue(task), 'task-card-bulk-selected': isSelected(task.id) }]"
              >
                <div class="task-card-top">
                  <div class="task-copy" @click="openTask(task)">
                    <h5>{{ task.title }}</h5>
                    <p>{{ task.detail || '这条记录已经收尾。' }}</p>
                  </div>
                  <div class="task-tags">
                    <span class="task-status-cycle" :title="statusCycleHint(task)" @click.stop="cycleTaskStatus(task)">
                      <el-tag size="small" :type="statusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
                      <el-icon class="status-cycle-hint"><Refresh /></el-icon>
                    </span>
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
                  <!-- 阶段 5 减法 (iter 50):recent view 1 步"↻ 改明天重开" — 复活已完成的旧任务
                       之前:重新打开(setTaskStatus 'open')只切状态,保留原计划日期(昨天/几天前)
                       必须:重新打开 → 跳 timeline 找卡 → 改日期(3+ 步摩擦)
                       现在:recent view 直接点 ↻ 改明天重开 → 1 步完成
                       视觉:蓝色调(冷思考/改期)与 focus.primary 改明天 / event card iter 45 色系一致
                       复用 updateTask 路径(已有,0 新增) -->
                  <el-button
                    size="small"
                    plain
                    class="focus-primary-postpone-btn"
                    :title="`1 步重新打开并改到 ${tomorrowDateString()}`"
                    @click="reopenRecentToTomorrow(task)"
                  >↻ 改明天重开</el-button>
                  <el-button size="small" plain @click="openCommentDrawer(task)">
                    <el-icon><ChatDotRound /></el-icon>
                    评论
                  </el-button>
                  <el-button size="small" plain @click="openTask(task)">查看</el-button>
                  <el-dropdown trigger="click" size="small" @command="(cmd) => handleTaskMore(task, cmd)">
                    <el-button size="small" plain class="task-more-btn" :title="'更多操作'">
                      <el-icon><MoreFilled /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="copy">复制为新草稿</el-dropdown-item>
                        <el-dropdown-item command="delete" divided>
                          <span style="color: #d97757;">删除(5 秒内可撤销)</span>
                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </article>
              <div v-if="!recentExpanded && board.recent_items.length > recentPageSize" class="recent-expand-row">
                <el-button plain round @click="recentExpanded = true">
                  查看更早的 {{ board.recent_items.length - recentPageSize }} 条 →
                </el-button>
              </div>
            </div>
          </div>
        </template>
      </section>

      <transition name="batch-bar">
      <div v-if="selectedTaskIds.size > 0" class="batch-bar-floating">
        <span class="batch-bar-count">已选 {{ selectedTaskIds.size }} 条</span>
        <el-button size="small" type="primary" plain @click="batchUpdateTasksGeneric('mark_done')">标完成</el-button>
        <el-button size="small" plain @click="batchUpdateTasksGeneric('move_to_today')">移到今天</el-button>
        <el-button size="small" plain @click="batchUpdateTasksGeneric('move_to_someday')">放一放</el-button>
        <el-dropdown trigger="click" size="small" @command="(p) => batchUpdateTasksGeneric('set_priority', { priority: p })">
          <el-button size="small" plain>
            <el-icon><Flag /></el-icon>
            改优先级
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="urgent">🔴 紧急</el-dropdown-item>
              <el-dropdown-item command="high">🟠 高</el-dropdown-item>
              <el-dropdown-item command="medium">🔵 中</el-dropdown-item>
              <el-dropdown-item command="low">⚪ 低</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button size="small" type="danger" plain @click="batchUpdateTasksGeneric('delete')">删除</el-button>
        <el-button size="small" text @click="clearSelection">清空</el-button>
      </div>
    </transition>
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
        <el-divider />
        <div class="settings-section">
          <h4>提醒通知</h4>
          <el-switch
            v-model="notificationEnabled"
            :loading="notificationRequesting"
            :disabled="!notificationSupported"
            active-text="启用浏览器通知"
            @change="toggleNotifications"
          />
          <div v-if="!notificationSupported" class="soft-note">
            当前浏览器不支持 Notification API
          </div>
          <div v-else-if="notificationPermission === 'denied'" class="soft-note">
            浏览器已拒绝通知权限,请在浏览器设置中开启
          </div>
          <div v-else-if="notificationEnabled" class="soft-note">
            {{ notifiedTaskIds.size }} 条事项已被提醒过 · 任务到达提醒时间时,系统会弹出通知
            <el-button v-if="lastNotificationPreview" size="small" text @click="testNotification">发送测试通知</el-button>
          </div>
        </div>
        <el-divider />
        <div class="settings-section">
          <h4>安静模式</h4>
          <el-switch
            v-model="quietMode"
            active-text="沉浸整理时减少打扰"
            @change="toggleQuietMode"
          />
          <div class="soft-note">
            关闭后:完成事项不再播放庆祝动画和"下一项推荐"提示,计时器暂停呼吸灯也会静默。
            数据无任何变化,所有事项、评论、录音都正常。
          </div>
        </div>
        <el-divider />
        <div class="settings-section">
          <h4>数据导出 / 导入</h4>
          <div class="soft-note">
            你的数据完全属于你。可以随时导出备份,或从备份恢复到当前档案。<br>
            <strong>导入不会覆盖任何已有事项</strong> — 每个导入的事项都会创建新 ID,和现有数据并存。
          </div>
          <div class="settings-export-buttons">
            <el-button :loading="exportingJSON" plain @click="exportAllDataJSON">
              <el-icon><Document /></el-icon>
              导出完整备份 (JSON)
            </el-button>
            <el-button :loading="exportingCSV" plain @click="exportTasksCSV">
              <el-icon><Files /></el-icon>
              导出任务列表 (CSV)
            </el-button>
            <el-button :loading="importingJSON" type="success" plain @click="triggerImportPicker">
              <el-icon><Files /></el-icon>
              从 JSON 备份恢复
            </el-button>
            <input
              ref="importFileInput"
              type="file"
              accept="application/json,.json"
              class="import-file-input"
              @change="onImportFileChange"
            />
          </div>
          <div v-if="importProgress" class="import-progress">
            {{ importProgress }}
          </div>
        </div>
        <el-divider />
        <div class="settings-section">
          <h4>系统日历同步 (ICS 订阅)</h4>
          <div class="soft-note">
            把链接添加到 <strong>macOS Calendar / iPhone / Google Calendar / Outlook</strong> 即可在系统日历里看到所有有日期的事项,改完自动同步。<br>
            <span class="warn-text">⚠️ 链接含访问密钥,等同于你的档案密码,请勿分享他人。需要时可点"重置密钥"轮换。</span>
          </div>
          <div class="ics-url-row">
            <el-input
              :model-value="calendarSubscribeUrl"
              readonly
              placeholder="登录后生成"
              class="ics-url-input"
            >
              <template #prefix><el-icon><Link /></el-icon></template>
            </el-input>
            <el-button type="primary" :disabled="!calendarSubscribeUrl" @click="copyCalendarUrl">
              <el-icon><DocumentCopy /></el-icon>
              复制链接
            </el-button>
          </div>
          <div class="ics-options-row">
            <el-checkbox v-model="calIncludeCancelled">包含已取消事项</el-checkbox>
            <el-button size="small" plain :disabled="!calendarSubscribeUrl" @click="() => window.open(calendarSubscribeUrl, '_blank')">
              在浏览器预览
            </el-button>
          </div>
          <details class="ics-help">
            <summary>怎么添加订阅?</summary>
            <ul>
              <li><strong>macOS Calendar</strong>: 文件 → 新建日历订阅 → 粘贴链接 → 设为每 15 分钟自动刷新</li>
              <li><strong>iPhone</strong>: 设置 → 日历 → 账户 → 添加账户 → 其他 → 添加已订阅的日历</li>
              <li><strong>Google Calendar</strong>: 设置 → 添加日历 → 来自 URL → 粘贴链接</li>
              <li><strong>Outlook</strong>: 添加日历 → 来自 Internet → 粘贴链接</li>
            </ul>
          </details>
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
            <el-option label="🚨 紧急" value="urgent" />
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
        <div class="quick-row quick-row-natural-date">
          <el-input
            v-model="taskNaturalDate"
            :placeholder="taskNaturalDateHint"
            clearable
            @keydown.enter.prevent="applyTaskNaturalDate"
            @input="onTaskNaturalDateInput"
          >
            <template #prefix><el-icon><MagicStick /></el-icon></template>
          </el-input>
          <el-button :disabled="!taskNaturalDate.trim()" plain @click="applyTaskNaturalDate">识别</el-button>
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
        <el-input v-model="taskForm.intent" placeholder="意图/目的（可选）" maxlength="80" show-word-limit style="margin-top:8px" />
        <el-select v-model="taskForm.energyLevel" placeholder="精力类型（可选）" clearable style="width:100%;margin-top:8px">
          <el-option label="需高度专注 (Deep)" value="deep" />
          <el-option label="事务性 (Shallow)" value="shallow" />
          <el-option label="需外出 (Errand)" value="errand" />
          <el-option label="创意/思考 (Creative)" value="creative" />
        </el-select>
        <div v-if="taskForm.rawText" class="detail-raw-text">
          <details>
            <summary style="cursor:pointer;font-size:13px;color:#909399;">原文</summary>
            <p style="margin:8px 0 0;font-size:13px;color:#909399;white-space:pre-wrap;line-height:1.6;">{{ taskForm.rawText }}</p>
          </details>
        </div>
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
              <p class="comment-content" v-html="renderCommentContent(comment.content)"></p>
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

    <el-drawer
      v-model="commentDrawerVisible"
      title="事项评论"
      :size="drawerSize"
      :close-on-click-modal="!isTaskRecording"
      :close-on-press-escape="!isTaskRecording"
      :before-close="handleCommentDrawerClose"
    >
      <div class="drawer-stack">
        <div class="task-recording-block">
          <div class="task-recording-head">
            <strong>沟通录音</strong>
            <div class="task-recording-head-right">
              <el-tooltip
                v-if="mediaRecorderSupported"
                :content="taskAutoContinue ? '已开启:到 5 分钟自动截断并继续录下一段(连续 5 段后停)' : '已关闭:不会自动续录,需手动停止'"
                placement="top"
              >
                <el-switch
                  v-model="taskAutoContinue"
                  size="small"
                  active-text="5 分钟自动续录"
                  inactive-text="手动停止"
                  inline-prompt
                />
              </el-tooltip>
              <el-button
                v-if="mediaRecorderSupported"
                size="small"
                :type="isTaskRecording ? 'danger' : 'primary'"
                plain
                @click="toggleTaskRecording"
              >
                <el-icon><Microphone /></el-icon>
                {{ isTaskRecording ? `停止录音 ${formatClock(taskRecordingTime)}` : '开始录音' }}
              </el-button>
              <span v-else class="soft-note">当前浏览器不支持录音</span>
            </div>
          </div>
          <p v-if="isTaskRecording && taskRecordingRestartCount > 0" class="soft-note">
            自动分段中 · 第 {{ taskRecordingRestartCount }} 段 · 每段最长 5 分钟
          </p>
          <p v-if="taskRecordingUploading" class="soft-note">录音上传中…</p>
          <div v-if="taskRecordings.length > 0" class="task-recording-list">
            <div v-for="rec in taskRecordings" :key="rec.id" class="task-recording-item" :data-recording-id="rec.id">
              <div class="comment-item-top">
                <strong>{{ rec.title }}</strong>
                <span>{{ formatDateTime(rec.created_at) }}{{ rec.duration_sec ? ' · ' + formatClock(rec.duration_sec) : '' }}</span>
              </div>
              <audio :src="recordingAudioUrl(rec)" controls preload="none" class="task-recording-audio" />
              <p v-if="rec.status === 'transcribing'" class="soft-note">转写中…</p>
              <template v-else-if="rec.transcript">
                <p
                  class="task-recording-transcript"
                  :class="{ 'transcript-collapsed': isTranscriptLong(rec) && !expandedTranscripts[rec.id] }"
                >{{ rec.transcript }}</p>
                <div class="task-recording-actions">
                  <el-button
                    v-if="isTranscriptLong(rec)"
                    size="small"
                    link
                    @click="toggleTranscript(rec.id)"
                  >{{ expandedTranscripts[rec.id] ? '收起' : '展开全文' }}</el-button>
                  <el-button size="small" link type="primary" @click="recordingToComment(rec)">转为评论</el-button>
                  <el-button
                    size="small"
                    link
                    type="success"
                    :loading="summarizingRecordingId === rec.id"
                    @click="summarizeTaskRecording(rec)"
                  >{{ rec.summary ? '重新总结' : '生成总结' }}</el-button>
                </div>
                <div v-if="rec.summary" class="task-recording-summary">
                  <span class="summary-label">AI 总结</span>
                  <p>{{ rec.summary }}</p>
                </div>
              </template>
              <el-button v-else size="small" link @click="transcribeTaskRecording(rec)">转写文字</el-button>
            </div>
          </div>
        </div>
        <div class="comment-timeline">
          <div v-for="comment in comments" :key="comment.id" class="comment-item" :data-comment-id="comment.id">
            <div class="comment-item-top">
              <strong>{{ comment.author }}</strong>
              <span>{{ formatDateTime(comment.created_at) }}</span>
            </div>
            <p class="comment-content" v-html="renderCommentContent(comment.content)"></p>
            <div v-if="parseCommentImages(comment.image_urls).length > 0" class="comment-images">
              <el-image
                v-for="(img, idx) in parseCommentImages(comment.image_urls)"
                :key="idx"
                :src="img"
                :preview-src-list="parseCommentImages(comment.image_urls)"
                :initial-index="idx"
                fit="cover"
                class="comment-image-thumb"
                :preview-teleported="true"
              />
            </div>
          </div>
          <div v-if="comments.length === 0" class="timeline-empty">
            还没有评论，补一句上下文也行。
          </div>
        </div>
        <el-input
          v-model="commentForm.content"
          type="textarea"
          :rows="4"
          placeholder="记录进展、补充想法、留一点上下文（可粘贴图片）"
          @paste="handleCommentPaste"
        />
        <div v-if="commentForm.imageUrls && commentForm.imageUrls.length > 0" class="comment-draft-images">
          <div v-for="(img, idx) in commentForm.imageUrls" :key="idx" class="comment-draft-image">
            <img :src="img" alt="待发送图片" />
            <button class="comment-draft-remove" @click="removeDraftImage(idx)" title="移除">×</button>
          </div>
        </div>
        <div class="drawer-actions">
          <input
            ref="commentImageInput"
            type="file"
            accept="image/*"
            multiple
            style="display:none"
            @change="handleCommentImageSelect"
          />
          <el-button :loading="uploadingCommentImage" plain @click="triggerCommentImageUpload">
            <el-icon><Picture /></el-icon>
            添加图片
          </el-button>
          <el-button type="primary" :loading="savingComment" :disabled="!commentForm.content.trim() && (!commentForm.imageUrls || commentForm.imageUrls.length === 0)" @click="submitComment">添加评论</el-button>
          <el-button @click="requestCloseCommentDrawer">关闭</el-button>
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
            <div class="ai-card-body">
              <strong class="ai-card-title">{{ item.title }}</strong>
              <p class="ai-card-detail">{{ item.detail || '无补充描述' }}</p>
            <details v-if="item.raw_text && item.raw_text !== item.title" class="ai-card-raw">
              <summary>原文</summary>
              <p>{{ item.raw_text }}</p>
            </details>
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
          <!-- 阶段 5:"我没做完"反思 — 让用户看见自己的放弃模式,而不是直接关掉应用 -->
          <div class="advice-section" v-if="review.cancellations.length > 0">
            <h4>我没做完的事 <span class="soft-note-inline">不是失败,是看清自己</span></h4>
            <div v-if="cancellationReasonSummary.length > 0" class="cancel-reason-chips">
              <span class="cancel-reason-label">放弃原因分布:</span>
              <span v-for="r in cancellationReasonSummary" :key="r.reason" class="cancel-reason-chip">
                {{ r.reason }} · {{ r.count }}
              </span>
            </div>
            <div class="focus-secondary cancellation-list">
              <div
                v-for="(item, idx) in review.cancellations"
                :key="`cancel-${idx}-${item.cancelled_at}`"
                class="focus-secondary-item cancel-item"
              >
                <div class="cancel-item-main">
                  <strong>{{ item.title }}</strong>
                  <span class="cancel-item-reason">原因:{{ item.reason }}</span>
                </div>
                <div class="cancel-item-meta">
                  <span>{{ item.cancelled_at }}</span>
                  <span v-if="item.rollover > 0" class="cancel-rollover">顺延 {{ item.rollover }} 次</span>
                  <!-- 阶段 6 减法:review 1 步复活,补完"看见 → 决定"闭环 -->
                  <button
                    v-if="item.task_id"
                    class="cancel-item-revive"
                    title="1 步复活:回到今天做掉它"
                    @click="reviveCancelledTask(item)"
                  >
                    🔄 重新捡起
                  </button>
                </div>
              </div>
            </div>
          </div>
          <!-- 阶段 5 延伸:"我推迟的事"反思 — 与取消对称,看见"我经常把什么往后推" -->
          <div class="advice-section" v-if="review.postpones.length > 0">
            <h4>我推迟的事 <span class="soft-note-inline">顺延本身不是问题,看清节奏才是</span></h4>
            <div v-if="postponeReasonSummary.length > 0" class="postpone-reason-chips">
              <span class="postpone-reason-label">推迟原因分布:</span>
              <span v-for="r in postponeReasonSummary" :key="r.reason" class="postpone-reason-chip">
                {{ r.reason }} · {{ r.count }}
              </span>
            </div>
            <div class="focus-secondary postpone-list">
              <div
                v-for="(item, idx) in review.postpones"
                :key="`postpone-${idx}-${item.postponed_at}`"
                class="focus-secondary-item postpone-item"
              >
                <div class="postpone-item-main">
                  <strong>{{ item.title }}</strong>
                  <span class="postpone-item-reason">原因:{{ item.reason }}</span>
                </div>
                <div class="postpone-item-meta">
                  <span>顺延于 {{ item.postponed_at }}</span>
                  <span v-if="item.planned_for" class="postpone-target">→ {{ item.planned_for }}</span>
                  <span v-if="item.rollover > 1" class="postpone-rollover">累计 {{ item.rollover }} 次</span>
                  <!-- 阶段 6 减法:review "我推迟的事" 1 步今天做 -->
                  <button
                    v-if="item.task_id"
                    class="postpone-item-today"
                    title="1 步把它放到今天做"
                    @click="doPostponedTaskToday(item)"
                  >
                    📅 今天做
                  </button>
                </div>
              </div>
            </div>
          </div>
          <!-- 阶段 6 减法:review 完成感受 — 让用户看见"我做完时感觉是…"
               与"我没做完的事"/"我推迟的事"对称,补完"做完了 → 感觉如何"的看见 -->
          <div class="advice-section" v-if="review.completion_feelings.length > 0">
            <h4>我做完了感觉… <span class="soft-note-inline">收尾时的状态,也是节奏</span></h4>
            <div v-if="completionFeelingSummary.length > 0" class="feeling-distribution-chips">
              <span class="feeling-distribution-label">完成感受分布:</span>
              <span v-for="f in completionFeelingSummary" :key="f.key" class="feeling-distribution-chip">
                {{ f.icon }} {{ f.label }} · {{ f.count }}
              </span>
            </div>
            <div class="focus-secondary feeling-list">
              <div
                v-for="(item, idx) in review.completion_feelings"
                :key="`feeling-${idx}-${item.completed_at}`"
                class="focus-secondary-item feeling-item"
              >
                <div class="feeling-item-main">
                  <strong>
                    <span class="feeling-emoji">{{ feelingIcon(item.feeling) }}</span>
                    {{ item.title }}
                  </strong>
                  <span class="feeling-item-label">{{ feelingLabel(item.feeling) }}</span>
                </div>
                <div class="feeling-item-meta">
                  <span>{{ item.completed_at }}</span>
                </div>
              </div>
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

    <el-drawer
      v-model="meetingDialogVisible"
      title="会议纪要"
      :size="meetingDrawerSize"
      :close-on-click-modal="meetingVoiceState !== 'recording'"
      :close-on-press-escape="meetingVoiceState !== 'recording'"
      :before-close="handleMeetingDrawerClose"
      @closed="resetMeetingForm"
    >
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
        <el-select
          v-model="meetingParticipantArray"
          multiple
          filterable
          allow-create
          default-first-option
          placeholder="参与人（输入后回车添加）"
          style="width:100%"
        ></el-select>
        <el-select
          v-model="meetingTagArray"
          multiple
          filterable
          allow-create
          default-first-option
          placeholder="标签（输入后回车创建）"
          style="width:100%"
        ></el-select>
        <div v-if="meetingVoiceState === 'recording' && meetingInterimText" class="meeting-interim">
          <el-tag size="small" type="danger" effect="dark" class="recording-badge">录音中</el-tag>
          <span class="interim-text">{{ meetingInterimText }}</span>
        </div>
        <el-input
          v-model="meetingForm.content"
          type="textarea"
          :autosize="{ minRows: 6, maxRows: 24 }"
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
          <div
            v-for="(item, idx) in parsedActionItems"
            :key="idx"
            class="meeting-action-item"
            :class="{ 'action-done': item.done }"
          >
            <el-checkbox
              :model-value="!!item.done"
              @change="toggleActionItem(idx)"
              size="small"
            />
            <el-tag size="small" type="primary" class="meeting-action-tag">{{ item.assignee || '未指定' }}</el-tag>
            <span :class="{ 'action-task-done': item.done }">{{ item.task }}</span>
            <el-button size="small" text type="primary" @click="createTaskFromAction(item)" title="转为事项">
              转为事项
            </el-button>
          </div>
        </div>
      </div>
      <template #footer>
        <div class="drawer-actions">
          <div>
            <el-button v-if="meetingForm.id" size="small" plain @click="exportMeetingMarkdown(meetingForm)">导出</el-button>
            <el-button v-if="meetingForm.id" size="small" plain @click="deleteMeeting">删除此纪要</el-button>
          </div>
          <div>
            <el-button @click="requestCloseMeetingDrawer">关闭</el-button>
            <el-button plain @click="startMeetingVoice">
              <el-icon><Microphone /></el-icon>{{ meetingVoiceState === 'recording' ? '停止录音' : '录音' }}
            </el-button>
            <el-button type="primary" :loading="savingMeeting || meetingAudioProcessing" @click="saveMeeting">保存</el-button>
          </div>
        </div>
      </template>
    </el-drawer>

    <el-dialog
      v-model="shortcutHelpVisible"
      title="键盘快捷键"
      :width="dialogWidth"
      :fullscreen="isMobile"
      align-center
    >
      <ul class="shortcut-list">
        <li v-for="item in shortcutHints" :key="item.label" class="shortcut-item">
          <span class="shortcut-keys">
            <kbd v-for="k in item.keys" :key="k" class="shortcut-kbd">{{ k }}</kbd>
          </span>
          <span class="shortcut-label">{{ item.label }}</span>
        </li>
        <li class="shortcut-item shortcut-tip">
          <span class="shortcut-keys"><kbd class="shortcut-kbd">↑</kbd><kbd class="shortcut-kbd">↓</kbd></span>
          <span class="shortcut-label">在搜索结果中上下选择</span>
        </li>
        <li class="shortcut-item shortcut-tip">
          <span class="shortcut-keys"><kbd class="shortcut-kbd">Enter</kbd></span>
          <span class="shortcut-label">打开选中的搜索结果</span>
        </li>
      </ul>
      <template #footer>
        <el-button type="primary" @click="shortcutHelpVisible = false">我知道了</el-button>
      </template>
    </el-dialog>

    <!-- 全局快速添加 (Cmd/Ctrl+K 或 n) -->
    <el-dialog
      v-model="globalQuickAddVisible"
      :show-close="false"
      :width="isMobile ? '100%' : 560"
      :fullscreen="isMobile"
      align-center
      class="global-quick-add-dialog"
    >
      <div class="global-quick-add" @keydown.esc="closeGlobalQuickAdd">
        <div class="global-quick-add-head">
          <span class="global-quick-add-mode">{{ activeKind === 'work' ? '工作' : '生活' }}</span>
          <span class="global-quick-add-hint">按 <kbd>Enter</kbd> 保存 · <kbd>Esc</kbd> 关闭</span>
        </div>
        <textarea
          ref="globalQuickAddInput"
          v-model="globalQuickAddText"
          class="global-quick-add-textarea"
          placeholder="写一件事，例如：明天下午3点前给客户发合同"
          rows="2"
          @input="onGlobalQuickAddInput"
          @keydown.enter.prevent.exact="submitGlobalQuickAdd"
          @keydown.esc.stop="closeGlobalQuickAdd"
        />
        <div v-if="globalQuickAddPreview" class="global-quick-add-preview">
          <el-icon><Calendar /></el-icon>
          <span>已识别日期: <strong>{{ globalQuickAddPreview.label }}</strong></span>
        </div>
        <div class="global-quick-add-options">
          <div class="global-quick-add-opt-group">
            <span class="global-quick-add-opt-label">优先级</span>
            <el-radio-group v-model="globalQuickAddPriority" size="small">
              <el-radio-button label="urgent">紧急</el-radio-button>
              <el-radio-button label="high">高</el-radio-button>
              <el-radio-button label="medium">中</el-radio-button>
              <el-radio-button label="low">低</el-radio-button>
            </el-radio-group>
          </div>
          <div class="global-quick-add-opt-group">
            <span class="global-quick-add-opt-label">类型</span>
            <el-radio-group v-model="globalQuickAddEntryType" size="small">
              <el-radio-button label="task">任务</el-radio-button>
              <el-radio-button label="event">事件</el-radio-button>
            </el-radio-group>
          </div>
        </div>
        <div class="global-quick-add-foot">
          <span class="global-quick-add-foot-hint">💡 支持自然语言: 明天下午3点 / 3天后 / 下周一 / 12月15日</span>
          <div>
            <el-button @click="closeGlobalQuickAdd">取消</el-button>
            <el-button type="primary" :loading="globalQuickAddSaving" @click="submitGlobalQuickAdd">记录</el-button>
          </div>
        </div>
      </div>
    </el-dialog>

    <el-dialog
      v-model="routineDialogVisible"
      :title="editingRoutine && routines.find(x => x.id === editingRoutine.id) ? '编辑套路' : '新建套路'"
      :width="dialogWidth"
      :fullscreen="isMobile"
      align-center
      @close="editingRoutine = null"
    >
      <div v-if="editingRoutine" class="routine-editor">
        <div class="routine-editor-row">
          <label>套路名</label>
          <el-input v-model="editingRoutine.name" placeholder="例如: 周一上午 / 出差前夜 / 月初结算" maxlength="30" show-word-limit />
        </div>
        <div class="routine-editor-list">
          <div v-for="(it, idx) in editingRoutine.items" :key="idx" class="routine-editor-item">
            <div class="routine-editor-item-main">
              <el-input v-model="it.title" placeholder="任务标题" maxlength="80" />
              <el-select v-model="it.priority" style="width: 110px">
                <el-option label="🚨 紧急" value="urgent" />
                <el-option label="高" value="high" />
                <el-option label="中" value="medium" />
                <el-option label="低" value="low" />
              </el-select>
              <el-select v-model="it.bucket" style="width: 110px" :disabled="it.entryType === 'event'">
                <el-option label="计划中" value="planned" />
                <el-option label="收件箱" value="inbox" />
                <el-option label="放一放" value="someday" />
              </el-select>
              <el-input-number v-model="it.offsetDays" :min="0" :max="365" :step="1" controls-position="right" style="width: 110px" />
              <span class="routine-editor-offset-hint">天后</span>
            </div>
            <div class="routine-editor-item-actions">
              <el-button size="small" plain :disabled="idx === 0" @click="moveEditingRoutineItem(idx, -1)">↑</el-button>
              <el-button size="small" plain :disabled="idx === editingRoutine.items.length - 1" @click="moveEditingRoutineItem(idx, 1)">↓</el-button>
              <el-button size="small" plain @click="removeItemFromEditingRoutine(idx)">删除</el-button>
            </div>
          </div>
        </div>
        <el-button plain class="routine-editor-add" @click="addItemToEditingRoutine">+ 添加任务项</el-button>
      </div>
      <template #footer>
        <el-button @click="routineDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="persistEditingRoutine">保存套路</el-button>
      </template>
    </el-dialog>

    <transition name="undo-slide">
      <div v-if="undoState.visible" class="undo-snackbar">
        <div class="undo-message">
          <el-icon class="undo-icon"><Refresh /></el-icon>
          <span>{{ undoState.message }}</span>
        </div>
        <div class="undo-actions">
          <span class="undo-tick">{{ undoState.secondsLeft }}s</span>
          <el-button type="primary" size="small" plain @click="handleUndo">撤销</el-button>
          <el-button size="small" text @click="dismissUndo">关闭</el-button>
        </div>
      </div>
    </transition>

    <button
      v-if="isMobile && profileId"
      class="fab-quick-add"
      :class="{ 'fab-with-undo': undoState.visible }"
      aria-label="快速记录"
      @click="openGlobalQuickAdd"
    >
      <el-icon><Plus /></el-icon>
      <span>快速记录</span>
    </button>
  </div>
</template>

<script setup>
import { computed, defineComponent, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Calendar, ChatDotRound, Check, CircleClose, Clock, Document, EditPen, Files, Flag, InfoFilled, MagicStick, Microphone, Moon, MoreFilled, Picture, Plus, Refresh, Setting, Star, StarFilled, UserFilled, VideoPlay } from '@element-plus/icons-vue'
import { parseNaturalDate } from '../../utils/naturalDate'

const API_BASE = '/api/planner'

function truncateText(text, maxLen) {
  if (!text) return ''
  const chars = Array.from(text)
  if (chars.length <= maxLen) return text
  return chars.slice(0, maxLen).join('') + '...'
}

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
const recentExpanded = ref(false)
const recentPageSize = 30
const displayedRecent = computed(() => {
  const items = board.value.recent_items || []
  if (recentExpanded.value || items.length <= recentPageSize) return items
  return items.slice(0, recentPageSize)
})

// 阶段 5:时间线折叠 — 默认只显示 7 天内,更远的折叠到"以后再说"。
// 减法:打开时间线不被未来 3 个月的事占满。
const timelineFolded = ref(localStorage.getItem('planner_timeline_folded') !== '0')
const TIMELINE_FOLD_DAYS = 7

function weekAheadDateString() {
  const d = new Date()
  d.setDate(d.getDate() + TIMELINE_FOLD_DAYS)
  return d.toISOString().slice(0, 10)
}

const foldedTimelineGroups = computed(() => {
  if (!timelineFolded.value) return []
  const cutoff = weekAheadDateString()
  return (filteredTimelineGroups.value || []).filter(g => (g.date || '') > cutoff)
})
const displayedTimelineGroups = computed(() => {
  if (!timelineFolded.value) return filteredTimelineGroups.value || []
  const cutoff = weekAheadDateString()
  return (filteredTimelineGroups.value || []).filter(g => (g.date || '') <= cutoff)
})
const foldedTimelineItemCount = computed(() =>
  foldedTimelineGroups.value.reduce((sum, g) => sum + (g.items?.length || 0), 0)
)

// 顺延洞察:top 3 高顺延任务(按 rollover_count 倒序)。只取 status=open 的(已取消的不算)。
const topRolledOverTasks = computed(() => {
  const today = new Date().toISOString().slice(0, 10)
  const items = []
  for (const g of (board.value.groups || [])) {
    for (const t of (g.items || [])) {
      if (t.status !== 'open') continue
      const rollover = t.rollover_count || 0
      const isOverdue = (t.planned_for || '') < today
      if (rollover >= 1 || isOverdue) {
        items.push({ ...t, _rolloverScore: rollover * 10 + (isOverdue ? 1 : 0) })
      }
    }
  }
  items.sort((a, b) => (b._rolloverScore || 0) - (a._rolloverScore || 0))
  return items.slice(0, 3)
})
const rolloverInsightDismissedUntil = ref(localStorage.getItem('planner_rollover_dismissed_until') || '')
const rolloverInsightVisible = computed(() => {
  if (topRolledOverTasks.value.length === 0) return false
  if (!rolloverInsightDismissedUntil.value) return true
  return new Date().toISOString() >= rolloverInsightDismissedUntil.value
})
function dismissRolloverInsightToday() {
  const d = new Date()
  d.setHours(23, 59, 59, 999)
  const until = d.toISOString()
  rolloverInsightDismissedUntil.value = until
  try { localStorage.setItem('planner_rollover_dismissed_until', until) } catch (e) {}
}
async function rescheduleRolledOverToday(task) {
  await updateTask(task, {
    planned_for: new Date().toISOString().slice(0, 10),
    status: 'open'
  }, '已重新安排到今天')
}
// 阶段 5 减法 (iter 46):"🔁 顺延回顾" 1 步"↻ 顺延到明天"
// 补齐顺延决策第三选项(今天做 / 顺延 / 不再做了 = 3 选 1 决策矩阵)
// 场景:用户看到已顺延 3 次的事,"今天还是没空,先挪到明天"
// 原来:点 📅 今天做 → 后悔 → 改 task 日期(4-5 步摩擦)
// 现在:主页直接点 ↻ 顺延到明天 → 1 步完成 → 5s 撤销
// 视觉:蓝色调(与 focus.secondary 改天 / focus primary 改明天 色系一致)
// 与 focus.secondary 批量操作三色闭环(✓ 绿 / ↻ 蓝 / ✗ 棕)完全对称
async function postponeRolledOverToTomorrow(task) {
  const tomorrow = tomorrowDateString()
  await updateTask(task, {
    planned_for: tomorrow,
    status: 'open',
    postpone_reason: '顺延回顾 1 步顺延到明天'
  }, `已顺延到 ${tomorrow}`)
}
// 阶段 5 减法 (iter 50):recent view "↻ 改明天重开" 1 步
// 之前:重新打开 按钮(setTaskStatus 'open')只切状态,保留原计划日期(通常是昨天/几天前)
// 用户场景:看到昨天完成的"回客户邮件",想"等等,我还没回呢" → 重新打开(日期回到昨天)
// 必须:重新打开 → 跳 timeline 找卡 → 改日期(3+ 步摩擦)
// 现在:recent view 直接点 ↻ 改明天重开 → 1 步完成,任务复活到明天
// 视觉:蓝色调(冷思考/改期)与 focus.primary 改明天 / event card iter 45 色系一致
async function reopenRecentToTomorrow(task) {
  const tomorrow = tomorrowDateString()
  await updateTask(task, {
    planned_for: tomorrow,
    status: 'open',
    postpone_reason: 'recent 1 步改明天重开'
  }, `已重开到 ${tomorrow}`)
}
async function cancelRolledOver(task) {
  await updateTask(task, {
    status: 'cancelled',
    cancel_reason: '重新评估后放弃'
  }, '已放弃')
}

// 专注模式:timeline 内每个 group 内的 item 只保留 (urgent) OR (pinned) OR (today),
// 没有可见 item 的 group 直接隐藏。事件型时间线不受影响(事件自带时间感,不需要折叠)。
const focusedTimelineGroups = computed(() => {
  if (!focusMode.value) return null
  const today = new Date().toISOString().slice(0, 10)
  const source = displayedTimelineGroups.value || []
  const result = []
  for (const g of source) {
    const items = (g.items || []).filter((t) =>
      t.priority === 'urgent' || pinnedTaskIds.value.has(t.id) || (t.planned_for || '') === today
    )
    if (items.length > 0) result.push({ ...g, items })
  }
  return result
})
const focusHiddenItemCount = computed(() => {
  if (!focusMode.value) return 0
  const visibleIds = new Set()
  for (const g of focusedTimelineGroups.value || []) {
    for (const t of g.items || []) visibleIds.add(t.id)
  }
  let hidden = 0
  for (const g of displayedTimelineGroups.value || []) {
    for (const t of g.items || []) if (!visibleIds.has(t.id)) hidden++
  }
  return hidden
})

const displayedEventGroups = computed(() => {
  if (!timelineFolded.value) return board.value.event_groups || []
  const cutoff = weekAheadDateString()
  return (board.value.event_groups || []).filter(g => (g.date || '') <= cutoff)
})
const foldedEventGroups = computed(() => {
  if (!timelineFolded.value) return []
  const cutoff = weekAheadDateString()
  return (board.value.event_groups || []).filter(g => (g.date || '') > cutoff)
})
const foldedEventItemCount = computed(() =>
  foldedEventGroups.value.reduce((sum, g) => sum + (g.items?.length || 0), 0)
)

watch(timelineFolded, (v) => {
  try { localStorage.setItem('planner_timeline_folded', v ? '1' : '0') } catch (e) {}
})
function toggleTimelineFold() { timelineFolded.value = !timelineFolded.value }
// 注意:lowEnergyMode / focusMode 的 watch 和 toggle 函数已下移到声明之后(line 3953 附近)
// 避免 TDZ:Cannot access 'mp' before initialization(setup 阶段 watcher 提前访问未初始化 ref)
const modeHint = ref('当前是下班或休息时段，默认进入生活模式')
const isMobile = ref(false)
const quickAdvancedVisible = ref(false)
const installPromptAvailable = ref(false)
const isStandaloneMode = ref(false)

const createForm = reactive({
  name: '',
  password: '',
  notifyEmail: '',
  intent: '',
  energy_level: ''
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
  rawText: '',
  entryType: 'task',
  bucket: 'planned',
  plannedFor: '',
  remindAt: '',
  repeatType: 'none',
  repeatInterval: 1,
  repeatUntil: '',
  priority: 'medium',
  notifyEmail: '',
  intent: '',
  energy_level: ''
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

const meetingSearchQuery = ref('')
const triageActive = ref(false)
const triageIndex = ref(0)
const triageCompleted = ref(false)
const triageStats = reactive({ planned: 0, someday: 0, discarded: 0 })
const triageCurrent = computed(() => {
  if (!triageActive.value) return null
  const items = board.value.inbox_items
  return triageIndex.value < items.length ? items[triageIndex.value] : null
})
const triageSummaryMessage = computed(() => {
  const total = triageStats.planned + triageStats.someday + triageStats.discarded
  if (total === 0) return ''
  if (triageStats.discarded >= 3) return `放下了 ${triageStats.discarded} 件不做的事,心里又轻了一点。`
  if (triageStats.planned >= total * 0.7) return `几乎都安排上了,今天的节奏挺满。`
  if (triageStats.someday >= total * 0.5) return `放一放 ${triageStats.someday} 件,等以后再想起来也不迟。`
  return '收件箱清爽了,脑子也能跟着松一下。'
})
const meetingDateRange = ref(null)
const meetingDirty = ref(false)

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

function formatMonthLabel(dateStr) {
  if (!dateStr) return ''
  const parts = dateStr.split('-')
  if (parts.length < 2) return dateStr
  return parts[0] + '年' + parseInt(parts[1]) + '月'
}

const groupedMeetings = computed(() => {
  const filtered = meetings.value.filter(m => {
    if (meetingSearchQuery.value) {
      const q = meetingSearchQuery.value.toLowerCase()
      const titleMatch = (m.title || '').toLowerCase().includes(q)
      const contentMatch = (m.content || '').toLowerCase().includes(q)
      if (!titleMatch && !contentMatch) return false
    }
    if (meetingDateRange.value) {
      const md = m.meeting_date || ''
      if (meetingDateRange.value[0] && md < meetingDateRange.value[0]) return false
      if (meetingDateRange.value[1] && md > meetingDateRange.value[1]) return false
    }
    return true
  })
  const groups = {}
  for (const m of filtered) {
    const month = (m.meeting_date || '').slice(0, 7)
    if (!month) continue
    if (!groups[month]) groups[month] = []
    groups[month].push(m)
  }
  const sorted = Object.keys(groups).sort((a, b) => b.localeCompare(a))
  return sorted.map(month => ({
    month: formatMonthLabel(month),
    items: groups[month]
  }))
})

const isMeetingFiltered = computed(() => Boolean(meetingSearchQuery.value || meetingDateRange.value))

const meetingTagArray = computed({
  get: () => meetingForm.tagsText ? meetingForm.tagsText.split(/[,，、]/).map(s => s.trim()).filter(Boolean) : [],
  set: (val) => { meetingForm.tagsText = Array.isArray(val) ? val.join(', ') : val }
})
const meetingParticipantArray = computed({
  get: () => meetingForm.participantsText ? meetingForm.participantsText.split(/[,，、]/).map(s => s.trim()).filter(Boolean) : [],
  set: (val) => { meetingForm.participantsText = Array.isArray(val) ? val.join(', ') : val }
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

function formatClock(seconds) {
  const total = Math.max(0, Math.floor(seconds || 0))
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

const taskForm = reactive({
  id: '',
  kind: 'work',
  entryType: 'task',
  bucket: 'planned',
  title: '',
  detail: '',
  notes: '',
  rawText: '',
  plannedFor: '',
  remindAt: '',
  repeatType: 'none',
  repeatInterval: 1,
  repeatUntil: '',
  priority: 'medium',
  status: 'open',
  notifyEmail: '',
  intent: '',
  energyLevel: '',
  cancelReason: '',
  postponeReason: ''
})

const commentForm = reactive({
  content: '',
  imageUrls: []
})
const commentImageInput = ref(null)
const uploadingCommentImage = ref(false)

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
  highlights: [],
  cancellations: [],
  postpones: [],
  completion_feelings: []
})

// 阶段 5:聚合取消原因 — 让用户看见"我经常因为什么放弃"
const cancellationReasonSummary = computed(() => {
  const list = review.cancellations || []
  if (list.length === 0) return []
  const counts = {}
  for (const c of list) {
    const key = (c.reason || '未填原因').trim() || '未填原因'
    counts[key] = (counts[key] || 0) + 1
  }
  return Object.entries(counts)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 3)
    .map(([reason, count]) => ({ reason, count }))
})

// 阶段 5 延伸:聚合顺延原因 — 与取消对称,让用户看见"我经常把什么往后推"
const postponeReasonSummary = computed(() => {
  const list = review.postpones || []
  if (list.length === 0) return []
  const counts = {}
  for (const p of list) {
    const key = (p.reason || '未填原因').trim() || '未填原因'
    counts[key] = (counts[key] || 0) + 1
  }
  return Object.entries(counts)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 3)
    .map(([reason, count]) => ({ reason, count }))
})

// 阶段 6 减法:聚合完成感受 — 让用户看见"做完时我是轻松/学到/凑合的哪种"
const completionFeelingSummary = computed(() => {
  const list = review.completion_feelings || []
  if (list.length === 0) return []
  const counts = {}
  for (const f of list) {
    const key = (f.feeling || '').trim() || 'unset'
    counts[key] = (counts[key] || 0) + 1
  }
  return Object.entries(counts)
    .sort((a, b) => b[1] - a[1])
    .map(([key, count]) => {
      const meta = COMPLETION_FEELINGS.find((m) => m.key === key)
      return {
        key,
        count,
        icon: meta?.icon || '·',
        label: meta?.label || '未标感受'
      }
    })
})
function feelingIcon(key) {
  const meta = COMPLETION_FEELINGS.find((m) => m.key === key)
  return meta?.icon || '·'
}
function feelingLabel(key) {
  const meta = COMPLETION_FEELINGS.find((m) => m.key === key)
  return meta?.label || '未标感受'
}

// 阶段 6 减法:review 弹窗"我没做完的事" 1 步复活。
// 补完 review 的"看见 → 反思 → 决定"闭环:不让"看清自己"只停留在展示。
async function reviveCancelledTask(item) {
  if (!item || !item.task_id) return
  const today = new Date().toISOString().slice(0, 10)
  // 找到 board 里对应的 task 用于 updateTask(它接受 task 对象)
  const source = findTaskById(item.task_id)
  if (!source) {
    // fallback:直接调 API
    try {
      await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${item.task_id}`, {
        method: 'PUT',
        body: JSON.stringify({
          creator_key: creatorKey.value,
          status: 'open',
          planned_for: today,
          cancel_reason: ''
        })
      })
      await loadReview()
      ElMessage({ message: '已捡起,今天做掉它', type: 'success', duration: 2200 })
      await refreshBoard()
    } catch (e) {
      ElMessage.error(e.message || '复活失败')
    }
    return
  }
  try {
    await updateTask(source, {
      status: 'open',
      planned_for: today,
      cancel_reason: ''
    }, '已捡起')
    // 重新拉 review(让"我没做完的事"列表移除该条)
    await loadReview()
    await refreshBoard()
  } catch (e) {
    if (e?.message) ElMessage.error(e.message || '复活失败')
  }
}

// 阶段 6 减法:review 弹窗"我推迟的事" 1 步今天做。
async function doPostponedTaskToday(item) {
  if (!item || !item.task_id) return
  const today = new Date().toISOString().slice(0, 10)
  const source = findTaskById(item.task_id)
  if (!source) {
    try {
      await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${item.task_id}`, {
        method: 'PUT',
        body: JSON.stringify({
          creator_key: creatorKey.value,
          status: 'open',
          planned_for: today
        })
      })
      await loadReview()
      ElMessage({ message: '今天做掉它', type: 'success', duration: 2000 })
      await refreshBoard()
    } catch (e) {
      ElMessage.error(e.message || '操作失败')
    }
    return
  }
  try {
    await updateTask(source, {
      status: 'open',
      planned_for: today
    }, '今天做')
    await loadReview()
    await refreshBoard()
  } catch (e) {
    if (e?.message) ElMessage.error(e.message || '操作失败')
  }
}

function findTaskById(taskId) {
  if (!taskId) return null
  for (const g of (board.value.groups || [])) {
    for (const t of (g.items || [])) {
      if (t.id === taskId) return t
    }
  }
  for (const g of (board.value.event_groups || [])) {
    for (const t of (g.items || [])) {
      if (t.id === taskId) return t
    }
  }
  for (const t of (board.value.inbox_items || [])) {
    if (t.id === taskId) return t
  }
  for (const t of (board.value.recent_items || [])) {
    if (t.id === taskId) return t
  }
  return null
}

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
const shortcutHelpVisible = ref(false)
const globalQuickAddVisible = ref(false)
const globalQuickAddInput = ref(null)
const globalQuickAddText = ref('')
const globalQuickAddPriority = ref('medium')
const globalQuickAddEntryType = ref('task')
const globalQuickAddSaving = ref(false)
const globalQuickAddPreview = ref(null)
const exportingJSON = ref(false)
const exportingCSV = ref(false)
const importingJSON = ref(false)
const importProgress = ref('')
const importFileInput = ref(null)
const calIncludeCancelled = ref(false)  // ICS 订阅是否纳入已取消事项
// 主页"快速记录"输入的实时智能归类预览
const quickFormHints = ref(null)  // { date, time, priority, bucket, entryType, hint }

// 阶段 5 减法 (iter 24):多任务 1 步记录
// 「快速记录」输入框支持 / 或换行分隔多任务,1 次保存走 batch API
// 1 段时保持原单条逻辑不变(零回归);2+ 段时自动用 batch
function splitQuickTitles(value) {
  if (!value) return []
  return value
    .split(/\r?\n/)
    .flatMap(line => line.split(/\s*\/\s*/))
    .map(s => s.trim())
    .filter(Boolean)
}

const quickFormBatchTitles = computed(() => splitQuickTitles(quickForm.title))
const quickFormBatchCount = computed(() => quickFormBatchTitles.value.length)
const quickFormBatchLabel = computed(() => {
  const n = quickFormBatchCount.value
  return n > 1 ? `保存 ${n} 条` : '保存'
})
let _quickFormHintTimer = null
function onQuickFormTitleInput() {
  if (_quickFormHintTimer) clearTimeout(_quickFormHintTimer)
  _quickFormHintTimer = setTimeout(() => {
    const combined = [quickForm.title, quickForm.detail].filter(Boolean).join(' ')
    const h = extractHintsFromText(combined)
    // 只在识别到至少 1 个有意义的字段时显示
    if (h.date || h.priority || h.bucket) {
      quickFormHints.value = h
    } else {
      quickFormHints.value = null
    }
  }, 200)
}
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
const taskRecordings = ref([])
const expandedTranscripts = reactive({})
const summarizingRecordingId = ref('')
const isTaskRecording = ref(false)
const taskRecordingTime = ref(0)
const taskRecordingUploading = ref(false)
// 5 分钟自动截断 + 自动续录控制
const taskAutoContinue = ref(true)         // 是否启用自动分段续录
const userStoppedRecording = ref(false)     // 用户主动点了停止就不再续
let taskRecordingRestartCount = 0            // 当前会话内自动续录次数(防无限循环)
const undoState = ref({ visible: false, message: '', kind: '', secondsLeft: 0, onUndo: null })
let undoTimer = null
let undoTickTimer = null
const celebratedTaskIds = ref(new Set())
const celebrationBursts = ref([])
let celebrationCounter = 0
const completionLog = ref({})
const inboxSelectedIds = ref(new Set())
const selectedTaskIds = ref(new Set())
const notificationEnabled = ref(localStorage.getItem('planner_notification_enabled') === '1')
const notificationSupported = typeof window !== 'undefined' && 'Notification' in window
const notificationPermission = ref(notificationSupported ? Notification.permission : 'unsupported')
const notificationRequesting = ref(false)
const notifiedTaskIds = ref(new Set())
let notificationTimer = null
const lastNotificationPreview = ref('')
const quietMode = ref(localStorage.getItem('planner_quiet_mode') === '1')
const lowEnergyMode = ref(localStorage.getItem('planner_low_energy_mode') === '1')
const focusMode = ref(localStorage.getItem('planner_focus_mode') === '1')

// 修复 TDZ bug:watch / toggle 必须放在对应 ref 声明之后
watch(lowEnergyMode, (v) => {
  try { localStorage.setItem('planner_low_energy_mode', v ? '1' : '0') } catch (e) {}
})
function toggleLowEnergyMode() { lowEnergyMode.value = !lowEnergyMode.value }
watch(focusMode, (v) => {
  try { localStorage.setItem('planner_focus_mode', v ? '1' : '0') } catch (e) {}
})
function toggleFocusMode() { focusMode.value = !focusMode.value }

// 跨天延续:记住上次离开时间和当时的已完成总数
const lastSessionMemory = ref(null)
function readLastSessionMemory() {
  try {
    const raw = localStorage.getItem('planner_last_session')
    if (!raw) return null
    return JSON.parse(raw)
  } catch (e) { return null }
}
function writeLastSessionMemory() {
  try {
    const total = totalDoneCount.value
    const now = new Date().toISOString()
    localStorage.setItem('planner_last_session', JSON.stringify({ at: now, done_count: total }))
  } catch (e) {}
}
const crossDayHintVisible = ref(false)
function maybeShowCrossDayHint() {
  const last = readLastSessionMemory()
  if (!last || !last.at) return
  const lastDate = new Date(last.at)
  const now = new Date()
  const dayDiff = Math.floor((now - lastDate) / (1000 * 60 * 60 * 24))
  if (dayDiff >= 1) {
    lastSessionMemory.value = last
    crossDayHintVisible.value = true
  }
}
function dismissCrossDayHint() {
  crossDayHintVisible.value = false
}

// 阶段 6 减法:完成反思 — 完成 toast 弹"感觉怎么样"小卡片,1 步标 tag。
// 设计原则:不打扰收尾的爽感,小卡片 6 秒自动消失,用户可点 3 个 chip 中的一个
// (顺手/学到/划水)做 1 步打标,也可以完全忽略。tag 存到 completion_feeling,
// review 弹窗会聚合"完成时你的感受分布" — 让用户看见"我做完时感觉是…"
const COMPLETION_FEELINGS = [
  { key: 'smooth', icon: '🪶', label: '顺手', desc: '没费什么劲就做完了' },
  { key: 'learned', icon: '💡', label: '学到', desc: '做完反而有收获' },
  { key: 'rough', icon: '😐', label: '划水', desc: '勉强过,凑合收尾' }
]
const completionFeedbackVisible = ref(false)
const completionFeedbackTask = ref(null)
let completionFeedbackTimer = null

function showCompletionFeedback(task) {
  if (!task || quietMode.value) return
  completionFeedbackTask.value = task
  completionFeedbackVisible.value = true
  if (completionFeedbackTimer) clearTimeout(completionFeedbackTimer)
  // 6 秒后自动消失 — 给"顺手/划水"足够长的反应窗口,但不滞留
  completionFeedbackTimer = setTimeout(() => {
    dismissCompletionFeedback()
  }, 6000)
}
function dismissCompletionFeedback() {
  completionFeedbackVisible.value = false
  completionFeedbackTask.value = null
  if (completionFeedbackTimer) {
    clearTimeout(completionFeedbackTimer)
    completionFeedbackTimer = null
  }
}
async function setCompletionFeeling(feeling) {
  const task = completionFeedbackTask.value
  dismissCompletionFeedback()
  if (!task || !feeling) return
  // 1 步更新 — 不再走 updateTask 是因为 updateTask 会触发 toast
  // (这里不想要第二个 toast 干扰刚出现的完成反馈)
  try {
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${task.id}`, {
      method: 'PUT',
      body: JSON.stringify({ completion_feeling: feeling })
    })
    const meta = COMPLETION_FEELINGS.find((f) => f.key === feeling)
    if (meta && !quietMode.value) {
      ElMessage({
        message: `${meta.icon} 已记录:${meta.label}`,
        type: 'success',
        duration: 1600
      })
    }
    // 局部更新 board 中的 task 字段(无需 refresh 全表)
    const source = findTaskById(task.id)
    if (source) source.completion_feeling = feeling
  } catch (e) {
    if (e?.message) ElMessage.error(e.message || '记录感受失败')
  }
}
const timeGreeting = computed(() => {
  const h = new Date().getHours()
  if (h < 5) return { icon: '🌙', text: '夜深了' }
  if (h < 11) return { icon: '☀️', text: '早上好' }
  if (h < 13) return { icon: '🌤️', text: '中午好' }
  if (h < 18) return { icon: '⛅', text: '下午好' }
  if (h < 22) return { icon: '🌆', text: '晚上好' }
  return { icon: '🌙', text: '夜深了' }
})
const crossDayIcon = computed(() => {
  const last = lastSessionMemory.value
  if (!last) return '📅'
  const dayDiff = Math.floor((Date.now() - new Date(last.at).getTime()) / (1000 * 60 * 60 * 24))
  if (dayDiff >= 7) return '🌙'
  if (dayDiff >= 3) return '👋'
  return '📅'
})
const crossDayHeadline = computed(() => {
  const last = lastSessionMemory.value
  if (!last) return ''
  const dayDiff = Math.floor((Date.now() - new Date(last.at).getTime()) / (1000 * 60 * 60 * 24))
  if (dayDiff === 1) return '欢迎回来'
  if (dayDiff <= 3) return `${dayDiff} 天没见`
  if (dayDiff <= 6) return '快一周了'
  if (dayDiff <= 13) return '两周没见'
  return '好久不见'
})
const crossDayBody = computed(() => {
  const last = lastSessionMemory.value
  if (!last) return ''
  const lastDate = new Date(last.at)
  const weekday = ['日', '一', '二', '三', '四', '五', '六'][lastDate.getDay()]
  const dateStr = `${lastDate.getMonth() + 1}月${lastDate.getDate()}日(周${weekday})`
  const doneCount = last.done_count || 0
  if (doneCount === 0) return `上次 ${dateStr} 打开了,但没做完什么。慢慢来。`
  if (doneCount === 1) return `上次 ${dateStr} 完成 1 件,今天继续。`
  return `上次 ${dateStr} 完成 ${doneCount} 件,今天继续。`
})

function applyQuietModeClass() {
  if (typeof document === 'undefined') return
  const root = document.querySelector('.planner-app-root') || document.body
  if (quietMode.value) root.classList.add('planner-quiet')
  else root.classList.remove('planner-quiet')
}

function toggleQuietMode(value) {
  quietMode.value = !!value
  try {
    localStorage.setItem('planner_quiet_mode', quietMode.value ? '1' : '0')
  } catch (e) { /* localStorage may be disabled */ }
  applyQuietModeClass()
  if (quietMode.value) {
    ElMessage({
      message: '已开启安静模式:不再播放完成动画和推荐下一项提示',
      type: 'success',
      duration: 2400
    })
  }
}

if (typeof window !== 'undefined') {
  // 启动时同步 DOM 类,避免首屏先闪一下动画
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', applyQuietModeClass, { once: true })
  } else {
    applyQuietModeClass()
  }
}
const focusSession = ref(null)
const focusElapsedMs = ref(0)
const focusLog = ref({})
let focusTickTimer = null
const searchKeyword = ref('')
const serverSearchResults = ref([])
const serverSearchLoading = ref(false)
let serverSearchTimer = null
let serverSearchSeq = 0
const searchOpen = ref(false)
const searchSelectedIndex = ref(0)
const globalSearchInput = ref(null)
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

// 阶段 5 减法 (iter 32):主页 4 metric cards 全 0 检测
// 当 未完成 + 收件箱 + 事件 + 顺延 全是 0 时,合并为 1 个"今天清爽"卡
const allMetricsZero = computed(() => {
  return openCount.value === 0
    && (board.value.inbox_items?.length || 0) === 0
    && (board.value.counts.event_open || 0) === 0
    && (board.value.counts.rolled_over || 0) === 0
})
const urgentOpenCount = computed(() => {
  const groups = board.value.groups || []
  let count = 0
  for (const g of groups) {
    for (const t of (g.items || [])) {
      if (t.priority === 'urgent' && t.status !== 'done' && t.status !== 'cancelled') count++
    }
  }
  for (const t of (board.value.event_groups || []).flatMap(g => g.items || [])) {
    if (t.priority === 'urgent' && t.status !== 'done' && t.status !== 'cancelled') count++
  }
  for (const t of (board.value.inbox_items || [])) {
    if (t.priority === 'urgent' && t.status !== 'done' && t.status !== 'cancelled') count++
  }
  return count
})
// 哲学"1 步交互":hero 提示变成可点击的入口,用户读到"先分类"即可直接开始
function goInboxOrTriage() {
  const count = board.value.inbox_items?.length || 0
  if (count > 0) {
    activeView.value = 'inbox'
    startTriage()
  } else if (!quietMode.value) {
    // 哲学"情绪反弹"对策:收件箱为空是好事,给用户一个小确幸
    ElMessage({ message: '收件箱已清空,做得好。', type: 'success', duration: 2000 })
  }
}
function goRolledOver() {
  // 复用 timeline 过滤:只显示 is_rolled_over 的事项
  activeView.value = 'timeline'
  setPriorityFilter('overdue')
}
function goOpenTasks() {
  if (openCount.value === 0) {
    if (!quietMode.value) {
      ElMessage({ message: '今天没有未完成事项,给自己留点空。', type: 'success', duration: 2000 })
    }
    return
  }
  activeView.value = 'timeline'
  priorityFilter.value = 'all'
}

const heroHintAction = computed(() => {
  const inboxCount = board.value.inbox_items?.length || 0
  const needsTrim = board.value.focus?.needs_trim
  const rolledOver = board.value.counts?.rolled_over || 0
  // 优先级:收件箱 > 时间线过载 > 顺延堆积
  if (inboxCount > 0) {
    return {
      label: `现在分类 ${inboxCount} 条收件箱`,
      icon: 'Files',
      primary: true,
      onClick: () => { activeView.value = 'inbox'; startTriage() }
    }
  }
  if (needsTrim) {
    return {
      label: '去收敛时间线',
      icon: 'Refresh',
      primary: false,
      onClick: () => { activeView.value = 'timeline' }
    }
  }
  if (rolledOver >= 3) {
    return {
      label: `看看 ${rolledOver} 个顺延的事项`,
      icon: 'Refresh',
      primary: false,
      onClick: () => { activeView.value = 'timeline' }
    }
  }
  return null
})
const hasCreatorPrivileges = computed(() => Boolean(creatorKey.value))
const showInstallEntry = computed(() => !isStandaloneMode.value)
const primaryFocusTask = computed(() => board.value.focus?.primary || null)
const secondaryFocusTasks = computed(() => board.value.focus?.secondary || [])
const nextEventTask = computed(() => board.value.focus?.next_event || null)

const lowEnergyFocusTask = computed(() => {
  if (!lowEnergyMode.value) return null
  // 找事务性/未标的开放任务,跳过需专注/外出/创意的
  for (const g of board.value.groups || []) {
    for (const t of (g.items || [])) {
      if (t.status === 'done' || t.status === 'cancelled') continue
      if (!t.energy_level || t.energy_level === 'shallow') return t
    }
  }
  return null
})
const displayFocusTask = computed(() => {
  if (activePinnedFocus.value) return null  // 主动聚焦的优先
  if (lowEnergyMode.value) return lowEnergyFocusTask.value || primaryFocusTask.value
  return primaryFocusTask.value
})

// 阶段 5 减法 (iter 31):当前显示的 focus task(给主区域 1 步 = openTask 用)
const currentFocusTask = computed(() => {
  if (activePinnedFocus.value) return activePinnedFocus.value.primary
  return displayFocusTask.value
})
const todayAllDone = computed(() => {
  // 没有要做的:既没聚焦也没自动建议 + 今天至少完成 1 件
  if (activePinnedFocus.value) return false
  if (displayFocusTask.value) return false
  const done = board.value.recovery?.done_today || 0
  return done >= 1
})

// 阶段 5 减法 (iter 33):focus card 空状态 1 步置入候选
// 当 focus card 没有 current focus 时,从今天/明天/收件箱 聚合 top 3-5 候选
// 排除已 pinned / done / cancelled,按优先级 + 创建时间 排序
// 用户 1 步点击 → togglePin → focus card 立刻显示新 focus
const focusCandidateTasks = computed(() => {
  if (currentFocusTask.value || todayAllDone.value) return []
  const today = new Date().toISOString().slice(0, 10)
  const tomorrow = new Date(Date.now() + 86400000).toISOString().slice(0, 10)
  const priorityRank = { urgent: 0, high: 1, medium: 2, low: 3 }
  const pool = []
  const seen = new Set()
  // 1) 今天 + 明天 planned (timeline groups)
  for (const g of (board.value.groups || [])) {
    for (const t of (g.items || [])) {
      if (seen.has(t.id)) continue
      seen.add(t.id)
      if (t.status === 'done' || t.status === 'cancelled') continue
      if (pinnedTaskIds.value.has(t.id)) continue
      if (t.planned_for !== today && t.planned_for !== tomorrow) continue
      pool.push(t)
    }
  }
  // 2) 收件箱 (inbox)
  for (const t of (board.value.inbox_items || [])) {
    if (seen.has(t.id)) continue
    seen.add(t.id)
    if (t.status === 'done' || t.status === 'cancelled') continue
    if (pinnedTaskIds.value.has(t.id)) continue
    pool.push(t)
  }
  // 排序:优先级 asc,创建时间 asc(越久越靠前)
  pool.sort((a, b) => {
    const pa = priorityRank[a.priority] ?? 9
    const pb = priorityRank[b.priority] ?? 9
    if (pa !== pb) return pa - pb
    return (a.created_at || '').localeCompare(b.created_at || '')
  })
  return pool.slice(0, 5)
})

// 阶段 5 减法:今日 open + in_progress 数量(为"收尾今天"chip 提供数据)
// 只统计 planned_for = 今天的,事件不算(事件自带时间感)
const todayOpenCount = computed(() => {
  const today = new Date().toISOString().slice(0, 10)
  let count = 0
  for (const g of (board.value.groups || [])) {
    for (const t of (g.items || [])) {
      if ((t.status === 'open' || t.status === 'in_progress') && (t.planned_for || '') === today) {
        count++
      }
    }
  }
  return count
})

// 阶段 5 减法 (iter 29):今日总览数 = open + done,用于进度可视化
const todayTotalCount = computed(() => todayOpenCount.value + (board.value.recovery?.done_today || 0))
// 完成度 0~100,无今日事项时 100(空就等于满)
const todayProgressPercent = computed(() => {
  const total = todayTotalCount.value
  if (total === 0) return 100
  return Math.round((board.value.recovery?.done_today || 0) / total * 100)
})

// 阶段 5 减法 (iter 34):当前 focus.secondary 列表(合并 activePinned 和 system 两条来源)
// 用于"收尾 N 个次要"1 步批量完成按钮
const currentSecondaryList = computed(() => {
  if (activePinnedFocus.value) return activePinnedFocus.value.secondary || []
  return secondaryFocusTasks.value || []
})

const pinnedTaskIds = ref(new Set())
const isPinned = (taskId) => pinnedTaskIds.value.has(taskId)
function focusPinnedStorageKey() {
  return profileId.value ? `planner_focus_pinned_${profileId.value}` : ''
}
function completionLogStorageKey() {
  return profileId.value ? `planner_completion_log_${profileId.value}` : ''
}
function loadCompletionLog() {
  const key = completionLogStorageKey()
  if (!key) {
    completionLog.value = {}
    return
  }
  try {
    const raw = localStorage.getItem(key)
    completionLog.value = raw ? JSON.parse(raw) : {}
  } catch (error) {
    completionLog.value = {}
  }
}
function saveCompletionLog() {
  const key = completionLogStorageKey()
  if (!key) return
  const trimmed = {}
  const cutoff = Date.now() - 1000 * 60 * 60 * 24 * 45
  Object.entries(completionLog.value || {}).forEach(([date, count]) => {
    if (!date || !count) return
    if (new Date(date + 'T00:00:00').getTime() < cutoff) return
    trimmed[date] = count
  })
  completionLog.value = trimmed
  localStorage.setItem(key, JSON.stringify(trimmed))
}

// 任务模板(快捷短语):localStorage 存最近 8 条
const quickTemplates = ref([])
const QUICK_TEMPLATES_KEY = 'planner_quick_templates_v1'
const QUICK_TEMPLATES_MAX = 8

function loadQuickTemplates() {
  try {
    const raw = localStorage.getItem(QUICK_TEMPLATES_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    quickTemplates.value = Array.isArray(parsed) ? parsed.slice(0, QUICK_TEMPLATES_MAX) : []
  } catch (e) {
    quickTemplates.value = []
  }
}

function saveQuickTemplates() {
  try {
    localStorage.setItem(QUICK_TEMPLATES_KEY, JSON.stringify(quickTemplates.value.slice(0, QUICK_TEMPLATES_MAX)))
  } catch (e) { /* 忽略 */ }
}

// === 套路(多任务模板):localStorage 存 N 个套路,每个套路含若干项 ===
const routines = ref([])
const routineDialogVisible = ref(false)
const routinePopoverVisible = ref(false)
const editingRoutine = ref(null) // { id, name, items: [{ title, priority, kind, entryType, bucket, offsetDays }] }

function routineStorageKey() {
  return `planner_routines_${profileId.value || 'default'}`
}

function loadRoutines() {
  try {
    const raw = localStorage.getItem(routineStorageKey())
    const parsed = raw ? JSON.parse(raw) : []
    routines.value = Array.isArray(parsed) ? parsed : []
  } catch (e) {
    routines.value = []
  }
}

function saveRoutines() {
  try {
    localStorage.setItem(routineStorageKey(), JSON.stringify(routines.value))
  } catch (e) { /* 忽略 */ }
}

function newRoutineDraft() {
  return {
    id: 'r_' + Math.random().toString(36).slice(2, 10),
    name: '',
    items: []
  }
}

function openNewRoutineDialog() {
  editingRoutine.value = newRoutineDraft()
  routinePopoverVisible.value = false
  routineDialogVisible.value = true
}

function openEditRoutineDialog(r) {
  editingRoutine.value = JSON.parse(JSON.stringify(r))
  routinePopoverVisible.value = false
  routineDialogVisible.value = true
}

function addItemToEditingRoutine() {
  if (!editingRoutine.value) return
  editingRoutine.value.items.push({
    title: '',
    priority: 'medium',
    kind: activeKind.value,
    entryType: 'task',
    bucket: 'planned',
    offsetDays: 0
  })
}

function removeItemFromEditingRoutine(idx) {
  if (!editingRoutine.value) return
  editingRoutine.value.items.splice(idx, 1)
}

function moveEditingRoutineItem(idx, delta) {
  if (!editingRoutine.value) return
  const items = editingRoutine.value.items
  const next = idx + delta
  if (next < 0 || next >= items.length) return
  const [it] = items.splice(idx, 1)
  items.splice(next, 0, it)
}

function persistEditingRoutine() {
  if (!editingRoutine.value) return
  const r = editingRoutine.value
  const validItems = (r.items || []).filter(it => (it.title || '').trim())
  if (!r.name.trim() || validItems.length === 0) {
    ElMessage.warning('套路名和至少 1 个任务项不能为空')
    return
  }
  r.name = r.name.trim()
  r.items = validItems.map(it => ({
    title: it.title.trim(),
    priority: it.priority || 'medium',
    kind: it.kind || activeKind.value,
    entryType: it.entryType || 'task',
    bucket: it.bucket || 'planned',
    offsetDays: Number(it.offsetDays || 0)
  }))
  const idx = routines.value.findIndex(x => x.id === r.id)
  if (idx >= 0) routines.value[idx] = r
  else routines.value.push(r)
  saveRoutines()
  routineDialogVisible.value = false
  ElMessage.success(idx >= 0 ? '套路已更新' : '套路已保存')
}

function deleteRoutine(r) {
  if (!r || !r.id) return
  routines.value = routines.value.filter(x => x.id !== r.id)
  saveRoutines()
}

async function applyRoutine(r) {
  if (!r || !r.items || r.items.length === 0) return
  routinePopoverVisible.value = false
  const validItems = r.items.filter(it => (it.title || '').trim())
  if (validItems.length === 0) return
  const base = new Date()
  const baseDate = new Date(base.getFullYear(), base.getMonth(), base.getDate())
  const tasks = validItems.map((it, idx) => {
    const offset = Number(it.offsetDays || 0)
    const d = new Date(baseDate)
    d.setDate(d.getDate() + offset)
    const dateStr = d.toISOString().slice(0, 10)
    return {
      kind: it.kind || activeKind.value,
      entry_type: it.entryType || 'task',
      bucket: it.bucket || 'planned',
      title: it.title,
      priority: it.priority || 'medium',
      status: 'open',
      planned_for: dateStr
    }
  })
  try {
    const res = await fetch(`${API_BASE}/profile/${profileId.value}/tasks/batch?creator_key=${encodeURIComponent(creatorKey.value)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tasks })
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `应用套路失败: HTTP ${res.status}`)
    }
    ElMessage.success(`已应用套路「${r.name}」, 创建 ${tasks.length} 个任务`)
    await refreshBoard()
  } catch (error) {
    ElMessage.error(error.message || '应用套路失败')
  }
}

// 用 title+priority+kind+bucket+entryType 五元组做模板签名,避免重复保存。
function quickTemplateSignature(t) {
  return [t.title || '', t.priority || 'medium', t.kind || 'work', t.bucket || 'inbox', t.entryType || 'task'].join('|')
}

function pushQuickTemplate(t) {
  if (!t || !t.title) return
  const sig = quickTemplateSignature(t)
  const filtered = quickTemplates.value.filter(x => quickTemplateSignature(x) !== sig)
  filtered.unshift({ ...t, used_at: Date.now() })
  quickTemplates.value = filtered.slice(0, QUICK_TEMPLATES_MAX)
  saveQuickTemplates()
}

function applyQuickTemplate(t) {
  if (!t) return
  quickForm.title = t.title || ''
  quickForm.priority = t.priority || 'medium'
  if (t.kind) activeKind.value = t.kind
  if (t.entryType) quickForm.entryType = t.entryType
  if (t.bucket && t.entryType !== 'event') quickForm.bucket = t.bucket
  // 把模板放到最前(记录使用频次)
  pushQuickTemplate(t)
  // 聚焦标题框,让用户继续编辑
  nextTick(() => {
    const el = document.querySelector('.quick-grid-core input')
    if (el) {
      el.focus()
      const len = (quickForm.title || '').length
      try { el.setSelectionRange(len, len) } catch (e) { /* ignore */ }
    }
  })
}

// 阶段 5 减法 (iter 27):常用模板 1 步应用 — 点击直接创建 + 5s 撤销兜底。
// 适用场景:模板就是 verbatim 用(买菜/接娃/写日报),不再绕到表单再点保存。
// 原编辑流(填表 → 改)移到 ✎ 图标,保持"想编辑"和"想直接做"两路并行。
async function applyQuickTemplateDirect(t) {
  if (!t || !t.title) return
  const title = (t.title || '').trim()
  if (!title) return
  const kind = t.kind || 'work'
  const entryType = t.entryType || 'task'
  const bucket = entryType === 'event' ? 'planned' : (t.bucket || 'planned')
  // planned → 默认挂今天;inbox/someday → 不带日期(让"我今天该做什么"不被污染)
  const today = new Date().toISOString().slice(0, 10)
  const plannedFor = bucket === 'planned' ? today : ''
  const payload = {
    kind,
    entry_type: entryType,
    bucket,
    title,
    detail: '',
    raw_text: title,
    planned_for: plannedFor,
    remind_at: '',
    repeat_type: 'none',
    repeat_interval: 1,
    repeat_until: '',
    priority: t.priority || 'medium',
    notify_email: '',
    intent: '',
    energy_level: ''
  }
  let createdId = ''
  try {
    const resp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks`, {
      method: 'POST',
      body: JSON.stringify(payload)
    })
    const data = await resp.json().catch(() => ({}))
    createdId = data?.task?.id || data?.id || ''
    if (!createdId) throw new Error('未拿到新建任务 id')
    // 切到对应 kind 标签,让用户看到这条新加的事项
    activeKind.value = kind
    // 用频次靠前
    pushQuickTemplate(t)
    await refreshBoard()
    const kindLabel = kind === 'work' ? '工作' : '生活'
    const where = bucket === 'inbox'
      ? ' · 收件箱'
      : bucket === 'someday'
        ? ' · 放一放'
        : (plannedFor ? ` · 计划 ${plannedFor}` : '')
    const onUndo = async () => {
      if (!createdId) return
      try {
        await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${createdId}`, {
          method: 'DELETE'
        })
        await refreshBoard()
      } catch (e) { /* 撤销失败静默 */ }
    }
    showUndoSnackbar({
      kind: 'task',
      message: `已创建: ${title} (${kindLabel})${where}`,
      onUndo
    })
  } catch (error) {
    // 失败时回滚频次,避免错点把模板顶到首位
    ElMessage.error(error.message || '应用模板失败')
  }
}

function removeQuickTemplate(t) {
  const sig = quickTemplateSignature(t)
  quickTemplates.value = quickTemplates.value.filter(x => quickTemplateSignature(x) !== sig)
  saveQuickTemplates()
}

function saveCurrentAsTemplate() {
  const title = (quickForm.title || '').trim()
  if (!title) {
    ElMessage.warning('先写一个标题再保存为模板')
    return
  }
  pushQuickTemplate({
    title,
    priority: quickForm.priority || 'medium',
    kind: activeKind.value,
    entryType: quickForm.entryType || 'task',
    bucket: quickForm.bucket || 'inbox'
  })
  ElMessage.success('已保存为快捷短语')
}

loadQuickTemplates()

// 自然语言日期输入:状态 + 应用函数
const quickNaturalDate = ref('')
const quickNaturalDateHint = ref('✨ 也支持自然语言: 明天下午3点 / 3天后 / 下周一 / 12月15日')
const taskNaturalDate = ref('')
const taskNaturalDateHint = ref('✨ 也支持自然语言: 明天下午3点 / 3天后 / 下周一 / 12月15日')

function _previewNaturalDate(input) {
  if (!input || !input.trim()) return null
  return parseNaturalDate(input.trim())
}

// ===== 智能归类:从一段文字里提取 日期/时间/优先级/bucket/entryType =====
// 哲学"原则 2:让系统判断,用户只选择"。
// 用户在标题里写"明天下午3点前 给客户发合同 紧急",系统一次识别完,用户不需要开高级设置。
//
// 返回值(可能全空):
//   {
//     date: 'YYYY-MM-DD' | null,   // 解析出的日期
//     time: 'HH:mm' | null,        // 解析出的具体时间
//     priority: 'urgent'|'high'|'medium'|'low' | null,
//     bucket: 'planned'|'inbox'|'someday' | null,
//     entryType: 'task'|'event' | null,
//     hint: string,                // 给用户看的中文提示
//     cleanText: string            // 去掉日期/优先级关键词后的干净文本(可选)
//   }
function extractHintsFromText(rawText) {
  const out = { date: null, time: null, priority: null, bucket: null, entryType: null, hint: '', cleanText: null }
  if (!rawText || !rawText.trim()) return out
  const text = rawText.trim()

  // 1) 日期 / 时间(用现有 parseNaturalDate)
  const nd = parseNaturalDate(text)
  if (nd) {
    out.date = nd.date
    out.time = nd.time || null
  }

  // 2) 优先级关键词(全角/半角 空格都不影响)
  //    顺序很重要:先匹配"紧急"再匹配"高"
  const priorityMatchers = [
    { key: 'urgent',  pattern: /(🚨|!!|!|紧急|急|十万火急|非常紧急)/ },
    { key: 'high',    pattern: /(重要|高优[先级]?|高|优先)/ },
    { key: 'low',     pattern: /(不急|低优[先级]?|低优先级|有空再说|缓一缓)/ },
    { key: 'medium',  pattern: /(普通|一般)/ }
  ]
  for (const m of priorityMatchers) {
    if (m.pattern.test(text)) {
      out.priority = m.key
      break
    }
  }

  // 3) "改天/以后再说" → someday
  if (/(改天|以后|以后再说|回头|不急|有空再|再说|下次|哪天有空)/.test(text)) {
    out.bucket = 'someday'
  }

  // 4) entryType:有具体时间 → event;否则 task
  if (out.time) {
    out.entryType = 'event'
  } else {
    out.entryType = 'task'
  }

  // 5) bucket:有日期 → planned(除非上面已经标了 someday);纯文字无日期 → inbox
  if (!out.bucket) {
    out.bucket = out.date ? 'planned' : 'inbox'
  }

  // 6) 拼中文提示(给用户看)
  const parts = []
  if (out.date) parts.push(out.time ? `${out.date} ${out.time}` : out.date)
  const priorityLabelMap = { urgent: '🚨 紧急', high: '高优先级', medium: '中优先级', low: '低优先级' }
  if (out.priority) parts.push(priorityLabelMap[out.priority])
  const bucketLabelMap = { planned: '计划中', inbox: '收件箱', someday: '放一放' }
  if (out.bucket) parts.push(bucketLabelMap[out.bucket])
  if (out.entryType === 'event') parts.push('事件')
  out.hint = parts.join(' · ')

  return out
}

function onQuickNaturalDateInput() {
  const r = _previewNaturalDate(quickNaturalDate.value)
  if (!r) {
    quickNaturalDateHint.value = '✨ 也支持自然语言: 明天下午3点 / 3天后 / 下周一 / 12月15日'
    return
  }
  const timePart = r.time ? ` ${r.time}` : ''
  quickNaturalDateHint.value = `✓ 已识别: ${r.date}${timePart}`
}

function onTaskNaturalDateInput() {
  const r = _previewNaturalDate(taskNaturalDate.value)
  if (!r) {
    taskNaturalDateHint.value = '✨ 也支持自然语言: 明天下午3点 / 3天后 / 下周一 / 12月15日'
    return
  }
  const timePart = r.time ? ` ${r.time}` : ''
  taskNaturalDateHint.value = `✓ 已识别: ${r.date}${timePart}`
}

function openGlobalQuickAdd() {
  if (!profileId.value) return
  globalQuickAddText.value = ''
  globalQuickAddPriority.value = 'medium'
  globalQuickAddEntryType.value = 'task'
  globalQuickAddPreview.value = null
  globalQuickAddVisible.value = true
  nextTick(() => {
    if (globalQuickAddInput.value && globalQuickAddInput.value.focus) {
      globalQuickAddInput.value.focus()
    }
  })
}

function closeGlobalQuickAdd() {
  if (globalQuickAddSaving.value) return
  globalQuickAddVisible.value = false
}

function onGlobalQuickAddInput() {
  const text = globalQuickAddText.value || ''
  // 用统一的 extractHintsFromText,涵盖日期+时间+优先级+bucket
  const h = extractHintsFromText(text)
  if (h.date || h.priority || h.bucket) {
    globalQuickAddPreview.value = {
      date: h.date,
      time: h.time,
      priority: h.priority,
      bucket: h.bucket,
      entryType: h.entryType,
      label: h.hint
    }
  } else {
    globalQuickAddPreview.value = null
  }
}

async function submitGlobalQuickAdd() {
  if (globalQuickAddSaving.value) return
  const text = (globalQuickAddText.value || '').trim()
  if (!text) {
    ElMessage.warning('先写一件事')
    return
  }
  let title = text
  let plannedFor = new Date().toISOString().slice(0, 10)
  let remindAt = ''
  // 用统一的 extractHintsFromText 一次拿全
  const hints = extractHintsFromText(text)
  let entryType = globalQuickAddEntryType.value
  let bucket = globalQuickAddEntryType.value === 'event' ? 'planned' : 'planned'
  let priority = globalQuickAddPriority.value
  if (hints.date) {
    // 把识别到的日期/时间/优先级从 title 中剥离,让标题更干净
    title = text
      .replace(/\s*(今天|明天|后天|大后天|下周[一星]?[一二三四五六日天]?|周[一二三四五六日天]|上午|下午|早上|晚上|傍晚|夜里|夜间|am|pm|\d{1,2}[:点]\d{0,2}|\d{1,2}点\d{0,2}分?|\d+天[之以]?后|\d+周[之以]?后|\d+个?月[之以]?后|\d{1,2}月\d{1,2}[日号]?|\d{4}[-/.]\d{1,2}[-/.]\d{1,2}|\d{1,2}[\/\-]\d{1,2})\s*/gi, ' ')
      .replace(/\s*(🚨|!!|!|紧急|十万火急|非常紧急|重要|高优[先级]?|低优[先级]?|不急|低优先级|有空再说|缓一缓|改天|以后再说|回头)\s*/gi, ' ')
      .replace(/\s+/g, ' ')
      .trim() || text
    plannedFor = hints.date
    if (hints.time) remindAt = `${hints.date}T${hints.time}`
    else remindAt = `${hints.date}T09:00`
  }
  // entryType/bucket/priority 智能归类
  if (hints.entryType) entryType = hints.entryType
  if (hints.bucket) bucket = hints.bucket
  if (hints.priority && globalQuickAddPriority.value === 'medium') {
    // 默认 medium,只在用户没主动改过其他选项时升级
    priority = hints.priority
  }
  globalQuickAddSaving.value = true
  try {
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks`, {
      method: 'POST',
      body: JSON.stringify({
        kind: activeKind.value,
        entry_type: entryType,
        bucket: entryType === 'event' ? 'planned' : bucket,
        title,
        detail: '',
        planned_for: plannedFor,
        remind_at: remindAt,
        priority,
        raw_text: text
      })
    })
    ElMessage.success({
      message: `已记录: ${title}`,
      duration: 2000
    })
    // 模板记忆:把这次输入存为快捷模板
    pushQuickTemplate({
      title,
      priority: globalQuickAddPriority.value,
      kind: activeKind.value,
      entryType: globalQuickAddEntryType.value,
      bucket: 'planned'
    })
    globalQuickAddVisible.value = false
    await refreshBoard()
  } catch (error) {
    ElMessage.error(error.message || '保存失败')
  } finally {
    globalQuickAddSaving.value = false
  }
}

function downloadBlob(content, filename, mimeType) {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

async function exportAllDataJSON() {
  if (!profileId.value || !creatorKey.value) {
    ElMessage.warning('请先登录档案')
    return
  }
  exportingJSON.value = true
  try {
    const headers = await plannerFetch(`${API_BASE}/profile/${profileId.value}?creator_key=${creatorKey.value}`)
    if (!headers.ok) throw new Error(`读取档案失败: HTTP ${headers.status}`)
    const profile = await headers.json()

    const timelineResp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/timeline?creator_key=${creatorKey.value}`)
    if (!timelineResp.ok) throw new Error(`读取事项失败: HTTP ${timelineResp.status}`)
    const timelineData = await timelineResp.json()
    const boardData = timelineData.board || {}
    // 从 board 五个分组聚合去重
    const seen = new Set()
    const tasks = []
    const collect = (item) => {
      const real = item && (item.planner_task || item.PlannerTask || item)
      if (!real || !real.id || seen.has(real.id)) return
      seen.add(real.id)
      tasks.push(real)
    }
    ;(boardData.groups || []).forEach(g => (g.items || []).forEach(collect))
    ;(boardData.event_groups || []).forEach(g => (g.items || []).forEach(collect))
    ;(boardData.inbox_items || []).forEach(collect)
    ;(boardData.someday_items || []).forEach(collect)
    ;(boardData.recent_items || []).forEach(collect)

    const meetingsResp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/meetings?creator_key=${creatorKey.value}`)
    const meetings = meetingsResp.ok ? await meetingsResp.json() : { meetings: [] }

    const payload = {
      exported_at: new Date().toISOString(),
      schema_version: 1,
      profile: {
        id: profile.id,
        name: profile.name,
        notify_email: profile.notify_email,
        created_at: profile.created_at,
      },
      tasks,
      meetings: meetings.meetings || [],
    }
    const filename = `planner-${profile.name || profileId.value}-${new Date().toISOString().slice(0, 10)}.json`
    downloadBlob(JSON.stringify(payload, null, 2), filename, 'application/json')
    ElMessage.success({ message: `已导出 ${payload.tasks.length} 个事项 + ${payload.meetings.length} 个会议`, duration: 2500 })
  } catch (error) {
    ElMessage.error(error.message || '导出失败')
  } finally {
    exportingJSON.value = false
  }
}

function csvEscape(value) {
  const s = (value ?? '').toString()
  if (/[",\n\r]/.test(s)) {
    return '"' + s.replace(/"/g, '""') + '"'
  }
  return s
}

async function exportTasksCSV() {
  if (!profileId.value || !creatorKey.value) {
    ElMessage.warning('请先登录档案')
    return
  }
  exportingCSV.value = true
  try {
    const resp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/timeline?creator_key=${creatorKey.value}`)
    if (!resp.ok) throw new Error(`读取事项失败: HTTP ${resp.status}`)
    const data = await resp.json()
    const boardData = data.board || {}
    const seen = new Set()
    const tasks = []
    const collect = (item) => {
      const real = item && (item.planner_task || item.PlannerTask || item)
      if (!real || !real.id || seen.has(real.id)) return
      seen.add(real.id)
      tasks.push(real)
    }
    ;(boardData.groups || []).forEach(g => (g.items || []).forEach(collect))
    ;(boardData.event_groups || []).forEach(g => (g.items || []).forEach(collect))
    ;(boardData.inbox_items || []).forEach(collect)
    ;(boardData.someday_items || []).forEach(collect)
    ;(boardData.recent_items || []).forEach(collect)

    const headers = ['标题', '状态', '优先级', '类型', '分类', '计划日期', '提醒时间', '是否重复', '备注', '创建时间']
    const rows = tasks.map(t => [
      t.title,
      statusLabel(t.status),
      priorityLabel(t.priority),
      t.entry_type === 'event' ? '事件' : '任务',
      t.kind === 'work' ? '工作' : '生活',
      t.planned_for || '',
      t.remind_at ? String(t.remind_at).slice(0, 16).replace('T', ' ') : '',
      t.repeat_type && t.repeat_type !== 'none' ? `${t.repeat_type}${t.repeat_interval ? `/${t.repeat_interval}` : ''}` : '',
      (t.notes || t.detail || '').replace(/\n/g, ' '),
      t.created_at ? String(t.created_at).slice(0, 16).replace('T', ' ') : '',
    ])
    const csv = [headers, ...rows].map(row => row.map(csvEscape).join(',')).join('\n')
    const bom = '﻿'
    const filename = `planner-tasks-${new Date().toISOString().slice(0, 10)}.csv`
    downloadBlob(bom + csv, filename, 'text/csv;charset=utf-8')
    ElMessage.success({ message: `已导出 ${tasks.length} 行 CSV`, duration: 2500 })
  } catch (error) {
    ElMessage.error(error.message || '导出失败')
  } finally {
    exportingCSV.value = false
  }
}

// ===== JSON 备份导入 =====
// 哲学"数据平滑迁移":只追加,不覆盖/不删除。
// 每个导入的事项都会得到新 ID,和现有数据并存,零破坏。

function triggerImportPicker() {
  if (!profileId.value || !creatorKey.value) {
    ElMessage.warning('请先登录档案')
    return
  }
  const input = importFileInput.value
  if (input) input.click()
}

async function onImportFileChange(event) {
  const file = event.target.files && event.target.files[0]
  // 重置 input,允许选同一个文件
  event.target.value = ''
  if (!file) return
  if (file.size > 50 * 1024 * 1024) {
    ElMessage.error('文件太大 (> 50MB),请确认是 planner 导出的备份')
    return
  }
  let text
  try {
    text = await file.text()
  } catch (e) {
    ElMessage.error('读取文件失败: ' + (e.message || e))
    return
  }
  let payload
  try {
    payload = JSON.parse(text)
  } catch (e) {
    ElMessage.error('文件不是有效的 JSON')
    return
  }
  if (!payload || typeof payload !== 'object') {
    ElMessage.error('备份格式不对,应该是导出的 JSON 备份文件')
    return
  }
  if (payload.schema_version !== 1) {
    ElMessage.error(`备份版本不匹配 (期望 v1,实际 v${payload.schema_version || '?'}),无法导入`)
    return
  }
  const tasks = Array.isArray(payload.tasks) ? payload.tasks : []
  const meetings = Array.isArray(payload.meetings) ? payload.meetings : []
  if (tasks.length === 0 && meetings.length === 0) {
    ElMessage.warning('备份里没有任何事项或会议')
    return
  }
  // 确认对话框
  try {
    await ElMessageBox.confirm(
      `将从「${payload.profile?.name || '未知档案'}」备份中导入\n` +
      `· ${tasks.length} 个事项\n` +
      `· ${meetings.length} 个会议\n\n` +
      `已有数据不会被覆盖,每个导入的事项会得到新 ID。\n` +
      `继续?`,
      '确认导入',
      { type: 'warning', confirmButtonText: '开始导入', cancelButtonText: '取消' }
    )
  } catch {
    return  // 用户取消
  }
  await runImport(payload)
}

// ===== ICS 日历订阅(阶段 4:数据主权) =====
// 哲学:数据要能流出去到用户真正每天用的工具(macOS Calendar / iPhone / Google Calendar)
// 一个 URL 包含所有未来事项,系统日历 App 可周期性拉取自动同步
// 兼容历史:仅追加新接口,不破坏 per-task /tasks/:id/calendar
const calendarSubscribeUrl = computed(() => {
  if (!profileId.value || !creatorKey.value) return ''
  const origin = typeof window !== 'undefined' ? window.location.origin : ''
  const params = new URLSearchParams({ creator_key: creatorKey.value })
  if (calIncludeCancelled.value) params.set('include_cancelled', '1')
  return `${origin}${API_BASE}/profile/${profileId.value}/calendar.ics?${params.toString()}`
})

async function copyCalendarUrl() {
  const url = calendarSubscribeUrl.value
  if (!url) {
    ElMessage.warning('请先登录')
    return
  }
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(url)
    } else {
      // 兼容非安全上下文(内网 http)
      const ta = document.createElement('textarea')
      ta.value = url
      ta.style.position = 'fixed'
      ta.style.left = '-9999px'
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    ElMessage.success({ message: '订阅链接已复制', duration: 2000 })
  } catch (e) {
    ElMessage.error('复制失败,请手动复制下方链接')
  }
}

function downloadSingleTaskICS(task) {
  if (!task || !task.id || !profileId.value || !creatorKey.value) return
  const url = `${API_BASE}/profile/${profileId.value}/tasks/${task.id}/calendar?creator_key=${encodeURIComponent(creatorKey.value)}`
  window.open(url, '_blank')
}

async function runImport(payload) {
  importingJSON.value = true
  const tasks = Array.isArray(payload.tasks) ? payload.tasks : []
  const meetings = Array.isArray(payload.meetings) ? payload.meetings : []
  let taskDone = 0
  let taskFail = 0
  let meetingDone = 0
  let meetingFail = 0
  const errors = []
  try {
    // 任务批量导入(后端单次最多 50 条)
    const TASK_BATCH = 50
    for (let i = 0; i < tasks.length; i += TASK_BATCH) {
      const chunk = tasks.slice(i, i + TASK_BATCH)
      const reqTasks = chunk.map(stripTaskForImport)
      importProgress.value = `导入事项 ${i + 1}-${Math.min(i + TASK_BATCH, tasks.length)} / ${tasks.length}`
      try {
        const resp = await fetch(`${API_BASE}/profile/${profileId.value}/tasks/batch?creator_key=${encodeURIComponent(creatorKey.value)}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ tasks: reqTasks })
        })
        if (!resp.ok) {
          const err = await resp.json().catch(() => ({}))
          throw new Error(err.error || `HTTP ${resp.status}`)
        }
        const data = await resp.json()
        taskDone += data.created_count || reqTasks.length
      } catch (e) {
        taskFail += chunk.length
        errors.push(`事项批次 ${i + 1}-${Math.min(i + TASK_BATCH, tasks.length)}: ${e.message}`)
      }
    }
    // 会议逐条导入(后端无批量端点)
    for (let i = 0; i < meetings.length; i++) {
      const m = meetings[i]
      importProgress.value = `导入会议 ${i + 1} / ${meetings.length}`
      try {
        const resp = await fetch(`${API_BASE}/profile/${profileId.value}/meetings?creator_key=${encodeURIComponent(creatorKey.value)}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(stripMeetingForImport(m))
        })
        if (!resp.ok) {
          const err = await resp.json().catch(() => ({}))
          throw new Error(err.error || `HTTP ${resp.status}`)
        }
        meetingDone += 1
      } catch (e) {
        meetingFail += 1
        errors.push(`会议「${m.title || m.id}」: ${e.message}`)
      }
    }
    // 完成
    importProgress.value = ''
    const parts = []
    if (tasks.length) parts.push(`${taskDone}/${tasks.length} 个事项${taskFail ? ` (失败 ${taskFail})` : ''}`)
    if (meetings.length) parts.push(`${meetingDone}/${meetings.length} 个会议${meetingFail ? ` (失败 ${meetingFail})` : ''}`)
    if (taskFail === 0 && meetingFail === 0) {
      ElMessage.success({ message: `导入完成: ${parts.join(' + ')}`, duration: 3500 })
    } else {
      ElMessage.warning({
        message: `部分导入完成: ${parts.join(' + ')}。失败原因:\n${errors.slice(0, 3).join('\n')}${errors.length > 3 ? `\n... 还有 ${errors.length - 3} 条失败` : ''}`,
        duration: 6000
      })
    }
    await refreshBoard()
  } finally {
    importingJSON.value = false
    setTimeout(() => { importProgress.value = '' }, 5000)
  }
}

// 从导出的 task 中提取后端 create API 接受的字段
// 删掉后端不认的字段(id/profile_id/created_at/updated_at/comments 等),避免 schema 不匹配
function stripTaskForImport(t) {
  if (!t || typeof t !== 'object') return null
  return {
    kind: t.kind || 'life',
    entry_type: t.entry_type || 'task',
    bucket: t.bucket || 'planned',
    title: t.title || '(无标题)',
    detail: t.detail || '',
    notes: t.notes || '',
    status: t.status || 'open',
    priority: t.priority || 'medium',
    planned_for: t.planned_for || '',
    remind_at: t.remind_at || '',
    repeat_type: t.repeat_type || 'none',
    repeat_interval: Number(t.repeat_interval) || 1,
    repeat_until: t.repeat_until || '',
    notify_email: t.notify_email || '',
    cancel_reason: t.cancel_reason || '',
    raw_text: t.raw_text || '',
    postpone_reason: t.last_postpone_reason || t.postpone_reason || '',
    intent: t.intent || '',
    energy_level: t.energy_level || ''
  }
}

function stripMeetingForImport(m) {
  if (!m || typeof m !== 'object') return null
  return {
    title: m.title || '(无标题会议)',
    content: m.content || '',
    summary: m.summary || '',
    action_items: m.action_items || m.actionItems || '[]',
    participants: m.participants || '[]',
    recording_url: m.recording_url || m.recordingUrl || '',
    duration_minutes: Number(m.duration_minutes) || 0,
    meeting_date: m.meeting_date || m.meetingDate || '',
    meeting_time: m.meeting_time || m.meetingTime || '',
    tags: m.tags || '[]',
    status: m.status || 'draft'
  }
}

function applyQuickNaturalDate() {
  const r = _previewNaturalDate(quickNaturalDate.value)
  if (!r) {
    ElMessage.warning('没能识别这个日期,试试: 明天下午3点 / 3天后 / 下周一 / 12月15日')
    return
  }
  quickForm.plannedFor = r.date
  if (r.time) {
    quickForm.remindAt = `${r.date}T${r.time}`
  } else if (!quickForm.remindAt) {
    quickForm.remindAt = `${r.date}T09:00`
  }
  ElMessage.success(`已应用: ${r.date}${r.time ? ' ' + r.time : ' 09:00'}`)
  quickNaturalDate.value = ''
  quickNaturalDateHint.value = '✨ 也支持自然语言: 明天下午3点 / 3天后 / 下周一 / 12月15日'
}

function applyTaskNaturalDate() {
  const r = _previewNaturalDate(taskNaturalDate.value)
  if (!r) {
    ElMessage.warning('没能识别这个日期,试试: 明天下午3点 / 3天后 / 下周一 / 12月15日')
    return
  }
  taskForm.plannedFor = r.date
  if (r.time) {
    taskForm.remindAt = `${r.date}T${r.time}`
  } else if (!taskForm.remindAt) {
    taskForm.remindAt = `${r.date}T09:00`
  }
  ElMessage.success(`已应用: ${r.date}${r.time ? ' ' + r.time : ' 09:00'}`)
  taskNaturalDate.value = ''
  taskNaturalDateHint.value = '✨ 也支持自然语言: 明天下午3点 / 3天后 / 下周一 / 12月15日'
}
function recordCompletionToday() {
  const today = plannerTodayString()
  const current = completionLog.value[today] || 0
  completionLog.value = { ...completionLog.value, [today]: current + 1 }
  saveCompletionLog()
}
const completionStreak = computed(() => {
  const log = completionLog.value || {}
  let streak = 0
  let cursor = new Date(plannerTodayString() + 'T00:00:00')
  if (!log[plannerTodayString()]) {
    cursor.setDate(cursor.getDate() - 1)
  }
  while (true) {
    const key = cursor.toISOString().slice(0, 10)
    const count = log[key] || 0
    if (count > 0) {
      streak++
      cursor.setDate(cursor.getDate() - 1)
    } else {
      break
    }
    if (streak > 365) break
  }
  return streak
})

const last7Days = computed(() => {
  const log = completionLog.value || {}
  const days = []
  const today = plannerTodayString()
  const max = Math.max(
    1,
    ...Array.from({ length: 7 }, (_, i) => {
      const d = new Date(today + 'T00:00:00')
      d.setDate(d.getDate() - i)
      const key = d.toISOString().slice(0, 10)
      return Number(log[key] || 0)
    })
  )
  for (let i = 6; i >= 0; i--) {
    const d = new Date(today + 'T00:00:00')
    d.setDate(d.getDate() - i)
    const key = d.toISOString().slice(0, 10)
    const count = Number(log[key] || 0)
    const weekday = ['日', '一', '二', '三', '四', '五', '六'][d.getDay()]
    days.push({
      date: key,
      count,
      isToday: i === 0,
      weekday,
      // 至少 6% 让 0 件的天也保留一点"占位感"
      barHeight: count === 0 ? 6 : Math.max(10, Math.round((count / max) * 100))
    })
  }
  return days
})

const last7DaysTotal = computed(() => last7Days.value.reduce((s, d) => s + d.count, 0))
const totalDoneCount = computed(() => {
  // 全时累计已完成数:优先用后端 counts.done(更准),否则用本地 completionLog 求和
  const apiTotal = board.value.counts?.done
  if (typeof apiTotal === 'number') return apiTotal
  return Object.values(completionLog.value || {}).reduce((s, n) => s + Number(n || 0), 0)
})

const last7DaysTitle = computed(() => {
  const days = last7Days.value
  const max = Math.max(1, ...days.map(d => d.count))
  return days
    .map(d => `${d.isToday ? '今天 ' : ''}${d.weekday}(${d.date.slice(5)}): ${d.count} 件`)
    .join('\n') + `\n最高单日: ${max} 件 · 共: ${last7DaysTotal.value} 件`
})

const allInboxSelected = computed(() => {
  const items = board.value.inbox_items || []
  if (items.length === 0) return false
  return items.every((t) => inboxSelectedIds.value.has(t.id))
})
function toggleInboxSelected(taskId, checked) {
  const next = new Set(inboxSelectedIds.value)
  if (checked) {
    next.add(taskId)
  } else {
    next.delete(taskId)
  }
  inboxSelectedIds.value = next
}
function toggleSelectAllInbox(checked) {
  const items = board.value.inbox_items || []
  if (checked) {
    inboxSelectedIds.value = new Set(items.map((t) => t.id))
  } else {
    inboxSelectedIds.value = new Set()
  }
}
function clearInboxSelection() {
  inboxSelectedIds.value = new Set()
}
async function batchUpdateTasks(action) {
  const ids = Array.from(inboxSelectedIds.value)
  if (ids.length === 0) return
  const labels = {
    move_to_today: '移到今天',
    move_to_someday: '移到以后',
    mark_done: '标完成',
    delete: '删除'
  }
  if (action === 'delete') {
    try {
      await ElMessageBox.confirm(`确定要删除选中的 ${ids.length} 条事项吗？`, '批量删除', {
        type: 'warning',
        confirmButtonText: '删除',
        cancelButtonText: '取消'
      })
    } catch (e) {
      return
    }
  }
  try {
    const response = await plannerFetch(
      `${API_BASE}/profile/${profileId.value}/tasks/batch-update`,
      {
        method: 'POST',
        body: JSON.stringify({
          creator_key: creatorKey.value,
          task_ids: ids,
          action
        })
      }
    )
    const data = await response.json()
    if (data && data.code === 0) {
      const updated = (data.updated || []).length
      const deleted = (data.deleted || []).length
      ElMessage.success(`${labels[action] || '完成'}:更新 ${updated} 条,删除 ${deleted} 条`)
      inboxSelectedIds.value = new Set()
      if (action === 'mark_done' && updated > 0) {
        for (let i = 0; i < updated; i++) recordCompletionToday()
      }
      await refreshBoard()
    } else {
      ElMessage.error((data && (data.error || data.message)) || '操作失败')
    }
  } catch (error) {
    ElMessage.error(error.message || '操作失败')
  }
}

// === 通用多选 (跨视图) ===
function isSelected(taskId) {
  return selectedTaskIds.value.has(taskId)
}
function toggleSelected(taskId, checked) {
  const next = new Set(selectedTaskIds.value)
  if (checked) next.add(taskId)
  else next.delete(taskId)
  selectedTaskIds.value = next
}
function clearSelection() {
  selectedTaskIds.value = new Set()
}
function setSelectedFromList(items, selected) {
  const next = new Set(selectedTaskIds.value)
  for (const t of (items || [])) {
    if (!t || !t.id) continue
    if (selected) next.add(t.id)
    else next.delete(t.id)
  }
  selectedTaskIds.value = next
}
function batchAction(action) {
  if (action === 'delete') {
    return batchUpdateTasksGeneric(action)
  }
  return batchUpdateTasksGeneric(action)
}

const BATCH_LABELS = {
  move_to_today: '移到今天',
  move_to_someday: '移到以后',
  mark_done: '标完成',
  delete: '删除'
}

async function batchUpdateTasksGeneric(action, extra) {
  const ids = Array.from(selectedTaskIds.value)
  if (ids.length === 0) return
  if (action === 'delete') {
    try {
      await ElMessageBox.confirm(`确定要删除选中的 ${ids.length} 条事项吗？`, '批量删除', {
        type: 'warning',
        confirmButtonText: '删除',
        cancelButtonText: '取消'
      })
    } catch (e) {
      return
    }
  }
  // 阶段 5 减法 (iter 21):批量操作 1 步撤销 — 与单删除 / 单完成 (iter 18) 对称。
  // 批量改之前先 snapshot 每个 task,操作成功后弹 undo snackbar。
  // delete:用 restoreTask 全量重建(含 comments)
  // 其他(update status / date / bucket):逐个 updateTask 改回 snapshot 中的旧值
  const snapshots = []
  if (ids.length > 0 && ids.length <= 30) {
    // 上限 30 防意外 N+1,30 已经是批量操作合理上限
    for (const id of ids) {
      const s = await snapshotTaskForUndo(id)
      if (s) snapshots.push(s)
    }
  }
  try {
    const response = await plannerFetch(
      `${API_BASE}/profile/${profileId.value}/tasks/batch-update`,
      {
        method: 'POST',
        body: JSON.stringify({
          creator_key: creatorKey.value,
          task_ids: ids,
          action,
          // 阶段 5 减法 (iter 25):批量改优先级时附 priority 字段
          // 后端 set_priority action 在 planner_tasks.go:764 已支持
          ...(action === 'set_priority' && extra?.priority ? { priority: extra.priority } : {})
        })
      }
    )
    const data = await response.json()
    if (data && data.code === 0) {
      const updated = (data.updated || []).length
      const deleted = (data.deleted || []).length
      const actionLabel = action === 'set_priority' && extra?.priority
        ? `改优先级为 ${priorityLabel(extra.priority)}`
        : (BATCH_LABELS[action] || '完成')
      ElMessage.success(`${actionLabel}:更新 ${updated} 条,删除 ${deleted} 条`)
      clearSelection()
      if (action === 'mark_done' && updated > 0) {
        for (let i = 0; i < updated; i++) recordCompletionToday()
      }
      // 阶段 5 减法 (iter 21):弹批量 undo snackbar,与单条 iter 18 对称
      // 只在有 snapshot 的情况下弹(<= 30 才有 snapshot)
      if (snapshots.length > 0) {
        showUndoSnackbar({
          kind: 'batch',
          message: `${BATCH_LABELS[action] || '操作'}完成 ${snapshots.length} 条`,
          onUndo: () => undoBatchOperation(action, snapshots)
        })
      }
      await refreshBoard()
    } else {
      ElMessage.error((data && (data.error || data.message)) || '操作失败')
    }
  } catch (error) {
    ElMessage.error(error.message || '操作失败')
  }
}

// 阶段 5 减法 (iter 21):批量操作 1 步撤销实现
// delete:对每个 snapshot 调 restoreTask(全量重建,含 comments)
// 其他:对每个 snapshot 调 updateTask 改回原 status / planned_for / bucket
async function undoBatchOperation(action, snapshots) {
  if (!Array.isArray(snapshots) || snapshots.length === 0) return
  if (action === 'delete') {
    // 重建:用 restoreTask 逐个恢复(含 comments),错误吞掉不让用户看到一个失败就崩
    for (const s of snapshots) {
      try {
        await restoreTask(s)
      } catch (e) {
        if (e?.message) console.error('restore failed', s?.id, e)
      }
    }
    if (!quietMode.value) {
      ElMessage({ message: `已撤回,${snapshots.length} 条已恢复`, type: 'info', duration: 1800 })
    }
    await refreshBoard()
    return
  }
  // 非 delete 场景:把每个 task 改回 snapshot 中的原值
  // 取每个字段的 snapshot 原值,只传有变化的字段
  let restored = 0
  for (const s of snapshots) {
    if (!s || !s.id) continue
    const patch = {}
    if (action === 'mark_done' && s.status) {
      patch.status = s.status === 'done' ? 'open' : s.status
    } else if (action === 'move_to_today' || action === 'move_to_someday') {
      if (s.planned_for !== undefined) patch.planned_for = s.planned_for
      if (s.bucket !== undefined) patch.bucket = s.bucket
    } else if (action === 'set_priority' && s.priority) {
      // 阶段 5 减法 (iter 25):批量改优先级撤销 — 改回 snapshot 中的 priority
      patch.priority = s.priority
    }
    if (Object.keys(patch).length === 0) continue
    try {
      const resp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${s.id}`, {
        method: 'PUT',
        body: JSON.stringify(patch)
      })
      if (resp.ok) restored++
    } catch (e) {
      if (e?.message) console.error('undo batch PUT failed', s.id, e)
    }
  }
  if (!quietMode.value) {
    ElMessage({ message: `已撤回,${restored} 条已恢复原状态`, type: 'info', duration: 1800 })
  }
  await refreshBoard()
}

// 阶段 5 减法:End-of-Day 收尾仪式 — 1 步把今天剩余的全部标完成。
// 与 iter 8 "完成最后一件 → 收尾仪式"对称:iter 8 是被动触发,iter 15 是主动批量。
// 用 mini 弹窗(2 按钮)兜底,避免误触。
async function finishTodayAll() {
  const today = new Date().toISOString().slice(0, 10)
  const ids = []
  for (const g of (board.value.groups || [])) {
    for (const t of (g.items || [])) {
      if ((t.status === 'open' || t.status === 'in_progress') && (t.planned_for || '') === today) {
        ids.push(t.id)
      }
    }
  }
  if (ids.length === 0) return
  try {
    await ElMessageBox.confirm(
      `今天还有 ${ids.length} 件没完成,一次性全部标完成?`,
      '收尾今天',
      {
        confirmButtonText: '全部收尾',
        cancelButtonText: '再想想',
        type: 'info'
      }
    )
  } catch {
    return
  }
  try {
    const response = await plannerFetch(
      `${API_BASE}/profile/${profileId.value}/tasks/batch-update`,
      {
        method: 'POST',
        body: JSON.stringify({
          creator_key: creatorKey.value,
          task_ids: ids,
          action: 'mark_done'
        })
      }
    )
    const data = await response.json()
    if (data && data.code === 0) {
      const updated = (data.updated || []).length
      if (updated > 0) {
        for (let i = 0; i < updated; i++) recordCompletionToday()
        ElMessage({
          message: `收尾了 ${updated} 件 · 今天做得很好`,
          type: 'success',
          duration: 2400
        })
        // 触发 iter 8 的收尾仪式(todayAllDone 由 done_today >= 1 自动判断)
        await refreshBoard()
      }
    } else {
      ElMessage.error((data && (data.error || data.message)) || '收尾失败')
    }
  } catch (error) {
    ElMessage.error(error.message || '收尾失败')
  }
}

// 阶段 5 减法 (iter 34):focus.secondary "✓ 收尾 N 个次要" 1 步批量完成
// 消除"N 个次要逐条点"的 N 步摩擦,用 batch-update API + 5s 撤销兜底
// 与 iter 21 批量撤销 + iter 18 单完成撤销 完全对称
async function completeAllSecondary() {
  const list = currentSecondaryList.value
  if (!list || list.length < 2) return
  const ids = list.map(t => t.id)
  // 先 snapshot (与 iter 21 undo 模式对齐)
  const snapshots = []
  for (const t of list) {
    snapshots.push({
      id: t.id,
      status: t.status,
      title: t.title
    })
  }
  try {
    const response = await plannerFetch(
      `${API_BASE}/profile/${profileId.value}/tasks/batch-update`,
      {
        method: 'POST',
        body: JSON.stringify({
          creator_key: creatorKey.value,
          task_ids: ids,
          action: 'mark_done'
        })
      }
    )
    const data = await response.json()
    if (data && data.code === 0) {
      const updated = (data.updated || []).length
      if (updated > 0) {
        for (let i = 0; i < updated; i++) recordCompletionToday()
        showUndoSnackbar({
          kind: 'batch',
          message: `收尾了 ${updated} 个次要 · 1 步撤回`,
          onUndo: () => undoBatchOperation('mark_done', snapshots)
        })
        await refreshBoard()
      }
    } else {
      ElMessage.error((data && (data.error || data.message)) || '收尾次要失败')
    }
  } catch (error) {
    ElMessage.error(error.message || '收尾次要失败')
  }
}

// 阶段 5 减法 (iter 36):focus.secondary "↻ 顺延 N 个到明天" 1 步批量
// 与 iter 34 "✓ 收尾" 1 步批量对称,覆盖"今天不做了,改到明天"场景
// 用 individual update 循环(不依赖新后端 action),snapshot + 5s 撤销兜底
async function postponeAllSecondaryToTomorrow() {
  const list = currentSecondaryList.value
  if (!list || list.length < 2) return
  const tomorrow = tomorrowDateString()
  // snapshot 原 planned_for / status,用于撤销
  const snapshots = list.map(t => ({
    id: t.id,
    title: t.title,
    planned_for: t.planned_for || '',
    status: t.status
  }))
  // 逐个 update (no backend change, 用现有 updateTask 路径)
  let updated = 0
  for (const t of list) {
    try {
      const response = await plannerFetch(
        `${API_BASE}/profile/${profileId.value}/tasks/${t.id}`,
        {
          method: 'PUT',
          body: JSON.stringify({
            creator_key: creatorKey.value,
            planned_for: tomorrow,
            status: 'open',
            postpone_reason: '1 步批量顺延到明天'
          })
        }
      )
      if (response.ok) updated++
    } catch (e) {
      if (e?.message) console.error('bulk postpone failed', t.id, e)
    }
  }
  if (updated > 0) {
    showUndoSnackbar({
      kind: 'batch',
      message: `顺延了 ${updated} 个到明天 · 1 步撤回`,
      onUndo: async () => {
        for (const s of snapshots) {
          try {
            await plannerFetch(
              `${API_BASE}/profile/${profileId.value}/tasks/${s.id}`,
              {
                method: 'PUT',
                body: JSON.stringify({
                  creator_key: creatorKey.value,
                  planned_for: s.planned_for,
                  status: s.status
                })
              }
            )
          } catch (e) {
            if (e?.message) console.error('bulk postpone undo failed', s.id, e)
          }
        }
        await refreshBoard()
        if (!quietMode.value) {
          ElMessage({ message: `已撤回,${updated} 个次要恢复原计划`, type: 'info', duration: 1800 })
        }
      }
    })
    await refreshBoard()
  } else {
    ElMessage.error('顺延失败,请稍后再试')
  }
}

// 阶段 5 减法 (iter 44):focus.secondary "✗ 取消 N 个" 1 步批量
// 与 iter 34 "✓ 收尾" + iter 36 "↻ 顺延" 形成状态机 1 步批量完整闭环
// 覆盖"今天这堆都不做了/条件变化"场景,无需逐个取消
// 用 individual update 循环(不依赖新后端 action),snapshot + 5s 撤销兜底
// 默认原因"不再需要"(用户最常用的取消原因,iter 13 调研)
async function cancelAllSecondary() {
  const list = currentSecondaryList.value
  if (!list || list.length < 2) return
  // snapshot 原 status / planned_for,用于撤销
  const snapshots = list.map(t => ({
    id: t.id,
    title: t.title,
    planned_for: t.planned_for || '',
    status: t.status
  }))
  // 逐个 cancel
  let updated = 0
  for (const t of list) {
    try {
      const response = await plannerFetch(
        `${API_BASE}/profile/${profileId.value}/tasks/${t.id}`,
        {
          method: 'PUT',
          body: JSON.stringify({
            creator_key: creatorKey.value,
            status: 'cancelled',
            cancel_reason: '1 步批量取消'
          })
        }
      )
      if (response.ok) updated++
    } catch (e) {
      if (e?.message) console.error('bulk cancel failed', t.id, e)
    }
  }
  if (updated > 0) {
    showUndoSnackbar({
      kind: 'batch',
      message: `取消了 ${updated} 个次要 · 1 步撤回`,
      onUndo: async () => {
        for (const s of snapshots) {
          try {
            await plannerFetch(
              `${API_BASE}/profile/${profileId.value}/tasks/${s.id}`,
              {
                method: 'PUT',
                body: JSON.stringify({
                  creator_key: creatorKey.value,
                  status: s.status,
                  planned_for: s.planned_for
                })
              }
            )
          } catch (e) {
            if (e?.message) console.error('bulk cancel undo failed', s.id, e)
          }
        }
        await refreshBoard()
        if (!quietMode.value) {
          ElMessage({ message: `已撤回,${updated} 个次要恢复原状态`, type: 'info', duration: 1800 })
        }
      }
    })
    await refreshBoard()
  } else {
    ElMessage.error('取消失败,请稍后再试')
  }
}

// 阶段 5 减法 (iter 38):focus 主卡 1 步"📅 改明天"
// 现在:点"📅 改明天" → 1 步完成 → 5s 内可撤回
async function postponeFocusPrimaryToTomorrow() {
  const task = currentFocusTask.value
  if (!task || !task.id) return
  if (task.status === 'done' || task.status === 'cancelled') return
  const tomorrow = tomorrowDateString()
  // snapshot 原 planned_for / status,用于撤销
  const snapshot = {
    id: task.id,
    title: task.title,
    planned_for: task.planned_for || '',
    status: task.status
  }
  try {
    const response = await plannerFetch(
      `${API_BASE}/profile/${profileId.value}/tasks/${task.id}`,
      {
        method: 'PUT',
        body: JSON.stringify({
          creator_key: creatorKey.value,
          planned_for: tomorrow,
          status: 'open',
          postpone_reason: 'focus 主卡 1 步改明天'
        })
      }
    )
    if (response.ok) {
      showUndoSnackbar({
        kind: 'task',
        message: `已改到 ${tomorrow} · 1 步撤回`,
        onUndo: async () => {
          try {
            await plannerFetch(
              `${API_BASE}/profile/${profileId.value}/tasks/${snapshot.id}`,
              {
                method: 'PUT',
                body: JSON.stringify({
                  creator_key: creatorKey.value,
                  planned_for: snapshot.planned_for,
                  status: snapshot.status
                })
              }
            )
            await refreshBoard()
            if (!quietMode.value) {
              ElMessage({ message: '已撤回,恢复原计划', type: 'info', duration: 1800 })
            }
          } catch (e) {
            if (e?.message) console.error('focus primary postpone undo failed', e)
          }
        }
      })
      await refreshBoard()
    } else {
      ElMessage.error('改明天失败,请稍后再试')
    }
  } catch (e) {
    if (e?.message) console.error('focus primary postpone failed', e)
    ElMessage.error('改明天失败,请稍后再试')
  }
}

function notifiedStorageKey() {
  return profileId.value ? `planner_notified_${profileId.value}` : ''
}
function loadNotifiedTaskIds() {
  const key = notifiedStorageKey()
  if (!key) {
    notifiedTaskIds.value = new Set()
    return
  }
  try {
    const raw = localStorage.getItem(key)
    if (!raw) {
      notifiedTaskIds.value = new Set()
      return
    }
    const data = JSON.parse(raw)
    const cutoff = Date.now() - 1000 * 60 * 60 * 24 * 7
    const filtered = Object.entries(data || {}).filter(([_, ts]) => ts > cutoff)
    notifiedTaskIds.value = new Set(filtered.map(([id]) => id))
    saveNotifiedTaskIds()
  } catch (error) {
    notifiedTaskIds.value = new Set()
  }
}
function saveNotifiedTaskIds() {
  const key = notifiedStorageKey()
  if (!key) return
  const obj = {}
  notifiedTaskIds.value.forEach((id) => {
    obj[id] = Date.now()
  })
  localStorage.setItem(key, JSON.stringify(obj))
}

async function toggleNotifications(enabled) {
  if (!notificationSupported) {
    ElMessage.warning('当前浏览器不支持通知')
    notificationEnabled.value = false
    return
  }
  localStorage.setItem('planner_notification_enabled', enabled ? '1' : '0')
  if (!enabled) {
    if (notificationTimer) {
      clearInterval(notificationTimer)
      notificationTimer = null
    }
    return
  }
  if (Notification.permission === 'granted') {
    notificationPermission.value = 'granted'
    startNotificationLoop()
    return
  }
  notificationRequesting.value = true
  try {
    const result = await Notification.requestPermission()
    notificationPermission.value = result
    if (result === 'granted') {
      startNotificationLoop()
      ElMessage.success('通知已启用,任务提醒时系统会弹出')
    } else if (result === 'denied') {
      notificationEnabled.value = false
      localStorage.setItem('planner_notification_enabled', '0')
      ElMessage.warning('通知被拒绝,可在浏览器设置中重新开启')
    }
  } catch (error) {
    notificationEnabled.value = false
    localStorage.setItem('planner_notification_enabled', '0')
    ElMessage.error('请求通知权限失败')
  } finally {
    notificationRequesting.value = false
  }
}

function startNotificationLoop() {
  if (notificationTimer) clearInterval(notificationTimer)
  checkReminders()
  notificationTimer = setInterval(checkReminders, 60 * 1000)
}

function checkReminders() {
  if (!notificationEnabled.value) return
  if (Notification.permission !== 'granted') return
  const now = Date.now()
  const allTasks = []
  board.value.groups?.forEach((g) => (allTasks.push(...(g.items || []))))
  if (board.value.event_groups) board.value.event_groups.forEach((g) => (allTasks.push(...(g.items || []))))
  if (board.value.inbox_items) allTasks.push(...board.value.inbox_items)
  if (board.value.someday_items) allTasks.push(...board.value.someday_items)
  allTasks.forEach((task) => {
    if (!task.remind_at) return
    if (task.status === 'done' || task.status === 'cancelled') return
    if (notifiedTaskIds.value.has(task.id)) return
    const remindTime = new Date(task.remind_at).getTime()
    if (Number.isNaN(remindTime)) return
    if (remindTime > now) return
    fireNotification(task)
    notifiedTaskIds.value.add(task.id)
  })
  saveNotifiedTaskIds()
}

function fireNotification(task) {
  if (!('Notification' in window) || Notification.permission !== 'granted') return
  const title = `⏰ ${task.title}`
  const body = task.detail || '该到提醒时间了'
  try {
    const notification = new Notification(title, {
      body,
      tag: `planner_${task.id}`,
      icon: '/favicon.ico',
      requireInteraction: false
    })
    lastNotificationPreview.value = title
    notification.onclick = () => {
      window.focus()
      try { openTask(task) } catch (e) {}
      notification.close()
    }
    setTimeout(() => notification.close(), 30 * 1000)
  } catch (e) {
    // ignore
  }
}

function testNotification() {
  if (Notification.permission !== 'granted') return
  try {
    const n = new Notification('测试通知', {
      body: '通知正常工作 · 任务提醒时会这样弹出',
      tag: 'planner_test'
    })
    setTimeout(() => n.close(), 5000)
  } catch (e) {
    // ignore
  }
}

function focusSessionStorageKey() {
  return profileId.value ? `planner_focus_session_${profileId.value}` : ''
}
function focusLogStorageKey() {
  return profileId.value ? `planner_focus_log_${profileId.value}` : ''
}
function loadFocusSession() {
  const key = focusSessionStorageKey()
  if (!key) {
    focusSession.value = null
    return
  }
  try {
    const raw = localStorage.getItem(key)
    if (!raw) {
      focusSession.value = null
      return
    }
    const data = JSON.parse(raw)
    if (!data || !data.taskId) {
      focusSession.value = null
      return
    }
    if (data.isPaused) {
      focusSession.value = data
    } else {
      const runningFor = Date.now() - (data.startedAt || 0)
      if (runningFor > 1000 * 60 * 60 * 12) {
        focusSession.value = null
        localStorage.removeItem(key)
        return
      }
      focusSession.value = data
    }
    startFocusTick()
  } catch (error) {
    focusSession.value = null
  }
}
function saveFocusSession() {
  const key = focusSessionStorageKey()
  if (!key) return
  if (!focusSession.value) {
    localStorage.removeItem(key)
    return
  }
  localStorage.setItem(key, JSON.stringify(focusSession.value))
}
function loadFocusLog() {
  const key = focusLogStorageKey()
  if (!key) {
    focusLog.value = {}
    return
  }
  try {
    const raw = localStorage.getItem(key)
    focusLog.value = raw ? JSON.parse(raw) : {}
  } catch (error) {
    focusLog.value = {}
  }
}
function saveFocusLog() {
  const key = focusLogStorageKey()
  if (!key) return
  const cutoff = Date.now() - 1000 * 60 * 60 * 24 * 30
  const trimmed = {}
  Object.entries(focusLog.value || {}).forEach(([date, ms]) => {
    if (!date || !ms) return
    if (new Date(date + 'T00:00:00').getTime() < cutoff) return
    trimmed[date] = ms
  })
  focusLog.value = trimmed
  localStorage.setItem(key, JSON.stringify(trimmed))
}
function addFocusMinutesToday(ms) {
  if (ms <= 0) return
  const today = plannerTodayString()
  focusLog.value = { ...focusLog.value, [today]: (focusLog.value[today] || 0) + ms }
  saveFocusLog()
}
const focusTodayMs = computed(() => {
  const today = plannerTodayString()
  return focusLog.value[today] || 0
})
function formatFocusDuration(ms) {
  if (!ms || ms < 0) return '00:00'
  const totalSec = Math.floor(ms / 1000)
  const h = Math.floor(totalSec / 3600)
  const m = Math.floor((totalSec % 3600) / 60)
  const s = totalSec % 60
  if (h > 0) {
    return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  }
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}
function startFocusTick() {
  if (focusTickTimer) return
  computeFocusElapsed()
  focusTickTimer = setInterval(computeFocusElapsed, 1000)
}
function stopFocusTick() {
  if (focusTickTimer) {
    clearInterval(focusTickTimer)
    focusTickTimer = null
  }
}
function computeFocusElapsed() {
  if (!focusSession.value) {
    focusElapsedMs.value = 0
    return
  }
  const session = focusSession.value
  let total = session.accumulatedMs || 0
  if (!session.isPaused && session.startedAt) {
    total += Date.now() - session.startedAt
  }
  focusElapsedMs.value = total
}
async function startFocus(task) {
  if (!task) return
  if (focusSession.value && focusSession.value.taskId !== task.id) {
    try {
      await ElMessageBox.confirm('已有正在专注的事项,开始新专注会结束当前专注并保存时长。', '切换专注', {
        type: 'info',
        confirmButtonText: '切换',
        cancelButtonText: '取消'
      })
    } catch (e) {
      return
    }
    await stopFocus({ silent: true })
  }
  const now = Date.now()
  focusSession.value = {
    taskId: task.id,
    taskTitle: task.title,
    startedAt: now,
    accumulatedMs: 0,
    isPaused: false
  }
  saveFocusSession()
  startFocusTick()
  if (task.status !== 'in_progress' && task.status !== 'done' && task.status !== 'cancelled') {
    try {
      await updateTask(task, { status: 'in_progress' }, null)
    } catch (e) {
      // ignore
    }
  }
}
function toggleFocusPause() {
  if (!focusSession.value) return
  if (focusSession.value.isPaused) {
    focusSession.value = {
      ...focusSession.value,
      isPaused: false,
      startedAt: Date.now()
    }
  } else {
    let total = focusSession.value.accumulatedMs || 0
    if (focusSession.value.startedAt) {
      total += Date.now() - focusSession.value.startedAt
    }
    focusSession.value = {
      ...focusSession.value,
      isPaused: true,
      accumulatedMs: total,
      startedAt: null
    }
  }
  saveFocusSession()
  computeFocusElapsed()
}
async function stopFocus(options = {}) {
  if (!focusSession.value) return
  const session = focusSession.value
  let total = session.accumulatedMs || 0
  if (!session.isPaused && session.startedAt) {
    total += Date.now() - session.startedAt
  }
  if (total > 0) {
    addFocusMinutesToday(total)
  }
  const minutes = Math.max(1, Math.round(total / 60000))
  focusSession.value = null
  focusElapsedMs.value = 0
  stopFocusTick()
  saveFocusSession()
  if (!options.silent) {
    ElMessage.success(`已保存 ${minutes} 分钟专注时长`)
  }
}
async function completeFocusedTask() {
  if (!focusSession.value) return
  const session = focusSession.value
  let total = session.accumulatedMs || 0
  if (!session.isPaused && session.startedAt) {
    total += Date.now() - session.startedAt
  }
  if (total > 0) {
    addFocusMinutesToday(total)
  }
  const minutes = Math.max(1, Math.round(total / 60000))
  let task = findTaskInBoard(session.taskId)
  if (!task && session.taskId) {
    try {
      const resp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${session.taskId}?creator_key=${creatorKey.value}`)
      if (resp.ok) {
        const data = await resp.json()
        task = data.task || data
      }
    } catch {}
  }
  focusSession.value = null
  focusElapsedMs.value = 0
  stopFocusTick()
  saveFocusSession()
  if (task) {
    triggerCelebration(task)
    ElMessage.success(`专注 ${minutes} 分钟,已标记完成`)
    await updateTask(task, { status: 'done' }, null)
  } else {
    ElMessage.success(`专注 ${minutes} 分钟已保存`)
  }
}

function loadPinnedTaskIds() {
  const key = focusPinnedStorageKey()
  if (!key) {
    pinnedTaskIds.value = new Set()
    return
  }
  try {
    const raw = localStorage.getItem(key)
    if (!raw) {
      pinnedTaskIds.value = new Set()
      return
    }
    const list = JSON.parse(raw)
    pinnedTaskIds.value = new Set(Array.isArray(list) ? list : [])
  } catch (error) {
    pinnedTaskIds.value = new Set()
  }
}
function savePinnedTaskIds() {
  const key = focusPinnedStorageKey()
  if (!key) return
  localStorage.setItem(key, JSON.stringify(Array.from(pinnedTaskIds.value)))
}
function togglePin(task) {
  if (!task || !task.id) return
  const next = new Set(pinnedTaskIds.value)
  if (next.has(task.id)) {
    next.delete(task.id)
  } else {
    next.add(task.id)
    if (next.size > 7) {
      const first = next.values().next().value
      next.delete(first)
      ElMessage.info('聚焦列表最多保留 7 个,先取消最久前的再 pin')
    }
  }
  pinnedTaskIds.value = next
  savePinnedTaskIds()
}

const pinnedTasks = computed(() => {
  if (pinnedTaskIds.value.size === 0) return []
  const all = []
  board.value.groups?.forEach((g) => (all.push(...(g.items || []))))
  if (board.value.event_groups) board.value.event_groups.forEach((g) => (all.push(...(g.items || []))))
  if (board.value.inbox_items) all.push(...board.value.inbox_items)
  if (board.value.someday_items) all.push(...board.value.someday_items)
  const map = new Map(all.map((t) => [t.id, t]))
  const pinned = []
  pinnedTaskIds.value.forEach((id) => {
    const t = map.get(id)
    if (t) pinned.push(t)
  })
  return pinned
})
const activePinnedFocus = computed(() => {
  const open = pinnedTasks.value.filter(
    (t) => t.status !== 'done' && t.status !== 'cancelled'
  )
  if (open.length === 0) return null
  const primary = open[0]
  const secondary = open.slice(1, 3)
  return { primary, secondary, extras: open.length - 1 - secondary.length }
})

const tomorrowCount = computed(() => {
  const tomorrow = tomorrowDateString()
  let count = 0
  for (const g of board.value.groups || []) {
    for (const t of (g.items || [])) {
      if (t.display_date === tomorrow || t.planned_for === tomorrow) {
        if (t.status !== 'done' && t.status !== 'cancelled') count++
      }
    }
  }
  for (const g of board.value.event_groups || []) {
    for (const t of (g.items || [])) {
      if (t.display_date === tomorrow || t.planned_for === tomorrow) {
        if (t.status !== 'done' && t.status !== 'cancelled') count++
      }
    }
  }
  return count
})

const viewOptions = computed(() => [
  { key: 'timeline', label: '时间线', count: board.value.groups.reduce((sum, group) => sum + group.items.length, 0) },
  { key: 'tomorrow', label: '明天', count: tomorrowCount.value },
  { key: 'events', label: '事件', count: board.value.event_groups.reduce((sum, group) => sum + group.items.length, 0) },
  { key: 'inbox', label: '收件箱', count: board.value.inbox_items.length },
  { key: 'someday', label: '放一放', count: board.value.someday_items.length },
  { key: 'minutes', label: '会议纪要', count: meetings.value.length },
  { key: 'recent', label: '最近', count: board.value.recent_items.length }
])

const tomorrowGroups = computed(() => {
  const tomorrow = tomorrowDateString()
  const items = []
  for (const g of board.value.groups || []) {
    for (const t of (g.items || [])) {
      if (t.display_date === tomorrow || t.planned_for === tomorrow) {
        if (t.status !== 'done' && t.status !== 'cancelled') {
          items.push(t)
        }
      }
    }
  }
  for (const g of board.value.event_groups || []) {
    for (const t of (g.items || [])) {
      if (t.display_date === tomorrow || t.planned_for === tomorrow) {
        if (t.status !== 'done' && t.status !== 'cancelled') {
          items.push(t)
        }
      }
    }
  }
  if (items.length === 0) return []
  return [{ date: tomorrow, label: '明天', items }]
})

const priorityFilter = ref(localStorage.getItem('planner_priority_filter') || 'all')
const PRIORITY_FILTERS = [
  { key: 'all', label: '全部', match: () => true },
  { key: 'urgent', label: '🚨 紧急', match: (t) => t.priority === 'urgent' },
  { key: 'high', label: '高优先级', match: (t) => t.priority === 'high' || t.priority === 'urgent' },
  { key: 'today', label: '今天', match: (t) => t.is_today || t.display_date === plannerTodayString() },
  { key: 'overdue', label: '顺延', match: (t) => t.is_rolled_over }
]

function plannerTodayString() {
  return new Date().toISOString().slice(0, 10)
}

const priorityFilterOptions = computed(() => {
  const allItems = []
  for (const g of board.value.groups || []) allItems.push(...(g.items || []))
  return PRIORITY_FILTERS.map(f => ({
    key: f.key,
    label: f.label,
    count: allItems.filter(f.match).length
  }))
})

function setPriorityFilter(key) {
  priorityFilter.value = priorityFilter.value === key ? 'all' : key
  localStorage.setItem('planner_priority_filter', priorityFilter.value)
}

// 阶段 5 减法 (iter 39):主页 🚨 紧急 chip 1 步 = 切时间线 + 切紧急 filter
// 复用已有 priorityFilter + activeView,与 timeline filter-strip 形成"双入口"对称
// 无紧急事项时不可点(urgentOpenCount=0 直接 return);
// 已在紧急 filter 时再次点击保持不变(可读性 > 抖动,想切回全部用 timeline 顶部 "全部" chip)
function jumpToUrgent() {
  if (urgentOpenCount.value === 0) return
  activeView.value = 'timeline'
  if (priorityFilter.value !== 'urgent') {
    priorityFilter.value = 'urgent'
    localStorage.setItem('planner_priority_filter', 'urgent')
  }
}

// 阶段 5 减法 (iter 51):主页 🔥 连续 chip 1 步 = 最近 view
// 复用已有 activeView 系统,与 🚨 紧急 iter 39 形成"双入口"对称
// 场景:用户看到连续 N 天 → 想知道最近完成的具体是什么 → 直接点 chip → 跳最近 view
// 原来:必须手动点 view tab "最近" (第 1 步) + 找到该 tab (在第 7 个位置) = 2 步摩擦 + 视觉搜索
// 现在:主页 🔥 连续 chip 直接 1 步跳 → 看到所有完成/取消的回顾
// 无完成历史时不可点(streak=0 且 last7DaysTotal=0 → 直接 return)
// 已在最近 view 时再次点击保持不变(可读性 > 抖动,想切回时间线用 view tab)
// 与 iter 39 对称:🚨 紧急 → 时间线+紧急 filter / 🔥 连续 → 最近 view
function jumpToStreak() {
  if (completionStreak.value === 0 && last7DaysTotal.value === 0) return
  activeView.value = 'recent'
}

const filteredTimelineGroups = computed(() => {
  const filter = PRIORITY_FILTERS.find(f => f.key === priorityFilter.value)
  if (!filter || filter.key === 'all') return board.value.groups || []
  return (board.value.groups || []).map(g => ({
    ...g,
    items: (g.items || []).filter(filter.match)
  })).filter(g => g.items.length > 0)
})

const currentViewMeta = computed(() => {
  return {
    timeline: {
      title: activeKind.value === 'work' ? '主时间线' : '生活时间线',
      body: activeKind.value === 'work' ? '把真正要推进的任务压到少量关键动作上。' : '生活事项也值得被温柔安排。'
    },
    tomorrow: {
      title: `明天 · ${tomorrowDateString()}`,
      body: '把今天先不急的事挪到这里，节奏会更舒服。'
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
  quickForm.rawText = ''
  quickForm.entryType = 'task'
  quickForm.bucket = 'planned'
  quickForm.plannedFor = new Date().toISOString().slice(0, 10)
  quickForm.remindAt = ''
  quickForm.repeatType = 'none'
  quickForm.repeatInterval = 1
  quickForm.repeatUntil = ''
  quickForm.priority = 'medium'
  quickForm.notifyEmail = profile.notify_email || ''
  quickForm.intent = ''
  quickForm.energy_level = ''
  if (isMobile.value) {
    quickAdvancedVisible.value = false
  }
}

function tomorrowDateString() {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  return d.toISOString().slice(0, 10)
}

const repeatQuickOptions = [
  { value: 'none', label: '一次性' },
  { value: 'daily', label: '每天' },
  { value: 'weekdays', label: '工作日' },
  { value: 'weekly', label: '每周' },
  { value: 'monthly', label: '每月' }
]
function applyQuickRepeat(value) {
  if (value !== 'none' && !quickForm.remindAt) {
    quickAdvancedVisible.value = true
    ElMessage.info('先在「高级设置」里设置提醒时间,再选重复方式')
    return
  }
  quickForm.repeatType = value
  if (value !== 'none') {
    quickForm.repeatInterval = 1
  }
}

function applyQuickPreset(type) {
  if (type === 'tomorrow') {
    quickForm.title = ''
    quickForm.detail = ''
    quickForm.rawText = ''
    quickForm.entryType = 'task'
    quickForm.bucket = 'planned'
    quickForm.plannedFor = tomorrowDateString()
    quickForm.remindAt = ''
    quickForm.repeatType = 'none'
    quickForm.repeatInterval = 1
    quickForm.repeatUntil = ''
    quickForm.priority = 'medium'
    quickForm.notifyEmail = profile.notify_email || ''
    quickForm.intent = ''
    quickForm.energy_level = ''
    if (isMobile.value) quickAdvancedVisible.value = false
    return
  }
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

function logout() {
  clearSession()
  Object.assign(profile, { id: '', name: '', notify_email: '', expires_at: '', mode_default: 'life' })
  board.value = createEmptyBoard()
  meetings.value = []
  activeView.value = 'timeline'
  selectedTaskIds.value = new Set()
}

function startTriage() {
  triageIndex.value = 0
  triageActive.value = true
  triageCompleted.value = false
  triageStats.planned = 0
  triageStats.someday = 0
  triageStats.discarded = 0
}

async function triageDecide(status) {
  const item = triageCurrent.value
  if (!item) return

  if (status === 'today' || status === 'planned') {
    const targetDate = status === 'today'
      ? new Date().toISOString().slice(0, 10)
      : (() => { const d = new Date(); d.setDate(d.getDate() + 1); return d.toISOString().slice(0, 10) })()
    await updateTask(item, { bucket: 'planned', planned_for: targetDate }, status === 'today' ? '安排到今天' : '安排到明天')
    triageStats.planned++
  } else if (status === 'someday') {
    await moveTaskBucket(item, 'someday')
    triageStats.someday++
  } else if (status === 'discard') {
    const snapshot = await snapshotTaskForUndo(item.id)
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${item.id}`, {
      method: 'DELETE'
    })
    await refreshBoard()
    showUndoSnackbar({
      kind: 'task',
      message: `已丢弃: ${snapshot?.title || '事项'}`,
      snapshot,
      onUndo: () => restoreTask(snapshot)
    })
    triageStats.discarded++
  }

  const items = board.value.inbox_items
  if (triageIndex.value < items.length - 1) {
    triageIndex.value++
  } else {
    triageActive.value = false
    triageCompleted.value = true
  }
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
  loadRoutines()
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
    syncCompletionLogFromBoard()
    if (notificationEnabled.value) checkReminders()
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    timelineLoading.value = false
  }
  // 跨天延续:仅在首次加载时检查(避免每次刷新都弹)
  if (!crossDayHintChecked.value) {
    maybeShowCrossDayHint()
    crossDayHintChecked.value = true
  }
  writeLastSessionMemory()
  loadMeetings()
}
const crossDayHintChecked = ref(false)

function syncCompletionLogFromBoard() {
  const today = plannerTodayString()
  const doneToday = board.value.recovery?.done_today || 0
  if (doneToday > 0) {
    const current = completionLog.value[today] || 0
    if (doneToday > current) {
      completionLog.value = { ...completionLog.value, [today]: doneToday }
      saveCompletionLog()
    }
  }
  const recentDays = new Set()
  recentDays.add(today)
  const items = board.value.recent_items || []
  items.forEach((it) => {
    if (!it.completed_at) return
    const date = it.completed_at.slice(0, 10)
    if (date) recentDays.add(date)
  })
  if (recentDays.size === 0) return
  const counts = {}
  items.forEach((it) => {
    if (!it.completed_at) return
    const date = it.completed_at.slice(0, 10)
    if (!date) return
    counts[date] = (counts[date] || 0) + 1
  })
  let changed = false
  const next = { ...completionLog.value }
  Object.entries(counts).forEach(([date, count]) => {
    if ((next[date] || 0) < count) {
      next[date] = count
      changed = true
    }
  })
  if (changed) {
    completionLog.value = next
    saveCompletionLog()
  }
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
  // 阶段 5 减法 (iter 24):多任务 1 步记录
  // 检测到 2+ 段 → 走 batch;1 段 → 走原单条(零回归)
  const titles = quickFormBatchTitles.value
  if (titles.length >= 2) {
    return createQuickTasksBatch(titles)
  }
  // 智能归类优先级(高 → 低):
  //   1) 显式 quickNaturalDate 字段(用户在 高级设置 主动设的)
  //   2) 标题/详情里自然语言识别
  //   3) 现有表单值(用户已手动改)
  const hints = quickFormHints.value
  if (quickNaturalDate.value && quickNaturalDate.value.trim()) {
    const r = _previewNaturalDate(quickNaturalDate.value)
    if (r) {
      if (!quickForm.plannedFor) quickForm.plannedFor = r.date
      if (!quickForm.remindAt) {
        quickForm.remindAt = r.time ? `${r.date}T${r.time}` : `${r.date}T09:00`
      }
    }
  } else if (hints) {
    // 智能归类:不覆盖用户已设置的值
    if (!quickForm.plannedFor && hints.date) {
      quickForm.plannedFor = hints.date
    }
    if (!quickForm.remindAt && hints.date) {
      quickForm.remindAt = hints.time ? `${hints.date}T${hints.time}` : `${hints.date}T09:00`
    }
    if (quickForm.priority === 'medium' && hints.priority && hints.priority !== 'medium') {
      // 默认是 medium,只在用户没主动改过时升级
      quickForm.priority = hints.priority
    }
    // entryType / bucket 在发送时再覆盖
  }
  savingQuick.value = true
  try {
    // 智能归类:有具体时间 → event;无具体时间 → task
    // bucket:有日期 → planned(除非 hint 说 someday);无日期 → inbox
    const hints = quickFormHints.value
    const inferredEntryType = hints?.entryType || quickForm.entryType
    const inferredBucket = hints?.bucket || quickForm.bucket
    // 把识别出的关键词从标题里剥掉,让标题更干净
    const cleanTitle = hints
      ? quickForm.title
          .replace(/\s*(今天|明天|后天|大后天|下周[一星]?[一二三四五六日天]?|周[一二三四五六日天]|上午|下午|早上|晚上|傍晚|夜里|夜间|am|pm|\d{1,2}[:点]\d{0,2}|\d{1,2}点\d{0,2}分?|\d+天[之以]?后|\d+周[之以]?后|\d+个?月[之以]?后|\d{1,2}月\d{1,2}[日号]?|\d{4}[-/.]\d{1,2}[-/.]\d{1,2}|\d{1,2}[\/\-]\d{1,2})\s*/gi, ' ')
          .replace(/\s*(🚨|!!|!|紧急|十万火急|非常紧急|重要|高优[先级]?|低优[先级]?|不急|低优先级|有空再说|缓一缓|改天|以后再说|回头)\s*/gi, ' ')
          .replace(/\s+/g, ' ')
          .trim() || quickForm.title
      : quickForm.title
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks`, {
      method: 'POST',
      body: JSON.stringify({
        kind: activeKind.value,
        entry_type: inferredEntryType,
        bucket: inferredEntryType === 'event' ? 'planned' : inferredBucket,
        title: cleanTitle,
        detail: quickForm.detail,
        raw_text: quickForm.rawText,
        planned_for: quickForm.plannedFor,
        remind_at: quickForm.remindAt,
        repeat_type: quickForm.repeatType,
        repeat_interval: quickForm.repeatInterval,
        repeat_until: quickForm.repeatUntil,
        priority: quickForm.priority,
        notify_email: quickForm.notifyEmail,
        intent: quickForm.intent,
        energy_level: quickForm.energy_level
      })
    })
    ElMessage.success(activeKind.value === 'work' ? '工作事项已记录' : '生活事项已记录')
    // 自动保存为模板(只保留"常用模式"——标题去重后前 8 条)
    pushQuickTemplate({
      title: quickForm.title,
      priority: quickForm.priority,
      kind: activeKind.value,
      entryType: quickForm.entryType,
      bucket: quickForm.bucket
    })
    fillTodayDefaults()
    await refreshBoard()
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    savingQuick.value = false
  }
}

// 阶段 5 减法 (iter 24):多任务 1 步记录 — 调 /tasks/batch 一次创建多条
// 每条独立跑 extractHintsFromText → 智能归类(date/priority/bucket/entryType)
// 共享 quickForm 的 notifyEmail/energy_level/repeat 等"全局默认"
// 失败时用后端 failed_index 精确定位哪条挂了
async function createQuickTasksBatch(titles) {
  if (!titles || titles.length === 0) return
  if (titles.length > 50) {
    ElMessage.warning('一次最多写入 50 条,已截取前 50 条')
    titles = titles.slice(0, 50)
  }
  // 每条独立归类(不污染 quickForm)
  const tasksPayload = titles.map(rawText => {
    const hints = extractHintsFromText(rawText)
    // 清洗标题:剥离识别出的时间/日期/优先级关键词
    const cleanTitle = rawText
      .replace(/\s*(今天|明天|后天|大后天|下周[一星]?[一二三四五六日天]?|周[一二三四五六日天]|上午|下午|早上|晚上|傍晚|夜里|夜间|am|pm|\d{1,2}[:点]\d{0,2}|\d{1,2}点\d{0,2}分?|\d+天[之以]?后|\d+周[之以]?后|\d+个?月[之以]?后|\d{1,2}月\d{1,2}[日号]?|\d{4}[-/.]\d{1,2}[-/.]\d{1,2}|\d{1,2}[\/\-]\d{1,2})\s*/gi, ' ')
      .replace(/\s*(🚨|!!|!|紧急|十万火急|非常紧急|重要|高优[先级]?|低优[先级]?|不急|低优先级|有空再说|缓一缓|改天|以后再说|回头)\s*/gi, ' ')
      .replace(/\s+/g, ' ')
      .trim() || rawText
    const date = hints.date
    const time = hints.time
    const entryType = hints.entryType || quickForm.entryType || 'task'
    const bucket = entryType === 'event' ? 'planned' : (hints.bucket || quickForm.bucket || 'planned')
    const priority = hints.priority || quickForm.priority || 'medium'
    const plannedFor = date || quickForm.plannedFor || ''
    const remindAt = date
      ? (time ? `${date}T${time}` : `${date}T09:00`)
      : (quickForm.remindAt || '')
    return {
      kind: activeKind.value,
      entry_type: entryType,
      bucket,
      title: cleanTitle,
      detail: '',
      raw_text: rawText,
      planned_for: plannedFor,
      remind_at: remindAt,
      repeat_type: 'none',
      repeat_interval: 1,
      repeat_until: '',
      priority,
      notify_email: quickForm.notifyEmail,
      intent: '',
      energy_level: quickForm.energy_level || ''
    }
  })
  savingQuick.value = true
  try {
    const resp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/batch`, {
      method: 'POST',
      body: JSON.stringify({ tasks: tasksPayload })
    })
    const data = await resp.json().catch(() => ({}))
    const count = data?.created_count || tasksPayload.length
    // 成功反馈:列出前 3 条标题,过长省略
    const sampleTitles = tasksPayload.slice(0, 3).map(t => t.title)
    const tail = tasksPayload.length > 3 ? ` 等 ${tasksPayload.length} 条` : ''
    ElMessage.success({
      message: `已记录 ${count} 条:${sampleTitles.join(' / ')}${tail}`,
      duration: 2500
    })
    fillTodayDefaults()
    await refreshBoard()
  } catch (error) {
    // 后端会在 400 响应里回 failed_index 和具体错误
    const msg = error?.message || '批量保存失败'
    ElMessage.error(msg)
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
  taskForm.rawText = task.raw_text || ''
  taskForm.notifyEmail = task.notify_email || ''
  taskForm.intent = task.intent || ''
  taskForm.energyLevel = task.energy_level || ''
  taskForm.cancelReason = task.cancel_reason || ''
  taskForm.postponeReason = task.last_postpone_reason || ''
  loadTaskCommentsPreview(task.id)
  loadTaskActivities(task.id)
  taskDrawerVisible.value = true
}

async function copyTask(task) {
  if (!task || !task.id) return
  const today = new Date().toISOString().slice(0, 10)
  const payload = {
    kind: task.kind,
    entry_type: task.entry_type || 'task',
    bucket: task.bucket || 'planned',
    title: task.title,
    detail: task.detail || '',
    notes: task.notes || '',
    priority: task.priority || 'medium',
    status: 'open',
    planned_for: today,
    raw_text: task.raw_text || '',
    notify_email: task.notify_email || '',
    intent: task.intent || '',
    energy_level: task.energy_level || ''
  }
  try {
    const res = await fetch(`${API_BASE}/profile/${profileId.value}/tasks?creator_key=${encodeURIComponent(creatorKey.value)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `复制失败: HTTP ${res.status}`)
    }
    const data = await res.json()
    ElMessage.success('已复制, 在新草稿中继续编辑')
    await refreshBoard()
    if (data && data.id) {
      const fresh = (board.value.recent_items || []).concat(board.value.inbox_items || [], board.value.someday_items || [])
        .concat((board.value.groups || []).flatMap(g => g.items || []))
        .concat((board.value.event_groups || []).flatMap(g => g.items || []))
        .find(t => t.id === data.id)
      if (fresh) openTask(fresh)
    }
  } catch (error) {
    ElMessage.error(error.message || '复制失败')
  }
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
        raw_text: taskForm.rawText,
        planned_for: taskForm.plannedFor,
        remind_at: taskForm.remindAt,
        repeat_type: taskForm.repeatType,
        repeat_interval: taskForm.repeatInterval,
        repeat_until: taskForm.repeatUntil,
        priority: taskForm.priority,
        status: taskForm.status,
        notify_email: taskForm.notifyEmail,
        cancel_reason: taskForm.cancelReason,
        postpone_reason: taskForm.postponeReason,
        intent: taskForm.intent,
        energy_level: taskForm.energyLevel
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
    await ElMessageBox.confirm('删除后该事项和评论都会清空,5 秒内可在底部撤销。', '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '再想想'
    })
    const taskIdToDelete = taskForm.id
    const taskSnapshot = await snapshotTaskForUndo(taskIdToDelete)
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskIdToDelete}`, {
      method: 'DELETE'
    })
    taskDrawerVisible.value = false
    await refreshBoard()
    showUndoSnackbar({
      kind: 'task',
      message: `已删除: ${taskSnapshot?.title || '事项'}`,
      snapshot: taskSnapshot,
      onUndo: () => restoreTask(taskSnapshot)
    })
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

// 阶段 5 1 步化:从卡片直接删除,无二次确认,撤销 toast 兜底。
// 撤销比确认更温柔 — 用户不会被"警告"吓到,5 秒内可改主意。
async function quickDeleteTask(task) {
  if (!task || !task.id) return
  const taskId = task.id
  const snapshot = await snapshotTaskForUndo(taskId)
  if (!snapshot) {
    ElMessage.error('无法生成删除快照,请稍后再试')
    return
  }
  try {
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskId}`, {
      method: 'DELETE'
    })
    await refreshBoard()
    showUndoSnackbar({
      kind: 'task',
      message: `已删除: ${snapshot.title || '事项'}`,
      snapshot,
      onUndo: () => restoreTask(snapshot)
    })
  } catch (error) {
    ElMessage.error(error.message || '删除失败')
  }
}

function showUndoSnackbar({ message, onUndo, kind = 'task' }) {
  if (undoTimer) clearTimeout(undoTimer)
  if (undoTickTimer) clearInterval(undoTickTimer)
  undoState.value = { visible: true, message, kind, secondsLeft: 5, onUndo }
  undoTickTimer = setInterval(() => {
    if (undoState.value.secondsLeft > 0) {
      undoState.value.secondsLeft--
    } else {
      clearInterval(undoTickTimer)
      undoTickTimer = null
    }
  }, 1000)
  undoTimer = setTimeout(() => {
    undoState.value.visible = false
    undoState.value.onUndo = null
    undoTimer = null
  }, 5500)
}

function dismissUndo() {
  if (undoTimer) clearTimeout(undoTimer)
  if (undoTickTimer) clearInterval(undoTickTimer)
  undoState.value = { visible: false, message: '', kind: '', secondsLeft: 0, onUndo: null }
  undoTimer = null
  undoTickTimer = null
}

let quickAddTitleRef = null
function bindQuickAddInput(el) {
  quickAddTitleRef = el
}
function focusQuickAdd() {
  fillTodayDefaults()
  quickAdvancedVisible.value = false
  nextTick(() => {
    let input = null
    if (quickAddTitleRef) {
      input = quickAddTitleRef.$el ? quickAddTitleRef.$el.querySelector('input') : quickAddTitleRef
    }
    if (!input) {
      input = document.querySelector('.quick-grid-core input')
    }
    if (input) {
      input.focus()
      input.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  })
}

function focusGlobalSearch() {
  nextTick(() => {
    const el = globalSearchInput.value
    if (el) {
      el.focus()
      el.select()
    }
  })
}

function isEditableTarget(target) {
  if (!target) return false
  const tag = (target.tagName || '').toLowerCase()
  if (tag === 'input' || tag === 'textarea' || tag === 'select') return true
  if (target.isContentEditable) return true
  return false
}

function closeAllDrawers() {
  settingsVisible.value = false
  taskDrawerVisible.value = false
  commentDrawerVisible.value = false
  activityDrawerVisible.value = false
  aiDialogVisible.value = false
  adviceDialogVisible.value = false
  adminDialogVisible.value = false
  adminDetailVisible.value = false
  reviewDialogVisible.value = false
  meetingDialogVisible.value = false
  shortcutHelpVisible.value = false
  closeSearch()
}

function toggleShortcutHelp() {
  shortcutHelpVisible.value = !shortcutHelpVisible.value
}

const shortcutHints = [
  { keys: ['⌘', 'K'], label: '全局快速新建（任意位置）' },
  { keys: ['n'], label: '快速新建事项' },
  { keys: ['/'], label: '聚焦搜索框' },
  { keys: ['1', '2', '3'], label: '处理收件箱:1 计划 / 2 放一放 / 3 不做' },
  { keys: ['Esc'], label: '关闭所有抽屉/弹窗(triage 模式下退出处理)' },
  { keys: ['?'], label: '打开/关闭快捷键说明' },
]

let globalKeydownHandler = null
function handleGlobalKeydown(event) {
  if (!event) return
  const key = event.key
  const target = event.target
  const inEditable = isEditableTarget(target)

  // Cmd/Ctrl + K: 全局快速添加(优先于 editable 检查,即使焦点在 input 也能触发)
  if ((event.metaKey || event.ctrlKey) && (key === 'k' || key === 'K')) {
    event.preventDefault()
    openGlobalQuickAdd()
    return
  }
  if (key === 'Escape') {
    if (globalQuickAddVisible.value) {
      closeGlobalQuickAdd()
      return
    }
    if (triageActive.value) {
      // 阶段 5:在 triage 模式下 Esc 退出处理,而不是关掉所有 drawer
      triageActive.value = false
      return
    }
    closeAllDrawers()
    return
  }
  if (inEditable) return
  if (event.metaKey || event.ctrlKey || event.altKey) return

  // 阶段 5:triage 模式快捷键 — 1=计划,2=放一放,3=不做,让用户连续处理收件箱像流水线
  if (triageActive.value && triageCurrent.value) {
    if (key === '1') {
      event.preventDefault()
      triageDecide('planned')
      return
    }
    if (key === '2') {
      event.preventDefault()
      triageDecide('someday')
      return
    }
    if (key === '3') {
      event.preventDefault()
      triageDecide('discard')
      return
    }
  }

  if (key === '/') {
    event.preventDefault()
    focusGlobalSearch()
    return
  }
  if (key === '?') {
    event.preventDefault()
    toggleShortcutHelp()
    return
  }
  if (key === 'n' || key === 'N') {
    if (!profileId.value) return
    event.preventDefault()
    openGlobalQuickAdd()
    return
  }
}

const searchResults = computed(() => {
  const q = (searchKeyword.value || '').trim().toLowerCase()
  if (!q) return []
  const results = []
  const seen = new Set()
  const addItem = (t) => {
    if (!t || !t.id || seen.has(t.id)) return
    const title = (t.title || '').toLowerCase()
    const detail = (t.detail || '').toLowerCase()
    const notes = (t.notes || '').toLowerCase()
    const raw = (t.raw_text || '').toLowerCase()
    if (title.includes(q) || detail.includes(q) || notes.includes(q) || raw.includes(q)) {
      seen.add(t.id)
      results.push(t)
    }
  }
  for (const g of board.value.groups || []) for (const t of (g.items || [])) addItem(t)
  for (const g of board.value.event_groups || []) for (const t of (g.items || [])) addItem(t)
  for (const t of (board.value.inbox_items || [])) addItem(t)
  for (const t of (board.value.someday_items || [])) addItem(t)
  for (const t of (board.value.recent_items || [])) addItem(t)
  return results
})

function highlightMatch(text, q) {
  if (!text || !q) return escapeHtml(text || '')
  const safe = escapeHtml(text)
  const escapedQ = q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const re = new RegExp(`(${escapedQ})`, 'gi')
  return safe.replace(re, '<mark class="search-hit">$1</mark>')
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function searchMoveSelection(delta) {
  const max = Math.min(searchResults.value.length, 8)
  if (max === 0) return
  searchSelectedIndex.value = (searchSelectedIndex.value + delta + max) % max
}

function searchOpenSelected() {
  const list = searchResults.value.slice(0, 8)
  const item = list[searchSelectedIndex.value]
  if (item) openSearchResult(item)
  else searchCreateFromKeyword()
}

function openSearchResult(item) {
  closeSearch()
  openTask(item)
}

function clearSearch() {
  searchKeyword.value = ''
  searchSelectedIndex.value = 0
}

function closeSearch() {
  searchOpen.value = false
  searchSelectedIndex.value = 0
}

// 阶段 5 减法 (iter 26):搜索无结果时 Enter / 「快速创建」→ 1 步直接创建
// 原 2 步流程:Enter → 填进表单 → 手动点保存(用户可能漏掉,form 残留干扰下次输入)
// 优化后:Enter → 智能归类 + 调 /tasks/batch(单条也走 batch 节省代码) + 5s undo 兜底
// 多任务支持(跟 iter 24 主页面板对齐):「买菜 / 给妈打电话」一次建 2 条
async function searchCreateFromKeyword() {
  const title = (searchKeyword.value || '').trim()
  if (!title) return
  const keyword = title
  closeSearch()
  // 跟 iter 24 主页面板一致:支持 / 分隔多任务
  const titles = splitQuickTitles(keyword)
  if (titles.length === 0) return
  // 每条独立跑 extractHintsFromText 智能归类(日期/优先级/bucket/entryType)
  const tasksPayload = titles.map(rawText => {
    const hints = extractHintsFromText(rawText)
    const cleanTitle = rawText
      .replace(/\s*(今天|明天|后天|大后天|下周[一星]?[一二三四五六日天]?|周[一二三四五六日天]|上午|下午|早上|晚上|傍晚|夜里|夜间|am|pm|\d{1,2}[:点]\d{0,2}|\d{1,2}点\d{0,2}分?|\d+天[之以]?后|\d+周[之以]?后|\d+个?月[之以]?后|\d{1,2}月\d{1,2}[日号]?|\d{4}[-/.]\d{1,2}[-/.]\d{1,2}|\d{1,2}[\/\-]\d{1,2})\s*/gi, ' ')
      .replace(/\s*(🚨|!!|!|紧急|十万火急|非常紧急|重要|高优[先级]?|低优[先级]?|不急|低优先级|有空再说|缓一缓|改天|以后再说|回头)\s*/gi, ' ')
      .replace(/\s+/g, ' ')
      .trim() || rawText
    const date = hints.date
    const time = hints.time
    const entryType = hints.entryType || 'task'
    const bucket = entryType === 'event' ? 'planned' : (hints.bucket || 'planned')
    const priority = hints.priority || 'medium'
    return {
      kind: activeKind.value,
      entry_type: entryType,
      bucket,
      title: cleanTitle,
      detail: '',
      raw_text: rawText,
      planned_for: date || '',
      remind_at: date ? (time ? `${date}T${time}` : `${date}T09:00`) : '',
      repeat_type: 'none',
      repeat_interval: 1,
      repeat_until: '',
      priority,
      notify_email: '',
      intent: '',
      energy_level: ''
    }
  })
  try {
    const resp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/batch`, {
      method: 'POST',
      body: JSON.stringify({ tasks: tasksPayload })
    })
    const data = await resp.json().catch(() => ({}))
    const count = data?.created_count || tasksPayload.length
    if (count === 1) {
      const t = tasksPayload[0]
      const where = t.planned_for ? ` · 计划 ${t.planned_for}` : (t.bucket === 'someday' ? ' · 放一放' : '')
      ElMessage.success({ message: `已创建: ${t.title}${where}`, duration: 2000 })
    } else {
      const sample = tasksPayload.slice(0, 2).map(t => t.title).join(' / ')
      const tail = tasksPayload.length > 2 ? ` 等 ${tasksPayload.length} 条` : ''
      ElMessage.success({ message: `已创建 ${count} 条:${sample}${tail}`, duration: 2500 })
    }
    await refreshBoard()
  } catch (error) {
    ElMessage.error(error?.message || '创建失败')
  }
}

function findTaskInBoard(taskId) {
  if (!taskId) return null
  const all = []
  for (const g of board.value.groups || []) all.push(...(g.items || []))
  for (const g of board.value.event_groups || []) all.push(...(g.items || []))
  all.push(...(board.value.inbox_items || []))
  all.push(...(board.value.someday_items || []))
  all.push(...(board.value.recent_items || []))
  return all.find(t => t && t.id === taskId) || null
}

function openServerSearchResult(hit) {
  if (!hit || !hit.task_id) return
  const task = findTaskInBoard(hit.task_id)
  if (!task) {
    ElMessage.warning('任务不在当前视图,尝试刷新页面或切换时间范围')
    return
  }
  closeSearch()
  openTask(task)
  if (hit.match_kind === 'comment' && hit.source_id) {
    setTimeout(() => highlightCommentById(hit.source_id), 400)
  } else if (hit.match_kind === 'recording' && hit.source_id) {
    setTimeout(() => highlightRecordingById(hit.source_id), 400)
  }
}

function highlightCommentById(commentId) {
  const el = document.querySelector(`[data-comment-id="${commentId}"]`)
  if (!el) return
  el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  el.classList.add('comment-highlight-flash')
  setTimeout(() => el.classList.remove('comment-highlight-flash'), 1600)
}

function highlightRecordingById(recordingId) {
  const el = document.querySelector(`[data-recording-id="${recordingId}"]`)
  if (!el) return
  el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  el.classList.add('comment-highlight-flash')
  setTimeout(() => el.classList.remove('comment-highlight-flash'), 1600)
}

function matchKindLabel(kind) {
  if (kind === 'comment') return '评论'
  if (kind === 'recording') return '录音'
  return '任务'
}

function matchKindTagType(kind) {
  if (kind === 'comment') return 'success'
  if (kind === 'recording') return 'warning'
  return ''
}

function runServerSearch(keyword) {
  const q = (keyword || '').trim()
  if (serverSearchTimer) {
    clearTimeout(serverSearchTimer)
    serverSearchTimer = null
  }
  if (!q || !profileId.value) {
    serverSearchResults.value = []
    serverSearchLoading.value = false
    return
  }
  serverSearchLoading.value = true
  const mySeq = ++serverSearchSeq
  serverSearchTimer = setTimeout(async () => {
    try {
      const resp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/search?q=${encodeURIComponent(q)}&limit=30`)
      const data = await resp.json()
      // 防止过期请求覆盖新结果
      if (mySeq !== serverSearchSeq) return
      serverSearchResults.value = data.hits || []
    } catch (error) {
      if (mySeq !== serverSearchSeq) return
      serverSearchResults.value = []
    } finally {
      if (mySeq === serverSearchSeq) serverSearchLoading.value = false
    }
  }, 280)
}

watch(searchKeyword, () => {
  searchSelectedIndex.value = 0
  if (searchKeyword.value) searchOpen.value = true
  runServerSearch(searchKeyword.value)
})

async function handleUndo() {
  const cb = undoState.value.onUndo
  dismissUndo()
  if (typeof cb === 'function') {
    await cb()
  }
}

// 阶段 6 减法:完成撤销 — 让"误点完成"也能 1 步撤回到 open。
// 哲学:删除有 undo,完成也该有 — 不对称会让用户对"完成"操作产生不安全感。
// 复用 backend 的 UpdateTask(status=open) 路径:
//   - CompletedAt 会被后端自动置 nil(default 分支)
//   - CancelReason 会被清空
//   - CompletionCount 不重置(代表历史真实完成的次数,符合数据平滑原则)
async function undoCompleteTask(task) {
  if (!task || !task.id) return
  const taskId = task.id
  // 撤回时立即隐藏可能正在飘出的"感觉怎么样"卡,避免和撤回 toast 撞车
  dismissCompletionFeedback()
  try {
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskId}`, {
      method: 'PUT',
      body: JSON.stringify({ status: 'open' })
    })
    await refreshBoard()
    if (!quietMode.value) {
      ElMessage({ message: '已撤回,回到待办', type: 'info', duration: 1800 })
    }
  } catch (error) {
    ElMessage.error(error.message || '撤回失败')
  }
}

onBeforeUnmount(() => {
  if (undoTimer) clearTimeout(undoTimer)
  if (undoTickTimer) clearInterval(undoTickTimer)
})

async function snapshotTaskForUndo(taskId) {
  if (!taskId) return null
  try {
    const taskResp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskId}`)
    const taskData = await taskResp.json()
    const task = taskData.task
    if (!task) return null
    let comments = []
    try {
      const cResp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskId}/comments`)
      const cData = await cResp.json()
      comments = (cData.comments || []).map(c => ({ content: c.content, author: c.author }))
    } catch {}
    return {
      kind: task.kind,
      entry_type: task.entry_type,
      bucket: task.bucket,
      title: task.title,
      detail: task.detail,
      notes: task.notes,
      priority: task.priority,
      status: task.status,
      planned_for: task.planned_for,
      remind_at: task.remind_at,
      repeat_type: task.repeat_type,
      repeat_interval: task.repeat_interval,
      repeat_until: task.repeat_until,
      notify_email: task.notify_email,
      intent: task.intent,
      energy_level: task.energy_level,
      raw_text: task.raw_text,
      cancel_reason: task.cancel_reason,
      comments
    }
  } catch {
    return null
  }
}

async function restoreTask(snapshot) {
  if (!snapshot) return
  try {
    const payload = { ...snapshot }
    delete payload.comments
    const resp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks`, {
      method: 'POST',
      body: JSON.stringify(payload)
    })
    const data = await resp.json()
    const newId = data.task?.id
    if (newId && Array.isArray(snapshot.comments)) {
      for (const c of snapshot.comments) {
        try {
          await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${newId}/comments`, {
            method: 'POST',
            body: JSON.stringify({ content: c.content, author: c.author })
          })
        } catch {}
      }
    }
    await refreshBoard()
    ElMessage.success('已恢复: ' + snapshot.title)
  } catch (error) {
    ElMessage.error('恢复失败: ' + (error.message || '未知错误'))
  }
}

async function updateTask(task, payload, successMessage = '') {
  try {
    const resp = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${task.id}`, {
      method: 'PUT',
      body: JSON.stringify(payload)
    })
    const body = await resp.json().catch(() => ({}))
    const updated = body?.task || task
    // 阶段 5 减法:重复任务完成时,后端会建下次实例并回传 next_planned_for
    // 我们用这个信息在现有 toast 后面追加一句"下次自动建到 X"
    const nextPlannedFor = body?.next_planned_for || ''
    // 阶段 5 减法 (iter 20):族系完成次数,用于 toast 展示"第 N 次完成"
    const familyCount = body?.family_completion_count || 0
    const isCompletion = payload.status === 'done' && task.status !== 'done'
    if (isCompletion) {
      const count = updated.completion_count || 0
      // 取更新后的今日完成数 (= 已 done_today + 1,因为我们刚刚完成了一件)
      const todayDone = (board.value.recovery?.done_today || 0) + 1
      const rhythmTail = todayDone >= 3
        ? `今天已经做完 ${todayDone} 件,节奏不错`
        : todayDone >= 1
          ? `今日 ${todayDone} 件,慢慢来`
          : '今日已收尾'
      if (count >= 2) {
        ElMessage({
          message: `今日 ${todayDone} 件 · 这是第 ${count} 次`,
          type: 'success',
          duration: 2400
        })
      } else {
        ElMessage({
          message: rhythmTail,
          type: 'success',
          duration: 2000
        })
      }
      // 阶段 5 减法 (iter 19+20):重复任务完成时,后端自动建了下次实例
      // 在收尾 toast 飘完后追加"第 N 次完成 · 下次自动建到 X 月 X 日"
      // 让用户知道"系统已替你把下次安排好,这是你坚持第几次",0 操作成本
      if (nextPlannedFor && !quietMode.value) {
        setTimeout(() => {
          const familyTail = familyCount > 1 ? `第 ${familyCount} 次完成 · ` : ''
          ElMessage({
            message: `🔁 ${familyTail}下次自动建到 ${formatShortDate(nextPlannedFor)}`,
            type: 'info',
            duration: 2800
          })
        }, 2200)
      }
    } else if (successMessage) {
      ElMessage.success(successMessage)
    }
    await refreshBoard()
    if (isCompletion) {
      recordCompletionToday()
      await maybeShowNextUp(task)
      // 阶段 6 减法:完成 1 步撤销 — 复用现有 undoState 系统,
      // 5 秒内可 1 步改回 open,补完"完成"操作的对称安全感
      // (删除有 undo,完成也该有,这才一致)
      showUndoSnackbar({
        kind: 'task',
        message: `已完成: ${task.title || '事项'}`,
        onUndo: () => undoCompleteTask(updated)
      })
      // 阶段 6:完成反馈卡 — 在 toast 和"下一项"消息都飘完后,延时 1.4s 出现
      // 让"感觉怎么样"的温柔追问不打断"刚做完"的爽感节奏
      if (!task.completion_feeling) {
        setTimeout(() => showCompletionFeedback(updated), 1400)
      }
    }
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function maybeShowNextUp(completedTask) {
  if (quietMode.value) return
  // 阶段 6 减法:如果用户在 700ms 等待窗口内撤销了完成,
  // 撤回后 board 已刷新,任务不再是 done — 此时不应再推"下一项",
  // 否则会出现"刚撤回"和"接下来推荐"两条冲突消息
  if (completedTask && completedTask.id) {
    const current = findTaskById(completedTask.id)
    if (current && current.status !== 'done') return
  }
  await nextTick()
  setTimeout(() => {
    const openTasks = []
    board.value.groups?.forEach((g) => {
      (g.items || []).forEach((t) => {
        if (t.status !== 'done' && t.status !== 'cancelled') openTasks.push(t)
      })
    })
    // 阶段 5:如果任务有"语境化文案"(拖了很久/顺延多次),优先展示这条,
    // 不再叠加"推荐下一项",避免一次完成刷两条 toast 干扰
    const contextMsg = completionContextMessage(completedTask)
    if (contextMsg) {
      ElMessage({ message: contextMsg, type: 'success', duration: 2600 })
      return
    }
    if (openTasks.length === 0) {
      const recentDone = (board.value.recovery?.done_today || 0)
      if (recentDone >= 1) {
        ElMessage({
          message: '今天主推进的待办都清完了，给自己留点空吧。',
          type: 'success',
          duration: 3200,
          showClose: true
        })
      }
      return
    }
    // 低能量模式:优先选事务性/未标的,跳过需专注/外出/创意的
    let next
    if (lowEnergyMode.value) {
      const easyPool = openTasks.filter((t) => !t.energy_level || t.energy_level === 'shallow')
      next = easyPool[0] || openTasks[0]
    } else {
      next = openTasks[0]
    }
    if (!next) return
    const nextMsg = lowEnergyMode.value
      ? `已完成「${completedTask.title}」 · 简单一点的下一项:「${next.title}」`
      : `已完成「${completedTask.title}」, 接下来推荐「${next.title}」`
    ElMessage({
      message: nextMsg,
      type: 'info',
      duration: 3500,
      showClose: true
    })
    // 滚动到下一件 + 短暂高亮
    nextTick(() => {
      const el = document.querySelector(`[data-task-id="${next.id}"]`)
      if (el) {
        el.scrollIntoView({ behavior: 'smooth', block: 'center' })
        el.classList.add('task-card-next-up')
        setTimeout(() => el.classList.remove('task-card-next-up'), 1800)
      }
    })
  }, 700)
}

async function setTaskStatus(task, status) {
  if (status === 'cancelled') {
    await cancelTask(task)
    return
  }
  if (status === 'done') {
    triggerCelebration(task)
  }
  await updateTask(task, { status }, status === 'done' ? '已完成' : '状态已更新')
}

function nextStatusOf(status) {
  return { open: 'in_progress', in_progress: 'done', done: 'open', cancelled: 'open' }[status] || 'open'
}

function statusCycleHint(task) {
  const next = nextStatusOf(task.status)
  return `当前: ${statusLabel(task.status)} · 点击切换到 ${statusLabel(next)}`
}

async function cycleTaskStatus(task) {
  if (!task || !task.id) return
  if (task.status === 'cancelled') return
  const next = nextStatusOf(task.status)
  await setTaskStatus(task, next)
}

// 阶段 5 减法:任务卡片 ⋯ 菜单的统一分发。
// 收进来的低频操作:开始/结束专注、复制为新草稿、显式切到进行中、1 步删除
async function handleTaskMore(task, cmd) {
  if (!task || !cmd) return
  if (cmd === 'in_progress') {
    await setTaskStatus(task, 'in_progress')
  } else if (cmd === 'focus') {
    await startFocus(task)
  } else if (cmd === 'copy') {
    await copyTask(task)
  } else if (cmd === 'delete') {
    await quickDeleteTask(task)
  }
}

function triggerCelebration(task) {
  if (!task) return
  if (quietMode.value) return
  celebratedTaskIds.value = new Set([...celebratedTaskIds.value, task.id])
  setTimeout(() => {
    const next = new Set(celebratedTaskIds.value)
    next.delete(task.id)
    celebratedTaskIds.value = next
  }, 1600)
  const id = ++celebrationCounter
  celebrationBursts.value.push({ id, x: 50, y: 50, ts: Date.now() })
  setTimeout(() => {
    celebrationBursts.value = celebrationBursts.value.filter((b) => b.id !== id)
  }, 1400)
}

// 阶段 5 认知科学:返回任务的"语境化文案",用于完成时温柔反馈。
// 返回 null 时由 maybeShowNextUp 接管(用"推荐下一项"消息)。
function completionContextMessage(task) {
  if (!task) return null
  const rollover = task.rollover_count || 0
  if (rollover >= 2) {
    return `顺延 ${rollover} 次还是推完了,做得好。`
  }
  const ageMs = Date.now() - new Date(task.created_at || Date.now()).getTime()
  const ageDays = Math.floor(ageMs / (1000 * 60 * 60 * 24))
  if (ageDays >= 14) {
    return `拖了 ${ageDays} 天,今天落定了。`
  }
  if (ageDays >= 3) {
    return '用了一点时间,不过今天搞定了。'
  }
  return null
}

async function cancelTask(task) {
  // 哲学:取消不是"失败",只是"这件事现在不需要了"。给用户最轻松的 1 步交互。
  // iter 13:走 1 步 popover(由按钮或 CancelReasonPopover 调用);默认采用「不再需要」。
  await cancelTaskWithReason(task, '不再需要')
}

async function cancelTaskWithReason(task, reason) {
  // 1 步提交:取消 + 记录原因 + 温柔的反馈(按原因给一句话)。
  const finalReason = (reason && reason.trim()) || '不再需要'
  try {
    writeLastCancelReason(finalReason)
    await updateTask(task, { status: 'cancelled', cancel_reason: finalReason }, '已取消')
    if (!quietMode.value) {
      const feedback = cancelReasonFeedback(finalReason)
      if (feedback) ElMessage({ message: feedback, type: 'info', duration: 2200 })
    }
  } catch (error) {
    if (error.message) ElMessage.error(error.message || '取消失败')
  }
}

function cancelReasonFeedback(reason) {
  // 温柔的反馈:不评价、不催促、不打鸡血,只是确认"放下"这件事。
  const map = {
    '已委派给他人': '已委托出去,期待对方推进',
    '条件变化': '环境变了,放下也是智慧',
    '不是现在': '留个口子,以后想得起再回来',
    '不再需要': '放过不需要的事,留点精力给重要的'
  }
  return map[reason] || ''
}

function readLastCancelReason() {
  try {
    return localStorage.getItem('planner.lastCancelReason') || ''
  } catch {
    return ''
  }
}

function writeLastCancelReason(reason) {
  try {
    localStorage.setItem('planner.lastCancelReason', reason || '')
  } catch {}
}

async function postponeTask(task) {
  const base = task.display_date || task.planned_for || new Date().toISOString().slice(0, 10)
  const date = new Date(`${base}T00:00:00`)
  date.setDate(date.getDate() + 1)
  const next = date.toISOString().slice(0, 10)
  try {
    const { value } = await ElMessageBox.prompt(
      '顺延原因(可选,留空也行)',
      '顺延事项',
      {
        inputPlaceholder: '例如：今天优先级下调 / 等待他人反馈',
        inputValue: '今天优先级下调'
      }
    )
    await updateTask(task, { planned_for: next, status: 'open', postpone_reason: value || '手动顺延' }, '已顺延一天')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '顺延失败')
    }
  }
}

async function postponeToTomorrow(task) {
  const tomorrow = tomorrowDateString()
  if (!task || !task.id) return
  await updateTask(task, {
    planned_for: tomorrow,
    status: 'open',
    postpone_reason: '一键改到明天'
  }, `已改到 ${tomorrow}`)
}

async function postponeToNextMonday(task) {
  if (!task || !task.id) return
  const result = parseNaturalDate('下周一')
  if (!result) return
  await updateTask(task, {
    planned_for: result.date,
    status: 'open',
    postpone_reason: '一键改到下周一'
  }, `已改到 ${result.date}`)
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
  commentForm.imageUrls = []
  comments.value = []
  taskRecordings.value = []
  await loadComments(task.id)
  await loadTaskRecordings(task.id)
}

async function openActivityDrawer(task) {
  if (!task?.id) return
  await loadTaskActivities(task.id)
  activityDrawerVisible.value = true
}

async function loadComments(taskId) {
  try {
    const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskId}/comments`)
    const data = await response.json()
    comments.value = data.comments || []
    if (taskForm.id === taskId) {
      taskCommentsPreview.value = [...comments.value].slice(-3).reverse()
    }
  } catch (error) {
    comments.value = []
  }
}

async function postComment(taskId, content, imageUrls = []) {
  await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${taskId}/comments`, {
    method: 'POST',
    body: JSON.stringify({ content, image_urls: imageUrls })
  })
  await loadComments(taskId)
  await loadTaskActivities(taskId)
  await refreshBoard()
}

async function submitComment() {
  if (!currentCommentTask.value) return
  const text = commentForm.content.trim()
  const images = commentForm.imageUrls || []
  if (!text && images.length === 0) {
    ElMessage.warning('写一句或加张图再发送')
    return
  }
  savingComment.value = true
  try {
    await postComment(currentCommentTask.value.id, text, images)
    commentForm.content = ''
    commentForm.imageUrls = []
    ElMessage.success('评论已保存')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    savingComment.value = false
  }
}

function parseCommentImages(raw) {
  if (!raw) return []
  if (Array.isArray(raw)) return raw.filter(Boolean)
  try {
    const list = JSON.parse(raw)
    return Array.isArray(list) ? list.filter(Boolean) : []
  } catch (e) {
    return []
  }
}

function triggerCommentImageUpload() {
  if (commentImageInput.value) {
    commentImageInput.value.click()
  }
}

function removeDraftImage(idx) {
  if (!commentForm.imageUrls) return
  commentForm.imageUrls = commentForm.imageUrls.filter((_, i) => i !== idx)
}

async function uploadCommentImageFiles(files) {
  if (!files || files.length === 0) return 0
  uploadingCommentImage.value = true
  let added = 0
  try {
    const uploads = []
    for (const file of files) {
      if (!file.type || !file.type.startsWith('image/')) {
        ElMessage.warning(`已跳过非图片文件: ${file.name || '粘贴内容'}`)
        continue
      }
      if (file.size > 5 * 1024 * 1024) {
        ElMessage.warning(`图片过大(最大 5MB): ${file.name || '粘贴内容'}`)
        continue
      }
      const form = new FormData()
      form.append('file', file, file.name || `pasted-${Date.now()}.png`)
      const resp = await fetch('/api/paste/upload', {
        method: 'POST',
        body: form
      })
      const data = await resp.json()
      if (data && data.url) {
        uploads.push(data.url)
        added += 1
      }
    }
    if (uploads.length > 0) {
      commentForm.imageUrls = [...(commentForm.imageUrls || []), ...uploads]
      ElMessage.success(`已添加 ${uploads.length} 张图片`)
    }
  } catch (error) {
    ElMessage.error(error.message || '图片上传失败')
  } finally {
    uploadingCommentImage.value = false
  }
  return added
}

async function handleCommentImageSelect(event) {
  const files = Array.from(event.target?.files || [])
  await uploadCommentImageFiles(files)
  if (event.target) event.target.value = ''
}

async function handleCommentPaste(event) {
  if (!event || !event.clipboardData) return
  const items = Array.from(event.clipboardData.items || [])
  const imageFiles = items
    .filter((it) => it.kind === 'file' && it.type && it.type.startsWith('image/'))
    .map((it) => it.getAsFile())
    .filter(Boolean)
  if (imageFiles.length === 0) return
  event.preventDefault()
  const added = await uploadCommentImageFiles(imageFiles)
  if (added === 0) {
    ElMessage.warning('剪贴板中的图片未能上传')
  }
}

function recordingAudioUrl(rec) {
  if (!rec || !rec.audio_url) return ''
  const params = new URLSearchParams()
  if (profileId.value) params.set('profile_id', profileId.value)
  if (password.value) params.set('password', password.value)
  if (creatorKey.value) params.set('creator_key', creatorKey.value)
  const query = params.toString()
  return query ? `${rec.audio_url}?${query}` : rec.audio_url
}

async function loadTaskRecordings(taskId) {
  try {
    const response = await plannerFetch(`${API_BASE.replace('/planner', '')}/voicememo/task/${taskId}/list?profile_id=${profileId.value}`)
    const data = await response.json()
    taskRecordings.value = data.items || []
    if (taskRecordings.value.some(r => r.status === 'transcribing')) {
      startTaskRecordingPoll()
    }
  } catch (error) {
    // recordings are optional context; surface but don't block the drawer
    taskRecordings.value = []
  }
}

function toggleTaskRecording() {
  if (isTaskRecording.value) {
    stopTaskRecording()
  } else {
    startTaskRecording()
  }
}

const MAX_TASK_RECORDING_SECONDS = 5 * 60
// 防止续录失败时无限递归:连续失败超过这个次数就停下来,避免刷错误
const MAX_RECORDING_RESTARTS = 5

async function startTaskRecording() {
  if (!currentCommentTask.value) return
  if (taskMediaRecorder && taskMediaRecorder.state !== 'inactive') {
    ElMessage.warning('录音正在进行中')
    return
  }
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    const mimeType = MediaRecorder.isTypeSupported('audio/webm;codecs=opus')
      ? 'audio/webm;codecs=opus'
      : MediaRecorder.isTypeSupported('audio/webm')
        ? 'audio/webm'
        : 'audio/mp4'

    taskMediaRecorder = new MediaRecorder(stream, { mimeType })
    taskAudioChunks = []

    taskMediaRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) taskAudioChunks.push(e.data)
    }
    taskMediaRecorder.onerror = (e) => {
      console.error('[task recording] MediaRecorder error', e)
      ElMessage.error('录音出错,已自动停止')
      cleanupTaskRecordingStream()
      isTaskRecording.value = false
      if (taskRecordingTimer) {
        clearInterval(taskRecordingTimer)
        taskRecordingTimer = null
      }
    }
    taskMediaRecorder.onstop = async () => {
      cleanupTaskRecordingStream()
      const chunks = taskAudioChunks.slice()
      taskAudioChunks = []
      let uploadOk = false
      if (chunks.length > 0) {
        const blob = new Blob(chunks, { type: taskMediaRecorder?.mimeType || mimeType })
        uploadOk = await uploadTaskRecording(blob)
      }
      // 用户没主动停 + 还在录音状态 + 连续失败没超限 → 自动开下一段
      if (
        uploadOk &&
        taskAutoContinue.value &&
        currentCommentTask.value &&
        !userStoppedRecording.value &&
        taskRecordingRestartCount < MAX_RECORDING_RESTARTS
      ) {
        taskRecordingRestartCount += 1
        ElMessage.info(`已自动分段 (${taskRecordingRestartCount}/${MAX_RECORDING_RESTARTS})`)
        setTimeout(() => {
          // 二次确认:用户可能在这几毫秒里点了停止
          if (
            taskAutoContinue.value &&
            currentCommentTask.value &&
            !userStoppedRecording.value
          ) {
            startTaskRecording().catch((e) => {
              console.error('[task recording] auto-restart failed', e)
            })
          }
        }, 400)
      } else if (!uploadOk && taskAutoContinue.value && !userStoppedRecording.value) {
        // 上传失败时不再续录,但保持当前 UI 状态让用户能手动重试
        isTaskRecording.value = false
        userStoppedRecording.value = false
      } else {
        isTaskRecording.value = false
        userStoppedRecording.value = false
      }
    }

    taskMediaRecorder.start(1000)
    isTaskRecording.value = true
    taskRecordingTime.value = 0
    taskRecordingStartTime = Date.now()
    if (taskRecordingTimer) clearInterval(taskRecordingTimer)
    taskRecordingTimer = setInterval(() => {
      taskRecordingTime.value = (Date.now() - taskRecordingStartTime) / 1000
      // 5 分钟自动截断:到点就停(会触发 onstop → 上传 → 自动续录)
      if (taskRecordingTime.value >= MAX_TASK_RECORDING_SECONDS) {
        if (taskMediaRecorder && taskMediaRecorder.state !== 'inactive') {
          taskMediaRecorder.stop()
        }
      }
    }, 200)
  } catch (error) {
    console.error('[task recording] start failed', error)
    ElMessage.error('无法访问麦克风,请检查权限设置')
    isTaskRecording.value = false
  }
}

function cleanupTaskRecordingStream() {
  if (taskMediaRecorder && taskMediaRecorder.stream) {
    taskMediaRecorder.stream.getTracks().forEach(t => {
      try { t.stop() } catch (_) { /* ignore */ }
    })
  }
}

function stopTaskRecording() {
  userStoppedRecording.value = true
  taskRecordingRestartCount = 0
  if (taskMediaRecorder && taskMediaRecorder.state !== 'inactive') {
    try {
      taskMediaRecorder.stop()
    } catch (e) {
      console.error('[task recording] stop failed', e)
      // 异常时也要清理 UI
      cleanupTaskRecordingStream()
      isTaskRecording.value = false
      if (taskRecordingTimer) {
        clearInterval(taskRecordingTimer)
        taskRecordingTimer = null
      }
    }
  } else {
    isTaskRecording.value = false
    if (taskRecordingTimer) {
      clearInterval(taskRecordingTimer)
      taskRecordingTimer = null
    }
  }
}

async function uploadTaskRecording(blob) {
  const task = currentCommentTask.value
  if (!task) return false
  taskRecordingUploading.value = true
  try {
    const form = new FormData()
    form.append('file', blob, `task_${task.id}_${Date.now()}.webm`)
    form.append('profile_id', profileId.value)
    form.append('planner_task_id', task.id)
    form.append('duration', String(Math.round(taskRecordingTime.value)))
    await plannerFetch(`${API_BASE.replace('/planner', '')}/voicememo/upload`, {
      method: 'POST',
      body: form
    })
    await loadTaskRecordings(task.id)
    ElMessage.success('录音已关联到事项')
    return true
  } catch (error) {
    console.error('[task recording] upload failed', error)
    ElMessage.error('录音上传失败: ' + error.message)
    return false
  } finally {
    taskRecordingUploading.value = false
  }
}

const TRANSCRIPT_COLLAPSE_THRESHOLD = 120

function isTranscriptLong(rec) {
  return rec && rec.transcript && Array.from(rec.transcript).length > TRANSCRIPT_COLLAPSE_THRESHOLD
}

function toggleTranscript(id) {
  expandedTranscripts[id] = !expandedTranscripts[id]
}

async function summarizeTaskRecording(rec) {
  if (!rec || !rec.transcript) return
  summarizingRecordingId.value = rec.id
  try {
    const response = await plannerFetch(`${API_BASE.replace('/planner', '')}/voicememo/${rec.id}/summarize?profile_id=${profileId.value}`, {
      method: 'POST'
    })
    const data = await response.json()
    if (data.summary) {
      rec.summary = data.summary
      ElMessage.success('总结已生成')
    } else {
      ElMessage.warning('未能生成总结')
    }
  } catch (error) {
    ElMessage.error('总结失败: ' + error.message)
  } finally {
    summarizingRecordingId.value = ''
  }
}

async function transcribeTaskRecording(rec) {
  try {
    await plannerFetch(`${API_BASE.replace('/planner', '')}/voicememo/${rec.id}/transcribe?profile_id=${profileId.value}`, {
      method: 'POST'
    })
    rec.status = 'transcribing'
    startTaskRecordingPoll()
    ElMessage.success('已开始语音转文字…')
  } catch (error) {
    ElMessage.error(error.message)
  }
}

function startTaskRecordingPoll() {
  if (taskRecordingPollTimer) return
  taskRecordingPollTimer = setInterval(async () => {
    const task = currentCommentTask.value
    if (!task || !commentDrawerVisible.value) {
      clearInterval(taskRecordingPollTimer)
      taskRecordingPollTimer = null
      return
    }
    const pending = taskRecordings.value.filter(r => r.status === 'transcribing')
    if (pending.length === 0) {
      clearInterval(taskRecordingPollTimer)
      taskRecordingPollTimer = null
      return
    }
    await loadTaskRecordings(task.id)
  }, 3000)
}

async function recordingToComment(rec) {
  const text = (rec.transcript || '').trim()
  if (!text || !currentCommentTask.value) return
  try {
    await postComment(currentCommentTask.value.id, text)
    ElMessage.success('转写内容已写入评论')
  } catch (error) {
    ElMessage.error(error.message)
  }
}

function importCalendar(task) {
  if (!task || !task.id) return
  // 后端 ICS 端点用 creator_key 鉴权(password 只对档案级生效)
  const key = creatorKey.value || password.value
  if (!key) {
    ElMessage.warning('请先登录')
    return
  }
  const url = `${API_BASE}/profile/${profileId.value}/tasks/${task.id}/calendar?creator_key=${encodeURIComponent(key)}`
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
    nextTick(() => { meetingDirty.value = false })
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
    const meetingId = meetingForm.id
    const snapshot = buildMeetingSnapshotForUndo(meetingForm)
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/meetings/${meetingId}`, {
      method: 'DELETE'
    })
    meetingDialogVisible.value = false
    await loadMeetings()
    showUndoSnackbar({
      kind: 'meeting',
      message: `\u5df2\u5220\u9664: ${snapshot.title || '\u4f1a\u8bae'}`,
      onUndo: () => restoreMeeting(snapshot)
    })
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.message || '\u5220\u9664\u5931\u8d25')
  }
}

async function deleteMeetingConfirm(m) {
  try {
    await ElMessageBox.confirm(`\u786e\u5b9a\u5220\u9664\u300c${m.title}\u300d\uff1f`, '\u5220\u9664\u786e\u8ba4', { type: 'warning' })
    const snapshot = buildMeetingPayloadFromApi(m)
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/meetings/${m.id}`, {
      method: 'DELETE'
    })
    await loadMeetings()
    showUndoSnackbar({
      kind: 'meeting',
      message: `\u5df2\u5220\u9664: ${snapshot.title || '\u4f1a\u8bae'}`,
      onUndo: () => restoreMeeting(snapshot)
    })
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.message || '\u5220\u9664\u5931\u8d25')
  }
}

function buildMeetingSnapshotForUndo(form) {
  let participants = []
  let tags = []
  try { participants = JSON.parse(form.participantsText || '[]') } catch { participants = [] }
  try { tags = JSON.parse(form.tagsText || '[]') } catch { tags = [] }
  return {
    title: form.title,
    content: form.content,
    summary: form.summary,
    action_items: form.actionItems,
    participants: JSON.stringify(participants),
    tags: JSON.stringify(tags),
    meeting_date: form.meetingDate || new Date().toISOString().slice(0, 10),
    meeting_time: form.meetingTime,
    duration_minutes: form.durationMinutes,
    status: form.status || 'draft'
  }
}

function buildMeetingPayloadFromApi(m) {
  return {
    title: m.title,
    content: m.content,
    summary: m.summary,
    action_items: m.action_items,
    participants: m.participants,
    tags: m.tags,
    meeting_date: m.meeting_date,
    meeting_time: m.meeting_time,
    duration_minutes: m.duration_minutes,
    status: m.status
  }
}

async function restoreMeeting(snapshot) {
  if (!snapshot) return
  try {
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/meetings`, {
      method: 'POST',
      body: JSON.stringify(snapshot)
    })
    await loadMeetings()
    ElMessage.success('\u5df2\u6062\u590d: ' + (snapshot.title || '\u4f1a\u8bae'))
  } catch (error) {
    ElMessage.error('\u6062\u590d\u5931\u8d25: ' + (error.message || '\u672a\u77e5\u9519\u8bef'))
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

function toggleActionItem(idx) {
  try {
    const items = JSON.parse(meetingForm.actionItems || '[]')
    if (!Array.isArray(items) || idx < 0 || idx >= items.length) return
    items[idx].done = !items[idx].done
    meetingForm.actionItems = JSON.stringify(items)
    if (meetingForm.id) {
      saveMeeting()
    }
  } catch {}
}

async function createTaskFromAction(item) {
  try {
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks`, {
      method: 'POST',
      body: JSON.stringify({
        kind: activeKind.value,
        entry_type: 'task',
        bucket: 'planned',
        title: item.task,
        detail: `待办来源：会议「${meetingForm.title}」${item.assignee ? '，负责人：' + item.assignee : ''}`,
        notes: `来自会议纪要 ${meetingForm.meetingDate || ''}`,
        priority: 'medium',
        planned_for: new Date().toISOString().slice(0, 10)
      })
    })
    ElMessage.success('已从会议待办创建事项')
    await refreshBoard()
  } catch (error) {
    ElMessage.error(error.message)
  }
}

function exportMeetingMarkdown(m) {
  const title = m.title || '未命名会议'
  const date = m.meeting_date || ''
  const time = m.meeting_time || ''
  let md = `# ${title}\n\n`
  md += `**日期**：${date}${time ? ' ' + time : ''}\n`
  if (m.duration_minutes) {
    const h = Math.floor(m.duration_minutes / 60)
    const min = m.duration_minutes % 60
    md += `**时长**：${h > 0 ? h + '小时' : ''}${min > 0 ? min + '分钟' : ''}\n`
  }
  try {
    const participants = JSON.parse(m.participants || '[]')
    if (Array.isArray(participants) && participants.length > 0) {
      md += `**参与人**：${participants.join('、')}\n`
    }
  } catch {}
  try {
    const tags = JSON.parse(m.tags || '[]')
    if (Array.isArray(tags) && tags.length > 0) {
      md += `**标签**：${tags.join('、')}\n`
    }
  } catch {}
  if (m.status === 'finalized') md += '**状态**：已定稿\n'
  md += '\n---\n\n'
  if (m.content) {
    md += `## 会议内容\n\n${m.content}\n\n`
  }
  if (m.summary) {
    md += `## 摘要\n\n${m.summary}\n\n`
  }
  if (m.action_items && m.action_items !== '[]') {
    try {
      const actions = JSON.parse(m.action_items)
      if (Array.isArray(actions) && actions.length > 0) {
        md += '## 待办事项\n\n'
        for (const a of actions) {
          const done = a.done ? '[x]' : '[ ]'
          const assignee = a.assignee ? '(' + a.assignee + ')' : ''
          md += `- ${done} ${a.task} ${assignee}\n`
        }
      }
    } catch {}
  }
  const blob = new Blob([md], { type: 'text/markdown;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = title.replace(/[\\/:*?"<>|]/g, '_') + '.md'
  a.click()
  URL.revokeObjectURL(url)
}

async function handleMeetingDrawerClose(done) {
  if (meetingVoiceState.value === 'recording') {
    ElMessage.warning('正在录音，请先点「停止录音」再关闭，避免内容丢失')
    return
  }
  if (meetingDirty.value) {
    try {
      await ElMessageBox.confirm('有未保存的更改，确定关闭吗？', '未保存的更改', {
        confirmButtonText: '关闭',
        cancelButtonText: '取消',
        type: 'warning'
      })
      meetingDirty.value = false
      done()
    } catch {
      done(false)
    }
  } else {
    done()
  }
}

function requestCloseMeetingDrawer() {
  handleMeetingDrawerClose((cancel) => {
    if (cancel !== false) {
      meetingDialogVisible.value = false
    }
  })
}

watch(meetingDialogVisible, (visible) => {
  if (visible) {
    // 等待 applyMeetingToForm 的字段变更 watcher 跑完再清,避免初始加载被误标脏
    nextTick(() => { meetingDirty.value = false })
  }
})

function handleCommentDrawerClose(done) {
  if (isTaskRecording.value) {
    ElMessage.warning('正在录音，请先点「停止录音」再关闭，避免内容丢失')
    return
  }
  done()
}

function requestCloseCommentDrawer() {
  if (isTaskRecording.value) {
    ElMessage.warning('正在录音，请先点「停止录音」再关闭，避免内容丢失')
    return
  }
  commentDrawerVisible.value = false
}

watch(commentDrawerVisible, (visible) => {
  if (!visible && isTaskRecording.value) {
    stopTaskRecording()
  }
})

watch(
  () => [meetingForm.title, meetingForm.content, meetingForm.summary, meetingForm.actionItems,
         meetingForm.participantsText, meetingForm.tagsText, meetingForm.meetingDate,
         meetingForm.meetingTime, meetingForm.durationMinutes, meetingForm.status],
  () => {
    if (meetingDialogVisible.value && meetingForm.id) {
      meetingDirty.value = true
    }
  },
  { deep: false }
)

let meetingRecognition = null
let meetingMediaRecorder = null
let meetingMediaStream = null
let meetingVoiceChunks = []
let meetingTranscriptText = ''
const meetingInterimText = ref('')

let taskMediaRecorder = null
let taskAudioChunks = []
let taskRecordingStartTime = 0
let taskRecordingTimer = null
let taskRecordingPollTimer = null

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
          raw_text: item.raw_text,
          priority: item.priority,
          status: item.status,
          planned_for: item.planned_for,
          remind_at: item.remind_at,
          cancel_reason: item.cancel_reason,
          intent: item.intent,
          energy_level: item.energy_level
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
    review.cancellations = data.review?.cancellations || []
    review.postpones = data.review?.postpones || []
    review.completion_feelings = data.review?.completion_feelings || []
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
    quickForm.rawText = finalText
    if (!quickForm.title.trim()) {
      const segments = finalText.split(/[。！!？?\n]/).map(item => item.trim()).filter(Boolean)
      quickForm.title = segments[0] || truncateText(finalText, 40).replace(/\.\.\.$/, '')
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
  return { urgent: '🚨 紧急', high: '高优先级', medium: '中优先级', low: '低优先级' }[priority] || '中优先级'
}

function priorityTagType(priority) {
  return { urgent: 'danger', high: 'danger', medium: 'warning', low: 'info' }[priority] || ''
}

function prioritySortKey(priority) {
  return { urgent: 0, high: 1, medium: 2, low: 3 }[priority] ?? 2
}

function energyLabel(level) {
  return { deep: '需专注', shallow: '事务性', errand: '需外出', creative: '创意型' }[level] || ''
}

function energyIcon(level) {
  return { deep: '🧠', shallow: '📋', errand: '🚶', creative: '💡' }[level] || ''
}

function energyTagType(level) {
  if (level === 'shallow') return 'success'
  if (level === 'creative') return 'warning'
  return 'info'
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

function bucketTagType(bucket) {
  if (bucket === 'inbox') return 'info'
  if (bucket === 'someday') return 'warning'
  return ''
}

function taskOverdueDays(task) {
  if (!task) return 0
  if (task.status === 'done' || task.status === 'cancelled') return 0
  const planned = task.planned_for || task.display_date
  if (!planned) return 0
  const due = new Date(planned + 'T00:00:00')
  if (Number.isNaN(due.getTime())) return 0
  const now = new Date()
  now.setHours(0, 0, 0, 0)
  const diff = Math.floor((now.getTime() - due.getTime()) / 86400000)
  return diff > 0 ? diff : 0
}

function isTaskOverdue(task) {
  return taskOverdueDays(task) > 0
}

// 阶段 5 看见自己:任务创建至今的天数 — 让用户看见"承诺 vs 实际"的差距。
// 只对未收尾(open/in_progress)状态有效,3 天以内不显示(避免噪音)。
function taskAgeDays(task) {
  if (!task) return 0
  if (task.status === 'done' || task.status === 'cancelled') return 0
  if (!task.created_at) return 0
  const created = new Date(task.created_at)
  if (Number.isNaN(created.getTime())) return 0
  const now = new Date()
  const diff = Math.floor((now.getTime() - created.getTime()) / 86400000)
  return diff >= 0 ? diff : 0
}

function taskAgeLabel(task) {
  const days = taskAgeDays(task)
  if (days === 0) return ''
  if (days < 30) return `已挂 ${days} 天`
  if (days < 365) return `已挂 ${Math.floor(days / 30)} 个月`
  return `已挂 ${Math.floor(days / 365)} 年`
}

function habitSeedEligible(task) {
  if (!task) return false
  if ((task.completion_count || 0) < 3) return false
  if ((task.repeat_type || 'none') !== 'none') return false
  return true
}

async function applyHabitSeed(task, payload) {
  // 1 步提交:remind_at 智能默认 + repeat_type 设为对应频率。
  // 不改 completion_count、不改 planned_for,保留历史节奏数据。
  if (!task || !task.id) return
  try {
    await updateTask(task, {
      repeat_type: payload.repeat_type,
      remind_at: payload.remind_at
    }, '已设成习惯')
    if (!quietMode.value) {
      ElMessage({ message: '节奏建立起来了,慢慢来', type: 'success', duration: 2200 })
    }
  } catch (error) {
    if (error?.message) ElMessage.error(error.message || '设置失败')
  }
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

// 阶段 5 减法 (iter 19):把 "2026-06-27" 格式化为 "6 月 27 日" 用于"下次自动建到 X"toast
function formatShortDate(value) {
  if (!value) return ''
  const date = new Date(value + 'T00:00:00')
  if (Number.isNaN(date.getTime())) return value
  return `${date.getMonth() + 1} 月 ${date.getDate()} 日`
}

// 阶段 5 减法 (iter 20):从 notes 解析"第 N 次"
// 后端在创建 next instance 时会在 notes 末尾追加 "· 第 N 次"
// 解析后用于 task card 展示"🔁 第 N 次"chip
function parseFamilyRound(task) {
  if (!task || !task.notes) return 0
  const m = String(task.notes).match(/第\s*(\d+)\s*次/)
  return m ? Number(m[1]) : 0
}

const COMMENT_LINK_LABEL_MAX = 48

function escapeCommentHtml(text) {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

function renderCommentContent(content) {
  const raw = (content ?? '').toString()
  if (!raw) return ''
  const urlPattern = /(https?:\/\/[^\s<]+)/g
  let result = ''
  let lastIndex = 0
  let match
  while ((match = urlPattern.exec(raw)) !== null) {
    result += escapeCommentHtml(raw.slice(lastIndex, match.index))
    const url = match[0]
    const safeUrl = escapeCommentHtml(url)
    const label = url.length > COMMENT_LINK_LABEL_MAX
      ? escapeCommentHtml(url.slice(0, COMMENT_LINK_LABEL_MAX) + '…')
      : safeUrl
    result += `<a href="${safeUrl}" target="_blank" rel="noopener noreferrer" class="comment-link" title="${safeUrl}">${label}</a>`
    lastIndex = match.index + url.length
  }
  result += escapeCommentHtml(raw.slice(lastIndex))
  return result.replace(/\n/g, '<br>')
}

watch(activeKind, () => {
  if (profileId.value) {
    refreshBoard()
    if (reviewDialogVisible.value) {
      loadReview()
    }
  }
})

watch(profileId, (value) => {
  if (value) {
    loadPinnedTaskIds()
    loadCompletionLog()
    loadNotifiedTaskIds()
    loadFocusSession()
    loadFocusLog()
  } else {
    pinnedTaskIds.value = new Set()
    completionLog.value = {}
    notifiedTaskIds.value = new Set()
    focusSession.value = null
    focusLog.value = {}
    stopFocusTick()
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
  globalKeydownHandler = handleGlobalKeydown
  window.addEventListener('keydown', globalKeydownHandler)
  if (notificationEnabled.value && notificationSupported && Notification.permission === 'granted') {
    startNotificationLoop()
  }
  restoreSession()
  loadRoutines()
  fillTodayDefaults()
  resetTaskForm()
  if (profileId.value) {
    try {
      await loadProfile()
      loadRoutines()
      await refreshBoard()
    } catch (error) {
      clearSession()
    }
  }
})

onBeforeUnmount(() => {
  resetVoiceDraft()
  stopMeetingRecording()
  stopTaskRecording()
  if (taskRecordingPollTimer) {
    clearInterval(taskRecordingPollTimer)
    taskRecordingPollTimer = null
  }
  revokeMeetingLocalAudio()
  window.removeEventListener('resize', syncViewportFlags)
  if (beforeInstallPromptHandler) {
    window.removeEventListener('beforeinstallprompt', beforeInstallPromptHandler)
  }
  if (appInstalledHandler) {
    window.removeEventListener('appinstalled', appInstalledHandler)
  }
  if (globalKeydownHandler) {
    window.removeEventListener('keydown', globalKeydownHandler)
    globalKeydownHandler = null
  }
  if (notificationTimer) {
    clearInterval(notificationTimer)
    notificationTimer = null
  }
  stopFocusTick()
})

// 通用:把任务推迟到指定日期(1 步,卡片上直接点)
async function postponeTo(task, date, reasonLabel) {
  if (!task || !task.id || !date) return
  await updateTask(task, {
    planned_for: date,
    status: 'open',
    postpone_reason: reasonLabel || '一键改期'
  }, `已改到 ${date}`)
}

// 卡片上的"日期徽章"——点开就地改日期 / 推迟 / 移到收件箱
// 哲学:能 1 步就别 2 步(以前要开抽屉,现在点日期即可)
// 阶段 5 减法 (iter 23):已完成任务,日期 chip 自动切换为"完成于 X"
// 解决"卡片还显示计划日期但实际早完成"的视觉混乱
// 完整体验:芯片 ✓ 06-25 14:30,hover 看完整时间和"晚 N 天"提示
const TaskDateChip = defineComponent({
  name: 'TaskDateChip',
  props: {
    task: { type: Object, required: true }
  },
  emits: ['update', 'moveToInbox'],
  setup(props, { emit, expose }) {
    expose()
    const visible = ref(false)
    const newDate = ref('')

    // 已完成任务:展示完成时间,而不是计划日期
    const completionInfo = computed(() => {
      const t = props.task
      if (!t || t.status !== 'done' || !t.completed_at) return null
      const completed = new Date(t.completed_at)
      if (Number.isNaN(completed.getTime())) return null
      // 短格式:06-25 14:30
      const short = completed.toLocaleString('zh-CN', {
        month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'
      })
      // 完整时间 + 早晚天数
      const full = completed.toLocaleString('zh-CN', {
        year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'
      })
      let diffTip = ''
      if (t.planned_for) {
        const planned = new Date(t.planned_for + 'T00:00:00')
        if (!Number.isNaN(planned.getTime())) {
          const diffDays = Math.round((completed - planned) / 86400000)
          if (diffDays > 0) diffTip = ` · 晚 ${diffDays} 天完成`
          else if (diffDays < 0) diffTip = ` · 提前 ${-diffDays} 天完成`
          else diffTip = ' · 准时完成'
        }
      }
      return { short, title: `完成于 ${full}${diffTip}` }
    })

    watch(visible, (v) => {
      if (v) newDate.value = props.task.planned_for || ''
    })

    function quickPostpone(kind) {
      let date = ''
      let label = ''
      if (kind === 'tomorrow') {
        const d = new Date()
        d.setDate(d.getDate() + 1)
        date = d.toISOString().slice(0, 10)
        label = '一键改到明天'
      } else if (kind === 'nextMonday') {
        const r = parseNaturalDate('下周一')
        date = r?.date || ''
        label = '一键改到下周一'
      } else if (kind === 'inbox') {
        emit('moveToInbox')
        visible.value = false
        return
      }
      if (date) {
        emit('update', date, label)
        visible.value = false
      }
    }

    function applyCustomDate(val) {
      if (val) {
        emit('update', val, '自定义日期')
        visible.value = false
      }
    }

    return { visible, newDate, quickPostpone, applyCustomDate, completionInfo }
  },
  template: `
    <span v-if="completionInfo" class="task-date-chip task-date-chip-completed" :title="completionInfo.title">
      <el-icon><Check /></el-icon>
      <span>完成于 {{ completionInfo.short }}</span>
    </span>
    <el-popover
      v-else
      v-model:visible="visible"
      placement="bottom-start"
      :width="280"
      trigger="click"
      popper-class="task-date-popover"
    >
      <template #reference>
        <button class="task-date-chip" @click.stop>
          <el-icon><Calendar /></el-icon>
          <span>{{ task.planned_for || '未排期' }}</span>
        </button>
      </template>
      <div class="date-popover-content">
        <el-date-picker
          v-model="newDate"
          type="date"
          value-format="YYYY-MM-DD"
          placeholder="选择新日期"
          style="width: 100%"
          @change="applyCustomDate"
        />
        <div class="date-popover-quick">
          <el-button size="small" @click="quickPostpone('tomorrow')">明天</el-button>
          <el-button size="small" @click="quickPostpone('nextMonday')">下周一</el-button>
          <el-button size="small" type="warning" plain @click="quickPostpone('inbox')">→ 收件箱</el-button>
        </div>
        <p class="date-popover-hint">提示:日期就是按钮,不用进抽屉改</p>
      </div>
    </el-popover>
  `
})

// 阶段 5 减法:取消 1 步化。
// 替代 ElMessageBox.prompt 模态:点击按钮 → 4 个预设原因 chip → 1 步完成。
// 自定义原因通过"其他"输入框(用户主动选,2 步但表达自由)。
const CancelReasonPopover = defineComponent({
  name: 'CancelReasonPopover',
  props: {
    buttonSize: { type: String, default: 'small' },
    buttonText: { type: String, default: '取消' },
    // 阶段 5 减法 (iter 44):iconOnly 模式 — 只显示 icon 不显示文字
    // 用于 focus.secondary item-row 等空间紧凑场景
    iconOnly: { type: Boolean, default: false }
  },
  emits: ['cancel'],
  setup(_props, { emit }) {
    const visible = ref(false)
    const customReason = ref('')
    const lastReason = ref(readLastCancelReason())

    const presets = [
      { value: '不再需要', emoji: '🪶' },
      { value: '条件变化', emoji: '🔄' },
      { value: '不是现在', emoji: '⏸️' },
      { value: '已委派给他人', emoji: '🤝' }
    ]

    function apply(reason) {
      const final = (reason && reason.trim()) || customReason.value.trim() || '不再需要'
      emit('cancel', final)
      visible.value = false
      customReason.value = ''
    }

    return { visible, customReason, presets, lastReason, apply }
  },
  template: `
    <el-popover
      v-model:visible="visible"
      placement="bottom-start"
      :width="300"
      trigger="click"
      popper-class="cancel-reason-popover"
    >
      <template #reference>
        <el-button size="small" plain class="task-cancel-btn" :class="{ 'task-cancel-btn-icon-only': iconOnly }" :title="'选择取消原因'">
          <el-icon><CircleClose /></el-icon>
          <span v-if="!iconOnly">{{ buttonText || '取消' }}</span>
        </el-button>
      </template>
      <div class="cancel-popover-content">
        <p class="cancel-popover-title">为什么取消?</p>
        <div class="cancel-reason-grid">
          <button
            v-for="r in presets"
            :key="r.value"
            class="cancel-reason-chip"
            :class="{ 'cancel-reason-chip-active': lastReason === r.value }"
            @click="apply(r.value)"
          >
            <span class="cancel-reason-emoji">{{ r.emoji }}</span>
            <span>{{ r.value }}</span>
          </button>
        </div>
        <el-input
          v-model="customReason"
          placeholder="其他原因(可选,回车提交)"
          size="small"
          class="cancel-popover-input"
          @keydown.enter="apply('')"
        />
        <p class="cancel-popover-hint">点 chip 立即完成 · 上次用过的会高亮</p>
      </div>
    </el-popover>
  `
})

// 阶段 5 减法:习惯种子 1 步化(把"已完成 3 次"的最佳时机抓住)。
// 之前:点 🌱 → 打开完整 task drawer → 找"提醒时间" → 选时间 → "重复提醒"下拉激活 → 选频率 → 保存(5+ 步)。
// 现在:点 🌱 → popover → 4 个频率 chip → 1 步完成(remind_at 智能默认)。
const HabitSeedPopover = defineComponent({
  name: 'HabitSeedPopover',
  props: {
    task: { type: Object, required: true }
  },
  emits: ['set'],
  setup(props, { emit }) {
    const visible = ref(false)

    const presets = [
      { value: 'daily', label: '每天', emoji: '☀️' },
      { value: 'weekdays', label: '工作日', emoji: '💼' },
      { value: 'weekly', label: '每周', emoji: '📅' },
      { value: 'monthly', label: '每月', emoji: '🌙' }
    ]

    function smartRemindAt() {
      const now = new Date()
      const dateStr = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
      // 如果任务已有提醒时间,沿用其小时分钟
      if (props.task.remind_at) {
        try {
          const existing = new Date(props.task.remind_at)
          if (!isNaN(existing.getTime())) {
            const hh = String(existing.getHours()).padStart(2, '0')
            const mm = String(existing.getMinutes()).padStart(2, '0')
            return `${dateStr}T${hh}:${mm}`
          }
        } catch {}
      }
      // 默认 19:00(晚上回顾/打卡时段,符合"已完成 3 次"的心智)
      return `${dateStr}T19:00`
    }

    function apply(repeatType) {
      emit('set', { repeat_type: repeatType, remind_at: smartRemindAt() })
      visible.value = false
    }

    return { visible, presets, apply }
  },
  template: `
    <el-popover
      v-model:visible="visible"
      placement="bottom-start"
      :width="280"
      trigger="click"
      popper-class="habit-seed-popover"
    >
      <template #reference>
        <button class="task-habit-seed" @click.stop>
          🌱 想做习惯吗?设个重复
        </button>
      </template>
      <div class="habit-popover-content">
        <p class="habit-popover-title">设成多久重复一次?</p>
        <div class="habit-frequency-grid">
          <button
            v-for="p in presets"
            :key="p.value"
            class="habit-frequency-chip"
            @click="apply(p.value)"
          >
            <span class="habit-frequency-emoji">{{ p.emoji }}</span>
            <span>{{ p.label }}</span>
          </button>
        </div>
        <p class="habit-popover-hint">提醒时间会沿用之前的,默认晚上 19:00</p>
      </div>
    </el-popover>
  `
})

// 阶段 5 减法 (iter 22):评论 1 步化。
// 替代"按钮 → drawer 打开 → 输入 → 提交"4 步流程:点按钮 → popover → 输入 → Enter,1 步完成。
// popover 内同时显示最近 3 条评论预览,让用户能看见上下文。
// 录音 / 完整评论历史仍在 drawer 里(看完整链接打开)。
const QuickCommentPopover = defineComponent({
  name: 'QuickCommentPopover',
  props: {
    task: { type: Object, required: true },
    buttonSize: { type: String, default: 'small' },
    // 阶段 5 减法 (iter 37):允许调用方覆盖按钮文案,事件卡用「📝 记一下」更顺势
    buttonText: { type: String, default: '💬 评论' },
    buttonIcon: { type: String, default: 'ChatDotRound' }
  },
  emits: ['posted', 'open-drawer'],
  setup(props, { emit }) {
    const visible = ref(false)
    const draft = ref('')
    const submitting = ref(false)
    const previewComments = ref([])
    const loadingPreview = ref(false)

    async function refreshPreview() {
      if (!props.task || !props.task.id) return
      loadingPreview.value = true
      try {
        const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${props.task.id}/comments`)
        const data = await response.json().catch(() => ({}))
        const list = Array.isArray(data.comments) ? data.comments : []
        // 倒序取最近 3 条
        previewComments.value = list.slice(-3).reverse()
      } catch (e) {
        previewComments.value = []
      } finally {
        loadingPreview.value = false
      }
    }

    watch(visible, (v) => {
      if (v) {
        draft.value = ''
        refreshPreview()
      }
    })

    async function submit() {
      const text = draft.value.trim()
      if (!text || submitting.value) return
      if (!props.task || !props.task.id) return
      submitting.value = true
      try {
        await postComment(props.task.id, text, [])
        draft.value = ''
        if (!quietMode.value) {
          ElMessage({ message: '评论已保存', type: 'success', duration: 1400 })
        }
        emit('posted')
        await refreshPreview()
      } catch (e) {
        ElMessage.error(e?.message || '保存失败')
      } finally {
        submitting.value = false
      }
    }

    function onKeydown(e) {
      if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
        e.preventDefault()
        submit()
      }
    }

    function openDrawer() {
      visible.value = false
      emit('open-drawer', props.task)
    }

    return { visible, draft, submitting, previewComments, loadingPreview, submit, onKeydown, openDrawer, buttonText: props.buttonText, buttonIcon: props.buttonIcon }
  },
  template: `
    <el-popover
      v-model:visible="visible"
      placement="bottom-start"
      :width="320"
      trigger="click"
      popper-class="quick-comment-popover"
    >
      <template #reference>
        <el-button
          :size="buttonSize"
          plain
          @click.stop
        >
          <el-icon><component :is="buttonIcon" /></el-icon>
          {{ buttonText }}
        </el-button>
      </template>
      <div class="quick-comment-content">
        <p class="quick-comment-title">1 步加 1 句话进展</p>
        <textarea
          v-model="draft"
          class="quick-comment-input"
          rows="2"
          maxlength="200"
          placeholder="写一句进展,Enter 发送,Shift+Enter 换行"
          @keydown="onKeydown"
          @click.stop
        />
        <div class="quick-comment-actions">
          <span class="quick-comment-count">{{ draft.length }} / 200</span>
          <el-button
            size="small"
            type="primary"
            :loading="submitting"
            :disabled="!draft.trim()"
            @click="submit"
          >发送 (Enter)</el-button>
        </div>
        <div v-if="loadingPreview" class="quick-comment-loading">载入最近评论…</div>
        <div v-else-if="previewComments.length > 0" class="quick-comment-preview">
          <p class="quick-comment-preview-label">最近评论</p>
          <div v-for="c in previewComments" :key="c.id" class="quick-comment-preview-item">
            <span class="preview-content">{{ c.content }}</span>
            <span class="preview-time">{{ formatDateTime(c.created_at) }}</span>
          </div>
          <button class="quick-comment-drawer-link" @click="openDrawer">看完整评论/录音 →</button>
        </div>
        <div v-else class="quick-comment-empty">还没有评论,写下第一条</div>
      </div>
    </el-popover>
  `
})
</script>

<style scoped>
.planner-shell {
  position: relative;
  min-height: 100vh;
  overflow-x: clip;
  overflow-y: visible;
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
.view-tab {
  border: none;
  cursor: pointer;
}

.planner-topbar {
  position: relative;
  z-index: 20;
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

.hero-hint {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 6px;
  padding: 16px 20px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.08), rgba(59, 130, 246, 0.04));
  border-style: dashed;
}
.hero-hint-greeting {
  margin: 0 0 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--planner-muted, #6b7280);
  display: flex;
  align-items: center;
  gap: 4px;
}
.hero-hint-greeting-icon {
  font-size: 14px;
  line-height: 1;
}
.hero-hint-text {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--planner-text, #1f2937);
}
.hero-hint-sub {
  margin: 0;
  font-size: 12px;
  color: var(--planner-muted, #6b7280);
}
.hero-hint-actions {
  margin-top: 8px;
}
.hero-hint-actions .el-button {
  font-size: 12px;
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

.metric-card-clickable {
  border: 1px solid transparent;
  cursor: pointer;
  transition: background 0.18s ease, border-color 0.18s ease, transform 0.12s ease;
  text-align: left;
  font: inherit;
  color: inherit;
  width: 100%;
}
.metric-card-clickable:hover {
  background: rgba(99, 102, 241, 0.08);
  border-color: rgba(99, 102, 241, 0.25);
  transform: translateY(-1px);
}
.metric-card-clickable:active {
  transform: translateY(0);
}
.metric-card-alert {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.14), rgba(99, 102, 241, 0.05));
}
.metric-card-warn {
  background: linear-gradient(135deg, rgba(244, 114, 182, 0.10), rgba(99, 102, 241, 0.04));
}

/* 阶段 5 减法 (iter 32):全 0 合并的"今天清爽"卡 — 4 metric card 减为 1 个
   用绿色调 + ✨ 表达"轻松/休息"语义,与 iter 29 "今天清爽" 视觉一致 */
.metric-card-calm {
  background: linear-gradient(135deg, rgba(180, 220, 170, 0.4), rgba(220, 235, 200, 0.5)) !important;
  border-color: rgba(140, 180, 100, 0.45) !important;
  color: #3a4a2a;
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 8px;
  cursor: default;
  grid-column: span 4;
  justify-content: center;
  padding: 14px 16px;
}
.metric-card-calm-icon {
  font-size: 18px;
  line-height: 1;
}
.metric-card-calm-label {
  font-size: 14px;
  font-weight: 600;
  color: #4a6b3a;
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

.focus-meta-family {
  background: linear-gradient(135deg, #fff7ed, #ffedd5) !important;
  color: #c2410c !important;
  font-weight: 600;
  border: 1px solid rgba(251, 146, 60, 0.35);
}

.focus-secondary-item {
  border: none;
  border-radius: 14px;
  background: transparent;
  padding: 4px 6px;
  text-align: left;
  cursor: pointer;
  display: grid;
  gap: 6px;
  flex: 1;
  min-width: 0;
  transition: background 0.15s ease;
}

.focus-secondary-item:hover {
  background: rgba(255, 255, 255, 0.5);
}

/* 阶段 5 减法 (iter 43):focus.secondary 单件行容器 — 包裹"点击完成"区 + "→ 明天"按钮
   单/批 1 步对称:iter 36 批量顺延 ↻ 顶部按钮 + iter 43 单件 → 明天 行内按钮
   改天维度在 secondary 区域 100% 覆盖,无需回 timeline 找卡 */
.focus-secondary-item-row {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 240px;
  flex: 1 1 240px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.72);
  padding: 6px 6px 6px 8px;
  transition: box-shadow 0.15s ease;
}

.focus-secondary-item-row:hover {
  box-shadow: 0 2px 8px rgba(15, 63, 152, 0.08);
}

/* 阶段 5 减法 (iter 43):focus.secondary 行内"→ 明天"按钮
   蓝色调与 iter 36/38 顺延色系一致(冷思考/改天)
   小尺寸,避免抢主操作"点击完成"区视觉焦点 */
.focus-secondary-item-postpone {
  flex-shrink: 0;
  border: 1px solid rgba(59, 130, 246, 0.3);
  border-radius: 14px;
  background: linear-gradient(135deg, #eff6ff, #dbeafe);
  color: #1e40af;
  font-size: 12px;
  font-weight: 600;
  padding: 6px 10px;
  cursor: pointer;
  white-space: nowrap;
  transition: transform 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
}

.focus-secondary-item-postpone:hover {
  background: linear-gradient(135deg, #dbeafe, #bfdbfe);
  border-color: rgba(59, 130, 246, 0.5);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.2);
}

.focus-secondary-item strong {
  color: #21324c;
}

.focus-secondary-item span {
  color: #6c7d91;
  font-size: 12px;
}

/* 阶段 5 减法 (iter 34 + iter 36):focus.secondary 顶部 1 步批量操作按钮组
   iter 34: ✓ 收尾 (绿色,完成场景)
   iter 36: ↻ 顺延到明天 (蓝色,改天场景)
   两个按钮并排,row 容器自适应宽度,移动端堆叠 */
.focus-secondary-bulk-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  flex: 1 1 100%;
  margin-bottom: 4px;
}

.focus-secondary-bulk {
  border: 1px solid rgba(34, 197, 94, 0.3);
  border-radius: 18px;
  background: linear-gradient(135deg, #dcfce7, #bbf7d0);
  padding: 12px 16px;
  text-align: center;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  color: #166534;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 160px;
  flex: 1 1 160px;
  transition: transform 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
}

.focus-secondary-bulk:hover {
  background: linear-gradient(135deg, #bbf7d0, #86efac);
  border-color: rgba(34, 197, 94, 0.5);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(34, 197, 94, 0.2);
}

/* 阶段 5 减法 (iter 36):↻ 顺延按钮 — 蓝色调,与 ✓ 收尾(绿)区分
   "改天"语义用冷色(蓝),"完成"语义用暖色(绿),色彩帮助用户秒懂 */
.focus-secondary-bulk-postpone {
  border-color: rgba(59, 130, 246, 0.3);
  background: linear-gradient(135deg, #dbeafe, #bfdbfe);
  color: #1e40af;
}
.focus-secondary-bulk-postpone:hover {
  background: linear-gradient(135deg, #bfdbfe, #93c5fd);
  border-color: rgba(59, 130, 246, 0.5);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.2);
}

/* 阶段 5 减法 (iter 44):focus.secondary "✗ 取消" 1 步批量按钮
   与 ✓ 收尾(绿)/ ↻ 顺延(蓝) 形成状态机 1 步批量完整闭环
   棕色调(取消/不再做)与 task-cancel-btn 保持一致
   "改天"(蓝)与"取消"(棕)都是"今天不做了"的两个分支决策
   hover 加深棕色边框,提示"严肃操作" */
.focus-secondary-bulk-cancel {
  border-color: rgba(166, 122, 90, 0.3);
  background: linear-gradient(135deg, #f5ebe0, #e7d5c0);
  color: #8a5a3b;
}
.focus-secondary-bulk-cancel:hover {
  background: linear-gradient(135deg, #e7d5c0, #d4b896);
  border-color: rgba(166, 122, 90, 0.5);
  box-shadow: 0 4px 12px rgba(166, 122, 90, 0.2);
}

/* 阶段 5 减法 (iter 38):focus 主卡"📅 改明天" 1 步按钮
   与 .focus-secondary-bulk-postpone 同色系(蓝色 = 冷思考/改天)
   视觉上 focus 主卡完成(绿)+ 改天(蓝)对称,色温帮助用户秒懂
   focus.secondary 已 iter 36 用同色,保持视觉一致性 */
.focus-primary-postpone-btn {
  border-color: rgba(59, 130, 246, 0.3) !important;
  background: linear-gradient(135deg, #eff6ff, #dbeafe) !important;
  color: #1e40af !important;
}
.focus-primary-postpone-btn:hover {
  background: linear-gradient(135deg, #dbeafe, #bfdbfe) !important;
  border-color: rgba(59, 130, 246, 0.5) !important;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.18) !important;
}

@media (max-width: 640px) {
  .focus-secondary-bulk {
    min-width: 100%;
  }
}

/* 阶段 5 减法 (iter 33):focus card 空状态 1 步置入候选
   候选 chip: 1 步点击 = togglePin, 颜色按优先级 (urgent 红 / high 橙 / medium 蓝 / low 灰) */
.focus-candidates {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 4px;
}

/* 阶段 5 减法 (iter 47):focus candidate row 容器 — 包裹"置入" chip + 行内 "✓" 按钮
   与 focus.secondary item-row (iter 43/44) 模式 1:1 对称:
   chip 主操作(置入) + 行内次操作(完成),flex row 容器管理两区
   1 步决策矩阵:用户看候选 1 件小事 → 选"置入"(大事/重要)或"✓ 完成"(小事/简单) */
.focus-candidate-row {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 240px;
  flex: 1 1 240px;
}

.focus-candidate-chip {
  border: 1px solid rgba(99, 124, 165, 0.18);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.78);
  padding: 10px 12px;
  text-align: left;
  cursor: pointer;
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 8px;
  min-width: 200px;
  flex: 1 1 200px;
  max-width: 100%;
  transition: transform 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
}

.focus-candidate-chip:hover {
  background: linear-gradient(135deg, #fff7ed, #fef3c7);
  border-color: rgba(251, 146, 60, 0.4);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(251, 146, 60, 0.18);
}

.focus-candidate-chip.priority-urgent {
  border-color: rgba(220, 38, 38, 0.3);
  background: linear-gradient(135deg, #fef2f2, #fee2e2);
}
.focus-candidate-chip.priority-urgent:hover {
  background: linear-gradient(135deg, #fecaca, #fca5a5);
  border-color: rgba(220, 38, 38, 0.5);
}
.focus-candidate-chip.priority-high {
  border-color: rgba(234, 88, 12, 0.25);
  background: linear-gradient(135deg, #fff7ed, #ffedd5);
}
.focus-candidate-chip.priority-medium {
  border-color: rgba(59, 130, 246, 0.2);
  background: linear-gradient(135deg, #eff6ff, #dbeafe);
}
.focus-candidate-chip.priority-low {
  border-color: rgba(148, 163, 184, 0.25);
  background: rgba(248, 250, 252, 0.78);
  opacity: 0.85;
}

.focus-candidate-pin {
  font-size: 14px;
  opacity: 0.7;
}

.focus-candidate-title {
  color: #21324c;
  font-weight: 600;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 240px;
}

.focus-candidate-source {
  color: #6c7d91;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.7);
  white-space: nowrap;
}

/* 阶段 5 减法 (iter 47):focus candidate 行内 "✓" 完成按钮
   绿色调(暖色/完成)与 focus.secondary 收尾色系一致
   紧凑圆形按钮,避免抢主操作"置入"区视觉焦点
   复用 setTaskStatus + 5s 撤销,@click.stop 防止冒泡到 chip 主点击 */
.focus-candidate-complete {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border: 1px solid rgba(34, 197, 94, 0.3);
  border-radius: 50%;
  background: linear-gradient(135deg, #dcfce7, #bbf7d0);
  color: #166534;
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
}

.focus-candidate-complete:hover {
  background: linear-gradient(135deg, #bbf7d0, #86efac);
  border-color: rgba(34, 197, 94, 0.5);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(34, 197, 94, 0.25);
}

@media (max-width: 640px) {
  .focus-candidate-chip {
    min-width: 100%;
  }
  .focus-candidate-title {
    max-width: 200px;
  }
}

.compact-actions {
  margin-top: 0;
}

.review-strip {
  margin-top: 12px;
}

/* 阶段 5:取消原因聚合 — 让用户看见自己的模式 */
.soft-note-inline {
  font-size: 12px;
  color: #8a99af;
  font-weight: 400;
  margin-left: 8px;
}

.cancel-reason-chips {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin: 4px 0 10px;
}

.cancel-reason-label {
  font-size: 12px;
  color: #6c7d91;
  margin-right: 2px;
}

.cancel-reason-chip {
  font-size: 12px;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(245, 222, 200, 0.65);
  color: #8a5a3b;
  border: 1px solid rgba(180, 140, 100, 0.3);
}

.cancellation-list {
  flex-direction: column;
  gap: 8px;
}

.cancel-item {
  background: rgba(252, 246, 238, 0.72);
  border: 1px solid rgba(212, 188, 162, 0.35);
  border-radius: 14px;
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  cursor: default;
  min-width: 0;
  flex: 1 1 auto;
}

.cancel-item-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.cancel-item-reason {
  font-size: 12px;
  color: #8a6b54;
  font-weight: 400;
}

.cancel-item-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 11px;
  color: #9aa5b6;
  align-items: center;
}

.cancel-rollover {
  color: #b58660;
}

/* 阶段 6 减法:review 弹窗"重新捡起"按钮 — 与取消暖色系呼应,但语义是"重新开始"(绿) */
.cancel-item-revive {
  background: rgba(140, 180, 100, 0.18);
  border: 1px dashed rgba(140, 180, 100, 0.5);
  color: #4a6b3a;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, transform 0.1s;
  font-family: inherit;
  margin-left: auto;
}
.cancel-item-revive:hover {
  background: rgba(140, 180, 100, 0.32);
  border-color: rgba(140, 180, 100, 0.85);
  transform: translateY(-1px);
}

/* 阶段 6 减法:review 弹窗"今天做"按钮 — 与顺延冷蓝呼应(让"再做"对称) */
.postpone-item-today {
  background: rgba(91, 141, 239, 0.15);
  border: 1px dashed rgba(91, 141, 239, 0.5);
  color: #355080;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, transform 0.1s;
  font-family: inherit;
  margin-left: auto;
}
.postpone-item-today:hover {
  background: rgba(91, 141, 239, 0.28);
  border-color: rgba(91, 141, 239, 0.85);
  transform: translateY(-1px);
}

/* 阶段 5 延伸:顺延聚合 — 配色用雾蓝(冷),与暖黄取消形成冷暖对照 */
.postpone-reason-chips {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin: 4px 0 10px;
}

.postpone-reason-label {
  font-size: 12px;
  color: #6c7d91;
  margin-right: 2px;
}

.postpone-reason-chip {
  font-size: 12px;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(210, 220, 235, 0.55);
  color: #4a6080;
  border: 1px solid rgba(150, 170, 195, 0.35);
}

.postpone-list {
  flex-direction: column;
  gap: 8px;
}

.postpone-item {
  background: rgba(238, 243, 250, 0.72);
  border: 1px solid rgba(170, 190, 215, 0.4);
  border-radius: 14px;
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  cursor: default;
  min-width: 0;
  flex: 1 1 auto;
}

.postpone-item-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.postpone-item-reason {
  font-size: 12px;
  color: #5a7090;
  font-weight: 400;
}

.postpone-item-meta {
  display: flex;
  gap: 12px;
  font-size: 11px;
  color: #8d9bb0;
  flex-wrap: wrap;
  align-items: center;
}

.postpone-target {
  color: #4a6080;
}

.postpone-rollover {
  color: #6b81a0;
  font-weight: 500;
}

/* 阶段 6 减法:完成感受区 — review "我做完了感觉…" 的展示样式
   与 postpone 镜像但用暖色系,体现"已完成"的温度 */
.feeling-distribution-chips {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.feeling-distribution-label {
  font-size: 12px;
  color: #8a6840;
}
.feeling-distribution-chip {
  display: inline-block;
  padding: 3px 10px;
  background: rgba(255, 235, 200, 0.55);
  border: 1px solid rgba(220, 170, 110, 0.4);
  border-radius: 999px;
  font-size: 12px;
  color: #6b4519;
  font-weight: 500;
}
.feeling-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.feeling-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  background: rgba(255, 245, 230, 0.45);
  border: 1px solid rgba(220, 170, 110, 0.25);
  border-radius: 10px;
}
.feeling-item-main {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1;
}
.feeling-emoji {
  font-size: 18px;
  flex-shrink: 0;
}
.feeling-item-main strong {
  font-size: 14px;
  color: #4a3818;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.feeling-item-label {
  font-size: 12px;
  color: #8a6840;
  flex-shrink: 0;
}
.feeling-item-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: #8a6840;
  flex-shrink: 0;
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

.recent-expand-row {
  display: flex;
  justify-content: center;
  padding: 12px 0 4px;
}

.recent-expand-row .el-button {
  font-weight: 600;
  color: #64748b;
  border-color: rgba(148, 163, 184, 0.4);
}

.recent-expand-row .el-button:hover {
  color: #2563eb;
  border-color: #2563eb;
  background: rgba(37, 99, 235, 0.06);
}

/* 阶段 5:时间线折叠 banner — 一行提示 + 1 步展开 */
.timeline-fold-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  margin: 4px 0 12px;
  background: rgba(248, 250, 252, 0.7);
  border: 1px dashed rgba(148, 163, 184, 0.4);
  border-radius: 12px;
  color: #64748b;
  font-size: 13px;
}

.timeline-fold-banner strong {
  color: #475569;
  font-weight: 600;
}

.timeline-fold-banner .el-button {
  font-size: 12px;
  color: #64748b;
  border-color: rgba(148, 163, 184, 0.5);
  background: white;
  border-radius: 999px;
  padding: 4px 14px;
}

.timeline-fold-banner .el-button:hover {
  color: #2563eb;
  border-color: #2563eb;
  background: rgba(37, 99, 235, 0.04);
}

.routine-popover-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.routine-empty p {
  margin: 0;
  color: #64748b;
  font-size: 13px;
}

.routine-empty-hint {
  color: #94a3b8;
  font-size: 12px;
  margin-top: 4px;
}

.routine-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 320px;
  overflow-y: auto;
}

.routine-item {
  padding: 10px 12px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.06), rgba(99, 102, 241, 0.02));
  border: 1px solid rgba(99, 102, 241, 0.15);
  border-radius: 10px;
}

.routine-item-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.routine-item-name {
  font-size: 14px;
  color: #1e293b;
}

.routine-item-count {
  font-size: 11px;
  color: #64748b;
  background: rgba(255, 255, 255, 0.6);
  padding: 1px 8px;
  border-radius: 999px;
}

.routine-item-actions {
  display: flex;
  gap: 6px;
}

.routine-item-actions .el-button {
  padding: 4px 10px;
  font-size: 12px;
}

.routine-popover-footer {
  display: flex;
  justify-content: flex-end;
  border-top: 1px dashed rgba(148, 163, 184, 0.3);
  padding-top: 10px;
}

.routine-editor {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.routine-editor-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.routine-editor-row label {
  font-size: 13px;
  font-weight: 600;
  color: #475569;
  white-space: nowrap;
  min-width: 60px;
}

.routine-editor-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 50vh;
  overflow-y: auto;
}

.routine-editor-item {
  padding: 10px;
  background: rgba(248, 250, 252, 0.7);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.routine-editor-item-main {
  display: flex;
  gap: 6px;
  align-items: center;
  flex-wrap: wrap;
}

.routine-editor-item-main .el-input {
  flex: 1;
  min-width: 180px;
}

.routine-editor-offset-hint {
  font-size: 12px;
  color: #64748b;
  white-space: nowrap;
}

.routine-editor-item-actions {
  display: flex;
  gap: 4px;
  justify-content: flex-end;
}

.routine-editor-add {
  align-self: flex-start;
}

.timeline-empty {
  padding: 28px 20px;
  text-align: center;
  color: #75859b;
  display: grid;
  gap: 6px;
}

.task-recording-block {
  display: grid;
  gap: 10px;
  padding: 12px 14px;
  border: 1px solid #e3e8f0;
  border-radius: 12px;
  background: #f8fafc;
}

.task-recording-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}

.task-recording-head-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.task-recording-list {
  display: grid;
  gap: 12px;
}

.task-recording-item {
  display: grid;
  gap: 6px;
  padding: 10px 12px;
  border-radius: 10px;
  background: #fff;
  border: 1px solid #edf1f7;
}

.task-recording-audio {
  width: 100%;
  height: 36px;
}

.task-recording-transcript {
  margin: 0;
  white-space: pre-wrap;
  color: #36435a;
}

.task-recording-transcript.transcript-collapsed {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.task-recording-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 12px;
}

.task-recording-summary {
  margin-top: 2px;
  padding: 8px 10px;
  border-radius: 8px;
  background: #f0f9f3;
  border: 1px solid #d6efe0;
}

.task-recording-summary .summary-label {
  font-size: 12px;
  font-weight: 600;
  color: #2f9e6c;
}

.task-recording-summary p {
  margin: 4px 0 0;
  white-space: pre-wrap;
  color: #36435a;
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

.task-card.priority-urgent {
  background: linear-gradient(135deg, rgba(254, 226, 226, 0.95), rgba(255, 255, 255, 0.92));
  border: 1.5px solid rgba(239, 68, 68, 0.45);
  box-shadow: 0 8px 20px rgba(239, 68, 68, 0.12);
  position: relative;
}

.focus-card-urgent {
  border: 2px solid #ef4444;
  box-shadow: 0 0 0 4px rgba(239, 68, 68, 0.12);
}

/* 阶段 5 减法 (iter 30):focus card in_progress 状态 — 1 步"开始"后,
   用蓝色脉动标识"你正在做这件事",给用户明确的"专注中"信号
   完成时自动消失,接续下一个 primary */
.focus-card-in-progress {
  border: 1.5px solid rgba(99, 132, 255, 0.55);
  box-shadow: 0 0 0 3px rgba(99, 132, 255, 0.12);
  position: relative;
}
.focus-card-in-progress::before {
  content: '▶ 专注中';
  position: absolute;
  top: -10px;
  left: 14px;
  background: linear-gradient(135deg, #6366f1, #818cf8);
  color: white;
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 4px;
  letter-spacing: 0.5px;
  box-shadow: 0 2px 4px rgba(99, 132, 255, 0.3);
  animation: focus-pulse 2s ease-in-out infinite;
}
@keyframes focus-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.summary-chip-urgent {
  background: linear-gradient(135deg, #ef4444, #f87171) !important;
  color: white !important;
  font-weight: 600;
  animation: urgent-pulse 2.4s ease-in-out infinite;
}

/* 阶段 5 减法 (iter 39):主页 🚨 紧急 chip 1 步可点
   summary-chip-clickable 与其他 summary chip 共享 cursor + hover 视觉
   summary-chip-urgent-active:正在查看紧急 filter 时,加深 + 内描边,告知"已激活"
   颜色保持红系(紧急 = 警示色),与脉冲动画叠加形成强提示 */
.summary-chip-clickable {
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}
.summary-chip-clickable:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 16px rgba(15, 63, 152, 0.15);
}
.summary-chip-urgent-active {
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.45) inset, 0 4px 12px rgba(239, 68, 68, 0.25);
  animation: none;
}

@keyframes urgent-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.4); }
  50% { box-shadow: 0 0 0 6px rgba(239, 68, 68, 0); }
}

.task-card.priority-urgent::before {
  content: '🚨 紧急';
  position: absolute;
  top: -10px;
  left: 14px;
  background: #ef4444;
  color: white;
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 999px;
  letter-spacing: 0.5px;
}

.preset-btn-strong {
  background: linear-gradient(135deg, #1e293b, #334155) !important;
  color: #fff !important;
  font-weight: 600;
}

.preset-btn-strong:hover {
  background: linear-gradient(135deg, #0f172a, #1e293b) !important;
}

/* 自然语言日期输入行 */
.quick-row-natural-date .el-input__inner {
  font-size: 13px;
}
.quick-row-natural-date .el-input__inner::placeholder {
  color: var(--el-color-info);
  font-style: normal;
}

/* 任务模板(快捷短语) */
.quick-templates {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin: 4px 0 6px;
  padding: 6px 10px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}
.quick-templates-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-right: 4px;
  user-select: none;
}
.quick-template-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: #fff;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 999px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;
  max-width: 220px;
}
.quick-template-chip:hover {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.quick-template-title {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 140px;
}
.quick-template-priority {
  display: inline-block;
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 4px;
  background: var(--el-color-info-light-9);
  color: var(--el-color-info);
  flex-shrink: 0;
}
.quick-template-priority.qp-urgent {
  background: #fef0f0;
  color: #f56c6c;
}
.quick-template-priority.qp-high {
  background: #fdf6ec;
  color: #e6a23c;
}
.quick-template-priority.qp-medium {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}
.quick-template-priority.qp-low {
  background: #f0f9eb;
  color: #67c23a;
}
.quick-template-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.05);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1;
  margin-left: 2px;
  flex-shrink: 0;
  cursor: pointer;
}
.quick-template-remove:hover {
  background: #f56c6c;
  color: #fff;
}
.quick-template-edit {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.05);
  color: var(--el-text-color-secondary);
  font-size: 11px;
  line-height: 1;
  margin-left: 2px;
  flex-shrink: 0;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}
.quick-template-edit:hover {
  background: var(--el-color-primary);
  color: #fff;
}

.timeline-filter-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 12px 0 16px;
}

.filter-chip {
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: rgba(255, 255, 255, 0.78);
  color: #475569;
  padding: 6px 12px;
  border-radius: 999px;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.filter-chip strong {
  color: #1e293b;
  font-weight: 600;
}

.filter-chip:hover {
  background: rgba(255, 255, 255, 0.95);
  border-color: rgba(99, 102, 241, 0.3);
}

.filter-chip.active {
  background: linear-gradient(135deg, #4f46e5, #6366f1);
  color: white;
  border-color: transparent;
}

.filter-chip.active strong {
  color: white;
}

.filter-chip-urgent.active {
  background: linear-gradient(135deg, #ef4444, #f87171);
  box-shadow: 0 4px 12px rgba(239, 68, 68, 0.3);
}

.undo-snackbar {
  position: fixed;
  left: 50%;
  bottom: 24px;
  transform: translateX(-50%);
  background: #1e293b;
  color: #fff;
  border-radius: 14px;
  padding: 12px 18px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.32);
  z-index: 2400;
  max-width: calc(100vw - 32px);
  font-size: 14px;
}

.undo-snackbar .undo-message {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1 1 auto;
}

.undo-snackbar .undo-message > span {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 50vw;
}

.undo-snackbar .undo-icon {
  color: #fbbf24;
  flex: none;
}

.undo-snackbar .undo-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: none;
}

.undo-snackbar .undo-tick {
  color: rgba(255, 255, 255, 0.6);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.undo-snackbar .el-button {
  margin-left: 0;
}

.undo-snackbar .el-button--small.is-plain {
  background: rgba(255, 255, 255, 0.92);
  color: #1e293b;
  border-color: transparent;
}

.undo-snackbar .el-button.is-text {
  color: rgba(255, 255, 255, 0.7);
}

.undo-slide-enter-active,
.undo-slide-leave-active {
  transition: all 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}

.undo-slide-enter-from,
.undo-slide-leave-to {
  opacity: 0;
  transform: translate(-50%, 24px);
}

.fab-quick-add {
  position: fixed;
  right: 18px;
  bottom: 18px;
  z-index: 2350;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 14px 18px;
  border: none;
  border-radius: 999px;
  background: linear-gradient(135deg, #4f46e5, #6366f1);
  color: white;
  font-size: 15px;
  font-weight: 600;
  box-shadow: 0 12px 24px rgba(79, 70, 229, 0.32);
  cursor: pointer;
  transition: bottom 0.28s cubic-bezier(0.4, 0, 0.2, 1), transform 0.2s ease;
}

.fab-quick-add.fab-with-undo {
  bottom: 96px;
}

.fab-quick-add:active {
  transform: scale(0.96);
}

.fab-quick-add .el-icon {
  font-size: 18px;
}

@media (max-width: 480px) {
  .fab-quick-add {
    right: 14px;
    bottom: 14px;
    padding: 12px 16px;
    font-size: 14px;
  }
  .fab-quick-add.fab-with-undo {
    bottom: 84px;
  }
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

.comment-content {
  overflow-wrap: anywhere;
  word-break: break-word;
  white-space: pre-wrap;
}

.comment-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.comment-image-thumb {
  width: 96px;
  height: 96px;
  border-radius: 8px;
  object-fit: cover;
  border: 1px solid var(--el-border-color-lighter);
  cursor: zoom-in;
  background: #f5f7fa;
}

.comment-draft-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.comment-draft-image {
  position: relative;
  width: 96px;
  height: 96px;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  background: #f5f7fa;
}

.comment-draft-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.comment-draft-remove {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: none;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  transition: background 0.15s ease;
}

.comment-draft-remove:hover {
  background: rgba(220, 38, 38, 0.85);
}

.comment-link {
  color: var(--el-color-primary);
  text-decoration: underline;
  overflow-wrap: anywhere;
  word-break: break-all;
}

.ai-card-body {
  min-width: 0;
  overflow: hidden;
}

.ai-card-title {
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.ai-card-detail {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
  line-height: 1.5;
  color: #606266;
}

.ai-card-raw {
  margin-top: 4px;
  font-size: 12px;
  color: #909399;
}

.ai-card-raw summary {
  cursor: pointer;
  font-size: 12px;
  color: #909399;
}

.ai-card-raw p {
  margin: 4px 0 0;
  font-size: 12px;
  color: #909399;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 120px;
  overflow-y: auto;
}

.task-tags {
  overflow-x: auto;
  flex-shrink: 0;
  -webkit-overflow-scrolling: touch;
}

.task-tags .el-tag {
  flex-shrink: 0;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.task-overdue-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 4px 2px 10px;
  border-radius: 999px;
  background: linear-gradient(135deg, #ef4444, #f87171);
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
  box-shadow: 0 2px 6px rgba(239, 68, 68, 0.25);
}

/* 阶段 5 看见自己:"已挂 N 天"软徽章 — 不是指责,是温柔观察。
   比超期徽章弱一档,只提示事实,让用户自己决定。*/
.task-age-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border-radius: 999px;
  background: rgba(245, 222, 200, 0.6);
  color: #8a5a3b;
  font-size: 11px;
  font-weight: 500;
  border: 1px solid rgba(180, 140, 100, 0.25);
  cursor: default;
  transition: background 0.15s ease;
}
.task-age-badge:hover {
  background: rgba(245, 222, 200, 0.85);
}

.overdue-snooze-btn {
  border: none;
  background: rgba(255, 255, 255, 0.92);
  color: #b91c1c;
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 999px;
  cursor: pointer;
  letter-spacing: 0.02em;
  transition: background 0.15s ease, transform 0.1s ease;
}

.overdue-snooze-btn:hover {
  background: #fff;
  transform: translateY(-1px);
}

.overdue-snooze-btn:active {
  transform: translateY(0);
}

.task-card.task-card-overdue {
  position: relative;
  border-left: 3px solid #ef4444 !important;
  background: linear-gradient(90deg, rgba(239, 68, 68, 0.06), transparent 60%) !important;
}

.task-card.task-card-overdue:hover {
  background: linear-gradient(90deg, rgba(239, 68, 68, 0.12), rgba(255, 255, 255, 0.6) 80%) !important;
}

.task-card-select {
  margin-right: 8px;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.task-card:hover .task-card-select,
.task-card.task-card-bulk-selected .task-card-select,
.task-card:focus-within .task-card-select {
  opacity: 1;
}

.task-card.task-card-bulk-selected {
  background: linear-gradient(90deg, rgba(99, 102, 241, 0.08), rgba(99, 102, 241, 0.02)) !important;
  border-color: rgba(99, 102, 241, 0.3) !important;
}

.batch-bar-floating {
  position: fixed;
  left: 50%;
  bottom: 24px;
  transform: translateX(-50%);
  z-index: 200;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 999px;
  background: linear-gradient(135deg, #1e293b, #334155);
  color: #fff;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.3);
  max-width: calc(100vw - 32px);
  flex-wrap: wrap;
  justify-content: center;
}

.batch-bar-floating .el-button {
  color: #fff;
  border-color: rgba(255, 255, 255, 0.3);
  background: rgba(255, 255, 255, 0.08);
}

.batch-bar-floating .el-button:hover {
  background: rgba(255, 255, 255, 0.18);
  border-color: rgba(255, 255, 255, 0.5);
}

.batch-bar-floating .el-button.is-text {
  background: transparent;
  border-color: transparent;
  color: rgba(255, 255, 255, 0.7);
}

.batch-bar-floating .el-button.is-text:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.08);
}

.batch-bar-floating .batch-bar-count {
  font-weight: 600;
  font-size: 13px;
  margin-right: 4px;
}

.batch-bar-floating .el-button--danger.is-plain {
  color: #fca5a5;
  border-color: rgba(252, 165, 165, 0.4);
}

.batch-bar-floating .el-button--danger.is-plain:hover {
  color: #fff;
  background: rgba(239, 68, 68, 0.3);
  border-color: rgba(252, 165, 165, 0.7);
}

.task-status-cycle {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  cursor: pointer;
  border-radius: 6px;
  padding: 1px 2px;
  transition: background 0.15s ease;
}

.task-status-cycle:hover {
  background: rgba(99, 102, 241, 0.08);
}

.task-status-cycle .status-cycle-hint {
  font-size: 11px;
  color: #94a3b8;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.task-status-cycle:hover .status-cycle-hint {
  opacity: 1;
}

.task-actions {
  margin-top: 12px;
}

/* 阶段 5 减法:⋯ 按钮的样式 — 与其他 el-button 视觉一致,但更克制 */
.task-more-btn {
  padding: 4px 8px !important;
  min-width: auto;
}

.task-more-btn .el-icon {
  font-size: 14px;
}

/* 阶段 5 1 步化:优先级标签可点击 — 1 步改优先级 */
.clickable-priority {
  cursor: pointer;
  transition: transform 0.1s ease, box-shadow 0.15s ease;
}

.clickable-priority:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(99, 102, 241, 0.18);
}

.energy-hint-tag {
  color: var(--el-text-color-secondary);
  background: rgba(0, 0, 0, 0.02);
  border: 1px dashed rgba(0, 0, 0, 0.15);
  opacity: 0.7;
}
.energy-hint-tag:hover {
  opacity: 1;
}

/* 阶段 5 减法 (iter 20):重复任务接力标识 chip
   浅暖紫底色 — 与"重复"语义匹配,不刺眼,与撤销/未做的红/灰区分 */
.family-round-chip {
  background: linear-gradient(135deg, rgba(180, 150, 220, 0.15), rgba(220, 200, 240, 0.25)) !important;
  color: #6b4d8a !important;
  border: 1px solid rgba(150, 120, 200, 0.4) !important;
  font-weight: 500;
  letter-spacing: 0.2px;
}

.task-completion-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 12px;
  color: #8a5a3b;
  background: rgba(245, 222, 200, 0.55);
  border: 1px solid rgba(180, 130, 90, 0.25);
}

.task-habit-seed {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border-radius: 6px;
  font-size: 12px;
  color: #4a6b3a;
  background: rgba(220, 235, 200, 0.55);
  border: 1px dashed rgba(140, 180, 100, 0.45);
  cursor: pointer;
  transition: transform 0.1s ease, box-shadow 0.15s ease, background 0.15s ease;
  font-family: inherit;
}
.task-habit-seed:hover {
  transform: translateY(-1px);
  background: rgba(220, 235, 200, 0.85);
  box-shadow: 0 2px 6px rgba(140, 180, 100, 0.3);
}

/* 阶段 5 减法:习惯种子 1 步化 popover */
.habit-popover-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.habit-popover-title {
  margin: 0;
  font-size: 13px;
  font-weight: 500;
  color: #3a4a2a;
}
.habit-frequency-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px;
}
.habit-frequency-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(240, 248, 230, 0.85);
  border: 1px solid rgba(140, 180, 100, 0.35);
  color: #3a4a2a;
  font-size: 12px;
  padding: 7px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, transform 0.1s;
  font-family: inherit;
  text-align: left;
}
.habit-frequency-chip:hover {
  background: rgba(140, 180, 100, 0.18);
  border-color: rgba(140, 180, 100, 0.65);
  transform: translateY(-1px);
}
.habit-frequency-emoji {
  font-size: 14px;
  line-height: 1;
}
.habit-popover-hint {
  margin: 0;
  font-size: 11px;
  color: var(--planner-muted, #6b7280);
  text-align: center;
  opacity: 0.75;
}

/* 阶段 5 减法 (iter 22):1 步评论 popover 样式
   浅冷色系,与抽屉的"重编辑"区分(暖色 chip / 编辑态)
   强调"快速记录",不强调"完整工具" */
.quick-comment-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.quick-comment-title {
  margin: 0;
  font-size: 12px;
  color: var(--planner-muted, #6b7280);
  font-weight: 500;
}
.quick-comment-input {
  width: 100%;
  border: 1px solid var(--el-border-color-lighter, #dcdfe6);
  border-radius: 8px;
  padding: 8px 10px;
  font-size: 13px;
  font-family: inherit;
  resize: none;
  outline: none;
  transition: border-color 0.18s ease, box-shadow 0.18s ease;
  box-sizing: border-box;
  background: var(--el-bg-color, #fff);
  color: var(--planner-text, #1f2937);
}
.quick-comment-input:focus {
  border-color: var(--el-color-primary, #409eff);
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.18);
}
.quick-comment-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.quick-comment-count {
  font-size: 11px;
  color: var(--planner-muted, #9ca3af);
  opacity: 0.85;
}
.quick-comment-loading,
.quick-comment-empty {
  font-size: 12px;
  color: var(--planner-muted, #9ca3af);
  text-align: center;
  padding: 6px 0;
  opacity: 0.85;
}
.quick-comment-preview {
  display: flex;
  flex-direction: column;
  gap: 4px;
  border-top: 1px dashed var(--el-border-color-lighter, #e4e7ed);
  padding-top: 8px;
  margin-top: 4px;
}
.quick-comment-preview-label {
  margin: 0;
  font-size: 11px;
  color: var(--planner-muted, #6b7280);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.quick-comment-preview-item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  padding: 4px 0;
  font-size: 12px;
  line-height: 1.45;
}
.preview-content {
  color: var(--planner-text, #1f2937);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.preview-time {
  color: var(--planner-muted, #9ca3af);
  font-size: 10px;
  flex-shrink: 0;
  white-space: nowrap;
}
.quick-comment-drawer-link {
  margin-top: 4px;
  padding: 4px 0;
  background: transparent;
  border: none;
  color: var(--el-color-primary, #409eff);
  font-size: 11px;
  cursor: pointer;
  text-align: center;
  opacity: 0.85;
  transition: opacity 0.15s ease;
}
.quick-comment-drawer-link:hover {
  opacity: 1;
  text-decoration: underline;
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

/* Meeting search bar */
.meeting-search-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}
.meeting-search-input {
  flex: 1;
}
.meeting-date-filter {
  width: 280px;
}

/* Meeting month divider */
.meeting-month-divider {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 0 8px;
  margin-top: 8px;
  border-bottom: 1px solid var(--border-color, #e4e7ed);
}
.meeting-month-divider:first-child {
  margin-top: 0;
}
.meeting-month-label {
  font-weight: 700;
  font-size: 1.05em;
  color: var(--text-primary, #303133);
}
.meeting-month-count {
  font-size: 0.85em;
  color: var(--text-secondary, #909399);
}

/* Meeting skeleton */
.meeting-skeleton {
  padding: 16px;
  border-radius: 8px;
  background: var(--el-fill-color-blank, #fff);
  margin-bottom: 8px;
}

/* Action item done state */
.meeting-action-item.action-done {
  opacity: 0.6;
}
.meeting-action-item .action-task-done {
  text-decoration: line-through;
  color: var(--text-secondary, #909399);
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

  .triage-panel {
    padding: 16px;
    border-radius: 18px;
  }

  .triage-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .triage-actions {
    display: grid;
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .triage-actions .el-button {
    width: 100%;
    margin: 0;
  }

  .inbox-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .inbox-toolbar .el-button {
    width: 100%;
  }

  .meeting-action-item {
    flex-wrap: wrap;
  }

  .meeting-action-item .el-button {
    margin-left: 0;
    margin-top: 4px;
  }
}

@media (max-width: 560px) {
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

  .triage-actions {
    grid-template-columns: 1fr;
  }

  .triage-card {
    padding: 12px;
  }

  .triage-card h5 {
    font-size: 15px;
  }

  .meeting-action-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .meeting-action-item .el-button {
    width: 100%;
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

/* Triage panel */
.triage-panel {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 16px;
}
.triage-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.triage-card {
  background: var(--el-fill-color-light);
  border-radius: 8px;
  padding: 16px;
}
.triage-card h5 {
  margin: 0 0 8px;
  font-size: 16px;
}
.triage-card p {
  margin: 0 0 12px;
  color: var(--el-text-color-secondary);
}
.triage-actions {
  display: flex;
  gap: 8px;
  margin-top: 16px;
}
.triage-kbd-hint {
  margin: 12px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.triage-discard-btn {
  color: #d97757 !important;
  border-color: rgba(217, 119, 87, 0.4) !important;
}
.triage-discard-btn:hover {
  background: rgba(217, 119, 87, 0.08) !important;
  color: #b85a3d !important;
}
.triage-summary-card {
  background: linear-gradient(135deg, rgba(180, 220, 170, 0.18), rgba(255, 252, 240, 0.92));
  border: 1px solid rgba(120, 170, 130, 0.4);
  border-radius: 14px;
  padding: 22px 24px;
  margin-bottom: 16px;
  display: grid;
  gap: 14px;
}
.triage-summary-headline {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}
.triage-summary-icon {
  font-size: 28px;
  line-height: 1;
}
.triage-summary-headline strong {
  display: block;
  font-size: 17px;
  margin-bottom: 4px;
  color: #3a6048;
}
.triage-summary-headline p {
  margin: 0;
  color: #4a6855;
  font-size: 13px;
}
.triage-summary-stats {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.triage-summary-stat {
  flex: 1 1 0;
  min-width: 90px;
  background: rgba(255, 255, 255, 0.65);
  border: 1px solid rgba(180, 200, 180, 0.5);
  border-radius: 10px;
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.triage-summary-stat strong {
  font-size: 22px;
  line-height: 1.1;
}
.triage-summary-stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.triage-summary-stat-today strong { color: #3a6080; }
.triage-summary-stat-someday strong { color: #8a6a3b; }
.triage-summary-stat-discard strong { color: #b85a3d; }
.triage-summary-message {
  margin: 0;
  font-size: 13px;
  color: #4a6855;
  background: rgba(255, 255, 255, 0.5);
  border-radius: 8px;
  padding: 8px 12px;
}
.triage-summary-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.rollover-insight-card {
  background: linear-gradient(135deg, rgba(255, 240, 220, 0.55), rgba(255, 252, 245, 0.92));
  border: 1px solid rgba(200, 170, 110, 0.35);
  border-radius: 16px;
  padding: 16px 20px;
  margin: 16px 0;
  display: grid;
  gap: 10px;
}
.rollover-insight-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}
.rollover-insight-head strong {
  display: block;
  font-size: 15px;
  color: #8a5a3b;
  margin-bottom: 2px;
}
.rollover-insight-head p {
  margin: 0;
  font-size: 13px;
  color: #7a6a55;
}
.rollover-insight-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  gap: 8px;
}
.rollover-insight-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: rgba(255, 255, 255, 0.6);
  border: 1px solid rgba(200, 170, 110, 0.2);
  border-radius: 10px;
}
.rollover-insight-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
  cursor: pointer;
}
.rollover-insight-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--planner-text, #1f2937);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rollover-insight-meta {
  font-size: 12px;
  color: #8a6a4a;
}
.rollover-insight-meta strong {
  color: #b3612a;
  font-size: 13px;
  margin: 0 1px;
}
.rollover-insight-date {
  margin-left: 4px;
  color: #9a8a7a;
}
.rollover-insight-actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}
.rollover-discard-btn {
  color: #a67a5a !important;
  border-color: rgba(166, 122, 90, 0.35) !important;
}
.rollover-discard-btn:hover {
  background: rgba(166, 122, 90, 0.08) !important;
  color: #8a5a3a !important;
}
/* 阶段 5 减法 (iter 46):顺延回顾 ↻ 顺延到明天 按钮
   蓝色调(冷思考/改期)与 focus.secondary 改天 / focus primary 改明天 色系一致
   与 focus.secondary 批量操作三色闭环(✓ 绿 / ↻ 蓝 / ✗ 棕)完全对称 */
.rollover-postpone-btn {
  color: #1e40af !important;
  border-color: rgba(59, 130, 246, 0.35) !important;
}
.rollover-postpone-btn:hover {
  background: rgba(59, 130, 246, 0.08) !important;
  color: #1e3a8a !important;
  border-color: rgba(59, 130, 246, 0.55) !important;
}
.crossday-hint-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 14px;
  background: linear-gradient(135deg, rgba(180, 210, 230, 0.18), rgba(255, 255, 255, 0.95));
  border: 1px solid rgba(120, 160, 190, 0.32);
  border-radius: 14px;
  padding: 14px 18px;
  margin: 16px 0;
}
.crossday-hint-content {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}
.crossday-hint-icon {
  font-size: 24px;
  line-height: 1;
  flex-shrink: 0;
}
.crossday-hint-content strong {
  display: block;
  font-size: 15px;
  color: #355080;
  margin-bottom: 2px;
}
.crossday-hint-content p {
  margin: 0;
  font-size: 13px;
  color: #4a6080;
}

/* 阶段 6 减法:完成反思卡 — 任务刚完成时温柔追问"感觉怎么样"
   设计要点:浅暖色背景,3 chip + 1 跳过,不阻塞现有操作 */
.completion-feedback-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: linear-gradient(135deg, rgba(255, 220, 180, 0.22), rgba(255, 245, 230, 0.95));
  border: 1px solid rgba(220, 170, 110, 0.4);
  border-radius: 14px;
  padding: 14px 18px;
  margin: 16px 0;
  box-shadow: 0 4px 18px rgba(200, 150, 90, 0.12);
}
.completion-feedback-head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.completion-feedback-icon {
  font-size: 22px;
  line-height: 1;
  flex-shrink: 0;
}
.completion-feedback-head strong {
  display: block;
  font-size: 15px;
  color: #6b4519;
  margin-bottom: 4px;
}
.completion-feedback-head p {
  margin: 0;
  font-size: 13px;
  color: #8a6840;
  line-height: 1.5;
}
.completion-feedback-chips {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.completion-feedback-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.85);
  border: 1px solid rgba(200, 150, 100, 0.45);
  border-radius: 999px;
  color: #6b4519;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.18s ease;
}
.completion-feedback-chip:hover {
  background: rgba(255, 255, 255, 1);
  border-color: rgba(180, 120, 70, 0.7);
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(180, 120, 70, 0.18);
}
.completion-feedback-chip .chip-icon {
  font-size: 15px;
}
.completion-feedback-skip {
  margin-left: auto;
  padding: 4px 10px;
  background: transparent;
  border: none;
  color: #8a6840;
  font-size: 12px;
  cursor: pointer;
  opacity: 0.7;
  transition: opacity 0.15s ease;
}
.completion-feedback-skip:hover {
  opacity: 1;
}
.completion-feedback-fade-enter-active,
.completion-feedback-fade-leave-active {
  transition: opacity 0.32s ease, transform 0.32s ease;
}
.completion-feedback-fade-enter-from,
.completion-feedback-fade-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

.kbd-hint {
  display: inline-block;
  min-width: 18px;
  padding: 0 5px;
  margin-right: 4px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.6);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  line-height: 16px;
  text-align: center;
  color: var(--planner-text, #1f2937);
  vertical-align: middle;
}
.inbox-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

/* Meeting action item */
.meeting-action-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
}
.meeting-action-item .el-button {
  margin-left: auto;
  flex-shrink: 0;
}

/* Global search */
.global-search {
  position: relative;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: rgba(255, 255, 255, 0.75);
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 999px;
  min-width: 240px;
  max-width: 360px;
  flex: 1 1 240px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
}
.global-search:focus-within {
  background: #ffffff;
  border-color: var(--planner-accent, #2563eb);
  box-shadow: 0 0 0 3px var(--planner-accent-soft, rgba(37, 99, 235, 0.18));
}
.global-search-active {
  background: #ffffff;
  border-color: var(--planner-accent, #2563eb);
}
.global-search-icon {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}
.global-search-input {
  flex: 1 1 auto;
  border: none;
  outline: none;
  background: transparent;
  font-size: 13px;
  color: inherit;
  min-width: 0;
}
.global-search-input::placeholder {
  color: var(--el-text-color-placeholder);
}
.global-search-clear {
  cursor: pointer;
  font-size: 18px;
  line-height: 1;
  color: var(--el-text-color-secondary);
  padding: 0 4px;
  flex-shrink: 0;
  user-select: none;
}
.global-search-clear:hover {
  color: #f56c6c;
}
.global-search-kbd {
  font-family: inherit;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  background: rgba(15, 23, 42, 0.06);
  border: 1px solid rgba(15, 23, 42, 0.1);
  border-radius: 4px;
  padding: 1px 6px;
  flex-shrink: 0;
}
.search-dropdown {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  right: 0;
  background: #ffffff;
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  box-shadow: 0 12px 40px rgba(15, 23, 42, 0.15);
  z-index: 30;
  max-height: 60vh;
  overflow-y: auto;
  padding: 6px 0;
}
.search-dropdown-enter-active,
.search-dropdown-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.search-dropdown-enter-from,
.search-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
.search-results {
  list-style: none;
  margin: 0;
  padding: 0;
}
.search-results li {
  display: block;
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 8px;
  margin: 2px 6px;
  transition: background 0.12s ease;
}
.search-results li:hover,
.search-result-active {
  background: var(--planner-accent-soft, rgba(37, 99, 235, 0.1));
}
.search-result-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.search-result-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.search-result-meta {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.search-result-date {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.search-result-detail {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
}
.search-hit {
  background: rgba(245, 108, 108, 0.18);
  color: inherit;
  border-radius: 2px;
  padding: 0 1px;
  font-weight: 700;
}
.search-empty {
  padding: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.search-more {
  padding: 8px 12px 4px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  border-top: 1px dashed var(--el-border-color-light);
  margin-top: 4px;
}
@media (max-width: 768px) {
  .global-search {
    min-width: 0;
    flex: 1 1 auto;
    max-width: none;
  }
  .search-dropdown {
    max-height: 50vh;
  }
}

/* Global Quick Add (Cmd/Ctrl+K) */
.global-quick-add-dialog .el-dialog__header {
  display: none;
}
.global-quick-add-dialog .el-dialog__body {
  padding: 0;
}
.global-quick-add {
  padding: 20px 22px 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.global-quick-add-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.global-quick-add-mode {
  background: var(--planner-accent-soft, rgba(37, 99, 235, 0.14));
  color: var(--planner-accent, #2563eb);
  padding: 2px 10px;
  border-radius: 999px;
  font-weight: 600;
}
.global-quick-add-hint kbd {
  font-family: inherit;
  font-size: 10px;
  background: rgba(15, 23, 42, 0.06);
  border: 1px solid rgba(15, 23, 42, 0.1);
  border-radius: 3px;
  padding: 1px 5px;
  margin: 0 2px;
}
.global-quick-add-textarea {
  width: 100%;
  border: none;
  outline: none;
  background: transparent;
  font-size: 17px;
  line-height: 1.5;
  color: inherit;
  resize: none;
  padding: 6px 0;
  font-family: inherit;
}
.global-quick-add-textarea::placeholder {
  color: var(--el-text-color-placeholder);
}
.global-quick-add-preview {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: var(--planner-accent-soft, rgba(37, 99, 235, 0.1));
  border-radius: 8px;
  color: var(--planner-accent, #2563eb);
  font-size: 13px;
}
.global-quick-add-preview strong {
  font-weight: 700;
}
.global-quick-add-options {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding-top: 4px;
}
.global-quick-add-opt-group {
  display: flex;
  align-items: center;
  gap: 8px;
}
.global-quick-add-opt-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-weight: 600;
}
.global-quick-add-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
  padding-top: 12px;
  flex-wrap: wrap;
}
.global-quick-add-foot-hint {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

/* Shortcut help */
.shortcut-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.shortcut-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 10px 4px;
  border-bottom: 1px dashed var(--el-border-color-lighter);
}
.shortcut-item:last-child {
  border-bottom: none;
}
.shortcut-keys {
  display: inline-flex;
  gap: 4px;
  flex-shrink: 0;
  min-width: 80px;
}
.shortcut-kbd {
  font-family: inherit;
  font-size: 12px;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-bottom-width: 2px;
  border-radius: 6px;
  padding: 2px 8px;
  min-width: 22px;
  text-align: center;
}
.shortcut-label {
  font-size: 13px;
  color: var(--el-text-color-regular);
  flex: 1 1 auto;
}
.shortcut-tip {
  opacity: 0.75;
}

.topbar-shortcut-hint {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-regular);
  line-height: 1;
}
.topbar-quick-add {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-weight: 600;
}
.topbar-quick-add-label {
  margin-left: 2px;
}
@media (max-width: 480px) {
  .topbar-quick-add-label {
    display: none;
  }
  .topbar-quick-add {
    padding: 8px;
    min-width: 36px;
  }
}

/* Settings section */
.settings-section {
  padding: 8px 0;
}
.settings-section h4 {
  margin: 0 0 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-regular);
}
.settings-section .el-switch {
  margin-bottom: 8px;
}

/* Focus mode floating bar */
.focus-bar {
  position: fixed;
  top: 12px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 1500;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 10px 16px;
  background: linear-gradient(135deg, #5b8def, #2563eb);
  color: #fff;
  border-radius: 14px;
  box-shadow: 0 12px 32px rgba(37, 99, 235, 0.32);
  max-width: calc(100% - 24px);
  font-size: 14px;
}
.focus-bar-paused {
  background: linear-gradient(135deg, #909399, #606266);
  box-shadow: 0 12px 32px rgba(96, 98, 102, 0.32);
}
.focus-bar-pulse {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 0 0 0 rgba(255, 255, 255, 0.7);
  flex-shrink: 0;
  animation: focusPulse 1.6s ease-out infinite;
}
.focus-bar-paused .focus-bar-pulse {
  animation: none;
  background: rgba(255, 255, 255, 0.6);
}
@keyframes focusPulse {
  0% { box-shadow: 0 0 0 0 rgba(255, 255, 255, 0.7); }
  70% { box-shadow: 0 0 0 10px rgba(255, 255, 255, 0); }
  100% { box-shadow: 0 0 0 0 rgba(255, 255, 255, 0); }
}
.focus-bar-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.focus-bar-label {
  font-size: 11px;
  opacity: 0.85;
  letter-spacing: 0.5px;
}
.focus-bar-title {
  font-size: 14px;
  font-weight: 600;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.focus-bar-timer {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 0 8px;
  border-left: 1px solid rgba(255, 255, 255, 0.3);
  border-right: 1px solid rgba(255, 255, 255, 0.3);
}
.focus-bar-time {
  font-size: 18px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.5px;
}
.focus-bar-state {
  font-size: 10px;
  opacity: 0.75;
}
.focus-bar-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}
.focus-bar-actions .el-button {
  background: rgba(255, 255, 255, 0.15);
  border-color: rgba(255, 255, 255, 0.3);
  color: #fff;
}
.focus-bar-actions .el-button:hover {
  background: rgba(255, 255, 255, 0.28);
  border-color: rgba(255, 255, 255, 0.5);
  color: #fff;
}
.focus-bar-actions .el-button--primary {
  background: rgba(255, 255, 255, 0.92);
  color: #2563eb;
  border-color: rgba(255, 255, 255, 0.92);
}
.focus-bar-actions .el-button--primary:hover {
  background: #fff;
  color: #1d4ed8;
}
.focus-bar-actions .el-button--success {
  background: rgba(103, 194, 58, 0.95);
  border-color: rgba(103, 194, 58, 0.95);
  color: #fff;
}
.focus-bar-actions .el-button--success:hover {
  background: #67c23a;
  color: #fff;
}
.focus-bar-enter-active,
.focus-bar-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}
.focus-bar-enter-from,
.focus-bar-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-12px);
}
.focus-start-btn-active {
  background: linear-gradient(135deg, rgba(37, 99, 235, 0.16), rgba(37, 99, 235, 0.08)) !important;
  border-color: #2563eb !important;
  color: #2563eb !important;
  font-weight: 600;
}
.summary-chip-focus {
  background: rgba(15, 23, 42, 0.04);
  font-variant-numeric: tabular-nums;
}
.summary-chip-focus-active {
  background: linear-gradient(135deg, rgba(37, 99, 235, 0.12), rgba(91, 141, 239, 0.18));
  border-color: rgba(37, 99, 235, 0.4);
  color: #1d4ed8;
}
.summary-chip-focus-active strong {
  color: #1d4ed8;
  font-size: 16px;
}

/* 阶段 5 减法:End-of-Day 收尾 chip — 浅绿渐变(与收尾仪式同色系,健康/收尾语义) */
.summary-chip-finish {
  background: linear-gradient(135deg, rgba(180, 220, 170, 0.55), rgba(220, 235, 200, 0.65));
  border-color: rgba(140, 180, 100, 0.55);
  color: #3a4a2a;
  cursor: pointer;
  font-variant-numeric: tabular-nums;
  transition: transform 0.1s ease, box-shadow 0.15s ease;
}
.summary-chip-finish:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(140, 180, 100, 0.35);
  background: linear-gradient(135deg, rgba(180, 220, 170, 0.75), rgba(220, 235, 200, 0.85));
}
.summary-chip-finish strong {
  color: #4a6b3a;
  font-size: 16px;
  margin: 0 2px;
}
.summary-chip-finish-label {
  font-size: 12px;
}

/* 阶段 5 减法 (iter 29):今日进度 chip — 合并"今日完成" + "收尾今天"为 1 个可视化 chip
   - 进度条让"还差几件" 1 步可见
   - 全部完成时绿色 + 庆祝文案
   - 1 件都没做时灰色,提示"开始动起来"
   - 0 件今日时变体(直接显示"今天清爽")
*/
.summary-chip-progress {
  cursor: pointer;
  position: relative;
  font-variant-numeric: tabular-nums;
  background: linear-gradient(135deg, rgba(99, 132, 255, 0.18), rgba(167, 139, 250, 0.18));
  border-color: rgba(99, 132, 255, 0.35);
  min-width: 96px;
  transition: transform 0.1s ease, box-shadow 0.15s ease;
}
.summary-chip-progress:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(99, 132, 255, 0.3);
}
.summary-chip-progress-full {
  background: linear-gradient(135deg, rgba(180, 220, 170, 0.65), rgba(220, 235, 200, 0.75)) !important;
  border-color: rgba(140, 180, 100, 0.6) !important;
  color: #3a4a2a !important;
  cursor: default;
}
.summary-chip-progress-full:hover {
  transform: none;
  box-shadow: none;
}
.summary-chip-progress-empty {
  background: linear-gradient(135deg, rgba(160, 160, 160, 0.18), rgba(180, 180, 180, 0.22)) !important;
  border-color: rgba(160, 160, 160, 0.3) !important;
  color: #6b6b6b !important;
  cursor: default;
}
.summary-chip-progress-label {
  font-size: 12px;
  margin-right: 2px;
}
.summary-chip-progress-divider {
  margin: 0 2px;
  opacity: 0.5;
  font-weight: 400;
}
.summary-chip-progress-track {
  position: absolute;
  left: 8px;
  right: 8px;
  bottom: 4px;
  height: 4px;
  border-radius: 2px;
  background: rgba(255, 255, 255, 0.5);
  overflow: hidden;
}
.summary-chip-progress-full .summary-chip-progress-track {
  display: none;
}
.summary-chip-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #10b981, #34d399);
  border-radius: 2px;
  transition: width 0.35s cubic-bezier(0.16, 1, 0.3, 1);
}
.summary-chip-progress-empty .summary-chip-progress-fill {
  background: linear-gradient(90deg, #94a3b8, #cbd5e1);
}
@media (max-width: 640px) {
  .focus-bar {
    top: 8px;
    padding: 8px 12px;
    gap: 8px;
    flex-wrap: wrap;
    justify-content: space-between;
  }
  .focus-bar-title {
    max-width: 140px;
    font-size: 13px;
  }
  .focus-bar-time {
    font-size: 16px;
  }
  .focus-bar-actions .el-button {
    padding: 4px 8px;
    font-size: 12px;
  }
}
.settings-section {
  padding: 8px 0;
}
.settings-section h4 {
  margin: 0 0 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-regular);
}
.settings-section .el-switch {
  margin-bottom: 8px;
}

/* Quick repeat chips */
.quick-repeat-row {
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.quick-row-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  font-weight: 500;
  flex-shrink: 0;
}
.quick-repeat-chips {
  display: inline-flex;
  gap: 6px;
  flex-wrap: wrap;
}
.repeat-chip {
  padding: 5px 12px;
  border-radius: 999px;
  border: 1px solid var(--el-border-color-lighter);
  background: rgba(255, 255, 255, 0.7);
  color: var(--el-text-color-regular);
  cursor: pointer;
  font-size: 13px;
  transition: all 0.15s ease;
}
.repeat-chip:hover:not(:disabled) {
  border-color: var(--planner-accent, #2563eb);
  color: var(--planner-accent, #2563eb);
}
.repeat-chip-active {
  background: var(--planner-accent, #2563eb);
  border-color: var(--planner-accent, #2563eb);
  color: #fff;
  font-weight: 500;
}
.repeat-chip-active:hover {
  color: #fff;
  background: var(--planner-accent, #2563eb);
  filter: brightness(1.08);
}
.repeat-chip:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.quick-repeat-hint {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #e6a23c;
}

/* Pin toggle on task cards */
.pin-toggle:hover {
  border-color: #e6a23c;
  color: #e6a23c;
  background: rgba(230, 162, 60, 0.08);
}
.pin-toggle-active {
  background: linear-gradient(135deg, #fadb14, #e6a23c);
  border-color: #e6a23c;
  color: #fff;
  box-shadow: 0 2px 6px rgba(230, 162, 60, 0.35);
}
.pin-toggle-active:hover {
  background: linear-gradient(135deg, #ffc53d, #d99026);
  color: #fff;
}
.focus-card-pinned {
  border-color: #e6a23c;
  background: linear-gradient(135deg, rgba(250, 219, 20, 0.07), rgba(255, 255, 255, 0.95));
  box-shadow: 0 8px 24px rgba(230, 162, 60, 0.18);
}
.focus-card-pinned::before {
  background: linear-gradient(180deg, #fadb14, #e6a23c) !important;
}
.focus-card-rest {
  border-color: rgba(120, 150, 180, 0.4);
  background: linear-gradient(135deg, rgba(180, 200, 220, 0.10), rgba(255, 255, 255, 0.95));
}
.focus-card-rest::before {
  background: linear-gradient(180deg, #a8c0d8, #7898b8) !important;
}
.focus-card-done {
  border-color: rgba(120, 170, 130, 0.42);
  background: linear-gradient(135deg, rgba(180, 220, 170, 0.14), rgba(255, 252, 240, 0.95));
}
.focus-card-done::before {
  background: linear-gradient(180deg, #a8c898, #7ea878) !important;
}
.focus-pin-icon {
  color: #e6a23c;
  margin-right: 4px;
  vertical-align: -2px;
}
.focus-card-actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

/* 阶段 5 减法 (iter 31):focus card 主区域 1 步可点(进入详情)
   hover 时显示下划线 + cursor pointer,提示"这是可点的" */
.focus-card-content-clickable {
  cursor: pointer;
  border-radius: 6px;
  padding: 4px 6px;
  margin: -4px -6px;
  transition: background 0.15s ease;
}
.focus-card-content-clickable:hover {
  background: rgba(99, 132, 255, 0.08);
}
.focus-card-content-clickable:hover h3 {
  text-decoration: underline;
  text-decoration-color: rgba(99, 132, 255, 0.5);
  text-underline-offset: 3px;
}

/* 阶段 5 减法 (iter 35):事件卡 body 1 步可点(与 focus card iter 31 对称)
   事件用绿色调 hover 区分(事件 = 行动/出发感),focus card 用蓝紫(任务 = 思维感) */
.event-card-content-clickable {
  cursor: pointer;
  border-radius: 6px;
  padding: 4px 6px;
  margin: -4px -6px;
  transition: background 0.15s ease;
}
.event-card-content-clickable:hover {
  background: rgba(34, 197, 94, 0.1);
}
.event-card-content-clickable:hover h3 {
  text-decoration: underline;
  text-decoration-color: rgba(34, 197, 94, 0.55);
  text-underline-offset: 3px;
}
.summary-chip-streak {
  background: rgba(15, 23, 42, 0.04);
}
.summary-chip-streak-active {
  background: linear-gradient(135deg, rgba(255, 120, 50, 0.12), rgba(255, 200, 80, 0.18));
  border-color: rgba(255, 145, 60, 0.4);
  color: #b35a16;
}
.summary-chip-streak-active strong {
  color: #d3520a;
  font-size: 18px;
}
/* 阶段 5 减法 (iter 51):🔥 连续 chip 1 步 = 最近 view · active 反馈
   与 🚨 紧急 iter 39 视觉对称(暖色加深 + inset 描边)
   颜色保持橙系(连续 = 火焰色),与 streak-active 叠加形成"既连续又在看"双重提示 */
.summary-chip-streak-view-active {
  box-shadow: 0 0 0 3px rgba(255, 120, 50, 0.35) inset, 0 4px 12px rgba(255, 145, 60, 0.22);
}
.summary-chip-tomorrow {
  cursor: pointer;
  transition: transform 0.1s ease, box-shadow 0.15s ease;
}
.summary-chip-tomorrow:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(99, 102, 241, 0.18);
}
.summary-chip-tomorrow-has {
  background: linear-gradient(135deg, rgba(99, 130, 200, 0.10), rgba(150, 180, 230, 0.14));
  border-color: rgba(99, 130, 200, 0.3);
  color: #355080;
}
.summary-chip-tomorrow-has strong {
  color: #355080;
  font-size: 18px;
}
.summary-chip-energy {
  cursor: pointer;
  transition: transform 0.1s ease, box-shadow 0.15s ease, background 0.15s ease;
}
.summary-chip-energy:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(120, 130, 150, 0.18);
}
.summary-chip-energy-on {
  background: linear-gradient(135deg, rgba(180, 200, 220, 0.18), rgba(150, 175, 200, 0.22));
  border-color: rgba(120, 150, 180, 0.45);
  color: #4a6080;
}
.summary-chip-energy-label {
  font-size: 13px;
  margin-left: 2px;
}
.summary-chip-focus {
  cursor: pointer;
  transition: transform 0.1s ease, box-shadow 0.15s ease, background 0.15s ease;
  user-select: none;
}
.summary-chip-focus:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(99, 102, 241, 0.18);
}
.summary-chip-focus-on {
  background: linear-gradient(135deg, rgba(190, 160, 230, 0.16), rgba(220, 195, 240, 0.22));
  border-color: rgba(150, 110, 200, 0.45);
  color: #5a3a8a;
}
.summary-chip-focus-on strong {
  color: #5a3a8a;
}
.summary-chip-focus-label {
  font-weight: 600;
  font-size: 13px;
  margin-left: 2px;
}
.timeline-fold-banner-focus {
  background: linear-gradient(135deg, rgba(190, 160, 230, 0.10), rgba(255, 255, 255, 0.95));
  border-color: rgba(150, 110, 200, 0.32);
  color: #5a3a8a;
}
.summary-chip-mode {
  cursor: pointer;
  transition: transform 0.1s ease, box-shadow 0.15s ease, background 0.15s ease;
  user-select: none;
}
.summary-chip-mode:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(99, 102, 241, 0.18);
}
.summary-chip-mode-work {
  background: linear-gradient(135deg, rgba(99, 130, 220, 0.10), rgba(140, 165, 220, 0.14));
  border-color: rgba(99, 130, 220, 0.32);
  color: #2a4a8a;
}
.summary-chip-mode-work strong, .summary-chip-mode-work .summary-chip-mode-label {
  color: #2a4a8a;
}
.summary-chip-mode-life {
  background: linear-gradient(135deg, rgba(120, 175, 130, 0.10), rgba(180, 210, 150, 0.16));
  border-color: rgba(120, 175, 130, 0.34);
  color: #3a6048;
}
.summary-chip-mode-life strong, .summary-chip-mode-life .summary-chip-mode-label {
  color: #3a6048;
}
.summary-chip-mode-icon {
  font-size: 15px;
  margin-right: 1px;
}
.summary-chip-mode-label {
  font-weight: 600;
  font-size: 13px;
}
.summary-chip-mode-hint {
  font-size: 11px;
  opacity: 0.6;
  margin-left: 4px;
  border-left: 1px solid currentColor;
  padding-left: 6px;
}
.summary-chip-sub {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  margin-left: 2px;
}

/* 近 7 天完成 mini 图 */
.summary-chip-week {
  min-width: 0;
}
.week-bars {
  display: inline-flex;
  align-items: flex-end;
  gap: 2px;
  height: 22px;
  margin: 0 4px;
}
.week-bar {
  display: inline-block;
  width: 4px;
  background: rgba(99, 102, 241, 0.18);
  border-radius: 2px;
  height: var(--h, 6%);
  transition: height 0.25s ease, background 0.2s ease;
}
.week-bar-active {
  background: linear-gradient(180deg, #f59e0b, #fb923c);
}
.week-bar-today {
  outline: 1.5px solid #f59e0b;
  outline-offset: 1px;
}

/* Inbox batch operations */
.pin-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  background: rgba(255, 255, 255, 0.7);
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
  font-size: 14px;
}
.pin-toggle:hover {
  border-color: #e6a23c;
  color: #e6a23c;
  background: rgba(230, 162, 60, 0.08);
}
.pin-toggle-active {
  background: linear-gradient(135deg, #fadb14, #e6a23c);
  border-color: #e6a23c;
  color: #fff;
  box-shadow: 0 2px 6px rgba(230, 162, 60, 0.35);
}
.pin-toggle-active:hover {
  background: linear-gradient(135deg, #ffc53d, #d99026);
  color: #fff;
}
.inbox-select {
  margin-right: 8px;
  flex-shrink: 0;
}
.inbox-select + h5 {
  display: inline;
}
.task-copy {
  display: flex;
  align-items: flex-start;
  gap: 4px;
}
.task-card-selected {
  border-color: var(--planner-accent, #2563eb);
  background: var(--planner-accent-soft, rgba(37, 99, 235, 0.08));
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.15);
}
.inbox-batch-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  margin: 8px 0 12px;
  background: linear-gradient(135deg, rgba(37, 99, 235, 0.08), rgba(103, 194, 58, 0.06));
  border: 1px solid var(--planner-accent, #2563eb);
  border-radius: 10px;
  flex-wrap: wrap;
}
.inbox-batch-bar > span {
  font-size: 13px;
  font-weight: 600;
  color: var(--planner-accent, #2563eb);
  margin-right: 4px;
}
.batch-bar-enter-active,
.batch-bar-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.batch-bar-enter-from,
.batch-bar-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
@media (max-width: 480px) {
  .pin-toggle {
    width: 32px;
    height: 32px;
  }
  .inbox-batch-bar {
    gap: 6px;
  }
  .inbox-batch-bar .el-button {
    font-size: 12px;
    padding: 4px 8px;
  }
}

/* Quick form live hints (smart auto-classify) */
.quick-form-hints {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: linear-gradient(90deg, rgba(99, 102, 241, 0.08), rgba(99, 102, 241, 0.02));
  border: 1px dashed rgba(99, 102, 241, 0.25);
  border-radius: 8px;
  font-size: 12px;
  color: var(--planner-accent, #6366f1);
  animation: hint-fade-in 0.25s ease-out;
}
.quick-form-hints .el-icon {
  font-size: 13px;
}
.quick-form-hints strong {
  font-weight: 600;
}
@keyframes hint-fade-in {
  from { opacity: 0; transform: translateY(-2px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 阶段 5 减法 (iter 24):多任务预览 */
.quick-batch-preview {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 6px;
  padding: 8px 12px;
  background: linear-gradient(90deg, rgba(16, 185, 129, 0.08), rgba(16, 185, 129, 0.02));
  border: 1px dashed rgba(16, 185, 129, 0.3);
  border-radius: 8px;
  font-size: 12px;
  color: #047857;
  animation: hint-fade-in 0.25s ease-out;
}
.quick-batch-preview .el-icon {
  font-size: 13px;
  margin-top: 2px;
}
.quick-batch-label {
  font-weight: 600;
  white-space: nowrap;
  margin-top: 1px;
}
.quick-batch-list {
  flex: 1;
  min-width: 0;
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.quick-batch-list li {
  display: flex;
  align-items: baseline;
  gap: 6px;
  font-size: 12px;
  line-height: 1.5;
}
.quick-batch-num {
  color: #059669;
  font-weight: 600;
  flex-shrink: 0;
  min-width: 18px;
}
.quick-batch-title {
  font-weight: 500;
  color: #064e3b;
  word-break: break-word;
}
.quick-batch-hint {
  color: #6b7280;
  font-size: 11px;
  margin-left: auto;
  white-space: nowrap;
  flex-shrink: 0;
}
.quick-batch-more {
  color: #6b7280;
  font-style: italic;
  font-size: 11px;
}

/* Import picker (hidden file input) */
.import-file-input {
  display: none;
}
.import-progress {
  margin-top: 8px;
  padding: 8px 12px;
  background: var(--planner-soft, rgba(99, 102, 241, 0.08));
  border-radius: 8px;
  font-size: 12px;
  color: var(--planner-accent, #6366f1);
}

/* ICS calendar subscription */
.ics-url-row {
  display: flex;
  gap: 8px;
  align-items: stretch;
  margin: 12px 0 8px;
}
.ics-url-input {
  flex: 1;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
}
.ics-url-input :deep(.el-input__inner) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
}
.ics-options-row {
  display: flex;
  align-items: center;
  gap: 16px;
  margin: 4px 0 12px;
  font-size: 12px;
  color: var(--planner-muted, #6b7280);
}
.warn-text {
  color: #b91c1c;
  font-size: 12px;
}
.ics-help {
  margin-top: 8px;
  font-size: 12px;
  color: var(--planner-muted, #6b7280);
}
.ics-help summary {
  cursor: pointer;
  padding: 4px 0;
  user-select: none;
  color: var(--planner-accent, #6366f1);
}
.ics-help summary:hover {
  text-decoration: underline;
}
.ics-help ul {
  margin: 8px 0 0;
  padding-left: 20px;
  line-height: 1.8;
}
.ics-help strong {
  color: var(--planner-text, #1f2937);
}

/* Task date chip (1-step date/postpone) */
.task-date-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--planner-muted, #6b7280);
  font-size: 12px;
  padding: 2px 6px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}
.task-date-chip:hover {
  background: var(--planner-soft, rgba(99, 102, 241, 0.08));
  border-color: var(--planner-soft, rgba(99, 102, 241, 0.18));
  color: var(--planner-accent, #6366f1);
}
.task-date-chip .el-icon {
  font-size: 12px;
}
/* 阶段 5 减法 (iter 23):已完成任务的日期 chip,绿色不可点 */
.task-date-chip-completed {
  background: rgba(16, 185, 129, 0.08);
  border-color: rgba(16, 185, 129, 0.25);
  color: #059669;
  cursor: default;
}
.task-date-chip-completed:hover {
  background: rgba(16, 185, 129, 0.12);
  border-color: rgba(16, 185, 129, 0.4);
  color: #047857;
}

.date-popover-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.date-popover-quick {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 6px;
}
.date-popover-hint {
  margin: 0;
  font-size: 11px;
  color: var(--planner-muted, #6b7280);
  text-align: center;
  opacity: 0.75;
}

/* 阶段 5 减法:取消 1 步化 popover */
.task-cancel-btn {
  color: #a67a5a !important;
  border-color: rgba(166, 122, 90, 0.35) !important;
}
.task-cancel-btn:hover {
  background: rgba(166, 122, 90, 0.08) !important;
  border-color: rgba(166, 122, 90, 0.55) !important;
  color: #8a5a3b !important;
}
/* 阶段 5 减法 (iter 44):task-cancel-btn icon-only 模式 — 紧凑圆形按钮
   用于 focus.secondary item-row 等空间紧凑场景
   取消/改天 是用户对 1 个次要的 2 个主要决策,需要并排但不能挤压"点击完成"区
   与 .focus-secondary-item-postpone 等高同 padding,视觉一致 */
.task-cancel-btn.task-cancel-btn-icon-only {
  padding: 4px 8px !important;
  min-height: 26px !important;
  font-size: 13px !important;
  border-radius: 14px !important;
}
.task-cancel-btn.task-cancel-btn-icon-only .el-icon {
  font-size: 14px;
}
.cancel-popover-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.cancel-popover-title {
  margin: 0;
  font-size: 13px;
  font-weight: 500;
  color: #5a3a2a;
}
.cancel-reason-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px;
}
.cancel-reason-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(255, 248, 240, 0.85);
  border: 1px solid rgba(166, 122, 90, 0.25);
  color: #5a3a2a;
  font-size: 12px;
  padding: 6px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, transform 0.1s;
  text-align: left;
}
.cancel-reason-chip:hover {
  background: rgba(166, 122, 90, 0.1);
  border-color: rgba(166, 122, 90, 0.55);
  transform: translateY(-1px);
}
.cancel-reason-chip-active {
  background: rgba(166, 122, 90, 0.18);
  border-color: #a67a5a;
  color: #5a3a2a;
  font-weight: 500;
}
.cancel-reason-emoji {
  font-size: 14px;
  line-height: 1;
}
.cancel-popover-input :deep(.el-input__wrapper) {
  background: rgba(255, 248, 240, 0.7);
  border-radius: 8px;
}
.cancel-popover-hint {
  margin: 0;
  font-size: 11px;
  color: var(--planner-muted, #6b7280);
  text-align: center;
  opacity: 0.75;
}

/* Completion celebration */
.celebration-layer {
  position: fixed;
  top: 30%;
  left: 50%;
  transform: translateX(-50%);
  pointer-events: none;
  z-index: 2000;
}
.celebration-burst {
  position: absolute;
  width: 0;
  height: 0;
}
.celebration-spark {
  position: absolute;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--planner-accent, #2563eb);
  top: 0;
  left: 0;
  transform: translate(-50%, -50%);
  animation: sparkBurst 0.9s ease-out forwards;
  animation-delay: calc(var(--i) * 18ms);
  opacity: 0.9;
}
.celebration-spark:nth-child(1),
.celebration-spark:nth-child(5) { background: #f56c6c; }
.celebration-spark:nth-child(2),
.celebration-spark:nth-child(6) { background: #67c23a; }
.celebration-spark:nth-child(3),
.celebration-spark:nth-child(7) { background: #e6a23c; }
.celebration-spark:nth-child(4),
.celebration-spark:nth-child(8) { background: #909399; }
.celebration-spark:nth-child(1) { --tx: 90px;  --ty: 0px; }
.celebration-spark:nth-child(2) { --tx: 64px;  --ty: 64px; }
.celebration-spark:nth-child(3) { --tx: 0px;   --ty: 90px; }
.celebration-spark:nth-child(4) { --tx: -64px; --ty: 64px; }
.celebration-spark:nth-child(5) { --tx: -90px; --ty: 0px; }
.celebration-spark:nth-child(6) { --tx: -64px; --ty: -64px; }
.celebration-spark:nth-child(7) { --tx: 0px;   --ty: -90px; }
.celebration-spark:nth-child(8) { --tx: 64px;  --ty: -64px; }
@keyframes sparkBurst {
  0% {
    transform: translate(-50%, -50%) scale(0.4);
    opacity: 0;
  }
  20% {
    opacity: 1;
  }
  100% {
    transform:
      translate(
        calc(-50% + var(--tx)),
        calc(-50% + var(--ty))
      )
      scale(0.2);
    opacity: 0;
  }
}
.task-card-celebrate {
  animation: taskCompletePulse 1.2s ease-out;
}
@keyframes taskCompletePulse {
  0% { transform: scale(1); }
  20% { transform: scale(1.03); box-shadow: 0 0 0 4px rgba(103, 194, 58, 0.18); }
  100% { transform: scale(1); box-shadow: 0 0 0 0 rgba(103, 194, 58, 0); }
}

.task-card-next-up {
  animation: taskNextUp 1.8s ease;
}
@keyframes taskNextUp {
  0% { box-shadow: 0 0 0 0 rgba(99, 102, 241, 0); }
  20% { box-shadow: 0 0 0 6px rgba(99, 102, 241, 0.25); }
  100% { box-shadow: 0 0 0 0 rgba(99, 102, 241, 0); }
}
.planner-quiet .task-card-next-up {
  animation: none;
}

/* 安静模式:整体压下所有"庆祝类"动画与浮层,尊重用户沉浸整理的诉求 */
.planner-quiet .celebration-layer {
  display: none;
}
.planner-quiet .task-card-celebrate {
  animation: none;
}
.planner-quiet .focus-bar-pulse {
  animation: none;
  background: rgba(255, 255, 255, 0.6);
}

/* 全局搜索:分节标签 + 全文结果样式 */
.search-section-label {
  list-style: none;
  padding: 8px 12px 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  letter-spacing: 0.02em;
  user-select: none;
  cursor: default;
}
.search-section-label .search-section-loading {
  margin-left: 4px;
  color: var(--el-color-primary);
  animation: searchBlink 1.2s ease-in-out infinite;
}
.search-results-server {
  border-top: 1px dashed var(--el-border-color-lighter);
  margin-top: 4px;
  padding-top: 2px;
}
.search-result-server {
  cursor: pointer;
}
.search-result-server.search-result-active {
  background: var(--el-color-primary-light-9);
}
@keyframes searchBlink {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 1; }
}

/* 搜索结果跳转到评论/录音时的高亮闪烁 */
.comment-highlight-flash {
  animation: commentHighlight 1.6s ease-out;
  border-radius: 8px;
}
@keyframes commentHighlight {
  0%   { background: rgba(64, 158, 255, 0.25); box-shadow: 0 0 0 4px rgba(64, 158, 255, 0.18); }
  60%  { background: rgba(64, 158, 255, 0.12); box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.08); }
  100% { background: transparent; box-shadow: 0 0 0 0 rgba(64, 158, 255, 0); }
}
</style>
