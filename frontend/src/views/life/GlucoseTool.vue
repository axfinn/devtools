<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>血糖检测</h2>
      <el-button v-if="profileId" type="danger" size="small" @click="logout">退出</el-button>
    </div>

    <!-- 未登录 -->
    <div v-if="!profileId" class="login-section">
      <el-card>
        <h3>血糖档案</h3>
        <el-input v-model="loadForm.password" type="password" placeholder="输入密码登录" show-password class="mobile-input" />
        <el-button type="primary" size="large" class="login-btn" @click="loadProfile" :loading="loading">登录</el-button>
        <div class="create-link">
          <el-button link type="primary" @click="showCreate = true">创建新档案</el-button>
        </div>
      </el-card>

      <!-- 创建档案对话框 -->
      <el-dialog v-model="showCreate" title="创建档案" width="90%">
        <el-input v-model="createForm.password" type="password" placeholder="设置密码（至少4位）" show-password class="mobile-input" />
        <template #footer>
          <el-button @click="showCreate = false">取消</el-button>
          <el-button type="primary" @click="createProfile" :loading="creating">创建</el-button>
        </template>
      </el-dialog>
    </div>

    <!-- 已登录 -->
    <div v-else class="main-content">
      <!-- 语音输入 -->
      <div class="voice-section">
        <el-button
          v-if="!isRecording"
          type="primary"
          size="large"
          class="voice-btn"
          @click="toggleVoice"
        >
          <el-icon><Microphone /></el-icon>
          语音输入
        </el-button>
        <div v-else class="recording-box">
          <div class="recording-pulse"></div>
          <span>正在录音...</span>
          <el-button type="danger" @click="stopVoice">结束</el-button>
        </div>
        <div v-if="voiceText" class="voice-result">
          <span>{{ voiceText }}</span>
          <el-button type="success" size="small" @click="smartParseAI">解析</el-button>
        </div>
      </div>

      <!-- 快速输入 -->
      <el-card class="quick-card">
        <div class="quick-row">
          <span class="label">血糖值 (mmol/L)</span>
          <el-input-number v-model="recordForm.value" :min="1" :max="50" :step="0.1" :precision="1" size="large" class="glucose-input" />
        </div>

        <div class="quick-row">
          <span class="label">类型</span>
          <div class="quick-btns">
            <el-button
              v-for="t in typeOptions"
              :key="t.value"
              :type="recordForm.measureType === t.value ? 'primary' : 'default'"
              size="small"
              round
              @click="recordForm.measureType = t.value"
            >{{ t.label }}</el-button>
          </div>
        </div>

        <div class="quick-row">
          <span class="label">时间</span>
          <div class="quick-btns">
            <el-button size="small" round @click="recordForm.time = new Date()">现在</el-button>
            <el-button size="small" round @click="recordForm.time = new Date(Date.now() - 3600000)">1小时前</el-button>
            <el-button size="small" round @click="recordForm.time = new Date(Date.now() - 7200000)">2小时前</el-button>
          </div>
        </div>

        <div class="quick-row">
          <span class="label">餐饮 (可选)</span>
          <el-input v-model="recordForm.food" placeholder="如：米饭、面条、水果等" />
        </div>

        <div class="quick-row">
          <span class="label">备注 (可选)</span>
          <el-input v-model="recordForm.note" placeholder="其他情况" />
        </div>
      </el-card>

      <!-- 添加按钮 -->
      <el-button type="primary" size="large" class="add-btn" @click="addRecord" :loading="saving">
        <el-icon><Plus /></el-icon>
        添加记录
      </el-button>

      <!-- 统计 -->
      <div class="stats-grid">
        <div class="stat-box">
          <div class="stat-num">{{ (stats.avg_value || 0).toFixed(1) }}</div>
          <div class="stat-label">平均</div>
        </div>
        <div class="stat-box" :class="getTIRClass(stats.tir)">
          <div class="stat-num">{{ (stats.tir || 0).toFixed(0) }}%</div>
          <div class="stat-label">达标率</div>
        </div>
        <div class="stat-box danger">
          <div class="stat-num">{{ stats.max_value || 0 }}</div>
          <div class="stat-label">最高</div>
        </div>
        <div class="stat-box warning">
          <div class="stat-num">{{ stats.min_value || 0 }}</div>
          <div class="stat-label">最低</div>
        </div>
      </div>

      <!-- 图表 -->
      <el-card class="chart-card">
        <div ref="chartRef" style="height: 200px;"></div>
      </el-card>

      <!-- 记录列表 -->
      <el-card class="list-card">
        <template #header>
          <div class="list-header">
            <span>记录 ({{ records.length }})</span>
            <div class="list-actions">
              <el-button size="small" circle @click="showImport = true" title="导入">
                <el-icon><Upload /></el-icon>
              </el-button>
              <el-button size="small" circle @click="exportData" title="导出">
                <el-icon><Download /></el-icon>
              </el-button>
            </div>
          </div>
        </template>
        <div class="record-list">
          <div v-for="row in records.slice(0, 20)" :key="row.id" class="record-item">
            <div class="record-main">
              <span class="record-time">{{ formatTime(row.time) }}</span>
              <span class="record-value" :class="getGlucoseClass(row.value)">{{ row.value }}</span>
              <span class="record-type">{{ getTypeLabel(row.measure_type) }}</span>
            </div>
            <div v-if="row.food || row.note || row.voice_text" class="record-extra">
              <span v-if="row.voice_text" class="voice-text">🎤 {{ row.voice_text }}</span>
              <span v-if="row.food">{{ row.food }}</span>
              <span v-if="row.note">{{ row.note }}</span>
            </div>
            <div class="record-actions">
              <el-button size="small" text type="info" @click="viewHistory(row)">历</el-button>
              <el-button size="small" text type="primary" @click="openEdit(row)">改</el-button>
              <el-button size="small" text type="danger" @click="deleteRecord(row.id)">删</el-button>
            </div>
          </div>
          <div v-if="records.length === 0" class="empty-tip">暂无记录</div>
        </div>
      </el-card>

      <!-- 编辑对话框 -->
      <el-dialog v-model="showEdit" title="修改记录" width="90%">
        <el-form label-width="60px">
          <el-form-item label="血糖值">
            <el-input-number v-model="editForm.value" :min="1" :max="50" :step="0.1" />
          </el-form-item>
          <el-form-item label="类型">
            <el-select v-model="editForm.measureType" style="width: 100%;">
              <el-option v-for="t in typeOptions" :key="t.value" :label="t.label" :value="t.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="时间">
            <el-date-picker v-model="editForm.time" type="datetime" style="width: 100%;" />
          </el-form-item>
          <el-form-item label="餐饮">
            <el-input v-model="editForm.food" placeholder="如：米饭、面条等" />
          </el-form-item>
          <el-form-item label="备注">
            <el-input v-model="editForm.note" placeholder="其他情况" />
          </el-form-item>
          <el-form-item>
            <el-button type="info" size="small" @click="editingId = row.id; showHistory = true; loadHistory()">查看历史</el-button>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showEdit = false">取消</el-button>
          <el-button type="primary" @click="saveEdit">保存</el-button>
        </template>
      </el-dialog>

      <!-- 历史对话框 -->
      <el-dialog v-model="showHistory" title="变更历史" width="90%">
        <div v-if="historyLoading" class="loading-tip">加载中...</div>
        <div v-else>
          <div v-for="h in recordHistory" :key="h.id" class="history-item">
            <el-tag :type="h.action === 'create' ? 'success' : h.action === 'update' ? 'warning' : 'danger'" size="small">
              {{ h.action === 'create' ? '创建' : h.action === 'update' ? '修改' : '删除' }}
            </el-tag>
            <span>{{ h.change_desc }}</span>
            <span class="history-time">{{ formatTime(h.created_at) }}</span>
          </div>
          <div v-if="recordHistory.length === 0" class="empty-tip">暂无历史</div>
        </div>
      </el-dialog>

      <!-- 导入对话框 -->
      <el-dialog v-model="showImport" title="导入CSV" width="90%">
        <p style="font-size: 12px; color: #666;">格式：时间,血糖值,类型,餐饮,备注</p>
        <p style="font-size: 12px; color: #666;">例如：2024-01-15 08:00,6.5,fasting,米饭,</p>
        <el-input type="textarea" v-model="importText" :rows="6" placeholder="粘贴CSV数据..." />
        <template #footer>
          <el-button @click="showImport = false">取消</el-button>
          <el-button type="primary" @click="doImport" :loading="importing">导入</el-button>
        </template>
      </el-dialog>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Download, Microphone, Upload } from '@element-plus/icons-vue'
import { getECharts } from '../../utils/vendor-loaders'

const API_BASE = '/api'

// 状态
const profileId = ref(localStorage.getItem('glucose_profile_id') || '')
const password = ref(localStorage.getItem('glucose_password') || '')
const loadForm = ref({ password: '' })
const createForm = ref({ password: '' })
const loading = ref(false)
const creating = ref(false)
const saving = ref(false)
const showCreate = ref(false)

// 记录
const records = ref([])
const stats = ref({})
const chartRef = ref(null)
let chartInstance = null
let recognition = null
const isRecording = ref(false)
const voiceText = ref('')

// 表单
const recordForm = ref({
  value: null,
  time: new Date(),
  measureType: 'fasting',
  food: '',
  note: '',
  voiceText: ''
})

const typeOptions = [
  { label: '空腹', value: 'fasting' },
  { label: '餐后', value: '2h' },
  { label: '随机', value: 'random' },
  { label: '睡前', value: 'bedtime' }
]

// 编辑
const showEdit = ref(false)
const showHistory = ref(false)
const showImport = ref(false)
const importText = ref('')
const importing = ref(false)
const editForm = ref({
  value: null,
  measureType: 'fasting',
  time: new Date(),
  food: '',
  note: ''
})
const editingId = ref('')
const recordHistory = ref([])
const historyLoading = ref(false)

// 方法
const getTypeLabel = (type) => {
  const map = { fasting: '空腹', '1h': '1h', '2h': '2h', random: '随机', bedtime: '睡前', dawn: '凌晨' }
  return map[type] || type
}

const getGlucoseClass = (value) => {
  if (value < 3.9) return 'danger'
  if (value < 6.1) return 'success'
  if (value < 7.8) return 'warning'
  return 'danger'
}

const getTIRClass = (tir) => {
  if (!tir) return ''
  if (tir >= 70) return 'success'
  if (tir >= 50) return 'warning'
  return 'danger'
}

const formatTime = (time) => {
  const d = new Date(time)
  return `${d.getMonth()+1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(2,'0')}`
}

// API
async function apiRequest(url, options = {}) {
  const headers = { 'Content-Type': 'application/json' }
  if (password.value) headers['X-Password'] = password.value

  const res = await fetch(url, { ...options, headers })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || '请求失败')
  }
  return res.json()
}

async function createProfile() {
  if (createForm.value.password.length < 4) {
    ElMessage.warning('密码至少4位')
    return
  }
  creating.value = true
  try {
    const res = await apiRequest(`${API_BASE}/glucose`, {
      method: 'POST',
      body: JSON.stringify({ password: createForm.value.password })
    })
    profileId.value = res.id
    password.value = createForm.value.password
    localStorage.setItem('glucose_profile_id', res.id)
    localStorage.setItem('glucose_password', res.password)
    localStorage.setItem('glucose_creator_key', res.creator_key)
    showCreate.value = false
    await loadData()
    ElMessage.success('创建成功')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    creating.value = false
  }
}

async function loadProfile() {
  if (!loadForm.value.password) {
    ElMessage.warning('请输入密码')
    return
  }
  loading.value = true
  try {
    const res = await apiRequest(`${API_BASE}/glucose/login`, {
      method: 'POST',
      body: JSON.stringify({ password: loadForm.value.password })
    })
    profileId.value = res.id
    password.value = loadForm.value.password
    localStorage.setItem('glucose_profile_id', res.id)
    localStorage.setItem('glucose_password', loadForm.value.password)
    await loadData()
    ElMessage.success('登录成功')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function logout() {
  localStorage.removeItem('glucose_profile_id')
  localStorage.removeItem('glucose_password')
  profileId.value = ''
  password.value = ''
  records.value = []
  stats.value = {}
}

async function loadData() {
  await Promise.all([loadRecords(), loadStats()])
  await nextTick()
  await initChart()
}

async function loadRecords() {
  if (!profileId.value) return
  try {
    records.value = await apiRequest(`${API_BASE}/glucose/${profileId.value}/records`)
  } catch (e) {
    console.error(e)
  }
}

async function loadStats() {
  if (!profileId.value) return
  try {
    stats.value = await apiRequest(`${API_BASE}/glucose/${profileId.value}/stats`)
  } catch (e) {
    console.error(e)
  }
}

async function addRecord(voiceInputText = '') {
  if (!recordForm.value.value) {
    ElMessage.warning('请输入血糖值')
    return
  }
  saving.value = true
  try {
    await apiRequest(`${API_BASE}/glucose/${profileId.value}/records`, {
      method: 'POST',
      body: JSON.stringify({
        value: recordForm.value.value,
        measure_type: recordForm.value.measureType,
        time: recordForm.value.time.toISOString(),
        food: recordForm.value.food,
        note: recordForm.value.note,
        voice_text: voiceInputText
      })
    })
    recordForm.value.value = null
    recordForm.value.time = new Date()
    recordForm.value.food = ''
    recordForm.value.note = ''
    voiceText.value = ''
    await loadData()
    ElMessage.success('添加成功')
  } catch (e) {
    console.error('添加记录失败:', e)
    ElMessage.error('添加失败: ' + e.message)
  } finally {
    saving.value = false
  }
}

async function deleteRecord(id) {
  try {
    await ElMessageBox.confirm('确定删除?', '提示', { type: 'warning' })
    await apiRequest(`${API_BASE}/glucose/${profileId.value}/records/${id}`, { method: 'DELETE' })
    await loadData()
    ElMessage.success('删除成功')
  } catch (e) {
    if (e.message !== 'cancel') ElMessage.error(e.message)
  }
}

function openEdit(row) {
  editingId.value = row.id
  editForm.value = {
    value: row.value,
    measureType: row.measure_type,
    time: new Date(row.time),
    food: row.food || '',
    note: row.note || ''
  }
  showEdit.value = true
}

async function loadHistory() {
  if (!editingId.value) return
  historyLoading.value = true
  try {
    recordHistory.value = await apiRequest(`${API_BASE}/glucose/${profileId.value}/records/${editingId.value}/history`)
  } catch (e) {
    ElMessage.error('加载历史失败')
  } finally {
    historyLoading.value = false
  }
}

function viewHistory(row) {
  editingId.value = row.id
  recordHistory.value = []
  showHistory.value = true
  loadHistory()
}

async function saveEdit() {
  try {
    await apiRequest(`${API_BASE}/glucose/${profileId.value}/records/${editingId.value}`, {
      method: 'PUT',
      body: JSON.stringify({
        value: editForm.value.value,
        measure_type: editForm.value.measureType,
        time: editForm.value.time.toISOString(),
        food: editForm.value.food,
        note: editForm.value.note
      })
    })
    showEdit.value = false
    await loadData()
    ElMessage.success('保存成功')
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function doImport() {
  if (!importText.value.trim()) {
    ElMessage.warning('请输入数据')
    return
  }
  importing.value = true
  try {
    const lines = importText.value.trim().split('\n')
    const records = []

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i].trim()
      if (!line) continue
      if (i === 0 && line.includes('时间')) continue // 跳过表头

      const parts = line.split(',')
      if (parts.length < 2) continue

      const record = {
        value: parseFloat(parts[1]),
        measure_type: parts[2]?.trim() || 'random',
        time: parts[0] ? new Date(parts[0]).toISOString() : new Date().toISOString(),
        food: parts[3]?.trim() || '',
        note: parts[4]?.trim() || ''
      }

      if (record.value >= 1 && record.value <= 50) {
        records.push(record)
      }
    }

    if (records.length === 0) {
      ElMessage.warning('没有有效数据')
      return
    }

    // 调用批量导入API
    await apiRequest(`${API_BASE}/glucose/${profileId.value}/import`, {
      method: 'POST',
      body: JSON.stringify({ records })
    })

    showImport.value = false
    importText.value = ''
    await loadData()
    ElMessage.success(`导入成功 ${records.length} 条`)
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    importing.value = false
  }
}

function exportData() {
  if (records.value.length === 0) {
    ElMessage.warning('暂无记录')
    return
  }
  const headers = ['时间', '血糖值', '类型']
  const rows = records.value.map(r => [
    new Date(r.time).toLocaleString('zh-CN'),
    r.value,
    getTypeLabel(r.measure_type)
  ])
  const csv = [headers, ...rows].map(r => r.join(',')).join('\n')
  const blob = new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `血糖记录_${new Date().toISOString().split('T')[0]}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

// 语音
function toggleVoice() {
  try {
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition
    if (!SpeechRecognition) {
      ElMessage.error('浏览器不支持语音识别，请使用Chrome')
      return
    }
    recognition = new SpeechRecognition()
    recognition.lang = 'zh-CN'
    recognition.continuous = false
    recognition.interimResults = true

    recognition.onstart = () => {
      isRecording.value = true
      voiceText.value = ''
      ElMessage.success('请说话')
    }

    recognition.onresult = (event) => {
      let transcript = ''
      for (let i = 0; i < event.results.length; i++) {
        transcript += event.results[i][0].transcript
      }
      if (transcript) {
        voiceText.value = transcript
      }
    }

    recognition.onerror = (event) => {
      isRecording.value = false
      ElMessage.error('语音识别出错: ' + event.error)
    }

    recognition.onend = () => {
      isRecording.value = false
      if (voiceText.value) {
        ElMessage.success('识别完成: ' + voiceText.value)
      }
    }

    recognition.start()
  } catch (e) {
    console.error('语音识别异常:', e)
    ElMessage.error('语音识别失败: ' + e.message)
  }
}

function stopVoice() {
  if (recognition) {
    recognition.stop()
  }
  isRecording.value = false
}

async function smartParseAI() {
  if (!voiceText.value) return
  ElMessage.info('正在解析...')
  try {
    const res = await apiRequest(`${API_BASE}/glucose/${profileId.value}/voice-parse`, {
      method: 'POST',
      body: JSON.stringify({ text: voiceText.value })
    })
    const value = parseFloat(res.value)
    if (value > 0) {
      // 直接调用API添加记录
      saving.value = true
      const voiceInput = voiceText.value
      await apiRequest(`${API_BASE}/glucose/${profileId.value}/records`, {
        method: 'POST',
        body: JSON.stringify({
          value: value,
          measure_type: res.measure_type || 'random',
          time: new Date().toISOString(),
          food: res.food || '',
          note: '',
          voice_text: voiceInput
        })
      })
      voiceText.value = ''
      await loadData()
      saving.value = false
      ElMessage.success('添加成功: ' + value + ' mmol/L')
    } else {
      ElMessage.warning('未识别到血糖值')
    }
  } catch (e) {
    console.error('失败:', e)
    saving.value = false
    ElMessage.error('失败: ' + e.message)
  }
}

// 图表
async function initChart() {
  if (!chartRef.value || records.value.length === 0) return
  if (chartInstance) chartInstance.dispose()
  const echarts = await getECharts()

  const data = records.value.slice(0, 30).reverse().map(r => ({
    time: new Date(r.time),
    value: r.value,
    type: r.measure_type
  }))

  chartInstance = echarts.init(chartRef.value)
  chartInstance.setOption({
    grid: { left: 40, right: 20, top: 20, bottom: 30 },
    xAxis: {
      type: 'time',
      axisLabel: { fontSize: 10 }
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: 20,
      axisLabel: { fontSize: 10 }
    },
    visualMap: {
      show: false,
      pieces: [
        { lt: 3.9, color: '#f56c6c' },
        { gte: 3.9, lt: 6.1, color: '#67c23a' },
        { gte: 6.1, lt: 7.8, color: '#e6a23c' },
        { gte: 7.8, color: '#f56c6c' }
      ]
    },
    series: [{
      type: 'line',
      data: data.map(d => [d.time, d.value]),
      smooth: true,
      symbol: 'circle',
      symbolSize: 6
    }]
  })
}

onMounted(() => {
  if (profileId.value && password.value) {
    loadData()
  }
})
</script>

<style>
* {
  box-sizing: border-box;
}

.tool-container {
  padding: 20px;
}

.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.tool-header h2 {
  margin: 0;
  font-size: 20px;
  color: #333;
}

/* 登录 */
.login-section .el-card {
  margin-bottom: 15px;
}

.mobile-input {
  margin-bottom: 15px;
}

.login-btn {
  width: 100%;
  height: 44px;
  font-size: 16px;
}

.create-link {
  text-align: center;
  margin-top: 10px;
}

/* 语音 */
.voice-section {
  margin-bottom: 15px;
}

.voice-btn {
  width: 100%;
  height: 56px;
  font-size: 16px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
}

.recording-box {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 15px;
  background: #fef0f0;
  border-radius: 8px;
}

.recording-pulse {
  width: 12px;
  height: 12px;
  background: #f56c6c;
  border-radius: 50%;
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.3); }
}

.voice-result {
  margin-top: 10px;
  padding: 10px;
  background: #f0f9eb;
  border-radius: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

/* 快速输入 */
.quick-card {
  margin-bottom: 15px;
}

.quick-row {
  margin-bottom: 12px;
}

.quick-row:last-child {
  margin-bottom: 0;
}

.quick-row .label {
  font-size: 13px;
  color: #666;
  display: block;
  margin-bottom: 8px;
}

.glucose-input {
  width: 100%;
}

.glucose-input .el-input__wrapper {
  padding: 8px 12px;
}

.quick-btns {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.quick-btns .el-button {
  min-width: 45px;
}

/* 添加按钮 */
.add-btn {
  width: 100%;
  height: 50px;
  font-size: 16px;
  margin-bottom: 15px;
}

/* 统计 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  margin-bottom: 15px;
}

.stat-box {
  background: #fff;
  border-radius: 12px;
  padding: 15px 10px;
  text-align: center;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}

.stat-num {
  font-size: 22px;
  font-weight: bold;
  color: #333;
}

.stat-label {
  font-size: 12px;
  color: #999;
}

.stat-box.success .stat-num { color: #67c23a; }
.stat-box.warning .stat-num { color: #e6a23c; }
.stat-box.danger .stat-num { color: #f56c6c; }

/* 图表 */
.chart-card {
  margin-bottom: 15px;
}

/* 记录列表 */
.list-card {
  margin-bottom: 15px;
}

.list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.list-actions {
  display: flex;
  gap: 8px;
}

.record-list {
  max-height: 400px;
  overflow-y: auto;
}

.record-item {
  padding: 12px 0;
  border-bottom: 1px solid #eee;
}

.record-main {
  display: flex;
  align-items: center;
  gap: 10px;
}

.record-extra {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}

.record-extra .voice-text {
  color: #409eff;
  display: block;
  margin-bottom: 2px;
}

.record-time {
  font-size: 12px;
  color: #999;
  flex: 1;
}

.record-value {
  font-size: 18px;
  font-weight: bold;
  min-width: 50px;
  text-align: center;
}

.record-value.success { color: #67c23a; }
.record-value.warning { color: #e6a23c; }
.record-value.danger { color: #f56c6c; }

.record-type {
  font-size: 12px;
  color: #666;
  min-width: 40px;
}

.record-actions {
  display: flex;
  gap: 5px;
}

.empty-tip {
  text-align: center;
  padding: 30px;
  color: #999;
}

/* 历史 */
.history-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 0;
  border-bottom: 1px solid #eee;
  flex-wrap: wrap;
}

.history-time {
  font-size: 12px;
  color: #999;
  margin-left: auto;
}
</style>
