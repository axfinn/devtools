<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>批量文本替换</h2>
      <div class="header-tips">
        <el-tag type="success" effect="plain">实时预览</el-tag>
        <el-tag type="warning" effect="plain">Ctrl+Enter 执行</el-tag>
      </div>
    </div>

    <div class="replace-layout">
      <!-- 左侧：输入和规则 -->
      <div class="left-panel">
        <!-- 源文本 -->
        <div class="editor-panel">
          <div class="panel-header">
            <span>
              <el-icon class="panel-icon"><Document /></el-icon>
              源文本
            </span>
            <div class="header-actions">
              <el-button size="small" @click="pasteSource">
                <el-icon><DocumentAdd /></el-icon> 粘贴
              </el-button>
              <el-button size="small" @click="clearSource">
                <el-icon><Delete /></el-icon> 清空
              </el-button>
            </div>
          </div>
          <textarea
            ref="sourceTextarea"
            v-model="sourceText"
            class="code-editor"
            placeholder="请输入要处理的文本，或粘贴内容..."
            spellcheck="false"
            rows="8"
            @input="autoPreview"
          ></textarea>
          <div class="editor-footer">
            <span class="char-count">字符数：{{ sourceText.length }}</span>
            <span class="line-count">行数：{{ sourceText.split('\n').length }}</span>
          </div>
        </div>

        <!-- 替换规则 -->
        <div class="rules-panel">
          <div class="panel-header">
            <span>
              <el-icon class="panel-icon"><Setting /></el-icon>
              替换规则
            </span>
            <div class="header-actions">
              <el-button size="small" type="primary" @click="addRule">
                <el-icon><Plus /></el-icon> 添加规则
              </el-button>
              <el-dropdown @command="handleQuickRule">
                <el-button size="small">
                  快捷规则 <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="trim">去除首尾空格</el-dropdown-item>
                    <el-dropdown-item command="empty">删除空行</el-dropdown-item>
                    <el-dropdown-item command="nbsp">空格转 nbsp</el-dropdown-item>
                    <el-dropdown-item command="cn-en">中文转英文标点</el-dropdown-item>
                    <el-dropdown-item command="lower">全部小写</el-dropdown-item>
                    <el-dropdown-item command="upper">全部大写</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </div>

          <div class="rules-list">
            <TransitionGroup name="rule-slide">
              <div v-for="(rule, index) in rules" :key="rule.id" class="rule-item">
                <div class="rule-header">
                  <div class="rule-title">
                    <el-tag size="small" effect="plain">{{ index + 1 }}</el-tag>
                    <span class="rule-status" :class="{ active: rule.find && rule.active }">
                      {{ rule.find ? (rule.active ? '已匹配' : '无匹配') : '待配置' }}
                    </span>
                  </div>
                  <div class="rule-actions">
                    <el-button
                      size="small"
                      type="success"
                      text
                      :disabled="!rule.find"
                      @click="testRule(index)"
                    >
                      测试
                    </el-button>
                    <el-button
                      size="small"
                      type="danger"
                      text
                      @click="removeRule(index)"
                    >
                      删除
                    </el-button>
                  </div>
                </div>

                <div class="rule-content">
                  <div class="rule-row">
                    <el-input
                      v-model="rule.find"
                      placeholder="查找内容"
                      size="default"
                      clearable
                      :prefix-icon="Search"
                      @input="testRule(index)"
                      @keydown.enter="applyReplace"
                    >
                      <template #prepend>查找</template>
                    </el-input>
                  </div>

                  <div class="rule-row">
                    <el-input
                      v-model="rule.replace"
                      placeholder="替换为（留空则删除）"
                      size="default"
                      clearable
                      :prefix-icon="Refresh"
                      @input="testRule(index)"
                      @keydown.enter="applyReplace"
                    >
                      <template #prepend>替换</template>
                    </el-input>
                  </div>

                  <div class="rule-options">
                    <el-tooltip content="使用正则表达式语法" placement="top">
                      <el-checkbox v-model="rule.isRegex" size="small" @change="testRule(index)">
                        <el-icon><Promotion /></el-icon> 正则
                      </el-checkbox>
                    </el-tooltip>
                    <el-tooltip content="区分大小写" placement="top">
                      <el-checkbox v-model="rule.caseSensitive" size="small" @change="testRule(index)">
                        <el-icon><Document /></el-icon> 区分大小写
                      </el-checkbox>
                    </el-tooltip>
                    <el-tooltip content="整个单词匹配" placement="top">
                      <el-checkbox v-model="rule.wholeWord" size="small" @change="testRule(index)">
                        <el-icon><CircleCheck /></el-icon> 整词
                      </el-checkbox>
                    </el-tooltip>
                  </div>

                  <div v-if="rule.matchCount > 0" class="rule-match-info">
                    <el-tag size="small" type="success">匹配 {{ rule.matchCount }} 处</el-tag>
                  </div>
                </div>
              </div>
            </TransitionGroup>

            <el-empty v-if="rules.length === 0" :image-size="60" description="暂无替换规则，请添加" />
          </div>

          <div class="rules-actions">
            <el-button type="primary" size="large" @click="applyReplace" :disabled="!sourceText">
              <el-icon><Check /></el-icon> 执行替换
            </el-button>
            <el-button size="large" @click="clearRules">
              <el-icon><Delete /></el-icon> 清空规则
            </el-button>
            <el-button size="large" @click="copyResult" :disabled="!resultText">
              <el-icon><CopyDocument /></el-icon> 复制结果
            </el-button>
          </div>
        </div>
      </div>

      <!-- 右侧：结果 -->
      <div class="right-panel">
        <div class="editor-panel result-panel">
          <div class="panel-header">
            <span>
              <el-icon class="panel-icon"><Finished /></el-icon>
              替换结果
              <el-tag v-if="replaceStats.total > 0" size="small" type="success" style="margin-left: 8px">
                {{ replaceStats.total }} 处替换
              </el-tag>
            </span>
            <div class="header-actions">
              <el-button size="small" type="primary" @click="useAsSource" :disabled="!resultText">
                <el-icon><RefreshRight /></el-icon> 设为源文本
              </el-button>
              <el-button size="small" @click="clearResult">
                <el-icon><Delete /></el-icon> 清空
              </el-button>
            </div>
          </div>
          <textarea
            ref="resultTextarea"
            v-model="resultText"
            class="code-editor result-editor"
            placeholder="替换后的结果..."
            spellcheck="false"
            rows="14"
          ></textarea>
          <div class="editor-footer">
            <span class="char-count">字符数：{{ resultText.length }}</span>
            <span class="line-count">行数：{{ resultText.split('\n').length }}</span>
            <span v-if="resultText.length !== sourceText.length" class="diff-count">
              变化：{{ resultText.length - sourceText.length > 0 ? '+' : '' }}{{ resultText.length - sourceText.length }} 字符
            </span>
          </div>
        </div>

        <!-- 快捷操作 -->
        <div class="quick-actions">
          <el-card shadow="hover">
            <template #header>
              <div class="card-header">
                <span>快捷操作</span>
              </div>
            </template>
            <div class="action-grid">
              <el-button @click="swapSourceResult" :disabled="!resultText">
                <el-icon><Sort /></el-icon> 交换源/结果
              </el-button>
              <el-button @click="sortLines" :disabled="!sourceText">
                <el-icon><Sort /></el-icon> 行排序
              </el-button>
              <el-button @click="uniqueLines" :disabled="!sourceText">
                <el-icon><Delete /></el-icon> 去重
              </el-button>
              <el-button @click="reverseLines" :disabled="!sourceText">
                <el-icon><Refresh /></el-icon> 行反转
              </el-button>
            </div>
          </el-card>
        </div>

        <!-- 使用说明 -->
        <div class="tips-section">
          <el-collapse>
            <el-collapse-item title="使用说明" name="tips">
              <ul class="tips-list">
                <li>支持添加多条替换规则，按顺序依次执行</li>
                <li>勾选 <strong>正则</strong> 可使用正则匹配（支持捕获组 <code>$1</code> <code>$2</code>）</li>
                <li>勾选 <strong>整词</strong> 匹配整个单词</li>
                <li>留空"替换为"字段可实现删除功能</li>
                <li>点击 <strong>测试</strong> 查看单条规则匹配结果</li>
                <li>输入文本后会自动实时预览匹配数量</li>
                <li>快捷键：<code>Ctrl+Enter</code> 执行替换</li>
              </ul>
            </el-collapse-item>
          </el-collapse>
        </div>
      </div>
    </div>

    <!-- 测试规则弹窗 -->
    <el-dialog v-model="showTestDialog" title="规则测试" width="600px">
      <div v-if="testResult" class="test-result">
        <div class="test-header">
          <el-tag type="success">匹配 {{ testResult.count }} 处</el-tag>
          <el-tag v-if="testResult.count > 0" type="info">点击高亮查看详情</el-tag>
        </div>
        <div class="test-matches">
          <div
            v-for="(match, idx) in testResult.matches"
            :key="idx"
            class="match-item"
          >
            <span class="match-index">{{ idx + 1 }}</span>
            <code class="match-text">{{ match }}</code>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Document, DocumentAdd, Delete, Plus, ArrowDown, Search, Refresh,
  Promotion, CircleCheck, Check, CopyDocument, RefreshRight, Sort, Finished, Setting
} from '@element-plus/icons-vue'

// 源文本
const sourceText = ref('')
const resultText = ref('')
const sourceTextarea = ref(null)

// 替换规则
const rules = ref([
  { id: 1, find: '', replace: '', isRegex: false, caseSensitive: true, wholeWord: false, matchCount: 0, active: false }
])

// 统计信息
const replaceStats = ref({ total: 0, regexError: false })

// 测试弹窗
const showTestDialog = ref(false)
const testResult = ref(null)

// 生成唯一ID
let ruleId = 1

// 添加规则
const addRule = () => {
  ruleId++
  rules.value.push({
    id: ruleId,
    find: '',
    replace: '',
    isRegex: false,
    caseSensitive: true,
    wholeWord: false,
    matchCount: 0,
    active: false
  })
}

// 删除规则
const removeRule = (index) => {
  rules.value.splice(index, 1)
  if (rules.value.length === 0) {
    addRule()
  }
}

// 清空规则
const clearRules = () => {
  rules.value = [{ id: ++ruleId, find: '', replace: '', isRegex: false, caseSensitive: true, wholeWord: false, matchCount: 0, active: false }]
  replaceStats.value = { total: 0, regexError: false }
}

// 测试单条规则
const testRule = (index) => {
  const rule = rules.value[index]
  if (!rule.find || !sourceText.value) {
    rule.matchCount = 0
    rule.active = false
    return
  }

  try {
    let flags = 'g'
    if (!rule.caseSensitive) flags += 'i'

    let pattern = rule.find
    if (rule.wholeWord && !rule.isRegex) {
      pattern = '\\b' + escapeRegExp(pattern) + '\\b'
    }

    const regex = new RegExp(pattern, flags)
    const matches = sourceText.value.match(regex)
    rule.matchCount = matches ? matches.length : 0
    rule.active = rule.matchCount > 0
  } catch (e) {
    rule.matchCount = 0
    rule.active = false
  }
}

// 实时预览匹配数
const autoPreview = () => {
  rules.value.forEach((rule, index) => {
    testRule(index)
  })
}

// 转义正则特殊字符
function escapeRegExp(string) {
  return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

// 执行替换
const applyReplace = () => {
  try {
    const stats = { total: 0, regexError: false }
    let text = sourceText.value

    for (const rule of rules.value) {
      if (!rule.find) continue

      try {
        let flags = 'g'
        if (!rule.caseSensitive) flags += 'i'

        let pattern = rule.find
        if (rule.wholeWord && !rule.isRegex) {
          pattern = '\\b' + escapeRegExp(pattern) + '\\b'
        }

        const regex = new RegExp(pattern, flags)
        const matches = text.match(regex)
        if (matches) {
          stats.total += matches.length
        }
        text = text.replace(regex, rule.replace)
      } catch (e) {
        if (e.message.includes('Invalid regular expression')) {
          stats.regexError = true
          ElMessage.error(`正则表达式错误: ${rule.find}`)
        }
      }
    }

    resultText.value = text
    replaceStats.value = stats

    if (stats.total > 0) {
      ElMessage.success(`替换完成，共替换 ${stats.total} 处`)
    } else {
      ElMessage.info('未找到匹配内容')
    }
  } catch (e) {
    ElMessage.error('替换失败: ' + e.message)
  }
}

// 快捷规则
const handleQuickRule = (command) => {
  const rule = rules.value.find(r => r.find === '') || rules.value[0]
  if (!rule) {
    addRule()
  }

  const lastRule = rules.value[rules.value.length - 1]
  switch (command) {
    case 'trim':
      lastRule.find = '^\\s+|\\s+$'
      lastRule.replace = ''
      lastRule.isRegex = true
      break
    case 'empty':
      lastRule.find = '^\\s*$'
      lastRule.replace = ''
      lastRule.isRegex = true
      break
    case 'nbsp':
      lastRule.find = ' '
      lastRule.replace = '&nbsp;'
      lastRule.isRegex = false
      break
    case 'cn-en':
      lastRule.find = `，|。|；|：|！|？|""|''|（|）`
      lastRule.replace = ',|.|;|:|!|?"|"||(|||'
      lastRule.isRegex = true
      break
    case 'lower':
      lastRule.find = '([A-Z])'
      lastRule.replace = '($1)'
      lastRule.isRegex = true
      break
    case 'upper':
      lastRule.find = '([a-z])'
      lastRule.replace = '($1)'
      lastRule.isRegex = true
      break
  }
  autoPreview()
  applyReplace()
}

// 粘贴源文本
const pasteSource = async () => {
  try {
    const text = await navigator.clipboard.readText()
    sourceText.value = text
    autoPreview()
    ElMessage.success('已粘贴')
  } catch (e) {
    ElMessage.error('粘贴失败')
  }
}

// 清空源文本
const clearSource = () => {
  sourceText.value = ''
  resultText.value = ''
  replaceStats.value = { total: 0, regexError: false }
  rules.value.forEach(r => { r.matchCount = 0; r.active = false })
}

// 复制结果
const copyResult = async () => {
  try {
    await navigator.clipboard.writeText(resultText.value)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

// 清空结果
const clearResult = () => {
  resultText.value = ''
  replaceStats.value = { total: 0, regexError: false }
}

// 设为源文本
const useAsSource = () => {
  sourceText.value = resultText.value
  resultText.value = ''
  replaceStats.value = { total: 0, regexError: false }
  autoPreview()
}

// 交换源和结果
const swapSourceResult = () => {
  const temp = sourceText.value
  sourceText.value = resultText.value
  resultText.value = temp
  replaceStats.value = { total: 0, regexError: false }
  autoPreview()
}

// 行排序
const sortLines = () => {
  const lines = sourceText.value.split('\n')
  lines.sort((a, b) => a.localeCompare(b, 'zh-CN'))
  sourceText.value = lines.join('\n')
  applyReplace()
}

// 行去重
const uniqueLines = () => {
  const lines = sourceText.value.split('\n')
  const unique = [...new Set(lines)]
  sourceText.value = unique.join('\n')
  applyReplace()
}

// 行反转
const reverseLines = () => {
  const lines = sourceText.value.split('\n')
  lines.reverse()
  sourceText.value = lines.join('\n')
  applyReplace()
}

// 键盘快捷键
const handleKeydown = (e) => {
  if (e.ctrlKey && e.key === 'Enter') {
    applyReplace()
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.tool-container {
  max-width: 1400px;
  margin: 0 auto;
}

.tool-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.tool-header h2 {
  font-size: 24px;
  color: var(--color-primary);
  margin: 0;
}

.header-tips {
  display: flex;
  gap: 8px;
}

.replace-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.left-panel,
.right-panel {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.editor-panel {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  padding: 15px;
  border: 1px solid var(--border-base);
}

.result-panel {
  border-color: var(--color-success);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 500;
}

.panel-icon {
  margin-right: 6px;
  vertical-align: middle;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.code-editor {
  width: 100%;
  background: var(--editor-bg);
  color: var(--editor-text);
  border: 1px solid var(--border-base);
  border-radius: var(--radius-sm);
  padding: 12px;
  font-family: var(--font-family-mono);
  font-size: 14px;
  line-height: 1.6;
  resize: vertical;
  min-height: 150px;
}

.code-editor:focus {
  outline: none;
  border-color: var(--input-border-focus);
}

.result-editor {
  min-height: 250px;
}

.editor-footer {
  display: flex;
  gap: 15px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-secondary);
}

.char-count,
.line-count {
  opacity: 0.7;
}

.diff-count {
  color: var(--color-warning);
  font-weight: 500;
}

.rules-panel {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  padding: 15px;
  border: 1px solid var(--border-base);
}

.rules-list {
  max-height: 380px;
  overflow-y: auto;
  margin: 10px 0;
}

/* 规则项动画 */
.rule-slide-enter-active,
.rule-slide-leave-active {
  transition: all 0.3s ease;
}

.rule-slide-enter-from {
  opacity: 0;
  transform: translateX(-20px);
}

.rule-slide-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

.rule-item {
  background: var(--bg-primary);
  border: 1px solid var(--border-base);
  border-radius: var(--radius-sm);
  padding: 12px;
  margin-bottom: 10px;
  transition: all 0.2s ease;
}

.rule-item:hover {
  border-color: var(--color-primary);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.rule-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.rule-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.rule-status {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 10px;
  background: var(--bg-secondary);
  color: var(--text-secondary);
}

.rule-status.active {
  background: var(--color-success-light);
  color: var(--color-success);
}

.rule-actions {
  display: flex;
  gap: 4px;
}

.rule-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rule-row {
  width: 100%;
}

.rule-options {
  display: flex;
  gap: 15px;
  margin-top: 5px;
}

.rule-options .el-checkbox {
  display: flex;
  align-items: center;
}

.rule-match-info {
  margin-top: 5px;
}

.rules-actions {
  display: flex;
  gap: 10px;
  margin-top: 15px;
  padding-top: 15px;
  border-top: 1px solid var(--border-base);
}

.rules-actions .el-button {
  flex: 1;
}

/* 快捷操作 */
.quick-actions {
  margin-top: 5px;
}

.card-header {
  font-size: 14px;
  font-weight: 500;
}

.action-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
}

.action-grid .el-button {
  width: 100%;
}

/* 使用说明 */
.tips-section {
  margin-top: 10px;
}

.tips-list {
  margin: 10px 0 0 0;
  padding-left: 20px;
  line-height: 1.8;
}

.tips-list li {
  margin: 6px 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.tips-list code {
  background: var(--code-inline-bg);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-family: var(--font-family-mono);
  color: var(--color-primary);
}

/* 测试结果弹窗 */
.test-result {
  padding: 10px 0;
}

.test-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 15px;
}

.test-matches {
  max-height: 300px;
  overflow-y: auto;
}

.match-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px;
  border-radius: var(--radius-sm);
  margin-bottom: 5px;
  background: var(--bg-secondary);
}

.match-index {
  min-width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary);
  color: white;
  border-radius: 50%;
  font-size: 12px;
}

.match-text {
  font-family: var(--font-family-mono);
  color: var(--color-success);
  word-break: break-all;
}

/* 移动端适配 */
@media (max-width: 1024px) {
  .replace-layout {
    grid-template-columns: 1fr;
  }

  .tool-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }
}

@media (max-width: 768px) {
  .rules-actions {
    flex-wrap: wrap;
  }

  .rules-actions .el-button {
    min-width: 100px;
  }

  .rule-options {
    flex-direction: column;
    gap: 5px;
  }

  .action-grid {
    grid-template-columns: 1fr;
  }
}
</style>
