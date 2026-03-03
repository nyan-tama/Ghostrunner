package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"{{PROJECT_NAME}}/backend/internal/infrastructure"
)

// CacheHandler は Redis キャッシュ用のハンドラー。
type CacheHandler struct {
	cache *infrastructure.Cache
}

// NewCacheHandler は CacheHandler を生成する。
func NewCacheHandler(cache *infrastructure.Cache) *CacheHandler {
	return &CacheHandler{cache: cache}
}

type setCacheRequest struct {
	Key        string `json:"key" binding:"required"`
	Value      string `json:"value" binding:"required"`
	TTLSeconds int    `json:"ttl_seconds"`
}

// Set はキーに値を設定する。
// POST /api/cache
func (h *CacheHandler) Set(c *gin.Context) {
	var req setCacheRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key and value are required"})
		return
	}

	ttl := time.Duration(0)
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds) * time.Second
	}

	if err := h.cache.Set(c.Request.Context(), req.Key, req.Value, ttl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set cache"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key":         req.Key,
		"value":       req.Value,
		"ttl_seconds": req.TTLSeconds,
	})
}

// Get はキーの値を取得する。
// GET /api/cache/:key
func (h *CacheHandler) Get(c *gin.Context) {
	key := c.Param("key")

	value, exists, err := h.cache.Get(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get cache"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":   key,
		"value": value,
	})
}

// List はキー一覧を取得する。
// GET /api/cache
func (h *CacheHandler) List(c *gin.Context) {
	pattern := c.DefaultQuery("pattern", "*")

	entries, err := h.cache.Keys(c.Request.Context(), pattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list keys"})
		return
	}

	c.JSON(http.StatusOK, entries)
}

// Delete はキーを削除する。
// DELETE /api/cache/:key
func (h *CacheHandler) Delete(c *gin.Context) {
	key := c.Param("key")

	if err := h.cache.Delete(c.Request.Context(), key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete cache"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "key deleted"})
}
