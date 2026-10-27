import { describe, it, expect } from 'vitest'
import { escapeHtml, inlineMd, renderMarkdown, safeHref } from './newsMarkdown'

/**
 * 把渲染结果挂到 DOM 上解析，拿到真实属性 —— 只做字符串匹配分不清
 * 「属性里的裸引号」和「属性值里的实体」，DOM 才是最终判据。
 */
function firstAnchor(html) {
  const box = document.createElement('div')
  box.innerHTML = html
  return box.querySelector('a')
}

/** 属性注入载荷：一个 " 就能闭合 href 属性并塞进事件属性 */
const INJECT = '[点我](http://x" onmouseover="alert(1))'

describe('newsMarkdown · escapeHtml', () => {
  it('转义 & < > " \' 五个字符（引号是属性上下文闭合的关键）', () => {
    expect(escapeHtml(`&<>"'`)).toBe('&amp;&lt;&gt;&quot;&#39;')
  })

  it('普通文本原样保留', () => {
    expect(escapeHtml('今日要闻 2026-10-27')).toBe('今日要闻 2026-10-27')
  })
})

describe('newsMarkdown · safeHref', () => {
  it('http/https/mailto 通过', () => {
    expect(safeHref('http://example.com/a')).toBe('http://example.com/a')
    expect(safeHref('https://example.com/a')).toBe('https://example.com/a')
    expect(safeHref('mailto:someone@example.com')).toBe('mailto:someone@example.com')
  })

  it('非白名单协议降级为 #', () => {
    expect(safeHref('javascript:alert(1)')).toBe('#')
    expect(safeHref('data:text/html,x')).toBe('#')
    expect(safeHref('vbscript:msgbox(1)')).toBe('#')
  })

  it('含引号 / 空白 / 尖括号的 URL 降级为 #', () => {
    expect(safeHref('http://x" onmouseover="alert(1)')).toBe('#')
    expect(safeHref('http://x"onmouseover="alert(1)')).toBe('#')
    expect(safeHref('http://x/a b')).toBe('#')
    expect(safeHref('http://x/<script>')).toBe('#')
  })

  it('& 在属性里被转义，但浏览器解析回来仍是原 URL', () => {
    expect(safeHref('https://example.com/p?q=1&r=2')).toBe('https://example.com/p?q=1&amp;r=2')
  })
})

describe('newsMarkdown · A1 href 属性注入', () => {
  it('注入载荷渲染不出事件属性，且链接降级为 #', () => {
    const html = inlineMd(INJECT)
    expect(html).not.toContain('onmouseover=')
    expect(html).not.toMatch(/\son[a-z]+\s*=/i)
    expect(html).toContain('<a href="#" target="_blank" rel="noopener noreferrer">点我</a>')
    const a = firstAnchor(html)
    expect(a.getAttribute('href')).toBe('#')
    expect(a.getAttribute('onmouseover')).toBeNull()
  })

  it('href 属性边界不被引号破坏（DOM 上没有多出来的属性）', () => {
    const html = inlineMd(INJECT)
    const quotes = html.match(/"/g) || []
    expect(quotes.length % 2).toBe(0)
    const names = [...firstAnchor(html).attributes].map((x) => x.name)
    expect(names).toEqual(['href', 'target', 'rel'])
  })

  it('链接文本里的引号同样被转义', () => {
    const html = inlineMd('[a"b](https://example.com/x)')
    expect(html).toContain('a&quot;b')
    expect(firstAnchor(html).getAttribute('href')).toBe('https://example.com/x')
  })
})

describe('newsMarkdown · A2 正常链接仍然可用', () => {
  it('带查询串的 https 链接指向原 URL', () => {
    const html = inlineMd('[a](https://example.com/p?q=1&r=2)')
    expect(html).toContain('href="https://example.com/p?q=1&amp;r=2"')
    const a = firstAnchor(html)
    expect(a.getAttribute('href')).toBe('https://example.com/p?q=1&r=2')
    expect(a.getAttribute('target')).toBe('_blank')
    expect(a.getAttribute('rel')).toBe('noopener noreferrer')
  })

  it('mailto 链接保留', () => {
    const a = firstAnchor(inlineMd('[写信](mailto:someone@example.com)'))
    expect(a.getAttribute('href')).toBe('mailto:someone@example.com')
  })
})

describe('newsMarkdown · A3 协议白名单仍然生效', () => {
  it.each([
    '[a](javascript:alert(1))',
    '[a](data:text/html,x)',
    '[a](vbscript:msgbox(1))',
    '[a](JAVASCRIPT:alert(1))',
    '[a](ftp://example.com/f)'
  ])('%s 降级为 href="#"', (input) => {
    const html = inlineMd(input)
    expect(html).toContain('href="#"')
    expect(firstAnchor(html).getAttribute('href')).toBe('#')
  })
})

describe('newsMarkdown · A4 既有渲染效果不变', () => {
  it('加粗 / 斜体 / 行内代码', () => {
    expect(inlineMd('**粗**')).toBe('<strong>粗</strong>')
    expect(inlineMd('*斜*')).toBe('<em>斜</em>')
    expect(inlineMd('`码`')).toBe('<code>码</code>')
  })

  it('链接文本内的加粗照旧生效', () => {
    expect(inlineMd('[**粗**](https://example.com)')).toBe(
      '<a href="https://example.com" target="_blank" rel="noopener noreferrer"><strong>粗</strong></a>'
    )
  })

  it('块级渲染：标题 / 列表 / 段落', () => {
    expect(renderMarkdown('# 标题')).toContain('<h1 class="md-h md-h1">标题</h1>')
    expect(renderMarkdown('- 一\n- 二')).toContain('<li>一</li>')
    expect(renderMarkdown('正文')).toBe('<p>正文</p>')
  })

  it('块级渲染里的链接同样受白名单保护', () => {
    const html = renderMarkdown('- [点我](javascript:alert(1))')
    expect(html).toContain('href="#"')
  })
})

describe('newsMarkdown · A5 普通文本里的引号不破坏属性边界', () => {
  it('双引号 / 单引号被转义成实体', () => {
    const html = inlineMd('他说"今天"很好 \'行\'')
    expect(html).toContain('&quot;今天&quot;')
    expect(html).toContain('&#39;行&#39;')
    expect(html).not.toContain('"今天"')
  })

  it('文本里的引号 + 链接混排时链接依然完好', () => {
    const html = inlineMd('"引用"[a](https://example.com/x)')
    const a = firstAnchor(html)
    expect(a.getAttribute('href')).toBe('https://example.com/x')
    expect(html).toContain('&quot;引用&quot;')
  })
})

// ── A6 反事实：把缺陷版本（0503494）的实现原样搬过来，证明上面的断言可证伪 ──
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

describe('newsMarkdown · A6 反事实（同一断言在缺陷版本上必须失败）', () => {
  it('缺陷版本确实渲染出 onmouseover 事件属性', () => {
    const legacy = legacyInlineMd(INJECT)
    expect(legacy).toContain('onmouseover=')
    expect(firstAnchor(legacy).getAttribute('onmouseover')).toBe('alert(1')
  })

  it('修复后同一载荷不再命中', () => {
    expect(inlineMd(INJECT)).not.toContain('onmouseover=')
    expect(firstAnchor(inlineMd(INJECT)).getAttribute('onmouseover')).toBeNull()
  })
})
