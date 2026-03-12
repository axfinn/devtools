<template>
  <el-dialog v-model="visible" title="扫码添加物品" width="500px" @close="stopScan">
    <div class="scan-container">
      <div id="qr-reader" class="qr-reader"></div>
      <div v-if="scanResult" class="scan-result">
        <el-alert :title="'识别成功: ' + scanResult" type="success" :closable="false" />
        <div class="scan-actions" style="margin-top: 12px;">
          <el-button type="primary" @click="handleScanResult">添加到物品库</el-button>
          <el-button @click="scanResult = ''">继续扫描</el-button>
        </div>
      </div>
      <div style="margin-top: 16px; text-align: center; color: #909399;">
        <p>支持扫描商品条形码和二维码</p>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Html5Qrcode } from 'html5-qrcode'

const props = defineProps({
  modelValue: Boolean,
  profileId: String,
  creatorKey: String,
  aiEnabled: Boolean
})
const emit = defineEmits(['update:modelValue', 'scan-done'])

const visible = ref(props.modelValue)
const scanResult = ref('')
const isScanning = ref(false)
let html5QrCode = null

watch(() => props.modelValue, v => {
  visible.value = v
  if (v) {
    setTimeout(() => startScan(), 300)
  } else {
    stopScan()
  }
})
watch(visible, v => { emit('update:modelValue', v) })

async function startScan() {
  if (isScanning.value) return
  try {
    html5QrCode = new Html5Qrcode('qr-reader')
    isScanning.value = true
    await html5QrCode.start(
      { facingMode: 'environment' },
      { fps: 10, qrbox: { width: 250, height: 250 } },
      (decodedText) => {
        scanResult.value = decodedText
        stopScan()
      },
      () => {}
    )
  } catch (err) {
    console.error('启动扫码失败:', err)
    ElMessage.error('无法启动摄像头，请确保已授权')
    isScanning.value = false
  }
}

function stopScan() {
  if (html5QrCode && isScanning.value) {
    html5QrCode.stop().then(() => {
      html5QrCode = null
      isScanning.value = false
    }).catch(() => {
      isScanning.value = false
    })
  }
}

async function handleScanResult() {
  if (!scanResult.value) return
  if (props.aiEnabled) {
    try {
      const res = await fetch('/api/household/ai/add', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          text: '添加商品：' + scanResult.value,
          profile_id: props.profileId,
          creator_key: props.creatorKey
        })
      })
      const data = await res.json()
      if (data.code === 0) {
        ElMessage.success(`成功添加 ${data.count} 个物品`)
        visible.value = false
        scanResult.value = ''
        emit('scan-done', { refresh: true })
        return
      }
    } catch {}
  }
  // fallback: 条码查询
  await addScannedItem(scanResult.value)
}

async function addScannedItem(code) {
  try {
    const res = await fetch('/api/household/barcode/lookup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ barcode: code })
    })
    const data = await res.json()
    const prefill = data.code === 0 && data.name
      ? { name: data.name, category: data.category || '其他', unit: data.unit || '个', notes: '条码: ' + code }
      : { notes: '条码: ' + code }
    if (!prefill.name) ElMessage.info('未找到该条码对应的商品，请手动输入名称')
    visible.value = false
    scanResult.value = ''
    emit('scan-done', { prefill })
  } catch {
    visible.value = false
    scanResult.value = ''
    emit('scan-done', { prefill: { notes: '条码: ' + code } })
    ElMessage.info('请手动输入商品名称')
  }
}
</script>

<style scoped>
.scan-container { min-height: 300px; }
.qr-reader { width: 100%; border-radius: 8px; overflow: hidden; }
.scan-result { margin-top: 16px; }
.scan-actions { display: flex; justify-content: center; gap: 12px; }
</style>
