<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>每日菜谱</h2>
    </div>

    <!-- 健康提示 -->
    <el-alert type="success" :closable="false" class="health-tip">
      <template #title>
        <el-icon><InfoFilled /></el-icon>
        <span>本菜谱专为妊娠期糖尿病预防 + 肾结石预防设计，点击每天卡片可查看详细做法</span>
      </template>
    </el-alert>

    <!-- 周选择器 -->
    <div class="week-selector">
      <h3>第 {{ currentWeek }} 周菜谱</h3>
      <el-select v-model="currentWeek" size="large" style="width: 140px;">
        <el-option v-for="i in 4" :key="i" :label="'第' + i + '周'" :value="i" />
      </el-select>
    </div>

    <!-- 每日菜谱卡片 -->
    <div class="daily-cards">
      <el-card v-for="day in weeklyRecipes[currentWeek - 1]" :key="day.day" class="day-card">
        <template #header>
          <div class="day-header">
            <span class="day-name">{{ day.day }}</span>
            <el-tag size="small" type="info">{{ day.suggestion }}</el-tag>
          </div>
        </template>

        <div class="meals-simple">
          <div class="meal-item" :class="{ done: isDone(day.day, 'breakfast') }">
            <div class="meal-content">
              <span class="meal-label">早餐</span>
              <span class="meal-name">{{ day.breakfast }}</span>
            </div>
            <div class="meal-actions">
              <el-button size="small" :type="isDone(day.day, 'breakfast') ? 'success' : 'default'" @click="toggleDone(day.day, 'breakfast')">
                {{ isDone(day.day, 'breakfast') ? '已做' : '做过了' }}
              </el-button>
              <el-link :href="getRecipeUrl(day.day)" target="_blank" type="primary" :underline="false">
                <el-button size="small" type="primary" plain>做法</el-button>
              </el-link>
            </div>
          </div>

          <div class="meal-item" :class="{ done: isDone(day.day, 'lunch') }">
            <div class="meal-content">
              <span class="meal-label">午餐</span>
              <span class="meal-name">{{ day.lunch1 }}<br>{{ day.lunch2 }}<br>{{ day.lunch3 }}</span>
            </div>
            <div class="meal-actions">
              <el-button size="small" :type="isDone(day.day, 'lunch') ? 'success' : 'default'" @click="toggleDone(day.day, 'lunch')">
                {{ isDone(day.day, 'lunch') ? '已做' : '做过了' }}
              </el-button>
              <el-link :href="getRecipeUrl(day.day)" target="_blank" type="primary" :underline="false">
                <el-button size="small" type="primary" plain>做法</el-button>
              </el-link>
            </div>
          </div>

          <div class="meal-item" :class="{ done: isDone(day.day, 'dinner') }">
            <div class="meal-content">
              <span class="meal-label">晚餐</span>
              <span class="meal-name">{{ day.dinner }}</span>
            </div>
            <div class="meal-actions">
              <el-button size="small" :type="isDone(day.day, 'dinner') ? 'success' : 'default'" @click="toggleDone(day.day, 'dinner')">
                {{ isDone(day.day, 'dinner') ? '已做' : '做过了' }}
              </el-button>
              <el-link :href="getRecipeUrl(day.day)" target="_blank" type="primary" :underline="false">
                <el-button size="small" type="primary" plain>做法</el-button>
              </el-link>
            </div>
          </div>
        </div>
      </el-card>
    </div>

    <!-- 统计 -->
    <el-card class="stats-card">
      <template #header>
        <span>本周记录</span>
      </template>
      <el-statistic :value="doneCount" title="本周已做菜品数" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { InfoFilled } from '@element-plus/icons-vue'

const currentWeek = ref(1)
const BLOG_BASE = 'https://blog.jaxiu.cn/blog/2026-02'

// 从 localStorage 读取已完成的菜品
const doneRecipes = ref(JSON.parse(localStorage.getItem('doneRecipes') || '{}'))

// 检查是否已完成
const isDone = (day, meal) => {
  const key = `${currentWeek.value}-${day}-${meal}`
  return doneRecipes.value[key]
}

// 切换完成状态
const toggleDone = (day, meal) => {
  const key = `${currentWeek.value}-${day}-${meal}`
  doneRecipes.value[key] = !doneRecipes.value[key]
  localStorage.setItem('doneRecipes', JSON.stringify(doneRecipes.value))
  ElMessage.success(doneRecipes.value[key] ? '已记录：这道菜做过了！' : '已取消记录')
}

// 获取菜谱博客链接
const getRecipeUrl = (day) => {
  const week = currentWeek.value
  const dayMap = { '周一': 1, '周二': 2, '周三': 3, '周四': 4, '周五': 5, '周六': 6, '周日': 7 }
  const dayNum = dayMap[day]
  return `${BLOG_BASE}/pregnancy-recipe-w${week}d${dayNum}/`
}

// 已完成数量
const doneCount = computed(() => {
  let count = 0
  const week = currentWeek.value
  weeklyRecipes.value[week - 1]?.forEach(day => {
    if (doneRecipes.value[`${week}-${day.day}-breakfast`]) count++
    if (doneRecipes.value[`${week}-${day.day}-lunch`]) count++
    if (doneRecipes.value[`${week}-${day.day}-dinner`]) count++
  })
  return count
})

// 每周菜谱数据 - 午餐改为3个菜
const weeklyRecipes = [
  [
    { day: '周一', breakfast: '燕麦粥+水煮蛋+小番茄', lunch1: '清蒸鲈鱼', lunch2: '西兰花炒肉片', lunch3: '蒜蓉油麦菜', dinner: '冬瓜排骨汤+清炒油麦菜', suggestion: '多喝水，午休后散步' },
    { day: '周二', breakfast: '全麦面包+牛奶+香蕉', lunch1: '番茄炒蛋', lunch2: '紫菜蛋花汤', lunch3: '清炒上海青', dinner: '清炒鸡肉+凉拌黄瓜', suggestion: '晚餐不宜太晚' },
    { day: '周三', breakfast: '玉米糊+鸡蛋羹+苹果', lunch1: '红烧肉（瘦）', lunch2: '麻婆豆腐', lunch3: '炒青菜', dinner: '鲫鱼豆腐汤+蒸南瓜', suggestion: '睡前少喝水' },
    { day: '周四', breakfast: '小米粥+咸鸭蛋（少）+坚果', lunch1: '炒鸡胸肉', lunch2: '胡萝卜炒木耳', lunch3: '凉拌黄瓜', dinner: '清蒸虾+凉拌木耳+米饭', suggestion: '每天运动30分钟' },
    { day: '周五', breakfast: '豆浆+油条（少）+水果', lunch1: '土豆烧牛肉', lunch2: '西红柿炒菜花', lunch3: '炒生菜', dinner: '冬瓜虾皮汤+炒生菜', suggestion: '周末去公园散步' },
    { day: '周六', breakfast: '牛奶+鸡蛋+全麦饼干', lunch1: '清蒸排骨', lunch2: '炒白菜', lunch3: '紫菜汤', dinner: '番茄鱼片汤+凉拌黄瓜', suggestion: '家人一起用餐' },
    { day: '周日', breakfast: '八宝粥+鸡蛋+坚果', lunch1: '红烧鸡翅', lunch2: '菠菜炒蛋', lunch3: '炒油麦菜', dinner: '紫菜蛋花汤+玉米+水果', suggestion: '准备下周食材' }
  ],
  [
    { day: '周一', breakfast: '燕麦牛奶+水煮蛋+葡萄', lunch1: '清蒸鲈鱼', lunch2: '炒西兰花', lunch3: '蒜蓉油麦菜', dinner: '玉米排骨汤+凉拌黄瓜', suggestion: '多喝水促进排石' },
    { day: '周二', breakfast: '全麦吐司+酸奶+蓝莓', lunch1: '炒牛肉丝', lunch2: '胡萝卜丝炒蛋', lunch3: '清炒生菜', dinner: '冬瓜肉丸汤+清炒生菜', suggestion: '饭后散步' },
    { day: '周三', breakfast: '玉米糊+荷包蛋+苹果', lunch1: '白灼虾', lunch2: '西葫芦炒蛋', lunch3: '炒菠菜', dinner: '番茄炒蛋+米饭+紫菜汤', suggestion: '控制盐量' },
    { day: '周四', breakfast: '豆浆+粗粮饼+香蕉', lunch1: '炖鸡腿', lunch2: '红烧土豆', lunch3: '炒青菜', dinner: '豆腐鱼头汤+凉拌木耳', suggestion: '睡前2小时不喝水' },
    { day: '周五', breakfast: '小米粥+鸡蛋+坚果', lunch1: '清蒸鳕鱼', lunch2: '西兰花炒虾仁', lunch3: '蒜蓉白菜', dinner: '丝瓜蛋花汤+炒白菜', suggestion: '午休后活动' },
    { day: '周六', breakfast: '牛奶+鸡蛋+全麦面包', lunch1: '红烧肉（瘦）', lunch2: '菠菜炒蛋', lunch3: '炒豆芽', dinner: '冬瓜虾皮汤+玉米', suggestion: '周末家庭日' },
    { day: '周日', breakfast: '八宝粥+咸鸭蛋+水果', lunch1: '炒鸡胸肉', lunch2: '胡萝卜炒肉片', lunch3: '清炒油麦菜', dinner: '鲫鱼豆腐汤+凉拌黄瓜', suggestion: '准备下周计划' }
  ],
  [
    { day: '周一', breakfast: '燕麦粥+鸡蛋+小番茄', lunch1: '清蒸鲈鱼', lunch2: '炒油麦菜', lunch3: '西兰花炒胡萝卜', dinner: '冬瓜排骨汤+凉拌黄瓜', suggestion: '每天8杯水' },
    { day: '周二', breakfast: '全麦面包+牛奶+苹果', lunch1: '番茄牛腩', lunch2: '麻婆豆腐', lunch3: '炒青菜', dinner: '紫菜蛋花汤+蒸南瓜', suggestion: '餐后活动' },
    { day: '周三', breakfast: '玉米糊+鸡蛋羹+香蕉', lunch1: '白灼虾', lunch2: '西兰花炒蛋', lunch3: '蒜蓉生菜', dinner: '豆腐肉末汤+炒生菜', suggestion: '少盐少油' },
    { day: '周四', breakfast: '豆浆+油条（少）+坚果', lunch1: '炒鸡胸肉', lunch2: '胡萝卜炒肉', lunch3: '凉拌木耳', dinner: '冬瓜虾皮汤+凉拌木耳', suggestion: '适度运动' },
    { day: '周五', breakfast: '小米粥+鸡蛋+水果', lunch1: '清蒸排骨', lunch2: '白菜炖豆腐', lunch3: '炒菠菜', dinner: '番茄鱼片汤+玉米', suggestion: '周末采购' },
    { day: '周六', breakfast: '牛奶+全麦饼干+蓝莓', lunch1: '红烧肉（瘦）', lunch2: '菠菜炒蛋', lunch3: '炒油麦菜', dinner: '丝瓜蛋花汤+炒青菜', suggestion: '家庭聚餐' },
    { day: '周日', breakfast: '八宝粥+鸡蛋+坚果', lunch1: '炖鸡翅', lunch2: '西葫芦炒蛋', lunch3: '清炒白菜', dinner: '紫菜汤+蒸南瓜+水果', suggestion: '总结下周' }
  ],
  [
    { day: '周一', breakfast: '燕麦牛奶+水煮蛋+葡萄', lunch1: '清蒸鲈鱼', lunch2: '西兰花炒虾仁', lunch3: '蒜蓉菠菜', dinner: '冬瓜排骨汤+凉拌黄瓜', suggestion: '多喝水' },
    { day: '周二', breakfast: '全麦吐司+酸奶+苹果', lunch1: '炒牛肉', lunch2: '油麦菜炒蛋', lunch3: '胡萝卜炒木耳', dinner: '豆腐鲫鱼汤+炒青菜', suggestion: '饭后散步' },
    { day: '周三', breakfast: '玉米糊+荷包蛋+香蕉', lunch1: '白灼虾', lunch2: '胡萝卜炒蛋', lunch3: '清炒生菜', dinner: '番茄炒蛋+米饭+紫菜汤', suggestion: '控制糖分' },
    { day: '周四', breakfast: '豆浆+粗粮饼+坚果', lunch1: '炖鸡腿', lunch2: '红烧土豆块', lunch3: '炒菠菜', dinner: '冬瓜虾皮汤+蒸南瓜', suggestion: '少油' },
    { day: '周五', breakfast: '小米粥+鸡蛋+水果', lunch1: '清蒸鳕鱼', lunch2: '白菜炒豆腐', lunch3: '蒜蓉油麦菜', dinner: '丝瓜肉片汤+凉拌木耳', suggestion: '周末计划' },
    { day: '周六', breakfast: '牛奶+鸡蛋+全麦面包', lunch1: '红烧肉（瘦）', lunch2: '炒青菜', lunch3: '紫菜蛋花汤', dinner: '玉米+水果+凉拌黄瓜', suggestion: '家庭时光' },
    { day: '周日', breakfast: '八宝粥+咸鸭蛋+坚果', lunch1: '炒鸡胸肉', lunch2: '胡萝卜炒肉片', lunch3: '清炒油麦菜', dinner: '鲫鱼豆腐汤+黄瓜', suggestion: '下周准备' }
  ]
]
</script>

<style scoped>
.tool-container {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.tool-header {
  margin-bottom: 20px;
}

.tool-header h2 {
  margin: 0;
  font-size: 24px;
  color: #303133;
}

.health-tip {
  margin-bottom: 20px;
}

.health-tip :deep(.el-alert__title) {
  display: flex;
  align-items: center;
  gap: 8px;
}

.week-selector {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.week-selector h3 {
  margin: 0;
  color: #409EFF;
  font-size: 20px;
}

.daily-cards {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 15px;
  margin-bottom: 20px;
}

.day-card {
  transition: transform 0.2s;
}

.day-card:hover {
  transform: translateY(-3px);
}

.day-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.day-name {
  font-weight: bold;
  font-size: 16px;
  color: #303133;
}

.meals-simple {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.meal-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  border-radius: 6px;
  background: #f5f7fa;
}

.meal-item.done {
  background: #f0f9eb;
  border-left: 3px solid #67C23A;
}

.meal-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.meal-label {
  font-size: 12px;
  color: #909399;
  font-weight: 500;
}

.meal-name {
  font-size: 13px;
  color: #303133;
  line-height: 1.4;
}

.meal-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.stats-card {
  margin-top: 20px;
}

@media (max-width: 768px) {
  .tool-container {
    padding: 10px;
  }

  .daily-cards {
    grid-template-columns: 1fr;
  }

  .week-selector {
    flex-direction: column;
    gap: 10px;
    align-items: flex-start;
  }
}
</style>
