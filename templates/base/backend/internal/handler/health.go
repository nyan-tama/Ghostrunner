package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler はヘルスチェック用のハンドラー。
type HealthHandler struct{}

// NewHealthHandler は HealthHandler を生成する。
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Handle はヘルスチェックレスポンスを返す。
// GET /api/health -> {"status": "ok"}
func (h *HealthHandler) Handle(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
