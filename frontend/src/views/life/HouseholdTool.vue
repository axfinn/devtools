<template>
  <div class="household-page">
    <!-- 档案未登录界面 -->
    <div v-if="!profileId" class="welcome-section">
      <div class="welcome-card">
        <h3>创建家庭物品档案</h3>
        <p>输入密码，创建您的专属物品管理档案。</p>
        <el-form :model="createForm" label-width="80px" style="max-width: 400px; margin: 20px auto;">
          <el-form-item label="档案名称">
            <el-input v-model="createForm.name" placeholder="如：我的家庭物品" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="createForm.password" type="password" placeholder="至少4个字符" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleCreateProfile" :loading="creating">创建档案</el-button>
          </el-form-item>
        </el-form>
      </div>

      <div class="welcome-card" style="margin-top: 20px;">
        <h3>加载已有档案</h3>
        <p>输入创建时设置的密码即可加载档案。</p>
        <el-form :model="loginForm" label-width="80px" style="max-width: 400px; margin: 20px auto;">
          <el-form-item label="密码">
            <el-input v-model="loginForm.password" type="password" placeholder="输入密码" show-password @keyup.enter="handleLogin" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleLogin" :loading="loading">加载档案</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 已登录：显示主界面 -->
    <template v-else>
      <!-- AI 对话助手按钮 -->
      <div v-if="aiEnabled" class="ai-chat-fab" @click.stop="showChatDrawer = true">
        <el-icon :size="28"><ChatDotRound /></el-icon>
      </div>

      <!-- 头部统计卡片 -->
      <section class="stats-section">
        <div class="stats-cards">
          <div class="stat-card">
            <div class="stat-icon total"><el-icon><Box /></el-icon></div>
            <div class="stat-content">
              <div class="stat-value">{{ stats.total }}</div>
              <div class="stat-label">物品总数</div>
            </div>
          </div>
          <div class="stat-card warning">
            <div class="stat-icon low-stock"><el-icon><Warning /></el-icon></div>
            <div class="stat-content">
              <div class="stat-value">{{ stats.low_stock }}</div>
              <div class="stat-label">库存不足</div>
            </div>
          </div>
          <div class="stat-card danger">
            <div class="stat-icon expiring"><el-icon><Timer /></el-icon></div>
            <div class="stat-content">
              <div class="stat-value">{{ stats.expiring }}</div>
              <div class="stat-label">即将过期</div>
            </div>
          </div>
          <div class="stat-card error">
            <div class="stat-icon expired"><el-icon><CircleClose /></el-icon></div>
            <div class="stat-content">
              <div class="stat-value">{{ stats.expired }}</div>
              <div class="stat-label">已过期</div>
            </div>
          </div>
        </div>
      </section>

      <!-- 档案操作栏 -->
      <section class="profile-bar">
        <div class="profile-info">
          <el-tag type="success">{{ profileName }}</el-tag>
          <el-button text @click="showExtendDialog = true">
            <el-icon><Timer /></el-icon>
            延期
          </el-button>
          <el-button text type="danger" @click="confirmDeleteProfile">
            <el-icon><Delete /></el-icon>
            删除档案
          </el-button>
        </div>
        <el-button @click="logoutProfile">退出登录</el-button>
      </section>

      <!-- 主工具栏 -->
      <section class="toolbar">
        <div class="toolbar-left">
          <el-button type="primary" @click="showAddDialog = true">
            <el-icon><Plus /></el-icon>
            添加物品
          </el-button>
          <el-button @click="showTemplateDialog = true">
            <el-icon><Document /></el-icon>
            模板库
          </el-button>
          <el-button @click="showLocationDialog = true">
            <el-icon><Location /></el-icon>
            位置库
          </el-button>
          <el-button @click="showScanDialog = true">
            <el-icon><Camera /></el-icon>
            扫码
          </el-button>
          <el-button @click="showReceiptDialog = true">
            <el-icon><Ticket /></el-icon>
            小票识别
          </el-button>
          <el-button v-if="aiEnabled" type="success" @click="showChatDrawer = true">
            <el-icon><MagicStick /></el-icon>
            AI 助手
          </el-button>
          <el-button type="warning" @click="showShoppingDialog = true">
            <el-icon><ShoppingCart /></el-icon>
            购物清单
          </el-button>
          <el-button @click="showExportDialog = true">
            <el-icon><Download /></el-icon>
            导出
          </el-button>
        </div>
        <div class="toolbar-right">
          <el-input
            v-model="searchText"
            placeholder="搜索物品..."
            clearable
            style="width: 200px;"
          >
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-select v-model="filterCategory" placeholder="全部分类" clearable style="width: 120px;" @change="loadItems">
            <el-option v-for="cat in categories" :key="cat" :label="cat" :value="cat" />
          </el-select>
          <el-checkbox v-model="showAlertsOnly" @change="loadItems">仅显示告警</el-checkbox>
        </div>
      </section>

      <!-- 物品列表 -->
      <section class="items-section">
        <el-table :data="filteredItems" style="width: 100%" row-key="id">
          <el-table-column prop="name" label="名称" min-width="120">
            <template #default="{ row }">
              <div class="item-name">
                <span>{{ row.name }}</span>
                <el-tag v-if="isLowStock(row)" type="danger" size="small">库存不足</el-tag>
                <el-tag v-if="isExpiring(row)" type="warning" size="small">即将过期</el-tag>
                <el-tag v-if="isExpired(row)" type="danger" size="small">已过期</el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="category" label="分类" width="100" />
          <el-table-column label="数量" width="180">
            <template #default="{ row }">
              <div class="quantity-cell">
                <el-button size="small" circle @click.stop="useItem(row, 1)">
                  <el-icon><Minus /></el-icon>
                </el-button>
                <span class="quantity-value" :class="{ 'low-stock': isLowStock(row) }">
                  {{ row.quantity }}{{ row.unit }}
                </span>
                <el-button size="small" circle @click.stop="restockItem(row, 1)">
                  <el-icon><Plus /></el-icon>
                </el-button>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="location" label="位置" width="100" />
          <el-table-column label="保质期" width="150">
            <template #default="{ row }">
              <span v-if="row.expiry_date && row.expiry_days > 0" :class="{ 'text-danger': isExpired(row), 'text-warning': isExpiring(row) }">
                {{ formatExpiry(row) }}
              </span>
              <span v-else class="text-muted">-</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="{ row }">
              <el-button-group>
                <el-button size="small" @click.stop="openItem(row)">开封</el-button>
                <el-button size="small" @click.stop="showItemQR(row)" title="生成二维码">
                  <el-icon><Picture /></el-icon>
                </el-button>
                <el-button size="small" @click.stop="openSpaceForItem(row)" title="定位到3D空间">
                  <el-icon><Location /></el-icon>
                </el-button>
                <el-button size="small" type="danger" text @click.stop="confirmDelete(row)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-button-group>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="filteredItems.length === 0" description="暂无物品，点击上方添加按钮开始管理" />
      </section>
    </template>

    <!-- 子组件对话框 -->
    <HouseholdAddDialog
      v-model="showAddDialog"
      :profile-id="profileId"
      :creator-key="creatorKey"
      :locations="locations"
      ref="addDialogRef"
      @added="loadItems"
    />

    <HouseholdTemplateDialog
      v-model="showTemplateDialog"
      :templates-by-category="templatesByCategory"
      @select="onTemplateSelect"
    />

    <HouseholdScanDialog
      v-model="showScanDialog"
      :profile-id="profileId"
      :creator-key="creatorKey"
      :ai-enabled="aiEnabled"
      @scan-done="onScanDone"
    />

    <HouseholdReceiptDialog
      v-model="showReceiptDialog"
      :profile-id="profileId"
      :creator-key="creatorKey"
      @added="loadItems"
    />

    <HouseholdQRDialog
      v-model="showQRDialog"
      :item="qrItem"
    />

    <HouseholdShoppingDialog
      v-model="showShoppingDialog"
      :profile-id="profileId"
      :creator-key="creatorKey"
      :items="items"
    />

    <HouseholdExportDialog
      v-model="showExportDialog"
      :items="items"
      :categories="categories"
    />

    <HouseholdExtendDialog
      v-model="showExtendDialog"
      :profile-id="profileId"
      :creator-key="creatorKey"
    />

    <HouseholdLocationDialog
      v-model="showLocationDialog"
      :profile-id="profileId"
      :creator-key="creatorKey"
      @changed="loadLocations"
    />

    <HouseholdChatDrawer
      v-model="showChatDrawer"
      :profile-id="profileId"
      :creator-key="creatorKey"
      @refresh="loadItems"
      @apply-location="onApplyLocation"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Minus, Delete, Search, Box, Warning, Timer, CircleClose,
  Document, MagicStick, ShoppingCart, ChatDotRound, Camera, Download, Picture, Ticket, Location
} from '@element-plus/icons-vue'

import HouseholdAddDialog from '../household/HouseholdAddDialog.vue'
import HouseholdTemplateDialog from '../household/HouseholdTemplateDialog.vue'
import HouseholdScanDialog from '../household/HouseholdScanDialog.vue'
import HouseholdReceiptDialog from '../household/HouseholdReceiptDialog.vue'
import HouseholdQRDialog from '../household/HouseholdQRDialog.vue'
import HouseholdShoppingDialog from '../household/HouseholdShoppingDialog.vue'
import HouseholdExportDialog from '../household/HouseholdExportDialog.vue'
import HouseholdExtendDialog from '../household/HouseholdExtendDialog.vue'
import HouseholdLocationDialog from '../household/HouseholdLocationDialog.vue'
import HouseholdChatDrawer from '../household/HouseholdChatDrawer.vue'

const API_BASE = '/api'

// 档案状态
const profileId = ref('')
const profileName = ref('')
const creatorKey = ref('')
const createForm = ref({ name: '', password: '' })
const loginForm = ref({ password: '' })
const creating = ref(false)
const loading = ref(false)

// 数据状态
const items = ref([])
const categories = ref([])
const locations = ref([])
const templates = ref([])
const templatesByCategory = ref({})
const stats = ref({ total: 0, low_stock: 0, expiring: 0, expired: 0 })
const aiEnabled = ref(false)

// 筛选
const searchText = ref('')
const filterCategory = ref('')
const showAlertsOnly = ref(false)

// 对话框开关
const showAddDialog = ref(false)
const showTemplateDialog = ref(false)
const showScanDialog = ref(false)
const showReceiptDialog = ref(false)
const showQRDialog = ref(false)
const showShoppingDialog = ref(false)
const showExportDialog = ref(false)
const showExtendDialog = ref(false)
const showLocationDialog = ref(false)
const showChatDrawer = ref(false)

const qrItem = ref(null)
const addDialogRef = ref(null)

const filteredItems = computed(() => {
  if (!searchText.value) return items.value
  const keyword = searchText.value.toLowerCase()
  return items.value.filter(item =>
    item.name.toLowerCase().includes(keyword) ||
    item.category.toLowerCase().includes(keyword) ||
    item.location.toLowerCase().includes(keyword)
  )
})

const profileQuery = () => creatorKey.value ? `creator_key=${creatorKey.value}` : ''

// 档案操作
async function handleCreateProfile() {
  if (!createForm.value.password || createForm.value.password.length < 4) {
    ElMessage.warning('密码至少4位')
    return
  }
  creating.value = true
  try {
    const res = await fetch(`${API_BASE}/household/profile`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: createForm.value.password, name: createForm.value.name || '我的家庭物品' })
    })
    const data = await res.json()
    if (data.code === 0) {
      profileId.value = data.id
      creatorKey.value = data.creator_key
      profileName.value = data.name
      localStorage.setItem('household_profile', JSON.stringify(data))
      ElMessage.success('档案创建成功')
      await loadItems()
    } else {
      ElMessage.error(data.error || '创建失败')
    }
  } catch { ElMessage.error('创建失败') }
  finally { creating.value = false }
}

async function handleLogin() {
  if (!loginForm.value.password) { ElMessage.warning('请输入密码'); return }
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/household/profile/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: loginForm.value.password })
    })
    const data = await res.json()
    if (data.code === 0) {
      profileId.value = data.id
      creatorKey.value = data.creator_key
      profileName.value = data.name
      localStorage.setItem('household_profile', JSON.stringify(data))
      ElMessage.success('登录成功')
      await loadItems()
    } else {
      ElMessage.error(data.error || '密码错误')
    }
  } catch { ElMessage.error('登录失败') }
  finally { loading.value = false }
}

function logoutProfile() {
  profileId.value = ''
  creatorKey.value = ''
  profileName.value = ''
  items.value = []
  localStorage.removeItem('household_profile')
}

function confirmDeleteProfile() {
  ElMessageBox.confirm('确定要删除此档案吗？此操作不可恢复！', '警告', {
    confirmButtonText: '确定删除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      const res = await fetch(`${API_BASE}/household/profile/${profileId.value}?${profileQuery()}`, { method: 'DELETE' })
      const data = await res.json()
      if (data.code === 0) { ElMessage.success('删除成功'); logoutProfile() }
      else ElMessage.error(data.error || '删除失败')
    } catch { ElMessage.error('删除失败') }
  }).catch(() => {})
}

// 物品数据操作
async function loadItems() {
  if (!profileId.value) return
  try {
    const params = new URLSearchParams()
    if (filterCategory.value) params.append('category', filterCategory.value)
    if (showAlertsOnly.value) params.append('alert', 'true')
    if (creatorKey.value) params.append('creator_key', creatorKey.value)
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items?${params}`)
    const data = await res.json()
    if (data.code === 0) {
      items.value = data.data || []
      stats.value = data.stats || { total: 0, low_stock: 0, expiring: 0, expired: 0 }
      loadLocations()
    }
  } catch { console.error('加载物品失败') }
}

async function loadLocations() {
  if (!profileId.value) return
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/locations?${profileQuery()}`)
    const data = await res.json()
    if (data.code === 0) {
      locations.value = data.data || []
      categories.value = [...new Set(items.value.map(i => i.category).filter(Boolean))]
    }
  } catch {}
}

async function loadTemplates() {
  try {
    const res = await fetch(`${API_BASE}/household/templates`)
    const data = await res.json()
    if (data.code === 0) {
      templates.value = data.data || []
      templatesByCategory.value = data.by_category || {}
    }
  } catch {}
}

async function checkAI() {
  try {
    const res = await fetch(`${API_BASE}/household/ai/check`)
    const data = await res.json()
    aiEnabled.value = data.enabled
  } catch { aiEnabled.value = false }
}

// 物品操作
async function useItem(item, amount) {
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items/${item.id}/use?${profileQuery()}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount })
    })
    const data = await res.json()
    if (data.code === 0) await loadItems()
  } catch { ElMessage.error('操作失败') }
}

async function restockItem(item, amount) {
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items/${item.id}/restock?${profileQuery()}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount })
    })
    const data = await res.json()
    if (data.code === 0) await loadItems()
  } catch { ElMessage.error('操作失败') }
}

async function openItem(item) {
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items/${item.id}/open?${profileQuery()}`, { method: 'POST' })
    const data = await res.json()
    if (data.code === 0) { ElMessage.success('已重置保质期'); await loadItems() }
  } catch { ElMessage.error('操作失败') }
}

function confirmDelete(item) {
  ElMessageBox.confirm(`确定要删除「${item.name}」吗？`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items/${item.id}?${profileQuery()}`, { method: 'DELETE' })
      const data = await res.json()
      if (data.code === 0) { ElMessage.success('删除成功'); await loadItems() }
    } catch { ElMessage.error('删除失败') }
  }).catch(() => {})
}

function showItemQR(item) {
  qrItem.value = item
  showQRDialog.value = true
}

function openSpaceForItem(item) {
  if (!item || !item.location) { ElMessage.info('该物品未设置位置'); return }
  window.open(`/household/space?highlight=${encodeURIComponent(item.location)}`, '_blank')
}

// 子组件事件处理
function onTemplateSelect(tpl) {
  showAddDialog.value = true
  addDialogRef.value?.fill({
    name: tpl.name,
    category: tpl.category,
    unit: tpl.unit,
    min_quantity: tpl.default_min_quantity || 1,
    expiry_days: tpl.default_expiry_days || 0
  })
}

function onScanDone({ refresh, prefill }) {
  if (refresh) {
    loadItems()
    return
  }
  if (prefill) {
    showAddDialog.value = true
    addDialogRef.value?.fill(prefill)
  }
}

async function onApplyLocation({ name, location }) {
  const item = items.value.find(i => i.name === name)
  if (!item) { ElMessage.warning('未找到匹配物品，请先添加或改名'); return }
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items/${item.id}?${profileQuery()}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ location })
    })
    const data = await res.json()
    if (data.code === 0) { ElMessage.success(`已更新位置：${location}`); await loadItems() }
    else ElMessage.error(data.error || '更新失败')
  } catch { ElMessage.error('更新失败') }
}

// 工具函数
function isLowStock(item) { return item.quantity <= item.min_quantity }

function isExpiring(item) {
  if (!item.expiry_date || item.expiry_days <= 0) return false
  const expiryDate = new Date(item.expiry_date)
  expiryDate.setDate(expiryDate.getDate() + item.expiry_days)
  const daysUntil = Math.ceil((expiryDate - new Date()) / (1000 * 60 * 60 * 24))
  return daysUntil > 0 && daysUntil <= 7
}

function isExpired(item) {
  if (!item.expiry_date || item.expiry_days <= 0) return false
  const expiryDate = new Date(item.expiry_date)
  expiryDate.setDate(expiryDate.getDate() + item.expiry_days)
  return expiryDate < new Date()
}

function formatExpiry(item) {
  if (!item.expiry_date || item.expiry_days <= 0) return '-'
  const expiryDate = new Date(item.expiry_date)
  expiryDate.setDate(expiryDate.getDate() + item.expiry_days)
  const daysUntil = Math.ceil((expiryDate - new Date()) / (1000 * 60 * 60 * 24))
  if (daysUntil < 0) return `已过期${-daysUntil}天`
  if (daysUntil === 0) return '今天过期'
  if (daysUntil <= 7) return `还剩${daysUntil}天`
  return expiryDate.toLocaleDateString()
}

onMounted(async () => {
  const saved = localStorage.getItem('household_profile')
  if (saved) {
    try {
      const profile = JSON.parse(saved)
      const res = await fetch(`${API_BASE}/household/profile/${profile.id}?creator_key=${profile.creator_key}`)
      const data = await res.json()
      if (data.code === 0) {
        profileId.value = profile.id
        creatorKey.value = profile.creator_key
        profileName.value = profile.name
        await loadItems()
      } else {
        localStorage.removeItem('household_profile')
      }
    } catch { localStorage.removeItem('household_profile') }
  }
  await Promise.all([loadTemplates(), checkAI()])
})
</script>

<style scoped>
.household-page { padding: 20px; }
.welcome-section { max-width: 500px; margin: 40px auto; }
.welcome-card { background: #fff; border-radius: 8px; padding: 30px; box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1); }
.welcome-card h3 { margin: 0 0 10px 0; color: #303133; }
.welcome-card p { color: #909399; margin: 0; }

.stats-section { margin-bottom: 20px; }
.stats-cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; }
.stat-card { background: #fff; border-radius: 8px; padding: 20px; display: flex; align-items: center; box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06); }
.stat-card.warning { border-left: 4px solid #e6a23c; }
.stat-card.danger, .stat-card.error { border-left: 4px solid #f56c6c; }
.stat-icon { width: 48px; height: 48px; border-radius: 8px; display: flex; align-items: center; justify-content: center; margin-right: 16px; font-size: 24px; }
.stat-icon.total { background: #ecf5ff; color: #409eff; }
.stat-icon.low-stock { background: #fef0f0; color: #f56c6c; }
.stat-icon.expiring { background: #fdf6ec; color: #e6a23c; }
.stat-icon.expired { background: #fef0f0; color: #f56c6c; }
.stat-value { font-size: 28px; font-weight: bold; color: #303133; }
.stat-label { font-size: 14px; color: #909399; }

.profile-bar { display: flex; justify-content: space-between; align-items: center; background: #fff; padding: 12px 20px; border-radius: 8px; margin-bottom: 20px; }
.profile-info { display: flex; align-items: center; gap: 12px; }

.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; flex-wrap: wrap; gap: 12px; }
.toolbar-left, .toolbar-right { display: flex; align-items: center; gap: 12px; }

.items-section { background: #fff; border-radius: 8px; padding: 20px; box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06); }
.item-name { display: flex; align-items: center; gap: 8px; }
.quantity-cell { display: flex; align-items: center; gap: 8px; }
.quantity-value { min-width: 50px; text-align: center; font-weight: bold; }
.quantity-value.low-stock { color: #f56c6c; }
.text-danger { color: #f56c6c; }
.text-warning { color: #e6a23c; }
.text-muted { color: #c0c4cc; }

.ai-chat-fab {
  position: fixed;
  right: 20px;
  bottom: 20px;
  width: 60px;
  height: 60px;
  border-radius: 50%;
  background: linear-gradient(135deg, #67c23a 0%, #85ce61 100%);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(103, 194, 58, 0.4);
  z-index: 1000;
  transition: transform 0.2s, box-shadow 0.2s;
}
.ai-chat-fab:hover { transform: scale(1.1); box-shadow: 0 6px 16px rgba(103, 194, 58, 0.5); }

@media (max-width: 768px) {
  .stats-cards { grid-template-columns: repeat(2, 1fr); }
  .toolbar { flex-direction: column; align-items: stretch; }
  .toolbar-left, .toolbar-right { flex-wrap: wrap; }
}
</style>
