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
      <div v-if="aiEnabled" class="ai-chat-fab" @click="showChatDrawer = true">
        <el-icon :size="28"><ChatDotRound /></el-icon>
      </div>

      <!-- 对话抽屉 -->
      <el-drawer v-model="showChatDrawer" title="AI 助手" direction="rtl" size="400px">
        <div class="chat-container">
          <div class="chat-messages" ref="chatMessagesRef">
            <div v-if="chatHistory.length === 0" class="chat-welcome">
              <el-icon :size="48" color="#67c23a"><ChatDotRound /></el-icon>
              <p>你好！我是家庭物品管理助手</p>
              <p class="hint">可以对我说：</p>
              <ul>
                <li>"帮我买三瓶洗衣液"</li>
                <li>"家里还有什么缺的？"</li>
                <li>"查看库存情况"</li>
                <li>"删除那瓶过期的酱油"</li>
              </ul>
            </div>
            <div
              v-for="(msg, idx) in chatHistory"
              :key="idx"
              class="chat-message"
              :class="msg.role"
            >
              <div class="message-avatar">
                <el-icon v-if="msg.role === 'user'"><User /></el-icon>
                <el-icon v-else><Service /></el-icon>
              </div>
              <div class="message-content">
                <div class="message-text" v-html="formatMessage(msg.content)"></div>
              </div>
            </div>
            <div v-if="chatLoading" class="chat-message assistant">
              <div class="message-avatar">
                <el-icon><Robot /></el-icon>
              </div>
              <div class="message-content">
                <div class="message-text typing">
                  <span class="dot">.</span><span class="dot">.</span><span class="dot">.</span>
                </div>
              </div>
            </div>
          </div>
          <div class="chat-input-area">
            <el-input
              v-model="chatInput"
              placeholder="输入消息或点击麦克风说话..."
              @keyup.enter="sendChatMessage"
              :disabled="chatLoading"
            >
              <template #append>
                <el-button @click="toggleVoice" :class="{ 'voice-active': isRecording }">
                  <el-icon><Microphone /></el-icon>
                </el-button>
              </template>
            </el-input>
            <el-button type="primary" @click="sendChatMessage" :loading="chatLoading" :disabled="!chatInput.trim()">
              发送
            </el-button>
          </div>
          <div class="chat-actions">
            <el-button text size="small" @click="clearChatHistory">
              <el-icon><Delete /></el-icon>
              清除历史
            </el-button>
          </div>
        </div>
      </el-drawer>

      <!-- 头部统计卡片 -->
      <section class="stats-section">
        <div class="stats-cards">
          <div class="stat-card">
            <div class="stat-icon total">
              <el-icon><Box /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ stats.total }}</div>
              <div class="stat-label">物品总数</div>
            </div>
          </div>
          <div class="stat-card warning">
            <div class="stat-icon low-stock">
              <el-icon><Warning /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ stats.low_stock }}</div>
              <div class="stat-label">库存不足</div>
            </div>
          </div>
          <div class="stat-card danger">
            <div class="stat-icon expiring">
              <el-icon><Timer /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ stats.expiring }}</div>
              <div class="stat-label">即将过期</div>
            </div>
          </div>
          <div class="stat-card error">
            <div class="stat-icon expired">
              <el-icon><CircleClose /></el-icon>
            </div>
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
          <el-button @click="showScanDialog = true">
            <el-icon><Camera /></el-icon>
            扫码
          </el-button>
          <el-button @click="showReceiptDialog = true">
            <el-icon><Ticket /></el-icon>
            小票识别
          </el-button>
          <el-button v-if="aiEnabled" type="success" @click="showAIDialog = true">
            <el-icon><MagicStick /></el-icon>
            AI 助手
          </el-button>
          <el-button type="warning" @click="showShoppingListDialog = true">
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
            @input="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
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

    <!-- 添加物品对话框 -->
    <el-dialog v-model="showAddDialog" title="添加物品" width="500px">
      <el-form :model="addForm" label-width="80px">
        <el-form-item label="名称" required>
          <el-input v-model="addForm.name" placeholder="物品名称" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="addForm.category" placeholder="选择分类" style="width: 100%;">
            <el-option v-for="cat in categoryOptions" :key="cat" :label="cat" :value="cat" />
          </el-select>
        </el-form-item>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="数量">
              <el-input-number v-model="addForm.quantity" :min="1" style="width: 100%;" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="单位">
              <el-input v-model="addForm.unit" placeholder="个/瓶/盒..." />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="最低库存">
          <el-input-number v-model="addForm.min_quantity" :min="1" style="width: 100%;" />
        </el-form-item>
        <el-form-item label="位置">
          <el-input v-model="addForm.location" placeholder="如：厨房、冰箱" />
        </el-form-item>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="生产日期">
              <el-date-picker v-model="addForm.expiry_date" type="date" placeholder="选择日期" style="width: 100%;" value-format="YYYY-MM-DD" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="保质期(天)">
              <el-input-number v-model="addForm.expiry_days" :min="0" placeholder="0=无保质期" style="width: 100%;" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="备注">
          <el-input v-model="addForm.notes" type="textarea" :rows="2" placeholder="可选备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" @click="handleAdd">确定</el-button>
      </template>
    </el-dialog>

    <!-- 模板库对话框 -->
    <el-dialog v-model="showTemplateDialog" title="物品模板库" width="600px">
      <div class="template-grid">
        <div v-for="(items, category) in templatesByCategory" :key="category" class="template-category">
          <div class="category-title">{{ category }}</div>
          <div class="template-items">
            <el-tag
              v-for="tpl in items"
              :key="tpl.id"
              class="template-tag"
              @click="addFromTemplate(tpl)"
            >
              {{ tpl.name }} ({{ tpl.unit }})
            </el-tag>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- AI 助手对话框 -->
    <el-dialog v-model="showAIDialog" title="AI 智能助手" width="600px">
      <el-tabs v-model="aiTab">
        <el-tab-pane label="智能添加" name="add">
          <div class="ai-add-section">
            <el-input
              v-model="aiText"
              type="textarea"
              :rows="4"
              placeholder="用自然语言描述要添加的物品，例如：&#10;• 家里洗衣液快用完了，买一瓶&#10;• 冰箱里的鸡蛋只剩6个了"
            />
            <el-button type="primary" style="margin-top: 12px;" :loading="aiLoading" @click="handleAIAdd">
              <el-icon><MagicStick /></el-icon>
              AI 智能识别添加
            </el-button>
            <div v-if="aiResult.length > 0" class="ai-result">
              <div class="ai-result-title">识别到的物品：</div>
              <el-tag v-for="item in aiResult" :key="item.id" type="success" style="margin-right: 8px; margin-bottom: 8px;">
                {{ item.name }} x {{ item.quantity }}{{ item.unit }}
              </el-tag>
            </div>
          </div>
        </el-tab-pane>
        <el-tab-pane label="推荐补充" name="restock">
          <div class="ai-restock-section">
            <el-button type="warning" :loading="loadingRestock" @click="loadRestockSuggestions">
              <el-icon><ShoppingCart /></el-icon>
              查看需要补充的物品
            </el-button>
            <div v-if="restockSuggestions.length > 0" class="restock-list">
              <el-card v-for="item in restockSuggestions" :key="item.id" class="restock-item">
                <div class="restock-info">
                  <div class="restock-name">{{ item.name }}</div>
                  <div class="restock-reason">{{ item.reason }}</div>
                </div>
                <el-button size="small" type="primary" @click="quickRestock(item)">
                  立即补充
                </el-button>
              </el-card>
            </div>
            <el-empty v-else-if="!loadingRestock" description="库存充足，无需补充" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>

    <!-- 扫码对话框 -->
    <el-dialog v-model="showScanDialog" title="扫码添加物品" width="500px" @close="stopScan">
      <div class="scan-container">
        <div id="qr-reader" class="qr-reader"></div>
        <div v-if="scanResult" class="scan-result">
          <el-alert :title="'识别成功: ' + scanResult" type="success" :closable="false" />
          <div class="scan-actions" style="margin-top: 12px;">
            <el-button type="primary" @click="handleScanResult">添加到物品库</el-button>
            <el-button @click="scanResult = ''">继续扫描</el-button>
          </div>
        </div>
        <div class="scan-tip" style="margin-top: 16px; text-align: center; color: #909399;">
          <p>支持扫描商品条形码和二维码</p>
        </div>
      </div>
    </el-dialog>

    <!-- 小票 OCR 识别对话框 -->
    <el-dialog v-model="showReceiptDialog" title="小票 OCR 识别" width="600px" @close="clearReceipt">
      <div class="receipt-container">
        <el-upload
          v-if="!receiptImage"
          class="receipt-upload"
          :auto-upload="false"
          :show-file-list="false"
          accept="image/*"
          @change="handleReceiptFileChange"
        >
          <div class="upload-placeholder">
            <el-icon :size="48" color="#909399"><Picture /></el-icon>
            <p>点击上传小票图片</p>
            <p class="hint">或拖拽图片到此处</p>
          </div>
        </el-upload>

        <div v-else class="receipt-preview">
          <el-image :src="receiptImage" fit="contain" style="max-height: 300px;" />
          <div class="receipt-actions">
            <el-button @click="clearReceipt">重新选择</el-button>
            <el-button type="primary" :loading="receiptLoading" @click="recognizeReceipt">
              <el-icon><Tickets /></el-icon>
              开始识别
            </el-button>
          </div>
        </div>

        <!-- 识别结果 -->
        <div v-if="receiptItems.length > 0" class="receipt-result">
          <el-divider>识别结果</el-divider>
          <div class="matched-items">
            <el-tag
              v-for="(item, idx) in receiptItems"
              :key="idx"
              :type="item.matched ? 'success' : 'info'"
              class="receipt-item-tag"
              closable
              @close="removeReceiptItem(idx)"
            >
              {{ item.name }} <span v-if="item.quantity">x{{ item.quantity }}</span>
              <span v-if="item.matched" class="matched-badge">已匹配</span>
            </el-tag>
          </div>
          <div class="receipt-result-actions">
            <el-button type="primary" @click="addReceiptItemsToLibrary">
              全部添加到物品库
            </el-button>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 物品二维码对话框 -->
    <el-dialog v-model="showQRDialog" title="物品二维码" width="300px">
      <div v-if="qrItem" style="text-align: center;">
        <div id="qr-code生成" class="qr-code生成"></div>
        <p style="margin-top: 12px;">{{ qrItem.name }}</p>
        <p style="color: #909399; font-size: 12px;">{{ qrItem.category }} - {{ qrItem.quantity }}{{ qrItem.unit }}</p>
        <el-button type="primary" size="small" style="margin-top: 12px;" @click="downloadQR">
          <el-icon><Download /></el-icon>
          下载二维码
        </el-button>
      </div>
    </el-dialog>

    <!-- 购物清单对话框 -->
    <el-dialog v-model="showShoppingListDialog" title="购物清单" width="600px">
      <div class="shopping-list-container">
        <div class="shopping-list-header">
          <el-alert
            :title="'共 ' + shoppingList.length + ' 项需要购买'"
            type="warning"
            :closable="false"
          />
        </div>

        <div v-if="shoppingList.length > 0" class="shopping-list-content">
          <el-table :data="shoppingList" style="width: 100%" max-height="400">
            <el-table-column prop="name" label="物品名称" />
            <el-table-column prop="category" label="分类" width="100" />
            <el-table-column prop="reason" label="原因" />
            <el-table-column label="操作" width="80">
              <template #default="{ row, $index }">
                <el-button size="small" type="danger" text @click="removeFromShoppingList($index)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <el-empty v-else description="库存充足，暂无需要购买的物品" />

        <div class="shopping-list-actions">
          <el-button @click="copyShoppingList">
            <el-icon><CopyDocument /></el-icon>
            复制清单
          </el-button>
          <el-button type="primary" @click="exportShoppingList('text')">
            <el-icon><Download /></el-icon>
            导出文本
          </el-button>
          <el-button @click="shareShoppingList">
            <el-icon><Share /></el-icon>
            生成分享链接
          </el-button>
        </div>
      </div>
    </el-dialog>

    <!-- 导出对话框 -->
    <el-dialog v-model="showExportDialog" title="导出物品数据" width="500px">
      <div class="export-container">
        <el-form label-width="80px">
          <el-form-item label="导出范围">
            <el-radio-group v-model="exportScope">
              <el-radio label="all">全部物品</el-radio>
              <el-radio label="category">按分类</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="exportScope === 'category'" label="选择分类">
            <el-select v-model="exportCategory" placeholder="选择分类" style="width: 100%;">
              <el-option v-for="cat in categories" :key="cat" :label="cat" :value="cat" />
            </el-select>
          </el-form-item>
          <el-form-item label="导出格式">
            <el-radio-group v-model="exportFormat">
              <el-radio label="text">文本</el-radio>
              <el-radio label="json">JSON</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-form>

        <el-divider />

        <div class="export-preview">
          <div class="preview-header">预览：</div>
          <pre class="preview-content">{{ exportPreview }}</pre>
        </div>

        <div class="export-actions">
          <el-button @click="copyExportData">
            <el-icon><CopyDocument /></el-icon>
            复制
          </el-button>
          <el-button type="primary" @click="downloadExportData">
            <el-icon><Download /></el-icon>
            下载文件
          </el-button>
        </div>
      </div>
    </el-dialog>

    <!-- 延期对话框 -->
    <el-dialog v-model="showExtendDialog" title="延长档案有效期" width="400px">
      <el-form label-width="100px">
        <el-form-item label="延长天数">
          <el-input-number v-model="extendDays" :min="30" :max="365" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showExtendDialog = false">取消</el-button>
        <el-button type="primary" @click="handleExtend">确定延期</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Minus, Delete, Search, Box, Warning, Timer, CircleClose,
  Document, MagicStick, ShoppingCart, ChatDotRound, User, Service, Microphone, Camera, Download, Picture, Ticket, Tickets, CopyDocument, Share
} from '@element-plus/icons-vue'

const API_BASE = '/api'

// 档案状态
const profileId = ref('')
const profileName = ref('')
const creatorKey = ref('')
const createForm = ref({ name: '', password: '' })
const loginForm = ref({ password: '' })
const creating = ref(false)
const loading = ref(false)

// 状态变量
const items = ref([])
const categories = ref([])
const templates = ref([])
const templatesByCategory = ref({})
const stats = ref({ total: 0, low_stock: 0, expiring: 0, expired: 0 })

// 筛选状态
const searchText = ref('')
const filterCategory = ref('')
const showAlertsOnly = ref(false)

// 对话框状态
const showAddDialog = ref(false)
const showTemplateDialog = ref(false)
const showAIDialog = ref(false)
const showExtendDialog = ref(false)
const extendDays = ref(90)

// 表单
const addForm = ref({
  name: '',
  category: '其他',
  quantity: 1,
  unit: '个',
  min_quantity: 1,
  location: '',
  expiry_date: '',
  expiry_days: 0,
  notes: ''
})

const categoryOptions = ['厨房', '卫生间', '卧室', '客厅', '玄关', '阳台', '其他']

// AI 状态
const aiEnabled = ref(false)
const aiTab = ref('add')
const aiText = ref('')
const aiLoading = ref(false)
const aiResult = ref([])
const restockSuggestions = ref([])
const loadingRestock = ref(false)

// 对话状态
const showChatDrawer = ref(false)

// 扫码状态
const showScanDialog = ref(false)
const scanResult = ref('')
const isScanning = ref(false)
let html5QrCode = null

// 二维码状态
const showQRDialog = ref(false)
const qrItem = ref(null)

// 小票 OCR 状态
const showReceiptDialog = ref(false)
const receiptImage = ref('')
const receiptLoading = ref(false)
const receiptItems = ref([])

// 购物清单状态
const showShoppingListDialog = ref(false)
const shoppingList = ref([])

// 导出状态
const showExportDialog = ref(false)
const exportScope = ref('all')
const exportCategory = ref('')
const exportFormat = ref('text')
const chatInput = ref('')
const chatHistory = ref([])
const chatLoading = ref(false)
const chatMessagesRef = ref(null)
const isRecording = ref(false)
let recognition = null

// 计算属性
const filteredItems = computed(() => {
  let result = items.value

  if (searchText.value) {
    const keyword = searchText.value.toLowerCase()
    result = result.filter(item =>
      item.name.toLowerCase().includes(keyword) ||
      item.category.toLowerCase().includes(keyword) ||
      item.location.toLowerCase().includes(keyword)
    )
  }

  return result
})

// 导出预览
const exportPreview = computed(() => {
  let data = items.value

  // 按分类过滤
  if (exportScope.value === 'category' && exportCategory.value) {
    data = data.filter(item => item.category === exportCategory.value)
  }

  if (exportFormat.value === 'json') {
    return JSON.stringify(data, null, 2)
  }

  // 文本格式
  let text = '物品清单\n'
  text += '=' .repeat(30) + '\n\n'

  // 按分类整理
  const byCategory = {}
  data.forEach(item => {
    if (!byCategory[item.category]) {
      byCategory[item.category] = []
    }
    byCategory[item.category].push(item)
  })

  Object.keys(byCategory).sort().forEach(cat => {
    text += `【${cat}】\n`
    byCategory[cat].forEach(item => {
      text += `  - ${item.name}: ${item.quantity}${item.unit}`
      if (item.location) text += ` (${item.location})`
      text += '\n'
    })
    text += '\n'
  })

  text += `共计 ${data.length} 项`

  return text
})

const profileQuery = () => {
  const params = new URLSearchParams()
  if (creatorKey.value) {
    params.set('creator_key', creatorKey.value)
  }
  return params.toString()
}

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
      body: JSON.stringify({
        password: createForm.value.password,
        name: createForm.value.name || '我的家庭物品'
      })
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
  } catch (e) {
    ElMessage.error('创建失败')
  } finally {
    creating.value = false
  }
}

async function handleLogin() {
  if (!loginForm.value.password) {
    ElMessage.warning('请输入密码')
    return
  }

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
  } catch (e) {
    ElMessage.error('登录失败')
  } finally {
    loading.value = false
  }
}

function logoutProfile() {
  profileId.value = ''
  creatorKey.value = ''
  profileName.value = ''
  items.value = []
  localStorage.removeItem('household_profile')
}

async function handleExtend() {
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/extend?${profileQuery()}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ creator_key: creatorKey.value, expires_in: extendDays.value })
    })
    const data = await res.json()
    if (data.code === 0) {
      ElMessage.success('延期成功')
      showExtendDialog.value = false
    } else {
      ElMessage.error(data.error || '延期失败')
    }
  } catch (e) {
    ElMessage.error('延期失败')
  }
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
      if (data.code === 0) {
        ElMessage.success('删除成功')
        logoutProfile()
      } else {
        ElMessage.error(data.error || '删除失败')
      }
    } catch (e) {
      ElMessage.error('删除失败')
    }
  }).catch(() => {})
}

// 物品操作
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
    }
  } catch (e) {
    console.error('加载物品失败:', e)
  }
}

async function loadTemplates() {
  try {
    const res = await fetch(`${API_BASE}/household/templates`)
    const data = await res.json()
    if (data.code === 0) {
      templates.value = data.data || []
      templatesByCategory.value = data.by_category || {}
    }
  } catch (e) {
    console.error('加载模板失败:', e)
  }
}

async function checkAI() {
  try {
    const res = await fetch(`${API_BASE}/household/ai/check`)
    const data = await res.json()
    aiEnabled.value = data.enabled
  } catch (e) {
    aiEnabled.value = false
  }
}

function handleSearch() {}

function isLowStock(item) {
  return item.quantity <= item.min_quantity
}

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

async function handleAdd() {
  if (!addForm.value.name) {
    ElMessage.warning('请输入物品名称')
    return
  }

  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items?${profileQuery()}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(addForm.value)
    })
    const data = await res.json()
    if (data.code === 0) {
      ElMessage.success('添加成功')
      showAddDialog.value = false
      addForm.value = {
        name: '',
        category: '其他',
        quantity: 1,
        unit: '个',
        min_quantity: 1,
        location: '',
        expiry_date: '',
        expiry_days: 0,
        notes: ''
      }
      await loadItems()
    } else {
      ElMessage.error(data.error || '添加失败')
    }
  } catch (e) {
    ElMessage.error('添加失败')
  }
}

function addFromTemplate(tpl) {
  addForm.value = {
    name: tpl.name,
    category: tpl.category,
    quantity: 1,
    unit: tpl.unit,
    min_quantity: tpl.default_min_quantity || 1,
    location: '',
    expiry_date: '',
    expiry_days: tpl.default_expiry_days || 0,
    notes: ''
  }
  showTemplateDialog.value = false
  showAddDialog.value = true
}

async function useItem(item, amount) {
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items/${item.id}/use?${profileQuery()}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount })
    })
    const data = await res.json()
    if (data.code === 0) {
      await loadItems()
    }
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

async function restockItem(item, amount) {
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items/${item.id}/restock?${profileQuery()}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount })
    })
    const data = await res.json()
    if (data.code === 0) {
      await loadItems()
    }
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

async function openItem(item) {
  try {
    const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items/${item.id}/open?${profileQuery()}`, {
      method: 'POST'
    })
    const data = await res.json()
    if (data.code === 0) {
      ElMessage.success('已重置保质期')
      await loadItems()
    }
  } catch (e) {
    ElMessage.error('操作失败')
  }
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
      if (data.code === 0) {
        ElMessage.success('删除成功')
        await loadItems()
      }
    } catch (e) {
      ElMessage.error('删除失败')
    }
  }).catch(() => {})
}

async function handleAIAdd() {
  if (!aiText.value.trim()) {
    ElMessage.warning('请输入要添加的物品描述')
    return
  }

  aiLoading.value = true
  try {
    const res = await fetch(`${API_BASE}/household/ai/add`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        text: aiText.value,
        profile_id: profileId.value,
        creator_key: creatorKey.value
      })
    })
    const data = await res.json()
    if (data.code === 0) {
      aiResult.value = data.data || []
      ElMessage.success(`成功识别并添加 ${data.count} 个物品`)
      aiText.value = ''
      await loadItems()
    } else {
      ElMessage.error(data.error || 'AI 识别失败')
    }
  } catch (e) {
    ElMessage.error('AI 识别失败')
  } finally {
    aiLoading.value = false
  }
}

async function loadRestockSuggestions() {
  loadingRestock.value = true
  try {
    const res = await fetch(`${API_BASE}/household/ai/restock?profile_id=${encodeURIComponent(profileId.value)}&${profileQuery()}`)
    const data = await res.json()
    if (data.code === 0) {
      restockSuggestions.value = data.suggestions || []
    }
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    loadingRestock.value = false
  }
}

async function quickRestock(item) {
  await restockItem({ id: item.id }, 1)
  restockSuggestions.value = restockSuggestions.value.filter(i => i.id !== item.id)
}

// 对话功能
async function sendChatMessage() {
  if (!chatInput.value.trim() || chatLoading.value) return

  const message = chatInput.value.trim()
  chatInput.value = ''

  // 添加用户消息
  chatHistory.value.push({ role: 'user', content: message })
  scrollToBottom()

  chatLoading.value = true
  try {
    const res = await fetch(`${API_BASE}/household/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        message,
        profile_id: profileId.value,
        creator_key: creatorKey.value
      })
    })
    const data = await res.json()
    if (data.code === 0) {
      // 尝试解析 JSON 字符串，提取 reply 字段
      let replyContent = data.reply
      try {
        // 去除 markdown 代码块标记
        let jsonStr = replyContent
        if (typeof jsonStr === 'string') {
          // 去除 ```json 或 ``` 标记
          jsonStr = jsonStr.replace(/```json\s*/g, '').replace(/```\s*/g, '').trim()
        }
        // 尝试解析 JSON
        if (jsonStr.startsWith('{')) {
          const parsed = JSON.parse(jsonStr)
          replyContent = parsed.reply || parsed.content || replyContent
        }
      } catch (e) {
        // 解析失败，使用原始内容
      }

      // 添加助手回复
      chatHistory.value.push({ role: 'assistant', content: replyContent })

      // 如果有物品变更，刷新列表
      if (data.items_added && data.items_added.length > 0) {
        await loadItems()
      }
    } else {
      chatHistory.value.push({ role: 'assistant', content: '抱歉，出了点问题：' + (data.error || '未知错误') })
    }
  } catch (e) {
    chatHistory.value.push({ role: 'assistant', content: '网络错误，请稍后重试' })
  } finally {
    chatLoading.value = false
    scrollToBottom()
  }
}

function scrollToBottom() {
  setTimeout(() => {
    if (chatMessagesRef.value) {
      chatMessagesRef.value.scrollTop = chatMessagesRef.value.scrollHeight
    }
  }, 100)
}

function formatMessage(text) {
  // 简单处理换行
  return text.replace(/\n/g, '<br>')
}

async function loadChatHistory() {
  if (!profileId.value) return

  try {
    const res = await fetch(`${API_BASE}/household/chat/history?profile_id=${profileId.value}`)
    const data = await res.json()
    if (data.code === 0 && data.history) {
      chatHistory.value = data.history.map(h => ({
        role: h.role,
        content: h.content
      }))
    }
  } catch (e) {
    console.error('加载对话历史失败:', e)
  }
}

async function clearChatHistory() {
  try {
    await fetch(`${API_BASE}/household/chat/history?profile_id=${profileId.value}`, {
      method: 'DELETE'
    })
    chatHistory.value = []
    ElMessage.success('对话历史已清除')
  } catch (e) {
    ElMessage.error('清除失败')
  }
}

// 语音输入
function toggleVoice() {
  if (isRecording.value) {
    stopRecording()
  } else {
    startRecording()
  }
}

function startRecording() {
  // 检查浏览器支持
  const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition
  if (!SpeechRecognition) {
    ElMessage.warning('当前浏览器不支持语音输入')
    return
  }

  recognition = new SpeechRecognition()
  recognition.lang = 'zh-CN'
  recognition.continuous = false
  recognition.interimResults = true

  recognition.onresult = (event) => {
    const transcript = Array.from(event.results)
      .map(result => result[0])
      .map(result => result.transcript)
      .join('')

    chatInput.value = transcript
  }

  recognition.onerror = (event) => {
    console.error('语音识别错误:', event.error)
    isRecording.value = false
    if (event.error !== 'no-speech') {
      ElMessage.warning('语音识别出错')
    }
  }

  recognition.onend = () => {
    isRecording.value = false
  }

  recognition.start()
  isRecording.value = true
}

function stopRecording() {
  if (recognition) {
    recognition.stop()
    isRecording.value = false
  }
}

// 扫码功能
import { Html5Qrcode } from 'html5-qrcode'

async function startScan() {
  if (isScanning.value) return

  try {
    html5QrCode = new Html5Qrcode('qr-reader')
    isScanning.value = true

    await html5QrCode.start(
      { facingMode: 'environment' },
      {
        fps: 10,
        qrbox: { width: 250, height: 250 }
      },
      (decodedText) => {
        scanResult.value = decodedText
        stopScan()
      },
      (errorMessage) => {
        // 忽略扫描错误
      }
    )
  } catch (err) {
    console.error('启动扫码失败:', err)
    ElMessage.error('无法启动摄像头，请确保已授权')
    isScanning.value = false
  }
}

function stopScan() {
  if (html5QrCode && isScanning.value) {
    html5QrCode.stop().then(() => {
      html5QrCode = null
      isScanning.value = false
    }).catch(err => {
      console.error('停止扫码失败:', err)
      isScanning.value = false
    })
  }
}

async function handleScanResult() {
  if (!scanResult.value) return

  // 使用 AI 智能解析条码
  if (aiEnabled.value) {
    aiLoading.value = true
    try {
      const res = await fetch(`${API_BASE}/household/ai/add`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          text: '添加商品：' + scanResult.value,
          profile_id: profileId.value,
          creator_key: creatorKey.value
        })
      })
      const data = await res.json()
      if (data.code === 0) {
        ElMessage.success(`成功添加 ${data.count} 个物品`)
        await loadItems()
        showScanDialog.value = false
        scanResult.value = ''
      } else {
        // 如果 AI 解析失败，直接添加为普通物品
        await addScannedItem(scanResult.value)
      }
    } catch (e) {
      await addScannedItem(scanResult.value)
    } finally {
      aiLoading.value = false
    }
  } else {
    await addScannedItem(scanResult.value)
  }
}

async function addScannedItem(code) {
  // 尝试调用后端接口解析条码
  try {
    const res = await fetch(`${API_BASE}/household/barcode/lookup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ barcode: code })
    })
    const data = await res.json()
    if (data.code === 0 && data.name) {
      // 后端返回了物品信息
      addForm.value = {
        name: data.name,
        category: data.category || '其他',
        quantity: 1,
        unit: data.unit || '个',
        min_quantity: 1,
        location: '',
        expiry_date: '',
        expiry_days: 0,
        notes: '条码: ' + code
      }
    } else {
      // 后端没有该条码，提示用户手动输入
      addForm.value = {
        name: '',
        category: '其他',
        quantity: 1,
        unit: '个',
        min_quantity: 1,
        location: '',
        expiry_date: '',
        expiry_days: 0,
        notes: '条码: ' + code
      }
      ElMessage.info('未找到该条码对应的商品，请手动输入名称')
      showScanDialog.value = false
      showAddDialog.value = true
      scanResult.value = ''
      return
    }
  } catch (e) {
    // 网络错误，使用默认信息
    addForm.value = {
      name: '',
      category: '其他',
      quantity: 1,
      unit: '个',
      min_quantity: 1,
      location: '',
      expiry_date: '',
      expiry_days: 0,
      notes: '条码: ' + code
    }
    ElMessage.info('请手动输入商品名称')
    showScanDialog.value = false
    showAddDialog.value = true
    scanResult.value = ''
    return
  }

  showScanDialog.value = false
  scanResult.value = ''
  showAddDialog.value = true
}

// 物品二维码生成
function showItemQR(item) {
  qrItem.value = item
  showQRDialog.value = true

  // 等待对话框渲染完成后生成二维码
  setTimeout(() => {
    generateQRCode(item)
  }, 300)
}

function generateQRCode(item) {
  const container = document.getElementById('qr-code生成')
  if (!container) return

  // 清除之前的二维码
  container.innerHTML = ''

  // 生成包含物品信息的 URL
  const qrData = JSON.stringify({
    type: 'household_item',
    id: item.id,
    name: item.name,
    category: item.category
  })

  // 使用 QRCode.js 生成二维码
  import('qrcode').then(({ default: QRCode }) => {
    QRCode.toCanvas(qrData, { width: 200, margin: 2 }, (err, canvas) => {
      if (err) {
        console.error('生成二维码失败:', err)
        return
      }
      container.appendChild(canvas)
    })
  })
}

function downloadQR() {
  const container = document.getElementById('qr-code生成')
  if (!container || !qrItem.value) return

  const canvas = container.querySelector('canvas')
  if (!canvas) return

  const link = document.createElement('a')
  link.download = `物品_${qrItem.value.name}_二维码.png`
  link.href = canvas.toDataURL('image/png')
  link.click()
  ElMessage.success('二维码已下载')
}

// 小票 OCR 识别
function handleReceiptFileChange(file) {
  const reader = new FileReader()
  reader.onload = (e) => {
    receiptImage.value = e.target.result
  }
  reader.readAsDataURL(file.raw)
}

function clearReceipt() {
  receiptImage.value = ''
  receiptItems.value = []
}

async function recognizeReceipt() {
  if (!receiptImage.value) return

  receiptLoading.value = true
  try {
    // 调用后端小票识别接口
    const res = await fetch(`${API_BASE}/household/ocr/receipt`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        image: receiptImage.value,
        profile_id: profileId.value,
        creator_key: creatorKey.value
      })
    })
    const data = await res.json()
    if (data.code === 0) {
      receiptItems.value = data.items || []
      if (data.items.length === 0) {
        ElMessage.warning('未识别到商品，请尝试重新拍照')
      } else {
        ElMessage.success(`识别到 ${data.items.length} 个商品`)
      }
    } else {
      ElMessage.error(data.error || '识别失败')
    }
  } catch (e) {
    console.error('小票识别失败:', e)
    ElMessage.error('识别失败，请稍后重试')
  } finally {
    receiptLoading.value = false
  }
}

function removeReceiptItem(index) {
  receiptItems.value.splice(index, 1)
}

async function addReceiptItemsToLibrary() {
  if (receiptItems.value.length === 0) return

  let addedCount = 0
  for (const item of receiptItems.value) {
    try {
      const res = await fetch(`${API_BASE}/household/profile/${profileId.value}/items?${profileQuery()}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: item.name,
          category: item.category || '其他',
          quantity: item.quantity || 1,
          unit: item.unit || '个',
          min_quantity: 1,
          notes: '从小票识别'
        })
      })
      const data = await res.json()
      if (data.code === 0) {
        addedCount++
      }
    } catch (e) {
      console.error('添加物品失败:', e)
    }
  }

  if (addedCount > 0) {
    ElMessage.success(`成功添加 ${addedCount} 个物品`)
    await loadItems()
    showReceiptDialog.value = false
    clearReceipt()
  } else {
    ElMessage.error('添加失败')
  }
}

// 购物清单功能
function generateShoppingList() {
  shoppingList.value = []

  items.value.forEach(item => {
    let reason = ''
    if (item.quantity <= item.min_quantity) {
      reason = `库存不足（当前 ${item.quantity}${item.unit}，需要 ${item.min_quantity}${item.unit}）`
    } else if (item.expiry_date && item.expiry_days > 0) {
      const expiryDate = new Date(item.expiry_date)
      expiryDate.setDate(expiryDate.getDate() + item.expiry_days)
      const daysUntil = Math.ceil((expiryDate - new Date()) / (1000 * 60 * 60 * 24))
      if (daysUntil <= 0) {
        reason = '已过期'
      } else if (daysUntil <= 7) {
        reason = `即将过期（还剩 ${daysUntil} 天）`
      }
    }

    if (reason) {
      shoppingList.value.push({
        id: item.id,
        name: item.name,
        category: item.category,
        reason
      })
    }
  })

  // 按分类排序
  shoppingList.value.sort((a, b) => a.category.localeCompare(b.category))
}

function removeFromShoppingList(index) {
  shoppingList.value.splice(index, 1)
}

function copyShoppingList() {
  let text = '购物清单\n'
  text += '='.repeat(20) + '\n\n'

  shoppingList.value.forEach((item, idx) => {
    text += `${idx + 1}. ${item.name} (${item.category})\n`
    text += `   ${item.reason}\n\n`
  })

  text += `共计 ${shoppingList.value.length} 项`

  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('清单已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

function exportShoppingList(format) {
  let content = ''
  let filename = ''
  let mimeType = 'text/plain'

  if (format === 'text') {
    content = '购物清单\n'
    content += '='.repeat(20) + '\n\n'
    shoppingList.value.forEach((item, idx) => {
      content += `${idx + 1}. ${item.name} (${item.category})\n`
      content += `   ${item.reason}\n\n`
    })
    content += `\n共计 ${shoppingList.value.length} 项`
    filename = '购物清单.txt'
  }

  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
  ElMessage.success('清单已导出')
}

async function shareShoppingList() {
  // 生成分享数据
  const shareData = {
    type: 'household_shopping_list',
    profile_id: profileId.value,
    items: shoppingList.value.map(item => ({
      name: item.name,
      category: item.category,
      reason: item.reason
    }))
  }

  try {
    const res = await fetch(`${API_BASE}/shorturl`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        original_url: `${window.location.origin}/household?share=${btoa(JSON.stringify(shareData))}`
      })
    })
    const data = await res.json()
    if (data.id) {
      const shareUrl = `${window.location.origin}/s/${data.id}`
      navigator.clipboard.writeText(shareUrl).then(() => {
        ElMessage.success('分享链接已复制到剪贴板')
      }).catch(() => {
        ElMessage.info(`分享链接: ${shareUrl}`)
      })
    } else {
      ElMessage.error('生成分享链接失败')
    }
  } catch (e) {
    console.error('分享失败:', e)
    ElMessage.error('分享失败')
  }
}

// 导出功能
function copyExportData() {
  navigator.clipboard.writeText(exportPreview.value).then(() => {
    ElMessage.success('已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

function downloadExportData() {
  let content = exportPreview.value
  let filename = ''
  let mimeType = ''

  if (exportFormat.value === 'json') {
    mimeType = 'application/json'
    filename = '物品清单.json'
  } else {
    mimeType = 'text/plain;charset=utf-8'
    filename = '物品清单.txt'
  }

  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
  ElMessage.success('文件已下载')
}

// 初始化
onMounted(async () => {
  // 尝试加载本地存储的档案
  const saved = localStorage.getItem('household_profile')
  if (saved) {
    try {
      const profile = JSON.parse(saved)
      // 验证档案是否有效
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
    } catch (e) {
      localStorage.removeItem('household_profile')
    }
  }

  await Promise.all([loadTemplates(), checkAI()])
})

// 监听抽屉打开，加载对话历史
watch(showChatDrawer, (val) => {
  if (val && profileId.value) {
    loadChatHistory()
  }
})

// 监听扫码对话框打开
watch(showScanDialog, (val) => {
  if (val) {
    setTimeout(() => {
      startScan()
    }, 300)
  } else {
    stopScan()
  }
})

// 监听购物清单对话框打开
watch(showShoppingListDialog, (val) => {
  if (val) {
    generateShoppingList()
  }
})
</script>

<style scoped>
.household-page {
  padding: 20px;
}

.welcome-section {
  max-width: 500px;
  margin: 40px auto;
}

.welcome-card {
  background: #fff;
  border-radius: 8px;
  padding: 30px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.welcome-card h3 {
  margin: 0 0 10px 0;
  color: #303133;
}

.welcome-card p {
  color: #909399;
  margin: 0;
}

.stats-section {
  margin-bottom: 20px;
}

.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.stat-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  display: flex;
  align-items: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.stat-card.warning {
  border-left: 4px solid #e6a23c;
}

.stat-card.danger, .stat-card.error {
  border-left: 4px solid #f56c6c;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  font-size: 24px;
}

.stat-icon.total {
  background: #ecf5ff;
  color: #409eff;
}

.stat-icon.low-stock {
  background: #fef0f0;
  color: #f56c6c;
}

.stat-icon.expiring {
  background: #fdf6ec;
  color: #e6a23c;
}

.stat-icon.expired {
  background: #fef0f0;
  color: #f56c6c;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: #303133;
}

.stat-label {
  font-size: 14px;
  color: #909399;
}

.profile-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #fff;
  padding: 12px 20px;
  border-radius: 8px;
  margin-bottom: 20px;
}

.profile-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  flex-wrap: wrap;
  gap: 12px;
}

.toolbar-left, .toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.items-section {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.item-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.quantity-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.quantity-value {
  min-width: 50px;
  text-align: center;
  font-weight: bold;
}

.quantity-value.low-stock {
  color: #f56c6c;
}

.text-danger {
  color: #f56c6c;
}

.text-warning {
  color: #e6a23c;
}

.text-muted {
  color: #c0c4cc;
}

.template-grid {
  max-height: 400px;
  overflow-y: auto;
}

.template-category {
  margin-bottom: 16px;
}

.category-title {
  font-weight: bold;
  margin-bottom: 8px;
  color: #409eff;
}

.template-items {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.template-tag {
  cursor: pointer;
}

.template-tag:hover {
  opacity: 0.8;
}

.ai-add-section, .ai-restock-section {
  padding: 10px 0;
}

.ai-result {
  margin-top: 16px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.ai-result-title {
  font-weight: bold;
  margin-bottom: 8px;
}

.restock-list {
  margin-top: 16px;
}

.restock-item {
  margin-bottom: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.restock-info {
  flex: 1;
}

.restock-name {
  font-weight: bold;
}

.restock-reason {
  font-size: 12px;
  color: #909399;
}

/* AI 对话助手样式 */
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

.ai-chat-fab:hover {
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(103, 194, 58, 0.5);
}

.chat-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  background: #f5f7fa;
}

.chat-welcome {
  text-align: center;
  padding: 40px 20px;
  color: #606266;
}

.chat-welcome p {
  margin: 10px 0;
}

.chat-welcome .hint {
  color: #909399;
  font-size: 12px;
  margin-top: 20px;
}

.chat-welcome ul {
  text-align: left;
  padding-left: 20px;
  font-size: 13px;
  color: #606266;
}

.chat-welcome li {
  margin: 8px 0;
}

.chat-message {
  display: flex;
  margin-bottom: 16px;
}

.chat-message.user {
  flex-direction: row-reverse;
}

.message-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #409eff;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.chat-message.assistant .message-avatar {
  background: #67c23a;
}

.message-content {
  max-width: 75%;
  margin: 0 10px;
}

.message-text {
  padding: 10px 14px;
  border-radius: 8px;
  background: white;
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;
}

.chat-message.user .message-text {
  background: #409eff;
  color: white;
}

.message-text.typing {
  color: #909399;
}

.message-text .dot {
  animation: dot 1.4s infinite;
}

.message-text .dot:nth-child(2) {
  animation-delay: 0.2s;
}

.message-text .dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes dot {
  0%, 20% { content: '.'; }
  40% { content: '..'; }
  60%, 100% { content: '...'; }
}

.chat-input-area {
  display: flex;
  gap: 10px;
  padding: 12px;
  background: white;
  border-top: 1px solid #ebeef5;
}

.chat-input-area .el-input {
  flex: 1;
}

.voice-active {
  color: #f56c6c !important;
  background: #fef0f0;
}

.chat-actions {
  padding: 8px 12px;
  background: #f5f7fa;
  border-top: 1px solid #ebeef5;
  text-align: center;
}

/* 扫码样式 */
.scan-container {
  min-height: 300px;
}

.qr-reader {
  width: 100%;
  border-radius: 8px;
  overflow: hidden;
}

.scan-result {
  margin-top: 16px;
}

.scan-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
}

.qr-code生成 {
  display: flex;
  justify-content: center;
  align-items: center;
}

/* 小票 OCR 样式 */
.receipt-container {
  min-height: 300px;
}

.receipt-upload {
  width: 100%;
  min-height: 300px;
  border: 2px dashed #dcdfe6;
  border-radius: 8px;
  cursor: pointer;
  transition: border-color 0.3s;
}

.receipt-upload:hover {
  border-color: #409eff;
}

.upload-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  color: #909399;
}

.upload-placeholder p {
  margin: 10px 0;
}

.upload-placeholder .hint {
  font-size: 12px;
}

.receipt-preview {
  text-align: center;
}

.receipt-actions {
  margin-top: 16px;
  display: flex;
  justify-content: center;
  gap: 12px;
}

.receipt-result {
  margin-top: 20px;
}

.matched-items {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.receipt-item-tag {
  padding: 8px 12px;
}

.matched-badge {
  font-size: 10px;
  color: #67c23a;
  margin-left: 4px;
}

.receipt-result-actions {
  margin-top: 16px;
  text-align: center;
}

/* 购物清单样式 */
.shopping-list-container {
  min-height: 300px;
}

.shopping-list-header {
  margin-bottom: 16px;
}

.shopping-list-content {
  margin: 16px 0;
}

.shopping-list-actions {
  margin-top: 16px;
  display: flex;
  justify-content: center;
  gap: 12px;
}

/* 导出样式 */
.export-container {
  min-height: 300px;
}

.export-preview {
  margin-top: 16px;
}

.preview-header {
  font-weight: bold;
  margin-bottom: 8px;
}

.preview-content {
  background: #f5f7fa;
  padding: 12px;
  border-radius: 4px;
  max-height: 200px;
  overflow: auto;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}

.export-actions {
  margin-top: 16px;
  display: flex;
  justify-content: center;
  gap: 12px;
}

@media (max-width: 768px) {
  .stats-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .toolbar-left, .toolbar-right {
    flex-wrap: wrap;
  }
}
</style>
