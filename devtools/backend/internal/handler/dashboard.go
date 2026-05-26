package handler

import (
	"errors"
	"log"
	"net/http"

	"ghostrunner/backend/internal/dashboard"

	"github.com/gin-gonic/gin"
)

// DashboardHandler はダッシュボード関連のHTTPハンドラを提供します
type DashboardHandler struct {
	svc dashboard.Service
}

// NewDashboardHandler は新しいDashboardHandlerを生成します
func NewDashboardHandler(svc dashboard.Service) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// HandleState はダッシュボードの状態を返します
// GET /api/dashboard/state
func (h *DashboardHandler) HandleState(c *gin.Context) {
	log.Println("[DashboardHandler] HandleState started")

	state, err := h.svc.GetState(c.Request.Context())
	if err != nil {
		log.Printf("[DashboardHandler] HandleState failed: error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "ダッシュボード状態の取得に失敗しました",
		})
		return
	}

	log.Printf("[DashboardHandler] HandleState completed: projects=%d", len(state.Projects))
	c.JSON(http.StatusOK, state)
}

// HandleAnswer は確認事項への回答を処理します
// POST /api/dashboard/answer
func (h *DashboardHandler) HandleAnswer(c *gin.Context) {
	log.Println("[DashboardHandler] HandleAnswer started")

	var req dashboard.AnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "リクエストが不正です",
		})
		return
	}

	if err := h.svc.Answer(c.Request.Context(), req); err != nil {
		log.Printf("[DashboardHandler] HandleAnswer failed: error=%v", err)

		if errors.Is(err, dashboard.ErrValidation) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		if errors.Is(err, dashboard.ErrAlreadyAnswered) {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error":   "既に回答済みか、行がずれています",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "回答の書き戻しに失敗しました",
		})
		return
	}

	log.Println("[DashboardHandler] HandleAnswer completed")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}
