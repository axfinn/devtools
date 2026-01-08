<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>时间戳转换工具</h2>
    </div>

    <div class="current-time-card">
      <div class="time-display">
        <div class="time-item">
          <span class="label">当前时间</span>
          <span class="value">{{ currentTime }}</span>
        </div>
        <div class="time-item">
          <span class="label">Unix 时间戳 (秒)</span>
          <span class="value timestamp" @click="copyToClipboard(currentTimestamp)">
            {{ currentTimestamp }}
            <el-icon><CopyDocument /></el-icon>
          </span>
        </div>
        <div class="time-item">
          <span class="label">Unix 时间戳 (毫秒)</span>
          <span class="value timestamp" @click="copyToClipboard(currentTimestampMs)">
            {{ currentTimestampMs }}
            <el-icon><CopyDocument /></el-icon>
          </span>
        </div>
      </div>
    </div>

    <div class="converter-section">
      <el-row :gutter="20">
        <el-col :span="12">
          <div class="converter-card">
            <h3>时间戳 → 日期时间</h3>
            <el-input
              v-model="timestampInput"
              placeholder="输入时间戳"
              size="large"
              @input="convertTimestamp"
            >
              <template #append>
                <el-select v-model="timestampUnit" style="width: 80px" @change="convertTimestamp">
                  <el-option label="秒" value="s" />
                  <el-option label="毫秒" value="ms" />
                </el-select>
              </template>
            </el-input>
            <div class="result" v-if="convertedDate">
              <div class="result-item">
                <span class="label">本地时间</span>
                <span class="value">{{ convertedDate.local }}</span>
              </div>
              <div class="result-item">
                <span class="label">UTC 时间</span>
                <span class="value">{{ convertedDate.utc }}</span>
              </div>
              <div class="result-item">
                <span class="label">ISO 8601</span>
                <span class="value">{{ convertedDate.iso }}</span>
              </div>
            </div>
          </div>
        </el-col>

        <el-col :span="12">
          <div class="converter-card">
            <h3>日期时间 → 时间戳</h3>
            <el-date-picker
              v-model="dateInput"
              type="datetime"
              placeholder="选择日期时间"
              size="large"
              style="width: 100%"
              @change="convertDate"
            />
            <div class="result" v-if="convertedTimestamp">
              <div class="result-item">
                <span class="label">秒</span>
                <span class="value timestamp" @click="copyToClipboard(convertedTimestamp.seconds)">
                  {{ convertedTimestamp.seconds }}
                  <el-icon><CopyDocument /></el-icon>
                </span>
              </div>
              <div class="result-item">
                <span class="label">毫秒</span>
                <span class="value timestamp" @click="copyToClipboard(convertedTimestamp.milliseconds)">
                  {{ convertedTimestamp.milliseconds }}
                  <el-icon><CopyDocument /></el-icon>
                </span>
              </div>
            </div>
          </div>
        </el-col>
      </el-row>
    </div>

    <div class="time-calc-section">
      <h3>时间计算</h3>
      <div class="calc-row">
        <el-date-picker
          v-model="calcDate"
          type="datetime"
          placeholder="选择起始时间"
          size="large"
        />
        <el-select v-model="calcOperation" style="width: 80px">
          <el-option label="+" value="add" />
          <el-option label="-" value="sub" />
        </el-select>
        <el-input-number v-model="calcValue" :min="0" size="large" />
        <el-select v-model="calcUnit" style="width: 100px">
          <el-option label="秒" value="seconds" />
          <el-option label="分钟" value="minutes" />
          <el-option label="小时" value="hours" />
          <el-option label="天" value="days" />
          <el-option label="周" value="weeks" />
          <el-option label="月" value="months" />
          <el-option label="年" value="years" />
        </el-select>
        <el-button type="primary" @click="calculateTime">=</el-button>
        <span class="calc-result" v-if="calcResult">{{ calcResult }}</span>
      </div>
    </div>

    <div class="quick-times">
      <h3>常用时间戳</h3>
      <el-table :data="quickTimes" border stripe>
        <el-table-column prop="label" label="描述" width="150" />
        <el-table-column prop="timestamp" label="时间戳 (秒)">
          <template #default="{ row }">
            <span class="timestamp" @click="copyToClipboard(row.timestamp)">
              {{ row.timestamp }}
              <el-icon><CopyDocument /></el-icon>
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="datetime" label="日期时间" />
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'

const currentTime = ref('')
const currentTimestamp = ref(0)
const currentTimestampMs = ref(0)
const timestampInput = ref('')
const timestampUnit = ref('s')
const dateInput = ref(null)
const convertedDate = ref(null)
const convertedTimestamp = ref(null)
const calcDate = ref(new Date())
const calcOperation = ref('add')
const calcValue = ref(1)
const calcUnit = ref('days')
const calcResult = ref('')

let timer = null

const updateCurrentTime = () => {
  const now = new Date()
  currentTime.value = now.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  })
  currentTimestamp.value = Math.floor(now.getTime() / 1000)
  currentTimestampMs.value = now.getTime()
}

onMounted(() => {
  updateCurrentTime()
  timer = setInterval(updateCurrentTime, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const convertTimestamp = () => {
  if (!timestampInput.value) {
    convertedDate.value = null
    return
  }

  try {
    let ts = parseInt(timestampInput.value)
    if (timestampUnit.value === 's') {
      ts = ts * 1000
    }
    const date = new Date(ts)

    if (isNaN(date.getTime())) {
      convertedDate.value = null
      return
    }

    convertedDate.value = {
      local: date.toLocaleString('zh-CN'),
      utc: date.toUTCString(),
      iso: date.toISOString()
    }
  } catch (e) {
    convertedDate.value = null
  }
}

const convertDate = () => {
  if (!dateInput.value) {
    convertedTimestamp.value = null
    return
  }

  const date = new Date(dateInput.value)
  convertedTimestamp.value = {
    seconds: Math.floor(date.getTime() / 1000),
    milliseconds: date.getTime()
  }
}

const calculateTime = () => {
  if (!calcDate.value) return

  const date = new Date(calcDate.value)
  const multipliers = {
    seconds: 1000,
    minutes: 60 * 1000,
    hours: 60 * 60 * 1000,
    days: 24 * 60 * 60 * 1000,
    weeks: 7 * 24 * 60 * 60 * 1000
  }

  let resultDate
  if (calcUnit.value === 'months') {
    resultDate = new Date(date)
    resultDate.setMonth(
      calcOperation.value === 'add'
        ? date.getMonth() + calcValue.value
        : date.getMonth() - calcValue.value
    )
  } else if (calcUnit.value === 'years') {
    resultDate = new Date(date)
    resultDate.setFullYear(
      calcOperation.value === 'add'
        ? date.getFullYear() + calcValue.value
        : date.getFullYear() - calcValue.value
    )
  } else {
    const offset = multipliers[calcUnit.value] * calcValue.value
    resultDate = new Date(
      calcOperation.value === 'add'
        ? date.getTime() + offset
        : date.getTime() - offset
    )
  }

  calcResult.value = resultDate.toLocaleString('zh-CN')
}

const quickTimes = computed(() => {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const tomorrow = new Date(today.getTime() + 24 * 60 * 60 * 1000)
  const weekStart = new Date(today.getTime() - today.getDay() * 24 * 60 * 60 * 1000)
  const monthStart = new Date(now.getFullYear(), now.getMonth(), 1)
  const yearStart = new Date(now.getFullYear(), 0, 1)

  return [
    {
      label: '今天开始',
      timestamp: Math.floor(today.getTime() / 1000),
      datetime: today.toLocaleString('zh-CN')
    },
    {
      label: '明天开始',
      timestamp: Math.floor(tomorrow.getTime() / 1000),
      datetime: tomorrow.toLocaleString('zh-CN')
    },
    {
      label: '本周开始',
      timestamp: Math.floor(weekStart.getTime() / 1000),
      datetime: weekStart.toLocaleString('zh-CN')
    },
    {
      label: '本月开始',
      timestamp: Math.floor(monthStart.getTime() / 1000),
      datetime: monthStart.toLocaleString('zh-CN')
    },
    {
      label: '本年开始',
      timestamp: Math.floor(yearStart.getTime() / 1000),
      datetime: yearStart.toLocaleString('zh-CN')
    }
  ]
})

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(String(text))
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}
</script>

<style scoped>
.tool-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.tool-header h2 {
  margin: 0;
  color: #e0e0e0;
}

.current-time-card {
  background: linear-gradient(135deg, #1e3a5f 0%, #2d5a87 100%);
  padding: 20px;
  border-radius: 12px;
}

.time-display {
  display: flex;
  justify-content: space-around;
  flex-wrap: wrap;
  gap: 20px;
}

.time-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.time-item .label {
  color: #a0c4e8;
  font-size: 14px;
}

.time-item .value {
  color: #fff;
  font-size: 20px;
  font-family: 'Consolas', 'Monaco', monospace;
}

.timestamp {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 5px;
}

.timestamp:hover {
  color: #409eff;
}

.converter-section {
  margin-top: 10px;
}

.converter-card {
  background-color: #1e1e1e;
  padding: 20px;
  border-radius: 8px;
}

.converter-card h3 {
  margin: 0 0 15px 0;
  color: #e0e0e0;
}

.result {
  margin-top: 15px;
  padding: 15px;
  background-color: #2d2d2d;
  border-radius: 6px;
}

.result-item {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #3d3d3d;
}

.result-item:last-child {
  border-bottom: none;
}

.result-item .label {
  color: #a0a0a0;
}

.result-item .value {
  color: #4caf50;
  font-family: 'Consolas', 'Monaco', monospace;
}

.time-calc-section {
  background-color: #1e1e1e;
  padding: 20px;
  border-radius: 8px;
}

.time-calc-section h3 {
  margin: 0 0 15px 0;
  color: #e0e0e0;
}

.calc-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.calc-result {
  color: #4caf50;
  font-size: 18px;
  font-family: 'Consolas', 'Monaco', monospace;
  padding: 10px 20px;
  background-color: #2d2d2d;
  border-radius: 6px;
}

.quick-times {
  background-color: #1e1e1e;
  padding: 20px;
  border-radius: 8px;
}

.quick-times h3 {
  margin: 0 0 15px 0;
  color: #e0e0e0;
}
</style>
