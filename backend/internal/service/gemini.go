// Package service はビジネスロジックを提供します
package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/genai"
)

// GeminiService はGemini API操作のインターフェースを定義します
type GeminiService interface {
	// ProvisionEphemeralToken はGemini Live API用のエフェメラルトークンを発行します
	ProvisionEphemeralToken(expireSeconds int) (*GeminiTokenResult, error)
}

// geminiServiceImpl はGeminiServiceの実装です
type geminiServiceImpl struct {
	client *genai.Client
}

// NewGeminiService は新しいGeminiServiceを生成します
// 環境変数 GEMINI_API_KEY が未設定の場合は nil を返します（オプショナル機能）
func NewGeminiService() GeminiService {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Printf("[GeminiService] GEMINI_API_KEY is not set, Gemini service will not be available")
		return nil
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
		HTTPOptions: genai.HTTPOptions{
			APIVersion: "v1alpha",
		},
	})
	if err != nil {
		log.Printf("[GeminiService] Failed to create client: %v", err)
		return nil
	}

	log.Printf("[GeminiService] Initialized with Google GenAI SDK (v1alpha)")
	return &geminiServiceImpl{
		client: client,
	}
}

// ProvisionEphemeralToken はGemini Live API用のエフェメラルトークンを発行します
func (s *geminiServiceImpl) ProvisionEphemeralToken(expireSeconds int) (*GeminiTokenResult, error) {
	log.Printf("[GeminiService] ProvisionEphemeralToken started: expireSeconds=%d", expireSeconds)

	// expireSecondsのバリデーション: 60〜86400 の範囲外の場合はデフォルト 3600 を使用
	if expireSeconds < 60 || expireSeconds > 86400 {
		log.Printf("[GeminiService] expireSeconds out of range, using default 3600: original=%d", expireSeconds)
		expireSeconds = 3600
	}

	ctx := context.Background()

	// 有効期限を計算
	expireTime := time.Now().Add(time.Duration(expireSeconds) * time.Second)
	newSessionExpireTime := time.Now().Add(1 * time.Minute)

	log.Printf("[GeminiService] ProvisionEphemeralToken: calling SDK Tokens.Create")

	// SDK を使用してエフェメラルトークンを作成
	token, err := s.client.AuthTokens.Create(ctx, &genai.CreateAuthTokenConfig{
		Uses:                 genai.Ptr(int32(1)),
		ExpireTime:           expireTime,
		NewSessionExpireTime: newSessionExpireTime,
	})
	if err != nil {
		log.Printf("[GeminiService] Failed to create auth token: %v", err)
		return nil, fmt.Errorf("failed to provision ephemeral token: %w", err)
	}

	log.Printf("[GeminiService] ProvisionEphemeralToken completed: name=%s", token.Name)

	return &GeminiTokenResult{
		Token:      token.Name,
		ExpireTime: expireTime.Format(time.RFC3339),
	}, nil
}
