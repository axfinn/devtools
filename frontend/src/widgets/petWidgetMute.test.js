/**
 * 云帆 GLB 宠物（pet-widget）全局静音能力 —— 行为级测试
 * ---------------------------------------------------------------------------
 * 被测实现: frontend/public/widgets/pet-widget.js
 *   （可用环境变量 PET_WIDGET_SRC=<path> 覆盖被测源码路径，用于反事实自检；
 *     不设时用默认的 public/widgets/pet-widget.js）
 *
 * 背景（FINN-175 · Stage 5 行为级验证）：
 *   Stage 3 修复（FINN-173，commit 290dbadd）给 pet-widget 加回了「全局静音」：
 *     · 持久化键 localStorage['pet-widget.enabled']（'1' 开启 / '0' 静音；缺失 = 开启）
 *     · 唯一的 MP3 出口 playVoice(file, volume, allowed)：allowed 为假时连 Audio 都不创建
 *     · 有效发声 = 全局开关 && 元素级 voice 属性
 *
 * 本文件钉死的契约（逐条对应 FINN-175 的 A1–A5）：
 *   A1 默认开启：localStorage 里没有 pet-widget.enabled 时，发声照旧（不改变首次访问体验）
 *   A2 静音 ⇒ 任何一条发声路径都不创建 Audio、也不朗读文本
 *   A3 元素级 voice 属性仍然有效（既有契约未被破坏）
 *   A4 全局 + 持久：状态写进 localStorage；多实例共享；新挂载实例读到同一个键
 *   A5 首次挂载的 voice 属性不得覆盖持久化状态
 *
 * 实现说明：
 *   pet-widget.js 是 public/ 下的 IIFE 脚本（不是 ES module，vite 也不允许从
 *   public/ 直接 import）⇒ 用 node:fs 读源码 + 间接 eval 到 jsdom 全局，注册出
 *   <pet-widget> custom element 后直接驱动其方法。Audio / speechSynthesis /
 *   SpeechSynthesisUtterance 全部显式打桩。
 *
 * ⚠️ localStorage 桩是**必需**的，不是装饰：
 *   本机 Node 26 的实验性 globalThis.localStorage 在没有 --localstorage-file 时
 *   不可用（jsdom 的 localStorage 在 about:blank 不透明源下也不可用），于是
 *   pet-widget 的 readGlobalVoice() 会走 catch 恒返回 true ⇒ 所有静音用例都会
 *   「恒绿」（测不出任何东西）。故这里自造 localStorage 桩，并用第一条用例断言
 *   桩确实生效（桩没生效 ⇒ 后面所有静音断言都是空断言）。
 */
import { describe, it, expect, beforeAll, beforeEach, afterEach, afterAll, vi } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = path.dirname(fileURLToPath(import.meta.url))
const FRONTEND_ROOT = path.resolve(HERE, '../..')
// 被测源码路径：可用 PET_WIDGET_SRC 覆盖（反事实自检用同一条命令跑基线源码）
const WIDGET_PATH = process.env.PET_WIDGET_SRC
  ? path.resolve(process.env.PET_WIDGET_SRC)
  : path.join(FRONTEND_ROOT, 'public/widgets/pet-widget.js')

const VOICE_BASE = '/widgets/voices/'
const ENABLED_KEY = 'pet-widget.enabled'

// ---------------------------------------------------------------------------
// localStorage 桩（必须在被测脚本 eval 之前就位 —— 脚本在加载时就读一次该键）
// ---------------------------------------------------------------------------
function createFakeLocalStorage() {
  const store = new Map()
  return {
    get length() { return store.size },
    key(i) { return Array.from(store.keys())[i] ?? null },
    getItem(k) { return store.has(String(k)) ? store.get(String(k)) : null },
    setItem(k, v) { store.set(String(k), String(v)) },
    removeItem(k) { store.delete(String(k)) },
    clear() { store.clear() }
  }
}

const fakeLS = createFakeLocalStorage()

function installFakeLocalStorage(ls) {
  const errors = []
  try {
    Object.defineProperty(globalThis, 'localStorage', { value: ls, configurable: true, writable: true })
  } catch (e) {
    errors.push(e)
    try { globalThis.localStorage = ls } catch (e2) { errors.push(e2) }
  }
  if (globalThis.window && globalThis.window !== globalThis) {
    try {
      Object.defineProperty(globalThis.window, 'localStorage', { value: ls, configurable: true, writable: true })
    } catch (e) {
      errors.push(e)
    }
  }
  return errors
}

const lsInstallErrors = installFakeLocalStorage(fakeLS)

// ---------------------------------------------------------------------------
// Audio / speechSynthesis 打桩
// ---------------------------------------------------------------------------
let audioSrcs = []   // playVoice 实际请求的音频 URL
let spokenTexts = [] // speechSynthesis 实际念出的文本

class FakeAudio {
  constructor(src) {
    this.src = src
    this.volume = 1
    audioSrcs.push(src)
  }
  play() { return Promise.resolve() }
}

class FakeUtterance {
  constructor(text) { this.text = text }
}

const originalAudio = globalThis.Audio
const originalUtterance = globalThis.SpeechSynthesisUtterance
const originalSynth = globalThis.window?.speechSynthesis
const originalConsoleError = console.error

// ---------------------------------------------------------------------------
// 工具
// ---------------------------------------------------------------------------
const mounted = []

/**
 * 造一个 <pet-widget> 实例（detached，不 append）。
 * voice=true ⇒ 带 voice 属性；voice=false ⇒ 完全不带（构造函数里 _voiceOn=false）。
 */
function makeWidget({ voice = true } = {}) {
  const el = document.createElement('pet-widget')
  if (voice) el.setAttribute('voice', '')
  return el
}

/**
 * 挂载实例。
 * 真实 _init() 会去 import three.js + 拉 GLB 模型，jsdom 里没有 WebGL 上下文必然
 * 失败（还会拖出 jsdom 的 "Not implemented: getContext" 噪音）。语音开关相关的
 * 初始化只有一行 `this._voiceOn = this.hasAttribute('voice')`，这里原样保留它，
 * 其余（渲染）不参与本文件的被测行为。
 */
function mountWidget(el) {
  el._init = function () {
    this._voiceOn = this.hasAttribute('voice')
  }
  document.body.appendChild(el)
  mounted.push(el)
  return el
}

/** 让点击 / idle 随机动作有可播的动作名 */
function withFriendlyActions(el) {
  el._friendly = ['Wave']
  return el
}

function muteGlobal() { fakeLS.setItem(ENABLED_KEY, '0') }

function resetSpies() {
  audioSrcs = []
  spokenTexts = []
}

beforeAll(() => {
  globalThis.Audio = FakeAudio
  globalThis.SpeechSynthesisUtterance = FakeUtterance
  globalThis.window.speechSynthesis = {
    speak(u) { spokenTexts.push(u && u.text) },
    cancel() {}
  }
  // 真实 _init 的失败路径会 console.error（three/模型加载异常），噪音且无关
  console.error = () => {}

  // 间接 eval：在全局作用域执行，脚本里的 HTMLElement / customElements / window 才能拿到
  const src = fs.readFileSync(WIDGET_PATH, 'utf8')
  ;(0, eval)(src)
})

afterAll(() => {
  globalThis.Audio = originalAudio
  globalThis.SpeechSynthesisUtterance = originalUtterance
  console.error = originalConsoleError
  if (originalSynth === undefined) delete globalThis.window.speechSynthesis
  else globalThis.window.speechSynthesis = originalSynth
})

beforeEach(() => {
  fakeLS.clear()
  resetSpies()
})

afterEach(() => {
  while (mounted.length) {
    const el = mounted.pop()
    if (el.parentNode) el.parentNode.removeChild(el)
  }
  vi.useRealTimers()
})

describe('pet-widget 全局静音（localStorage[\'pet-widget.enabled\']）', () => {
  it('桩自检：自造的 localStorage 确实生效（桩没生效 ⇒ 下面所有静音断言都是空断言）', () => {
    expect(lsInstallErrors).toEqual([])
    expect(globalThis.localStorage).toBe(fakeLS)
    fakeLS.setItem(ENABLED_KEY, '0')
    expect(localStorage.getItem(ENABLED_KEY)).toBe('0')
    fakeLS.removeItem(ENABLED_KEY)
    expect(localStorage.getItem(ENABLED_KEY)).toBe(null)
  })

  it('A1 默认开启：localStorage 没有 pet-widget.enabled 时 speak(\'theme_prism\') 照旧播 theme_prism.mp3', () => {
    expect(localStorage.getItem(ENABLED_KEY)).toBe(null)
    const el = makeWidget()
    el.speak('theme_prism')
    expect(audioSrcs).toEqual([`${VOICE_BASE}theme_prism.mp3`])
    expect(spokenTexts).toEqual([])
  })

  it('A2① 静音 ⇒ 主题语音 speak(\'theme_prism\') 不创建任何 Audio、不朗读', () => {
    muteGlobal()
    const el = makeWidget()
    el.speak('theme_prism')
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual([])
  })

  it('A2② 静音 ⇒ 挂载后 600ms 定时发起的 greeting 语音（其出口 speak(\'greeting\')）不创建任何 Audio', () => {
    muteGlobal()
    const el = mountWidget(makeWidget())
    el.speak('greeting')
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual([])
  })

  it('A2③ 静音 ⇒ speak(\'bye\') 不创建任何 Audio', () => {
    muteGlobal()
    const el = makeWidget()
    el.speak('bye')
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual([])
  })

  it('A2④ 静音 ⇒ speak(\'cheer\') 不创建任何 Audio，也不朗读「你已经很棒啦！」', () => {
    muteGlobal()
    const el = makeWidget()
    el.speak('cheer')
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual([])
  })

  it('A2⑤ 静音 ⇒ 点击动作（_onTap）不创建任何 Audio', () => {
    muteGlobal()
    const el = withFriendlyActions(makeWidget())
    el._onTap()
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual([])
  })

  it('A2⑥ 静音 ⇒ idle 随机动作语音（4–9 秒定时）不创建任何 Audio', () => {
    muteGlobal()
    vi.useFakeTimers()
    const el = withFriendlyActions(makeWidget())
    el._scheduleRandom()
    vi.advanceTimersByTime(10000)
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual([])
  })

  it('A2⑦ 静音 ⇒ setForm(\'cat\') 不创建任何 Audio', () => {
    muteGlobal()
    const el = makeWidget()
    el.setForm('cat')
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual([])
  })

  it('A2⑧ 静音 ⇒ setTheme(\'flame\') 不创建任何 Audio', () => {
    muteGlobal()
    const el = makeWidget()
    el.setTheme('flame')
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual([])
  })

  it('A2⑨ 静音 ⇒ 文本回退 speak(\'你好\') 不朗读', () => {
    muteGlobal()
    const el = makeWidget()
    el.speak('你好')
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual([])
  })

  it('A3 元素级 voice 属性仍然有效：元素不带 voice 属性时 speak(\'你好\') 不发声', () => {
    const el = makeWidget({ voice: false })
    el.speak('你好')
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual([])
  })

  it('A3 元素级 voice 属性仍然有效：元素带 voice 属性且全局开启时 speak(\'你好\') 正常朗读', () => {
    const el = makeWidget()
    el.speak('你好')
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual(['你好'])
  })

  it('A4① 开关状态持久化在 localStorage[\'pet-widget.enabled\']（连通后改 voice 属性 ⇒ 写 \'1\'/\'0\'）', () => {
    const el = mountWidget(makeWidget({ voice: false }))
    expect(localStorage.getItem(ENABLED_KEY)).toBe(null) // 首次挂载不写任何值
    el.setAttribute('voice', '')
    expect(localStorage.getItem(ENABLED_KEY)).toBe('1')
    el.removeAttribute('voice')
    expect(localStorage.getItem(ENABLED_KEY)).toBe('0')
  })

  it('A4② 两个独立实例共享同一全局开关状态', () => {
    const a = mountWidget(makeWidget())
    const b = mountWidget(makeWidget())
    a.removeAttribute('voice') // 连通后关掉 ⇒ 全局写 '0'
    expect(localStorage.getItem(ENABLED_KEY)).toBe('0')
    b.speak('greeting') // b 自己的 voice 属性还在，但全局已关 ⇒ 不发声
    expect(audioSrcs).toEqual([])
    a.setAttribute('voice', '')
    expect(localStorage.getItem(ENABLED_KEY)).toBe('1')
    b.speak('greeting')
    expect(audioSrcs).toEqual([`${VOICE_BASE}greeting_hi.mp3`])
  })

  it('A4③ 新挂载的实例读到同一个持久化键（模拟刷新/换页后仍然静音）', () => {
    muteGlobal() // 模拟上一次会话留下的静音状态
    const el = mountWidget(makeWidget())
    el.speak('greeting')
    expect(audioSrcs).toEqual([])
    expect(localStorage.getItem(ENABLED_KEY)).toBe('0')
  })

  it('A5 首次挂载的 voice 属性不得覆盖持久化的静音状态', () => {
    muteGlobal()
    const el = mountWidget(makeWidget()) // 带 voice 属性的实例
    el.speak('greeting')
    expect(audioSrcs).toEqual([])
    expect(localStorage.getItem(ENABLED_KEY)).toBe('0') // 持久化状态没被冲掉
  })

  it('正向对照：全局开启时点击动作与 idle 随机动作确实会发声（证明触发路径有效，防恒绿）', () => {
    vi.useFakeTimers()
    const tapped = withFriendlyActions(makeWidget())
    tapped._onTap()
    expect(audioSrcs).toEqual([`${VOICE_BASE}act_wave.mp3`])

    resetSpies()
    const idle = withFriendlyActions(makeWidget())
    idle._scheduleRandom()
    vi.advanceTimersByTime(10000)
    expect(audioSrcs.length).toBeGreaterThan(0)
    expect(audioSrcs[0]).toBe(`${VOICE_BASE}act_wave.mp3`)
  })
})
