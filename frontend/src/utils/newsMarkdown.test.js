import { describe, it, expect } from 'vitest'
import { escapeHtml, inlineMd, renderMarkdown, safeHref } from './newsMarkdown'

/**
 * ---------------------------------------------------------------------------
 * 修复前（origin/main @ 16205d5 的 frontend/src/views/life/NewsTool.vue）的实现，
 * **逐字抄来**只做两件事：
 *   ① 回归对照 —— 正常内容的渲染输出必须与修复前逐字节一致；
 *   ② 反事实对照 —— 证明本文件的断言**能证伪**（旧实现能注入，新实现不能）。
 * 产品代码不会调用这里任何一个函数。
 * ---------------------------------------------------------------------------
 */
function legacyEscapeHtml(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function legacyInlineMd(text) {
  let t = legacyEscapeHtml(text)
  t = t.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, txt, url) => {
    const safe = /^(https?:|mailto:)/i.test(url) ? url : '#'
    return `<a href="${safe}" target="_blank" rel="noopener noreferrer">${txt}</a>`
  })
  t = t.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
  t = t.replace(/(^|[^*])\*([^*\n]+)\*/g, '$1<em>$2</em>')
  t = t.replace(/`([^`]+)`/g, '<code>$1</code>')
  return t
}

function legacyRenderMarkdown(src) {
  if (!src) return ''
  const lines = src.split(/\r?\n/)
  let html = ''
  let inList = false
  const closeList = () => { if (inList) { html += '</ul>'; inList = false } }
  for (const line of lines) {
    if (/^\s*$/.test(line)) { closeList(); continue }
    const h = line.match(/^(#{1,6})\s+(.*)$/)
    if (h) {
      closeList()
      const lv = h[1].length
      html += `<h${lv} class="md-h md-h${lv}">${legacyInlineMd(h[2])}</h${lv}>`
      continue
    }
    if (/^\s*[-*]\s+/.test(line)) {
      if (!inList) { html += '<ul class="md-ul">'; inList = true }
      html += `<li>${legacyInlineMd(line.replace(/^\s*[-*]\s+/, ''))}</li>`
      continue
    }
    if (/^\s*\d+\.\s+/.test(line)) {
      if (!inList) { html += '<ul class="md-ul">'; inList = true }
      html += `<li>${legacyInlineMd(line.replace(/^\s*\d+\.\s+/, ''))}</li>`
      continue
    }
    closeList()
    html += `<p>${legacyInlineMd(line)}</p>`
  }
  closeList()
  return html
}

/**
 * 把渲染结果挂到 DOM 上解析，拿到真实属性 ——
 * 只做字符串匹配分不清「属性值里的裸引号」和「实体」，DOM 才是最终判据。
 */
function parseHtml(html) {
  const box = document.createElement('div')
  box.innerHTML = html
  return box
}

function firstAnchor(html) {
  return parseHtml(html).querySelector('a')
}

/** 属性注入载荷：一个 `"` 就能闭合 href 属性并塞进事件属性 */
const INJECT = '[点我](http://x" onmouseover="alert(1))'

describe('newsMarkdown · escapeHtml', () => {
  it('转义 & < > " \' 五个字符（引号是属性上下文闭合的关键）', () => {
    expect(escapeHtml(`&<>"'`)).toBe('&amp;&lt;&gt;&quot;&#39;')
  })

  it('& 最先转义，不产生二次转义', () => {
    expect(escapeHtml('&lt;script&gt;')).toBe('&amp;lt;script&amp;gt;')
  })

  it('普通文本原样保留', () => {
    expect(escapeHtml('今日要闻 2026-10-27')).toBe('今日要闻 2026-10-27')
  })
})

describe('newsMarkdown · safeHref', () => {
  it('http/https/mailto 放行', () => {
    expect(safeHref('http://example.com/a')).toBe('http://example.com/a')
    expect(safeHref('https://example.com/a')).toBe('https://example.com/a')
    expect(safeHref('mailto:someone@example.com')).toBe('mailto:someone@example.com')
  })

  it('协议白名单大小写不敏感', () => {
    expect(safeHref('HTTPS://example.com/a')).toBe('HTTPS://example.com/a')
    expect(safeHref('MailTo:someone@example.com')).toBe('MailTo:someone@example.com')
  })

  it('非白名单协议降级为 #', () => {
    expect(safeHref('javascript:alert(1)')).toBe('#')
    expect(safeHref('data:text/html,x')).toBe('#')
    expect(safeHref('vbscript:msgbox(1)')).toBe('#')
  })

  it('含引号 / 空白 / 尖括号的 URL 降级为 #', () => {
    expect(safeHref('http://x" onmouseover="alert(1)')).toBe('#')
    expect(safeHref("http://x' onmouseover='alert(1)")).toBe('#')
    expect(safeHref('http://x/a b')).toBe('#')
    expect(safeHref('http://x/<script>')).toBe('#')
  })

  it('& 在属性里被转义，但浏览器解析回来仍是原 URL', () => {
    expect(safeHref('https://example.com/p?q=1&r=2')).toBe('https://example.com/p?q=1&amp;r=2')
  })
})

describe('newsMarkdown · A1 href 属性注入（必测）', () => {
  it('输出不得出现注入的事件属性名', () => {
    const html = inlineMd(INJECT)
    expect(html).not.toMatch(/onmouseover\s*=/)
    expect(html).not.toMatch(/onmouseover/i)
  })

  it('DOM 解析后 anchor 只有 href/target/rel 三个属性，没有被注入的属性', () => {
    const anchor = firstAnchor(inlineMd(INJECT))
    expect(anchor).not.toBeNull()
    const names = Array.from(anchor.attributes).map((a) => a.name)
    expect(names).toEqual(['href', 'target', 'rel'])
    expect(names.some((n) => /^on/i.test(n))).toBe(false)
  })

  it('href 属性值里没有裸引号 / 尖括号，且降级为 #', () => {
    const anchor = firstAnchor(inlineMd(INJECT))
    const href = anchor.getAttribute('href')
    expect(href).toBe('#')
    expect(href).not.toContain('"')
    expect(href).not.toContain('<')
    expect(href).not.toContain('>')
  })

  it('单引号载荷同样被拦截', () => {
    const html = inlineMd("[点我](http://x' onmouseover='alert(1))")
    expect(html).not.toMatch(/onmouseover/i)
    expect(firstAnchor(html).getAttribute('href')).toBe('#')
  })

  it('反事实：修复前的实现确实会被注入（本断言可证伪）', () => {
    // 旧实现输出里存在被闭合的 href + onmouseover 事件属性
    expect(legacyInlineMd(INJECT)).toMatch(/onmouseover\s*=/)
    // 新实现没有
    expect(inlineMd(INJECT)).not.toMatch(/onmouseover\s*=/)
    // 且旧实现的 anchor 确实多出第 4 个属性（事件属性）
    const legacyAnchor = firstAnchor(legacyInlineMd(INJECT))
    expect(Array.from(legacyAnchor.attributes).map((a) => a.name)).toContain('onmouseover')
  })
})

describe('newsMarkdown · B 标签注入', () => {
  it('<script> 被转义成文本，不产生真实 script 标签', () => {
    const html = inlineMd('<script>alert(1)</script>')
    expect(html).not.toMatch(/<script/i)
    expect(html).toContain('&lt;script&gt;')
    expect(parseHtml(html).querySelector('script')).toBeNull()
  })

  it('<img onerror> 被转义成文本，不产生真实 img 标签', () => {
    const html = inlineMd('<img src=x onerror=alert(1)>')
    expect(html).not.toMatch(/<img/i)
    expect(html).toContain('&lt;img')
    expect(parseHtml(html).querySelector('img')).toBeNull()
  })

  it('标签注入的渲染结果与修复前一致（这块不是本次缺陷，行为不应变化）', () => {
    expect(inlineMd('<script>alert(1)</script>')).toBe(legacyInlineMd('<script>alert(1)</script>'))
  })
})

describe('newsMarkdown · C 协议白名单', () => {
  it.each([
    ['javascript:alert(1)', 'javascript:alert(1'],
    ['data:text/html,<script>alert(1)</script>', 'data:text/html,&lt;script&gt;alert(1)&lt;/script&gt;'],
    ['vbscript:msgbox(1)', 'vbscript:msgbox(1']
  ])('%s → href="#"', (_label, url) => {
    const html = inlineMd(`[点我](${url})`)
    expect(firstAnchor(html).getAttribute('href')).toBe('#')
  })

  it('白名单生效时保留 target / rel', () => {
    const anchor = firstAnchor(inlineMd('[a](javascript:alert(1))'))
    expect(anchor.getAttribute('target')).toBe('_blank')
    expect(anchor.getAttribute('rel')).toBe('noopener noreferrer')
  })
})

describe('newsMarkdown · D 正常链接', () => {
  it('href 保留原链接（& 渲染成实体是正确行为）', () => {
    const html = inlineMd('[标题](https://example.com/a?b=1&c=2)')
    const anchor = firstAnchor(html)
    // 原始 HTML 里 & 必须是实体
    expect(html).toContain('href="https://example.com/a?b=1&amp;c=2"')
    // 浏览器解析回来的值仍是原 URL
    expect(anchor.getAttribute('href')).toBe('https://example.com/a?b=1&c=2')
  })

  it('带 target="_blank" 与 rel="noopener noreferrer"', () => {
    const anchor = firstAnchor(inlineMd('[标题](https://example.com/a?b=1&c=2)'))
    expect(anchor.getAttribute('target')).toBe('_blank')
    expect(anchor.getAttribute('rel')).toBe('noopener noreferrer')
    expect(Array.from(anchor.attributes).map((a) => a.name)).toEqual(['href', 'target', 'rel'])
  })

  it('链接文本被转义，不会夹带标签', () => {
    const html = inlineMd('[<b>标题</b>](https://example.com/a)')
    expect(html).not.toMatch(/<b>/)
    expect(firstAnchor(html).textContent).toBe('<b>标题</b>')
  })
})

describe('newsMarkdown · E mailto', () => {
  it('mailto 正常放行', () => {
    const anchor = firstAnchor(inlineMd('[写信](mailto:someone@example.com)'))
    expect(anchor.getAttribute('href')).toBe('mailto:someone@example.com')
    expect(anchor.getAttribute('target')).toBe('_blank')
  })
})

describe('newsMarkdown · F 回归（正常内容渲染结果与修复前一致）', () => {
  const NORMAL_SAMPLES = [
    '今日要闻 2026-10-27',
    '**加粗** 与 *斜体* 与 `代码`',
    '[标题](https://example.com/a?b=1&c=2)',
    '[标题](http://example.com/a)',
    '[写信](mailto:someone@example.com)',
    '普通段落 & 带 & 符号',
    'emoji 🎉 与中文标点，。！？'
  ]

  it.each(NORMAL_SAMPLES)('inlineMd 逐字节一致：%s', (sample) => {
    expect(inlineMd(sample)).toBe(legacyInlineMd(sample))
  })

  it('renderMarkdown 逐字节一致（标题 / 列表 / 段落）', () => {
    const src = [
      '# 今日要闻',
      '',
      '第一段文字，含 [链接](https://example.com/a?b=1&c=2)。',
      '',
      '## 二级标题',
      '',
      '- 列表项一',
      '- 列表项二 **加粗**',
      '',
      '1. 有序项',
      '2. 有序项二'
    ].join('\n')
    expect(renderMarkdown(src)).toBe(legacyRenderMarkdown(src))
  })

  it('renderMarkdown 空输入返回空串', () => {
    expect(renderMarkdown('')).toBe('')
    expect(renderMarkdown(null)).toBe('')
    expect(renderMarkdown(undefined)).toBe('')
  })

  it('普通文本里的引号被转义成实体（不破坏属性边界）', () => {
    const html = inlineMd('他说"今天很好"，还有 \'单引号\'')
    expect(html).toContain('&quot;')
    expect(html).toContain('&#39;')
    expect(html).not.toContain('"今天很好"')
  })
})
