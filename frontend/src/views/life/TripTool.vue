<template>
  <div class="trip-shell">
    <header class="trip-header">
      <div>
        <p class="eyebrow">Travel Companion</p>
        <h1>旅游行程助手</h1>
        <p class="trip-tagline">
          落地一个真实行程(国庆 14 天),支持按天展开 / 配置活动 / 邮件提醒。
          所有数据走 <code>/api/trips</code>,复用后端主 SQLite + 共享 SMTP 通道。
        </p>
      </div>
      <div class="trip-header-actions">
        <el-button :loading="seeding" type="primary" plain @click="seedRealTrip">
          落地「国庆 14 日」fixture
        </el-button>
        <el-button :loading="refreshing" @click="refreshAll">刷新</el-button>
      </div>
    </header>

    <el-alert
      v-if="lastError"
      type="error"
      :title="lastError"
      show-icon
      :closable="false"
      style="margin-bottom: 16px"
    />

    <section class="trip-summary" v-if="trips.length">
      <el-card shadow="never">
        <template #header>
          <div class="trip-summary-head">
            <span><strong>{{ trips.length }}</strong> 个行程</span>
            <el-tag size="small" type="info">点击下方列表查看详情</el-tag>
          </div>
        </template>
        <div class="trip-list">
          <div
            v-for="t in trips"
            :key="t.id"
            class="trip-list-item"
            :class="{ active: t.id === selectedId }"
            @click="selectTrip(t.id)"
          >
            <div class="trip-list-row">
              <strong class="trip-list-name">{{ t.name }}</strong>
              <span class="trip-list-id">{{ t.id.slice(0, 8) }}</span>
            </div>
            <div class="trip-list-meta">
              <span>{{ t.start_date }} → {{ t.end_date }}</span>
              <span v-if="t.cover_cities && t.cover_cities.length">
                · {{ t.cover_cities.join(' / ') }}
              </span>
            </div>
          </div>
        </div>
      </el-card>
    </section>

    <section v-if="selectedTrip" class="trip-detail">
      <el-card shadow="never">
        <template #header>
          <div class="trip-detail-head">
            <div>
              <h2>{{ selectedTrip.name }}</h2>
              <p class="trip-detail-meta">
                {{ selectedTrip.start_date }} → {{ selectedTrip.end_date }}
                <span v-if="selectedTrip.description"> · {{ selectedTrip.description }}</span>
              </p>
            </div>
            <div class="trip-detail-actions">
              <el-button size="small" @click="loadMarkdown(selectedTrip.id)">
                复制 Markdown
              </el-button>
              <el-button size="small" type="warning" plain @click="processReminders">
                立即扫描提醒
              </el-button>
            </div>
          </div>
        </template>

        <el-tabs v-model="activeTab">
          <el-tab-pane label="按天行程" name="days">
            <div v-for="d in selectedTrip.days || []" :key="d.id" class="day-block">
              <div class="day-head">
                <span class="day-index">D{{ dayIndex(d) }}</span>
                <span class="day-date">{{ d.date }} · {{ weekdayCN(d.date) }}</span>
                <span class="day-dest" v-if="d.destination && d.destination.city">
                  · {{ d.destination.city }}
                </span>
              </div>
              <div v-if="d.activities && d.activities.length" class="activity-list">
                <div v-for="a in d.activities" :key="a.id" class="activity-row">
                  <span class="activity-time">{{ a.start_time || '—' }}</span>
                  <span class="activity-icon">{{ kindIcon(a.kind) }}</span>
                  <div class="activity-body">
                    <div class="activity-title">
                      <strong>[{{ kindCN(a.kind) }}]</strong> {{ a.title }}
                    </div>
                    <div class="activity-meta" v-if="a.location || a.note">
                      <span v-if="a.location">📍 {{ a.location }}</span>
                      <span v-if="a.note" class="activity-note">{{ a.note }}</span>
                    </div>
                    <div class="activity-reminder" v-if="a.remind_before_minutes">
                      ⏰ 提前 {{ a.remind_before_minutes }} 分钟提醒
                      <span v-if="a.remind_emails && a.remind_emails.length">
                        → {{ a.remind_emails.join(', ') }}
                      </span>
                    </div>
                  </div>
                  <el-button size="small" plain @click="openReminderDialog(a)">
                    {{ a.remind_before_minutes ? '改提醒' : '加提醒' }}
                  </el-button>
                </div>
              </div>
              <div v-else class="day-empty">本日暂无活动</div>
            </div>
          </el-tab-pane>

          <el-tab-pane label="整体 Markdown" name="md">
            <pre class="markdown-preview">{{ markdownContent || '(加载中...)' }}</pre>
          </el-tab-pane>

          <el-tab-pane label="行程设置" name="settings">
            <el-form label-width="100px" style="max-width: 600px">
              <el-form-item label="名称">
                <el-input v-model="settingsDraft.name" />
              </el-form-item>
              <el-form-item label="开始日期">
                <el-input v-model="settingsDraft.start_date" placeholder="YYYY-MM-DD" />
              </el-form-item>
              <el-form-item label="结束日期">
                <el-input v-model="settingsDraft.end_date" placeholder="YYYY-MM-DD" />
              </el-form-item>
              <el-form-item label="描述">
                <el-input v-model="settingsDraft.description" type="textarea" :rows="3" />
              </el-form-item>
              <el-form-item label="覆盖城市">
                <el-input v-model="settingsCities" placeholder="深圳, 香港, 澳门, ..." />
              </el-form-item>
              <el-form-item label="标签">
                <el-input v-model="settingsTags" placeholder="国庆, 亲子, ..." />
              </el-form-item>
              <el-form-item label="默认通知邮箱">
                <el-input v-model="settingsDraft.notify_email" placeholder="用于所有未单独指定的活动提醒" />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" :loading="saving" @click="saveSettings">保存</el-button>
                <el-button type="danger" plain :loading="deleting" @click="deleteTrip">删除行程</el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </el-card>
    </section>

    <section v-else class="trip-empty">
      <el-empty description="还没有行程,点击右上角『落地「国庆 14 日」fixture』试试" />
    </section>

    <el-dialog v-model="reminderDialogVisible" title="配置活动提醒" width="480px">
      <el-form label-width="120px" v-if="reminderDraft">
        <el-form-item label="活动">
          <div class="reminder-target">
            <strong>{{ reminderDraft.title }}</strong>
            <span v-if="reminderDraft.start_time"> · {{ reminderDraft.start_time }}</span>
          </div>
        </el-form-item>
        <el-form-item label="提前分钟">
          <el-input-number v-model="reminderDraft.remind_before_minutes" :min="0" :step="15" />
          <span class="reminder-hint">0 = 关闭提醒</span>
        </el-form-item>
        <el-form-item label="提醒邮箱">
          <el-input
            v-model="reminderEmailsDraft"
            type="textarea"
            :rows="2"
            placeholder="逗号分隔,留空走 Trip 默认"
          />
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="reminderResetDraft">
            重发已发过的提醒(清掉 sent_at)
          </el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reminderDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingReminder" @click="saveReminder">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { API_BASE } from '../../api.js'

const trips = ref([])
const selectedId = ref('')
const selectedTrip = ref(null)
const activeTab = ref('days')
const markdownContent = ref('')
const seeding = ref(false)
const refreshing = ref(false)
const saving = ref(false)
const deleting = ref(false)
const lastError = ref('')

const settingsDraft = ref({})
const settingsCities = ref('')
const settingsTags = ref('')

const reminderDialogVisible = ref(false)
const reminderDraft = ref(null)
const reminderEmailsDraft = ref('')
const reminderResetDraft = ref(false)
const savingReminder = ref(false)

const API = `${API_BASE}/api/trips`

async function loadTrips() {
  try {
    const r = await fetch(API)
    if (!r.ok) throw new Error(`list: HTTP ${r.status}`)
    const j = await r.json()
    trips.value = j.trips || []
    if (!selectedId.value && trips.value.length) {
      selectTrip(trips.value[0].id)
    }
    lastError.value = ''
  } catch (e) {
    lastError.value = `加载行程列表失败: ${e.message}`
  }
}

async function selectTrip(id) {
  selectedId.value = id
  activeTab.value = 'days'
  markdownContent.value = ''
  await loadTripDetail(id)
}

async function loadTripDetail(id) {
  try {
    const r = await fetch(`${API}/${id}`)
    if (!r.ok) throw new Error(`get: HTTP ${r.status}`)
    selectedTrip.value = await r.json()
    syncSettingsDraft()
  } catch (e) {
    lastError.value = `加载行程详情失败: ${e.message}`
  }
}

function syncSettingsDraft() {
  const t = selectedTrip.value
  if (!t) return
  settingsDraft.value = {
    name: t.name,
    start_date: t.start_date,
    end_date: t.end_date,
    description: t.description || '',
    notify_email: t.notify_email || '',
  }
  settingsCities.value = (t.cover_cities || []).join(', ')
  settingsTags.value = (t.tags || []).join(', ')
}

async function loadMarkdown(id) {
  try {
    const r = await fetch(`${API}/${id}/markdown`)
    if (!r.ok) throw new Error(`md: HTTP ${r.status}`)
    markdownContent.value = await r.text()
    activeTab.value = 'md'
    try {
      await navigator.clipboard.writeText(markdownContent.value)
      ElMessage.success('Markdown 已复制到剪贴板')
    } catch {
      // 剪贴板权限被拒时静默降级;用户仍可在 Tab 里手动复制
    }
  } catch (e) {
    lastError.value = `渲染 Markdown 失败: ${e.message}`
  }
}

async function seedRealTrip() {
  seeding.value = true
  try {
    const r = await fetch(`${API}/seed?force=true`, { method: 'POST' })
    if (!r.ok) throw new Error(`seed: HTTP ${r.status}`)
    const j = await r.json()
    ElMessage.success(j.created ? '已落地「国庆 14 日」fixture' : '已存在,跳过创建')
    await loadTrips()
    if (j.trip_id) selectTrip(j.trip_id)
  } catch (e) {
    lastError.value = `seed 失败: ${e.message}`
  } finally {
    seeding.value = false
  }
}

async function refreshAll() {
  refreshing.value = true
  try {
    await loadTrips()
    if (selectedId.value) await loadTripDetail(selectedId.value)
  } finally {
    refreshing.value = false
  }
}

async function saveSettings() {
  if (!selectedTrip.value) return
  saving.value = true
  try {
    const payload = {
      ...settingsDraft.value,
      cover_cities: splitCSV(settingsCities.value),
      tags: splitCSV(settingsTags.value),
    }
    // 后端目前没暴露 PUT trip;只支持 POST 创建。
    // 这里走「删了重建」路线 — 只对 seed 出来的 demo 安全,真实场景应加 PUT。
    if (!confirm('当前后端未实现 PUT /api/trips/:id,确认要「删除并重建」该行程吗?这会丢掉所有 days + activities。')) {
      saving.value = false
      return
    }
    const tripID = selectedTrip.value.id
    const del = await fetch(`${API}/${tripID}`, { method: 'DELETE' })
    if (!del.ok) throw new Error(`delete: HTTP ${del.status}`)
    const create = await fetch(API, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (!create.ok) throw new Error(`create: HTTP ${create.status}`)
    const j = await create.json()
    ElMessage.success('行程已重建')
    await loadTrips()
    if (j.trip && j.trip.id) selectTrip(j.trip.id)
  } catch (e) {
    lastError.value = `保存失败: ${e.message}`
  } finally {
    saving.value = false
  }
}

async function deleteTrip() {
  if (!selectedTrip.value) return
  if (!confirm(`确认删除「${selectedTrip.value.name}」?该操作级联删除 days + activities。`)) return
  deleting.value = true
  try {
    const r = await fetch(`${API}/${selectedTrip.value.id}`, { method: 'DELETE' })
    if (!r.ok) throw new Error(`delete: HTTP ${r.status}`)
    ElMessage.success('已删除')
    selectedTrip.value = null
    selectedId.value = ''
    await loadTrips()
  } catch (e) {
    lastError.value = `删除失败: ${e.message}`
  } finally {
    deleting.value = false
  }
}

async function processReminders() {
  try {
    const r = await fetch(`${API}/process-now`, { method: 'POST' })
    if (!r.ok) throw new Error(`process: HTTP ${r.status}`)
    const j = await r.json()
    ElMessage.success(`已扫描:发送 ${j.sent} 条 / 跳过 ${j.skipped} 条`)
    if (selectedId.value) await loadTripDetail(selectedId.value)
  } catch (e) {
    lastError.value = `扫描提醒失败: ${e.message}`
  }
}

function openReminderDialog(activity) {
  reminderDraft.value = {
    id: activity.id,
    title: activity.title,
    start_time: activity.start_time,
    remind_before_minutes: activity.remind_before_minutes || 0,
  }
  reminderEmailsDraft.value = (activity.remind_emails || []).join(', ')
  reminderResetDraft.value = false
  reminderDialogVisible.value = true
}

async function saveReminder() {
  if (!reminderDraft.value) return
  savingReminder.value = true
  try {
    const payload = {
      remind_before_minutes: Number(reminderDraft.value.remind_before_minutes) || 0,
      remind_emails: splitCSV(reminderEmailsDraft.value),
      reset: reminderResetDraft.value,
    }
    const r = await fetch(`${API}/activities/${reminderDraft.value.id}/reminder`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (!r.ok) throw new Error(`reminder: HTTP ${r.status}`)
    ElMessage.success(reminderResetDraft.value ? '提醒已更新,等待重发' : '提醒已更新')
    reminderDialogVisible.value = false
    if (selectedId.value) await loadTripDetail(selectedId.value)
  } catch (e) {
    lastError.value = `保存提醒失败: ${e.message}`
  } finally {
    savingReminder.value = false
  }
}

function dayIndex(d) {
  const days = selectedTrip.value?.days || []
  return days.indexOf(d) + 1
}

function weekdayCN(date) {
  if (!date) return ''
  const d = new Date(date)
  return ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][d.getDay()]
}

function kindIcon(k) {
  return ({
    transit: '🚌', sight: '🏛️', food: '🍜',
    lodging: '🏨', shopping: '🛍️', leisure: '🎯',
  })[k] || '•'
}

function kindCN(k) {
  return ({
    transit: '交通', sight: '景点', food: '餐饮',
    lodging: '住宿', shopping: '购物', leisure: '自由',
  })[k] || k
}

function splitCSV(s) {
  if (!s) return []
  return s.split(',').map((x) => x.trim()).filter(Boolean)
}

onMounted(() => {
  loadTrips()
})

watch(selectedTrip, () => {
  if (selectedTrip.value && activeTab.value === 'md' && !markdownContent.value) {
    loadMarkdown(selectedTrip.value.id)
  }
})
</script>

<style scoped>
.trip-shell {
  max-width: 1100px;
  margin: 0 auto;
  padding: 24px;
}
.trip-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}
.eyebrow {
  text-transform: uppercase;
  font-size: 11px;
  letter-spacing: 0.16em;
  color: #6b7280;
  margin: 0 0 4px;
}
.trip-header h1 {
  font-size: 28px;
  margin: 0;
}
.trip-tagline {
  color: #4b5563;
  margin: 8px 0 0;
  max-width: 640px;
}
.trip-tagline code {
  background: #f3f4f6;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 13px;
}
.trip-summary-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.trip-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 10px;
}
.trip-list-item {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 10px 12px;
  cursor: pointer;
  transition: all 0.15s;
}
.trip-list-item:hover {
  border-color: #93c5fd;
  background: #eff6ff;
}
.trip-list-item.active {
  border-color: #2563eb;
  background: #dbeafe;
}
.trip-list-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
}
.trip-list-name {
  font-size: 14px;
}
.trip-list-id {
  font-family: monospace;
  font-size: 11px;
  color: #6b7280;
}
.trip-list-meta {
  font-size: 12px;
  color: #6b7280;
  margin-top: 4px;
}
.trip-detail-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  flex-wrap: wrap;
}
.trip-detail-head h2 {
  margin: 0;
  font-size: 20px;
}
.trip-detail-meta {
  margin: 4px 0 0;
  color: #6b7280;
  font-size: 13px;
}
.day-block {
  border-top: 1px solid #f3f4f6;
  padding: 14px 0;
}
.day-block:first-child {
  border-top: none;
}
.day-head {
  display: flex;
  gap: 8px;
  align-items: baseline;
  margin-bottom: 8px;
}
.day-index {
  background: #2563eb;
  color: white;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: bold;
}
.day-date {
  font-weight: 600;
}
.day-dest {
  color: #6b7280;
}
.activity-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.activity-row {
  display: grid;
  grid-template-columns: 80px 30px 1fr auto;
  gap: 8px;
  align-items: center;
  padding: 8px;
  border-radius: 6px;
  background: #fafafa;
}
.activity-time {
  font-family: monospace;
  color: #4b5563;
  font-size: 13px;
}
.activity-icon {
  font-size: 18px;
}
.activity-title {
  font-size: 14px;
}
.activity-meta {
  font-size: 12px;
  color: #6b7280;
  margin-top: 2px;
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.activity-note {
  font-style: italic;
}
.activity-reminder {
  font-size: 12px;
  color: #2563eb;
  margin-top: 4px;
}
.day-empty {
  color: #9ca3af;
  font-size: 13px;
  font-style: italic;
}
.markdown-preview {
  background: #f9fafb;
  padding: 16px;
  border-radius: 8px;
  white-space: pre-wrap;
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-size: 13px;
  max-height: 600px;
  overflow: auto;
}
.reminder-target {
  font-size: 14px;
}
.reminder-hint {
  margin-left: 8px;
  font-size: 12px;
  color: #6b7280;
}
.trip-empty {
  margin-top: 32px;
}
</style>