package tts

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"

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
	client           clientInterface
	cache            Cache
	sfGroup          singleflight.Group
	defaultSpeakerID int
}

// NewService は環境変数を読んで Service を生成します。
// VOICEVOX は API キー不要のため、常に非 nil を返します。
//
// 環境変数:
//   - VOICEVOX_HOST: VOICEVOX Engine のアドレス(デフォルト: http://localhost:50021)
//   - VOICEVOX_SPEAKER_ID: 話者ID(デフォルト: 8 = 春日部つむぎ)
func NewService() Service {
	host := os.Getenv("VOICEVOX_HOST")
	if host == "" {
		host = DefaultBaseURL
	}

	speakerID := DefaultSpeakerID
	if raw := os.Getenv("VOICEVOX_SPEAKER_ID"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			log.Printf("[TTSService] VOICEVOX_SPEAKER_ID parse failed, using default: raw=%q, error=%v", raw, err)
		} else {
			speakerID = parsed
		}
	}

	cfg := Config{
		Host:      host,
		SpeakerID: speakerID,
	}

	log.Printf("[TTSService] Initialized: host=%s, speakerID=%d, cacheMaxBytes=%d, cacheTTL=%s",
		host, speakerID, CacheMaxBytes, CacheTTL)

	return &serviceImpl{
		client:           NewClient(cfg, 0),
		cache:            NewLRUCache(CacheMaxBytes, CacheTTL),
		defaultSpeakerID: speakerID,
	}
}

// Synthesize は params を正規化し、キャッシュ -> singleflight -> client の順で音声を取得します。
func (s *serviceImpl) Synthesize(ctx context.Context, params SynthesizeParams) (*SynthesizeResult, error) {
	normalized := s.normalize(params)

	key := cacheKey(normalized.Text, normalized.SpeakerID, normalized.OutputFormat)

	// キャッシュ参照
	if data, ok := s.cache.Get(key); ok {
		log.Printf("[TTSService] cache hit: keyPrefix=%s, bytes=%d", keyPrefix(key), len(data))
		return &SynthesizeResult{
			Audio:       data,
			ContentType: "audio/wav",
			FromCache:   true,
		}, nil
	}

	log.Printf("[TTSService] cache miss: keyPrefix=%s, calling VOICEVOX", keyPrefix(key))

	// singleflight で重複呼出を統合
	v, err, _ := s.sfGroup.Do(key, func() (any, error) {
		audio, callErr := s.client.Synthesize(ctx, normalized)
		if callErr != nil {
			return nil, callErr
		}
		s.cache.Set(key, audio)
		return audio, nil
	})

	if err != nil {
		log.Printf("[TTSService] VOICEVOX failed: error=%v", err)
		return nil, err
	}

	audio, ok := v.([]byte)
	if !ok {
		return nil, errors.New("unexpected singleflight value type")
	}

	return &SynthesizeResult{
		Audio:       audio,
		ContentType: "audio/wav",
		FromCache:   false,
	}, nil
}

// normalize は空フィールドをデフォルト値で埋めます。
func (s *serviceImpl) normalize(params SynthesizeParams) SynthesizeParams {
	out := params
	if out.SpeakerID == 0 {
		out.SpeakerID = s.defaultSpeakerID
	}
	out.OutputFormat = DefaultOutputFormat
	return out
}
