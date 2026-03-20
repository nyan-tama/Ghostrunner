// Package handler はHTTPハンドラーを提供します
package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"ghostrunner/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// PlanRequest は/api/planリクエストの構造体です
type PlanRequest struct {
	Project string `json:"project"` // プロジェクトのパス
	Args    string `json:"args"`    // /planコマンドの引数
}

// ContinueRequest は/api/plan/continueリクエストの構造体です
type ContinueRequest struct {
	Project   string `json:"project"`    // プロジェクトのパス
	SessionID string `json:"session_id"` // セッションID
	Answer    string `json:"answer"`     // ユーザーの回答
}

// PlanResponse は/api/planレスポンスの構造体です
type PlanResponse struct {
	Success   bool               `json:"success"`              // 成功フラグ
	SessionID string             `json:"session_id,omitempty"` // セッションID
	Output    string             `json:"output,omitempty"`     // 実行結果
	Questions []service.Question `json:"questions,omitempty"`  // 質問がある場合
	Completed bool               `json:"completed"`            // 完了したかどうか
	CostUSD   float64            `json:"cost_usd,omitempty"`   // コスト
	Error     string             `json:"error,omitempty"`      // エラーメッセージ
}

// PlanHandler はPlan関連のHTTPハンドラを提供します
type PlanHandler struct {
	claudeService service.ClaudeService
}

// NewPlanHandler は新しいPlanHandlerを生成します
func NewPlanHandler(claudeService service.ClaudeService) *PlanHandler {
	return &PlanHandler{
		claudeService: claudeService,
	}
}

// Handle は/api/planリクエストを処理します
// POST /api/plan
func (h *PlanHandler) Handle(c *gin.Context) {
	var req PlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[PlanHandler] Handle failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	log.Printf("[PlanHandler] Handle started: project=%s, args=%s", req.Project, req.Args)

	// プロジェクトパスのバリデーション
	if err := validateProjectPath(req.Project); err != nil {
		log.Printf("[PlanHandler] Handle failed: invalid project path, project=%s, error=%v", req.Project, err)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// argsのバリデーション
	if req.Args == "" {
		log.Printf("[PlanHandler] Handle failed: args is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   "argsは必須です",
		})
		return
	}

	// Claude CLIを実行
	result, err := h.claudeService.ExecutePlan(c.Request.Context(), req.Project, req.Args)
	if err != nil {
		log.Printf("[PlanHandler] Handle failed: project=%s, args=%s, error=%v", req.Project, req.Args, err)
		c.JSON(http.StatusInternalServerError, PlanResponse{
			Success: false,
			Error:   "Claude CLI実行に失敗しました: " + err.Error(),
		})
		return
	}

	log.Printf("[PlanHandler] Handle completed: project=%s, args=%s, sessionID=%s, questions=%d, completed=%v",
		req.Project, req.Args, result.SessionID, len(result.Questions), result.Completed)

	c.JSON(http.StatusOK, PlanResponse{
		Success:   true,
		SessionID: result.SessionID,
		Output:    result.Output,
		Questions: result.Questions,
		Completed: result.Completed,
		CostUSD:   result.CostUSD,
	})
}

// HandleStream は/api/plan/streamリクエストを処理します（Server-Sent Events）
// POST /api/plan/stream
func (h *PlanHandler) HandleStream(c *gin.Context) {
	var req PlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[PlanHandler] HandleStream failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	log.Printf("[PlanHandler] HandleStream started: project=%s, args=%s", req.Project, req.Args)

	// プロジェクトパスのバリデーション
	if err := validateProjectPath(req.Project); err != nil {
		log.Printf("[PlanHandler] HandleStream failed: invalid project path, project=%s, error=%v", req.Project, err)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// argsのバリデーション
	if req.Args == "" {
		log.Printf("[PlanHandler] HandleStream failed: args is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   "argsは必須です",
		})
		return
	}

	// SSEヘッダー設定
	setSSEHeaders(c)

	// イベントチャンネル作成
	eventCh := make(chan service.StreamEvent, 100)

	// ストリーミング実行を開始
	go func() {
		err := h.claudeService.ExecutePlanStream(c.Request.Context(), req.Project, req.Args, eventCh)
		if err != nil {
			log.Printf("[PlanHandler] HandleStream error: %v", err)
		}
	}()

	// イベントをSSEとして送信（selectベースのループで確実にコンテキストキャンセルを検出）
	writeSSEEvents(c, eventCh, "PlanHandler")

	log.Printf("[PlanHandler] HandleStream completed: project=%s, args=%s", req.Project, req.Args)
}

// HandleContinue は/api/plan/continueリクエストを処理します
// POST /api/plan/continue
func (h *PlanHandler) HandleContinue(c *gin.Context) {
	var req ContinueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[PlanHandler] HandleContinue failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	log.Printf("[PlanHandler] HandleContinue started: project=%s, sessionID=%s, answer=%s", req.Project, req.SessionID, req.Answer)

	// プロジェクトパスのバリデーション
	if err := validateProjectPath(req.Project); err != nil {
		log.Printf("[PlanHandler] HandleContinue failed: invalid project path, project=%s, error=%v", req.Project, err)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// セッションIDのバリデーション
	if req.SessionID == "" {
		log.Printf("[PlanHandler] HandleContinue failed: sessionID is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   "session_idは必須です",
		})
		return
	}

	// 回答のバリデーション
	if req.Answer == "" {
		log.Printf("[PlanHandler] HandleContinue failed: answer is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   "answerは必須です",
		})
		return
	}

	// セッションを継続
	result, err := h.claudeService.ContinueSession(c.Request.Context(), req.Project, req.SessionID, req.Answer)
	if err != nil {
		log.Printf("[PlanHandler] HandleContinue failed: project=%s, sessionID=%s, error=%v", req.Project, req.SessionID, err)
		c.JSON(http.StatusInternalServerError, PlanResponse{
			Success: false,
			Error:   "セッション継続に失敗しました: " + err.Error(),
		})
		return
	}

	log.Printf("[PlanHandler] HandleContinue completed: project=%s, sessionID=%s, questions=%d, completed=%v",
		req.Project, result.SessionID, len(result.Questions), result.Completed)

	c.JSON(http.StatusOK, PlanResponse{
		Success:   true,
		SessionID: result.SessionID,
		Output:    result.Output,
		Questions: result.Questions,
		Completed: result.Completed,
		CostUSD:   result.CostUSD,
	})
}

// HandleContinueStream は/api/plan/continue/streamリクエストを処理します（Server-Sent Events）
// POST /api/plan/continue/stream
func (h *PlanHandler) HandleContinueStream(c *gin.Context) {
	var req ContinueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[PlanHandler] HandleContinueStream failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	log.Printf("[PlanHandler] HandleContinueStream started: project=%s, sessionID=%s", req.Project, req.SessionID)

	// プロジェクトパスのバリデーション
	if err := validateProjectPath(req.Project); err != nil {
		log.Printf("[PlanHandler] HandleContinueStream failed: invalid project path, project=%s, error=%v", req.Project, err)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// セッションIDのバリデーション
	if req.SessionID == "" {
		log.Printf("[PlanHandler] HandleContinueStream failed: sessionID is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   "session_idは必須です",
		})
		return
	}

	// 回答のバリデーション
	if req.Answer == "" {
		log.Printf("[PlanHandler] HandleContinueStream failed: answer is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success: false,
			Error:   "answerは必須です",
		})
		return
	}

	// SSEヘッダー設定
	setSSEHeaders(c)

	// イベントチャンネル作成
	eventCh := make(chan service.StreamEvent, 100)

	// ストリーミング実行を開始
	go func() {
		err := h.claudeService.ContinueSessionStream(c.Request.Context(), req.Project, req.SessionID, req.Answer, eventCh)
		if err != nil {
			log.Printf("[PlanHandler] HandleContinueStream error: %v", err)
		}
	}()

	// イベントをSSEとして送信（selectベースのループで確実にコンテキストキャンセルを検出）
	writeSSEEvents(c, eventCh, "PlanHandler")

	log.Printf("[PlanHandler] HandleContinueStream completed: project=%s, sessionID=%s", req.Project, req.SessionID)
}

// validateProjectPath はプロジェクトパスをバリデーションします
func validateProjectPath(path string) error {
	if path == "" {
		return &ValidationError{Message: "projectは必須です"}
	}

	// パスをクリーンアップ
	cleanPath := filepath.Clean(path)

	// 絶対パスであることを確認
	if !filepath.IsAbs(cleanPath) {
		return &ValidationError{Message: "projectは絶対パスである必要があります"}
	}

	// ディレクトリが存在することを確認
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ValidationError{Message: "指定されたプロジェクトパスが存在しません"}
		}
		return &ValidationError{Message: "プロジェクトパスの確認に失敗しました"}
	}

	// ディレクトリであることを確認
	if !info.IsDir() {
		return &ValidationError{Message: "projectはディレクトリである必要があります"}
	}

	return nil
}

// ValidationError はバリデーションエラーを表します
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
