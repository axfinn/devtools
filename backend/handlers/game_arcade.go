package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"math"
	mathrand "math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	arcadeGameReaction = "reaction"
	arcadeGameHunt     = "hunt"
	arcadeGameBeat     = "beat"
	arcadeGameSequence = "sequence"

	arcadePhaseLobby     = "lobby"
	arcadePhaseCountdown = "countdown"
	arcadePhaseArming    = "arming"
	arcadePhaseLive      = "live"
	arcadePhaseResult    = "result"
	arcadePhaseFinished  = "finished"

	arcadeMaxPlayers = 8
	arcadeRounds     = 5
)

var (
	arcadePlayerColors  = []string{"#ff7a59", "#00b894", "#4f46e5", "#eab308", "#ec4899", "#14b8a6", "#f97316", "#8b5cf6"}
	reactionRoundAwards = []int{5, 3, 2, 1}
	huntRoundAwards     = []int{4, 2, 1}
	sequenceRoundAwards = []int{6, 4, 3, 2}
	huntEmojiSets       = [][]string{
		{"🍓", "🍒", "🍎", "🍉", "🍑", "🥝", "🍋", "🍇"},
		{"🐸", "🐼", "🦊", "🐵", "🐯", "🐶", "🐰", "🐻"},
		{"🚀", "🛸", "🛰️", "🌙", "⭐", "☄️", "🪐", "🌍"},
		{"🍔", "🍟", "🌮", "🍕", "🍩", "🍿", "🧁", "🍪"},
		{"⚽", "🏀", "🏈", "🎾", "🎯", "🎳", "🏐", "🏓"},
	}
)

type arcadeHub struct {
	mu    sync.RWMutex
	rooms map[string]*arcadeRoom
}

type arcadeRoom struct {
	ID           string
	Game         string
	Phase        string
	Round        int
	TotalRounds  int
	Message      string
	Prompt       string
	Players      map[string]*arcadePlayer
	CreatedAt    time.Time
	LastActivity time.Time
	UpdatedAt    time.Time
	CountdownAt  int64
	SignalAt     int64
	LiveAt       int64
	RoundEndsAt  int64
	ResultEndsAt int64
	Target       string
	Grid         []string
	WinnerIDs    []string
	BeatCycleMS  int
	BeatWindowMS int
	SequenceGoal int
	roundToken   int
	mu           sync.RWMutex
}

type arcadePlayer struct {
	ID          string
	Secret      string
	Nickname    string
	Color       string
	Score       int
	IsHost      bool
	Connected   bool
	JoinedAt    time.Time
	LastSeenAt  time.Time
	RoundState  string
	RoundAward  int
	MetricMS    int
	PickedIndex int
	Progress    int
	FinishedAt  int64
	conn        *websocket.Conn
	writeMu     sync.Mutex
}

type arcadePlayerView struct {
	ID          string `json:"id"`
	Nickname    string `json:"nickname"`
	Color       string `json:"color"`
	Score       int    `json:"score"`
	IsHost      bool   `json:"is_host"`
	Connected   bool   `json:"connected"`
	RoundState  string `json:"round_state"`
	RoundAward  int    `json:"round_award"`
	MetricMS    int    `json:"metric_ms"`
	PickedIndex int    `json:"picked_index"`
	Progress    int    `json:"progress"`
}

type arcadeSnapshot struct {
	RoomID       string             `json:"room_id"`
	Game         string             `json:"game"`
	Phase        string             `json:"phase"`
	Round        int                `json:"round"`
	TotalRounds  int                `json:"total_rounds"`
	Message      string             `json:"message"`
	Prompt       string             `json:"prompt,omitempty"`
	Players      []arcadePlayerView `json:"players"`
	CountdownAt  int64              `json:"countdown_at,omitempty"`
	SignalAt     int64              `json:"signal_at,omitempty"`
	LiveAt       int64              `json:"live_at,omitempty"`
	RoundEndsAt  int64              `json:"round_ends_at,omitempty"`
	ResultEndsAt int64              `json:"result_ends_at,omitempty"`
	Target       string             `json:"target,omitempty"`
	Grid         []string           `json:"grid,omitempty"`
	WinnerIDs    []string           `json:"winner_ids,omitempty"`
	CanStart     bool               `json:"can_start"`
	MaxPlayers   int                `json:"max_players"`
	BeatCycleMS  int                `json:"beat_cycle_ms,omitempty"`
	BeatWindowMS int                `json:"beat_window_ms,omitempty"`
	SequenceGoal int                `json:"sequence_goal,omitempty"`
}

type arcadeWSMessage struct {
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
	State   any    `json:"state,omitempty"`
}

type arcadeClientEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
}

type createArcadeRoomRequest struct {
	Game     string `json:"game"`
	Nickname string `json:"nickname"`
}

type joinArcadeRoomRequest struct {
	Nickname string `json:"nickname"`
}

func newArcadeHub() *arcadeHub {
	hub := &arcadeHub{
		rooms: make(map[string]*arcadeRoom),
	}
	go hub.cleanupLoop()
	return hub
}

func (h *arcadeHub) cleanupLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-30 * time.Minute)
		h.mu.Lock()
		for id, room := range h.rooms {
			room.mu.RLock()
			lastActivity := room.LastActivity
			connected := false
			for _, player := range room.Players {
				if player.Connected {
					connected = true
					break
				}
			}
			room.mu.RUnlock()

			if !connected && lastActivity.Before(cutoff) {
				delete(h.rooms, id)
			}
		}
		h.mu.Unlock()
	}
}

func (h *arcadeHub) createRoom(game, nickname string) (*arcadeRoom, *arcadePlayer, error) {
	if !isArcadeGameSupported(game) {
		return nil, nil, errArcadeBadGame
	}

	roomID := randomArcadeCode(3)
	playerID := randomArcadeCode(4)
	playerSecret := randomArcadeCode(8)
	now := time.Now()

	room := &arcadeRoom{
		ID:           roomID,
		Game:         game,
		Phase:        arcadePhaseLobby,
		Round:        0,
		TotalRounds:  arcadeRounds,
		Message:      defaultArcadeLobbyMessage(game),
		Players:      make(map[string]*arcadePlayer),
		CreatedAt:    now,
		LastActivity: now,
		UpdatedAt:    now,
	}
	player := &arcadePlayer{
		ID:          playerID,
		Secret:      playerSecret,
		Nickname:    nickname,
		Color:       arcadePlayerColors[0],
		IsHost:      true,
		JoinedAt:    now,
		LastSeenAt:  now,
		PickedIndex: -1,
	}
	room.Players[playerID] = player

	h.mu.Lock()
	defer h.mu.Unlock()
	for {
		if _, exists := h.rooms[roomID]; exists {
			roomID = randomArcadeCode(3)
			room.ID = roomID
			continue
		}
		break
	}
	h.rooms[room.ID] = room
	return room, player, nil
}

func (h *arcadeHub) getRoom(id string) *arcadeRoom {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rooms[strings.ToUpper(strings.TrimSpace(id))]
}

func (h *arcadeHub) joinRoom(roomID, nickname string) (*arcadeRoom, *arcadePlayer, error) {
	room := h.getRoom(roomID)
	if room == nil {
		return nil, nil, errArcadeRoomNotFound
	}

	room.mu.Lock()
	defer room.mu.Unlock()
	if room.Phase != arcadePhaseLobby {
		return nil, nil, errArcadeRoomClosed
	}
	if len(room.Players) >= arcadeMaxPlayers {
		return nil, nil, errArcadeRoomFull
	}
	for _, player := range room.Players {
		if strings.EqualFold(player.Nickname, nickname) {
			return nil, nil, errArcadeNicknameTaken
		}
	}

	now := time.Now()
	playerID := randomArcadeCode(4)
	playerSecret := randomArcadeCode(8)
	player := &arcadePlayer{
		ID:          playerID,
		Secret:      playerSecret,
		Nickname:    nickname,
		Color:       arcadePlayerColors[len(room.Players)%len(arcadePlayerColors)],
		JoinedAt:    now,
		LastSeenAt:  now,
		PickedIndex: -1,
	}
	room.Players[playerID] = player
	room.LastActivity = now
	room.UpdatedAt = now
	room.Message = defaultArcadeLobbyMessage(room.Game)

	return room, player, nil
}

func (h *GameHandler) CreateArcadeRoom(c *gin.Context) {
	var req createArcadeRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	nickname := sanitizeArcadeNickname(req.Nickname)
	if nickname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入昵称"})
		return
	}
	room, player, err := h.arcade.createRoom(strings.TrimSpace(req.Game), nickname)
	if err != nil {
		respondArcadeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"room_id":   room.ID,
		"game":      room.Game,
		"player_id": player.ID,
		"secret":    player.Secret,
	})
}

func (h *GameHandler) JoinArcadeRoom(c *gin.Context) {
	var req joinArcadeRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	nickname := sanitizeArcadeNickname(req.Nickname)
	if nickname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入昵称"})
		return
	}
	room, player, err := h.arcade.joinRoom(c.Param("id"), nickname)
	if err != nil {
		respondArcadeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"room_id":   room.ID,
		"game":      room.Game,
		"player_id": player.ID,
		"secret":    player.Secret,
	})
}

func (h *GameHandler) GetArcadeRoom(c *gin.Context) {
	room := h.arcade.getRoom(c.Param("id"))
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "房间不存在"})
		return
	}
	c.JSON(http.StatusOK, buildArcadeSnapshot(room))
}

func (h *GameHandler) ArcadeWS(c *gin.Context) {
	room := h.arcade.getRoom(c.Param("id"))
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "房间不存在"})
		return
	}

	playerID := strings.TrimSpace(c.Query("player_id"))
	secret := strings.TrimSpace(c.Query("secret"))
	if playerID == "" || secret == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少身份信息"})
		return
	}

	room.mu.Lock()
	player, ok := room.Players[playerID]
	if !ok || player.Secret != secret {
		room.mu.Unlock()
		c.JSON(http.StatusUnauthorized, gin.H{"error": "身份校验失败"})
		return
	}
	room.mu.Unlock()

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	attachArcadeConnection(room, player, conn)
	broadcastArcadeState(room)

	defer func() {
		room.mu.Lock()
		if player.conn == conn {
			player.Connected = false
			player.conn = nil
			player.LastSeenAt = time.Now()
			room.LastActivity = time.Now()
			room.UpdatedAt = time.Now()
		}
		room.mu.Unlock()
		conn.Close()
		broadcastArcadeState(room)
	}()

	for {
		var event arcadeClientEvent
		if err := conn.ReadJSON(&event); err != nil {
			return
		}
		h.handleArcadeEvent(room, playerID, event)
	}
}

func (h *GameHandler) handleArcadeEvent(room *arcadeRoom, playerID string, event arcadeClientEvent) {
	switch event.Type {
	case "start_match":
		if err := h.startArcadeMatch(room, playerID); err != nil {
			sendArcadeErrorToPlayer(room, playerID, err.Error())
		}
	case "restart_match":
		if err := h.resetArcadeToLobby(room, playerID); err != nil {
			sendArcadeErrorToPlayer(room, playerID, err.Error())
		}
	case "react":
		h.handleArcadeReaction(room, playerID)
	case "pick_tile":
		h.handleArcadePick(room, playerID, event.Index)
	case "tap_beat":
		h.handleArcadeBeatTap(room, playerID)
	case "pick_sequence":
		h.handleArcadeSequencePick(room, playerID, event.Index)
	}
}

func (h *GameHandler) startArcadeMatch(room *arcadeRoom, playerID string) error {
	room.mu.Lock()
	defer room.mu.Unlock()

	player := room.Players[playerID]
	if player == nil || !player.IsHost {
		return errArcadeHostOnly
	}
	if room.Phase != arcadePhaseLobby && room.Phase != arcadePhaseFinished {
		return errArcadeAlreadyStarted
	}
	if len(room.Players) < 2 {
		return errArcadeNeedMorePlayers
	}

	room.Round = 1
	room.WinnerIDs = nil
	room.roundToken++
	room.Message = "准备好了，第一回合要开始了"
	for _, p := range room.Players {
		p.Score = 0
		resetArcadePlayerRound(p)
	}

	switch room.Game {
	case arcadeGameReaction:
		startReactionRoundLocked(room)
	case arcadeGameHunt:
		startHuntRoundLocked(room)
	case arcadeGameBeat:
		startBeatRoundLocked(room)
	case arcadeGameSequence:
		startSequenceRoundLocked(room)
	}
	return nil
}

func (h *GameHandler) resetArcadeToLobby(room *arcadeRoom, playerID string) error {
	room.mu.Lock()
	defer room.mu.Unlock()

	player := room.Players[playerID]
	if player == nil || !player.IsHost {
		return errArcadeHostOnly
	}

	room.Phase = arcadePhaseLobby
	room.Round = 0
	room.WinnerIDs = nil
	room.Target = ""
	room.Prompt = ""
	room.Grid = nil
	room.CountdownAt = 0
	room.SignalAt = 0
	room.LiveAt = 0
	room.RoundEndsAt = 0
	room.ResultEndsAt = 0
	room.BeatCycleMS = 0
	room.BeatWindowMS = 0
	room.SequenceGoal = 0
	room.roundToken++
	room.Message = defaultArcadeLobbyMessage(room.Game)
	room.LastActivity = time.Now()
	room.UpdatedAt = room.LastActivity
	for _, p := range room.Players {
		p.Score = 0
		resetArcadePlayerRound(p)
	}
	go broadcastArcadeState(room)
	return nil
}

func (h *GameHandler) handleArcadeReaction(room *arcadeRoom, playerID string) {
	room.mu.Lock()
	player := room.Players[playerID]
	if player == nil {
		room.mu.Unlock()
		return
	}
	player.LastSeenAt = time.Now()
	room.LastActivity = player.LastSeenAt
	room.UpdatedAt = player.LastSeenAt

	switch room.Phase {
	case arcadePhaseCountdown, arcadePhaseArming:
		if player.RoundState != "" {
			room.mu.Unlock()
			return
		}
		player.RoundState = "false_start"
		player.RoundAward = -1
		player.MetricMS = -1
		room.mu.Unlock()
		broadcastArcadeState(room)
	case arcadePhaseLive:
		if player.RoundState != "" {
			room.mu.Unlock()
			return
		}
		player.RoundState = "reacted"
		player.MetricMS = int(time.Now().UnixMilli() - room.LiveAt)
		token := room.roundToken
		allDone := allArcadePlayersFinishedLocked(room)
		room.mu.Unlock()
		broadcastArcadeState(room)
		if allDone {
			go finishReactionRound(room, token)
		}
	default:
		room.mu.Unlock()
	}
}

func (h *GameHandler) handleArcadePick(room *arcadeRoom, playerID string, index int) {
	room.mu.Lock()
	player := room.Players[playerID]
	if player == nil || room.Game != arcadeGameHunt || room.Phase != arcadePhaseLive {
		room.mu.Unlock()
		return
	}
	if player.RoundState != "" || index < 0 || index >= len(room.Grid) {
		room.mu.Unlock()
		return
	}

	player.LastSeenAt = time.Now()
	room.LastActivity = player.LastSeenAt
	room.UpdatedAt = player.LastSeenAt
	player.PickedIndex = index
	if room.Grid[index] == room.Target {
		player.RoundState = "correct"
	} else {
		player.RoundState = "wrong"
	}
	token := room.roundToken
	allDone := allArcadePlayersFinishedLocked(room)
	room.mu.Unlock()

	broadcastArcadeState(room)
	if allDone {
		go finishHuntRound(room, token)
	}
}

func (h *GameHandler) handleArcadeBeatTap(room *arcadeRoom, playerID string) {
	room.mu.Lock()
	player := room.Players[playerID]
	if player == nil || room.Game != arcadeGameBeat || room.Phase != arcadePhaseLive {
		room.mu.Unlock()
		return
	}
	if player.RoundState != "" {
		room.mu.Unlock()
		return
	}

	now := time.Now()
	player.LastSeenAt = now
	room.LastActivity = now
	room.UpdatedAt = now

	elapsed := int(now.UnixMilli() - room.LiveAt)
	cycle := room.BeatCycleMS
	if cycle <= 0 {
		cycle = 1400
	}
	target := cycle / 2
	position := elapsed % cycle
	diff := position - target
	if diff < 0 {
		diff = -diff
	}
	player.MetricMS = diff

	switch {
	case diff <= 45:
		player.RoundState = "perfect"
		player.RoundAward = 5
	case diff <= 110:
		player.RoundState = "good"
		player.RoundAward = 3
	case diff <= 190:
		player.RoundState = "ok"
		player.RoundAward = 1
	default:
		player.RoundState = "miss"
		player.RoundAward = 0
	}
	player.Score += player.RoundAward

	token := room.roundToken
	allDone := allArcadePlayersFinishedLocked(room)
	room.mu.Unlock()

	broadcastArcadeState(room)
	if allDone {
		go finishBeatRound(room, token)
	}
}

func (h *GameHandler) handleArcadeSequencePick(room *arcadeRoom, playerID string, index int) {
	room.mu.Lock()
	player := room.Players[playerID]
	if player == nil || room.Game != arcadeGameSequence || room.Phase != arcadePhaseLive {
		room.mu.Unlock()
		return
	}
	if player.RoundState == "finished" || index < 0 || index >= len(room.Grid) {
		room.mu.Unlock()
		return
	}

	now := time.Now()
	player.LastSeenAt = now
	room.LastActivity = now
	room.UpdatedAt = now

	next := player.Progress + 1
	if room.SequenceGoal <= 0 {
		room.SequenceGoal = len(room.Grid)
	}
	if room.Grid[index] != intToArcadeString(next) {
		room.mu.Unlock()
		return
	}

	player.Progress++
	player.PickedIndex = index
	if player.Progress >= room.SequenceGoal {
		player.RoundState = "finished"
		player.FinishedAt = now.UnixMilli()
		player.MetricMS = int(player.FinishedAt - room.LiveAt)
	}
	token := room.roundToken
	allDone := allSequencePlayersFinishedLocked(room)
	room.mu.Unlock()

	broadcastArcadeState(room)
	if allDone {
		go finishSequenceRound(room, token)
	}
}

func attachArcadeConnection(room *arcadeRoom, player *arcadePlayer, conn *websocket.Conn) {
	room.mu.Lock()
	defer room.mu.Unlock()

	if player.conn != nil && player.conn != conn {
		player.conn.Close()
	}
	player.conn = conn
	player.Connected = true
	player.LastSeenAt = time.Now()
	room.LastActivity = player.LastSeenAt
	room.UpdatedAt = player.LastSeenAt
}

func buildArcadeSnapshot(room *arcadeRoom) arcadeSnapshot {
	room.mu.RLock()
	defer room.mu.RUnlock()
	return buildArcadeSnapshotLocked(room)
}

func buildArcadeSnapshotLocked(room *arcadeRoom) arcadeSnapshot {
	players := make([]arcadePlayerView, 0, len(room.Players))
	for _, player := range room.Players {
		players = append(players, arcadePlayerView{
			ID:          player.ID,
			Nickname:    player.Nickname,
			Color:       player.Color,
			Score:       player.Score,
			IsHost:      player.IsHost,
			Connected:   player.Connected,
			RoundState:  player.RoundState,
			RoundAward:  player.RoundAward,
			MetricMS:    player.MetricMS,
			PickedIndex: player.PickedIndex,
			Progress:    player.Progress,
		})
	}
	sort.Slice(players, func(i, j int) bool {
		if players[i].Score != players[j].Score {
			return players[i].Score > players[j].Score
		}
		return players[i].Nickname < players[j].Nickname
	})

	return arcadeSnapshot{
		RoomID:       room.ID,
		Game:         room.Game,
		Phase:        room.Phase,
		Round:        room.Round,
		TotalRounds:  room.TotalRounds,
		Message:      room.Message,
		Prompt:       room.Prompt,
		Players:      players,
		CountdownAt:  room.CountdownAt,
		SignalAt:     room.SignalAt,
		LiveAt:       room.LiveAt,
		RoundEndsAt:  room.RoundEndsAt,
		ResultEndsAt: room.ResultEndsAt,
		Target:       room.Target,
		Grid:         append([]string(nil), room.Grid...),
		WinnerIDs:    append([]string(nil), room.WinnerIDs...),
		CanStart:     room.Phase == arcadePhaseLobby || room.Phase == arcadePhaseFinished,
		MaxPlayers:   arcadeMaxPlayers,
		BeatCycleMS:  room.BeatCycleMS,
		BeatWindowMS: room.BeatWindowMS,
		SequenceGoal: room.SequenceGoal,
	}
}

func broadcastArcadeState(room *arcadeRoom) {
	room.mu.RLock()
	snapshot := buildArcadeSnapshotLocked(room)
	players := make([]*arcadePlayer, 0, len(room.Players))
	for _, player := range room.Players {
		players = append(players, player)
	}
	room.mu.RUnlock()

	payload, _ := json.Marshal(arcadeWSMessage{
		Type:  "snapshot",
		State: snapshot,
	})

	for _, player := range players {
		if player == nil || player.conn == nil {
			continue
		}
		player.writeMu.Lock()
		err := player.conn.WriteMessage(websocket.TextMessage, payload)
		player.writeMu.Unlock()
		if err != nil {
			room.mu.Lock()
			if player.conn != nil {
				player.conn.Close()
			}
			player.conn = nil
			player.Connected = false
			room.LastActivity = time.Now()
			room.UpdatedAt = room.LastActivity
			room.mu.Unlock()
		}
	}
}

func sendArcadeErrorToPlayer(room *arcadeRoom, playerID, message string) {
	room.mu.RLock()
	player := room.Players[playerID]
	room.mu.RUnlock()
	if player == nil || player.conn == nil {
		return
	}
	payload, _ := json.Marshal(arcadeWSMessage{
		Type:    "error",
		Message: message,
	})
	player.writeMu.Lock()
	_ = player.conn.WriteMessage(websocket.TextMessage, payload)
	player.writeMu.Unlock()
}

func startReactionRoundLocked(room *arcadeRoom) {
	now := time.Now()
	token := room.roundToken
	room.Game = arcadeGameReaction
	room.Phase = arcadePhaseCountdown
	room.Message = "红灯停，绿灯点，手快也别抢跑"
	room.Prompt = "等按钮亮起再按"
	room.Target = ""
	room.Grid = nil
	room.WinnerIDs = nil
	room.BeatCycleMS = 0
	room.BeatWindowMS = 0
	room.SequenceGoal = 0
	room.CountdownAt = now.Add(3 * time.Second).UnixMilli()
	room.SignalAt = 0
	room.LiveAt = 0
	room.RoundEndsAt = 0
	room.ResultEndsAt = 0
	room.LastActivity = now
	room.UpdatedAt = now
	for _, player := range room.Players {
		resetArcadePlayerRound(player)
	}
	go broadcastArcadeState(room)

	go func(roundToken int) {
		time.Sleep(3 * time.Second)
		armReactionRound(room, roundToken)
	}(token)
}

func armReactionRound(room *arcadeRoom, token int) {
	room.mu.Lock()
	if room.roundToken != token || room.Phase != arcadePhaseCountdown {
		room.mu.Unlock()
		return
	}
	delay := time.Duration(1200+mathrand.Intn(1800)) * time.Millisecond
	room.Phase = arcadePhaseArming
	room.Message = "别点，等按钮真的变亮"
	room.SignalAt = time.Now().Add(delay).UnixMilli()
	room.LastActivity = time.Now()
	room.UpdatedAt = room.LastActivity
	room.mu.Unlock()
	broadcastArcadeState(room)

	go func(roundToken int, wait time.Duration) {
		time.Sleep(wait)
		liveReactionRound(room, roundToken)
	}(token, delay)
}

func liveReactionRound(room *arcadeRoom, token int) {
	room.mu.Lock()
	if room.roundToken != token || room.Phase != arcadePhaseArming {
		room.mu.Unlock()
		return
	}
	now := time.Now()
	room.Phase = arcadePhaseLive
	room.Message = "现在，拍它"
	room.LiveAt = now.UnixMilli()
	room.RoundEndsAt = now.Add(2200 * time.Millisecond).UnixMilli()
	room.LastActivity = now
	room.UpdatedAt = now
	room.mu.Unlock()
	broadcastArcadeState(room)

	go func(roundToken int) {
		time.Sleep(2200 * time.Millisecond)
		finishReactionRound(room, roundToken)
	}(token)
}

func finishReactionRound(room *arcadeRoom, token int) {
	room.mu.Lock()
	if room.roundToken != token || room.Phase != arcadePhaseLive {
		room.mu.Unlock()
		return
	}

	reacted := make([]*arcadePlayer, 0, len(room.Players))
	for _, player := range room.Players {
		if player.RoundState == "reacted" {
			reacted = append(reacted, player)
		}
	}
	sort.Slice(reacted, func(i, j int) bool {
		return reacted[i].MetricMS < reacted[j].MetricMS
	})
	for i, player := range reacted {
		player.RoundState = "hit"
		if i < len(reactionRoundAwards) {
			player.RoundAward = reactionRoundAwards[i]
		} else {
			player.RoundAward = 1
		}
		player.Score += player.RoundAward
	}
	for _, player := range room.Players {
		if player.RoundState == "false_start" {
			player.Score = int(math.Max(float64(player.Score-1), 0))
		}
		if player.RoundState == "" {
			player.RoundState = "miss"
		}
	}

	room.Phase = arcadePhaseResult
	room.ResultEndsAt = time.Now().Add(2600 * time.Millisecond).UnixMilli()
	room.RoundEndsAt = 0
	room.CountdownAt = 0
	room.SignalAt = 0
	room.WinnerIDs = winningPlayerIDsLocked(room)
	room.Message = arcadeRoundResultMessage(room)
	room.LastActivity = time.Now()
	room.UpdatedAt = room.LastActivity
	nextRound := room.Round + 1
	room.mu.Unlock()

	broadcastArcadeState(room)
	go func(roundToken int, round int) {
		time.Sleep(2600 * time.Millisecond)
		startNextArcadeRound(room, roundToken, round)
	}(token, nextRound)
}

func startHuntRoundLocked(room *arcadeRoom) {
	now := time.Now()
	token := room.roundToken
	target, grid := buildHuntGrid()
	room.Game = arcadeGameHunt
	room.Phase = arcadePhaseLive
	room.Message = "找到目标表情，越快得分越高"
	room.Prompt = "找到目标后立刻点它"
	room.Target = target
	room.Grid = grid
	room.WinnerIDs = nil
	room.BeatCycleMS = 0
	room.BeatWindowMS = 0
	room.SequenceGoal = 0
	room.CountdownAt = 0
	room.SignalAt = 0
	room.LiveAt = now.UnixMilli()
	room.RoundEndsAt = now.Add(8 * time.Second).UnixMilli()
	room.ResultEndsAt = 0
	room.LastActivity = now
	room.UpdatedAt = now
	for _, player := range room.Players {
		resetArcadePlayerRound(player)
	}
	go broadcastArcadeState(room)

	go func(roundToken int) {
		time.Sleep(8 * time.Second)
		finishHuntRound(room, roundToken)
	}(token)
}

func finishHuntRound(room *arcadeRoom, token int) {
	room.mu.Lock()
	if room.roundToken != token || room.Phase != arcadePhaseLive || room.Game != arcadeGameHunt {
		room.mu.Unlock()
		return
	}

	correctPlayers := make([]*arcadePlayer, 0, len(room.Players))
	for _, player := range room.Players {
		if player.RoundState == "correct" {
			correctPlayers = append(correctPlayers, player)
		}
	}
	sort.Slice(correctPlayers, func(i, j int) bool {
		return correctPlayers[i].LastSeenAt.Before(correctPlayers[j].LastSeenAt)
	})
	for i, player := range correctPlayers {
		if i < len(huntRoundAwards) {
			player.RoundAward = huntRoundAwards[i]
		} else {
			player.RoundAward = 1
		}
		player.Score += player.RoundAward
	}
	for _, player := range room.Players {
		if player.RoundState == "" {
			player.RoundState = "miss"
		}
	}

	room.Phase = arcadePhaseResult
	room.ResultEndsAt = time.Now().Add(2600 * time.Millisecond).UnixMilli()
	room.RoundEndsAt = 0
	room.WinnerIDs = winningPlayerIDsLocked(room)
	room.Message = arcadeRoundResultMessage(room)
	room.LastActivity = time.Now()
	room.UpdatedAt = room.LastActivity
	nextRound := room.Round + 1
	room.mu.Unlock()

	broadcastArcadeState(room)
	go func(roundToken int, round int) {
		time.Sleep(2600 * time.Millisecond)
		startNextArcadeRound(room, roundToken, round)
	}(token, nextRound)
}

func startBeatRoundLocked(room *arcadeRoom) {
	now := time.Now()
	token := room.roundToken
	room.Game = arcadeGameBeat
	room.Phase = arcadePhaseLive
	room.Message = "在节拍条扫过中心时拍下去"
	room.Prompt = "中心蓝区 = 高分"
	room.Target = ""
	room.Grid = nil
	room.WinnerIDs = nil
	room.BeatCycleMS = 1400
	room.BeatWindowMS = 190
	room.SequenceGoal = 0
	room.CountdownAt = 0
	room.SignalAt = 0
	room.LiveAt = now.UnixMilli()
	room.RoundEndsAt = now.Add(4200 * time.Millisecond).UnixMilli()
	room.ResultEndsAt = 0
	room.LastActivity = now
	room.UpdatedAt = now
	for _, player := range room.Players {
		resetArcadePlayerRound(player)
	}
	go broadcastArcadeState(room)

	go func(roundToken int) {
		time.Sleep(4200 * time.Millisecond)
		finishBeatRound(room, roundToken)
	}(token)
}

func finishBeatRound(room *arcadeRoom, token int) {
	room.mu.Lock()
	if room.roundToken != token || room.Phase != arcadePhaseLive || room.Game != arcadeGameBeat {
		room.mu.Unlock()
		return
	}

	for _, player := range room.Players {
		if player.RoundState == "" {
			player.RoundState = "miss"
		}
	}

	room.Phase = arcadePhaseResult
	room.ResultEndsAt = time.Now().Add(2600 * time.Millisecond).UnixMilli()
	room.RoundEndsAt = 0
	room.WinnerIDs = winningPlayerIDsLocked(room)
	room.Message = arcadeRoundResultMessage(room)
	room.LastActivity = time.Now()
	room.UpdatedAt = room.LastActivity
	nextRound := room.Round + 1
	room.mu.Unlock()

	broadcastArcadeState(room)
	go func(roundToken int, round int) {
		time.Sleep(2600 * time.Millisecond)
		startNextArcadeRound(room, roundToken, round)
	}(token, nextRound)
}

func startSequenceRoundLocked(room *arcadeRoom) {
	now := time.Now()
	token := room.roundToken
	room.Game = arcadeGameSequence
	room.Phase = arcadePhaseLive
	room.Message = "从 1 开始按顺序点到最后"
	room.Prompt = "错了不会淘汰，但会浪费时间"
	room.Target = "1"
	room.Grid = buildSequenceGrid(12)
	room.WinnerIDs = nil
	room.BeatCycleMS = 0
	room.BeatWindowMS = 0
	room.SequenceGoal = len(room.Grid)
	room.CountdownAt = 0
	room.SignalAt = 0
	room.LiveAt = now.UnixMilli()
	room.RoundEndsAt = now.Add(14 * time.Second).UnixMilli()
	room.ResultEndsAt = 0
	room.LastActivity = now
	room.UpdatedAt = now
	for _, player := range room.Players {
		resetArcadePlayerRound(player)
	}
	go broadcastArcadeState(room)

	go func(roundToken int) {
		time.Sleep(14 * time.Second)
		finishSequenceRound(room, roundToken)
	}(token)
}

func finishSequenceRound(room *arcadeRoom, token int) {
	room.mu.Lock()
	if room.roundToken != token || room.Phase != arcadePhaseLive || room.Game != arcadeGameSequence {
		room.mu.Unlock()
		return
	}

	finishedPlayers := make([]*arcadePlayer, 0, len(room.Players))
	for _, player := range room.Players {
		if player.RoundState == "finished" {
			finishedPlayers = append(finishedPlayers, player)
		}
	}
	sort.Slice(finishedPlayers, func(i, j int) bool {
		return finishedPlayers[i].FinishedAt < finishedPlayers[j].FinishedAt
	})
	for i, player := range finishedPlayers {
		if i < len(sequenceRoundAwards) {
			player.RoundAward = sequenceRoundAwards[i]
		} else {
			player.RoundAward = 1
		}
		player.Score += player.RoundAward
	}
	for _, player := range room.Players {
		if player.RoundState == "" {
			player.RoundState = "timeout"
		}
	}

	room.Phase = arcadePhaseResult
	room.ResultEndsAt = time.Now().Add(2600 * time.Millisecond).UnixMilli()
	room.RoundEndsAt = 0
	room.WinnerIDs = winningPlayerIDsLocked(room)
	room.Message = arcadeRoundResultMessage(room)
	room.LastActivity = time.Now()
	room.UpdatedAt = room.LastActivity
	nextRound := room.Round + 1
	room.mu.Unlock()

	broadcastArcadeState(room)
	go func(roundToken int, round int) {
		time.Sleep(2600 * time.Millisecond)
		startNextArcadeRound(room, roundToken, round)
	}(token, nextRound)
}

func startNextArcadeRound(room *arcadeRoom, token int, nextRound int) {
	room.mu.Lock()
	if room.roundToken != token || room.Phase != arcadePhaseResult {
		room.mu.Unlock()
		return
	}
	if nextRound > room.TotalRounds {
		room.Phase = arcadePhaseFinished
		room.Round = room.TotalRounds
		room.Message = arcadeMatchSummaryLocked(room)
		room.ResultEndsAt = 0
		room.CountdownAt = 0
		room.SignalAt = 0
		room.LiveAt = 0
		room.RoundEndsAt = 0
		room.WinnerIDs = winningPlayerIDsLocked(room)
		room.LastActivity = time.Now()
		room.UpdatedAt = room.LastActivity
		room.mu.Unlock()
		broadcastArcadeState(room)
		return
	}

	room.Round = nextRound
	room.roundToken++
	switch room.Game {
	case arcadeGameReaction:
		startReactionRoundLocked(room)
		room.mu.Unlock()
		return
	case arcadeGameHunt:
		startHuntRoundLocked(room)
		room.mu.Unlock()
		return
	case arcadeGameBeat:
		startBeatRoundLocked(room)
		room.mu.Unlock()
		return
	case arcadeGameSequence:
		startSequenceRoundLocked(room)
		room.mu.Unlock()
		return
	}
	room.mu.Unlock()
}

func resetArcadePlayerRound(player *arcadePlayer) {
	player.RoundState = ""
	player.RoundAward = 0
	player.MetricMS = 0
	player.PickedIndex = -1
	player.Progress = 0
	player.FinishedAt = 0
}

func allArcadePlayersFinishedLocked(room *arcadeRoom) bool {
	for _, player := range room.Players {
		if player.RoundState == "" {
			return false
		}
	}
	return true
}

func allSequencePlayersFinishedLocked(room *arcadeRoom) bool {
	for _, player := range room.Players {
		if player.RoundState != "finished" {
			return false
		}
	}
	return true
}

func winningPlayerIDsLocked(room *arcadeRoom) []string {
	best := -1
	ids := make([]string, 0, len(room.Players))
	for _, player := range room.Players {
		if player.Score > best {
			best = player.Score
			ids = []string{player.ID}
		} else if player.Score == best {
			ids = append(ids, player.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func arcadeRoundResultMessage(room *arcadeRoom) string {
	if len(room.WinnerIDs) == 0 {
		return "这一回合结束了"
	}
	names := make([]string, 0, len(room.WinnerIDs))
	for _, id := range room.WinnerIDs {
		if player := room.Players[id]; player != nil {
			names = append(names, player.Nickname)
		}
	}
	if len(names) == 1 {
		return names[0] + " 暂时领先"
	}
	return strings.Join(names, " / ") + " 并列领先"
}

func arcadeMatchSummaryLocked(room *arcadeRoom) string {
	if len(room.WinnerIDs) == 0 {
		return "比赛结束"
	}
	if len(room.WinnerIDs) == 1 {
		if winner := room.Players[room.WinnerIDs[0]]; winner != nil {
			return winner.Nickname + " 拿下整场"
		}
	}
	return "比赛结束，平分秋色"
}

func buildHuntGrid() (string, []string) {
	set := huntEmojiSets[mathrand.Intn(len(huntEmojiSets))]
	targetIndex := mathrand.Intn(len(set))
	target := set[targetIndex]

	grid := make([]string, 0, 20)
	for len(grid) < 19 {
		item := set[mathrand.Intn(len(set))]
		if item == target {
			continue
		}
		grid = append(grid, item)
	}
	insertAt := mathrand.Intn(len(grid) + 1)
	grid = append(grid[:insertAt], append([]string{target}, grid[insertAt:]...)...)
	return target, grid
}

func defaultArcadeLobbyMessage(game string) string {
	switch game {
	case arcadeGameHunt:
		return "开房后拉人进来，准备玩表情猎场"
	case arcadeGameBeat:
		return "开房后拉人进来，准备玩节拍判定"
	case arcadeGameSequence:
		return "开房后拉人进来，准备玩数字接力"
	}
	return "开房后拉人进来，准备玩反应竞速"
}

func isArcadeGameSupported(game string) bool {
	switch strings.TrimSpace(game) {
	case arcadeGameReaction, arcadeGameHunt, arcadeGameBeat, arcadeGameSequence:
		return true
	default:
		return false
	}
}

func buildSequenceGrid(goal int) []string {
	grid := make([]string, 0, goal)
	for i := 1; i <= goal; i++ {
		grid = append(grid, intToArcadeString(i))
	}
	mathrand.Shuffle(len(grid), func(i, j int) {
		grid[i], grid[j] = grid[j], grid[i]
	})
	return grid
}

func intToArcadeString(value int) string {
	return strconv.Itoa(value)
}

func sanitizeArcadeNickname(raw string) string {
	name := strings.TrimSpace(raw)
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, "\r", "")
	runes := []rune(name)
	if len(runes) > 12 {
		name = string(runes[:12])
	}
	return name
}

func randomArcadeCode(bytesLen int) string {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return strings.ToUpper(hex.EncodeToString([]byte(time.Now().Format("150405"))))[:bytesLen*2]
	}
	return strings.ToUpper(hex.EncodeToString(buf))
}

var (
	errArcadeBadGame         = arcadeError("不支持这个游戏")
	errArcadeRoomNotFound    = arcadeError("房间不存在")
	errArcadeRoomClosed      = arcadeError("房间已经开打，下一局再来")
	errArcadeRoomFull        = arcadeError("房间满了")
	errArcadeNicknameTaken   = arcadeError("这个昵称已经有人用了")
	errArcadeHostOnly        = arcadeError("只有房主能开始")
	errArcadeAlreadyStarted  = arcadeError("这一局已经开始")
	errArcadeNeedMorePlayers = arcadeError("至少需要两个人")
)

type arcadeError string

func (e arcadeError) Error() string { return string(e) }

func respondArcadeError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}
