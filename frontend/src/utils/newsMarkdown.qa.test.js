import { describe, it, expect } from 'vitest'
import { escapeHtml, inlineMd, renderMarkdown } from './newsMarkdown'

/**
 * FINN-189 · Stage 5 独立回归用例（devtools-qa 自写，不是 Stage 3 用例的副本）
 *
 * 覆盖父单 FINN-184 的验收断言 A1–A6。判据方式刻意与 Stage 3 不同：
 * Stage 3 主要做「渲染字符串包含/不包含某段文本」；本文件一律以 **DOM 解析后的
 * 属性集** 为准（浏览器真正看到的东西），并额外用「属性分隔引号计数」这一
 * 不依赖实现细节的不变量做二次判据。
 *
 * 断言只 import escapeHtml / inlineMd / renderMarkdown 三个公共函数，
 * 不依赖 Stage 3 额外导出的 safeHref —— 这样反事实时可以把
 * origin/main 的缺陷实现原样装进 newsMarkdown.js 而不必改本文件。
 */

const LINK_ATTR_NAMES = ['href', 'target', 'rel']

/** 一段含一个链接的渲染结果里，裸双引号的数量 = 属性个数 × 2（属性分隔符） */
const LINK_QUOTE_COUNT = LINK_ATTR_NAMES.length * 2

/** 片段里允许出现的标签（Markdown 渲染不会产生别的标签） */
const ALLOWED_TAGS = ['a', 'strong', 'em', 'code', 'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li']

/** 把渲染结果交给浏览器同款解析器，拿到真实 DOM */
function parseFragment(html) {
  const box = document.createElement('div')
  box.innerHTML = html
  return box
}

/** 片段里第一个 <a> 的属性名列表（按文档顺序） */
function linkAttrNames(html) {
  const a = parseFragment(html).querySelector('a')
  return a ? [...a.attributes].map((x) => x.name) : []
}

/** 片段里所有元素的所有属性，用来抓「注入出来的事件属性」 */
function allAttributes(html) {
  const out = []
  for (const el of parseFragment(html).querySelectorAll('*')) {
    for (const attr of el.attributes) {
      out.push({ tag: el.tagName.toLowerCase(), name: attr.name, value: attr.value })
    }
  }
  return out
}

/** 事件属性（on*）：任何一个出现都说明属性被闭合注入了 */
function eventAttributes(html) {
  return allAttributes(html).filter((a) => /^on/i.test(a.name))
}

/** 片段里出现的标签名 */
function tagNames(html) {
  return [...parseFragment(html).querySelectorAll('*')].map((el) => el.tagName.toLowerCase())
}

/** 原始 HTML 里裸双引号的个数（属性分隔符）。文本里的引号必须已被转义成实体 */
function rawQuoteCount(html) {
  return (html.match(/"/g) || []).length
}

/** href 是否是安全值：'#' 降级，或干净的白名单协议 URL */
function hrefIsSafe(html) {
  const a = parseFragment(html).querySelector('a')
  if (!a) return false
  const href = a.getAttribute('href') || ''
  if (href === '#') return true
  if (!/^(https?:|mailto:)/i.test(href)) return false
  return !/[\s"'`<>]/.test(href)
}

/**
 * 「修复后必须成立」的全套判据（= A1 的结构判据 + href 安全判据）。
 * A6 用它做反事实：缺陷实现上至少有一条不成立，修复后必须全部成立。
 */
function criteriaHold(html) {
  return eventAttributes(html).length === 0 &&
    linkAttrNames(html).join(',') === LINK_ATTR_NAMES.join(',') &&
    rawQuoteCount(html) === LINK_QUOTE_COUNT &&
    hrefIsSafe(html)
}

/**
 * A1 载荷表：每条都试图用引号闭合 href 属性、注入事件属性。
 * 判据 = DOM 里不得出现任何 on* 属性，且 <a> 的属性集必须恰好是 href/target/rel。
 * discriminating=false 的载荷是「两种实现行为一致」的加固项（见 A6 说明）。
 */
const INJECTION_PAYLOADS = [
  { id: 'onmouseover', input: '[点我](http://x" onmouseover="alert(1))' },
  { id: 'onfocus', input: '[点我](https://x" onfocus="alert(1))' },
  { id: 'onerror', input: '[点我](http://x" onerror="alert(1))' },
  { id: 'onclick', input: '[点我](http://x" onclick="alert(1))' },
  { id: 'style', input: '[点我](http://x" style="width:100%)' },
  { id: 'autofocus', input: '[点我](http://x" autofocus="true)' },
  { id: '单引号', input: "[点我](http://x' onmouseover='alert(1))" },
  { id: '文本里塞标签', input: '[<img src=x onerror=alert(1)>](https://example.com)', discriminating: false },
  { id: '文本里塞引号+事件', input: '[a" onmouseover="alert(1)](https://example.com)' }
]

/** 能区分「缺陷实现」与「修复实现」的载荷（用于 A6 反事实） */
const DISCRIMINATING_PAYLOADS = INJECTION_PAYLOADS.filter((p) => p.discriminating !== false)

describe('FINN-189 · A1 href 属性注入（DOM 属性集判据）', () => {
  it.each(INJECTION_PAYLOADS)('载荷 [$id] 渲染不出任何事件属性', ({ input }) => {
    const html = inlineMd(input)
    expect(eventAttributes(html)).toEqual([])
  })

  it.each(INJECTION_PAYLOADS)('载荷 [$id] 的 <a> 属性集恰好是 href/target/rel', ({ input }) => {
    const html = inlineMd(input)
    expect(linkAttrNames(html)).toEqual(LINK_ATTR_NAMES)
  })

  it.each(INJECTION_PAYLOADS)('载荷 [$id] 不产生 HTML 注入（标签白名单）', ({ input }) => {
    for (const tag of tagNames(inlineMd(input))) {
      expect(ALLOWED_TAGS).toContain(tag)
    }
  })

  it.each(INJECTION_PAYLOADS)('载荷 [$id] 的裸引号计数 = 属性个数 × 2', ({ input }) => {
    expect(rawQuoteCount(inlineMd(input))).toBe(LINK_QUOTE_COUNT)
  })

  it('注入载荷不会把 href 变成可执行的 javascript: 地址', () => {
    const a = parseFragment(inlineMd('[点我](http://x" onmouseover="alert(1))')).querySelector('a')
    const href = a.getAttribute('href')
    expect(href).toBe('#')
    expect(/^\s*javascript:/i.test(href)).toBe(false)
  })

  it('块级渲染里的注入载荷同样受保护', () => {
    const html = renderMarkdown('- [点我](http://x" onmouseover="alert(1))')
    expect(eventAttributes(html)).toEqual([])
    expect(linkAttrNames(html)).toEqual(LINK_ATTR_NAMES)
  })
})

describe('FINN-189 · A2 正常链接仍然可用', () => {
  it('带查询串的 https 链接：DOM 里 href 还原成原 URL，且可被 URL 解析', () => {
    const html = inlineMd('[a](https://example.com/p?q=1&r=2)')
    const a = parseFragment(html).querySelector('a')
    expect(a.getAttribute('href')).toBe('https://example.com/p?q=1&r=2')
    const u = new URL(a.getAttribute('href'))
    expect(u.protocol).toBe('https:')
    expect(u.searchParams.get('q')).toBe('1')
    expect(u.searchParams.get('r')).toBe('2')
  })

  it('& 在源码里被转义成实体，但浏览器解析回来还是原 URL（可用性不受影响）', () => {
    const html = inlineMd('[a](https://example.com/p?q=1&r=2)')
    expect(html).toContain('&amp;')
    expect(html).not.toContain('?q=1&r=2')
  })

  it('保留 target=_blank rel=noopener noreferrer', () => {
    const a = parseFragment(inlineMd('[a](https://example.com/x)')).querySelector('a')
    expect(a.getAttribute('target')).toBe('_blank')
    expect(a.getAttribute('rel')).toBe('noopener noreferrer')
  })

  it('http 与 mailto 链接照常可用', () => {
    expect(parseFragment(inlineMd('[a](http://example.com/x)')).querySelector('a').getAttribute('href'))
      .toBe('http://example.com/x')
    expect(parseFragment(inlineMd('[写信](mailto:someone@example.com)')).querySelector('a').getAttribute('href'))
      .toBe('mailto:someone@example.com')
  })
})

describe('FINN-189 · A3 协议白名单仍然生效', () => {
  const BAD_SCHEMES = [
    '[a](javascript:alert(1))',
    '[a](JaVaScRiPt:alert(1))',
    '[a](data:text/html,x)',
    '[a](vbscript:msgbox(1))',
    '[a](ftp://example.com/f)',
    '[a](file:///etc/passwd)'
  ]

  it.each(BAD_SCHEMES)('%s 降级为 href="#"', (input) => {
    const a = parseFragment(inlineMd(input)).querySelector('a')
    expect(a.getAttribute('href')).toBe('#')
    expect(linkAttrNames(inlineMd(input))).toEqual(LINK_ATTR_NAMES)
  })

  it('降级后的链接同样不产生事件属性', () => {
    for (const input of BAD_SCHEMES) {
      expect(eventAttributes(inlineMd(input))).toEqual([])
    }
  })
})

describe('FINN-189 · A4 既有渲染效果不变', () => {
  it('加粗 / 斜体 / 行内代码', () => {
    expect(parseFragment(inlineMd('**粗**')).querySelector('strong').textContent).toBe('粗')
    expect(parseFragment(inlineMd('*斜*')).querySelector('em').textContent).toBe('斜')
    expect(parseFragment(inlineMd('`码`')).querySelector('code').textContent).toBe('码')
  })

  it('链接文本内的加粗照旧生效，且链接本身完好', () => {
    const box = parseFragment(inlineMd('[**粗**](https://example.com)'))
    expect(box.querySelector('a > strong').textContent).toBe('粗')
    expect(box.querySelector('a').getAttribute('href')).toBe('https://example.com')
  })

  it('块级渲染：标题 / 列表 / 段落', () => {
    expect(parseFragment(renderMarkdown('# 标题')).querySelector('h1').textContent).toBe('标题')
    expect(parseFragment(renderMarkdown('- 一\n- 二')).querySelectorAll('li')).toHaveLength(2)
    expect(parseFragment(renderMarkdown('正文')).querySelector('p').textContent).toBe('正文')
  })
})

describe('FINN-189 · A5 普通文本里的引号不破坏属性边界', () => {
  it('双引号 / 单引号被转义成实体，文本内容还原后不变', () => {
    const src = '他说"今天"很好 \'行\''
    const html = inlineMd(src)
    expect(html).toContain('&quot;')
    expect(html).toContain('&#39;')
    expect(rawQuoteCount(html)).toBe(0)
    expect(parseFragment(html).textContent).toBe(src)
  })

  it('escapeHtml 转义 & < > " \' 五个字符（& 必须最先，否则实体被二次转义）', () => {
    expect(escapeHtml(`&<>"'`)).toBe('&amp;&lt;&gt;&quot;&#39;')
  })

  it('文本里的引号 + 链接混排时链接依然完好', () => {
    const html = inlineMd('"引用"[a](https://example.com/x)')
    const a = parseFragment(html).querySelector('a')
    expect(a.getAttribute('href')).toBe('https://example.com/x')
    expect(linkAttrNames(html)).toEqual(LINK_ATTR_NAMES)
    expect(rawQuoteCount(html)).toBe(LINK_QUOTE_COUNT)
    expect(parseFragment(html).textContent).toContain('"引用"')
  })

  it('链接文本里的引号被转义，不影响属性边界', () => {
    const html = inlineMd('[a"b](https://example.com/x)')
    expect(html).toContain('&quot;')
    expect(rawQuoteCount(html)).toBe(LINK_QUOTE_COUNT)
    expect(parseFragment(html).querySelector('a').getAttribute('href')).toBe('https://example.com/x')
  })
})

// ── A6 反事实 ────────────────────────────────────────────────────────────
// 把 origin/main @ 16205d5 的缺陷实现（NewsTool.vue 里的 escapeHtml / inlineMd，
// 原样搬运，只为在测试里跑同一组载荷）放进来，证明上面的断言**可证伪**：
// 缺陷实现必须真的命中注入，修复后的实现必须真的不命中。
// 这段代码只用于测试，不得复制进生产代码。
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

describe('FINN-189 · A6 反事实（同一组载荷在缺陷实现上必须命中注入）', () => {
  it('缺陷实现的 escapeHtml 不转义引号（根因）', () => {
    expect(legacyEscapeHtml('"')).toBe('"')
    expect(legacyEscapeHtml("'")).toBe("'")
  })

  it.each(DISCRIMINATING_PAYLOADS)('缺陷实现对载荷 [$id] 不满足修复后的判据（闸门可证伪）', ({ input }) => {
    expect(criteriaHold(legacyInlineMd(input))).toBe(false)
  })

  it.each(INJECTION_PAYLOADS)('修复实现对载荷 [$id] 满足全部判据', ({ input }) => {
    expect(criteriaHold(inlineMd(input))).toBe(true)
  })

  it('缺陷实现确实注入出 onmouseover 事件属性，修复后同一载荷不再注入', () => {
    const payload = '[点我](http://x" onmouseover="alert(1))'
    const legacyAnchor = parseFragment(legacyInlineMd(payload)).querySelector('a')
    expect(legacyAnchor.getAttribute('onmouseover')).not.toBeNull()

    const fixedAnchor = parseFragment(inlineMd(payload)).querySelector('a')
    expect(fixedAnchor.getAttribute('onmouseover')).toBeNull()
    expect(eventAttributes(inlineMd(payload))).toEqual([])
  })
})
