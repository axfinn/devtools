<!--
  AssetsPanel.vue —— 资产面板(P2)
  两个 tab:
    - 我的模型(从 /api/avatar/me/assets + /api/avatar/models 拼)
    - 系统库(从 /api/avatar/models owner_id='' 的列表)
  每个 model 是一张缩略图卡(渐变色 + 标题 + 大小);
  点击 → emit select,父组件负责 setSkeletonVisible 之类;
  同时调 /me/assets 把模型登记进我的资产(P2 me_asset UI 落点)。
-->
<template>
  <div class="assets-panel">
    <el-tabs v-model="activeTab" class="assets-tabs">
      <el-tab-pane label="我的模型" name="mine">
        <div class="upload-row">
          <el-button size="small" type="primary" plain @click="onUploadClick">
            <el-icon><Upload /></el-icon> 上传
          </el-button>
          <input
            ref="fileInputRef"
            type="file"
            accept=".glb,.gltf"
            style="display:none"
            @change="onFileSelected"
          />
        </div>
        <div v-if="loading" class="loading">加载中…</div>
        <div v-else-if="!myItems.length" class="empty">还没有模型 — 点上方上传</div>
        <div v-else class="thumb-grid">
          <div
            v-for="m in myItems"
            :key="m.id"
            class="thumb"
            :class="{ active: activeId === m.id }"
            :style="thumbStyle(m)"
            @click="onSelect(m)"
          >
            <div class="thumb-meta">
              <strong>{{ m.title || m.original || m.id }}</strong>
              <small>{{ fmtSize(m.size) }} · {{ m.format }}</small>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="系统库" name="lib">
        <div v-if="loading" class="loading">加载中…</div>
        <div v-else-if="!libItems.length" class="empty">库为空</div>
        <div v-else class="thumb-grid">
          <div
            v-for="m in libItems"
            :key="m.id"
            class="thumb"
            :class="{ active: activeId === m.id }"
            :style="thumbStyle(m)"
            @click="onSelect(m)"
          >
            <div class="thumb-meta">
              <strong>{{ m.title || m.original || m.id }}</strong>
              <small>{{ fmtSize(m.size) }} · {{ m.format }}</small>
            </div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Upload } from '@element-plus/icons-vue'
import { useAssets, registerMyAsset } from './composable/useAssets.js'

const props = defineProps({
  activeId: { type: String, default: '' },
})
const emit = defineEmits(['select', 'register-me'])

const assets = useAssets()
const activeTab = ref('mine')
const fileInputRef = ref(null)
const loading = ref(false)

const myItems = computed(() => assets.myAssets.value || [])
const libItems = computed(() => {
  // 全库浏览 = assets.models(后端默认 owner_id='',所以等价于"全部")
  return assets.models.value || []
})

onMounted(async () => {
  loading.value = true
  try {
    await Promise.all([
      assets.refreshMyAssets(),
      assets.refreshModels({ limit: 50 }),
    ])
  } finally {
    loading.value = false
  }
})

function thumbStyle(m) {
  // hash title → hue
  const t = (m.title || m.original || m.id || 'x')
  let h = 0
  for (let i = 0; i < t.length; i++) h = (h * 31 + t.charCodeAt(i)) & 0xffff
  const hue = h % 360
  return { background: `linear-gradient(135deg, hsl(${hue} 55% 45%), hsl(${(hue + 60) % 360} 65% 30%))` }
}

function fmtSize(b) {
  if (!b) return '—'
  if (b < 1024) return `${b} B`
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`
  return `${(b / 1024 / 1024).toFixed(1)} MB`
}

function onSelect(m) {
  emit('select', m)
  // P2 me_asset UI:点库里的模型 → 写回 me_asset
  registerMyAsset({ modelId: m.id, title: m.title || '', note: 'from library' })
    .then(() => {
      emit('register-me', m)
      ElMessage.success(`已加入"我的资产": ${m.title || m.id}`)
      assets.refreshMyAssets()
    })
    .catch((e) => {
      // 匿名无 creator_key 会 401,这里静默处理,提示用户填 key
      if (e.status === 401) {
        ElMessage.warning('加入我的资产需要先设置 creator_key(开发模式下默认匿名)')
      } else {
        ElMessage.warning('加入我的资产失败:' + (e.message || '未知错误'))
      }
    })
}

function onUploadClick() {
  fileInputRef.value?.click()
}

async function onFileSelected(e) {
  const file = e.target.files?.[0]
  if (!file) return
  // 上传后自动 register
  try {
    const r = await assets.uploadModelFile(file, { title: file.name.replace(/\.[^.]+$/, '') })
    ElMessage.success(`上传成功: ${r.id}`)
    // 自动 register 到我的资产
    try {
      await registerMyAsset({ modelId: r.id, title: r.title || file.name, note: 'uploaded' })
      emit('register-me', r)
      await assets.refreshMyAssets()
    } catch (e2) {
      // 401 不致命
    }
    activeTab.value = 'mine'
  } catch (err) {
    ElMessage.error('上传失败:' + (err.message || err))
  } finally {
    e.target.value = ''
  }
}
</script>

<style scoped>
.assets-panel {
  padding: 4px;
}
.assets-tabs :deep(.el-tabs__header) {
  margin-bottom: 8px;
}
.assets-tabs :deep(.el-tabs__nav-wrap::after) {
  background: var(--border-light, #333);
}
.upload-row {
  margin-bottom: 8px;
}
.loading, .empty {
  font-size: 12px;
  color: var(--text-tertiary, #909399);
  text-align: center;
  padding: 24px 0;
}
.thumb-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 6px;
  max-height: 320px;
  overflow-y: auto;
}
.thumb {
  aspect-ratio: 4 / 3;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: flex-end;
  padding: 6px;
  color: #fff;
  transition: transform 100ms, outline 100ms;
  position: relative;
}
.thumb:hover {
  transform: translateY(-1px);
}
.thumb.active {
  outline: 2px solid var(--color-primary, #409eff);
  outline-offset: 1px;
}
.thumb-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  text-shadow: 0 1px 2px rgba(0,0,0,0.5);
  font-size: 11px;
  width: 100%;
}
.thumb-meta strong {
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.thumb-meta small {
  opacity: 0.85;
  font-family: 'JetBrains Mono', monospace;
}
</style>
