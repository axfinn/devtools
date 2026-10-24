import { describe, it, expect } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// FINN-165 Stage5 · EnglishTutor 三特性移植（FINN-163）的行为回归 + 静态校验
//
// 范式照抄本仓库先例 frontend/src/views/ai/resultShareStaticRules.test.js（FINN-66 Stage5 产物）：
// 用 node:fs 读 .vue 源码文本 → 切出 <template> / <script setup> 区段 → 做静态 / 行为断言。
// frontend/vitest.config.js 的 include 是 ['src/**/*.test.js', 'src/**/*.spec.js'] ⇒ 本文件被自动收集。
//
// 覆盖三组：
//   A. 行为断言 —— 从 <script setup> 里按「函数名起点 + 大括号配平」截出源码文本，
//      再用 new Function 把它取出来真跑一遍（证明 bug 真被修掉，而不只是 grep 到关键字）：
//        · detectTextLanguage —— 语种识别
//        · chooseVoice        —— 语种自适应朗读（核心修复点：英文文本必须用英文语音）
//        · formatHistoryTime  —— 历史记录相对时间戳
//   B. 反回退断言 —— 防止回退 FINN-128 / FINN-137 已合入的加固（parseJsonContent 平衡大括号解析、
//      F5 空响应提示、removeHistory / restoreHistory），同时锁住三特性自身的接线（离页停读、时间戳刷新）。
//   C. R3 / R4 静态断言 —— <template> 里不得出现 .value（模板会自动 unwrap ref）、
//      不得出现裸下标 obj.x[0]（field 缺失时炸挂载，要写 obj.x?.[0]）。
//
// 反事实自检（证明闸门不是恒真）—— 同一套抽取 + 断言逻辑作用在「未合入三特性」的版本上：
//   ETT_VUE_PATH=/tmp/ett-main/EnglishTutorTool.vue \
//     pnpm exec vitest run src/views/ai/EnglishTutorTool.tts.test.js
//   ⇒ detectTextLanguage / formatHistoryTime 根本抽不出来，chooseVoice 的英文文本断言也不成立 ⇒ 必须失败。
// 默认路径恒为同目录下的 EnglishTutorTool.vue（合进 main 后 CI 复用同一个测试）。

const here = path.dirname(fileURLToPath(import.meta.url))

// 允许用环境变量把被测 .vue 指到别处（反事实自检用）；默认就是同目录的真实文件。
const VUE_FILE = process.env.ETT_VUE_PATH
  ? path.resolve(process.env.ETT_VUE_PATH)
  : path.resolve(here, './EnglishTutorTool.vue')

// 语言标签在源文件里是 \uXXXX 转义原文（如 '中文'）；下面用等价的字面量，
// 运行时两者完全相同（PM 已裁决：不得因源文件是转义串就断言失败）。
const ZH = '中文'
const JA = '日文'
const KO = '韩文'
const EN = '英文'

// 4 个语音名不得改变（Azure Neural TTS voice name）
const VOICE_ZH = 'zh-CN-XiaoxiaoNeural'
const VOICE_JA = 'ja-JP-NanamiNeural'
const VOICE_KO = 'ko-KR-SunHiNeural'
const VOICE_EN = 'en-US-JennyNeural'

function readSource(file) {
  return fs.readFileSync(file, 'utf8')
}

function templateOf(source) {
  const start = source.indexOf('<template>')
  const end = source.lastIndexOf('</template>')
  expect(start, '应当有 <template> 块').toBeGreaterThanOrEqual(0)
  expect(end).toBeGreaterThan(start)
  return source.slice(start, end)
}

function scriptSetupOf(source) {
  const start = source.indexOf('<script setup>')
  const end = source.lastIndexOf('</script>')
  expect(start, '应当有 <script setup> 块').toBeGreaterThanOrEqual(0)
  expect(end).toBeGreaterThan(start)
  return source.slice(start, end)
}

// 跳过一行引号包裹的字符串（含转义），返回结束引号之后的位置
function skipQuoted(text, i) {
  const quote = text[i]
  let j = i + 1
  while (j < text.length) {
    const ch = text[j]
    if (ch === '\\') {
      j += 2
      continue
    }
    if (ch === quote) return j + 1
    j += 1
  }
  return text.length
}

// 跳过正则字面量（含字符类 [...]），返回结束斜杠之后的位置
function skipRegex(text, i) {
  let j = i + 1
  let inClass = false
  while (j < text.length) {
    const ch = text[j]
    if (ch === '\\') {
      j += 2
      continue
    }
    if (ch === '\n') return j // 不是正则，按普通字符处理
    if (ch === '[') inClass = true
    else if (ch === ']') inClass = false
    else if (ch === '/' && !inClass) return j + 1
    j += 1
  }
  return text.length
}

// 按「function <name>(」起点 + 大括号配平截出函数源码文本（不用贪婪正则）
function extractFunctionSource(script, name) {
  const marker = `function ${name}(`
  const start = script.indexOf(marker)
  if (start < 0) {
    throw new Error(`未找到函数定义：${name}`)
  }
  const bodyStart = script.indexOf('{', start + marker.length)
  if (bodyStart < 0) {
    throw new Error(`函数 ${name} 没有函数体`)
  }
  let depth = 0
  let i = bodyStart
  while (i < script.length) {
    const ch = script[i]
    if (ch === '/' && script[i + 1] === '/') {
      const nl = script.indexOf('\n', i)
      i = nl < 0 ? script.length : nl
      continue
    }
    if (ch === '/' && script[i + 1] === '*') {
      const end = script.indexOf('*/', i + 2)
      i = end < 0 ? script.length : end + 2
      continue
    }
    if (ch === '"' || ch === "'" || ch === '`') {
      i = skipQuoted(script, i)
      continue
    }
    if (ch === '/') {
      i = skipRegex(script, i)
      continue
    }
    if (ch === '{') {
      depth += 1
    } else if (ch === '}') {
      depth -= 1
      if (depth === 0) {
        return script.slice(start, i + 1)
      }
    }
    i += 1
  }
  throw new Error(`函数 ${name} 的大括号没有配平`)
}

const source = readSource(VUE_FILE)
const template = templateOf(source)
const script = scriptSetupOf(source)

// 抽不到函数时返回 null（依赖缺失时不致命：函数体里的引用会在调用时抛 ReferenceError）
function tryExtract(name) {
  try {
    return extractFunctionSource(script, name)
  } catch {
    return null
  }
}

// 抽一个函数出来执行：deps 是它依赖的其它函数（一起塞进同一个作用域）。
// 抽不到时返回 { fn: null, error }，让断言能报出「缺哪个函数」，而不是整份文件加载失败。
function makeFn(name, deps = []) {
  const parts = []
  for (const dep of deps) {
    const depSrc = tryExtract(dep)
    if (depSrc) parts.push(depSrc)
  }
  let own
  try {
    own = extractFunctionSource(script, name)
  } catch (err) {
    return { fn: null, error: err.message }
  }
  parts.push(own)
  try {
    const factory = new Function(`${parts.join('\n\n')}\n\nreturn { ${name} }`)
    return { fn: factory()[name], error: null }
  } catch (err) {
    return { fn: null, error: `函数 ${name} 装配失败：${err.message}` }
  }
}

// 抽取失败时直接抛，行为断言的失败信息里就能看到「未找到函数定义：xxx」
function requireFn(loaded) {
  if (loaded.error) {
    throw new Error(loaded.error)
  }
  return loaded.fn
}

const detect = makeFn('detectTextLanguage')
const voice = makeFn('chooseVoice', ['detectTextLanguage'])
const time = makeFn('formatHistoryTime')

describe('FINN-165 Stage5 A · 行为断言：detectTextLanguage（语种识别）', () => {
  it('应当能从 <script setup> 里抽出 detectTextLanguage', () => {
    expect(detect.error, '应当能抽取出 detectTextLanguage').toBeNull()
  })

  it('空输入 → 返回空串（回落到 targetLanguage）', () => {
    expect(requireFn(detect)('')).toBe('')
  })

  it('纯英文 → 英文', () => {
    expect(requireFn(detect)('hello world')).toBe(EN)
  })

  it('纯中文 → 中文', () => {
    expect(requireFn(detect)('你好')).toBe(ZH)
  })

  it('日文假名 → 日文', () => {
    expect(requireFn(detect)('こんにちは')).toBe(JA)
  })

  it('韩文谚文 → 韩文', () => {
    expect(requireFn(detect)('안녕하세요')).toBe(KO)
  })

  it('只有数字和符号 → 检测不到（返回空串）', () => {
    expect(requireFn(detect)('123 !@#')).toBe('')
  })

  it('null / undefined 输入 → 返回空串（不抛错）', () => {
    expect(requireFn(detect)(null)).toBe('')
    expect(requireFn(detect)(undefined)).toBe('')
  })
})

describe('FINN-165 Stage5 A · 行为断言：chooseVoice（语种自适应朗读 · 核心修复点）', () => {
  it('应当能从 <script setup> 里抽出 chooseVoice', () => {
    expect(voice.error, '应当能抽取出 chooseVoice').toBeNull()
  })

  it('英文文本 + 朗读目标语言是中文 → 仍用英文语音（bug 修复点）', () => {
    expect(requireFn(voice)('hello world', ZH)).toBe(VOICE_EN)
  })

  it('中文文本 + 朗读目标语言是英文 → 用中文语音', () => {
    expect(requireFn(voice)('你好', EN)).toBe(VOICE_ZH)
  })

  it('检测不到语种的文本（纯数字）→ 回落 targetLanguage', () => {
    expect(requireFn(voice)('123', ZH)).toBe(VOICE_ZH)
  })

  it('日文文本 → 日文语音', () => {
    expect(requireFn(voice)('こんにちは', ZH)).toBe(VOICE_JA)
  })

  it('韩文文本 → 韩文语音', () => {
    expect(requireFn(voice)('안녕하세요', ZH)).toBe(VOICE_KO)
  })

  it('英文文本 → 英文语音', () => {
    expect(requireFn(voice)('hello world', ZH)).toBe(VOICE_EN)
  })

  it('空文本 → 回落 targetLanguage（英文目标 ⇒ 英文语音）', () => {
    expect(requireFn(voice)('', EN)).toBe(VOICE_EN)
  })

  it('targetLanguage 也不是已知语种 → 落 default 分支（英文语音）', () => {
    expect(requireFn(voice)('123', 'fr-FR')).toBe(VOICE_EN)
  })

  it('4 个语音名不得改变', () => {
    const names = [
      requireFn(voice)('你好', ZH),
      requireFn(voice)('こんにちは', ZH),
      requireFn(voice)('안녕하세요', ZH),
      requireFn(voice)('hello', ZH),
    ]
    expect(names).toEqual([VOICE_ZH, VOICE_JA, VOICE_KO, VOICE_EN])
  })
})

describe('FINN-165 Stage5 A · 行为断言：formatHistoryTime（历史相对时间戳）', () => {
  it('应当能从 <script setup> 里抽出 formatHistoryTime', () => {
    expect(time.error, '应当能抽取出 formatHistoryTime').toBeNull()
  })

  it('时间戳 0 → 空串', () => {
    expect(requireFn(time)(0)).toBe('')
  })

  it('时间戳 NaN → 空串', () => {
    expect(requireFn(time)(NaN)).toBe('')
  })

  it('空值 / null → 空串', () => {
    expect(requireFn(time)(null)).toBe('')
    expect(requireFn(time)(undefined)).toBe('')
  })

  it('当前时刻 → 刚刚', () => {
    expect(requireFn(time)(Date.now())).toBe('刚刚')
  })

  it('5 分钟前 → "5 分钟前"', () => {
    expect(requireFn(time)(Date.now() - 5 * 60_000)).toBe('5 分钟前')
  })

  it('3 小时前 → "3 小时前"', () => {
    expect(requireFn(time)(Date.now() - 3 * 3600_000)).toBe('3 小时前')
  })

  it('2 天前 → "2 天前"', () => {
    expect(requireFn(time)(Date.now() - 2 * 86400_000)).toBe('2 天前')
  })

  it('30 天前 → 绝对日期 YYYY-MM-DD', () => {
    expect(requireFn(time)(Date.now() - 30 * 86400_000)).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })
})

describe('FINN-165 Stage5 B · 反回退断言（FINN-128 / FINN-137 已合入的加固不得被回退）', () => {
  it('parseJsonContent 存在，且仍用平衡大括号解析（含 let depth = 0）', () => {
    expect(script).toContain('function parseJsonContent(')
    expect(script).toContain('let depth = 0')
  })

  it('F5 空响应提示仍在（AI 返回为空，请稍后重试或更换模型）', () => {
    expect(script).toContain('AI 返回为空，请稍后重试或更换模型')
  })

  it('removeHistory / restoreHistory 仍存在', () => {
    expect(script).toContain('function removeHistory(')
    expect(script).toContain('function restoreHistory(')
  })

  it('特性三：refreshHistoryTimestamps 存在且被调用', () => {
    expect(script).toContain('function refreshHistoryTimestamps(')
    expect(script).toContain('refreshHistoryTimestamps()')
  })

  it('特性二（离页停读）：onBeforeUnmount 里调用 stopSpeak，stopSpeak 取消 speechSynthesis', () => {
    expect(script).toContain('function stopSpeak(')
    expect(script).toContain('onBeforeUnmount(() => {')
    const unmountIdx = script.indexOf('onBeforeUnmount(() => {')
    const stopIdx = script.indexOf('stopSpeak()', unmountIdx)
    expect(stopIdx, 'onBeforeUnmount 里应当调用 stopSpeak()').toBeGreaterThan(unmountIdx)
    expect(script).toContain('speechSynthesis.cancel()')
  })

  it('模板里渲染了历史相对时间戳（history-item-time）', () => {
    expect(template).toContain('history-item-time')
    expect(template).toContain('item.createdAtLabel')
  })

  it('调试残留自查：无 console.log / TODO / FIXME / XXX / HACK', () => {
    for (const needle of ['console.log', 'TODO', 'FIXME', 'XXX', 'HACK']) {
      expect(source.includes(needle), `源码里有调试残留：${needle}`).toBe(false)
    }
  })
})

describe('FINN-165 Stage5 C · R3 / R4 静态断言', () => {
  it('R3：模板里不得出现 .value（模板会自动 unwrap ref）', () => {
    expect(template.includes('.value'), '模板里出现了 .value').toBe(false)
  })

  it('R3 补强：不得对 ref/computed 绑定写 ?.value', () => {
    const bindings = new Set()
    const re = /\bconst\s+([A-Za-z_$][\w$]*)\s*=\s*(?:ref|computed|shallowRef)\s*\(/g
    let m
    while ((m = re.exec(script)) !== null) {
      bindings.add(m[1])
    }
    expect(bindings.size, '应当解析出 ref/computed 绑定').toBeGreaterThan(0)
    const violations = []
    for (const name of bindings) {
      if (new RegExp(`\\b${name}\\s*\\?\\.value\\b`).test(template)) {
        violations.push(name)
      }
    }
    expect(violations, `R3 违规：${violations.join(', ')}`).toEqual([])
  })

  it('R4：模板里不得出现裸下标 obj.x[0]（要写 obj.x?.[0]）', () => {
    const violations = []
    const re = /([A-Za-z_$][\w$]*)\s*\[/g
    let m
    while ((m = re.exec(template)) !== null) {
      const before = template.slice(Math.max(0, m.index - 3), m.index)
      if (before.endsWith('?.')) continue // 可选链允许
      violations.push(m[0].trim())
    }
    expect(violations, `R4 违规（模板裸下标）：${violations.join(', ')}`).toEqual([])
  })
})
