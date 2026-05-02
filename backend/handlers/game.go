package handlers

// GameHandler 仅保留新的多人街机房逻辑。
type GameHandler struct {
	arcade *arcadeHub
}

// NewGameHandler 创建游戏处理器。
func NewGameHandler() *GameHandler {
	return &GameHandler{
		arcade: newArcadeHub(),
	}
}
