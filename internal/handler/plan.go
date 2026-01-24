// Package handler はHTTPハンドラーを提供します
package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"ghostrunner/internal/service"

	"github.com/gin-gonic/gin"
)

// PlanRequest は/api/planリクエストの構造体です
type PlanRequest struct {
	Project string `json:"project"` // プロジェクトのパス
	Args    string `json:"args"`    // /planコマンドの引数
}

// PlanResponse は/api/planレスポンスの構造体です
type PlanResponse struct {
	Success bool   `json:"success"`          // 成功フラグ
	Output  string `json:"output,omitempty"` // 実行結果
	Error   string `json:"error,omitempty"`  // エラーメッセージ
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
	output, err := h.claudeService.ExecutePlan(c.Request.Context(), req.Project, req.Args)
	if err != nil {
		log.Printf("[PlanHandler] Handle failed: project=%s, args=%s, error=%v", req.Project, req.Args, err)
		c.JSON(http.StatusInternalServerError, PlanResponse{
			Success: false,
			Output:  output, // エラー時もoutputがあれば返す
			Error:   "Claude CLI実行に失敗しました: " + err.Error(),
		})
		return
	}

	log.Printf("[PlanHandler] Handle completed: project=%s, args=%s", req.Project, req.Args)
	c.JSON(http.StatusOK, PlanResponse{
		Success: true,
		Output:  output,
	})
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
