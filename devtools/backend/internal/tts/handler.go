package tts

import (
	"errors"
	"log"
	"net/http"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// Handler は POST /api/tts を処理する Gin ハンドラです。
type Handler struct {
	svc Service
}

// NewHandler は新しい Handler を生成します。
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// HandleSynthesize は POST /api/tts を処理します。
//
// 成功時: 200 + Content-Type: audio/wav + X-TTS-Cache: hit|miss + Body=WAV
// 失敗時: 4xx/5xx + JSON TTSErrorResponse
func (h *Handler) HandleSynthesize(c *gin.Context) {
	var req TTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[TTSHandler] HandleSynthesize failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, TTSErrorResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	// text バリデーション (W-1: handler 層で実施)
	if req.Text == "" {
		log.Printf("[TTSHandler] HandleSynthesize failed: text is empty")
		c.JSON(http.StatusBadRequest, TTSErrorResponse{
			Success: false,
			Error:   "テキストが不正です",
		})
		return
	}
	textLen := utf8.RuneCountInString(req.Text)
	if textLen > MaxTextLength {
		log.Printf("[TTSHandler] HandleSynthesize failed: text too long, textLen=%d, max=%d", textLen, MaxTextLength)
		c.JSON(http.StatusBadRequest, TTSErrorResponse{
			Success: false,
			Error:   "テキストが不正です",
		})
		return
	}

	log.Printf("[TTSHandler] HandleSynthesize started: textLen=%d", textLen)

	result, err := h.svc.Synthesize(c.Request.Context(), SynthesizeParams{Text: req.Text})
	if err != nil {
		status, message := mapErrorToStatus(err)
		log.Printf("[TTSHandler] HandleSynthesize failed: status=%d, error=%v", status, err)
		c.JSON(status, TTSErrorResponse{
			Success: false,
			Error:   message,
		})
		return
	}

	cacheHeader := "miss"
	if result.FromCache {
		cacheHeader = "hit"
	}
	c.Header("X-TTS-Cache", cacheHeader)

	log.Printf("[TTSHandler] HandleSynthesize completed: bytes=%d, cache=%s", len(result.Audio), cacheHeader)
	c.Data(http.StatusOK, result.ContentType, result.Audio)
}

// mapErrorToStatus はサービス層のエラーを HTTP ステータスとユーザー向けメッセージへ変換します。
func mapErrorToStatus(err error) (int, string) {
	if errors.Is(err, ErrTextEmpty) || errors.Is(err, ErrTextTooLong) {
		return http.StatusBadRequest, "テキストが不正です"
	}
	if errors.Is(err, ErrUpstreamTimeout) {
		return http.StatusGatewayTimeout, "VOICEVOX への接続に失敗しました"
	}

	var ue *UpstreamStatusError
	if errors.As(err, &ue) {
		if ue.Status == http.StatusTooManyRequests {
			return http.StatusTooManyRequests, "VOICEVOX レート制限"
		}
		return http.StatusBadGateway, "VOICEVOX から音声を取得できませんでした"
	}

	if errors.Is(err, ErrInvalidContentType) {
		return http.StatusBadGateway, "VOICEVOX から音声を取得できませんでした"
	}

	// その他は 502 に丸める
	return http.StatusBadGateway, "VOICEVOX から音声を取得できませんでした"
}
