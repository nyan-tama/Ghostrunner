// Package handler はHTTPハンドラーを提供します
package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ghostrunner/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// CreateHandler はプロジェクト生成関連のHTTPハンドラを提供します
type CreateHandler struct {
	createService service.CreateProjectService
}

// NewCreateHandler は新しいCreateHandlerを生成します
func NewCreateHandler(createService service.CreateProjectService) *CreateHandler {
	return &CreateHandler{
		createService: createService,
	}
}

// HandleValidate はプロジェクト名のバリデーションを実行します
// GET /api/projects/validate?name={name}
func (h *CreateHandler) HandleValidate(c *gin.Context) {
	name := c.Query("name")

	log.Printf("[CreateHandler] HandleValidate started: name=%s", name)

	result := h.createService.ValidateProjectName(name)

	log.Printf("[CreateHandler] HandleValidate completed: name=%s, valid=%v", name, result.Valid)

	c.JSON(http.StatusOK, result)
}

// CreateStreamRequest はプロジェクト生成のストリーミングリクエストです
type CreateStreamRequest struct {
	Name        string   `json:"name"`        // プロジェクト名
	Description string   `json:"description"` // プロジェクト概要
	Services    []string `json:"services"`    // 選択されたサービス
}

// HandleCreateStream はプロジェクト生成をSSEでストリーミング実行します
// POST /api/projects/create/stream
func (h *CreateHandler) HandleCreateStream(c *gin.Context) {
	var req CreateStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[CreateHandler] HandleCreateStream failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "リクエストが不正です",
		})
		return
	}

	log.Printf("[CreateHandler] HandleCreateStream started: name=%s, services=%v", req.Name, req.Services)

	// プロジェクト名のバリデーション
	result := h.createService.ValidateProjectName(req.Name)
	if !result.Valid {
		log.Printf("[CreateHandler] HandleCreateStream failed: validation error, name=%s, error=%s", req.Name, result.Error)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": result.Error,
		})
		return
	}

	// サービス名のバリデーション
	if err := validateServices(req.Services); err != nil {
		log.Printf("[CreateHandler] HandleCreateStream failed: invalid services, error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// SSEヘッダー設定
	setSSEHeaders(c)

	// イベントチャンネル作成
	eventCh := make(chan service.CreateEvent, 20)

	// プロジェクト生成を開始
	go h.createService.CreateProject(c.Request.Context(), &service.CreateRequest{
		Name:        req.Name,
		Description: req.Description,
		Services:    req.Services,
	}, eventCh)

	// イベントをSSEとして送信
	writeCreateSSEEvents(c, eventCh)

	log.Printf("[CreateHandler] HandleCreateStream completed: name=%s", req.Name)
}

// OpenRequest はプロジェクトをVS Codeで開くリクエストです
type OpenRequest struct {
	Path string `json:"path"` // プロジェクトパス
}

// HandleOpen はプロジェクトをVS Codeで開きます
// POST /api/projects/open
func (h *CreateHandler) HandleOpen(c *gin.Context) {
	var req OpenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[CreateHandler] HandleOpen failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "リクエストが不正です",
		})
		return
	}

	log.Printf("[CreateHandler] HandleOpen started: path=%s", req.Path)

	if req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "pathは必須です",
		})
		return
	}

	// パストラバーサル防止: プロジェクト生成先ディレクトリ配下のみ許可
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[CreateHandler] HandleOpen failed: cannot get home dir, error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "ホームディレクトリの取得に失敗しました",
		})
		return
	}
	cleanPath := filepath.Clean(req.Path)
	if !strings.HasPrefix(cleanPath, homeDir+"/") {
		log.Printf("[CreateHandler] HandleOpen rejected: path=%s is outside home dir", req.Path)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "許可されていないパスです",
		})
		return
	}

	// パスの存在チェック
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		log.Printf("[CreateHandler] HandleOpen failed: path not found, path=%s", req.Path)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "指定されたパスが見つかりません",
		})
		return
	}

	if err := h.createService.OpenInVSCode(cleanPath); err != nil {
		log.Printf("[CreateHandler] HandleOpen failed: path=%s, error=%v", cleanPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "VS Codeの起動に失敗しました",
		})
		return
	}

	log.Printf("[CreateHandler] HandleOpen completed: path=%s", req.Path)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "VS Codeでプロジェクトを開きました",
	})
}

// validateServices はサービス名のリストをバリデーションします
func validateServices(services []string) error {
	for _, svc := range services {
		if !service.AllowedServices[svc] {
			return fmt.Errorf("不明なサービスです: %s", svc)
		}
	}
	return nil
}

// writeCreateSSEEvents はCreateEventチャンネルからイベントを読み取り、SSE形式で送信します
func writeCreateSSEEvents(c *gin.Context, eventCh <-chan service.CreateEvent) {
	w := c.Writer
	flusher, ok := w.(interface{ Flush() })
	if !ok {
		log.Printf("[CreateHandler] ResponseWriter does not support Flush")
		return
	}

	keepalive := time.NewTicker(sseKeepaliveInterval)
	defer keepalive.Stop()

	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[CreateHandler] Client disconnected (context canceled)")
			return

		case event, ok := <-eventCh:
			if !ok {
				log.Printf("[CreateHandler] Event channel closed, stream completed")
				return
			}

			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("[CreateHandler] Marshal error: %v", err)
				continue
			}

			log.Printf("[CreateHandler] SSE sending: type=%s, step=%s, progress=%d", event.Type, event.Step, event.Progress)
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				log.Printf("[CreateHandler] SSE write error: %v", err)
				return
			}
			flusher.Flush()

		case <-keepalive.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				log.Printf("[CreateHandler] Keepalive write error: %v", err)
				return
			}
			flusher.Flush()
		}
	}
}
