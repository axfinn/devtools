<template>
  <div class="pet-tool">
    <section class="hero">
      <div class="hero-copy">
        <p class="eyebrow">MAGIC PET · 浏览器版</p>
        <h2>魔法宠物助手</h2>
        <p class="hero-desc">
          嵌入式的 3D 小生灵，配上预录的主题语音。
          点一下宠物就随机来一个友好动作，每 4–9 秒它还会自己动一下；切主题自动播放对应欢迎语。
        </p>
        <ul class="tips">
          <li>🎵 所有语音都是预生成的 MP3，零 API 调用、零延迟</li>
          <li>🐾 包含 5 主题 + 5 形态（只换配色）+ 通用打气 + 首次启动问候</li>
          <li>🎨 切主题会播放对应声线的欢迎语</li>
          <li>💪 点 💪 按钮或等 12 分钟获得一次自动打气</li>
          <li>🔇 舞台下方的 🔊/🔇 可静音，全站生效、刷新后仍保留</li>
        </ul>
      </div>

      <el-card class="stage" shadow="never">
        <div class="stage-wrap">
          <!-- FINN-29 A6 · pet-widget 异步注入：未 upgrade 时显示占位 -->
          <pet-widget
            v-if="petReady"
            ref="widget"
            :theme="theme"
            :form="form"
            :width="stageWidth"
            :height="stageHeight"
            voice
            no-greet
          />
          <div v-else class="stage-placeholder">
            <div class="stage-placeholder-emoji">🐾</div>
            <div class="stage-placeholder-text">魔法宠物加载中…</div>
          </div>
        </div>
        <div class="stage-controls">
          <div class="control-row">
            <span class="control-label">主题</span>
            <el-radio-group v-model="theme" size="small">
              <el-radio-button v-for="t in THEMES" :key="t.id" :value="t.id">
                {{ t.name }}
              </el-radio-button>
            </el-radio-group>
          </div>
          <div class="control-row">
            <span class="control-label">形态</span>
            <el-radio-group v-model="form" size="small">
              <el-radio-button
                v-for="f in FORMS"
                :key="f.id"
                :value="f.id"
                :title="'只改变配色（同一个模型，不换体型）'"
              >
                {{ f.icon }} {{ f.label }}
              </el-radio-button>
            </el-radio-group>
          </div>
          <div class="control-row">
            <span class="control-label">语音</span>
            <el-button size="small" @click="toggleVoice">
              {{ voiceOn ? '🔊 声音开启' : '🔇 已静音' }}
            </el-button>
            <span class="control-hint">静音状态全站生效，刷新后仍保留</span>
          </div>
        </div>
      </el-card>
    </section>

    <section class="config">
      <el-card shadow="never">
        <template #header>
          <span>操作速查</span>
        </template>
        <el-table :data="ACTIONS" size="small" border>
          <el-table-column prop="name" label="动作" width="100" />
          <el-table-column label="触发" width="120">
            <template #default="{ row }">
              <el-tag size="small" :type="TRIGGER_TAG_TYPE[row.trigger] || 'info'">
                {{ row.trigger }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="flavor" label="效果" />
          <el-table-column label="语音" width="160">
            <template #default="{ row }">
              <code v-if="row.voiceKey" style="font-size: 11px;">{{ row.voiceKey }}</code>
              <span v-else style="color: #999;">—</span>
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card shadow="never">
        <template #header>
          <div class="row-between">
            <span>嵌入到你自己的页面</span>
            <el-button size="small" @click="copySnippet">复制 HTML</el-button>
          </div>
        </template>
        <pre class="snippet"><code>&lt;script src="/widgets/pet-widget.js"&gt;&lt;/script&gt;
&lt;pet-widget theme="prism" width="280" height="320" voice&gt;&lt;/pet-widget&gt;</code></pre>
        <p class="hint" style="margin-top: 12px;">
          预录的 MP3 在 <code>/widgets/voices/</code> 目录下：<br />
          <code style="font-size: 11px;">theme_flame · theme_tide · theme_forest · theme_cosmic · theme_prism ·
          act_wave · act_jump · act_dance · act_attack · act_sleep · cheer · welcome</code>
        </p>
      </el-card>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
// FINN-29 A6 · pet-widget 异步注入工具
import { ensurePetWidget } from '../../utils/petWidgetLoader'

// 内联主题常量（与 pet-widget 包保持一致）
const THEMES = [
  { id: 'flame',   name: '烈焰' },
  { id: 'tide',    name: '深海' },
  { id: 'forest',  name: '森语' },
  { id: 'cosmic',  name: '星海' },
  { id: 'prism',   name: '棱镜' },
]

const ACTIONS = [
  { name: '点击',     trigger: 'tap',   flavor: '随机播放 8 个友好动作之一（挥手 / 点头 / 跳跃 …）', voiceKey: 'act_wave 等' },
  { name: '待机',     trigger: 'idle',  flavor: '每 4–9 秒自动换一个 idle 动作（可静音）',            voiceKey: 'act_* 随机' },
  { name: '首次出现', trigger: 'mount', flavor: '模型加载完成后 0.6 秒打个招呼',                       voiceKey: 'greeting_hi' },
  { name: '切换形态', trigger: 'form',  flavor: '同一个模型换配色（不换体型）',                        voiceKey: 'form_cat 等' },
  { name: '切换主题', trigger: 'theme', flavor: '换配色并播放该主题的欢迎语',                          voiceKey: 'theme_prism 等' },
  { name: '打气',     trigger: 'cheer', flavor: '随机一句鼓励语 + 一段语音',                           voiceKey: 'cheer_01 等' },
]

// 「触发」列的标签配色
const TRIGGER_TAG_TYPE = {
  tap: 'primary',
  idle: 'success',
  mount: 'info',
  form: 'info',
  theme: 'info',
  cheer: 'warning',
}

const FORMS = [
  { id: 'default', icon: '🔵', label: '默认' },
  { id: 'cat',     icon: '🐱', label: '小猫' },
  { id: 'robot',   icon: '🤖', label: '机械' },
  { id: 'ghost',   icon: '👻', label: '幽灵' },
  { id: 'star',    icon: '⭐', label: '星灵' },
]

const theme = ref('prism')
const form = ref('default')
const widget = ref(null)
// FINN-29 A6 · pet-widget 是否已升级（customElement upgrade 完成）
const petReady = ref(false)

// 全局语音总开关的持久化键与广播事件（与 pet-widget.js 内保持一致）
const VOICE_ENABLED_KEY = 'pet-widget.enabled'
const VOICE_CHANGE_EVENT = 'pet-voice-change'

// 语音开关初值读持久化的全局开关（缺失 = 开启）
function readVoiceEnabled() {
  try {
    const v = localStorage.getItem(VOICE_ENABLED_KEY)
    return v === null ? true : v === '1'
  } catch { return true } // 隐私模式 / 无 localStorage ⇒ 默认开启
}
const voiceOn = ref(readVoiceEnabled())

function toggleVoice() {
  voiceOn.value = !voiceOn.value
  // 直接写持久化键：即使页面上还没有 pet-widget 实例，静音选择也不会丢
  try { localStorage.setItem(VOICE_ENABLED_KEY, voiceOn.value ? '1' : '0') } catch { /* ignore */ }
  // 通知所有 widget（voice 属性变化会驱动 pet-widget 写回全局开关）
  document.querySelectorAll('pet-widget').forEach((el) => {
    if (voiceOn.value) el.setAttribute('voice', '')
    else el.removeAttribute('voice')
  })
}

// pet-widget 广播全局开关变化 ⇒ 同步本页按钮的图标状态
function onVoiceChange(e) {
  if (e?.detail && typeof e.detail.enabled === 'boolean') voiceOn.value = e.detail.enabled
}

const STAGE_W = 320
const STAGE_H = 360
const stageWidth = computed(() => Math.min(STAGE_W, (typeof window !== 'undefined' ? window.innerWidth : 1080) - 64))
const stageHeight = computed(() => STAGE_H)

function copySnippet() {
  const text = `<script src="/widgets/pet-widget.js"><\/script>
<pet-widget theme="prism" width="280" height="320" voice></pet-widget>`
  navigator.clipboard?.writeText(text)
}

watch(theme, (val) => {
  document.querySelectorAll('pet-widget').forEach((el) => {
    el.setTheme?.(val)
  })
})

watch(form, (val) => {
  document.querySelectorAll('pet-widget').forEach((el) => {
    el.setForm?.(val)
  })
})

onMounted(() => {
  // FINN-29 A6 · 异步注入；用户主动进入此页才拉取，不抢首屏带宽
  ensurePetWidget()
    .then(() => { petReady.value = true })
    .catch(() => { /* 保留占位 */ })
  // 订阅全局语音开关广播
  document.addEventListener(VOICE_CHANGE_EVENT, onVoiceChange)
})
onUnmounted(() => {
  document.removeEventListener(VOICE_CHANGE_EVENT, onVoiceChange)
})
</script>

<style scoped>
.pet-tool {
  max-width: 1080px;
  margin: 0 auto;
  padding: 24px;
}
.hero {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 24px;
  align-items: start;
  margin-bottom: 24px;
}
@media (max-width: 880px) {
  .hero { grid-template-columns: 1fr; }
}
.eyebrow {
  font-size: 11px;
  letter-spacing: 4px;
  color: #5cf2ff;
  margin: 0 0 4px;
}
.hero h2 {
  margin: 0 0 8px;
  font-size: 24px;
  background: linear-gradient(90deg, #5cf2ff, #b88dff);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}
.hero-desc {
  color: #a4abc4;
  font-size: 14px;
  line-height: 1.6;
  margin: 0 0 12px;
}
.tips {
  color: #a4abc4;
  font-size: 13px;
  margin: 0;
  padding-left: 20px;
}
.tips li { margin: 4px 0; }
.tips code {
  background: rgba(255,255,255,0.06);
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 11px;
}
.stage {
  display: flex;
  flex-direction: column;
  align-items: center;
  background: rgba(255,255,255,0.02);
}
.stage-wrap {
  display: flex;
  justify-content: center;
  padding: 8px 0;
}

/* FINN-29 A6 · pet-widget 未 upgrade 时的占位 */
.stage-placeholder {
  width: 320px;
  height: 360px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  border-radius: 16px;
  background: linear-gradient(135deg, rgba(92, 242, 255, 0.06), rgba(184, 141, 255, 0.06));
  color: #a4abc4;
}
.stage-placeholder-emoji {
  font-size: 48px;
  animation: stage-placeholder-bounce 1.2s ease-in-out infinite;
}
.stage-placeholder-text {
  font-size: 13px;
  opacity: 0.85;
}
@keyframes stage-placeholder-bounce {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-6px); }
}
.stage-controls {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 8px 0 0;
}
.control-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.control-label {
  font-size: 12px;
  color: #a4abc4;
  min-width: 32px;
}
.control-hint {
  font-size: 12px;
  color: #8a91a8;
}
.config {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
  gap: 16px;
}
.row-between {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}
.hint {
  font-size: 12px;
  color: #a4abc4;
  margin: 0 0 12px;
  line-height: 1.6;
}
.hint code {
  background: rgba(255,255,255,0.06);
  padding: 1px 5px;
  border-radius: 3px;
  font-size: 11px;
}
.snippet {
  background: rgba(0,0,0,0.4);
  border: 1px solid rgba(255,255,255,0.06);
  border-radius: 6px;
  padding: 12px;
  font-size: 12px;
  color: #cfd6ee;
  margin: 0;
  overflow-x: auto;
}
</style>
