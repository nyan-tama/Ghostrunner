package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"ghostrunner/backend/internal/dashboard"

	"github.com/gin-gonic/gin"
)

// DashboardHandler はダッシュボード関連のHTTPハンドラを提供します
type DashboardHandler struct {
	svc       dashboard.Service
	streamSvc dashboard.StreamService
}

// NewDashboardHandler は新しいDashboardHandlerを生成します。
// streamSvc は SSE 配信(HandleStream)用で、nil の場合 HandleStream は利用できません。
func NewDashboardHandler(svc dashboard.Service, streamSvc dashboard.StreamService) *DashboardHandler {
	return &DashboardHandler{svc: svc, streamSvc: streamSvc}
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

// HandleStream はダッシュボード状態のSSEストリーミングを提供します。
// 状態に実変化があるたびに State スナップショット全体を配信します。
// GET /api/dashboard/stream
func (h *DashboardHandler) HandleStream(c *gin.Context) {
	log.Println("[DashboardHandler] HandleStream started")

	if h.streamSvc == nil {
		log.Println("[DashboardHandler] HandleStream unavailable: streamSvc is nil")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "stream not available",
		})
		return
	}

	setSSEHeaders(c)

	ch, unsubscribe := h.streamSvc.Subscribe()
	defer unsubscribe()

	writeDashboardSSEEvents(c, ch)

	log.Println("[DashboardHandler] HandleStream completed")
}

// writeDashboardSSEEvents は State チャネルから受け取ったスナップショットを
// SSE形式(data: <JSON>)で送信します。既存の writeSSEEvents / writePatrolSSEEvents は
// 型固定のため流用できず、dashboard.State 専用の書き出しループを新設しています（W6）。
func writeDashboardSSEEvents(c *gin.Context, stateCh <-chan dashboard.State) {
	w := c.Writer
	flusher, ok := w.(interface{ Flush() })
	if !ok {
		log.Printf("[DashboardHandler] ResponseWriter does not support Flush")
		return
	}

	keepalive := time.NewTicker(sseKeepaliveInterval)
	defer keepalive.Stop()

	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[DashboardHandler] Client disconnected (context canceled)")
			return

		case state, ok := <-stateCh:
			if !ok {
				log.Printf("[DashboardHandler] State channel closed, stream completed")
				return
			}

			data, err := json.Marshal(state)
			if err != nil {
				log.Printf("[DashboardHandler] Marshal error: %v", err)
				continue
			}

			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				log.Printf("[DashboardHandler] SSE write error (client disconnected): %v", err)
				return
			}
			flusher.Flush()

		case <-keepalive.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				log.Printf("[DashboardHandler] Keepalive write error (client disconnected): %v", err)
				return
			}
			flusher.Flush()
		}
	}
}
