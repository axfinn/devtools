<template>
  <el-dialog v-model="visible" title="小票 OCR 识别" width="600px" @close="clearReceipt">
    <div class="receipt-container">
      <el-upload
        v-if="!receiptImage"
        class="receipt-upload"
        :auto-upload="false"
        :show-file-list="false"
        accept="image/*"
        @change="handleFileChange"
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
          <el-button type="primary" :loading="loading" @click="recognizeReceipt">
            <el-icon><Tickets /></el-icon>
            开始识别
          </el-button>
        </div>
      </div>

      <div v-if="receiptItems.length > 0" class="receipt-result">
        <el-divider>识别结果</el-divider>
        <div class="matched-items">
          <el-tag
            v-for="(item, idx) in receiptItems"
            :key="idx"
            :type="item.matched ? 'success' : 'info'"
            class="receipt-item-tag"
            closable
            @close="receiptItems.splice(idx, 1)"
          >
            {{ item.name }} <span v-if="item.quantity">x{{ item.quantity }}</span>
            <span v-if="item.matched" class="matched-badge">已匹配</span>
          </el-tag>
        </div>
        <div class="receipt-result-actions">
          <el-button type="primary" @click="addAllToLibrary">全部添加到物品库</el-button>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Picture, Tickets } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: Boolean,
  profileId: String,
  creatorKey: String
})
const emit = defineEmits(['update:modelValue', 'added'])

const visible = ref(props.modelValue)
const receiptImage = ref('')
const loading = ref(false)
const receiptItems = ref([])

watch(() => props.modelValue, v => { visible.value = v })
watch(visible, v => { emit('update:modelValue', v) })

function clearReceipt() {
  receiptImage.value = ''
  receiptItems.value = []
}

function handleFileChange(file) {
  const reader = new FileReader()
  reader.onload = (e) => { receiptImage.value = e.target.result }
  reader.readAsDataURL(file.raw)
}

async function recognizeReceipt() {
  if (!receiptImage.value) return
  loading.value = true
  try {
    const res = await fetch('/api/household/ocr/receipt', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        image: receiptImage.value,
        profile_id: props.profileId,
        creator_key: props.creatorKey
      })
    })
    const data = await res.json()
    if (data.code === 0) {
      receiptItems.value = data.items || []
      if (receiptItems.value.length === 0) {
        ElMessage.warning('未识别到商品，请尝试重新拍照')
      } else {
        ElMessage.success(`识别到 ${receiptItems.value.length} 个商品`)
      }
    } else {
      ElMessage.error(data.error || '识别失败')
    }
  } catch {
    ElMessage.error('识别失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

async function addAllToLibrary() {
  if (receiptItems.value.length === 0) return
  let count = 0
  for (const item of receiptItems.value) {
    try {
      const res = await fetch(`/api/household/profile/${props.profileId}/items?creator_key=${props.creatorKey}`, {
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
      if (data.code === 0) count++
    } catch {}
  }
  if (count > 0) {
    ElMessage.success(`成功添加 ${count} 个物品`)
    visible.value = false
    clearReceipt()
    emit('added')
  } else {
    ElMessage.error('添加失败')
  }
}
</script>

<style scoped>
.receipt-container { min-height: 300px; }
.receipt-upload {
  width: 100%;
  min-height: 300px;
  border: 2px dashed #dcdfe6;
  border-radius: 8px;
  cursor: pointer;
  transition: border-color 0.3s;
}
.receipt-upload:hover { border-color: #409eff; }
.upload-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  color: #909399;
}
.upload-placeholder p { margin: 10px 0; }
.upload-placeholder .hint { font-size: 12px; }
.receipt-preview { text-align: center; }
.receipt-actions { margin-top: 16px; display: flex; justify-content: center; gap: 12px; }
.receipt-result { margin-top: 20px; }
.matched-items { display: flex; flex-wrap: wrap; gap: 8px; padding: 12px; background: #f5f7fa; border-radius: 4px; }
.receipt-item-tag { padding: 8px 12px; }
.matched-badge { font-size: 10px; color: #67c23a; margin-left: 4px; }
.receipt-result-actions { margin-top: 16px; text-align: center; }
</style>
