// Package handler はHTTPハンドラーを提供します
package handler

import (
	"log"
	"net/http"

	"ghostrunner/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// GeminiTokenRequest はトークン発行リクエストの構造体です
type GeminiTokenRequest struct {
	ExpireSeconds int `json:"expireSeconds"` // トークン有効期限（秒）、デフォルト3600
}

// GeminiTokenResponse はトークン発行レスポンスの構造体です
type GeminiTokenResponse struct {
	Success    bool   `json:"success"`              // 成功フラグ
	Token      string `json:"token,omitempty"`      // エフェメラルトークン
	ExpireTime string `json:"expireTime,omitempty"` // トークン有効期限（ISO8601形式）
	Error      string `json:"error,omitempty"`      // エラーメッセージ
}

// GeminiHandler はGemini関連のHTTPハンドラを提供します
type GeminiHandler struct {
	geminiService service.GeminiService
}

// NewGeminiHandler は新しいGeminiHandlerを生成します
func NewGeminiHandler(geminiService service.GeminiService) *GeminiHandler {
	return &GeminiHandler{
		geminiService: geminiService,
	}
}

// HandleToken はエフェメラルトークン発行リクエストを処理します
// POST /api/gemini/token
func (h *GeminiHandler) HandleToken(c *gin.Context) {
	log.Printf("[GeminiHandler] HandleToken started")

	// サービスが利用不可（API キー未設定）の場合
	if h.geminiService == nil {
		log.Printf("[GeminiHandler] HandleToken failed: Gemini service is not available")
		c.JSON(http.StatusServiceUnavailable, GeminiTokenResponse{
			Success: false,
			Error:   "Gemini サービスが利用できません",
		})
		return
	}

	// リクエストをパース（空ボディも許容）
	var req GeminiTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// JSONパースエラーだが、空ボディの場合はデフォルト値を使用
		// gin.ShouldBindJSON は EOF エラーも返すので、空ボディを許容する
		if err.Error() != "EOF" {
			log.Printf("[GeminiHandler] HandleToken failed: invalid request, error=%v", err)
			c.JSON(http.StatusBadRequest, GeminiTokenResponse{
				Success: false,
				Error:   "リクエストが不正です",
			})
			return
		}
	}

	// デフォルト値を設定
	expireSeconds := req.ExpireSeconds
	if expireSeconds == 0 {
		expireSeconds = 3600 // デフォルト1時間
	}

	log.Printf("[GeminiHandler] HandleToken: expireSeconds=%d", expireSeconds)

	// トークンを発行
	result, err := h.geminiService.ProvisionEphemeralToken(expireSeconds)
	if err != nil {
		log.Printf("[GeminiHandler] HandleToken failed: error=%v", err)
		c.JSON(http.StatusInternalServerError, GeminiTokenResponse{
			Success: false,
			Error:   "トークン発行に失敗しました",
		})
		return
	}

	log.Printf("[GeminiHandler] HandleToken completed: expireTime=%s", result.ExpireTime)

	c.JSON(http.StatusOK, GeminiTokenResponse{
		Success:    true,
		Token:      result.Token,
		ExpireTime: result.ExpireTime,
	})
}
