package tts

import (
	"context"
	"errors"
	"log"
	"os"

	"golang.org/x/sync/singleflight"
)

// Service は TTS 合成のビジネスロジック層インターフェースです。
type Service interface {
	// Synthesize は音声を合成して返します。キャッシュヒット時は即返却します。
	Synthesize(ctx context.Context, params SynthesizeParams) (*SynthesizeResult, error)
}

// clientInterface は service がテストで client を差し替えるための非公開インターフェースです。
// 本物の *Client は自動的に満たすため production コードの変更は不要です。
type clientInterface interface {
	Synthesize(ctx context.Context, params SynthesizeParams) ([]byte, error)
}

// serviceImpl は Service の実装です。
type serviceImpl struct {
	client  clientInterface
	cache   Cache
	sfGroup *singleflight.Group
	cfg     Config
}

// NewService は環境変数を読んで Service を生成します。
// ELEVENLABS_API_KEY 未設定時は WARN ログ後 nil を返します
// (既存 OpenAIService と同型のオプショナル機能パターン)。
func NewService() Service {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" {
		log.Printf("[TTSService] ELEVENLABS_API_KEY is not set, TTS service will not be available")
		return nil
	}

	voiceID := os.Getenv("ELEVENLABS_DEFAULT_VOICE_ID")
	if voiceID == "" {
		log.Printf("[TTSService] ELEVENLABS_DEFAULT_VOICE_ID is not set, using fallback voiceID=%s", DefaultVoiceID)
		voiceID = DefaultVoiceID
	}

	modelID := os.Getenv("ELEVENLABS_DEFAULT_MODEL")
	if modelID == "" {
		log.Printf("[TTSService] ELEVENLABS_DEFAULT_MODEL is not set, using fallback modelID=%s", DefaultModelID)
		modelID = DefaultModelID
	}

	cfg := Config{
		APIKey:         apiKey,
		DefaultVoiceID: voiceID,
		DefaultModelID: modelID,
		BaseURL:        DefaultBaseURL,
	}

	log.Printf("[TTSService] Initialized: voiceID=%s, modelID=%s, cacheMaxBytes=%d, cacheTTL=%s",
		voiceID, modelID, CacheMaxBytes, CacheTTL)

	return &serviceImpl{
		client:  NewClient(cfg),
		cache:   NewLRUCache(CacheMaxBytes, CacheTTL),
		sfGroup: &singleflight.Group{},
		cfg:     cfg,
	}
}

// Synthesize は params を正規化し、キャッシュ→singleflight→client の順で音声を取得します。
func (s *serviceImpl) Synthesize(ctx context.Context, params SynthesizeParams) (*SynthesizeResult, error) {
	// パラメータ正規化(空フィールドを env デフォルトで埋める)
	normalized := s.normalize(params)

	if normalized.Text == "" {
		return nil, ErrTextEmpty
	}

	key := cacheKey(normalized.Text, normalized.VoiceID, normalized.ModelID, normalized.OutputFormat)

	// キャッシュ参照
	if data, ok := s.cache.Get(key); ok {
		log.Printf("[TTSService] cache hit: keyPrefix=%s, bytes=%d", keyPrefix(key), len(data))
		return &SynthesizeResult{
			Audio:       data,
			ContentType: "audio/mpeg",
			FromCache:   true,
		}, nil
	}

	log.Printf("[TTSService] cache miss: keyPrefix=%s, calling ElevenLabs", keyPrefix(key))

	// singleflight で重複呼出を統合
	v, err, _ := s.sfGroup.Do(key, func() (any, error) {
		audio, callErr := s.client.Synthesize(ctx, normalized)
		if callErr != nil {
			return nil, callErr
		}
		// エラーなしの時のみキャッシュ書き込み
		s.cache.Set(key, audio)
		return audio, nil
	})

	if err != nil {
		log.Printf("[TTSService] ElevenLabs failed: error=%v", err)
		return nil, err
	}

	audio, ok := v.([]byte)
	if !ok {
		// 起こり得ないが防御的に
		return nil, errors.New("unexpected singleflight value type")
	}

	return &SynthesizeResult{
		Audio:       audio,
		ContentType: "audio/mpeg",
		FromCache:   false,
	}, nil
}

// normalize は空フィールドを env デフォルトで埋め、OutputFormat を常に固定値にします。
func (s *serviceImpl) normalize(params SynthesizeParams) SynthesizeParams {
	out := params
	if out.VoiceID == "" {
		out.VoiceID = s.cfg.DefaultVoiceID
	}
	if out.ModelID == "" {
		out.ModelID = s.cfg.DefaultModelID
	}
	out.OutputFormat = DefaultOutputFormat
	return out
}
