<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>生活记账</h2>
      <div class="actions">
        <template v-if="profileId">
          <el-tag :type="saveStatus === 'saving' ? 'warning' : 'success'" size="small">
            {{ saveStatus === 'saving' ? '保存中...' : '已保存' }}
          </el-tag>
          <el-button @click="logoutProfile">
            退出并切换档案
          </el-button>
          <el-button type="warning" @click="showExtendDialog = true">
            <el-icon><Timer /></el-icon>
            延期
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
          <div class="balance-label">{{ periodLabel }}收入</div>
          <div class="balance-value">{{ formatMoney(stats.total_income) }}</div>
        </div>
        <div class="balance-card expense">
          <div class="balance-label">{{ periodLabel }}支出</div>
          <div class="balance-value">{{ formatMoney(stats.total_expense) }}</div>
        </div>
        <div class="balance-card balance">
          <div class="balance-label">{{ periodLabel }}结余</div>
          <div class="balance-value" :class="{ negative: stats.balance < 0 }">{{ formatMoney(stats.balance) }}</div>
        </div>
      </div>

      <!-- 记账成功动画 -->
      <transition name="slide-fade">
        <div v-if="showSuccessAnimation" class="success-animation">
          <div class="success-icon">✓</div>
          <div class="success-text">记账成功</div>
        </div>
      </transition>

      <el-tabs v-model="activeTab" type="border-card" @tab-change="onTabChange">
        <!-- Record Tab -->
        <el-tab-pane label="记账" name="record">
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
                <span>交易记录</span>
                <div class="tx-header-actions">
                  <el-button-group size="small">
                    <el-button :type="periodFilter === 'today' ? 'primary' : 'default'" @click="setPeriod('today')">今天</el-button>
                    <el-button :type="periodFilter === 'yesterday' ? 'primary' : 'default'" @click="setPeriod('yesterday')">昨天</el-button>
                    <el-button :type="periodFilter === 'week' ? 'primary' : 'default'" @click="setPeriod('week')">本周</el-button>
                    <el-button :type="periodFilter === 'month' ? 'primary' : 'default'" @click="setPeriod('month')">本月</el-button>
                  </el-button-group>
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
                </div>
              </div>
              <el-table :data="transactions" stripe style="width: 100%;" max-height="400" v-loading="loadingTxs">
                <el-table-column prop="date" label="日期" width="100" />
                <el-table-column prop="account_name" label="账户" width="80">
                  <template #default="{ row }">
                    <el-tag size="small" :style="{ borderColor: getAccountColor(row.account_id) }">
                      {{ row.account_name }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="category_name" label="分类" width="80">
                  <template #default="{ row }">
                    <span :style="{ color: row.category_color }">{{ row.category_name }}</span>
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
                <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip>
                  <template #default="{ row }">
                    <span v-if="row.voice_text" :title="row.voice_text">
                      <el-icon><Microphone /></el-icon>
                      {{ row.remark || '语音记账' }}
                    </span>
                    <span v-else>{{ row.remark }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="120" fixed="right">
                  <template #default="{ row }">
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
            </div>
          </div>
        </el-tab-pane>

        <!-- Stats Tab -->
        <el-tab-pane label="统计" name="stats">
          <div class="stats-section">
            <el-radio-group v-model="statsPeriod" @change="loadStats" size="small" style="margin-bottom: 20px;">
              <el-radio-button label="week">本周</el-radio-button>
              <el-radio-button label="month">本月</el-radio-button>
              <el-radio-button label="year">本年</el-radio-button>
            </el-radio-group>

            <!-- 统计汇总 -->
            <el-row :gutter="20" style="margin-bottom: 20px;">
              <el-col :span="8">
                <el-card shadow="hover">
                  <div class="stat-card">
                    <div class="stat-label">收入</div>
                    <div class="stat-value income">¥{{ stats.total_income?.toFixed(2) || '0.00' }}</div>
                  </div>
                </el-card>
              </el-col>
              <el-col :span="8">
                <el-card shadow="hover">
                  <div class="stat-card">
                    <div class="stat-label">支出</div>
                    <div class="stat-value expense">¥{{ stats.total_expense?.toFixed(2) || '0.00' }}</div>
                  </div>
                </el-card>
              </el-col>
              <el-col :span="8">
                <el-card shadow="hover">
                  <div class="stat-card">
                    <div class="stat-label">结余</div>
                    <div class="stat-value" :class="stats.balance >= 0 ? 'income' : 'expense'">¥{{ stats.balance?.toFixed(2) || '0.00' }}</div>
                  </div>
                </el-card>
              </el-col>
            </el-row>

            <!-- 支出分类图表 -->
            <el-row :gutter="20">
              <el-col :span="12">
                <div class="chart-card">
                  <h4>支出分类</h4>
                  <template v-if="stats.transaction_count > 0">
                    <div ref="categoryChartRef" style="height: 300px;"></div>
                  </template>
                  <el-empty v-else description="暂无支出记录" :image-size="80" />
                </div>
              </el-col>
              <el-col :span="12">
                <div class="chart-card">
                  <h4>月度趋势</h4>
                  <template v-if="Object.keys(stats.by_month || {}).length > 0">
                    <div ref="monthChartRef" style="height: 300px;"></div>
                  </template>
                  <el-empty v-else description="暂无月度数据" :image-size="80" />
                </div>
              </el-col>
            </el-row>

            <el-row :gutter="20" style="margin-top: 20px;">
              <el-col :span="24">
                <div class="chart-card">
                  <h4>账户分布</h4>
                  <template v-if="Object.keys(stats.by_account || {}).length > 0">
                    <div ref="accountChartRef" style="height: 250px;"></div>
                  </template>
                  <el-empty v-else description="暂无账户数据" :image-size="80" />
                </div>
              </el-col>
            </el-row>
          </div>
        </el-tab-pane>

        <!-- Calendar Tab -->
        <el-tab-pane label="日历" name="calendar">
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
        </el-tab-pane>

        <!-- Backup Tab -->
        <el-tab-pane label="备份" name="backup">
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
              导入备份数据将覆盖现有数据，请谨慎操作
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
          </div>
        </el-tab-pane>

        <!-- AI Analyze Tab -->
        <el-tab-pane label="AI 解读" name="analyze">
          <div class="analyze-section">
            <el-radio-group v-model="analyzePeriod" size="small" style="margin-bottom: 20px;">
              <el-radio-button label="week">本周</el-radio-button>
              <el-radio-button label="month">本月</el-radio-button>
              <el-radio-button label="year">本年</el-radio-button>
            </el-radio-group>

            <div style="margin-bottom: 20px;">
              <el-button type="primary" @click="runAnalyze" :loading="analyzing">
                <el-icon><MagicStick /></el-icon>
                开始 AI 分析
              </el-button>
              <el-button @click="analysisResult = ''" v-if="analysisResult">
                清空结果
              </el-button>
            </div>

            <!-- 分析结果展示 -->
            <div v-if="analyzing" class="loading-state">
              <el-icon class="is-loading" :size="40"><MagicStick /></el-icon>
              <p>AI 正在分析您的消费数据...</p>
            </div>

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
        </el-tab-pane>

        <!-- Settings Tab -->
        <el-tab-pane label="账户/分类" name="settings">
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
        </el-tab-pane>
      </el-tabs>

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

      <!-- Extend Dialog -->
      <el-dialog v-model="showExtendDialog" title="延期档案" width="300px">
        <p>确定要延期档案吗？</p>
        <el-input-number v-model="extendDays" :min="30" :max="730" />
        <span> 天</span>
        <template #footer>
          <el-button @click="showExtendDialog = false">取消</el-button>
          <el-button type="primary" @click="extendProfile">确定</el-button>
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
          <el-form-item label="备注">
            <el-input v-model="editForm.remark" placeholder="备注" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showEditDialog = false">取消</el-button>
          <el-button type="primary" @click="saveEditTransaction" :loading="savingEdit">保存</el-button>
        </template>
      </el-dialog>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getECharts } from '../../utils/vendor-loaders'

const API_BASE = '/api'

// 带密码认证的请求封装
async function expenseFetch(url, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    'X-Password': password.value,
    ...options.headers
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
const stats = ref({ total_income: 0, total_expense: 0, balance: 0, by_category: {}, by_account: {}, by_month: {} })

// UI State
const activeTab = ref('record')
const periodFilter = ref('')
const statsPeriod = ref('month')
const analyzePeriod = ref('month')
const loadingTxs = ref(false)
const addingTx = ref(false)
const analyzing = ref(false)
const analysisResult = ref('')

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
  date: new Date().toISOString().split('T')[0],
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

// Period label
const periodLabel = computed(() => {
  const labels = { week: '本周', month: '本月', year: '本年' }
  return labels[statsPeriod.value] || '本月'
})

// Calendar state
const calendarYear = ref(new Date().getFullYear())
const calendarMonth = ref(new Date().getMonth())
const calendarDays = ref([])
const dayTransactions = ref([])
const showDayDialog = ref(false)
const selectedDay = ref('')

// Backup state
const backingUp = ref(false)

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
    if (t.remark && t.remark.length > 0 && t.remark.length < 20) {
      remarkSet.add(t.remark)
    }
  })
  return Array.from(remarkSet).slice(0, 8)
})

// Generate calendar days
function generateCalendarDays() {
  const year = calendarYear.value
  const month = calendarMonth.value
  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)
  const startDay = firstDay.getDay()
  const daysInMonth = lastDay.getDate()

  const today = new Date()
  const todayStr = today.toISOString().split('T')[0]

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
    const dayData = transactions.value.filter(t => t.date === dateStr)
    const income = dayData.filter(t => t.type === 'income').reduce((sum, t) => sum + t.amount, 0)
    const expense = dayData.filter(t => t.type === 'expense').reduce((sum, t) => sum + t.amount, 0)

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
}

function nextMonth() {
  if (calendarMonth.value === 11) {
    calendarMonth.value = 0
    calendarYear.value++
  } else {
    calendarMonth.value++
  }
  generateCalendarDays()
}

function goToToday() {
  const today = new Date()
  calendarYear.value = today.getFullYear()
  calendarMonth.value = today.getMonth()
  generateCalendarDays()
}

function showDayDetail(day) {
  selectedDay.value = day.fullDate
  dayTransactions.value = transactions.value.filter(t => t.date === day.fullDate)
  showDayDialog.value = true
}

function duplicateTransaction(row) {
  // Copy to form
  txForm.value.account_id = row.account_id
  txForm.value.category_id = row.category_id
  txForm.value.type = row.type
  txForm.value.amount = row.amount
  txForm.value.remark = row.remark || ''
  txForm.value.date = new Date().toISOString().split('T')[0]

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

    const dateStr = new Date().toISOString().split('T')[0]

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

      if (!data.version || !data.accounts || !data.categories || !data.transactions) {
        ElMessage.warning('无效的备份文件')
        return
      }

      await ElMessageBox.confirm(
        '导入备份将覆盖现有数据，确定继续吗？',
        '警告',
        { type: 'warning' }
      )

      // Restore accounts
      for (const acc of data.accounts) {
        await fetch(`${API_BASE}/expense/${profileId.value}/accounts?password=${encodeURIComponent(password.value)}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: acc.name,
            type: acc.type,
            balance: acc.balance,
            color: acc.color,
            icon: acc.icon
          })
        })
      }

      // Restore categories
      for (const cat of data.categories) {
        await fetch(`${API_BASE}/expense/${profileId.value}/categories?password=${encodeURIComponent(password.value)}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: cat.name,
            type: cat.type,
            color: cat.color,
            icon: cat.icon
          })
        })
      }

      // Restore transactions
      let imported = 0
      for (const tx of data.transactions) {
        // Find matching account and category by name
        const acc = accounts.value.find(a => a.name === tx.account_name)
        const cat = categories.value.find(c => c.name === tx.category_name)

        if (acc && cat) {
          await fetch(`${API_BASE}/expense/${profileId.value}/transactions?password=${encodeURIComponent(password.value)}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              account_id: acc.id,
              category_id: cat.id,
              amount: tx.amount,
              type: tx.type,
              date: tx.date,
              remark: tx.remark
            })
          })
          imported++
        }
      }

      await loadData()

      ElMessage.success(`导入成功，共恢复 ${imported} 条记录`)
    } catch (err) {
      console.error('Restore error:', err)
      ElMessage.error('恢复失败，请检查文件格式')
    }
  }
  reader.readAsText(file)
  return false
}

// Methods
function formatMoney(amount) {
  return new Intl.NumberFormat('zh-CN', { style: 'currency', currency: 'CNY' }).format(amount)
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
      date: new Date().toISOString().split('T')[0]
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
      body: JSON.stringify(txData)
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
    await Promise.all([loadTransactions(), loadAccounts(), loadStats(), loadTodayStats()])

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

function setPeriod(period) {
  periodFilter.value = period
  loadTransactions()
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
  link.download = `记账_${new Date().toISOString().split('T')[0]}.csv`
  link.click()
  URL.revokeObjectURL(url)

  ElMessage.success('导出成功')
}

async function handleImport(file) {
  const reader = new FileReader()
  reader.onload = async (e) => {
    try {
      const text = e.target.result
      const lines = text.split('\n').filter(line => line.trim())

      if (lines.length < 2) {
        ElMessage.warning('文件内容为空')
        return
      }

      let imported = 0
      let skipped = 0

      // Skip header, parse data
      for (let i = 1; i < lines.length; i++) {
        const line = lines[i].trim()
        if (!line) continue

        // Simple CSV parsing
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

        // Find category
        const category = categories.value.find(c =>
          c.name === categoryName || c.name.includes(categoryName)
        )
        if (!category) {
          skipped++
          continue
        }

        // Find account
        const account = accounts.value.find(a =>
          a.name === accountName || a.name.includes(accountName)
        )
        if (!account) {
          skipped++
          continue
        }

        // Create transaction
        const res = await fetch(`${API_BASE}/expense/${profileId.value}/transactions?password=${encodeURIComponent(password.value)}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            account_id: account.id,
            category_id: category.id,
            amount,
            type,
            date: date || new Date().toISOString().split('T')[0],
            remark: remark || ''
          })
        })

        if (res.ok) {
          imported++
        } else {
          skipped++
        }
      }

      await Promise.all([loadTransactions(), loadAccounts(), loadStats()])

      if (imported > 0) {
        ElMessage.success(`成功导入 ${imported} 条记录${skipped > 0 ? `，跳过 ${skipped} 条` : ''}`)
      } else {
        ElMessage.warning(`未成功导入记录，跳过 ${skipped} 条`)
      }
    } catch (err) {
      console.error('Import error:', err)
      ElMessage.error('导入失败，请检查文件格式')
    }
  }
  reader.readAsText(file)
  return false // Prevent default upload
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
      localStorage.setItem('expense_profile_id', data.id)
      localStorage.setItem('expense_password', data.creator_key)
      localStorage.setItem('expense_user_password', createForm.value.password)
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
      localStorage.setItem('expense_profile_id', data.id)
      localStorage.setItem('expense_user_password', loadForm.value.password)
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
  localStorage.removeItem('expense_profile_id')
  localStorage.removeItem('expense_user_password')
  ElMessage.success('已退出当前档案')
}

async function loadData() {
  await Promise.all([
    loadAccounts(),
    loadCategories(),
    loadTransactions(),
    loadStats(),
    loadTodayStats()
  ])
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
    let url = `${API_BASE}/expense/${profileId.value}/transactions`
    if (periodFilter.value) {
      const now = new Date()
      const today = now.toISOString().split('T')[0]
      let startDate = ''
      let endDate = ''

      if (periodFilter.value === 'today') {
        startDate = today
        endDate = today
      } else if (periodFilter.value === 'yesterday') {
        const yesterday = new Date(now - 24 * 60 * 60 * 1000).toISOString().split('T')[0]
        startDate = yesterday
        endDate = yesterday
      } else if (periodFilter.value === 'week') {
        startDate = new Date(now - 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0]
      } else if (periodFilter.value === 'month') {
        startDate = new Date(now - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0]
      } else if (periodFilter.value === 'year') {
        startDate = new Date(now - 365 * 24 * 60 * 60 * 1000).toISOString().split('T')[0]
      }

      if (startDate) {
        url += `?start_date=${startDate}`
      }
      if (endDate) {
        url += `&end_date=${endDate}`
      }
    }
    const res = await expenseFetch(url)
    if (res.ok) {
      transactions.value = await res.json()
    }
  } catch (e) {
    console.error('Failed to load transactions:', e)
  } finally {
    loadingTxs.value = false
  }
}

async function loadStats() {
  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/stats?period=${statsPeriod.value}`)
    if (res.ok) {
      const data = await res.json()
      stats.value = data
      await nextTick()
      // 延迟一点确保DOM完全渲染
      setTimeout(() => {
        void renderCharts()
      }, 100)
    } else {
      console.error('Stats API error:', res.status, await res.text())
    }
  } catch (e) {
    console.error('Failed to load stats:', e)
  }
}

// Load today's stats
async function loadTodayStats() {
  try {
    const today = new Date().toISOString().split('T')[0]
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
  if (!txForm.value.account_id || !txForm.value.category_id || !txForm.value.amount) {
    ElMessage.warning('请填写完整信息')
    return
  }

  addingTx.value = true
  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/transactions`, {
      method: 'POST',
      body: JSON.stringify(txForm.value)
    })

    if (res.ok) {
      triggerSuccessAnimation()
      txForm.value.amount = 0
      txForm.value.remark = ''
      await Promise.all([loadTransactions(), loadAccounts(), loadStats(), loadTodayStats()])
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
    const res = await fetch(`${API_BASE}/expense/${profileId.value}/transactions/${id}?password=${encodeURIComponent(password.value)}`, {
      method: 'DELETE'
    })
    if (res.ok) {
      ElMessage.success('删除成功')
      await Promise.all([loadTransactions(), loadAccounts(), loadStats()])
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
    remark: row.remark || ''
  }
  showEditDialog.value = true
}

async function saveEditTransaction() {
  if (!editForm.value.amount || editForm.value.amount <= 0) {
    ElMessage.warning('请输入金额')
    return
  }

  savingEdit.value = true
  try {
    const res = await fetch(`${API_BASE}/expense/${profileId.value}/transactions/${editingTxId.value}?password=${encodeURIComponent(password.value)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(editForm.value)
    })

    if (res.ok) {
      ElMessage.success('修改成功')
      showEditDialog.value = false
      await Promise.all([loadTransactions(), loadAccounts(), loadStats()])
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
    const res = await fetch(`${API_BASE}/expense/${profileId.value}/accounts?password=${encodeURIComponent(password.value)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
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
    const res = await fetch(`${API_BASE}/expense/${profileId.value}/accounts/${id}?password=${encodeURIComponent(password.value)}`, {
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
    const res = await fetch(`${API_BASE}/expense/${profileId.value}/categories?password=${encodeURIComponent(password.value)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
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
    const res = await fetch(`${API_BASE}/expense/${profileId.value}/categories/${id}?password=${encodeURIComponent(password.value)}`, {
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
    const res = await fetch(`${API_BASE}/expense/${profileId.value}/extend?creator_key=${encodeURIComponent(creatorKey.value)}&days=${extendDays.value}`, {
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
    const res = await fetch(`${API_BASE}/expense/${profileId.value}?creator_key=${encodeURIComponent(creatorKey.value)}`, {
      method: 'DELETE'
    })
    if (res.ok) {
      ElMessage.success('删除成功')
      localStorage.removeItem('expense_profile_id')
      localStorage.removeItem('expense_password')
      localStorage.removeItem('expense_user_password')
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
  try {
    const res = await expenseFetch(`${API_BASE}/expense/${profileId.value}/analyze`, {
      method: 'POST',
      body: JSON.stringify({ period: analyzePeriod.value })
    })

    const text = await res.text()

    if (res.ok) {
      const data = JSON.parse(text)
      analysisResult.value = data.analysis
      if (!data.ai_enabled) {
        ElMessage.info('未配置 DeepSeek API，使用基础分析')
      }
    } else {
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

// Tab 切换处理
function onTabChange(tabName) {
  if (tabName === 'stats') {
    // 切换到统计tab时重新加载和渲染图表
    loadStats()
  } else if (tabName === 'analyze') {
    // 切换到AI解读tab时，可以自动运行分析（可选）
    // runAnalyze()
  }
}

async function renderCharts() {
  // 确保DOM已渲染
  if (!stats.value) {
    return
  }

  const echarts = await getECharts()

  // 支出分类饼图
  if (categoryChartRef.value) {
    const byCategory = stats.value.by_category || {}
    const categoryKeys = Object.keys(byCategory)
    // 先销毁旧图表
    if (categoryChart) {
      categoryChart.dispose()
      categoryChart = null
    }

    if (categoryKeys.length > 0) {
      const categoryData = categoryKeys.map(name => ({ name, value: byCategory[name] }))
      categoryChart = echarts.init(categoryChartRef.value)
      categoryChart.setOption({
        tooltip: { trigger: 'item', formatter: '{b}: ¥{c} ({d}%)' },
        legend: { top: 'bottom' },
        series: [{
          type: 'pie',
          radius: ['40%', '70%'],
          data: categoryData,
          label: { show: true, formatter: '{b}: ¥{c}' }
        }]
      }, true) // true = notMerge，完全替换
    }
  }

  // 月度趋势柱状图
  if (monthChartRef.value) {
    const byMonth = stats.value.by_month || {}
    const monthKeys = Object.keys(byMonth)
    if (monthChart) {
      monthChart.dispose()
      monthChart = null
    }

    if (monthKeys.length > 0) {
      const monthData = monthKeys.map(month => ({ month, value: byMonth[month] }))
      monthData.sort((a, b) => a.month.localeCompare(b.month))

      monthChart = echarts.init(monthChartRef.value)
      monthChart.setOption({
        tooltip: { trigger: 'axis', formatter: '{b}: ¥{c}' },
        xAxis: { type: 'category', data: monthData.map(d => d.month) },
        yAxis: { type: 'value' },
        series: [{
          type: 'bar',
          data: monthData.map(d => d.value),
          itemStyle: { color: '#F56C6C' }
        }]
      }, true)
    }
  }

  // 账户分布饼图
  if (accountChartRef.value) {
    const byAccount = stats.value.by_account || {}
    const accountKeys = Object.keys(byAccount)
    if (accountChart) {
      accountChart.dispose()
      accountChart = null
    }

    if (accountKeys.length > 0) {
      const accountData = accountKeys.map(name => ({ name, value: byAccount[name] }))
      accountChart = echarts.init(accountChartRef.value)
      accountChart.setOption({
        tooltip: { trigger: 'item', formatter: '{b}: ¥{c} ({d}%)' },
        legend: { top: 'bottom' },
        series: [{
          type: 'pie',
          radius: '60%',
          data: accountData,
          label: { show: true, formatter: '{b}: ¥{c}' }
        }]
      }, true)
    }
  }
}

// Check for saved profile
onMounted(() => {
  const savedProfileId = localStorage.getItem('expense_profile_id')
  const savedPassword = localStorage.getItem('expense_user_password')

  if (savedProfileId && savedPassword) {
    profileId.value = savedProfileId
    password.value = savedPassword
    creatorKey.value = localStorage.getItem('expense_password') || ''
    loadData()
  }
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

.tx-header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.text-income {
  color: #67C23A;
  font-weight: bold;
}

.text-expense {
  color: #F56C6C;
  font-weight: bold;
}

.stats-section {
  padding: 10px 0;
}

.chart-card {
  background: #fff;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
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

/* Backup */
.backup-section {
  padding: 10px 0;
  text-align: center;
}

.backup-section .el-button {
  margin: 10px;
}

@media (max-width: 768px) {
  .balance-overview {
    grid-template-columns: 1fr 1fr;
  }

  .today-stats {
    grid-column: span 2;
  }

  .quick-add-form :deep(.el-form-item) {
    margin-bottom: 10px;
  }

  .el-col {
    margin-bottom: 20px;
  }
}
</style>
