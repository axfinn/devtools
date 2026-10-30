import { reactive, computed } from 'vue'

// ── Singleton reactive state (survives route navigation) ──
function initialVolume() {
  try {
    const v = parseFloat(localStorage.getItem('media_player_volume'))
    if (isFinite(v) && v >= 0 && v <= 1) return v
  } catch {}
  return 1
}

const state = reactive({
  playlist: [],        // all playable items: { id, title, result_type, assetUrl, kind }
  currentIndex: -1,
  isPlaying: false,
  loopMode: 'all',     // 'off' | 'one' | 'all'
  shuffleMode: false,
  currentTime: 0,
  duration: 0,
  volume: initialVolume(),
})

let audioEl = null

// Derive currently playing item
function currentTrack() {
  if (state.currentIndex < 0 || state.currentIndex >= state.playlist.length) return null
  return state.playlist[state.currentIndex]
}

function buildPlaylist(items) {
  const list = []
  for (const item of items) {
    const asset = item.primaryAsset || item._assets?.[0]
    if (!asset?.asset_url || (asset.kind !== 'audio' && asset.kind !== 'video')) continue
    list.push({
      id: item.id,
      title: item.title || '未命名',
      result_type: asset.kind,
      assetUrl: asset.asset_url,
      kind: asset.kind,
    })
  }
  return list
}

function onAudioError() { skipNext() }
function onLoadedMetadata() { state.duration = audioEl?.duration || 0 }
function onTimeUpdate() { state.currentTime = audioEl?.currentTime || 0 }

function createAudio(url) {
  if (audioEl) {
    audioEl.pause()
    audioEl.removeEventListener('ended', onTrackEnded)
    audioEl.removeEventListener('error', onAudioError)
    audioEl.removeEventListener('loadedmetadata', onLoadedMetadata)
    audioEl.removeEventListener('timeupdate', onTimeUpdate)
    audioEl.src = ''
  }
  audioEl = new Audio(url)
  audioEl.volume = state.volume
  audioEl.addEventListener('ended', onTrackEnded)
  audioEl.addEventListener('error', onAudioError)
  audioEl.addEventListener('loadedmetadata', onLoadedMetadata)
  audioEl.addEventListener('timeupdate', onTimeUpdate)
}

function setVolume(v) {
  const next = Math.min(1, Math.max(0, Number(v) || 0))
  state.volume = next
  if (audioEl) audioEl.volume = next
  try { localStorage.setItem('media_player_volume', String(next)) } catch {}
}

function onTrackEnded() {
  if (state.loopMode === 'one') {
    if (audioEl) {
      audioEl.currentTime = 0
      audioEl.play().catch(() => {})
    }
  } else {
    skipNext()
  }
}

function skipNext() {
  if (state.playlist.length === 0) return
  let next
  if (state.shuffleMode) {
    next = Math.floor(Math.random() * state.playlist.length)
  } else if (state.loopMode === 'all' || state.currentIndex < state.playlist.length - 1) {
    next = (state.currentIndex + 1) % state.playlist.length
  } else {
    // loopMode === 'off' and at the end — stop
    stop()
    return
  }
  playIndex(next)
}

function skipPrev() {
  if (state.playlist.length === 0) return
  const prev = state.currentIndex <= 0 ? state.playlist.length - 1 : state.currentIndex - 1
  playIndex(prev)
}

function playIndex(idx) {
  if (idx < 0 || idx >= state.playlist.length) return
  state.currentIndex = idx
  const track = state.playlist[idx]
  if (!track) return

  if (track.kind === 'audio') {
    createAudio(track.assetUrl)
    audioEl.play().then(() => {
      state.isPlaying = true
    }).catch(() => {
      state.isPlaying = false
    })
    // 必须在这里 return:下面那句无条件 `state.isPlaying = true` 会把 catch 里
    // 设的 false 覆盖掉,导致播放被浏览器拒绝时 UI 仍显示"播放中"。
    return
  }
  // Video playback is managed in-component since it needs a DOM element
  state.isPlaying = true
}

function play(itemOrIndex) {
  if (typeof itemOrIndex === 'number') {
    playIndex(itemOrIndex)
    return
  }
  // Find item in playlist
  const idx = state.playlist.findIndex(t => t.id === itemOrIndex.id)
  if (idx >= 0) {
    playIndex(idx)
  } else {
    // Add single item to playlist and play
    const track = itemOrIndex.primaryAsset || itemOrIndex._assets?.[0]
    if (!track?.asset_url) return
    state.playlist = [{
      id: itemOrIndex.id,
      title: itemOrIndex.title || '未命名',
      result_type: track.kind,
      assetUrl: track.asset_url,
      kind: track.kind,
    }]
    playIndex(0)
  }
}

function togglePlay() {
  if (!audioEl) return
  if (state.isPlaying) {
    audioEl.pause()
    state.isPlaying = false
  } else {
    audioEl.play().then(() => {
      state.isPlaying = true
    }).catch(() => {})
  }
}

function stop() {
  if (audioEl) {
    audioEl.pause()
    audioEl.removeEventListener('ended', onTrackEnded)
    audioEl.removeEventListener('error', onAudioError)
    audioEl.removeEventListener('loadedmetadata', onLoadedMetadata)
    audioEl.removeEventListener('timeupdate', onTimeUpdate)
    audioEl.src = ''
    audioEl = null
  }
  state.isPlaying = false
  state.currentIndex = -1
  state.currentTime = 0
  state.duration = 0
  state.playlist = []
}

function setLoopMode(mode) {
  state.loopMode = mode
}

function toggleShuffle() {
  state.shuffleMode = !state.shuffleMode
}

function seek(time) {
  if (audioEl) {
    audioEl.currentTime = time
  }
}

function setPlaylist(items) {
  state.playlist = buildPlaylist(items)
}

function addToPlaylist(item) {
  const track = item.primaryAsset || item._assets?.[0]
  if (!track?.asset_url) return
  const exists = state.playlist.find(t => t.id === item.id)
  if (!exists) {
    state.playlist.push({
      id: item.id,
      title: item.title || '未命名',
      result_type: track.kind,
      assetUrl: track.asset_url,
      kind: track.kind,
    })
  }
}

function playAll(items, startIndex = 0) {
  state.playlist = buildPlaylist(items)
  if (state.playlist.length > 0) {
    playIndex(Math.min(startIndex, state.playlist.length - 1))
  }
}

// Format seconds to mm:ss
function fmtTime(s) {
  if (!s || !isFinite(s)) return '0:00'
  const m = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return `${m}:${sec.toString().padStart(2, '0')}`
}

export function useMediaPlayer() {
  return {
    state,
    currentTrack,
    play,
    stop,
    togglePlay,
    skipNext,
    skipPrev,
    seek,
    setVolume,
    setLoopMode,
    toggleShuffle,
    setPlaylist,
    addToPlaylist,
    playAll,
    buildPlaylist,
    fmtTime,
  }
}
