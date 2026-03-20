// Package handler はHTTPハンドラーを提供します
package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"ghostrunner/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// PatrolHandler は巡回関連のHTTPハンドラを提供します
type PatrolHandler struct {
	patrolService service.PatrolService
}

// NewPatrolHandler は新しいPatrolHandlerを生成します
func NewPatrolHandler(patrolService service.PatrolService) *PatrolHandler {
	return &PatrolHandler{
		patrolService: patrolService,
	}
}

// PatrolRegisterRequest はプロジェクト登録リクエストです
type PatrolRegisterRequest struct {
	Path string `json:"path"` // プロジェクトの絶対パス
}

// PatrolRemoveRequest はプロジェクト解除リクエストです
type PatrolRemoveRequest struct {
	Path string `json:"path"` // プロジェクトの絶対パス
}

// PatrolResumeRequest は承認待ちプロジェクト再開リクエストです
type PatrolResumeRequest struct {
	ProjectPath string `json:"projectPath"` // プロジェクトのパス
	Answer      string `json:"answer"`      // ユーザーの回答
}

// PatrolResponse は巡回APIの共通レスポンスです
type PatrolResponse struct {
	Success bool   `json:"success"`         // 成功フラグ
	Error   string `json:"error,omitempty"` // エラーメッセージ
}

// PatrolProjectsResponse はプロジェクト一覧レスポンスです
type PatrolProjectsResponse struct {
	Success  bool                    `json:"success"`            // 成功フラグ
	Projects []service.PatrolProject `json:"projects,omitempty"` // プロジェクト一覧
}

// PatrolStatesResponse はプロジェクト状態レスポンスです
type PatrolStatesResponse struct {
	Success bool                             `json:"success"`          // 成功フラグ
	States  map[string]*service.ProjectState `json:"states,omitempty"` // プロジェクト状態
}

// PatrolScanResponse はスキャン結果レスポンスです
type PatrolScanResponse struct {
	Success bool                 `json:"success"`           // 成功フラグ
	Results []service.ScanResult `json:"results,omitempty"` // スキャン結果
}

// HandleRegister はプロジェクト登録を処理します
// POST /api/patrol/projects
func (h *PatrolHandler) HandleRegister(c *gin.Context) {
	var req PatrolRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[PatrolHandler] HandleRegister failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, PatrolResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	log.Printf("[PatrolHandler] HandleRegister started: path=%s", req.Path)

	if req.Path == "" {
		c.JSON(http.StatusBadRequest, PatrolResponse{
			Success: false,
			Error:   "pathは必須です",
		})
		return
	}

	if err := h.patrolService.RegisterProject(req.Path); err != nil {
		log.Printf("[PatrolHandler] HandleRegister failed: path=%s, error=%v", req.Path, err)
		c.JSON(http.StatusBadRequest, PatrolResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	log.Printf("[PatrolHandler] HandleRegister completed: path=%s", req.Path)

	c.JSON(http.StatusOK, PatrolResponse{Success: true})
}

// HandleRemove はプロジェクト解除を処理します
// POST /api/patrol/projects/remove
func (h *PatrolHandler) HandleRemove(c *gin.Context) {
	var req PatrolRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[PatrolHandler] HandleRemove failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, PatrolResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	log.Printf("[PatrolHandler] HandleRemove started: path=%s", req.Path)

	if req.Path == "" {
		c.JSON(http.StatusBadRequest, PatrolResponse{
			Success: false,
			Error:   "pathは必須です",
		})
		return
	}

	if err := h.patrolService.UnregisterProject(req.Path); err != nil {
		log.Printf("[PatrolHandler] HandleRemove failed: path=%s, error=%v", req.Path, err)
		c.JSON(http.StatusBadRequest, PatrolResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	log.Printf("[PatrolHandler] HandleRemove completed: path=%s", req.Path)

	c.JSON(http.StatusOK, PatrolResponse{Success: true})
}

// HandleListProjects は登録済みプロジェクト一覧を返します
// GET /api/patrol/projects
func (h *PatrolHandler) HandleListProjects(c *gin.Context) {
	log.Printf("[PatrolHandler] HandleListProjects started")

	projects := h.patrolService.ListProjects()

	log.Printf("[PatrolHandler] HandleListProjects completed: count=%d", len(projects))

	c.JSON(http.StatusOK, PatrolProjectsResponse{
		Success:  true,
		Projects: projects,
	})
}

// HandleScan はプロジェクトスキャンを実行します
// GET /api/patrol/scan
func (h *PatrolHandler) HandleScan(c *gin.Context) {
	log.Printf("[PatrolHandler] HandleScan started")

	results := h.patrolService.ScanProjects()

	log.Printf("[PatrolHandler] HandleScan completed: count=%d", len(results))

	c.JSON(http.StatusOK, PatrolScanResponse{
		Success: true,
		Results: results,
	})
}

// HandleStart は巡回を開始します
// POST /api/patrol/start
func (h *PatrolHandler) HandleStart(c *gin.Context) {
	log.Printf("[PatrolHandler] HandleStart started")

	if err := h.patrolService.StartPatrol(); err != nil {
		log.Printf("[PatrolHandler] HandleStart failed: error=%v", err)
		c.JSON(http.StatusConflict, PatrolResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	log.Printf("[PatrolHandler] HandleStart completed")

	c.JSON(http.StatusOK, PatrolResponse{Success: true})
}

// HandleStop は巡回を停止します
// POST /api/patrol/stop
func (h *PatrolHandler) HandleStop(c *gin.Context) {
	log.Printf("[PatrolHandler] HandleStop started")

	h.patrolService.StopPatrol()

	log.Printf("[PatrolHandler] HandleStop completed")

	c.JSON(http.StatusOK, PatrolResponse{Success: true})
}

// HandleResume は承認待ちプロジェクトを再開します
// POST /api/patrol/resume
func (h *PatrolHandler) HandleResume(c *gin.Context) {
	var req PatrolResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[PatrolHandler] HandleResume failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, PatrolResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	log.Printf("[PatrolHandler] HandleResume started: projectPath=%s", req.ProjectPath)

	if req.ProjectPath == "" {
		c.JSON(http.StatusBadRequest, PatrolResponse{
			Success: false,
			Error:   "projectPathは必須です",
		})
		return
	}

	if req.Answer == "" {
		c.JSON(http.StatusBadRequest, PatrolResponse{
			Success: false,
			Error:   "answerは必須です",
		})
		return
	}

	if err := h.patrolService.ResumeProject(req.ProjectPath, req.Answer); err != nil {
		log.Printf("[PatrolHandler] HandleResume failed: projectPath=%s, error=%v", req.ProjectPath, err)
		c.JSON(http.StatusBadRequest, PatrolResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	log.Printf("[PatrolHandler] HandleResume completed: projectPath=%s", req.ProjectPath)

	c.JSON(http.StatusOK, PatrolResponse{Success: true})
}

// HandleStates は全プロジェクトの実行状態を返します
// GET /api/patrol/states
func (h *PatrolHandler) HandleStates(c *gin.Context) {
	log.Printf("[PatrolHandler] HandleStates started")

	states := h.patrolService.GetStates()

	log.Printf("[PatrolHandler] HandleStates completed: count=%d", len(states))

	c.JSON(http.StatusOK, PatrolStatesResponse{
		Success: true,
		States:  states,
	})
}

// HandleStream はSSEストリーミングを提供します
// GET /api/patrol/stream
func (h *PatrolHandler) HandleStream(c *gin.Context) {
	log.Printf("[PatrolHandler] HandleStream started")

	// SSEヘッダー設定
	setSSEHeaders(c)

	// サブスクリプション取得
	eventCh, unsubscribe := h.patrolService.Subscribe()
	defer unsubscribe()

	// SSEイベントを送信
	writePatrolSSEEvents(c, eventCh)

	log.Printf("[PatrolHandler] HandleStream completed")
}

// HandlePollingStart はポーリングを開始します
// POST /api/patrol/polling/start
func (h *PatrolHandler) HandlePollingStart(c *gin.Context) {
	log.Printf("[PatrolHandler] HandlePollingStart started")

	h.patrolService.StartPolling()

	log.Printf("[PatrolHandler] HandlePollingStart completed")

	c.JSON(http.StatusOK, PatrolResponse{Success: true})
}

// HandlePollingStop はポーリングを停止します
// POST /api/patrol/polling/stop
func (h *PatrolHandler) HandlePollingStop(c *gin.Context) {
	log.Printf("[PatrolHandler] HandlePollingStop started")

	h.patrolService.StopPolling()

	log.Printf("[PatrolHandler] HandlePollingStop completed")

	c.JSON(http.StatusOK, PatrolResponse{Success: true})
}

// writePatrolSSEEvents はPatrolEventチャンネルからイベントを読み取り、SSE形式で送信します
func writePatrolSSEEvents(c *gin.Context, eventCh <-chan service.PatrolEvent) {
	w := c.Writer
	flusher, ok := w.(interface{ Flush() })
	if !ok {
		log.Printf("[PatrolHandler] ResponseWriter does not support Flush")
		return
	}

	keepalive := time.NewTicker(sseKeepaliveInterval)
	defer keepalive.Stop()

	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[PatrolHandler] Client disconnected (context canceled)")
			return

		case event, ok := <-eventCh:
			if !ok {
				log.Printf("[PatrolHandler] Event channel closed, stream completed")
				return
			}

			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("[PatrolHandler] Marshal error: %v", err)
				continue
			}

			log.Printf("[PatrolHandler] SSE sending: type=%s, projectPath=%s", event.Type, event.ProjectPath)
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				log.Printf("[PatrolHandler] SSE write error (client disconnected): %v", err)
				return
			}
			flusher.Flush()

		case <-keepalive.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				log.Printf("[PatrolHandler] Keepalive write error (client disconnected): %v", err)
				return
			}
			flusher.Flush()
		}
	}
}
