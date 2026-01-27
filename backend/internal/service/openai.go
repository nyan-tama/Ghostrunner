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

const (
	// openAIRealtimeSessionsURL はOpenAI Realtime APIのセッション作成エンドポイント
	openAIRealtimeSessionsURL = "https://api.openai.com/v1/realtime/sessions"

	// デフォルト値
	defaultOpenAIModel = "gpt-4o-realtime-preview-2024-12-17"
	defaultOpenAIVoice = "verse"
)

// OpenAIService はOpenAI Realtime API操作のインターフェースを定義します
type OpenAIService interface {
	// CreateRealtimeSession はOpenAI Realtime API用のエフェメラルキーを発行します
	CreateRealtimeSession(model, voice string) (*OpenAISessionResult, error)
}

// openaiServiceImpl はOpenAIServiceの実装です
type openaiServiceImpl struct {
	apiKey     string
	httpClient *http.Client
}

// NewOpenAIService は新しいOpenAIServiceを生成します
// 環境変数 OPENAI_API_KEY が未設定の場合は nil を返します（オプショナル機能）
func NewOpenAIService() OpenAIService {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Printf("[OpenAIService] OPENAI_API_KEY is not set, OpenAI service will not be available")
		return nil
	}

	log.Printf("[OpenAIService] Initialized with OpenAI API")
	return &openaiServiceImpl{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// openAISessionRequest はセッション作成リクエストの構造体
type openAISessionRequest struct {
	Model string `json:"model"`
	Voice string `json:"voice"`
}

// openAISessionResponse はセッション作成レスポンスの構造体
type openAISessionResponse struct {
	ID           string                    `json:"id"`
	Object       string                    `json:"object"`
	Model        string                    `json:"model"`
	ExpiresAt    int64                     `json:"expires_at"`
	ClientSecret openAISessionClientSecret `json:"client_secret"`
	Error        *openAIErrorResponse      `json:"error,omitempty"`
}

// openAISessionClientSecret はclient_secretフィールドの構造体
type openAISessionClientSecret struct {
	Value     string `json:"value"`
	ExpiresAt int64  `json:"expires_at"`
}

// openAIErrorResponse はOpenAI APIのエラーレスポンス
type openAIErrorResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// CreateRealtimeSession はOpenAI Realtime API用のエフェメラルキーを発行します
func (s *openaiServiceImpl) CreateRealtimeSession(model, voice string) (*OpenAISessionResult, error) {
	log.Printf("[OpenAIService] CreateRealtimeSession started: model=%s, voice=%s", model, voice)

	// デフォルト値の設定
	if model == "" {
		model = defaultOpenAIModel
	}
	if voice == "" {
		voice = defaultOpenAIVoice
	}

	// リクエストボディを作成
	reqBody := openAISessionRequest{
		Model: model,
		Voice: voice,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("[OpenAIService] CreateRealtimeSession failed: failed to marshal request body, error=%v", err)
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// HTTPリクエストを作成
	req, err := http.NewRequest(http.MethodPost, openAIRealtimeSessionsURL, bytes.NewReader(jsonBody))
	if err != nil {
		log.Printf("[OpenAIService] CreateRealtimeSession failed: failed to create request, error=%v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	// リクエストを送信
	log.Printf("[OpenAIService] CreateRealtimeSession: sending request to OpenAI API")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[OpenAIService] CreateRealtimeSession failed: failed to send request, error=%v", err)
		return nil, fmt.Errorf("failed to send request to OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	// レスポンスボディを読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[OpenAIService] CreateRealtimeSession failed: failed to read response body, error=%v", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// エラーレスポンスのチェック
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error openAIErrorResponse `json:"error"`
		}
		if parseErr := json.Unmarshal(body, &errResp); parseErr == nil && errResp.Error.Message != "" {
			log.Printf("[OpenAIService] CreateRealtimeSession failed: OpenAI API error, status=%d, message=%s, type=%s, code=%s",
				resp.StatusCode, errResp.Error.Message, errResp.Error.Type, errResp.Error.Code)
			return nil, fmt.Errorf("OpenAI API error: %s", errResp.Error.Message)
		}
		log.Printf("[OpenAIService] CreateRealtimeSession failed: OpenAI API returned error, status=%d, body=%s",
			resp.StatusCode, string(body))
		return nil, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	// レスポンスをパース
	var sessionResp openAISessionResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		log.Printf("[OpenAIService] CreateRealtimeSession failed: failed to parse response, error=%v", err)
		return nil, fmt.Errorf("failed to parse OpenAI API response: %w", err)
	}

	// client_secret.value を抽出
	if sessionResp.ClientSecret.Value == "" {
		log.Printf("[OpenAIService] CreateRealtimeSession failed: client_secret.value is empty")
		return nil, fmt.Errorf("OpenAI API returned empty client_secret")
	}

	// 有効期限をISO8601形式に変換
	expireTime := time.Unix(sessionResp.ClientSecret.ExpiresAt, 0).UTC().Format(time.RFC3339)

	log.Printf("[OpenAIService] CreateRealtimeSession completed: id=%s, expireTime=%s", sessionResp.ID, expireTime)

	return &OpenAISessionResult{
		Token:      sessionResp.ClientSecret.Value,
		ExpireTime: expireTime,
	}, nil
}
