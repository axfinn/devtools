<template>
  <div class="pet-tool">
    <section class="hero">
      <div class="hero-copy">
        <p class="eyebrow">MAGIC PET · 浏览器版</p>
        <h2>魔法宠物助手</h2>
        <p class="hero-desc">
          嵌入式的 3D 小生灵，配上 minimax T2A 语音播报。
          点击 / 双击 / 长按宠物体验不同动作，切主题、配置 TTS 凭证、听它说话。
        </p>
        <ul class="tips">
          <li>🔊 已从 devtools 全局凭证 <code>AI_KEY_STORAGE</code> 自动读 TTS token</li>
          <li>🐾 同一句话只生成一次，结果缓存在浏览器 Cache Storage</li>
          <li>🎨 切主题会触发对应声线播放主题欢迎语</li>
        </ul>
      </div>

      <el-card class="stage" shadow="never">
        <div class="stage-wrap">
          <pet-widget
            ref="widget"
            :theme="theme"
            :width="stageWidth"
            :height="stageHeight"
            voice
          />
        </div>
        <div class="stage-controls">
          <el-radio-group v-model="theme" size="small">
            <el-radio-button v-for="t in THEMES" :key="t.id" :value="t.id">
              {{ t.name }}
            </el-radio-button>
          </el-radio-group>
        </div>
      </el-card>
    </section>

    <section class="config">
      <el-card shadow="never">
        <template #header>
          <div class="row-between">
            <span>TTS 凭证</span>
            <el-tag v-if="tokenSet" type="success" size="small">已配置</el-tag>
            <el-tag v-else type="warning" size="small">未配置</el-tag>
          </div>
        </template>
        <p class="hint">
          浏览器调用 minimax T2A 需要 Bearer Token。优先级：① 下方输入框 → ② devtools 全局
          <code>AI_KEY_STORAGE</code> → ③ <code>ai_gateway_super_admin_password</code>
        </p>
        <el-input
          v-model="tokenDraft"
          type="password"
          show-password
          placeholder="可选：覆盖 devtools 全局凭证"
          @keyup.enter="saveToken"
        >
          <template #append>
            <el-button @click="saveToken">保存到 widget</el-button>
          </template>
        </el-input>
        <div class="row-between" style="margin-top: 12px;">
          <el-button @click="testSpeak" type="primary" :disabled="!tokenSet">
            ▶ 测试说话
          </el-button>
          <el-button @click="clearToken" v-if="tokenSet">清除 widget 凭证</el-button>
        </div>
        <p v-if="lastError" class="err">⚠ {{ lastError }}</p>
      </el-card>

      <el-card shadow="never">
        <template #header>
          <span>操作速查</span>
        </template>
        <el-table :data="ACTIONS" size="small" border>
          <el-table-column prop="name" label="动作" width="100" />
          <el-table-column label="触发" width="120">
            <template #default="{ row }">
              <el-tag
                size="small"
                :type="row.trigger === 'tap' ? 'primary' : row.trigger === 'double' ? 'success' : 'danger'"
              >
                {{ row.trigger }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="flavor" label="效果" />
          <el-table-column label="语音" width="200">
            <template #default="{ row }">
              <code v-if="row.voiceText" style="font-size: 11px;">"{{ row.voiceText }}"</code>
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
&lt;pet-widget theme="prism" width="280" height="320" voice
            tts-token="sk-..."&gt;&lt;/pet-widget&gt;</code></pre>
      </el-card>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'

// 内联主题 + 动作常量（避免跨项目依赖；保持与 pet-widget 同源）
const THEMES = [
  { id: 'flame',   name: '烈焰' },
  { id: 'tide',    name: '深海' },
  { id: 'forest',  name: '森语' },
  { id: 'cosmic',  name: '星海' },
  { id: 'prism',   name: '棱镜' },
]

const ACTIONS = [
  { name: '待机',     trigger: 'tap',   flavor: '悬浮、眨眼、微微摆头',  voiceText: '' },
  { name: '打招呼',   trigger: 'tap',   flavor: '向你点头致意',          voiceText: '嗨～' },
  { name: '跳跃',     trigger: 'tap',   flavor: '抛物线 + 落地 squash',  voiceText: '呀！' },
  { name: '旋转',     trigger: 'double',flavor: '360° 转一圈',           voiceText: '' },
  { name: '舞蹈',     trigger: 'double',flavor: '按主题节奏扭动 2.4s',   voiceText: '跳起来啦！' },
  { name: '魔法攻击', trigger: 'hold',  flavor: '蓄力 + 释放 + 粒子',    voiceText: '元素爆发！' },
  { name: '打盹',     trigger: 'tap',   flavor: '眼睛闭起，下沉',        voiceText: '嗯…好困…' },
]

const theme = ref('prism')
const tokenDraft = ref('')
const tokenSet = ref(false)
const lastError = ref('')
const widget = ref(null)

const STAGE_W = 320
const STAGE_H = 360
const stageWidth = computed(() => Math.min(STAGE_W, (typeof window !== 'undefined' ? window.innerWidth : 1080) - 64))
const stageHeight = computed(() => STAGE_H)

function checkToken() {
  if (typeof localStorage === 'undefined') return
  tokenSet.value = !!(
    localStorage.getItem('pet-widget.ttsToken') ||
    localStorage.getItem('AI_KEY_STORAGE') ||
    localStorage.getItem('ai_gateway_super_admin_password')
  )
}

function broadcastToken(token) {
  document.querySelectorAll('pet-widget').forEach((el) => {
    el.setTtsToken?.(token)
  })
}

function saveToken() {
  const v = tokenDraft.value.trim()
  if (v) {
    localStorage.setItem('pet-widget.ttsToken', v)
    broadcastToken(v)
  } else {
    localStorage.removeItem('pet-widget.ttsToken')
    broadcastToken(null)
  }
  tokenDraft.value = ''
  checkToken()
}

function clearToken() {
  localStorage.removeItem('pet-widget.ttsToken')
  broadcastToken(null)
  checkToken()
}

function testSpeak() {
  document.querySelectorAll('pet-widget').forEach((el) => {
    el.speak?.('你好，魔法宠物已上线！')
  })
}

function copySnippet() {
  const text = `<script src="/widgets/pet-widget.js"><\/script>
<pet-widget theme="prism" width="280" height="320" voice
            tts-token="sk-..."></pet-widget>`
  navigator.clipboard?.writeText(text)
}

function onWidgetError(e) {
  lastError.value = e.detail?.message || '未知错误'
}

onMounted(() => {
  checkToken()
  const saved = localStorage.getItem('pet-widget.ttsToken')
  if (saved) broadcastToken(saved)
  window.addEventListener('pet:error', onWidgetError)
})

onUnmounted(() => {
  window.removeEventListener('pet:error', onWidgetError)
})

watch(theme, (val) => {
  document.querySelectorAll('pet-widget').forEach((el) => {
    el.setTheme?.(val)
  })
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
.stage-controls {
  width: 100%;
  display: flex;
  justify-content: center;
  padding: 8px 0 0;
}
.config {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 16px;
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
.row-between {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}
.err {
  color: #ff6e8a;
  font-size: 12px;
  margin: 8px 0 0;
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
