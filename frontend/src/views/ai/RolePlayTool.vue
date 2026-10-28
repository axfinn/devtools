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

      <!-- ═══════════ 场景组切换 ═══════════ -->
      <div class="rp-groups">
        <div class="group-chips">
          <button
            v-for="g in groups"
            :key="g.id"
            class="group-chip"
            :class="{ active: g.id === currentGroupId }"
            :title="g.topic || '（未设定场景）'"
            @click="switchGroup(g.id)"
          >
            <span class="g-name">{{ g.name }}</span>
            <span v-if="g.messages && g.messages.length" class="g-count">{{ g.messages.length }}</span>
          </button>
        </div>
        <div class="group-ops">
          <button class="group-op primary" @click="createGroup()">＋ 新建场景</button>
          <button class="group-op" @click="renameGroup">✎ 重命名</button>
          <button class="group-op danger" @click="deleteGroup">🗑 删除</button>
        </div>
      </div>

      <!-- ═══════════ 进度条（演出中显示） ═══════════ -->
      <div v-if="showState !== 'idle' || completedRounds > 0" class="rp-progress">
        <div class="pg-bar">
          <div class="pg-fill" :style="{ width: progressPct + '%' }"></div>
        </div>
        <span class="pg-text">
          第 {{ Math.min(Math.max(currentRound, 1), totalRounds) }} / {{ totalRounds }} 轮
          <template v-if="showState === 'running' && currentRoleName"> · 正在发言：{{ currentRoleName }}</template>
          <template v-else-if="showState === 'paused'"> · 已暂停</template>
        </span>
      </div>

      <!-- ═══════════ 模型加载失败提示（点此重试） ═══════════ -->
      <div v-if="modelsError" class="rp-models-err">
        ⚠ {{ modelsError }}
        <button class="rp-mini-btn" :disabled="modelsLoading" @click="loadModels">{{ modelsLoading ? '重试中…' : '重试' }}</button>
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
              <div class="msg-content" :class="{ streaming: msg.streaming }">
                <span v-if="msg.streaming && !msg.content" class="typing-dots"><i></i><i></i><i></i></span>
                <template v-else>{{ msg.content }}<span v-if="msg.streaming" class="type-cursor"></span></template>
              </div>
            </div>
          </div>
        </template>

        <div v-if="interruptedMsg" class="rp-interrupted">⚠ {{ interruptedMsg }} <button class="rp-mini-btn" @click="resumeShow">继续演出</button></div>
        <div v-if="transcript.length && showState === 'idle'" class="rp-finished">演出结束 · 共 {{ completedRounds }} 轮</div>
      </div>

      <!-- ═══════════ 控制台 ═══════════ -->
      <footer class="rp-console">
        <div class="console-row main-row">
          <button v-if="showState === 'running'" class="rp-btn" @click="pauseShow">
            ⏸ 暂停（本条说完）
          </button>
          <button v-else class="rp-btn primary" :disabled="!canStart || modelsLoading" @click="beginShow">
            <template v-if="completedRounds > 0 && completedRounds < totalRounds">▶ 继续（剩 {{ totalRounds - completedRounds }} 轮）</template>
            <template v-else>▶ 开演（{{ totalRounds }} 轮）</template>
          </button>
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
            <p v-if="!script.roles.length" class="roles-empty">当前组还没有角色，点下方「🎭 从角色库选」挑几个人，或「＋ 自定义角色」。</p>
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
              <button v-if="script.roles.length" class="role-add danger" @click="clearRoles">🗑 清空全部角色</button>
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
import { ElMessage, ElMessageBox } from 'element-plus'
import { API_BASE } from '../../api'
import { useAdminAuth } from '../../composables/useAdminAuth'

// ══════════════ 常量与持久化 ══════════════
const GROUPS_KEY = 'roleplay-groups-v2'
const LEGACY_SCRIPT_KEY = 'roleplay-script'
const LEGACY_TRANSCRIPT_KEY = 'roleplay-transcript'
const MAX_ROUNDS = 200
const CONTEXT_WINDOW = 24      // 每次请求只带最近 N 条对话，防上下文爆炸，也逼模型向前推进
const MAX_REPLY_TOKENS = 700   // 单条发言上限；给推理模型留足"思考+说话"的 token，避免内容被截断

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

const groups = ref([])         // 场景组：[{id,name,model,topic,rounds,roles,messages,completedRounds,nextRoleIdx,updatedAt}]
const currentGroupId = ref('')
const currentGroup = computed(() => groups.value.find(g => g.id === currentGroupId.value) || groups.value[0])
const transcript = ref([])     // 当前组的消息 [{id, roleId, roleName, color, content, round, ts, streaming?}]
const showState = ref('idle')  // idle | running | paused
const completedRounds = ref(0)
const nextRoleIdx = ref(0)     // 暂停/恢复后从哪个角色接着说
const currentRound = ref(0)
const currentRoleId = ref('')
const interruptedMsg = ref('')
const panelOpen = ref(true)
const modelsLoading = ref(false)
const modelsError = ref('')
const modelInfo = ref({ minimax: { available: false, models: [] }, ollama: { available: false, models: [], error: '' } })

let abortController = null
let stopAll = false
let pauseRequested = false
let skipFlag = false

// ── 场景组持久化：一组 = 一套剧本 + 它的对话记录 ──
function defaultRoles() {
  return [newRole('角色 A', '', ROLE_COLORS[0], '', true), newRole('角色 B', '', ROLE_COLORS[1], '', true)]
}

function normalizeGroup(g) {
  return {
    id: g.id || uid(),
    name: (g.name || '场景').slice(0, 40),
    model: g.model || '',
    topic: g.topic || '',
    rounds: Math.min(Math.max(parseInt(g.rounds) || 10, 1), MAX_ROUNDS),
    roles: (Array.isArray(g.roles) ? g.roles.slice(0, 6) : defaultRoles()).map(r => ({ ...newRole(), ...r, enabled: r.enabled !== false })),
    messages: Array.isArray(g.messages) ? g.messages.map(m => ({ ...m, streaming: undefined })) : [],
    completedRounds: Math.min(Math.max(parseInt(g.completedRounds) || 0, 0), MAX_ROUNDS),
    nextRoleIdx: Math.max(parseInt(g.nextRoleIdx) || 0, 0),
    updatedAt: g.updatedAt || Date.now(),
  }
}

// 把当前编辑态写回所在组
function snapshotCurrent() {
  const g = currentGroup.value
  if (!g) return
  g.model = script.value.model
  g.topic = script.value.topic
  g.rounds = script.value.rounds
  g.roles = JSON.parse(JSON.stringify(script.value.roles))
  g.messages = transcript.value.map(m => ({ ...m, streaming: undefined }))
  g.completedRounds = completedRounds.value
  g.nextRoleIdx = nextRoleIdx.value
  g.updatedAt = Date.now()
}

let persistTimer = null
function persist(immediate = false) {
  const write = () => {
    snapshotCurrent()
    try {
      localStorage.setItem(GROUPS_KEY, JSON.stringify({ groups: groups.value, currentId: currentGroupId.value, lastModel: script.value.model }))
    } catch { /* 配额满时静默，避免中断演出 */ }
  }
  if (immediate) { clearTimeout(persistTimer); write(); return }
  clearTimeout(persistTimer)
  persistTimer = setTimeout(write, 400)   // 演出中高频写入做节流
}

// 原 saveScript/saveTranscript 保留同名，内部走整组持久化
function saveScript() { persist() }
function saveTranscript() { persist() }

function loadViewFromGroup(g) {
  script.value = {
    model: g.model || '',
    topic: g.topic || '',
    rounds: g.rounds || 10,
    roles: JSON.parse(JSON.stringify(g.roles || defaultRoles())),
  }
  transcript.value = JSON.parse(JSON.stringify(g.messages || []))
  completedRounds.value = g.completedRounds || 0
  nextRoleIdx.value = g.nextRoleIdx || 0
  currentRound.value = completedRounds.value || 0
  currentRoleId.value = ''
  interruptedMsg.value = ''
  undo.value = { topic: null, roles: {} }
  showState.value = 'idle'
}

function loadGroups() {
  let loaded = null
  try { loaded = JSON.parse(localStorage.getItem(GROUPS_KEY) || 'null') } catch { loaded = null }
  if (loaded && Array.isArray(loaded.groups) && loaded.groups.length) {
    groups.value = loaded.groups.map(normalizeGroup)
    currentGroupId.value = groups.value.some(g => g.id === loaded.currentId) ? loaded.currentId : groups.value[0].id
  } else {
    // 迁移旧版单场景数据，避免用户之前的剧本丢失
    let oldScript = null, oldTr = null
    try { oldScript = JSON.parse(localStorage.getItem(LEGACY_SCRIPT_KEY) || 'null') } catch { /* ignore */ }
    try { oldTr = JSON.parse(localStorage.getItem(LEGACY_TRANSCRIPT_KEY) || 'null') } catch { /* ignore */ }
    const g = normalizeGroup({
      id: uid(),
      name: '场景 1',
      model: oldScript?.model || '',
      topic: oldScript?.topic || '',
      rounds: oldScript?.rounds || 10,
      roles: oldScript?.roles || defaultRoles(),
      messages: oldTr?.messages || [],
      completedRounds: oldTr?.completedRounds || 0,
      nextRoleIdx: oldTr?.nextRoleIdx || 0,
    })
    groups.value = [g]
    currentGroupId.value = g.id
  }
  loadViewFromGroup(currentGroup.value)
}

// ── 场景组操作 ──
function createGroup(fromTemplate = null) {
  if (showState.value === 'running') { ElMessage.warning('先停止演出再切换场景'); return }
  persist(true)
  const n = groups.value.length + 1
  const g = normalizeGroup({
    id: uid(),
    name: `场景 ${n}`,
    model: script.value.model,
    topic: fromTemplate?.topic || '',
    rounds: script.value.rounds,
    roles: fromTemplate ? fromTemplate.roles.map((r, i) => newRole(r.name, r.persona, ROLE_COLORS[i % ROLE_COLORS.length], r.emoji || '')) : defaultRoles(),
  })
  groups.value.push(g)
  currentGroupId.value = g.id
  loadViewFromGroup(g)
  persist(true)
  ElMessage.success(`已新建「${g.name}」`)
}

function switchGroup(id) {
  if (id === currentGroupId.value) return
  if (showState.value === 'running') { ElMessage.warning('先停止演出再切换场景'); return }
  persist(true)
  const g = groups.value.find(x => x.id === id)
  if (!g) return
  currentGroupId.value = id
  loadViewFromGroup(g)
  persist(true)
}

async function renameGroup() {
  const g = currentGroup.value
  if (!g) return
  try {
    const { value } = await ElMessageBox.prompt('给这个场景组起个名字', '重命名', {
      inputValue: g.name, confirmButtonText: '确定', cancelButtonText: '取消',
      inputValidator: v => (v && v.trim() ? true : '名字不能为空'),
    })
    g.name = value.trim().slice(0, 40)
    persist(true)
  } catch { /* 用户取消 */ }
}

async function deleteGroup() {
  const g = currentGroup.value
  if (!g) return
  if (groups.value.length === 1) { ElMessage.warning('至少保留一个场景组，可用「清空」重来'); return }
  try {
    await ElMessageBox.confirm(`删除「${g.name}」及其对话记录？此操作不可恢复。`, '删除场景组', {
      type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消',
    })
  } catch { return }
  if (showState.value === 'running') { ElMessage.warning('先停止演出'); return }
  const idx = groups.value.findIndex(x => x.id === g.id)
  groups.value.splice(idx, 1)
  const next = groups.value[Math.max(0, idx - 1)]
  currentGroupId.value = next.id
  loadViewFromGroup(next)
  persist(true)
  ElMessage.success('已删除')
}

// 一键清空当前组的全部角色（不必逐个删除）
async function clearRoles() {
  if (showState.value === 'running') { ElMessage.warning('先停止演出再改剧本'); return }
  try {
    await ElMessageBox.confirm('清空当前组的全部角色？对话记录会保留。', '清空角色', {
      type: 'warning', confirmButtonText: '清空', cancelButtonText: '取消',
    })
  } catch { return }
  script.value.roles = []
  persist(true)
  libraryOpen.value = true   // 直接打开角色库，方便重新挑人
  ElMessage.success('已清空角色，可从角色库重新选择')
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
  if (modelsLoading.value) return
  modelsLoading.value = true
  modelsError.value = ''
  try {
    const res = await fetch(`${API_BASE}/api/chat-plus/models`, { headers: auth.authHeader() })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    modelInfo.value = data
    const all = modelGroups.value.flatMap(g => g.items)
    if (!script.value.model || !all.some(m => m.value === script.value.model)) {
      script.value.model = all[0]?.value || ''
    }
    if (!all.length) {
      const ollErr = data?.ollama?.error || ''
      modelsError.value = ollErr
        ? `未获取到可用模型：Ollama 拉取失败（${ollErr}），且服务端未配置 MiniMax`
        : '未获取到任何可用模型（检查服务端 Ollama 是否在线 / MiniMax 是否已配置）'
    }
  } catch (e) {
    modelsError.value = `模型列表加载失败：${netErrMsg(e)}`
  } finally {
    modelsLoading.value = false
  }
}

// ══════════════ 校验与模板 ══════════════
const activeRoles = computed(() => script.value.roles.filter(r => r.enabled && r.name.trim()))
const canStart = computed(() => activeRoles.value.length >= 2 && !!script.value.model)
const startBlockReason = computed(() => {
  if (!script.value.model) return '请先选择模型'
  if (activeRoles.value.length < 2) return '至少需要 2 个启用且有名字的角色'
  return ''
})
const progressPct = computed(() => totalRounds.value ? Math.round((completedRounds.value / totalRounds.value) * 100) : 0)
const totalRounds = computed(() => Math.min(Math.max(script.value.rounds || 10, 1), MAX_ROUNDS))
const currentRoleName = computed(() => {
  const r = transcript.value[transcript.value.length - 1]
  return r && r.streaming ? r.roleName : (script.value.roles.find(x => x.id === currentRoleId.value)?.name || '')
})

function applyTemplate(t) {
  if (transcript.value.length) { createGroup(t); return }   // 已有记录 → 开新场景组，别把旧对话串进来
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
async function callModel(system, user, maxTokens = 1000) {
  if (!script.value.model) throw new Error('请先选择模型')
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), 150000) // 推理模型先想后说，给足时间，别中途截断
  try {
    const res = await fetch(`${API_BASE}/api/chat-plus/chat/stream`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...auth.authHeader() },
      body: JSON.stringify({
        model: script.value.model,
        messages: [{ role: 'system', content: system }, { role: 'user', content: user }],
        temperature: 0.75,
        max_tokens: maxTokens,
        stream: true,
        // 推理模型（如 nex-n2.5-mini）会先吐 reasoning 再吐正文，max_tokens 要留足，否则正文被挤掉、看起来"不完整"
        extra_body: { frequency_penalty: 0.2, presence_penalty: 0.1 },
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
    const system = '你是生活化的场景设计师，擅长把多角色对话放进一个真实可信的日常处境里。'
    const user = prev.trim()
      ? `请润色下面这段多角色对话的场景设定，让它像真实生活里会自然发生的一幕：说清大概的时间地点、这些人为什么会凑在一起、彼此大概是什么关系或什么由头聊起来。自然、可信、接地气，不要硬编冲突或戏剧性。\n要求：2-4 句、不超过 200 字、只输出润色后的文本（不要解释、不要引号、不要分点）。\n\n角色阵容：${rolesBrief()}\n\n原设定：${prev}`
      : `请根据下面的角色阵容，起草一段多角色对话的场景设定：一个具体、日常、可信的处境（谁、在哪、因为什么凑在一起聊）。不要刻意制造冲突或压力，就像真实生活。\n要求：2-4 句、不超过 200 字、只输出文本（不要解释、不要引号）。\n\n角色阵容：${rolesBrief()}`
    const out = await callModel(system, user, 1200)
    if (!out) throw new Error('模型没有返回内容')
    undo.value = { ...undo.value, topic: prev }
    script.value.topic = out.slice(0, 600)
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
    const system = '你是生活化的人设设计师，擅长为角色写出像身边真实的人一样的性格与说话方式。'
    const user = prev.trim()
      ? `请润色下面这个角色的人设，让它在真人对话里"像个活人"：大概什么身份/背景、平时怎么说话（口头禅、语速、爱用什么词）、有什么小习惯或小固执、遇到什么会吐槽或较真。自然可信，不需要戏剧冲突。\n要求：1-2 句、不超过 130 字、只输出润色后的人设文本（不要解释、不要引号、不要"人设："前缀）。\n\n角色名：${role.name}\n其他角色：${others}\n当前话题：${script.value.topic || '（未设定）'}\n现有人设：${prev}`
      : `请为下面这个角色设计一段人设，要像真人：大概什么身份/背景、说话习惯具体（口头禅、语速、爱用什么词、什么时候会急或吐槽）、一点小固执。与其他角色有差异即可，不必刻意制造对立。\n要求：1-2 句、不超过 130 字、只输出人设文本（不要解释、不要引号、不要"人设："前缀）。\n\n角色名：${role.name}\n其他角色：${others}\n当前话题：${script.value.topic || '（未设定）'}`
    const out = await callModel(system, user, 800)
    if (!out) throw new Error('模型没有返回内容')
    undo.value = { ...undo.value, roles: { ...undo.value.roles, [role.id]: prev } }
    role.persona = out.slice(0, 300)
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

// 真人闲聊：没有"剧情/阶段/冲突升级"的强行设定，顺着聊、会跑题、会有废话和小情绪
function buildSystemPrompt(role) {
  const others = activeRoles.value.filter(x => x.id !== role.id).map(x => x.name).join('、')
  const hist = transcript.value.filter(m => !m.streaming)
  const lastMsg = hist[hist.length - 1]
  const lastLine = lastMsg
    ? `上一条是「${lastMsg.roleName}」刚说的。自然接住就行——顺着聊、反问、吐槽、补一句、甚至突然岔开都行，不用硬接。`
    : '你是第一个开口的，随便起个话头，别端着。'
  return [
    `你就是「${role.name}」，正和 ${others} 聊天。${role.persona || '性格自然点，有点自己的小脾气就好'}`,
    script.value.topic ? `【聊天的由头】${script.value.topic}（只是开场话题，聊歪了、跑题了都正常）` : '',
    lastLine,
    '说人话，就像真人面对面闲聊那样：',
    '1. 只说你嘴里吐出来的话。别写动作、表情、心理、旁白，别用括号、星号、舞台提示。',
    '2. 口语化：短句、语气词（嗯、诶、哈、行吧、哎哟）、可以卡壳、重复、改口、被逗笑接不上话。',
    '3. 顺着聊：同意就附和，不同意就杠，好奇就追问，觉得好笑就笑；也可以聊着聊着突然想起别的事，很正常。',
    '4. 有人的小毛病：会跑题、会啰嗦、会纠结小事、会被带偏、也会突然怼一句；别句句逻辑严密、别永远滴水不漏。',
    '5. 别把天聊成汇报、别把话写成文章：不要分点、不要"首先/其次/总之"、不要总结陈词、不要书面长句、不要客服腔。',
    '6. 偶尔重复、绕回老话题、冷场前的废话都是真聊天的一部分；但如果发现自己开始一字不差地复读刚才的话，就换个说法或者岔开。',
    '长度：1-3 句、20-90 字，像微信语音转文字那样自然。只输出你这句话本身，别加名字前缀。',
  ].filter(Boolean).join('\n')
}

// 模型爱加"剧本味"（动作括号、星号动作、旁白、名字前缀、潜台词），统一剥掉
const BRACKET_NOISE = [
  /（[^（）]{0,40}）/g,      // 全角括号：动作/神态/心理
  /\([^()]{0,40}\)/g,       // 半角括号
  /\*[^*\n]{1,40}\*/g,       // *动作*
  /【[^】]{0,20}】/g,         // 【旁白】【沉默片刻】
  /(潜台词|内心独白|旁白|画外音)\s*[:：][^\n]*/g,
]
function cleanSpoken(t, roleName = '') {
  let s = t || ''
  if (roleName) {
    const n = roleName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    s = s.replace(new RegExp(`^\\s*[「『"]?${n}[」』"]?\\s*[:：]\\s*`), '')
  }
  for (const re of BRACKET_NOISE) s = s.replace(re, '')
  s = s.replace(/^[「『"]|[」』"]$/g, '')          // 整体引号
  return s.replace(/[ \t]{2,}/g, ' ').replace(/\n{2,}/g, '\n').trim()
}

// 历史映射：当前角色自己的旧发言 → assistant，其他人 → user（带名字前缀），保持轮流感
function buildPayloadMessages(role) {
  const msgs = [{ role: 'system', content: buildSystemPrompt(role) }]
  const history = transcript.value.filter(m => !m.streaming).slice(-CONTEXT_WINDOW)
  if (history.length === 0) {
    msgs.push({ role: 'user', content: `还没人开口，你先说话。别客套，直接切入正题。` })
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
// 统一的"开始/继续"入口：无论全新开演还是从中断处恢复，都走这里。
// 关键：每次进入都要把 pauseRequested / skipFlag / stopAll 这些中断标志复位，
// 否则从暂停恢复时会立刻又被旧标志打断，表现为"莫名其妙结束"。
function beginShow() {
  if (!canStart.value) { ElMessage.warning(startBlockReason.value); return }
  // 上一场已经演到最后一轮：再点「开演」应当从头开一场新的。
  // 否则 runLoop 的 startRound 会被夹到 totalRounds，只重演最后一轮、把内容叠一遍。
  if (completedRounds.value >= totalRounds.value) {
    completedRounds.value = 0
    nextRoleIdx.value = 0
    currentRound.value = 0
  }
  stopAll = false
  pauseRequested = false
  skipFlag = false
  interruptedMsg.value = ''
  showState.value = 'running'
  runLoop()
}

function pauseShow() { pauseRequested = true; ElMessage.info('本条发言结束后暂停') }
function resumeShow() { beginShow() }   // 与"开演"复用同一条复位逻辑
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
  try {
    // 续跑起点：已完成的轮数 +1 即"正在进行的这一轮"；该轮从 nextRoleIdx 指向的角色接着说
    let startRound = completedRounds.value + 1
    if (startRound > totalRounds.value) startRound = totalRounds.value
    outer:
    for (let r = startRound; r <= totalRounds.value; r++) {
      currentRound.value = r
      const roles = activeRoles.value
      if (roles.length < 2) { interruptedMsg.value = '启用的角色不足 2 个，演出中止'; break outer }
      // 恢复本轮时从 nextRoleIdx 继续；全新轮次从 0 开始
      const startIdx = (r === startRound) ? nextRoleIdx.value : 0
      for (let i = startIdx; i < roles.length; i++) {
        if (stopAll || pauseRequested) break outer   // 用户主动暂停/停止
        nextRoleIdx.value = i
        currentRoleId.value = roles[i].id
        const ok = await speakRole(roles[i], r)
        if (!ok) break outer                          // 网络失败等 → 进入暂停态
        if (i === roles.length - 1) {
          completedRounds.value = r
          nextRoleIdx.value = 0
        } else {
          nextRoleIdx.value = i + 1
        }
        saveTranscript()
      }
    }
  } catch (e) {
    // 任何未预期的异常都显式暴露，而不是静默停在"暂停"态
    interruptedMsg.value = `演出异常中断：${netErrMsg(e)}`
  } finally {
    if (stopAll) {
      showState.value = 'idle'
      ElMessage.success(`演出已停止，共完成 ${completedRounds.value} 轮`)
    } else if (completedRounds.value >= totalRounds.value) {
      showState.value = 'idle'   // 自然跑完
    } else {
      showState.value = 'paused'
      if (interruptedMsg.value) ElMessage.warning('演出已暂停，点「继续」可接着演')
    }
    saveTranscript()
  }
}

// 单条发言：流式请求 + 一次自动重试；失败返回 false（进入暂停态，可手动继续）
async function speakRole(role, round) {
  const msg = { id: uid(), roleId: role.id, roleName: role.name, color: role.color, emoji: role.emoji || '', content: '', round, ts: Date.now(), streaming: true }
  transcript.value.push(msg)
  scrollToBottom(true)

  // 真人不会零延迟接话：0.6-1.8s 的"思考/组织语言"停顿，期间显示 ···
  const thinkMs = 600 + Math.random() * 1200
  const wakeAt = Date.now() + thinkMs
  while (Date.now() < wakeAt) {
    if (stopAll) { transcript.value.splice(transcript.value.indexOf(msg), 1); return false }
    await new Promise(r => setTimeout(r, Math.min(120, wakeAt - Date.now())))
  }
  if (stopAll) { transcript.value.splice(transcript.value.indexOf(msg), 1); return false }

  let raw = ''
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
          temperature: 0.9,
          top_p: 0.92,
          max_tokens: MAX_REPLY_TOKENS,
          stream: true,
          // 轻度抑制复读即可，别太重——惩罚过重会让话变生硬、跳脱
          extra_body: { frequency_penalty: 0.3, presence_penalty: 0.15 },
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
        for (const rawLine of lines) {
          const line = rawLine.trim()
          if (!line.startsWith('data:')) continue
          const data = line.slice(5).trim()
          if (data === '[DONE]') continue
          let chunk
          try { chunk = JSON.parse(data) } catch { continue }
          const piece = chunk?.choices?.[0]?.delta?.content || ''
          if (piece) {
            raw += piece
            msg.content = cleanSpoken(raw, role.name)
            if (msg.content.length % 60 < piece.length) scrollToBottom()
          }
        }
      }
      msg.streaming = false
      msg.content = cleanSpoken(raw, role.name)
      // 空回复兜底：真人沉默也就是一句"……"
      if (!msg.content.trim()) msg.content = '……'
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
  loadGroups()
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

/* 场景组 */
.rp-groups { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.group-chips { display: flex; gap: 6px; overflow-x: auto; max-width: 100%; padding-bottom: 2px; scrollbar-width: thin; }
.group-chip {
  display: inline-flex; align-items: center; gap: 6px; flex-shrink: 0;
  border: 1px solid var(--border-light, #e5e7eb); background: var(--bg-card, #fff);
  color: var(--text-secondary, #555); font-size: 13px; padding: 5px 12px; border-radius: 999px; cursor: pointer;
  max-width: 180px;
}
.group-chip:hover { border-color: var(--color-primary, #6366f1); }
.group-chip.active {
  color: #fff; border-color: transparent;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
}
.g-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.g-count {
  font-size: 11px; padding: 0 6px; border-radius: 999px;
  background: color-mix(in srgb, currentColor 16%, transparent);
}
.group-ops { display: flex; gap: 6px; flex-shrink: 0; }
.group-op {
  border: 1px solid var(--border-light, #e5e7eb); background: var(--bg-card, #fff); cursor: pointer;
  color: var(--text-secondary, #555); font-size: 12.5px; padding: 5px 10px; border-radius: 8px;
}
.group-op:hover { border-color: var(--color-primary, #6366f1); color: var(--color-primary, #6366f1); }
.group-op.primary { color: var(--color-primary, #6366f1); border-color: color-mix(in srgb, var(--color-primary, #6366f1) 40%, transparent); }
.group-op.danger:hover { border-color: var(--el-color-danger, #f56c6c); color: var(--el-color-danger, #f56c6c); }

/* 进度 */
.rp-progress { display: flex; align-items: center; gap: 12px; }
.pg-bar { flex: 1; height: 6px; border-radius: 3px; background: var(--bg-hover, #eee); overflow: hidden; }
.pg-fill { height: 100%; border-radius: 3px; background: linear-gradient(90deg, #6366f1, #8b5cf6); transition: width 0.4s ease; }
.pg-text { font-size: 13px; color: var(--text-secondary, #555); white-space: nowrap; }

/* 模型加载失败提示 */
.rp-models-err {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
  padding: 8px 14px; border-radius: 12px; font-size: 13px;
  color: #d97706; background: rgba(217, 119, 6, 0.1); border: 1px dashed rgba(217, 119, 6, 0.45);
}

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

/* 「正在说话」三点动画 */
.typing-dots { display: inline-flex; gap: 4px; align-items: center; height: 1em; }
.typing-dots i {
  width: 6px; height: 6px; border-radius: 50%; background: var(--text-tertiary, #999);
  animation: tp-bounce 1.2s infinite ease-in-out;
}
.typing-dots i:nth-child(2) { animation-delay: 0.15s; }
.typing-dots i:nth-child(3) { animation-delay: 0.3s; }
@keyframes tp-bounce { 0%, 80%, 100% { transform: translateY(0); opacity: 0.5; } 40% { transform: translateY(-4px); opacity: 1; } }

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
.role-add.danger { border-style: solid; color: var(--el-color-danger, #f56c6c); border-color: color-mix(in srgb, var(--el-color-danger, #f56c6c) 35%, transparent); }
.role-add.danger:hover { border-color: var(--el-color-danger, #f56c6c); background: color-mix(in srgb, var(--el-color-danger, #f56c6c) 6%, transparent); }
.roles-empty { margin: 0 0 8px; font-size: 13px; color: var(--text-tertiary, #999); }

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
  .rp-groups { gap: 6px; }
  .group-ops { width: 100%; }
  .group-op { flex: 1; text-align: center; }
  .group-chip { max-width: 140px; }
  .rp-hint { display: none; }
}
</style>
