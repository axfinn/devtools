<template>
  <el-dialog v-model="visible" title="添加物品" width="500px">
    <el-form :model="form" label-width="80px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" placeholder="物品名称" />
      </el-form-item>
      <el-form-item label="分类">
        <el-select v-model="form.category" placeholder="选择分类" style="width: 100%;">
          <el-option v-for="cat in categoryOptions" :key="cat" :label="cat" :value="cat" />
        </el-select>
      </el-form-item>
      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item label="数量">
            <el-input-number v-model="form.quantity" :min="1" style="width: 100%;" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="单位">
            <el-input v-model="form.unit" placeholder="个/瓶/盒..." />
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="最低库存">
        <el-input-number v-model="form.min_quantity" :min="1" style="width: 100%;" />
      </el-form-item>
      <el-form-item label="位置">
        <el-autocomplete
          v-model="form.location"
          :fetch-suggestions="queryLocationSuggestions"
          placeholder="如：厨房、冰箱"
          style="width: 100%;"
        />
      </el-form-item>
      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item label="生产日期">
            <el-date-picker v-model="form.expiry_date" type="date" placeholder="选择日期" style="width: 100%;" value-format="YYYY-MM-DD" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="保质期(天)">
            <el-input-number v-model="form.expiry_days" :min="0" placeholder="0=无保质期" style="width: 100%;" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="备注">
        <el-input v-model="form.notes" type="textarea" :rows="2" placeholder="可选备注" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="handleSubmit">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps({
  modelValue: Boolean,
  profileId: String,
  creatorKey: String,
  locations: { type: Array, default: () => [] }
})
const emit = defineEmits(['update:modelValue', 'added'])

const visible = ref(props.modelValue)
watch(() => props.modelValue, v => { visible.value = v })
watch(visible, v => { emit('update:modelValue', v) })

const categoryOptions = ['厨房', '卫生间', '卧室', '客厅', '玄关', '阳台', '其他']

const defaultForm = () => ({
  name: '',
  category: '其他',
  quantity: 1,
  unit: '个',
  min_quantity: 1,
  location: '',
  expiry_date: '',
  expiry_days: 0,
  notes: ''
})

const form = ref(defaultForm())

function fill(data) {
  form.value = { ...defaultForm(), ...data }
}
defineExpose({ fill })

function queryLocationSuggestions(query, cb) {
  const normalized = query.trim().toLowerCase()
  const results = props.locations
    .filter(loc => !normalized || loc.toLowerCase().includes(normalized))
    .map(loc => ({ value: loc }))
  cb(results)
}

async function handleSubmit() {
  if (!form.value.name) {
    ElMessage.warning('请输入物品名称')
    return
  }
  try {
    const res = await fetch(`/api/household/profile/${props.profileId}/items?creator_key=${props.creatorKey}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form.value)
    })
    const data = await res.json()
    if (data.code === 0) {
      ElMessage.success('添加成功')
      visible.value = false
      form.value = defaultForm()
      emit('added')
    } else {
      ElMessage.error(data.error || '添加失败')
    }
  } catch {
    ElMessage.error('添加失败')
  }
}
</script>
