<template>
  <div class="planner-shell" :class="`mode-${activeKind}`">
    <div class="planner-backdrop planner-backdrop-a"></div>
    <div class="planner-backdrop planner-backdrop-b"></div>

    <section v-if="!profileId" class="entry-layout">
      <div class="entry-hero">
        <p class="eyebrow">Planner Archive</p>
        <h1>工作和生活分开管理，手机上也顺手。</h1>
        <p class="entry-copy">
          工作事项和生活事项完全分区，支持语音输入、提醒邮件、手机日历导入和 AI 整理。
        </p>
        <div class="entry-tips">
          <span>工作日 09:00-18:00 默认工作模式</span>
          <span>下班后默认生活模式</span>
          <span>一键切换，不锁死</span>
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
        <div>
          <p class="eyebrow">{{ activeKind === 'work' ? 'Work Zone' : 'Life Zone' }}</p>
          <h2>{{ profile.name || '事项档案' }}</h2>
          <p class="subtle">{{ modeHint }}</p>
        </div>
        <div class="topbar-actions">
          <el-button circle @click="settingsVisible = true">
            <el-icon><Setting /></el-icon>
          </el-button>
          <el-button circle @click="refreshAll">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </header>

      <section class="mode-switcher">
        <button
          class="mode-tab"
          :class="{ active: activeKind === 'work' }"
          @click="switchKind('work')"
        >
          <span class="mode-tab-title">工作模式</span>
          <span class="mode-tab-sub">推进、专注、交付</span>
        </button>
        <button
          class="mode-tab"
          :class="{ active: activeKind === 'life' }"
          @click="switchKind('life')"
        >
          <span class="mode-tab-title">生活模式</span>
          <span class="mode-tab-sub">照顾自己，也照顾节奏</span>
        </button>
      </section>

      <section class="hero-card">
        <div>
          <p class="hero-label">{{ activeKind === 'work' ? '今日推进感' : '今日生活感' }}</p>
          <h3>{{ currentTip.title }}</h3>
          <p>{{ currentTip.body }}</p>
        </div>
        <div class="hero-stats">
          <div class="hero-stat">
            <span>待处理</span>
            <strong>{{ timelineCounts.open + timelineCounts.in_progress }}</strong>
          </div>
          <div class="hero-stat">
            <span>已完成</span>
            <strong>{{ timelineCounts.done }}</strong>
          </div>
          <div class="hero-stat">
            <span>顺延中</span>
            <strong>{{ timelineCounts.rolled_over }}</strong>
          </div>
        </div>
      </section>

      <section class="quick-panel">
        <div class="panel-heading">
          <div>
            <h3>{{ activeKind === 'work' ? '快速记录工作事项' : '快速记录生活事项' }}</h3>
            <p>{{ activeKind === 'work' ? '先记下来，再排优先级。' : '别让小事占着脑子，先落下来。' }}</p>
          </div>
          <div class="panel-heading-actions">
            <el-button plain @click="openVoice('quick')">
              <el-icon><Microphone /></el-icon>
              语音输入
            </el-button>
            <el-button type="primary" plain @click="aiDialogVisible = true">
              <el-icon><MagicStick /></el-icon>
              AI 整理
            </el-button>
          </div>
        </div>

        <div class="quick-grid">
          <el-input
            v-model="quickForm.title"
            placeholder="比如：确认联调结果 / 买水果 / 预约体检"
            maxlength="80"
            show-word-limit
          />
          <el-input v-model="quickForm.detail" type="textarea" :rows="3" placeholder="补充描述、上下文、边界条件" />
          <div class="quick-row">
            <el-date-picker v-model="quickForm.plannedFor" type="date" value-format="YYYY-MM-DD" placeholder="计划日期" />
            <el-select v-model="quickForm.priority" placeholder="优先级">
              <el-option label="高优先级" value="high" />
              <el-option label="中优先级" value="medium" />
              <el-option label="低优先级" value="low" />
            </el-select>
          </div>
          <div class="quick-row">
            <el-date-picker
              v-model="quickForm.remindAt"
              type="datetime"
              value-format="YYYY-MM-DDTHH:mm"
              placeholder="提醒时间"
            />
            <el-input v-model="quickForm.notifyEmail" placeholder="提醒邮箱，默认用档案邮箱" />
          </div>
          <div class="quick-actions">
            <el-button :loading="savingQuick" type="primary" @click="createQuickTask">
              保存事项
            </el-button>
            <el-button @click="fillTodayDefaults">恢复默认</el-button>
          </div>
        </div>
      </section>

      <transition name="mode-panel" mode="out-in">
        <section :key="activeKind" class="timeline-panel">
          <div class="timeline-header">
            <div>
              <h3>{{ activeKind === 'work' ? '工作时间线' : '生活时间线' }}</h3>
              <p>{{ activeKind === 'work' ? '尽量让工作事项聚焦在少量关键动作上。' : '生活事项也值得被温柔安排。' }}</p>
            </div>
            <el-button plain @click="refreshTimeline">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>

          <div v-if="timelineLoading" class="timeline-empty">正在加载时间线...</div>
          <div v-else-if="timelineGroups.length === 0" class="timeline-empty">
            <strong>{{ activeKind === 'work' ? '还没有工作事项' : '还没有生活事项' }}</strong>
            <span>{{ activeKind === 'work' ? '先记下一件今天最值得推进的事。' : '先记下一件能让你更轻松的小事。' }}</span>
          </div>

          <div v-else class="timeline-groups">
            <div v-for="group in timelineGroups" :key="group.date" class="timeline-group">
              <div class="timeline-group-head">
                <div>
                  <h4>{{ group.label }}</h4>
                  <p>{{ group.date }}</p>
                </div>
                <el-tag size="small">{{ group.items.length }} 项</el-tag>
              </div>

              <transition-group name="task-fade" tag="div" class="task-list">
                <article
                  v-for="task in group.items"
                  :key="task.id"
                  class="task-card"
                  :class="[`status-${task.status}`, `priority-${task.priority}`]"
                  @click="openTask(task)"
                >
                  <div class="task-card-top">
                    <div>
                      <h5>{{ task.title }}</h5>
                      <p>{{ task.detail || '点击补充详情、评论和提醒' }}</p>
                    </div>
                    <div class="task-tags">
                      <el-tag size="small" :type="priorityTagType(task.priority)">{{ priorityLabel(task.priority) }}</el-tag>
                      <el-tag size="small" :type="statusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
                    </div>
                  </div>

                  <div class="task-meta">
                    <span v-if="task.remind_at">提醒 {{ formatDateTime(task.remind_at) }}</span>
                    <span v-else>计划 {{ task.planned_for }}</span>
                    <span v-if="task.is_rolled_over" class="rolled-tag">未完成，已顺延到今天</span>
                  </div>

                  <div class="task-actions" @click.stop>
                    <el-button size="small" type="primary" plain @click="setTaskStatus(task, task.status === 'done' ? 'open' : 'done')">
                      {{ task.status === 'done' ? '重新打开' : '完成' }}
                    </el-button>
                    <el-button size="small" plain @click="postponeTask(task)">顺延一天</el-button>
                    <el-button size="small" plain @click="openCommentDrawer(task)">评论</el-button>
                    <el-button size="small" plain @click="importCalendar(task)">导入日历</el-button>
                  </div>
                </article>
              </transition-group>
            </div>
          </div>
        </section>
      </transition>
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
        <div class="soft-note">延期天数不是默认动作，只有你填写时才会顺延。</div>
      </div>
    </el-drawer>

    <el-drawer v-model="taskDrawerVisible" :title="taskForm.id ? '编辑事项' : '事项详情'" size="480px">
      <div class="drawer-stack">
        <el-input v-model="taskForm.title" placeholder="事项标题" />
        <el-select v-model="taskForm.kind">
          <el-option label="工作事项" value="work" />
          <el-option label="生活事项" value="life" />
        </el-select>
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
        <el-date-picker v-model="taskForm.plannedFor" type="date" value-format="YYYY-MM-DD" placeholder="计划日期" />
        <el-date-picker v-model="taskForm.remindAt" type="datetime" value-format="YYYY-MM-DDTHH:mm" placeholder="提醒时间" />
        <el-input v-model="taskForm.notifyEmail" placeholder="提醒邮箱" />
        <el-input v-model="taskForm.detail" type="textarea" :rows="4" placeholder="详情" />
        <el-input v-model="taskForm.notes" type="textarea" :rows="4" placeholder="备注 / 评论摘要 / 拆解点" />
        <div class="drawer-actions">
          <el-button type="primary" :loading="savingTask" @click="saveTask">保存</el-button>
          <el-button plain @click="taskDrawerVisible = false">关闭</el-button>
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
            还没有评论，补一句备注也行。
          </div>
        </div>
        <el-input v-model="commentForm.content" type="textarea" :rows="4" placeholder="记录进展、补充想法、留一点上下文" />
        <div class="drawer-actions">
          <el-button type="primary" :loading="savingComment" @click="submitComment">添加评论</el-button>
          <el-button @click="commentDrawerVisible = false">关闭</el-button>
        </div>
      </div>
    </el-drawer>

    <el-dialog v-model="aiDialogVisible" title="AI 整理事项" width="680px">
      <div class="drawer-stack">
        <div class="panel-heading-actions">
          <el-button plain @click="openVoice('ai')">
            <el-icon><Microphone /></el-icon>
            语音转文字
          </el-button>
          <span class="soft-note">{{ activeKind === 'work' ? '例如：今天要确认接口、补测试、晚上发周报。' : '例如：下班后买水果，周末整理衣柜，提醒妈妈复诊。' }}</span>
        </div>
        <el-input v-model="aiText" type="textarea" :rows="7" placeholder="把脑子里的事项直接说出来或写下来，AI 负责拆成结构化事项。" />
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
              <el-tag size="small" :type="priorityTagType(item.priority)">{{ priorityLabel(item.priority) }}</el-tag>
              <el-tag size="small">{{ item.planned_for }}</el-tag>
            </div>
          </article>
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
            <p>{{ statusLabel(task.status) }} · {{ task.planned_for }}</p>
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
import { MagicStick, Microphone, Refresh, Setting } from '@element-plus/icons-vue'

const API_BASE = '/api/planner'
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
  plannedFor: '',
  remindAt: '',
  priority: 'medium',
  notifyEmail: ''
})

const taskForm = reactive({
  id: '',
  kind: 'work',
  title: '',
  detail: '',
  notes: '',
  plannedFor: '',
  remindAt: '',
  priority: 'medium',
  status: 'open',
  notifyEmail: ''
})

const commentForm = reactive({
  content: ''
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
const adminDialogVisible = ref(false)
const adminDetailVisible = ref(false)
const parsingAI = ref(false)

const activeKind = ref('life')
const modeHint = ref('当前是下班或休息时段，默认进入生活模式')
const timelineGroups = ref([])
const comments = ref([])
const aiText = ref('')
const aiSuggestions = ref([])
const adminItems = ref([])
const adminPassword = ref('')
const adminKeyword = ref('')
const currentCommentTask = ref(null)
const recognitionState = ref('')
const adminDetail = reactive({
  profile: {
    id: '',
    name: '',
    notify_email: ''
  },
  tasks: [],
  extendDays: null
})

const timelineCounts = reactive({
  open: 0,
  in_progress: 0,
  done: 0,
  cancelled: 0,
  rolled_over: 0
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

function defaultModeByTime() {
  const now = new Date()
  const day = now.getDay()
  const hour = now.getHours()
  if (day >= 1 && day <= 5 && hour >= 9 && hour < 18) {
    return 'work'
  }
  return 'life'
}

function fillTodayDefaults() {
  quickForm.title = ''
  quickForm.detail = ''
  quickForm.plannedFor = new Date().toISOString().slice(0, 10)
  quickForm.remindAt = ''
  quickForm.priority = 'medium'
  quickForm.notifyEmail = profile.notify_email || ''
}

async function plannerFetch(url, options = {}) {
  const headers = { ...(options.headers || {}) }
  if (password.value) {
    headers['X-Password'] = password.value
  }
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...headers
    }
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
    await refreshTimeline()
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

function plannerModeCopy() {
  return activeKind.value === 'work' ? '当前聚焦工作事项。' : '当前聚焦生活事项。'
}

function switchKind(kind) {
  activeKind.value = kind
  localStorage.setItem('planner_active_kind', kind)
}

async function refreshTimeline() {
  if (!profileId.value) return
  timelineLoading.value = true
  try {
    const response = await plannerFetch(`${API_BASE}/profile/${profileId.value}/timeline?kind=${activeKind.value}`)
    const data = await response.json()
    timelineGroups.value = data.groups || []
    Object.assign(timelineCounts, {
      open: data.counts?.open || 0,
      in_progress: data.counts?.in_progress || 0,
      done: data.counts?.done || 0,
      cancelled: data.counts?.cancelled || 0,
      rolled_over: data.counts?.rolled_over || 0
    })
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    timelineLoading.value = false
  }
}

async function refreshAll() {
  await loadProfile()
  await refreshTimeline()
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
    await refreshTimeline()
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    savingQuick.value = false
  }
}

function openTask(task) {
  taskForm.id = task.id
  taskForm.kind = task.kind
  taskForm.title = task.title
  taskForm.detail = task.detail || ''
  taskForm.notes = task.notes || ''
  taskForm.priority = task.priority || 'medium'
  taskForm.status = task.status || 'open'
  taskForm.plannedFor = task.planned_for || ''
  taskForm.remindAt = task.remind_at ? task.remind_at.slice(0, 16) : ''
  taskForm.notifyEmail = task.notify_email || ''
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
        title: taskForm.title,
        detail: taskForm.detail,
        notes: taskForm.notes,
        planned_for: taskForm.plannedFor,
        remind_at: taskForm.remindAt,
        priority: taskForm.priority,
        status: taskForm.status,
        notify_email: taskForm.notifyEmail
      })
    })
    taskDrawerVisible.value = false
    ElMessage.success('事项已更新')
    await refreshTimeline()
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    savingTask.value = false
  }
}

async function setTaskStatus(task, status) {
  try {
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${task.id}`, {
      method: 'PUT',
      body: JSON.stringify({ status })
    })
    await refreshTimeline()
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function postponeTask(task) {
  const base = task.display_date || task.planned_for || new Date().toISOString().slice(0, 10)
  const date = new Date(`${base}T00:00:00`)
  date.setDate(date.getDate() + 1)
  const next = date.toISOString().slice(0, 10)
  try {
    await plannerFetch(`${API_BASE}/profile/${profileId.value}/tasks/${task.id}`, {
      method: 'PUT',
      body: JSON.stringify({ planned_for: next, status: 'open' })
    })
    ElMessage.success('已顺延一天')
    await refreshTimeline()
  } catch (error) {
    ElMessage.error(error.message)
  }
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
  timelineGroups.value = []
  comments.value = []
  aiSuggestions.value = []
  profile.name = ''
  profile.notify_email = ''
  profile.expires_at = ''
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
          title: item.title,
          detail: item.detail,
          notes: item.notes,
          priority: item.priority,
          status: item.status,
          planned_for: item.planned_for,
          remind_at: item.remind_at
        })
      })
    }
    aiDialogVisible.value = false
    aiSuggestions.value = []
    aiText.value = ''
    await refreshTimeline()
    ElMessage.success('AI 整理出的事项已写入')
  } catch (error) {
    ElMessage.error(error.message)
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
  if (!adminPassword.value || !adminDetail.profile.id) {
    return
  }
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
  recognitionState.value = 'listening'
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
  recognition.onend = () => {
    recognitionState.value = ''
  }
  recognition.onerror = () => {
    recognitionState.value = ''
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
    refreshTimeline()
  }
})

onMounted(async () => {
  restoreSession()
  fillTodayDefaults()
  if (profileId.value) {
    try {
      await loadProfile()
      await refreshTimeline()
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
  padding: 20px 14px 80px;
  color: #132238;
  transition: background 0.45s ease, color 0.45s ease;
}

.mode-work {
  --planner-accent: #2563eb;
  --planner-accent-soft: rgba(37, 99, 235, 0.14);
  --planner-card: rgba(250, 252, 255, 0.86);
  background:
    radial-gradient(circle at top left, rgba(59, 130, 246, 0.22), transparent 32%),
    linear-gradient(180deg, #eef5ff 0%, #f8fbff 48%, #f4f8ff 100%);
}

.mode-life {
  --planner-accent: #ef6a5b;
  --planner-accent-soft: rgba(239, 106, 91, 0.14);
  --planner-card: rgba(255, 252, 248, 0.88);
  background:
    radial-gradient(circle at top right, rgba(255, 179, 102, 0.2), transparent 28%),
    linear-gradient(180deg, #fff8f2 0%, #fffdf8 45%, #f8fff8 100%);
}

.planner-backdrop {
  position: absolute;
  inset: auto auto 0 0;
  border-radius: 999px;
  filter: blur(20px);
  pointer-events: none;
  opacity: 0.65;
}

.planner-backdrop-a {
  width: 220px;
  height: 220px;
  top: 32px;
  right: -30px;
  background: rgba(255, 255, 255, 0.5);
  animation: floatOrb 9s ease-in-out infinite;
}

.planner-backdrop-b {
  width: 170px;
  height: 170px;
  bottom: 40px;
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
.hero-card,
.quick-panel,
.timeline-panel,
.planner-topbar,
.mode-switcher,
.admin-card {
  border: 1px solid rgba(255, 255, 255, 0.6);
  background: var(--planner-card);
  backdrop-filter: blur(18px);
  box-shadow: 0 20px 40px rgba(15, 23, 42, 0.08);
}

.entry-hero,
.hero-card,
.quick-panel,
.timeline-panel,
.planner-topbar {
  border-radius: 26px;
  padding: 22px;
}

.entry-grid {
  display: grid;
  gap: 16px;
}

.entry-card {
  border-radius: 24px;
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
.panel-heading-actions {
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
.entry-tips span {
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

.entry-tips {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 16px;
}

.entry-tips span,
.soft-note {
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.6);
  font-size: 12px;
}

.mode-switcher {
  position: sticky;
  top: 12px;
  z-index: 12;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  padding: 8px;
  margin: 16px 0;
  border-radius: 24px;
}

.mode-tab {
  border: none;
  border-radius: 18px;
  background: transparent;
  padding: 14px 16px;
  text-align: left;
  cursor: pointer;
  transition: transform 0.25s ease, background 0.25s ease, box-shadow 0.25s ease;
}

.mode-tab.active {
  background: linear-gradient(135deg, var(--planner-accent), color-mix(in srgb, var(--planner-accent) 72%, white));
  color: #fff;
  transform: translateY(-2px);
  box-shadow: 0 14px 30px color-mix(in srgb, var(--planner-accent) 28%, transparent);
}

.mode-tab-sub {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  opacity: 0.8;
}

.hero-card {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  gap: 20px;
  align-items: center;
}

.hero-label {
  margin: 0 0 8px;
  font-size: 13px;
  font-weight: 700;
  color: var(--planner-accent);
}

.hero-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.hero-stat {
  padding: 14px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.62);
}

.hero-stat span {
  display: block;
  font-size: 12px;
  color: #66788d;
}

.hero-stat strong {
  display: block;
  margin-top: 8px;
  font-size: 28px;
}

.quick-panel,
.timeline-panel {
  margin-top: 16px;
}

.quick-grid,
.drawer-stack {
  display: grid;
  gap: 12px;
}

.quick-row {
  align-items: stretch;
}

.quick-row > * {
  flex: 1;
}

.quick-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.timeline-empty {
  padding: 28px 20px;
  text-align: center;
  color: #75859b;
  display: grid;
  gap: 6px;
}

.timeline-groups {
  display: grid;
  gap: 18px;
}

.timeline-group {
  display: grid;
  gap: 12px;
}

.timeline-group-head h4,
.task-card h5 {
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
  cursor: pointer;
  transition: transform 0.24s ease, box-shadow 0.24s ease;
}

.task-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 16px 28px rgba(15, 23, 42, 0.09);
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

.task-tags,
.task-meta,
.task-actions,
.admin-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.task-meta {
  margin-top: 12px;
  font-size: 12px;
  color: #66788d;
}

.task-actions {
  margin-top: 12px;
}

.rolled-tag {
  color: var(--planner-accent);
  font-weight: 600;
}

.comment-timeline,
.admin-list,
.ai-suggestions {
  display: grid;
  gap: 10px;
}

.comment-item,
.admin-card,
.ai-card {
  border-radius: 18px;
  padding: 14px;
  background: rgba(255, 255, 255, 0.72);
}

.comment-item p,
.ai-card p,
.admin-card p {
  margin: 8px 0 0;
  color: #66788d;
}

.mode-panel-enter-active,
.mode-panel-leave-active,
.task-fade-enter-active,
.task-fade-leave-active {
  transition: all 0.28s ease;
}

.mode-panel-enter-from,
.mode-panel-leave-to,
.task-fade-enter-from,
.task-fade-leave-to {
  opacity: 0;
  transform: translateY(10px);
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
    grid-template-columns: 1.1fr 1fr;
    align-items: start;
  }

  .entry-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .planner-shell {
    padding: 14px 10px 72px;
  }

  .hero-card,
  .quick-row,
  .planner-topbar,
  .timeline-header,
  .panel-heading {
    grid-template-columns: 1fr;
    display: grid;
  }

  .hero-stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .task-actions .el-button {
    flex: 1 1 calc(50% - 4px);
    min-width: 0;
  }

  .panel-heading-actions,
  .topbar-actions {
    justify-content: flex-start;
  }
}
</style>
