// Package handler はHTTPハンドラーを提供します
package handler

import (
	"log"
	"net/http"

	"ghostrunner/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// OpenAISessionRequest はセッション作成リクエストの構造体です
type OpenAISessionRequest struct {
	Model string `json:"model"` // 使用するモデル（オプション）
	Voice string `json:"voice"` // 音声タイプ（オプション）
}

// OpenAISessionResponse はセッション作成レスポンスの構造体です
type OpenAISessionResponse struct {
	Success    bool   `json:"success"`              // 成功フラグ
	Token      string `json:"token,omitempty"`      // エフェメラルキー
	ExpireTime string `json:"expireTime,omitempty"` // トークン有効期限（ISO8601形式）
	Error      string `json:"error,omitempty"`      // エラーメッセージ
}

// OpenAIHandler はOpenAI関連のHTTPハンドラを提供します
type OpenAIHandler struct {
	openaiService service.OpenAIService
}

// NewOpenAIHandler は新しいOpenAIHandlerを生成します
func NewOpenAIHandler(openaiService service.OpenAIService) *OpenAIHandler {
	return &OpenAIHandler{
		openaiService: openaiService,
	}
}

// HandleSession はエフェメラルキー発行リクエストを処理します
// POST /api/openai/realtime/session
func (h *OpenAIHandler) HandleSession(c *gin.Context) {
	log.Printf("[OpenAIHandler] HandleSession started")

	// サービスが利用不可（API キー未設定）の場合
	if h.openaiService == nil {
		log.Printf("[OpenAIHandler] HandleSession failed: OpenAI service is not available")
		c.JSON(http.StatusServiceUnavailable, OpenAISessionResponse{
			Success: false,
			Error:   "OpenAI サービスが利用できません",
		})
		return
	}

	// リクエストをパース（空ボディも許容）
	var req OpenAISessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// JSONパースエラーだが、空ボディの場合はデフォルト値を使用
		// gin.ShouldBindJSON は EOF エラーも返すので、空ボディを許容する
		if err.Error() != "EOF" {
			log.Printf("[OpenAIHandler] HandleSession failed: invalid request, error=%v", err)
			c.JSON(http.StatusBadRequest, OpenAISessionResponse{
				Success: false,
				Error:   "リクエストが不正です",
			})
			return
		}
	}

	log.Printf("[OpenAIHandler] HandleSession: model=%s, voice=%s", req.Model, req.Voice)

	// セッションを作成
	result, err := h.openaiService.CreateRealtimeSession(req.Model, req.Voice)
	if err != nil {
		log.Printf("[OpenAIHandler] HandleSession failed: error=%v", err)
		c.JSON(http.StatusInternalServerError, OpenAISessionResponse{
			Success: false,
			Error:   "セッション作成に失敗しました",
		})
		return
	}

	log.Printf("[OpenAIHandler] HandleSession completed: expireTime=%s", result.ExpireTime)

	c.JSON(http.StatusOK, OpenAISessionResponse{
		Success:    true,
		Token:      result.Token,
		ExpireTime: result.ExpireTime,
	})
}
