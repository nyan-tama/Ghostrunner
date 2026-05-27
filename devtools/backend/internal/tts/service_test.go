package tts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/singleflight"
)

// --- モック client ---

type mockClient struct {
	mu             sync.Mutex
	synthesizeFunc func(ctx context.Context, params SynthesizeParams) ([]byte, error)
	callCount      int
	lastParams     SynthesizeParams
	receivedParams []SynthesizeParams
}

func (m *mockClient) Synthesize(ctx context.Context, params SynthesizeParams) ([]byte, error) {
	m.mu.Lock()
	m.callCount++
	m.lastParams = params
	m.receivedParams = append(m.receivedParams, params)
	fn := m.synthesizeFunc
	m.mu.Unlock()

	if fn != nil {
		return fn(ctx, params)
	}
	return []byte("audio-data"), nil
}

func (m *mockClient) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// newTestService は注入可能な serviceImpl をテスト用に生成します。
func newTestService(t *testing.T, mc *mockClient) *serviceImpl {
	t.Helper()
	return &serviceImpl{
		client:  mc,
		cache:   NewLRUCache(10*1024*1024, time.Hour),
		sfGroup: &singleflight.Group{},
		cfg: Config{
			APIKey:         "test-api-key",
			DefaultVoiceID: "default-voice",
			DefaultModelID: "default-model",
			BaseURL:        DefaultBaseURL,
		},
	}
}

// --- cache miss → client → cache 格納 ---

func TestService_Synthesize_CacheMiss(t *testing.T) {
	mc := &mockClient{
		synthesizeFunc: func(ctx context.Context, params SynthesizeParams) ([]byte, error) {
			return []byte("audio-1"), nil
		},
	}
	svc := newTestService(t, mc)

	result, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FromCache {
		t.Errorf("expected FromCache=false on first call")
	}
	if string(result.Audio) != "audio-1" {
		t.Errorf("unexpected audio: %q", string(result.Audio))
	}
	if mc.Calls() != 1 {
		t.Errorf("expected client called 1 time, got %d", mc.Calls())
	}
	if result.ContentType != "audio/mpeg" {
		t.Errorf("expected ContentType=audio/mpeg, got %q", result.ContentType)
	}
}

// --- cache hit: 2 回目は client を呼ばない ---

func TestService_Synthesize_CacheHit(t *testing.T) {
	mc := &mockClient{
		synthesizeFunc: func(ctx context.Context, params SynthesizeParams) ([]byte, error) {
			return []byte("audio-1"), nil
		},
	}
	svc := newTestService(t, mc)

	// 初回
	_, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 回目
	result, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.FromCache {
		t.Errorf("expected FromCache=true on second call")
	}
	if mc.Calls() != 1 {
		t.Errorf("expected client called only once, got %d", mc.Calls())
	}
	if string(result.Audio) != "audio-1" {
		t.Errorf("unexpected cached audio: %q", string(result.Audio))
	}
}

// --- singleflight: 同一キーの並行呼出は 1 回に統合 ---

func TestService_Synthesize_SingleflightDeduplication(t *testing.T) {
	var callCount int32
	mc := &mockClient{
		synthesizeFunc: func(ctx context.Context, params SynthesizeParams) ([]byte, error) {
			atomic.AddInt32(&callCount, 1)
			time.Sleep(50 * time.Millisecond)
			return []byte("audio-data"), nil
		},
	}
	svc := newTestService(t, mc)

	const N = 10
	var wg sync.WaitGroup
	results := make([][]byte, N)
	errs := make([]error, N)

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "same"})
			if err != nil {
				errs[i] = err
				return
			}
			results[i] = r.Audio
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Errorf("expected client called 1 time (singleflight), got %d", got)
	}
	for i := 0; i < N; i++ {
		if errs[i] != nil {
			t.Errorf("goroutine %d got error: %v", i, errs[i])
			continue
		}
		if string(results[i]) != "audio-data" {
			t.Errorf("goroutine %d got unexpected audio: %q", i, string(results[i]))
		}
	}
}

// --- singleflight: 異なるキーは並行に呼出される ---

func TestService_Synthesize_DifferentKeysParallel(t *testing.T) {
	mc := &mockClient{
		synthesizeFunc: func(ctx context.Context, params SynthesizeParams) ([]byte, error) {
			return []byte("audio-for-" + params.Text), nil
		},
	}
	svc := newTestService(t, mc)

	var wg sync.WaitGroup
	for _, text := range []string{"text-1", "text-2"} {
		wg.Add(1)
		go func(text string) {
			defer wg.Done()
			_, _ = svc.Synthesize(context.Background(), SynthesizeParams{Text: text})
		}(text)
	}
	wg.Wait()

	if mc.Calls() != 2 {
		t.Errorf("expected client called 2 times for distinct keys, got %d", mc.Calls())
	}
}

// --- OutputFormat 自動補完 ---

func TestService_Synthesize_OutputFormatNormalization(t *testing.T) {
	mc := &mockClient{}
	svc := newTestService(t, mc)

	_, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "x", OutputFormat: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mc.lastParams.OutputFormat != DefaultOutputFormat {
		t.Errorf("expected OutputFormat=%q, got %q", DefaultOutputFormat, mc.lastParams.OutputFormat)
	}
}

// --- VoiceID/ModelID env デフォルト補完 ---

func TestService_Synthesize_VoiceAndModelDefaults(t *testing.T) {
	mc := &mockClient{}
	svc := newTestService(t, mc)

	_, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mc.lastParams.VoiceID != "default-voice" {
		t.Errorf("expected VoiceID=default-voice, got %q", mc.lastParams.VoiceID)
	}
	if mc.lastParams.ModelID != "default-model" {
		t.Errorf("expected ModelID=default-model, got %q", mc.lastParams.ModelID)
	}

	// 同じ text を再度 → cache hit になり、cacheKey が同じである(VoiceID/ModelID 補完が決定的)
	mc.callCount = 0
	r, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.FromCache {
		t.Errorf("expected FromCache=true on second call (key stable across normalize)")
	}
	if mc.Calls() != 0 {
		t.Errorf("expected client not called on second call, got %d", mc.Calls())
	}
}

// --- text 空 → ErrTextEmpty(VoiceID 補完前判定なので env デフォルト埋めても空のまま) ---

func TestService_Synthesize_EmptyTextReturnsError(t *testing.T) {
	mc := &mockClient{}
	svc := newTestService(t, mc)

	_, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: ""})
	if !errors.Is(err, ErrTextEmpty) {
		t.Errorf("expected ErrTextEmpty, got %v", err)
	}
	if mc.Calls() != 0 {
		t.Errorf("expected client not called on empty text, got %d", mc.Calls())
	}
}

// --- client エラー時はキャッシュに格納しない ---

func TestService_Synthesize_ClientErrorNotCached(t *testing.T) {
	upstreamErr := &UpstreamStatusError{Status: 500, Body: "server error"}
	mc := &mockClient{
		synthesizeFunc: func(ctx context.Context, params SynthesizeParams) ([]byte, error) {
			return nil, upstreamErr
		},
	}
	svc := newTestService(t, mc)

	_, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "fail"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// 2 回目: client がもう一度呼ばれること
	_, err = svc.Synthesize(context.Background(), SynthesizeParams{Text: "fail"})
	if err == nil {
		t.Fatalf("expected error on retry, got nil")
	}
	if mc.Calls() != 2 {
		t.Errorf("expected client retried, got calls=%d", mc.Calls())
	}
}

// --- client エラーのパススルー(UpstreamStatusError, ErrUpstreamTimeout, ErrInvalidContentType) ---

func TestService_Synthesize_ErrorPassthrough(t *testing.T) {
	tests := []struct {
		name      string
		clientErr error
		assertion func(t *testing.T, err error)
	}{
		{
			name:      "UpstreamStatusError",
			clientErr: &UpstreamStatusError{Status: 429, Body: "rate limit"},
			assertion: func(t *testing.T, err error) {
				var ue *UpstreamStatusError
				if !errors.As(err, &ue) {
					t.Errorf("expected *UpstreamStatusError, got %v", err)
					return
				}
				if ue.Status != 429 {
					t.Errorf("expected Status=429, got %d", ue.Status)
				}
			},
		},
		{
			name:      "ErrUpstreamTimeout",
			clientErr: fmt.Errorf("%w: timeout", ErrUpstreamTimeout),
			assertion: func(t *testing.T, err error) {
				if !errors.Is(err, ErrUpstreamTimeout) {
					t.Errorf("expected errors.Is(ErrUpstreamTimeout), got %v", err)
				}
			},
		},
		{
			name:      "ErrInvalidContentType",
			clientErr: ErrInvalidContentType,
			assertion: func(t *testing.T, err error) {
				if !errors.Is(err, ErrInvalidContentType) {
					t.Errorf("expected errors.Is(ErrInvalidContentType), got %v", err)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clientErr := tc.clientErr
			mc := &mockClient{
				synthesizeFunc: func(ctx context.Context, params SynthesizeParams) ([]byte, error) {
					return nil, clientErr
				},
			}
			svc := newTestService(t, mc)
			_, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "x"})
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			tc.assertion(t, err)
		})
	}
}

// --- NewService: API キー未設定で nil ---

func TestNewService_NilWhenAPIKeyMissing(t *testing.T) {
	// env を一時的にクリア
	orig := os.Getenv("ELEVENLABS_API_KEY")
	if err := os.Unsetenv("ELEVENLABS_API_KEY"); err != nil {
		t.Fatalf("unsetenv: %v", err)
	}
	defer func() {
		if orig != "" {
			_ = os.Setenv("ELEVENLABS_API_KEY", orig)
		}
	}()

	if svc := NewService(); svc != nil {
		t.Errorf("expected NewService() to return nil when ELEVENLABS_API_KEY unset, got %v", svc)
	}
}
