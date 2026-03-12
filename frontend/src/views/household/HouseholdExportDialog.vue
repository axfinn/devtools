<template>
  <el-dialog v-model="visible" title="导出物品数据" width="500px">
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
        <el-button @click="copyData">
          <el-icon><CopyDocument /></el-icon>
          复制
        </el-button>
        <el-button type="primary" @click="downloadData">
          <el-icon><Download /></el-icon>
          下载文件
        </el-button>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { CopyDocument, Download } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: Boolean,
  items: { type: Array, default: () => [] },
  categories: { type: Array, default: () => [] }
})
const emit = defineEmits(['update:modelValue'])

const visible = ref(props.modelValue)
const exportScope = ref('all')
const exportCategory = ref('')
const exportFormat = ref('text')

watch(() => props.modelValue, v => { visible.value = v })
watch(visible, v => { emit('update:modelValue', v) })

const exportPreview = computed(() => {
  let data = props.items
  if (exportScope.value === 'category' && exportCategory.value) {
    data = data.filter(item => item.category === exportCategory.value)
  }
  if (exportFormat.value === 'json') {
    return JSON.stringify(data, null, 2)
  }
  let text = '物品清单\n' + '='.repeat(30) + '\n\n'
  const byCategory = {}
  data.forEach(item => {
    if (!byCategory[item.category]) byCategory[item.category] = []
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

function copyData() {
  navigator.clipboard.writeText(exportPreview.value)
    .then(() => ElMessage.success('已复制到剪贴板'))
    .catch(() => ElMessage.error('复制失败'))
}

function downloadData() {
  const isJson = exportFormat.value === 'json'
  const blob = new Blob([exportPreview.value], { type: isJson ? 'application/json' : 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = isJson ? '物品清单.json' : '物品清单.txt'
  link.click()
  URL.revokeObjectURL(url)
  ElMessage.success('文件已下载')
}
</script>

<style scoped>
.export-container { min-height: 300px; }
.export-preview { margin-top: 16px; }
.preview-header { font-weight: bold; margin-bottom: 8px; }
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
.export-actions { margin-top: 16px; display: flex; justify-content: center; gap: 12px; }
</style>
