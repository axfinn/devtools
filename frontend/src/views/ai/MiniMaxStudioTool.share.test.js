import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ElementPlus, { ElMessage } from 'element-plus'
import { createRouter, createMemoryHistory } from 'vue-router'
import MiniMaxStudioTool from './MiniMaxStudioTool.vue'
import AIGatewaySpeechPanel from '../../components/AIGatewaySpeechPanel.vue'

const SHARE_ENDPOINT = '/api/minimax/result-shares'

function installStorage(values = {}) {
  const store = { ...values }
  vi.stubGlobal('localStorage', {
    getItem: (k) => (k in store ? store[k] : null),
    setItem: (k, v) => { store[k] = String(v) },
    removeItem: (k) => { delete store[k] },
    clear: () => { Object.keys(store).forEach((k) => delete store[k]) },
  })
}

function jsonResponse(data = {}) {
  return {
    ok: true,
    status: 200,
    json: async () => data,
    text: async () => JSON.stringify(data),
    blob: async () => new Blob([]),
  }
}

function makeFetchMock() {
  return vi.fn(async (url) => {
    const target = String(url)
    if (target.includes(SHARE_ENDPOINT)) {
      return jsonResponse({ share_url: 'https://t.jaxiu.cn/s/qa71' })
    }
    return jsonResponse({})
  })
}

// 只认「创建分享」的 POST（父组件挂载时也会 GET 分享列表）
function sharePostCalls(fetchMock) {
  return fetchMock.mock.calls.filter((call) => (
    String(call[0]).includes(SHARE_ENDPOINT) && call[1] && call[1].method === 'POST'
  ))
}

async function mountParent() {
  const router = createRouter({ history: createMemoryHistory(), routes: [] })
  await router.push('/')
  await router.isReady()
  const wrapper = mount(MiniMaxStudioTool, {
    global: { plugins: [ElementPlus, router] },
  })
  await flushPromises()
  return wrapper
}

// 造一个语音面板实例并跑出 draft（(A) 的真实产物）
async function speechDraft({ audioUrl, format = 'mp3' }) {
  const panel = mount(AIGatewaySpeechPanel, {
    props: { superAdminPassword: 'super-pwd', shareEnabled: true },
    global: { plugins: [ElementPlus] },
  })
  await flushPromises()
  panel.vm.syncForm.format = format
  panel.vm.syncResponse = { data: { audio_url: audioUrl } }
  panel.vm.syncAudioUrl = audioUrl
  await panel.vm.shareSyncResult()
  const emitted = panel.emitted('share')
  panel.unmount()
  expect(emitted, '语音面板应当产出 share draft').toBeTruthy()
  return emitted[emitted.length - 1][0]
}

describe('(A)+(B) 保存分享：draft → 父组件 POST', () => {
  let errorSpy
  let warningSpy

  beforeEach(() => {
    errorSpy = vi.spyOn(ElMessage, 'error').mockImplementation(() => {})
    warningSpy = vi.spyOn(ElMessage, 'warning').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('T4b 端到端：http:// 直链 → 请求仍发出一次，请求体 assets 长度为 0，url 未进资产位', async () => {
    installStorage({ minimax_studio_super_admin_password: 'sa_pwd' })
    const fetchMock = makeFetchMock()
    vi.stubGlobal('fetch', fetchMock)

    const plain = 'http://cdn.example.com/voice-1.mp3'
    const draft = await speechDraft({ audioUrl: plain })

    const parent = await mountParent()
    parent.vm.handleShareFromChild(draft)
    await parent.vm.submitResultShare()

    const calls = sharePostCalls(fetchMock)
    // 1) 请求仍被发出，且只发一次
    expect(calls).toHaveLength(1)
    const body = JSON.parse(calls[0][1].body)
    // 2) 请求体 assets 是长度 0 的数组
    expect(Array.isArray(body.assets)).toBe(true)
    expect(body.assets).toHaveLength(0)
    // 3) 有用户可见提示
    expect(warningSpy).toHaveBeenCalled()
    const warnText = warningSpy.mock.calls.map((c) => String(c[0])).join(' ')
    expect(warnText).toMatch(/https 直链|未包含音频资产/)
    // 4) 该 http 直链未进入请求体的资产位 / 标题 / 摘要
    expect(JSON.stringify(body.assets)).not.toContain(plain)
    expect(body.title).not.toContain(plain)
    expect(body.summary).not.toContain(plain)
    expect(body.payload).toBeTruthy()
    expect(body.result_type).toBe('speech')
    expect(errorSpy).not.toHaveBeenCalled()
  })

  it('https 直链 → 请求体 assets[0].url 为该直链（url 分支落到请求体）', async () => {
    installStorage({ minimax_studio_super_admin_password: 'sa_pwd' })
    const fetchMock = makeFetchMock()
    vi.stubGlobal('fetch', fetchMock)

    const direct = 'https://cdn.example.com/voice-1.wav'
    const draft = await speechDraft({ audioUrl: direct, format: 'wav' })

    const parent = await mountParent()
    parent.vm.handleShareFromChild(draft)
    await parent.vm.submitResultShare()

    const calls = sharePostCalls(fetchMock)
    expect(calls).toHaveLength(1)
    const body = JSON.parse(calls[0][1].body)
    expect(body.assets).toHaveLength(1)
    expect(body.assets[0].url).toBe(direct)
    expect(body.assets[0].data_url ?? null).toBeNull()
    expect(body.assets[0].content_type).toBe('audio/wav')
    expect(body.result_type).toBe('speech')
    expect(body.payload.mode).toBe('sync')
    // 请求体不含音频 base64
    expect(calls[0][1].body).not.toContain('base64')
  })
})
