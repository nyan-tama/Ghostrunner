package tts

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// newTestCache は clock を注入可能なテスト用 lruCache を返します。
func newTestCache(maxBytes int, ttl time.Duration, clock func() time.Time) *lruCache {
	c := NewLRUCache(maxBytes, ttl)
	if clock != nil {
		c.clock = clock
	}
	return c
}

// --- Set/Get の基本動作 ---

func TestLRUCache_SetGet_HitAndMiss(t *testing.T) {
	c := NewLRUCache(1024, time.Hour)

	c.Set("key-a", []byte("hello"))

	data, ok := c.Get("key-a")
	if !ok {
		t.Fatalf("expected hit for key-a, got miss")
	}
	if string(data) != "hello" {
		t.Errorf("expected data=hello, got %q", string(data))
	}

	if _, ok := c.Get("nonexistent"); ok {
		t.Errorf("expected miss for nonexistent key, got hit")
	}
}

// --- TTL ---

func TestLRUCache_TTL(t *testing.T) {
	tests := []struct {
		name        string
		advance     time.Duration
		expectHit   bool
		expectBytes int
	}{
		{
			name:        "within TTL hits",
			advance:     23 * time.Hour,
			expectHit:   true,
			expectBytes: 5,
		},
		{
			name:        "after TTL misses and is removed",
			advance:     24*time.Hour + time.Second,
			expectHit:   false,
			expectBytes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			now := base
			c := newTestCache(1024, 24*time.Hour, func() time.Time { return now })

			c.Set("k", []byte("hello"))

			now = base.Add(tt.advance)
			_, ok := c.Get("k")
			if ok != tt.expectHit {
				t.Errorf("expected hit=%v, got %v", tt.expectHit, ok)
			}
			if got := c.Bytes(); got != tt.expectBytes {
				t.Errorf("expected Bytes()=%d, got %d", tt.expectBytes, got)
			}
		})
	}
}

// --- LRU エビクション(バイト数上限) ---

func TestLRUCache_EvictionByByteLimit(t *testing.T) {
	// maxBytes=100、各エントリ30バイト → 4 件目で最古が evict される
	c := NewLRUCache(100, time.Hour)

	c.Set("a", make([]byte, 30))
	c.Set("b", make([]byte, 30))
	c.Set("c", make([]byte, 30))
	if got := c.Bytes(); got != 90 {
		t.Errorf("expected Bytes()=90, got %d", got)
	}

	c.Set("d", make([]byte, 30))

	if got := c.Bytes(); got > 100 {
		t.Errorf("expected Bytes()<=100, got %d", got)
	}
	if _, ok := c.Get("a"); ok {
		t.Errorf("expected a to be evicted, but found")
	}
	for _, k := range []string{"b", "c", "d"} {
		if _, ok := c.Get(k); !ok {
			t.Errorf("expected %s still cached", k)
		}
	}
}

// --- 単一エントリが maxBytes を超える ---

func TestLRUCache_OversizeEntryIsRejected(t *testing.T) {
	c := NewLRUCache(50, time.Hour)

	c.Set("huge", make([]byte, 100))

	if got := c.Bytes(); got != 0 {
		t.Errorf("expected Bytes()=0 (oversize entry rejected), got %d", got)
	}
	if got := c.Len(); got != 0 {
		t.Errorf("expected Len()=0, got %d", got)
	}
	if _, ok := c.Get("huge"); ok {
		t.Errorf("expected miss for oversize key, got hit")
	}
}

// --- LRU recency 更新(Get が最後アクセス時刻を進める) ---

func TestLRUCache_GetRefreshesRecency(t *testing.T) {
	c := NewLRUCache(90, time.Hour)

	// A, B, C を 30B ずつ投入(合計 90)
	c.Set("a", make([]byte, 30))
	c.Set("b", make([]byte, 30))
	c.Set("c", make([]byte, 30))

	// A を Get して最近アクセスに更新
	if _, ok := c.Get("a"); !ok {
		t.Fatalf("expected a to be cached")
	}

	// 新規 D を投入 → 最古は B のはず
	c.Set("d", make([]byte, 30))

	if _, ok := c.Get("a"); !ok {
		t.Errorf("expected a still cached (recency refreshed)")
	}
	if _, ok := c.Get("b"); ok {
		t.Errorf("expected b evicted as least recently used")
	}
	if _, ok := c.Get("c"); !ok {
		t.Errorf("expected c still cached")
	}
	if _, ok := c.Get("d"); !ok {
		t.Errorf("expected d cached")
	}
}

// --- 同キーへの再 Set: 古いバイト数を差し引いて新バイト数を加算 ---

func TestLRUCache_ResetOnSameKey(t *testing.T) {
	c := NewLRUCache(1024, time.Hour)

	c.Set("k", make([]byte, 10))
	if got := c.Bytes(); got != 10 {
		t.Fatalf("expected Bytes()=10, got %d", got)
	}

	c.Set("k", make([]byte, 30))
	if got := c.Bytes(); got != 30 {
		t.Errorf("expected Bytes()=30 after overwrite, got %d", got)
	}
	if got := c.Len(); got != 1 {
		t.Errorf("expected Len()=1, got %d", got)
	}
}

// --- Len()/Bytes() の正しさ ---

func TestLRUCache_LenAndBytes(t *testing.T) {
	c := NewLRUCache(1024, time.Hour)
	if c.Len() != 0 || c.Bytes() != 0 {
		t.Fatalf("expected empty cache, got Len=%d Bytes=%d", c.Len(), c.Bytes())
	}

	c.Set("a", make([]byte, 10))
	c.Set("b", make([]byte, 20))
	c.Set("c", make([]byte, 30))

	if got := c.Len(); got != 3 {
		t.Errorf("expected Len()=3, got %d", got)
	}
	if got := c.Bytes(); got != 60 {
		t.Errorf("expected Bytes()=60, got %d", got)
	}
}

// --- 並行 Get/Set ---

func TestLRUCache_ConcurrentSafe(t *testing.T) {
	c := NewLRUCache(10*1024*1024, time.Hour)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("k%d", i%10)
			c.Set(key, []byte(fmt.Sprintf("data-%d", i)))
			_, _ = c.Get(key)
			_ = c.Len()
			_ = c.Bytes()
		}(i)
	}
	wg.Wait()
}

// --- cacheKey: 決定性 ---

func TestCacheKey_Deterministic(t *testing.T) {
	k1 := cacheKey("あ", "v", "m", "f")
	k2 := cacheKey("あ", "v", "m", "f")
	if k1 != k2 {
		t.Errorf("expected same key for same inputs, got %q vs %q", k1, k2)
	}
	if len(k1) != 64 {
		t.Errorf("expected sha256 hex (64 chars), got %d chars", len(k1))
	}
}

// --- cacheKey: 衝突回避(\x00 区切り) ---

func TestCacheKey_BoundaryCollisionAvoidance(t *testing.T) {
	// text="a"+voice="bc" と text="ab"+voice="c" を同一キーにしてはいけない
	k1 := cacheKey("a", "bc", "m", "f")
	k2 := cacheKey("ab", "c", "m", "f")
	if k1 == k2 {
		t.Errorf("expected different keys, got same: %q", k1)
	}
}

// --- cacheKey: 各フィールドの寄与 ---

func TestCacheKey_FieldsAreDistinct(t *testing.T) {
	base := cacheKey("text", "voice", "model", "format")

	cases := []struct {
		name string
		key  string
	}{
		{"text changes", cacheKey("TEXT", "voice", "model", "format")},
		{"voice changes", cacheKey("text", "VOICE", "model", "format")},
		{"model changes", cacheKey("text", "voice", "MODEL", "format")},
		{"format changes", cacheKey("text", "voice", "model", "FORMAT")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.key == base {
				t.Errorf("expected different key when %s, got same: %q", tc.name, tc.key)
			}
		})
	}
}
