<template>
  <div class="monitor-tool tool-container">
    <!-- 登录 -->
    <div v-if="!authenticated" class="login-wrap">
      <el-card shadow="hover" class="login-card">
        <template #header>
          <div class="card-header">
            <span><el-icon><DataAnalysis /></el-icon> 监控与访客分析</span>
          </div>
        </template>
        <div class="hero">
          <div class="hero-title">HTTP 流量、访客活跃与归档仪表盘</div>
          <div class="hero-subtitle">
            超级管理员登录后可查看请求趋势、状态码分布、热门接口、AI 网关摘要，并管理请求明细的归档和清理。
          </div>
        </div>
        <el-form @submit.prevent="login">
          <el-form-item label="超级管理员密码">
            <el-input
              v-model="passwordInput"
              type="password"
              show-password
              placeholder="与 monitoring / ai_gateway / console 管理密码一致"
              @keyup.enter="login"
            />
          </el-form-item>
          <el-button type="primary" :loading="loggingIn" @click="login">进入</el-button>
        </el-form>
      </el-card>
    </div>

    <!-- 主体 -->
    <div v-else class="main-content">
      <el-card class="filter-card">
        <template #header>
          <div class="card-header">
            <span>时间范围与基础信息</span>
            <div class="header-actions">
              <el-radio-group v-model="rangeKey" size="small" @change="loadOverview">
                <el-radio-button label="1h">1 小时</el-radio-button>
                <el-radio-button label="24h">24 小时</el-radio-button>
                <el-radio-button label="7d">7 天</el-radio-button>
                <el-radio-button label="30d">30 天</el-radio-button>
                <el-radio-button label="90d">90 天</el-radio-button>
              </el-radio-group>
              <el-button size="small" @click="loadAll" :loading="loadingAll">手动刷新</el-button>
              <el-button size="small" @click="logout">退出</el-button>
            </div>
          </div>
        </template>
        <div v-if="overview" class="kpi-grid">
          <div class="kpi">
            <div class="kpi-label">总请求</div>
            <div class="kpi-value">{{ formatNumber(overview.request_count) }}</div>
          </div>
          <div class="kpi">
            <div class="kpi-label">独立 IP</div>
            <div class="kpi-value">{{ formatNumber(overview.unique_ips) }}</div>
          </div>
          <div class="kpi">
            <div class="kpi-label">DAU</div>
            <div class="kpi-value">{{ formatNumber(overview.dau) }}</div>
            <div class="kpi-sub">最近 1 天活跃访客</div>
          </div>
          <div class="kpi">
            <div class="kpi-label">WAU</div>
            <div class="kpi-value">{{ formatNumber(overview.wau) }}</div>
            <div class="kpi-sub">最近 7 天</div>
          </div>
          <div class="kpi">
            <div class="kpi-label">MAU</div>
            <div class="kpi-value">{{ formatNumber(overview.mau) }}</div>
            <div class="kpi-sub">最近 30 天</div>
          </div>
          <div class="kpi">
            <div class="kpi-label">错误率</div>
            <div class="kpi-value" :class="errorRateClass">
              {{ (errorRate * 100).toFixed(2) }}%
            </div>
            <div class="kpi-sub">{{ formatNumber(overview.status_5xx + overview.status_4xx) }} 条 4xx/5xx</div>
          </div>
          <div class="kpi">
            <div class="kpi-label">平均延迟</div>
            <div class="kpi-value">{{ overview.average_latency_ms.toFixed(1) }} ms</div>
            <div class="kpi-sub">P95 ≈ {{ p95Label }} ms</div>
          </div>
          <div class="kpi">
            <div class="kpi-label">最慢请求</div>
            <div class="kpi-value">{{ formatNumber(overview.max_latency_ms) }} ms</div>
          </div>
        </div>
      </el-card>

      <el-tabs v-model="mainTab" type="border-card" class="main-tabs">
        <!-- 总览 -->
        <el-tab-pane name="overview" label="总览">
          <el-row :gutter="16">
            <el-col :lg="16" :md="24">
              <el-card>
                <template #header>
                  <span>请求趋势（按小时）</span>
                </template>
                <div ref="timelineRef" class="chart-canvas" v-loading="loadingOverview"></div>
              </el-card>
            </el-col>
            <el-col :lg="8" :md="24">
              <el-card>
                <template #header>
                  <span>状态码分布</span>
                </template>
                <div ref="statusRef" class="chart-canvas small" v-loading="loadingOverview"></div>
              </el-card>
            </el-col>
          </el-row>

          <el-card class="block-card">
            <template #header>
              <div class="card-header">
                <span>状态码明细</span>
                <span class="small-text">当前范围：{{ rangeKey }}</span>
              </div>
            </template>
            <div v-if="overview" class="status-grid">
              <div class="status-cell ok">
                <div class="status-num">{{ formatNumber(overview.status_2xx) }}</div>
                <div class="status-label">2xx 成功</div>
              </div>
              <div class="status-cell">
                <div class="status-num">{{ formatNumber(overview.status_3xx) }}</div>
                <div class="status-label">3xx 重定向</div>
              </div>
              <div class="status-cell warn">
                <div class="status-num">{{ formatNumber(overview.status_4xx) }}</div>
                <div class="status-label">4xx 客户端</div>
              </div>
              <div class="status-cell danger">
                <div class="status-num">{{ formatNumber(overview.status_5xx) }}</div>
                <div class="status-label">5xx 服务端</div>
              </div>
              <div class="status-cell">
                <div class="status-num">{{ formatBytes(overview.request_bytes) }}</div>
                <div class="status-label">请求体总字节</div>
              </div>
              <div class="status-cell">
                <div class="status-num">{{ formatBytes(overview.response_bytes) }}</div>
                <div class="status-label">响应体总字节</div>
              </div>
            </div>
          </el-card>

          <el-card class="block-card">
            <template #header>
              <div class="card-header">
                <span>热门接口</span>
                <span class="small-text">按请求量排序（合并已归档与实时数据）</span>
              </div>
            </template>
            <el-table :data="endpoints" v-loading="loadingOverview" stripe size="small" max-height="380">
              <el-table-column prop="method" label="方法" width="90" />
              <el-table-column prop="route" label="路由模板" min-width="220" show-overflow-tooltip />
              <el-table-column prop="requests" label="请求" width="90" sortable />
              <el-table-column prop="errors" label="错误" width="80" sortable />
              <el-table-column label="错误率" width="110">
                <template #default="{ row }">{{ (row.error_rate * 100).toFixed(2) }}%</template>
              </el-table-column>
              <el-table-column label="平均延迟" width="120">
                <template #default="{ row }">{{ row.average_latency_ms.toFixed(1) }} ms</template>
              </el-table-column>
              <el-table-column label="P-max" width="100">
                <template #default="{ row }">{{ formatNumber(row.max_latency_ms) }} ms</template>
              </el-table-column>
              <el-table-column label="响应体积" width="120">
                <template #default="{ row }">{{ formatBytes(row.response_bytes) }}</template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-tab-pane>

        <!-- AI 网关 -->
        <el-tab-pane name="ai" label="AI 网关">
          <el-row :gutter="16">
            <el-col :lg="10" :md="24">
              <el-card>
                <template #header>
                  <span>AI 网关汇总</span>
                </template>
                <div v-if="aiSummary" class="kpi-grid compact">
                  <div class="kpi">
                    <div class="kpi-label">网关请求</div>
                    <div class="kpi-value">{{ formatNumber(aiSummary.request_count) }}</div>
                    <div class="kpi-sub">{{ formatNumber(aiSummary.error_count) }} 个错误</div>
                  </div>
                  <div class="kpi">
                    <div class="kpi-label">输入 Token</div>
                    <div class="kpi-value">{{ formatNumber(aiSummary.input_tokens) }}</div>
                  </div>
                  <div class="kpi">
                    <div class="kpi-label">输出 Token</div>
                    <div class="kpi-value">{{ formatNumber(aiSummary.output_tokens) }}</div>
                  </div>
                  <div class="kpi">
                    <div class="kpi-label">总费用</div>
                    <div class="kpi-value">¥ {{ aiSummary.total_cost.toFixed(4) }}</div>
                  </div>
                  <div class="kpi">
                    <div class="kpi-label">平均延迟</div>
                    <div class="kpi-value">{{ aiSummary.average_latency_ms.toFixed(1) }} ms</div>
                  </div>
                </div>
              </el-card>
            </el-col>
            <el-col :lg="14" :md="24">
              <el-card>
                <template #header><span>按模型分布</span></template>
                <el-table :data="aiSummary?.models || []" stripe size="small" max-height="320">
                  <el-table-column prop="model" label="模型" min-width="160" show-overflow-tooltip />
                  <el-table-column prop="provider" label="提供商" min-width="120" show-overflow-tooltip />
                  <el-table-column prop="requests" label="请求" width="90" sortable />
                  <el-table-column prop="errors" label="错误" width="80" sortable />
                  <el-table-column label="Tokens" width="110">
                    <template #default="{ row }">{{ formatNumber(row.total_tokens) }}</template>
                  </el-table-column>
                  <el-table-column label="费用" width="120">
                    <template #default="{ row }">¥ {{ row.total_cost.toFixed(4) }}</template>
                  </el-table-column>
                  <el-table-column label="平均延迟" width="120">
                    <template #default="{ row }">{{ row.average_latency_ms.toFixed(1) }} ms</template>
                  </el-table-column>
                </el-table>
              </el-card>
            </el-col>
          </el-row>

          <el-card class="block-card">
            <template #header>
              <div class="card-header">
                <span>AI 网关请求明细</span>
                <el-button size="small" @click="loadAI">刷新</el-button>
              </div>
            </template>
            <el-table :data="aiLogs" stripe size="small" max-height="380">
              <el-table-column prop="created_at" label="时间" width="170">
                <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
              </el-table-column>
              <el-table-column prop="model" label="模型" min-width="140" show-overflow-tooltip />
              <el-table-column label="Tokens" width="180">
                <template #default="{ row }">
                  in {{ formatNumber(row.input_tokens || 0) }} / out {{ formatNumber(row.output_tokens || 0) }}
                </template>
              </el-table-column>
              <el-table-column label="费用" width="120">
                <template #default="{ row }">¥ {{ Number(row.cost || 0).toFixed(4) }}</template>
              </el-table-column>
              <el-table-column label="延迟" width="100">
                <template #default="{ row }">{{ row.latency_ms }} ms</template>
              </el-table-column>
              <el-table-column prop="status_code" label="状态" width="80" />
              <el-table-column prop="client_ip" label="客户端" width="130" />
            </el-table>
          </el-card>
        </el-tab-pane>

        <!-- 明细 -->
        <el-tab-pane name="logs" label="请求明细">
          <el-card class="block-card">
            <template #header>
              <div class="card-header">
                <span>筛选条件</span>
                <div class="header-actions">
                  <el-button size="small" @click="loadLogs" :loading="loadingLogs">查询</el-button>
                  <el-button size="small" type="warning" @click="confirmArchive" :loading="archiving">手动归档</el-button>
                  <el-button size="small" type="danger" @click="openDeleteDialog">批量删除</el-button>
                </div>
              </div>
            </template>
            <el-form :inline="true" class="log-filter">
              <el-form-item label="状态码">
                <el-select v-model="filters.status" placeholder="全部" clearable style="width: 140px;" @change="onFilterChange">
                  <el-option value="2xx" label="2xx 成功" />
                  <el-option value="3xx" label="3xx 重定向" />
                  <el-option value="4xx" label="4xx 客户端错误" />
                  <el-option value="5xx" label="5xx 服务端错误" />
                </el-select>
              </el-form-item>
              <el-form-item label="关键词">
                <el-input v-model="filters.keyword" placeholder="路径、IP、UA 包含" clearable style="width: 240px;" @keyup.enter="loadLogs" @clear="onFilterChange" />
              </el-form-item>
            </el-form>

            <el-alert
              v-if="logsTotal !== null"
              type="info"
              :closable="false"
              show-icon
              class="hint-alert"
            >
              <template #title>
                命中 {{ formatNumber(logsTotal) }} 条，仅展示当前分页。明细表默认保留 {{ retentionLabel }} 天，超期将按归档策略自动清理。
              </template>
            </el-alert>

            <el-table :data="logs" stripe size="small" max-height="420" v-loading="loadingLogs">
              <el-table-column prop="created_at" label="时间" width="160">
                <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
              </el-table-column>
              <el-table-column prop="method" label="方法" width="80" />
              <el-table-column label="路径 / 路由" min-width="220" show-overflow-tooltip>
                <template #default="{ row }">
                  <div>{{ row.path }}</div>
                  <div class="small-text">{{ row.route !== row.path ? row.route : '' }}</div>
                </template>
              </el-table-column>
              <el-table-column prop="status_code" label="状态" width="80">
                <template #default="{ row }">
                  <el-tag :type="statusTag(row.status_code)" size="small">{{ row.status_code }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="延迟" width="100">
                <template #default="{ row }">{{ row.latency_ms }} ms</template>
              </el-table-column>
              <el-table-column prop="client_ip" label="客户端" width="120" />
              <el-table-column label="入口" width="120">
                <template #default="{ row }">
                  <el-tag :type="row.archived ? 'success' : 'info'" size="small">
                    {{ row.archived ? '已归档' : '实时' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="错误" min-width="160" show-overflow-tooltip>
                <template #default="{ row }">
                  <span v-if="row.error_message" class="error-text">{{ row.error_message }}</span>
                  <span v-else class="muted">—</span>
                </template>
              </el-table-column>
            </el-table>

            <el-pagination
              v-model:current-page="filters.offsetPage"
              :page-size="filters.limit"
              :total="logsTotal || 0"
              layout="prev, pager, next, total, jumper"
              class="pager"
              @current-change="loadLogs"
            />
          </el-card>
        </el-tab-pane>

        <!-- 服务状态 -->
        <el-tab-pane name="sessions" label="会话">
          <el-row :gutter="16">
            <el-col :lg="10" :md="24">
              <el-card>
                <template #header><span>近 24 小时会话流量</span></template>
                <div v-if="sessionSummary" class="kpi-grid compact">
                  <div class="kpi">
                    <div class="kpi-label">WS 会话</div>
                    <div class="kpi-value">{{ formatNumber(sessionSummary.ws_sessions) }}</div>
                    <div class="kpi-sub">握手成功数</div>
                  </div>
                  <div class="kpi">
                    <div class="kpi-label">WS 帧流量</div>
                    <div class="kpi-value">
                      {{ formatNumber((sessionSummary.ws_frames_in || 0) + (sessionSummary.ws_frames_out || 0)) }}
                    </div>
                    <div class="kpi-sub">
                      {{ sessionSummary.ws_frames_in }} 入 / {{ sessionSummary.ws_frames_out }} 出
                    </div>
                  </div>
                  <div class="kpi">
                    <div class="kpi-label">WS 净流量</div>
                    <div class="kpi-value">
                      {{ formatBytes((sessionSummary.ws_bytes_in || 0) + (sessionSummary.ws_bytes_out || 0)) }}
                    </div>
                  </div>
                  <div class="kpi">
                    <div class="kpi-label">CONNECT 隧道</div>
                    <div class="kpi-value">{{ formatNumber(sessionSummary.tunnel_sessions) }}</div>
                    <div class="kpi-sub">已建立的会话</div>
                  </div>
                  <div class="kpi">
                    <div class="kpi-label">隧道流出量</div>
                    <div class="kpi-value">{{ formatBytes(sessionSummary.tunnel_bytes_out) }}</div>
                    <div class="kpi-sub">{{ formatBytes(sessionSummary.tunnel_bytes_in) }} 入向</div>
                  </div>
                </div>
              </el-card>
            </el-col>
            <el-col :lg="14" :md="24">
              <el-card>
                <template #header>
                  <div class="card-header">
                    <span>会话明细</span>
                    <div class="header-actions">
                      <el-radio-group v-model="sessionType" size="small" @change="loadSessions">
                        <el-radio-button label="all">全部</el-radio-button>
                        <el-radio-button label="websocket">WS</el-radio-button>
                        <el-radio-button label="connect_tunnel">CONNECT</el-radio-button>
                      </el-radio-group>
                      <el-button size="small" @click="loadSessions" :loading="loadingSessions">刷新</el-button>
                    </div>
                  </div>
                </template>
                <el-table :data="sessionLogs" stripe size="small" max-height="420" v-loading="loadingSessions">
                  <el-table-column prop="created_at" label="开始时间" width="160">
                    <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
                  </el-table-column>
                  <el-table-column label="类型" width="80">
                    <template #default="{ row }">
                      <el-tag :type="row.session_type === 'websocket' ? 'success' : 'warning'" size="small">
                        {{ row.session_type === 'websocket' ? 'WS' : 'CONNECT' }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="route" label="路由" min-width="220" show-overflow-tooltip />
                  <el-table-column prop="target" label="目标 / 客户端" min-width="160" show-overflow-tooltip />
                  <el-table-column label="时长" width="100">
                    <template #default="{ row }">{{ formatUptime(Math.floor(row.duration_ms / 1000)) }}</template>
                  </el-table-column>
                  <el-table-column label="流量 / 帧" width="170">
                    <template #default="{ row }">
                      <template v-if="row.session_type === 'websocket'">
                        帧 {{ row.frames_in }}/{{ row.frames_out }} · {{ formatBytes((row.bytes_in || 0) + (row.bytes_out || 0)) }}
                      </template>
                      <template v-else>
                        {{ formatBytes(row.bytes_in) }} ↓ / {{ formatBytes(row.bytes_out) }} ↑
                      </template>
                    </template>
                  </el-table-column>
                  <el-table-column prop="close_reason" label="关闭原因" min-width="120" show-overflow-tooltip />
                </el-table>
                <div v-if="sessionTotal > 0" class="small-text" style="margin-top: 8px;">
                  共 {{ formatNumber(sessionTotal) }} 条（当前只显示前 50）
                </div>
              </el-card>
            </el-col>
          </el-row>
        </el-tab-pane>

        <el-tab-pane name="service" label="服务状态">
          <el-row :gutter="16">
            <el-col :lg="14" :md="24">
              <el-card>
                <template #header>
                  <span>运行时信息</span>
                </template>
                <div v-if="service" class="status-grid">
                  <div class="status-cell ok">
                    <div class="status-num">在线</div>
                    <div class="status-label">服务状态</div>
                  </div>
                  <div class="status-cell">
                    <div class="status-num">{{ formatUptime(service.uptime_seconds) }}</div>
                    <div class="status-label">运行时长</div>
                  </div>
                  <div class="status-cell">
                    <div class="status-num">{{ service.detail_retention_days }} 天</div>
                    <div class="status-label">明细保留期</div>
                  </div>
                  <div class="status-cell">
                    <div class="status-num">{{ service.archive_retention_days }} 天</div>
                    <div class="status-label">聚合/活跃保留期</div>
                  </div>
                  <div class="status-cell">
                    <div class="status-num">{{ formatBytes(service.storage?.database_bytes) }}</div>
                    <div class="status-label">数据库体积</div>
                  </div>
                  <div class="status-cell">
                    <div class="status-num">{{ formatNumber(service.storage?.detail_rows) }}</div>
                    <div class="status-label">明细行数</div>
                  </div>
                </div>
              </el-card>
            </el-col>
            <el-col :lg="10" :md="24">
              <el-card>
                <template #header><span>存储明细</span></template>
                <el-descriptions v-if="service" :column="1" border size="small">
                  <el-descriptions-item label="监控开关">
                    {{ service.monitoring_enabled ? '已启用' : '未启用' }}
                  </el-descriptions-item>
                  <el-descriptions-item label="明细已归档">
                    {{ formatNumber(service.storage?.archived_detail_rows) }} 行
                  </el-descriptions-item>
                  <el-descriptions-item label="小时聚合">
                    {{ formatNumber(service.storage?.hourly_rows) }} 行
                  </el-descriptions-item>
                  <el-descriptions-item label="日活跃汇总">
                    {{ formatNumber(service.storage?.daily_visitor_rows) }} 行
                  </el-descriptions-item>
                  <el-descriptions-item label="明细最早时间">
                    {{ service.storage?.oldest_detail_at ? formatTime(service.storage.oldest_detail_at) : '—' }}
                  </el-descriptions-item>
                  <el-descriptions-item label="明细最新时间">
                    {{ service.storage?.newest_detail_at ? formatTime(service.storage.newest_detail_at) : '—' }}
                  </el-descriptions-item>
                  <el-descriptions-item label="上次归档">
                    {{ service.storage?.last_archive_at ? formatTime(service.storage.last_archive_at) : '尚未执行' }}
                  </el-descriptions-item>
                </el-descriptions>
              </el-card>
            </el-col>
          </el-row>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- 删除确认弹窗 -->
    <el-dialog v-model="deleteDialogVisible" title="批量删除已归档明细" width="480px" align-center>
      <el-alert type="warning" :closable="false" show-icon class="dialog-alert">
        <template #title>
          删除前会自动重新归档指定时间之前的数据，确保不影响聚合趋势。
        </template>
      </el-alert>
      <el-form label-position="top" class="delete-form">
        <el-form-item label="截止时间（RFC3339；留空=当前时刻）">
          <el-input v-model="deleteForm.before" placeholder="例如 2026-07-20T00:00:00Z" />
        </el-form-item>
        <el-form-item label="删除范围">
          <el-radio-group v-model="deleteForm.scope">
            <el-radio value="all">全部</el-radio>
            <el-radio value="errors">仅 4xx / 5xx</el-radio>
            <el-radio value="success">仅 2xx / 3xx</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="确认">
          <el-checkbox v-model="deleteForm.confirm">我已了解删除后无法恢复</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="deleteDialogVisible = false">取消</el-button>
        <el-button type="danger" :loading="deleting" :disabled="!deleteForm.confirm" @click="performDelete">执行删除</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { DataAnalysis } from '@element-plus/icons-vue'
import { getECharts } from '../../utils/vendor-loaders'

const API_BASE = '/api/monitor'
const SESSION_KEY = 'monitor_admin_password'

const passwordInput = ref('')
const authenticated = ref(false)
const loggingIn = ref(false)
const loadingAll = ref(false)
const loadingOverview = ref(false)
const loadingLogs = ref(false)
const archiving = ref(false)
const deleting = ref(false)

const rangeKey = ref('24h')
const mainTab = ref('overview')
const sessionType = ref('all')
const sessionLogs = ref([])
const sessionTotal = ref(0)
const sessionSummary = ref(null)
const loadingSessions = ref(false)

const overview = ref(null)
const endpoints = ref([])
const aiSummary = ref(null)
const aiLogs = ref([])
const logs = ref([])
const logsTotal = ref(0)
const service = ref(null)

const filters = ref({
  status: '',
  keyword: '',
  limit: 20,
  offsetPage: 1
})

const deleteDialogVisible = ref(false)
const deleteForm = ref({
  before: '',
  scope: 'all',
  confirm: false
})

const timelineRef = ref(null)
const statusRef = ref(null)
let timelineChart = null
let statusChart = null

function adminPassword() {
  return sessionStorage.getItem(SESSION_KEY) || ''
}

function authHeaders(extra = {}) {
  return {
    'X-Super-Admin-Password': adminPassword(),
    ...extra
  }
}

async function login() {
  if (!passwordInput.value.trim()) {
    ElMessage.warning('请输入密码')
    return
  }
  loggingIn.value = true
  try {
    const res = await fetch(`${API_BASE}/verify?super_admin_password=${encodeURIComponent(passwordInput.value)}`, {
      headers: { 'X-Super-Admin-Password': passwordInput.value }
    })
    if (!res.ok) {
      ElMessage.error('密码错误或未配置')
      return
    }
    sessionStorage.setItem(SESSION_KEY, passwordInput.value)
    authenticated.value = true
    await loadAll()
  } catch (err) {
    ElMessage.error('登录失败：' + err.message)
  } finally {
    loggingIn.value = false
  }
}

function logout() {
  sessionStorage.removeItem(SESSION_KEY)
  authenticated.value = false
  passwordInput.value = ''
  overview.value = null
  endpoints.value = []
  aiSummary.value = null
  aiLogs.value = []
  logs.value = []
  service.value = null
}

async function loadAll() {
  loadingAll.value = true
  try {
    await Promise.all([loadOverview(), loadAI(), loadService(), loadSessions()])
  } finally {
    loadingAll.value = false
  }
}

async function loadSessions() {
  loadingSessions.value = true
  try {
    const params = new URLSearchParams({
      range: rangeKey.value,
      type: sessionType.value === 'all' ? '' : sessionType.value,
      limit: '50',
      offset: '0',
      super_admin_password: adminPassword()
    })
    const res = await fetch(`${API_BASE}/sessions?${params.toString()}`, {
      headers: authHeaders()
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error(data.error || `请求失败 (${res.status})`)
    }
    sessionSummary.value = data.summary || null
    sessionLogs.value = data.logs || []
    sessionTotal.value = data.total || 0
  } catch (err) {
    ElMessage.error('会话读取失败：' + err.message)
  } finally {
    loadingSessions.value = false
  }
}

async function loadOverview() {
  loadingOverview.value = true
  try {
    const res = await fetch(
      `${API_BASE}/overview?range=${rangeKey.value}&limit=12&super_admin_password=${encodeURIComponent(adminPassword())}`,
      { headers: authHeaders() }
    )
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error(data.error || `请求失败 (${res.status})`)
    }
    overview.value = data.overview
    endpoints.value = data.endpoints || []
    await nextTick()
    await renderCharts()
  } catch (err) {
    ElMessage.error('概览读取失败：' + err.message)
  } finally {
    loadingOverview.value = false
  }
}

async function loadAI() {
  try {
    const res = await fetch(
      `${API_BASE}/ai?range=${rangeKey.value}&limit=50&super_admin_password=${encodeURIComponent(adminPassword())}`,
      { headers: authHeaders() }
    )
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error(data.error || `请求失败 (${res.status})`)
    }
    aiSummary.value = data.summary || null
    aiLogs.value = data.logs || []
  } catch (err) {
    ElMessage.error('AI 摘要读取失败：' + err.message)
  }
}

async function loadService() {
  try {
    const res = await fetch(
      `${API_BASE}/service?super_admin_password=${encodeURIComponent(adminPassword())}`,
      { headers: authHeaders() }
    )
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error(data.error || `请求失败 (${res.status})`)
    }
    service.value = data
  } catch (err) {
    ElMessage.error('服务状态读取失败：' + err.message)
  }
}

async function loadLogs() {
  loadingLogs.value = true
  const offset = (filters.value.offsetPage - 1) * filters.value.limit
  const params = new URLSearchParams({
    range: rangeKey.value,
    status: filters.value.status,
    keyword: filters.value.keyword,
    limit: String(filters.value.limit),
    offset: String(offset),
    super_admin_password: adminPassword()
  })
  try {
    const res = await fetch(`${API_BASE}/logs?${params.toString()}`, {
      headers: authHeaders()
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error(data.error || `请求失败 (${res.status})`)
    }
    logs.value = data.logs || []
    logsTotal.value = data.total || 0
  } catch (err) {
    ElMessage.error('明细读取失败：' + err.message)
  } finally {
    loadingLogs.value = false
  }
}

function onFilterChange() {
  filters.value.offsetPage = 1
  loadLogs()
}

async function confirmArchive() {
  try {
    await ElMessageBox.confirm(
      '手动归档会将所有明细汇总到小时/日活表，并按保留策略清理过期数据。是否继续？',
      '手动归档',
      { confirmButtonText: '执行归档', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  archiving.value = true
  try {
    const res = await fetch(
      `${API_BASE}/archive?super_admin_password=${encodeURIComponent(adminPassword())}`,
      { method: 'POST', headers: authHeaders() }
    )
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error(data.error || `归档失败 (${res.status})`)
    }
    const r = data.result || {}
    ElMessage.success(
      `已归档 ${r.archived_details} 条明细，自动清理 ${r.deleted_details} 条明细，` +
      `${r.deleted_hourly} 条聚合、${r.deleted_visitors} 条日活`
    )
    await Promise.all([loadOverview(), loadLogs(), loadService()])
  } catch (err) {
    ElMessage.error('归档失败：' + err.message)
  } finally {
    archiving.value = false
  }
}

function openDeleteDialog() {
  deleteForm.value = { before: '', scope: 'all', confirm: false }
  deleteDialogVisible.value = true
}

async function performDelete() {
  if (!deleteForm.value.confirm) {
    ElMessage.warning('请先勾选确认')
    return
  }
  deleting.value = true
  try {
    const res = await fetch(
      `${API_BASE}/logs?super_admin_password=${encodeURIComponent(adminPassword())}`,
      {
        method: 'DELETE',
        headers: authHeaders({ 'Content-Type': 'application/json' }),
        body: JSON.stringify(deleteForm.value)
      }
    )
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error(data.error || `删除失败 (${res.status})`)
    }
    ElMessage.success(`已归档 ${data.archived} 条，自动清理 ${data.deleted} 条明细`)
    deleteDialogVisible.value = false
    await Promise.all([loadOverview(), loadLogs(), loadService()])
  } catch (err) {
    ElMessage.error('删除失败：' + err.message)
  } finally {
    deleting.value = false
  }
}

async function renderCharts() {
  if (!overview.value) return
  const echarts = await getECharts()
  if (timelineRef.value) {
    if (timelineChart) timelineChart.dispose()
    timelineChart = echarts.init(timelineRef.value)
    const buckets = overview.value.timeline || []
    timelineChart.setOption({
      tooltip: { trigger: 'axis' },
      legend: { top: 0 },
      grid: { left: 60, right: 30, top: 36, bottom: 40 },
      xAxis: { type: 'category', data: buckets.map(b => shortBucket(b.bucket)) },
      yAxis: [
        { type: 'value', name: '请求', position: 'left' },
        { type: 'value', name: '延迟(ms)', position: 'right' }
      ],
      series: [
        {
          name: '请求',
          type: 'line',
          smooth: true,
          areaStyle: { opacity: 0.1 },
          data: buckets.map(b => b.requests)
        },
        {
          name: '2xx',
          type: 'bar',
          stack: 'status',
          itemStyle: { color: '#67c23a' },
          data: buckets.map(b => b.status_2xx)
        },
        {
          name: '3xx',
          type: 'bar',
          stack: 'status',
          itemStyle: { color: '#909399' },
          data: buckets.map(b => b.status_3xx)
        },
        {
          name: '4xx',
          type: 'bar',
          stack: 'status',
          itemStyle: { color: '#e6a23c' },
          data: buckets.map(b => b.status_4xx)
        },
        {
          name: '5xx',
          type: 'bar',
          stack: 'status',
          itemStyle: { color: '#f56c6c' },
          data: buckets.map(b => b.status_5xx)
        },
        {
          name: '平均延迟',
          type: 'line',
          yAxisIndex: 1,
          smooth: true,
          itemStyle: { color: '#409eff' },
          data: buckets.map(b => b.average_latency_ms)
        }
      ]
    })
  }
  if (statusRef.value) {
    if (statusChart) statusChart.dispose()
    statusChart = echarts.init(statusRef.value)
    statusChart.setOption({
      tooltip: { trigger: 'item' },
      legend: { bottom: 0 },
      series: [
        {
          type: 'pie',
          radius: ['45%', '70%'],
          avoidLabelOverlap: false,
          label: { formatter: '{b}: {c}' },
          data: [
            { value: overview.value.status_2xx, name: '2xx', itemStyle: { color: '#67c23a' } },
            { value: overview.value.status_3xx, name: '3xx', itemStyle: { color: '#909399' } },
            { value: overview.value.status_4xx, name: '4xx', itemStyle: { color: '#e6a23c' } },
            { value: overview.value.status_5xx, name: '5xx', itemStyle: { color: '#f56c6c' } }
          ]
        }
      ]
    })
  }
}

function resizeCharts() {
  timelineChart?.resize()
  statusChart?.resize()
}

const errorRate = computed(() => {
  if (!overview.value || !overview.value.request_count) return 0
  return (overview.value.status_4xx + overview.value.status_5xx) / overview.value.request_count
})

const errorRateClass = computed(() => {
  if (errorRate.value >= 0.1) return 'danger'
  if (errorRate.value >= 0.03) return 'warn'
  return 'ok'
})

const p95Label = computed(() => {
  if (!overview.value) return '0'
  const sorted = (endpoints.value || [])
    .filter(e => e.max_latency_ms)
    .map(e => e.max_latency_ms)
    .sort((a, b) => a - b)
  if (!sorted.length) return Math.round(overview.value.max_latency_ms * 0.8).toString()
  const idx = Math.min(sorted.length - 1, Math.floor(sorted.length * 0.95))
  return Math.round(sorted[idx])
})

const retentionLabel = computed(() => service.value?.detail_retention_days ?? 30)

function statusTag(code) {
  if (code >= 500) return 'danger'
  if (code >= 400) return 'warning'
  if (code >= 300) return 'info'
  if (code >= 200) return 'success'
  return ''
}

function formatNumber(value) {
  const n = Number(value || 0)
  if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M'
  if (n >= 1e4) return (n / 1e4).toFixed(1) + 'w'
  return n.toLocaleString('zh-CN')
}

function formatBytes(value) {
  const n = Number(value || 0)
  if (!n) return '0 B'
  if (n >= 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n >= 1024 * 1024) return (n / 1024 / 1024).toFixed(2) + ' MB'
  if (n >= 1024) return (n / 1024).toFixed(2) + ' KB'
  return n + ' B'
}

function formatTime(value) {
  if (!value) return '—'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

function formatUptime(value) {
  const seconds = Number(value || 0)
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}d ${h}h`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

function shortBucket(value) {
  if (!value) return ''
  const date = new Date(value)
  return `${(date.getMonth() + 1).toString().padStart(2, '0')}-${date.getDate().toString().padStart(2, '0')} ${date.getHours().toString().padStart(2, '0')}:00`
}

let resizeHandler = null
let pollTimer = null

onMounted(() => {
  const saved = sessionStorage.getItem(SESSION_KEY)
  if (saved) {
    passwordInput.value = saved
    login().then(() => {
      pollTimer = setInterval(() => {
        if (mainTab.value === 'overview') loadOverview()
        else if (mainTab.value === 'ai') loadAI()
        else if (mainTab.value === 'logs') loadLogs()
        else if (mainTab.value === 'sessions') loadSessions()
        else if (mainTab.value === 'service') loadService()
      }, 60000)
    })
  }
  resizeHandler = () => resizeCharts()
  window.addEventListener('resize', resizeHandler)
})

onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
  if (resizeHandler) window.removeEventListener('resize', resizeHandler)
  if (timelineChart) timelineChart.dispose()
  if (statusChart) statusChart.dispose()
})
</script>

<style scoped>
.monitor-tool.tool-container {
  padding: 16px 20px 40px;
  max-width: 1400px;
  margin: 0 auto;
}

.login-wrap {
  max-width: 460px;
  margin: 60px auto 0;
}

.hero-title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 6px;
}

.hero-subtitle {
  color: #909399;
  font-size: 13px;
  line-height: 1.5;
  margin-bottom: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 500;
}

.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.filter-card {
  margin-bottom: 16px;
}

.main-tabs {
  margin-bottom: 24px;
}

.block-card {
  margin-top: 16px;
}

.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 14px;
}

.kpi-grid.compact .kpi-value {
  font-size: 18px;
}

.kpi {
  background: #f5f7fa;
  border-radius: 8px;
  padding: 14px 16px;
}

.kpi-label {
  font-size: 13px;
  color: #606266;
  margin-bottom: 4px;
}

.kpi-value {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
}

.kpi-value.danger { color: #f56c6c; }
.kpi-value.warn { color: #e6a23c; }
.kpi-value.ok { color: #67c23a; }

.kpi-sub {
  margin-top: 4px;
  font-size: 11px;
  color: #909399;
}

.status-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 12px;
}

.status-cell {
  background: #f5f7fa;
  border-radius: 6px;
  padding: 14px;
  text-align: center;
}

.status-cell.ok { background: #f0f9eb; }
.status-cell.warn { background: #fdf6ec; }
.status-cell.danger { background: #fef0f0; }

.status-num {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.status-label {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.chart-canvas {
  width: 100%;
  height: 320px;
}

.chart-canvas.small {
  height: 320px;
}

.log-filter {
  margin-bottom: 12px;
}

.hint-alert {
  margin-bottom: 12px;
}

.pager {
  margin-top: 14px;
  justify-content: flex-end;
  display: flex;
}

.dialog-alert {
  margin-bottom: 16px;
}

.delete-form .el-form-item {
  margin-bottom: 14px;
}

.small-text {
  font-size: 12px;
  color: #909399;
}

.muted {
  color: #c0c4cc;
}

.error-text {
  color: #f56c6c;
  font-size: 12px;
}
</style>
