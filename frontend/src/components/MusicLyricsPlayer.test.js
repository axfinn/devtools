import { describe, it, expect } from 'vitest'
import {
  parseLyrics,
  detectMode,
  findActiveIndex,
  computeSeekTime,
  formatLyricsTime,
} from './MusicLyricsPlayer.vue'

describe('parseLyrics', () => {
  it('parses a single LRC line with seconds-only timestamp', () => {
    const out = parseLyrics('[00:12]第一行歌词')
    expect(out).toEqual([{ time: 12, text: '第一行歌词' }])
  })

  it('parses LRC with millisecond precision', () => {
    const out = parseLyrics('[01:23.456]细颗粒时间')
    expect(out).toEqual([{ time: 83.456, text: '细颗粒时间' }])
  })

  it('parses multi-line LRC, sorted by time', () => {
    const lrc = `[00:30]第三行
[00:10]第一行
[00:20]第二行`
    const out = parseLyrics(lrc)
    expect(out.map(l => l.text)).toEqual(['第一行', '第二行', '第三行'])
    expect(out.map(l => l.time)).toEqual([10, 20, 30])
  })

  it('expands multiple timestamps on one line into multiple entries', () => {
    const lrc = '[00:10][00:20][00:30]副歌'
    const out = parseLyrics(lrc)
    expect(out).toEqual([
      { time: 10, text: '副歌' },
      { time: 20, text: '副歌' },
      { time: 30, text: '副歌' },
    ])
  })

  it('parses plain text without timestamps', () => {
    const text = `第一行
第二行
第三行`
    const out = parseLyrics(text)
    expect(out).toEqual([
      { time: null, text: '第一行' },
      { time: null, text: '第二行' },
      { time: null, text: '第三行' },
    ])
  })

  it('splits a single-line Chinese paragraph by sentence-ending punctuation', () => {
    const text = '春风拂过城市的街角。我们抬头看着星空。却想起远方的你。'
    const out = parseLyrics(text)
    expect(out.length).toBe(3)
    expect(out[0].text).toMatch(/春风/)
    expect(out.every(l => l.time === null)).toBe(true)
  })

  it('drops LRC metadata lines like [ti:title]', () => {
    const lrc = `[ti:夏日情歌]
[ar:Unknown]
[00:10]开始
[00:20]继续`
    const out = parseLyrics(lrc)
    expect(out).toHaveLength(2)
    expect(out[0]).toEqual({ time: 10, text: '开始' })
    expect(out[1]).toEqual({ time: 20, text: '继续' })
  })

  it('skips empty lines', () => {
    const text = `[00:10]第一行

[00:20]第二行

`
    const out = parseLyrics(text)
    expect(out).toHaveLength(2)
  })

  it('returns [] for empty / non-string input', () => {
    expect(parseLyrics('')).toEqual([])
    expect(parseLyrics(null)).toEqual([])
    expect(parseLyrics(undefined)).toEqual([])
    expect(parseLyrics(123)).toEqual([])
  })

  it('handles mixed LRC + plain-text lines, keeps order, sorts by time where possible', () => {
    const text = `[00:10]有时间的
纯文本前导
[00:05]最早时间
纯文本结尾`
    const out = parseLyrics(text)
    // 4 行,纯文本 time=null,排序时排到末尾
    expect(out).toHaveLength(4)
    const textsInOrder = out.map(l => l.text)
    // LRC 行会按 time 升序排在前面,纯文本行 time=null 排末尾
    expect(textsInOrder.slice(0, 2)).toEqual(['最早时间', '有时间的'])
  })

  it('handles windows-style \\r\\n line endings', () => {
    const text = '[00:10]第一\r\n[00:20]第二'
    const out = parseLyrics(text)
    expect(out).toHaveLength(2)
  })
})

describe('detectMode', () => {
  it('returns "lrc" when >= 30% lines have timestamps', () => {
    const lines = [
      { time: 0, text: 'a' },
      { time: 5, text: 'b' },
      { time: 10, text: 'c' },
      { time: null, text: 'd' },
    ]
    expect(detectMode(lines)).toBe('lrc')
  })

  it('returns "linear" when < 30% lines have timestamps', () => {
    const lines = [
      { time: 0, text: 'a' },
      { time: null, text: 'b' },
      { time: null, text: 'c' },
      { time: null, text: 'd' },
    ]
    expect(detectMode(lines)).toBe('linear')
  })

  it('returns "linear" for empty input', () => {
    expect(detectMode([])).toBe('linear')
  })
})

describe('findActiveIndex', () => {
  const lrc = [
    { time: 0, text: 'a' },
    { time: 5, text: 'b' },
    { time: 10, text: 'c' },
    { time: 15, text: 'd' },
  ]

  it('LRC mode: returns last line whose time <= currentTime', () => {
    expect(findActiveIndex(lrc, 0, 20, 'lrc')).toBe(0)
    expect(findActiveIndex(lrc, 3, 20, 'lrc')).toBe(0)
    expect(findActiveIndex(lrc, 5, 20, 'lrc')).toBe(1)
    expect(findActiveIndex(lrc, 7, 20, 'lrc')).toBe(1)
    expect(findActiveIndex(lrc, 12, 20, 'lrc')).toBe(2)
    expect(findActiveIndex(lrc, 16, 20, 'lrc')).toBe(3)
  })

  it('LRC mode: returns -1 when before first timestamp', () => {
    const later = [{ time: 10, text: 'late' }]
    expect(findActiveIndex(later, 5, 30, 'lrc')).toBe(-1)
  })

  it('LRC mode: stays on last line after the last timestamp', () => {
    expect(findActiveIndex(lrc, 100, 20, 'lrc')).toBe(3)
  })

  it('linear mode: scales by currentTime / duration', () => {
    const lines = Array.from({ length: 10 }, (_, i) => ({ time: null, text: `l${i}` }))
    expect(findActiveIndex(lines, 0, 100, 'linear')).toBe(0)
    expect(findActiveIndex(lines, 25, 100, 'linear')).toBe(2)
    expect(findActiveIndex(lines, 50, 100, 'linear')).toBe(5)
    expect(findActiveIndex(lines, 75, 100, 'linear')).toBe(7)
    expect(findActiveIndex(lines, 99, 100, 'linear')).toBe(9)
    expect(findActiveIndex(lines, 100, 100, 'linear')).toBe(9)
    expect(findActiveIndex(lines, 200, 100, 'linear')).toBe(9)
  })

  it('linear mode: returns 0 when duration is 0 or invalid', () => {
    const lines = [{ time: null, text: 'a' }, { time: null, text: 'b' }]
    expect(findActiveIndex(lines, 5, 0, 'linear')).toBe(0)
    expect(findActiveIndex(lines, 5, null, 'linear')).toBe(0)
  })

  it('returns -1 for empty lines', () => {
    expect(findActiveIndex([], 5, 10, 'lrc')).toBe(-1)
    expect(findActiveIndex([], 5, 10, 'linear')).toBe(-1)
  })
})

describe('computeSeekTime', () => {
  const lrc = [
    { time: 10, text: 'a' },
    { time: 20, text: 'b' },
  ]

  it('LRC mode: returns the real timestamp', () => {
    expect(computeSeekTime(lrc[0], 0, 2, 100, 'lrc')).toBe(10)
    expect(computeSeekTime(lrc[1], 1, 2, 100, 'lrc')).toBe(20)
  })

  it('linear mode: evenly distributes by index / total * duration', () => {
    const lines = Array.from({ length: 4 }, (_, i) => ({ time: null, text: `l${i}` }))
    expect(computeSeekTime(lines[0], 0, 4, 100, 'linear')).toBe(0)
    expect(computeSeekTime(lines[1], 1, 4, 100, 'linear')).toBe(25)
    expect(computeSeekTime(lines[2], 2, 4, 100, 'linear')).toBe(50)
    expect(computeSeekTime(lines[3], 3, 4, 100, 'linear')).toBe(75)
  })

  it('returns 0 when no time available and no duration', () => {
    expect(computeSeekTime({ time: null, text: 'x' }, 0, 1, 0, 'linear')).toBe(0)
  })
})

describe('formatLyricsTime', () => {
  it('formats seconds to mm:ss', () => {
    expect(formatLyricsTime(0)).toBe('0:00')
    expect(formatLyricsTime(7)).toBe('0:07')
    expect(formatLyricsTime(65)).toBe('1:05')
    expect(formatLyricsTime(125)).toBe('2:05')
    expect(formatLyricsTime(600)).toBe('10:00')
  })

  it('handles invalid / negative / non-finite', () => {
    expect(formatLyricsTime(NaN)).toBe('0:00')
    expect(formatLyricsTime(-1)).toBe('0:00')
    expect(formatLyricsTime(null)).toBe('0:00')
    expect(formatLyricsTime(undefined)).toBe('0:00')
  })
})
