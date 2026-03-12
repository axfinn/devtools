<template>
  <el-dialog v-model="visible" title="位置库" width="500px" @open="loadLibrary">
    <div class="location-library">
      <div class="location-form">
        <el-input v-model="form.name" placeholder="输入位置名称，例如：厨房水槽下" />
        <el-button type="primary" @click="saveLocation">{{ form.id ? '更新' : '新增' }}</el-button>
        <el-button v-if="form.id" @click="resetForm">取消</el-button>
      </div>
      <el-table :data="library" style="width: 100%" max-height="320">
        <el-table-column prop="name" label="位置名称" />
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button size="small" text @click="editLocation(row)">编辑</el-button>
            <el-button size="small" type="danger" text @click="deleteLocation(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="library.length === 0" description="暂无位置记录" />
    </div>
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
const emit = defineEmits(['update:modelValue', 'changed'])

const visible = ref(props.modelValue)
const library = ref([])
const form = ref({ id: '', name: '' })

watch(() => props.modelValue, v => { visible.value = v })
watch(visible, v => { emit('update:modelValue', v) })

async function loadLibrary() {
  if (!props.profileId) return
  try {
    const res = await fetch(`/api/household/profile/${props.profileId}/locations/library?creator_key=${props.creatorKey}`)
    const data = await res.json()
    if (data.code === 0) library.value = data.data || []
  } catch {}
}

function editLocation(row) { form.value = { id: row.id, name: row.name } }
function resetForm() { form.value = { id: '', name: '' } }

async function saveLocation() {
  const name = form.value.name.trim()
  if (!name) { ElMessage.warning('请输入位置名称'); return }
  try {
    let res
    if (form.value.id) {
      res = await fetch(`/api/household/profile/${props.profileId}/locations/library/${form.value.id}?creator_key=${props.creatorKey}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name })
      })
    } else {
      res = await fetch(`/api/household/profile/${props.profileId}/locations/library?creator_key=${props.creatorKey}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name })
      })
    }
    const data = await res.json()
    if (data.code === 0) {
      ElMessage.success(form.value.id ? '更新成功' : '新增成功')
      resetForm()
      await loadLibrary()
      emit('changed')
    } else {
      ElMessage.error(data.error || '操作失败')
    }
  } catch { ElMessage.error('操作失败') }
}

async function deleteLocation(row) {
  try {
    const res = await fetch(`/api/household/profile/${props.profileId}/locations/library/${row.id}?creator_key=${props.creatorKey}`, { method: 'DELETE' })
    const data = await res.json()
    if (data.code === 0) {
      ElMessage.success('删除成功')
      await loadLibrary()
      emit('changed')
    } else {
      ElMessage.error(data.error || '删除失败')
    }
  } catch { ElMessage.error('删除失败') }
}
</script>

<style scoped>
.location-library { min-height: 300px; }
.location-form { display: flex; gap: 8px; margin-bottom: 12px; }
</style>
