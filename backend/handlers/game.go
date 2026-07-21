package handlers

import (
	"devtools/models"
)

// GameHandler 仅保留新的多人街机房逻辑。
type GameHandler struct {
	db     *models.DB
	arcade *arcadeHub
}

// NewGameHandler 创建游戏处理器。
func NewGameHandler(db *models.DB) *GameHandler {
	return &GameHandler{
		db:     db,
		arcade: newArcadeHub(),
	}
}
