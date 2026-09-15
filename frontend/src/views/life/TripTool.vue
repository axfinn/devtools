<template>
  <div class="trip-shell">
    <header class="trip-header">
      <div>
        <p class="eyebrow">Travel Companion</p>
        <h1>旅游行程助手</h1>
        <p class="trip-tagline">
          预算 / 行程 / 费用 / 总结 四位一体。出行前先设置预算,出行中快速记一笔,
          回来后一键导出总结。所有数据走 <code>/api/trips</code>,复用后端主 SQLite + 共享 SMTP 通道。
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
            <div class="trip-list-meta" v-if="t.budget_total">
              💰 {{ formatYuanFromCents(t.budget_total) }}
              {{ t.budget_currency || 'CNY' }}
            </div>
          </div>
        </div>
      </el-card>
    </section>

    <section v-if="selectedTrip" class="trip-detail">
      <!-- 预算顶栏 -->
      <el-card shadow="never" class="budget-card">
        <div class="budget-row">
          <div class="budget-info">
            <div class="budget-label">整程预算</div>
            <div class="budget-value">
              {{ formatYuanFromCents(selectedTrip.budget_total || 0) }}
              <span class="budget-currency">{{ selectedTrip.budget_currency || 'CNY' }}</span>
            </div>
          </div>
          <div class="budget-info">
            <div class="budget-label">实际花费</div>
            <div class="budget-value">{{ formatYuanFromCents(budgetSpent) }}</div>
          </div>
          <div class="budget-info">
            <div class="budget-label">
              {{ budgetSpent > (selectedTrip.budget_total || 0) ? '已超支' : '剩余' }}
            </div>
            <div
              class="budget-value"
              :class="{
                'over-budget': budgetSpent > (selectedTrip.budget_total || 0),
                'budget-ok': budgetSpent <= (selectedTrip.budget_total || 0),
              }"
            >
              {{ formatYuanFromCents(budgetRemaining) }}
            </div>
          </div>
          <div class="budget-info">
            <div class="budget-label">已记账</div>
            <div class="budget-value">{{ expenses.length }} 笔</div>
          </div>
        </div>
        <el-progress
          v-if="selectedTrip.budget_total"
          :percentage="budgetPercent"
          :status="budgetSpent > (selectedTrip.budget_total || 0) ? 'exception' : ''"
          :stroke-width="14"
          class="budget-progress"
        />
        <div class="budget-actions">
          <el-button size="small" @click="openBudgetDialog">设置预算</el-button>
          <el-button size="small" type="primary" plain @click="quickExpenseDialog = true">
            + 快速记一笔
          </el-button>
        </div>
      </el-card>

      <el-card shadow="never" class="detail-card">
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
          <!-- 按天行程:支持出行中 inline 调整 -->
          <el-tab-pane label="按天行程" name="days">
            <div v-for="d in selectedTrip.days || []" :key="d.id" class="day-block">
              <div class="day-head">
                <span class="day-index">D{{ dayIndex(d) }}</span>
                <span class="day-date">{{ d.date }} · {{ weekdayCN(d.date) }}</span>
                <span class="day-dest" v-if="d.destination && d.destination.city">
                  · {{ d.destination.city }}
                </span>
                <div class="day-head-actions">
                  <el-button size="small" plain @click="openDayEditDialog(d)">改天</el-button>
                  <el-button
                    size="small"
                    type="primary"
                    plain
                    @click="openQuickExpenseForDay(d)"
                  >
                    + 记一笔
                  </el-button>
                </div>
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
                  <div class="activity-actions">
                    <el-button size="small" plain @click="openActivityEditDialog(a, d)">
                      调整
                    </el-button>
                    <el-button size="small" plain @click="openReminderDialog(a)">
                      {{ a.remind_before_minutes ? '改提醒' : '加提醒' }}
                    </el-button>
                  </div>
                </div>
              </div>
              <div v-else class="day-empty">本日暂无活动</div>
            </div>
          </el-tab-pane>

          <!-- 费用 Tab -->
          <el-tab-pane label="费用" name="expenses">
            <div class="expenses-head">
              <span>共 <strong>{{ expenses.length }}</strong> 笔,合计
                <strong>{{ formatYuanFromCents(budgetSpent) }}</strong>
              </span>
              <el-button size="small" type="primary" plain @click="quickExpenseDialog = true">
                + 新增
              </el-button>
            </div>
            <el-table :data="expenses" stripe style="width: 100%">
              <el-table-column label="日期" prop="date" width="120" />
              <el-table-column label="类别" width="100">
                <template #default="{ row }">
                  <span>{{ expenseIcon(row.category) }} {{ expenseKindCN(row.category) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="金额" width="140">
                <template #default="{ row }">
                  {{ formatYuanFromCents(row.amount_cents) }}
                  <span class="exp-currency">{{ row.currency || selectedTrip.budget_currency || 'CNY' }}</span>
                </template>
              </el-table-column>
              <el-table-column label="支付" prop="payment_method" width="90" />
              <el-table-column label="备注" prop="note" />
              <el-table-column label="操作" width="120">
                <template #default="{ row }">
                  <el-button size="small" plain @click="openExpenseEditDialog(row)">改</el-button>
                  <el-button
                    size="small"
                    type="danger"
                    plain
                    @click="deleteExpense(row)"
                  >删</el-button>
                </template>
              </el-table-column>
            </el-table>
            <div v-if="!expenses.length" class="empty-hint">
              还没有费用记录,出行中点击右上「+ 记一笔」随手记一笔。
            </div>
          </el-tab-pane>

          <!-- 总结 Tab -->
          <el-tab-pane label="总结" name="summary">
            <div v-if="summary" class="summary-view">
              <div class="summary-stats">
                <div class="summary-stat">
                  <div class="stat-label">预算</div>
                  <div class="stat-value">
                    {{ formatYuanFromCents(summary.trip.budget_total || 0) }}
                  </div>
                </div>
                <div class="summary-stat">
                  <div class="stat-label">实际花费</div>
                  <div class="stat-value">{{ formatYuanFromCents(summary.total_spent_cents) }}</div>
                </div>
                <div class="summary-stat" v-if="summary.trip.budget_total">
                  <div class="stat-label">
                    {{ summary.over_budget ? '超支' : '剩余' }}
                  </div>
                  <div
                    class="stat-value"
                    :class="summary.over_budget ? 'over-budget' : 'budget-ok'"
                  >
                    {{ formatYuanFromCents(Math.abs(summary.remaining_cents)) }}
                  </div>
                </div>
                <div class="summary-stat">
                  <div class="stat-label">笔数</div>
                  <div class="stat-value">{{ summary.expense_count }}</div>
                </div>
                <div class="summary-stat" v-if="summary.days_covered">
                  <div class="stat-label">日均</div>
                  <div class="stat-value">
                    {{ formatYuanFromCents(summary.average_per_day_cents) }}
                  </div>
                </div>
              </div>

              <div class="summary-block" v-if="summary.top_categories && summary.top_categories.length">
                <h3>按类别</h3>
                <div v-for="c in summary.top_categories" :key="c.category" class="cat-bar-row">
                  <span class="cat-name">{{ expenseKindCN(c.category) }} {{ expenseIcon(c.category) }}</span>
                  <div class="cat-bar">
                    <div
                      class="cat-bar-fill"
                      :style="{ width: c.percent + '%' }"
                    ></div>
                  </div>
                  <span class="cat-meta">
                    {{ formatYuanFromCents(c.cents) }} · {{ c.percent.toFixed(1) }}%
                  </span>
                </div>
              </div>

              <div class="summary-block" v-if="hasByDay(summary)">
                <h3>按天</h3>
                <div v-for="d in byDayEntries(summary)" :key="d.date" class="day-bar-row">
                  <span class="day-date">{{ d.date }}</span>
                  <div class="cat-bar">
                    <div
                      class="cat-bar-fill day-bar-fill"
                      :style="{ width: d.percent + '%' }"
                    ></div>
                  </div>
                  <span class="cat-meta">{{ formatYuanFromCents(d.cents) }}</span>
                </div>
              </div>

              <div class="summary-block" v-if="summary.other_currencies && summary.other_currencies.length">
                <el-alert
                  type="warning"
                  :title="`检测到其他币种支出:${summary.other_currencies.join(', ')},未折算入总花费`"
                  :closable="false"
                />
              </div>

              <div class="summary-actions">
                <el-button type="primary" plain @click="copySummaryMarkdown">复制 Markdown 总结</el-button>
                <el-button @click="loadSummaryMarkdown">查看完整 Markdown</el-button>
              </div>
              <pre v-if="summaryMarkdown" class="markdown-preview">{{ summaryMarkdown }}</pre>
            </div>
            <div v-else class="empty-hint">加载总结中...</div>
          </el-tab-pane>

          <!-- 整体 Markdown -->
          <el-tab-pane label="行程单" name="md">
            <pre class="markdown-preview">{{ markdownContent || '(加载中...)' }}</pre>
          </el-tab-pane>

          <!-- 行程设置 -->
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

    <!-- 提醒对话框 -->
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

    <!-- 预算对话框 -->
    <el-dialog v-model="budgetDialogVisible" title="设置整程预算" width="420px">
      <el-form label-width="100px">
        <el-form-item label="预算金额">
          <el-input-number
            v-model="budgetDraftYuan"
            :min="0"
            :step="100"
            :precision="2"
            style="width: 220px"
          />
          <span style="margin-left: 8px">{{ budgetDraftCurrency }}</span>
        </el-form-item>
        <el-form-item label="货币">
          <el-input v-model="budgetDraftCurrency" maxlength="3" placeholder="CNY" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="budgetDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingBudget" @click="saveBudget">保存</el-button>
      </template>
    </el-dialog>

    <!-- 快速记一笔对话框 -->
    <el-dialog
      v-model="quickExpenseDialog"
      :title="expenseDraft.id ? '编辑一笔费用' : '记一笔费用'"
      width="480px"
    >
      <el-form label-width="100px">
        <el-form-item label="日期">
          <el-input v-model="expenseDraft.date" placeholder="YYYY-MM-DD" />
        </el-form-item>
        <el-form-item label="类别">
          <el-select v-model="expenseDraft.category" style="width: 100%">
            <el-option
              v-for="k in ['transport', 'lodging', 'food', 'sight', 'shopping', 'misc']"
              :key="k"
              :label="expenseKindCN(k)"
              :value="k"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="金额">
          <el-input-number
            v-model="expenseDraftYuan"
            :min="0"
            :step="10"
            :precision="2"
            style="width: 220px"
          />
          <span style="margin-left: 8px">{{ expenseDraft.currency || selectedTrip?.budget_currency || 'CNY' }}</span>
        </el-form-item>
        <el-form-item label="支付方式">
          <el-select v-model="expenseDraft.payment_method" style="width: 100%" clearable>
            <el-option v-for="p in ['cash', 'card', 'alipay', 'wechat', 'other']" :key="p" :label="p" :value="p" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input
            v-model="expenseDraft.note"
            type="textarea"
            :rows="2"
            placeholder="比如:八合里牛肉火锅,3 人"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="quickExpenseDialog = false">取消</el-button>
        <el-button type="primary" :loading="savingExpense" @click="saveExpense">保存</el-button>
      </template>
    </el-dialog>

    <!-- 调整某项活动 -->
    <el-dialog v-model="activityEditDialogVisible" title="调整活动" width="480px">
      <el-form label-width="100px" v-if="activityEditDraft">
        <el-form-item label="标题">
          <el-input v-model="activityEditDraft.title" />
        </el-form-item>
        <el-form-item label="类别">
          <el-select v-model="activityEditDraft.kind" style="width: 100%">
            <el-option
              v-for="k in ['transit', 'sight', 'food', 'leisure', 'shopping', 'lodging']"
              :key="k"
              :label="kindCN(k)"
              :value="k"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="时间">
          <el-input v-model="activityEditDraft.start_time" placeholder="HH:MM 或 上午/下午/晚上" />
        </el-form-item>
        <el-form-item label="地点">
          <el-input v-model="activityEditDraft.location" />
        </el-form-item>
        <el-form-item label="时长(分)">
          <el-input-number v-model="activityEditDraft.duration_min" :min="0" :step="15" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="activityEditDraft.note" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="activityEditDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingActivity" @click="saveActivityEdit">保存</el-button>
      </template>
    </el-dialog>

    <!-- 调整某天 -->
    <el-dialog v-model="dayEditDialogVisible" title="调整某天" width="420px">
      <el-form label-width="100px" v-if="dayEditDraft">
        <el-form-item label="日期">
          <el-input v-model="dayEditDraft.date" placeholder="YYYY-MM-DD" />
        </el-form-item>
        <el-form-item label="城市">
          <el-input v-model="dayEditDraft.destination.city" />
        </el-form-item>
        <el-form-item label="区域">
          <el-input v-model="dayEditDraft.destination.region" />
        </el-form-item>
        <el-form-item label="国家">
          <el-input v-model="dayEditDraft.destination.country" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dayEditDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingDay" @click="saveDayEdit">保存</el-button>
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
const summaryMarkdown = ref('')
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

// 预算
const budgetDialogVisible = ref(false)
const budgetDraftYuan = ref(0)
const budgetDraftCurrency = ref('CNY')
const savingBudget = ref(false)

// 费用
const expenses = ref([])
const quickExpenseDialog = ref(false)
const expenseDraft = ref(emptyExpense())
const expenseDraftYuan = ref(0)
const savingExpense = ref(false)

// 总结
const summary = ref(null)

// 调整活动
const activityEditDialogVisible = ref(false)
const activityEditDraft = ref(null)
const savingActivity = ref(false)

// 调整天
const dayEditDialogVisible = ref(false)
const dayEditDraft = ref(null)
const savingDay = ref(false)

const API = `${API_BASE}/api/trips`

function emptyExpense() {
  return {
    id: '',
    trip_id: '',
    date: '',
    category: 'food',
    amount_cents: 0,
    currency: 'CNY',
    note: '',
    payment_method: '',
    activity_id: '',
  }
}

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
  summaryMarkdown.value = ''
  await loadTripDetail(id)
  await loadExpenses(id)
  await loadSummary(id)
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

async function loadExpenses(tripID) {
  try {
    const r = await fetch(`${API}/${tripID}/expenses`)
    if (!r.ok) throw new Error(`expenses: HTTP ${r.status}`)
    const j = await r.json()
    expenses.value = j.expenses || []
  } catch (e) {
    lastError.value = `加载费用失败: ${e.message}`
  }
}

async function loadSummary(tripID) {
  try {
    const r = await fetch(`${API}/${tripID}/summary`)
    if (!r.ok) throw new Error(`summary: HTTP ${r.status}`)
    summary.value = await r.json()
  } catch (e) {
    lastError.value = `加载总结失败: ${e.message}`
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

async function loadSummaryMarkdown() {
  if (!selectedId.value) return
  try {
    const r = await fetch(`${API}/${selectedId.value}/summary/markdown`)
    if (!r.ok) throw new Error(`summary-md: HTTP ${r.status}`)
    summaryMarkdown.value = await r.text()
  } catch (e) {
    lastError.value = `加载总结 Markdown 失败: ${e.message}`
  }
}

async function copySummaryMarkdown() {
  if (!selectedId.value) return
  try {
    const r = await fetch(`${API}/${selectedId.value}/summary/markdown`)
    if (!r.ok) throw new Error(`summary-md: HTTP ${r.status}`)
    const md = await r.text()
    summaryMarkdown.value = md
    try {
      await navigator.clipboard.writeText(md)
      ElMessage.success('总结 Markdown 已复制到剪贴板')
    } catch {
      ElMessage.warning('已加载到下方预览,请手动复制')
    }
  } catch (e) {
    lastError.value = `生成总结失败: ${e.message}`
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
    if (selectedId.value) {
      await loadTripDetail(selectedId.value)
      await loadExpenses(selectedId.value)
      await loadSummary(selectedId.value)
    }
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
    if (!confirm('当前后端未实现 PUT /api/trips/:id,确认要「删除并重建」该行程吗?这会丢掉所有 days + activities + expenses。')) {
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
  if (!confirm(`确认删除「${selectedTrip.value.name}」?该操作级联删除 days + activities + expenses。`)) return
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

// ---- 预算 ----

function openBudgetDialog() {
  if (!selectedTrip.value) return
  budgetDraftYuan.value = (selectedTrip.value.budget_total || 0) / 100
  budgetDraftCurrency.value = selectedTrip.value.budget_currency || 'CNY'
  budgetDialogVisible.value = true
}

async function saveBudget() {
  if (!selectedTrip.value) return
  savingBudget.value = true
  try {
    const cents = Math.round((budgetDraftYuan.value || 0) * 100)
    const r = await fetch(`${API}/${selectedTrip.value.id}/budget`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        budget_total_cents: cents,
        budget_currency: budgetDraftCurrency.value,
      }),
    })
    if (!r.ok) throw new Error(`budget: HTTP ${r.status}`)
    ElMessage.success('预算已更新')
    budgetDialogVisible.value = false
    await refreshAll()
  } catch (e) {
    lastError.value = `保存预算失败: ${e.message}`
  } finally {
    savingBudget.value = false
  }
}

// ---- 费用 ----

function openQuickExpenseForDay(day) {
  if (!selectedTrip.value || !day) return
  expenseDraft.value = emptyExpense()
  expenseDraft.value.trip_id = selectedTrip.value.id
  expenseDraft.value.date = day.date
  expenseDraft.value.currency = selectedTrip.value.budget_currency || 'CNY'
  expenseDraftYuan.value = 0
  quickExpenseDialog.value = true
}

function openExpenseEditDialog(row) {
  expenseDraft.value = { ...row }
  expenseDraftYuan.value = (row.amount_cents || 0) / 100
  quickExpenseDialog.value = true
}

async function saveExpense() {
  if (!selectedTrip.value) return
  savingExpense.value = true
  try {
    const cents = Math.round((expenseDraftYuan.value || 0) * 100)
    if (cents <= 0) {
      ElMessage.warning('金额必须大于 0')
      savingExpense.value = false
      return
    }
    const payload = {
      date: expenseDraft.value.date,
      category: expenseDraft.value.category,
      amount_cents: cents,
      currency: expenseDraft.value.currency,
      payment_method: expenseDraft.value.payment_method || '',
      note: expenseDraft.value.note || '',
    }
    let r
    if (expenseDraft.value.id) {
      r = await fetch(`${API}/expenses/${expenseDraft.value.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
    } else {
      r = await fetch(`${API}/${selectedTrip.value.id}/expenses`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
    }
    if (!r.ok) throw new Error(`expense: HTTP ${r.status}`)
    ElMessage.success(expenseDraft.value.id ? '已更新' : '已记账')
    quickExpenseDialog.value = false
    await loadExpenses(selectedTrip.value.id)
    await loadSummary(selectedTrip.value.id)
  } catch (e) {
    lastError.value = `保存费用失败: ${e.message}`
  } finally {
    savingExpense.value = false
  }
}

async function deleteExpense(row) {
  if (!confirm(`删除「${row.date} ${expenseKindCN(row.category)} ${formatYuanFromCents(row.amount_cents)}」?`)) return
  try {
    const r = await fetch(`${API}/expenses/${row.id}`, { method: 'DELETE' })
    if (!r.ok) throw new Error(`delete: HTTP ${r.status}`)
    ElMessage.success('已删除')
    await loadExpenses(selectedTrip.value.id)
    await loadSummary(selectedTrip.value.id)
  } catch (e) {
    lastError.value = `删除费用失败: ${e.message}`
  }
}

// ---- 调整活动 ----

function openActivityEditDialog(activity) {
  activityEditDraft.value = {
    id: activity.id,
    title: activity.title,
    kind: activity.kind,
    location: activity.location || '',
    start_time: activity.start_time || '',
    duration_min: activity.duration_min || 0,
    note: activity.note || '',
  }
  activityEditDialogVisible.value = true
}

async function saveActivityEdit() {
  if (!activityEditDraft.value) return
  savingActivity.value = true
  try {
    const payload = {
      title: activityEditDraft.value.title,
      kind: activityEditDraft.value.kind,
      location: activityEditDraft.value.location,
      start_time: activityEditDraft.value.start_time,
      duration_min: activityEditDraft.value.duration_min,
      note: activityEditDraft.value.note,
    }
    const r = await fetch(`${API}/activities/${activityEditDraft.value.id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (!r.ok) throw new Error(`patch: HTTP ${r.status}`)
    ElMessage.success('活动已更新')
    activityEditDialogVisible.value = false
    if (selectedId.value) await loadTripDetail(selectedId.value)
  } catch (e) {
    lastError.value = `调整活动失败: ${e.message}`
  } finally {
    savingActivity.value = false
  }
}

// ---- 调整天 ----

function openDayEditDialog(day) {
  dayEditDraft.value = {
    id: day.id,
    date: day.date,
    destination: {
      city: day.destination?.city || '',
      region: day.destination?.region || '',
      country: day.destination?.country || '',
    },
  }
  dayEditDialogVisible.value = true
}

async function saveDayEdit() {
  if (!dayEditDraft.value) return
  savingDay.value = true
  try {
    const payload = {
      date: dayEditDraft.value.date,
      city: dayEditDraft.value.destination.city,
      region: dayEditDraft.value.destination.region,
      country: dayEditDraft.value.destination.country,
    }
    const r = await fetch(`${API}/days/${dayEditDraft.value.id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (!r.ok) throw new Error(`patch: HTTP ${r.status}`)
    ElMessage.success('当天已更新')
    dayEditDialogVisible.value = false
    if (selectedId.value) await loadTripDetail(selectedId.value)
  } catch (e) {
    lastError.value = `调整当天失败: ${e.message}`
  } finally {
    savingDay.value = false
  }
}

// ---- 计算属性与格式化 ----

const budgetSpent = computed(() => {
  if (!selectedTrip.value) return 0
  // 用 summary.total_spent_cents 更稳;但 summary 是异步加载的,
  // 没加载好时用 expenses 本地累加兜底
  if (summary.value && summary.value.trip && summary.value.trip.id === selectedTrip.value.id) {
    return summary.value.total_spent_cents || 0
  }
  const cur = selectedTrip.value.budget_currency || 'CNY'
  return (expenses.value || [])
    .filter((e) => !e.currency || e.currency === cur)
    .reduce((s, e) => s + (e.amount_cents || 0), 0)
})

const budgetRemaining = computed(() => {
  if (!selectedTrip.value) return 0
  return (selectedTrip.value.budget_total || 0) - budgetSpent.value
})

const budgetPercent = computed(() => {
  const total = selectedTrip.value?.budget_total || 0
  if (!total) return 0
  return Math.min(100, Math.round((budgetSpent.value / total) * 100))
})

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

function expenseIcon(k) {
  return ({
    transport: '🚌', lodging: '🏨', food: '🍜',
    sight: '🎫', shopping: '🛍️', misc: '📦',
  })[k] || '•'
}

function expenseKindCN(k) {
  return ({
    transport: '交通', lodging: '住宿', food: '餐饮',
    sight: '门票', shopping: '购物', misc: '杂项',
  })[k] || k
}

function formatYuanFromCents(cents) {
  if (cents == null) return '¥0.00'
  const n = Number(cents) || 0
  const sign = n < 0 ? '-' : ''
  const abs = Math.abs(n)
  const whole = Math.floor(abs / 100)
  const frac = abs % 100
  return `${sign}¥${whole.toLocaleString()}.${String(frac).padStart(2, '0')}`
}

function splitCSV(s) {
  if (!s) return []
  return s.split(',').map((x) => x.trim()).filter(Boolean)
}

function hasByDay(summary) {
  return summary && summary.by_day_cents && Object.keys(summary.by_day_cents).length > 0
}

function byDayEntries(summary) {
  if (!summary || !summary.by_day_cents) return []
  const entries = Object.entries(summary.by_day_cents).map(([date, cents]) => ({
    date,
    cents,
    percent: summary.total_spent_cents
      ? Math.min(100, (cents / summary.total_spent_cents) * 100)
      : 0,
  }))
  entries.sort((a, b) => a.date.localeCompare(b.date))
  return entries
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

/* 预算顶栏 */
.budget-card {
  margin-bottom: 12px;
}
.budget-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 8px;
}
.budget-info {
  display: flex;
  flex-direction: column;
}
.budget-label {
  font-size: 12px;
  color: #6b7280;
  margin-bottom: 4px;
}
.budget-value {
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}
.budget-currency {
  font-size: 12px;
  color: #6b7280;
  margin-left: 4px;
  font-weight: normal;
}
.budget-progress {
  margin: 8px 0;
}
.budget-actions {
  display: flex;
  gap: 8px;
}
.over-budget {
  color: #dc2626;
}
.budget-ok {
  color: #059669;
}

.detail-card {
  margin-bottom: 16px;
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
  align-items: center;
  margin-bottom: 8px;
  flex-wrap: wrap;
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
.day-head-actions {
  margin-left: auto;
  display: flex;
  gap: 6px;
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
.activity-actions {
  display: flex;
  gap: 4px;
}
.day-empty {
  color: #9ca3af;
  font-size: 13px;
  font-style: italic;
}

/* 费用表 */
.expenses-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.exp-currency {
  font-size: 11px;
  color: #6b7280;
  margin-left: 4px;
}
.empty-hint {
  color: #9ca3af;
  font-size: 13px;
  text-align: center;
  padding: 32px 0;
}

/* 总结 */
.summary-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}
.summary-stat {
  background: #f9fafb;
  border-radius: 8px;
  padding: 12px;
}
.stat-label {
  font-size: 12px;
  color: #6b7280;
  margin-bottom: 4px;
}
.stat-value {
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}
.summary-block {
  margin: 16px 0;
}
.summary-block h3 {
  margin: 0 0 8px;
  font-size: 14px;
  color: #374151;
}
.cat-bar-row,
.day-bar-row {
  display: grid;
  grid-template-columns: 120px 1fr 160px;
  gap: 8px;
  align-items: center;
  margin-bottom: 4px;
  font-size: 13px;
}
.cat-bar {
  background: #e5e7eb;
  border-radius: 999px;
  height: 16px;
  overflow: hidden;
}
.cat-bar-fill {
  background: linear-gradient(90deg, #3b82f6, #2563eb);
  height: 100%;
  border-radius: 999px;
  transition: width 0.3s;
}
.day-bar-fill {
  background: linear-gradient(90deg, #10b981, #059669);
}
.cat-name {
  color: #374151;
}
.day-date {
  color: #374151;
  font-family: monospace;
}
.cat-meta {
  color: #6b7280;
  text-align: right;
  font-size: 12px;
}
.summary-actions {
  display: flex;
  gap: 8px;
  margin: 16px 0;
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