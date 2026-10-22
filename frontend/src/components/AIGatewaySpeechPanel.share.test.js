import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ElementPlus, { ElMessage } from 'element-plus'
import AIGatewaySpeechPanel from './AIGatewaySpeechPanel.vue'

// guessAudioMime(format) 的期望值（被测实现里的私有函数，测试里显式复述规格）
const MIME_BY_FORMAT = {
  mp3: 'audio/mpeg',
  wav: 'audio/wav',
  flac: 'audio/flac',
  pcm: 'audio/pcm',
}

// 20MB 闸门（实现里 SHARE_AUDIO_MAX_BYTES）
const OVER_LIMIT = 20 * 1024 * 1024 + 1
const UNDER_LIMIT = 3

// 面板挂载时（有凭证）会拉一次语音资源（音色/文件/任务），mock 要能吃下这些请求
function jsonResponse(data = {}) {
  return {
    ok: true,
    status: 200,
    json: async () => data,
    text: async () => JSON.stringify(data),
    blob: async () => new Blob([]),
  }
}

// blob: URL 的 fetch mock：返回一个可控体积的音频 blob
function makeFetchMock(size = UNDER_LIMIT) {
  return vi.fn(async (url) => {
    if (String(url).startsWith('blob:')) {
      return {
        ok: true,
        status: 200,
        blob: async () => {
          if (size > UNDER_LIMIT) {
            // 超限场景只需要 size，够实现判闸门即可
            return { size }
          }
          return new Blob([new Uint8Array([1, 2, 3])], { type: 'audio/mpeg' })
        },
        json: async () => ({}),
        text: async () => '{}',
      }
    }
    return jsonResponse({ voices: [], files: [], tasks: [] })
  })
}

function urlsOf(fetchMock, needle) {
  return fetchMock.mock.calls
    .map((call) => String(call[0]))
    .filter((u) => u.includes(needle))
}

async function mountPanel(props = { superAdminPassword: 'super-pwd' }) {
  const wrapper = mount(AIGatewaySpeechPanel, {
    props,
    global: { plugins: [ElementPlus] },
  })
  await flushPromises() // 让挂载期的资源加载请求先落定，避免干扰 fetch 计数断言
  return wrapper
}

function lastDraft(wrapper) {
  const emitted = wrapper.emitted('share')
  expect(emitted, '应当 emit 一次 share 事件').toBeTruthy()
  return emitted[emitted.length - 1][0]
}

describe('(A) AIGatewaySpeechPanel 保存分享', () => {
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

  describe('shareEnabled 门控', () => {
    it('shareEnabled=false → 不渲染「保存分享」按钮', async () => {
      const wrapper = await mountPanel({ superAdminPassword: 'pwd', shareEnabled: false })
      wrapper.vm.syncAudioUrl = 'blob:http://localhost/abc'
      await wrapper.vm.$nextTick()
      const labels = wrapper.findAll('button').map((b) => b.text())
      expect(labels).not.toContain('保存分享')
    })

    it('shareEnabled=true → 渲染「保存分享」按钮', async () => {
      const wrapper = await mountPanel({ superAdminPassword: 'pwd', shareEnabled: true })
      wrapper.vm.syncAudioUrl = 'blob:http://localhost/abc'
      await wrapper.vm.$nextTick()
      const labels = wrapper.findAll('button').map((b) => b.text())
      expect(labels).toContain('保存分享')
    })
  })

  describe('同步路径：产出 resultType=speech 的 draft', () => {
    it('内联 hex 路径 → 资产是 data URL（不是 blob:），raw 剔除内联音频', async () => {
      const fetchMock = makeFetchMock()
      vi.stubGlobal('fetch', fetchMock)
      const wrapper = await mountPanel()
      wrapper.vm.syncForm.format = 'mp3'
      wrapper.vm.syncResponse = { data: { audio: 'aabbccdd', file_id: 'file-1' } }
      wrapper.vm.syncAudioUrl = 'blob:http://localhost/abc'

      await wrapper.vm.shareSyncResult()

      const draft = lastDraft(wrapper)
      expect(draft.resultType).toBe('speech')
      expect(draft.sourceLabel).toBe('语音合成')
      expect(draft.model).toBe(wrapper.vm.syncForm.model)
      expect(draft.payload.mode).toBe('sync')
      expect(draft.payload.text).toBe(wrapper.vm.syncForm.text)
      // R6：资产必须是 data URL，不能是 blob:
      expect(draft.assets).toHaveLength(1)
      expect(draft.assets[0].data_url).toMatch(/^data:audio\/mpeg;base64,/)
      expect(draft.assets[0].data_url).not.toMatch(/^blob:/)
      expect(draft.assets[0].kind).toBe('audio')
      // U1：内联音频本体不得进入 payload.raw
      expect(JSON.stringify(draft.payload.raw)).not.toContain('aabbccdd')
      expect(draft.payload.raw.audio).toBeUndefined()
      expect(draft.payload.raw.data.audio).toBeUndefined()
    })

    it('file_id 预览路径 → raw 保留 file_id，资产仍是 data URL', async () => {
      vi.stubGlobal('fetch', makeFetchMock())
      const wrapper = await mountPanel()
      wrapper.vm.syncForm.format = 'mp3'
      wrapper.vm.syncResponse = { data: { file: { file_id: 'file-abc' } } }
      wrapper.vm.syncAudioUrl = 'blob:http://localhost/abc'

      await wrapper.vm.shareSyncResult()

      const draft = lastDraft(wrapper)
      expect(draft.resultType).toBe('speech')
      expect(JSON.stringify(draft.payload.raw)).toContain('file-abc')
      expect(draft.assets[0].data_url).toMatch(/^data:audio\/mpeg;base64,/)
    })

    it('同步两条路径的 content_type 都是 guessAudioMime(format)，且与文件名扩展名一致', async () => {
      const cases = [
        { format: 'mp3', mime: 'audio/mpeg', ext: 'mp3' },
        { format: 'wav', mime: 'audio/wav', ext: 'wav' },
        { format: 'flac', mime: 'audio/flac', ext: 'flac' },
      ]
      for (const c of cases) {
        for (const path of ['inline-hex', 'file-id-preview']) {
          vi.stubGlobal('fetch', makeFetchMock())
          const wrapper = await mountPanel()
          wrapper.vm.syncForm.format = c.format
          wrapper.vm.syncResponse = path === 'inline-hex'
            ? { data: { audio: 'aabbccdd' } }
            : { data: { file: { file_id: 'file-abc' } } }
          wrapper.vm.syncAudioUrl = 'blob:http://localhost/abc'

          await wrapper.vm.shareSyncResult()

          const draft = lastDraft(wrapper)
          const asset = draft.assets[0]
          expect(asset.content_type, `${path}/${c.format} 的 mime`).toBe(c.mime)
          expect(asset.content_type).toBe(MIME_BY_FORMAT[c.format])
          expect(asset.filename.endsWith(`.${c.ext}`), `${path}/${c.format} 文件名扩展名`).toBe(true)
          expect(asset.filename).toBe(`tts-${wrapper.vm.syncForm.model}.${c.format}`)
          wrapper.unmount()
        }
      }
    })
  })

  describe('T4：https 直链走资产 url 分支', () => {
    it('https 直链 → assets[0].url 为该直链，不下载、不做体积闸门', async () => {
      const fetchMock = makeFetchMock(OVER_LIMIT)
      vi.stubGlobal('fetch', fetchMock)
      const wrapper = await mountPanel()
      wrapper.vm.syncForm.format = 'wav'
      const direct = 'https://cdn.example.com/voice-1.wav'
      wrapper.vm.syncResponse = { data: { audio_url: direct } }
      wrapper.vm.syncAudioUrl = direct

      await wrapper.vm.shareSyncResult()

      const draft = lastDraft(wrapper)
      expect(draft.assets).toHaveLength(1)
      expect(draft.assets[0].url).toBe(direct)
      expect(draft.assets[0].data_url ?? null).toBeNull()
      expect(draft.assets[0].kind).toBe('audio')
      // 直链分支不受 20MB 闸门约束：不应触发任何下载
      expect(urlsOf(fetchMock, 'cdn.example.com'), 'https 直链不应被下载').toEqual([])
      // 请求体不含音频 base64
      expect(JSON.stringify(draft)).not.toContain('base64')
      expect(errorSpy).not.toHaveBeenCalled()
    })

    it('https 直链的 content_type 跟随 format，而不是写死 mp3', async () => {
      vi.stubGlobal('fetch', makeFetchMock())
      const wrapper = await mountPanel()
      wrapper.vm.syncForm.format = 'flac'
      const direct = 'https://cdn.example.com/voice-1.flac'
      wrapper.vm.syncResponse = { data: { audio_url: direct } }
      wrapper.vm.syncAudioUrl = direct

      await wrapper.vm.shareSyncResult()

      const draft = lastDraft(wrapper)
      expect(draft.assets[0].content_type).toBe('audio/flac')
      expect(draft.assets[0].content_type).not.toBe('audio/mpeg')
      expect(draft.assets[0].filename.endsWith('.flac')).toBe(true)
    })
  })

  describe('T4b：http:// 直链 → 丢弃资产 + 提示 + 请求仍发出', () => {
    it('http 直链 → draft 仍产出，assets 是空数组，有提示，url 未进入可保存资产位', async () => {
      const fetchMock = makeFetchMock()
      vi.stubGlobal('fetch', fetchMock)
      const wrapper = await mountPanel()
      wrapper.vm.syncForm.format = 'mp3'
      const plain = 'http://cdn.example.com/voice-1.mp3'
      wrapper.vm.syncResponse = { data: { audio_url: plain } }
      wrapper.vm.syncAudioUrl = plain

      await wrapper.vm.shareSyncResult()

      // 1) 请求仍被发出（draft 产出）
      const draft = lastDraft(wrapper)
      expect(draft).toBeTruthy()
      expect(draft.resultType).toBe('speech')
      // 2) assets 是长度 0 的数组，不是 undefined / [{data_url:null}]
      expect(Array.isArray(draft.assets)).toBe(true)
      expect(draft.assets).toHaveLength(0)
      // 3) 触发用户可见提示
      expect(warningSpy).toHaveBeenCalled()
      const warnText = warningSpy.mock.calls.map((c) => String(c[0])).join(' ')
      expect(warnText).toMatch(/https 直链|未包含音频资产/)
      // 4) 该 http 直链不得进入任何「可保存资产 / 展示字段」位
      expect(JSON.stringify(draft.assets)).not.toContain(plain)
      expect(draft.title).not.toContain(plain)
      expect(draft.summary).not.toContain(plain)
      expect(draft.payload.text).not.toContain(plain)
      // 前端不应去下载这个 http 直链
      expect(urlsOf(fetchMock, 'cdn.example.com'), 'http 直链不应被下载').toEqual([])
      expect(errorSpy).not.toHaveBeenCalled()
    })

    // 口径说明：PM 的 T4b 清单第 4 条写的是「该 http 直链字符串不出现在请求体任何字段（含 payload）」，
    // 但方案 v3 §1.4（U1）写死「raw 只删内联音频本体，其余字段原样保留」，
    // 即 payload.raw 保留上游 data.audio_url 回显。当前实现按方案执行 → 本条严格口径暂不满足。
    // 标记为 expected-fail：若将来实现侧剔除 audio_url，这里会自动红（提醒去掉标记）。
    it.fails('T4b 严格口径（待 PM 拍板）：http 直链不出现在请求体任何字段（含 payload.raw）', async () => {
      const fetchMock = makeFetchMock()
      vi.stubGlobal('fetch', fetchMock)
      const wrapper = await mountPanel()
      wrapper.vm.syncForm.format = 'mp3'
      const plain = 'http://cdn.example.com/voice-1.mp3'
      wrapper.vm.syncResponse = { data: { audio_url: plain } }
      wrapper.vm.syncAudioUrl = plain

      await wrapper.vm.shareSyncResult()

      const draft = lastDraft(wrapper)
      expect(JSON.stringify(draft)).not.toContain(plain)
    })
  })

  describe('U9 / U11(A)：20MB 闸门', () => {
    it('blob 分支超 20MB → 抛错提示，不静默降级（不产出 draft）', async () => {
      vi.stubGlobal('fetch', makeFetchMock(OVER_LIMIT))
      const wrapper = await mountPanel()
      wrapper.vm.syncForm.format = 'mp3'
      wrapper.vm.syncResponse = { data: { audio: 'aabbccdd' } }
      wrapper.vm.syncAudioUrl = 'blob:http://localhost/big'

      await wrapper.vm.shareSyncResult()

      expect(errorSpy, '超限必须报错，不能静默降级').toHaveBeenCalled()
      const errText = errorSpy.mock.calls.map((c) => String(c[0])).join(' | ')
      expect(errText).toMatch(/过大/)
      expect(wrapper.emitted('share')).toBeFalsy()
    })

    it('blob 分支未超限 → 正常产出 data URL 资产', async () => {
      vi.stubGlobal('fetch', makeFetchMock(UNDER_LIMIT))
      const wrapper = await mountPanel()
      wrapper.vm.syncForm.format = 'mp3'
      wrapper.vm.syncResponse = { data: { audio: 'aabbccdd' } }
      wrapper.vm.syncAudioUrl = 'blob:http://localhost/small'

      await wrapper.vm.shareSyncResult()

      const draft = lastDraft(wrapper)
      expect(draft.assets).toHaveLength(1)
      expect(draft.assets[0].data_url).toMatch(/^data:audio\/mpeg;base64,/)
      expect(errorSpy).not.toHaveBeenCalled()
    })
  })

  describe('异步路径：产出 resultType=speech 的 draft', () => {
    it('异步 → 资产 mime 固定 audio/mpeg，文件名 .mp3，raw 带 task_id', async () => {
      vi.stubGlobal('fetch', makeFetchMock())
      const wrapper = await mountPanel()
      wrapper.vm.asyncForm.mode = 'text'
      wrapper.vm.asyncForm.text = '异步播报文本'
      wrapper.vm.asyncAudioUrl = 'blob:http://localhost/async'
      wrapper.vm.currentTask = { data: { task_id: 'task-999', status: 'Success' } }

      await wrapper.vm.shareAsyncResult()

      const draft = lastDraft(wrapper)
      expect(draft.resultType).toBe('speech')
      expect(draft.payload.task_id).toBe('task-999')
      expect(draft.payload.text).toBe('异步播报文本')
      expect(draft.assets).toHaveLength(1)
      expect(draft.assets[0].content_type).toBe('audio/mpeg')
      expect(draft.assets[0].filename.endsWith('.mp3')).toBe(true)
      expect(draft.assets[0].data_url).toMatch(/^data:audio\/mpeg;base64,/)
      expect(draft.assets[0].data_url).not.toMatch(/^blob:/)
    })

    it('异步超 20MB → 抛错提示，不产出 draft', async () => {
      vi.stubGlobal('fetch', makeFetchMock(OVER_LIMIT))
      const wrapper = await mountPanel()
      wrapper.vm.asyncAudioUrl = 'blob:http://localhost/async-big'
      wrapper.vm.currentTask = { data: { task_id: 'task-999' } }

      await wrapper.vm.shareAsyncResult()

      expect(errorSpy).toHaveBeenCalled()
      const errText = errorSpy.mock.calls.map((c) => String(c[0])).join(' | ')
      expect(errText).toMatch(/过大/)
      expect(wrapper.emitted('share')).toBeFalsy()
    })
  })

  describe('边界：无结果 / 无凭证 → 只提示，不产出 draft', () => {
    it('同步：无音频结果 → 提示且不 emit', async () => {
      vi.stubGlobal('fetch', makeFetchMock())
      const wrapper = await mountPanel()
      wrapper.vm.syncAudioUrl = ''

      await wrapper.vm.shareSyncResult()

      expect(errorSpy).toHaveBeenCalled()
      expect(wrapper.emitted('share')).toBeFalsy()
    })

    it('异步：无音频结果 → 提示且不 emit', async () => {
      vi.stubGlobal('fetch', makeFetchMock())
      const wrapper = await mountPanel()
      wrapper.vm.asyncAudioUrl = ''

      await wrapper.vm.shareAsyncResult()

      expect(errorSpy).toHaveBeenCalled()
      expect(wrapper.emitted('share')).toBeFalsy()
    })

    it('同步：无凭证 → 提示且不 emit', async () => {
      const fetchMock = makeFetchMock()
      vi.stubGlobal('fetch', fetchMock)
      const wrapper = await mountPanel({ superAdminPassword: '' })
      wrapper.vm.syncAudioUrl = 'blob:http://localhost/abc'
      wrapper.vm.syncResponse = { data: { audio: 'aabbccdd' } }

      await wrapper.vm.shareSyncResult()

      expect(errorSpy).toHaveBeenCalled()
      expect(wrapper.emitted('share')).toBeFalsy()
      expect(urlsOf(fetchMock, 'blob:')).toEqual([])
    })

    it('异步：无凭证 → 提示且不 emit', async () => {
      const fetchMock = makeFetchMock()
      vi.stubGlobal('fetch', fetchMock)
      const wrapper = await mountPanel({ superAdminPassword: '' })
      wrapper.vm.asyncAudioUrl = 'blob:http://localhost/abc'

      await wrapper.vm.shareAsyncResult()

      expect(errorSpy).toHaveBeenCalled()
      expect(wrapper.emitted('share')).toBeFalsy()
      expect(urlsOf(fetchMock, 'blob:')).toEqual([])
    })
  })
})
