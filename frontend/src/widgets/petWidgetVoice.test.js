/**
 * 云帆 GLB 宠物（pet-widget）主题语音语义回归测试
 * ---------------------------------------------------------------
 * 被测实现: frontend/public/widgets/pet-widget.js
 *
 * 背景（FINN-168 回归修复）：
 *   新版 pet-widget.js 把「主题切换欢迎语」从旧的 React 打包产物迁到
 *   speak(kind) 上，前端调用方（MiniMaxStudioTool.vue / PetTool.vue）传的是
 *   'theme_<id>' 这样的 key。修复前 speak() 没有主题分支，key 会一路落到
 *   末尾的文本回退分支，被 speechSynthesis 原样念出来（"theme_prism"），
 *   而不是播放预录的 theme_<id>.mp3。
 *
 * 契约（本文件逐条钉死）：
 *   1. speak('theme_<id>')  → 播 theme_<id>.mp3，**绝不**把 key 念出来
 *   2. setTheme(<id>)        → 换色后播一次 theme_<id>.mp3（对齐 main 的旧行为）
 *   3. 其它 kind（cheer/greeting/bye）行为不变
 *   4. 普通文本仍走文本回退（分支原契约，不得被删）
 *   5. 未知主题不发声，也绝不回退成文本朗读
 *
 * 实现说明：
 *   pet-widget.js 是 public/ 下的 IIFE 脚本（不是 ES module，且 vite 不允许
 *   从 public/ 直接 import），因此这里用 node:fs 读源码 + 间接 eval 在 jsdom
 *   全局里执行，注册出 <pet-widget> custom element 后直接驱动其方法。
 *   Audio / speechSynthesis / SpeechSynthesisUtterance 全部显式打桩，
 *   断言「播了哪个文件」和「念了什么文本」两件事。
 */
import { describe, it, expect, beforeAll, beforeEach, afterAll } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = path.dirname(fileURLToPath(import.meta.url))
const FRONTEND_ROOT = path.resolve(HERE, '../..')
const WIDGET_PATH = path.join(FRONTEND_ROOT, 'public/widgets/pet-widget.js')
const VOICE_BASE = '/widgets/voices/'

// 记录 playVoice 实际请求的音频 URL
let audioSrcs = []
// 记录 speechSynthesis 实际念出的文本
let spokenTexts = []

class FakeAudio {
  constructor(src) {
    this.src = src
    this.volume = 1
    audioSrcs.push(src)
  }
  play() {
    return Promise.resolve()
  }
}

class FakeUtterance {
  constructor(text) {
    this.text = text
  }
}

const originalAudio = globalThis.Audio
const originalUtterance = globalThis.SpeechSynthesisUtterance
const originalSynth = globalThis.window?.speechSynthesis

/**
 * 造一个 detached 实例：不 append 到文档（connectedCallback 会触发 _init 拉 three.js），
 * 只用公开属性 voice 控制「语音开关」这一维度（attributeChangedCallback 会同步 _voiceOn）。
 */
function makeDetachedWidget({ voice = true } = {}) {
  const el = document.createElement('pet-widget')
  if (voice) el.setAttribute('voice', '')
  else el.removeAttribute('voice')
  return el
}

beforeAll(() => {
  globalThis.Audio = FakeAudio
  globalThis.SpeechSynthesisUtterance = FakeUtterance
  globalThis.window.speechSynthesis = {
    speak(u) {
      spokenTexts.push(u && u.text)
    },
    cancel() {},
  }
  // 间接 eval：在全局作用域执行，脚本里的 HTMLElement / customElements / window 才能拿到
  const src = fs.readFileSync(WIDGET_PATH, 'utf8')
  ;(0, eval)(src)
})

afterAll(() => {
  globalThis.Audio = originalAudio
  globalThis.SpeechSynthesisUtterance = originalUtterance
  if (originalSynth === undefined) delete globalThis.window.speechSynthesis
  else globalThis.window.speechSynthesis = originalSynth
})

/** 清空两个桩的记录（beforeEach 与用例内多次调用前共用） */
function resetSpies() {
  audioSrcs = []
  spokenTexts = []
}

beforeEach(resetSpies)

describe('pet-widget 主题语音语义（speak / setTheme）', () => {
  const THEMES = ['flame', 'tide', 'forest', 'cosmic', 'prism']

  it('speak(\'theme_<id>\') 播放 theme_<id>.mp3，而不是念出 key', () => {
    for (const id of THEMES) {
      resetSpies()
      makeDetachedWidget().speak(`theme_${id}`)
      expect(audioSrcs).toEqual([`${VOICE_BASE}theme_${id}.mp3`])
      expect(spokenTexts).toEqual([])
    }
  })

  it('speak(\'theme_prism\') 不会触发任何文本朗读（回归点）', () => {
    const el = makeDetachedWidget()
    el.speak('theme_prism')
    expect(spokenTexts).toHaveLength(0)
    expect(spokenTexts.join(' ')).not.toContain('theme_')
  })

  it('setTheme(<id>) 换色后播一次 theme_<id>.mp3', () => {
    for (const id of THEMES) {
      resetSpies()
      makeDetachedWidget().setTheme(id)
      expect(audioSrcs).toEqual([`${VOICE_BASE}theme_${id}.mp3`])
      expect(spokenTexts).toEqual([])
    }
  })

  it('未知主题既不发声也不回退成文本朗读', () => {
    makeDetachedWidget().speak('theme_unknown')
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual([])
  })

  it('cheer 仍走原分支：播 cheer_XX.mp3 + 文本「你已经很棒啦！」', () => {
    makeDetachedWidget().speak('cheer')
    expect(audioSrcs).toHaveLength(1)
    expect(audioSrcs[0]).toMatch(/^\/widgets\/voices\/cheer_\d{2}\.mp3$/)
    expect(spokenTexts).toEqual(['你已经很棒啦！'])
  })

  it('greeting / bye 仍走原分支，播对应 mp3 且不念文本', () => {
    makeDetachedWidget().speak('greeting')
    expect(audioSrcs).toEqual([`${VOICE_BASE}greeting_hi.mp3`])
    expect(spokenTexts).toEqual([])

    resetSpies()
    makeDetachedWidget().speak('bye')
    expect(audioSrcs).toEqual([`${VOICE_BASE}greeting_bye.mp3`])
    expect(spokenTexts).toEqual([])
  })

  it('普通文本仍走文本回退（分支原契约，不得被删）', () => {
    makeDetachedWidget().speak('你好')
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual(['你好'])
  })

  it('语音开关关闭时，普通文本不朗读（既有行为不变）', () => {
    makeDetachedWidget({ voice: false }).speak('你好')
    expect(audioSrcs).toEqual([])
    expect(spokenTexts).toEqual([])
  })
})
