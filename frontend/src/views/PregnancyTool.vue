<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>孕期管理</h2>
      <div class="actions">
        <template v-if="profileId">
          <el-tag :type="saveStatus === 'saving' ? 'warning' : 'success'" size="small">
            {{ saveStatus === 'saving' ? '保存中...' : '已保存' }}
          </el-tag>
          <el-button type="warning" @click="showExtendDialog = true">
            <el-icon><Timer /></el-icon>
            延期
          </el-button>
          <el-button type="danger" @click="confirmDelete">
            <el-icon><Delete /></el-icon>
            删除档案
          </el-button>
        </template>
        <el-button v-if="hasLocalProfiles" type="primary" @click="showMyProfiles = true">
          <el-icon><Folder /></el-icon>
          我的档案
        </el-button>
      </div>
    </div>

    <!-- No profile loaded: Create or Load -->
    <div v-if="!profileId" class="welcome-section">
      <div class="welcome-card">
        <h3>创建孕期档案</h3>
        <p>输入预产期和密码，创建您的专属孕期管理档案。</p>
        <el-form :model="createForm" label-width="80px" style="max-width: 400px; margin: 20px auto;">
          <el-form-item label="预产期">
            <el-date-picker v-model="createForm.edd" type="date" placeholder="选择预产期"
              format="YYYY-MM-DD" value-format="YYYY-MM-DD" style="width: 100%;" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="createForm.password" type="password" placeholder="至少4个字符"
              show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="createProfile" :loading="creating">创建档案</el-button>
          </el-form-item>
        </el-form>
      </div>

      <div class="welcome-card" style="margin-top: 20px;">
        <h3>加载已有档案</h3>
        <p>输入创建时设置的密码即可加载档案。</p>
        <el-form :model="loadForm" label-width="80px" style="max-width: 400px; margin: 20px auto;">
          <el-form-item label="密码">
            <el-input v-model="loadForm.password" type="password" placeholder="输入密码"
              show-password @keyup.enter="loadProfile" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="loadProfile" :loading="loading">加载档案</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- Profile loaded: Main content -->
    <template v-else>
      <!-- Overview banner -->
      <div class="overview-banner">
        <div class="pregnancy-info">
          <div class="week-display">
            <span class="week-number">{{ currentWeek }}</span>
            <span class="week-label">周 {{ currentDay }} 天</span>
          </div>
          <div class="info-details">
            <div class="info-item">
              <span class="label">预产期</span>
              <span class="value">{{ profileEDD }}</span>
            </div>
            <div class="info-item">
              <span class="label">剩余</span>
              <span class="value">{{ daysRemaining }} 天</span>
            </div>
            <div class="info-item">
              <span class="label">宝宝大小</span>
              <span class="value">{{ babySize }}</span>
            </div>
          </div>
        </div>
        <div class="progress-bar-container">
          <el-progress :percentage="pregnancyProgress" :stroke-width="12"
            :color="progressColor" :format="() => pregnancyProgress + '%'" />
        </div>
      </div>

      <el-tabs v-model="activeTab" type="border-card">
        <!-- Hospital Bag Tab -->
        <el-tab-pane label="待产包" name="hospitalBag">
          <div class="checklist-section">
            <div class="section-header">
              <h3>待产包清单</h3>
              <el-progress :percentage="hospitalBagProgress" :stroke-width="8"
                style="width: 200px;" />
            </div>
            <el-collapse v-model="hospitalBagCollapse">
              <el-collapse-item v-for="(items, category) in profileData.hospital_bag"
                :key="category" :name="category">
                <template #title>
                  <div class="collapse-title">
                    <span>{{ hospitalBagLabels[category] }}</span>
                    <el-tag size="small" type="info">
                      {{ getCheckedCount(items) }}/{{ items.length }}
                    </el-tag>
                  </div>
                </template>
                <div class="checklist-items">
                  <div v-for="(item, idx) in items" :key="idx" class="checklist-item">
                    <el-checkbox v-model="item.checked" @change="scheduleAutoSave">
                      {{ item.name }}
                    </el-checkbox>
                    <el-button link type="danger" size="small" @click="removeItem('hospital_bag', category, idx)">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                  <div class="add-item">
                    <el-input v-model="newItems.hospital_bag[category]"
                      :placeholder="'添加' + hospitalBagLabels[category] + '...'"
                      size="small" @keyup.enter="addItem('hospital_bag', category)">
                      <template #append>
                        <el-button @click="addItem('hospital_bag', category)">
                          <el-icon><Plus /></el-icon>
                        </el-button>
                      </template>
                    </el-input>
                  </div>
                </div>
              </el-collapse-item>
            </el-collapse>
          </div>
        </el-tab-pane>

        <!-- Baby Essentials Tab -->
        <el-tab-pane label="宝宝用品" name="babyEssentials">
          <div class="checklist-section">
            <div class="section-header">
              <h3>宝宝用品清单</h3>
              <el-progress :percentage="babyEssentialsProgress" :stroke-width="8"
                style="width: 200px;" />
            </div>
            <el-collapse v-model="babyEssentialsCollapse">
              <el-collapse-item v-for="(items, category) in profileData.baby_essentials"
                :key="category" :name="category">
                <template #title>
                  <div class="collapse-title">
                    <span>{{ babyEssentialsLabels[category] }}</span>
                    <el-tag size="small" type="info">
                      {{ getCheckedCount(items) }}/{{ items.length }}
                    </el-tag>
                  </div>
                </template>
                <div class="checklist-items">
                  <div v-for="(item, idx) in items" :key="idx" class="checklist-item">
                    <el-checkbox v-model="item.checked" @change="scheduleAutoSave">
                      {{ item.name }}
                    </el-checkbox>
                    <el-button link type="danger" size="small" @click="removeItem('baby_essentials', category, idx)">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                  <div class="add-item">
                    <el-input v-model="newItems.baby_essentials[category]"
                      :placeholder="'添加' + babyEssentialsLabels[category] + '...'"
                      size="small" @keyup.enter="addItem('baby_essentials', category)">
                      <template #append>
                        <el-button @click="addItem('baby_essentials', category)">
                          <el-icon><Plus /></el-icon>
                        </el-button>
                      </template>
                    </el-input>
                  </div>
                </div>
              </el-collapse-item>
            </el-collapse>
          </div>
        </el-tab-pane>

        <!-- Prenatal Checks Tab -->
        <el-tab-pane label="产检" name="prenatalChecks">
          <div class="prenatal-section">
            <h3>产检时间表</h3>
            <el-timeline>
              <el-timeline-item v-for="(check, idx) in profileData.prenatal_checks"
                :key="idx"
                :type="check.done ? 'success' : (isCurrentCheck(check.week) ? 'primary' : 'info')"
                :hollow="!check.done"
                :timestamp="'孕' + check.week + '周'"
                placement="top">
                <div class="check-card" :class="{ done: check.done, current: isCurrentCheck(check.week) }">
                  <div class="check-header">
                    <el-checkbox v-model="check.done" @change="scheduleAutoSave">
                      <strong>{{ check.name }}</strong>
                    </el-checkbox>
                  </div>
                  <p class="check-items">{{ check.items }}</p>
                  <el-input v-model="check.notes" type="textarea" :rows="1"
                    placeholder="添加备注..." size="small" @change="scheduleAutoSave" />
                </div>
              </el-timeline-item>
            </el-timeline>
          </div>
        </el-tab-pane>

        <!-- Weight Tab -->
        <el-tab-pane label="体重" name="weight">
          <div class="weight-section">
            <h3>体重记录</h3>
            <div class="weight-input">
              <el-form inline>
                <el-form-item label="日期">
                  <el-date-picker v-model="newWeight.date" type="date"
                    format="YYYY-MM-DD" value-format="YYYY-MM-DD" size="default" />
                </el-form-item>
                <el-form-item label="体重(kg)">
                  <el-input-number v-model="newWeight.value" :precision="1" :step="0.1"
                    :min="30" :max="200" size="default" />
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" @click="addWeight">记录</el-button>
                </el-form-item>
              </el-form>
            </div>

            <!-- Weight chart (inline SVG) -->
            <div v-if="profileData.weight_records && profileData.weight_records.length > 1" class="weight-chart">
              <svg :viewBox="'0 0 ' + chartWidth + ' ' + chartHeight" class="chart-svg">
                <line v-for="(line, i) in chartGridLines" :key="'grid-' + i"
                  :x1="line.x1" :y1="line.y1" :x2="line.x2" :y2="line.y2"
                  stroke="var(--border-light)" stroke-dasharray="4" />
                <text v-for="(label, i) in chartYLabels" :key="'ylabel-' + i"
                  :x="label.x" :y="label.y" fill="var(--text-secondary)" font-size="11"
                  text-anchor="end">{{ label.text }}</text>
                <polyline :points="chartPoints" fill="none"
                  stroke="var(--color-primary)" stroke-width="2" />
                <circle v-for="(pt, i) in chartDots" :key="'dot-' + i"
                  :cx="pt.x" :cy="pt.y" r="4" fill="var(--color-primary)" />
              </svg>
            </div>

            <!-- Weight table -->
            <el-table v-if="profileData.weight_records && profileData.weight_records.length"
              :data="sortedWeightRecords" stripe size="small" style="margin-top: 15px;">
              <el-table-column prop="date" label="日期" width="150" />
              <el-table-column prop="value" label="体重(kg)" width="120" />
              <el-table-column label="变化" width="120">
                <template #default="{ $index }">
                  <span v-if="$index < sortedWeightRecords.length - 1"
                    :style="{ color: getWeightChangeColor($index) }">
                    {{ getWeightChange($index) }}
                  </span>
                  <span v-else>-</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="80">
                <template #default="{ $index }">
                  <el-button link type="danger" size="small" @click="removeWeight($index)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- Fetal Movement Tab -->
        <el-tab-pane label="胎动" name="fetalMovement">
          <div class="fetal-section">
            <h3>胎动记录</h3>
            <div class="counter-area">
              <div class="counter-display">
                <span class="counter-number">{{ fetalCount }}</span>
                <span class="counter-label">次</span>
              </div>
              <div class="counter-timer" v-if="fetalTimerRunning">
                <span>计时: {{ fetalTimerDisplay }}</span>
              </div>
              <div class="counter-actions">
                <el-button type="primary" size="large" circle class="kick-button"
                  @click="recordKick" :disabled="!fetalTimerRunning">
                  <el-icon :size="32"><Plus /></el-icon>
                </el-button>
              </div>
              <div class="counter-controls">
                <el-button v-if="!fetalTimerRunning" type="success" @click="startFetalTimer">
                  开始计数
                </el-button>
                <el-button v-else type="danger" @click="stopFetalTimer">
                  结束并保存
                </el-button>
                <el-button v-if="fetalTimerRunning" @click="resetFetalCounter">
                  重置
                </el-button>
              </div>
            </div>

            <el-table v-if="profileData.fetal_movements && profileData.fetal_movements.length"
              :data="profileData.fetal_movements" stripe size="small" style="margin-top: 20px;">
              <el-table-column prop="date" label="日期" width="150" />
              <el-table-column prop="start_time" label="开始时间" width="120" />
              <el-table-column prop="duration" label="时长(分钟)" width="120" />
              <el-table-column prop="count" label="次数" width="100" />
              <el-table-column label="操作" width="80">
                <template #default="{ $index }">
                  <el-button link type="danger" size="small"
                    @click="removeFetalRecord($index)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- Knowledge Tab -->
        <el-tab-pane label="知识库" name="knowledge">
          <div class="knowledge-section">
            <h3>孕期知识</h3>
            <el-segmented v-model="knowledgeCategory" :options="knowledgeCategoryOptions"
              style="margin-bottom: 15px;" />

            <el-collapse v-model="knowledgeCollapse">
              <el-collapse-item v-for="(info, idx) in filteredKnowledgeData" :key="info.id || idx" :name="info.id || idx"
                :class="{ 'current-week-knowledge': info.weeks && isInWeekRange(info.weeks) }">
                <template #title>
                  <div class="knowledge-title">
                    <el-tag v-if="info.weeks && isInWeekRange(info.weeks)" type="success" size="small">当前</el-tag>
                    <el-tag v-if="info.category === 'guide'" type="warning" size="small">专题</el-tag>
                    <span>{{ info.title }}</span>
                  </div>
                </template>
                <div class="knowledge-content">
                  <div v-if="info.development" class="knowledge-item">
                    <h4>发育特征</h4>
                    <p>{{ info.development }}</p>
                  </div>
                  <div v-if="info.size" class="knowledge-item">
                    <h4>胎儿大小</h4>
                    <p>{{ info.size }}</p>
                  </div>
                  <div v-if="info.symptoms" class="knowledge-item">
                    <h4>常见症状</h4>
                    <p>{{ info.symptoms }}</p>
                  </div>
                  <div v-if="info.checkups" class="knowledge-item">
                    <h4>本阶段产检</h4>
                    <p>{{ info.checkups }}</p>
                  </div>
                  <div v-if="info.tips" class="knowledge-item">
                    <h4>注意事项</h4>
                    <p>{{ info.tips }}</p>
                  </div>
                  <div v-if="info.danger_signs" class="knowledge-item knowledge-danger">
                    <h4>危险信号</h4>
                    <p>{{ info.danger_signs }}</p>
                  </div>
                  <div v-if="info.diet" class="knowledge-item">
                    <h4>饮食建议</h4>
                    <p>{{ info.diet }}</p>
                  </div>
                  <div v-if="info.taboo" class="knowledge-item knowledge-warning">
                    <h4>饮食禁忌</h4>
                    <p>{{ info.taboo }}</p>
                  </div>
                  <div v-if="info.exercise" class="knowledge-item">
                    <h4>运动建议</h4>
                    <p>{{ info.exercise }}</p>
                  </div>
                  <div v-if="info.emotional" class="knowledge-item">
                    <h4>情绪心理</h4>
                    <p>{{ info.emotional }}</p>
                  </div>
                  <div v-if="info.partner" class="knowledge-item">
                    <h4>准爸爸须知</h4>
                    <p>{{ info.partner }}</p>
                  </div>
                  <div v-if="info.sections" class="knowledge-sections">
                    <div v-for="(sec, si) in info.sections" :key="si" class="knowledge-item">
                      <h4>{{ sec.title }}</h4>
                      <p>{{ sec.content }}</p>
                    </div>
                  </div>
                  <div v-if="info.blogUrl" class="knowledge-blog-link">
                    <a :href="info.blogUrl" target="_blank" rel="noopener">
                      <el-button type="primary" link size="small">
                        查看完整文章 &rarr;
                      </el-button>
                    </a>
                  </div>
                </div>
              </el-collapse-item>
            </el-collapse>
          </div>
        </el-tab-pane>
      </el-tabs>
    </template>

    <!-- My Profiles Dialog -->
    <el-dialog v-model="showMyProfiles" title="我的档案" width="500px">
      <div v-for="(profile, id) in localProfiles" :key="id" class="profile-item">
        <span>{{ id }}</span>
        <div>
          <el-button size="small" type="primary" @click="loadByCreatorKey(id, profile)">加载</el-button>
          <el-button size="small" type="danger" @click="removeLocalProfile(id)">移除</el-button>
        </div>
      </div>
      <el-empty v-if="!hasLocalProfiles" description="暂无本地档案" />
    </el-dialog>

    <!-- Extend Dialog -->
    <el-dialog v-model="showExtendDialog" title="延期档案" width="400px">
      <el-form label-width="80px">
        <el-form-item label="延期天数">
          <el-input-number v-model="extendDays" :min="30" :max="730" :step="30" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showExtendDialog = false">取消</el-button>
        <el-button type="primary" @click="extendProfile" :loading="extending">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Plus, Folder, Timer } from '@element-plus/icons-vue'

const API_BASE = import.meta.env.DEV ? 'http://localhost:8080' : ''

// State
const profileId = ref('')
const profileEDD = ref('')
const profileData = reactive({
  hospital_bag: { mom: [], baby: [], documents: [], other: [] },
  baby_essentials: { feeding: [], diaper: [], clothing: [], bathing: [], bedding: [], outdoor: [] },
  prenatal_checks: [],
  weight_records: [],
  fetal_movements: []
})
const creatorKey = ref('')
const saveStatus = ref('saved')
const activeTab = ref('hospitalBag')
const creating = ref(false)
const loading = ref(false)
const extending = ref(false)
const showMyProfiles = ref(false)
const showExtendDialog = ref(false)
const extendDays = ref(365)

const createForm = reactive({ edd: '', password: '' })
const loadForm = reactive({ password: '' })

// New item inputs for checklists
const newItems = reactive({
  hospital_bag: { mom: '', baby: '', documents: '', other: '' },
  baby_essentials: { feeding: '', diaper: '', clothing: '', bathing: '', bedding: '', outdoor: '' }
})

// Weight
const newWeight = reactive({
  date: new Date().toISOString().slice(0, 10),
  value: 60.0
})

// Fetal movement
const fetalCount = ref(0)
const fetalTimerRunning = ref(false)
const fetalTimerStart = ref(null)
const fetalTimerElapsed = ref(0)
let fetalInterval = null

// Collapse states
const hospitalBagCollapse = ref(['mom', 'baby', 'documents', 'other'])
const babyEssentialsCollapse = ref(['feeding', 'diaper', 'clothing', 'bathing', 'bedding', 'outdoor'])
const knowledgeCollapse = ref([])

// Labels
const hospitalBagLabels = {
  mom: '妈妈用品',
  baby: '宝宝用品',
  documents: '证件资料',
  other: '其他'
}
const babyEssentialsLabels = {
  feeding: '喂养用品',
  diaper: '尿布用品',
  clothing: '衣物',
  bathing: '洗护用品',
  bedding: '寝具',
  outdoor: '出行用品'
}

// Auto-save debounce
let autoSaveTimer = null
function scheduleAutoSave() {
  saveStatus.value = 'saving'
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  autoSaveTimer = setTimeout(() => saveData(), 2000)
}

// Local profile storage
const localProfiles = ref({})
const hasLocalProfiles = computed(() => Object.keys(localProfiles.value).length > 0)

function loadLocalProfiles() {
  try {
    const stored = localStorage.getItem('pregnancy_profiles')
    if (stored) localProfiles.value = JSON.parse(stored)
  } catch { /* ignore */ }
}

function saveLocalProfile(id, key) {
  localProfiles.value[id] = key
  localStorage.setItem('pregnancy_profiles', JSON.stringify(localProfiles.value))
}

function removeLocalProfile(id) {
  delete localProfiles.value[id]
  localStorage.setItem('pregnancy_profiles', JSON.stringify(localProfiles.value))
}

// Pregnancy calculations
const currentWeek = computed(() => {
  if (!profileEDD.value) return 0
  const edd = new Date(profileEDD.value)
  const now = new Date()
  const lmp = new Date(edd.getTime() - 280 * 86400000)
  const days = Math.floor((now - lmp) / 86400000)
  return Math.max(0, Math.floor(days / 7))
})

const currentDay = computed(() => {
  if (!profileEDD.value) return 0
  const edd = new Date(profileEDD.value)
  const now = new Date()
  const lmp = new Date(edd.getTime() - 280 * 86400000)
  const days = Math.floor((now - lmp) / 86400000)
  return Math.max(0, days % 7)
})

const daysRemaining = computed(() => {
  if (!profileEDD.value) return 0
  const edd = new Date(profileEDD.value)
  const now = new Date()
  return Math.max(0, Math.ceil((edd - now) / 86400000))
})

const pregnancyProgress = computed(() => {
  const total = 280
  const elapsed = currentWeek.value * 7 + currentDay.value
  return Math.min(100, Math.round((elapsed / total) * 100))
})

const progressColor = computed(() => {
  const p = pregnancyProgress.value
  if (p < 33) return '#67c23a'
  if (p < 66) return '#e6a23c'
  return '#f56c6c'
})

const babySizeData = [
  { week: 4, size: '罂粟籽' }, { week: 5, size: '芝麻' }, { week: 6, size: '扁豆' },
  { week: 7, size: '蓝莓' }, { week: 8, size: '覆盆子' }, { week: 9, size: '葡萄' },
  { week: 10, size: '金桔' }, { week: 11, size: '无花果' }, { week: 12, size: '青柠' },
  { week: 13, size: '柠檬' }, { week: 14, size: '桃子' }, { week: 15, size: '苹果' },
  { week: 16, size: '牛油果' }, { week: 17, size: '石榴' }, { week: 18, size: '甜椒' },
  { week: 19, size: '芒果' }, { week: 20, size: '香蕉' }, { week: 21, size: '胡萝卜' },
  { week: 22, size: '木瓜' }, { week: 23, size: '葡萄柚' }, { week: 24, size: '玉米' },
  { week: 25, size: '花椰菜' }, { week: 26, size: '葱' }, { week: 27, size: '菜花' },
  { week: 28, size: '茄子' }, { week: 29, size: '南瓜' }, { week: 30, size: '大白菜' },
  { week: 31, size: '椰子' }, { week: 32, size: '哈密瓜' }, { week: 33, size: '菠萝' },
  { week: 34, size: '甜瓜' }, { week: 35, size: '蜜瓜' }, { week: 36, size: '长生果' },
  { week: 37, size: '冬瓜' }, { week: 38, size: '韭菜' }, { week: 39, size: '小西瓜' },
  { week: 40, size: '西瓜' }
]

const babySize = computed(() => {
  const w = currentWeek.value
  const found = babySizeData.find(d => d.week === w)
  return found ? found.size : (w < 4 ? '太小了' : '足月宝宝')
})

// Checklist helpers
function getCheckedCount(items) {
  if (!items || !Array.isArray(items)) return 0
  return items.filter(i => i.checked).length
}

const hospitalBagProgress = computed(() => {
  const bag = profileData.hospital_bag
  let total = 0, checked = 0
  for (const cat in bag) {
    if (Array.isArray(bag[cat])) {
      total += bag[cat].length
      checked += getCheckedCount(bag[cat])
    }
  }
  return total ? Math.round((checked / total) * 100) : 0
})

const babyEssentialsProgress = computed(() => {
  const items = profileData.baby_essentials
  let total = 0, checked = 0
  for (const cat in items) {
    if (Array.isArray(items[cat])) {
      total += items[cat].length
      checked += getCheckedCount(items[cat])
    }
  }
  return total ? Math.round((checked / total) * 100) : 0
})

function addItem(section, category) {
  const name = newItems[section][category]?.trim()
  if (!name) return
  profileData[section][category].push({ name, checked: false })
  newItems[section][category] = ''
  scheduleAutoSave()
}

function removeItem(section, category, idx) {
  profileData[section][category].splice(idx, 1)
  scheduleAutoSave()
}

// Prenatal check helpers
function isCurrentCheck(weekStr) {
  const w = currentWeek.value
  if (weekStr.includes('-')) {
    const [start, end] = weekStr.split('-').map(Number)
    return w >= start && w <= end
  }
  return w === Number(weekStr)
}

// Weight helpers
const sortedWeightRecords = computed(() => {
  if (!profileData.weight_records) return []
  return [...profileData.weight_records].sort((a, b) => b.date.localeCompare(a.date))
})

function addWeight() {
  if (!newWeight.date || !newWeight.value) return
  profileData.weight_records.push({
    date: newWeight.date,
    value: newWeight.value
  })
  newWeight.date = new Date().toISOString().slice(0, 10)
  scheduleAutoSave()
}

function removeWeight(idx) {
  const sorted = sortedWeightRecords.value
  const record = sorted[idx]
  const realIdx = profileData.weight_records.findIndex(
    r => r.date === record.date && r.value === record.value
  )
  if (realIdx >= 0) {
    profileData.weight_records.splice(realIdx, 1)
    scheduleAutoSave()
  }
}

function getWeightChange(idx) {
  const records = sortedWeightRecords.value
  if (idx >= records.length - 1) return '-'
  const diff = records[idx].value - records[idx + 1].value
  return (diff > 0 ? '+' : '') + diff.toFixed(1) + 'kg'
}

function getWeightChangeColor(idx) {
  const records = sortedWeightRecords.value
  if (idx >= records.length - 1) return ''
  const diff = records[idx].value - records[idx + 1].value
  return diff > 0 ? 'var(--color-warning)' : 'var(--color-success)'
}

// Weight chart
const chartWidth = 600
const chartHeight = 250
const chartPadding = { top: 20, right: 20, bottom: 30, left: 50 }

const chartData = computed(() => {
  if (!profileData.weight_records || profileData.weight_records.length < 2) return null
  const sorted = [...profileData.weight_records].sort((a, b) => a.date.localeCompare(b.date))
  const values = sorted.map(r => r.value)
  const min = Math.floor(Math.min(...values) - 1)
  const max = Math.ceil(Math.max(...values) + 1)
  return { sorted, min, max }
})

const chartGridLines = computed(() => {
  if (!chartData.value) return []
  const { min, max } = chartData.value
  const lines = []
  const innerH = chartHeight - chartPadding.top - chartPadding.bottom
  const steps = 4
  for (let i = 0; i <= steps; i++) {
    const y = chartPadding.top + (innerH / steps) * i
    lines.push({
      x1: chartPadding.left, y1: y,
      x2: chartWidth - chartPadding.right, y2: y
    })
  }
  return lines
})

const chartYLabels = computed(() => {
  if (!chartData.value) return []
  const { min, max } = chartData.value
  const labels = []
  const innerH = chartHeight - chartPadding.top - chartPadding.bottom
  const steps = 4
  for (let i = 0; i <= steps; i++) {
    const y = chartPadding.top + (innerH / steps) * i
    const val = max - ((max - min) / steps) * i
    labels.push({ x: chartPadding.left - 5, y: y + 4, text: val.toFixed(1) })
  }
  return labels
})

const chartPoints = computed(() => {
  if (!chartData.value) return ''
  const { sorted, min, max } = chartData.value
  const innerW = chartWidth - chartPadding.left - chartPadding.right
  const innerH = chartHeight - chartPadding.top - chartPadding.bottom
  return sorted.map((r, i) => {
    const x = chartPadding.left + (innerW / (sorted.length - 1)) * i
    const y = chartPadding.top + innerH - ((r.value - min) / (max - min)) * innerH
    return `${x},${y}`
  }).join(' ')
})

const chartDots = computed(() => {
  if (!chartData.value) return []
  const { sorted, min, max } = chartData.value
  const innerW = chartWidth - chartPadding.left - chartPadding.right
  const innerH = chartHeight - chartPadding.top - chartPadding.bottom
  return sorted.map((r, i) => ({
    x: chartPadding.left + (innerW / (sorted.length - 1)) * i,
    y: chartPadding.top + innerH - ((r.value - min) / (max - min)) * innerH
  }))
})

// Fetal movement
const fetalTimerDisplay = computed(() => {
  const s = fetalTimerElapsed.value
  const m = Math.floor(s / 60)
  const sec = s % 60
  return `${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`
})

function startFetalTimer() {
  fetalCount.value = 0
  fetalTimerElapsed.value = 0
  fetalTimerRunning.value = true
  fetalTimerStart.value = new Date()
  fetalInterval = setInterval(() => {
    fetalTimerElapsed.value++
  }, 1000)
}

function recordKick() {
  fetalCount.value++
}

function resetFetalCounter() {
  fetalCount.value = 0
  fetalTimerElapsed.value = 0
  if (fetalTimerStart.value) fetalTimerStart.value = new Date()
}

function stopFetalTimer() {
  if (fetalInterval) {
    clearInterval(fetalInterval)
    fetalInterval = null
  }
  fetalTimerRunning.value = false

  if (fetalCount.value > 0) {
    const now = new Date()
    profileData.fetal_movements.unshift({
      date: now.toISOString().slice(0, 10),
      start_time: fetalTimerStart.value.toTimeString().slice(0, 5),
      duration: Math.round(fetalTimerElapsed.value / 60),
      count: fetalCount.value
    })
    scheduleAutoSave()
  }

  fetalCount.value = 0
  fetalTimerElapsed.value = 0
}

function removeFetalRecord(idx) {
  profileData.fetal_movements.splice(idx, 1)
  scheduleAutoSave()
}

// Knowledge category filter
const knowledgeCategory = ref('all')
const knowledgeCategoryOptions = [
  { label: '全部', value: 'all' },
  { label: '孕早期(1-12周)', value: 'early' },
  { label: '孕中期(13-28周)', value: 'mid' },
  { label: '孕晚期(29-40周)', value: 'late' },
  { label: '专题指南', value: 'guide' }
]

const filteredKnowledgeData = computed(() => {
  if (knowledgeCategory.value === 'all') return knowledgeData
  if (knowledgeCategory.value === 'guide') return knowledgeData.filter(d => d.category === 'guide')
  if (knowledgeCategory.value === 'early') return knowledgeData.filter(d => d.weeks && d.weeks[0] >= 1 && d.weeks[1] <= 12)
  if (knowledgeCategory.value === 'mid') return knowledgeData.filter(d => d.weeks && d.weeks[0] >= 13 && d.weeks[1] <= 28)
  if (knowledgeCategory.value === 'late') return knowledgeData.filter(d => d.weeks && d.weeks[0] >= 29)
  return knowledgeData
})

// Knowledge data
const knowledgeData = [
  // ==================== 孕早期 ====================
  {
    id: 'w1-4', weeks: [1, 4], title: '孕1-4周：受精与着床', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-early-trimester/',
    development: '精子与卵子在输卵管内结合形成受精卵，受精卵经过约6-7天的旅程到达子宫腔并着床于子宫内膜。着床时受精卵仅有针尖大小，此时细胞快速分裂，形成胚泡。胚泡分为内细胞团（将发育为胎儿）和外滋养层（将发育为胎盘）。',
    size: '受精卵比针尖还小，约0.1-0.2mm，肉眼几乎不可见。',
    symptoms: '大多数女性还不知道自己怀孕。少数人可能在着床时有轻微的点滴出血（着床出血），持续1-2天，颜色较浅。部分人可能感到轻微腹胀或乳房轻微胀痛。',
    tips: '如果计划怀孕，从备孕期就应开始服用叶酸（每天0.4-0.8mg），可降低神经管缺陷风险达70%。避免烟酒和有害物质接触。用排卵试纸或基础体温法确认排卵期。停止使用可能致畸的药物，包括维A酸类护肤品。',
    danger_signs: '如有大量出血、剧烈腹痛，需排除宫外孕可能。',
    diet: '均衡饮食，重点补充叶酸。多吃深绿色蔬菜（菠菜、西兰花）、豆类、全谷物、柑橘类水果。每天保证优质蛋白质摄入。开始戒掉咖啡或控制在每天200mg以内。',
    taboo: '禁止饮酒（即使少量也可能影响胚胎）。避免生鱼片、未煮熟的肉蛋、未消毒奶制品。避免含汞较高的鱼类（鲨鱼、旗鱼、金枪鱼）。停用含维A酸的护肤品。',
    exercise: '保持正常运动即可，不需要特别改变。散步、游泳、低强度瑜伽都很适合。避免高温环境（桑拿、温泉、热瑜伽）。',
    emotional: '如果是计划怀孕，等待验孕结果可能让人焦虑。保持平常心，过度紧张反而不利于受孕和着床。',
    partner: '陪伴伴侣做孕前检查。一起戒烟戒酒。了解怀孕基础知识。提前规划经济和生活安排。'
  },
  {
    id: 'w5-6', weeks: [5, 6], title: '孕5-6周：确认怀孕', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-early-trimester/',
    development: '胚胎像一粒小苹果种子。神经管开始形成，将发育为大脑和脊髓。心脏开始以原始形态跳动（约每分钟100-120次），虽然还只有罂粟籽大小。胚胎呈C形弯曲，有了头尾之分。',
    size: '约2-4mm，像一粒苹果种子或扁豆。',
    symptoms: '月经推迟是最明显的信号。早孕试纸可以测出阳性。许多人开始出现孕吐（尤其是早晨）、乳房胀痛加重、极度疲劳、频繁排尿。对气味变得敏感，可能出现食欲改变。',
    checkups: '首次验孕确认。可以去医院抽血查HCG和孕酮值，确认怀孕并初步评估胚胎发育情况。HCG应隔天翻倍增长。',
    tips: '确认怀孕后建卡（部分医院要求12周前）。开始记录末次月经日期（计算预产期的关键）。避免接触猫砂（弓形虫风险）。通知单位相关人员，了解产假政策。',
    danger_signs: 'HCG翻倍不理想需排除宫外孕。剧烈一侧腹痛+阴道出血是宫外孕典型症状，需立即就医。少量褐色分泌物可能正常，但鲜红色出血需要重视。',
    diet: '少量多餐对抗孕吐。起床前先吃几片苏打饼干。饮用姜茶或含姜食物可缓解恶心。避免油腻、辛辣、气味强烈的食物。补充维生素B6（25mg，每日3次）可缓解孕吐。',
    taboo: '绝对禁止饮酒和吸烟（包括二手烟）。避免咖啡因过量（每天不超过200mg）。不吃生冷海鲜。避免食用未经巴氏消毒的乳制品和果汁。',
    exercise: '温和的散步和拉伸运动。如果孕吐严重，不要勉强运动。避免腹部冲击性运动。',
    emotional: '得知怀孕可能带来兴奋、紧张、甚至恐惧，这些都是正常的。允许自己有复杂的情绪。如果感到过度焦虑，可以和伴侣或朋友倾诉。',
    partner: '陪伴验孕和第一次产检。理解孕早期伴侣的不适和情绪波动。主动承担更多家务。学习孕期禁忌知识。'
  },
  {
    id: 'w7-8', weeks: [7, 8], title: '孕7-8周：第一次听到胎心', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-early-trimester/',
    development: '胚胎快速发育！手臂和腿的萌芽出现，像小小的桨。面部特征开始形成——眼睛的色素、鼻孔、嘴唇的轮廓。大脑每分钟产生约100个新的神经细胞。肝脏开始制造红血球。到8周末，胚胎的所有主要器官和身体系统都已开始发育。',
    size: '约1.2-1.6cm，从头到臀约有一颗蓝莓到覆盆子大小。体重约1克。',
    symptoms: '孕吐可能达到高峰（最严重的时期通常在6-9周）。乳房继续增大、变敏感。子宫开始增大但从外表还看不出来。疲劳感明显，可能需要比平时多睡1-2小时。',
    checkups: '第一次正式产检！B超确认宫内妊娠、胎心胎芽。可以听到胎心跳动（约每分钟120-160次）——这是非常激动人心的时刻！抽血查血常规、血型（ABO+Rh）、肝肾功能、甲状腺功能、传染病筛查（乙肝、HIV、梅毒）、尿常规。',
    tips: '看到胎心是一个重要的里程碑！有胎心后流产风险大幅降低。开始每天记录饮食和体重。选择宽松舒适的衣服。如果工作环境有辐射或化学物质接触，需告知单位调换岗位。',
    danger_signs: '如果B超未见胎心，不要过于担心——可能是排卵推迟导致实际孕周偏小，1-2周后复查。持续性阴道出血伴腹痛需立即就医。孕吐严重到无法进食进水（妊娠剧吐）需就医输液。',
    diet: '应对孕吐：避免空腹，随身带零食。冷食比热食气味小，可能更容易接受。酸味食物（柠檬水、酸梅）有助缓解恶心。保证每天至少摄入1500ml水分，可以少量多次。',
    taboo: '避免食用芦荟、薏米（可能引起宫缩）。不吃腌制食品和含亚硝酸盐的加工肉类。避免高糖高脂零食。远离含甲醛的新装修环境。',
    exercise: '可以做温和的瑜伽和散步。如有先兆流产症状，医生可能建议卧床休息。避免骑自行车和滑雪等平衡要求高的运动。',
    emotional: '看到胎心的喜悦和对未来的担忧可能并存。孕激素的变化会让情绪更加起伏。可以开始写孕期日记记录感受。',
    partner: '一起去第一次产检，共同见证胎心。学会理解伴侣因激素变化导致的情绪波动。帮忙做饭时注意避免气味刺激。夜里主动照顾频繁起夜的伴侣。'
  },
  {
    id: 'w9-10', weeks: [9, 10], title: '孕9-10周：从胚胎到胎儿', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-early-trimester/',
    development: '从第9周开始，胚胎正式升级为"胎儿"！所有主要器官已经形成（器官形成期结束）。手指和脚趾开始分开，不再是蹼状。骨骼开始从软骨硬化为真正的骨头。胎儿已经可以做出微小的动作，虽然妈妈还感觉不到。',
    size: '约2.5-3cm，像一颗葡萄或金桔。体重约4克。头部占身体的一半左右。',
    symptoms: '孕吐可能依然存在但有些人开始好转。腰围可能开始变粗。皮肤可能变化——有些人变得更好（孕期光泽），有些人可能长痘。静脉可能更加明显（血容量增加）。',
    checkups: '如果第一次产检在8周做过，这期间通常不需要额外检查。如果之前未做过产检，现在应该尽快完成。',
    tips: '现在是致畸敏感期的尾声，继续避免接触有害物质。可以开始研究孕期保险和生育保险政策。如果有宠物猫，让他人负责清理猫砂。',
    danger_signs: '突发的严重腹痛、大量出血需立即急诊。持续高烧（>38.5°C）需及时就医，持续高烧可能影响胎儿神经管发育。',
    diet: '器官形成期接近尾声，但营养依然关键。增加优质蛋白（鸡蛋、鱼、瘦肉、豆腐）。每天1-2份奶制品补充钙质。多吃富含铁的食物（红肉、菠菜、黑木耳）预防后期贫血。',
    taboo: '继续避免生食。不要自行服用任何药物（包括中药），用药需咨询医生。避免含双酚A（BPA）的塑料容器加热食物。',
    exercise: '可以开始系统的孕期运动计划。游泳是整个孕期最推荐的运动之一——浮力减轻关节压力，水压促进血液循环。孕期瑜伽有助于保持柔韧性和心理平静。',
    emotional: '即将度过孕早期最危险的阶段，不安感可能开始减轻。考虑何时向亲友宣布喜讯——很多人选择12周NT检查通过后。',
    partner: '注意伴侣的饮食营养，可以一起制定健康食谱。开始了解产检流程和重要时间节点。商量是否需要更换更大的住所或调整房间布局。'
  },
  {
    id: 'w11-12', weeks: [11, 12], title: '孕11-12周：NT检查里程碑', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-early-trimester/',
    development: '胎儿已具备人形！头发和指甲开始生长。肾脏开始产生尿液。外生殖器开始分化，但B超还不能准确分辨性别。胎儿可以打哈欠、打嗝、吞咽羊水。反射动作出现——碰触掌心会握拳。',
    size: '约5-6cm，像一颗青柠或鸡蛋大小。体重约14克。',
    symptoms: '孕吐逐渐好转（大多数人在12-14周完全消失）。食欲可能开始恢复。小腹微微隆起，穿紧身裤可能感到不适。头发可能变得更浓密有光泽（雌激素的好处！）。',
    checkups: 'NT检查（11-13+6周）——通过超声测量胎儿颈后透明带厚度，评估唐氏综合征等染色体异常风险。NT值<2.5mm通常被认为正常。同时可选做早期唐筛抽血。这是一个重要的筛查节点！建议同时建档（围产保健手册）。',
    tips: 'NT检查通过是一个里程碑，很多家庭选择此时向亲友宣布怀孕。开始购买孕妇内衣（无钢圈、棉质）。可以开始涂抹预防妊娠纹的产品（橄榄油、妊娠纹霜）。办理母子健康手册。',
    danger_signs: 'NT值增厚不一定意味着有问题，可能需要进一步检查（如无创DNA或羊水穿刺）。不要自行百度吓自己，听医生的专业建议。',
    diet: '食欲恢复后注意不要暴饮暴食。孕早期体重增长应控制在1-2kg。开始注重"质而非量"——每餐都包含蛋白质、复合碳水和蔬果。每天补充DHA（200-300mg），可从鱼油或藻油获取。',
    taboo: '不要因为食欲好转就放纵吃甜食和垃圾食品。避免食用含铅较高的食物（松花蛋）。不用含水杨酸的护肤品。',
    exercise: '进入孕中期前的过渡期，可以适当增加运动量。推荐每天30分钟中等强度运动。凯格尔运动（盆底肌训练）从现在开始坚持到产后——收缩盆底肌5秒，放松5秒，重复10-15次，每天3组。',
    emotional: '通过NT检查后会有如释重负的感觉。可以开始享受怀孕的美好。有些妈妈开始和肚子里的宝宝说话。',
    partner: '一起去做NT检查，看看宝宝在B超里的样子。开始商量起名字！帮伴侣涂抹妊娠纹预防产品（这也是一种亲密的方式）。'
  },

  // ==================== 孕中期 ====================
  {
    id: 'w13-14', weeks: [13, 14], title: '孕13-14周：进入黄金孕中期', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-mid-trimester/',
    development: '欢迎来到孕中期！胎儿开始了快速生长阶段。面部表情更加丰富——可以皱眉、斜视。声带形成。肝脏开始分泌胆汁，胰腺开始产生胰岛素。胎儿开始练习呼吸动作（吸入和排出羊水）。',
    size: '约7-8cm（头臀长），像一颗柠檬。体重约25-45克。身体比例开始变得更协调，头不再占身体一半了。',
    symptoms: '大多数人孕吐消失，食欲大增——这是被称为"蜜月期"的孕中期！精力明显恢复。可能出现鼻塞（孕期鼻炎，激素导致鼻黏膜肿胀）。腹部开始隆起，可能需要孕妇裤了。',
    checkups: '常规产检（血压、体重、尿常规）。如果早期唐筛未做，现在是安排中期唐筛或无创DNA的好时机。',
    tips: '趁精力好，处理一些重要事务——旅行、装修准备、采购。开始购买孕妇装和宽松鞋子（脚可能变大半个码）。可以开始胎教——每天固定时间和宝宝说话、放音乐。侧卧睡姿开始习惯（左侧卧最佳）。',
    danger_signs: '孕中期出血虽然比较少见，但仍需重视。突发的腹部紧缩感可能是宫缩，需要就医评估。',
    diet: '热量需求增加约300kcal/天（相当于多吃一个鸡蛋+一杯牛奶+一片面包）。重点补充：钙（1000mg/天）、铁（27mg/天）、DHA（200mg/天）。多吃深色蔬果——胡萝卜、南瓜、蓝莓、草莓。',
    taboo: '避免暴饮暴食（孕中期是体重增长最快的时期）。不要吃太多高糖水果（控制每天200-400克）。减少加工食品和外卖。',
    exercise: '孕中期是运动的黄金时期！推荐：散步（每天30-45分钟）、孕期瑜伽、游泳、固定自行车。运动时保持"能说话"的强度。开始练习深呼吸和放松技巧。',
    emotional: '精力恢复后情绪也更稳定。这是享受孕期的最佳时期。可以和伴侣计划一次"babymoon"旅行。',
    partner: '这段时间伴侣状态最好，一起享受！帮忙研究婴儿用品。陪伴做产检。开始改善家居环境——甲醛检测、安全隐患排查。'
  },
  {
    id: 'w15-16', weeks: [15, 16], title: '孕15-16周：唐筛关键期', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-mid-trimester/',
    development: '胎儿开始活跃运动——翻滚、踢腿、伸手，但初产妇可能还感觉不到。皮肤透薄，可以看到下面的血管网络。开始长出胎毛（细软的绒毛覆盖全身，起保护作用）。味蕾形成，可以"品尝"不同口味的羊水。',
    size: '约10-12cm，像一个牛油果。体重约80-100克。',
    symptoms: '腹部隆起明显，不再能隐藏。可能出现妊娠纹的前兆——皮肤发痒。牙龈可能变得敏感易出血。鼻塞可能持续。有些人开始感到轻微的腰痛。',
    checkups: '中期唐筛（15-20周）：抽血检测AFP、HCG、uE3等指标，评估唐氏综合征和神经管缺陷风险。或选择无创DNA检测（准确率更高，约99%）。高风险者需考虑羊水穿刺（确诊检查）。',
    tips: '唐筛结果"高风险"不等于宝宝有问题——筛查的特点是宁可错报不可漏报。听从医生建议做进一步检查。可以开始注意防晒——孕期容易出现妊娠斑（黄褐斑）。购买孕妇枕（U型或C型），改善睡眠质量。',
    danger_signs: '宫颈机能不全可能在这个时期发现（无痛性宫颈缩短）——定期B超监测宫颈长度很重要，尤其有过早产史或宫颈手术史的孕妈。',
    diet: '继续重点补充钙、铁、DHA。开始注意碘的摄入（使用碘盐、吃海带/紫菜），碘对胎儿大脑发育至关重要。每天1-2个鸡蛋提供优质蛋白和卵磷脂。核桃、杏仁等坚果每天一小把（约20克）。',
    taboo: '限制咖啡因摄入（咖啡、浓茶、可乐、巧克力都含有）。避免食用桂圆（性热，可能引起出血）。不吃山楂（可能刺激宫缩）。',
    exercise: '保持每周至少150分钟中等强度运动。孕中期适合的运动：快走、孕期有氧操、水中有氧运动。避免仰卧位运动（子宫可能压迫下腔静脉）。',
    emotional: '等待唐筛结果可能焦虑——尽量分散注意力。和有经验的妈妈朋友聊聊天。可以加入孕妈群交流经验。',
    partner: '陪伴等待唐筛结果。如果结果异常，一起冷静地咨询医生。开始参加孕妈课程或爸爸训练营。了解生育保险报销流程。'
  },
  {
    id: 'w17-18', weeks: [17, 18], title: '孕17-18周：初感胎动', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-mid-trimester/',
    development: '胎儿的骨骼从软骨逐渐硬化为骨质。脂肪开始在皮肤下积累。指纹形成——每个宝宝都是独一无二的！胎儿可以听到妈妈的心跳和肠胃蠕动的声音。网膜（视网膜）开始对光线敏感。',
    size: '约13-14cm，像一个石榴或甜椒。体重约150-190克。',
    symptoms: '初产妈妈可能在18-20周首次感受到胎动！经产妈妈可能更早（16周左右）。初次胎动像是蝴蝶扑翅、气泡破裂或鱼在游动的感觉，很轻很柔。腹部继续增大。可能出现腿部抽筋（尤其夜间）。',
    tips: '第一次感到胎动是非常美妙的时刻！但不必担心如果还没感觉到——每个人感知时间不同。开始注意睡姿——左侧卧可以改善子宫和胎盘血液循环。可以购买托腹带减轻腰部负担。',
    danger_signs: '如果此前一直能感到胎动突然消失超过12小时，应就医检查。下腹持续疼痛或阴道流血需要立即就医。',
    diet: '铁的需求明显增加——红肉是最好的铁来源（血红素铁吸收率高）。配合维C丰富的食物促进铁吸收。每天保证500ml牛奶或等量乳制品。预防便秘：多喝水、多吃膳食纤维、每天吃一些红薯或燕麦。',
    taboo: '避免饮用含糖饮料。少吃精制碳水化合物（白面包、白米饭改为全麦/糙米）。不要暴食甜品——为糖耐量检查做准备。',
    exercise: '可以尝试孕妇普拉提（在专业教练指导下）。游泳特别适合缓解腰痛。睡前做小腿拉伸可预防夜间抽筋。避免跳跃和突然改变方向的运动。',
    emotional: '感受到胎动让怀孕变得更加真实，与宝宝的联结感增强。这是开始胎教的好时机——每天和宝宝说话、讲故事、唱歌。',
    partner: '把手放在伴侣肚子上一起等待胎动（虽然此时从外面可能还摸不到）。开始和宝宝说话——宝宝能听到并会逐渐熟悉你的声音。一起选择胎教音乐。'
  },
  {
    id: 'w19-20', weeks: [19, 20], title: '孕19-20周：孕程过半！', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-mid-trimester/',
    development: '恭喜！你已经走过了孕程的一半！胎儿的感官系统快速发育——触觉、味觉、嗅觉、听觉、视觉都在发展。女宝宝的子宫和卵巢已经形成，卵巢中已有约600万个原始卵母细胞。男宝宝的睾丸开始下降。皮肤表面覆盖了白色的胎脂，保护皮肤不被羊水浸泡。',
    size: '约15-16cm（头臀长），从头到脚约25cm。像一根香蕉。体重约250-300克。',
    symptoms: '胎动更加明显和频繁。腹部继续增大，可能开始出现妊娠纹。腰痛可能加重。部分人可能出现头晕（血压波动）。腿部水肿可能开始出现。',
    checkups: '大排畸超声（20-24周）——这是整个孕期最重要的超声检查之一！系统筛查胎儿所有器官结构：大脑、心脏、脊柱、四肢、面部、内脏等。可以选择三维或四维超声，能看到宝宝的面部轮廓。这次B超通常可以看出性别（如果你想知道的话）。',
    tips: '大排畸是一个重要的里程碑检查。提前预约，因为这个检查很抢手。检查时宝宝体位不好可能需要散步后重新检查。开始考虑分娩医院的选择——公立还是私立？是否有无痛分娩？',
    danger_signs: '大排畸发现"软指标"异常（如肾盂扩张、脉络丛囊肿）大多是暂时的，后期会自行消失。但结构性异常需要认真评估。信任医生的判断。',
    diet: '体重管理开始变得重要——孕中期每周增长约0.5kg为宜。增加锌的摄入（牡蛎、牛肉、南瓜子），锌对胎儿生长发育非常重要。保证充足的维生素D——适当晒太阳（每天15-20分钟）或补充维生素D3。',
    taboo: '控制钠的摄入减轻水肿——少吃腌制品、方便面、膨化食品。避免趴睡和长时间仰卧。不要穿高跟鞋。',
    exercise: '继续规律运动。孕妇游泳课是很好的选择。做骨盆摇摆运动缓解腰痛。可以开始练习拉玛泽呼吸法——这对将来分娩很有帮助。',
    emotional: '大排畸通过会带来巨大的安心感。看到宝宝的四维照片可能让你更加兴奋。这是一个好时机——给宝宝拍第一张"照片"留念。',
    partner: '大排畸一定要陪伴！一起看宝宝的样子，讨论像谁。如果发现任何问题，给予伴侣最大的支持和安慰。可以开始布置婴儿房。'
  },
  {
    id: 'w21-24', weeks: [21, 24], title: '孕21-24周：胎儿存活分界线', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-mid-trimester/',
    development: '这是一个重要的时间节点——24周被称为"胎儿存活分界线"，即胎儿在此之后出生有存活的可能（虽然需要大量医疗支持）。胎儿的肺部开始产生表面活性物质（帮助出生后肺部扩张）。大脑飞速发育，可以对声音做出反应。眼睛可以睁开了！皮肤仍然是红色半透明的，很皱。',
    size: '约28-30cm（头到脚），体重约500-600克。像一根大玉米。',
    symptoms: '胎动变得有力，从外面可以看到或摸到肚皮的动静。子宫底到达肚脐水平。可能出现假性宫缩（肚子发紧但不疼，持续几秒到一分钟）——这是正常的。手脚水肿可能加重。可能出现烧心/胃食管反流。',
    checkups: '糖耐量检查（OGTT，24-28周）——这是筛查妊娠期糖尿病的关键检查。检查前一晚10点后禁食，次日空腹抽血，喝75克葡萄糖水，分别在1小时和2小时后再抽血。正常值：空腹<5.1，1h<10.0，2h<8.5（mmol/L）。',
    tips: '开始数胎动——每天固定时间（最好是晚餐后），安静坐下或侧卧，计数1小时内的胎动次数，正常应≥3次。如果2小时<6次，需要就医。开始了解分娩知识——阅读相关书籍或参加分娩准备课。购买婴儿安全座椅（出生后回家就需要）。',
    danger_signs: '规律性腹痛（每10分钟一次或更频繁）是早产征兆！阴道大量流液（破水）需立即就医。突然的严重头痛、视力模糊、上腹疼痛可能是子痫前期的征兆。如果胎动突然减少或消失，立即就医。',
    diet: '糖耐量检查前1-2周开始注意饮食——不必刻意控制，保持正常健康饮食即可。无论检查结果如何，都应控制精制糖摄入。增加蛋白质摄入至每天75-100克。每天至少5份蔬果。补充镁（深色蔬菜、坚果）预防腿部抽筋。',
    taboo: '不要为了糖耐检查前突击控制饮食——这可能导致结果不准确。避免高盐饮食加重水肿。睡前不宜大量进食以减轻胃食管反流。',
    exercise: '运动强度可能需要适当降低。游泳和散步仍然是最佳选择。做蹲起练习有助于打开骨盆，为分娩做准备。避免长时间站立——如果必须站立，每30分钟活动一下。',
    emotional: '随着肚子变大和分娩日期逐渐临近，可能出现对分娩的担忧。这很正常——参加分娩准备课可以有效减轻焦虑。和已经分娩过的朋友聊聊真实经验。',
    partner: '陪伴做糖耐量检查（等待时间较长）。一起参加分娩准备课程。学习如何帮助伴侣做腰背按摩。了解什么情况需要紧急送医。'
  },
  {
    id: 'w25-28', weeks: [25, 28], title: '孕25-28周：进入孕晚期前奏', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-mid-trimester/',
    development: '胎儿的大脑发育进入关键期——脑沟回（大脑的褶皱）开始形成。肺部继续成熟，但如果现在出生仍需呼吸机支持。皮下脂肪快速积累，皮肤逐渐变得不透明。胎儿开始有睡眠-觉醒周期，可能在妈妈想休息时特别活跃。眼睛可以追踪光源。',
    size: '约35-37cm，体重约800-1100克。像一颗茄子或小南瓜。',
    symptoms: '胎动非常活跃，有时候能看到肚子上"跳舞"。假性宫缩更加频繁。可能出现手指麻木（腕管综合征——孕期水肿压迫神经）。睡眠可能受到影响——找不到舒服的姿势。背痛加重。',
    checkups: '常规产检（血压、体重、宫高、腹围、胎心）。如果糖耐量异常，需要开始监测血糖。血常规检查评估是否贫血——孕中晚期贫血很常见（血容量增加稀释了血红蛋白）。28周后如果是Rh阴性血型，需要注射抗D免疫球蛋白。',
    tips: '开始准备待产包！不要等到最后才准备。了解医院的入院流程和所需证件。和医生讨论分娩计划——是否想要无痛分娩？对侧切有什么想法？是否存在需要剖宫产的指征？开始做产假交接计划。',
    danger_signs: '子痫前期的高危期——定期监测血压。如果出现持续性头痛、视力异常、上腹疼痛、突发严重水肿（尤其是面部和手部），立即就医。胎动突然减少需要马上做胎心监护。',
    diet: '增加铁的补充——如果血红蛋白<110g/L，医生可能会开铁剂。铁剂与维生素C同服吸收更好，避免与钙片和茶同服。继续补充DHA，这个时期大脑发育需要大量DHA。每天25-30克膳食纤维预防便秘（燕麦、红薯、各种蔬菜）。',
    taboo: '不要忽视定期产检——孕晚期并发症的早期发现很重要。避免长途旅行。减少盐的摄入。不要自行决定补充铁剂的剂量。',
    exercise: '运动时注意平衡——重心已经明显前移。散步速度可以放慢。孕妇游泳课可以继续。做猫式伸展缓解背痛。避免快速站起（可能头晕）。',
    emotional: '开始出现对分娩的焦虑是正常的。可以写一份分娩计划书，把你的想法和担忧告诉医生。和伴侣深入讨论产后的生活安排。',
    partner: '和伴侣一起准备待产包。了解分娩的全过程——宫缩、开指、二产程、三产程。学习如何在产房支持伴侣。确保任何时候都能联系到并快速赶到医院。'
  },

  // ==================== 孕晚期 ====================
  {
    id: 'w29-30', weeks: [29, 30], title: '孕29-30周：冲刺阶段开始', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-late-trimester/',
    development: '胎儿进入快速增重阶段——从现在到出生，体重将增加约3倍！大脑飞速发育，神经连接大量建立。骨骼快速吸收钙质变硬（但头骨故意保持较软，方便通过产道）。肺部的表面活性物质继续增加。胎儿已经可以做梦了（REM睡眠出现）。',
    size: '约38-40cm，体重约1.2-1.5kg。像一颗大白菜。',
    symptoms: '呼吸开始感到困难——增大的子宫把膈肌往上顶。尿频更严重。可能出现痔疮（增大的子宫压迫直肠静脉）。失眠加重。假性宫缩更频繁但不规律。肋骨可能被顶到疼。',
    checkups: '从28-30周开始，产检频率增加到每2周一次。小排畸超声（30-32周）——再次检查胎儿发育情况和胎位。如果是臀位或横位，还有时间自然转正。监测血压和尿蛋白（排除子痫前期）。',
    tips: '准备好待产包并放在触手可及的地方。确认医院路线（包括备用路线）。了解分娩的预兆——见红、破水、规律宫缩的区别。开始了解母乳喂养知识——正确的含乳姿势、常见问题处理。',
    danger_signs: '从现在开始更需要警惕早产征兆：规律宫缩（每10分钟一次或更频繁持续1小时以上）、腰部持续钝痛、阴道分泌物增多或性质改变、阴道出血或流液。',
    diet: '胎儿大量吸收钙质——每天保证1200mg钙摄入。睡前一杯温牛奶有助睡眠和补钙。增加蛋白质至每天80-100克。少食多餐减轻胃部压力。每天25-30克膳食纤维+充足水分预防便秘和痔疮。',
    taboo: '不要吃得太撑（胃被子宫压缩了空间）。避免产气食物加重胃胀（豆类、洋葱）。晚餐后至少2小时再躺下（预防胃食管反流）。',
    exercise: '以散步和温和伸展为主。练习分娩呼吸法——胸式呼吸、腹式呼吸、哈气呼吸。做蹲起和骨盆摇摆帮助宝宝入盆。凯格尔运动继续坚持。',
    emotional: '随着预产期逐渐临近，焦虑和紧张感可能增强。这是完全正常的。和有经验的妈妈交流。参加医院的产前课程。冥想和深呼吸有助于减轻焦虑。',
    partner: '预习如何做拉玛泽呼吸法的指导者。了解产房陪产的注意事项。确保手机24小时有电、畅通。准备好去医院的路线和紧急联系电话。'
  },
  {
    id: 'w31-32', weeks: [31, 32], title: '孕31-32周：宝宝开始入盆', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-late-trimester/',
    development: '胎儿的五种感觉都已经发育完善。对声音的反应更明显——大声噪音可能让宝宝吓一跳。皮肤不再那么皱了，皮下脂肪让宝宝看起来更饱满。胎毛开始脱落。指甲已经长到指尖。如果这时候出生，存活率超过95%。',
    size: '约40-42cm，体重约1.5-2kg。像一颗大椰子。',
    symptoms: '胎动模式可能改变——因为空间有限，大幅度翻转减少，但推挤和踢的力度更大。可能感到耻骨联合疼痛。走路可能出现"摇摆步态"。乳房可能开始分泌初乳（少量黄色液体）。失眠、频繁的假性宫缩。',
    checkups: '小排畸超声评估胎儿大小、羊水量、胎盘位置和成熟度。确认胎位——如果是臀位，医生可能建议做外倒转术或讨论剖宫产计划。',
    tips: '如果初乳泄漏，可以使用防溢乳垫。开始练习安装和使用婴儿安全座椅。确认新生儿医保办理流程。准备宝宝回家穿的衣服和包被。预约月嫂（如果需要）。',
    danger_signs: '胎动突然减少或增多都需要注意。如果每天胎动少于10次（连续2天），需要做胎心监护。下肢不对称性水肿（一条腿比另一条明显肿）需排除深静脉血栓。',
    diet: '继续高蛋白、高钙、高铁饮食。适当补充维生素K（绿叶蔬菜），有助于新生儿凝血功能。少食多餐（6-8餐/天）。预防便秘和痔疮——多喝水、多吃蔬菜。',
    taboo: '避免长时间坐着或站着不动（增加血栓风险）。不要自行服用泻药。减少辛辣刺激食物（加重痔疮）。',
    exercise: '散步时间可以适当缩短但保持频率。做墙壁深蹲——背靠墙，慢慢蹲下再起来，锻炼大腿和骨盆肌肉。温水泡脚缓解水肿和助眠。',
    emotional: '开始有"筑巢本能"——想要把家里收拾整齐，为宝宝准备好一切。这是自然的本能，但不要过度劳累。可以列清单让家人帮忙。',
    partner: '帮忙安装婴儿床和安全座椅。练习给婴儿换纸尿裤（可以用娃娃练习）。和伴侣讨论产后分工。了解产后抑郁的知识——早期发现很重要。'
  },
  {
    id: 'w33-34', weeks: [33, 34], title: '孕33-34周：最后的生长加速', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-late-trimester/',
    development: '胎儿的免疫系统在快速发展——通过胎盘接收妈妈的抗体。肺部继续成熟。大脑的体积在最后几周增加了约三分之一。胎儿每天大约有90%的时间在睡觉，但活跃的时候非常活跃。',
    size: '约43-45cm，体重约2-2.5kg。像一个菠萝。',
    symptoms: '可能感到骨盆压力增大（宝宝头部下降入盆）。呼吸可能稍有改善（子宫下降减轻了对膈肌的压力），但尿频更加严重。耻骨疼痛加重。全身酸痛。夜间可能出现腿部不安宁综合征（腿部有蚁走感，需要动一动）。',
    checkups: '34-36周做GBS筛查（B族链球菌）——用棉签在阴道和肛门附近采样。GBS阳性（约20-30%的孕妇携带）不代表有问题，但分娩时需要静脉注射抗生素保护新生儿。每周胎心监护可能从34周开始。',
    tips: '确认待产包已齐全。了解医院的入院流程——什么情况下来医院、先去哪里、需要带什么。确认陪产人员的时间安排。如果做剖宫产，和医生确认手术日期。提前了解产后恢复和坐月子注意事项。',
    danger_signs: '持续性头痛或视力改变需立即就医（可能是子痫前期加重）。规律宫缩每5分钟一次持续1小时以上是临产征兆。阴道大量流液（破水）需立即就医——注意颜色，无色或淡黄色正常，绿色或棕色可能是胎便污染。',
    diet: '最后阶段的营养储备很重要。增加优质脂肪摄入（鱼油、牛油果、橄榄油）——支持胎儿大脑的最后发育冲刺。保持充足的铁储备——产后出血是正常的，铁储备好才能更快恢复。',
    taboo: '避免过度补充——体重增长过快可能导致巨大儿（>4kg），增加分娩风险。不要突然进行大量运动。避免长途旅行。',
    exercise: '做好产前运动——猫牛式、骨盆底练习、分娩球上的骨盆摇摆。每天散步15-30分钟。睡前做放松伸展缓解不适。',
    emotional: '做好心理准备——分娩可能和你想象的不一样，保持开放心态。写下你的分娩愿望但也要做好B计划。和伴侣深入沟通产后的期待和担忧。',
    partner: '把医院的路线走一遍（包括夜间）。确认手机有充足的电量。准备陪产时需要的东西（零食、水、充电器）。学习如何在产后照顾伴侣和新生儿。'
  },
  {
    id: 'w35-36', weeks: [35, 36], title: '孕35-36周：临近足月', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-late-trimester/',
    development: '胎儿几乎发育完全！肾脏已经完全成熟。肝脏开始处理一些废物。皮下脂肪继续积累，让宝宝更加圆润可爱。大多数胎儿此时已经头朝下准备出生了。指甲可能已经超过指尖。',
    size: '约45-47cm，体重约2.5-3kg。像一颗大蜜瓜。',
    symptoms: '宫底下降（"入盆"），呼吸变轻松但骨盆和膀胱压力更大。耻骨联合疼痛可能达到高峰。假性宫缩更加频繁。可能有少量黏液排出（宫颈分泌物）。极度疲劳和不适感。',
    checkups: '从36周开始每周产检。每次产检包括：血压、体重、宫高腹围、胎心、尿蛋白。胎心监护（NST）每周1-2次。B超评估胎儿大小、羊水量、胎位。医生可能进行宫颈检查评估成熟度。',
    tips: '36周后随时可能分娩！确保待产包放在车里或门口。保存好医院和医生的联系电话。确认新生儿所需物品已到位。了解产后办理出生证明、上户口的流程。拍些孕期纪念照。',
    danger_signs: '区分真假宫缩：真宫缩会越来越强、间隔越来越短、不会因走动或休息而消失。如果宫缩规律且每5-7分钟一次，准备去医院。破水后无论有无宫缩都应立即就医。',
    diet: '保持营养均衡但不要过量。每天保证充足的水分（2-2.5L）。吃一些有助于软化宫颈的食物（如红枣，但不要过量）。准备一些分娩时可以吃的零食（巧克力、能量棒、功能饮料）。',
    taboo: '不要离家太远旅行。避免提重物。不要过度焦虑——紧张反而可能延迟产程。',
    exercise: '分娩球上的运动非常好——坐在球上轻轻弹跳或做画圆运动。散步可以帮助宝宝入盆。爬楼梯（安全的情况下）有助于启动分娩。但不要过度劳累。',
    emotional: '可能出现矛盾心理——既盼望宝宝快来，又害怕分娩。写一封给宝宝的信。准备一个让自己放松的分娩playlist。记住：千百万的女性都做到了，你也可以。',
    partner: '确保随时待命。手机24小时畅通。背包里放好需要的东西。了解分娩的进程——什么时候该叫医生、什么时候可以打无痛。准备做一个称职的产房陪护。'
  },
  {
    id: 'w37-38', weeks: [37, 38], title: '孕37-38周：足月啦！', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-late-trimester/',
    development: '37周起胎儿已经是足月！此时出生的宝宝基本不需要额外的医疗支持。胎儿在练习吸吮和吞咽为出生后吃奶做准备。肠道中积累了胎便（墨绿色，出生后1-2天排出）。脑部仍在快速发育——这个过程将持续到出生后好几年。',
    size: '约48-50cm，体重约2.8-3.3kg。已经是一个大西瓜了！',
    symptoms: '可能出现"见红"——少量带血丝的黏液排出，说明宫颈在准备。假性宫缩转为不规律的真宫缩（前驱宫缩）。腹部发紧的频率增加。可能出现"筑巢冲动"——突然精力充沛想打扫卫生。排便可能增多（身体在为分娩做准备）。',
    checkups: '每周产检。胎心监护。宫颈检查——医生可能告诉你宫颈开指情况和宫颈评分（Bishop评分）。B超确认羊水量和胎儿状况。如果超过预产期或有并发症，医生可能讨论催产。',
    tips: '识别分娩的三大征兆：①见红（黏液带血丝）——通常在分娩前24-48小时，不需要立即去医院。②破水（突然大量液体流出或持续慢漏）——立即去医院，不要洗澡。③规律宫缩（初产妇5分钟一次持续1小时，经产妇10分钟一次）——出发去医院。',
    danger_signs: '胎动突然减少（每2小时少于6次）需立即就医。破水后羊水颜色异常（绿色/棕色/有臭味）需要紧急处理。持续性剧烈腹痛（不同于宫缩的间歇性）需排除胎盘早剥。',
    diet: '保持正常饮食，不要因为紧张而不吃东西——分娩需要大量体力。准备一些易消化的高能量食物：红牛/功能饮料、巧克力、香蕉、蜂蜜水。分娩开始后可以少量进食。',
    taboo: '不要因为"想让宝宝快来"而尝试不靠谱的催生方法。不要自行灌肠或服用蓖麻油。保持正常作息。',
    exercise: '散步是最好的运动。爬楼梯可以帮助宫缩启动。分娩球上做骨盆运动。做深蹲帮助宝宝下降。但注意不要单独外出——万一宫缩来了需要有人帮忙。',
    emotional: '终点就在眼前！焦虑和期待并存是完全正常的。做几次深呼吸，告诉自己已经做好了准备。分娩是自然的过程，你的身体知道怎么做。',
    partner: '这几周保持随时待命状态。背包放在门口。复习呼吸技巧和按摩方法。分娩开始时保持冷静——你的镇定会传递给伴侣。记住：你是她最重要的支持者。'
  },
  {
    id: 'w39-40', weeks: [39, 40], title: '孕39-40周：预产期倒计时', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-late-trimester/',
    development: '胎儿已经完全发育成熟！平均出生体重约3-3.5kg，身长约50cm。但只有约5%的宝宝会在预产期当天出生——前后两周都属正常。胎盘开始逐渐老化，功能有所下降。',
    size: '约50cm，体重约3-3.5kg。一个真正的小宝宝了！',
    symptoms: '宫缩可能变得更规律。可能感到"胎儿下沉"——走路时有东西要掉出来的感觉。排尿更加频繁。部分人可能出现腹泻。情绪可能在焦急和平静之间反复。',
    checkups: '每周产检+胎心监护。40周仍未分娩的话，医生会评估是否需要催产。通常41-41+6周会建议催产。超过42周的过期妊娠风险增加。B超评估羊水和胎盘。',
    tips: '耐心等待！宝宝会在准备好的时候来。保持日常活动——散步、做家务（适度）。确保手机充满电。把入院需要的证件（身份证、医保卡、母子手册、产检资料）放在一起。通知家人随时待命。',
    danger_signs: '超过41周未发动需要医生评估。胎动明显减少需立即就医。大量阴道出血（不是少量见红）需要急诊。破水后超过12-24小时未分娩增加感染风险。',
    diet: '正常饮食，保持体力储备。分娩是马拉松——需要充足的能量。可以吃些助产食物（蜂蜜水、红牛）。不要因为焦虑而暴饮暴食。',
    taboo: '超过预产期不要自行尝试"催生偏方"。不要因为还没发动就过于焦虑——每个宝宝都有自己的时间表。',
    exercise: '散步是最安全有效的运动。爬楼梯（有人陪同）。分娩球运动。保持适度活动但充分休息——你需要储存体力迎接分娩。',
    emotional: '预产期到了宝宝还没来？不要焦虑！这完全正常。利用这段时间好好休息——这可能是很长时间内你最后的安稳觉了。和宝宝说话，告诉ta妈妈准备好了。',
    partner: '不要不断问"有感觉吗？"——这会增加伴侣的压力。帮她放松，一起做些轻松的活动。确保车里有油、手机有电、医院的路线熟记于心。'
  },
  {
    id: 'w41-42', weeks: [41, 42], title: '孕41周+：过了预产期', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-late-trimester/',
    development: '宝宝继续生长和发育。胎盘功能可能开始下降，胎儿需要密切监测。过了41周出生的宝宝往往体重偏大。羊水可能开始减少。',
    size: '可能超过50cm，体重可能达到3.5-4kg或更大。',
    symptoms: '和39-40周类似，但可能更加不适。焦虑情绪更强。假性宫缩和前驱宫缩交替出现。',
    checkups: '超过40周后通常每2-3天做一次胎心监护和B超（评估羊水）。41周时医生通常会讨论催产计划。催产方式包括：人工破膜、缩宫素点滴、前列腺素凝胶/球囊。',
    tips: '信任你的医生团队。催产不等于不好——在医学指征下催产是为了母婴安全。了解催产过程：可能需要12-24小时甚至更长。催产过程中可以使用无痛分娩。如果催产失败，医生可能建议剖宫产。',
    danger_signs: '过期妊娠（>42周）风险增加：胎盘功能不足、羊水过少、巨大儿、胎儿窘迫。所以医生通常不会让你等到42周。如果胎动减少、羊水偏少，可能需要提前干预。',
    diet: '保持正常饮食和充足水分。准备好分娩时的零食和饮料。',
    taboo: '不要拒绝医生建议的催产——过期妊娠的风险真实存在。不要听信不靠谱的催产方法。',
    exercise: '散步和爬楼梯可能有助于发动宫缩。保持活动但不要过度疲劳。',
    emotional: '超过预产期的等待是最煎熬的。每一个"宝宝来了吗？"的询问都可能让你崩溃。告诉亲友你会主动通知。相信宝宝和你的身体。深呼吸。',
    partner: '帮伴侣挡住外界的催促。给予最大的陪伴和安慰。准备好随时出发。在催产过程中全程陪伴。'
  },

  // ==================== 专题指南 ====================
  {
    id: 'guide-delivery', category: 'guide', title: '专题：分娩方式选择', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-delivery-postpartum/',
    sections: [
      { title: '顺产（阴道分娩）', content: '优点：产后恢复快（通常2-3天出院），出血少，宝宝经过产道挤压肺部更成熟、免疫力更好，更有利于母乳喂养的启动。\n\n过程：第一产程（宫颈扩张0-10cm）初产妇平均12-16小时；第二产程（推送宝宝出来）初产妇1-3小时；第三产程（娩出胎盘）5-30分钟。\n\n无痛分娩（硬膜外麻醉）：可以在宫口开到2-3cm时进行，有效缓解约90%的疼痛。不会影响宫缩和产程进展。对宝宝几乎无影响。强烈推荐！' },
      { title: '剖宫产', content: '医学指征包括：胎位不正（臀位/横位）、前置胎盘、胎儿窘迫、脐带脱垂、头盆不称、双胎/多胎（部分情况）、有子宫手术史等。\n\n过程：手术时间约30-60分钟。通常使用腰麻，妈妈是清醒的，可以第一时间看到宝宝。\n\n恢复：住院5-7天。术后24小时开始下床活动。伤口约6周愈合。再次怀孕建议间隔2年以上。' },
      { title: '如何选择？', content: '没有绝对的"好"或"不好"——适合你和宝宝的就是最好的。如果没有医学指征，鼓励尝试顺产。和你的产检医生充分沟通，了解你的具体情况。做好两手准备——计划顺产但也了解剖宫产，有些情况下可能需要临时转变方案。分娩计划不是"必须执行"的合同，灵活应对最重要。' }
    ]
  },
  {
    id: 'guide-breastfeeding', category: 'guide', title: '专题：母乳喂养指南', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-delivery-postpartum/',
    sections: [
      { title: '为什么推荐母乳？', content: 'WHO建议纯母乳喂养至6个月，之后添加辅食继续母乳至2岁或以上。母乳含有最适合婴儿的营养和免疫物质（抗体、白细胞、乳铁蛋白），随宝宝月龄自动调整成分。降低宝宝感染、过敏、肥胖、糖尿病等风险。帮助妈妈子宫恢复、降低乳腺癌和卵巢癌风险。最经济实惠——免费、随时可得、温度完美！' },
      { title: '成功母乳的关键', content: '产后30分钟内尽早吸吮（"黄金1小时"）。按需哺乳，不要定时定量——宝宝想吃就喂。正确的含乳姿势至关重要——宝宝要含住大部分乳晕而不仅是乳头，下巴贴住乳房，嘴巴张大。频繁吸吮刺激乳汁分泌——前几天的初乳虽少但非常珍贵。避免过早使用奶瓶和安抚奶嘴（至少等4-6周母乳建立后）。' },
      { title: '常见困难及处理', content: '乳头疼痛/皲裂：多数是含乳姿势不对。纠正含乳后涂抹母乳或羊脂膏在乳头上帮助愈合。\n\n涨奶：产后2-5天乳房可能非常涨痛——频繁哺乳+冷敷可缓解。\n\n堵奶/乳腺炎：硬块出现时频繁哺乳，从硬块方向向乳头按摩。发烧>38.5°C可能是乳腺炎，需就医可能需要抗生素。\n\n奶量不足的焦虑：大多数妈妈都能产生足够的奶。判断奶够不够看宝宝每天6-8片湿尿布、体重正常增长。' },
      { title: '职场妈妈的母乳喂养', content: '产假期间建立稳定的母乳供应。复工前2-3周开始练习用吸奶器和储存母乳。准备：双边电动吸奶器、母乳储存袋、便携冰包。上班期间每3-4小时吸奶一次。母乳室温可保存4小时，冰箱冷藏3-5天，冰箱冷冻6个月。' }
    ]
  },
  {
    id: 'guide-postpartum', category: 'guide', title: '专题：产后恢复与坐月子', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-delivery-postpartum/',
    sections: [
      { title: '产后身体恢复', content: '恶露：产后会有持续4-6周的阴道排出物——从鲜红色逐渐变为粉色再到黄白色。使用产妇专用卫生巾。如果恶露增多或有臭味需就医。\n\n子宫恢复：约6周恢复到孕前大小。母乳喂养促进子宫收缩。\n\n伤口护理：顺产侧切/撕裂——每次如厕后用清水冲洗，保持干燥。剖宫产伤口——保持干燥清洁，7天拆线（或使用可吸收线不需拆）。\n\n产后42天复查很重要——检查子宫恢复、伤口愈合、盆底功能。' },
      { title: '科学坐月子', content: '可以洗澡洗头！产后当天或第二天即可淋浴（剖宫产等伤口防水贴后）。用温水，洗完迅速擦干吹干。\n\n可以刷牙！用温水和软毛牙刷。\n\n可以适当走动！产后6-12小时建议下床活动，预防血栓。\n\n需要通风！但避免直吹。\n\n饮食均衡即可，不需要天天喝浓汤——油腻的汤反而可能堵奶。\n\n前2周以排恶露、恢复体力为主，之后逐渐增加活动量。' },
      { title: '产后情绪', content: '产后3-5天的"baby blues"很常见（约80%的新妈妈）——表现为莫名想哭、情绪低落、焦虑、易怒。通常1-2周内自行好转。\n\n产后抑郁（约10-15%）：症状持续超过2周且影响日常生活——持续悲伤、对宝宝缺乏兴趣、失眠或嗜睡、食欲变化、有伤害自己或宝宝的想法。这不是你的错，这是一种需要治疗的疾病。请立即寻求帮助！\n\n预防：充足睡眠（宝宝睡你也睡）、接受家人帮助、适当运动、保持社交、和伴侣沟通感受。' },
      { title: '产后运动恢复', content: '产后第1天：可以做凯格尔运动和深呼吸。\n\n产后1-2周：温和散步，逐渐增加距离。\n\n产后6周（复查通过后）：可以开始系统的产后运动——盆底肌修复训练、腹直肌分离修复。\n\n产后3-6个月：可以逐渐恢复孕前的运动强度。\n\n重点关注：盆底肌修复（预防漏尿和子宫脱垂）和腹直肌分离修复（不要做仰卧起坐！）。建议做专业的盆底康复评估。' }
    ]
  },
  {
    id: 'guide-newborn', category: 'guide', title: '专题：新生儿护理入门', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-delivery-postpartum/',
    sections: [
      { title: '基本照护', content: '喂奶：新生儿胃容量很小（第1天仅5-7ml），需要频繁哺乳（每天8-12次）。不要严格按时间表——按需喂养。判断吃饱：每天6-8片湿尿布，体重每周增长150-200克。\n\n睡眠：新生儿每天睡16-18小时，每次2-4小时（白天黑夜不分）。仰卧睡（降低SIDS风险）。婴儿床上不放枕头、被子、玩偶等松软物品。\n\n排便：出生1-2天排胎便（墨绿色黏稠），之后逐渐变为黄色糊状（母乳宝宝）或黄绿色（奶粉宝宝）。母乳宝宝每天排便0-8次都可能正常。' },
      { title: '脐带护理', content: '脐带残端通常在出生后1-3周自然脱落。保持干燥清洁即可。每天用碘伏棉签擦拭脐带根部一圈。纸尿裤折叠到脐带以下，避免摩擦和尿液浸湿。不要人为撕扯或拉拽脐带。如果脐部发红、有脓性分泌物或异味，需就医。' },
      { title: '洗澡', content: '脐带未脱落前用温水擦浴。脐带脱落后可以盆浴。水温37-38°C（用手肘测试，感觉温暖不烫）。先放冷水再加热水。一只手始终托住宝宝。洗澡时间不超过5-10分钟。不需要每天用沐浴露——清水即可，每周1-2次使用婴儿沐浴露。洗完后立即擦干，尤其是皮肤褶皱处。' },
      { title: '常见问题处理', content: '黄疸：约60%的足月儿和80%的早产儿会出现生理性黄疸（出生后2-3天出现，1-2周消退）。多吃多排促进胆红素排出。如果黄疸出现过早、程度过重或消退过晚，需就医检查并可能需要蓝光治疗。\n\n湿疹：保持皮肤滋润（每天多次涂抹保湿霜），穿纯棉衣物，避免过热。严重时需就医使用药膏。\n\n肠绞痛：通常在出生后2-3周开始，3个月左右好转。表现为每天固定时段无原因的剧烈哭闹。可以尝试：飞机抱、白噪音、轻轻摇晃、温暖肚子。\n\n吐奶/溢奶：非常常见！喂完奶拍嗝（竖抱轻拍背部5-10分钟），喂完后保持头高位15-20分钟。' },
      { title: '何时就医？', content: '新生儿出现以下情况需立即就医：体温>38°C或<36°C；持续不吃奶或吸吮无力；呼吸急促（>60次/分钟）或呼吸困难；皮肤发灰/发紫/发花；严重黄疸（皮肤和眼白明显变黄）；脐部红肿有脓；持续呕吐或腹泻；嗜睡难以唤醒；尖锐的高频哭声。新生儿生病进展很快，有疑问宁可多看一次医生。' }
    ]
  },
  {
    id: 'guide-diet-taboo', category: 'guide', title: '专题：孕期饮食完全指南', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-diet-checklist/',
    sections: [
      { title: '必须补充的关键营养素', content: '叶酸（0.4-0.8mg/天）：孕前3个月至孕12周必须补充，预防神经管缺陷。食物来源：菠菜、西兰花、豆类、柑橘。\n\n铁（27mg/天）：孕中晚期需求大增。食物来源：红肉、动物肝脏、菠菜、黑木耳。配合维C促进吸收。\n\n钙（1000-1200mg/天）：骨骼和牙齿发育必需。食物来源：牛奶、酸奶、豆腐、小鱼干、芝麻。\n\nDHA（200-300mg/天）：大脑和视网膜发育关键。食物来源：三文鱼、沙丁鱼、鳕鱼；或直接补充藻油DHA。\n\n碘（230μg/天）：甲状腺功能和宝宝大脑发育。食物来源：碘盐、海带、紫菜、虾。\n\n蛋白质（孕中晚期增加25g/天）：食物来源：鸡蛋、鱼、肉、豆制品、奶制品。' },
      { title: '绝对禁止的食物', content: '酒精：任何剂量都不安全，可能导致胎儿酒精谱系障碍（面部异常、智力障碍、行为问题）。包括料酒做的菜如果酒精未完全挥发也要小心。\n\n生鱼片/生肉/生蛋：李斯特菌、沙门氏菌、寄生虫感染风险。寿司、溏心蛋、三分熟牛排都不行。\n\n未消毒乳制品：软奶酪（布里、卡门贝尔）、未经巴氏杀菌的牛奶。\n\n高汞鱼类：鲨鱼、旗鱼、方头鱼、大眼金枪鱼。\n\n药物：未经医生许可的任何药物，包括中草药。' },
      { title: '需要限制的食物', content: '咖啡因：每天不超过200mg（约1杯中杯咖啡）。注意茶、可乐、巧克力也含咖啡因。\n\n糖分：过多摄入增加妊娠期糖尿病风险。少喝奶茶、果汁、碳酸饮料。\n\n盐分：过多摄入加重水肿，增加高血压风险。少吃腌制食品、方便面、膨化食品。\n\n加工食品：含防腐剂、色素、反式脂肪酸。尽量吃新鲜食物。\n\n部分中药食材：桂圆（性热可能出血）、山楂（刺激宫缩）、薏米（可能引起宫缩）、藏红花（活血）。' },
      { title: '每日膳食参考', content: '早餐：全麦面包+鸡蛋+牛奶+水果。\n加餐：一把坚果+酸奶。\n午餐：糙米饭+鱼/肉+两种蔬菜+豆腐汤。\n加餐：水果+全麦饼干。\n晚餐：杂粮饭+瘦肉+蔬菜+紫菜蛋花汤。\n睡前：一杯温牛奶。\n\n每日饮水：2000-2500ml。\n水果：200-400g/天（不要过量，水果含糖高）。\n蔬菜：400-500g/天（深色蔬菜占一半以上）。' }
    ]
  },
  {
    id: 'guide-emotion', category: 'guide', title: '专题：孕期心理健康', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-diet-checklist/',
    sections: [
      { title: '孕期常见情绪变化', content: '孕期激素剧烈波动（HCG、雌激素、孕酮）会直接影响情绪调节。这不是"矫情"，而是生理性的变化。\n\n孕早期：焦虑（担心流产和胎儿健康）、孕吐导致的沮丧、对身体变化的不安。\n\n孕中期：情绪相对平稳，可能有"孕傻"（记忆力下降是正常的）。\n\n孕晚期：对分娩的恐惧、对能否胜任母亲角色的担忧、对产后生活的焦虑、身体不适导致的烦躁。' },
      { title: '如何应对焦虑', content: '允许自己有负面情绪——不要因为"应该高兴"而压抑真实感受。找人倾诉——伴侣、家人、朋友、孕妈群、心理咨询师。减少信息焦虑——不要过度百度症状，信任你的医生。保持适度运动——散步、瑜伽、游泳都能改善情绪。练习冥想和正念——每天10分钟的呼吸练习。保持社交——不要因为怀孕就把自己关在家里。做让自己开心的事——看剧、听音乐、和朋友吃饭、逛街。' },
      { title: '孕期抑郁', content: '约10-20%的孕妈会经历孕期抑郁，而不是只有产后才会抑郁。症状包括：持续两周以上的悲伤或空虚感、对日常活动失去兴趣、失眠或嗜睡、食欲明显变化、注意力无法集中、反复想到死亡或自杀。\n\n如果你有这些感受，请一定寻求帮助。孕期抑郁是可以治疗的——心理咨询和部分抗抑郁药物在孕期是安全的。这不是你的错，这不代表你不爱宝宝。寻求帮助是最勇敢的选择。' },
      { title: '给准爸爸的心理指南', content: '你可能也会经历焦虑——对经济的担忧、对角色转变的不安、对分娩的紧张。这些都是正常的。\n\n你能做的最重要的事：倾听，不评判。她说"好累好难受"时，不要说"别人不也这样过来了"，而是说"我知道你很辛苦，我能做什么帮到你？"\n\n参与孕期的每个环节——产检、购物、布置婴儿房。让她感到你们是一个团队。\n\n如果发现伴侣有抑郁倾向，温柔但坚定地建议寻求专业帮助。' }
    ]
  },
  {
    id: 'guide-checklist', category: 'guide', title: '专题：孕期办事清单', blogUrl: 'https://blog.jaxiu.cn/blog/2026-02/pregnancy-diet-checklist/',
    sections: [
      { title: '孕早期（1-12周）', content: '确认怀孕（验孕棒+医院抽血确认）。\n选择产检医院并预约第一次产检。\n开始服用叶酸和孕期维生素。\n办理母子健康手册（社区卫生服务中心）。\n了解单位产假政策和生育保险待遇。\n通知直属领导（可以等NT检查通过后）。\n购买孕期必备品：叶酸、孕妇维生素、宽松衣物。\n如有宠物，安排他人清理猫砂。\n家中装修应暂停或确保通风。' },
      { title: '孕中期（13-28周）', content: '按时做唐筛/无创DNA/大排畸等关键检查。\n购买孕妇装、宽松鞋、孕妇枕。\n开始涂抹妊娠纹预防产品。\n选择分娩医院（对比普通医院和妇产专科医院）。\n参加孕妇学校/分娩准备课程。\n开始准备待产包清单。\n开始选购婴儿大件（婴儿床、推车、安全座椅）。\n考虑是否需要月嫂，提前预约。\n布置婴儿房或婴儿角。\n安排"babymoon"旅行（24-28周最佳）。\n拍孕期纪念照。' },
      { title: '孕晚期（29-40周）', content: '完成待产包准备并放在固定位置。\n确认医院入院流程和路线。\n学习分娩知识：呼吸法、分娩流程、无痛分娩。\n和医生讨论分娩计划。\n预约月嫂/月子中心。\n购齐新生儿必备品。\n安装婴儿安全座椅。\n准备入院证件：身份证、医保卡、母子手册、产检资料。\n办理生育保险相关手续。\n了解出生证明、户口办理流程。\n洗涤消毒新生儿衣物和用品。\n做好工作交接。\n手机存好医院/医生/120紧急电话。' },
      { title: '产后办事', content: '办理出生医学证明（出生后7天内在医院办理）。\n办理新生儿户口（出生后30天内到户籍所在地派出所）。\n办理新生儿医保（出生后3个月内办理可追溯报销出生费用）。\n申领生育保险待遇。\n新生儿42天体检。\n妈妈42天产后复查。\n办理独生子女证（如适用）。\n更新家庭保险计划。' }
    ]
  }
]

function isInWeekRange(weeks) {
  const w = currentWeek.value
  return w >= weeks[0] && w <= weeks[1]
}

// API calls
async function createProfile() {
  if (!createForm.edd || !createForm.password) {
    ElMessage.warning('请填写预产期和密码')
    return
  }
  if (createForm.password.length < 4) {
    ElMessage.warning('密码至少需要4个字符')
    return
  }
  creating.value = true
  try {
    const res = await fetch(`${API_BASE}/api/pregnancy`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ edd: createForm.edd, password: createForm.password })
    })
    const data = await res.json()
    if (res.status === 409) {
      ElMessage.warning('该密码已被使用，请直接加载档案')
      loadForm.password = createForm.password
      return
    }
    if (!res.ok) throw new Error(data.error)

    profileId.value = data.id
    creatorKey.value = data.creator_key
    profileEDD.value = createForm.edd
    saveLocalProfile(data.id, data.creator_key)

    // Load the full profile data from server
    await loadByCreatorKey(data.id, data.creator_key)
    ElMessage.success('档案创建成功')
  } catch (e) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    creating.value = false
  }
}

async function loadProfile() {
  if (!loadForm.password) {
    ElMessage.warning('请输入密码')
    return
  }
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/api/pregnancy/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: loadForm.password })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error)

    profileId.value = data.id
    profileEDD.value = data.edd
    applyData(data.data)
    ElMessage.success('档案加载成功')
  } catch (e) {
    ElMessage.error(e.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadByCreatorKey(id, key) {
  try {
    const res = await fetch(`${API_BASE}/api/pregnancy/${id}/creator?creator_key=${encodeURIComponent(key)}`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error)

    profileId.value = id
    creatorKey.value = key
    profileEDD.value = data.edd
    applyData(data.data)
    showMyProfiles.value = false
    ElMessage.success('档案加载成功')
  } catch (e) {
    ElMessage.error(e.message || '加载失败')
  }
}

function applyData(data) {
  if (!data) return
  const d = typeof data === 'string' ? JSON.parse(data) : data
  if (d.hospital_bag) Object.assign(profileData.hospital_bag, d.hospital_bag)
  if (d.baby_essentials) Object.assign(profileData.baby_essentials, d.baby_essentials)
  if (d.prenatal_checks) profileData.prenatal_checks = d.prenatal_checks
  if (d.weight_records) profileData.weight_records = d.weight_records
  if (d.fetal_movements) profileData.fetal_movements = d.fetal_movements
}

async function saveData() {
  if (!profileId.value || !creatorKey.value) return
  try {
    const res = await fetch(`${API_BASE}/api/pregnancy/${profileId.value}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        action: 'update_data',
        creator_key: creatorKey.value,
        data: {
          hospital_bag: profileData.hospital_bag,
          baby_essentials: profileData.baby_essentials,
          prenatal_checks: profileData.prenatal_checks,
          weight_records: profileData.weight_records,
          fetal_movements: profileData.fetal_movements
        }
      })
    })
    if (!res.ok) {
      const data = await res.json()
      throw new Error(data.error)
    }
    saveStatus.value = 'saved'
  } catch (e) {
    ElMessage.error('保存失败: ' + (e.message || '未知错误'))
    saveStatus.value = 'error'
  }
}

async function extendProfile() {
  if (!profileId.value || !creatorKey.value) return
  extending.value = true
  try {
    const res = await fetch(`${API_BASE}/api/pregnancy/${profileId.value}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        action: 'extend',
        creator_key: creatorKey.value,
        expires_in: extendDays.value
      })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error)
    showExtendDialog.value = false
    ElMessage.success('延期成功')
  } catch (e) {
    ElMessage.error(e.message || '延期失败')
  } finally {
    extending.value = false
  }
}

async function confirmDelete() {
  try {
    await ElMessageBox.confirm('确定要删除该档案吗？此操作不可恢复。', '删除确认', {
      type: 'warning'
    })
    const res = await fetch(
      `${API_BASE}/api/pregnancy/${profileId.value}?creator_key=${encodeURIComponent(creatorKey.value)}`,
      { method: 'DELETE' }
    )
    if (!res.ok) {
      const data = await res.json()
      throw new Error(data.error)
    }
    removeLocalProfile(profileId.value)
    profileId.value = ''
    creatorKey.value = ''
    profileEDD.value = ''
    ElMessage.success('档案已删除')
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e.message || '删除失败')
  }
}

// Lifecycle
onMounted(() => {
  loadLocalProfiles()
  // Auto-load first local profile
  const profiles = localProfiles.value
  const ids = Object.keys(profiles)
  if (ids.length > 0) {
    loadByCreatorKey(ids[0], profiles[ids[0]])
  }

  // Auto-expand current week knowledge
  const currentIdx = knowledgeData.findIndex(info => isInWeekRange(info.weeks))
  if (currentIdx >= 0) knowledgeCollapse.value = [currentIdx]
})

onUnmounted(() => {
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  if (fetalInterval) clearInterval(fetalInterval)
})
</script>

<style scoped>
.tool-container {
  gap: 20px;
}

.welcome-section {
  max-width: 600px;
  margin: 0 auto;
}

.welcome-card {
  background-color: var(--card-bg);
  border: 1px solid var(--card-border);
  padding: 30px;
  border-radius: var(--radius-lg);
  text-align: center;
}

.welcome-card h3 {
  margin: 0 0 10px 0;
  color: var(--text-primary);
}

.welcome-card p {
  color: var(--text-secondary);
  margin-bottom: 20px;
}

/* Overview banner */
.overview-banner {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 50%, #fda085 100%);
  padding: 25px;
  border-radius: var(--radius-lg);
  color: #fff;
}

html.dark .overview-banner {
  background: linear-gradient(135deg, #5b2a6e 0%, #8b2252 50%, #a0522d 100%);
}

.pregnancy-info {
  display: flex;
  align-items: center;
  gap: 30px;
  flex-wrap: wrap;
}

.week-display {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.week-number {
  font-size: 48px;
  font-weight: 700;
  line-height: 1;
}

.week-label {
  font-size: 18px;
  opacity: 0.9;
}

.info-details {
  display: flex;
  gap: 25px;
  flex-wrap: wrap;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-item .label {
  font-size: 12px;
  opacity: 0.8;
}

.info-item .value {
  font-size: 16px;
  font-weight: 600;
}

.progress-bar-container {
  margin-top: 15px;
}

.progress-bar-container :deep(.el-progress__text) {
  color: #fff;
}

/* Checklist section */
.checklist-section .section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.checklist-section .section-header h3 {
  margin: 0;
  color: var(--text-primary);
}

.collapse-title {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}

.checklist-items {
  padding: 5px 0;
}

.checklist-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
}

.add-item {
  margin-top: 10px;
  max-width: 400px;
}

/* Prenatal checks */
.prenatal-section h3 {
  margin: 0 0 20px 0;
  color: var(--text-primary);
}

.check-card {
  background-color: var(--bg-secondary);
  padding: 12px 15px;
  border-radius: var(--radius-sm);
  border-left: 3px solid var(--border-light);
}

.check-card.done {
  border-left-color: var(--color-success);
  opacity: 0.8;
}

.check-card.current {
  border-left-color: var(--color-primary);
  background-color: var(--card-bg);
}

.check-header {
  margin-bottom: 5px;
}

.check-items {
  color: var(--text-secondary);
  font-size: 13px;
  margin: 5px 0 8px 24px;
}

/* Weight section */
.weight-section h3 {
  margin: 0 0 15px 0;
  color: var(--text-primary);
}

.weight-input {
  margin-bottom: 20px;
}

.weight-chart {
  background-color: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: var(--radius-md);
  padding: 15px;
  overflow-x: auto;
}

.chart-svg {
  width: 100%;
  max-width: 600px;
  height: auto;
}

/* Fetal movement */
.fetal-section h3 {
  margin: 0 0 20px 0;
  color: var(--text-primary);
}

.counter-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  padding: 30px;
  background-color: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: var(--radius-lg);
}

.counter-display {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.counter-number {
  font-size: 64px;
  font-weight: 700;
  color: var(--color-primary);
  font-family: var(--font-family-mono);
}

.counter-label {
  font-size: 24px;
  color: var(--text-secondary);
}

.counter-timer {
  font-size: 20px;
  color: var(--text-secondary);
  font-family: var(--font-family-mono);
}

.kick-button {
  width: 80px !important;
  height: 80px !important;
}

.counter-controls {
  display: flex;
  gap: 10px;
}

/* Knowledge section */
.knowledge-section h3 {
  margin: 0 0 15px 0;
  color: var(--text-primary);
}

.knowledge-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.current-week-knowledge :deep(.el-collapse-item__header) {
  font-weight: 600;
}

.knowledge-content {
  padding: 5px 0;
}

.knowledge-item {
  margin-bottom: 12px;
}

.knowledge-item h4 {
  margin: 0 0 5px 0;
  color: var(--color-primary);
  font-size: 14px;
}

.knowledge-item p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.8;
  white-space: pre-line;
}

.knowledge-danger h4 {
  color: var(--el-color-danger) !important;
}

.knowledge-warning h4 {
  color: var(--el-color-warning) !important;
}

.knowledge-sections {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.knowledge-blog-link {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px dashed var(--border-light);
  text-align: right;
}

.knowledge-blog-link a {
  text-decoration: none;
}

/* Profile list */
.profile-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid var(--border-light);
}

.profile-item:last-child {
  border-bottom: none;
}

/* Responsive */
@media (max-width: 768px) {
  .pregnancy-info {
    flex-direction: column;
    align-items: flex-start;
    gap: 15px;
  }

  .info-details {
    gap: 15px;
  }

  .actions {
    flex-wrap: wrap;
  }

  .week-number {
    font-size: 36px;
  }
}
</style>
