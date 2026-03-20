package handler

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"{{PROJECT_NAME}}/backend/internal/infrastructure"
)

// StorageHandler はオブジェクトストレージ用のハンドラー。
type StorageHandler struct {
	storage *infrastructure.Storage
}

// NewStorageHandler は StorageHandler を生成する。
func NewStorageHandler(storage *infrastructure.Storage) *StorageHandler {
	return &StorageHandler{storage: storage}
}

// Upload はファイルをアップロードする。
// POST /api/storage/upload (multipart/form-data)
func (h *StorageHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	baseName := header.Filename[:len(header.Filename)-len(ext)]
	key := fmt.Sprintf("%s_%d%s", baseName, time.Now().UnixMilli(), ext)

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := h.storage.Upload(c.Request.Context(), key, file, contentType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key":           key,
		"original_name": header.Filename,
		"size":          header.Size,
		"content_type":  contentType,
	})
}

// List はファイル一覧を取得する。
// GET /api/storage/files
func (h *StorageHandler) List(c *gin.Context) {
	prefix := c.Query("prefix")

	files, err := h.storage.List(c.Request.Context(), prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list files"})
		return
	}

	c.JSON(http.StatusOK, files)
}

// Download はファイルをダウンロードする。
// GET /api/storage/files/:key
func (h *StorageHandler) Download(c *gin.Context) {
	key := c.Param("key")

	body, contentType, err := h.storage.Download(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	defer body.Close()

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", key))
	io.Copy(c.Writer, body)
}

// Delete はファイルを削除する。
// DELETE /api/storage/files/:key
func (h *StorageHandler) Delete(c *gin.Context) {
	key := c.Param("key")

	if err := h.storage.Delete(c.Request.Context(), key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file deleted"})
}
