<template>
  <div class="rp-page">
    <!-- ═══════════ 密码门 ═══════════ -->
    <div v-if="!auth.isAuthenticated.value" class="rp-gate">
      <div class="gate-card">
        <div class="gate-logo">🎭</div>
        <h2>角色剧场</h2>
        <p class="gate-sub">多个 AI 角色围绕你的话题自动圆桌对话 · 复用 AI Chat 的访问密码</p>
        <el-input
          v-model="auth.passwordInput.value"
          type="password"
          size="large"
          placeholder="访问密码"
          show-password
          @keyup.enter="auth.login"
        />
        <el-button type="primary" size="large" class="gate-btn" :loading="auth.loggingIn.value" @click="auth.login">
          进入剧场
        </el-button>
      </div>
    </div>

    <div v-else class="rp-shell">
      <!-- ═══════════ 顶栏 ═══════════ -->
      <header class="rp-topbar">
        <div class="tb-left">
          <h1 class="rp-title">🎭 角色剧场</h1>
          <span class="rp-sub">多角色模拟对话</span>
        </div>
        <el-select
          v-model="script.model"
          placeholder="选择模型"
          class="rp-model-select"
          :loading="modelsLoading"
          size="default"
        >
          <el-option-group v-for="g in modelGroups" :key="g.label" :label="g.label">
            <el-option v-for="m in g.items" :key="m.value" :value="m.value" :label="m.label">
              <span>{{ m.label }}</span>
              <span v-if="m.tag" class="model-option-tag">{{ m.tag }}</span>
            </el-option>
          </el-option-group>
        </el-select>
        <button class="rp-icon-btn" title="导出 Markdown" @click="exportMarkdown">⬇</button>
        <button class="rp-icon-btn" title="锁定（重新输入密码）" @click="auth.logout">🔒</button>
      </header>

      <!-- ═══════════ 进度条（演出中显示） ═══════════ -->
      <div v-if="showState !== 'idle' || completedRounds > 0" class="rp-progress">
        <div class="pg-bar">
          <div class="pg-fill" :style="{ width: progressPct + '%' }"></div>
        </div>
        <span class="pg-text">
          第 {{ Math.min(currentRound, totalRounds) }} / {{ totalRounds }} 轮
          <template v-if="showState === 'running' && currentRoleName"> · 正在发言：{{ currentRoleName }}</template>
          <template v-else-if="showState === 'paused'"> · 已暂停</template>
        </span>
      </div>

      <!-- ═══════════ 舞台：聊天记录 ═══════════ -->
      <div class="rp-stage" ref="stageRef">
        <!-- 空状态 / 模板 -->
        <div v-if="transcript.length === 0" class="rp-empty">
          <div class="empty-logo">🎭</div>
          <h2>安排一场圆桌对话</h2>
          <p class="empty-sub">配置 2-6 个角色与话题，AI 们会轮流发言、互相接话。选一个剧本快速开始：</p>
          <div class="tpl-grid">
            <button v-for="t in TEMPLATES" :key="t.name" class="tpl-card" @click="applyTemplate(t)">
              <span class="tpl-emoji">{{ t.emoji }}</span>
              <span class="tpl-name">{{ t.name }}</span>
              <span class="tpl-desc">{{ t.desc }}</span>
            </button>
          </div>
        </div>

        <!-- 消息流 -->
        <template v-for="msg in transcript" :key="msg.id">
          <div class="rp-msg">
            <div class="msg-avatar" :style="{ background: msg.color + '22', color: msg.color, borderColor: msg.color + '55' }">
              {{ msg.emoji || msg.roleName.slice(0, 1) }}
            </div>
            <div class="msg-body">
              <div class="msg-meta">
                <span class="msg-role" :style="{ color: msg.color }">{{ msg.roleName }}</span>
                <span class="msg-round">第 {{ msg.round }} 轮</span>
              </div>
              <div class="msg-content" :class="{ streaming: msg.streaming }">{{ msg.content }}<span v-if="msg.streaming" class="type-cursor"></span></div>
            </div>
          </div>
        </template>

        <div v-if="interruptedMsg" class="rp-interrupted">⚠ {{ interruptedMsg }} <button class="rp-mini-btn" @click="resumeShow">继续演出</button></div>
        <div v-if="transcript.length && showState === 'idle'" class="rp-finished">演出结束 · 共 {{ completedRounds }} 轮</div>
      </div>

      <!-- ═══════════ 控制台 ═══════════ -->
      <footer class="rp-console">
        <div class="console-row main-row">
          <button v-if="showState === 'idle'" class="rp-btn primary" :disabled="!canStart || modelsLoading" @click="startShow">
            ▶ 开演（{{ totalRounds }} 轮）
          </button>
          <button v-else-if="showState === 'running'" class="rp-btn" @click="pauseShow">
            ⏸ 暂停（本条说完）
          </button>
          <button v-else class="rp-btn primary" :disabled="!canStart" @click="resumeShow">▶ 继续</button>
          <button v-if="showState === 'running'" class="rp-btn ghost" @click="skipCurrent">跳过本条</button>
          <button v-if="showState !== 'idle'" class="rp-btn danger" @click="stopShow">■ 停止</button>
          <button v-if="showState === 'idle' && transcript.length" class="rp-btn danger ghost" @click="clearShow">清空重来</button>
          <span class="flex-spacer"></span>
          <div class="round-picker" v-if="showState === 'idle'">
            <span class="rp-label">轮数</span>
            <el-input-number v-model="script.rounds" :min="1" :max="200" :step="5" size="small" controls-position="right" />
            <span class="rp-hint">1 轮 = 每个角色各发言一次 · 最多 200</span>
          </div>
        </div>
        <p v-if="startBlockReason && showState === 'idle'" class="rp-warn">{{ startBlockReason }}</p>

        <!-- 角色与剧本（收起/展开） -->
        <div class="script-panel" :class="{ open: panelOpen }">
          <button class="panel-toggle" @click="panelOpen = !panelOpen">
            {{ panelOpen ? '▾ 收起剧本台' : '▸ 展开剧本台（角色 / 话题）' }}
          </button>
          <div v-if="panelOpen" class="panel-body">
            <div class="setting-item topic-item">
              <div class="label-row">
                <label>话题 / 场景设定（导演指令）</label>
                <button class="polish-btn" :disabled="polishing.topic" @click="polishTopic">
                  <span v-if="polishing.topic" class="spin">◌</span>{{ polishing.topic ? ' 润色中…' : '✨ ' + (script.topic.trim() ? '润色' : '帮我起草') }}
                </button>
                <button v-if="undo.topic !== null" class="undo-btn" @click="undoTopic">撤销</button>
              </div>
              <el-input
                v-model="script.topic"
                type="textarea"
                :rows="2"
                maxlength="500"
                show-word-limit
                placeholder="例：围绕「要不要给个人项目引入微服务」展开一场有分歧、有结论的讨论"
              />
            </div>
            <div class="roles-grid">
              <div v-for="(role, i) in script.roles" :key="role.id" class="role-card" :style="{ '--role-color': role.color }">
                <div class="role-head">
                  <span class="role-dot" :style="{ background: role.color }"></span>
                  <el-input v-model="role.name" size="small" maxlength="12" placeholder="角色名" class="role-name-input" />
                  <button class="polish-mini" :disabled="polishing[role.id]" title="润色这个人设" @click="polishRole(role)">
                    <span v-if="polishing[role.id]" class="spin">◌</span><span v-else>✨</span>
                  </button>
                  <el-switch v-model="role.enabled" size="small" />
                  <button
                    v-if="script.roles.length > 2"
                    class="role-del"
                    title="删除角色"
                    @click="script.roles.splice(i, 1)"
                  >✕</button>
                </div>
                <el-input
                  v-model="role.persona"
                  type="textarea"
                  :rows="2"
                  maxlength="200"
                  placeholder="人设：立场、性格、说话风格…"
                />
                <button v-if="role.id in undo.roles" class="undo-btn inline" @click="undoRole(role)">撤销润色</button>
              </div>
              <button v-if="script.roles.length < 6" class="role-add" @click="addRole">＋ 自定义角色</button>
              <button class="role-add library" @click="libraryOpen = true">
                🎭 从角色库选（{{ ROLE_LIBRARY.reduce((n, c) => n + c.roles.length, 0) }} 个预设）
              </button>
            </div>
          </div>
        </div>
      </footer>

      <!-- ═══════════ 角色库弹层 ═══════════ -->
      <div v-if="libraryOpen" class="lib-backdrop" @click.self="libraryOpen = false">
        <div class="lib-panel">
          <header class="lib-head">
            <div>
              <strong>预设角色库</strong>
              <span class="lib-count">已选 {{ script.roles.length }} / 6</span>
            </div>
            <button class="rp-icon-btn" @click="libraryOpen = false">✕</button>
          </header>
          <div class="lib-body">
            <section v-for="cat in ROLE_LIBRARY" :key="cat.category" class="lib-section">
              <h4>{{ cat.category }}</h4>
              <div class="lib-grid">
                <button
                  v-for="r in cat.roles"
                  :key="r.name"
                  class="lib-card"
                  :class="{ added: hasRoleNamed(r.name) }"
                  :disabled="hasRoleNamed(r.name) || !canAddFromLibrary"
                  @click="addFromLibrary(r)"
                >
                  <span class="lib-emoji">{{ r.emoji }}</span>
                  <span class="lib-name">{{ r.name }}</span>
                  <span class="lib-persona">{{ r.persona }}</span>
                  <span v-if="hasRoleNamed(r.name)" class="lib-badge">已加入</span>
                </button>
              </div>
            </section>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { API_BASE } from '../../api'
import { useAdminAuth } from '../../composables/useAdminAuth'

// ══════════════ 常量与持久化 ══════════════
const SCRIPT_KEY = 'roleplay-script'
const TRANSCRIPT_KEY = 'roleplay-transcript'
const MAX_ROUNDS = 200
const CONTEXT_WINDOW = 30      // 每次请求只带最近 N 条对话，防上下文爆炸
const MAX_REPLY_TOKENS = 700   // 单条发言上限，防单角色独白刷屏

const ROLE_COLORS = ['#6366f1', '#0ea5e9', '#10b981', '#f59e0b', '#ef4444', '#a855f7']

const TEMPLATES = [
  {
    name: '技术方案评审',
    emoji: '🏗',
    desc: '激进架构师 vs 保守老兵 vs 产品经理',
    topic: '我们三个正在评审一个技术方案，讨论要务实、有分歧、有结论。',
    roles: [
      { name: '架构师', emoji: '🏗', persona: '激进，偏爱新技术，追求优雅设计，容易被性能参数说服' },
      { name: '老兵', emoji: '🪖', persona: '保守务实，总考虑运维成本和故障案例，喜欢反问边界情况' },
      { name: '产品经理', emoji: '📋', persona: '只关心交付时间和用户价值，经常把话题拉回业务' },
    ],
  },
  {
    name: '正反辩论赛',
    emoji: '⚔️',
    desc: '正方 vs 反方，火药味十足',
    topic: '一场正式辩论赛，双方轮流陈词、直接反驳对方论点，针锋相对。',
    roles: [
      { name: '正方', emoji: '⚔️', persona: '立场坚定，擅长举例和数据论证，语气有力' },
      { name: '反方', emoji: '🛡', persona: '犀利毒舌，专抓对方逻辑漏洞，善用归谬法' },
    ],
  },
  {
    name: '面试模拟',
    emoji: '💼',
    desc: '面试官轮流提问，候选人应答',
    topic: '一场模拟面试，两位面试官从不同角度轮流提问并追问细节。',
    roles: [
      { name: '技术面试官', emoji: '💻', persona: '深入追问技术细节和系统设计，不给提示' },
      { name: 'HR 面试官', emoji: '🤝', persona: '关注软素质、动机与团队协作，偶尔缓解气氛' },
      { name: '候选人', emoji: '🙋', persona: '自信但诚实，答不上来会承认，善于结构化表达' },
    ],
  },
  {
    name: '三国军议',
    emoji: '🏯',
    desc: '诸葛亮 / 关羽 / 张飞 议事',
    topic: '三国时期军帐议事，主公问计，三人各抒己见，风格差异明显。',
    roles: [
      { name: '诸葛亮', emoji: '🪶', persona: '羽扇纶巾，言辞谨慎，惯用比喻和全局分析' },
      { name: '关羽', emoji: '🗡', persona: '傲气忠义，重武轻谋，话语简短有力' },
      { name: '张飞', emoji: '🐴', persona: '粗中有细，嗓门大爱打断，偶尔冒出精妙主意' },
    ],
  },
]

// ══════════════ 预设角色库 ══════════════
const ROLE_LIBRARY = [
  {
    category: '思想者 · 历史人物',
    roles: [
      { name: '苏格拉底', emoji: '🏛', persona: '只通过连续提问逼近真相，从不直接给结论，善于戳破对方的隐含假设' },
      { name: '诸葛亮', emoji: '🪶', persona: '羽扇纶巾，言辞谨慎，惯用比喻与全局推演，先谈形势再谈对策' },
      { name: '鲁迅', emoji: '🖋', persona: '冷峻犀利，一针见血，善于揭穿漂亮话背后的私心，句子短而有力' },
      { name: '苏轼', emoji: '🍵', persona: '旷达幽默，遇事先自嘲，总能把沉重话题聊出人味，爱引诗' },
      { name: '王阳明', emoji: '🧘', persona: '讲知行合一，反对空谈，遇到争论就追问"你究竟做到了什么"' },
    ],
  },
  {
    category: '职业角色',
    roles: [
      { name: '架构师', emoji: '🏗', persona: '偏爱简洁架构与清晰边界，张口就是取舍与代价，讨厌临时补丁' },
      { name: '产品经理', emoji: '📋', persona: '只关心用户价值与交付时间，常把技术讨论拉回"谁会用、值不值"' },
      { name: '测试工程师', emoji: '🧪', persona: '专职找漏洞，专挑边界条件和异常路径，最爱问"如果这里失败了呢"' },
      { name: '投资人', emoji: '💹', persona: '关心市场空间、竞争壁垒与退出路径，喜欢用类比和数字施压' },
      { name: '律师', emoji: '⚖️', persona: '谨慎精确，先界定概念与责任，习惯列出风险条款和反例' },
      { name: '医生', emoji: '🩺', persona: '循证思维，讲概率与副作用，反对绝对化表述，总提醒"先说剂量"' },
      { name: 'HR', emoji: '🤝', persona: '关注人与组织的匹配度，擅长提问动机，说话留三分余地' },
      { name: '数据分析师', emoji: '📊', persona: '凡事先问数据口径，讨厌拍脑袋，爱指出样本偏差' },
    ],
  },
  {
    category: '性格典型',
    roles: [
      { name: '杠精', emoji: '🌶', persona: '为抬杠而抬杠，专抓字眼和极端案例，被说服了也要换个角度继续杠' },
      { name: '乐观小白', emoji: '🌱', persona: '什么都不懂但充满热情，常提天真的关键问题，反而戳中要害' },
      { name: '悲观老哥', emoji: '☔', persona: '凡事想到最坏结果，用过去的失败经验泼冷水，但常常说得有道理' },
      { name: '极简主义者', emoji: '🪨', persona: '一句话能说完绝不说两句，厌恶复杂，反复追问"能不能不做"' },
      { name: '中二少年', emoji: '⚡️', persona: '中二病晚期，说话带中二修辞和宣言式夸张，但偶尔有惊人洞见' },
      { name: '冷面侦探', emoji: '🕵️', persona: '只讲证据与推理，语气冷静，善于从细节里揪出矛盾' },
    ],
  },
  {
    category: '虚构角色',
    roles: [
      { name: '机器人 AS-01', emoji: '🤖', persona: '严格按逻辑与概率表达，不理解情绪但努力模仿，偶尔冒出生硬的幽默' },
      { name: '外星观察员', emoji: '👽', persona: '以局外人视角观察人类，对人类习以为常的事提奇怪但深刻的问题' },
      { name: '老船长', emoji: '⚓️', persona: '阅历丰富，爱用航海的比喻讲风险与决策，语气粗粝但可靠' },
      { name: '毒舌前辈', emoji: '😼', persona: '嘴上不饶人，先把你骂醒再给方案，实则护短' },
    ],
  },
]

// ══════════════ 密码门（与 AI Chat 同一凭据） ══════════════
const auth = useAdminAuth({
  storageKey: 'chat-plus-access-token',
  verifyEndpoint: '/api/chat-plus/verify',
  credentialMode: 'body',
  headerName: 'X-Chat-Password',
  trustStored: true,
})

// ══════════════ 剧本状态 ══════════════
let roleSeq = 0
function newRole(name = '', persona = '', color = '', emoji = '', placeholder = false) {
  return { id: `r${Date.now().toString(36)}${roleSeq++}`, name, persona, color: color || ROLE_COLORS[0], emoji, enabled: true, placeholder }
}

const libraryOpen = ref(false)
const canAddFromLibrary = computed(() => script.value.roles.length < 6)
function hasRoleNamed(name) {
  return script.value.roles.some(r => r.name.trim() === name)
}
function addFromLibrary(preset) {
  // 首次从角色库选人：清掉未编辑的占位角色，避免出现「角色 A + 苏格拉底」这种组合
  if (script.value.roles.every(r => r.placeholder && !r.persona)) script.value.roles = []
  if (!canAddFromLibrary.value) { ElMessage.warning('最多 6 个角色'); return }
  if (hasRoleNamed(preset.name)) { ElMessage.info(`「${preset.name}」已在剧本中`); return }
  const idx = script.value.roles.length
  script.value.roles.push(newRole(preset.name, preset.persona, ROLE_COLORS[idx % ROLE_COLORS.length], preset.emoji))
  ElMessage.success(`已加入「${preset.name}」`)
}

const script = ref({
  model: '',
  topic: '',
  rounds: 10,
  roles: [newRole('角色 A', '', ROLE_COLORS[0], '', true), newRole('角色 B', '', ROLE_COLORS[1], '', true)],
})

const transcript = ref([])     // [{id, roleId, roleName, color, content, round, ts, streaming?}]
const showState = ref('idle')  // idle | running | paused
const completedRounds = ref(0)
const nextRoleIdx = ref(0)     // 暂停/恢复后从哪个角色接着说
const currentRound = ref(0)
const currentRoleId = ref('')
const interruptedMsg = ref('')
const panelOpen = ref(true)
const modelsLoading = ref(false)
const modelInfo = ref({ minimax: { available: false, models: [] }, ollama: { available: false, models: [], error: '' } })

let abortController = null
let stopAll = false
let pauseRequested = false
let skipFlag = false

function loadScript() {
  try {
    const s = JSON.parse(localStorage.getItem(SCRIPT_KEY) || 'null')
    if (s && Array.isArray(s.roles) && s.roles.length >= 2) {
      script.value = {
        model: s.model || '',
        topic: s.topic || '',
        rounds: Math.min(Math.max(parseInt(s.rounds) || 10, 1), MAX_ROUNDS),
        roles: s.roles.slice(0, 6).map(r => ({ ...newRole(), ...r, enabled: r.enabled !== false })),
      }
    }
  } catch { /* 默认 */ }
}
function saveScript() {
  localStorage.setItem(SCRIPT_KEY, JSON.stringify(script.value))
}
function loadTranscript() {
  try {
    const t = JSON.parse(localStorage.getItem(TRANSCRIPT_KEY) || 'null')
    if (t && Array.isArray(t.messages)) {
      transcript.value = t.messages
      completedRounds.value = Math.min(Math.max(parseInt(t.completedRounds) || 0, 0), MAX_ROUNDS)
      nextRoleIdx.value = Math.min(Math.max(parseInt(t.nextRoleIdx) || 0, 0), t.messages.length)
    }
  } catch { /* 默认 */ }
}
function saveTranscript() {
  const msgs = transcript.value.map(m => ({ ...m, streaming: undefined }))
  localStorage.setItem(TRANSCRIPT_KEY, JSON.stringify({ messages: msgs, completedRounds: completedRounds.value, nextRoleIdx: nextRoleIdx.value }))
}

watch(script, () => {
  for (const r of script.value.roles) if (r.placeholder && (r.persona || (r.name && r.name !== '角色 A' && r.name !== '角色 B'))) r.placeholder = false
  saveScript()
}, { deep: true })

// ══════════════ 模型列表（复用 chat-plus 接口） ══════════════
const modelGroups = computed(() => {
  const groups = []
  const ol = modelInfo.value.ollama
  if (ol.available && ol.models.length) {
    groups.push({
      label: `Ollama 本地（${ol.models.length}）`,
      items: ol.models.map(m => ({ value: m.id, label: m.name, tag: m.parameter_size || '' })),
    })
  }
  const mm = modelInfo.value.minimax
  if (mm.available) {
    groups.push({ label: 'MiniMax 云端', items: mm.models.map(m => ({ value: m.id, label: m.label, tag: m.description })) })
  }
  return groups
})

async function loadModels() {
  modelsLoading.value = true
  try {
    const res = await fetch(`${API_BASE}/api/chat-plus/models`, { headers: auth.authHeader() })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    modelInfo.value = await res.json()
    if (!script.value.model || !modelGroups.value.some(g => g.items.some(m => m.value === script.value.model))) {
      script.value.model = modelGroups.value.flatMap(g => g.items)[0]?.value || ''
    }
  } catch { /* 模型加载失败不阻塞页面，开演时再校验 */ }
  finally { modelsLoading.value = false }
}

// ══════════════ 校验与模板 ══════════════
const activeRoles = computed(() => script.value.roles.filter(r => r.enabled && r.name.trim()))
const canStart = computed(() => activeRoles.value.length >= 2 && !!script.value.model)
const startBlockReason = computed(() => {
  if (!script.value.model) return '请先选择模型'
  if (activeRoles.value.length < 2) return '至少需要 2 个启用且有名字的角色'
  return ''
})
const progressPct = computed(() => totalRounds.value ? Math.round((Math.min(currentRound.value - (currentRound.value > completedRounds.value ? 1 : 0), totalRounds.value) / totalRounds.value) * 100) : 0)
const totalRounds = computed(() => Math.min(Math.max(script.value.rounds || 10, 1), MAX_ROUNDS))
const currentRoleName = computed(() => {
  const r = transcript.value[transcript.value.length - 1]
  return r && r.streaming ? r.roleName : (script.value.roles.find(x => x.id === currentRoleId.value)?.name || '')
})

function applyTemplate(t) {
  script.value.topic = t.topic
  script.value.roles = t.roles.map((r, i) => newRole(r.name, r.persona, ROLE_COLORS[i % ROLE_COLORS.length], r.emoji || ''))
  ElMessage.success(`已载入剧本「${t.name}」`)
}
function addRole() {
  if (script.value.roles.length >= 6) return
  script.value.roles.push(newRole(`角色 ${String.fromCharCode(65 + script.value.roles.length)}`, '', ROLE_COLORS[script.value.roles.length % ROLE_COLORS.length]))
}

// ══════════════ AI 润色（场景 / 角色人设） ══════════════
const polishing = ref({})          // { topic: bool, [roleId]: bool }
const undo = ref({ topic: null, roles: {} })   // 润色前的原文（null = 无记录），供撤销

// 复用 chat/stream 端点做一次性的"补全"，把 SSE 拼成完整文本
async function callModel(system, user, maxTokens = 400) {
  if (!script.value.model) throw new Error('请先选择模型')
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), 90000) // 兜底超时，防止卡死在"润色中"
  try {
    const res = await fetch(`${API_BASE}/api/chat-plus/chat/stream`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...auth.authHeader() },
      body: JSON.stringify({
        model: script.value.model,
        messages: [{ role: 'system', content: system }, { role: 'user', content: user }],
        temperature: 0.7,
        max_tokens: maxTokens,
        stream: true,
      }),
      signal: ctrl.signal,
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error(data.error || `HTTP ${res.status}`)
    }
    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buf = '', out = ''
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buf += decoder.decode(value, { stream: true })
      const lines = buf.split('\n')
      buf = lines.pop()
      for (const raw of lines) {
        const line = raw.trim()
        if (!line.startsWith('data:')) continue
        const data = line.slice(5).trim()
        if (data === '[DONE]') continue
        let chunk
        try { chunk = JSON.parse(data) } catch { continue }
        out += chunk?.choices?.[0]?.delta?.content || ''
      }
    }
    return cleanModelText(out)
  } finally {
    clearTimeout(timer)
  }
}

// 模型偶尔会带引号、"润色后："前缀或 markdown 引用，统一剥掉
function cleanModelText(t) {
  let s = (t || '').trim()
  s = s.replace(/^```[a-z]*\s*/i, '').replace(/```$/,'').trim()
  s = s.replace(/^(润色后|优化后|建议版本|场景设定|人设)[:：]\s*/,'').trim()
  s = s.replace(/^["'“”「『](.*)["'“”」』]$/s, '$1').trim()
  return s
}

function rolesBrief() {
  return activeRoles.value.map(r => `${r.name}${r.persona ? `（${r.persona}）` : ''}`).join('；') || '（尚无角色）'
}

async function polishTopic() {
  if (polishing.value.topic) return
  if (!script.value.model) { ElMessage.warning('请先选择模型'); return }
  const prev = script.value.topic
  polishing.value = { ...polishing.value, topic: true }
  try {
    const system = '你是资深对话编剧，专门设计多角色模拟对话的"导演指令"。'
    const user = prev.trim()
      ? `请润色下面这段多角色对话的场景设定，使其更清晰、更有张力：明确讨论焦点、隐含冲突或要达成的目标、必要的背景约束，让角色能自然产生分歧与互动。\n要求：1-3 句、不超过 150 字、只输出润色后的文本（不要解释、不要引号、不要分点）。\n\n角色阵容：${rolesBrief()}\n\n原设定：${prev}`
      : `请根据下面的角色阵容，起草一段多角色对话的场景设定（导演指令）：给出一个具体、有争议、能让角色自然互相接话的话题，并点明讨论目标。\n要求：1-3 句、不超过 150 字、只输出文本（不要解释、不要引号）。\n\n角色阵容：${rolesBrief()}`
    const out = await callModel(system, user, 400)
    if (!out) throw new Error('模型没有返回内容')
    undo.value = { ...undo.value, topic: prev }
    script.value.topic = out.slice(0, 500)
    ElMessage.success(prev.trim() ? '场景已润色，不满意可点「撤销」' : '已为你起草场景')
  } catch (e) {
    ElMessage.error(`润色失败：${netErrMsg(e)}`)
  } finally {
    polishing.value = { ...polishing.value, topic: false }
  }
}

async function polishRole(role) {
  if (polishing.value[role.id]) return
  if (!script.value.model) { ElMessage.warning('请先选择模型'); return }
  const prev = role.persona
  polishing.value = { ...polishing.value, [role.id]: true }
  try {
    const others = activeRoles.value.filter(r => r.id !== role.id).map(r => r.name).join('、') || '（暂无）'
    const system = '你是角色设定师，擅长为人设写出可被 AI 稳定扮演的性格与说话风格。'
    const user = prev.trim()
      ? `请润色下面这个角色的人设，使其立场更鲜明、性格更有辨识度、说话风格可被 AI 稳定模仿，并能与其他角色形成张力。\n要求：1-2 句、不超过 120 字、只输出润色后的人设文本（不要解释、不要引号、不要"人设："前缀）。\n\n角色名：${role.name}\n其他角色：${others}\n当前话题：${script.value.topic || '（未设定）'}\n现有人设：${prev}`
      : `请为下面这个角色设计一段人设（立场、性格、说话风格），要能与其他角色形成明显差异和张力。\n要求：1-2 句、不超过 120 字、只输出人设文本（不要解释、不要引号、不要"人设："前缀）。\n\n角色名：${role.name}\n其他角色：${others}\n当前话题：${script.value.topic || '（未设定）'}`
    const out = await callModel(system, user, 300)
    if (!out) throw new Error('模型没有返回内容')
    undo.value = { ...undo.value, roles: { ...undo.value.roles, [role.id]: prev } }
    role.persona = out.slice(0, 200)
    ElMessage.success(prev.trim() ? `「${role.name}」人设已润色` : `已为「${role.name}」生成人设`)
  } catch (e) {
    ElMessage.error(`润色失败：${netErrMsg(e)}`)
  } finally {
    polishing.value = { ...polishing.value, [role.id]: false }
  }
}

function undoTopic() {
  if (undo.value.topic === null) return
  script.value.topic = undo.value.topic
  undo.value = { ...undo.value, topic: null }
  ElMessage.info('已恢复原场景')
}
function undoRole(role) {
  const prev = undo.value.roles[role.id]
  if (prev === undefined) return
  role.persona = prev
  const roles = { ...undo.value.roles }
  delete roles[role.id]
  undo.value = { ...undo.value, roles }
  ElMessage.info('已恢复原人设')
}

function buildSystemPrompt(role) {  const others = activeRoles.value.filter(r => r.id !== role.id).map(r => r.name).join('、')
  return [
    `你在一个多角色模拟对话剧场中扮演「${role.name}」。${role.persona || ''}`,
    script.value.topic ? `【场景/话题】${script.value.topic}` : '',
    `其他在场角色：${others}。`,
    '规则：',
    `1. 只输出「${role.name}」的发言正文；不要旁白、不要替其他角色说话、不要输出自己的名字前缀。`,
    '2. 要回应其他角色上一条发言，可以补充、反驳、提问，保持对话自然推进。',
    `3. 每次发言控制在 200 字以内，不说空话套话。`,
  ].filter(Boolean).join('\n')
}

// 历史映射：当前角色自己的旧发言 → assistant，其他人 → user（带名字前缀），保持轮流感
function buildPayloadMessages(role) {
  const msgs = [{ role: 'system', content: buildSystemPrompt(role) }]
  const history = transcript.value.filter(m => !m.streaming).slice(-CONTEXT_WINDOW)
  if (history.length === 0) {
    msgs.push({ role: 'user', content: `（对话现在开始。请以「${role.name}」的身份发起第一条发言。）` })
  }
  for (const m of history) {
    if (m.roleId === role.id) {
      msgs.push({ role: 'assistant', content: m.content })
    } else {
      msgs.push({ role: 'user', content: `【${m.roleName}】：${m.content}` })
    }
  }
  return msgs
}

// ══════════════ 演出主循环（前端驱动，零后端改动） ══════════════
async function startShow() {
  if (!canStart.value) { ElMessage.warning(startBlockReason.value); return }
  stopAll = false; pauseRequested = false; interruptedMsg.value = ''
  showState.value = 'running'
  await runLoop()
}

function pauseShow() { pauseRequested = true; ElMessage.info('本条发言结束后暂停') }
function resumeShow() {
  if (showState.value !== 'running') { interruptedMsg.value = ''; showState.value = 'running'; runLoop() }
}
function stopShow() {
  stopAll = true
  abortController?.abort() // 立即切断当前发言（partial 保留）
}
function skipCurrent() {
  skipFlag = true
  abortController?.abort()
}
function clearShow() {
  transcript.value = []
  completedRounds.value = 0
  nextRoleIdx.value = 0
  currentRound.value = 0
  interruptedMsg.value = ''
  saveTranscript()
}

async function runLoop() {
  interruptedMsg.value = ''
  let stoppedByUser = false
  try {
    outer:
    for (let r = completedRounds.value + (nextRoleIdx.value === 0 ? 1 : 0); r <= totalRounds.value; r++) {
      currentRound.value = r
      const roles = activeRoles.value
      if (roles.length < 2) break
      for (let i = nextRoleIdx.value; i < roles.length; i++) {
        if (stopAll || pauseRequested) { stoppedByUser = !stopAll; break outer }
        nextRoleIdx.value = i
        currentRoleId.value = roles[i].id
        const ok = await speakRole(roles[i], r)
        if (!ok) break outer // 网络失败已暂停
        nextRoleIdx.value = (i + 1) % roles.length
        if (i + 1 >= roles.length) {
          completedRounds.value = r
          nextRoleIdx.value = 0
        }
        saveTranscript()
      }
    }
  } finally {
    showState.value = stopAll ? 'idle' : 'paused'
    saveTranscript()
    if (stopAll) ElMessage.success(`演出已停止，共完成 ${completedRounds.value} 轮`)
  }
}

// 单条发言：流式请求 + 一次自动重试；失败返回 false（进入暂停态，可手动继续）
async function speakRole(role, round) {
  const msg = { id: uid(), roleId: role.id, roleName: role.name, color: role.color, emoji: role.emoji || '', content: '', round, ts: Date.now(), streaming: true }
  transcript.value.push(msg)
  scrollToBottom(true)

  for (let attempt = 0; attempt < 2; attempt++) {
    skipFlag = false
    abortController = new AbortController()
    try {
      const res = await fetch(`${API_BASE}/api/chat-plus/chat/stream`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...auth.authHeader() },
        body: JSON.stringify({
          model: script.value.model,
          messages: buildPayloadMessages(role),
          temperature: 0.8,
          max_tokens: MAX_REPLY_TOKENS,
          stream: true,
        }),
        signal: abortController.signal,
      })
      if (!res.ok) {
        const data = await res.json().catch(() => ({}))
        throw new Error(data.error || `HTTP ${res.status}`)
      }
      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buf = ''
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buf += decoder.decode(value, { stream: true })
        const lines = buf.split('\n')
        buf = lines.pop()
        for (const raw of lines) {
          const line = raw.trim()
          if (!line.startsWith('data:')) continue
          const data = line.slice(5).trim()
          if (data === '[DONE]') continue
          let chunk
          try { chunk = JSON.parse(data) } catch { continue }
          const piece = chunk?.choices?.[0]?.delta?.content || ''
          if (piece) { msg.content += piece; if (msg.content.length % 60 < piece.length) scrollToBottom() }
        }
      }
      msg.streaming = false
      // 空回复兜底：塞一条占位，避免对话断裂
      if (!msg.content.trim()) msg.content = '（沉默片刻，没有发言）'
      return true
    } catch (e) {
      msg.streaming = false
      if (skipFlag) { // 主动跳过：保留 partial，继续下一条
        if (!msg.content.trim()) transcript.value.splice(transcript.value.indexOf(msg), 1)
        return true
      }
      if (e?.name === 'AbortError') { // stopShow 触发
        if (!msg.content.trim()) transcript.value.splice(transcript.value.indexOf(msg), 1)
        return false
      }
      // 真实网络错误：重试一次，再失败 → 暂停 + 提示
      if (attempt === 0) {
        await new Promise(r => setTimeout(r, 1500))
        msg.streaming = true
        continue
      }
      if (!msg.content.trim()) transcript.value.splice(transcript.value.indexOf(msg), 1)
      interruptedMsg.value = `第 ${round} 轮「${role.name}」发言失败：${netErrMsg(e)}`
      return false
    } finally {
      abortController = null
    }
  }
  return false
}

function netErrMsg(e) {
  const msg = (e?.message || '').toLowerCase()
  if (msg.includes('load failed') || msg.includes('failed to fetch') || msg.includes('networkerror') || msg.includes('timed out')) {
    return '网络连接中断（常见于切网/锁屏/弱网）'
  }
  return e?.message || String(e)
}

function uid() { return Date.now().toString(36) + Math.random().toString(36).slice(2, 8) }

// ══════════════ 滚动与导出 ══════════════
const stageRef = ref(null)
let stickBottom = true
function onStageScroll() {
  const el = stageRef.value
  if (!el) return
  stickBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80
}
function scrollToBottom(force = false) {
  if (!force && !stickBottom) return
  nextTick(() => { const el = stageRef.value; if (el) el.scrollTop = el.scrollHeight })
}

function exportMarkdown() {
  const lines = [`# 角色剧场对话记录`, '', `- 话题：${script.value.topic || '（未设定）'}`, `- 模型：${script.value.model}`, `- 轮数：${completedRounds.value}`, '']
  for (const m of transcript.value) lines.push(`**【${m.roleName}】**（第 ${m.round} 轮）：${m.content}`, '')
  const blob = new Blob([lines.join('\n')], { type: 'text/markdown;charset=utf-8' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `roleplay-${new Date().toISOString().slice(0, 10)}.md`
  a.click()
  URL.revokeObjectURL(a.href)
}

// ══════════════ 生命周期 ══════════════
onMounted(async () => {
  loadScript()
  loadTranscript()
  panelOpen.value = transcript.value.length === 0
  await auth.tryStored()
  if (auth.isAuthenticated.value) loadModels()
})

watch(() => auth.isAuthenticated.value, (authed) => { if (authed && !modelsLoading.value) loadModels() })
</script>

<style scoped>
.rp-page { padding: 16px; display: flex; flex-direction: column; }

/* 密码门 */
.rp-gate { display: flex; align-items: center; justify-content: center; min-height: 60vh; }
.gate-card {
  display: flex; flex-direction: column; gap: 14px; align-items: center;
  padding: 40px 36px; border-radius: 16px; max-width: 380px; width: 100%;
  background: var(--bg-card, var(--bg-secondary, #fff));
  border: 1px solid var(--border-light, #e5e7eb); box-shadow: var(--shadow, 0 4px 16px rgba(0,0,0,.06));
}
.gate-logo { font-size: 44px; }
.gate-card h2 { margin: 0; font-size: 22px; color: var(--text-primary, #111); }
.gate-sub { margin: 0; font-size: 13px; color: var(--text-tertiary, #888); text-align: center; }
.gate-card :deep(.el-input) { width: 100%; }
.gate-btn { width: 100%; }
.gate-err { font-size: 13px; color: var(--el-color-danger, #f56c6c); }

/* 布局骨架 */
.rp-shell { display: flex; flex-direction: column; gap: 12px; }
.rp-topbar { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.tb-left { display: flex; align-items: baseline; gap: 10px; }
.rp-title { margin: 0; font-size: 20px; color: var(--text-primary, #111); }
.rp-sub { font-size: 12.5px; color: var(--text-tertiary, #888); }
.rp-model-select { width: 240px; }
.rp-icon-btn {
  border: 1px solid var(--border-light, #e5e7eb); background: var(--bg-card, transparent); color: var(--text-secondary, #555);
  width: 36px; height: 36px; border-radius: 10px; cursor: pointer; font-size: 16px;
}
.rp-icon-btn:hover { color: var(--color-primary, #6366f1); border-color: var(--color-primary, #6366f1); }

/* 进度 */
.rp-progress { display: flex; align-items: center; gap: 12px; }
.pg-bar { flex: 1; height: 6px; border-radius: 3px; background: var(--bg-hover, #eee); overflow: hidden; }
.pg-fill { height: 100%; border-radius: 3px; background: linear-gradient(90deg, #6366f1, #8b5cf6); transition: width 0.4s ease; }
.pg-text { font-size: 13px; color: var(--text-secondary, #555); white-space: nowrap; }

/* 舞台 */
.rp-stage {
  border: 1px solid var(--border-light, #e5e7eb); border-radius: 14px;
  background: var(--bg-card, var(--bg-secondary, #fafafa));
  padding: 18px; display: flex; flex-direction: column; gap: 14px;
  max-height: calc(100dvh - 320px); min-height: 260px; overflow-y: auto;
}
.rp-empty { text-align: center; padding: 30px 10px; }
.empty-logo { font-size: 46px; }
.rp-empty h2 { margin: 10px 0 6px; font-size: 20px; color: var(--text-primary, #111); }
.empty-sub { margin: 0 0 18px; font-size: 13.5px; color: var(--text-tertiary, #888); }
.tpl-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 10px; max-width: 760px; margin: 0 auto; }
.tpl-card {
  display: flex; flex-direction: column; gap: 4px; align-items: flex-start; text-align: left;
  padding: 14px; border-radius: 12px; cursor: pointer;
  border: 1px solid var(--border-light, #e5e7eb); background: var(--bg-card, #fff);
  transition: transform 0.15s, border-color 0.15s;
}
.tpl-card:hover { transform: translateY(-2px); border-color: var(--color-primary, #6366f1); }
.tpl-emoji { font-size: 22px; }
.tpl-name { font-weight: 600; font-size: 14.5px; color: var(--text-primary, #111); }
.tpl-desc { font-size: 12.5px; color: var(--text-tertiary, #888); }

.rp-msg { display: flex; gap: 10px; }
.msg-avatar {
  width: 36px; height: 36px; flex-shrink: 0; border-radius: 11px;
  display: flex; align-items: center; justify-content: center;
  font-weight: 700; font-size: 15px; border: 1.5px solid;
}
.msg-body { min-width: 0; flex: 1; }
.msg-meta { display: flex; gap: 8px; align-items: baseline; margin-bottom: 3px; }
.msg-role { font-size: 13.5px; font-weight: 600; }
.msg-round { font-size: 11.5px; color: var(--text-tertiary, #999); }
.msg-content {
  font-size: 14.5px; line-height: 1.7; color: var(--text-primary, #222); white-space: pre-wrap; word-break: break-word;
  background: var(--bg-secondary, #f5f5f5); border-radius: 4px 12px 12px 12px; padding: 9px 13px;
}
.msg-content.streaming::after { content: ''; }
.type-cursor { display: inline-block; width: 2px; height: 1em; background: var(--color-primary, #6366f1); margin-left: 2px; vertical-align: text-bottom; animation: blink 0.8s infinite; }
@keyframes blink { 50% { opacity: 0; } }

.rp-interrupted {
  align-self: center; display: flex; gap: 10px; align-items: center;
  padding: 6px 14px; border-radius: 999px; font-size: 13px;
  color: #d97706; background: rgba(217, 119, 6, 0.1); border: 1px dashed rgba(217, 119, 6, 0.45);
}
.rp-mini-btn {
  border: none; border-radius: 999px; padding: 2px 12px; cursor: pointer; font-size: 12.5px; color: #fff;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
}
.rp-finished { text-align: center; font-size: 13px; color: var(--text-tertiary, #999); padding: 8px; }

/* 控制台 */
.rp-console {
  border: 1px solid var(--border-light, #e5e7eb); border-radius: 14px;
  background: var(--bg-card, #fff); padding: 12px 14px;
  display: flex; flex-direction: column; gap: 8px;
}
.console-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.flex-spacer { flex: 1; }
.rp-btn {
  border: 1px solid var(--border-light, #dcdfe6); background: var(--bg-card, #fff); color: var(--text-primary, #333);
  padding: 8px 18px; border-radius: 10px; cursor: pointer; font-size: 14px;
  transition: filter 0.15s, transform 0.1s;
}
.rp-btn:hover:not(:disabled) { filter: brightness(0.97); }
.rp-btn:active:not(:disabled) { transform: scale(0.98); }
.rp-btn:disabled { opacity: 0.45; cursor: not-allowed; }
.rp-btn.primary { background: linear-gradient(135deg, #6366f1, #8b5cf6); border: none; color: #fff; }
.rp-btn.danger { color: var(--el-color-danger, #f56c6c); border-color: var(--el-color-danger-light-5, #fde2e2); }
.rp-btn.danger.ghost { background: transparent; }
.rp-btn.ghost { background: transparent; }

.round-picker { display: flex; align-items: center; gap: 8px; }
.rp-label { font-size: 13px; color: var(--text-secondary, #555); }
.rp-hint { font-size: 12px; color: var(--text-tertiary, #999); }
.rp-warn { margin: 0; font-size: 12.5px; color: #d97706; }

/* 剧本台 */
.script-panel { border-top: 1px dashed var(--border-light, #e5e7eb); padding-top: 8px; }
.panel-toggle {
  border: none; background: none; cursor: pointer; color: var(--text-secondary, #666); font-size: 13px; padding: 2px 0;
}
.panel-toggle:hover { color: var(--color-primary, #6366f1); }
.panel-body { display: flex; flex-direction: column; gap: 12px; padding-top: 10px; }
.setting-item label { display: block; font-size: 13px; color: var(--text-secondary, #555); margin-bottom: 6px; }
.label-row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.label-row label { margin-bottom: 0; }
.polish-btn {
  border: 1px solid color-mix(in srgb, var(--color-primary, #6366f1) 40%, transparent);
  background: color-mix(in srgb, var(--color-primary, #6366f1) 8%, transparent);
  color: var(--color-primary, #6366f1); font-size: 12px; padding: 2px 10px; border-radius: 999px; cursor: pointer;
}
.polish-btn:hover:not(:disabled) { background: color-mix(in srgb, var(--color-primary, #6366f1) 16%, transparent); }
.polish-btn:disabled { opacity: 0.65; cursor: default; }
.polish-mini {
  border: none; background: none; cursor: pointer; font-size: 13px; line-height: 1; padding: 2px 4px;
  color: var(--color-primary, #6366f1); border-radius: 6px;
}
.polish-mini:hover:not(:disabled) { background: color-mix(in srgb, var(--color-primary, #6366f1) 12%, transparent); }
.polish-mini:disabled { opacity: 0.6; cursor: default; }
.undo-btn {
  border: none; background: none; cursor: pointer; font-size: 12px; color: var(--text-tertiary, #999);
  text-decoration: underline; padding: 0 2px;
}
.undo-btn:hover { color: var(--color-primary, #6366f1); }
.undo-btn.inline { align-self: flex-start; margin-top: 4px; }
.spin { display: inline-block; animation: rp-spin 1s linear infinite; }
@keyframes rp-spin { to { transform: rotate(360deg); } }
.roles-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(250px, 1fr)); gap: 10px; }
.role-card {
  border: 1px solid var(--border-light, #e5e7eb); border-radius: 12px; padding: 10px;
  display: flex; flex-direction: column; gap: 8px; background: var(--bg-secondary, #fafafa);
  border-left: 3px solid var(--role-color, #6366f1);
}
.role-head { display: flex; align-items: center; gap: 8px; }
.role-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.role-name-input { flex: 1; }
.role-del { border: none; background: none; cursor: pointer; color: var(--text-tertiary, #999); font-size: 13px; padding: 2px; }
.role-del:hover { color: var(--el-color-danger, #f56c6c); }
.role-add {
  border: 1.5px dashed var(--border-light, #dcdfe6); border-radius: 12px; background: none; cursor: pointer;
  color: var(--text-tertiary, #999); font-size: 13.5px; min-height: 84px;
}
.role-add:hover { border-color: var(--color-primary, #6366f1); color: var(--color-primary, #6366f1); }
.role-add.library { border-style: solid; color: var(--color-primary, #6366f1); border-color: color-mix(in srgb, var(--color-primary, #6366f1) 35%, transparent); }
.role-add.library:hover { background: color-mix(in srgb, var(--color-primary, #6366f1) 6%, transparent); }

/* 角色库弹层 */
.lib-backdrop {
  position: fixed; inset: 0; z-index: 2100; background: rgba(0, 0, 0, 0.45);
  display: flex; align-items: center; justify-content: center; padding: 20px;
}
.lib-panel {
  width: min(920px, 100%); max-height: min(80vh, 720px); display: flex; flex-direction: column;
  background: var(--bg-card, #fff); border-radius: 16px; overflow: hidden;
  box-shadow: var(--shadow-heavy, 0 20px 60px rgba(0, 0, 0, 0.3));
}
.lib-head {
  display: flex; align-items: center; justify-content: space-between; gap: 10px;
  padding: 14px 18px; border-bottom: 1px solid var(--border-light, #eee);
}
.lib-head strong { font-size: 16px; color: var(--text-primary, #111); margin-right: 10px; }
.lib-count { font-size: 12.5px; color: var(--text-tertiary, #999); }
.lib-body { overflow-y: auto; padding: 14px 18px 20px; }
.lib-section { margin-bottom: 18px; }
.lib-section h4 { margin: 0 0 10px; font-size: 13px; font-weight: 600; color: var(--text-secondary, #666); }
.lib-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(190px, 1fr)); gap: 10px; }
.lib-card {
  position: relative; display: flex; flex-direction: column; gap: 3px; text-align: left;
  padding: 11px 12px; border-radius: 12px; cursor: pointer;
  border: 1px solid var(--border-light, #e5e7eb); background: var(--bg-secondary, #fafafa);
  transition: border-color 0.15s, transform 0.12s;
}
.lib-card:hover:not(:disabled) { border-color: var(--color-primary, #6366f1); transform: translateY(-1px); }
.lib-card:disabled { opacity: 0.5; cursor: not-allowed; }
.lib-card.added { opacity: 0.65; }
.lib-emoji { font-size: 18px; line-height: 1; }
.lib-name { font-size: 14px; font-weight: 600; color: var(--text-primary, #111); }
.lib-persona {
  font-size: 12px; color: var(--text-tertiary, #888); line-height: 1.45;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
}
.lib-badge {
  position: absolute; top: 8px; right: 8px; font-size: 10.5px; padding: 1px 7px; border-radius: 999px;
  color: #fff; background: var(--color-primary, #6366f1);
}

.model-option-tag { float: right; font-size: 12px; color: var(--text-tertiary, #999); }

/* 移动端 */
@media (max-width: 768px) {
  .rp-page { padding: 10px; }
  .rp-stage { max-height: calc(100dvh - 380px); padding: 12px; }
  .rp-model-select { width: min(52vw, 220px); }
  .rp-title { font-size: 17px; }
  .rp-sub { display: none; }
  .msg-content { font-size: 15px; }
  .tpl-grid { grid-template-columns: 1fr 1fr; }
  .roles-grid { grid-template-columns: 1fr; }
  .rp-hint { display: none; }
}
</style>
