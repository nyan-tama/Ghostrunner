package tts

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCache returns an lruCache with injectable clock for testing.
func newTestCache(maxBytes int, ttl time.Duration, clock func() time.Time) *lruCache {
	c := NewLRUCache(maxBytes, ttl)
	if clock != nil {
		c.clock = clock
	}
	return c
}

// ---------------------------------------------------------------------------
// cacheKey
// ---------------------------------------------------------------------------

func TestCacheKey_Determinism(t *testing.T) {
	k1 := cacheKey("hello", 8, "wav")
	k2 := cacheKey("hello", 8, "wav")
	assert.Equal(t, k1, k2, "same input must produce same hash")
}

func TestCacheKey_DifferentInputs(t *testing.T) {
	tests := []struct {
		name string
		a    [3]any // text, speakerID, outputFormat
		b    [3]any
	}{
		{
			name: "different text",
			a:    [3]any{"hello", 8, "wav"},
			b:    [3]any{"world", 8, "wav"},
		},
		{
			name: "different speakerID",
			a:    [3]any{"hello", 8, "wav"},
			b:    [3]any{"hello", 9, "wav"},
		},
		{
			name: "different outputFormat",
			a:    [3]any{"hello", 8, "wav"},
			b:    [3]any{"hello", 8, "mp3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ka := cacheKey(tt.a[0].(string), tt.a[1].(int), tt.a[2].(string))
			kb := cacheKey(tt.b[0].(string), tt.b[1].(int), tt.b[2].(string))
			assert.NotEqual(t, ka, kb)
		})
	}
}

func TestCacheKey_SeparatorCollisionResistance(t *testing.T) {
	// Without null-byte separators, "a" + "bc" and "ab" + "c" could collide.
	k1 := cacheKey("a", 1, "bc")
	k2 := cacheKey("a", 1, "b")
	assert.NotEqual(t, k1, k2, "null-byte boundaries must prevent collision")

	// Another pattern: text containing digits that could merge with speakerID
	k3 := cacheKey("text8", 1, "wav")
	k4 := cacheKey("text", 81, "wav")
	assert.NotEqual(t, k3, k4)
}

// ---------------------------------------------------------------------------
// Set / Get
// ---------------------------------------------------------------------------

func TestLRUCache_SetGet_HitAndMiss(t *testing.T) {
	c := NewLRUCache(1024, time.Hour)

	c.Set("key-a", []byte("hello"))

	data, ok := c.Get("key-a")
	require.True(t, ok, "expected hit")
	assert.Equal(t, []byte("hello"), data)

	_, ok = c.Get("nonexistent")
	assert.False(t, ok, "expected miss for nonexistent key")
}

func TestLRUCache_Set_ZeroByte(t *testing.T) {
	c := NewLRUCache(1024, time.Hour)
	c.Set("empty", []byte{})
	_, ok := c.Get("empty")
	assert.False(t, ok, "zero-byte value should not be stored")
	assert.Equal(t, 0, c.Len())
}

func TestLRUCache_Set_Nil(t *testing.T) {
	c := NewLRUCache(1024, time.Hour)
	c.Set("nil-key", nil)
	_, ok := c.Get("nil-key")
	assert.False(t, ok, "nil value should not be stored")
	assert.Equal(t, 0, c.Bytes())
}

func TestLRUCache_Set_SameKeyOverwrite(t *testing.T) {
	c := NewLRUCache(1024, time.Hour)

	c.Set("k", []byte("aaa"))
	assert.Equal(t, 3, c.Bytes())

	c.Set("k", []byte("bbbbb"))
	assert.Equal(t, 5, c.Bytes(), "bytes should reflect new value size")
	assert.Equal(t, 1, c.Len())

	data, ok := c.Get("k")
	require.True(t, ok)
	assert.Equal(t, []byte("bbbbb"), data)
}

// ---------------------------------------------------------------------------
// LRU eviction
// ---------------------------------------------------------------------------

func TestLRUCache_Eviction_ExceedMaxBytes(t *testing.T) {
	// maxBytes = 10; add entries totaling > 10 bytes
	c := NewLRUCache(10, time.Hour)

	c.Set("a", make([]byte, 4))
	c.Set("b", make([]byte, 4))
	assert.Equal(t, 8, c.Bytes())
	assert.Equal(t, 2, c.Len())

	// This addition pushes total to 12 > 10, so oldest ("a") gets evicted
	c.Set("c", make([]byte, 4))
	assert.LessOrEqual(t, c.Bytes(), 10)
	// "a" should be evicted
	_, ok := c.Get("a")
	assert.False(t, ok, "oldest entry should be evicted")
	// "b" and "c" should remain
	_, ok = c.Get("b")
	assert.True(t, ok)
	_, ok = c.Get("c")
	assert.True(t, ok)
}

func TestLRUCache_ByteTracking(t *testing.T) {
	c := NewLRUCache(1024, time.Hour)

	c.Set("x", make([]byte, 100))
	assert.Equal(t, 100, c.Bytes())

	c.Set("y", make([]byte, 200))
	assert.Equal(t, 300, c.Bytes())

	// Overwrite x with smaller value
	c.Set("x", make([]byte, 50))
	assert.Equal(t, 250, c.Bytes())
}

// ---------------------------------------------------------------------------
// TTL
// ---------------------------------------------------------------------------

func TestLRUCache_TTL(t *testing.T) {
	tests := []struct {
		name        string
		advance     time.Duration
		expectHit   bool
		expectBytes int
	}{
		{
			name:        "within TTL hits",
			advance:     50 * time.Millisecond,
			expectHit:   true,
			expectBytes: 5,
		},
		{
			name:        "after TTL misses and is removed",
			advance:     150 * time.Millisecond,
			expectHit:   false,
			expectBytes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			now := base
			c := newTestCache(1024, 100*time.Millisecond, func() time.Time { return now })

			c.Set("k", []byte("hello"))

			now = base.Add(tt.advance)
			_, ok := c.Get("k")
			assert.Equal(t, tt.expectHit, ok)
			assert.Equal(t, tt.expectBytes, c.Bytes())
		})
	}
}

// ---------------------------------------------------------------------------
// Concurrent access
// ---------------------------------------------------------------------------

func TestLRUCache_Concurrent(t *testing.T) {
	c := NewLRUCache(1024*1024, time.Hour)
	var wg sync.WaitGroup

	const goroutines = 50
	const opsPerGoroutine = 100

	wg.Add(goroutines * 2)

	// Writers
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				key := string(rune('A'+id%26)) + string(rune('0'+i%10))
				c.Set(key, []byte("data"))
			}
		}(g)
	}

	// Readers
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				key := string(rune('A'+id%26)) + string(rune('0'+i%10))
				c.Get(key)
			}
		}(g)
	}

	wg.Wait()
	// No race condition panic = success. Also sanity check.
	assert.GreaterOrEqual(t, c.Len(), 0)
}
