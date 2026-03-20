// Package handler はHTTPハンドラーを提供します
package handler

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"ghostrunner/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// ImageData は画像データを表します（handler層の定義）
type ImageData struct {
	Name     string `json:"name"`     // ファイル名
	Data     string `json:"data"`     // Base64エンコードされたデータ
	MimeType string `json:"mimeType"` // MIMEタイプ
}

// CommandRequest は/api/commandリクエストの構造体です
type CommandRequest struct {
	Project string      `json:"project"`          // プロジェクトのパス
	Command string      `json:"command"`          // 実行するコマンド（plan, fullstack, go, nextjs）
	Args    string      `json:"args"`             // コマンドの引数
	Images  []ImageData `json:"images,omitempty"` // 画像データ（オプション）
}

// CommandContinueRequest は/api/command/continueリクエストの構造体です
type CommandContinueRequest struct {
	Project   string `json:"project"`    // プロジェクトのパス
	SessionID string `json:"session_id"` // セッションID
	Answer    string `json:"answer"`     // ユーザーの回答
}

// CommandResponse は/api/commandレスポンスの構造体です
type CommandResponse struct {
	Success   bool               `json:"success"`              // 成功フラグ
	SessionID string             `json:"session_id,omitempty"` // セッションID
	Output    string             `json:"output,omitempty"`     // 実行結果
	Questions []service.Question `json:"questions,omitempty"`  // 質問がある場合
	Completed bool               `json:"completed"`            // 完了したかどうか
	CostUSD   float64            `json:"cost_usd,omitempty"`   // コスト
	Error     string             `json:"error,omitempty"`      // エラーメッセージ
}

// CommandHandler はCommand関連のHTTPハンドラを提供します
type CommandHandler struct {
	claudeService service.ClaudeService
}

// NewCommandHandler は新しいCommandHandlerを生成します
func NewCommandHandler(claudeService service.ClaudeService) *CommandHandler {
	return &CommandHandler{
		claudeService: claudeService,
	}
}

// Handle は/api/commandリクエストを処理します
// POST /api/command
func (h *CommandHandler) Handle(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[CommandHandler] Handle failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	log.Printf("[CommandHandler] Handle started: project=%s, command=%s, args=%s", req.Project, req.Command, req.Args)

	// プロジェクトパスのバリデーション
	if err := validateProjectPath(req.Project); err != nil {
		log.Printf("[CommandHandler] Handle failed: invalid project path, project=%s, error=%v", req.Project, err)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// コマンドのバリデーション
	if req.Command == "" {
		log.Printf("[CommandHandler] Handle failed: command is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "commandは必須です",
		})
		return
	}

	if !service.AllowedCommands[req.Command] {
		log.Printf("[CommandHandler] Handle failed: command not allowed, project=%s, command=%s", req.Project, req.Command)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "許可されていないコマンドです: " + req.Command,
		})
		return
	}

	// argsのバリデーション
	if req.Args == "" {
		log.Printf("[CommandHandler] Handle failed: args is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "argsは必須です",
		})
		return
	}

	// 画像のバリデーション
	if err := validateImages(req.Images); err != nil {
		log.Printf("[CommandHandler] Handle failed: invalid images, project=%s, error=%v", req.Project, err)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Claude CLIを実行
	result, err := h.claudeService.ExecuteCommand(c.Request.Context(), req.Project, req.Command, req.Args, toServiceImages(req.Images))
	if err != nil {
		log.Printf("[CommandHandler] Handle failed: project=%s, command=%s, error=%v", req.Project, req.Command, err)
		c.JSON(http.StatusInternalServerError, CommandResponse{
			Success: false,
			Error:   "Claude CLI実行に失敗しました: " + err.Error(),
		})
		return
	}

	log.Printf("[CommandHandler] Handle completed: project=%s, command=%s, sessionID=%s, questions=%d, completed=%v",
		req.Project, req.Command, result.SessionID, len(result.Questions), result.Completed)

	c.JSON(http.StatusOK, CommandResponse{
		Success:   true,
		SessionID: result.SessionID,
		Output:    result.Output,
		Questions: result.Questions,
		Completed: result.Completed,
		CostUSD:   result.CostUSD,
	})
}

// HandleStream は/api/command/streamリクエストを処理します（Server-Sent Events）
// POST /api/command/stream
func (h *CommandHandler) HandleStream(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[CommandHandler] HandleStream failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	log.Printf("[CommandHandler] HandleStream started: project=%s, command=%s, args=%s", req.Project, req.Command, req.Args)

	// プロジェクトパスのバリデーション
	if err := validateProjectPath(req.Project); err != nil {
		log.Printf("[CommandHandler] HandleStream failed: invalid project path, project=%s, error=%v", req.Project, err)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// コマンドのバリデーション
	if req.Command == "" {
		log.Printf("[CommandHandler] HandleStream failed: command is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "commandは必須です",
		})
		return
	}

	if !service.AllowedCommands[req.Command] {
		log.Printf("[CommandHandler] HandleStream failed: command not allowed, project=%s, command=%s", req.Project, req.Command)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "許可されていないコマンドです: " + req.Command,
		})
		return
	}

	// argsのバリデーション
	if req.Args == "" {
		log.Printf("[CommandHandler] HandleStream failed: args is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "argsは必須です",
		})
		return
	}

	// 画像のバリデーション
	if err := validateImages(req.Images); err != nil {
		log.Printf("[CommandHandler] HandleStream failed: invalid images, project=%s, error=%v", req.Project, err)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// SSEヘッダー設定
	setSSEHeaders(c)

	// イベントチャンネル作成
	eventCh := make(chan service.StreamEvent, 100)

	// 画像データを変換
	serviceImages := toServiceImages(req.Images)

	// ストリーミング実行を開始
	go func() {
		err := h.claudeService.ExecuteCommandStream(c.Request.Context(), req.Project, req.Command, req.Args, serviceImages, eventCh)
		if err != nil {
			log.Printf("[CommandHandler] HandleStream error: %v", err)
		}
	}()

	// イベントをSSEとして送信（selectベースのループで確実にコンテキストキャンセルを検出）
	writeSSEEvents(c, eventCh, "CommandHandler")

	log.Printf("[CommandHandler] HandleStream completed: project=%s, command=%s", req.Project, req.Command)
}

// HandleContinue は/api/command/continueリクエストを処理します
// POST /api/command/continue
func (h *CommandHandler) HandleContinue(c *gin.Context) {
	var req CommandContinueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[CommandHandler] HandleContinue failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	log.Printf("[CommandHandler] HandleContinue started: project=%s, sessionID=%s, answer=%s", req.Project, req.SessionID, req.Answer)

	// プロジェクトパスのバリデーション
	if err := validateProjectPath(req.Project); err != nil {
		log.Printf("[CommandHandler] HandleContinue failed: invalid project path, project=%s, error=%v", req.Project, err)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// セッションIDのバリデーション
	if req.SessionID == "" {
		log.Printf("[CommandHandler] HandleContinue failed: sessionID is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "session_idは必須です",
		})
		return
	}

	// 回答のバリデーション
	if req.Answer == "" {
		log.Printf("[CommandHandler] HandleContinue failed: answer is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "answerは必須です",
		})
		return
	}

	// セッションを継続
	result, err := h.claudeService.ContinueSession(c.Request.Context(), req.Project, req.SessionID, req.Answer)
	if err != nil {
		log.Printf("[CommandHandler] HandleContinue failed: project=%s, sessionID=%s, error=%v", req.Project, req.SessionID, err)
		c.JSON(http.StatusInternalServerError, CommandResponse{
			Success: false,
			Error:   "セッション継続に失敗しました: " + err.Error(),
		})
		return
	}

	log.Printf("[CommandHandler] HandleContinue completed: project=%s, sessionID=%s, questions=%d, completed=%v",
		req.Project, result.SessionID, len(result.Questions), result.Completed)

	c.JSON(http.StatusOK, CommandResponse{
		Success:   true,
		SessionID: result.SessionID,
		Output:    result.Output,
		Questions: result.Questions,
		Completed: result.Completed,
		CostUSD:   result.CostUSD,
	})
}

// HandleContinueStream は/api/command/continue/streamリクエストを処理します（Server-Sent Events）
// POST /api/command/continue/stream
func (h *CommandHandler) HandleContinueStream(c *gin.Context) {
	var req CommandContinueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[CommandHandler] HandleContinueStream failed: invalid request, error=%v", err)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "リクエストが不正です",
		})
		return
	}

	log.Printf("[CommandHandler] HandleContinueStream started: project=%s, sessionID=%s", req.Project, req.SessionID)

	// プロジェクトパスのバリデーション
	if err := validateProjectPath(req.Project); err != nil {
		log.Printf("[CommandHandler] HandleContinueStream failed: invalid project path, project=%s, error=%v", req.Project, err)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// セッションIDのバリデーション
	if req.SessionID == "" {
		log.Printf("[CommandHandler] HandleContinueStream failed: sessionID is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, CommandResponse{
			Success: false,
			Error:   "session_idは必須です",
		})
		return
	}

	// 回答のバリデーション
	if req.Answer == "" {
		log.Printf("[CommandHandler] HandleContinueStream failed: answer is empty, project=%s", req.Project)
		c.JSON(http.StatusBadRequest, CommandResponse{
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
			log.Printf("[CommandHandler] HandleContinueStream error: %v", err)
		}
	}()

	// イベントをSSEとして送信（selectベースのループで確実にコンテキストキャンセルを検出）
	writeSSEEvents(c, eventCh, "CommandHandler")

	log.Printf("[CommandHandler] HandleContinueStream completed: project=%s, sessionID=%s", req.Project, req.SessionID)
}

// validateImages は画像データをバリデーションします
func validateImages(images []ImageData) error {
	if len(images) == 0 {
		return nil
	}

	if len(images) > service.MaxImageCount {
		return fmt.Errorf("画像は最大%d枚までです", service.MaxImageCount)
	}

	for i, img := range images {
		// MIMEタイプのバリデーション
		if !service.AllowedImageMimeTypes[img.MimeType] {
			return fmt.Errorf("画像%d: サポートされていない形式です（JPEG, PNG, GIF, WebPのみ対応）", i+1)
		}

		// Base64デコードして画像データの妥当性をチェック
		decoded, err := base64.StdEncoding.DecodeString(img.Data)
		if err != nil {
			return fmt.Errorf("画像%d: Base64デコードに失敗しました", i+1)
		}

		// サイズチェック
		if len(decoded) > service.MaxImageSize {
			return fmt.Errorf("画像%d: サイズが5MBを超えています", i+1)
		}
	}

	return nil
}

// toServiceImages はhandler.ImageDataをservice.ImageDataに変換します
func toServiceImages(images []ImageData) []service.ImageData {
	if len(images) == 0 {
		return nil
	}

	result := make([]service.ImageData, len(images))
	for i, img := range images {
		result[i] = service.ImageData{
			Name:     img.Name,
			Data:     img.Data,
			MimeType: img.MimeType,
		}
	}
	return result
}
