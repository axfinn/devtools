<template>
  <el-dialog v-model="visible" title="购物清单" width="600px">
    <div class="shopping-list-container">
      <div class="shopping-list-header">
        <el-alert :title="'共 ' + shoppingList.length + ' 项需要购买'" type="warning" :closable="false" />
      </div>

      <div class="todo-section">
        <el-divider>待购买任务</el-divider>
        <el-table :data="todos" style="width: 100%" max-height="220" v-loading="todosLoading">
          <el-table-column prop="name" label="物品名称" />
          <el-table-column prop="category" label="分类" width="100" />
          <el-table-column prop="reason" label="原因" />
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag v-if="row.status === 'done'" type="success" size="small">已完成</el-tag>
              <el-tag v-else type="warning" size="small">待购买</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120">
            <template #default="{ row }">
              <el-button size="small" type="success" text @click="updateTodoStatus(row, row.status === 'done' ? 'open' : 'done')">
                {{ row.status === 'done' ? '重开' : '完成' }}
              </el-button>
              <el-button size="small" type="danger" text @click="deleteTodo(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="!todosLoading && todos.length === 0" description="暂无待购买任务" />
        <div class="todo-actions">
          <el-button size="small" @click="createTodosFromList">从购物清单生成任务</el-button>
          <el-button size="small" type="primary" :loading="aiMergeLoading" @click="mergeTodosWithAI">AI 去重合并</el-button>
        </div>
      </div>

      <div v-if="shoppingList.length > 0" class="shopping-list-content">
        <el-table :data="shoppingList" style="width: 100%" max-height="400">
          <el-table-column prop="name" label="物品名称" />
          <el-table-column prop="category" label="分类" width="100" />
          <el-table-column prop="reason" label="原因" />
          <el-table-column label="操作" width="80">
            <template #default="{ $index }">
              <el-button size="small" type="danger" text @click="shoppingList.splice($index, 1)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <el-empty v-else description="库存充足，暂无需要购买的物品" />

      <div class="shopping-list-actions">
        <el-button @click="copyList">
          <el-icon><CopyDocument /></el-icon>
          复制清单
        </el-button>
        <el-button type="primary" @click="exportList">
          <el-icon><Download /></el-icon>
          导出文本
        </el-button>
        <el-button @click="shareList">
          <el-icon><Share /></el-icon>
          生成分享链接
        </el-button>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Delete, CopyDocument, Download, Share } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: Boolean,
  profileId: String,
  creatorKey: String,
  items: { type: Array, default: () => [] }
})
const emit = defineEmits(['update:modelValue'])

const visible = ref(props.modelValue)
const shoppingList = ref([])
const todos = ref([])
const todosLoading = ref(false)
const aiMergeLoading = ref(false)

watch(() => props.modelValue, v => {
  visible.value = v
  if (v) {
    generateShoppingList()
    loadTodos()
  }
})
watch(visible, v => { emit('update:modelValue', v) })

function generateShoppingList() {
  shoppingList.value = []
  props.items.forEach(item => {
    let reason = ''
    if (item.quantity <= item.min_quantity) {
      reason = `库存不足（当前 ${item.quantity}${item.unit}，需要 ${item.min_quantity}${item.unit}）`
    } else if (item.expiry_date && item.expiry_days > 0) {
      const expiryDate = new Date(item.expiry_date)
      expiryDate.setDate(expiryDate.getDate() + item.expiry_days)
      const daysUntil = Math.ceil((expiryDate - new Date()) / (1000 * 60 * 60 * 24))
      if (daysUntil <= 0) reason = '已过期'
      else if (daysUntil <= 7) reason = `即将过期（还剩 ${daysUntil} 天）`
    }
    if (reason) {
      shoppingList.value.push({ id: item.id, name: item.name, category: item.category, reason })
    }
  })
  shoppingList.value.sort((a, b) => a.category.localeCompare(b.category))
}

async function loadTodos() {
  if (!props.profileId) return
  todosLoading.value = true
  try {
    const res = await fetch(`/api/household/todos?profile_id=${props.profileId}&creator_key=${props.creatorKey}`)
    const data = await res.json()
    if (data.code === 0) todos.value = data.data || []
  } catch {
    console.error('加载待办失败')
  } finally {
    todosLoading.value = false
  }
}

async function updateTodoStatus(todo, status) {
  try {
    const res = await fetch(`/api/household/todos/${todo.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ profile_id: props.profileId, creator_key: props.creatorKey, status })
    })
    const data = await res.json()
    if (data.code === 0) await loadTodos()
    else ElMessage.error(data.error || '更新失败')
  } catch { ElMessage.error('更新失败') }
}

async function deleteTodo(todo) {
  try {
    const res = await fetch(`/api/household/todos/${todo.id}?profile_id=${props.profileId}&creator_key=${props.creatorKey}`, { method: 'DELETE' })
    const data = await res.json()
    if (data.code === 0) await loadTodos()
    else ElMessage.error(data.error || '删除失败')
  } catch { ElMessage.error('删除失败') }
}

async function createTodosFromList() {
  if (shoppingList.value.length === 0) return
  let added = 0
  for (const item of shoppingList.value) {
    try {
      const res = await fetch('/api/household/todos', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ profile_id: props.profileId, creator_key: props.creatorKey, name: item.name, category: item.category || '其他', reason: item.reason || '从购物清单生成' })
      })
      const data = await res.json()
      if (data.code === 0) added++
    } catch {}
  }
  if (added > 0) {
    ElMessage.success(`已创建 ${added} 个待购买任务`)
    await loadTodos()
  } else {
    ElMessage.error('创建失败')
  }
}

async function mergeTodosWithAI() {
  if (!props.profileId || todos.value.length < 2) return
  aiMergeLoading.value = true
  try {
    const res = await fetch('/api/household/ai/todos/merge', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ profile_id: props.profileId, creator_key: props.creatorKey })
    })
    const data = await res.json()
    if (data.code === 0) {
      if (data.merged > 0) ElMessage.success(`已合并 ${data.merged} 项任务`)
      else ElMessage.info(data.message || '没有可合并任务')
      await loadTodos()
    } else {
      ElMessage.error(data.error || 'AI 合并失败')
    }
  } catch { ElMessage.error('AI 合并失败') }
  finally { aiMergeLoading.value = false }
}

function copyList() {
  let text = '购物清单\n' + '='.repeat(20) + '\n\n'
  shoppingList.value.forEach((item, idx) => {
    text += `${idx + 1}. ${item.name} (${item.category})\n   ${item.reason}\n\n`
  })
  text += `共计 ${shoppingList.value.length} 项`
  navigator.clipboard.writeText(text).then(() => ElMessage.success('清单已复制到剪贴板')).catch(() => ElMessage.error('复制失败'))
}

function exportList() {
  let content = '购物清单\n' + '='.repeat(20) + '\n\n'
  shoppingList.value.forEach((item, idx) => {
    content += `${idx + 1}. ${item.name} (${item.category})\n   ${item.reason}\n\n`
  })
  content += `\n共计 ${shoppingList.value.length} 项`
  const blob = new Blob([content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = '购物清单.txt'
  link.click()
  URL.revokeObjectURL(url)
  ElMessage.success('清单已导出')
}

async function shareList() {
  const shareData = {
    type: 'household_shopping_list',
    profile_id: props.profileId,
    items: shoppingList.value.map(item => ({ name: item.name, category: item.category, reason: item.reason }))
  }
  try {
    const res = await fetch('/api/shorturl', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ original_url: `${window.location.origin}/household?share=${btoa(JSON.stringify(shareData))}` })
    })
    const data = await res.json()
    if (data.id) {
      const shareUrl = `${window.location.origin}/s/${data.id}`
      navigator.clipboard.writeText(shareUrl).then(() => ElMessage.success('分享链接已复制到剪贴板')).catch(() => ElMessage.info(`分享链接: ${shareUrl}`))
    } else {
      ElMessage.error('生成分享链接失败')
    }
  } catch { ElMessage.error('分享失败') }
}
</script>

<style scoped>
.shopping-list-container { min-height: 300px; }
.shopping-list-header { margin-bottom: 16px; }
.shopping-list-content { margin: 16px 0; }
.shopping-list-actions { margin-top: 16px; display: flex; justify-content: center; gap: 12px; }
.todo-section { margin-bottom: 16px; }
.todo-actions { margin-top: 8px; text-align: right; }
</style>
