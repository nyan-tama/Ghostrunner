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
// svc に nil を渡しても初期化は成功し、リクエスト時に 503 を返します
// (無条件登録パターン、ELEVENLABS_API_KEY 未設定環境でも main.go の配線を保つため)。
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// HandleSynthesize は POST /api/tts を処理します。
//
// 成功時: 200 + Content-Type: audio/mpeg + X-TTS-Cache: hit|miss + Body=MP3
// 失敗時: 4xx/5xx + JSON TTSErrorResponse
func (h *Handler) HandleSynthesize(c *gin.Context) {
	// svc 未利用(API キー未設定)時は 503
	if h.svc == nil {
		log.Printf("[TTSHandler] HandleSynthesize rejected: TTS service is not available")
		c.JSON(http.StatusServiceUnavailable, TTSErrorResponse{
			Success: false,
			Error:   "ElevenLabs サービスが利用できません",
		})
		return
	}

	var req TTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[TTSHandler] HandleSynthesize failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, TTSErrorResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	// text バリデーション
	if req.Text == "" {
		log.Printf("[TTSHandler] HandleSynthesize failed: text is empty")
		c.JSON(http.StatusBadRequest, TTSErrorResponse{
			Success: false,
			Error:   "text が不正です",
		})
		return
	}
	textLen := utf8.RuneCountInString(req.Text)
	if textLen > MaxTextLength {
		log.Printf("[TTSHandler] HandleSynthesize failed: text too long, textLen=%d, max=%d", textLen, MaxTextLength)
		c.JSON(http.StatusBadRequest, TTSErrorResponse{
			Success: false,
			Error:   "text が不正です",
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
//
// errors.Is で sentinel を判定し、UpstreamStatusError は errors.As で
// 取り出してステータス値(429 のみ 429、それ以外は 502)で分岐します。
// ラップされたエラーも errors.Is/As で貫通するため、Service 層が
// fmt.Errorf("ctx: %w", err) で包んでも正しく分類されます。
func mapErrorToStatus(err error) (int, string) {
	if errors.Is(err, ErrAPIKeyMissing) {
		return http.StatusServiceUnavailable, "ElevenLabs サービスが利用できません"
	}
	if errors.Is(err, ErrTextEmpty) || errors.Is(err, ErrTextTooLong) {
		return http.StatusBadRequest, "text が不正です"
	}
	if errors.Is(err, ErrUpstreamTimeout) {
		return http.StatusGatewayTimeout, "ElevenLabs リクエストタイムアウト"
	}

	var ue *UpstreamStatusError
	if errors.As(err, &ue) {
		if ue.Status == http.StatusTooManyRequests {
			return http.StatusTooManyRequests, "ElevenLabs レート制限"
		}
		return http.StatusBadGateway, "ElevenLabs から音声を取得できませんでした"
	}

	if errors.Is(err, ErrInvalidContentType) {
		return http.StatusBadGateway, "ElevenLabs から音声を取得できませんでした"
	}

	// その他は 502 に丸める(セキュリティ上、フロントに詳細を出さない)
	return http.StatusBadGateway, "ElevenLabs から音声を取得できませんでした"
}
