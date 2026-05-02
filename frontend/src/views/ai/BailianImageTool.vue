<template>
  <div class="bailian-page">
    <section class="hero-card">
      <div>
        <p class="eyebrow">ALIYUN BAILIAN IMAGE STUDIO</p>
        <h2>百炼图片模型调试台</h2>
        <p class="hero-desc">
          支持模型选择、图片上传、Prompt 调试、任务列表、全流程流水和开放 API 文档。
        </p>
      </div>
      <div class="hero-actions">
        <el-input
          v-model="adminPassword"
          type="password"
          show-password
          placeholder="输入管理密码后才能调用"
          @keyup.enter="savePassword"
        />
        <el-button type="primary" @click="savePassword">保存密码</el-button>
        <el-button @click="fetchDocs">API 文档</el-button>
      </div>
    </section>

    <section class="main-grid">
      <el-card class="panel-card">
        <template #header>
          <div class="panel-header">
            <span>任务调试</span>
            <div class="panel-actions">
              <el-button text @click="loadModels">刷新模型</el-button>
            </div>
          </div>
        </template>

        <div v-if="!authed" class="locked-box">
          <el-icon><Lock /></el-icon>
          <p>先输入管理密码，再加载模型和提交任务。</p>
        </div>

        <el-form v-else label-position="top" class="tool-form">
          <el-form-item label="模型">
            <el-select v-model="form.model" class="w-full" @change="onModelChange">
              <el-option
                v-for="model in models"
                :key="model.name"
                :value="model.name"
                :label="model.name"
                :disabled="!model.enabled || model.is_expired"
              >
                <div class="model-option">
                  <span>{{ model.name }}</span>
                  <span class="model-meta">
                    {{ model.type }} | 剩余 {{ model.remaining_quota }}/{{ model.total_quota || '∞' }}
                  </span>
                </div>
              </el-option>
            </el-select>
            <div v-if="selectedModel" class="model-hint">
              <span>{{ selectedModel.description || '未填写说明' }}</span>
              <span>截止: {{ formatDate(selectedModel.expires_at) || '未限制' }}</span>
            </div>
          </el-form-item>

          <el-form-item label="Prompt">
            <el-input v-model="form.prompt" type="textarea" :rows="5" placeholder="描述你要生成或编辑的视频/图片效果" />
          </el-form-item>

          <el-form-item label="Negative Prompt">
            <el-input
              v-model="form.negative_prompt"
              type="textarea"
              :rows="3"
              placeholder="例如：模糊、低质量、畸形手、文字错乱"
            />
          </el-form-item>

          <el-form-item :label="selectedModel?.type === 'image2video' ? '参考图（1 张）' : '参考图（最多 3 张）'">
            <input ref="fileInputRef" type="file" accept="image/*" multiple class="hidden-input" @change="handleFileChange" />
            <div class="upload-toolbar">
              <el-button @click="triggerFileSelect">上传图片</el-button>
              <el-button text @click="clearImages">清空</el-button>
            </div>
            <div v-if="imagePreviews.length" class="preview-grid">
              <div v-for="(img, index) in imagePreviews" :key="index" class="preview-card">
                <img :src="img" alt="preview" />
                <button class="preview-remove" @click.prevent="removeImage(index)">×</button>
              </div>
            </div>
            <div v-else class="empty-upload">
              支持上传图片并转为 base64 请求。也可以在高级参数里传公网 URL。
            </div>
          </el-form-item>

          <div class="inline-grid">
            <el-form-item v-if="selectedModel?.type !== 'image2video'" label="尺寸">
              <el-input v-model="form.size" placeholder="例如 1328*1328 / 1024*1024" />
            </el-form-item>
            <el-form-item v-if="selectedModel?.type !== 'image2video'" label="张数">
              <el-input-number v-model="form.count" :min="1" :max="4" class="w-full" />
            </el-form-item>
            <el-form-item v-if="selectedModel?.type === 'image2video'" label="分辨率">
              <el-input v-model="form.resolution" placeholder="例如 1280x720" />
            </el-form-item>
            <el-form-item v-if="selectedModel?.type === 'image2video'" label="时长（秒）">
              <el-input-number v-model="form.duration" :min="1" :max="12" class="w-full" />
            </el-form-item>
          </div>

          <div class="inline-grid">
            <el-form-item label="Seed">
              <el-input-number v-model="form.seed" :min="1" :max="999999999" class="w-full" />
            </el-form-item>
            <el-form-item v-if="selectedModel?.type === 'image2video'" label="FPS">
              <el-input-number v-model="form.fps" :min="1" :max="60" class="w-full" />
            </el-form-item>
            <el-form-item label="等待轮询秒数">
              <el-input-number v-model="form.wait_seconds" :min="0" :max="180" class="w-full" />
            </el-form-item>
          </div>

          <div class="inline-grid">
            <el-form-item label="Client Name">
              <el-input v-model="form.client_name" placeholder="给其他业务区分调用来源" />
            </el-form-item>
            <el-form-item label="Client Request ID">
              <el-input v-model="form.client_request_id" placeholder="可选，业务侧关联单号" />
            </el-form-item>
          </div>

          <el-form-item label="高级参数 JSON">
            <el-input
              v-model="parametersText"
              type="textarea"
              :rows="6"
              placeholder='{"style":"photorealistic","watermark":false}'
            />
          </el-form-item>

          <el-form-item>
            <div class="submit-row">
              <el-switch v-model="form.auto_poll" active-text="自动轮询异步结果" />
              <el-button type="primary" :loading="submitting" @click="submitTask">提交任务</el-button>
            </div>
          </el-form-item>
        </el-form>
      </el-card>

      <el-card class="panel-card result-card">
        <template #header>
          <div class="panel-header">
            <span>最新任务</span>
            <div class="panel-actions">
              <el-button text @click="loadTasks">刷新列表</el-button>
            </div>
          </div>
        </template>

        <div v-if="currentTask" class="current-task">
          <div class="task-topline">
            <el-tag :type="statusTagType(currentTask.status)">{{ currentTask.status }}</el-tag>
            <span>{{ currentTask.model }}</span>
            <span>{{ formatTime(currentTask.created_at) }}</span>
          </div>
          <p class="task-prompt">{{ currentTask.prompt }}</p>
          <div v-if="currentInputImages.length" class="detail-block compact-block">
            <h4>输入图片</h4>
            <div class="asset-grid">
              <div v-for="(asset, index) in currentInputImages" :key="`current-input-${index}`" class="asset-card">
                <img :src="asset.value" :alt="asset.label || `input-${index}`" @click="openMediaPreview(asset)" />
                <div class="asset-toolbar">
                  <el-button size="small" @click="openMediaPreview(asset)">预览</el-button>
                  <el-button size="small" @click="downloadAsset(asset)">下载</el-button>
                </div>
              </div>
            </div>
          </div>
          <div class="asset-grid" v-if="currentAssets.length">
            <div v-for="(asset, index) in currentAssets" :key="`current-result-${index}`" class="asset-card">
              <img
                v-if="asset.type !== 'video'"
                :src="asset.value"
                :alt="`asset-${index}`"
                @click="openMediaPreview(asset)"
              />
              <video v-else :src="asset.value" controls />
              <div class="asset-toolbar">
                <el-button size="small" @click="openMediaPreview(asset)">预览</el-button>
                <el-button size="small" @click="downloadAsset(asset)">下载</el-button>
              </div>
            </div>
          </div>
          <el-alert v-else title="当前任务还没有返回媒体结果" type="info" :closable="false" />

          <div class="current-actions">
            <el-button v-if="canPoll(currentTask)" @click="pollTask(currentTask)">轮询状态</el-button>
            <el-button @click="openTask(currentTask)">查看详情</el-button>
          </div>
        </div>
        <div v-else class="empty-upload">还没有任务记录。</div>
      </el-card>
    </section>

    <el-card class="panel-card table-card">
      <template #header>
        <div class="panel-header">
          <span>任务列表</span>
          <div class="panel-actions">
            <el-select v-model="filters.status" clearable placeholder="状态筛选" style="width: 140px" @change="loadTasks">
              <el-option label="queued" value="queued" />
              <el-option label="submitted" value="submitted" />
              <el-option label="running" value="running" />
              <el-option label="succeeded" value="succeeded" />
              <el-option label="failed" value="failed" />
            </el-select>
            <el-select v-model="filters.model" clearable placeholder="模型筛选" style="width: 220px" @change="loadTasks">
              <el-option v-for="model in models" :key="model.name" :label="model.name" :value="model.name" />
            </el-select>
          </div>
        </div>
      </template>

      <el-table :data="tasks" v-loading="loadingTasks" stripe>
        <el-table-column prop="created_at" label="时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="model" label="模型" min-width="220" />
        <el-table-column prop="task_type" label="类型" width="120" />
        <el-table-column prop="status" label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="配额" width="120">
          <template #default="{ row }">{{ row.quota_used }}/{{ row.quota_total || '∞' }}</template>
        </el-table-column>
        <el-table-column label="Prompt" min-width="320">
          <template #default="{ row }">
            <span class="prompt-cell">{{ row.prompt }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button text @click="openTask(row)">详情</el-button>
            <el-button v-if="canPoll(row)" text @click="pollTask(row)">轮询</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="panel-card docs-page-card">
      <template #header>
        <div class="panel-header">
          <span>接入文档</span>
          <div class="panel-actions">
            <el-button text @click="fetchDocs">刷新文档</el-button>
          </div>
        </div>
      </template>

      <div class="docs-summary-grid">
        <div class="detail-block">
          <h4>调用说明</h4>
          <p class="docs-paragraph">
            页面内除了手工调试，也开放标准任务接口给其他业务使用。业务方传入管理密码、模型、Prompt、图片和高级参数后，
            后端会统一校验额度、写任务表、记录请求快照、保存响应快照，并持续沉淀任务流水。
          </p>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="鉴权 Header">X-Admin-Password: your-password</el-descriptions-item>
            <el-descriptions-item label="提交任务">POST /api/bailian/tasks</el-descriptions-item>
            <el-descriptions-item label="任务列表">GET /api/bailian/tasks</el-descriptions-item>
            <el-descriptions-item label="任务详情">GET /api/bailian/tasks/:id</el-descriptions-item>
            <el-descriptions-item label="状态轮询">POST /api/bailian/tasks/:id/poll</el-descriptions-item>
          </el-descriptions>
        </div>

        <div class="detail-block">
          <h4>链路回溯</h4>
          <ul class="doc-list">
            <li>任务列表可按模型和状态筛选，直接查看历史。</li>
            <li>任务详情包含请求体、响应体、外部任务号、结果媒体和全流程事件。</li>
            <li>支持图片 base64 上传，也支持在参数中传公网图片地址。</li>
            <li>本地额度会按模型计数并校验截止时间，避免免费额度透支。</li>
          </ul>
        </div>
      </div>

      <el-tabs>
        <el-tab-pane label="快速开始">
          <div class="detail-block">
            <h4>接入步骤</h4>
            <ol class="doc-steps">
              <li>先拿到管理密码，并确认要调用的模型已启用且还有本地剩余额度。</li>
              <li>准备 Prompt、可选 Negative Prompt、可选图片和高级参数 JSON。</li>
              <li>调用 `POST /api/bailian/tasks` 创建任务。</li>
              <li>如果是异步模型，继续调用 `POST /api/bailian/tasks/:id/poll` 或 `GET /api/bailian/tasks/:id` 查状态。</li>
              <li>排查问题时直接看任务详情里的请求快照、响应快照和任务流水。</li>
            </ol>
          </div>
        </el-tab-pane>
        <el-tab-pane label="字段说明">
          <el-table :data="bailianFieldDocs" size="small" stripe>
            <el-table-column prop="field" label="字段" width="220" />
            <el-table-column prop="required" label="必填" width="90" />
            <el-table-column prop="description" label="说明" min-width="260" />
            <el-table-column prop="example" label="示例" min-width="220" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="cURL">
          <pre class="json-box">{{ curlExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="JavaScript">
          <pre class="json-box">{{ jsExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="Python">
          <pre class="json-box">{{ pythonExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="任务流程">
          <pre class="json-box">{{ flowExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="响应示例">
          <pre class="json-box">{{ bailianResponseExample }}</pre>
        </el-tab-pane>
        <el-tab-pane label="错误码">
          <el-table :data="bailianErrorDocs" size="small" stripe>
            <el-table-column prop="code" label="状态码" width="100" />
            <el-table-column prop="scene" label="场景" width="180" />
            <el-table-column prop="meaning" label="含义" min-width="220" />
            <el-table-column prop="fix" label="处理建议" min-width="260" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="常见问题">
          <div class="faq-list">
            <div v-for="item in bailianFaqs" :key="item.q" class="faq-item">
              <h4>{{ item.q }}</h4>
              <p>{{ item.a }}</p>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>

      <div v-if="docs" class="detail-block">
        <h4>接口清单</h4>
        <el-descriptions :column="1" border>
          <el-descriptions-item v-for="route in docs.routes" :key="`${route.method}-${route.path}`" :label="`${route.method} ${route.path}`">
            {{ route.description }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-card>

    <el-dialog v-model="taskDialogVisible" width="960px" destroy-on-close>
      <template #header>
        <div class="dialog-title">
          <span>任务详情</span>
          <el-tag v-if="taskDetail" :type="statusTagType(taskDetail.status)">{{ taskDetail.status }}</el-tag>
        </div>
      </template>

      <div v-if="taskDetail" class="detail-layout">
        <div class="detail-block">
          <h4>基本信息</h4>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="ID">{{ taskDetail.id }}</el-descriptions-item>
            <el-descriptions-item label="模型">{{ taskDetail.model }}</el-descriptions-item>
            <el-descriptions-item label="类型">{{ taskDetail.task_type }}</el-descriptions-item>
            <el-descriptions-item label="来源">{{ taskDetail.source }}</el-descriptions-item>
            <el-descriptions-item label="状态">{{ taskDetail.status }}</el-descriptions-item>
            <el-descriptions-item label="Vendor 状态">{{ taskDetail.vendor_status || '-' }}</el-descriptions-item>
            <el-descriptions-item label="External Task ID">{{ taskDetail.external_task_id || '-' }}</el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ formatTime(taskDetail.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="Client Name">{{ taskDetail.client_name || '-' }}</el-descriptions-item>
            <el-descriptions-item label="额度快照">{{ taskDetail.quota_used }}/{{ taskDetail.quota_total || '∞' }}</el-descriptions-item>
          </el-descriptions>
        </div>

        <div class="detail-block">
          <h4>Prompt</h4>
          <pre class="json-box">{{ taskDetail.prompt }}</pre>
        </div>

        <div class="detail-block" v-if="taskInputImages.length">
          <h4>输入图片</h4>
          <div class="asset-grid">
            <div v-for="(asset, index) in taskInputImages" :key="`detail-input-${index}`" class="asset-card">
              <img :src="asset.value" :alt="asset.label || `input-${index}`" @click="openMediaPreview(asset)" />
              <div class="asset-toolbar">
                <el-button size="small" @click="openMediaPreview(asset)">预览</el-button>
                <el-button size="small" @click="downloadAsset(asset)">下载</el-button>
              </div>
            </div>
          </div>
        </div>

        <div class="detail-block" v-if="taskAssets.length">
          <h4>结果媒体</h4>
          <div class="asset-grid">
            <div v-for="(asset, index) in taskAssets" :key="`detail-result-${index}`" class="asset-card">
              <img
                v-if="asset.type !== 'video'"
                :src="asset.value"
                :alt="`asset-${index}`"
                @click="openMediaPreview(asset)"
              />
              <video v-else :src="asset.value" controls />
              <div class="asset-toolbar">
                <el-button size="small" @click="openMediaPreview(asset)">预览</el-button>
                <el-button size="small" @click="downloadAsset(asset)">下载</el-button>
              </div>
            </div>
          </div>
        </div>

        <div class="detail-block">
          <h4>请求快照</h4>
          <pre class="json-box">{{ prettyJSON(taskDetail.request_body) }}</pre>
        </div>

        <div class="detail-block">
          <h4>响应快照</h4>
          <pre class="json-box">{{ prettyJSON(taskDetail.response_body) }}</pre>
        </div>

        <div class="detail-block">
          <h4>任务流水</h4>
          <el-timeline>
            <el-timeline-item
              v-for="event in taskEvents"
              :key="event.id"
              :timestamp="formatTime(event.created_at)"
              :type="timelineType(event.status)"
            >
              <div class="event-line">
                <strong>{{ event.stage }}</strong>
                <span>{{ event.message }}</span>
              </div>
              <pre class="json-box small">{{ prettyJSON(event.payload) }}</pre>
            </el-timeline-item>
          </el-timeline>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="docsVisible" title="通用 API 文档" width="860px">
      <div v-if="docs" class="docs-layout">
        <el-alert title="所有调用都必须携带管理密码，避免免费额度被滥用。" type="warning" :closable="false" />
        <el-descriptions :column="1" border>
          <el-descriptions-item v-for="route in docs.routes" :key="`${route.method}-${route.path}`" :label="`${route.method} ${route.path}`">
            {{ route.description }}
          </el-descriptions-item>
        </el-descriptions>

        <div class="detail-block">
          <h4>鉴权方式</h4>
          <pre class="json-box">{{ prettyJSON(docs.auth) }}</pre>
        </div>

        <div class="detail-block">
          <h4>请求示例</h4>
          <pre class="json-box">{{ prettyJSON(docs.request_example) }}</pre>
        </div>

        <div class="detail-block">
          <h4>cURL 示例</h4>
          <pre class="json-box">{{ curlExample }}</pre>
        </div>

        <div class="detail-block">
          <h4>注意事项</h4>
          <ul class="doc-list">
            <li v-for="(item, index) in docs.notes" :key="index">{{ item }}</li>
          </ul>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="mediaPreviewVisible" :title="mediaPreviewTitle" width="min(92vw, 1180px)" destroy-on-close>
      <div v-if="mediaPreviewAsset" class="media-preview-body">
        <img
          v-if="mediaPreviewAsset.type !== 'video'"
          class="media-preview-image"
          :src="mediaPreviewAsset.value"
          :alt="mediaPreviewTitle"
        />
        <video v-else class="media-preview-video" :src="mediaPreviewAsset.value" controls autoplay />
      </div>
      <template #footer>
        <el-button @click="mediaPreviewVisible = false">关闭</el-button>
        <el-button type="primary" @click="downloadAsset(mediaPreviewAsset)">下载</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Lock } from '@element-plus/icons-vue'

const API_BASE = '/api'
const PASSWORD_KEY = 'bailian_admin_password'

const adminPassword = ref(localStorage.getItem(PASSWORD_KEY) || '')
const authed = ref(false)
const models = ref([])
const tasks = ref([])
const docs = ref(null)
const currentTask = ref(null)
const taskDetail = ref(null)
const taskEvents = ref([])
const docsVisible = ref(false)
const taskDialogVisible = ref(false)
const loadingTasks = ref(false)
const submitting = ref(false)
const fileInputRef = ref(null)
const mediaPreviewVisible = ref(false)
const mediaPreviewAsset = ref(null)
const mediaPreviewTitle = ref('媒体预览')
const imagePreviews = ref([])
const parametersText = ref('{\n  "watermark": false\n}')

const filters = ref({
  model: '',
  status: ''
})

const form = ref({
  model: 'qwen-image-2.0-pro',
  prompt: '',
  negative_prompt: '',
  size: '1328*1328',
  count: 1,
  seed: null,
  duration: 5,
  resolution: '1280x720',
  fps: 24,
  wait_seconds: 30,
  auto_poll: true,
  client_name: 'devtools-console',
  client_request_id: ''
})

const selectedModel = computed(() => models.value.find(item => item.name === form.value.model) || null)
const currentAssets = computed(() => parseAssets(currentTask.value?.result_json))
const taskAssets = computed(() => parseAssets(taskDetail.value?.result_json))
const currentInputImages = computed(() => parseTaskInputImages(currentTask.value))
const taskInputImages = computed(() => parseTaskInputImages(taskDetail.value))
const origin = typeof window !== 'undefined' ? window.location.origin : ''
const curlExample = computed(() => {
  const body = {
    admin_password: 'your-password',
    model: 'qwen-image-2.0-pro',
    prompt: '一只橘猫穿着宇航服，电影级海报风格',
    images: ['data:image/png;base64,...'],
    size: '1328*1328',
    auto_poll: true,
    wait_seconds: 30,
    client_name: 'marketing-site'
  }
  return `curl -X POST ${origin}/api/bailian/tasks \\
  -H "Content-Type: application/json" \\
  -H "X-Admin-Password: your-password" \\
  -d '${JSON.stringify(body, null, 2)}'`
})
const jsExample = computed(() => `const response = await fetch("${origin}/api/bailian/tasks", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "X-Admin-Password": "your-password"
  },
  body: JSON.stringify({
    model: "qwen-image-2.0-pro",
    prompt: "一张电商海报，极简高级感",
    images: ["data:image/png;base64,..."],
    parameters: {
      watermark: false,
      style: "photorealistic"
    },
    auto_poll: true,
    wait_seconds: 30,
    client_name: "cms-service",
    client_request_id: "order-1001"
  })
})

const data = await response.json()
console.log(data.task)`)

const pythonExample = computed(() => `import requests

resp = requests.post(
    "${origin}/api/bailian/tasks",
    headers={
        "Content-Type": "application/json",
        "X-Admin-Password": "your-password",
    },
    json={
        "model": "wan2.6-i2v-flash",
        "prompt": "让人物朝镜头微笑并抬手打招呼",
        "images": ["data:image/png;base64,..."],
        "duration": 5,
        "resolution": "1280x720",
        "auto_poll": False,
        "client_name": "video-worker",
    },
    timeout=180,
)
task = resp.json()["task"]
print(task["id"], task["status"])`)

const flowExample = computed(() => `1. 业务调用 POST /api/bailian/tasks 提交任务
2. 后端校验管理密码、模型状态、本地额度和过期时间
3. 写入任务表和事件流水，保存请求快照
4. 调用百炼上游接口并保存响应快照
5. 异步任务继续调用 POST /api/bailian/tasks/:id/poll 轮询状态
6. 最终通过 GET /api/bailian/tasks/:id 获取结果媒体和完整流水`)
const bailianResponseExample = computed(() => `{
  "task": {
    "id": "task_xxx",
    "model": "qwen-image-2.0-pro",
    "status": "succeeded",
    "task_type": "multimodal",
    "external_task_id": "dashscope_xxx",
    "result_json": {
      "assets": [
        { "type": "image", "value": "https://..." }
      ]
    }
  }
}`)

const bailianFieldDocs = [
  { field: 'model', required: '是', description: '百炼模型名，必须是当前后台已启用模型。', example: 'qwen-image-2.0-pro' },
  { field: 'prompt', required: '是', description: '生成或编辑图片/视频的主要提示词。', example: '一张高级感产品海报' },
  { field: 'negative_prompt', required: '否', description: '不希望出现的内容，可提升结果稳定性。', example: '模糊, 畸形手, 低质量' },
  { field: 'images', required: '按模型', description: '输入图片数组，支持 base64 Data URL 或公网地址。', example: '["data:image/png;base64,..."]' },
  { field: 'size / resolution', required: '按模型', description: '图片尺寸或视频分辨率，建议与模型默认值一致。图片建议传 width*height。', example: '1328*1328 / 1280x720' },
  { field: 'count / duration / fps', required: '否', description: '输出张数、视频时长和帧率。', example: '1 / 5 / 24' },
  { field: 'parameters', required: '否', description: '高级 JSON 参数，原样透传给模型。', example: '{"watermark":false}' },
  { field: 'client_name', required: '否', description: '业务来源标识，方便任务历史检索。', example: 'marketing-site' },
  { field: 'client_request_id', required: '否', description: '业务侧自己的请求单号，便于串联外部日志。', example: 'order-1001' },
  { field: 'X-Admin-Password', required: '是', description: '调用鉴权头，防止额度被滥用。', example: 'your-password' }
]

const bailianErrorDocs = [
  { code: '400', scene: '参数错误', meaning: '缺少 Prompt、模型或图片数量不符合要求。', fix: '按字段说明检查 body，特别是模型类型对应的输入图数量。' },
  { code: '401', scene: '密码错误', meaning: '管理密码缺失或不正确。', fix: '确认请求头 X-Admin-Password 或 body 中密码是否一致。' },
  { code: '403', scene: '模型不可用', meaning: '模型被禁用、过期或免费额度已耗尽。', fix: '在页面查看模型剩余额度和到期时间，必要时调整配置。' },
  { code: '404', scene: '任务不存在', meaning: '任务 ID 错误或已被清理。', fix: '确认任务 ID 是否正确，并在保留期内查询。' },
  { code: '500', scene: '上游失败', meaning: '百炼返回错误或轮询阶段失败。', fix: '查看任务详情中的请求快照、响应快照和任务流水定位原因。' }
]

const bailianFaqs = [
  { q: '图片应该怎么传？', a: '推荐直接传 Data URL 格式的 base64，最稳定；如果业务已有公网文件地址，也可以在 images 或 parameters 中传 URL。' },
  { q: '为什么任务创建成功但结果还没出来？', a: '很多图片和视频模型是异步的，需要继续轮询任务状态，直到状态变成 succeeded 或 failed。' },
  { q: '如何追踪一条业务请求？', a: '创建任务时带上 client_name 和 client_request_id，之后就能在历史任务和详情页里按业务标识回溯。' },
  { q: '怎么知道额度快用完了？', a: '模型列表和任务记录里都会显示本地额度快照，到达 total_quota 或 expires_at 后后台会直接拒绝调用。' }
]

const savePassword = async () => {
  if (!adminPassword.value) {
    ElMessage.error('请输入管理密码')
    return
  }
  localStorage.setItem(PASSWORD_KEY, adminPassword.value)
  try {
    await loadModels()
    authed.value = true
    ElMessage.success('管理密码已保存')
    loadTasks()
  } catch (err) {
    localStorage.removeItem(PASSWORD_KEY)
  }
}

const authHeaders = () => ({
  'Content-Type': 'application/json',
  'X-Admin-Password': adminPassword.value
})

const loadModels = async () => {
  const res = await fetch(`${API_BASE}/bailian/models`, {
    headers: { 'X-Admin-Password': adminPassword.value }
  })
  const data = await res.json()
  if (!res.ok) {
    throw new Error(data.error || '加载模型失败')
  }
  models.value = data.models || []
  if (!models.value.find(item => item.name === form.value.model)) {
    form.value.model = models.value[0]?.name || ''
  }
  onModelChange()
}

const loadTasks = async () => {
  if (!adminPassword.value) return
  loadingTasks.value = true
  try {
    const params = new URLSearchParams()
    if (filters.value.model) params.set('model', filters.value.model)
    if (filters.value.status) params.set('status', filters.value.status)
    const query = params.toString()
    const res = await fetch(`${API_BASE}/bailian/tasks${query ? `?${query}` : ''}`, {
      headers: { 'X-Admin-Password': adminPassword.value }
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载任务失败')
    tasks.value = data.tasks || []
    currentTask.value = tasks.value[0] || null
  } catch (err) {
    ElMessage.error(err.message || '加载任务失败')
  } finally {
    loadingTasks.value = false
  }
}

const fetchDocs = async (showDialog = true) => {
  try {
    const res = await fetch(`${API_BASE}/bailian/docs`)
    const data = await res.json()
    docs.value = data
    if (showDialog) {
      docsVisible.value = true
    }
  } catch (err) {
    ElMessage.error('加载文档失败')
  }
}

const submitTask = async () => {
  if (!form.value.model || !form.value.prompt.trim()) {
    ElMessage.error('模型和 Prompt 必填')
    return
  }

  let parameters = {}
  try {
    parameters = parametersText.value ? JSON.parse(parametersText.value) : {}
  } catch (err) {
    ElMessage.error('高级参数 JSON 格式不合法')
    return
  }

  submitting.value = true
  try {
    const payload = {
      ...form.value,
      admin_password: adminPassword.value,
      images: imagePreviews.value,
      parameters
    }
    const res = await fetch(`${API_BASE}/bailian/tasks`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify(payload)
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '提交失败')
    currentTask.value = data.task
    tasks.value = [data.task, ...tasks.value.filter(item => item.id !== data.task.id)]
    ElMessage.success('任务已提交')
    if (data.task) {
      await openTask(data.task)
    }
  } catch (err) {
    ElMessage.error(err.message || '提交失败')
  } finally {
    submitting.value = false
  }
}

const openTask = async (task) => {
  try {
    const res = await fetch(`${API_BASE}/bailian/tasks/${task.id}`, {
      headers: { 'X-Admin-Password': adminPassword.value }
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '获取任务详情失败')
    taskDetail.value = data.task
    taskEvents.value = data.events || []
    taskDialogVisible.value = true
  } catch (err) {
    ElMessage.error(err.message || '获取任务详情失败')
  }
}

const pollTask = async (task) => {
  try {
    const res = await fetch(`${API_BASE}/bailian/tasks/${task.id}/poll`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ wait_seconds: 1, admin_password: adminPassword.value })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '轮询失败')
    currentTask.value = data.task
    taskDetail.value = data.task
    taskEvents.value = data.events || []
    await loadTasks()
    ElMessage.success('状态已刷新')
  } catch (err) {
    ElMessage.error(err.message || '轮询失败')
  }
}

const handleFileChange = async (event) => {
  const files = Array.from(event.target.files || [])
  const maxCount = selectedModel.value?.type === 'image2video' ? 1 : 3
  const slots = Math.max(maxCount - imagePreviews.value.length, 0)
  const picked = files.slice(0, slots)
  const dataUrls = await Promise.all(picked.map(fileToDataUrl))
  imagePreviews.value = [...imagePreviews.value, ...dataUrls]
  event.target.value = ''
}

const triggerFileSelect = () => fileInputRef.value?.click()
const clearImages = () => { imagePreviews.value = [] }
const removeImage = (index) => { imagePreviews.value.splice(index, 1) }

const onModelChange = () => {
  if (!selectedModel.value) return
  if (selectedModel.value.type === 'image2video') {
    form.value.resolution = selectedModel.value.default_size || '1280x720'
    imagePreviews.value = imagePreviews.value.slice(0, 1)
  } else {
    form.value.size = selectedModel.value.default_size || '1328*1328'
  }
}

const fileToDataUrl = (file) => new Promise((resolve, reject) => {
  const reader = new FileReader()
  reader.onload = () => resolve(reader.result)
  reader.onerror = reject
  reader.readAsDataURL(file)
})

const parseAssets = (raw) => {
  if (!raw) return []
  try {
    const parsed = typeof raw === 'string' ? JSON.parse(raw) : raw
    return Array.isArray(parsed.assets) ? parsed.assets : []
  } catch (err) {
    return []
  }
}

const parseTaskInputImages = (task) => {
  if (!task) return []
  const images = []
  const seen = new Set()
  const addImage = (value, fallbackLabel = '输入图片') => {
    if (typeof value !== 'string') return
    const trimmed = value.trim()
    if (!trimmed) return
    if (!(trimmed.startsWith('data:image') || trimmed.startsWith('http://') || trimmed.startsWith('https://'))) return
    if (seen.has(trimmed)) return
    seen.add(trimmed)
    images.push({
      type: 'image',
      value: trimmed,
      label: fallbackLabel
    })
  }

  try {
    const parsed = typeof task.input_images === 'string' ? JSON.parse(task.input_images || '[]') : task.input_images
    if (Array.isArray(parsed)) {
      parsed.forEach((item, index) => {
        if (typeof item === 'string') {
          addImage(item, `输入图片 ${index + 1}`)
          return
        }
        if (item && typeof item === 'object') {
          addImage(item.value || item.url || item.image, item.label || `输入图片 ${index + 1}`)
        }
      })
    }
  } catch (err) {
    // ignore invalid historic payloads
  }

  try {
    const request = typeof task.request_body === 'string' ? JSON.parse(task.request_body || '{}') : task.request_body
    const messages = request?.input?.messages
    if (Array.isArray(messages)) {
      messages.forEach((message) => {
        if (!Array.isArray(message?.content)) return
        message.content.forEach((item, index) => addImage(item?.image, `输入图片 ${index + 1}`))
      })
    }
    addImage(request?.input?.img_url, '输入图片 1')
  } catch (err) {
    // request snapshot may be truncated
  }

  return images
}

const prettyJSON = (value) => {
  if (!value) return '{}'
  try {
    const parsed = typeof value === 'string' ? JSON.parse(value) : value
    return JSON.stringify(parsed, null, 2)
  } catch (err) {
    return String(value)
  }
}

const openMediaPreview = (asset) => {
  if (!asset?.value) return
  mediaPreviewAsset.value = asset
  mediaPreviewTitle.value = asset.label || (asset.type === 'video' ? '视频预览' : '图片预览')
  mediaPreviewVisible.value = true
}

const downloadAsset = async (asset) => {
  if (!asset?.value) return
  try {
    const url = asset.value
    const filename = buildAssetFilename(asset)
    if (url.startsWith('data:')) {
      triggerDownload(url, filename)
      return
    }
    const res = await fetch(url)
    if (!res.ok) {
      throw new Error('下载失败')
    }
    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)
    triggerDownload(blobUrl, filename)
    setTimeout(() => URL.revokeObjectURL(blobUrl), 1000)
  } catch (err) {
    window.open(asset.value, '_blank', 'noopener')
  }
}

const buildAssetFilename = (asset) => {
  const extension = asset?.type === 'video' ? 'mp4' : detectImageExtension(asset?.value)
  const taskId = taskDetail.value?.id || currentTask.value?.id || 'bailian'
  return `${taskId}.${extension}`
}

const detectImageExtension = (value) => {
  if (typeof value !== 'string') return 'png'
  const mimeMatch = value.match(/^data:image\/([a-zA-Z0-9.+-]+);base64,/)
  if (mimeMatch?.[1]) {
    return mimeMatch[1] === 'jpeg' ? 'jpg' : mimeMatch[1]
  }
  try {
    const pathname = new URL(value).pathname
    const extension = pathname.split('.').pop()
    if (extension && extension.length <= 5) {
      return extension
    }
  } catch (err) {
    // ignore invalid URLs
  }
  return 'png'
}

const triggerDownload = (href, filename) => {
  const link = document.createElement('a')
  link.href = href
  link.download = filename
  link.target = '_blank'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

const formatTime = (value) => {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

const formatDate = (value) => {
  if (!value) return ''
  return new Date(value).toLocaleDateString()
}

const statusTagType = (status) => {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'running') return 'warning'
  return 'info'
}

const timelineType = (status) => {
  if (status === 'failed') return 'danger'
  if (status === 'succeeded') return 'success'
  return 'primary'
}

const canPoll = (task) => ['queued', 'submitted', 'running'].includes(task?.status)

onMounted(async () => {
  fetchDocs(false)
  if (!adminPassword.value) return
  try {
    await loadModels()
    authed.value = true
    loadTasks()
  } catch (err) {
    authed.value = false
    ElMessage.error(err.message || '管理密码校验失败')
  }
})
</script>

<style scoped>
.bailian-page {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 18px;
  background:
    radial-gradient(circle at top right, rgba(255, 180, 70, 0.22), transparent 28%),
    radial-gradient(circle at left center, rgba(32, 174, 255, 0.16), transparent 34%),
    linear-gradient(180deg, #f6f0e5 0%, #f4f7fb 100%);
  min-height: 100%;
}

.hero-card {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  padding: 28px;
  border-radius: 24px;
  background: linear-gradient(135deg, #142033 0%, #30496b 52%, #d97a30 100%);
  color: #fffaf2;
  box-shadow: 0 18px 50px rgba(20, 32, 51, 0.18);
}

.eyebrow {
  font-size: 12px;
  letter-spacing: 0.18em;
  margin-bottom: 10px;
  color: rgba(255, 245, 225, 0.72);
}

.hero-card h2 {
  margin: 0;
  font-size: 30px;
}

.hero-desc {
  max-width: 680px;
  margin-top: 12px;
  line-height: 1.7;
}

.hero-actions {
  display: grid;
  grid-template-columns: minmax(220px, 280px);
  gap: 10px;
  align-content: center;
}

.main-grid {
  display: grid;
  grid-template-columns: 1.2fr 0.8fr;
  gap: 18px;
}

.panel-card {
  border-radius: 22px;
  border: 1px solid rgba(22, 36, 58, 0.08);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.panel-header span {
  font-weight: 700;
  color: #1e2b3c;
}

.panel-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.tool-form {
  display: flex;
  flex-direction: column;
}

.inline-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

.upload-toolbar,
.submit-row,
.current-actions {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.preview-grid,
.asset-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
  margin-top: 12px;
}

.preview-card,
.asset-card,
.asset-grid img,
.asset-grid video {
  border-radius: 16px;
  overflow: hidden;
  background: #eef4fb;
  border: 1px solid rgba(33, 55, 87, 0.08);
}

.preview-card {
  position: relative;
  min-height: 150px;
}

.preview-card img,
.asset-grid img,
.asset-grid video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.asset-card {
  position: relative;
  min-height: 150px;
}

.asset-card img,
.asset-card video {
  cursor: zoom-in;
}

.asset-toolbar {
  position: absolute;
  left: 10px;
  right: 10px;
  bottom: 10px;
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  opacity: 0;
  transition: opacity 0.18s ease;
}

.asset-card:hover .asset-toolbar,
.asset-card:focus-within .asset-toolbar {
  opacity: 1;
}

.asset-toolbar :deep(.el-button) {
  backdrop-filter: blur(8px);
  background: rgba(15, 23, 36, 0.72);
  border-color: rgba(255, 255, 255, 0.12);
  color: #f8fafc;
}

.preview-remove {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 999px;
  background: rgba(14, 20, 29, 0.7);
  color: white;
  cursor: pointer;
}

.locked-box,
.empty-upload {
  border: 1px dashed rgba(39, 68, 102, 0.24);
  border-radius: 16px;
  padding: 24px;
  text-align: center;
  color: #6a7787;
  background: rgba(245, 248, 252, 0.8);
}

.locked-box .el-icon {
  font-size: 28px;
  margin-bottom: 10px;
}

.hidden-input {
  display: none;
}

.model-option {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.model-meta,
.model-hint {
  color: #6b7480;
  font-size: 12px;
}

.model-hint {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-top: 6px;
}

.result-card {
  min-height: 100%;
}

.current-task {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.task-topline {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  align-items: center;
  color: #4c5868;
}

.task-prompt,
.prompt-cell {
  color: #253243;
  line-height: 1.6;
}

.prompt-cell {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.table-card :deep(.el-card__body) {
  padding-top: 0;
}

.dialog-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.detail-layout {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.compact-block h4 {
  margin-bottom: 0;
}

.detail-block h4 {
  margin: 0 0 10px;
  color: #1f2a3a;
}

.json-box {
  margin: 0;
  padding: 14px;
  border-radius: 14px;
  background: #0f1724;
  color: #dce7f7;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 320px;
  overflow: auto;
}

.json-box.small {
  max-height: 160px;
}

.event-line {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 8px;
}

.docs-layout {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.media-preview-body {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 320px;
  background: #0f1724;
  border-radius: 18px;
  padding: 16px;
}

.media-preview-image,
.media-preview-video {
  max-width: 100%;
  max-height: min(72vh, 860px);
  border-radius: 12px;
}

.docs-page-card {
  border-radius: 22px;
}

.docs-summary-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
}

.docs-paragraph {
  margin: 0;
  color: #334155;
  line-height: 1.8;
}

.doc-steps {
  margin: 0;
  padding-left: 18px;
  color: #334155;
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
  color: #334155;
  line-height: 1.8;
}

.doc-list {
  margin: 0;
  padding-left: 18px;
  color: #334155;
}

@media (max-width: 1100px) {
  .main-grid {
    grid-template-columns: 1fr;
  }

  .hero-card {
    flex-direction: column;
  }

  .docs-summary-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .bailian-page {
    padding: 14px;
  }

  .inline-grid {
    grid-template-columns: 1fr;
  }

  .hero-card h2 {
    font-size: 24px;
  }

  .upload-toolbar,
  .submit-row,
  .current-actions,
  .model-hint {
    flex-direction: column;
    align-items: stretch;
  }

  .asset-toolbar {
    opacity: 1;
    position: static;
    padding: 10px;
    background: linear-gradient(180deg, transparent 0%, rgba(15, 23, 36, 0.72) 100%);
  }
}
</style>
