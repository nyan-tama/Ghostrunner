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
	// openAIClientSecretsURL はOpenAI Realtime API GA版のclient_secrets作成エンドポイント
	openAIClientSecretsURL = "https://api.openai.com/v1/realtime/client_secrets"

	// デフォルト値（GA版）
	defaultOpenAIModel = "gpt-realtime"
	defaultOpenAIVoice = "verse"

	// デフォルトの有効期限（秒）
	defaultExpiresAfterSeconds = 600
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

// openAIClientSecretsRequest はGA版client_secrets作成リクエストの構造体
type openAIClientSecretsRequest struct {
	ExpiresAfter openAIExpiresAfter       `json:"expires_after"`
	Session      openAISessionConfig      `json:"session"`
}

// openAIExpiresAfter は有効期限の設定
type openAIExpiresAfter struct {
	Anchor  string `json:"anchor"`
	Seconds int    `json:"seconds"`
}

// openAISessionConfig はセッション設定
type openAISessionConfig struct {
	Type  string              `json:"type"`
	Model string              `json:"model"`
	Audio *openAIAudioConfig  `json:"audio,omitempty"`
}

// openAIAudioConfig は音声設定
type openAIAudioConfig struct {
	Output *openAIAudioOutput `json:"output,omitempty"`
}

// openAIAudioOutput は音声出力設定
type openAIAudioOutput struct {
	Voice string `json:"voice,omitempty"`
}

// openAIClientSecretsResponse はGA版client_secrets作成レスポンスの構造体
type openAIClientSecretsResponse struct {
	Value     string `json:"value"`
	ExpiresAt int64  `json:"expires_at"`
	Error     *openAIErrorResponse `json:"error,omitempty"`
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

	// GA版リクエストボディを作成
	reqBody := openAIClientSecretsRequest{
		ExpiresAfter: openAIExpiresAfter{
			Anchor:  "created_at",
			Seconds: defaultExpiresAfterSeconds,
		},
		Session: openAISessionConfig{
			Type:  "realtime",
			Model: model,
			Audio: &openAIAudioConfig{
				Output: &openAIAudioOutput{
					Voice: voice,
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("[OpenAIService] CreateRealtimeSession failed: failed to marshal request body, error=%v", err)
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// HTTPリクエストを作成
	req, err := http.NewRequest(http.MethodPost, openAIClientSecretsURL, bytes.NewReader(jsonBody))
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
	var secretsResp openAIClientSecretsResponse
	if err := json.Unmarshal(body, &secretsResp); err != nil {
		log.Printf("[OpenAIService] CreateRealtimeSession failed: failed to parse response, error=%v", err)
		return nil, fmt.Errorf("failed to parse OpenAI API response: %w", err)
	}

	// value を抽出
	if secretsResp.Value == "" {
		log.Printf("[OpenAIService] CreateRealtimeSession failed: value is empty")
		return nil, fmt.Errorf("OpenAI API returned empty client_secret")
	}

	// 有効期限をISO8601形式に変換
	expireTime := time.Unix(secretsResp.ExpiresAt, 0).UTC().Format(time.RFC3339)

	log.Printf("[OpenAIService] CreateRealtimeSession completed: token=%s..., expireTime=%s", secretsResp.Value[:10], expireTime)

	return &OpenAISessionResult{
		Token:      secretsResp.Value,
		ExpireTime: expireTime,
	}, nil
}
