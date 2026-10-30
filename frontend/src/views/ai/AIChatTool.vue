<template>
  <div class="tool-container chatplus-page">
    <!-- ═══════════ 密码门 ═══════════ -->
    <div v-if="!auth.isAuthenticated.value" class="auth-gate">
      <div class="auth-card">
        <div class="auth-logo">✦</div>
        <h3>AI Chat</h3>
        <p>局域网 Ollama + MiniMax 双引擎对话</p>
        <el-form @submit.prevent="auth.login" class="auth-form">
          <el-input
            v-model="auth.passwordInput.value"
            type="password"
            show-password
            placeholder="请输入访问密码"
            size="large"
            @keyup.enter="auth.login"
          />
          <el-button type="primary" size="large" :loading="auth.loggingIn.value" class="auth-btn" @click="auth.login">
            进入
          </el-button>
        </el-form>
      </div>
    </div>

    <!-- ═══════════ 主界面 ═══════════ -->
    <div v-else class="chatplus-shell">
      <!-- 移动端抽屉遮罩 -->
      <div v-if="sidebarOpen && isMobileViewport" class="cp-backdrop" @click="sidebarOpen = false"></div>
      <!-- 侧栏：会话列表 -->
      <aside class="cp-sidebar" :class="{ open: sidebarOpen }">
        <button class="cp-new-chat" @click="newConversation">
          <span class="ic">＋</span>
          <span v-if="!sidebarCollapsed">新对话</span>
        </button>
        <div class="cp-conv-list">
          <div
            v-for="conv in conversations"
            :key="conv.id"
            class="cp-conv-item"
            :class="{ active: conv.id === currentConvId }"
            :title="conv.title"
            @click="switchConversation(conv.id)"
          >
            <span class="conv-title">{{ conv.title || '新对话' }}</span>
            <button class="conv-del" title="删除" @click.stop="deleteConversation(conv.id)">
              <el-icon><Delete /></el-icon>
            </button>
          </div>
          <div v-if="conversations.length === 0" class="cp-conv-empty">暂无会话</div>
        </div>
        <div class="cp-sidebar-footer">
          <button class="cp-side-btn" @click="toggleSidebar" title="关闭侧栏">
            <span class="ic">☰</span>
            <span>关闭侧栏</span>
          </button>
          <button class="cp-side-btn" @click="auth.logout">
            <span class="ic">⎋</span>
            <span>退出登录</span>
          </button>
        </div>
      </aside>

      <!-- 主区域 -->
      <main class="cp-main">
        <!-- 顶栏 -->
        <header class="cp-topbar">
          <button v-if="!sidebarOpen" class="cp-icon-btn" @click="toggleSidebar" title="打开侧栏">☰</button>
          <el-select
            v-model="settings.model"
            placeholder="选择模型"
            size="default"
            class="cp-model-select"
            :loading="modelsLoading"
          >
            <el-option-group v-for="g in modelGroups" :key="g.label" :label="g.label">
              <el-option
                v-for="m in g.items"
                :key="m.value"
                :value="m.value"
                :label="m.label"
              >
                <span class="model-option-label">{{ m.label }}</span>
                <span v-if="m.tag" class="model-option-tag">{{ m.tag }}</span>
              </el-option>
            </el-option-group>
          </el-select>
          <span v-if="engineStatus" class="engine-badge" :class="engineStatus.type">{{ engineStatus.text }}</span>
          <div class="topbar-spacer"></div>
          <el-popover trigger="click" width="320" placement="bottom-end">
            <template #reference>
              <button class="cp-icon-btn" title="对话设置">⚙</button>
            </template>
            <div class="cp-settings-pop">
              <div class="setting-item">
                <label>系统提示词（仅对当前设置生效）</label>
                <el-input v-model="settings.systemPrompt" type="textarea" :rows="4" placeholder="可选：为 AI 设定角色，例如「你是一个严谨的技术助手」" />
              </div>
              <div class="setting-item">
                <label>随机性 Temperature：{{ settings.temperature.toFixed(1) }}</label>
                <el-slider v-model="settings.temperature" :min="0" :max="1.5" :step="0.1" />
              </div>
            </div>
          </el-popover>
        </header>

        <!-- 消息区 -->
        <div class="cp-messages" ref="messagesRef" @scroll="onScroll">
          <!-- 欢迎屏 -->
          <div v-if="messages.length === 0 && !streaming" class="cp-welcome">
            <div class="welcome-logo">✦</div>
            <h2>有什么可以帮你？</h2>
            <p class="welcome-sub">本地 Ollama 与 MiniMax 云端模型随时切换 · 对话已同步到服务端，换设备打开也是同一份</p>
            <div class="welcome-chips">
              <button v-for="s in SUGGESTIONS" :key="s" class="chip" @click="quickAsk(s)">{{ s }}</button>
            </div>
          </div>

          <!-- 消息列表 -->
          <template v-for="msg in displayMessages" :key="msg.id">
            <div class="cp-msg" :class="msg.role">
              <div class="msg-avatar" :class="msg.role">{{ msg.role === 'user' ? '你' : '✦' }}</div>
              <div class="msg-body">
                <!-- 思考过程（可折叠） -->
                <details v-if="msg.reasoning" class="msg-reasoning" :open="msg.streaming && !msg.content">
                  <summary>思考过程</summary>
                  <div class="reasoning-content">{{ msg.reasoning }}</div>
                </details>
                <div v-if="msg.images && msg.images.length" class="msg-images">
                  <img v-for="im in msg.images" :key="im.id" :src="im.url || im.dataUrl" alt="" />
                </div>
                <!-- 本地出图（ComfyUI）：进度条 + 结果图 -->
                <div v-if="msg.kind === 'image'" class="msg-gen">
                  <div v-if="msg.generating" class="gen-progress">
                    <div class="gen-bar"><i :style="{ width: Math.max(2, msg.progress || 0) + '%' }"></i></div>
                    <span class="gen-phase">{{ msg.phase || '准备中' }} · {{ msg.progress || 0 }}%<template v-if="msg.elapsedMs"> · 已等 {{ Math.round(msg.elapsedMs / 1000) }}s</template></span>
                  </div>
                  <div v-if="msg.genImages && msg.genImages.length" class="gen-grid">
                    <figure v-for="im in msg.genImages" :key="im.url" class="gen-item">
                      <img :src="im.url" :alt="msg.prompt || 'AI 生成图片'" loading="lazy" @click="previewImage(im.url)" />
                      <figcaption>
                        <a :href="im.url" target="_blank" rel="noopener" :download="im.filename || 'aichat.png'">下载原图</a>
                        <span v-if="im.width && im.height">{{ im.width }}×{{ im.height }}</span>
                        <span v-if="msg.genModel" class="gen-model-tag">{{ msg.genModel }}</span>
                      </figcaption>
                    </figure>
                  </div>
                  <div v-if="msg.genError" class="gen-error">⚠ {{ msg.genError }}</div>
                </div>
                <div class="msg-content" :class="{ streaming: msg.streaming }" v-html="renderMarkdown(msg.content)"></div>
                <!-- 中断标记 + 续写入口 -->
                <div v-if="msg.interrupted && !msg.failed" class="msg-interrupted">
                  <span>⚠ 连接中断，回复不完整</span>
                  <button v-if="msg === lastAssistant && !streaming && msg.content" class="continue-btn" @click="continueGeneration">继续生成</button>
                </div>
                <!-- 弱网自动重连提示 -->
                <div v-if="msg.reconnecting" class="msg-interrupted">
                  <span>⚠ 网络不稳，正在自动续传…</span>
                </div>
                <!-- 任务失败：把服务端的原因如实显示出来，可一键重试 -->
                <div v-if="msg.failed" class="msg-failed">
                  <span>⚠ 生成失败：{{ msg.failed }}</span>
                  <button v-if="msg === lastAssistant && !streaming" class="continue-btn" @click="regenerate">重试</button>
                </div>
                <!-- 操作行 -->
                <div v-if="!msg.streaming && !msg.generating" class="msg-actions">
                  <button v-if="msg.role === 'assistant' && msg.kind !== 'image'" class="action-btn" title="复制回复" @click="copyMessage(msg)">
                    <el-icon><CopyDocument /></el-icon>
                  </button>
                  <button
                    v-if="msg.role === 'assistant' && msg.kind === 'image' && !streaming"
                    class="action-btn"
                    title="重新生成这张图"
                    @click="regenerateImage(msg)"
                  >
                    <el-icon><RefreshRight /></el-icon>
                  </button>
                  <button
                    v-if="msg.role === 'assistant' && msg.kind !== 'image' && msg === lastAssistant && !streaming"
                    class="action-btn"
                    title="重新生成"
                    @click="regenerate"
                  >
                    <el-icon><RefreshRight /></el-icon>
                  </button>
                </div>
              </div>
            </div>
          </template>
        </div>

        <!-- 输入区 -->
        <footer class="cp-composer" @drop="onDrop" @dragover.prevent="dragOver = true" @dragleave="dragOver = false">
          <div v-if="errorMsg" class="cp-error">
            <el-icon><WarningFilled /></el-icon>
            <span>{{ errorMsg }}</span>
            <button class="err-close" @click="errorMsg = ''">✕</button>
          </div>

          <!-- 待发送图片缩略图 -->
          <div v-if="pendingImages.length || pendingMask" class="img-thumbs">
            <div v-for="im in pendingImages" :key="im.id" class="img-thumb">
              <img :src="im.dataUrl" alt="" />
              <span class="img-kb">{{ im.kb }}KB</span>
              <button class="img-del" title="移除" @click="removePendingImage(im.id)">✕</button>
            </div>
            <div v-if="pendingMask" class="img-thumb mask-thumb" title="遮罩：白色=要重绘区域">
              <img :src="pendingMask.dataUrl" alt="" />
              <span class="img-kb">遮罩</span>
              <button class="img-del" title="移除遮罩" @click="pendingMask = null">✕</button>
            </div>
            <span class="img-count">{{ pendingImages.length }}/{{ imageLimit }}</span>
          </div>
          <div v-if="imageContextWarn" class="img-warn">⚠ 图片总体积较大（约 {{ pendingImagesKB }}KB），视觉模型上下文可能吃紧，建议减少图片或换更大显存的模型</div>

          <div class="composer-box" :class="{ focused: composerFocused, dragover: dragOver }" @paste="onPaste">
            <button
              class="gen-btn"
              :class="{ active: imageMode }"
              :disabled="!imageAvailable"
              :title="imageAvailable ? (imageMode ? '退出出图，回到对话' : '出图（本地 ComfyUI / 在线 MiniMax）') : '出图不可用：本地 ComfyUI 与在线 MiniMax 都未配置'"
              @click="toggleImageMode"
            >🎨</button>
            <button class="img-btn" :class="{ active: canAttachImage }" :disabled="!canAttachImage" :title="attachTitle" @click="openImagePicker">
              <el-icon><Picture /></el-icon>
            </button>
            <button
              v-if="genModeNeedsMask"
              class="img-btn"
              :class="{ active: !!pendingMask }"
              :title="pendingMask ? '更换遮罩（白色=要重绘区域）' : '添加遮罩（白色=要重绘区域）'"
              @click="openMaskPicker"
            >
              <el-icon><Brush /></el-icon>
            </button>
            <el-input
              v-model="userInput"
              type="textarea"
              :autosize="{ minRows: 1, maxRows: 8 }"
              :placeholder="composerPlaceholder"
              resize="none"
              @focus="composerFocused = true"
              @blur="composerFocused = false"
              @keydown="onKeydown"
              class="composer-input"
            />
            <button v-if="streaming || imageGenerating" class="send-btn stop" title="停止生成" @click="stopGeneration">
              <span class="stop-square"></span>
            </button>
            <button v-else class="send-btn" :disabled="sendDisabled" title="发送" @click="sendMessage">
              <el-icon><Promotion /></el-icon>
            </button>
          </div>
          <div class="composer-hint" :class="{ 'has-image': imageMode }">
            <template v-if="imageMode">🎨 出图：
              <el-select
                v-model="genMode"
                class="gen-mode-select"
                size="small"
                placeholder="模式"
              >
                <el-option
                  v-for="m in availableGenModes"
                  :key="m.value"
                  :label="m.label"
                  :value="m.value"
                >
                  <span class="mo-label">{{ m.label }}</span>
                  <span class="mo-meta">{{ m.hint }}</span>
                </el-option>
              </el-select>
              <span class="gen-mode-hint">{{ genModeMeta.hint }}</span>
              <el-select
                v-model="settings.imageModel"
                class="img-model-select"
                size="small"
                :loading="imageModelsLoading"
                placeholder="模型"
              >
                <el-option
                  v-for="m in imageModels"
                  :key="m.id"
                  :label="m.label"
                  :value="m.id"
                  :disabled="m.available === false"
                >
                  <span class="mo-label">{{ m.label }}</span>
                  <span class="mo-meta">{{ m.available === false ? '不可用' : m.speed_hint }}</span>
                </el-option>
              </el-select>
              <!-- 云端模型可以一次出多张（本地 ComfyUI 单卡串行，只出 1 张） -->
              <el-select
                v-if="currentImageModel?.provider === 'minimax'"
                v-model="settings.imageCount"
                class="gen-mode-select"
                size="small"
                placeholder="张数"
              >
                <el-option v-for="n in onlineImageCounts" :key="n" :label="n + ' 张'" :value="n" />
              </el-select>
              <template v-if="currentImageModel?.provider === 'minimax'">
                · 云端 · 每次最多 {{ imageStatus.minimax_max_n || 1 }} 张<template v-if="currentImageModel?.speed_hint"> · {{ currentImageModel.speed_hint }}</template>
              </template>
              <template v-else>
                · {{ currentImageModel?.steps || imageStatus.steps }} 步<template v-if="currentImageModel?.speed_hint"> · {{ currentImageModel.speed_hint }}</template>
              </template>
              <template v-if="!currentImageModel?.supports_edit"> · 不支持改图</template>
              <template v-if="imageNeedsRef"> · 需附参考图</template>
              ·
              <button
                class="opt-pill"
                :class="{ busy: promptOptLoading }"
                :disabled="promptOptDisabled"
                :title="promptOptDisabledReason || '让 AI 把这句话改写成更适合出图的提示词，结果可再改'"
                @click="openPromptOptimize"
              >✨ {{ promptOptLoading ? '优化中…' : '优化提示词' }}</button>
            </template>
            <template v-else-if="selectedModelVision">📷 已启用视觉：图片将压缩至最长边 ≤{{ IMG_MAX_DIM }}px 后发送 · </template>
            内容由 AI 生成，请仔细甄别 · MiniMax 密钥保存在服务端，不会暴露给浏览器
          </div>
          <input ref="imageInputRef" type="file" accept="image/*" multiple hidden @change="e => { addImageFiles(e.target.files); e.target.value = '' }" />
          <input ref="maskInputRef" type="file" accept="image/*" hidden @change="e => { addMaskFile(e.target.files); e.target.value = '' }" />
        </footer>
      </main>

      <!-- ═══════════ AI 优化提示词 ═══════════ -->
      <!-- 只是「改写建议」：结果永远落回输入框，用户可以随便改、也可以直接放弃 -->
      <el-dialog
        v-model="promptOptOpen"
        title="AI 优化提示词"
        width="min(620px, 92vw)"
        append-to-body
        :close-on-click-modal="false"
        class="prompt-opt-dialog"
        @closed="cancelPromptOptimize"
      >
        <div class="po-body">
          <div class="po-head">
            <div class="po-model">
              <span class="po-label">优化模型</span>
              <el-select
                v-model="settings.promptOptimizeModel"
                class="po-model-select"
                size="small"
                placeholder="跟随聊天模型"
                @change="onPromptOptModelChange"
              >
                <el-option value="" label="跟随聊天模型" />
                <el-option-group v-for="g in modelGroups" :key="g.label" :label="g.label">
                  <el-option
                    v-for="m in g.items"
                    :key="m.value"
                    :value="m.value"
                    :label="m.label"
                  >
                    <span class="model-option-label">{{ m.label }}</span>
                    <span v-if="m.tag" class="model-option-tag">{{ m.tag }}</span>
                  </el-option>
                </el-option-group>
              </el-select>
              <span class="po-model-hint">
                当前：{{ promptOptEffectiveModelLabel }}<template v-if="promptOptWillSeeImage"> · 会看图</template>
              </span>
            </div>
            <div class="po-origin">
              <span class="po-label">你的原话</span>
              <span class="po-origin-text">{{ promptOptOriginal || '（只附了参考图，让 AI 先看懂画面再写）' }}</span>
            </div>
          </div>

          <div v-if="promptOptLoading" class="po-loading">
            <span class="po-spinner"></span>
            <span class="po-loading-main">
              正在改写提示词<span v-if="promptOptElapsed >= 2"> · 已等 {{ promptOptElapsed }}s</span>…
            </span>
            <span v-if="promptOptSlow" class="po-loading-sub">
              本地大模型偏慢，常见 30~120 秒。任务在服务端跑，现在关掉这个框也不会再报 524，会直接停掉。
            </span>
          </div>
          <div v-else-if="promptOptError" class="po-error">
            <span>⚠ {{ promptOptError }}</span>
          </div>
          <div v-else class="po-result">
            <span class="po-label">优化结果 · 可直接改</span>
            <el-input
              v-model="promptOptResult"
              type="textarea"
              :autosize="{ minRows: 4, maxRows: 12 }"
              resize="none"
              placeholder="AI 改写后的提示词会出现在这里"
            />
            <div class="po-meta">
              <span v-if="promptOptModelLabel">由 {{ promptOptModelLabel }} 生成</span>
              <span v-if="promptOptUsedImage"> · 已参考你附的图</span>
              <span v-else-if="promptOptImage"> · 当前模型看不了图，仅按文字改写</span>
              <span class="po-count">{{ promptOptResult.length }} 字</span>
            </div>
          </div>
        </div>
        <template #footer>
          <div class="po-footer">
            <button v-if="promptOptLoading" class="po-btn ghost" @click="stopPromptOptimize">停止</button>
            <button v-else class="po-btn ghost" @click="runPromptOptimize">重新优化</button>
            <div class="po-footer-right">
              <button class="po-btn" @click="promptOptOpen = false">放弃</button>
              <button class="po-btn" :disabled="!promptOptResult.trim()" @click="applyPromptOptimize">采纳到输入框</button>
              <button class="po-btn primary" :disabled="!promptOptResult.trim()" @click="applyPromptOptimizeAndSend">采纳并生成</button>
            </div>
          </div>
        </template>
      </el-dialog>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, nextTick, onMounted, onUnmounted, onActivated, onDeactivated, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Delete, CopyDocument, RefreshRight, Promotion, WarningFilled, Picture, Brush } from '@element-plus/icons-vue'
import MarkdownIt from 'markdown-it'
import hljs from '../../utils/highlight'
import 'highlight.js/styles/github-dark.css'
import texmath from 'markdown-it-texmath'
import katex from 'katex'
import 'katex/dist/katex.min.css'
import { API_BASE } from '../../api'
import { useAdminAuth } from '../../composables/useAdminAuth'
import { drainTaskFrames, applyTaskEvent } from './imageTaskStream'

// ══════════════ 常量 ══════════════
const CONV_LIST_KEY = 'chat-plus-conv-list'
const MSGS_KEY_PREFIX = 'chat-plus-msgs-'
const SETTINGS_KEY = 'chat-plus-settings'
const CURRENT_CONV_KEY = 'chat-plus-current-conv'

const SUGGESTIONS = [
  '用通俗的语言解释一下 Transformer 的注意力机制',
  '帮我写一个 Go 语言的并发安全 LRU 缓存',
  '把这段话润色得更专业：……',
  '给我一周减脂期的三餐搭配建议',
]

// ══════════════ 密码门 ══════════════
const auth = useAdminAuth({
  storageKey: 'chat-plus-access-token',
  verifyEndpoint: '/api/chat-plus/verify',
  credentialMode: 'body',
  headerName: 'X-Chat-Password',
  trustStored: true,
})

// ══════════════ Markdown 渲染 ══════════════
const md = new MarkdownIt({
  html: false, // 防 XSS：模型输出也可能被注入 HTML
  linkify: true,
  breaks: true,
  highlight(str, lang) {
    const langLabel = lang || 'text'
    const encoded = encodeURIComponent(str)
    let body
    if (lang && hljs.getLanguage(lang)) {
      try {
        body = hljs.highlight(str, { language: lang, ignoreIllegals: true }).value
      } catch { body = '' }
    }
    if (!body) body = md.utils.escapeHtml(str)
    return `<div class="code-block"><div class="code-head"><span class="code-lang">${md.utils.escapeHtml(langLabel)}</span>` +
      `<button class="code-copy" data-code="${encoded}">复制</button></div>` +
      `<pre class="hljs"><code>${body}</code></pre></div>`
  },
})
md.use(texmath, { engine: katex, delimiters: ['dollars', 'brackets'], katexOptions: { throwOnError: false } })

function renderMarkdown(content) {
  if (!content) return ''
  return md.render(content)
}

// 代码块复制：事件委托（v-html 内容无法绑定局部事件）
function handleContentClick(e) {
  const btn = e.target.closest?.('.code-copy')
  if (!btn) return
  const code = decodeURIComponent(btn.dataset.code || '')
  navigator.clipboard.writeText(code).then(() => {
    btn.textContent = '已复制'
    setTimeout(() => { btn.textContent = '复制' }, 1500)
  })
}

// ══════════════ 会话与消息存储 ══════════════
// 架构（多端同步）：服务端 SQLite 是唯一真相，localStorage 只是「离线缓存 + 首屏秒开」。
// 之前全部只存 localStorage，等于每个浏览器一份私有历史，换设备打开就是空白。
const conversations = ref([])   // [{id,title,createdAt,updatedAt}]
const currentConvId = ref('')
const messages = ref([])        // 当前会话消息 [{id,role,content,reasoning,model,ts}]
const settings = ref({ model: '', systemPrompt: '', temperature: 0.7, imageMode: false, imageModel: '', imageCount: 1, genMode: '', promptOptimizeModel: '' })
const userInput = ref('')
const composerFocused = ref(false)
const streaming = ref(false)
const errorMsg = ref('')
const sidebarOpen = ref(true)
const isMobileViewport = ref(false)
const modelsLoading = ref(false)
const modelInfo = ref({ minimax: { available: false, models: [] }, ollama: { available: false, models: [], error: '' } })

// ══════════════ 图片输入（视觉模型） ══════════════
const MAX_IMAGES = 4            // 单条消息最多图片数
const IMG_MAX_DIM = 1024       // 压缩后最长边上限（控制视觉 token 体积，防上下文爆）
const IMG_QUALITY = 0.8        // JPEG 压缩质量
const IMG_TOTAL_BUDGET = 4     // 单次请求最多携带的历史+当前图片总数（硬上限，防多轮累积爆上下文）
const pendingImages = ref([])  // [{ id, dataUrl, w, h, kb }] 待发送图片（已压缩）
const dragOver = ref(false)
const imageInputRef = ref(null)
const maskInputRef = ref(null)         // 遮罩图文件选择器（局部重绘 / 外补 用）

// ══════════════ 本地出图（局域网 ComfyUI） ══════════════
// 与对话是两条独立链路：对话走 Ollama/MiniMax，出图走 ComfyUI 的扩散模型，
// 后端做成了「提交任务 + SSE 进度」，因为单张 1024² 要 3~5 分钟，同步等必超时。
// imageMode 挂在 settings 上，随设置一起持久化，刷新后仍是上次的模式。
const imageMode = computed({
  get: () => !!settings.value.imageMode,
  set: (v) => { settings.value.imageMode = !!v },
})
const imageGenerating = ref(false)    // 有出图任务在跑（用于禁用发送/显示停止）
const imageStatus = ref({ available: false, model: '', steps: 0, resolution: 0, error: '' })
let imageAbort = null                 // 出图任务的 AbortController

// 出图模式清单（与后端 buildComfyWorkflow 的分发对齐）。multi=允许多图，mask=需要遮罩图，
// ref=需要底图/参考图，hint 展示在工具栏。
const GEN_MODES = [
  { value: 't2i', label: '文生图', ref: false, multi: false, mask: false, hint: '凭文字生成' },
  { value: 'edit', label: '图生图', ref: true, multi: false, mask: false, hint: '参考图+指令改图' },
  { value: 'inpaint', label: '局部重绘', ref: true, multi: false, mask: true, hint: '画遮罩擦除/重绘' },
  { value: 'outpaint', label: '扩图', ref: true, multi: false, mask: true, hint: '自动向外扩展' },
  { value: 'restore', label: '修复', ref: true, multi: false, mask: false, hint: '老照片/人脸增强' },
  { value: 'multi_ref', label: '多图融合', ref: true, multi: true, mask: false, hint: '多张图内容结合' },
  { value: 'composite', label: '拼合', ref: true, multi: true, mask: false, hint: '多图拼成一张' },
]
const genMode = computed({
  get: () => settings.value.genMode || 't2i',
  set: (v) => { settings.value.genMode = v },
})
// 切模式时清空不兼容的待发图片/遮罩，避免脏状态
watch(genMode, () => {
  const multi = genModeMeta.value.multi
  if (!multi && pendingImages.value.length > 1) pendingImages.value = pendingImages.value.slice(0, 1)
  if (!genModeMeta.value.mask) pendingMask.value = null
})
// 当前模式允许的最大图片数：对话模式最多 MAX_IMAGES 张；出图模式按 genMode 细分
// —— 拼合/多图融合最多 2 张（按用户要求），单图改图类 1 张，文生图 0 张。
const imageLimit = computed(() => {
  if (!imageMode.value) return MAX_IMAGES
  if (genModeMeta.value.multi) return 2
  return genModeMeta.value.ref ? 1 : 0
})
const genModeMulti = computed(() => genModeMeta.value.multi)
// 该模式是否需要底图/参考图
const genModeNeedsRef = computed(() => genModeMeta.value.ref)
// 该模式是否需要遮罩（局部重绘 / 扩图）
const genModeNeedsMask = computed(() => genModeMeta.value.mask)
const pendingMask = ref(null)         // 遮罩图 { id, dataUrl, w, h }（局部重绘 / 扩图用，白色=要重绘区域）

// 可切换的出图模型（后端 GET /image/models）。一个模型 = 一整套工作流参数
// （unet + 文本编码器 + VAE + 采样步数），所以切换就等于换「换脑」，
// flux2 klein 4 步 20 秒出图，qwen-image 25 步质量更好但要 3~5 分钟。
const imageModels = ref([])
const imageModelsLoading = ref(false)
const currentImageModel = computed(() => imageModels.value.find(m => m.id === settings.value.imageModel) || null)
const usableImageModels = computed(() => imageModels.value.filter(m => m.available !== false))
// 云端模型（MiniMax image-01）只能文生图 / 图生图：其余模式（局部重绘、扩图、修复、
// 多图融合、拼合）都依赖本地扩散工作流，云端做不到，所以下拉框里直接不出现 ——
// 免得用户选完之后才被后端拒绝。
const onlineModelSelected = computed(() => currentImageModel.value?.provider === 'minimax')
const availableGenModes = computed(() => onlineModelSelected.value
  ? GEN_MODES.filter(m => m.value === 't2i' || m.value === 'edit')
  : GEN_MODES)
const genModeMeta = computed(() => availableGenModes.value.find(m => m.value === genMode.value) || availableGenModes.value[0])
// 云端一次出几张的可选项（1 ~ 上游配置的上限，默认上限 4）
const onlineImageCounts = computed(() => {
  const max = Math.max(1, Math.min(Number(imageStatus.value.minimax_max_n) || 1, 9))
  return Array.from({ length: max }, (_, i) => i + 1)
})
// 切到云端模型时把已不支持的模式退回去（比如原来停在「局部重绘」），并清掉遮罩
watch(onlineModelSelected, (online) => {
  if (online && !availableGenModes.value.some(m => m.value === genMode.value)) {
    genMode.value = 't2i'
    pendingMask.value = null
  }
})
// 该模式是否需要附底图/参考图（多图融合、拼合、改图、重绘、扩图、修复都需要）
const imageNeedsRef = computed(() => genModeNeedsRef.value)

function uid() { return Date.now().toString(36) + Math.random().toString(36).slice(2, 8) }

// 读取图片文件/URL → 用 canvas 等比压缩到 IMG_MAX_DIM 以内 → 输出 JPEG dataURL
function compressImage(src) {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => {
      let { width: w, height: h } = img
      const scale = Math.min(1, IMG_MAX_DIM / Math.max(w, h))
      w = Math.max(1, Math.round(w * scale))
      h = Math.max(1, Math.round(h * scale))
      const canvas = document.createElement('canvas')
      canvas.width = w; canvas.height = h
      const ctx = canvas.getContext('2d')
      ctx.fillStyle = '#fff'
      ctx.fillRect(0, 0, w, h)
      ctx.drawImage(img, 0, 0, w, h)
      const dataUrl = canvas.toDataURL('image/jpeg', IMG_QUALITY)
      const kb = Math.round((dataUrl.length * 3) / 4 / 1024)
      resolve({ id: uid(), dataUrl, w, h, kb })
    }
    img.onerror = () => reject(new Error('图片读取失败'))
    img.src = src
  })
}

async function addImageFiles(fileList) {
  const files = Array.from(fileList || []).filter(f => f.type.startsWith('image/'))
  if (!files.length) return
  if (!canAttachImage.value) {
    if (imageMode.value) {
      if (genMode.value === 't2i') ElMessage.warning('文生图模式不需要参考图')
      else ElMessage.warning(`出图不可用：${imageStatus.value.error || '本地 ComfyUI 与在线 MiniMax 都未配置'}`)
    } else {
      ElMessage.warning('当前模型不支持图片，请先选一个视觉模型（如 llava / qwen-vl）')
    }
    return
  }
  for (const f of files) {
    if (pendingImages.value.length >= imageLimit.value) {
      ElMessage.warning(imageMode.value
        ? (genModeMulti.value
            ? `「${genModeMeta.value.label}」最多同时附加 ${imageLimit.value} 张图片`
            : '该模式只支持 1 张参考图（作为编辑对象）')
        : `最多同时附加 ${MAX_IMAGES} 张图片`)
      break
    }
    try {
      const url = URL.createObjectURL(f)
      const comp = await compressImage(url)
      URL.revokeObjectURL(url)
      pendingImages.value.push(comp)
    } catch (e) {
      ElMessage.error(e.message || '图片处理失败')
    }
  }
}

function removePendingImage(id) {
  pendingImages.value = pendingImages.value.filter(x => x.id !== id)
}

function openImagePicker() {
  if (!canAttachImage.value) {
    if (imageMode.value) {
      if (genMode.value === 't2i') ElMessage.warning('文生图模式不需要参考图')
      else ElMessage.warning(`出图不可用：${imageStatus.value.error || '本地 ComfyUI 与在线 MiniMax 都未配置'}`)
    } else {
      ElMessage.warning('当前模型不支持图片，请先选一个视觉模型（如 llava / qwen-vl）')
    }
    return
  }
  imageInputRef.value?.click()
}

// 遮罩图：局部重绘 / 外补专用。白色像素 = 要重绘 / 生成的区域。
// 不压缩（保留清晰边缘），直接存 dataURL；体积偏大但这是出图素材不是历史垃圾，可接受。
function openMaskPicker() {
  maskInputRef.value?.click()
}
async function addMaskFile(fileList) {
  const files = Array.from(fileList || []).filter(f => f.type.startsWith('image/'))
  const f = files[0]
  if (!f) return
  try {
    const url = URL.createObjectURL(f)
    const dataUrl = await new Promise((resolve, reject) => {
      const img = new Image()
      img.onload = () => {
        const canvas = document.createElement('canvas')
        canvas.width = img.width; canvas.height = img.height
        const ctx = canvas.getContext('2d')
        ctx.drawImage(img, 0, 0)
        resolve(canvas.toDataURL('image/png'))
      }
      img.onerror = () => reject(new Error('遮罩读取失败'))
      img.src = url
    })
    URL.revokeObjectURL(url)
    pendingMask.value = { id: uid(), dataUrl, w: 0, h: 0 }
  } catch (e) {
    ElMessage.error(e.message || '遮罩处理失败')
  }
}

function onPaste(e) {
  const items = e.clipboardData?.items || []
  const imgs = []
  for (const it of items) {
    if (it.type.startsWith('image/')) {
      const f = it.getAsFile()
      if (f) imgs.push(f)
    }
  }
  if (imgs.length) { e.preventDefault(); addImageFiles(imgs) }
}

function onDrop(e) {
  dragOver.value = false
  if (e.dataTransfer?.files?.length) {
    e.preventDefault()
    addImageFiles(e.dataTransfer.files)
  }
}

// 估算当前待发送图片总体积（KB），用于上下文风险提示
const pendingImagesKB = computed(() => pendingImages.value.reduce((s, x) => s + (x.kb || 0), 0))
const imageContextWarn = computed(() => pendingImagesKB.value > 1200) // 单条图片过重时提醒

function loadConversations() {
  try { conversations.value = JSON.parse(localStorage.getItem(CONV_LIST_KEY) || '[]') } catch { conversations.value = [] }
}
function saveConversations() {
  try { localStorage.setItem(CONV_LIST_KEY, JSON.stringify(conversations.value)) } catch { /* 忽略配额 */ }
}
function loadMessages(convId) {
  messages.value = loadMessagesFor(convId)
  // 刷新/重启页面后，出图任务仍在服务端后台跑（任务表保留 2 小时）。
  // 有 task_id 的就交给轮询接管；没有的才收尾，避免进度条永远停在半截。
  for (const m of messages.value) {
    if (m && m.generating && !m.taskId) {
      m.generating = false
      m.genError = m.genError || '页面已刷新，本次生成中断（可点「重新生成」）'
    }
  }
  // 先把本地缓存画出来（秒开），再去服务端对齐（另一台设备发的消息也能拉下来）
  if (convId) refreshMessagesFromServer(convId)
  resumePendingImageTasks()
}
function loadMessagesFor(convId) {
  try { return JSON.parse(localStorage.getItem(MSGS_KEY_PREFIX + convId) || '[]') } catch { return [] }
}
function saveMessages() {
  saveMessagesFor(currentConvId.value, messages.value)
  scheduleMessagesSync(currentConvId.value, messages.value)
}
function saveMessagesFor(convId, list) {
  if (!convId) return
  try {
    localStorage.setItem(MSGS_KEY_PREFIX + convId, JSON.stringify(list))
  } catch {
    // 含图片的历史可能超过 localStorage 配额，静默降级（服务端那份是完整的）
  }
}

// ── 服务端同步（会话 / 消息）─────────────────────────────────────────────
function authHeaders(extra) {
  return { ...(extra || {}), ...auth.authHeader() }
}

async function apiJSON(path, opts = {}) {
  const res = await fetch(`${API_BASE}${path}`, { ...opts, headers: authHeaders(opts.headers) })
  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try {
      const d = await res.json()
      if (d?.error) msg = d.error
    } catch { /* 非 JSON 响应 */ }
    const err = new Error(msg)
    err.status = res.status
    throw err
  }
  return res.json().catch(() => ({}))
}

// 会话：本地有、服务端没有的（升级前的老会话 / 离线期间新建的）补传上去，避免"升级后历史消失"
async function syncConversationsFromServer() {
  if (!auth.isAuthenticated.value) return
  let data
  try {
    data = await apiJSON('/api/chat-plus/conversations')
  } catch {
    return // 服务端不可达：继续用本地缓存
  }
  const server = Array.isArray(data?.conversations) ? data.conversations : []
  const tombstones = deletedConvIds()
  // 已删除但服务端还留着的会话：补删，别让它回到列表
  for (const c of server) {
    if (!tombstones.has(c.id)) continue
    deleteConversationOnServer(c.id).then(() => unmarkConvDeleted(c.id)).catch(() => {})
  }
  const live = server.filter(c => !tombstones.has(c.id))
  const serverIds = new Set(live.map(c => c.id))
  const merged = live.map(c => ({
    id: c.id,
    title: c.title || '新对话',
    createdAt: Date.parse(c.created_at) || Date.now(),
    updatedAt: Date.parse(c.updated_at) || Date.now(),
  }))
  for (const c of conversations.value) {
    if (!c?.id || serverIds.has(c.id) || tombstones.has(c.id)) continue
    merged.push(c)
    pushConversation(c).catch(() => {})
    backfillMessages(c.id)
  }
  merged.sort((a, b) => (b.updatedAt || 0) - (a.updatedAt || 0))
  conversations.value = merged
  saveConversations()
}

async function pushConversation(conv) {
  if (!conv?.id || !auth.isAuthenticated.value) return
  await apiJSON('/api/chat-plus/conversations', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id: conv.id, title: conv.title || '' }),
  })
}

async function deleteConversationOnServer(convId) {
  if (!convId || !auth.isAuthenticated.value) return
  await apiJSON(`/api/chat-plus/conversations/${encodeURIComponent(convId)}`, { method: 'DELETE' })
}

// 删除墓碑：删掉的会话记在本地，避免服务端列表下一轮同步又把它拉回来
// （弱网/离线时 DELETE 可能没到达，下次同步会重试）。
const DELETED_CONVS_KEY = 'chat-plus-deleted-convs'
function deletedConvIds() {
  try { return new Set(JSON.parse(localStorage.getItem(DELETED_CONVS_KEY) || '[]')) } catch { return new Set() }
}
function markConvDeleted(id) {
  const s = deletedConvIds()
  s.add(id)
  try { localStorage.setItem(DELETED_CONVS_KEY, JSON.stringify([...s].slice(-200))) } catch { /* 忽略 */ }
}
function unmarkConvDeleted(id) {
  const s = deletedConvIds()
  if (!s.delete(id)) return
  try { localStorage.setItem(DELETED_CONVS_KEY, JSON.stringify([...s])) } catch { /* 忽略 */ }
}

function toServerMessage(m) {
  const status = m.genError ? 'failed'
    : m.failed ? 'failed'
      : (m.interrupted ? 'canceled' : (m.streaming || m.generating ? 'running' : 'ok'))
  const out = {
    id: m.id,
    role: m.role,
    content: m.kind === 'image' ? (m.prompt || '') : (m.content || ''),
    reasoning: m.reasoning || '',
    task_id: m.taskId || '',
    kind: m.kind || 'text',
    status,
    error_message: m.failed || m.genError || '',
    ts: m.ts || Date.now(),
  }
  // 出图消息：结果图本身在服务端（ComfyUI 任务产物），只需同步 URL
  if (m.kind === 'image') {
    if (Array.isArray(m.genImages) && m.genImages.length) {
      out.images = m.genImages.map(gi => ({
        id: gi.url, dataUrl: '', url: gi.url, w: gi.width || 0, h: gi.height || 0,
      }))
    }
    return out
  }
  if (Array.isArray(m.images) && m.images.length) {
    out.images = m.images.map(im => ({
      id: im.id || '',
      dataUrl: im.dataUrl || '',
      url: im.url || '',
      w: im.w || 0,
      h: im.h || 0,
    }))
  }
  return out
}

// 图片一旦上了服务端就只保留 URL：base64 存 localStorage 很容易撞 5MB 配额，
// 撞了之后「新消息干脆存不进去」，表现成刷新后消息丢失。
function adoptServerImages(list, saved) {
  const byId = new Map((saved || []).map(s => [s.id, s]))
  for (const m of list) {
    const s = byId.get(m.id)
    if (!s || !Array.isArray(s.images) || !s.images.length) continue
    const hasDataUrl = (m.images || []).some(im => im.dataUrl)
    if (!hasDataUrl) continue
    m.images = s.images.map(im => ({ id: im.id, url: im.url, w: im.w, h: im.h }))
  }
}

async function pushMessages(convId, list) {
  if (!convId || !list?.length || !auth.isAuthenticated.value) return
  try {
    const data = await apiJSON(`/api/chat-plus/conversations/${encodeURIComponent(convId)}/messages`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ messages: list.map(toServerMessage) }),
    })
    adoptServerImages(list, data?.messages)
    // 用 url 版覆盖本地缓存（不再往 localStorage 塞 base64）
    saveMessagesFor(convId, convId === currentConvId.value ? messages.value : list)
  } catch {
    // 同步失败不打断聊天：本地缓存还在，下次保存会重试
  }
}

// 消息同步节流：流式过程中正文一直在变，400ms 合并一次就够
const msgSyncTimers = new Map()
function scheduleMessagesSync(convId, list) {
  if (!convId || !auth.isAuthenticated.value) return
  const snapshot = [...list]
  clearTimeout(msgSyncTimers.get(convId))
  msgSyncTimers.set(convId, setTimeout(() => {
    msgSyncTimers.delete(convId)
    pushMessages(convId, snapshot)
  }, 400))
}

// 把本地（离线期间产生/升级前的老消息）补到服务端
async function backfillMessages(convId) {
  const local = loadMessagesFor(convId)
  if (!local.length) return
  let server
  try {
    server = await pullMessages(convId, 0)
  } catch {
    return
  }
  const ids = new Set(server.map(m => m.id))
  const missing = local.filter(m => m?.id && !ids.has(m.id))
  if (missing.length) await pushMessages(convId, missing)
}

async function pullMessages(convId, sinceTs) {
  const data = await apiJSON(`/api/chat-plus/conversations/${encodeURIComponent(convId)}/messages?since_ts=${sinceTs || 0}`)
  return Array.isArray(data?.messages) ? data.messages : []
}

function lastMessageTs() {
  let max = 0
  for (const m of messages.value) if (m?.ts > max) max = m.ts
  return max
}

// mergeServerMessages 把服务端消息并进当前列表：
// 同 id 用服务端内容补全（流式结束后以服务端为准），新的追加，最后按时间排序。
function mergeServerMessages(convId, list) {
  if (convId !== currentConvId.value || !Array.isArray(list) || !list.length) return false
  let changed = false
  const hidden = hiddenTaskIds(convId)
  for (const sm of list) {
    if (!sm?.id) continue
    // 「重新生成」替换掉的旧回复：服务端还留着，别又合并回来
    if (hidden.has(sm.id) || (sm.task_id && hidden.has(sm.task_id))) continue
    const imgs = Array.isArray(sm.images)
      ? sm.images.map(im => ({ id: im.id, url: im.url, w: im.w, h: im.h, model: im.model }))
      : []
    const existing = messages.value.find(m => m && (m.id === sm.id || (sm.task_id && m.taskId === sm.task_id)))
    if (existing) {
      // 正在流式接收的那条不要被服务端旧快照覆盖
      if (streamMsg.value === existing) continue
      let touched = false
      if ((sm.content || '').length > (existing.content || '').length) { existing.content = sm.content; touched = true }
      if ((sm.reasoning || '').length > (existing.reasoning || '').length) { existing.reasoning = sm.reasoning; touched = true }
      if (imgs.length && !(existing.images || []).length) { existing.images = imgs; touched = true }
      if (sm.task_id && !existing.taskId) { existing.taskId = sm.task_id; touched = true }
      // 出图消息：后端任务终态会自己落库（前端掉线时这是唯一的图来源），
      // 服务端有图就补进 genImages 并清掉本地的「失败/中断」标记。
      if (existing.kind === 'image' && imgs.length && !(existing.genImages || []).length) {
        existing.genImages = imgs.map(im => ({ url: im.url, width: im.w, height: im.h, filename: '' }))
        if (!existing.genModel && imgs[0].model) {
          existing.genModelId = imgs[0].model
          existing.genModel = imageModelLabel(imgs[0].model)
        }
        existing.generating = false
        existing.genError = ''
        existing.progress = 100
        existing.phase = '完成'
        touched = true
      }
      // 服务端给出真实失败原因时覆盖本地误判（比如「网络连接中断」其实是判早了）
      if (existing.kind === 'image' && sm.status === 'failed' && sm.error_message
        && existing.genError !== sm.error_message) {
        existing.genError = sm.error_message
        existing.generating = false
        touched = true
      }
      // 服务端说还在跑、本地又没有结果图：交给轮询接管（刷新/换设备/断网后回到页面时的主要路径）
      if (sm.status === 'running' && existing.kind === 'image'
        && !(existing.genImages || []).length && !existing.genError && !existing.generating) {
        existing.generating = true
        touched = true
      }
      if (sm.status === 'failed' && !existing.failed) { existing.failed = sm.error_message || '生成失败'; touched = true }
      if (sm.status === 'canceled' && !existing.interrupted) { existing.interrupted = true; touched = true }
      if (touched) changed = true
      continue
    }
    if (sm.role !== 'assistant' && sm.role !== 'user') continue
    // 出图消息：正文是提示词，图片在 images 里（服务端产物 URL）
    if (sm.kind === 'image') {
      messages.value.push(reactive({
        id: sm.id,
        role: sm.role,
        kind: 'image',
        prompt: sm.content || '',
        refImage: '',
        content: '',
        genImages: imgs.map(im => ({ url: im.url, width: im.w, height: im.h, filename: '' })),
        genError: sm.status === 'failed' ? (sm.error_message || '生成失败') : '',
        progress: 0,
        phase: '',
        // 任务还在服务端排队/跑：立刻生成中，由 resumePendingImageTasks 轮询取结果
        generating: sm.status === 'running' && !!sm.task_id,
        taskId: sm.task_id || undefined,
        ts: sm.ts || Date.now(),
      }))
      changed = true
      continue
    }
    messages.value.push(reactive({
      id: sm.id,
      role: sm.role,
      content: sm.content || '',
      reasoning: sm.reasoning || '',
      images: imgs,
      model: sm.model || '',
      ts: sm.ts || Date.now(),
      taskId: sm.task_id || undefined,
      kind: sm.kind || 'text',
      interrupted: sm.status === 'canceled' || undefined,
      failed: sm.status === 'failed' ? (sm.error_message || '生成失败') : undefined,
    }))
    changed = true
  }
  if (changed) {
    messages.value.sort((a, b) => (a.ts || 0) - (b.ts || 0))
    saveMessagesFor(convId, messages.value)
    scrollToBottom()
    resumePendingImageTasks()
  }
  return changed
}

// refreshMessagesFromServer 全量对齐（进入会话 / 多端轮询 / reconcile 时调用）
async function refreshMessagesFromServer(convId) {
  if (!convId || !auth.isAuthenticated.value) return
  try {
    const list = await pullMessages(convId, 0)
    mergeServerMessages(convId, list)
  } catch { /* 离线：保持本地缓存 */ }
}
function loadSettings() {
  try {
    const s = JSON.parse(localStorage.getItem(SETTINGS_KEY) || '{}')
    settings.value = {
      model: s.model || '',
      systemPrompt: s.systemPrompt || '',
      temperature: typeof s.temperature === 'number' ? s.temperature : 0.7,
      // 重启后默认回到对话模式：出图是否可用要等探活结果，先不自动进去
      imageMode: false,
      // 上次选的出图模型（id）。服务端预设校验不过会自动回落到默认预设
      imageModel: s.imageModel || '',
      // 云端出图一次出几张（1~4）。本地 ComfyUI 忽略此值，恒定 1 张
      imageCount: Number(s.imageCount) > 0 ? Math.min(Number(s.imageCount), 9) : 1,
      // 提示词优化用的模型（'' = 跟随聊天模型）；选过就记住，别每次都重选
      promptOptimizeModel: s.promptOptimizeModel || '',
    }
  } catch { /* 保持默认 */ }
}
function saveSettings() {
  localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings.value))
}

// ══════════════ 模型列表 ══════════════
const modelGroups = computed(() => {
  const groups = []
  const ol = modelInfo.value.ollama
  if (ol.available && ol.models.length) {
    groups.push({
      label: `Ollama 本地（${ol.models.length}）`,
      items: ol.models.map(m => ({
        value: m.id,
        label: m.name,
        tag: m.parameter_size || (m.size_gb ? `${m.size_gb.toFixed(1)}GB` : ''),
        vision: !!m.vision,
      })),
    })
  }
  const mm = modelInfo.value.minimax
  if (mm.available) {
    groups.push({
      label: 'MiniMax 云端',
      items: mm.models.map(m => ({ value: m.id, label: m.label, tag: m.description })),
    })
  }
  return groups
})

const currentModelLabel = computed(() => {
  for (const g of modelGroups.value) {
    const found = g.items.find(m => m.value === settings.value.model)
    if (found) return found.label
  }
  return settings.value.model || '选择模型'
})

// 当前模型是否支持图片输入（仅 Ollama 视觉模型为 true）
const selectedModelVision = computed(() => {
  for (const g of modelGroups.value) {
    const found = g.items.find(m => m.value === settings.value.model)
    if (found) return !!found.vision
  }
  return false
})

const engineStatus = computed(() => {
  const { minimax, ollama } = modelInfo.value
  const parts = []
  if (ollama.available) parts.push(`Ollama ${ollama.models.length}`)
  if (minimax.available) parts.push('MiniMax')
  if (imageStatus.value.available && imageMode.value) parts.push('本地出图')
  if (parts.length === 0) return { type: 'off', text: ollama.error ? '引擎不可用' : '加载中…' }
  return { type: 'on', text: parts.join(' · ') }
})

// 出图能力是否可用（后端探活 ComfyUI 的结果）
const imageAvailable = computed(() => !!(imageStatus.value.available || imageStatus.value.minimax_available))

// 能否附加图片：对话模式 = 当视觉模型输入用；出图模式 = 当参考图用（文生图不需要参考图）
const canAttachImage = computed(() => {
  if (!imageMode.value) return selectedModelVision.value
  if (genMode.value === 't2i') return false
  return imageAvailable.value
})

const attachTitle = computed(() => {
  if (imageMode.value) {
    if (genMode.value === 't2i') return '文生图模式不需要参考图'
    return imageAvailable.value ? '附加参考图（按指令修改这张图）' : '出图不可用'
  }
  return selectedModelVision.value ? '附加图片（视觉模型）' : '当前模型不支持图片'
})

const composerPlaceholder = computed(() => (imageMode.value
  ? '描述你想要的画面…（附一张图则改成「按指令修改这张图」）'
  : '输入消息…（Enter 发送，Shift + Enter 换行；可粘贴/拖拽图片）'))

const sendDisabled = computed(() => {
  if (imageMode.value) {
    if (imageGenerating.value || !userInput.value.trim()) return true
    // 只装/选了「只能改图」的模型时，必须附参考图
    if (imageNeedsRef.value && !pendingImages.value.length) return true
    return false
  }
  return (!userInput.value.trim() && pendingImages.value.length === 0) || modelsLoading.value
})

// 出图能力探活。后端未配置或 ComfyUI 没启动时自动退出出图模式，
// 避免用户写完提示词点发送才看到报错。
async function loadImageStatus() {
  try {
    const res = await fetch(`${API_BASE}/api/chat-plus/image/status`, { headers: auth.authHeader() })
    if (!res.ok) return
    imageStatus.value = await res.json()
    if (!imageStatus.value.available) imageMode.value = false
    else loadImageModels()
  } catch {
    // 探活失败不打断页面：出图按钮保持禁用即可
  }
}

function toggleImageMode() {
  if (!imageAvailable.value) {
    ElMessage.warning(imageStatus.value.error
      ? `出图不可用：${imageStatus.value.error}`
      : '出图不可用')
    return
  }
  imageMode.value = !imageMode.value
  // 切回对话模式时，参考图/遮罩可能超出视觉模型张数上限，直接清空更直观
  if (!imageMode.value) {
    pendingImages.value = []
    pendingMask.value = null
  } else if (!imageModels.value.length) {
    loadImageModels()
  }
}

// 模型 id → 展示名；清单还没拉到时先显示 id，别把标签空着
function imageModelLabel(id) {
  if (!id) return ''
  return imageModels.value.find(m => m.id === id)?.label || id
}

// 拉取可切换的出图模型清单。失败不打断页面：沿用后端默认模型即可。
async function loadImageModels() {
  if (imageModelsLoading.value) return
  imageModelsLoading.value = true
  try {
    const res = await fetch(`${API_BASE}/api/chat-plus/image/models`, { headers: auth.authHeader() })
    if (!res.ok) return
    const data = await res.json()
    imageModels.value = Array.isArray(data.models) ? data.models : []
    const stillValid = imageModels.value.some(m => m.id === settings.value.imageModel && m.available !== false)
    if (!stillValid) {
      const def = imageModels.value.find(m => m.id === data.default_model && m.available !== false)
      const fallback = def || usableImageModels.value[0] || imageModels.value[0]
      settings.value.imageModel = fallback ? fallback.id : ''
    }
  } catch {
    // 模型清单拿不到就退回后端默认（请求里不传 model）
  } finally {
    imageModelsLoading.value = false
  }
}

// 参考图与模型能力对齐：flux2 只做文生图，附了参考图就必须换到支持改图的模型。
// 返回 false 表示当前是不可自动修复的情况（例如只装了改图模型却没附图）。
// 注意：inpaint/outpaint/restore/multi_ref/composite 这几种模式，后端会自动切到
// 对应家族预设（qwen-edit / qwen21），前端选什么模型都不影响，所以直接放行。
function ensureModelMatchesRef(hasRef, mode) {
  if (mode === 'inpaint' || mode === 'outpaint' || mode === 'restore' ||
      mode === 'multi_ref' || mode === 'composite') {
    return true
  }
  const cur = currentImageModel.value
  if (!cur) return true
  if (hasRef && cur.supports_edit === false) {
    const alt = usableImageModels.value.find(m => m.supports_edit !== false)
    if (!alt) {
      ElMessage.warning(`模型「${cur.label}」不支持参考图编辑，且没有其它可用模型`)
      return false
    }
    settings.value.imageModel = alt.id
    ElMessage.info(`「${cur.label}」不支持改图，已切换到「${alt.label}」`)
  }
  if (!hasRef && cur.supports_t2i === false) {
    ElMessage.warning(`模型「${cur.label}」只能改图，请先附一张参考图`)
    return false
  }
  return true
}

async function loadModels() {
  modelsLoading.value = true
  try {
    const res = await fetch(`${API_BASE}/api/chat-plus/models`, { headers: auth.authHeader() })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error(data.error || `HTTP ${res.status}`)
    }
    modelInfo.value = await res.json()
    // 默认模型：优先 MiniMax-M3，其次第一个可用模型
    if (!settings.value.model || !modelGroups.value.some(g => g.items.some(m => m.value === settings.value.model))) {
      const first = modelGroups.value.flatMap(g => g.items)[0]
      settings.value.model = first ? first.value : ''
    }
    if (!modelInfo.value.ollama.available && modelInfo.value.ollama.error) {
      errorMsg.value = `Ollama（${modelInfo.value.ollama.base_url}）暂不可用：${modelInfo.value.ollama.error}`
    }
  } catch (e) {
    errorMsg.value = `获取模型列表失败：${netErrMsg(e)}`
  } finally {
    modelsLoading.value = false
  }
}

// ══════════════ 会话管理 ══════════════
function newConversation() {
  switchConversation('__draft__')
}
// 把浏览器网络层错误的原始文案（Safari「Load failed」/Chrome「Failed to fetch」等）
// 映射成可读提示——这类错误意味着请求根本没到达服务端（切网/锁屏/服务重启/公网链路抖动）
function netErrMsg(e) {
  const msg = (e?.message || '').toLowerCase()
  if (e?.name === 'AbortError') return '请求已取消'
  if (
    msg.includes('load failed') || msg.includes('failed to fetch') ||
    msg.includes('networkerror') || msg.includes('network error') ||
    msg.includes('the internet connection') || msg.includes('timed out') ||
    msg.includes('connection') && msg.includes('lost')
  ) {
    return '网络连接中断，请求未到达服务器（常见于切换 Wi-Fi/蜂窝、锁屏后台、弱网或服务刚重启）。请检查网络后点击重试。'
  }
  return `请求失败：${e?.message || e}`
}

function switchConversation(id) {
  if (isMobileViewport.value) sidebarOpen.value = false // 移动端选完会话自动收起抽屉
  if (id === '__draft__') {
    currentConvId.value = ''
    messages.value = []
    return
  }
  currentConvId.value = id
  try { localStorage.setItem(CURRENT_CONV_KEY, id) } catch { /* 忽略配额异常 */ }
  loadMessages(id)
  // 刷新/切回会话后，把服务端已跑完（或仍在跑）的任务对上来
  reconcileChatTasks()
}
function deleteConversation(id) {
  conversations.value = conversations.value.filter(c => c.id !== id)
  localStorage.removeItem(MSGS_KEY_PREFIX + id)
  localStorage.removeItem(HIDDEN_TASKS_PREFIX + id)
  saveConversations()
  // 服务端也要删，否则另一台设备/下次刷新会把这条会话又拉回来；
  // 先立墓碑，DELETE 没送达（离线）时下次同步会重试。
  markConvDeleted(id)
  deleteConversationOnServer(id).then(() => unmarkConvDeleted(id)).catch(() => {})
  if (currentConvId.value === id) switchConversation('__draft__')
}
function ensureConversation(firstUserText) {
  if (currentConvId.value) return
  const conv = {
    id: uid(),
    title: (firstUserText || '新对话').slice(0, 30),
    createdAt: Date.now(),
    updatedAt: Date.now(),
  }
  conversations.value.unshift(conv)
  currentConvId.value = conv.id
  saveConversations()
  pushConversation(conv).catch(() => {})
}
function touchConversation() {
  const conv = conversations.value.find(c => c.id === currentConvId.value)
  if (conv) {
    conv.updatedAt = Date.now()
    saveConversations()
  }
}

// ══════════════ 发送 / 流式接收 ══════════════
const messagesRef = ref(null)
let stickToBottom = true
let abortController = null

const displayMessages = computed(() => {
  const list = [...messages.value]
  if (streaming.value && streamMsg.value) list.push(streamMsg.value)
  return list
})
const lastAssistant = computed(() => [...messages.value].reverse().find(m => m.role === 'assistant'))
const streamMsg = ref(null)

function scrollToBottom(force = false) {
  if (!stickToBottom && !force) return
  nextTick(() => {
    const el = messagesRef.value
    if (el) el.scrollTop = el.scrollHeight
  })
}
function onScroll() {
  const el = messagesRef.value
  if (!el) return
  stickToBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80
}

function buildPayloadMessages(extra = []) {
  const msgs = []
  if (settings.value.systemPrompt.trim()) {
    msgs.push({ role: 'system', content: settings.value.systemPrompt.trim() })
  }
  // 先算本次请求实际要带哪些图片：从最新消息往回取，总张数不超过 IMG_TOTAL_BUDGET；
  // 更早的图片整组丢弃、改成文字占位——硬性防止多轮累积把上下文撑爆。
  const sendImgs = new Map() // index -> images[]
  let budget = IMG_TOTAL_BUDGET
  for (let i = messages.value.length - 1; i >= 0; i--) {
    const imgs = messages.value[i].images || []
    if (!imgs.length) continue
    if (imgs.length <= budget) {
      sendImgs.set(i, imgs)
      budget -= imgs.length
    }
  }
  messages.value.forEach((m, i) => {
    const text = m.content || ''
    const imgs = sendImgs.get(i) || []
    if (imgs.length) {
      // 视觉模型：content 用 OpenAI 多模态数组格式（文本 + image_url）。
      // 历史消息同步过来的图片只有 url（base64 没跟着同步），后端转发给 Ollama 前会自己内联成 base64。
      const content = []
      if (text.trim()) content.push({ type: 'text', text })
      for (const im of imgs) {
        const src = im.dataUrl || im.url
        if (src) content.push({ type: 'image_url', image_url: { url: src } })
      }
      msgs.push({ role: m.role, content })
    } else if (text.trim() || (m.images && m.images.length)) {
      // 图片被丢弃的历史消息：留一句占位，让模型知道"这里原本有图"
      const note = (m.images && m.images.length)
        ? `${text}${text.trim() ? '\n' : ''}（此前的图片已省略以节省上下文）`
        : text
      if (note.trim()) msgs.push({ role: m.role, content: note })
    }
  })
  for (const e of extra) msgs.push(e) // 隐藏指令：只进本次请求，不进界面也不进历史
  return msgs
}

function quickAsk(text) {
  userInput.value = text
  sendMessage()
}

function onKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
    e.preventDefault()
    sendMessage()
  }
}

async function sendMessage() {
  const text = userInput.value.trim()
  // 出图模式走完全不同的链路（ComfyUI），先分流再走对话逻辑
  if (imageMode.value) {
    await sendImageTask(text)
    return
  }
  if ((!text && pendingImages.value.length === 0) || streaming.value) return
  if (!settings.value.model) {
    ElMessage.warning('请先选择模型')
    return
  }
  if (pendingImages.value.length && !selectedModelVision.value) {
    ElMessage.warning('当前模型不支持图片，请先选一个视觉模型，或移除已附加的图片')
    return
  }
  errorMsg.value = ''
  const imgs = pendingImages.value.map(x => ({ id: x.id, dataUrl: x.dataUrl, w: x.w, h: x.h }))
  userInput.value = ''
  pendingImages.value = []
  ensureConversation(text || (imgs.length ? '[图片]' : ''))
  messages.value.push({ id: uid(), role: 'user', content: text, images: imgs, ts: Date.now() })
  saveMessages()
  scrollToBottom(true)
  await doStream()
}

async function regenerate() {
  if (streaming.value) return
  // 去掉最后一条 assistant 消息后重发
  const idx = messages.value.map(m => m.role).lastIndexOf('assistant')
  if (idx >= 0) {
    // 记住被替换掉的任务，避免刷新后被 reconcile 又补回来
    hideTaskId(currentConvId.value, messages.value[idx]?.taskId)
    messages.value.splice(idx, 1)
  }
  saveMessages()
  await doStream()
}

async function doStream(extra = []) {
  streaming.value = true
  stickToBottom = true
  const convId = currentConvId.value
  const msg = reactive({ id: uid(), role: 'assistant', content: '', reasoning: '', model: settings.value.model, ts: Date.now(), streaming: true, convId })
  streamMsg.value = msg
  scrollToBottom(true)

  abortController = new AbortController()
  const signal = abortController.signal

  try {
    // 弱网架构：先建"服务端任务"，再订阅它的输出。
    // 连接断了/页面刷新都不影响服务端继续生成，重连时凭 from=<seq> 只补增量。
    const task = await createChatTask({
      model: settings.value.model,
      messages: buildPayloadMessages(extra),
      temperature: settings.value.temperature,
      stream: true,
      conversation_id: convId,
    }, signal)
    msg.taskId = task.task_id
    // 消息 id 直接沿用任务 id：服务端落库用的是同一个 id，多端合并时不会出现一条回复两份
    if (task.task_id) msg.id = task.task_id
    await watchChatTask(msg, task.task_id, 0, signal)
  } catch (e) {
    if (!signal.aborted) {
      const reason = netErrMsg(e)
      errorMsg.value = reason
      if (!msg.content && !msg.reasoning) msg.failed = reason
      else msg.interrupted = true
    }
  } finally {
    finalizeStream(msg)
  }
}

// createChatTask 建任务。这一跳是整个链路的入口，一次 TCP 抖动就判失败太脆：
// 网络错误 / 5xx / 429 各重试一次以上（4xx 参数错误不重试，重试也没用）。
async function createChatTask(payload, signal) {
  let lastErr = null
  for (let attempt = 0; attempt < 3; attempt++) {
    if (signal?.aborted) throw new DOMException('aborted', 'AbortError')
    try {
      const res = await fetch(`${API_BASE}/api/chat-plus/tasks`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...auth.authHeader() },
        body: JSON.stringify(payload),
        signal,
      })
      if (res.ok) return res.json()
      const data = await res.json().catch(() => ({}))
      const err = new Error(data.error || `HTTP ${res.status}`)
      err.status = res.status
      // 4xx（除 429 限流外）都是"重试也没用"的确定性错误
      if (res.status < 500 && res.status !== 429) throw err
      lastErr = err
    } catch (e) {
      if (signal?.aborted) throw e
      if (e.status && e.status < 500 && e.status !== 429) throw e
      lastErr = e
    }
    await delay(700 * (attempt + 1))
  }
  throw lastErr || new Error('创建任务失败')
}

// ══════════════ 任务订阅（断线续传） ══════════════
const TASK_STREAM_RETRY_MAX = 4      // SSE 连续失败几次后改轮询
const TASK_RECONNECT_BASE_MS = 600
const IMAGE_TASK_POLL_MS = 2000      // 出图轮询间隔
// 最长等待：后端主等待(timeout_sec, 默认 1800s) + 宽限续等(30min) 都要能覆盖，
// 否则前端先放弃，用户就会觉得「图没出来」，其实 ComfyUI 还在跑。
const IMAGE_TASK_MAX_MS = 60 * 60 * 1000

function delay(ms) {
  return new Promise(r => setTimeout(r, ms))
}

// consumeChatTaskStream 解析任务 SSE：id: 行是片段序号（续传游标），
// data: 行是 OpenAI chunk；服务端判定断流/失败时会发 error chunk。
async function consumeChatTaskStream(res, msg, startSeq) {
  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buf = ''
  let seq = startSeq
  let done = false
  let error = ''
  let errorType = ''
  while (true) {
    const { done: readDone, value } = await reader.read()
    if (readDone) break
    buf += decoder.decode(value, { stream: true })
    const lines = buf.split('\n')
    buf = lines.pop()
    for (const raw of lines) {
      const line = raw.trim()
      if (!line) continue
      if (line.startsWith('id:')) {
        const n = parseInt(line.slice(3).trim(), 10)
        if (!Number.isNaN(n)) seq = n
        continue
      }
      if (!line.startsWith('data:')) continue   // 心跳注释行
      const data = line.slice(5).trim()
      if (data === '[DONE]') { done = true; continue }
      let chunk
      try { chunk = JSON.parse(data) } catch { continue }
      if (chunk?.error) {
        error = chunk.error.message || String(chunk.error)
        errorType = chunk.error.type || chunk.error.code || ''
        continue
      }
      const delta = chunk?.choices?.[0]?.delta || {}
      const reasoningPiece = delta.reasoning || delta.reasoning_content || ''
      if (reasoningPiece) msg.reasoning += reasoningPiece
      if (delta.content) {
        msg.content += delta.content
        scrollToBottom()
      } else if (reasoningPiece) {
        scrollToBottom()
      }
    }
    if (done) break
  }
  return { seq, done, error, errorType }
}

async function fetchChatTask(taskId, signal) {
  const res = await fetch(`${API_BASE}/api/chat-plus/tasks/${encodeURIComponent(taskId)}`, {
    headers: auth.authHeader(),
    signal,
  })
  if (!res.ok) return null
  return res.json().catch(() => null)
}

// applyTaskSnapshot 用服务端权威内容覆盖本地（轮询/过期兜底时用）。
function applyTaskSnapshot(msg, snap) {
  if (!snap) return
  if (typeof snap.content === 'string' && snap.content.length >= msg.content.length) msg.content = snap.content
  if (typeof snap.reasoning === 'string' && snap.reasoning.length >= msg.reasoning.length) msg.reasoning = snap.reasoning
  scrollToBottom()
}

// pollChatTask 兜底：SSE 被代理掐断/弱网反复抖动时退化为轮询快照，保证结果不丢。
async function pollChatTask(msg, taskId, signal) {
  for (let i = 0; i < 800; i++) {
    if (signal?.aborted) throw new DOMException('aborted', 'AbortError')
    const snap = await fetchChatTask(taskId, signal)
    if (snap) {
      applyTaskSnapshot(msg, snap)
      if (snap.done) {
        if (snap.status === 'failed') {
          msg.failed = snap.error_message || '生成失败'
          errorMsg.value = `生成失败：${msg.failed}`
        } else if (snap.status === 'canceled') {
          msg.interrupted = true
        }
        return
      }
    }
    await delay(1500)
  }
  msg.interrupted = true
  errorMsg.value = '生成仍在后台进行，稍后重新进入本对话即可看到结果'
}

// watchChatTask 订阅任务输出；断线自动按 seq 续传，连续失败退化为轮询。
async function watchChatTask(msg, taskId, fromSeq, signal) {
  let cursor = fromSeq || 0
  let failures = 0
  let finalError = ''

  while (true) {
    if (signal?.aborted) throw new DOMException('aborted', 'AbortError')
    try {
      const res = await fetch(`${API_BASE}/api/chat-plus/tasks/${encodeURIComponent(taskId)}/stream?from=${cursor}`, {
        headers: auth.authHeader(),
        signal,
      })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      msg.reconnecting = false   // 重新连上了，先收起"续传中"提示
      const out = await consumeChatTaskStream(res, msg, cursor)
      cursor = out.seq
      failures = 0
      if (out.error) {
        // "已停止生成"不是失败：用户自己点的停止，不能弹成红色 error
        if (out.errorType === 'task_canceled' || /已停止/.test(out.error)) {
          msg.interrupted = true
          return
        }
        finalError = out.error
        break
      }
      if (out.done) return
      // 流结束但任务没完成 = 链路被掐断，立刻带上游标重连
      throw new Error('stream interrupted')
    } catch (e) {
      if (signal?.aborted) throw e
      failures++
      if (failures >= TASK_STREAM_RETRY_MAX) {
        await pollChatTask(msg, taskId, signal)
        return
      }
      msg.reconnecting = true
      await delay(Math.min(TASK_RECONNECT_BASE_MS * failures, 4000))
    }
  }

  if (finalError) {
    msg.failed = finalError
    errorMsg.value = `生成失败：${finalError}`
  }
}

// resumeChatTask 页面刷新/切回后，接管仍在后台跑的任务。
async function resumeChatTask(msg, taskId, fromSeq) {
  if (streaming.value) return
  streaming.value = true
  stickToBottom = true
  streamMsg.value = msg
  abortController = new AbortController()
  const signal = abortController.signal
  try {
    await watchChatTask(msg, taskId, fromSeq, signal)
  } catch (e) {
    if (!signal.aborted) {
      msg.interrupted = true
      errorMsg.value = netErrMsg(e)
    }
  } finally {
    finalizeStream(msg)
  }
}

// finalizeStream 收尾：把流式消息落进所属会话（即使期间切换了会话也不会写错地方）。
function finalizeStream(msg) {
  msg.streaming = false
  msg.reconnecting = false
  const convId = msg.convId || currentConvId.value
  const hasSomething = msg.content || msg.reasoning || msg.failed
  if (hasSomething) {
    if (convId && convId === currentConvId.value) {
      if (!messages.value.includes(msg)) messages.value.push(msg)
      saveMessages()
    } else if (convId) {
      const list = loadMessagesFor(convId)
      if (!list.some(m => m && m.taskId && m.taskId === msg.taskId)) list.push(msg)
      saveMessagesFor(convId, list)
      scheduleMessagesSync(convId, list)
    }
    touchConversation()
  }
  streamMsg.value = null
  streaming.value = false
  abortController = null
  scrollToBottom()
}

// reconcileChatTasks 刷新/重进页面后按会话对齐任务：
// 已完成 → 补上回复；失败 → 显示失败原因；仍在跑 → 重新接管并续传。
async function reconcileChatTasks() {
  const convId = currentConvId.value
  if (!convId || !auth.isAuthenticated.value) return
  // 先按服务端消息对齐（另一台设备发的内容、以及服务端已经落库的回复都在这里）
  await refreshMessagesFromServer(convId)
  try {
    const res = await fetch(`${API_BASE}/api/chat-plus/tasks?conversation_id=${encodeURIComponent(convId)}&limit=20`, {
      headers: auth.authHeader(),
    })
    if (!res.ok) return
    const data = await res.json()
    const tasks = Array.isArray(data?.tasks) ? data.tasks : []
    if (!tasks.length) return

    let changed = false
    const hidden = hiddenTaskIds(convId)
    for (const t of tasks.slice().reverse()) {
      if (!t?.task_id) continue
      if (hidden.has(t.task_id)) continue
      if (messages.value.some(m => m && m.taskId === t.task_id)) continue
      if (streamMsg.value && streamMsg.value.taskId === t.task_id) continue

      if (t.status === 'running') {
        if (streaming.value) continue
        const msg = reactive({
          id: t.task_id, role: 'assistant', content: t.content || '', reasoning: t.reasoning || '',
          model: t.model, ts: Date.parse(t.created_at) || Date.now(),
          streaming: true, taskId: t.task_id, convId,
        })
        resumeChatTask(msg, t.task_id, t.seq || 0)
        continue
      }

      messages.value.push(reactive({
        id: t.task_id,
        role: 'assistant',
        content: t.content || '',
        reasoning: t.reasoning || '',
        model: t.model,
        ts: Date.parse(t.created_at) || Date.now(),
        taskId: t.task_id,
        interrupted: t.status === 'canceled' || undefined,
        failed: t.status === 'failed' ? (t.error_message || '生成失败，请重试') : undefined,
      }))
      changed = true
    }
    if (changed) saveMessages()
    // 本地有、服务端没有的消息（离线期间发的）补传上去
    backfillMessages(convId)
  } catch {
    // 只是恢复增强，失败不影响正常使用
  }
}

// ══════════════ 多端同步轮询 ══════════════
// 另一台设备发的问题 / 生成好的回复，最多 8 秒内出现在这里。
// 只拉增量（since_ts）且流式期间跳过——此时同一条消息正在本地被逐字写入。
let syncTimer = null

function startSyncPolling() {
  stopSyncPolling()
  syncTimer = setInterval(async () => {
    if (document.visibilityState !== 'visible') return
    if (!auth.isAuthenticated.value || !currentConvId.value || streaming.value) return
    const convId = currentConvId.value
    try {
      const list = await pullMessages(convId, lastMessageTs())
      if (list.length) {
        if (mergeServerMessages(convId, list)) syncConversationsFromServer()
      }
    } catch { /* 离线/服务重启：下个周期再试 */ }
  }, 8000)
}

function stopSyncPolling() {
  if (syncTimer) {
    clearInterval(syncTimer)
    syncTimer = null
  }
}

function stopGeneration() {
  // 异步任务必须显式停服务端，否则关掉页面它还在跑（白烧 token）
  const taskId = streamMsg.value?.taskId
  if (streamMsg.value) streamMsg.value.interrupted = true
  if (taskId) {
    fetch(`${API_BASE}/api/chat-plus/tasks/${encodeURIComponent(taskId)}/stop`, {
      method: 'POST',
      headers: auth.authHeader(),
    }).catch(() => {})
  }
  abortController?.abort()
  imageAbort?.abort()
}

// 被"重新生成"替换掉的旧任务：记住它，避免刷新后被 reconcile 补回来
const HIDDEN_TASKS_PREFIX = 'chat-plus-hidden-tasks-'

function hiddenTaskIds(convId) {
  try { return new Set(JSON.parse(localStorage.getItem(HIDDEN_TASKS_PREFIX + convId) || '[]')) } catch { return new Set() }
}

function hideTaskId(convId, taskId) {
  if (!convId || !taskId) return
  const ids = hiddenTaskIds(convId)
  ids.add(taskId)
  try {
    const list = [...ids]
    localStorage.setItem(HIDDEN_TASKS_PREFIX + convId, JSON.stringify(list.slice(-100)))
  } catch { /* 忽略 */ }
}

// ══════════════ AI 帮写提示词 ══════════════
// 出图效果八成看提示词，但真实使用里用户只会写「把背景换成海边」这种一句大白话。
// 这里把原话（+ 可选参考图）交给文本大模型，改写成扩散模型能吃的结构化提示词，
// 结果只落回输入框：用户可以先改再采纳，也可以直接放弃 —— 决定权始终在用户手上。
const promptOptOpen = ref(false)
const promptOptLoading = ref(false)
const promptOptResult = ref('')
const promptOptOriginal = ref('')
const promptOptImage = ref('')        // 打开时快照参考图，避免优化过程中图片被移除
const promptOptModelLabel = ref('')
const promptOptUsedImage = ref(false)
const promptOptError = ref('')
const promptOptElapsed = ref(0)       // 已等待秒数：慢模型时给用户一个「它还在跑」的实感
const promptOptSlow = ref(false)      // 超过 8s 补一句解释
let promptOptSeq = 0                  // 连点「重新优化」时只认最后一次响应
let promptOptTaskId = ''              // 服务端任务号（异步化后前端不再死等一个长连接）
let promptOptPollTimer = null
let promptOptElapsedTimer = null
let promptOptSleepResolve = null

const promptOptDisabled = computed(() => promptOptLoading.value || imageGenerating.value || streaming.value)

const promptOptDisabledReason = computed(() => {
  if (imageGenerating.value) return '正在出图，等这张画完再优化'
  if (!userInput.value.trim() && !pendingImages.value.length) return '先写一句想画什么/怎么改，或附一张参考图'
  return ''
})

// 优化模型：'' = 跟随聊天页当前模型；选过就记住（settings 有 deep watch 自动存 localStorage）
const chatModelItems = computed(() => modelGroups.value.flatMap(g => g.items))
const promptOptEffectiveModel = computed(() => settings.value.promptOptimizeModel || settings.value.model || '')
const promptOptEffectiveEntry = computed(
  () => chatModelItems.value.find(m => m.value === promptOptEffectiveModel.value) || null
)
const promptOptEffectiveModelLabel = computed(() => {
  if (!promptOptEffectiveModel.value) return '服务端默认（MiniMax）'
  return promptOptEffectiveEntry.value?.label || promptOptEffectiveModel.value
})
// 和后端规则保持一致：只有「局域网 Ollama + 视觉模型」会真的把参考图喂进去
const promptOptWillSeeImage = computed(() => {
  const id = promptOptEffectiveModel.value
  return !!promptOptImage.value && id.startsWith('ollama:') && !!promptOptEffectiveEntry.value?.vision
})

// 换模型后立刻用新模型重跑一次：不然弹窗里显示的文案来自旧模型，很容易看混。
// 正在跑也要重跑——本地模型慢的时候，用户换模型就是想赶紧拿到结果，
// 不能让它先把旧模型那一次等完。
function onPromptOptModelChange() {
  if (!promptOptOpen.value) return
  if (promptOptLoading.value) cancelPromptOptimize()
  runPromptOptimize()
}

function openPromptOptimize() {
  const text = userInput.value.trim()
  const ref = pendingImages.value[0]
  if (!text && !ref) {
    ElMessage.warning('先写一句你想画什么/怎么改，或者附一张参考图')
    return
  }
  promptOptOpen.value = true
  promptOptOriginal.value = text
  promptOptImage.value = ref?.dataUrl || ''
  promptOptResult.value = ''
  promptOptError.value = ''
  promptOptModelLabel.value = ''
  promptOptUsedImage.value = false
  runPromptOptimize()
}

async function runPromptOptimize() {
  if (promptOptLoading.value) return
  const seq = ++promptOptSeq
  const text = promptOptOriginal.value
  const image = promptOptImage.value
  const startedAt = Date.now()
  stopPromptOptPolling()
  cancelPromptOptTask() // 上一轮还在服务端跑就喊停，别让它白烧算力、也别抢本地模型
  promptOptLoading.value = true
  promptOptSlow.value = false
  promptOptError.value = ''
  promptOptElapsed.value = 0
  promptOptElapsedTimer = setInterval(() => {
    if (seq !== promptOptSeq) return
    promptOptElapsed.value = Math.round((Date.now() - startedAt) / 1000)
    if (promptOptElapsed.value >= 8) promptOptSlow.value = true
  }, 500)

  try {
    // 模式由「有没有原话 / 有没有图」推断，用户不用理解 t2i / edit 的区别
    const body = {
      prompt: text,
      // 弹窗里明确选过就按选的来（strict：不可用要报错，不许偷偷换模型）；
      // 没选则跟随聊天页当前模型，后端在它不可用时会回落到服务端默认。
      model: promptOptEffectiveModel.value,
      strict: !!settings.value.promptOptimizeModel,
      image_model: currentImageModel.value?.label || '',
      image_family: currentImageModel.value?.family || '',
      mode: !image ? 't2i' : (text ? 'edit' : 'describe'),
    }
    if (image) body.image = image

    // 异步任务：先建单，再短轮询。
    // 之前是一个长连接死等模型返回，而优化动辄 40~120s，会被网关按「响应超时」掐断
    // （Cloudflare 100s 直接回 524），用户只看到一句 524，其实后端还在算。
    // 现在每个请求都只有几十毫秒，代理层超时彻底不生效。
    const res = await fetch(`${API_BASE}/api/chat-plus/image/prompt/optimize/tasks`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...auth.authHeader() },
      body: JSON.stringify(body),
    })

    // 旧后端没有任务接口：退回同步接口，别让功能直接不可用
    if (res.status === 404) {
      await runPromptOptimizeSync(seq, body, startedAt)
      return
    }

    const data = await res.json().catch(() => ({}))
    if (seq !== promptOptSeq) return
    if (!res.ok) {
      const err = new Error(data.error || `HTTP ${res.status}`)
      err.biz = true // 服务端已经给了人话，别再被 netErrMsg 包一层「请求失败」
      throw err
    }
    if (!data.task_id) {
      const err = new Error('服务端没有返回任务号，请重试')
      err.biz = true
      throw err
    }
    promptOptTaskId = data.task_id
    await pollPromptOptimizeTask(seq, data.task_id)
  } catch (e) {
    if (seq === promptOptSeq) promptOptError.value = e?.biz ? e.message : netErrMsg(e)
  } finally {
    stopPromptOptPolling()
    if (seq === promptOptSeq) {
      promptOptLoading.value = false
      promptOptSlow.value = false
    }
  }
}

// 服务端任务上限 300s，这里留点余量；超过就明确告诉用户别再干等
const PROMPT_OPT_MAX_WAIT_MS = 6 * 60 * 1000

function bizError(msg) {
  const err = new Error(msg)
  err.biz = true
  return err
}

// 可被 stopPromptOptPolling 提前唤醒的等待：直接 clearTimeout 会留下一个永远不 resolve 的 Promise
function sleepPromptOpt(ms) {
  return new Promise(resolve => {
    promptOptSleepResolve = resolve
    promptOptPollTimer = setTimeout(() => {
      promptOptPollTimer = null
      promptOptSleepResolve = null
      resolve()
    }, ms)
  })
}

function stopPromptOptPolling() {
  if (promptOptPollTimer) clearTimeout(promptOptPollTimer)
  promptOptPollTimer = null
  if (promptOptElapsedTimer) clearInterval(promptOptElapsedTimer)
  promptOptElapsedTimer = null
  if (promptOptSleepResolve) {
    const resolve = promptOptSleepResolve
    promptOptSleepResolve = null
    resolve()
  }
}

// 通知服务端别再算了（关弹窗 / 重新优化时）。不 await：不该为了省算力卡住界面
function cancelPromptOptTask() {
  const id = promptOptTaskId
  promptOptTaskId = ''
  if (!id) return
  fetch(`${API_BASE}/api/chat-plus/image/prompt/optimize/tasks/${id}/cancel`, {
    method: 'POST',
    headers: { ...auth.authHeader() },
  }).catch(() => {})
}

async function pollPromptOptimizeTask(seq, taskId) {
  const deadline = Date.now() + PROMPT_OPT_MAX_WAIT_MS
  // 前 10 秒密一点（快的模型几百毫秒就回），之后放宽，避免无谓打请求
  while (seq === promptOptSeq) {
    await sleepPromptOpt(promptOptElapsed.value < 10 ? 800 : 1500)
    if (seq !== promptOptSeq) return
    if (Date.now() > deadline) throw bizError('等太久了，换一个更快的优化模型再试')

    const res = await fetch(`${API_BASE}/api/chat-plus/image/prompt/optimize/tasks/${taskId}`, {
      headers: { ...auth.authHeader() },
    })
    if (seq !== promptOptSeq) return
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      if (res.status === 404) throw bizError('任务已过期（可能服务重启了），请重新点一次优化')
      throw bizError(data.error || `HTTP ${res.status}`)
    }

    if (data.status === 'done') {
      promptOptResult.value = data.prompt || ''
      promptOptModelLabel.value = data.model || ''
      promptOptUsedImage.value = !!data.used_image
      if (!promptOptResult.value) throw bizError('AI 没有返回可用的提示词，请重试')
      return
    }
    if (data.status === 'failed') throw bizError(data.error || '优化失败，请重试')
    if (data.status === 'canceled') throw bizError('已停止优化')
    // running：继续轮询
  }
}

// 兼容老后端的同步路径（正常情况下不会走到）。同样用 AbortController 兜住网关超时。
async function runPromptOptimizeSync(seq, body, startedAt) {
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), PROMPT_OPT_MAX_WAIT_MS)
  try {
    const res = await fetch(`${API_BASE}/api/chat-plus/image/prompt/optimize`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...auth.authHeader() },
      body: JSON.stringify(body),
      signal: ctrl.signal,
    })
    const data = await res.json().catch(() => ({}))
    if (seq !== promptOptSeq) return
    if (!res.ok) throw bizError(data.error || `HTTP ${res.status}`)
    promptOptResult.value = data.prompt || ''
    promptOptModelLabel.value = data.model || ''
    promptOptUsedImage.value = !!data.used_image
    if (!promptOptResult.value) throw bizError('AI 没有返回可用的提示词，请重试')
  } catch (e) {
    if (seq !== promptOptSeq) return
    if (e?.name === 'AbortError') {
      throw bizError(`等了 ${Math.round((Date.now() - startedAt) / 1000)} 秒还没结果，请换一个更快的模型`)
    }
    throw e
  } finally {
    clearTimeout(timer)
  }
}

// 关闭弹窗：作废还在飞的请求，避免它晚到又把结果塞进来
function cancelPromptOptimize() {
  promptOptSeq++
  promptOptLoading.value = false
  stopPromptOptPolling()
  cancelPromptOptTask()
}

// 弹窗里的「停止」：和关弹窗的区别只是留在原地并给一句反馈
function stopPromptOptimize() {
  cancelPromptOptimize()
  promptOptError.value = '已停止优化'
}

function applyPromptOptimize() {
  const text = promptOptResult.value.trim()
  if (!text) return
  userInput.value = text
  promptOptOpen.value = false
  ElMessage.success('已放入输入框，可以继续修改')
}

async function applyPromptOptimizeAndSend() {
  const text = promptOptResult.value.trim()
  if (!text) return
  userInput.value = text
  promptOptOpen.value = false
  await nextTick()
  sendMessage()
}

// ══════════════ 出图（本地 ComfyUI / 在线 MiniMax） ══════════════
// 后端是「提交任务 → SSE 推进度」的异步模型，前端只负责把进度画在消息里。
// 本地与云端共用这套协议，差别只在后端按模型预设分流，前端无感。
async function sendImageTask(text) {
  if (imageGenerating.value) return
  if (!text) {
    ElMessage.warning('出图需要先写一句画面描述')
    return
  }
  if (!imageAvailable.value) {
    ElMessage.warning(`出图不可用：${imageStatus.value.error || '本地 ComfyUI 与在线 MiniMax 都未配置'}`)
    return
  }
  const mode = genMode.value
  const allImages = pendingImages.value.map(p => p.dataUrl)
  const hasRef = allImages.length > 0
  // 先确认模型能力与「有没有附参考图」匹配：不匹配会在这里就被拦下或自动换模型，
  // 免得任务跑到服务端才因为 flux2 不支持改图而失败。
  // （inpaint/outpaint/restore/multi_ref/composite 由后端自动切家族，这里直接放行）
  if (!ensureModelMatchesRef(hasRef, mode)) return
  const modelId = settings.value.imageModel || ''
  const modelLabel = currentImageModel.value?.label || ''
  // 遮罩：仅局部重绘 / 外补需要，且必须是 PNG（白色=重绘区，边缘清晰）。
  const mask = (mode === 'inpaint' || mode === 'outpaint')
    ? (pendingMask.value?.dataUrl || '')
    : ''
  if ((mode === 'inpaint' || mode === 'outpaint') && !mask) {
    ElMessage.warning('「局部重绘 / 外补」需要一张遮罩图（白色=要重绘区域）')
    return
  }
  const imgs = pendingImages.value.map(p => ({ id: p.id, dataUrl: p.dataUrl, w: p.w, h: p.h }))
  userInput.value = ''
  pendingImages.value = []
  pendingMask.value = null
  ensureConversation(text)
  messages.value.push({ id: uid(), role: 'user', content: text, images: imgs, ts: Date.now() })

  const msg = {
    id: uid(), role: 'assistant', kind: 'image',
    prompt: text, mode,
    images: allImages,                       // 多图融合/拼合传数组；单图模式后端取首张作底图
    mask,                                    // 局部重绘 / 外补遮罩
    refImage: allImages[0] || '',             // 兼容旧渲染/旧消息
    genModelId: modelId, genModel: modelLabel,
    // 云端可以一次出多张；本地固定 1 张（重跑时沿用当时的张数）
    count: currentImageModel.value?.provider === 'minimax' ? (settings.value.imageCount || 1) : 1,
    content: '', genImages: [], genError: '',
    progress: 0, phase: '排队中', generating: true, ts: Date.now(),
  }
  messages.value.push(msg)
  saveMessages()
  scrollToBottom(true)
  await runImageTask(msg)
  touchConversation()
}

// 跑（或重跑）一次出图任务，进度直接写在 msg 上
async function runImageTask(msg) {
  if (imageGenerating.value) return
  imageGenerating.value = true
  msg.generating = true
  msg.genImages = []
  msg.genError = ''
  msg.progress = 0
  msg.elapsedMs = 0
  msg.phase = '排队中'
  imageAbort = new AbortController()
  scrollToBottom()

  try {
    // conversation_id / message_id 一并上报：后端任务终态（成功图或失败原因）会直接
    // 落成服务端消息，前端掉线、关页面、切设备都不影响结果落地。
    const body = {
      prompt: msg.prompt,
      mode: msg.mode || 't2i',
      conversation_id: currentConvId.value || '',
      message_id: msg.id || '',
    }
    // 多图融合/拼合传 images 数组；单图模式（edit/inpaint/outpaint/restore）后端取首张作底图。
    if ((msg.images || []).length) {
      body.images = msg.images
    } else if (msg.refImage) {
      body.image = msg.refImage
    }
    // 局部重绘 / 外补遮罩（白色=重绘区）。
    if (msg.mask) body.mask = msg.mask
    // 一次出几张：仅云端模型支持 >1（本地 ComfyUI 单卡串行，后端会拒绝 n>1）
    if (msg.count > 1) body.n = msg.count
    // 模型预设：空 = 用服务端默认。老消息（没有 genModelId）保持后端默认行为。
    if (msg.genModelId) body.model = msg.genModelId
    const res = await fetch(`${API_BASE}/api/chat-plus/image/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...auth.authHeader() },
      body: JSON.stringify(body),
      signal: imageAbort.signal,
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`)
    // 立刻把 task_id 落到消息里：刷新页面/换设备后靠它把后台任务的结果拉回来
    msg.taskId = data.task_id
    saveMessages()
    // 任务已经进了服务端队列，接下来只是「听进度」。
    // SSE 是快通道，任何一跳链路抖动（NAT/反代空闲超时、切网、锁屏）都可能把它掐掉，
    // 但任务本身还在后台跑 —— 所以断了就转轮询，不能直接判失败。
    let settled = false
    try {
      settled = await streamImageTask(data.task_id, msg, imageAbort.signal)
    } catch (e) {
      if (e?.name === 'AbortError') throw e
      if (e?.httpStatus === 404) throw e // 任务真没了，轮询也没意义
      msg.reconnecting = true            // 仅链路断，后台照跑
    }
    if (!settled) {
      msg.reconnecting = true
      await pollImageTask(data.task_id, msg, imageAbort.signal)
    }
  } catch (e) {
    if (e?.name === 'AbortError') {
      msg.genError = '已取消生成'
      msg.taskId = '' // 用户主动取消：别再让后台接管把它拉起来
    } else {
      msg.genError = e?.biz ? e.message : netErrMsg(e)
    }
  } finally {
    msg.reconnecting = false
    msg.generating = false
    imageGenerating.value = false
    imageAbort = null
    saveMessages()
    scrollToBottom()
    resumePendingImageTasks() // 还有别的后台任务在跑就接着接管
  }
}

// SSE 快通道。返回 true = 已拿到终态（成功/失败）；false = 流正常结束但没终态。
async function streamImageTask(taskId, msg, signal) {
  const res = await fetch(`${API_BASE}/api/chat-plus/image/tasks/${encodeURIComponent(taskId)}/stream`, {
    headers: { Accept: 'text/event-stream', ...auth.authHeader() },
    signal,
  })
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    const err = new Error(data.error || `进度连接失败 HTTP ${res.status}`)
    err.httpStatus = res.status
    throw err
  }
  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buf = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })
    const { tasks, rest } = drainTaskFrames(buf)
    buf = rest
    for (const task of tasks) {
      const state = applyTaskEvent(msg, task)
      scrollToBottom()
      if (state === 'succeeded' || state === 'failed') return true
    }
  }
  return false
}

// 轮询兜底：出图是服务端后台任务，连接怎么断都不影响它跑完，
// 只要拿 task_id 反复拉快照就能把结果取回来。
async function pollImageTask(taskId, msg, signal) {
  const deadline = Date.now() + IMAGE_TASK_MAX_MS
  while (Date.now() < deadline) {
    if (signal?.aborted) throw new DOMException('aborted', 'AbortError')
    await delay(IMAGE_TASK_POLL_MS)
    if (signal?.aborted) throw new DOMException('aborted', 'AbortError')
    const res = await fetch(`${API_BASE}/api/chat-plus/image/tasks/${encodeURIComponent(taskId)}`, {
      headers: auth.authHeader(),
      signal,
    }).catch(() => null) // 单次抖动不致命，继续下一轮
    if (res && res.status === 404) {
      const err = new Error('任务已丢失（服务端重启过），请点「重新生成」重试')
      err.biz = true // 业务错误：别被 netErrMsg 包成「请求失败：…」
      throw err
    }
    if (!res || !res.ok) continue
    const task = await res.json().catch(() => null)
    if (!task) continue
    const state = applyTaskEvent(msg, task)
    scrollToBottom()
    if (state === 'succeeded' || state === 'failed') return
  }
  // 不设 genError：任务没失败，只是我们等腻了。保留 taskId，下次进会话由
  // resumePendingImageTasks 接着拉；服务端终态也会自己落库。
  msg.generating = false
  msg.phase = '仍在后台生成，稍后回到这个会话可看到结果'
}

// 接管仍在服务端跑的出图任务：出图是后台任务，页面刷新/切网/换设备都不该让它「消失」，
// 只要有 task_id 就轮询到终态。
async function resumePendingImageTasks() {
  if (imageGenerating.value) return
  const pending = messages.value.find(m => m && m.kind === 'image'
    && m.taskId && m.generating && !(m.genImages || []).length && !m.genError)
  if (!pending) return
  imageGenerating.value = true
  imageAbort = new AbortController()
  const signal = imageAbort.signal
  pending.reconnecting = true
  try {
    await pollImageTask(pending.taskId, pending, signal)
  } catch (e) {
    if (e?.name !== 'AbortError') pending.genError = pending.genError || (e?.biz ? e.message : netErrMsg(e))
  } finally {
    pending.generating = false
    pending.reconnecting = false
    imageGenerating.value = false
    imageAbort = null
    saveMessages()
    scrollToBottom()
  }
}

async function regenerateImage(msg) {
  if (imageGenerating.value) return
  await runImageTask(msg)
  touchConversation()
}

function previewImage(url) {
  if (url) window.open(url, '_blank', 'noopener')
}

// 从被网络中断的 assistant 消息处续写：截断内容已在历史里，
// 「请继续」只作为隐藏指令进本次请求（不进界面/不进后续历史）
function continueGeneration() {
  if (streaming.value) return
  doStream([{ role: 'user', content: '你上一条回复因网络中断被截断了。请从中断处直接继续输出剩余内容，不要重复已输出的部分，也不要提及中断本身。' }])
}

function copyMessage(msg) {
  navigator.clipboard.writeText(msg.content).then(() => ElMessage.success('已复制'))
}

function toggleSidebar() {
  sidebarOpen.value = !sidebarOpen.value
}

// ══════════════ 生命周期 ══════════════
let mqMobile = null
function handleViewportChange(e) {
  isMobileViewport.value = e.matches
  if (e.matches) sidebarOpen.value = false // 转竖屏/小屏时默认收起抽屉
}

// keep-alive 下切页时 body class 必须跟随激活状态，否则会锁死其他页面的滚动
function activateChatShell() {
  document.body.classList.add('chatplus-active')
  document.body.classList.toggle('chatplus-gate', !auth.isAuthenticated.value)
}
function deactivateChatShell() {
  document.body.classList.remove('chatplus-active')
  document.body.classList.remove('chatplus-gate')
}

onMounted(async () => {
  document.addEventListener('click', handleContentClick)
  activateChatShell()
  mqMobile = window.matchMedia('(max-width: 768px)')
  isMobileViewport.value = mqMobile.matches
  sidebarOpen.value = !isMobileViewport.value
  if (mqMobile.addEventListener) mqMobile.addEventListener('change', handleViewportChange)
  loadSettings()
  loadConversations()
  startSyncPolling()
  // 回到上次看的那条对话（刷新后能立刻看到刚生成/失败的结果，而不是空白草稿）
  try {
    const lastConv = localStorage.getItem(CURRENT_CONV_KEY) || ''
    if (lastConv && conversations.value.some(c => c.id === lastConv)) switchConversation(lastConv)
  } catch { /* 忽略 */ }
  await auth.tryStored() // 成功后由上面的 watch 触发 loadModels，避免重复请求
})

onUnmounted(() => {
  document.removeEventListener('click', handleContentClick)
  deactivateChatShell()
  stopSyncPolling()
  if (mqMobile && mqMobile.removeEventListener) mqMobile.removeEventListener('change', handleViewportChange)
})
// 未登录时隐藏全局宠物，避免宠物窗口压在密码卡片/输入框上；
// 登录成功（含首次输入密码）后必须再拉一次模型列表，否则下拉为空导致无法发送
watch(
  () => auth.isAuthenticated.value,
  (authed) => {
    document.body.classList.toggle('chatplus-gate', !authed)
    if (authed) {
      loadModels()
      loadImageStatus()
      // 拉到服务端会话列表（多端共用同一份），再对齐当前会话消息
      syncConversationsFromServer().then(() => {
        if (currentConvId.value) return
        const lastConv = localStorage.getItem(CURRENT_CONV_KEY) || ''
        if (lastConv && conversations.value.some(c => c.id === lastConv)) {
          switchConversation(lastConv)
          return
        }
        // 本设备第一次打开（没有"上次会话"记录）：直接进最近一条，
        // 否则新设备进来是张白纸，很容易被当成"历史全丢了"。
        if (!lastConv && conversations.value.length) switchConversation(conversations.value[0].id)
      })
      // 首次进入/输密码后：恢复上次未看到的生成结果（成功内容 / 失败原因）
      reconcileChatTasks()
    }
  },
  { immediate: true }
)

// 设置持久化（模型、系统提示词、温度、出图模式）
watch(settings, saveSettings, { deep: true })

onDeactivated(() => {
  deactivateChatShell()
  stopSyncPolling()
  if (mqMobile && mqMobile.removeEventListener) mqMobile.removeEventListener('change', handleViewportChange)
})
onActivated(() => {
  activateChatShell()
  if (mqMobile) {
    isMobileViewport.value = mqMobile.matches
    if (mqMobile.addEventListener) mqMobile.addEventListener('change', handleViewportChange)
  }
  startSyncPolling()
  // 回到本页时若模型列表为空（如首次登录时接口失败），补拉一次
  if (auth.isAuthenticated.value && modelGroups.value.length === 0) loadModels()
  // 出图能力可能随 ComfyUI 启停变化，每次回到页面都重新探活
  if (auth.isAuthenticated.value) loadImageStatus()
  // 切回来时把另一台设备的新消息对一次
  if (auth.isAuthenticated.value && currentConvId.value) refreshMessagesFromServer(currentConvId.value)
})
</script>

<style scoped>
.chatplus-page {
  padding: 0 !important;
  height: calc(100dvh - 40px); /* 桌面端：main-content 上下各 20px 内边距 */
  min-height: 0;
  gap: 0;
  overflow: hidden;
  box-sizing: border-box;
}

/* ═══════════ 密码门 ═══════════ */
.auth-gate {
  display: flex; align-items: center; justify-content: center;
  height: 100%; min-height: 420px;
}
.auth-card {
  text-align: center; padding: 44px 40px; border-radius: 20px;
  background: var(--bg-primary); border: 1px solid var(--border-light);
  box-shadow: 0 8px 32px var(--shadow-light); max-width: 380px; width: 90%;
}
.auth-logo {
  width: 56px; height: 56px; margin: 0 auto 16px; border-radius: 16px;
  display: flex; align-items: center; justify-content: center;
  font-size: 26px; color: #fff;
  background: linear-gradient(135deg, #409eff, #7c5cff);
}
.auth-card h3 { margin: 0 0 6px; font-size: 20px; color: var(--text-primary); }
.auth-card p { margin: 0 0 22px; font-size: 13px; color: var(--text-secondary); }
.auth-btn { width: 100%; margin-top: 14px; border-radius: 10px; }

/* ═══════════ 布局 ═══════════ */
.chatplus-shell {
  display: flex; height: 100%; overflow: hidden;
}

/* 侧栏 */
.cp-sidebar {
  width: 250px; flex-shrink: 0;
  display: flex; flex-direction: column;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-light);
  transition: width 0.2s ease, transform 0.25s ease;
  overflow: hidden;
}
.cp-sidebar:not(.open) { width: 0; border-right: none; }
.cp-backdrop { display: none; }
.cp-new-chat {
  margin: 12px; padding: 10px 14px;
  display: flex; align-items: center; gap: 8px;
  border: 1px dashed var(--border-base); border-radius: 10px;
  background: var(--bg-primary); color: var(--text-primary);
  cursor: pointer; font-size: 14px; white-space: nowrap;
  transition: all 0.15s;
}
.cp-new-chat:hover { border-color: var(--color-primary); color: var(--color-primary); }
.cp-new-chat .ic { font-size: 16px; font-weight: 600; }
.cp-conv-list { flex: 1; overflow-y: auto; padding: 0 8px; }
.cp-conv-item {
  display: flex; align-items: center; justify-content: space-between; gap: 6px;
  padding: 9px 12px; margin-bottom: 2px; border-radius: 8px;
  cursor: pointer; color: var(--text-secondary); font-size: 13px;
  transition: background 0.12s;
  white-space: nowrap;
}
.cp-conv-item:hover { background: var(--bg-hover); }
.cp-conv-item.active { background: var(--bg-active); color: var(--color-primary); }
.conv-title { overflow: hidden; text-overflow: ellipsis; flex: 1; }
.conv-del {
  opacity: 0; border: none; background: none; cursor: pointer;
  color: var(--text-tertiary); padding: 2px; flex-shrink: 0;
  display: flex; align-items: center;
}
.cp-conv-item:hover .conv-del { opacity: 1; }
.conv-del:hover { color: var(--color-danger); }
.cp-conv-empty { text-align: center; color: var(--text-tertiary); font-size: 12px; padding: 24px 0; }
.cp-sidebar-footer { padding: 8px; border-top: 1px solid var(--border-light); }
.cp-side-btn {
  width: 100%; display: flex; align-items: center; gap: 8px;
  padding: 8px 12px; border: none; background: none; border-radius: 8px;
  color: var(--text-secondary); font-size: 13px; cursor: pointer; white-space: nowrap;
}
.cp-side-btn:hover { background: var(--bg-hover); }

/* 主区域 */
.cp-main { flex: 1; display: flex; flex-direction: column; min-width: 0; background: var(--bg-primary); }

/* 顶栏 */
.cp-topbar {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 20px; border-bottom: 1px solid var(--border-light);
  background: var(--bg-primary);
}
.topbar-spacer { flex: 1; }
.cp-icon-btn {
  width: 34px; height: 34px; border: none; border-radius: 8px; cursor: pointer;
  background: none; color: var(--text-secondary); font-size: 17px;
  display: flex; align-items: center; justify-content: center;
}
.cp-icon-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.cp-model-select { width: 260px; }
.model-option-label { margin-right: 8px; }
.model-option-tag { font-size: 12px; color: var(--text-tertiary); }
.engine-badge {
  font-size: 12px; padding: 2px 10px; border-radius: 999px;
  border: 1px solid var(--border-base);
}
.engine-badge.on { color: var(--color-success); border-color: var(--color-success); }
.engine-badge.off { color: var(--text-tertiary); border-color: var(--border-base); }

.cp-settings-pop .setting-item { margin-bottom: 16px; }
.cp-settings-pop label { display: block; font-size: 13px; color: var(--text-secondary); margin-bottom: 8px; }

/* 消息区 */
.cp-messages { flex: 1; overflow-y: auto; scroll-behavior: auto; }
.cp-welcome {
  max-width: 680px; margin: 0 auto; padding: 80px 24px 40px; text-align: center;
}
.welcome-logo {
  width: 64px; height: 64px; margin: 0 auto 20px; border-radius: 20px;
  display: flex; align-items: center; justify-content: center;
  font-size: 30px; color: #fff;
  background: linear-gradient(135deg, #409eff, #7c5cff);
}
.cp-welcome h2 { margin: 0 0 10px; color: var(--text-primary); font-size: 24px; }
.welcome-sub { margin: 0 0 28px; color: var(--text-tertiary); font-size: 13px; }
.welcome-chips { display: flex; flex-wrap: wrap; gap: 10px; justify-content: center; }
.chip {
  padding: 9px 16px; border-radius: 12px; font-size: 13px; cursor: pointer;
  border: 1px solid var(--border-base); background: var(--bg-primary);
  color: var(--text-secondary); transition: all 0.15s;
}
.chip:hover { border-color: var(--color-primary); color: var(--color-primary); transform: translateY(-1px); }

/* 消息行 */
.cp-msg {
  display: flex; gap: 14px; max-width: 820px; margin: 0 auto;
  padding: 14px 24px;
}
.msg-avatar {
  width: 34px; height: 34px; border-radius: 10px; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  font-size: 13px; font-weight: 600;
}
.msg-avatar.user { background: var(--bg-active); color: var(--color-primary); }
.msg-avatar.assistant { background: linear-gradient(135deg, #409eff, #7c5cff); color: #fff; }
.msg-body { flex: 1; min-width: 0; }
.msg-content {
  font-size: 14.5px; line-height: 1.75; color: var(--text-primary);
  word-break: break-word; overflow-wrap: anywhere;
}
.msg-content :deep(p) { margin: 0 0 10px; }
.msg-content :deep(p:last-child) { margin-bottom: 0; }
.msg-content :deep(ul), .msg-content :deep(ol) { padding-left: 22px; margin: 8px 0; }
.msg-content :deep(blockquote) {
  margin: 10px 0; padding: 6px 14px; border-left: 3px solid var(--color-primary);
  background: var(--bg-secondary); border-radius: 0 8px 8px 0; color: var(--text-secondary);
}
.msg-content :deep(table) { border-collapse: collapse; margin: 10px 0; width: 100%; }
.msg-content :deep(th), .msg-content :deep(td) {
  border: 1px solid var(--border-base); padding: 6px 12px; font-size: 13px;
}
.msg-content :deep(a) { color: var(--link-color); }

/* 消息内图片 */
.msg-images { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 8px; }
.msg-images img {
  max-width: 240px; max-height: 240px; border-radius: 10px; border: 1px solid var(--border-base);
  object-fit: cover; background: var(--bg-secondary);
}

/* 输入区图片按钮 + 缩略图 */
.img-btn {
  flex-shrink: 0; width: 38px; height: 38px; border-radius: 10px; border: 1px solid var(--border-base);
  background: var(--bg-primary); color: var(--text-secondary); cursor: pointer; display: flex;
  align-items: center; justify-content: center; font-size: 18px; transition: all 0.15s;
}
.img-btn:hover:not(:disabled) { border-color: var(--color-primary); color: var(--color-primary); }
.img-btn.active { color: var(--color-primary); }
.img-btn:disabled { opacity: 0.4; cursor: not-allowed; }

/* ═══════════ 本地出图（ComfyUI） ═══════════ */
.gen-btn {
  flex-shrink: 0; width: 38px; height: 38px; border-radius: 10px; border: 1px solid var(--border-base);
  background: var(--bg-primary); cursor: pointer; display: flex; align-items: center;
  justify-content: center; font-size: 17px; line-height: 1; transition: all 0.15s;
}
.gen-btn:hover:not(:disabled) { border-color: var(--color-primary); }
.gen-btn.active {
  border-color: var(--color-primary);
  background: color-mix(in srgb, var(--color-primary) 12%, transparent);
}
.gen-btn:disabled { opacity: 0.4; cursor: not-allowed; }

.msg-gen { margin-bottom: 8px; }
.gen-progress {
  display: flex; flex-direction: column; gap: 6px; padding: 10px 12px;
  border-radius: 10px; background: var(--bg-secondary); border: 1px solid var(--border-base);
}
.gen-bar { height: 6px; border-radius: 3px; background: var(--border-base); overflow: hidden; }
.gen-bar i {
  display: block; height: 100%; border-radius: 3px;
  background: linear-gradient(90deg, #409eff, #7c5cff); transition: width 0.4s ease;
}
.gen-phase { font-size: 12px; color: var(--text-secondary); }
.gen-grid { display: flex; flex-wrap: wrap; gap: 10px; }
.gen-item { margin: 0; border-radius: 12px; overflow: hidden; border: 1px solid var(--border-base); background: var(--bg-secondary); }
.gen-item img { display: block; max-width: min(420px, 100%); height: auto; cursor: zoom-in; }
.gen-item figcaption {
  display: flex; justify-content: space-between; gap: 12px;
  padding: 6px 10px; font-size: 12px; color: var(--text-secondary);
}
.gen-item figcaption a { color: var(--color-primary); text-decoration: none; }
.gen-error { margin-top: 6px; font-size: 12px; color: #f56c6c; }
.gen-model-tag {
  padding: 0 6px; border-radius: 999px; font-size: 11px;
  color: var(--text-secondary, #555); background: var(--bg-hover, rgba(0, 0, 0, 0.06));
}
.composer-box.dragover { border-color: var(--color-primary); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-primary) 18%, transparent); }

.img-thumbs { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; padding: 8px 10px 0; }
.img-thumb { position: relative; width: 64px; height: 64px; border-radius: 8px; overflow: hidden; border: 1px solid var(--border-base); background: var(--bg-secondary); }
.img-thumb img { width: 100%; height: 100%; object-fit: cover; }
.img-thumb .img-kb { position: absolute; left: 0; right: 0; bottom: 0; font-size: 10px; text-align: center; color: #fff; background: rgba(0,0,0,0.5); padding: 1px 0; }
.img-thumb .img-del { position: absolute; top: 1px; right: 1px; width: 16px; height: 16px; border: none; border-radius: 50%; background: rgba(0,0,0,0.6); color: #fff; font-size: 10px; line-height: 1; cursor: pointer; }
.img-count { font-size: 12px; color: var(--text-tertiary); }
.img-warn { margin: 6px 12px 0; font-size: 12px; color: #b45309; background: #fffbeb; border: 1px solid #fde68a; padding: 6px 10px; border-radius: 8px; }
.msg-content :deep(code:not(.hljs)) {
  background: var(--code-inline-bg); padding: 2px 6px; border-radius: 5px;
  font-size: 13px; font-family: 'SF Mono', Menlo, Consolas, monospace;
}
.msg-content.streaming::after {
  content: '▊'; color: var(--color-primary); animation: blink 1s steps(1) infinite;
}
@keyframes blink { 50% { opacity: 0; } }

/* 代码块 */
.msg-content :deep(.code-block) {
  margin: 12px 0; border-radius: 12px; overflow: hidden;
  border: 1px solid var(--border-base);
}
.msg-content :deep(.code-head) {
  display: flex; align-items: center; justify-content: space-between;
  padding: 6px 14px; background: #1e1e1e; border-bottom: 1px solid #333;
}
.msg-content :deep(.code-lang) { font-size: 12px; color: #888; }
.msg-content :deep(.code-copy) {
  border: none; background: none; color: #aaa; font-size: 12px; cursor: pointer;
  padding: 2px 6px; border-radius: 4px;
}
.msg-content :deep(.code-copy:hover) { color: #fff; background: #333; }
.msg-content :deep(pre.hljs) { margin: 0; padding: 14px; overflow-x: auto; font-size: 13px; }
.msg-content :deep(pre.hljs code) { background: transparent; padding: 0; }

/* 思考过程 */
.msg-reasoning {
  margin-bottom: 10px; border: 1px solid var(--border-lighter);
  border-radius: 10px; overflow: hidden; background: var(--bg-tertiary);
}
.msg-reasoning summary {
  cursor: pointer; padding: 7px 12px; font-size: 12.5px; color: var(--text-tertiary);
  user-select: none;
}
.reasoning-content {
  padding: 4px 14px 12px; font-size: 13px; line-height: 1.7;
  color: var(--text-secondary); white-space: pre-wrap;
}

/* 操作行 */
.msg-actions { display: flex; gap: 4px; margin-top: 6px; opacity: 0; transition: opacity 0.15s; }
.cp-msg:hover .msg-actions { opacity: 1; }
.action-btn {
  border: none; background: none; cursor: pointer; color: var(--text-tertiary);
  padding: 4px; border-radius: 6px; display: flex; align-items: center;
}
.action-btn:hover { background: var(--bg-hover); color: var(--color-primary); }

/* 中断标记 */
.msg-interrupted {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
  margin-top: 6px; padding: 5px 10px;
  font-size: 12.5px; color: var(--warning-color, #d97706);
  background: color-mix(in srgb, #d97706 10%, transparent);
  border: 1px dashed color-mix(in srgb, #d97706 45%, transparent);
  border-radius: 8px; width: fit-content;
}
.continue-btn {
  border: none; cursor: pointer; font-size: 12.5px; padding: 2px 10px;
  border-radius: 999px; color: #fff;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
}
.continue-btn:hover { filter: brightness(1.1); }

/* 生成失败（异步任务终态） */
.msg-failed {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
  margin-top: 6px; padding: 5px 10px;
  font-size: 12.5px; color: var(--danger-color, #dc2626);
  background: color-mix(in srgb, #dc2626 9%, transparent);
  border: 1px dashed color-mix(in srgb, #dc2626 40%, transparent);
  border-radius: 8px; width: fit-content;
}

/* 输入区 */
.cp-composer { padding: 10px 24px 14px; }
.cp-error {
  max-width: 820px; margin: 0 auto 10px; display: flex; align-items: center; gap: 8px;
  padding: 9px 14px; border-radius: 10px; font-size: 13px;
  background: var(--danger-light); color: var(--danger-dark);
}
.err-close { margin-left: auto; border: none; background: none; cursor: pointer; color: inherit; }
.composer-box {
  max-width: 820px; margin: 0 auto;
  display: flex; align-items: flex-end; gap: 10px;
  border: 1.5px solid var(--border-base); border-radius: 16px;
  padding: 10px 12px; background: var(--bg-primary);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.composer-box.focused { border-color: var(--color-primary); box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.12); }
.composer-input { flex: 1; }
.composer-input :deep(.el-textarea__inner) {
  background: transparent; border: none; box-shadow: none;
  color: var(--text-primary); font-size: 14.5px; padding: 4px 6px;
}
.send-btn {
  width: 36px; height: 36px; flex-shrink: 0; border: none; border-radius: 12px;
  background: linear-gradient(135deg, #409eff, #7c5cff); color: #fff;
  font-size: 16px; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: opacity 0.15s, transform 0.15s;
}
.send-btn:hover:not(:disabled) { transform: scale(1.05); }
.send-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.send-btn.stop { background: var(--color-danger); }
.stop-square { width: 12px; height: 12px; background: #fff; border-radius: 2px; }
.composer-hint {
  max-width: 820px; margin: 8px auto 0; text-align: center;
  font-size: 11.5px; color: var(--text-quaternary);
}
/* 出图模型切换：做成行内小下拉，别把提示行撑成一条工具栏 */
.img-model-select {
  width: auto; min-width: 132px; display: inline-block; vertical-align: middle;
  margin: 0 2px;
}
.img-model-select :deep(.el-select__wrapper) {
  min-height: 22px; padding: 1px 6px; border-radius: 8px;
  background: var(--bg-hover, rgba(0, 0, 0, 0.04)); box-shadow: none;
  font-size: 11.5px;
}
.img-model-select :deep(.el-select__placeholder) { color: var(--text-secondary, #555); }
.mo-label { float: left; }
.mo-meta { float: right; margin-left: 14px; font-size: 11px; color: var(--text-quaternary, #999); }
/* 出图模式切换：与模型下拉同款行内小下拉 */
.gen-mode-select {
  width: auto; min-width: 96px; display: inline-block; vertical-align: middle;
  margin: 0 2px;
}
.gen-mode-select :deep(.el-select__wrapper) {
  min-height: 22px; padding: 1px 6px; border-radius: 8px;
  background: var(--bg-hover, rgba(0, 0, 0, 0.04)); box-shadow: none;
  font-size: 11.5px;
}
.gen-mode-select :deep(.el-select__placeholder) { color: var(--text-secondary, #555); }
.gen-mode-hint { margin: 0 4px; color: var(--text-tertiary, #888); }
/* 遮罩缩略图：用淡橙描边和参考图区分开 */
.img-thumb.mask-thumb { border-color: #e6a23c; }
.img-thumb.mask-thumb img { object-fit: contain; background: repeating-conic-gradient(#eee 0% 25%, #fff 0% 50%) 50% / 12px 12px; }

/* 出图模式下的「优化提示词」小按钮：和模型下拉同一行的轻量 pill，
   不做成大按钮是因为它属于「辅助」而不是主流程（主流程仍然是写完就发）。 */
.opt-pill {
  display: inline-flex; align-items: center; gap: 3px;
  margin: 2px 2px; padding: 1px 9px; border-radius: 999px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 45%, transparent);
  background: color-mix(in srgb, var(--color-primary) 10%, transparent);
  color: var(--color-primary); font-size: 11.5px; line-height: 18px;
  cursor: pointer; transition: all 0.15s; vertical-align: middle;
}
.opt-pill:hover:not(:disabled) { background: color-mix(in srgb, var(--color-primary) 18%, transparent); }
.opt-pill:disabled { opacity: 0.5; cursor: not-allowed; }
.opt-pill.busy { opacity: 1; cursor: progress; }

/* ═══════════ AI 优化提示词弹窗 ═══════════ */
/* 弹窗被 append-to-body，所以这里用独立类名 + 扁平选择器，不依赖父级作用域 */
.po-body { display: flex; flex-direction: column; gap: 14px; }
.po-head { display: flex; flex-direction: column; gap: 12px; }
/* 优化模型一行：模型名往往很长，给下拉固定上限，剩下的交给「当前：xxx」自动省略 */
.po-model { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.po-model .po-label { margin-bottom: 0; flex-shrink: 0; }
.po-model-select { width: 200px; }
.po-model-hint {
  font-size: 11.5px; color: var(--text-quaternary, #999); min-width: 0;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.po-label {
  display: block; margin-bottom: 6px; font-size: 12px; font-weight: 600;
  color: var(--text-tertiary, #888);
}
.po-origin { font-size: 13px; }
.po-origin-text {
  color: var(--text-secondary, #555); line-height: 1.6; word-break: break-word;
  padding: 8px 10px; border-radius: 8px; border-left: 3px solid var(--border-base);
  background: var(--bg-secondary, rgba(0, 0, 0, 0.03)); display: block;
}
.po-loading {
  display: flex; align-items: center; flex-wrap: wrap; gap: 8px; font-size: 13px;
  color: var(--text-secondary, #555); padding: 14px 4px;
}
/* 慢的时候多给一句解释，独占一行，别把主文案挤变形 */
.po-loading-sub {
  flex-basis: 100%; font-size: 11.5px; line-height: 1.6;
  color: var(--text-quaternary, #999); padding-left: 22px;
}
.po-spinner {
  width: 14px; height: 14px; border-radius: 50%;
  border: 2px solid color-mix(in srgb, var(--color-primary) 30%, transparent);
  border-top-color: var(--color-primary); animation: po-spin 0.8s linear infinite;
}
@keyframes po-spin { to { transform: rotate(360deg); } }
.po-error {
  font-size: 13px; line-height: 1.6; color: var(--danger-color, #dc2626);
  padding: 10px 12px; border-radius: 8px;
  background: color-mix(in srgb, #dc2626 8%, transparent);
  border: 1px dashed color-mix(in srgb, #dc2626 35%, transparent);
}
.po-meta {
  display: flex; align-items: center; gap: 2px; margin-top: 6px;
  font-size: 11.5px; color: var(--text-quaternary, #999);
}
.po-count { margin-left: auto; }
.po-footer { display: flex; align-items: center; gap: 8px; }
.po-footer-right { display: flex; gap: 8px; margin-left: auto; }
.po-btn {
  padding: 7px 14px; border-radius: 9px; font-size: 13px; cursor: pointer;
  border: 1px solid var(--border-base); background: var(--bg-primary, #fff);
  color: var(--text-primary); transition: all 0.15s;
}
.po-btn:hover:not(:disabled) { border-color: var(--color-primary); color: var(--color-primary); }
.po-btn:disabled { opacity: 0.45; cursor: not-allowed; }
.po-btn.ghost { border-style: dashed; color: var(--text-secondary, #555); }
.po-btn.primary {
  border: none; color: #fff; background: linear-gradient(135deg, #409eff, #7c5cff);
}
.po-btn.primary:hover:not(:disabled) { color: #fff; filter: brightness(1.06); }

/* 窄屏适配：侧栏改抽屉、模型选择器收缩、输入区安全区、防 iOS 聚焦缩放 */
@media (max-width: 768px) {
  .chatplus-page { height: 100%; } /* 高度由 body.chatplus-active 的 .mobile-main 精确约束 */
  .cp-backdrop {
    display: block; position: fixed; top: 56px; left: 0; right: 0; bottom: 0;
    z-index: 1999; background: rgba(0, 0, 0, 0.45);
  }
  .cp-sidebar {
    position: fixed; top: 56px; bottom: 0; left: 0; z-index: 2000;
    width: 78vw; max-width: 300px;
    transform: translateX(-105%);
    box-shadow: 6px 0 24px var(--shadow-heavy);
    border-right: 1px solid var(--border-light);
  }
  .cp-sidebar.open { width: 78vw; transform: translateX(0); }
  .cp-sidebar:not(.open) { width: 78vw; border-right: 1px solid var(--border-light); }
  .cp-topbar { padding: 8px 10px; gap: 8px; }
  .cp-model-select { width: min(52vw, 220px); }
  .engine-badge { display: none; }
  .cp-icon-btn { width: 38px; height: 38px; font-size: 19px; }
  .cp-msg { padding: 12px 12px; gap: 10px; }
  .msg-avatar { width: 30px; height: 30px; border-radius: 9px; font-size: 12px; }
  .msg-content { font-size: 15px; line-height: 1.7; }
  .cp-welcome { padding: 48px 16px 24px; }
  .cp-welcome h2 { font-size: 20px; }
  .welcome-chips { gap: 8px; }
  .chip { padding: 8px 12px; font-size: 13px; }
  .cp-messages { -webkit-overflow-scrolling: touch; }
  .cp-composer { padding: 8px 10px calc(10px + env(safe-area-inset-bottom)); }
  .composer-box { padding: 8px 8px 8px 10px; border-radius: 14px; }
  .composer-input :deep(.el-textarea__inner) { font-size: 16px; } /* <16px 会触发 iOS 聚焦自动放大 */
  .composer-hint { display: none; }
  /* 出图模式必须留着这一行：模型切换就挂在这里，藏了手机上就没法换模型 */
  .composer-hint.has-image { display: block; font-size: 11px; line-height: 1.9; }
  .msg-content :deep(pre.hljs) { font-size: 12.5px; }
}

/* 超窄屏 */
@media (max-width: 380px) {
  .cp-model-select { width: 46vw; }
  .cp-msg { padding: 10px 8px; }
}
</style>

<style>
/* 聊天页自管滚动（沿用 ChatRoom 的 body.chat-room-active 模式）：
   锁定外层滚动、隐藏页脚；移动端让 .mobile-main 精确填满固定头部(56px)之下区域，
   避免自然滚动把输入栏推出屏幕。仅在本页激活（含 keep-alive 唤醒）时生效。 */
body.chatplus-active .main-content {
  overflow: hidden;
}
body.chatplus-active .page-footer {
  display: none;
}
/* 密码门阶段隐藏全局宠物（避免压住登录卡片） */
body.chatplus-gate .pet-window,
body.chatplus-gate .pet-launcher {
  display: none !important;
}
@media (max-width: 768px) {
  body.chatplus-active .mobile-main {
    margin-top: 56px;
    height: calc(100dvh - 56px);
    height: calc(100svh - 56px); /* 键盘弹起时用可视区小高度，防输入栏被顶出屏幕 */
    padding: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
}
</style>
