<template>
  <div class="pet-tool">
    <section class="hero">
      <div class="hero-copy">
        <p class="eyebrow">MAGIC PET · 浏览器版</p>
        <h2>魔法宠物助手</h2>
        <p class="hero-desc">
          嵌入式的 3D 小生灵，配上预录的 12 段主题语音。
          点击 / 双击 / 长按宠物体验不同动作，切主题自动播放对应欢迎语。
        </p>
        <ul class="tips">
          <li>🎵 所有语音都是预生成的 MP3，零 API 调用、零延迟</li>
          <li>🐾 包含 5 主题 + 5 动作 + 通用打气 + 首次启动问候</li>
          <li>🎨 切主题会播放对应声线的欢迎语</li>
          <li>💪 点 💪 按钮或等 12 分钟获得一次自动打气</li>
        </ul>
      </div>

      <el-card class="stage" shadow="never">
        <div class="stage-wrap">
          <pet-widget
            ref="widget"
            :theme="theme"
            :form="form"
            :width="stageWidth"
            :height="stageHeight"
            voice
          />
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
              <el-radio-button v-for="f in FORMS" :key="f.id" :value="f.id">
                {{ f.icon }} {{ f.label }}
              </el-radio-button>
            </el-radio-group>
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
              <el-tag
                size="small"
                :type="row.trigger === 'tap' ? 'primary' : row.trigger === 'double' ? 'success' : 'danger'"
              >
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

// 内联主题常量（与 pet-widget 包保持一致）
const THEMES = [
  { id: 'flame',   name: '烈焰' },
  { id: 'tide',    name: '深海' },
  { id: 'forest',  name: '森语' },
  { id: 'cosmic',  name: '星海' },
  { id: 'prism',   name: '棱镜' },
]

const ACTIONS = [
  { name: '待机',     trigger: 'tap',   flavor: '悬浮、眨眼、微微摆头',     voiceKey: '' },
  { name: '打招呼',   trigger: 'tap',   flavor: '向你点头致意',              voiceKey: 'act_wave' },
  { name: '跳跃',     trigger: 'tap',   flavor: '抛物线 + 落地 squash',      voiceKey: 'act_jump' },
  { name: '旋转',     trigger: 'double',flavor: '360° 转一圈',                voiceKey: '' },
  { name: '舞蹈',     trigger: 'double',flavor: '按主题节奏扭动 2.4s',       voiceKey: 'act_dance' },
  { name: '魔法攻击', trigger: 'hold',  flavor: '蓄力 + 释放 + 粒子',         voiceKey: 'act_attack' },
  { name: '打盹',     trigger: 'tap',   flavor: '眼睛闭起，下沉',             voiceKey: 'act_sleep' },
]

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

onMounted(() => {})
onUnmounted(() => {})
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
