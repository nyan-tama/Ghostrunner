// Package service はビジネスロジックを提供します
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// GeminiService はGemini API操作のインターフェースを定義します
type GeminiService interface {
	// ProvisionEphemeralToken はGemini Live API用のエフェメラルトークンを発行します
	ProvisionEphemeralToken(expireSeconds int) (*GeminiTokenResult, error)
}

// geminiServiceImpl はGeminiServiceの実装です
type geminiServiceImpl struct {
	apiKey     string
	httpClient *http.Client
}

// NewGeminiService は新しいGeminiServiceを生成します
// 環境変数 GEMINI_API_KEY が未設定の場合は nil を返します（オプショナル機能）
func NewGeminiService() GeminiService {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Printf("[GeminiService] GEMINI_API_KEY is not set, Gemini service will not be available")
		return nil
	}

	log.Printf("[GeminiService] Initialized with API key")
	return &geminiServiceImpl{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// provisionEphemeralTokenURL はエフェメラルトークン発行APIのURL
const provisionEphemeralTokenURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-live-001:provisionEphemeralToken"

// provisionEphemeralTokenRequest はトークン発行APIのリクエストボディ
type provisionEphemeralTokenRequest struct {
	ExpireTime expireTimeValue `json:"expire_time"`
}

// expireTimeValue はexpire_timeフィールドの値
type expireTimeValue struct {
	Seconds int `json:"seconds"`
}

// provisionEphemeralTokenResponse はトークン発行APIのレスポンス
type provisionEphemeralTokenResponse struct {
	EphemeralToken string `json:"ephemeralToken"`
	ExpireTime     string `json:"expireTime"`
}

// ProvisionEphemeralToken はGemini Live API用のエフェメラルトークンを発行します
func (s *geminiServiceImpl) ProvisionEphemeralToken(expireSeconds int) (*GeminiTokenResult, error) {
	log.Printf("[GeminiService] ProvisionEphemeralToken started: expireSeconds=%d", expireSeconds)

	// expireSecondsのバリデーション: 60〜86400 の範囲外の場合はデフォルト 3600 を使用
	if expireSeconds < 60 || expireSeconds > 86400 {
		log.Printf("[GeminiService] expireSeconds out of range, using default 3600: original=%d", expireSeconds)
		expireSeconds = 3600
	}

	// リクエストボディを構築
	reqBody := provisionEphemeralTokenRequest{
		ExpireTime: expireTimeValue{
			Seconds: expireSeconds,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	log.Printf("[GeminiService] ProvisionEphemeralToken: calling API")

	// HTTPリクエストを作成
	req, err := http.NewRequest(http.MethodPost, provisionEphemeralTokenURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", s.apiKey)

	// APIを呼び出し
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	// レスポンスボディを読み取り
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// ステータスコードをチェック
	if resp.StatusCode != http.StatusOK {
		log.Printf("[GeminiService] API returned non-200 status: status=%d, body=%s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("failed to provision ephemeral token: API returned status %d", resp.StatusCode)
	}

	// レスポンスをパース
	var tokenResp provisionEphemeralTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	log.Printf("[GeminiService] ProvisionEphemeralToken completed: expireTime=%s", tokenResp.ExpireTime)

	return &GeminiTokenResult{
		Token:      tokenResp.EphemeralToken,
		ExpireTime: tokenResp.ExpireTime,
	}, nil
}
