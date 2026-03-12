<template>
  <el-dialog v-model="visible" title="延长档案有效期" width="400px">
    <el-form label-width="100px">
      <el-form-item label="延长天数">
        <el-input-number v-model="extendDays" :min="30" :max="365" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="handleExtend">确定延期</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps({
  modelValue: Boolean,
  profileId: String,
  creatorKey: String
})
const emit = defineEmits(['update:modelValue', 'extended'])

const visible = ref(props.modelValue)
const extendDays = ref(90)

watch(() => props.modelValue, v => { visible.value = v })
watch(visible, v => { emit('update:modelValue', v) })

async function handleExtend() {
  try {
    const res = await fetch(`/api/household/profile/${props.profileId}/extend?creator_key=${props.creatorKey}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ creator_key: props.creatorKey, expires_in: extendDays.value })
    })
    const data = await res.json()
    if (data.code === 0) {
      ElMessage.success('延期成功')
      visible.value = false
      emit('extended')
    } else {
      ElMessage.error(data.error || '延期失败')
    }
  } catch {
    ElMessage.error('延期失败')
  }
}
</script>
