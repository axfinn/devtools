<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>生活记账</h2>
      <div class="actions">
        <template v-if="profileId">
          <el-tag :type="saveStatus === 'saving' ? 'warning' : 'success'" size="small">
            {{ saveStatus === 'saving' ? '保存中...' : '已保存' }}
          </el-tag>
          <el-tag type="info" size="small">永久档案</el-tag>
          <el-button @click="logoutProfile">
            退出并切换档案
          </el-button>
          <el-button type="danger" @click="confirmDelete">
            <el-icon><Delete /></el-icon>
            删除档案
          </el-button>
        </template>
      </div>
    </div>

    <!-- No profile: Create or Load -->
    <div v-if="!profileId" class="welcome-section">
      <div class="welcome-card">
        <h3>创建记账档案</h3>
        <p>输入密码，创建您的专属记账档案。</p>
        <el-form :model="createForm" label-width="80px" style="max-width: 400px; margin: 20px auto;">
          <el-form-item label="密码">
            <el-input v-model="createForm.password" type="password" placeholder="至少4个字符"
              show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="createProfile" :loading="creating">创建档案</el-button>
          </el-form-item>
        </el-form>
      </div>

      <div class="welcome-card" style="margin-top: 20px;">
        <h3>加载已有档案</h3>
        <p>输入创建时设置的密码即可加载档案。</p>
        <el-form :model="loadForm" label-width="80px" style="max-width: 400px; margin: 20px auto;">
          <el-form-item label="密码">
            <el-input v-model="loadForm.password" type="password" placeholder="输入密码"
              show-password @keyup.enter="loadProfile" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="loadProfile" :loading="loading">加载档案</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- Profile loaded: Main content -->
    <template v-else>
      <!-- Account balance overview -->
      <div class="balance-overview">
        <!-- 今日收支 -->
        <div class="today-stats">
          <div class="today-label">今日</div>
          <div class="today-values">
            <span class="today-income">收 {{ formatMoney(todayStats.income) }}</span>
            <span class="today-expense">支 {{ formatMoney(todayStats.expense) }}</span>
          </div>
        </div>

        <!-- 周期统计 -->
        <div class="balance-card income">
          <div class="balance-label">{{ recordRangeLabel }}收入</div>
          <div class="balance-value">{{ formatMoney(recordMonthSummary.income) }}</div>
        </div>
        <div class="balance-card expense">
          <div class="balance-label">{{ recordRangeLabel }}支出</div>
          <div class="balance-value">{{ formatMoney(recordMonthSummary.expense) }}</div>
        </div>
        <div class="balance-card balance">
          <div class="balance-label">{{ recordRangeLabel }}结余</div>
          <div class="balance-value" :class="{ negative: recordMonthSummary.income - recordMonthSummary.expense < 0 }">
            {{ formatMoney(recordMonthSummary.income - recordMonthSummary.expense) }}
          </div>
        </div>
      </div>

      <!-- 记账成功动画 -->
      <transition name="slide-fade">
        <div v-if="showSuccessAnimation" class="success-animation">
          <div class="success-icon">✓</div>
          <div class="success-text">记账成功</div>
        </div>
      </transition>

      <!-- Custom Tab Bar - Desktop & Mobile -->
      <div class="custom-tab-bar">
        <button
          v-for="tab in tabItems"
          :key="tab.name"
          class="tab-pill"
          :class="{ active: activeTab === tab.name }"
          @click="switchTab(tab.name)"
        >
          <span class="tab-pill-icon">{{ tab.icon }}</span>
          <span class="tab-pill-label">{{ tab.label }}</span>
        </button>
      </div>

      <div class="tab-content-wrapper">
        <!-- Record Tab -->
        <div v-show="activeTab === 'record'" class="tab-pane">
          <div class="record-section">
            <!-- 快捷记账区 -->
            <div class="quick-entry">
              <!-- 金额输入 + 类型切换 -->
              <div class="amount-row">
                <div class="type-toggle">
                  <button :class="{ active: txForm.type === 'expense' }" @click="txForm.type = 'expense'">支出</button>
                  <button :class="{ active: txForm.type === 'income' }" @click="txForm.type = 'income'">收入</button>
                </div>
                <div class="amount-input-wrapper">
                  <span class="currency">¥</span>
                  <input
                    v-model.number="txForm.amount"
                    type="number"
                    class="amount-input"
                    placeholder="0"
                    @keyup.enter="quickAddTransaction"
                    ref="amountInput"
                  />
                </div>
              </div>

              <!-- 分类快捷按钮 -->
              <div class="category-grid">
                <div
                  v-for="cat in quickCategories"
                  :key="cat.id"
                  class="category-btn"
                  :class="{ active: txForm.category_id === cat.id }"
                  :style="{ '--cat-color': cat.color }"
                  @click="txForm.category_id = cat.id"
                >
                  <span class="cat-icon">{{ cat.icon }}</span>
                  <span class="cat-name">{{ cat.name }}</span>
                </div>
              </div>

              <!-- 备注快捷输入 -->
              <div class="remark-row">
                <el-input
                  v-model="txForm.remark"
                  placeholder="添加备注（可点击下方快捷标签）"
                  @keyup.enter="quickAddTransaction"
                />
                <div class="remark-tags" v-if="recentRemarks.length > 0">
                  <el-tag
                    v-for="remark in recentRemarks"
                    :key="remark"
                    size="small"
                    class="remark-tag"
                    @click="txForm.remark = remark"
                  >
                    {{ remark }}
                  </el-tag>
                </div>
              </div>

              <!-- 更多分类选择 -->
              <div class="more-categories">
                <el-dropdown trigger="click" @command="handleCategorySelect">
                  <el-button type="info" plain size="small">
                    更多分类 <el-icon><ArrowDown /></el-icon>
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu class="category-dropdown">
                      <el-dropdown-item disabled>支出分类</el-dropdown-item>
                      <el-dropdown-item v-for="cat in expenseCategories" :key="cat.id" :command="cat.id">
                        <span :style="{ color: cat.color }">●</span> {{ cat.name }}
                      </el-dropdown-item>
                      <el-dropdown-item disabled>收入分类</el-dropdown-item>
                      <el-dropdown-item v-for="cat in incomeCategories" :key="cat.id" :command="cat.id">
                        <span :style="{ color: cat.color }">●</span> {{ cat.name }}
                      </el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>

              <div class="datetime-row">
                <el-date-picker
                  v-model="txForm.date"
                  type="date"
                  format="YYYY-MM-DD"
                  value-format="YYYY-MM-DD"
                  class="datetime-field"
                />
                <el-time-picker
                  v-model="txForm.time"
                  format="HH:mm"
                  value-format="HH:mm"
                  placeholder="时间"
                  class="datetime-field"
                />
                <span class="datetime-hint">默认上海时间</span>
              </div>

              <!-- 操作按钮 -->
              <div class="action-row">
                <div class="account-selector">
                  <el-dropdown @command="handleAccountChange">
                    <el-button size="small">
                      {{ currentAccount?.name || '选择账户' }}
                      <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item v-for="acc in accounts" :key="acc.id" :command="acc.id">
                          <span :style="{ color: acc.color }">{{ acc.name }}</span>
                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
                <el-button type="primary" size="large" @click="quickAddTransaction" :loading="addingTx" class="submit-btn">
                  记账
                </el-button>
                <el-button size="large" @click="showVoicePanel = true" type="success">
                  <el-icon><Microphone /></el-icon>
                  语音记账
                </el-button>
              </div>
            </div>

            <!-- 语音/快捷输入面板 -->
            <div v-if="showVoicePanel" class="voice-panel">
              <div class="voice-panel-header">
                <span>快速记账</span>
                <el-button link @click="showVoicePanel = false">
                  <el-icon><Close /></el-icon>
                </el-button>
              </div>

              <!-- 标签切换 -->
              <div class="voice-tabs">
                <button :class="{ active: voiceMode === 'voice' }" @click="voiceMode = 'voice'">
                  <el-icon><Microphone /></el-icon> 语音
                </button>
                <button :class="{ active: voiceMode === 'manual' }" @click="voiceMode = 'manual'">
                  <el-icon><Edit /></el-icon> 手动
                </button>
              </div>

              <!-- 语音模式 -->
              <div v-if="voiceMode === 'voice'" class="voice-mode">
                <!-- 开始录音按钮 -->
                <div v-if="!isRecording" class="voice-wave" @click="toggleVoice">
                  <div class="wave-icon">
                    <el-icon :size="48"><Microphone /></el-icon>
                  </div>
                  <div class="wave-text">点击说话记账</div>
                </div>

                <!-- 录音中状态 -->
                <div v-else class="recording-box">
                  <div class="recording-pulse"></div>
                  <span>正在录音...</span>
                  <el-button type="danger" @click="stopVoice">结束</el-button>
                </div>

                <!-- 识别结果 -->
                <div v-if="voiceText && !isRecording" class="voice-result-card">
                  <div class="result-text">"{{ voiceText }}"</div>
                  <div class="voice-confirm-btns">
                    <el-button type="primary" @click="confirmVoiceRecord" :loading="voiceRecordStatus === 'processing'">
                      确认记账
                    </el-button>
                    <el-button @click="voiceText = ''">取消</el-button>
                  </div>
                </div>

                <!-- 切换到手动输入 -->
                <div v-if="!isRecording" style="margin-top: 15px;">
                  <el-button link type="primary" @click="voiceMode = 'manual'">
                    切换到手动输入
                  </el-button>
                </div>

                <!-- 状态显示 -->
                <div v-if="voiceRecordStatus" class="voice-status" :class="voiceRecordStatus">
                  <template v-if="voiceRecordStatus === 'processing'">
                    <el-icon class="is-loading"><Loading /></el-icon>
                    {{ voiceRecordMessage }}
                  </template>
                  <template v-else-if="voiceRecordStatus === 'success'">
                    <el-icon><CircleCheck /></el-icon>
                    {{ voiceRecordMessage }}
                  </template>
                  <template v-else-if="voiceRecordStatus === 'failed'">
                    <el-icon><CircleClose /></el-icon>
                    {{ voiceRecordMessage }}
                  </template>
                </div>

                <!-- 识别结果：始终显示 -->
                <div v-if="voiceText" class="voice-result-card">
                  <div class="result-text">"{{ voiceText }}"</div>
                  <div class="voice-confirm-btns">
                    <el-button type="primary" @click="confirmVoiceRecord" :loading="voiceRecordStatus === 'processing'">
                      确认记账
                    </el-button>
                    <el-button @click="cancelVoiceRecord">取消</el-button>
                  </div>
                </div>
              </div>

              <!-- 手动模式 -->
              <div v-else class="manual-mode">
                <el-input
                  v-model="manualText"
                  type="textarea"
                  :rows="3"
                  placeholder="例如：中午吃饭花了35块"
                />
                <el-button type="primary" @click="parseManualText" :loading="parsingManual" style="margin-top: 10px; width: 100%;">
                  智能解析
                </el-button>
              </div>

              <!-- 解析结果 -->
              <div v-if="parsedResult" class="parsed-result">
                <div class="parsed-amount">
                  <span class="label">金额:</span>
                  <span class="value">{{ parsedResult.amount }} 元</span>
                </div>
                <div class="parsed-category">
                  <span class="label">分类:</span>
                  <el-tag :type="parsedResult.type === 'income' ? 'success' : 'danger'">
                    {{ parsedResult.category || '未识别' }}
                  </el-tag>
                </div>
                <div class="parsed-type">
                  <span class="label">类型:</span>
                  <el-tag :type="parsedResult.type === 'income' ? 'success' : 'danger'">
                    {{ parsedResult.type === 'income' ? '收入' : '支出' }}
                  </el-tag>
                </div>
                <el-button type="success" size="large" @click="confirmParsedRecord" style="margin-top: 15px; width: 100%;">
                  确认记账
                </el-button>
              </div>
            </div>

            <!-- Transaction list -->
            <div class="tx-list">
              <div class="tx-list-header">
                <div>
                  <span>交易记录</span>
                  <div class="tx-list-subtitle">
                    {{ recordRangeLabel }}共 {{ transactions.length }} 条，当前筛出 {{ filteredTransactions.length }} 条
                  </div>
                </div>
                <div class="tx-header-actions">
                  <div class="record-month-nav">
                    <el-button size="small" @click="shiftRecordMonth(-1)">
                      <el-icon><ArrowLeft /></el-icon>
                    </el-button>
                    <span class="record-month-label">{{ recordRangeLabel }}</span>
                    <el-button size="small" @click="shiftRecordMonth(1)">
                      <el-icon><ArrowRight /></el-icon>
                    </el-button>
                  </div>
                  <el-button size="small" @click="exportData">
                    <el-icon><Download /></el-icon>
                    导出
                  </el-button>
                  <el-upload
                    :show-file-list="false"
                    :before-upload="handleImport"
                    accept=".csv"
                    action=""
                  >
                    <el-button size="small">
                      <el-icon><Upload /></el-icon>
                      导入
                    </el-button>
                  </el-upload>
                  <el-button size="small" @click="resetRecordFilters">
                    <el-icon><Refresh /></el-icon>
                    重置筛选
                  </el-button>
                </div>
              </div>

              <div class="record-summary-grid">
                <div class="record-summary-card">
                  <div class="summary-label">筛选后支出</div>
                  <div class="summary-value expense">¥{{ recordSummary.expense.toFixed(2) }}</div>
                </div>
                <div class="record-summary-card">
                  <div class="summary-label">筛选后收入</div>
                  <div class="summary-value income">¥{{ recordSummary.income.toFixed(2) }}</div>
                </div>
                <div class="record-summary-card">
                  <div class="summary-label">最大支出分类</div>
                  <div class="summary-text">{{ recordSummary.topCategory || '暂无' }}</div>
                </div>
                <div class="record-summary-card">
                  <div class="summary-label">最高单笔支出</div>
                  <div class="summary-text">{{ recordSummary.maxExpense ? `${recordSummary.maxExpense.category_name || '未分类'} · ${formatMoney(recordSummary.maxExpense.amount)}` : '暂无' }}</div>
                </div>
              </div>

              <div class="record-filters">
                <el-input
                  v-model="recordKeyword"
                  clearable
                  placeholder="搜索备注 / 细项 / 标签"
                  class="record-filter-item record-filter-keyword"
                >
                  <template #prefix><el-icon><Search /></el-icon></template>
                </el-input>
                <el-select v-model="recordTypeFilter" clearable placeholder="类型" class="record-filter-item">
                  <el-option label="支出" value="expense" />
                  <el-option label="收入" value="income" />
                </el-select>
                <el-select v-model="recordAccountFilter" clearable placeholder="账户" class="record-filter-item">
                  <el-option v-for="acc in accounts" :key="acc.id" :label="acc.name" :value="acc.id" />
                </el-select>
                <el-select v-model="recordCategoryFilter" clearable placeholder="分类" class="record-filter-item">
                  <el-option v-for="cat in categories" :key="cat.id" :label="cat.name" :value="cat.id" />
                </el-select>
              </div>

              <el-table :data="pagedTransactions" stripe style="width: 100%;" max-height="520" v-loading="loadingTxs">
                <el-table-column prop="date" label="日期" width="132">
                  <template #default="{ row }">
                    <div class="tx-date-cell">
                      <div>{{ row.date }}</div>
                      <div class="tx-date-time">{{ getTransactionTime(row) || '全天' }}</div>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column prop="account_name" label="账户" width="110">
                  <template #default="{ row }">
                    <el-tag size="small" :style="{ borderColor: getAccountColor(row.account_id) }">
                      {{ row.account_name }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="category_name" label="分类" width="120">
                  <template #default="{ row }">
                    <div class="tx-category-cell">
                      <span :style="{ color: row.category_color }">{{ row.category_name }}</span>
                      <span class="tx-subcategory" v-if="getTransactionSubLabel(row) && getTransactionSubLabel(row) !== row.category_name">
                        {{ getTransactionSubLabel(row) }}
                      </span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column prop="type" label="类型" width="60">
                  <template #default="{ row }">
                    <el-tag :type="row.type === 'income' ? 'success' : 'danger'" size="small">
                      {{ row.type === 'income' ? '收入' : '支出' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="amount" label="金额" width="100">
                  <template #default="{ row }">
                    <span :class="{ 'text-income': row.type === 'income', 'text-expense': row.type === 'expense' }">
                      {{ row.type === 'income' ? '+' : '-' }}{{ formatMoney(row.amount) }}
                    </span>
                  </template>
                </el-table-column>
                <el-table-column prop="remark" label="备注 / 明细" min-width="260" show-overflow-tooltip>
                  <template #default="{ row }">
                    <div class="tx-remark-cell">
                      <div class="tx-remark-title">
                        <span v-if="row.voice_text" :title="row.voice_text" class="tx-voice-remark">
                          <el-icon><Microphone /></el-icon>
                          {{ getTransactionMainRemark(row) || '语音记账' }}
                        </span>
                        <span v-else>{{ getTransactionMainRemark(row) || '未填写备注' }}</span>
                      </div>
                      <div v-if="getTransactionDetailRemark(row)" class="tx-remark-detail">
                        {{ getTransactionDetailRemark(row) }}
                      </div>
                      <div v-if="getTransactionMetaChips(row).length > 0" class="tx-meta-chips">
                        <el-tag
                          v-for="chip in getTransactionMetaChips(row)"
                          :key="chip"
                          size="small"
                          effect="plain"
                        >
                          {{ chip }}
                        </el-tag>
                      </div>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="150" fixed="right">
                  <template #default="{ row }">
                    <el-button link type="info" size="small" @click="openTransactionDetail(row)" title="详情">
                      <el-icon><View /></el-icon>
                    </el-button>
                    <el-button link type="success" size="small" @click="duplicateTransaction(row)" title="复账">
                      <el-icon><CopyDocument /></el-icon>
                    </el-button>
                    <el-button link type="primary" size="small" @click="openEditDialog(row)">
                      <el-icon><Edit /></el-icon>
                    </el-button>
                    <el-button link type="danger" size="small" @click="deleteTransaction(row.id)">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>

              <!-- Mobile card view -->
              <div class="mobile-tx-cards" v-if="pagedTransactions.length > 0">
                <div
                  v-for="tx in pagedTransactions"
                  :key="tx.id"
                  class="mobile-tx-card"
                  @click="openTransactionDetail(tx)"
                >
                  <div class="mtc-left">
                    <span class="mtc-cat-dot" :style="{ background: tx.category_color || '#909399' }"></span>
                  </div>
                  <div class="mtc-center">
                    <div class="mtc-title">{{ getTransactionMainRemark(tx) || tx.category_name || '未分类' }}</div>
                    <div class="mtc-meta">
                      <span class="mtc-cat-tag" :style="{ color: tx.category_color }">{{ tx.category_name }}</span>
                      <span class="mtc-sep">·</span>
                      <span>{{ tx.account_name }}</span>
                      <span class="mtc-sep">·</span>
                      <span>{{ tx.date }} {{ getTransactionTime(tx) || '' }}</span>
                    </div>
                  </div>
                  <div class="mtc-right">
                    <span :class="tx.type === 'income' ? 'text-income' : 'text-expense'">
                      {{ tx.type === 'income' ? '+' : '-' }}{{ formatMoney(tx.amount) }}
                    </span>
                    <div class="mtc-actions" @click.stop>
                      <el-button link size="small" @click="openEditDialog(tx)"><el-icon><Edit /></el-icon></el-button>
                      <el-button link size="small" type="danger" @click="deleteTransaction(tx.id)"><el-icon><Delete /></el-icon></el-button>
                    </div>
                  </div>
                </div>
                <el-empty v-if="pagedTransactions.length === 0" description="暂无记录" :image-size="60" />
              </div>

              <div class="tx-pagination">
                <el-pagination
                  v-model:current-page="recordPage"
                  v-model:page-size="recordPageSize"
                  background
                  layout="total, sizes, prev, pager, next"
                  :total="filteredTransactions.length"
                  :page-sizes="[20, 50, 100, 200]"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- History Tab -->
        <div v-show="activeTab === 'history'" class="tab-pane">
          <div class="history-section">
            <div class="history-toolbar">
              <el-select v-model="historyMonth" placeholder="选择月份" class="history-toolbar-item">
                <el-option v-for="item in historyMonthOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
              <el-select v-model="historyTypeFilter" clearable placeholder="类型" class="history-toolbar-item">
                <el-option label="支出" value="expense" />
                <el-option label="收入" value="income" />
              </el-select>
              <el-select v-model="historyCategoryFilter" clearable placeholder="分类" class="history-toolbar-item">
                <el-option v-for="cat in historyCategoryOptions" :key="cat.id" :label="cat.name" :value="cat.id" />
              </el-select>
              <el-input v-model="historyKeyword" clearable placeholder="搜索备注 / 细项" class="history-toolbar-item history-keyword">
                <template #prefix><el-icon><Search /></el-icon></template>
              </el-input>
            </div>

            <div class="history-summary">
              <div>筛选后记录 {{ filteredHistoryTransactions.length }} 条</div>
              <div>支出 {{ formatMoney(historySummary.expense) }}</div>
              <div>收入 {{ formatMoney(historySummary.income) }}</div>
            </div>

            <div v-if="historyGroups.length === 0" class="history-empty">
              <el-empty description="当前条件下没有历史记录" />
            </div>
            <div v-else class="history-groups">
              <div v-for="group in historyGroups" :key="group.date" class="history-day-card">
                <div class="history-day-header">
                  <div>
                    <strong>{{ group.date }}</strong>
                    <span class="history-day-count">{{ group.items.length }} 笔</span>
                  </div>
                  <div class="history-day-amounts">
                    <span class="text-expense" v-if="group.expense > 0">支 {{ formatMoney(group.expense) }}</span>
                    <span class="text-income" v-if="group.income > 0">收 {{ formatMoney(group.income) }}</span>
                  </div>
                </div>
                <div class="history-item-list">
                  <button
                    v-for="item in group.items"
                    :key="item.id"
                    class="history-item"
                    type="button"
                    @click="openTransactionDetail(item)"
                  >
                    <div class="history-item-main">
                      <span class="history-item-category" :style="{ color: item.category_color }">{{ item.category_name || '未分类' }}</span>
                      <span class="history-item-sub">{{ getHistoryItemSummary(item) }}</span>
                    </div>
                    <div class="history-item-side">
                      <span class="history-item-time">{{ getTransactionTime(item) || '全天' }}</span>
                      <span :class="item.type === 'income' ? 'text-income' : 'text-expense'">
                        {{ item.type === 'income' ? '+' : '-' }}{{ formatMoney(item.amount) }}
                      </span>
                    </div>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Stats Tab -->
        <div v-show="activeTab === 'stats'" class="tab-pane">
          <div class="stats-section">
            <div class="stats-hero">
              <div>
                <div class="stats-kicker">统计哲学</div>
                <h3>钱从哪来，花到哪去，什么时候失控</h3>
                <p>{{ statsRangeLabel }} 内的数据会先讲清现金流，再解释结构和异常，所有图表都会跟随下面筛选条件同步变化。</p>
              </div>
              <div class="stats-hero-chips">
                <span class="stats-hero-chip">筛选后 {{ filteredStatsTransactions.length }} 条</span>
                <span class="stats-hero-chip">活跃日 {{ statsOverview.activeDays }}</span>
                <span class="stats-hero-chip">重点分类 {{ statsOverview.topCategory?.name || '暂无' }}</span>
              </div>
            </div>

            <div class="stats-control-row">
              <el-radio-group v-model="statsPeriod" @change="loadStats" size="small">
                <el-radio-button label="month">月视图</el-radio-button>
                <el-radio-button label="year">年视图</el-radio-button>
                <el-radio-button label="custom">自定义</el-radio-button>
              </el-radio-group>
              <div v-if="statsPeriod === 'month'" class="stats-month-nav">
                <el-button size="small" @click="shiftStatsMonth(-1)">
                  <el-icon><ArrowLeft /></el-icon>
                </el-button>
                <span class="stats-month-label">{{ statsRangeLabel }}</span>
                <el-button size="small" @click="shiftStatsMonth(1)">
                  <el-icon><ArrowRight /></el-icon>
                </el-button>
              </div>
              <el-date-picker
                v-if="statsPeriod === 'custom'"
                v-model="statsCustomRange"
                type="daterange"
                unlink-panels
                range-separator="至"
                start-placeholder="开始日期"
                end-placeholder="结束日期"
                value-format="YYYY-MM-DD"
                class="stats-range-picker"
                @change="loadStats"
              />
            </div>

            <div class="stats-detail-filters">
              <el-select v-model="statsDetailTypeFilter" clearable placeholder="类型" class="stats-filter-item">
                <el-option label="支出" value="expense" />
                <el-option label="收入" value="income" />
              </el-select>
              <el-select v-model="statsDetailAccountFilter" clearable placeholder="账户" class="stats-filter-item">
                <el-option v-for="acc in accounts" :key="acc.id" :label="acc.name" :value="acc.id" />
              </el-select>
              <el-select v-model="statsDetailCategoryFilter" clearable placeholder="分类" class="stats-filter-item">
                <el-option v-for="cat in statsFilterCategoryOptions" :key="cat.id" :label="cat.name" :value="cat.id" />
              </el-select>
              <el-input
                v-model="statsDetailKeyword"
                clearable
                placeholder="筛选备注 / 子类 / 细项"
                class="stats-filter-item stats-filter-keyword"
              >
                <template #prefix><el-icon><Search /></el-icon></template>
              </el-input>
            </div>

            <div class="stats-overview-grid">
              <div class="stats-overview-card">
                <div class="stats-overview-label">收入</div>
                <div class="stats-overview-value text-income">{{ formatMoney(statsOverview.income) }}</div>
              </div>
              <div class="stats-overview-card">
                <div class="stats-overview-label">支出</div>
                <div class="stats-overview-value text-expense">{{ formatMoney(statsOverview.expense) }}</div>
              </div>
              <div class="stats-overview-card">
                <div class="stats-overview-label">结余</div>
                <div class="stats-overview-value" :class="statsOverview.balance >= 0 ? 'text-income' : 'text-expense'">{{ formatMoney(statsOverview.balance) }}</div>
              </div>
              <div class="stats-overview-card">
                <div class="stats-overview-label">储蓄率</div>
                <div class="stats-overview-value">{{ `${(statsOverview.savingsRate * 100).toFixed(1)}%` }}</div>
              </div>
              <div class="stats-overview-card">
                <div class="stats-overview-label">活跃天数</div>
                <div class="stats-overview-value">{{ statsOverview.activeDays }}</div>
              </div>
              <div class="stats-overview-card">
                <div class="stats-overview-label">日均支出</div>
                <div class="stats-overview-value">{{ formatMoney(statsOverview.avgPerDay) }}</div>
              </div>
            </div>

            <div class="stats-insight-panel">
              <div class="stats-insight-header">
                <div>
                  <h4>解读摘要</h4>
                  <p>先看结构，再看节奏，最后盯住最大波动。</p>
                </div>
                <div class="stats-insight-badge">
                  最大单笔
                  <strong>{{ statsDetailSummary.maxExpense ? formatMoney(statsDetailSummary.maxExpense.amount) : '暂无' }}</strong>
                </div>
              </div>
              <ul v-if="statsInsightList.length > 0" class="stats-insight-list">
                <li v-for="item in statsInsightList" :key="item">{{ item }}</li>
              </ul>
              <el-empty v-else description="当前筛选下还没有足够数据形成洞察" :image-size="70" />
            </div>

            <div class="stats-chart-grid">
              <div class="chart-card">
                <div class="deep-table-header">
                  <h4>支出结构</h4>
                  <span>看钱主要被哪些分类吃掉</span>
                </div>
                <template v-if="statsCategoryRows.length > 0">
                  <div ref="categoryChartRef" style="height: 320px;"></div>
                </template>
                <el-empty v-else description="暂无支出结构数据" :image-size="80" />
              </div>
              <div class="chart-card">
                <div class="deep-table-header">
                  <h4>时间节奏</h4>
                  <span>看收入、支出与净流的变化</span>
                </div>
                <template v-if="statsTrendRows.length > 0">
                  <div ref="monthChartRef" style="height: 320px;"></div>
                </template>
                <el-empty v-else description="暂无时间趋势数据" :image-size="80" />
              </div>
            </div>

            <div class="chart-card" style="margin-top: 20px;">
              <div class="deep-table-header">
                <h4>账户流向</h4>
                <span>账户不是分类，而是现金流经过的通道</span>
              </div>
              <template v-if="statsAccountRows.length > 0">
                <div ref="accountChartRef" style="height: 280px;"></div>
              </template>
              <el-empty v-else description="暂无账户流向数据" :image-size="80" />
            </div>

            <div class="stats-table-grid">
              <div class="chart-card">
                <div class="deep-table-header">
                  <h4>分类结构</h4>
                  <span>按分类聚合后的主要支出</span>
                </div>
                <el-table :data="statsCategoryRows.slice(0, 10)" size="small" max-height="360" v-loading="statsDetailLoading">
                  <el-table-column prop="name" label="分类" min-width="120" />
                  <el-table-column prop="count" label="笔数" width="80" />
                  <el-table-column prop="amount" label="金额" width="120">
                    <template #default="{ row }">{{ formatMoney(row.amount) }}</template>
                  </el-table-column>
                  <el-table-column prop="share" label="占比" width="90">
                    <template #default="{ row }">{{ row.share.toFixed(1) }}%</template>
                  </el-table-column>
                </el-table>
              </div>

              <div class="chart-card">
                <div class="deep-table-header">
                  <h4>子类 / 细项聚合</h4>
                  <span>{{ filteredStatsTransactions.length }} 条明细内聚合</span>
                </div>
                <el-table :data="statsSubcategoryRows" size="small" max-height="360" v-loading="statsDetailLoading">
                  <el-table-column prop="name" label="细项 / 子类" min-width="160" />
                  <el-table-column prop="count" label="笔数" width="80" />
                  <el-table-column prop="amount" label="金额" width="120">
                    <template #default="{ row }">{{ formatMoney(row.amount) }}</template>
                  </el-table-column>
                </el-table>
              </div>

              <div class="chart-card">
                <div class="deep-table-header">
                  <h4>重点日期</h4>
                  <span>高支出日期优先排查</span>
                </div>
                <el-table :data="statsDayRows" size="small" max-height="360" v-loading="statsDetailLoading">
                  <el-table-column prop="name" label="日期" width="120" />
                  <el-table-column prop="count" label="笔数" width="80" />
                  <el-table-column prop="amount" label="支出" width="120">
                    <template #default="{ row }">{{ formatMoney(row.amount) }}</template>
                  </el-table-column>
                </el-table>
              </div>

              <div class="chart-card">
                <div class="deep-table-header">
                  <h4>高额支出</h4>
                  <span>先看最有破坏力的单笔</span>
                </div>
                <el-table :data="statsTopTransactions" size="small" max-height="360" v-loading="statsDetailLoading">
                  <el-table-column prop="date" label="日期" width="110" />
                  <el-table-column label="内容" min-width="180">
                    <template #default="{ row }">
                      <div class="tx-category-cell">
                        <strong>{{ row.category_name || '未分类' }}</strong>
                        <span class="tx-subcategory">{{ getHistoryItemSummary(row) }}</span>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column prop="amount" label="金额" width="120">
                    <template #default="{ row }">{{ formatMoney(row.amount) }}</template>
                  </el-table-column>
                </el-table>
              </div>
            </div>
          </div>
        </div>

        <!-- Calendar Tab -->
        <div v-show="activeTab === 'calendar'" class="tab-pane">
          <div class="calendar-section">
            <div class="calendar-header">
              <el-button size="small" @click="prevMonth">
                <el-icon><ArrowLeft /></el-icon>
              </el-button>
              <span class="calendar-title">{{ calendarYear }}年{{ calendarMonth + 1 }}月</span>
              <el-button size="small" @click="nextMonth">
                <el-icon><ArrowRight /></el-icon>
              </el-button>
              <el-button size="small" @click="goToToday" style="margin-left: 10px;">今天</el-button>
            </div>

            <div class="calendar-grid">
              <div class="calendar-weekday" v-for="day in ['日', '一', '二', '三', '四', '五', '六']" :key="day">{{ day }}</div>
              <div
                v-for="(day, idx) in calendarDays"
                :key="idx"
                class="calendar-day"
                :class="{
                  'other-month': !day.currentMonth,
                  'today': day.isToday,
                  'has-data': day.expense > 0 || day.income > 0
                }"
                @click="day.expense > 0 || day.income > 0 ? showDayDetail(day) : null"
              >
                <div class="day-number">{{ day.date }}</div>
                <div class="day-amount" v-if="day.expense > 0 || day.income > 0">
                  <div class="day-expense" v-if="day.expense > 0">-{{ day.expense }}</div>
                  <div class="day-income" v-if="day.income > 0">+{{ day.income }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Backup Tab -->
        <div v-show="activeTab === 'backup'" class="tab-pane">
          <div class="backup-section">
            <el-alert
              title="数据备份"
              type="info"
              :closable="false"
              style="margin-bottom: 20px;"
            >
              导出所有数据包括账户、分类、交易记录，可以用于数据迁移或备份
            </el-alert>

            <el-button type="primary" size="large" @click="backupData('json')" :loading="backingUp">
              <el-icon><Download /></el-icon>
              导出 JSON
            </el-button>
            <el-button type="success" size="large" @click="backupData('csv')" :loading="backingUp">
              <el-icon><Download /></el-icon>
              导出 CSV
            </el-button>

            <el-divider />

            <el-alert
              title="数据恢复"
              type="warning"
              :closable="false"
              style="margin-bottom: 20px;"
            >
              导入会向当前档案追加数据；建议在空白档案中恢复或导入
            </el-alert>

            <el-upload
              :show-file-list="true"
              :before-upload="handleRestore"
              accept=".json"
              action=""
            >
              <el-button type="warning" size="large">
                <el-icon><Upload /></el-icon>
                导入备份数据
              </el-button>
            </el-upload>

            <el-divider />

            <el-alert
              title="从 NFS 分享导入"
              type="info"
              :closable="false"
              style="margin-bottom: 20px;"
            >
              支持粘贴 NFS 分享链接导入 CSV / JSON 账单，例如 `https://t.jaxiu.cn/nfs/a852f4ae`
            </el-alert>

            <div class="space-y-3" style="max-width: 560px;">
              <el-input
                v-model="nfsImportLink"
                placeholder="输入 NFS 分享链接"
                clearable
              />
              <el-input
                v-model="nfsImportPassword"
                type="password"
                placeholder="如果分享设了访问密码，在这里填写"
                show-password
                clearable
              />
              <el-button type="primary" size="large" @click="importFromNFSShare" :loading="importingFromNFS">
                从 NFS 分享导入
              </el-button>
            </div>
          </div>
        </div>

        <!-- AI Analyze Tab -->
        <div v-show="activeTab === 'analyze'" class="tab-pane">
          <div class="analyze-section">
            <el-radio-group v-model="analyzePeriod" size="small" style="margin-bottom: 20px;">
              <el-radio-button label="week">本周</el-radio-button>
              <el-radio-button label="month">本月</el-radio-button>
              <el-radio-button label="year">本年</el-radio-button>
              <el-radio-button label="custom">自定义</el-radio-button>
            </el-radio-group>
            <div v-if="analyzePeriod === 'custom'" style="margin-bottom: 20px;">
              <el-date-picker
                v-model="analyzeCustomRange"
                type="daterange"
                unlink-panels
                range-separator="至"
                start-placeholder="开始日期"
                end-placeholder="结束日期"
                value-format="YYYY-MM-DD"
                style="width: 100%; max-width: 420px;"
              />
            </div>

            <div class="ai-focus-row">
              <span class="ai-focus-label">分析重点</span>
              <el-radio-group v-model="analysisFocus" size="small">
                <el-radio-button label="overview">总览</el-radio-button>
                <el-radio-button label="savings">节流</el-radio-button>
                <el-radio-button label="anomaly">异常</el-radio-button>
                <el-radio-button label="category">分类结构</el-radio-button>
              </el-radio-group>
            </div>

            <div style="margin-bottom: 20px;">
              <el-button type="primary" @click="runAnalyze" :loading="analyzing">
                <el-icon><MagicStick /></el-icon>
                提交 AI 分析
              </el-button>
              <el-button @click="clearAnalyzeState" v-if="analysisResult || analysisStatus">
                清空结果
              </el-button>
            </div>

            <!-- 分析结果展示 -->
            <div v-if="analyzing" class="loading-state">
              <el-icon class="is-loading" :size="40"><MagicStick /></el-icon>
              <p>AI 任务已提交，正在异步生成分析结果...</p>
            </div>

            <el-alert
              v-else-if="analysisStatus === 'running' || analysisStatus === 'queued'"
              title="AI 解读任务正在后台执行，页面会自动刷新结果。"
              type="info"
              :closable="false"
              style="margin-bottom: 20px;"
            />

            <el-alert
              v-else-if="analysisStatus === 'failed'"
              title="AI 解读失败，请稍后重试。"
              type="error"
              :closable="false"
              style="margin-bottom: 20px;"
            />

            <el-empty v-else-if="!analysisResult" description="点击上方按钮获取 AI 消费分析" />

            <div v-else class="analysis-result">
              <div class="ai-badge">
                <el-icon><MagicStick /></el-icon>
                AI 分析结果
              </div>
              <div class="analysis-content">
                <pre>{{ analysisResult }}</pre>
              </div>
            </div>
          </div>
        </div>

        <!-- Settings Tab -->
        <div v-show="activeTab === 'settings'" class="tab-pane">
          <div class="settings-section">
            <el-row :gutter="20">
              <el-col :span="12">
                <div class="settings-card">
                  <div class="settings-header">
                    <h4>账户管理</h4>
                    <el-button type="primary" size="small" @click="showAddAccount = true">添加账户</el-button>
                  </div>
                  <el-table :data="accounts" size="small">
                    <el-table-column prop="name" label="名称" />
                    <el-table-column prop="type" label="类型" width="80">
                      <template #default="{ row }">
                        {{ accountTypeLabels[row.type] || row.type }}
                      </template>
                    </el-table-column>
                    <el-table-column prop="balance" label="余额">
                      <template #default="{ row }">
                        {{ formatMoney(row.balance) }}
                      </template>
                    </el-table-column>
                    <el-table-column label="操作" width="80">
                      <template #default="{ row }">
                        <el-button link type="danger" size="small" @click="deleteAccount(row.id)">
                          <el-icon><Delete /></el-icon>
                        </el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                </div>
              </el-col>
              <el-col :span="12">
                <div class="settings-card">
                  <div class="settings-header">
                    <h4>分类管理</h4>
                    <el-button type="primary" size="small" @click="showAddCategory = true">添加分类</el-button>
                  </div>
                  <el-table :data="categories" size="small">
                    <el-table-column prop="name" label="名称" />
                    <el-table-column prop="type" label="类型" width="80">
                      <template #default="{ row }">
                        <el-tag :type="row.type === 'income' ? 'success' : 'danger'" size="small">
                          {{ row.type === 'income' ? '收入' : '支出' }}
                        </el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column label="操作" width="80">
                      <template #default="{ row }">
                        <el-button link type="danger" size="small" @click="deleteCategory(row.id)">
                          <el-icon><Delete /></el-icon>
                        </el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                </div>
              </el-col>
            </el-row>
          </div>
        </div>
      </div>

      <!-- Add Account Dialog -->
      <el-dialog v-model="showAddAccount" title="添加账户" width="400px">
        <el-form :model="accountForm" label-width="60px">
          <el-form-item label="名称">
            <el-input v-model="accountForm.name" placeholder="账户名称" />
          </el-form-item>
          <el-form-item label="类型">
            <el-select v-model="accountForm.type" placeholder="选择类型" style="width: 100%;">
              <el-option label="现金" value="cash" />
              <el-option label="银行卡" value="bank" />
              <el-option label="支付宝" value="alipay" />
              <el-option label="微信" value="wechat" />
              <el-option label="信用卡" value="credit" />
            </el-select>
          </el-form-item>
          <el-form-item label="余额">
            <el-input-number v-model="accountForm.balance" :precision="2" style="width: 100%;" />
          </el-form-item>
          <el-form-item label="颜色">
            <el-color-picker v-model="accountForm.color" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showAddAccount = false">取消</el-button>
          <el-button type="primary" @click="addAccount">确定</el-button>
        </template>
      </el-dialog>

      <!-- Add Category Dialog -->
      <el-dialog v-model="showAddCategory" title="添加分类" width="400px">
        <el-form :model="categoryForm" label-width="60px">
          <el-form-item label="名称">
            <el-input v-model="categoryForm.name" placeholder="分类名称" />
          </el-form-item>
          <el-form-item label="类型">
            <el-radio-group v-model="categoryForm.type">
              <el-radio label="expense">支出</el-radio>
              <el-radio label="income">收入</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="颜色">
            <el-color-picker v-model="categoryForm.color" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showAddCategory = false">取消</el-button>
          <el-button type="primary" @click="addCategory">确定</el-button>
        </template>
      </el-dialog>

      <!-- Edit Transaction Dialog -->
      <el-dialog v-model="showEditDialog" title="修改记账" width="450px">
        <el-form :model="editForm" label-width="70px">
          <el-form-item label="类型">
            <el-radio-group v-model="editForm.type" size="small">
              <el-radio-button label="expense">支出</el-radio-button>
              <el-radio-button label="income">收入</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="金额">
            <el-input-number v-model="editForm.amount" :min="0" :precision="2" style="width: 100%;" />
          </el-form-item>
          <el-form-item label="账户">
            <el-select v-model="editForm.account_id" style="width: 100%;">
              <el-option v-for="acc in accounts" :key="acc.id" :label="acc.name" :value="acc.id">
                <span :style="{ color: acc.color }">{{ acc.name }}</span>
              </el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="分类">
            <el-select v-model="editForm.category_id" style="width: 100%;">
              <el-option-group v-for="group in categorizedCategories" :key="group.type" :label="group.label">
                <el-option v-for="cat in group.categories" :key="cat.id" :label="cat.name" :value="cat.id">
                  <span :style="{ color: cat.color }">{{ cat.name }}</span>
                </el-option>
              </el-option-group>
            </el-select>
          </el-form-item>
          <el-form-item label="日期">
            <el-date-picker v-model="editForm.date" type="date" format="YYYY-MM-DD" value-format="YYYY-MM-DD" style="width: 100%;" />
          </el-form-item>
          <el-form-item label="时间">
            <el-time-picker
              v-model="editForm.time"
              format="HH:mm"
              value-format="HH:mm"
              placeholder="记录时间"
              style="width: 100%;"
            />
          </el-form-item>
          <el-form-item label="备注">
            <el-input v-model="editForm.remark" placeholder="备注" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showEditDialog = false">取消</el-button>
          <el-button type="primary" @click="saveEditTransaction" :loading="savingEdit">保存</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="showTransactionDetailDialog" title="交易详情" width="520px">
        <template v-if="selectedTransaction">
          <div class="tx-detail-panel">
            <div class="tx-detail-row">
              <span class="tx-detail-label">日期</span>
              <span>{{ selectedTransaction.date }} {{ getTransactionTime(selectedTransaction) || '' }}</span>
            </div>
            <div class="tx-detail-row">
              <span class="tx-detail-label">类型</span>
              <el-tag :type="selectedTransaction.type === 'income' ? 'success' : 'danger'" size="small">
                {{ selectedTransaction.type === 'income' ? '收入' : '支出' }}
              </el-tag>
            </div>
            <div class="tx-detail-row">
              <span class="tx-detail-label">账户</span>
              <span>{{ selectedTransaction.account_name || '未设置' }}</span>
            </div>
            <div class="tx-detail-row">
              <span class="tx-detail-label">分类</span>
              <span>{{ selectedTransaction.category_name || '未分类' }}</span>
            </div>
            <div class="tx-detail-row">
              <span class="tx-detail-label">细项</span>
              <span>{{ getTransactionSubLabel(selectedTransaction) || '无' }}</span>
            </div>
            <div class="tx-detail-row">
              <span class="tx-detail-label">金额</span>
              <strong>{{ formatMoney(selectedTransaction.amount) }}</strong>
            </div>
            <div class="tx-detail-row align-start">
              <span class="tx-detail-label">备注</span>
              <div class="tx-detail-content">
                <div>{{ getTransactionMainRemark(selectedTransaction) || '未填写备注' }}</div>
                <div v-if="getTransactionDetailRemark(selectedTransaction)" class="tx-detail-muted">
                  {{ getTransactionDetailRemark(selectedTransaction) }}
                </div>
              </div>
            </div>
            <div class="tx-detail-row align-start" v-if="getTransactionMetaChips(selectedTransaction).length > 0">
              <span class="tx-detail-label">标签</span>
              <div class="tx-meta-chips">
                <el-tag
                  v-for="chip in getTransactionMetaChips(selectedTransaction)"
                  :key="chip"
                  size="small"
                  effect="plain"
                >
                  {{ chip }}
                </el-tag>
              </div>
            </div>
            <div class="tx-detail-row align-start" v-if="selectedTransaction.voice_text">
              <span class="tx-detail-label">语音原文</span>
              <div class="tx-detail-content tx-detail-muted">{{ selectedTransaction.voice_text }}</div>
            </div>
          </div>
        </template>
      </el-dialog>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getECharts } from '../../utils/vendor-loaders'

const API_BASE = '/api'
const expenseSessionKeys = {
  profileId: 'expense_profile_id',
  password: 'expense_user_password',
  creatorKey: 'expense_creator_key'
}

function saveExpenseSession() {
  sessionStorage.setItem(expenseSessionKeys.profileId, profileId.value)
  sessionStorage.setItem(expenseSessionKeys.password, password.value)
  if (creatorKey.value) {
    sessionStorage.setItem(expenseSessionKeys.creatorKey, creatorKey.value)
  } else {
    sessionStorage.removeItem(expenseSessionKeys.creatorKey)
  }
}

function clearExpenseSession() {
  Object.values(expenseSessionKeys).forEach(key => {
    sessionStorage.removeItem(key)
    localStorage.removeItem(key)
  })
  localStorage.removeItem('expense_password')
}

// 带密码认证的请求封装
async function expenseFetch(url, options = {}) {
  const headers = {
    'X-Password': password.value,
    ...options.headers
  }
  if (!headers['Content-Type'] && options.body && !(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }
  return fetch(url, { ...options, headers })
}

async function expenseCreatorFetch(url, options = {}) {
  const headers = {
    'X-Creator-Key': creatorKey.value,
    ...options.headers
  }
  if (!headers['Content-Type'] && options.body && !(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }
  return fetch(url, { ...options, headers })
}

// State
const profileId = ref('')
const password = ref('')
const creatorKey = ref('')
const saveStatus = ref('saved')

// Forms
const createForm = ref({ password: '' })
const loadForm = ref({ password: '' })
const creating = ref(false)
const loading = ref(false)

// Data
const accounts = ref([])
const categories = ref([])
const transactions = ref([])
const calendarTransactions = ref([])
const stats = ref({ total_income: 0, total_expense: 0, balance: 0, by_category: {}, by_account: {}, by_month: {} })

// UI State
const activeTab = ref('record')

const tabItems = [
  { name: 'record', label: '记账', icon: '📝' },
  { name: 'history', label: '历史', icon: '📋' },
  { name: 'stats', label: '统计', icon: '📊' },
  { name: 'calendar', label: '日历', icon: '📅' },
  { name: 'backup', label: '备份', icon: '💾' },
  { name: 'analyze', label: 'AI', icon: '🤖' },
  { name: 'settings', label: '设置', icon: '⚙' }
]

async function switchTab(tabName) {
  if (activeTab.value === tabName) return
  activeTab.value = tabName
  await onTabChange(tabName)
}
const recordMonthCursor = ref(getShanghaiCurrentMonthCursor())
const statsPeriod = ref('month')
const statsMonthCursor = ref(getShanghaiCurrentMonthCursor())
const analyzePeriod = ref('month')
const statsCustomRange = ref([])
const analyzeCustomRange = ref([])
const loadingTxs = ref(false)
const addingTx = ref(false)
const analyzing = ref(false)
const analysisResult = ref('')
const analysisJobId = ref('')
const analysisStatus = ref('')
let analysisPollTimer = null

// Charts
const categoryChartRef = ref(null)
const monthChartRef = ref(null)
const accountChartRef = ref(null)
let categoryChart = null
let monthChart = null
let accountChart = null

// Form for transaction
const txForm = ref({
  account_id: '',
  category_id: '',
  type: 'expense',
  amount: 0,
  date: '',
  time: '',
  remark: ''
})

// Settings dialogs
const showAddAccount = ref(false)
const showAddCategory = ref(false)
const showExtendDialog = ref(false)
const extendDays = ref(365)

// Edit transaction
const showEditDialog = ref(false)
const savingEdit = ref(false)
const editingTxId = ref('')
const editForm = ref({
  type: 'expense',
  amount: 0,
  account_id: '',
  category_id: '',
  date: '',
  time: '',
  tags: '',
  remark: ''
})

const accountForm = ref({ name: '', type: 'cash', balance: 0, color: '#409EFF' })
const categoryForm = ref({ name: '', type: 'expense', color: '#F56C6C' })

const accountTypeLabels = {
  cash: '现金',
  bank: '银行卡',
  alipay: '支付宝',
  wechat: '微信',
  credit: '信用卡'
}

// Voice input state
const isRecording = ref(false)
const voiceText = ref('')
const voiceAmount = ref(0)
const voiceCategory = ref('')
let recognition = null

// Voice panel state
const showVoicePanel = ref(false)
const voiceMode = ref('voice') // 'voice' or 'manual'
const manualText = ref('')
const parsingManual = ref(false)
const parsedResult = ref(null)

// 语音记账状态
const voiceRecordStatus = ref('') // '', 'processing', 'success', 'failed'
const voiceRecordMessage = ref('')

// Success animation
const showSuccessAnimation = ref(false)

// Today's stats
const todayStats = ref({ income: 0, expense: 0 })

// Calendar state
const calendarYear = ref(getShanghaiCalendarAnchor().year)
const calendarMonth = ref(getShanghaiCalendarAnchor().month)
const calendarDays = ref([])
const dayTransactions = ref([])
const showDayDialog = ref(false)
const selectedDay = ref('')

// Backup state
const backingUp = ref(false)
const importingFromNFS = ref(false)
const nfsImportLink = ref('')
const nfsImportPassword = ref('')
const historyTransactions = ref([])
const historyMonth = ref('')
const historyKeyword = ref('')
const historyTypeFilter = ref('')
const historyCategoryFilter = ref('')
const recordKeyword = ref('')
const recordTypeFilter = ref('')
const recordAccountFilter = ref('')
const recordCategoryFilter = ref('')
const recordPage = ref(1)
const recordPageSize = ref(50)
const statsDetailTransactions = ref([])
const statsDetailLoading = ref(false)
const statsDetailKeyword = ref('')
const statsDetailTypeFilter = ref('')
const statsDetailAccountFilter = ref('')
const statsDetailCategoryFilter = ref('')
const analysisFocus = ref('overview')
const showTransactionDetailDialog = ref(false)
const selectedTransaction = ref(null)
const recordLoaded = ref(false)
const historyLoaded = ref(false)
const statsLoaded = ref(false)
const calendarLoaded = ref(false)

// Computed
const categorizedCategories = computed(() => {
  const expense = categories.value.filter(c => c.type === 'expense')
  const income = categories.value.filter(c => c.type === 'income')
  return [
    { type: 'expense', label: '支出', categories: expense },
    { type: 'income', label: '收入', categories: income }
  ]
})

// 快捷分类 - 取前8个
const quickCategories = computed(() => {
  const cats = categories.value.filter(c => c.type === txForm.value.type)
  return cats.slice(0, 8).map(cat => ({
    ...cat,
    icon: getCategoryIcon(cat.name)
  }))
})

// 当前选中的账户
const currentAccount = computed(() => {
  return accounts.value.find(a => a.id === txForm.value.account_id)
})

const amountInput = ref(null)

// 支出分类
const expenseCategories = computed(() => {
  return categories.value.filter(c => c.type === 'expense')
})

// 收入分类
const incomeCategories = computed(() => {
  return categories.value.filter(c => c.type === 'income')
})

// 最近使用的备注（从历史记录中提取）
const recentRemarks = computed(() => {
  const remarkSet = new Set()
  transactions.value.forEach(t => {
    const title = getTransactionMainRemark(t)
    if (title && title.length > 0 && title.length < 20) {
      remarkSet.add(title)
    }
  })
  return Array.from(remarkSet).slice(0, 8)
})

const filteredTransactions = computed(() => {
  const keyword = recordKeyword.value.trim().toLowerCase()
  return transactions.value.filter(row => {
    if (recordTypeFilter.value && row.type !== recordTypeFilter.value) return false
    if (recordAccountFilter.value && row.account_id !== recordAccountFilter.value) return false
    if (recordCategoryFilter.value && row.category_id !== recordCategoryFilter.value) return false
    if (keyword) {
      const haystack = [
        row.category_name,
        row.account_name,
        row.remark,
        row.tags,
        row.voice_text,
        getTransactionSubLabel(row)
      ].join(' ').toLowerCase()
      if (!haystack.includes(keyword)) return false
    }
    return true
  })
})

const pagedTransactions = computed(() => {
  const start = (recordPage.value - 1) * recordPageSize.value
  return filteredTransactions.value.slice(start, start + recordPageSize.value)
})

const recordMonthSummary = computed(() => summarizeTransactions(transactions.value))

const recordSummary = computed(() => {
  return summarizeTransactions(filteredTransactions.value)
})

function summarizeTransactions(rows) {
  let income = 0
  let expense = 0
  const byCategory = new Map()
  let maxExpense = null
  rows.forEach(row => {
    if (row.type === 'income') {
      income += Number(row.amount) || 0
      return
    }
    const amount = Number(row.amount) || 0
    expense += amount
    byCategory.set(row.category_name || '未分类', (byCategory.get(row.category_name || '未分类') || 0) + amount)
    if (!maxExpense || amount > maxExpense.amount) {
      maxExpense = row
    }
  })
  let topCategory = ''
  let topAmount = 0
  byCategory.forEach((amount, name) => {
    if (amount > topAmount) {
      topAmount = amount
      topCategory = name
    }
  })
  return { income, expense, topCategory, maxExpense }
}

const calendarTotalsByDate = computed(() => {
  const totals = new Map()
  calendarTransactions.value.forEach(tx => {
    const current = totals.get(tx.date) || { income: 0, expense: 0 }
    if (tx.type === 'income') {
      current.income += Number(tx.amount) || 0
    } else {
      current.expense += Number(tx.amount) || 0
    }
    totals.set(tx.date, current)
  })
  return totals
})

const filteredStatsTransactions = computed(() => {
  const keyword = statsDetailKeyword.value.trim().toLowerCase()
  return statsDetailTransactions.value.filter(row => {
    if (statsDetailTypeFilter.value && row.type !== statsDetailTypeFilter.value) return false
    if (statsDetailAccountFilter.value && row.account_id !== statsDetailAccountFilter.value) return false
    if (statsDetailCategoryFilter.value && row.category_id !== statsDetailCategoryFilter.value) return false
    if (keyword) {
      const haystack = [
        row.category_name,
        row.account_name,
        row.remark,
        row.tags,
        row.voice_text,
        getTransactionSubLabel(row)
      ].join(' ').toLowerCase()
      if (!haystack.includes(keyword)) return false
    }
    return true
  })
})

const historyCategoryOptions = computed(() => {
  if (!historyTypeFilter.value) {
    return categories.value
  }
  return categories.value.filter(cat => cat.type === historyTypeFilter.value)
})

const recordRangeLabel = computed(() => `${recordMonthCursor.value.replace('-', ' 年 ')} 月`)

const statsDetailSummary = computed(() => {
  let expense = 0
  let income = 0
  let expenseCount = 0
  let maxExpense = null
  filteredStatsTransactions.value.forEach(row => {
    const amount = Number(row.amount) || 0
    if (row.type === 'income') {
      income += amount
      return
    }
    expense += amount
    expenseCount++
    if (!maxExpense || amount > maxExpense.amount) {
      maxExpense = row
    }
  })
  return {
    expense,
    income,
    avgExpense: expenseCount > 0 ? expense / expenseCount : 0,
    maxExpense
  }
})

const statsSubcategoryRows = computed(() => aggregateRows(filteredStatsTransactions.value, row => getTransactionSubLabel(row), 'expense'))
const statsDayRows = computed(() => aggregateRows(filteredStatsTransactions.value, row => row.date, 'expense'))
const historyMonthOptions = computed(() => {
  const seen = new Set()
  return historyTransactions.value
    .map(row => String(row.date || '').slice(0, 7))
    .filter(value => {
      if (!value || seen.has(value)) return false
      seen.add(value)
      return true
    })
    .sort((a, b) => b.localeCompare(a))
    .map(value => ({
      value,
      label: value.replace('-', ' 年 ') + ' 月'
    }))
})

const filteredHistoryTransactions = computed(() => {
  const keyword = historyKeyword.value.trim().toLowerCase()
  return historyTransactions.value.filter(row => {
    if (historyMonth.value && !String(row.date || '').startsWith(historyMonth.value)) return false
    if (historyTypeFilter.value && row.type !== historyTypeFilter.value) return false
    if (historyCategoryFilter.value && row.category_id !== historyCategoryFilter.value) return false
    if (keyword) {
      const haystack = [
        row.category_name,
        row.account_name,
        row.remark,
        row.tags,
        row.voice_text,
        getTransactionSubLabel(row)
      ].join(' ').toLowerCase()
      if (!haystack.includes(keyword)) return false
    }
    return true
  })
})

const historySummary = computed(() => {
  return filteredHistoryTransactions.value.reduce((summary, row) => {
    const amount = Number(row.amount) || 0
    if (row.type === 'income') {
      summary.income += amount
    } else {
      summary.expense += amount
    }
    return summary
  }, { income: 0, expense: 0 })
})

const historyGroups = computed(() => {
  const bucket = new Map()
  filteredHistoryTransactions.value.forEach(row => {
    const key = row.date || '未设置日期'
    const current = bucket.get(key) || { date: key, expense: 0, income: 0, items: [] }
    if (row.type === 'income') {
      current.income += Number(row.amount) || 0
    } else {
      current.expense += Number(row.amount) || 0
    }
    current.items.push(row)
    bucket.set(key, current)
  })

  return Array.from(bucket.values())
    .sort((a, b) => String(b.date || '').localeCompare(String(a.date || '')))
    .map(group => ({
      ...group,
      items: group.items.slice().sort((a, b) => {
        const timeA = getTransactionTime(a) || '00:00'
        const timeB = getTransactionTime(b) || '00:00'
        if (timeA === timeB) return (Number(b.amount) || 0) - (Number(a.amount) || 0)
        return timeB.localeCompare(timeA)
      })
    }))
})

const statsRangeLabel = computed(() => {
  const labels = { month: `${statsMonthCursor.value.replace('-', ' 年 ')} 月`, year: '本年', custom: '自定义时间段' }
  if (statsPeriod.value === 'custom' && statsCustomRange.value?.length === 2) {
    return `${statsCustomRange.value[0]} 至 ${statsCustomRange.value[1]}`
  }
  return labels[statsPeriod.value] || '当前时间段'
})

const statsFilterCategoryOptions = computed(() => {
  if (!statsDetailTypeFilter.value) {
    return categories.value
  }
  return categories.value.filter(cat => cat.type === statsDetailTypeFilter.value)
})

const statsCategoryRows = computed(() => {
  const totalExpense = statsDetailSummary.value.expense || 0
  return aggregateRows(filteredStatsTransactions.value, row => row.category_name || '未分类', 'expense')
    .map(row => ({
      ...row,
      share: totalExpense > 0 ? (row.amount / totalExpense) * 100 : 0
    }))
})

const statsTrendRows = computed(() => {
  const bucket = new Map()
  filteredStatsTransactions.value.forEach(row => {
    const key = statsPeriod.value === 'year'
      ? String(row.date || '').slice(0, 7)
      : String(row.date || '')
    if (!key) return
    const current = bucket.get(key) || { name: key, income: 0, expense: 0, count: 0, net: 0 }
    const amount = Number(row.amount) || 0
    if (row.type === 'income') {
      current.income += amount
    } else {
      current.expense += amount
    }
    current.count += 1
    current.net = current.income - current.expense
    bucket.set(key, current)
  })

  return Array.from(bucket.values()).sort((a, b) => a.name.localeCompare(b.name))
})

const statsAccountRows = computed(() => {
  const bucket = new Map()
  filteredStatsTransactions.value.forEach(row => {
    const key = row.account_name || '未设置账户'
    const current = bucket.get(key) || { name: key, income: 0, expense: 0, net: 0, count: 0 }
    const amount = Number(row.amount) || 0
    if (row.type === 'income') {
      current.income += amount
    } else {
      current.expense += amount
    }
    current.count += 1
    current.net = current.income - current.expense
    bucket.set(key, current)
  })

  return Array.from(bucket.values()).sort((a, b) => {
    const flowA = Math.max(a.expense, a.income)
    const flowB = Math.max(b.expense, b.income)
    if (flowB === flowA) return b.count - a.count
    return flowB - flowA
  })
})

const statsTopTransactions = computed(() => {
  return filteredStatsTransactions.value
    .filter(row => row.type === 'expense')
    .slice()
    .sort((a, b) => {
      const amountDiff = (Number(b.amount) || 0) - (Number(a.amount) || 0)
      if (amountDiff !== 0) return amountDiff
      return String(b.date || '').localeCompare(String(a.date || ''))
    })
    .slice(0, 8)
})

const statsOverview = computed(() => {
  const activeDays = new Set(filteredStatsTransactions.value.map(row => row.date).filter(Boolean)).size
  const totalDays = Math.max(activeDays, 1)
  const savingsRate = statsDetailSummary.value.income > 0
    ? (statsDetailSummary.value.income - statsDetailSummary.value.expense) / statsDetailSummary.value.income
    : 0
  return {
    income: statsDetailSummary.value.income,
    expense: statsDetailSummary.value.expense,
    balance: statsDetailSummary.value.income - statsDetailSummary.value.expense,
    activeDays,
    avgPerDay: statsDetailSummary.value.expense / totalDays,
    savingsRate,
    topCategory: statsCategoryRows.value[0] || null,
    topDay: statsDayRows.value[0] || null
  }
})

const statsInsightList = computed(() => {
  const insights = []

  if (statsCategoryRows.value[0]) {
    const top = statsCategoryRows.value[0]
    insights.push(`最大支出重心在「${top.name}」，占筛选后支出的 ${top.share.toFixed(1)}%。`)
  }

  if (statsOverview.value.topDay) {
    insights.push(`支出峰值出现在 ${statsOverview.value.topDay.name}，当天共 ${statsOverview.value.topDay.count} 笔，支出 ${formatMoney(statsOverview.value.topDay.amount)}。`)
  }

  if (statsDetailSummary.value.maxExpense) {
    const tx = statsDetailSummary.value.maxExpense
    insights.push(`最大单笔是 ${tx.date} 的「${tx.category_name || '未分类'}」，金额 ${formatMoney(tx.amount)}。`)
  }

  if (statsSubcategoryRows.value[0]) {
    const sub = statsSubcategoryRows.value[0]
    insights.push(`最常出现的细项是「${sub.name}」，累计 ${sub.count} 笔，金额 ${formatMoney(sub.amount)}。`)
  }

  return insights.slice(0, 4)
})

// Generate calendar days
function generateCalendarDays() {
  const year = calendarYear.value
  const month = calendarMonth.value
  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)
  const startDay = firstDay.getDay()
  const daysInMonth = lastDay.getDate()

  const todayStr = getShanghaiTodayDate()

  const days = []

  // Previous month days
  const prevMonth = new Date(year, month, 0)
  const prevMonthDays = prevMonth.getDate()
  for (let i = startDay - 1; i >= 0; i--) {
    const d = prevMonthDays - i
    const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(d).padStart(2, '0')}`
    days.push({
      date: d,
      fullDate: dateStr,
      currentMonth: false,
      isToday: false,
      income: 0,
      expense: 0
    })
  }

  // Current month days
  for (let d = 1; d <= daysInMonth; d++) {
    const dateStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`
    const dayData = calendarTotalsByDate.value.get(dateStr) || { income: 0, expense: 0 }
    const income = dayData.income
    const expense = dayData.expense

    days.push({
      date: d,
      fullDate: dateStr,
      currentMonth: true,
      isToday: dateStr === todayStr,
      income,
      expense
    })
  }

  // Next month days
  const remaining = 42 - days.length
  for (let d = 1; d <= remaining; d++) {
    const dateStr = `${year}-${String(month + 2).padStart(2, '0')}-${String(d).padStart(2, '0')}`
    days.push({
      date: d,
      fullDate: dateStr,
      currentMonth: false,
      isToday: false,
      income: 0,
      expense: 0
    })
  }

  calendarDays.value = days
}

function prevMonth() {
  if (calendarMonth.value === 0) {
    calendarMonth.value = 11
    calendarYear.value--
  } else {
    calendarMonth.value--
  }
  generateCalendarDays()
  if (profileId.value && password.value && activeTab.value === 'calendar') {
    void loadCalendarTransactions()
  }
}

function nextMonth() {
  if (calendarMonth.value === 11) {
    calendarMonth.value = 0
    calendarYear.value++
  } else {
    calendarMonth.value++
  }
  generateCalendarDays()
  if (profileId.value && password.value && activeTab.value === 'calendar') {
    void loadCalendarTransactions()
  }
}

function goToToday() {
  const today = getShanghaiCalendarAnchor()
  calendarYear.value = today.year
  calendarMonth.value = today.month
  generateCalendarDays()
  if (profileId.value && password.value && activeTab.value === 'calendar') {
    void loadCalendarTransactions()
  }
}

function showDayDetail(day) {
  selectedDay.value = day.fullDate
  dayTransactions.value = calendarTransactions.value.filter(t => t.date === day.fullDate)
  showDayDialog.value = true
}

function duplicateTransaction(row) {
  // Copy to form
  txForm.value.account_id = row.account_id
  txForm.value.category_id = row.category_id
  txForm.value.type = row.type
  txForm.value.amount = row.amount
  txForm.value.remark = row.remark || ''
  const now = getShanghaiNowParts()
  txForm.value.date = now.date
  txForm.value.time = now.time

  // Switch to record tab
  activeTab.value = 'record'

  ElMessage.success('已复制到表单，请确认后记账')
}

// Backup functions
async function backupData(format = 'json') {
  backingUp.value = true
  try {
    // Get all data
    const accountsRes = await expenseFetch(`${API_BASE}/expense/${profileId.value}/accounts`)
    const accountsData = await accountsRes.json()

    const categoriesRes = await expenseFetch(`${API_BASE}/expense/${profileId.value}/categories`)
    const categoriesData = await categoriesRes.json()

    const transactionsRes = await expenseFetch(`${API_BASE}/expense/${profileId.value}/transactions`)
    const transactionsData = await transactionsRes.json()

    const dateStr = getShanghaiTodayDate()

    if (format === 'csv') {
      // Export as CSV
      const headers = ['日期', '类型', '分类', '账户', '金额', '备注', '标签']
      const rows = transactionsData.map(tx => [
        tx.date,
        tx.type === 'income' ? '收入' : '支出',
        tx.category_name || '',
        tx.account_name || '',
        tx.amount,
        tx.remark || '',
        tx.tags || ''
      ])

      const csvContent = [
        '\ufeff' + headers.join(','),
        ...rows.map(row => row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(','))
      ].join('\n')

      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `记账记录_${dateStr}.csv`
      link.click()
      URL.revokeObjectURL(url)
    } else {
      // Export as JSON
      const backup = {
        version: 1,
        exportTime: new Date().toISOString(),
        profileId: profileId.value,
        accounts: accountsData,
        categories: categoriesData,
        transactions: transactionsData
      }

      const blob = new Blob([JSON.stringify(backup, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `记账备份_${dateStr}.json`
      link.click()
      URL.revokeObjectURL(url)
    }

    ElMessage.success(format === 'csv' ? 'CSV 导出成功' : '备份导出成功')
  } catch (e) {
    console.error('Backup error:', e)
    ElMessage.error('备份失败')
  } finally {
    backingUp.value = false
  }
}

async function handleRestore(file) {
  const reader = new FileReader()
  reader.onload = async (e) => {
    try {
      const data = JSON.parse(e.target.result)
      const imported = await restoreBackupData(data)
      ElMessage.success(`导入成功，共恢复 ${imported} 条记录`)
    } catch (err) {
      console.error('Restore error:', err)
      ElMessage.error('恢复失败，请检查文件格式')
    }
  }
  reader.readAsText(file)
  return false
}

async function restoreBackupData(data) {
  if (!data.version || !data.accounts || !data.categories || !data.transactions) {
    throw new Error('invalid backup data')
  }

  await ElMessageBox.confirm(
    '导入备份会向当前档案追加数据。建议在空白档案中恢复，确定继续吗？',
    '警告',
    { type: 'warning' }
  )

  const accountMap = new Map(accounts.value.map(acc => [acc.name, acc]))
  for (const acc of data.accounts) {
    if (accountMap.has(acc.name)) {
      continue
    }
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/accounts`, {
      method: 'POST',
      body: JSON.stringify({
        name: acc.name,
        type: acc.type,
        balance: acc.balance,
        color: acc.color,
        icon: acc.icon
      })
    })
    if (res.ok) {
      const created = await res.json()
      accountMap.set(created.name, created)
    }
  }

  const categoryMap = new Map(categories.value.map(cat => [`${cat.type}:${cat.name}`, cat]))
  for (const cat of data.categories) {
    const key = `${cat.type}:${cat.name}`
    if (categoryMap.has(key)) {
      continue
    }
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/categories`, {
      method: 'POST',
      body: JSON.stringify({
        name: cat.name,
        type: cat.type,
        color: cat.color,
        icon: cat.icon
      })
    })
    if (res.ok) {
      const created = await res.json()
      categoryMap.set(`${created.type}:${created.name}`, created)
    }
  }

  let imported = 0
  for (const tx of data.transactions) {
    const acc = accountMap.get(tx.account_name)
    const cat = categoryMap.get(`${tx.type}:${tx.category_name}`)

    if (acc && cat) {
      const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/transactions`, {
        method: 'POST',
        body: JSON.stringify({
          account_id: acc.id,
          category_id: cat.id,
          amount: tx.amount,
          type: tx.type,
          date: tx.date,
          remark: tx.remark,
          tags: tx.tags || '',
          voice_text: tx.voice_text || ''
        })
      })
      if (res.ok) {
        imported++
      }
    }
  }

  await loadData()
  return imported
}

// Methods
function formatMoney(amount) {
  return new Intl.NumberFormat('zh-CN', { style: 'currency', currency: 'CNY' }).format(amount)
}

function getShanghaiNowParts() {
  const formatter = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  })
  const parts = Object.fromEntries(formatter.formatToParts(new Date()).map(part => [part.type, part.value]))
  return {
    date: `${parts.year}-${parts.month}-${parts.day}`,
    time: `${parts.hour}:${parts.minute}`
  }
}

function formatDateParts(year, month, day) {
  return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
}

function getShanghaiDateParts(date = new Date()) {
  const formatter = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  })
  const parts = Object.fromEntries(formatter.formatToParts(date).map(part => [part.type, part.value]))
  return {
    year: Number(parts.year),
    month: Number(parts.month),
    day: Number(parts.day),
    date: `${parts.year}-${parts.month}-${parts.day}`,
    monthCursor: `${parts.year}-${parts.month}`
  }
}

function getShanghaiCurrentMonthCursor() {
  return getShanghaiDateParts().monthCursor
}

function getShanghaiTodayDate() {
  return getShanghaiDateParts().date
}

function getShanghaiCalendarAnchor() {
  const parts = getShanghaiDateParts()
  return {
    year: parts.year,
    month: parts.month - 1
  }
}

function resetTxDateTime() {
  const now = getShanghaiNowParts()
  txForm.value.date = now.date
  txForm.value.time = now.time
}

function splitTransactionRemark(row) {
  const raw = String(row?.remark || '').trim()
  if (!raw) {
    return { title: '', detail: '' }
  }
  const parts = raw.split(/\s*\/\s*/)
  if (parts.length <= 1) {
    return { title: raw, detail: '' }
  }
  return {
    title: parts[0].trim(),
    detail: parts.slice(1).join(' / ').trim()
  }
}

function parseTransactionTags(tags) {
  const meta = {}
  String(tags || '').split(',').forEach(item => {
    const trimmed = item.trim()
    if (!trimmed) return
    const idx = trimmed.indexOf(':')
    if (idx === -1) return
    const key = trimmed.slice(0, idx)
    const value = trimmed.slice(idx + 1)
    if (key && value) {
      meta[key] = value
    }
  })
  return meta
}

function upsertTransactionTag(tags, key, value) {
  const entries = String(tags || '')
    .split(',')
    .map(item => item.trim())
    .filter(Boolean)
    .filter(item => !item.startsWith(`${key}:`))

  if (value) {
    entries.push(`${key}:${value}`)
  }

  return entries.join(',')
}

function buildTransactionPayload(formLike) {
  return {
    account_id: formLike.account_id,
    category_id: formLike.category_id,
    type: formLike.type,
    amount: formLike.amount,
    date: formLike.date,
    remark: formLike.remark || '',
    tags: upsertTransactionTag(formLike.tags || '', 'time', formLike.time || '')
  }
}

function getTransactionMainRemark(row) {
  return splitTransactionRemark(row).title
}

function getTransactionDetailRemark(row) {
  return splitTransactionRemark(row).detail
}

function getTransactionTime(row) {
  return parseTransactionTags(row?.tags).time || ''
}

function getTransactionSubLabel(row) {
  const detail = getTransactionDetailRemark(row)
  if (detail) return detail
  const title = getTransactionMainRemark(row)
  if (title) return title
  return row?.category_name || ''
}

function getHistoryItemSummary(row) {
  const parts = []
  if (row?.account_name) {
    parts.push(row.account_name)
  }
  const detail = getTransactionDetailRemark(row) || getTransactionMainRemark(row)
  if (detail && detail !== row?.category_name) {
    parts.push(detail)
  }
  return parts.join(' · ') || '无备注'
}

function getTransactionMetaChips(row) {
  const meta = parseTransactionTags(row?.tags)
  const chips = []
  if (meta.book) chips.push(`账本 ${meta.book}`)
  if (meta.reimburse) chips.push(`报销 ${meta.reimburse}`)
  if (meta.budget) chips.push(`预算 ${meta.budget}`)
  return chips
}

function aggregateRows(rows, labelGetter, type = '') {
  const bucket = new Map()
  rows.forEach(row => {
    if (type && row.type !== type) return
    const name = String(labelGetter(row) || '未标注').trim() || '未标注'
    const current = bucket.get(name) || { name, amount: 0, count: 0 }
    current.amount += Number(row.amount) || 0
    current.count += 1
    bucket.set(name, current)
  })
  return Array.from(bucket.values())
    .sort((a, b) => {
      if (b.amount === a.amount) return b.count - a.count
      return b.amount - a.amount
    })
    .slice(0, 15)
}

function resetRecordFilters() {
  recordKeyword.value = ''
  recordTypeFilter.value = ''
  recordAccountFilter.value = ''
  recordCategoryFilter.value = ''
  recordPage.value = 1
}

function openTransactionDetail(row) {
  selectedTransaction.value = row
  showTransactionDetailDialog.value = true
}

function resolvePeriodRange(period, customRange = []) {
  const today = getShanghaiDateParts()
  if (period === 'custom') {
    const custom = getCustomRangeParams(customRange)
    return custom ? { startDate: custom.start_date, endDate: custom.end_date } : null
  }
  if (period === 'year') {
    return { startDate: `${today.year}-01-01`, endDate: today.date }
  }
  return getMonthCursorRange()
}

function getMonthDateRange(year, month) {
  const normalized = new Date(year, month, 1)
  const normalizedYear = normalized.getFullYear()
  const normalizedMonth = normalized.getMonth() + 1
  const endDay = new Date(normalizedYear, normalizedMonth, 0).getDate()
  return {
    startDate: formatDateParts(normalizedYear, normalizedMonth, 1),
    endDate: formatDateParts(normalizedYear, normalizedMonth, endDay)
  }
}

function getMonthCursorRange(cursor = statsMonthCursor.value) {
  const [year, month] = String(cursor || '').split('-').map(Number)
  if (!year || !month) {
    const current = getShanghaiCalendarAnchor()
    return getMonthDateRange(current.year, current.month)
  }
  return getMonthDateRange(year, month - 1)
}

resetTxDateTime()

function shiftRecordMonth(step) {
  const [year, month] = String(recordMonthCursor.value || '').split('-').map(Number)
  const fallback = getShanghaiCalendarAnchor()
  const next = new Date(year || fallback.year, (month || fallback.month + 1) - 1 + step, 1)
  recordMonthCursor.value = `${next.getFullYear()}-${String(next.getMonth() + 1).padStart(2, '0')}`
  if (profileId.value && password.value && activeTab.value === 'record') {
    void ensureTabData('record', { force: true })
  }
}

function shiftStatsMonth(step) {
  const [year, month] = String(statsMonthCursor.value || '').split('-').map(Number)
  const fallback = getShanghaiCalendarAnchor()
  const next = new Date(year || fallback.year, (month || fallback.month + 1) - 1 + step, 1)
  statsMonthCursor.value = `${next.getFullYear()}-${String(next.getMonth() + 1).padStart(2, '0')}`
  if (profileId.value && password.value && activeTab.value === 'stats') {
    void loadStats()
  }
}

// Voice input methods - 参考血糖模块
function toggleVoice() {
  const API = window.SpeechRecognition || window.webkitSpeechRecognition
  if (!API) {
    ElMessage.error('浏览器不支持语音识别')
    return
  }

  if (isRecording.value) {
    // 停止录音
    recognition?.stop()
    isRecording.value = false
    return
  }

  // 开始录音
  recognition = new API()
  recognition.lang = 'zh-CN'

  recognition.onstart = () => {
    isRecording.value = true
    voiceText.value = ''
    voiceRecordStatus.value = ''
    voiceRecordMessage.value = ''
  }

  recognition.onresult = (e) => {
    let text = ''
    for (let i = 0; i < e.results.length; i++) {
      text += e.results[i][0].transcript
    }
    voiceText.value = text
  }

  recognition.onend = () => {
    isRecording.value = false
  }

  recognition.onerror = () => {
    isRecording.value = false
    ElMessage.warning('语音识别失败，请重试')
  }

  recognition.start()
}

// 停止语音
function stopVoice() {
  recognition?.stop()
  isRecording.value = false
}

// Parse manual text input
async function parseManualText() {
  if (!manualText.value.trim()) {
    ElMessage.warning('请输入记账内容')
    return
  }

  parsingManual.value = true
  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/voice-parse`, {
      method: 'POST',
      body: JSON.stringify({ text: manualText.value })
    })
    const data = await res.json()
    if (data.amount > 0) {
      parsedResult.value = {
        amount: data.amount,
        category: data.category,
        type: data.type,
        remark: data.remark || manualText.value
      }
      // 自动填充到表单
      txForm.value.amount = data.amount
      txForm.value.type = data.type || 'expense'
      if (data.category) {
        const cat = categories.value.find(c => c.name === data.category)
        if (cat) {
          txForm.value.category_id = cat.id
        }
      }
    } else {
      ElMessage.warning('未能识别出金额，请手动输入')
    }
  } catch (e) {
    ElMessage.error('解析失败')
    console.error(e)
  } finally {
    parsingManual.value = false
  }
}

// Confirm and submit parsed record
async function confirmParsedRecord() {
  if (!parsedResult.value) return

  // 设置表单值
  txForm.value.amount = parsedResult.value.amount
  txForm.value.type = parsedResult.value.type
  txForm.value.remark = parsedResult.value.remark || ''

  // 查找分类
  if (parsedResult.value.category) {
    const cat = categories.value.find(c => c.name === parsedResult.value.category)
    if (cat) {
      txForm.value.category_id = cat.id
    }
  }

  // 提交
  await addTransaction()

  // 关闭面板并重置
  showVoicePanel.value = false
  manualText.value = ''
  parsedResult.value = null
  voiceText.value = ''
}

// 语音识别完成后直接记账
async function parseAndSubmitVoice(text) {
  if (!profileId.value) {
    voiceRecordStatus.value = 'failed'
    voiceRecordMessage.value = '请先创建或加载档案'
    return false
  }

  voiceRecordStatus.value = 'processing'
  voiceRecordMessage.value = '正在解析...'
  voiceText.value = text

  if (!profileId.value || !password.value) {
    voiceRecordStatus.value = 'failed'
    voiceRecordMessage.value = '请先创建或加载档案'
    return false
  }

  try {
    // 1. 解析语音
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/voice-parse`, {
      method: 'POST',
      body: JSON.stringify({ text: text })
    })

    if (!res.ok) {
      const errText = await res.text()
      throw new Error('解析请求失败: ' + errText)
    }

    const data = await res.json()
    if (!data.amount || data.amount <= 0) {
      voiceRecordStatus.value = 'failed'
      voiceRecordMessage.value = '未能识别金额，请手动记账'
      return false
    }

    voiceRecordMessage.value = `识别到 ${data.amount} 元，正在提交...`

    // 2. 构建交易数据
    const accountId = txForm.value.account_id || (accounts.value.length > 0 ? accounts.value[0].id : null)

    if (!accountId) {
      voiceRecordStatus.value = 'failed'
      voiceRecordMessage.value = '请先添加账户'
      return false
    }

    const txData = {
      type: data.type || 'expense',
      amount: data.amount,
      remark: data.remark || text,
      voice_text: text, // 保存原始语音输入
      account_id: accountId,
      ...getShanghaiNowParts()
    }

    // 查找分类
    if (data.category) {
      const cat = categories.value.find(c => c.name === data.category)
      if (cat) {
        txData.category_id = cat.id
      }
    }

    // 如果没找到分类，使用默认
    if (!txData.category_id) {
      const defaultCats = categories.value.filter(c => c.type === (data.type || 'expense'))
      if (defaultCats.length > 0) {
        txData.category_id = defaultCats[0].id
      }
    }

    // 3. 提交记账
    const submitRes = await expenseFetch(`${API_BASE}/expense/${profileId.value}/transactions`, {
      method: 'POST',
      body: JSON.stringify({
        ...buildTransactionPayload(txData),
        voice_text: txData.voice_text
      })
    })

    if (!submitRes.ok) {
      const errText = await submitRes.text()
      throw new Error('提交失败: ' + errText)
    }

    await submitRes.json()

    // 4. 成功
    voiceRecordStatus.value = 'success'
    voiceRecordMessage.value = `记账成功！${data.amount}元 (${data.category || '支出'})`

    // 刷新数据
    await refreshLoadedViews()

    // 3秒后清除状态
    setTimeout(() => {
      if (voiceRecordStatus.value === 'success') {
        voiceRecordStatus.value = ''
        voiceRecordMessage.value = ''
      }
    }, 3000)

    return true
  } catch (e) {
    console.error('语音记账错误:', e)
    voiceRecordStatus.value = 'failed'
    voiceRecordMessage.value = '记账失败: ' + (e.message || '未知错误')
    return false
  }
}

async function parseVoiceInput(text) {
  // Try to use AI-powered parsing first
  if (profileId.value && password.value) {
    try {
      const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/voice-parse`, {
        method: 'POST',
        body: JSON.stringify({ text })
      })
      const data = await res.json()
      if (data.amount > 0) {
        voiceAmount.value = data.amount
        voiceCategory.value = data.category || ''

        // Update the parsed result for display in the panel
        parsedResult.value = {
          amount: data.amount,
          category: data.category,
          type: data.type || 'expense',
          remark: data.remark || text
        }

        // Update form
        txForm.value.amount = data.amount
        txForm.value.type = data.type || 'expense'
        txForm.value.remark = data.remark || text

        // Find matching category
        if (data.category) {
          const cat = categories.value.find(c => c.name === data.category)
          if (cat) {
            txForm.value.category_id = cat.id
          }
        }

        // Show confidence indicator
        if (data.confidence && data.confidence < 0.6) {
          ElMessage.warning('识别置信度较低，请确认金额和分类是否正确')
        }
        return
      }
    } catch (e) {
      console.error('Voice parse API error:', e)
    }
  }

  // Fallback to local parsing
  // Parse amount from text
  const amountPatterns = [
    /(\d+(?:\.\d{1,2})?)\s*元/,
    /(\d+(?:\.\d{1,2})?)\s*块/,
    /花?了?\s*(\d+(?:\.\d{1,2})?)/,
    /收?到?\s*(\d+(?:\.\d{1,2})?)/,
    /(\d+)/
  ]

  let amount = 0
  for (const pattern of amountPatterns) {
    const match = text.match(pattern)
    if (match) {
      amount = parseFloat(match[1])
      break
    }
  }

  // Parse category
  const categoryKeywords = {
    '餐饮': ['吃饭', '餐饮', '午餐', '晚餐', '早餐', '外卖', '奶茶', '咖啡', '水果', '零食'],
    '交通': ['交通', '打车', '地铁', '公交', '加油', '停车', '打车', '出租车', '滴滴'],
    '购物': ['购物', '买', '淘宝', '京东', '快递'],
    '居住': ['房租', '水电', '物业', '住房'],
    '医疗': ['医疗', '药', '医院', '看病'],
    '教育': ['教育', '培训', '学费', '书'],
    '娱乐': ['娱乐', '电影', '游戏', '旅游', 'KTV'],
    '工资': ['工资', '发工资', '薪资', '收入'],
    '奖金': ['奖金', '年终奖', '分红'],
  }

  let matchedCategory = ''
  const lowerText = text.toLowerCase()

  for (const [cat, keywords] of Object.entries(categoryKeywords)) {
    for (const keyword of keywords) {
      if (lowerText.includes(keyword)) {
        matchedCategory = cat
        break
      }
    }
    if (matchedCategory) break
  }

  // Determine type (expense or income)
  const incomeKeywords = ['工资', '奖金', '收入', '赚钱', '收款', '收到']
  let isIncome = false
  for (const keyword of incomeKeywords) {
    if (lowerText.includes(keyword)) {
      isIncome = true
      break
    }
  }

  voiceAmount.value = amount

  // Find matching category
  if (matchedCategory) {
    const cat = categories.value.find(c => c.name === matchedCategory)
    if (cat) {
      txForm.value.category_id = cat.id
      txForm.value.type = cat.type
      voiceCategory.value = matchedCategory
    }
  }

  // Auto-fill form
  if (amount > 0) {
    txForm.value.amount = amount
    txForm.value.type = isIncome ? 'income' : 'expense'
    txForm.value.remark = text
  }
}

// 确认语音记账
async function confirmVoiceRecord() {
  if (!voiceText.value.trim()) {
    ElMessage.warning('请先说话')
    return
  }
  voiceRecordStatus.value = 'processing'
  voiceRecordMessage.value = '正在解析...'

  try {
    const result = await parseAndSubmitVoice(voiceText.value)
    if (result) {
      ElMessage.success('记账成功！')
    }
  } finally {
    // 重置状态
    voiceText.value = ''
    voiceRecordStatus.value = ''
    voiceRecordMessage.value = ''
  }
}

// 取消语音记账
function cancelVoiceRecord() {
  voiceText.value = ''
  voiceRecordStatus.value = ''
  voiceRecordMessage.value = ''
  voiceAmount.value = 0
  voiceCategory.value = ''
  parsedResult.value = null
}

function getCategoryIcon(name) {
  const iconMap = {
    '餐饮': '🍜',
    '交通': '🚗',
    '购物': '🛍️',
    '居住': '🏠',
    '医疗': '💊',
    '教育': '📚',
    '娱乐': '🎮',
    '工资': '💰',
    '奖金': '🎁',
    '投资收益': '📈',
    '其他支出': '📝',
    '其他收入': '💵'
  }
  return iconMap[name] || '📌'
}

function handleAccountChange(accountId) {
  txForm.value.account_id = accountId
}

function handleCategorySelect(categoryId) {
  const cat = categories.value.find(c => c.id === categoryId)
  if (cat) {
    txForm.value.category_id = cat.id
    txForm.value.type = cat.type
  }
}

function quickAddTransaction() {
  if (!txForm.value.amount || txForm.value.amount <= 0) {
    ElMessage.warning('请输入金额')
    return
  }
  if (!txForm.value.account_id) {
    ElMessage.warning('请选择账户')
    return
  }
  if (!txForm.value.category_id) {
    ElMessage.warning('请选择分类')
    return
  }
  addTransaction()
}

function getCustomRangeParams(range) {
  if (!Array.isArray(range) || range.length !== 2 || !range[0] || !range[1]) {
    return null
  }
  return {
    start_date: range[0],
    end_date: range[1]
  }
}

function exportData() {
  if (transactions.value.length === 0) {
    ElMessage.warning('没有数据可导出')
    return
  }

  // CSV header
  let csv = '\uFEFF日期,类型,分类,账户,金额,备注\n'

  // Add data rows
  transactions.value.forEach(t => {
    const type = t.type === 'income' ? '收入' : '支出'
    const date = t.date
    const category = t.category_name || ''
    const account = t.account_name || ''
    const amount = t.type === 'income' ? t.amount : -t.amount
    const remark = (t.remark || '').replace(/"/g, '""')
    csv += `${date},${type},${category},${account},${amount},"${remark}"\n`
  })

  // Download
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `记账_${getShanghaiTodayDate()}.csv`
  link.click()
  URL.revokeObjectURL(url)

  ElMessage.success('导出成功')
}

async function handleImport(file) {
  const reader = new FileReader()
  reader.onload = async (e) => {
    try {
      const result = await importCSVText(e.target.result)
      if (result.imported > 0) {
        ElMessage.success(`成功导入 ${result.imported} 条记录${result.skipped > 0 ? `，跳过 ${result.skipped} 条` : ''}`)
      } else {
        ElMessage.warning(`未成功导入记录，跳过 ${result.skipped} 条`)
      }
    } catch (err) {
      console.error('Import error:', err)
      ElMessage.error('导入失败，请检查文件格式')
    }
  }
  reader.readAsText(file)
  return false // Prevent default upload
}

async function importCSVText(text) {
  const lines = String(text || '').split('\n').filter(line => line.trim())
  if (lines.length < 2) {
    throw new Error('empty csv')
  }

  let imported = 0
  let skipped = 0

  for (let i = 1; i < lines.length; i++) {
    const line = lines[i].trim()
    if (!line) continue

    const parts = parseCSVLine(line)
    if (parts.length < 5) {
      skipped++
      continue
    }

    const [date, typeStr, categoryName, accountName, amountStr, remark] = parts
    const amount = Math.abs(parseFloat(amountStr))
    if (isNaN(amount) || amount <= 0) {
      skipped++
      continue
    }

    const type = typeStr.includes('收入') || typeStr.includes('收') ? 'income' : 'expense'

    const category = categories.value.find(c =>
      c.type === type && (c.name === categoryName || c.name.includes(categoryName))
    )
    if (!category) {
      skipped++
      continue
    }

    const account = accounts.value.find(a =>
      a.name === accountName || a.name.includes(accountName)
    )
    if (!account) {
      skipped++
      continue
    }

    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/transactions`, {
      method: 'POST',
      body: JSON.stringify({
        account_id: account.id,
        category_id: category.id,
        amount,
        type,
        date: date || getShanghaiTodayDate(),
        remark: remark || ''
      })
    })

    if (res.ok) {
      imported++
    } else {
      skipped++
    }
  }

  await refreshLoadedViews({ includeCategories: true })
  return { imported, skipped }
}

function parseNFSShareId(link) {
  try {
    const url = new URL(link, location.origin)
    const match = url.pathname.match(/\/nfs\/([a-f0-9]{8})$/i)
    return match ? match[1] : ''
  } catch (err) {
    return ''
  }
}

async function importFromNFSShare() {
  if (!profileId.value || !password.value) {
    ElMessage.warning('请先加载记账档案')
    return
  }

  const shareId = parseNFSShareId(nfsImportLink.value.trim())
  if (!shareId) {
    ElMessage.warning('请输入有效的 NFS 分享链接')
    return
  }

  importingFromNFS.value = true
  try {
    const pwdQuery = nfsImportPassword.value ? `?password=${encodeURIComponent(nfsImportPassword.value)}` : ''

    const infoRes = await fetch(`/api/nfsshare/${shareId}/info`)
    if (!infoRes.ok) {
      throw new Error('无法读取分享信息')
    }
    const info = await infoRes.json()

    const fileRes = await fetch(`/api/nfsshare/${shareId}${pwdQuery}`)
    if (!fileRes.ok) {
      const errText = await fileRes.text()
      throw new Error(errText || '下载分享文件失败')
    }

    const content = await fileRes.text()
    const fileName = String(info.name || '').toLowerCase()

    if (fileName.endsWith('.json')) {
      const imported = await restoreBackupData(JSON.parse(content))
      ElMessage.success(`NFS 导入成功，共恢复 ${imported} 条记录`)
    } else if (fileName.endsWith('.csv')) {
      const result = await importCSVText(content)
      if (result.imported > 0) {
        ElMessage.success(`NFS 导入成功：${result.imported} 条${result.skipped > 0 ? `，跳过 ${result.skipped} 条` : ''}`)
      } else {
        ElMessage.warning(`NFS 文件已读取，但未成功导入记录，跳过 ${result.skipped} 条`)
      }
    } else {
      throw new Error(`暂不支持导入该文件类型：${info.name || '未知文件'}`)
    }
  } catch (err) {
    console.error('NFS import error:', err)
    ElMessage.error(err.message || 'NFS 导入失败')
  } finally {
    importingFromNFS.value = false
  }
}

function parseCSVLine(line) {
  const result = []
  let current = ''
  let inQuotes = false

  for (let i = 0; i < line.length; i++) {
    const char = line[i]
    if (char === '"') {
      inQuotes = !inQuotes
    } else if (char === ',' && !inQuotes) {
      result.push(current.trim())
      current = ''
    } else {
      current += char
    }
  }
  result.push(current.trim())
  return result
}

function getAccountColor(accountId) {
  const acc = accounts.value.find(a => a.id === accountId)
  return acc ? acc.color : '#409EFF'
}

async function createProfile() {
  if (createForm.value.password.length < 4) {
    ElMessage.warning('密码至少需要4个字符')
    return
  }

  creating.value = true
  try {
    const res = await fetch(`${API_BASE}/expense`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: createForm.value.password })
    })
    const data = await res.json()

    if (res.ok) {
      profileId.value = data.id
      creatorKey.value = data.creator_key
      password.value = createForm.value.password
      resetTxDateTime()
      saveExpenseSession()
      await loadData()
      ElMessage.success('档案创建成功')
    } else {
      ElMessage.error(data.error || '创建失败')
    }
  } catch (e) {
    ElMessage.error('创建失败')
  } finally {
    creating.value = false
  }
}

async function loadProfile() {
  if (!loadForm.value.password) {
    ElMessage.warning('请输入密码')
    return
  }

  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/expense/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: loadForm.value.password })
    })
    const data = await res.json()

    if (res.ok) {
      profileId.value = data.id
      password.value = loadForm.value.password
      resetTxDateTime()
      saveExpenseSession()
      await loadData()
      ElMessage.success('登录成功')
    } else {
      ElMessage.error(data.error || '登录失败')
    }
  } catch (e) {
    ElMessage.error('登录失败')
  } finally {
    loading.value = false
  }
}

function logoutProfile() {
  profileId.value = ''
  creatorKey.value = ''
  password.value = ''
  loadForm.value.password = ''
  createForm.value.password = ''
  activeTab.value = 'record'
  clearExpenseSession()
  ElMessage.success('已退出当前档案')
}

async function loadData() {
  resetTabLoadState()
  await loadInitialData()
  if (activeTab.value !== 'record') {
    await ensureTabData(activeTab.value, { force: true })
  }
}

function resetTabLoadState() {
  recordLoaded.value = false
  historyLoaded.value = false
  statsLoaded.value = false
  calendarLoaded.value = false
}

async function loadInitialData() {
  if (!profileId.value || !password.value) {
    return
  }

  await Promise.all([
    loadAccounts(),
    loadCategories(),
    loadTodayStats()
  ])

  await ensureTabData('record', { force: true })
}

async function ensureTabData(tabName = activeTab.value, { force = false } = {}) {
  if (!profileId.value || !password.value) {
    return
  }

  if (tabName === 'record') {
    if (force || !recordLoaded.value) {
      await loadTransactions()
      recordLoaded.value = true
    }
    return
  }

  if (tabName === 'history') {
    if (force || !historyLoaded.value) {
      await loadHistoryTransactions()
      historyLoaded.value = true
    }
    return
  }

  if (tabName === 'stats') {
    if (force || !statsLoaded.value) {
      await loadStats()
      statsLoaded.value = true
    }
    return
  }

  if (tabName === 'calendar') {
    if (force || !calendarLoaded.value) {
      await loadCalendarTransactions()
      calendarLoaded.value = true
    }
    return
  }
}

async function refreshLoadedViews({ includeCategories = false } = {}) {
  const tasks = [
    loadAccounts(),
    loadTodayStats(),
    ensureTabData('record', { force: true })
  ]

  if (includeCategories) {
    tasks.push(loadCategories())
  }
  if (historyLoaded.value) {
    tasks.push(ensureTabData('history', { force: true }))
  }
  if (statsLoaded.value) {
    tasks.push(ensureTabData('stats', { force: true }))
  }
  if (calendarLoaded.value) {
    tasks.push(ensureTabData('calendar', { force: true }))
  }

  await Promise.all(tasks)
}

async function loadAccounts() {
  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/accounts`)
    if (res.ok) {
      accounts.value = await res.json()
      if (accounts.value.length > 0 && !txForm.value.account_id) {
        txForm.value.account_id = accounts.value[0].id
      }
    }
  } catch (e) {
    console.error('Failed to load accounts:', e)
  }
}

async function loadCategories() {
  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/categories`)
    if (res.ok) {
      categories.value = await res.json()
      // Set default category
      const expenseCats = categories.value.filter(c => c.type === 'expense')
      if (expenseCats.length > 0 && !txForm.value.category_id) {
        txForm.value.category_id = expenseCats[0].id
      }
    }
  } catch (e) {
    console.error('Failed to load categories:', e)
  }
}

async function loadTransactions() {
  loadingTxs.value = true
  try {
    const range = getMonthCursorRange(recordMonthCursor.value)
    const url = `${API_BASE}/expense/${profileId.value}/transactions?start_date=${encodeURIComponent(range.startDate)}&end_date=${encodeURIComponent(range.endDate)}`
    const res = await expenseFetch(url)
    if (res.ok) {
      transactions.value = await res.json()
      recordPage.value = 1
    }
  } catch (e) {
    console.error('Failed to load transactions:', e)
  } finally {
    loadingTxs.value = false
  }
}

async function loadHistoryTransactions() {
  if (!profileId.value || !password.value) {
    historyTransactions.value = []
    historyMonth.value = ''
    return
  }

  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/transactions`)
    if (res.ok) {
      const data = await res.json()
      historyTransactions.value = Array.isArray(data)
        ? data.slice().sort((a, b) => {
            if (a.date === b.date) {
              return (getTransactionTime(b) || '').localeCompare(getTransactionTime(a) || '')
            }
            return String(b.date || '').localeCompare(String(a.date || ''))
          })
        : []
    }
  } catch (e) {
    console.error('Failed to load history transactions:', e)
  }
}

async function loadCalendarTransactions() {
  if (!profileId.value || !password.value) {
    calendarTransactions.value = []
    generateCalendarDays()
    return
  }

  const range = getMonthDateRange(calendarYear.value, calendarMonth.value)
  try {
    const url = `${API_BASE}/expense/${profileId.value}/transactions?start_date=${encodeURIComponent(range.startDate)}&end_date=${encodeURIComponent(range.endDate)}`
    const res = await expenseFetch(url)
    if (res.ok) {
      calendarTransactions.value = await res.json()
      generateCalendarDays()
    }
  } catch (e) {
    console.error('Failed to load calendar transactions:', e)
  }
}

async function loadStats() {
  try {
    let url = `${API_BASE}/expense/${profileId.value}/stats?period=${statsPeriod.value}`
    if (statsPeriod.value === 'month') {
      const range = getMonthCursorRange(statsMonthCursor.value)
      url = `${API_BASE}/expense/${profileId.value}/stats?period=custom&start_date=${encodeURIComponent(range.startDate)}&end_date=${encodeURIComponent(range.endDate)}`
    } else if (statsPeriod.value === 'custom') {
      const customRange = getCustomRangeParams(statsCustomRange.value)
      if (!customRange) {
        stats.value = { total_income: 0, total_expense: 0, balance: 0, by_category: {}, by_account: {}, by_month: {}, transaction_count: 0 }
        statsDetailTransactions.value = []
        return
      }
      url += `&start_date=${encodeURIComponent(customRange.start_date)}&end_date=${encodeURIComponent(customRange.end_date)}`
    }
    const res = await expenseFetch(url)
    if (res.ok) {
      const data = await res.json()
      stats.value = data
      await loadStatsDetailTransactions()
      await nextTick()
      void renderCharts()
    } else {
      console.error('Stats API error:', res.status, await res.text())
    }
  } catch (e) {
    console.error('Failed to load stats:', e)
  }
}

async function loadStatsDetailTransactions() {
  if (!profileId.value || !password.value) {
    return
  }
  const range = resolvePeriodRange(statsPeriod.value, statsCustomRange.value)
  if (!range) {
    statsDetailTransactions.value = []
    return
  }

  statsDetailLoading.value = true
  try {
    const url = `${API_BASE}/expense/${profileId.value}/transactions?start_date=${encodeURIComponent(range.startDate)}&end_date=${encodeURIComponent(range.endDate)}`
    const res = await expenseFetch(url)
    if (res.ok) {
      statsDetailTransactions.value = await res.json()
    }
  } catch (e) {
    console.error('Failed to load stats detail transactions:', e)
  } finally {
    statsDetailLoading.value = false
  }
}

// Load today's stats
async function loadTodayStats() {
  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/stats?period=today`)
    if (res.ok) {
      const data = await res.json()
      todayStats.value = { income: data.total_income, expense: data.total_expense }
    }
  } catch (e) {
    console.error('Failed to load today stats:', e)
  }
}

function triggerSuccessAnimation() {
  showSuccessAnimation.value = true
  setTimeout(() => {
    showSuccessAnimation.value = false
  }, 1500)
}

async function addTransaction() {
  if (!txForm.value.account_id || !txForm.value.category_id || !txForm.value.amount || !txForm.value.date) {
    ElMessage.warning('请填写完整信息')
    return
  }

  addingTx.value = true
  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/transactions`, {
      method: 'POST',
      body: JSON.stringify(buildTransactionPayload(txForm.value))
    })

    if (res.ok) {
      triggerSuccessAnimation()
      txForm.value.amount = 0
      txForm.value.remark = ''
      resetTxDateTime()
      await refreshLoadedViews()
    } else {
      const data = await res.json()
      ElMessage.error(data.error || '记账失败')
    }
  } catch (e) {
    ElMessage.error('记账失败')
  } finally {
    addingTx.value = false
  }
}

async function deleteTransaction(id) {
  try {
    await ElMessageBox.confirm('确定要删除这条记录吗？', '提示', { type: 'warning' })
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/transactions/${id}`, {
      method: 'DELETE'
    })
    if (res.ok) {
      ElMessage.success('删除成功')
      await refreshLoadedViews()
    }
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

function openEditDialog(row) {
  editingTxId.value = row.id
  editForm.value = {
    type: row.type,
    amount: row.amount,
    account_id: row.account_id,
    category_id: row.category_id,
    date: row.date,
    time: getTransactionTime(row) || '',
    tags: row.tags || '',
    remark: row.remark || ''
  }
  showEditDialog.value = true
}

async function saveEditTransaction() {
  if (!editForm.value.amount || editForm.value.amount <= 0 || !editForm.value.date) {
    ElMessage.warning('请输入金额')
    return
  }

  savingEdit.value = true
  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/transactions/${editingTxId.value}`, {
      method: 'PUT',
      body: JSON.stringify(buildTransactionPayload(editForm.value))
    })

    if (res.ok) {
      ElMessage.success('修改成功')
      showEditDialog.value = false
      await refreshLoadedViews()
    } else {
      const data = await res.json()
      ElMessage.error(data.error || '修改失败')
    }
  } catch (e) {
    ElMessage.error('修改失败')
  } finally {
    savingEdit.value = false
  }
}

async function addAccount() {
  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/accounts`, {
      method: 'POST',
      body: JSON.stringify(accountForm.value)
    })
    if (res.ok) {
      ElMessage.success('添加成功')
      showAddAccount.value = false
      accountForm.value = { name: '', type: 'cash', balance: 0, color: '#409EFF' }
      await loadAccounts()
    }
  } catch (e) {
    ElMessage.error('添加失败')
  }
}

async function deleteAccount(id) {
  try {
    await ElMessageBox.confirm('确定要删除这个账户吗？', '提示', { type: 'warning' })
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/accounts/${id}`, {
      method: 'DELETE'
    })
    if (res.ok) {
      ElMessage.success('删除成功')
      await loadAccounts()
    }
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

async function addCategory() {
  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/categories`, {
      method: 'POST',
      body: JSON.stringify(categoryForm.value)
    })
    if (res.ok) {
      ElMessage.success('添加成功')
      showAddCategory.value = false
      categoryForm.value = { name: '', type: 'expense', color: '#F56C6C' }
      await loadCategories()
    }
  } catch (e) {
    ElMessage.error('添加失败')
  }
}

async function deleteCategory(id) {
  try {
    await ElMessageBox.confirm('确定要删除这个分类吗？', '提示', { type: 'warning' })
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/categories/${id}`, {
      method: 'DELETE'
    })
    if (res.ok) {
      ElMessage.success('删除成功')
      await loadCategories()
    }
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

async function extendProfile() {
  try {
    const res = await expenseCreatorFetch(`${API_BASE}/expense/${profileId.value}/extend?days=${extendDays.value}`, {
      method: 'PUT'
    })
    if (res.ok) {
      ElMessage.success('延期成功')
      showExtendDialog.value = false
    }
  } catch (e) {
    ElMessage.error('延期失败')
  }
}

async function confirmDelete() {
  try {
    await ElMessageBox.confirm('确定要删除档案吗？此操作不可恢复！', '警告', { type: 'error' })
    const res = await expenseCreatorFetch(`${API_BASE}/expense/${profileId.value}`, {
      method: 'DELETE'
    })
    if (res.ok) {
      ElMessage.success('删除成功')
      clearExpenseSession()
      profileId.value = ''
      creatorKey.value = ''
      password.value = ''
    }
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

async function runAnalyze() {
  if (!profileId.value || !password.value) {
    ElMessage.error('请先登录')
    return
  }

  analyzing.value = true
  analysisResult.value = ''
  analysisStatus.value = ''
  stopAnalyzePolling()
  try {
    const payload = { period: analyzePeriod.value, focus: analysisFocus.value }
    if (analyzePeriod.value === 'custom') {
      const customRange = getCustomRangeParams(analyzeCustomRange.value)
      if (!customRange) {
        ElMessage.warning('请选择完整的开始和结束日期')
        return
      }
      payload.start_date = customRange.start_date
      payload.end_date = customRange.end_date
    }

    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/analyze`, {
      method: 'POST',
      body: JSON.stringify(payload)
    })
    if (res.ok) {
      const data = await res.json()
      analysisJobId.value = data.job_id || ''
      analysisStatus.value = data.status || 'queued'
      if (analysisJobId.value) {
        startAnalyzePolling()
      }
    } else {
      const text = await res.text()
      try {
        const data = JSON.parse(text)
        ElMessage.error(data.error || '分析失败，状态码: ' + res.status)
      } catch {
        ElMessage.error('分析失败: ' + text)
      }
    }
  } catch (e) {
    console.error('Exception:', e)
    ElMessage.error('分析异常: ' + e.message)
  } finally {
    analyzing.value = false
  }
}

function stopAnalyzePolling() {
  if (analysisPollTimer) {
    clearTimeout(analysisPollTimer)
    analysisPollTimer = null
  }
}

function clearAnalyzeState() {
  stopAnalyzePolling()
  analysisJobId.value = ''
  analysisStatus.value = ''
  analysisResult.value = ''
}

function startAnalyzePolling() {
  stopAnalyzePolling()
  void pollAnalyzeResult()
}

async function pollAnalyzeResult() {
  if (!analysisJobId.value || !profileId.value) {
    return
  }

  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/analyze/${analysisJobId.value}`)
    const data = await res.json()
    if (!res.ok) {
      throw new Error(data.error || '获取分析任务失败')
    }

    analysisStatus.value = data.status || ''
    if (data.status === 'completed') {
      analysisResult.value = data.analysis || ''
      if (!data.ai_enabled) {
        ElMessage.info('AI 未启用或暂时不可用，已返回基础分析')
      }
      stopAnalyzePolling()
      return
    }

    if (data.status === 'failed') {
      stopAnalyzePolling()
      ElMessage.error(data.error || 'AI 分析失败')
      return
    }

    analysisPollTimer = setTimeout(() => {
      void pollAnalyzeResult()
    }, 2000)
  } catch (e) {
    stopAnalyzePolling()
    ElMessage.error(e.message || '获取 AI 分析结果失败')
  }
}

// Tab 切换处理
async function onTabChange(tabName) {
  if (profileId.value && password.value) {
    await ensureTabData(tabName)
  }
}

async function renderCharts() {
  if (activeTab.value !== 'stats') {
    return
  }

  const echarts = await getECharts()

  if (categoryChartRef.value) {
    if (categoryChart) {
      categoryChart.dispose()
      categoryChart = null
    }

    if (statsCategoryRows.value.length > 0) {
      const categoryData = statsCategoryRows.value.slice(0, 8)
      categoryChart = echarts.init(categoryChartRef.value)
      categoryChart.setOption({
        grid: { left: 70, right: 20, top: 20, bottom: 20 },
        tooltip: {
          trigger: 'axis',
          axisPointer: { type: 'shadow' },
          formatter: params => {
            const item = params?.[0]
            return item ? `${item.name}<br/>支出 ${formatMoney(item.value)}` : ''
          }
        },
        xAxis: { type: 'value', axisLabel: { formatter: value => `¥${value}` } },
        yAxis: { type: 'category', data: categoryData.map(item => item.name), inverse: true },
        series: [{
          type: 'bar',
          data: categoryData.map(item => Number(item.amount.toFixed(2))),
          barWidth: 18,
          itemStyle: {
            color: '#f97316',
            borderRadius: [0, 10, 10, 0]
          },
          label: {
            show: true,
            position: 'right',
            formatter: params => `${statsCategoryRows.value[params.dataIndex]?.share.toFixed(1) || '0.0'}%`
          }
        }]
      }, true)
    }
  }

  if (monthChartRef.value) {
    if (monthChart) {
      monthChart.dispose()
      monthChart = null
    }

    if (statsTrendRows.value.length > 0) {
      monthChart = echarts.init(monthChartRef.value)
      monthChart.setOption({
        tooltip: { trigger: 'axis' },
        legend: { top: 0 },
        grid: { left: 45, right: 20, top: 40, bottom: 30 },
        xAxis: { type: 'category', data: statsTrendRows.value.map(item => item.name) },
        yAxis: { type: 'value' },
        series: [{
          name: '支出',
          type: 'bar',
          data: statsTrendRows.value.map(item => Number(item.expense.toFixed(2))),
          itemStyle: { color: '#ef4444', borderRadius: [6, 6, 0, 0] }
        }, {
          name: '收入',
          type: 'bar',
          data: statsTrendRows.value.map(item => Number(item.income.toFixed(2))),
          itemStyle: { color: '#22c55e', borderRadius: [6, 6, 0, 0] }
        }, {
          name: '净流',
          type: 'line',
          smooth: true,
          data: statsTrendRows.value.map(item => Number(item.net.toFixed(2))),
          itemStyle: { color: '#2563eb' },
          lineStyle: { width: 3 }
        }]
      }, true)
    }
  }

  if (accountChartRef.value) {
    if (accountChart) {
      accountChart.dispose()
      accountChart = null
    }

    if (statsAccountRows.value.length > 0) {
      const rows = statsAccountRows.value.slice(0, 8)
      accountChart = echarts.init(accountChartRef.value)
      accountChart.setOption({
        tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
        legend: { top: 0 },
        grid: { left: 50, right: 20, top: 40, bottom: 30 },
        xAxis: { type: 'category', data: rows.map(item => item.name) },
        yAxis: { type: 'value' },
        series: [{
          name: '支出',
          type: 'bar',
          data: rows.map(item => Number(item.expense.toFixed(2))),
          itemStyle: { color: '#fb7185', borderRadius: [6, 6, 0, 0] }
        }, {
          name: '收入',
          type: 'bar',
          data: rows.map(item => Number(item.income.toFixed(2))),
          itemStyle: { color: '#4ade80', borderRadius: [6, 6, 0, 0] }
        }, {
          name: '净流',
          type: 'line',
          smooth: true,
          data: rows.map(item => Number(item.net.toFixed(2))),
          itemStyle: { color: '#0f766e' }
        }]
      }, true)
    }
  }
}

// Check for saved profile
onMounted(() => {
  const savedProfileId = sessionStorage.getItem(expenseSessionKeys.profileId)
  const savedPassword = sessionStorage.getItem(expenseSessionKeys.password)
  const savedCreatorKey = sessionStorage.getItem(expenseSessionKeys.creatorKey)
  localStorage.removeItem(expenseSessionKeys.profileId)
  localStorage.removeItem(expenseSessionKeys.password)
  localStorage.removeItem(expenseSessionKeys.creatorKey)
  localStorage.removeItem('expense_password')

  if (savedProfileId && savedPassword) {
    profileId.value = savedProfileId
    password.value = savedPassword
    creatorKey.value = savedCreatorKey || ''
    loadData()
  }
})

onUnmounted(() => {
  stopAnalyzePolling()
})

// Watch for type change - auto select first category
watch(() => txForm.value.type, (newType) => {
  const cats = categories.value.filter(c => c.type === newType)
  if (cats.length > 0 && !cats.find(c => c.id === txForm.value.category_id)) {
    txForm.value.category_id = cats[0].id
  }
})

// Watch for accounts loaded - auto select first
watch(() => accounts.value, (newAccounts) => {
  if (newAccounts.length > 0 && !txForm.value.account_id) {
    txForm.value.account_id = newAccounts[0].id
  }
}, { immediate: true })

watch([recordKeyword, recordTypeFilter, recordAccountFilter, recordCategoryFilter], () => {
  recordPage.value = 1
})

watch(historyMonthOptions, (options) => {
  const values = options.map(item => item.value)
  if (values.length === 0) {
    historyMonth.value = ''
    return
  }
  if (!historyMonth.value || !values.includes(historyMonth.value)) {
    historyMonth.value = values[0]
  }
}, { immediate: true })

watch(historyTypeFilter, () => {
  if (historyCategoryFilter.value && !historyCategoryOptions.value.find(cat => cat.id === historyCategoryFilter.value)) {
    historyCategoryFilter.value = ''
  }
})

watch(statsDetailTypeFilter, () => {
  if (statsDetailCategoryFilter.value && !statsFilterCategoryOptions.value.find(cat => cat.id === statsDetailCategoryFilter.value)) {
    statsDetailCategoryFilter.value = ''
  }
})

watch([statsDetailKeyword, statsDetailTypeFilter, statsDetailAccountFilter, statsDetailCategoryFilter], () => {
  if (activeTab.value === 'stats' && statsLoaded.value) {
    nextTick(() => {
      void renderCharts()
    })
  }
})

// Watch for window resize
watch(() => activeTab.value, (newTab) => {
  if (newTab === 'stats') {
    nextTick(() => {
      void renderCharts()
    })
  }
  if (newTab === 'calendar') {
    nextTick(() => {
      generateCalendarDays()
    })
  }
})
</script>

<style scoped>
.tool-container {
  padding: 20px;
}

.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.tool-header h2 {
  margin: 0;
}

/* ====== Custom Tab Bar ====== */
.custom-tab-bar {
  display: flex;
  gap: 6px;
  padding: 8px 12px;
  background: #fff;
  border-radius: 14px;
  border: 1px solid #e2e8f0;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
  -ms-overflow-style: none;
  -webkit-overflow-scrolling: touch;
  margin-bottom: 16px;
  white-space: nowrap;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}
.custom-tab-bar::-webkit-scrollbar {
  display: none;
}
.tab-pill {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 8px 14px;
  border-radius: 10px;
  border: 1px solid transparent;
  background: #f8fafc;
  color: #64748b;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
  white-space: nowrap;
}
.tab-pill:hover {
  background: #eff6ff;
  color: #3b82f6;
}
.tab-pill.active {
  background: #3b82f6;
  color: #fff;
  border-color: #3b82f6;
  box-shadow: 0 2px 8px rgba(59,130,246,0.3);
}
.tab-pill-icon {
  font-size: 15px;
  line-height: 1;
}
.tab-pill-label {
  font-size: 13px;
}

.tab-content-wrapper {
  min-height: 200px;
}

/* ====== Balance Overview ====== */

.actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.welcome-section {
  max-width: 500px;
  margin: 40px auto;
}

.welcome-card {
  background: #fff;
  padding: 30px;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.welcome-card h3 {
  margin: 0 0 10px 0;
  text-align: center;
}

.welcome-card p {
  color: #666;
  text-align: center;
  margin-bottom: 20px;
}

.balance-overview {
  display: grid;
  grid-template-columns: 150px repeat(3, 1fr);
  gap: 15px;
  margin-bottom: 20px;
}

.today-stats {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 15px;
  border-radius: 8px;
  color: #fff;
  text-align: center;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.today-label {
  font-size: 14px;
  opacity: 0.9;
  margin-bottom: 8px;
}

.today-values {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
}

.today-income {
  color: #90EE90;
}

.today-expense {
  color: #FFB6C1;
}

.balance-card {
  background: #fff;
  padding: 15px;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  text-align: center;
}

.balance-card.income .balance-value {
  color: #67C23A;
}

.balance-card.expense .balance-value {
  color: #F56C6C;
}

.balance-card.balance .balance-value.negative {
  color: #F56C6C;
}

/* Success animation */
.success-animation {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  background: rgba(103, 194, 58, 0.95);
  color: white;
  padding: 30px 50px;
  border-radius: 12px;
  text-align: center;
  z-index: 9999;
  box-shadow: 0 10px 40px rgba(103, 194, 58, 0.4);
}

.success-icon {
  font-size: 48px;
  margin-bottom: 10px;
}

.success-text {
  font-size: 20px;
  font-weight: bold;
}

.slide-fade-enter-active {
  transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.2s cubic-bezier(1, 0.5, 0.8, 1);
}

.slide-fade-enter-from,
.slide-fade-leave-to {
  transform: translate(-50%, -50%) scale(0.5);
  opacity: 0;
}

.balance-label {
  font-size: 14px;
  color: #666;
  margin-bottom: 10px;
}

.balance-value {
  font-size: 24px;
  font-weight: bold;
}

.record-section {
  padding: 10px 0;
}

.quick-add-form {
  background: #f5f7fa;
  padding: 15px;
  border-radius: 8px;
  margin-bottom: 20px;
}

/* 快捷记账区 */
.quick-entry {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.amount-row {
  display: flex;
  align-items: center;
  gap: 15px;
  margin-bottom: 20px;
}

.type-toggle {
  display: flex;
  background: #f0f0f0;
  border-radius: 25px;
  padding: 4px;
}

.type-toggle button {
  padding: 8px 20px;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 20px;
  font-size: 14px;
  transition: all 0.3s;
}

.type-toggle button.active {
  background: #fff;
  color: #409EFF;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.type-toggle button:first-child.active {
  color: #F56C6C;
}

.type-toggle button:last-child.active {
  color: #67C23A;
}

.amount-input-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  background: #f5f7fa;
  border-radius: 12px;
  padding: 10px 20px;
}

.currency {
  font-size: 28px;
  color: #909399;
  margin-right: 8px;
}

.amount-input {
  flex: 1;
  border: none;
  background: transparent;
  font-size: 36px;
  font-weight: bold;
  color: #303133;
  outline: none;
  width: 100%;
}

.amount-input::placeholder {
  color: #c0c4cc;
}

.category-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  margin-bottom: 20px;
}

.category-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 15px 10px;
  background: #f5f7fa;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
  border: 2px solid transparent;
}

.category-btn:hover {
  background: #ecf5ff;
}

.category-btn.active {
  border-color: var(--cat-color, #409EFF);
  background: #ecf5ff;
}

.cat-icon {
  font-size: 24px;
  margin-bottom: 4px;
}

.cat-name {
  font-size: 12px;
  color: #606266;
}

.remark-row {
  margin-bottom: 12px;
}

.datetime-row {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.datetime-field {
  width: 180px;
  max-width: 100%;
}

.datetime-hint {
  font-size: 12px;
  color: #64748b;
}

.remark-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}

.remark-tag {
  cursor: pointer;
}

.remark-tag:hover {
  background: #409EFF;
  color: #fff;
}

.more-categories {
  margin-bottom: 12px;
}

.category-dropdown {
  max-height: 300px;
  overflow-y: auto;
}

.action-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.account-selector {
  flex: 1;
}

.submit-btn {
  flex: 2;
  height: 44px;
  font-size: 16px;
}

.voice-result {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
  padding: 10px;
  background: #e6f7ff;
  border-radius: 4px;
  animation: pulse 1.5s infinite;
}

/* Voice Panel */
.voice-panel {
  margin-top: 15px;
  padding: 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 12px;
  color: white;
}

.voice-panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
  font-size: 18px;
  font-weight: bold;
}

.voice-tabs {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}

.voice-tabs button {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px;
  background: rgba(255,255,255,0.2);
  border: none;
  border-radius: 8px;
  color: white;
  font-size: 16px;
  cursor: pointer;
  transition: all 0.3s;
}

.voice-tabs button.active {
  background: white;
  color: #667eea;
}

/* Voice Mode */
.voice-mode {
  text-align: center;
  padding: 20px 0;
}

/* 录音中 */
.recording-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 20px;
}

.recording-pulse {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  background: #f56c6c;
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.2); opacity: 0.7; }
  100% { transform: scale(1); opacity: 1; }
}

.voice-wave {
  width: 120px;
  height: 120px;
  margin: 0 auto;
  background: rgba(255,255,255,0.3);
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.3s;
}

.voice-wave:hover {
  background: rgba(255,255,255,0.5);
  transform: scale(1.05);
}

.voice-wave.listening {
  background: #f56c6c;
  animation: wave 1s infinite;
}

@keyframes wave {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.1); }
}

.wave-icon {
  color: white;
}

.wave-text {
  color: white;
  font-size: 14px;
  margin-top: 5px;
}

.voice-result-card {
  margin-top: 15px;
  padding: 15px;
  background: rgba(255,255,255,0.9);
  border-radius: 8px;
  color: #333;
}

.result-text {
  font-size: 18px;
  text-align: center;
  margin-bottom: 15px;
}

.voice-confirm-btns {
  display: flex;
  justify-content: center;
  gap: 10px;
}

/* Voice Status */
.voice-status {
  margin-top: 15px;
  padding: 15px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 16px;
}

.voice-status.processing {
  background: #e6f7ff;
  color: #1890ff;
}

.voice-status.success {
  background: #f6ffed;
  color: #52c41a;
}

.voice-status.failed {
  background: #fff2f0;
  color: #ff4d4f;
}

/* Manual Mode */
.manual-mode {
  padding: 10px 0;
}

/* Parsed Result */
.parsed-result {
  margin-top: 20px;
  padding: 15px;
  background: rgba(255,255,255,0.95);
  border-radius: 8px;
  color: #333;
}

.parsed-amount, .parsed-category, .parsed-type {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.parsed-amount .label, .parsed-category .label, .parsed-type .label {
  font-weight: bold;
}

.parsed-amount .value {
  font-size: 24px;
  color: #67c23a;
  font-weight: bold;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.tx-list {
  margin-top: 20px;
}

.tx-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  font-weight: bold;
  flex-wrap: wrap;
  gap: 10px;
}

.tx-list-subtitle {
  margin-top: 4px;
  font-size: 12px;
  font-weight: 400;
  color: #909399;
}

.tx-header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.record-month-nav {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-radius: 999px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
}

.record-month-label {
  min-width: 88px;
  text-align: center;
  font-size: 13px;
  font-weight: 600;
  color: #334155;
}

.record-summary-grid,
.stats-deep-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}

.record-summary-card,
.stats-deep-card {
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 14px;
}

.summary-label,
.stats-deep-label {
  font-size: 12px;
  color: #909399;
  margin-bottom: 8px;
}

.summary-value,
.stats-deep-value {
  font-size: 22px;
  font-weight: 700;
  color: #303133;
}

.summary-text,
.stats-deep-text {
  font-size: 14px;
  color: #303133;
  line-height: 1.6;
}

.record-filters,
.stats-detail-filters {
  display: grid;
  grid-template-columns: 2fr repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}

.tx-date-cell,
.tx-category-cell,
.tx-remark-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tx-date-time,
.tx-subcategory,
.tx-remark-detail {
  font-size: 12px;
  color: #909399;
}

.tx-voice-remark {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.tx-meta-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tx-pagination {
  margin-top: 14px;
  display: flex;
  justify-content: flex-end;
}

.text-income {
  color: #67C23A;
  font-weight: bold;
}

.text-expense {
  color: #F56C6C;
  font-weight: bold;
}

.history-section {
  padding: 10px 0;
}

.history-toolbar {
  display: grid;
  grid-template-columns: 180px repeat(2, minmax(0, 1fr)) 2fr;
  gap: 12px;
  margin-bottom: 16px;
}

.history-toolbar-item {
  width: 100%;
}

.history-summary {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}

.history-summary > div {
  padding: 10px 14px;
  border-radius: 999px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  color: #475569;
  font-size: 13px;
}

.history-empty {
  background: #fff;
  border-radius: 16px;
  border: 1px dashed #dbeafe;
  padding: 28px 16px;
}

.history-groups {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.history-day-card {
  background: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
  border: 1px solid #dbeafe;
  border-radius: 18px;
  padding: 16px;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.04);
}

.history-day-header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  margin-bottom: 12px;
}

.history-day-count {
  margin-left: 8px;
  color: #64748b;
  font-size: 12px;
}

.history-day-amounts {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}

.history-item-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.history-item {
  width: 100%;
  border: 0;
  background: #fff;
  border-radius: 14px;
  padding: 12px 14px;
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  text-align: left;
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
  box-shadow: inset 0 0 0 1px #e5e7eb;
}

.history-item:hover {
  transform: translateY(-1px);
  background: #f8fbff;
  box-shadow: inset 0 0 0 1px #bfdbfe, 0 10px 24px rgba(59, 130, 246, 0.08);
}

.history-item-main,
.history-item-side {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.history-item-side {
  align-items: flex-end;
  flex-shrink: 0;
}

.history-item-category {
  font-size: 15px;
  font-weight: 700;
}

.history-item-sub,
.history-item-time {
  color: #64748b;
  font-size: 12px;
}

.stats-section {
  padding: 10px 0;
}

.stats-hero {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  padding: 22px;
  border-radius: 20px;
  background: linear-gradient(135deg, #fff7ed 0%, #eff6ff 100%);
  border: 1px solid #fed7aa;
  margin-bottom: 18px;
}

.stats-kicker {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #c2410c;
  margin-bottom: 8px;
}

.stats-hero h3 {
  margin: 0 0 8px;
  font-size: 24px;
  line-height: 1.2;
  color: #0f172a;
}

.stats-hero p {
  margin: 0;
  max-width: 640px;
  color: #475569;
  line-height: 1.7;
}

.stats-hero-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-content: flex-start;
  justify-content: flex-end;
}

.stats-hero-chip {
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.78);
  border: 1px solid #cbd5e1;
  padding: 10px 14px;
  font-size: 13px;
  color: #334155;
  white-space: nowrap;
}

.stats-control-row {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.stats-range-picker {
  width: 100%;
  max-width: 420px;
}

.stats-month-nav {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.82);
  border: 1px solid #cbd5e1;
}

.stats-month-label {
  min-width: 100px;
  text-align: center;
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
}

.stats-overview-grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 18px;
}

.stats-overview-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 16px;
  padding: 16px;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.04);
}

.stats-overview-label {
  font-size: 12px;
  color: #64748b;
  margin-bottom: 8px;
}

.stats-overview-value {
  font-size: 22px;
  font-weight: 700;
  color: #0f172a;
}

.stats-insight-panel {
  background: #fff;
  border: 1px solid #dbeafe;
  border-radius: 20px;
  padding: 20px;
  margin-bottom: 18px;
}

.stats-insight-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 14px;
}

.stats-insight-header h4 {
  margin: 0 0 6px;
}

.stats-insight-header p {
  margin: 0;
  color: #64748b;
  font-size: 13px;
}

.stats-insight-badge {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 120px;
  padding: 12px 14px;
  border-radius: 14px;
  background: #eff6ff;
  color: #1d4ed8;
  font-size: 12px;
}

.stats-insight-list {
  margin: 0;
  padding-left: 20px;
  color: #334155;
  line-height: 1.8;
}

.stats-chart-grid,
.stats-table-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}

.deep-table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.deep-table-header span {
  font-size: 12px;
  color: #909399;
}

.chart-card {
  background: #fff;
  padding: 20px;
  border-radius: 18px;
  border: 1px solid #e2e8f0;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.04);
}

.chart-card h4 {
  margin: 0 0 15px 0;
}

.stat-card {
  text-align: center;
}

.stat-card .stat-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.stat-card .stat-value {
  font-size: 24px;
  font-weight: bold;
}

.stat-card .stat-value.income {
  color: #67c23a;
}

.stat-card .stat-value.expense {
  color: #f56c6c;
}

.analyze-section {
  padding: 10px 0;
}

.ai-focus-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 18px;
  flex-wrap: wrap;
}

.ai-focus-label {
  color: #606266;
  font-size: 14px;
}

.loading-state {
  text-align: center;
  padding: 40px;
  color: #909399;
}

.loading-state p {
  margin-top: 15px;
  font-size: 14px;
}

.analysis-result {
  background: #f5f7fa;
  padding: 20px;
  border-radius: 8px;
  position: relative;
}

.analysis-content {
  background: #fff;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.ai-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 5px 12px;
  border-radius: 15px;
  font-size: 12px;
  margin-bottom: 15px;
}

.analysis-result pre {
  white-space: pre-wrap;
  word-wrap: break-word;
  margin: 0;
  line-height: 1.8;
}

.settings-section {
  padding: 10px 0;
}

.settings-card {
  background: #fff;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.settings-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.settings-header h4 {
  margin: 0;
}

.tx-detail-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.tx-detail-row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  border-bottom: 1px solid #f1f5f9;
  padding-bottom: 10px;
}

.tx-detail-row.align-start {
  align-items: flex-start;
}

.tx-detail-label {
  min-width: 72px;
  color: #909399;
}

.tx-detail-content {
  flex: 1;
  text-align: right;
}

.tx-detail-muted {
  color: #909399;
  font-size: 12px;
  line-height: 1.6;
}

/* Calendar */
.calendar-section {
  padding: 10px 0;
}

.calendar-header {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20px;
}

.calendar-title {
  font-size: 18px;
  font-weight: bold;
  margin: 0 15px;
}

/* ====== Mobile Transaction Cards ====== */
.mobile-tx-cards {
  display: none;
}
.mobile-tx-card {
  display: flex;
  gap: 12px;
  padding: 14px;
  background: #fff;
  border-radius: 14px;
  border: 1px solid #e2e8f0;
  margin-bottom: 10px;
  cursor: pointer;
  transition: all 0.15s;
  align-items: center;
}
.mobile-tx-card:hover {
  border-color: #bfdbfe;
  box-shadow: 0 4px 12px rgba(59,130,246,0.08);
}
.mtc-left {
  flex-shrink: 0;
}
.mtc-cat-dot {
  display: block;
  width: 12px;
  height: 12px;
  border-radius: 50%;
}
.mtc-center {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.mtc-title {
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mtc-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #64748b;
  flex-wrap: wrap;
}
.mtc-cat-tag {
  font-weight: 600;
}
.mtc-sep {
  color: #cbd5e1;
}
.mtc-right {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  font-size: 16px;
  font-weight: 700;
}
.mtc-actions {
  display: flex;
  gap: 2px;
}

/* ================================================
   MOBILE OPTIMIZATION — three breakpoint tiers
   ================================================ */

/* --- Tier 1: ≤900px — tablets and small desktops --- */
@media (max-width: 900px) {
  .tool-container {
    padding: 12px 10px;
  }
  .tool-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }
  .tool-header .actions {
    flex-wrap: wrap;
    gap: 6px;
  }

  /* Balance */
  .balance-overview {
    grid-template-columns: 1fr 1fr;
  }
  .today-stats {
    grid-column: span 2;
  }

  /* Tab bar - compact */
  .custom-tab-bar {
    padding: 6px 8px;
    gap: 4px;
    border-radius: 12px;
  }
  .tab-pill {
    padding: 6px 10px;
    font-size: 12px;
  }
  .tab-pill-icon {
    font-size: 14px;
  }

  /* Record filters & summary */
  .record-summary-grid,
  .stats-deep-grid {
    grid-template-columns: 1fr 1fr;
  }
  .record-filters,
  .stats-detail-filters,
  .history-toolbar {
    grid-template-columns: 1fr 1fr;
  }

  /* Amount row */
  .amount-row {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
  }
  .amount-input-wrapper {
    justify-content: center;
  }

  /* Category grid - 3 cols on tablet */
  .category-grid {
    grid-template-columns: repeat(3, 1fr);
  }

  /* Stats overview - 3 cols on tablet */
  .stats-overview-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  /* Stats charts - single column */
  .stats-chart-grid,
  .stats-table-grid {
    grid-template-columns: 1fr;
  }

  /* Settings */
  .settings-section .el-row .el-col {
    flex: 0 0 100%;
    max-width: 100%;
    margin-bottom: 16px;
  }

  /* Calendar */
  .calendar-day {
    min-height: 48px;
    padding: 6px 2px;
  }
  .day-number {
    font-size: 12px;
  }
  .day-amount {
    font-size: 10px;
  }

  /* Charts */
  .chart-card .stats-insight-header {
    flex-direction: column;
  }

  /* AI section */
  .ai-focus-row {
    flex-direction: column;
    align-items: flex-start;
  }

  /* Show mobile cards, hide table */
  .mobile-tx-cards {
    display: block;
  }
  .record-section .el-table,
  .tx-pagination .el-pagination {
    /* Still show pagination controls */
  }
  .record-section .el-table {
    display: none;
  }

  /* Date picker */
  .datetime-row {
    flex-direction: column;
    align-items: stretch;
  }
  .datetime-field {
    width: 100%;
  }
  .action-row {
    flex-wrap: wrap;
  }
  .action-row .el-button {
    flex: 1;
    min-width: 100px;
  }
}

/* --- Tier 2: ≤640px — phones in landscape / large phones --- */
@media (max-width: 640px) {
  .tool-container {
    padding: 8px 6px;
  }

  /* Balance */
  .balance-overview {
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }
  .today-stats {
    grid-column: span 2;
  }
  .balance-card {
    padding: 10px;
  }
  .balance-value {
    font-size: 18px;
  }

  /* Tab bar — ultra compact */
  .custom-tab-bar {
    padding: 4px 6px;
    gap: 3px;
    border-radius: 10px;
  }
  .tab-pill {
    padding: 6px 8px;
    gap: 3px;
    font-size: 11px;
    border-radius: 8px;
  }
  .tab-pill-icon {
    font-size: 13px;
  }

  /* All filters & summaries — full width */
  .record-summary-grid,
  .stats-deep-grid,
  .record-filters,
  .stats-detail-filters,
  .history-toolbar {
    grid-template-columns: 1fr;
  }

  /* Quick entry */
  .quick-entry {
    padding: 12px;
  }
  .amount-input {
    font-size: 28px;
  }
  .currency {
    font-size: 22px;
  }
  .type-toggle button {
    padding: 6px 14px;
    font-size: 13px;
  }

  /* Category grid — 3 cols for phones */
  .category-grid {
    grid-template-columns: repeat(3, 1fr);
    gap: 6px;
  }
  .category-btn {
    padding: 10px 6px;
    border-radius: 10px;
  }
  .cat-icon {
    font-size: 20px;
  }
  .cat-name {
    font-size: 11px;
  }

  /* Stats overview — 2 cols */
  .stats-overview-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }
  .stats-overview-card {
    padding: 12px;
  }
  .stats-overview-value {
    font-size: 18px;
  }

  /* Stats hero */
  .stats-hero {
    flex-direction: column;
    padding: 14px;
  }
  .stats-hero h3 {
    font-size: 18px;
  }
  .stats-hero-chips {
    justify-content: flex-start;
  }
  .stats-control-row {
    flex-direction: column;
    align-items: stretch;
  }

  /* History */
  .history-day-card {
    padding: 12px;
  }
  .history-item {
    padding: 10px 12px;
    gap: 8px;
  }
  .history-item-category {
    font-size: 14px;
  }
  .history-day-header,
  .history-item {
    flex-direction: column;
    align-items: flex-start;
  }
  .history-item-side {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    width: 100%;
  }

  /* Pagination — compact */
  .tx-pagination {
    justify-content: center;
  }
  .tx-pagination :deep(.el-pagination) {
    flex-wrap: wrap;
    justify-content: center;
    gap: 4px;
  }
  .tx-pagination :deep(.el-pagination__sizes) {
    display: none;
  }

  /* Dialogs */
  .tx-detail-row {
    flex-direction: column;
  }
  .tx-detail-content {
    text-align: left;
  }

  /* Charts */
  .chart-card {
    padding: 12px;
  }

  /* Backup */
  .backup-section .el-button {
    width: 100%;
    margin: 5px 0;
  }

  /* Voice panel */
  .voice-panel {
    padding: 14px;
  }
  .voice-tabs button {
    padding: 8px;
    font-size: 13px;
  }
  .voice-wave {
    width: 90px;
    height: 90px;
  }

  /* Record list header */
  .tx-list-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  .tx-header-actions {
    width: 100%;
    justify-content: flex-start;
    flex-wrap: wrap;
  }

  /* Calendar */
  .calendar-grid {
    gap: 1px;
  }
  .calendar-weekday {
    padding: 6px 2px;
    font-size: 10px;
  }
  .calendar-day {
    min-height: 42px;
    padding: 4px 1px;
  }
  .day-number {
    font-size: 11px;
    margin-bottom: 2px;
  }
  .day-amount {
    font-size: 9px;
  }

  /* Amount row */
  .amount-row {
    margin-bottom: 12px;
  }
}

/* --- Tier 3: ≤400px — small phones (iPhone SE, etc.) --- */
@media (max-width: 400px) {
  .tool-container {
    padding: 6px 4px;
  }

  .balance-overview {
    grid-template-columns: 1fr;
    gap: 6px;
  }
  .today-stats {
    grid-column: span 1;
  }

  /* Tab bar — icon-only mode */
  .tab-pill-label {
    display: none;
  }
  .tab-pill {
    padding: 8px 10px;
  }
  .tab-pill-icon {
    font-size: 16px;
  }

  /* Category grid — tighter */
  .category-grid {
    grid-template-columns: repeat(3, 1fr);
    gap: 5px;
  }
  .category-btn {
    padding: 8px 4px;
  }
  .cat-icon {
    font-size: 18px;
    margin-bottom: 2px;
  }
  .cat-name {
    font-size: 10px;
  }

  /* Stats overview — single column */
  .stats-overview-grid {
    grid-template-columns: 1fr;
    gap: 6px;
  }

  .quick-entry {
    padding: 10px;
  }
  .amount-input {
    font-size: 24px;
  }
  .action-row .submit-btn {
    font-size: 14px;
    height: 40px;
  }

  /* Mobile tx card tighter */
  .mobile-tx-card {
    padding: 10px 12px;
    gap: 8px;
  }
  .mtc-title {
    font-size: 14px;
  }
  .mtc-right {
    font-size: 14px;
  }

  .history-day-card {
    padding: 10px;
  }
}

/* Calendar (global) */
.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
  background: #eee;
  border-radius: 8px;
  overflow: hidden;
}

.calendar-weekday {
  background: #f5f7fa;
  padding: 10px;
  text-align: center;
  font-weight: bold;
  font-size: 12px;
}

.calendar-day {
  background: #fff;
  padding: 8px 4px;
  min-height: 60px;
  cursor: pointer;
  transition: background 0.2s;
}

.calendar-day:hover {
  background: #ecf5ff;
}

.calendar-day.other-month {
  background: #fafafa;
  color: #ccc;
}

.calendar-day.today {
  background: #ecf5ff;
}

.calendar-day.has-data {
  background: #f0f9ff;
}

.day-number {
  font-size: 14px;
  font-weight: bold;
  margin-bottom: 4px;
}

.day-amount {
  font-size: 11px;
}

.day-expense {
  color: #F56C6C;
}

.day-income {
  color: #67C23A;
}

/* Backup (global) */
.backup-section {
  padding: 10px 0;
  text-align: center;
}

.backup-section .el-button {
  margin: 10px;
}
</style>
