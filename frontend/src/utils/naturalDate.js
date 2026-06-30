// 自然语言日期/时间解析
// 返回 { date: 'YYYY-MM-DD', time: 'HH:mm'(可空) } 或 null
//
// 支持的格式(可任意组合):
//   日期: 今天 / 明天 / 后天 / 大后天 / 周X(默认下一个) / 下周X
//         X天后 / X周后 / X月后 / X月X日 / X/X / X-X / XXXX-X-X
//   时间: 早X / 上午X / 下午X / 晚上X(自动 12/24h 转换)
//         X:XX / X点 / X点半 / X点X分
//   组合: "明天下午3点" / "后天早9:30" / "下周一上午10点"

export function parseNaturalDate(raw, now) {
  if (raw === null || raw === undefined) return null
  const baseNow = now instanceof Date ? new Date(now) : new Date()
  if (isNaN(baseNow.getTime())) return null
  let s = String(raw).trim().replace(/\s+/g, '')
  if (!s) return null

  let hour = null
  let minute = 0
  let timeMatched = false

  // === TIME ===
  // 24h HH:MM
  let m = s.match(/(\d{1,2}):(\d{2})/)
  if (m) {
    const h = parseInt(m[1], 10)
    const mn = parseInt(m[2], 10)
    if (h >= 0 && h <= 23 && mn >= 0 && mn < 60) {
      hour = h
      minute = mn
      timeMatched = true
    }
  }
  // 早/上午 X[:点][:分]
  if (!timeMatched) {
    m = s.match(/(?:^|[^周])(早|上午|am)[:：]?(\d{1,2})[:点]?(\d{0,2})/)
    if (m) {
      const h = parseInt(m[2], 10)
      const mn = m[3] ? parseInt(m[3], 10) : 0
      if (h >= 0 && h <= 12 && mn >= 0 && mn < 60) {
        hour = h === 12 ? 0 : h // 上午12点 = 0点
        minute = mn
        timeMatched = true
      }
    }
  }
  // 下午/晚上/傍晚/夜里 X[:点][:分]
  if (!timeMatched) {
    m = s.match(/(下午|午后|晚上|夜里|夜间|傍晚|pm)[:：]?(\d{1,2})[:点]?(\d{0,2})/)
    if (m) {
      let h = parseInt(m[2], 10)
      if (h < 12) h += 12
      const mn = m[3] ? parseInt(m[3], 10) : 0
      if (h >= 0 && h <= 23 && mn >= 0 && mn < 60) {
        hour = h
        minute = mn
        timeMatched = true
      }
    }
  }
  // X点X分 / X点 / X点半
  if (!timeMatched) {
    m = s.match(/(\d{1,2})点(?:半|(\d{1,2})分?)?/)
    if (m) {
      const h = parseInt(m[1], 10)
      const mn = m[2] ? parseInt(m[2], 10) : (s.includes('点半') ? 30 : 0)
      if (h >= 0 && h <= 23 && mn >= 0 && mn < 60) {
        hour = h
        minute = mn
        timeMatched = true
      }
    }
  }

  // === DATE ===
  let year = baseNow.getFullYear()
  let month = null
  let day = null
  let dateMatched = false

  // 绝对: YYYY-M-D / YYYY/M/D / YYYY.M.D
  m = s.match(/(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})/)
  if (m) {
    year = parseInt(m[1], 10)
    month = parseInt(m[2], 10) - 1
    day = parseInt(m[3], 10)
    dateMatched = true
  }
  // M月D[日号]
  if (!dateMatched) {
    m = s.match(/(\d{1,2})月(\d{1,2})[日号]?/)
    if (m) {
      month = parseInt(m[1], 10) - 1
      day = parseInt(m[2], 10)
      dateMatched = true
    }
  }
  // M/D / M-D
  if (!dateMatched) {
    m = s.match(/(\d{1,2})[\/\-](\d{1,2})/)
    if (m) {
      month = parseInt(m[1], 10) - 1
      day = parseInt(m[2], 10)
      dateMatched = true
    }
  }

  // 相对日期
  const today = new Date(baseNow)
  today.setHours(0, 0, 0, 0)
  let target = null

  const dayMap = { '一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '日': 0, '天': 0 }

  if (!dateMatched) {
    if (s.includes('今天')) {
      target = new Date(today)
    } else if (s.includes('大后天')) {
      target = new Date(today)
      target.setDate(target.getDate() + 3)
    } else if (s.includes('后天')) {
      target = new Date(today)
      target.setDate(target.getDate() + 2)
    } else if (s.includes('明天') || s.includes('明日') || s.includes('明早') || s.includes('明晚')) {
      target = new Date(today)
      target.setDate(target.getDate() + 1)
    } else if (s.includes('下周') || s.includes('下星期')) {
      m = s.match(/下周[一星]?([一二三四五六日天])/)
      if (m && dayMap[m[1]] !== undefined) {
        target = new Date(today)
        const cur = target.getDay()
        let add = (dayMap[m[1]] - cur + 7) % 7
        if (add === 0) add = 7
        target.setDate(target.getDate() + add)
      }
    } else if (s.match(/周[一二三四五六日天]/)) {
      m = s.match(/周([一二三四五六日天])/)
      if (m && dayMap[m[1]] !== undefined) {
        target = new Date(today)
        const cur = target.getDay()
        let add = (dayMap[m[1]] - cur + 7) % 7
        if (add === 0) add = 7
        target.setDate(target.getDate() + add)
      }
    } else {
      m = s.match(/(\d+)\s*天[之以]?后/)
      if (m) {
        target = new Date(today)
        target.setDate(target.getDate() + parseInt(m[1], 10))
      } else {
        m = s.match(/(\d+)\s*周[之以]?后/)
        if (m) {
          target = new Date(today)
          target.setDate(target.getDate() + parseInt(m[1], 10) * 7)
        } else {
          m = s.match(/(\d+)\s*个?月[之以]?后/)
          if (m) {
            target = new Date(today)
            target.setMonth(target.getMonth() + parseInt(m[1], 10))
          }
        }
      }
    }

    if (target) {
      year = target.getFullYear()
      month = target.getMonth()
      day = target.getDate()
      dateMatched = true
    }
  }

  if (!dateMatched) return null
  if (month === null || day === null) return null

  // 边界校验(2月30日之类)
  const date = new Date(year, month, day)
  if (isNaN(date.getTime())) return null
  if (date.getFullYear() !== year || date.getMonth() !== month || date.getDate() !== day) return null

  const dateStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`
  const timeStr = timeMatched
    ? `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`
    : ''
  return { date: dateStr, time: timeStr }
}
