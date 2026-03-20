package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache は Redis キャッシュを管理する構造体。
// ローカル Docker Redis と Upstash（サーバーレス Redis）の両方に対応する。
type Cache struct {
	client *redis.Client
}

// NewCache は Redis への接続を確立し、Cache 構造体を返す。
// redisURL は redis:// (ローカル) または rediss:// (Upstash TLS) を受け付ける。
func NewCache(redisURL string) (*Cache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse REDIS_URL: %w", err)
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Cache{client: client}, nil
}

// CacheEntry はキャッシュエントリの情報。
type CacheEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	TTL   int64  `json:"ttl_seconds"`
}

// Set はキーに値を設定する。ttl が 0 の場合は有効期限なし。
func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}
	return nil
}

// Get はキーの値を取得する。キーが存在しない場合は空文字と false を返す。
func (c *Cache) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to get key %s: %w", key, err)
	}
	return val, true, nil
}

// Delete はキーを削除する。
func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

// Keys はパターンに一致するキー一覧を返す。
func (c *Cache) Keys(ctx context.Context, pattern string) ([]CacheEntry, error) {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	entries := make([]CacheEntry, 0, len(keys))
	for _, key := range keys {
		val, err := c.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		ttl, err := c.client.TTL(ctx, key).Result()
		if err != nil {
			ttl = 0
		}
		ttlSeconds := int64(-1)
		if ttl > 0 {
			ttlSeconds = int64(ttl.Seconds())
		}
		entries = append(entries, CacheEntry{
			Key:   key,
			Value: val,
			TTL:   ttlSeconds,
		})
	}
	return entries, nil
}

// Close は Redis 接続を閉じる。
func (c *Cache) Close() error {
	return c.client.Close()
}
