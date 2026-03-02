package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"{{PROJECT_NAME}}/backend/internal/domain/model"
	"{{PROJECT_NAME}}/backend/internal/infrastructure"
)

// CreateSampleRequest はサンプル作成リクエスト。
type CreateSampleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateSampleRequest はサンプル更新リクエスト。
type UpdateSampleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SampleHandler はサンプルCRUD用のハンドラー。
// MVPではservice層を省略し、handlerがDatabaseを直接使用する。
// プロジェクトが成長した段階で internal/service/ 層を導入する。
type SampleHandler struct {
	database *infrastructure.Database
}

// NewSampleHandler は SampleHandler を生成する。
func NewSampleHandler(db *infrastructure.Database) *SampleHandler {
	return &SampleHandler{database: db}
}

// List は全サンプルを取得する。
// GET /api/samples
func (h *SampleHandler) List(c *gin.Context) {
	var samples []model.Sample
	if err := h.database.DB().Find(&samples).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch samples"})
		return
	}
	c.JSON(http.StatusOK, samples)
}

// Get は指定IDのサンプルを取得する。
// GET /api/samples/:id
func (h *SampleHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var sample model.Sample
	if err := h.database.DB().First(&sample, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sample not found"})
		return
	}
	c.JSON(http.StatusOK, sample)
}

// Create は新しいサンプルを作成する。
// POST /api/samples
func (h *SampleHandler) Create(c *gin.Context) {
	var req CreateSampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sample := model.Sample{
		Name:        req.Name,
		Description: req.Description,
	}
	if err := h.database.DB().Create(&sample).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create sample"})
		return
	}
	c.JSON(http.StatusCreated, sample)
}

// Update は指定IDのサンプルを更新する。
// PUT /api/samples/:id
func (h *SampleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var sample model.Sample
	if err := h.database.DB().First(&sample, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sample not found"})
		return
	}

	var req UpdateSampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		sample.Name = req.Name
	}
	if req.Description != "" {
		sample.Description = req.Description
	}

	if err := h.database.DB().Save(&sample).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update sample"})
		return
	}
	c.JSON(http.StatusOK, sample)
}

// Delete は指定IDのサンプルを削除する。
// DELETE /api/samples/:id
func (h *SampleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	var sample model.Sample
	if err := h.database.DB().First(&sample, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sample not found"})
		return
	}

	if err := h.database.DB().Delete(&sample).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete sample"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sample deleted"})
}
