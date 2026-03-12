<template>
  <el-dialog v-model="visible" title="物品二维码" width="300px" @open="generateQR">
    <div v-if="item" style="text-align: center;">
      <div ref="qrContainer" class="qr-container"></div>
      <p style="margin-top: 12px;">{{ item.name }}</p>
      <p style="color: #909399; font-size: 12px;">{{ item.category }} - {{ item.quantity }}{{ item.unit }}</p>
      <el-button type="primary" size="small" style="margin-top: 12px;" @click="downloadQR">
        <el-icon><Download /></el-icon>
        下载二维码
      </el-button>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Download } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: Boolean,
  item: { type: Object, default: null }
})
const emit = defineEmits(['update:modelValue'])

const visible = ref(props.modelValue)
const qrContainer = ref(null)

watch(() => props.modelValue, v => { visible.value = v })
watch(visible, v => { emit('update:modelValue', v) })

function generateQR() {
  if (!props.item) return
  setTimeout(() => {
    if (!qrContainer.value) return
    qrContainer.value.innerHTML = ''
    const qrData = JSON.stringify({
      type: 'household_item',
      id: props.item.id,
      name: props.item.name,
      category: props.item.category
    })
    import('qrcode').then(({ default: QRCode }) => {
      QRCode.toCanvas(qrData, { width: 200, margin: 2 }, (err, canvas) => {
        if (err) { console.error('生成二维码失败:', err); return }
        qrContainer.value.appendChild(canvas)
      })
    })
  }, 300)
}

function downloadQR() {
  if (!qrContainer.value || !props.item) return
  const canvas = qrContainer.value.querySelector('canvas')
  if (!canvas) return
  const link = document.createElement('a')
  link.download = `物品_${props.item.name}_二维码.png`
  link.href = canvas.toDataURL('image/png')
  link.click()
  ElMessage.success('二维码已下载')
}
</script>

<style scoped>
.qr-container { display: flex; justify-content: center; align-items: center; }
</style>
