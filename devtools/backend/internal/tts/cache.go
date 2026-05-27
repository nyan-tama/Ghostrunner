package tts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// Cache は TTS 音声バイナリを保持するキャッシュのインターフェースです。
type Cache interface {
	// Get はキーに対応するデータを返します。存在しないか TTL 切れの場合は (nil, false)。
	Get(key string) ([]byte, bool)
	// Set はキーにデータを保存します。バイト数上限を超えた分は LRU で evict されます。
	Set(key string, value []byte)
	// Len は現在のエントリ数を返します。
	Len() int
	// Bytes は現在保持しているバイト数の合計を返します。
	Bytes() int
}

// cacheEntry は LRU に格納する 1 エントリです。
type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

// lruCache は LRU + TTL + バイト数上限のキャッシュ実装です。
// hashicorp/golang-lru/v2 自体はスレッドセーフですが、バイト数計算と
// 組み合わせるため明示的に sync.Mutex を取ります。
type lruCache struct {
	mu        sync.Mutex
	lru       *lru.Cache[string, cacheEntry]
	maxBytes  int
	currBytes int
	ttl       time.Duration
	clock     func() time.Time // テスト用注入
}

// NewLRUCache は新しい lruCache を生成します。
// maxBytes はバイト数上限、ttl はエントリの有効期限です。
//
// hashicorp/golang-lru の容量(エントリ数)は内部実装の都合上、十分大きい
// 値を渡しておき、バイト数管理を主軸に evict を制御します。
//
// arenaSize (1<<20) は意図的に大きな定数で、実容量は maxBytes で制御します。
// lru.New はサイズが正でない場合のみ失敗しますが、本関数ではあり得ないため、
// 想定外のエラー時は将来のリグレッション検知を兼ねて panic させます。
func NewLRUCache(maxBytes int, ttl time.Duration) *lruCache {
	// エントリ数上限は実質無制限に近い値にする(バイト数で制御するため)
	// 最低 1 は必要だが、極端に大きい値もメモリ消費に寄与しないため十分大きい値を採用
	const arenaSize = 1 << 20 // 1,048,576 エントリ
	c, err := lru.New[string, cacheEntry](arenaSize)
	if err != nil {
		// lru.New はサイズ 0 以下でのみ失敗する。ここに来たらプログラミングエラー。
		panic(fmt.Sprintf("tts: NewLRUCache: lru.New unexpectedly failed (arenaSize=%d): %v", arenaSize, err))
	}
	return &lruCache{
		lru:      c,
		maxBytes: maxBytes,
		ttl:      ttl,
		clock:    time.Now,
	}
}

// Get はキーに対応するデータを返します。TTL 切れの場合は削除して miss 扱いにします。
func (c *lruCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.lru.Get(key)
	if !ok {
		return nil, false
	}

	if c.clock().After(entry.expiresAt) {
		// 期限切れ: 削除して miss を返す
		if c.lru.Remove(key) {
			c.currBytes -= len(entry.data)
		}
		return nil, false
	}

	return entry.data, true
}

// Set はキーにデータを保存します。バイト数上限を超えた分は LRU で evict します。
// 単一エントリが maxBytes を超える場合はエントリを保存しません。
func (c *lruCache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	size := len(value)
	if size == 0 {
		return
	}

	// 単一エントリが上限を超える場合は保存しない(無限ループ防止)
	if size > c.maxBytes {
		log.Printf("[TTSService] cache: entry too large, skipped: keyPrefix=%s, size=%d, maxBytes=%d",
			keyPrefix(key), size, c.maxBytes)
		return
	}

	// 既存エントリの上書き分を currBytes から差し引く
	if existing, ok := c.lru.Peek(key); ok {
		c.currBytes -= len(existing.data)
	}

	// 新規エントリ追加分を加算
	c.currBytes += size

	// 上限超過分を最古から evict
	for c.currBytes > c.maxBytes {
		evictedKey, evictedEntry, ok := c.lru.RemoveOldest()
		if !ok {
			break
		}
		c.currBytes -= len(evictedEntry.data)
		log.Printf("[TTSService] cache evicted: keyPrefix=%s, bytes=%d, currBytes=%d",
			keyPrefix(evictedKey), len(evictedEntry.data), c.currBytes)
	}

	expiresAt := c.clock().Add(c.ttl)
	c.lru.Add(key, cacheEntry{data: value, expiresAt: expiresAt})
}

// Len は現在のエントリ数を返します。
func (c *lruCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lru.Len()
}

// Bytes は現在保持しているバイト数の合計を返します。
func (c *lruCache) Bytes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currBytes
}

// cacheKey は (text, voiceID, modelID, outputFormat) を SHA256 して hex 化したキーです。
// 区切り文字 \x00 を挟むことでフィールド境界の衝突を防ぎます。
func cacheKey(text, voiceID, modelID, outputFormat string) string {
	h := sha256.New()
	h.Write([]byte(text))
	h.Write([]byte{0x00})
	h.Write([]byte(voiceID))
	h.Write([]byte{0x00})
	h.Write([]byte(modelID))
	h.Write([]byte{0x00})
	h.Write([]byte(outputFormat))
	return hex.EncodeToString(h.Sum(nil))
}

// keyPrefix はログ出力用に key の先頭 16 文字を返します。
func keyPrefix(key string) string {
	const n = 16
	if len(key) <= n {
		return key
	}
	return key[:n]
}
