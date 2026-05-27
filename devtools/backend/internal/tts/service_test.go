package tts

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient records calls and returns configured responses.
type mockClient struct {
	mu        sync.Mutex
	calls     []SynthesizeParams
	audio     []byte
	err       error
	callCount atomic.Int32
	delay     time.Duration
}

func (m *mockClient) Synthesize(_ context.Context, params SynthesizeParams) ([]byte, error) {
	m.callCount.Add(1)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	m.calls = append(m.calls, params)
	m.mu.Unlock()
	return m.audio, m.err
}

// newTestService builds a serviceImpl with injected mock client and cache.
func newTestService(client *mockClient, speakerID int) *serviceImpl {
	if speakerID == 0 {
		speakerID = DefaultSpeakerID
	}
	return &serviceImpl{
		client:           client,
		cache:            NewLRUCache(CacheMaxBytes, CacheTTL),
		defaultSpeakerID: speakerID,
	}
}

// ---------------------------------------------------------------------------
// NewService env defaults
// ---------------------------------------------------------------------------

func TestNewService_EnvDefaults(t *testing.T) {
	// Clear env to test defaults
	os.Unsetenv("VOICEVOX_HOST")
	os.Unsetenv("VOICEVOX_SPEAKER_ID")

	svc := NewService()
	impl, ok := svc.(*serviceImpl)
	require.True(t, ok)
	assert.Equal(t, DefaultSpeakerID, impl.defaultSpeakerID)
}

func TestNewService_EnvHost(t *testing.T) {
	t.Setenv("VOICEVOX_HOST", "http://custom:9999")
	t.Setenv("VOICEVOX_SPEAKER_ID", "")

	svc := NewService()
	impl := svc.(*serviceImpl)
	// The client should use the custom host. We verify indirectly by checking
	// the client was created (non-nil).
	assert.NotNil(t, impl.client)
	assert.Equal(t, DefaultSpeakerID, impl.defaultSpeakerID)
}

func TestNewService_EnvSpeakerID(t *testing.T) {
	t.Setenv("VOICEVOX_HOST", "")
	t.Setenv("VOICEVOX_SPEAKER_ID", "3")

	svc := NewService()
	impl := svc.(*serviceImpl)
	assert.Equal(t, 3, impl.defaultSpeakerID)
}

func TestNewService_EnvSpeakerID_ParseFailure(t *testing.T) {
	t.Setenv("VOICEVOX_SPEAKER_ID", "not-a-number")

	svc := NewService()
	impl := svc.(*serviceImpl)
	assert.Equal(t, DefaultSpeakerID, impl.defaultSpeakerID, "parse failure should fall back to default")
}

// ---------------------------------------------------------------------------
// Synthesize: cache behavior
// ---------------------------------------------------------------------------

func TestService_Synthesize_CacheMiss(t *testing.T) {
	mc := &mockClient{audio: []byte("wav-data")}
	svc := newTestService(mc, 0)

	result, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "hello"})
	require.NoError(t, err)
	assert.False(t, result.FromCache)
	assert.Equal(t, []byte("wav-data"), result.Audio)
	assert.Equal(t, "audio/wav", result.ContentType)
	assert.Equal(t, int32(1), mc.callCount.Load(), "client should be called on cache miss")
}

func TestService_Synthesize_CacheHit(t *testing.T) {
	mc := &mockClient{audio: []byte("wav-data")}
	svc := newTestService(mc, 0)

	// First call: cache miss
	_, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "hello"})
	require.NoError(t, err)

	// Second call: cache hit
	result, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "hello"})
	require.NoError(t, err)
	assert.True(t, result.FromCache)
	assert.Equal(t, []byte("wav-data"), result.Audio)
	assert.Equal(t, int32(1), mc.callCount.Load(), "client should NOT be called on cache hit")
}

func TestService_Synthesize_ClientError_NotCached(t *testing.T) {
	mc := &mockClient{err: ErrUpstreamTimeout}
	svc := newTestService(mc, 0)

	// First call: error
	_, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "hello"})
	require.Error(t, err)

	// Fix the client
	mc.err = nil
	mc.audio = []byte("recovered")

	// Second call: should hit client again (not cached)
	result, err := svc.Synthesize(context.Background(), SynthesizeParams{Text: "hello"})
	require.NoError(t, err)
	assert.False(t, result.FromCache)
	assert.Equal(t, int32(2), mc.callCount.Load())
}

// ---------------------------------------------------------------------------
// Synthesize: singleflight
// ---------------------------------------------------------------------------

func TestService_Synthesize_Singleflight_SameKey(t *testing.T) {
	mc := &mockClient{
		audio: []byte("wav"),
		delay: 100 * time.Millisecond,
	}
	svc := newTestService(mc, 0)

	const concurrency = 10
	var wg sync.WaitGroup
	wg.Add(concurrency)

	errs := make([]error, concurrency)
	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = svc.Synthesize(context.Background(), SynthesizeParams{Text: "same"})
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "goroutine %d", i)
	}
	// With singleflight, the client should be called only once
	assert.Equal(t, int32(1), mc.callCount.Load(), "singleflight should deduplicate concurrent same-key calls")
}

func TestService_Synthesize_Singleflight_DifferentKeys(t *testing.T) {
	mc := &mockClient{
		audio: []byte("wav"),
		delay: 50 * time.Millisecond,
	}
	svc := newTestService(mc, 0)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = svc.Synthesize(context.Background(), SynthesizeParams{Text: "text-A"})
	}()
	go func() {
		defer wg.Done()
		_, _ = svc.Synthesize(context.Background(), SynthesizeParams{Text: "text-B"})
	}()
	wg.Wait()

	assert.Equal(t, int32(2), mc.callCount.Load(), "different keys should NOT be merged in singleflight")
}

// ---------------------------------------------------------------------------
// Synthesize: normalize defaults
// ---------------------------------------------------------------------------

func TestService_Synthesize_SpeakerIDZero_UsesDefault(t *testing.T) {
	mc := &mockClient{audio: []byte("wav")}
	svc := newTestService(mc, 42)

	_, err := svc.Synthesize(context.Background(), SynthesizeParams{
		Text:      "test",
		SpeakerID: 0,
	})
	require.NoError(t, err)

	mc.mu.Lock()
	defer mc.mu.Unlock()
	require.Len(t, mc.calls, 1)
	assert.Equal(t, 42, mc.calls[0].SpeakerID, "SpeakerID=0 should be normalized to defaultSpeakerID")
}

func TestService_Synthesize_OutputFormat_DefaultsToWav(t *testing.T) {
	mc := &mockClient{audio: []byte("wav")}
	svc := newTestService(mc, 0)

	_, err := svc.Synthesize(context.Background(), SynthesizeParams{
		Text:         "test",
		OutputFormat: "",
	})
	require.NoError(t, err)

	mc.mu.Lock()
	defer mc.mu.Unlock()
	require.Len(t, mc.calls, 1)
	assert.Equal(t, "wav", mc.calls[0].OutputFormat, "empty OutputFormat should default to wav")
}
