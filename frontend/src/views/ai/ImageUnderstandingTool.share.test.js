import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus, { ElMessage } from 'element-plus'
import ImageUnderstandingTool from './ImageUnderstandingTool.vue'

const SHARE_ENDPOINT = '/api/minimax/result-shares'
const TOOLS_ENDPOINT = '/api/image-understanding/tools'

// 10MB 闸门（实现里 SHARE_IMAGE_MAX_BYTES）
const OVER_LIMIT = 10 * 1024 * 1024 + 1
const UNDER_LIMIT = 2 * 1024 * 1024

// 造一个可辨识的 base64 载荷，断言它不进 payload.raw
const IMAGE_B64 = 'QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVphYmNkZWZnaGlqa2xtbm9wcXJzdHV2d3h5eg=='
const IMAGE_DATA_URL = `data:image/png;base64,${IMAGE_B64}`

function installStorage(values = {}) {
  const store = { ...values }
  vi.stubGlobal('localStorage', {
    getItem: (k) => (k in store ? store[k] : null),
    setItem: (k, v) => { store[k] = String(v) },
    removeItem: (k) => { delete store[k] },
    clear: () => { Object.keys(store).forEach((k) => delete store[k]) },
  })
}

function makeFetchMock() {
  return vi.fn(async (url) => {
    const target = String(url)
    if (target.includes(SHARE_ENDPOINT)) {
      return { ok: true, text: async () => JSON.stringify({ share_url: 'https://t.jaxiu.cn/s/qa71' }) }
    }
    if (target.includes(TOOLS_ENDPOINT)) {
      return {
        ok: true,
        text: async () => JSON.stringify({ tools: [{ name: 'MiniMax-MCP', description: '图像理解工具' }] }),
      }
    }
    return { ok: true, text: async () => '{}' }
  })
}

function shareCalls(fetchMock) {
  return fetchMock.mock.calls.filter((call) => String(call[0]).includes(SHARE_ENDPOINT))
}

function bodyOf(call) {
  return JSON.parse(call[1].body)
}

function mountTool() {
  return mount(ImageUnderstandingTool, {
    global: { plugins: [ElementPlus] },
  })
}

describe('(B) ImageUnderstandingTool 保存分享', () => {
  let errorSpy
  let warningSpy

  beforeEach(() => {
    errorSpy = vi.spyOn(ElMessage, 'error').mockImplementation(() => {})
    warningSpy = vi.spyOn(ElMessage, 'warning').mockImplementation(() => {})
    vi.stubGlobal('navigator', {
      ...globalThis.navigator,
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('有图 + 有凭证 → POST /api/minimax/result-shares，字段齐全', async () => {
    installStorage({ minimax_studio_api_key: 'dtk_ai_test' })
    const fetchMock = makeFetchMock()
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mountTool()
    wrapper.vm.selectedTool = 'MiniMax-MCP'
    wrapper.vm.prompt = '这张图里有什么'
    wrapper.vm.resultText = '图里是一只猫'
    wrapper.vm.imageBase64 = IMAGE_DATA_URL
    wrapper.vm.imageFile = { name: 'cat.png', size: UNDER_LIMIT }
    wrapper.vm.rawResult = JSON.stringify({ model: 'MiniMax-M3', content: '图里是一只猫', image: IMAGE_DATA_URL })

    await wrapper.vm.shareMcpResult()

    const calls = shareCalls(fetchMock)
    expect(calls).toHaveLength(1)
    const body = bodyOf(calls[0])
    expect(calls[0][1].method).toBe('POST')
    expect(body.title).toBeTruthy()
    expect(body.summary).toBeTruthy()
    expect(body.result_type).toBe('image_understanding')
    expect(body.model).toBe('MiniMax-MCP')
    expect(body.payload.text).toBe('图里是一只猫')
    expect(body.payload.prompt).toBe('这张图里有什么')
    expect(body.payload.tool).toBe('MiniMax-MCP')
    expect(body.assets).toHaveLength(1)
    expect(body.assets[0].data_url).toBe(IMAGE_DATA_URL)
    expect(body.assets[0].kind).toBe('image')
    expect(body.assets[0].filename).toBe('cat.png')
    // content_type 由 data URL 头推导
    expect(body.assets[0].content_type).toBe('image/png')
    expect(errorSpy).not.toHaveBeenCalled()
  })

  it('U10：payload.raw 是 JSON 字符串，且不含本次上传图片的 base64', async () => {
    installStorage({ minimax_studio_super_admin_password: 'sa_pwd' })
    const fetchMock = makeFetchMock()
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mountTool()
    wrapper.vm.selectedTool = 'MiniMax-MCP'
    wrapper.vm.prompt = '描述这张图'
    wrapper.vm.resultText = '识别结果文本'
    wrapper.vm.imageBase64 = IMAGE_DATA_URL
    wrapper.vm.imageFile = { name: 'cat.png', size: UNDER_LIMIT }
    wrapper.vm.rawResult = JSON.stringify({ model: 'MiniMax-M3', content: '识别结果文本', image: IMAGE_DATA_URL })

    await wrapper.vm.shareMcpResult()

    const body = bodyOf(shareCalls(fetchMock)[0])
    // U12：raw 是 JSON 字符串
    expect(typeof body.payload.raw).toBe('string')
    const parsed = JSON.parse(body.payload.raw)
    expect(parsed.content).toBe('识别结果文本')
    // U10：raw 不含本次上传图片的 base64
    expect(body.payload.raw).not.toContain(IMAGE_B64)
  })

  it('U8：无图场景 → assets 是空数组（不是 [{data_url:null}]）', async () => {
    installStorage({ minimax_studio_api_key: 'dtk_ai_test' })
    const fetchMock = makeFetchMock()
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mountTool()
    wrapper.vm.selectedTool = 'MiniMax-MCP'
    wrapper.vm.prompt = '纯文本提问'
    wrapper.vm.resultText = '只有文本结果'
    wrapper.vm.imageBase64 = ''
    wrapper.vm.imageFile = null
    wrapper.vm.rawResult = JSON.stringify({ content: '只有文本结果' })

    await wrapper.vm.shareMcpResult()

    const calls = shareCalls(fetchMock)
    expect(calls).toHaveLength(1)
    const body = bodyOf(calls[0])
    expect(Array.isArray(body.assets)).toBe(true)
    expect(body.assets).toHaveLength(0)
    expect(body.assets.some((a) => a && 'data_url' in a && !a.data_url)).toBe(false)
    expect(body.payload.text).toBe('只有文本结果')
  })

  it('U11：图片超 10MB → 降级为只分享文本（assets 空数组、请求仍发出、不报错）', async () => {
    installStorage({ minimax_studio_api_key: 'dtk_ai_test' })
    const fetchMock = makeFetchMock()
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mountTool()
    wrapper.vm.selectedTool = 'MiniMax-MCP'
    wrapper.vm.prompt = '大图识别'
    wrapper.vm.resultText = '大图的识别结果'
    wrapper.vm.imageBase64 = IMAGE_DATA_URL
    wrapper.vm.imageFile = { name: 'huge.png', size: OVER_LIMIT }
    wrapper.vm.rawResult = JSON.stringify({ content: '大图的识别结果' })

    await wrapper.vm.shareMcpResult()

    const calls = shareCalls(fetchMock)
    expect(calls, '超限也要把请求发出去').toHaveLength(1)
    const body = bodyOf(calls[0])
    expect(Array.isArray(body.assets)).toBe(true)
    expect(body.assets).toHaveLength(0)
    expect(body.result_type).toBe('image_understanding')
    expect(body.payload.text).toBe('大图的识别结果')
    expect(errorSpy, '超限降级不是错误').not.toHaveBeenCalled()
    // 有「图片过大」的用户可见提示（方案 T11）
    expect(warningSpy).toHaveBeenCalled()
    const warnText = warningSpy.mock.calls.map((c) => String(c[0])).join(' ')
    expect(warnText).toMatch(/图片过大|仅保存识别文本/)
  })

  it('边界：无结果文本 → 不发请求', async () => {
    installStorage({ minimax_studio_api_key: 'dtk_ai_test' })
    const fetchMock = makeFetchMock()
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mountTool()
    wrapper.vm.resultText = ''

    await wrapper.vm.shareMcpResult()

    expect(shareCalls(fetchMock)).toHaveLength(0)
  })

  it('边界：无凭证 → 不发请求，只提示用户', async () => {
    installStorage({})
    const fetchMock = makeFetchMock()
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mountTool()
    wrapper.vm.selectedTool = 'MiniMax-MCP'
    wrapper.vm.resultText = '有结果但没凭证'

    await wrapper.vm.shareMcpResult()

    expect(shareCalls(fetchMock)).toHaveLength(0)
    expect(errorSpy).toHaveBeenCalled()
  })
})
