package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"devtools/config"

	"github.com/gin-gonic/gin"
)

// GameHandler 游戏处理器
type GameHandler struct {
	minimax config.MiniMaxConfig
}

// NewGameHandler 创建游戏处理器
func NewGameHandler(minimax config.MiniMaxConfig) *GameHandler {
	return &GameHandler{minimax: minimax}
}

// ============ 通用游戏状态 ============

// TicTacToeState 井字棋状态
type TicTacToeState struct {
	Board   [9]int    `json:"board"`   // 0=空, 1=玩家, 2=AI
	Current int      `json:"current"` // 当前玩家 1或2
	Winner  int      `json:"winner"`  // 赢家 0=无, 1=玩家, 2=AI, 3=平局
	Mode    string   `json:"mode"`    // "pvp" 或 "ai"
	AIType  string   `json:"ai_type"` // "random" 或 "smart"
}

// GomokuState 五子棋状态
type GomokuState struct {
	Board   [15][15]int `json:"board"`   // 0=空, 1=玩家, 2=AI
	Current int         `json:"current"` // 当前玩家
	Winner  int         `json:"winner"`  // 赢家
	Mode    string      `json:"mode"`
}

// GuessNumberState 猜数字状态
type GuessNumberState struct {
	Target       int      `json:"target"`       // 目标数字
	PlayerGuesses []int   `json:"player_guesses"`
	AIGuesses    []AIGuess `json:"ai_guesses"`
	PlayerScore  int      `json:"player_score"`
	AIScore      int      `json:"ai_score"`
	Round        int      `json:"round"` // 当前轮次 1-5
	Winner       int      `json:"winner"`
	Mode         string   `json:"mode"`
}

// AIGuess AI猜测记录
type AIGuess struct {
	Guess int    `json:"guess"`
	Hint  string `json:"hint"` // "big" 或 "small"
}

// RPSState 石头剪刀布状态
type RPSState struct {
	PlayerChoice string `json:"player_choice"`
	AIChoice     string `json:"ai_choice"`
	PlayerScore  int    `json:"player_score"`
	AIScore      int    `json:"ai_score"`
	Winner       string `json:"winner"` // "player", "ai", "draw"
	Round        int    `json:"round"`
	Mode         string `json:"mode"`
}

// DiceState 色子比大小状态
type DiceState struct {
	PlayerDice [3]int `json:"player_dice"`
	AIDice     [3]int `json:"ai_dice"`
	PlayerSum  int    `json:"player_sum"`
	AISum      int    `json:"ai_sum"`
	PlayerScore int   `json:"player_score"`
	AIScore     int   `json:"ai_score"`
	Winner      string `json:"winner"`
	Round       int    `json:"round"`
	Mode        string `json:"mode"`
}

// ============ API Handlers ============

// GetGameInfo 获取游戏信息
func (h *GameHandler) GetGameInfo(c *gin.Context) {
	game := c.Query("game")
	games := map[string]interface{}{
		"tictactoe": map[string]interface{}{
			"name":        "井字棋",
			"description": "经典井字棋游戏，三子连线获胜",
			"icon":        "Grid",
			"players":     2,
			"ai_support":  true,
		},
		"gomoku": map[string]interface{}{
			"name":        "五子棋",
			"description": "五子连珠获胜，体验对弈乐趣",
			"icon":        "Connection",
			"players":     2,
			"ai_support":  true,
		},
		"guessnumber": map[string]interface{}{
			"name":        "猜数字",
			"description": "猜猜数字是多少，5轮定胜负",
			"icon":        "Search",
			"players":     2,
			"ai_support":  true,
		},
		"rps": map[string]interface{}{
			"name":        "石头剪刀布",
			"description": "经典猜拳游戏，看谁运气好",
			"icon":        "Coin",
			"players":     2,
			"ai_support":  true,
		},
		"dice": map[string]interface{}{
			"name":        "色子比大小",
			"description": "投掷三颗色子，比比谁更幸运",
			"icon":        "Histogram",
			"players":     2,
			"ai_support":  true,
		},
	}

	if game != "" {
		if g, ok := games[game]; ok {
			c.JSON(http.StatusOK, g)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "游戏不存在"})
		return
	}
	c.JSON(http.StatusOK, games)
}

// InitTicTacToe 初始化井字棋
func (h *GameHandler) InitTicTacToe(c *gin.Context) {
	var req struct {
		Mode string `json:"mode"` // "pvp" 或 "ai"
		AIType string `json:"ai_type"` // "random" 或 "smart"
	}
	c.ShouldBindJSON(&req)

	if req.Mode == "" {
		req.Mode = "ai"
	}
	if req.AIType == "" {
		req.AIType = "random"
	}

	state := &TicTacToeState{
		Board:   [9]int{0, 0, 0, 0, 0, 0, 0, 0, 0},
		Current: 1, // 玩家先手
		Winner:  0,
		Mode:    req.Mode,
		AIType:  req.AIType,
	}

	c.JSON(http.StatusOK, state)
}

// MoveTicTacToe 井字棋落子
func (h *GameHandler) MoveTicTacToe(c *gin.Context) {
	var req struct {
		Position int `json:"position"` // 0-8
		State    TicTacToeState `json:"state"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
		return
	}

	state := req.State

	// 验证玩家落子
	if req.Position < 0 || req.Position > 8 || state.Board[req.Position] != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的位置"})
		return
	}

	// 玩家落子
	state.Board[req.Position] = 1
	state.Current = 2

	// 检查玩家是否获胜
	if checkTicTacToeWinner(state.Board, 1) {
		state.Winner = 1
		c.JSON(http.StatusOK, state)
		return
	}

	// 检查平局
	if isBoardFull(state.Board) {
		state.Winner = 3
		c.JSON(http.StatusOK, state)
		return
	}

	// AI 落子
	if state.Mode == "ai" {
		aiPos := getTicTacToeAIMove(state.Board, state.AIType)
		if aiPos >= 0 {
			state.Board[aiPos] = 2
			state.Current = 1

			// 检查AI是否获胜
			if checkTicTacToeWinner(state.Board, 2) {
				state.Winner = 2
			}
		}
	}

	c.JSON(http.StatusOK, state)
}

func checkTicTacToeWinner(board [9]int, player int) bool {
	lines := [][]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // 行
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // 列
		{0, 4, 8}, {2, 4, 6}, // 对角线
	}
	for _, line := range lines {
		if board[line[0]] == player && board[line[1]] == player && board[line[2]] == player {
			return true
		}
	}
	return false
}

func isBoardFull(board [9]int) bool {
	for _, v := range board {
		if v == 0 {
			return false
		}
	}
	return true
}

func getTicTacToeAIMove(board [9]int, aiType string) int {
	if aiType == "smart" {
		// 简单智能AI：优先获胜，其次阻挡玩家
		for i := 0; i < 9; i++ {
			if board[i] == 0 {
				// 尝试获胜
				board[i] = 2
				if checkTicTacToeWinner(board, 2) {
					board[i] = 0
					return i
				}
				board[i] = 0
			}
		}
		for i := 0; i < 9; i++ {
			if board[i] == 0 {
				// 阻挡玩家
				board[i] = 1
				if checkTicTacToeWinner(board, 1) {
					board[i] = 0
					return i
				}
				board[i] = 0
			}
		}
	}

	// 随机AI或没有好的位置
	empty := []int{}
	for i, v := range board {
		if v == 0 {
			empty = append(empty, i)
		}
	}
	if len(empty) == 0 {
		return -1
	}
	return empty[rand.Intn(len(empty))]
}

// InitGomoku 初始化五子棋
func (h *GameHandler) InitGomoku(c *gin.Context) {
	var req struct {
		Mode string `json:"mode"`
	}
	c.ShouldBindJSON(&req)
	if req.Mode == "" {
		req.Mode = "ai"
	}

	state := &GomokuState{
		Board:   [15][15]int{},
		Current: 1,
		Winner:  0,
		Mode:    req.Mode,
	}
	c.JSON(http.StatusOK, state)
}

// MoveGomoku 五子棋落子
func (h *GameHandler) MoveGomoku(c *gin.Context) {
	var req struct {
		Row   int `json:"row"`
		Col   int `json:"col"`
		State GomokuState `json:"state"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
		return
	}

	state := req.State

	// 验证落子
	if req.Row < 0 || req.Row > 14 || req.Col < 0 || req.Col > 14 || state.Board[req.Row][req.Col] != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效位置"})
		return
	}

	// 玩家落子
	state.Board[req.Row][req.Col] = 1
	state.Current = 2

	// 检查玩家是否获胜
	if checkGomokuWinner(state.Board, 1) {
		state.Winner = 1
		c.JSON(http.StatusOK, state)
		return
	}

	// AI落子
	if state.Mode == "ai" {
		aiRow, aiCol := getGomokuAIMove(state.Board)
		state.Board[aiRow][aiCol] = 2
		state.Current = 1

		if checkGomokuWinner(state.Board, 2) {
			state.Winner = 2
		}
	}

	c.JSON(http.StatusOK, state)
}

func checkGomokuWinner(board [15][15]int, player int) bool {
	// 检查四个方向：横、竖、两条对角线
	dirs := [][]int{
		{0, 1},  // 横
		{1, 0},  // 竖
		{1, 1},  // 主对角线
		{1, -1}, // 副对角线
	}

	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			if board[r][c] != player {
				continue
			}
			for _, dir := range dirs {
				dr, dc := dir[0], dir[1]
				count := 1
				for i := 1; i < 5; i++ {
					nr, nc := r+dr*i, c+dc*i
					if nr < 0 || nr >= 15 || nc < 0 || nc >= 15 {
						break
					}
					if board[nr][nc] != player {
						break
					}
					count++
				}
				if count >= 5 {
					return true
				}
			}
		}
	}
	return false
}

func getGomokuAIMove(board [15][15]int) (int, int) {
	// 简单AI：找最接近已有棋子的空位
	empty := []struct{ row, col, score int }{}

	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			if board[r][c] != 0 {
				continue
			}

			score := 0
			// 简单评分：周围有己方棋子和敌方棋子
			for dr := -1; dr <= 1; dr++ {
				for dc := -1; dc <= 1; dc++ {
					if dr == 0 && dc == 0 {
						continue
					}
					nr, nc := r+dr, c+dc
					if nr >= 0 && nr < 15 && nc >= 0 && nc < 15 {
						if board[nr][nc] == 2 {
							score += 2
						} else if board[nr][nc] == 1 {
							score += 1
						}
					}
				}
			}
			empty = append(empty, struct{ row, col, score int }{r, c, score})
		}
	}

	if len(empty) == 0 {
		return 7, 7 // 默认中心
	}

	// 按分数排序，分数高的优先
	for i := 0; i < len(empty)-1; i++ {
		for j := i + 1; j < len(empty); j++ {
			if empty[j].score > empty[i].score {
				empty[i], empty[j] = empty[j], empty[i]
			}
		}
	}

	// 取最高分的几个随机选
	topScore := empty[0].score
	topMoves := []struct{ row, col int }{}
	for _, e := range empty {
		if e.score == topScore {
			topMoves = append(topMoves, struct{ row, col int }{e.row, e.col})
		}
	}

	m := topMoves[rand.Intn(len(topMoves))]
	return m.row, m.col
}

// InitGuessNumber 初始化猜数字
func (h *GameHandler) InitGuessNumber(c *gin.Context) {
	var req struct {
		Mode string `json:"mode"`
	}
	c.ShouldBindJSON(&req)
	if req.Mode == "" {
		req.Mode = "ai"
	}

	state := &GuessNumberState{
		Target:       rand.Intn(100) + 1, // 1-100
		PlayerGuesses: []int{},
		AIGuesses:    []AIGuess{},
		PlayerScore:  0,
		AIScore:      0,
		Round:        1,
		Winner:       0,
		Mode:         req.Mode,
	}

	c.JSON(http.StatusOK, state)
}

// Guess 猜数字
func (h *GameHandler) Guess(c *gin.Context) {
	var req struct {
		Guess int             `json:"guess"`
		State GuessNumberState `json:"state"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
		return
	}

	state := req.State

	if req.Guess < 1 || req.Guess > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "数字必须在1-100之间"})
		return
	}

	state.PlayerGuesses = append(state.PlayerGuesses, req.Guess)

	var hint string
	if req.Guess == state.Target {
		hint = "correct"
		state.PlayerScore++
	} else if req.Guess > state.Target {
		hint = "big"
	} else {
		hint = "small"
	}

	// 检查是否结束（猜对或5轮用完）
	if hint == "correct" || len(state.PlayerGuesses) >= 5 {
		if hint != "correct" {
			// 玩家没猜对，AI猜
			aiGuess := state.Target // AI直接知道答案了（简单实现）
			aiHint := "correct"
			state.AIGuesses = append(state.AIGuesses, AIGuess{aiGuess, aiHint})
			state.AIScore++
		}
		state.Winner = determineGuessWinner(state.PlayerScore, state.AIScore)
		c.JSON(http.StatusOK, state)
		return
	}

	// AI回合
	if state.Mode == "ai" {
		// 智能AI：根据之前的猜测缩小范围
		aiGuess := getAIGuess(state.AIGuesses, state.Target)
		var aiHint string
		if aiGuess == state.Target {
			aiHint = "correct"
			state.AIScore++
		} else if aiGuess > state.Target {
			aiHint = "big"
		} else {
			aiHint = "small"
		}
		state.AIGuesses = append(state.AIGuesses, AIGuess{aiGuess, aiHint})
	}

	state.Round++

	// 5轮结束
	if state.Round > 5 {
		state.Winner = determineGuessWinner(state.PlayerScore, state.AIScore)
	}

	c.JSON(http.StatusOK, state)
}

func getAIGuess(guesses []AIGuess, target int) int {
	// AI实际根据提示来猜（简化：AI知道答案）
	return target
}

func determineGuessWinner(playerScore, aiScore int) int {
	if playerScore > aiScore {
		return 1
	} else if aiScore > playerScore {
		return 2
	}
	return 3
}

// InitRPS 初始化石头剪刀布
func (h *GameHandler) InitRPS(c *gin.Context) {
	var req struct {
		Mode string `json:"mode"`
	}
	c.ShouldBindJSON(&req)
	if req.Mode == "" {
		req.Mode = "ai"
	}

	state := &RPSState{
		PlayerScore: 0,
		AIScore:     0,
		Round:       1,
		Mode:        req.Mode,
	}
	c.JSON(http.StatusOK, state)
}

// PlayRPS 石头剪刀布出拳
func (h *GameHandler) PlayRPS(c *gin.Context) {
	var req struct {
		Choice string   `json:"choice"`
		State  RPSState `json:"state"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
		return
	}

	if req.Choice != "rock" && req.Choice != "paper" && req.Choice != "scissors" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效选择"})
		return
	}

	state := req.State
	state.PlayerChoice = req.Choice

	choices := []string{"rock", "paper", "scissors"}
	aiChoice := choices[rand.Intn(3)]
	state.AIChoice = aiChoice

	// 判断胜负
	if req.Choice == aiChoice {
		state.Winner = "draw"
	} else if (req.Choice == "rock" && aiChoice == "scissors") ||
		(req.Choice == "paper" && aiChoice == "rock") ||
		(req.Choice == "scissors" && aiChoice == "paper") {
		state.Winner = "player"
		state.PlayerScore++
	} else {
		state.Winner = "ai"
		state.AIScore++
	}

	state.Round++

	// 10局结束
	if state.Round > 10 {
		if state.PlayerScore > state.AIScore {
			state.Winner = "player"
		} else if state.AIScore > state.PlayerScore {
			state.Winner = "ai"
		} else {
			state.Winner = "draw"
		}
	}

	c.JSON(http.StatusOK, state)
}

// InitDice 初始化色子比大小
func (h *GameHandler) InitDice(c *gin.Context) {
	var req struct {
		Mode string `json:"mode"`
	}
	c.ShouldBindJSON(&req)
	if req.Mode == "" {
		req.Mode = "ai"
	}

	state := &DiceState{
		PlayerScore: 0,
		AIScore:     0,
		Round:       1,
		Mode:        req.Mode,
	}
	c.JSON(http.StatusOK, state)
}

// RollDice 投掷色子
func (h *GameHandler) RollDice(c *gin.Context) {
	var req struct {
		State DiceState `json:"state"`
	}
	c.ShouldBindJSON(&req)

	state := req.State

	// 投掷玩家色子
	state.PlayerDice = [3]int{rand.Intn(6) + 1, rand.Intn(6) + 1, rand.Intn(6) + 1}
	state.PlayerSum = state.PlayerDice[0] + state.PlayerDice[1] + state.PlayerDice[2]

	// AI色子
	state.AIDice = [3]int{rand.Intn(6) + 1, rand.Intn(6) + 1, rand.Intn(6) + 1}
	state.AISum = state.AIDice[0] + state.AIDice[1] + state.AIDice[2]

	if state.PlayerSum > state.AISum {
		state.Winner = "player"
		state.PlayerScore++
	} else if state.AISum > state.PlayerSum {
		state.Winner = "ai"
		state.AIScore++
	} else {
		state.Winner = "draw"
	}

	state.Round++

	// 10轮结束
	if state.Round > 10 {
		if state.PlayerScore > state.AIScore {
			state.Winner = "player_final"
		} else if state.AIScore > state.PlayerScore {
			state.Winner = "ai_final"
		} else {
			state.Winner = "draw_final"
		}
	}

	c.JSON(http.StatusOK, state)
}

// AIChat 用大模型进行游戏对话（用于某些需要AI互动的游戏）
type AIChatRequest struct {
	Game    string   `json:"game"`
	Message string   `json:"message"`
	History []string `json:"history"`
}

func (h *GameHandler) AIChat(c *gin.Context) {
	var req AIChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
		return
	}

	// 构建prompt
	systemPrompt := getGameSystemPrompt(req.Game)

	response, err := h.callMiniMaxForGame(systemPrompt, req.Message, req.History)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

func getGameSystemPrompt(game string) string {
	prompts := map[string]string{
		"riddle": "你是一个有趣的猜谜主持人，出一些简单的中文谜语让玩家猜。谜语要有趣、适合放松心情。回复格式：\n谜面：（谜面内容）\n答案：（谜底）\n提示：（简单提示）",
		"fortune": "你是一个运势分析师，根据给定的主题（大吉/吉/中平/小凶）生成今日运势。回复包含：运势描述（20字内）、幸运数字、幸运颜色、幸运时段、今日提示、AI点评。",
		"quiz": "你是一个趣味知识问答主持人，根据用户要求的类别出题。回复格式：\n问题：（问题内容）\nA. （选项1）\nB. （选项2）\nC. （选项3）\nD. （选项4）\n答案：（正确答案）\n解释：（简单解释20字内）",
		"idiom": "你是一个成语接龙游戏搭档。玩家说一个成语，你接下一个成语（首字接尾字，可以谐音）。回复简短，只回复成语即可。",
		"story":  "你是一个讲故事的高手，根据玩家的选择继续故事。回复要简短有趣（50字以内），给玩家2-3个选择。",
		"joke":   "你是一个幽默大师，讲一些轻松有趣的笑话或脑筋急转弯。",
	}
	if p, ok := prompts[game]; ok {
		return p
	}
	return "你是一个有趣的游戏AI，陪玩家玩轻松的小游戏。"
}

func (h *GameHandler) callMiniMaxForGame(systemPrompt, message string, history []string) (string, error) {
	apiKey := h.minimax.APIKey
	if apiKey == "" {
		return "", fmt.Errorf("API key not configured")
	}

	// 构建消息
	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
	}
	for _, histItem := range history {
		messages = append(messages, map[string]string{"role": "user", "content": histItem})
	}
	messages = append(messages, map[string]string{"role": "user", "content": message})

	reqBody := map[string]interface{}{
		"model": "MiniMax-M2.2",
		"messages": messages,
	}

	body, _ := json.Marshal(reqBody)
	apiURL := "https://api.minimaxi.com/v1/text/chatcompletion_v2"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if errMsg, ok := result["error"].(string); ok {
		return "", fmt.Errorf("%s", errMsg)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	msg, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid message format")
	}

	content, ok := msg["content"].(string)
	if !ok {
		return "", fmt.Errorf("invalid content format")
	}

	// 清理回复
	content = strings.TrimSpace(content)
	return content, nil
}
