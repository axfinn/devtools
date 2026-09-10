// useAssets.js —— 虚拟形象模块后端 API 包装(P1)
//
// 设计要点:
//   - 包 /api/avatar/{models,me/assets,clips,share} 全部端点
//   - 不引入新依赖(沿用项目 fetch + JSON 约定,见 ImageViewerTool.vue)
//   - 鉴权:creator_key 从 localStorage 取;password 由调用方显式传入(上传/删除时)
//   - 错误统一抛 {status, message} 让组件用 Element Plus el-message 提示
//
// 后续 P2 会扩展:模型上传 UI、share dialog、me_asset 切换模型等;此文件保持稳定契约。

const API_BASE = '/api/avatar'

// 默认 creator_key —— 浏览器无登录态时用 localStorage 中的占位,空则不上传 header。
function defaultCreatorKey() {
  try {
    return localStorage.getItem('avatar_creator_key') || ''
  } catch (_e) {
    return ''
  }
}

function persistCreatorKey(key) {
  try {
    if (key) localStorage.setItem('avatar_creator_key', key)
    else localStorage.removeItem('avatar_creator_key')
  } catch (_e) { /* 隐私模式/无痕可能抛,忽略 */ }
}

// avatarFetch —— 统一 fetch 包装:
//   - 注入 X-Creator-Key(可选)
//   - JSON 解析,4xx/5xx 抛 {status, message}
//   - 404 不抛,直接返回 null(列表为空常见)
async function avatarFetch(path, opts = {}) {
  const headers = new Headers(opts.headers || {})
  const creatorKey = opts.creatorKey ?? defaultCreatorKey()
  if (creatorKey && !headers.has('X-Creator-Key')) {
    headers.set('X-Creator-Key', creatorKey)
  }
  let body = opts.body
  if (body && typeof body === 'object' && !(body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
    body = JSON.stringify(body)
  }
  const res = await fetch(API_BASE + path, { ...opts, headers, body })
  if (res.status === 404) return null
  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try {
      const data = await res.json()
      msg = data.error || msg
    } catch (_e) { /* 响应不是 JSON,沿用 status */ }
    const err = new Error(msg)
    err.status = res.status
    throw err
  }
  // 204 No Content
  if (res.status === 204) return null
  const ct = res.headers.get('Content-Type') || ''
  return ct.includes('application/json') ? res.json() : res.text()
}

// ---- 模型库 ----

export async function listModels({ ownerId = '', limit = 50, offset = 0, creatorKey } = {}) {
  const qs = new URLSearchParams()
  if (ownerId) qs.set('owner_id', ownerId)
  qs.set('limit', String(limit))
  qs.set('offset', String(offset))
  const data = await avatarFetch(`/models?${qs.toString()}`, { creatorKey })
  return data || { items: [], count: 0, limit, offset }
}

export async function getModel(id, { creatorKey } = {}) {
  return avatarFetch(`/models/${encodeURIComponent(id)}`, { creatorKey })
}

export async function uploadModel(file, meta = {}, { creatorKey, password } = {}) {
  const fd = new FormData()
  fd.append('file', file)
  if (meta.title) fd.append('title', meta.title)
  if (meta.format) fd.append('format', meta.format)
  if (meta.boneCount != null) fd.append('bone_count', String(meta.boneCount))
  if (meta.polyCount != null) fd.append('poly_count', String(meta.polyCount))
  if (meta.hasSkeleton) fd.append('has_skeleton', '1')
  if (creatorKey) fd.append('creator_key', creatorKey)
  if (password) fd.append('password', password)
  return avatarFetch('/models', { method: 'POST', body: fd, creatorKey })
}

export async function deleteModel(id, { creatorKey, password } = {}) {
  const qs = new URLSearchParams()
  if (creatorKey && !password) qs.set('owner_id', creatorKey)
  if (password) qs.set('password', password)
  const path = `/models/${encodeURIComponent(id)}${qs.toString() ? '?' + qs.toString() : ''}`
  return avatarFetch(path, { method: 'DELETE', creatorKey })
}

export function modelFileURL(id) {
  return `${API_BASE}/models/${encodeURIComponent(id)}/file`
}

// ---- 我的资产 ----

export async function listMyAssets({ creatorKey } = {}) {
  return avatarFetch('/me/assets', { creatorKey })
}

export async function registerMyAsset({ modelId, title, note }, { creatorKey } = {}) {
  return avatarFetch('/me/assets', {
    method: 'POST',
    creatorKey,
    body: { owner_id: creatorKey || defaultCreatorKey(), model_id: modelId, title: title || '', note: note || '' },
  })
}

export async function deleteMyAsset(id, { creatorKey } = {}) {
  return avatarFetch(`/me/assets/${encodeURIComponent(id)}`, { method: 'DELETE', creatorKey })
}

// ---- 姿态片段 ----

export async function uploadClip(payload, { creatorKey } = {}) {
  return avatarFetch('/clips', { method: 'POST', creatorKey, body: payload })
}

export async function getClip(id, { password, creatorKey } = {}) {
  const qs = new URLSearchParams()
  if (password) qs.set('password', password)
  const path = `/clips/${encodeURIComponent(id)}${qs.toString() ? '?' + qs.toString() : ''}`
  return avatarFetch(path, { creatorKey })
}

export async function deleteClip(id, { creatorKey, password } = {}) {
  const qs = new URLSearchParams()
  if (creatorKey && !password) qs.set('owner_id', creatorKey)
  if (password) qs.set('password', password)
  const path = `/clips/${encodeURIComponent(id)}${qs.toString() ? '?' + qs.toString() : ''}`
  return avatarFetch(path, { method: 'DELETE', creatorKey })
}

// ---- 分享链接 ----

export async function createShare({ targetType, targetId, password, expiresInDays }, { creatorKey } = {}) {
  return avatarFetch('/share', {
    method: 'POST',
    creatorKey,
    body: { target_type: targetType, target_id: targetId, password, expires_in_days: expiresInDays },
  })
}

export async function getShare(code, { password } = {}) {
  const qs = new URLSearchParams()
  if (password) qs.set('password', password)
  const path = `/share/${encodeURIComponent(code)}${qs.toString() ? '?' + qs.toString() : ''}`
  return avatarFetch(path)
}

export async function deleteShare(code, { creatorKey } = {}) {
  return avatarFetch(`/share/${encodeURIComponent(code)}`, { method: 'DELETE', creatorKey })
}

// ---- Vue composable(P1 起对外暴露的最常用接口) ----
//
// 用法:
//   const assets = useAssets()
//   await assets.refreshModels()
//   assets.models  // ref([])
//   assets.uploadModelFile(file, { title: 'demo' })
//
// 不依赖 reactive store(避免 Pinia 重型引入);用 module-scope ref + 单例。
import { ref } from 'vue'

const _models = ref([])
const _myAssets = ref([])
const _clips = ref([])
const _shares = ref([])
const _loading = ref(false)
const _lastError = ref('')

async function _refresh(fn, target) {
  _loading.value = true
  _lastError.value = ''
  try {
    const data = await fn()
    target.value = data?.items || data || []
    return data
  } catch (e) {
    _lastError.value = e.message || '加载失败'
    return null
  } finally {
    _loading.value = false
  }
}

export function useAssets() {
  return {
    models: _models,
    myAssets: _myAssets,
    clips: _clips,
    shares: _shares,
    loading: _loading,
    lastError: _lastError,
    creatorKey: defaultCreatorKey,
    setCreatorKey(key) {
      persistCreatorKey(key)
    },
    refreshModels(opts = {}) {
      return _refresh(() => listModels(opts), _models)
    },
    refreshMyAssets(opts = {}) {
      return _refresh(() => listMyAssets(opts), _myAssets)
    },
    refreshClips(opts = {}) {
      // 现阶段没列所有 clips 的端点;返回空数组保持契约一致。
      _clips.value = []
      return Promise.resolve({ items: [], count: 0 })
    },
    refreshShares() {
      _shares.value = []
      return Promise.resolve({ items: [], count: 0 })
    },
    async uploadModelFile(file, meta = {}, opts = {}) {
      const created = await uploadModel(file, meta, opts)
      // 上传成功后刷新列表
      await _refresh(() => listModels({}), _models)
      return created
    },
    async removeModel(id, opts = {}) {
      const r = await deleteModel(id, opts)
      _models.value = _models.value.filter((m) => m.id !== id)
      return r
    },
    async createShareLink(targetType, targetId, opts = {}) {
      return createShare({ targetType, targetId, ...opts })
    },
  }
}
