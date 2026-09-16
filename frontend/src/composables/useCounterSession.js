import { computed, ref } from 'vue'

/**
 * useCounterSession —— 运动会话（开始/结束时间、时长、按次目标、跨零点、敲击时间轴）。
 *
 * ── 设计红线（方案 §C / 硬约束）──────────────────────────────────────────────
 * 1. **独立存储键**：会话数据只进 `counter_v4_training`，绝不挂到 `counter_v4_records`
 *    的日记录上（`normalizeRecords()` 会重建记录并丢弃未知字段）。
 * 2. **计数不叠加**：`session.count` 只是 `todayCount` 增量的投影，不写 `dailyRecords`、
 *    不进 `grandTotal` / `streak` / `weekData` / `calCells`。
 * 3. **会话目标独立于每日目标**：本模块**不得**读取「每日目标」字段（L2 #43 负向 grep
 *    按该字段名断言本文件无命中），默认值恒为 `0`（自由训练）。
 * 4. **时间轴数据源是 `active.taps`**，不是组件里那个 15s 滑动窗口事件数组
 *    （L2 #44 负向 grep 按后者的旧字段名断言本文件无命中）。
 * 5. `taps` 写入点 = `applyDelta()` 里 `actualDelta !== 0` 的分支（由 `recordTap()` 落库），
 *    因此末条 `delta` 恒等于组件的 `lastActionDelta`，撤销 = 精确弹出末条。
 *
 * 本模块 0 网络请求，全部状态在 localStorage（客户端能算的一律不联网，R2）。
 */

/** 唯一新增的存储键；旧代码从不读写它 */
export const TRAINING_STORAGE_KEY = 'counter_v4_training'
export const TRAINING_STORAGE_VERSION = 1

/** 后台静默超过该时长即自动收尾（回前台两阶段顺序里的 (a) 步） */
export const SESSION_IDLE_MS = 30 * 60 * 1000

/** 会话级敲击日志上限（FIFO，超出丢弃最旧的） */
export const MAX_SESSION_TAPS = 500

/** 会话历史保留条数 */
export const MAX_SESSIONS = 200

/** 只有最近这么多条会话保留 `taps`（时间轴可回溯修正），更早的只留统计值 —— 配额保护 */
export const MAX_TAPS_SESSIONS = 20

export const DEFAULT_PREFS = Object.freeze({
  trainingMode: false, // 全屏训练层开关（会话结束自动关）
  goalMode: 'count', // "count" | "duration"
  goalValue: 0, // 0 = 本次不设目标（自由训练）—— 独立按次目标，绝不由每日目标派生
  lastGoalValue: 0, // 上一次会话用过的目标，仅用于「上次：200」提示
  voiceCue: true, // 训练中是否语音/音效节拍提示
  lockExitLongPressMs: 700,
  wakeLockEnabled: true // 训练时保持屏幕常亮；不支持时静默降级
})

function defaultStorage() {
  try {
    return typeof localStorage === 'undefined' ? null : localStorage
  } catch (_) {
    return null
  }
}

function formatLocalDateKey(ms) {
  const date = new Date(ms)
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${date.getFullYear()}-${month}-${day}`
}

function startOfLocalDay(ms) {
  const date = new Date(ms)
  date.setHours(0, 0, 0, 0)
  return date.getTime()
}

function toNonNegativeInt(value, fallback = 0) {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return fallback
  return Math.max(0, Math.floor(numeric))
}

function clampInt(value, min, max, fallback) {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return fallback
  return Math.min(max, Math.max(min, Math.round(numeric)))
}

function defaultIdFactory(atMs) {
  const suffix = Math.random().toString(16).slice(2, 6)
  return `s-${atMs}-${suffix}`
}

/** 达标判定：`value === 0`（自由训练）恒为 false */
export function isSessionReached(goal, count, activeMs) {
  const value = Number(goal?.value) || 0
  if (value <= 0) return false
  if (goal?.mode === 'duration') return Number(activeMs) >= value * 60000
  return Number(count) >= value
}

/**
 * @param {{
 *   now?: () => number,
 *   storage?: Storage|null,
 *   storageKey?: string,
 *   idleMs?: number,
 *   maxTaps?: number,
 *   maxSessions?: number,
 *   maxTapsSessions?: number,
 *   idFactory?: (atMs: number) => string,
 *   dateKey?: (ms: number) => string,
 *   dayStart?: (ms: number) => number
 * }} options
 */
export function createCounterSession({
  now = () => Date.now(),
  storage = defaultStorage(),
  storageKey = TRAINING_STORAGE_KEY,
  idleMs = SESSION_IDLE_MS,
  maxTaps = MAX_SESSION_TAPS,
  maxSessions = MAX_SESSIONS,
  maxTapsSessions = MAX_TAPS_SESSIONS,
  idFactory = defaultIdFactory,
  dateKey = formatLocalDateKey,
  dayStart = startOfLocalDay
} = {}) {
  const prefs = ref({ ...DEFAULT_PREFS })
  const active = ref(null)
  const sessions = ref([])
  const lastHiddenAtMs = ref(null)
  const pendingSplitFrom = ref(null)

  const trainingMode = computed(() => prefs.value.trainingMode === true)
  const hasActive = computed(() => active.value !== null)

  // ───────────────────────── 归一化（任何异常输入都回退到合法结构） ─────────────────────────

  function normalizePrefs(raw) {
    const next = { ...DEFAULT_PREFS }
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return next

    if (typeof raw.trainingMode === 'boolean') next.trainingMode = raw.trainingMode
    if (raw.goalMode === 'count' || raw.goalMode === 'duration') next.goalMode = raw.goalMode
    if (typeof raw.voiceCue === 'boolean') next.voiceCue = raw.voiceCue
    if (typeof raw.wakeLockEnabled === 'boolean') next.wakeLockEnabled = raw.wakeLockEnabled
    next.goalValue = toNonNegativeInt(raw.goalValue, DEFAULT_PREFS.goalValue)
    next.lastGoalValue = toNonNegativeInt(raw.lastGoalValue, DEFAULT_PREFS.lastGoalValue)
    next.lockExitLongPressMs = clampInt(
      raw.lockExitLongPressMs,
      200,
      3000,
      DEFAULT_PREFS.lockExitLongPressMs
    )
    return next
  }

  function normalizeGoal(raw) {
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
      return { mode: 'count', value: 0 }
    }
    return {
      mode: raw.mode === 'duration' ? 'duration' : 'count',
      value: toNonNegativeInt(raw.value, 0)
    }
  }

  function normalizeTaps(raw) {
    if (!Array.isArray(raw)) return []
    const list = []
    for (const entry of raw) {
      if (!entry || typeof entry !== 'object') continue
      const at = Number(entry.at)
      const delta = Number(entry.delta)
      if (!Number.isFinite(at) || !Number.isFinite(delta)) continue
      const kind = ['tap', 'minus', 'auto', 'undo'].includes(entry.kind) ? entry.kind : 'tap'
      list.push({ at, delta, kind })
    }
    return list.slice(-maxTaps)
  }

  function normalizeActive(raw) {
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null

    const startedAt = Number(raw.startedAt)
    if (!Number.isFinite(startedAt) || startedAt <= 0) return null

    const goal = normalizeGoal(raw.goal)
    const taps = normalizeTaps(raw.taps)

    return {
      id: typeof raw.id === 'string' && raw.id ? raw.id : idFactory(startedAt),
      startedAt,
      startDate: typeof raw.startDate === 'string' ? raw.startDate : dateKey(startedAt),
      goal,
      countAtStart: Math.max(0, Number(raw.countAtStart) || 0),
      activeMs: Math.max(0, Number(raw.activeMs) || 0),
      pausedMs: Math.max(0, Number(raw.pausedMs) || 0),
      backgroundMs: Math.max(0, Number(raw.backgroundMs) || 0),
      activeSinceMs: Number.isFinite(Number(raw.activeSinceMs)) ? Number(raw.activeSinceMs) : null,
      pausedSinceMs: Number.isFinite(Number(raw.pausedSinceMs)) ? Number(raw.pausedSinceMs) : null,
      hiddenSinceMs: Number.isFinite(Number(raw.hiddenSinceMs)) ? Number(raw.hiddenSinceMs) : null,
      lastHitAtMs: Number.isFinite(Number(raw.lastHitAtMs)) ? Number(raw.lastHitAtMs) : null,
      speedPeak: Math.max(0, Number(raw.speedPeak) || 0),
      splitFrom: typeof raw.splitFrom === 'string' ? raw.splitFrom : null,
      endReason: null,
      taps,
      tapsTotal: Math.max(0, Number(raw.tapsTotal) || taps.length),
      tapsDropped: Math.max(0, Number(raw.tapsDropped) || 0)
    }
  }

  function normalizeSession(raw) {
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null

    const startedAt = Number(raw.startedAt)
    const endedAt = Number(raw.endedAt)
    if (!Number.isFinite(startedAt) || !Number.isFinite(endedAt)) return null

    const safeStartedAt = Math.max(0, startedAt)
    const safeEndedAt = Math.max(safeStartedAt, endedAt)
    const count = toNonNegativeInt(raw.count, 0)
    const goal = normalizeGoal(raw.goal)
    const activeMs = toNonNegativeInt(raw.activeMs, 0)
    const readyReasons = ['manual', 'idle', 'midnight', 'reset']
    const taps = normalizeTaps(raw.taps)

    return {
      id: typeof raw.id === 'string' && raw.id ? raw.id : idFactory(safeStartedAt),
      startedAt: safeStartedAt,
      endedAt: safeEndedAt,
      startDate: typeof raw.startDate === 'string' ? raw.startDate : dateKey(safeStartedAt),
      count,
      durationMs: safeEndedAt - safeStartedAt,
      activeMs,
      pausedMs: toNonNegativeInt(raw.pausedMs, 0),
      backgroundMs: toNonNegativeInt(raw.backgroundMs, 0),
      goal,
      reached: typeof raw.reached === 'boolean'
        ? raw.reached
        : isSessionReached(goal, count, activeMs),
      speedPeak: toNonNegativeInt(raw.speedPeak, 0),
      endReason: readyReasons.includes(raw.endReason) ? raw.endReason : 'manual',
      splitFrom: typeof raw.splitFrom === 'string' ? raw.splitFrom : null,
      tapsTotal: toNonNegativeInt(raw.tapsTotal, taps.length),
      tapsDropped: toNonNegativeInt(raw.tapsDropped, 0),
      taps
    }
  }

  function normalizeSessions(raw) {
    if (!Array.isArray(raw)) return []
    const list = raw.map(normalizeSession).filter(Boolean)
    list.sort((left, right) => right.startedAt - left.startedAt)
    return list.slice(0, maxSessions)
  }

  // ───────────────────────── 持久化 ─────────────────────────

  /** `active.taps` 不落盘（内存态，见 §C1.7 落盘策略） */
  function stripActiveTaps(session) {
    const { taps, ...rest } = session
    return rest
  }

  function serializeSession(session, index) {
    if (index < maxTapsSessions) return session
    const { taps, ...rest } = session
    return rest
  }

  function buildPayload() {
    return {
      version: TRAINING_STORAGE_VERSION,
      prefs: { ...prefs.value },
      active: active.value ? stripActiveTaps(active.value) : null,
      sessions: sessions.value.map(serializeSession),
      lastHiddenAtMs: lastHiddenAtMs.value,
      pendingSplitFrom: pendingSplitFrom.value
    }
  }

  function writeRaw(payload) {
    if (!storage) return false
    try {
      storage.setItem(storageKey, JSON.stringify(payload))
      return true
    } catch (_) {
      return false
    }
  }

  /** 单键整体写；配额异常 → 丢弃最旧 50 条会话重试一次 → 仍失败静默放弃 */
  function flush() {
    const payload = buildPayload()
    if (writeRaw(payload)) return true

    const trimmed = {
      ...payload,
      sessions: payload.sessions.slice(0, Math.max(0, payload.sessions.length - 50))
    }
    if (writeRaw(trimmed)) {
      sessions.value = sessions.value.slice(0, trimmed.sessions.length)
      return true
    }
    return false
  }

  function init() {
    let parsed = null
    try {
      const raw = storage ? storage.getItem(storageKey) : null
      parsed = raw ? JSON.parse(raw) : null
    } catch (_) {
      parsed = null
    }

    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      parsed = {}
    }

    prefs.value = normalizePrefs(parsed.prefs)
    active.value = normalizeActive(parsed.active)
    sessions.value = normalizeSessions(parsed.sessions)
    lastHiddenAtMs.value = Number.isFinite(Number(parsed.lastHiddenAtMs))
      ? Number(parsed.lastHiddenAtMs)
      : null
    pendingSplitFrom.value = typeof parsed.pendingSplitFrom === 'string'
      ? parsed.pendingSplitFrom
      : null

    return snapshot()
  }

  function snapshot() {
    return {
      prefs: { ...prefs.value },
      active: active.value ? { ...active.value } : null,
      sessions: sessions.value.map((item) => ({ ...item })),
      lastHiddenAtMs: lastHiddenAtMs.value,
      pendingSplitFrom: pendingSplitFrom.value
    }
  }

  // ───────────────────────── 会话生命周期 ─────────────────────────

  function settleActiveSegment(atMs) {
    const current = active.value
    if (!current) return
    if (current.pausedSinceMs != null) {
      current.pausedMs += Math.max(0, atMs - current.pausedSinceMs)
      current.pausedSinceMs = null
      return
    }
    if (current.activeSinceMs != null) {
      current.activeMs += Math.max(0, atMs - current.activeSinceMs)
      current.activeSinceMs = null
    }
  }

  function createActive({ atMs, countAtStart, goal, splitFrom = null }) {
    active.value = {
      id: idFactory(atMs),
      startedAt: atMs,
      startDate: dateKey(atMs),
      goal: { ...goal },
      countAtStart,
      activeMs: 0,
      pausedMs: 0,
      backgroundMs: 0,
      activeSinceMs: atMs,
      pausedSinceMs: null,
      hiddenSinceMs: null,
      lastHitAtMs: null,
      speedPeak: 0,
      splitFrom,
      endReason: null,
      taps: [],
      tapsTotal: 0,
      tapsDropped: 0
    }
    return active.value
  }

  /** 开始一次新会话。`goalValue` 缺省 = 0（自由训练），绝不由每日目标派生 */
  function startSession({ todayCount = 0, goalValue = null, goalMode = null, at = now() } = {}) {
    if (active.value) return active.value

    const goal = {
      mode: goalMode || prefs.value.goalMode,
      value: goalValue == null ? 0 : toNonNegativeInt(goalValue, 0)
    }

    prefs.value = {
      ...prefs.value,
      trainingMode: true,
      goalMode: goal.mode,
      goalValue: goal.value,
      lastGoalValue: goal.value > 0 ? goal.value : prefs.value.lastGoalValue
    }

    createActive({
      atMs: at,
      countAtStart: toNonNegativeInt(todayCount, 0),
      goal,
      splitFrom: null
    })
    pendingSplitFrom.value = null
    flush()
    return active.value
  }

  function countOf(todayCount) {
    const current = active.value
    if (!current) return 0
    return Math.max(0, Math.round((Number(todayCount) || 0) - current.countAtStart))
  }

  function endActive(reason, todayCount, endedAtMs) {
    const current = active.value
    if (!current) return null

    const endMs = Math.max(current.startedAt, endedAtMs)
    settleActiveSegment(endMs)

    const count = Math.max(0, (Number(todayCount) || 0) - current.countAtStart)
    const goal = { ...current.goal }
    const activeMs = Math.round(current.activeMs)

    const finished = {
      id: current.id,
      startedAt: current.startedAt,
      endedAt: endMs,
      startDate: current.startDate,
      count,
      durationMs: endMs - current.startedAt,
      activeMs,
      pausedMs: Math.round(current.pausedMs),
      backgroundMs: Math.round(current.backgroundMs),
      goal,
      reached: isSessionReached(goal, count, activeMs),
      speedPeak: Math.round(current.speedPeak),
      endReason: reason,
      splitFrom: current.splitFrom,
      tapsTotal: current.tapsTotal,
      tapsDropped: current.tapsDropped,
      taps: current.taps.slice()
    }

    active.value = null
    sessions.value = [finished, ...sessions.value]
      .sort((left, right) => right.startedAt - left.startedAt)
      .slice(0, maxSessions)

    return finished
  }

  function endSession({ todayCount = 0, reason = 'manual', at = now() } = {}) {
    const finished = endActive(reason, todayCount, at)
    prefs.value = { ...prefs.value, trainingMode: false }
    flush()
    return finished
  }

  function setTrainingMode(flag) {
    prefs.value = { ...prefs.value, trainingMode: flag === true }
    flush()
  }

  function updatePrefs(patch) {
    if (!patch || typeof patch !== 'object') return prefs.value
    prefs.value = normalizePrefs({ ...prefs.value, ...patch })
    flush()
    return prefs.value
  }

  function pauseSession(at = now()) {
    const current = active.value
    if (!current || current.pausedSinceMs != null) return false
    settleActiveSegment(at)
    current.pausedSinceMs = at
    flush()
    return true
  }

  function resumeSession(at = now()) {
    const current = active.value
    if (!current || current.pausedSinceMs == null) return false
    current.pausedMs += Math.max(0, at - current.pausedSinceMs)
    current.pausedSinceMs = null
    current.activeSinceMs = at
    flush()
    return true
  }

  function isPaused() {
    return active.value != null && active.value.pausedSinceMs != null
  }

  // ───────────────────────── 后台 / 回前台 ─────────────────────────

  /** 切后台 / 路由离开：不结束会话，只结算当前活跃段并记下时间点 */
  function handleBackground(at = now()) {
    lastHiddenAtMs.value = at
    const current = active.value
    if (!current) {
      flush()
      return
    }
    settleActiveSegment(at)
    current.hiddenSinceMs = at
    flush()
  }

  function idleEndAtMs(at) {
    const hiddenSince = active.value?.hiddenSinceMs ?? lastHiddenAtMs.value ?? at
    const lastActivity = active.value?.lastHitAtMs ?? hiddenSince
    return Math.min(at, Math.max(lastActivity, hiddenSince))
  }

  /**
   * 回前台。**严格三步，顺序不得调换**：
   * (a) 先按 `idleMs` 结算 idle → (b) 再跑跨零点检查（由调用方注入 `runRollover`）→ (c) 最后消费 `pendingSplitFrom`
   *
   * 为什么必须先 (a)：`checkDayRollover` 的前台守卫只能看到「现在是 visible」，
   * 区分不了「跨零点时人在前台」与「跨零点后人才回前台」。先结算 idle 就不会产出假连续训练。
   */
  function handleForeground(at = now(), { todayCount = 0, runRollover = null } = {}) {
    const result = { away: null, idleEnded: false, resumed: false, continued: false }

    // (a) 结算 idle
    const hiddenAt = lastHiddenAtMs.value
    if (hiddenAt != null) {
      const away = Math.max(0, at - hiddenAt)
      result.away = away

      if (away > idleMs) {
        if (active.value) {
          endActive('idle', todayCount, idleEndAtMs(at))
          result.idleEnded = true
        }
        pendingSplitFrom.value = null
      } else if (active.value) {
        active.value.backgroundMs += away
        active.value.activeSinceMs = at
        active.value.hiddenSinceMs = null
        result.resumed = true
      }
    }

    // (b) 跨零点检查（组件注入，内部会调 handleRollover）
    if (typeof runRollover === 'function') runRollover()

    // (c) 消费 pendingSplitFrom
    if (pendingSplitFrom.value !== null) {
      const previousId = pendingSplitFrom.value
      pendingSplitFrom.value = null
      if (active.value === null && !result.idleEnded) {
        const previous = sessions.value.find((item) => item.id === previousId)
        createActive({
          atMs: dayStart(at),
          countAtStart: 0,
          goal: previous ? previous.goal : { mode: prefs.value.goalMode, value: 0 },
          splitFrom: previousId
        })
        result.continued = true
      }
    }

    flush()
    return result
  }

  /**
   * 跨零点拆分。调用方必须**先**取 todayCount 快照再清零（见 §C1.8）。
   * `autoContinue === false` 时只收尾、不续开，并记 `pendingSplitFrom` 等回前台补开。
   */
  function handleRollover(at = now(), { todayCount = 0, autoContinue = false } = {}) {
    const previous = active.value
    if (!previous) {
      pendingSplitFrom.value = null
      flush()
      return null
    }

    const dayBoundary = dayStart(at) - 1 // 当日 23:59:59.999
    const previousId = previous.id
    const previousGoal = { ...previous.goal }
    const finished = endActive('midnight', todayCount, dayBoundary)

    if (autoContinue) {
      // 新段的 countAtStart 恒为字面量 0（清零后），绝不是「取当前 todayCount」
      createActive({
        atMs: dayStart(at),
        countAtStart: 0,
        goal: previousGoal,
        splitFrom: previousId
      })
      pendingSplitFrom.value = null
    } else {
      pendingSplitFrom.value = previousId
    }

    flush()
    return finished
  }

  // ───────────────────────── 敲击日志 / 时间轴 ─────────────────────────

  /** 写入点：`applyDelta()` 里 `actualDelta !== 0` 的分支 */
  function recordTap({ at = now(), delta, kind = 'tap' } = {}) {
    const current = active.value
    const numericDelta = Number(delta)
    if (!current || !Number.isFinite(numericDelta) || numericDelta === 0) return null

    const entry = { at, delta: numericDelta, kind }
    current.taps.push(entry)
    current.tapsTotal += 1
    while (current.taps.length > maxTaps) {
      current.taps.shift()
      current.tapsDropped += 1
    }
    if (numericDelta > 0) {
      current.lastHitAtMs = at
    }
    return entry
  }

  /** 撤销上一次计数动作：弹出末条并返回它（组件据此反向调整 todayCount） */
  function popLastTap() {
    const current = active.value
    if (!current || current.taps.length === 0) return null
    return current.taps.pop() || null
  }

  /** 时间轴删除任意一条：返回被删条目（`delta` 可正可负，带符号扣减） */
  function removeTapAt(index) {
    const current = active.value
    if (!current) return null
    const numericIndex = Number(index)
    if (!Number.isInteger(numericIndex) || numericIndex < 0 || numericIndex >= current.taps.length) {
      return null
    }
    const [removed] = current.taps.splice(numericIndex, 1)
    return removed || null
  }

  /**
   * 已结束会话（`sessions[]`）的时间轴删除 —— 「结束后回溯修正单次敲击」用。
   * 与 `removeTapAt()` 的差别：这里的 `count` 是**冻结值**（不像 active 那样由 todayCount 派生），
   * 所以必须同步按 `delta` 反向修正 `count`（下限 0），否则总结面板的次数会与时间轴对不上。
   * 组件负责同步 `todayCount`（本模块不持有日统计，见红线 2）。
   */
  function removeSessionTap(sessionId, index) {
    const found = sessions.value.find((item) => item.id === sessionId)
    if (!found) return null
    const numericIndex = Number(index)
    if (!Number.isInteger(numericIndex) || numericIndex < 0 || numericIndex >= found.taps.length) {
      return null
    }
    const [removed] = found.taps.splice(numericIndex, 1)
    if (!removed) return null
    found.count = Math.max(0, found.count - removed.delta)
    flush()
    return removed
  }

  function noteSpeedPeak(value) {
    const current = active.value
    const numeric = Number(value)
    if (!current || !Number.isFinite(numeric)) return
    current.speedPeak = Math.max(current.speedPeak, numeric)
  }

  // ───────────────────────── 清理 ─────────────────────────

  function clearSessions() {
    sessions.value = []
    flush()
  }

  /** 「清除全部数据」：连 prefs 一起回到默认，并抹掉整个键 */
  function resetAll() {
    active.value = null
    sessions.value = []
    lastHiddenAtMs.value = null
    pendingSplitFrom.value = null
    prefs.value = { ...DEFAULT_PREFS }
    try {
      storage?.removeItem(storageKey)
    } catch (_) {
      // ignore
    }
  }

  /** 时间轴数据源：未结束的会话读 active.taps，已结束的读 sessions[i].taps */
  function tapsForSession(sessionId) {
    if (active.value && active.value.id === sessionId) {
      return { taps: active.value.taps, total: active.value.tapsTotal, dropped: active.value.tapsDropped }
    }
    const found = sessions.value.find((item) => item.id === sessionId)
    if (!found) return null
    return { taps: found.taps, total: found.tapsTotal, dropped: found.tapsDropped }
  }

  return {
    // state
    prefs,
    active,
    sessions,
    lastHiddenAtMs,
    pendingSplitFrom,
    trainingMode,
    hasActive,
    // 常量（供 UI 展示上限说明）
    idleMs,
    maxTaps,
    maxSessions,
    // lifecycle
    init,
    flush,
    snapshot,
    startSession,
    endSession,
    setTrainingMode,
    updatePrefs,
    pauseSession,
    resumeSession,
    isPaused,
    handleBackground,
    handleForeground,
    handleRollover,
    // taps
    recordTap,
    popLastTap,
    removeTapAt,
    removeSessionTap,
    noteSpeedPeak,
    countOf,
    tapsForSession,
    // cleanup
    clearSessions,
    resetAll
  }
}
