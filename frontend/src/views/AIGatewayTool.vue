<template>
  <div class="tool-container ai-gateway-page">
    <div class="hero">
      <div>
        <h2>AI Gateway</h2>
        <p>统一发放 API Key，对外开放文本与图片/视频模型能力。</p>
      </div>
      <div class="hero-actions">
        <el-input
          v-model="superAdminPassword"
          type="password"
          show-password
          placeholder="超级管理员密码"
          @keyup.enter="init"
        />
        <el-button type="primary" @click="init">进入管理</el-button>
        <el-button @click="loadDocs">查看文档</el-button>
        <el-button type="success" @click="loadAnthropicDocs">Anthropic 接入</el-button>
        <el-button type="info" @click="loadMinimaxDocs">MiniMax 接入</el-button>
      </div>
    </div>

    <el-row :gutter="18">
      <el-col :lg="10" :md="24">
        <el-card>
          <template #header>
            <span>签发 API Key</span>
          </template>
          <el-form label-position="top">
            <el-form-item label="名称">
              <el-input v-model="form.name" placeholder="如 marketing-service" />
            </el-form-item>
            <el-form-item label="允许模型（每行一个，留空表示全部）">
              <el-input v-model="modelsText" type="textarea" :rows="6" placeholder="deepseek-chat&#10;qwen-image-2.0-pro" />
            </el-form-item>
            <el-form-item label="允许能力">
              <el-checkbox-group v-model="form.allowed_scopes">
                <el-checkbox value="chat">chat</el-checkbox>
                <el-checkbox value="media">media</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            <el-form-item label="有效期（天）">
              <el-input-number v-model="form.expires_days" :min="1" :max="3650" class="w-full" />
            </el-form-item>
            <el-form-item label="每小时限制">
              <el-input-number v-model="form.rate_limit_per_hour" :min="1" :max="100000" class="w-full" />
            </el-form-item>
            <el-form-item label="预算上限">
              <el-input-number v-model="form.budget_limit" :min="0" :precision="4" class="w-full" />
            </el-form-item>
            <el-form-item label="告警阈值">
              <el-input-number v-model="form.alert_threshold" :min="0.1" :max="1" :step="0.05" :precision="2" class="w-full" />
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="form.notes" type="textarea" :rows="3" />
            </el-form-item>
            <el-button type="primary" :loading="creating" @click="createKey">生成 Key</el-button>
          </el-form>

          <el-alert
            v-if="createdPlainKey"
            title="新 Key 已生成，明文只显示一次"
            type="success"
            :closable="false"
            style="margin-top: 16px;"
          >
            <template #default>
              <div class="plain-key-box">
                <code>{{ createdPlainKey }}</code>
                <el-button size="small" @click="copyPlainKey">复制</el-button>
              </div>
            </template>
          </el-alert>
        </el-card>
      </el-col>

      <el-col :lg="14" :md="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>API Key 列表</span>
              <el-button text @click="loadKeys">刷新</el-button>
            </div>
          </template>

          <el-table :data="keys" v-loading="loadingKeys" stripe>
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column prop="key_prefix" label="前缀" min-width="140" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="total_requests" label="总请求" width="90" />
            <el-table-column prop="total_tokens" label="总 Tokens" width="110" />
            <el-table-column label="累计费用" width="120">
              <template #default="{ row }">{{ formatCost(row.total_cost, row.billing_currency) }}</template>
            </el-table-column>
            <el-table-column label="过期时间" width="180">
              <template #default="{ row }">{{ formatTime(row.expires_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="180">
              <template #default="{ row }">
                <el-button text @click="viewKey(row)">详情</el-button>
                <el-button v-if="row.status === 'active'" text type="danger" @click="revokeKey(row)">吊销</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="18">
      <el-col :lg="10" :md="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>预算告警</span>
              <el-button text @click="loadAlerts">刷新</el-button>
            </div>
          </template>
          <el-table :data="alerts" size="small" max-height="340">
            <el-table-column prop="name" label="Key" min-width="120" />
            <el-table-column label="进度" min-width="200">
              <template #default="{ row }">
                <el-progress :percentage="Math.min(row.usage_ratio * 100, 100)" :status="row.level === 'critical' ? 'exception' : 'warning'" />
              </template>
            </el-table-column>
            <el-table-column label="费用" width="120">
              <template #default="{ row }">{{ formatCost(row.total_cost, row.currency) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :lg="14" :md="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>账单报表</span>
              <div class="card-header">
                <el-select v-model="reportGroupBy" style="width: 110px" @change="loadReports">
                  <el-option label="按天" value="day" />
                  <el-option label="按月" value="month" />
                </el-select>
                <el-input-number v-model="reportDays" :min="1" :max="3650" @change="loadReports" />
              </div>
            </div>
          </template>
          <el-table :data="reports" size="small" max-height="340">
            <el-table-column prop="period" label="周期" width="110" />
            <el-table-column prop="api_key_id" label="Key ID" width="90" />
            <el-table-column prop="provider" label="Provider" width="90" />
            <el-table-column prop="model" label="模型" min-width="180" />
            <el-table-column prop="request_count" label="请求数" width="90" />
            <el-table-column prop="total_tokens" label="Tokens" width="100" />
            <el-table-column label="费用" width="120">
              <template #default="{ row }">{{ formatCost(row.total_cost, row.currency) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <!-- MiniMax Token Plan 任务管理 -->
    <el-card class="minimax-tasks-card" v-if="superAdminPassword">
      <template #header>
        <div class="card-header">
          <span>MiniMax 媒体任务</span>
          <el-button text @click="loadMinimaxTasks">刷新</el-button>
        </div>
      </template>
      <el-table :data="minimaxTasks" v-loading="loadingMinimaxTasks" stripe size="small" max-height="400">
        <el-table-column prop="task_id" label="任务ID" width="180">
          <template #default="{ row }">
            <el-button text type="primary" size="small" @click="viewMinimaxTask(row)">{{ row.task_id }}</el-button>
          </template>
        </el-table-column>
        <el-table-column prop="model" label="模型" width="140">
          <template #default="{ row }">
            <el-tag size="small" type="success">{{ row.model }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'succeeded' ? 'success' : row.status === 'failed' ? 'danger' : row.status === 'running' ? 'warning' : 'info'">
              {{ row.status === 'succeeded' ? '成功' : row.status === 'failed' ? '失败' : row.status === 'running' ? '进行中' : '等待' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button text type="primary" size="small" @click="viewMinimaxTask(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Voice Cloning 音色克隆调试模块 -->
    <el-card class="voice-cloning-card" v-if="superAdminPassword">
      <template #header>
        <div class="card-header">
          <span>Voice Cloning 音色克隆（旧版兼容）</span>
          <el-tabs v-model="voiceCloningTab" class="voice-cloning-tabs">
            <el-tab-pane label="上传音色" name="upload" />
            <el-tab-pane label="音色列表" name="list" />
            <el-tab-pane label="音色测试" name="tts" />
          </el-tabs>
          <el-button text @click="loadVoiceClones">刷新</el-button>
        </div>
      </template>

      <el-alert type="warning" :closable="false" show-icon style="margin-bottom: 16px;">
        <template #title>
          这是历史兼容入口。新的 MiniMax Speech 官方接口测试请优先使用下方的 MiniMax Speech 面板。
        </template>
      </el-alert>

      <!-- 上传音色 -->
      <div v-show="voiceCloningTab === 'upload'" class="voice-upload-section">
        <el-form label-position="top">
          <el-form-item label="音色名称">
            <el-input v-model="uploadVoiceName" placeholder="如'我的音色'" style="max-width: 300px;" />
          </el-form-item>
          <el-form-item label="音频来源">
            <el-radio-group v-model="voiceSourceType">
              <el-radio value="file">文件上传</el-radio>
              <el-radio value="mic">麦克风录制</el-radio>
            </el-radio-group>
          </el-form-item>
          <!-- 文件上传模式 -->
          <el-form-item v-show="voiceSourceType === 'file'" label="音频文件（支持 wav/mp3/m4a，最大 10MB）">
            <input type="file" accept="audio/*" @change="handleAudioFileChange" />
          </el-form-item>
          <!-- 麦克风录制模式 -->
          <el-form-item v-show="voiceSourceType === 'mic'" label="麦克风录制">
            <div class="mic-recorder">
              <div class="mic-recorder-controls">
                <el-button
                  v-if="!isRecording && !recordedAudioUrl"
                  type="danger"
                  @click="startRecording"
                >
                  开始录制
                </el-button>
                <el-button
                  v-if="isRecording"
                  type="danger"
                  @click="stopRecording"
                >
                  停止录制
                </el-button>
              </div>
              <div v-if="isRecording" class="recording-status">
                <span class="recording-dot"></span>
                <span>录制中... {{ formatDuration(recordingDuration) }}</span>
              </div>
              <div v-if="recordedAudioUrl && !isRecording" class="recording-preview">
                <audio :src="recordedAudioUrl" controls style="width: 300px;" />
                <div class="recording-actions">
                  <el-button size="small" @click="playRecording">试听</el-button>
                  <el-button size="small" type="warning" @click="clearRecording">重新录制</el-button>
                </div>
                <div class="recording-info">
                  <el-tag size="small" type="info">录制时长: {{ formatDuration(recordingDuration) }}</el-tag>
                </div>
              </div>
              <div v-if="!isRecording && !recordedAudioUrl" class="recording-hint">
                <el-alert type="info" :closable="false" show-icon>
                  <template #title>
                    点击"开始录制"后请对准麦克风说话，建议录制 30 秒以上以获得更好的效果。
                    最大录制时长 5 分钟。
                  </template>
                </el-alert>
              </div>
            </div>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="uploadingVoice" @click="uploadVoiceClone">上传复刻</el-button>
          </el-form-item>
        </el-form>
        <el-alert type="info" :closable="false" style="margin-top: 12px;">
          <template #title>
            上传音频复刻音色后，可以通过音色 ID 使用该音色进行 TTS 语音合成。
            推荐使用 30秒以上的清晰语音音频以获得更好的效果。
          </template>
        </el-alert>
      </div>

      <!-- 音色列表 -->
      <div v-show="voiceCloningTab === 'list'" class="voice-list-section">
        <el-table :data="voiceClones" v-loading="loadingVoiceClones" stripe size="small" max-height="400">
          <el-table-column prop="voice_id" label="Voice ID" min-width="160">
            <template #default="{ row }">
              <code style="font-size: 12px;">{{ row.voice_id }}</code>
            </template>
          </el-table-column>
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="row.status === 'active' ? 'success' : 'info'">
                {{ row.status === 'active' ? '可用' : row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="170">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button text type="danger" size="small" @click="deleteVoiceClone(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="!loadingVoiceClones && voiceClones.length === 0" description="暂无音色，请上传音频复刻" />
      </div>

      <!-- 音色测试 TTS -->
      <div v-show="voiceCloningTab === 'tts'" class="voice-tts-section">
        <el-form label-position="top">
          <el-form-item label="选择音色">
            <el-select v-model="selectedVoice" placeholder="请选择音色" value-key="voice_id" style="width: 300px;">
              <el-option v-for="clone in voiceClones" :key="clone.voice_id" :label="`${clone.name} (${clone.voice_id})`" :value="clone" />
            </el-select>
          </el-form-item>
          <el-form-item label="模型">
            <el-select v-model="ttsModel" style="width: 200px;">
              <el-option label="speech-2.8-hd" value="speech-2.8-hd" />
              <el-option label="speech-2.8-turbo" value="speech-2.8-turbo" />
              <el-option label="speech-02-hd" value="speech-02-hd" />
              <el-option label="speech-02-turbo" value="speech-02-turbo" />
            </el-select>
          </el-form-item>
          <el-form-item label="合成文本">
            <el-input v-model="ttsText" type="textarea" :rows="4" placeholder="请输入要合成的文本" style="max-width: 500px;" />
          </el-form-item>
          <el-form-item label="语速">
            <el-slider v-model="ttsSpeed" :min="0.5" :max="2.0" :step="0.1" show-stops style="width: 200px;" />
            <span style="margin-left: 12px;">{{ ttsSpeed }}x</span>
          </el-form-item>
          <el-form-item label="音频格式">
            <el-select v-model="ttsAudioFormat" style="width: 120px;">
              <el-option label="mp3" value="mp3" />
              <el-option label="wav" value="wav" />
              <el-option label="pcm" value="pcm" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="testingTTS" @click="testVoiceTTS">合成语音</el-button>
          </el-form-item>
        </el-form>
        <!-- TTS 结果预览 -->
        <div v-if="ttsResultUrl" class="tts-result">
          <h4>合成结果</h4>
          <audio :src="ttsResultUrl" controls style="width: 100%; max-width: 500px;" />
        </div>
      </div>
    </el-card>

    <AIGatewaySpeechPanel
      :super-admin-password="superAdminPassword"
      :prefill-api-key="createdPlainKey"
    />

    <!-- 模型快速测试 -->
    <el-card class="test-card">
      <template #header>
        <div class="card-header">
          <span>模型快速测试</span>
          <div class="card-header">
            <el-input
              v-model="testPrompt"
              placeholder="自定义测试问题（可选）"
              style="width: 280px"
            />
            <el-button @click="loadCatalog">刷新模型</el-button>
            <el-button type="primary" :loading="testingAll" @click="testAllModels">全部测试</el-button>
          </div>
        </div>
      </template>

      <el-table :data="catalogChat" v-loading="loadingCatalog" stripe size="small">
        <el-table-column prop="model" label="模型" min-width="200" />
        <el-table-column prop="brand" label="品牌" width="80">
          <template #default="{ row }">{{ row.brand || row.provider }}</template>
        </el-table-column>
        <el-table-column prop="description" label="能力" min-width="200" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag
              :type="testResults[row.model]?.status === 'ok' ? 'success' : testResults[row.model]?.status === 'err' ? 'danger' : testResults[row.model]?.status === 'running' ? 'warning' : 'info'"
              size="small"
            >
              {{ testResults[row.model]?.status === 'ok' ? '✓ 成功' : testResults[row.model]?.status === 'err' ? '✗ 失败' : testResults[row.model]?.status === 'running' ? '测试中' : '待测' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="90">
          <template #default="{ row }">
            <span v-if="testResults[row.model]?.latency">{{ testResults[row.model].latency }}ms</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="Tokens" width="80">
          <template #default="{ row }">{{ testResults[row.model]?.tokens ?? '-' }}</template>
        </el-table-column>
        <el-table-column label="回复" min-width="260">
          <template #default="{ row }">
            <span v-if="testResults[row.model]?.status === 'err'" class="test-err">{{ testResults[row.model].error }}</span>
            <span v-else class="test-reply">{{ testResults[row.model]?.reply || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-button
              text
              type="primary"
              size="small"
              :loading="testResults[row.model]?.status === 'running'"
              @click="testModel(row.model)"
            >测试</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="docs-card">
      <template #header>
        <div class="card-header">
          <span>接入文档</span>
          <el-button text @click="loadDocs">刷新文档</el-button>
        </div>
      </template>

      <div class="docs-grid">
        <div class="docs-panel">
          <h4>接入方式</h4>
          <p class="docs-text">
            业务方使用平台签发的 API Key 直接调用统一网关。文本能力走
            `/api/ai-gateway/v1/chat/completions`，图片和视频走
            `/api/ai-gateway/v1/media/generations`，所有请求都会进入统一记账、限流和审计。
          </p>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="鉴权 Header">Authorization: Bearer &lt;API_KEY&gt;</el-descriptions-item>
            <el-descriptions-item label="文本接口">POST /api/ai-gateway/v1/chat/completions</el-descriptions-item>
            <el-descriptions-item label="媒体接口">POST /api/ai-gateway/v1/media/generations</el-descriptions-item>
            <el-descriptions-item label="任务查询">GET /api/ai-gateway/v1/media/tasks/:id</el-descriptions-item>
            <el-descriptions-item label="接口总文档">GET /api/ai-gateway/docs</el-descriptions-item>
          </el-descriptions>
        </div>

        <div class="docs-panel">
          <h4>返回与计费</h4>
          <ul class="docs-list">
            <li>文本接口会回传上游结果，并附带 `usage_summary`，包含输入、输出、总 Tokens 和费用。</li>
            <li>媒体接口返回统一任务对象，支持轮询查看状态、结果地址和失败原因。</li>
            <li>每个 API Key 都会累计请求数、Token、费用，并支持预算上限、告警阈值、日报和月报。</li>
          </ul>
        </div>
      </div>

      <el-tabs class="docs-tabs">
        <el-tab-pane label="快速开始">
          <div class="docs-panel-stack">
            <h4>接入步骤</h4>
            <ol class="docs-steps">
              <li>在当前页面生成 API Key，并确认允许的模型、作用域、限流和预算。</li>
              <li>业务服务端保存 API Key，不要暴露到浏览器前端。</li>
              <li>调用聊天接口或媒体接口，Header 带上 `Authorization: Bearer &lt;API_KEY&gt;`。</li>
              <li>读取响应中的 `usage_summary`、任务状态和错误信息，接入你自己的日志系统。</li>
              <li>在本页查看请求明细、累计 Token、费用、预算告警和日报月报。</li>
            </ol>
          </div>
        </el-tab-pane>
        <el-tab-pane label="字段说明">
          <el-table :data="gatewayFieldDocs" size="small" stripe>
            <el-table-column prop="field" label="字段" width="220" />
            <el-table-column prop="required" label="必填" width="80" />
            <el-table-column prop="description" label="说明" min-width="260" />
            <el-table-column prop="example" label="示例" min-width="220" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="cURL">
          <pre class="doc-json">{{ gatewayCurlExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="JavaScript">
          <pre class="doc-json">{{ gatewayJsExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="Python">
          <pre class="doc-json">{{ gatewayPythonExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="媒体示例">
          <pre class="doc-json">{{ gatewayMediaExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="响应示例">
          <pre class="doc-json">{{ gatewayResponseExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="错误码">
          <el-table :data="gatewayErrorDocs" size="small" stripe>
            <el-table-column prop="code" label="状态码" width="100" />
            <el-table-column prop="scene" label="场景" width="180" />
            <el-table-column prop="meaning" label="含义" min-width="220" />
            <el-table-column prop="fix" label="处理建议" min-width="260" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="常见问题">
          <div class="faq-list">
            <div v-for="item in gatewayFaqs" :key="item.q" class="faq-item">
              <h4>{{ item.q }}</h4>
              <p>{{ item.a }}</p>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>

      <div v-if="docs" class="docs-api-list">
        <h4>接口清单</h4>
        <el-descriptions :column="1" border>
          <el-descriptions-item v-for="route in docs.routes" :key="route.path" :label="`${route.method} ${route.path}`">
            {{ route.description }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-card>

    <el-dialog v-model="detailVisible" width="920px" title="Key 详情">
      <div v-if="currentKey">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="名称">{{ currentKey.name }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ currentKey.status }}</el-descriptions-item>
          <el-descriptions-item label="前缀">{{ currentKey.key_prefix }}</el-descriptions-item>
          <el-descriptions-item label="总请求">{{ currentKey.total_requests }}</el-descriptions-item>
          <el-descriptions-item label="输入 Tokens">{{ currentKey.total_input_tokens }}</el-descriptions-item>
          <el-descriptions-item label="输出 Tokens">{{ currentKey.total_output_tokens }}</el-descriptions-item>
          <el-descriptions-item label="总 Tokens">{{ currentKey.total_tokens }}</el-descriptions-item>
          <el-descriptions-item label="累计费用">{{ formatCost(currentKey.total_cost, currentKey.billing_currency) }}</el-descriptions-item>
          <el-descriptions-item label="允许模型">{{ currentKey.allowed_models }}</el-descriptions-item>
          <el-descriptions-item label="允许能力">{{ currentKey.allowed_scopes }}</el-descriptions-item>
        </el-descriptions>

        <div class="logs-section">
          <h4>最近请求明细</h4>
          <el-table :data="currentLogs" max-height="420">
            <el-table-column prop="created_at" label="时间" width="180">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column prop="provider" label="Provider" width="100" />
            <el-table-column prop="model" label="模型" min-width="180" />
            <el-table-column prop="request_type" label="类型" width="90" />
            <el-table-column prop="status_code" label="状态码" width="90" />
            <el-table-column prop="total_tokens" label="Tokens" width="90" />
            <el-table-column label="费用" width="100">
              <template #default="{ row }">{{ formatCost(row.estimated_cost, row.currency) }}</template>
            </el-table-column>
            <el-table-column prop="latency_ms" label="耗时(ms)" width="100" />
            <el-table-column prop="error_message" label="错误信息" min-width="220" />
          </el-table>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="docsVisible" width="920px" title="AI Gateway 接入文档">
      <div v-if="docs" class="docs-content">
        <el-alert :title="docs.summary" type="info" :closable="false" />

        <!-- 可用模型列表 -->
        <div v-if="docs.billing && docs.billing.pricing" style="margin-top: 16px;">
          <h4>可用模型</h4>
          <el-table :data="docs.billing.pricing" size="small" max-height="300">
            <el-table-column prop="Model" label="模型" />
            <el-table-column prop="Provider" label="提供商" width="100">
              <template #default="{ row }">
                <el-tag size="small">{{ row.Provider }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="InputPer1KTokens" label="输入/千token" width="100" />
            <el-table-column prop="OutputPer1KTokens" label="输出/千token" width="100" />
            <el-table-column prop="Currency" label="货币" width="80" />
          </el-table>
        </div>

        <el-divider />

        <h4>API 路由</h4>
        <el-descriptions :column="1" border>
          <el-descriptions-item v-for="route in docs.routes" :key="route.path" :label="`${route.method} ${route.path}`">
            {{ route.description }}
          </el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top: 16px;">调用示例</h4>
        <pre class="doc-json">{{ prettyJSON(docs.examples) }}</pre>
      </div>
    </el-dialog>

    <!-- Anthropic 协议接入文档对话框 -->
    <el-dialog v-model="anthropicDocsVisible" width="960px" title="Anthropic 协议接入文档">
      <div v-if="anthropicDocs" class="docs-content">
        <el-alert :title="anthropicDocs.summary" type="info" :closable="false" />

        <h4 style="margin-top: 16px;">支持的提供商</h4>
        <el-table :data="anthropicDocs.providers" size="small" stripe>
          <el-table-column prop="name" label="提供商" width="120">
            <template #default="{ row }">
              <el-tag size="small" type="success">{{ row.name }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="base_url" label="DevTools 端点" min-width="200">
            <template #default="{ row }">
              <code style="font-size: 12px;">{{ row.base_url }}/v1/messages</code>
            </template>
          </el-table-column>
          <el-table-column prop="upstream" label="上游地址" min-width="220">
            <template #default="{ row }">
              <code style="font-size: 11px; color: #909399;">{{ row.upstream }}</code>
            </template>
          </el-table-column>
          <el-table-column label="支持模型" min-width="280">
            <template #default="{ row }">
              <el-tag v-for="m in row.models" :key="m" size="small" style="margin-right: 4px; margin-bottom: 2px;">{{ m }}</el-tag>
            </template>
          </el-table-column>
        </el-table>

        <el-divider />

        <h4>API 路由</h4>
        <el-descriptions :column="1" border>
          <el-descriptions-item v-for="route in anthropicDocs.routes" :key="route.path" :label="`${route.method} ${route.path}`">
            {{ route.description }}
          </el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top: 16px;">请求示例</h4>
        <el-tabs>
          <el-tab-pane label="MiniMax">
            <pre class="doc-json">{{ JSON.stringify(anthropicDocs.examples.minimax.request, null, 2) }}</pre>
            <div style="margin-top: 12px;">
              <el-divider>Claude Code 配置 (MiniMax)</el-divider>
              <pre class="doc-code">{{ anthropicDocs.examples.minimax.claude_code_config.code }}</pre>
            </div>
          </el-tab-pane>
          <el-tab-pane label="DashScope">
            <pre class="doc-json">{{ JSON.stringify(anthropicDocs.examples.dashscope.request, null, 2) }}</pre>
            <div style="margin-top: 12px;">
              <el-divider>Claude Code 配置 (DashScope)</el-divider>
              <pre class="doc-code">{{ anthropicDocs.examples.dashscope.claude_code_config.code }}</pre>
            </div>
          </el-tab-pane>
        </el-tabs>

        <h4 style="margin-top: 16px;">SDK 调用示例</h4>
        <el-tabs>
          <el-tab-pane label="Python SDK">
            <pre class="doc-code">{{ anthropicDocs.examples.python_sdk.code }}</pre>
          </el-tab-pane>
          <el-tab-pane label="JavaScript SDK">
            <pre class="doc-code">{{ anthropicDocs.examples.javascript_sdk.code }}</pre>
          </el-tab-pane>
          <el-tab-pane label="cURL">
            <pre class="doc-code">{{ anthropicDocs.examples.curl.code }}</pre>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-dialog>

    <!-- MiniMax Token Plan 接入文档对话框 -->
    <el-dialog v-model="minimaxDocsVisible" width="960px" title="MiniMax Token Plan 接入文档">
      <div v-if="minimaxDocs" class="docs-content">
        <el-alert :title="minimaxDocs.summary" type="info" :closable="false" />

        <h4 style="margin-top: 16px;">支持的模型</h4>
        <el-table :data="Object.entries(minimaxDocs.model_descriptions || {}).map(([model, desc]) => ({ model, description: desc }))" size="small" stripe>
          <el-table-column prop="model" label="模型" width="160">
            <template #default="{ row }">
              <el-tag size="small" type="success">{{ row.model }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="说明" min-width="300" />
        </el-table>

        <el-divider />

        <h4>认证方式</h4>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="鉴权 Header">Authorization: Bearer &lt;API_KEY&gt;</el-descriptions-item>
          <el-descriptions-item label="Required Scope">media</el-descriptions-item>
          <el-descriptions-item label="Base URL">/api/minimax/token-plan</el-descriptions-item>
          <el-descriptions-item label="上游地址">{{ minimaxDocs.upstream }}</el-descriptions-item>
        </el-descriptions>

        <el-divider />

        <h4>API 路由</h4>
        <el-descriptions :column="1" border>
          <el-descriptions-item v-for="route in minimaxDocs.routes" :key="route.path" :label="`${route.method} ${route.path}`">
            {{ route.description }}
          </el-descriptions-item>
          <el-descriptions-item label="GET /api/minimax/token-plan/tasks">
            查询当前 API Key 的媒体任务列表
          </el-descriptions-item>
          <el-descriptions-item label="GET /api/minimax/token-plan/tasks/:id">
            查询指定任务的详情和结果
          </el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top: 16px;">模型请求示例</h4>
        <el-tabs>
          <el-tab-pane label="TTS HD">
            <pre class="doc-json">{{ JSON.stringify(minimaxDocs.examples?.tts_hd_request || {}, null, 2) }}</pre>
          </el-tab-pane>
          <el-tab-pane label="Hailuo 视频">
            <pre class="doc-json">{{ JSON.stringify(minimaxDocs.examples?.hailuo_request || {}, null, 2) }}</pre>
          </el-tab-pane>
          <el-tab-pane label="Music">
            <pre class="doc-json">{{ JSON.stringify(minimaxDocs.examples?.music_request || {}, null, 2) }}</pre>
          </el-tab-pane>
          <el-tab-pane label="Image">
            <pre class="doc-json">{{ JSON.stringify(minimaxDocs.examples?.image_request || {}, null, 2) }}</pre>
          </el-tab-pane>
        </el-tabs>

        <h4 style="margin-top: 16px;">SDK 调用示例</h4>
        <el-tabs>
          <el-tab-pane label="cURL">
            <pre class="doc-code">{{ minimaxDocs.examples?.curl?.code || '' }}</pre>
          </el-tab-pane>
          <el-tab-pane label="Python">
            <pre class="doc-code">import requests

resp = requests.post(
    "${origin}/api/minimax/token-plan/v1/generations",
    headers={
        "Authorization": "Bearer dtk_ai_xxx",
        "Content-Type": "application/json",
    },
    json={
        "model": "speech-2.8-hd",
        "text": "你好，这是语音合成测试",
    },
    timeout=120,
)
# 音频响应
with open("output.mp3", "wb") as f:
    f.write(resp.content)</pre>
          </el-tab-pane>
          <el-tab-pane label="JavaScript">
            <pre class="doc-code">const response = await fetch("${origin}/api/minimax/token-plan/v1/generations", {
  method: "POST",
  headers: {
    "Authorization": "Bearer dtk_ai_xxx",
    "Content-Type": "application/json"
  },
  body: JSON.stringify({
    model: "speech-2.8-hd",
    text: "你好，这是语音合成测试"
  })
});

// 音频响应
const blob = await response.blob();
const url = URL.createObjectURL(blob);
const a = document.createElement("a");
a.href = url;
a.download = "output.mp3";
a.click();</pre>
          </el-tab-pane>
        </el-tabs>

        <el-divider />

        <h4>异步任务使用说明（视频/音乐/图片）</h4>
        <div class="docs-panel-stack">
          <p class="docs-text">
            视频、音乐和图片生成采用异步模式：提交任务后返回 task_id，通过轮询查询任务状态和结果。
          </p>
          <el-tabs>
            <el-tab-pane label="异步调用流程">
              <pre class="doc-code">// 1. 提交生成任务
const createResp = await fetch("${origin}/api/minimax/token-plan/v1/generations", {
  method: "POST",
  headers: {
    "Authorization": "Bearer dtk_ai_xxx",
    "Content-Type": "application/json"
  },
  body: JSON.stringify({
    model: "Hailuo-2.3-Fast",
    prompt: "一只猫在草地上玩耍",
    duration: 6,
    resolution: "768P"
  })
});
const { task_id, status } = await createResp.json();
console.log("任务ID:", task_id);

// 2. 轮询任务状态
async function pollTask(taskId, maxAttempts = 60) {
  for (let i = 0; i < maxAttempts; i++) {
    await new Promise(r => setTimeout(r, 3000)); // 3秒间隔
    const resp = await fetch(\`\${origin}/api/minimax/token-plan/tasks/\${taskId}\`, {
      headers: { "Authorization": "Bearer dtk_ai_xxx" }
    });
    const task = await resp.json();
    console.log(\`状态: \${task.status}\`);
    if (task.status === "succeeded") {
      console.log("结果:", task.result_urls);
      return task;
    }
    if (task.status === "failed") {
      throw new Error("任务失败: " + task.error);
    }
  }
  throw new Error("任务超时");
}

// 3. 下载结果
const result = await pollTask(task_id);
if (result.result_urls && result.result_urls.length > 0) {
  const videoUrl = result.result_urls[0];
  window.open(videoUrl, "_blank");
}</pre>
            </el-tab-pane>
            <el-tab-pane label="任务列表查询">
              <pre class="doc-code">// 查询当前 API Key 的所有任务
const listResp = await fetch("${origin}/api/minimax/token-plan/tasks", {
  headers: { "Authorization": "Bearer dtk_ai_xxx" }
});
const { tasks, total } = await listResp.json();
console.log("任务总数:", total);
tasks.forEach(task => {
  console.log(\`[\${task.status}] \${task.model} - \${task.task_id}\`);
  if (task.result_urls) {
    console.log("  结果:", task.result_urls);
  }
});</pre>
            </el-tab-pane>
          </el-tabs>
        </div>

        <el-divider />

        <h4>模型特性说明</h4>
        <el-table :data="minimaxModelFeatures" size="small" stripe>
          <el-table-column prop="model" label="模型" width="160">
            <template #default="{ row }">
              <el-tag size="small" :type="row.async ? 'warning' : 'success'">{{ row.model }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="type" label="类型" width="100" />
          <el-table-column prop="mode" label="模式" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="row.async ? 'warning' : 'success'">{{ row.async ? '异步' : '同步' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="说明" min-width="300" />
        </el-table>
      </div>
    </el-dialog>

    <!-- MiniMax Token Plan 任务详情对话框 -->
    <el-dialog v-model="minimaxTaskVisible" width="860px" title="MiniMax 任务详情">
      <div v-if="currentMinimaxTask" class="minimax-task-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="任务ID">{{ currentMinimaxTask.task_id }}</el-descriptions-item>
          <el-descriptions-item label="模型">
            <el-tag size="small" type="success">{{ currentMinimaxTask.model }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag size="small" :type="currentMinimaxTask.status === 'succeeded' ? 'success' : currentMinimaxTask.status === 'failed' ? 'danger' : currentMinimaxTask.status === 'running' ? 'warning' : 'info'">
              {{ currentMinimaxTask.status === 'succeeded' ? '成功' : currentMinimaxTask.status === 'failed' ? '失败' : currentMinimaxTask.status === 'running' ? '进行中' : '等待' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(currentMinimaxTask.created_at) }}</el-descriptions-item>
          <el-descriptions-item v-if="currentMinimaxTask.completed_at" label="完成时间">{{ formatTime(currentMinimaxTask.completed_at) }}</el-descriptions-item>
          <el-descriptions-item v-if="currentMinimaxTask.external_task_id" label="外部任务ID">{{ currentMinimaxTask.external_task_id }}</el-descriptions-item>
          <el-descriptions-item v-if="currentMinimaxTask.error" label="错误信息" :span="2">
            <span style="color: #dc2626;">{{ currentMinimaxTask.error }}</span>
          </el-descriptions-item>
        </el-descriptions>

        <!-- 媒体预览区域 -->
        <div v-if="currentMinimaxTask.result_urls && currentMinimaxTask.result_urls.length > 0" class="media-preview-section">
          <h4>生成结果</h4>
          <p style="color: #888; font-size: 12px; margin-bottom: 8px;">
            <el-icon><Download /></el-icon> 使用我们的下载代理端点，外部也可访问（MiniMax 原始链接仅供调试）
          </p>
          <div class="media-preview-grid">
            <div v-for="(url, idx) in currentMinimaxTask.result_urls" :key="idx" class="media-preview-item">
              <!-- 音频预览 -->
              <div v-if="currentMinimaxTask.content_type && currentMinimaxTask.content_type.startsWith('audio/')" class="audio-preview">
                <audio controls :src="`/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`" style="width: 100%;"></audio>
                <div class="media-url-box">
                  <el-link :href="`/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`" target="_blank" type="primary" class="url-link">通过代理下载</el-link>
                  <el-button size="small" @click="copyText(`\${origin}/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`)">复制下载链接</el-button>
                </div>
              </div>
              <!-- 视频预览 -->
              <div v-else-if="currentMinimaxTask.content_type && currentMinimaxTask.content_type.startsWith('video/')" class="video-preview">
                <video controls :src="`/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`" style="width: 100%; max-height: 300px;"></video>
                <div class="media-url-box">
                  <el-link :href="`/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`" target="_blank" type="primary" class="url-link">通过代理下载</el-link>
                  <el-button size="small" @click="copyText(`\${origin}/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`)">复制下载链接</el-button>
                </div>
              </div>
              <!-- 图片预览 -->
              <div v-else-if="currentMinimaxTask.content_type && currentMinimaxTask.content_type.startsWith('image/')" class="image-preview">
                <el-image :src="`/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`" fit="contain" style="width: 100%; max-height: 300px;" :preview-src-list="[`/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`]" />
                <div class="media-url-box">
                  <el-link :href="`/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`" target="_blank" type="primary" class="url-link">通过代理下载</el-link>
                  <el-button size="small" @click="copyText(`\${origin}/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`)">复制下载链接</el-button>
                </div>
              </div>
              <!-- 通用链接 -->
              <div v-else class="generic-preview">
                <el-link :href="`/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`" target="_blank" type="primary">
                  <el-icon><Download /></el-icon> 下载 / 预览
                </el-link>
                <div class="media-url-box">
                  <el-link :href="`/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`" target="_blank" type="primary" class="url-link">通过代理下载</el-link>
                  <el-button size="small" @click="copyText(`\${origin}/api/minimax/token-plan/tasks/${currentMinimaxTask.task_id}/download`)">复制下载链接</el-button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 原始响应 JSON -->
        <div v-if="currentMinimaxTask.result" class="result-json-section">
          <h4>原始响应</h4>
          <pre class="doc-json">{{ typeof currentMinimaxTask.result === 'string' ? currentMinimaxTask.result : JSON.stringify(currentMinimaxTask.result, null, 2) }}</pre>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox, ElIcon } from 'element-plus'
import { Download } from '@element-plus/icons-vue'
import AIGatewaySpeechPanel from '../components/AIGatewaySpeechPanel.vue'

const API_BASE = '/api/ai-gateway'
const PASSWORD_KEY = 'ai_gateway_super_admin_password'

const superAdminPassword = ref(sessionStorage.getItem(PASSWORD_KEY) || '')
const creating = ref(false)
const loadingKeys = ref(false)
const keys = ref([])
const currentKey = ref(null)
const currentLogs = ref([])
const reports = ref([])
const alerts = ref([])
const detailVisible = ref(false)
const docsVisible = ref(false)
const docs = ref(null)
const anthropicDocsVisible = ref(false)
const anthropicDocs = ref(null)
const minimaxDocsVisible = ref(false)
const minimaxDocs = ref(null)
const createdPlainKey = ref('')
const modelsText = ref('')

const form = ref({
  name: '',
  allowed_scopes: ['chat', 'media'],
  expires_days: 90,
  rate_limit_per_hour: 1000,
  budget_limit: 0,
  alert_threshold: 0.8,
  notes: ''
})
const reportGroupBy = ref('day')
const reportDays = ref(30)
const origin = typeof window !== 'undefined' ? window.location.origin : ''

// MiniMax Token Plan 任务管理
const minimaxTasks = ref([])
const loadingMinimaxTasks = ref(false)
const minimaxTaskVisible = ref(false)
const currentMinimaxTask = ref(null)

// Voice Cloning 音色克隆管理
const voiceClones = ref([])
const loadingVoiceClones = ref(false)
const voiceCloningTab = ref('upload') // 'upload' | 'list' | 'tts'
const uploadVoiceName = ref('')
const uploadAudioFile = ref(null)
const uploadingVoice = ref(false)
const selectedVoice = ref(null)
const ttsText = ref('')
const ttsModel = ref('speech-2.8-hd')
const ttsSpeed = ref(1.0)
const ttsAudioFormat = ref('mp3')
const ttsResultUrl = ref('')
const testingTTS = ref(false)

// Voice Cloning 麦克风录制
const voiceSourceType = ref('file') // 'file' | 'mic'
const isRecording = ref(false)
const recordingDuration = ref(0)
const recordedAudioUrl = ref('')
const recordedChunks = ref([])
let mediaRecorder = null
let recordingTimer = null

// 模型快速测试
const testPrompt = ref('')
const testResults = ref({})    // { [model]: { status, reply, error, latency, tokens } }
const loadingCatalog = ref(false)
const catalogChat = ref([])
const testingAll = ref(false)

const loadCatalog = async () => {
  loadingCatalog.value = true
  try {
    const res = await fetch(`${API_BASE}/catalog`)
    const data = await res.json()
    catalogChat.value = (data.models || []).filter(m => m.type === 'chat')
  } finally {
    loadingCatalog.value = false
  }
}

const testModel = async (model) => {
  if (!superAdminPassword.value) {
    ElMessage.error('请先输入超级管理员密码')
    return
  }
  testResults.value = { ...testResults.value, [model]: { status: 'running' } }
  try {
    const res = await fetch(`${API_BASE}/admin/test-model`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Super-Admin-Password': superAdminPassword.value },
      body: JSON.stringify({ super_admin_password: superAdminPassword.value, model, prompt: testPrompt.value || undefined })
    })
    const data = await res.json()
    if (data.status === 'ok') {
      testResults.value = { ...testResults.value, [model]: { status: 'ok', reply: data.reply, latency: data.latency, tokens: data.tokens } }
    } else {
      testResults.value = { ...testResults.value, [model]: { status: 'err', error: data.error || '未知错误', latency: data.latency } }
    }
  } catch (err) {
    testResults.value = { ...testResults.value, [model]: { status: 'err', error: err.message } }
  }
}

const testAllModels = async () => {
  if (!superAdminPassword.value) {
    ElMessage.error('请先输入超级管理员密码')
    return
  }
  testingAll.value = true
  try {
    await Promise.allSettled(catalogChat.value.map(m => testModel(m.model)))
  } finally {
    testingAll.value = false
  }
}

const authHeaders = () => ({
  'Content-Type': 'application/json',
  'X-Super-Admin-Password': superAdminPassword.value
})

const init = async () => {
  if (!superAdminPassword.value) {
    ElMessage.error('请输入超级管理员密码')
    return
  }
  sessionStorage.setItem(PASSWORD_KEY, superAdminPassword.value)
  await loadKeys()
  await Promise.all([loadReports(), loadAlerts(), loadMinimaxTasks()])
}

const createKey = async () => {
  if (!form.value.name.trim()) {
    ElMessage.error('名称必填')
    return
  }
  creating.value = true
  try {
    const res = await fetch(`${API_BASE}/admin/keys`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({
        ...form.value,
        super_admin_password: superAdminPassword.value,
        allowed_models: modelsText.value.split('\n').map(item => item.trim()).filter(Boolean)
      })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '生成失败')
    createdPlainKey.value = data.plain_key
    ElMessage.success('API Key 已生成')
    await Promise.all([loadKeys(), loadAlerts()])
  } catch (err) {
    ElMessage.error(err.message || '生成失败')
  } finally {
    creating.value = false
  }
}

const loadReports = async () => {
  const params = new URLSearchParams({
    group_by: reportGroupBy.value,
    days: String(reportDays.value)
  })
  const res = await fetch(`${API_BASE}/admin/reports?${params.toString()}`, {
    headers: authHeaders()
  })
  const data = await res.json()
  if (res.ok) reports.value = data.rows || []
}

const loadAlerts = async () => {
  const res = await fetch(`${API_BASE}/admin/alerts`, {
    headers: { 'X-Super-Admin-Password': superAdminPassword.value }
  })
  const data = await res.json()
  if (res.ok) alerts.value = data.alerts || []
}

const loadKeys = async () => {
  loadingKeys.value = true
  try {
    const res = await fetch(`${API_BASE}/admin/keys`, {
      headers: authHeaders()
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载失败')
    keys.value = data.keys || []
  } catch (err) {
    ElMessage.error(err.message || '加载失败')
  } finally {
    loadingKeys.value = false
  }
}

const viewKey = async (row) => {
  const res = await fetch(`${API_BASE}/admin/keys/${row.id}`, {
    headers: { 'X-Super-Admin-Password': superAdminPassword.value }
  })
  const data = await res.json()
  if (!res.ok) {
    ElMessage.error(data.error || '加载详情失败')
    return
  }
  currentKey.value = data.key
  currentLogs.value = data.logs || []
  detailVisible.value = true
}

const revokeKey = async (row) => {
  await ElMessageBox.confirm(`确认吊销 ${row.name} ?`, '提示', { type: 'warning' })
  const res = await fetch(`${API_BASE}/admin/keys/${row.id}/revoke`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ super_admin_password: superAdminPassword.value })
  })
  const data = await res.json()
  if (!res.ok) {
    ElMessage.error(data.error || '吊销失败')
    return
  }
  ElMessage.success('已吊销')
  loadKeys()
}

const loadDocs = async (showDialog = true) => {
  const res = await fetch(`${API_BASE}/docs`)
  const data = await res.json()
  docs.value = data
  if (showDialog) {
    docsVisible.value = true
  }
}

const loadAnthropicDocs = async () => {
  const res = await fetch(`${API_BASE}/docs/anthropic`)
  const data = await res.json()
  // 替换文档中的占位符为当前域名
  const docStr = JSON.stringify(data)
  const replaced = docStr.replace(/your-devtools:8080/g, window.location.host)
  anthropicDocs.value = JSON.parse(replaced)
  anthropicDocsVisible.value = true
}

const loadMinimaxDocs = async () => {
  const res = await fetch(`/api/minimax/token-plan/docs`)
  const data = await res.json()
  // 替换文档中的占位符为当前域名
  const docStr = JSON.stringify(data)
  const replaced = docStr.replace(/your-devtools:8080/g, window.location.host)
  minimaxDocs.value = JSON.parse(replaced)
  minimaxDocsVisible.value = true
}

const loadMinimaxTasks = async () => {
  if (!superAdminPassword.value) {
    ElMessage.error('请先输入超级管理员密码')
    return
  }
  loadingMinimaxTasks.value = true
  try {
    const res = await fetch(`/api/minimax/token-plan/tasks?limit=50`, {
      headers: { 'X-Super-Admin-Password': superAdminPassword.value }
    })
    const data = await res.json()
    if (res.ok) {
      minimaxTasks.value = data.tasks || []
    } else {
      ElMessage.error(data.error || '加载任务失败')
    }
  } catch (err) {
    ElMessage.error('加载任务失败: ' + err.message)
  } finally {
    loadingMinimaxTasks.value = false
  }
}

const viewMinimaxTask = async (row) => {
  if (!superAdminPassword.value) {
    ElMessage.error('请先输入超级管理员密码')
    return
  }
  try {
    const res = await fetch(`/api/minimax/token-plan/tasks/${row.task_id}`, {
      headers: { 'X-Super-Admin-Password': superAdminPassword.value }
    })
    const data = await res.json()
    if (res.ok) {
      currentMinimaxTask.value = data
      minimaxTaskVisible.value = true
    } else {
      ElMessage.error(data.error || '加载任务详情失败')
    }
  } catch (err) {
    ElMessage.error('加载任务详情失败: ' + err.message)
  }
}

// Voice Cloning 相关函数
const loadVoiceClones = async () => {
  if (!superAdminPassword.value) {
    ElMessage.error('请先输入超级管理员密码')
    return
  }
  loadingVoiceClones.value = true
  try {
    const res = await fetch(`/api/minimax/voice-cloning/voices?limit=100`, {
      headers: { 'X-Super-Admin-Password': superAdminPassword.value }
    })
    const data = await res.json()
    if (res.ok) {
      voiceClones.value = data.voices || []
    } else {
      ElMessage.error(data.error || '加载音色列表失败')
    }
  } catch (err) {
    ElMessage.error('加载音色列表失败: ' + err.message)
  } finally {
    loadingVoiceClones.value = false
  }
}

const handleAudioFileChange = (event) => {
  const file = event.target.files[0]
  if (file) {
    if (file.size > 10 * 1024 * 1024) {
      ElMessage.error('音频文件大小不能超过 10MB')
      event.target.value = ''
      return
    }
    uploadAudioFile.value = file
  }
}

const uploadVoiceClone = async () => {
  if (!superAdminPassword.value) {
    ElMessage.error('请先输入超级管理员密码')
    return
  }
  if (!uploadVoiceName.value.trim()) {
    ElMessage.error('请输入音色名称')
    return
  }
  if (!uploadAudioFile.value) {
    ElMessage.error('请选择音频文件')
    return
  }
  uploadingVoice.value = true
  try {
    const formData = new FormData()
    formData.append('name', uploadVoiceName.value.trim())
    formData.append('audio_file', uploadAudioFile.value)
    const res = await fetch(`/api/minimax/voice-cloning/upload`, {
      method: 'POST',
      headers: { 'X-Super-Admin-Password': superAdminPassword.value },
      body: formData
    })
    const data = await res.json()
    if (res.ok) {
      ElMessage.success('音色创建成功: ' + data.voice_id)
      uploadVoiceName.value = ''
      uploadAudioFile.value = null
      voiceCloningTab.value = 'list'
      loadVoiceClones()
    } else {
      ElMessage.error(data.error || '上传失败')
    }
  } catch (err) {
    ElMessage.error('上传失败: ' + err.message)
  } finally {
    uploadingVoice.value = false
  }
}

const deleteVoiceClone = async (clone) => {
  if (!superAdminPassword.value) {
    ElMessage.error('请先输入超级管理员密码')
    return
  }
  try {
    await ElMessageBox.confirm(`确认删除音色 "${clone.name}" ?`, '提示', { type: 'warning' })
    const res = await fetch(`/api/minimax/voice-cloning/voices/${clone.id}`, {
      method: 'DELETE',
      headers: { 'X-Super-Admin-Password': superAdminPassword.value }
    })
    const data = await res.json()
    if (res.ok) {
      ElMessage.success('音色已删除')
      loadVoiceClones()
    } else {
      ElMessage.error(data.error || '删除失败')
    }
  } catch (err) {
    if (err !== 'cancel') {
      ElMessage.error('删除失败: ' + err.message)
    }
  }
}

const testVoiceTTS = async () => {
  if (!superAdminPassword.value) {
    ElMessage.error('请先输入超级管理员密码')
    return
  }
  if (!selectedVoice.value) {
    ElMessage.error('请选择音色')
    return
  }
  if (!ttsText.value.trim()) {
    ElMessage.error('请输入要合成的文本')
    return
  }
  testingTTS.value = true
  ttsResultUrl.value = ''
  try {
    const res = await fetch(`/api/minimax/voice-cloning/tts`, {
      method: 'POST',
      headers: {
        'X-Super-Admin-Password': superAdminPassword.value,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        model: ttsModel.value,
        text: ttsText.value.trim(),
        voice_id: selectedVoice.value.voice_id,
        speed: ttsSpeed.value,
        audio_format: ttsAudioFormat.value
      })
    })
    if (res.ok) {
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      ttsResultUrl.value = url
      ElMessage.success('语音合成成功')
    } else {
      const data = await res.json()
      ElMessage.error(data.error || '语音合成失败')
    }
  } catch (err) {
    ElMessage.error('语音合成失败: ' + err.message)
  } finally {
    testingTTS.value = false
  }
}

// 麦克风录制功能
const startRecording = async () => {
  if (isRecording.value) return
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    recordedChunks.value = []
    mediaRecorder = new MediaRecorder(stream, { mimeType: 'audio/webm' })
    mediaRecorder.ondataavailable = (e) => {
      if (e.data && e.data.size > 0) {
        recordedChunks.value.push(e.data)
      }
    }
    mediaRecorder.onstop = () => {
      const blob = new Blob(recordedChunks.value, { type: 'audio/webm' })
      const url = URL.createObjectURL(blob)
      recordedAudioUrl.value = url
      // 转换为 File 对象以供上传
      const file = new File([blob], 'recording.webm', { type: 'audio/webm' })
      uploadAudioFile.value = file
      // 停止所有 track
      stream.getTracks().forEach(track => track.stop())
    }
    mediaRecorder.start()
    isRecording.value = true
    recordingDuration.value = 0
    recordingTimer = setInterval(() => {
      recordingDuration.value++
      // 最大录制 5 分钟（300秒）
      if (recordingDuration.value >= 300) {
        stopRecording()
      }
    }, 1000)
    ElMessage.info('开始录制，请对准麦克风说话')
  } catch (err) {
    console.error('Failed to start recording:', err)
    if (err.name === 'NotAllowedError') {
      ElMessage.error('麦克风权限被拒绝，请在浏览器设置中允许使用麦克风')
    } else if (err.name === 'NotFoundError') {
      ElMessage.error('未找到麦克风设备，请确认已连接麦克风')
    } else {
      ElMessage.error('无法启动麦克风录制: ' + err.message)
    }
  }
}

const stopRecording = () => {
  if (!isRecording.value || !mediaRecorder) return
  mediaRecorder.stop()
  isRecording.value = false
  if (recordingTimer) {
    clearInterval(recordingTimer)
    recordingTimer = null
  }
  ElMessage.success('录制完成，时长: ' + formatDuration(recordingDuration.value))
}

const playRecording = () => {
  if (!recordedAudioUrl.value) return
  const audio = new Audio(recordedAudioUrl.value)
  audio.play()
}

const clearRecording = () => {
  if (recordedAudioUrl.value) {
    URL.revokeObjectURL(recordedAudioUrl.value)
  }
  recordedAudioUrl.value = ''
  uploadAudioFile.value = null
  recordedChunks.value = []
  recordingDuration.value = 0
  if (isRecording.value) {
    stopRecording()
  }
}

const formatDuration = (seconds) => {
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

const copyText = async (text) => {
  await navigator.clipboard.writeText(text)
  ElMessage.success('已复制')
}

const copyPlainKey = async () => {
  await navigator.clipboard.writeText(createdPlainKey.value)
  ElMessage.success('已复制')
}

const gatewayCurlExample = `curl -X POST ${origin}/api/ai-gateway/v1/chat/completions \\
  -H "Authorization: Bearer ai_xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "deepseek-v4-pro",
    "messages": [
      { "role": "system", "content": "你是一个业务助手" },
      { "role": "user", "content": "帮我生成一段活动宣传文案" }
    ],
    "reasoning_effort": "medium"
  }'`

const gatewayJsExample = `const response = await fetch("${origin}/api/ai-gateway/v1/chat/completions", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer ai_xxx"
  },
  body: JSON.stringify({
    model: "deepseek-reasoner",
    messages: [
      { role: "user", content: "输出一份周报摘要" }
    ],
    reasoning_effort: "medium"
  })
})

const data = await response.json()
console.log(data.usage_summary)
console.log(data.choices?.[0]?.message?.content)`

const gatewayPythonExample = `import requests

resp = requests.post(
    "${origin}/api/ai-gateway/v1/chat/completions",
    headers={
        "Authorization": "Bearer ai_xxx",
        "Content-Type": "application/json",
    },
    json={
        "model": "deepseek-v4-flash",
        "messages": [
            {"role": "user", "content": "生成 3 条商品卖点"}
        ],
    },
    timeout=120,
)
data = resp.json()
print(data.get("usage_summary"))
print(data["choices"][0]["message"]["content"])`

const gatewayMediaExample = `curl -X POST ${origin}/api/ai-gateway/v1/media/generations \\
  -H "Authorization: Bearer ai_xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "qwen-image-2.0-pro",
    "prompt": "一张北欧客厅海报，柔和自然光",
    "images": [],
    "parameters": {
      "size": "1328x1328",
      "watermark": false
    }
  }'

curl ${origin}/api/ai-gateway/v1/media/tasks/task_xxx \\
  -H "Authorization: Bearer ai_xxx"`

const gatewayResponseExample = `{
  "id": "chatcmpl_xxx",
  "model": "deepseek-reasoner",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "这是生成结果"
      }
    }
  ],
  "reasoning_content": "我先分析业务目标、受众和表达风格...",
  "usage_summary": {
    "input_tokens": 123,
    "output_tokens": 456,
    "total_tokens": 579,
    "estimated_cost": 0.0347,
    "currency": "CNY"
  }
}`

const gatewayFieldDocs = [
  { field: 'model', required: '是', description: '要调用的模型名，必须在当前 API Key 白名单内。', example: 'deepseek-v4-pro / deepseek-reasoner' },
  { field: 'messages', required: '聊天必填', description: '聊天消息数组，兼容 OpenAI 风格。', example: '[{"role":"user","content":"你好"}]' },
  { field: 'reasoning_effort', required: '否', description: 'DeepSeek 推理模型可选，控制推理强度。', example: 'low / medium / high' },
  { field: 'prompt', required: '媒体必填', description: '图片或视频生成提示词。', example: '北欧风客厅海报' },
  { field: 'images', required: '按模型', description: '媒体模型输入图，支持 base64 或公网 URL。', example: '["data:image/png;base64,..."]' },
  { field: 'parameters', required: '否', description: '模型高级参数，原样透传给对应提供商。', example: '{"size":"1328x1328"}' },
  { field: 'Authorization', required: '是', description: '统一鉴权头，格式为 Bearer API Key。', example: 'Bearer ai_xxx' },
  { field: 'usage_summary', required: '响应', description: '网关追加的统一计费信息，便于业务记账。', example: '{"total_tokens":579}' }
]

const gatewayErrorDocs = [
  { code: '400', scene: '参数错误', meaning: '缺少 model、messages、prompt 等必要字段。', fix: '先按字段说明补齐必填项，并确认 JSON 格式合法。' },
  { code: '401', scene: '鉴权失败', meaning: 'API Key 缺失、错误、被吊销或已过期。', fix: '确认 Authorization 头是否正确，必要时重新签发 Key。' },
  { code: '403', scene: '权限不足', meaning: '当前 Key 没有访问对应 scope 或模型。', fix: '检查 Key 的 allowed_scopes 和 allowed_models 配置。' },
  { code: '429', scene: '触发限流', meaning: '超过每小时请求限制。', fix: '降低调用频率，或提高该 Key 的 rate limit。' },
  { code: '500', scene: '上游异常', meaning: '模型供应商返回错误或网关内部调用失败。', fix: '查看请求明细里的 error_message 和 latency，结合日志排查。' }
]

const gatewayFaqs = [
  { q: 'API Key 应该放在哪里？', a: '放在服务端配置或密钥中心，不要直接放到浏览器页面、App 安装包或公开仓库里。' },
  { q: '如何统计每个业务方的费用？', a: '建议每个业务系统单独签发一个 Key，这样后台会自动按 Key 统计请求数、Token、费用和报表。' },
  { q: '为什么有些模型没有 Token？', a: '图片和视频模型通常按次计费，没有文本 Token 概念，网关会写入 request_cost 并累计 total_cost。' },
  { q: '如何排查某次调用失败？', a: '在 Key 详情里查看最近请求明细，重点看状态码、错误信息、模型名、耗时和费用字段。' }
]

// MiniMax 模型特性表
const minimaxModelFeatures = [
  { model: 'speech-2.8-hd', type: 'TTS', async: false, description: '高清语音合成（推荐），同步返回音频' },
  { model: 'speech-2.6-hd', type: 'TTS', async: false, description: '高清语音合成，同步返回音频' },
  { model: 'speech-02-hd', type: 'TTS', async: false, description: 'TTS HD 模型，同步返回音频' },
  { model: 'Hailuo-2.3-Fast', type: '视频', async: true, description: 'Hailuo 视频生成 Fast 版（768P 6s），异步任务' },
  { model: 'Hailuo-2.3', type: '视频', async: true, description: 'Hailuo 视频生成标准版（768P 6s），异步任务' },
  { model: 'Music-2.5', type: '音乐', async: true, description: '音乐生成（最长 5 分钟），异步任务' },
  { model: 'image-01', type: '图片', async: true, description: '图像生成，异步任务' }
]

const prettyJSON = (value) => JSON.stringify(value, null, 2)
const formatTime = (value) => value ? new Date(value).toLocaleString() : '-'
const formatCost = (value, currency = 'CNY') => `${currency} ${Number(value || 0).toFixed(4)}`

onMounted(() => {
  loadDocs(false)
  loadCatalog()
})
</script>

<style scoped>
.ai-gateway-page {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.hero {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 24px;
  border-radius: 22px;
  background: linear-gradient(135deg, #142033 0%, #365f5f 55%, #c88044 100%);
  color: #fff;
}

.hero h2 {
  margin: 0 0 10px;
}

.hero-actions {
  display: grid;
  grid-template-columns: minmax(220px, 280px);
  gap: 10px;
}

.card-header,
.plain-key-box {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.plain-key-box code,
.doc-json {
  display: block;
  padding: 14px;
  background: #0f1724;
  color: #dce7f7;
  border-radius: 14px;
  white-space: pre-wrap;
  word-break: break-all;
}

.logs-section {
  margin-top: 18px;
}

.docs-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.docs-card {
  border-radius: 22px;
}

.docs-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
}

.docs-panel h4,
.docs-api-list h4 {
  margin: 0 0 12px;
}

.docs-text {
  margin: 0;
  color: #475569;
  line-height: 1.75;
}

.docs-list {
  margin: 0;
  padding-left: 18px;
  color: #475569;
  line-height: 1.8;
}

.docs-tabs,
.docs-api-list {
  margin-top: 18px;
}

.docs-panel-stack h4 {
  margin: 0 0 12px;
}

.docs-steps {
  margin: 0;
  padding-left: 18px;
  color: #475569;
  line-height: 1.9;
}

.faq-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.faq-item h4 {
  margin: 0 0 8px;
}

.faq-item p {
  margin: 0;
  color: #475569;
  line-height: 1.8;
}

.test-card {
  border-radius: 22px;
}

.test-reply {
  color: #334155;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.test-err {
  color: #dc2626;
  font-size: 12px;
}

@media (max-width: 900px) {
  .hero {
    flex-direction: column;
  }

  .docs-grid {
    grid-template-columns: 1fr;
  }
}

.minimax-tasks-card {
  border-radius: 22px;
}

.minimax-task-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.media-preview-section {
  margin-top: 8px;
}

.media-preview-section h4 {
  margin: 0 0 12px;
}

.media-preview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.media-preview-item {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 12px;
  background: #fafafa;
}

.media-url-box {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.url-link {
  font-size: 11px;
  word-break: break-all;
}

.audio-preview audio,
.video-preview video {
  border-radius: 8px;
}

.result-json-section {
  margin-top: 8px;
}

.result-json-section h4 {
  margin: 0 0 12px;
}

/* Voice Cloning 样式 */
.voice-cloning-card {
  border-radius: 22px;
}

.voice-cloning-tabs {
  flex: 1;
  margin: 0 16px;
}

.voice-cloning-tabs .el-tabs__header {
  margin-bottom: 0;
}

.voice-upload-section,
.voice-list-section,
.voice-tts-section {
  padding: 16px 0;
}

.tts-result {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #ebeef5;
}

.tts-result h4 {
  margin: 0 0 12px;
}

/* 麦克风录制样式 */
.mic-recorder {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mic-recorder-controls {
  display: flex;
  gap: 8px;
}

.recording-status {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #f56c6c;
  font-weight: 500;
}

.recording-dot {
  width: 12px;
  height: 12px;
  background-color: #f56c6c;
  border-radius: 50%;
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
  100% {
    opacity: 1;
  }
}

.recording-preview {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  background-color: #f5f7fa;
  border-radius: 8px;
}

.recording-actions {
  display: flex;
  gap: 8px;
}

.recording-info {
  display: flex;
  gap: 8px;
}

.recording-hint {
  max-width: 400px;
}
</style>
