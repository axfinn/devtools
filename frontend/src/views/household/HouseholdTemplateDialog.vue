<template>
  <el-dialog v-model="visible" title="物品模板库" width="600px">
    <div class="template-grid">
      <div v-for="(items, category) in templatesByCategory" :key="category" class="template-category">
        <div class="category-title">{{ category }}</div>
        <div class="template-items">
          <el-tag
            v-for="tpl in items"
            :key="tpl.id"
            class="template-tag"
            @click="handleSelect(tpl)"
          >
            {{ tpl.name }} ({{ tpl.unit }})
          </el-tag>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  modelValue: Boolean,
  templatesByCategory: { type: Object, default: () => ({}) }
})
const emit = defineEmits(['update:modelValue', 'select'])

const visible = ref(props.modelValue)
watch(() => props.modelValue, v => { visible.value = v })
watch(visible, v => { emit('update:modelValue', v) })

function handleSelect(tpl) {
  emit('select', tpl)
  visible.value = false
}
</script>

<style scoped>
.template-grid {
  max-height: 400px;
  overflow-y: auto;
}
.template-category {
  margin-bottom: 16px;
}
.category-title {
  font-weight: bold;
  margin-bottom: 8px;
  color: #409eff;
}
.template-items {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.template-tag {
  cursor: pointer;
}
.template-tag:hover {
  opacity: 0.8;
}
</style>
