// Package service はビジネスロジックを提供します
package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ClaudeService はClaude CLI操作のインターフェースを定義します
type ClaudeService interface {
	// ExecuteCommand はカスタムコマンドを実行します
	ExecuteCommand(ctx context.Context, project, command, args string, images []ImageData) (*CommandResult, error)
	// ExecuteCommandStream はカスタムコマンドをストリーミングで実行します
	ExecuteCommandStream(ctx context.Context, project, command, args string, images []ImageData, eventCh chan<- StreamEvent) error
	// ExecutePlan は/planコマンドを実行します（互換性維持）
	ExecutePlan(ctx context.Context, project, args string) (*CommandResult, error)
	// ExecutePlanStream は/planコマンドをストリーミングで実行します（互換性維持）
	ExecutePlanStream(ctx context.Context, project, args string, eventCh chan<- StreamEvent) error
	// ContinueSession はセッションを継続して回答を送信します
	ContinueSession(ctx context.Context, project, sessionID, answer string) (*CommandResult, error)
	// ContinueSessionStream はセッションをストリーミングで継続します
	ContinueSessionStream(ctx context.Context, project, sessionID, answer string, eventCh chan<- StreamEvent) error
}

// claudeServiceImpl はClaudeServiceの実装です
type claudeServiceImpl struct {
	timeout time.Duration
}

// NewClaudeService は新しいClaudeServiceを生成します
func NewClaudeService() ClaudeService {
	return &claudeServiceImpl{
		timeout: 60 * time.Minute, // 60分タイムアウト
	}
}

// ExecuteCommand はカスタムコマンドを実行します
func (s *claudeServiceImpl) ExecuteCommand(ctx context.Context, project, command, args string, images []ImageData) (*CommandResult, error) {
	log.Printf("[ClaudeService] ExecuteCommand started: project=%s, command=%s, args=%s, images=%d", project, command, truncateLog(args, 100), len(images))

	// コマンドバリデーション
	if !AllowedCommands[command] {
		return nil, fmt.Errorf("command not allowed: %s", command)
	}

	// 画像を一時ファイルに保存
	imagePaths, cleanup, err := saveImagesToTemp(images)
	if err != nil {
		return nil, fmt.Errorf("failed to save images: %w", err)
	}
	defer cleanup()

	// プロンプト構築: "/<command> <args>"
	prompt := buildPromptWithImages(command, args, imagePaths)
	return s.executeCommand(ctx, project, prompt, "", nil)
}

// ExecuteCommandStream はカスタムコマンドをストリーミングで実行します
func (s *claudeServiceImpl) ExecuteCommandStream(ctx context.Context, project, command, args string, images []ImageData, eventCh chan<- StreamEvent) error {
	log.Printf("[ClaudeService] ExecuteCommandStream started: project=%s, command=%s, args=%s, images=%d", project, command, truncateLog(args, 100), len(images))

	// コマンドバリデーション
	if !AllowedCommands[command] {
		close(eventCh)
		return fmt.Errorf("command not allowed: %s", command)
	}

	// 画像を一時ファイルに保存
	imagePaths, cleanup, err := saveImagesToTemp(images)
	if err != nil {
		close(eventCh)
		return fmt.Errorf("failed to save images: %w", err)
	}
	defer cleanup()

	// プロンプト構築: "/<command> <args>"
	prompt := buildPromptWithImages(command, args, imagePaths)
	return s.executeCommandStream(ctx, project, prompt, "", nil, eventCh)
}

// ExecutePlan は/planコマンドを実行します（互換性維持）
func (s *claudeServiceImpl) ExecutePlan(ctx context.Context, project, args string) (*CommandResult, error) {
	log.Printf("[ClaudeService] ExecutePlan started: project=%s, args=%s", project, truncateLog(args, 100))

	return s.ExecuteCommand(ctx, project, "plan", args, nil)
}

// ContinueSession はセッションを継続して回答を送信します
func (s *claudeServiceImpl) ContinueSession(ctx context.Context, project, sessionID, answer string) (*CommandResult, error) {
	log.Printf("[ClaudeService] ContinueSession started: project=%s, sessionID=%s, answer=%s", project, sessionID, truncateLog(answer, 100))

	return s.executeCommand(ctx, project, answer, sessionID, nil)
}

// ExecutePlanStream は/planコマンドをストリーミングで実行します（互換性維持）
func (s *claudeServiceImpl) ExecutePlanStream(ctx context.Context, project, args string, eventCh chan<- StreamEvent) error {
	log.Printf("[ClaudeService] ExecutePlanStream started: project=%s, args=%s", project, truncateLog(args, 100))

	return s.ExecuteCommandStream(ctx, project, "plan", args, nil, eventCh)
}

// ContinueSessionStream はセッションをストリーミングで継続します
func (s *claudeServiceImpl) ContinueSessionStream(ctx context.Context, project, sessionID, answer string, eventCh chan<- StreamEvent) error {
	log.Printf("[ClaudeService] ContinueSessionStream started: project=%s, sessionID=%s", project, sessionID)

	return s.executeCommandStream(ctx, project, answer, sessionID, nil, eventCh)
}

// executeCommandStream はCLIコマンドをストリーミングで実行します
func (s *claudeServiceImpl) executeCommandStream(ctx context.Context, project, prompt, sessionID string, imagePaths []string, eventCh chan<- StreamEvent) error {
	defer close(eventCh)

	// タイムアウト付きコンテキストを作成
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// コマンド引数を構築（stream-jsonモード）
	// stream-jsonは--verboseが必要
	// bypassPermissionsを使用して全ての許可をバイパスする（ExitPlanMode等も許可される）
	cmdArgs := []string{"-p", prompt, "--output-format", "stream-json", "--verbose", "--permission-mode", "bypassPermissions"}
	if sessionID != "" {
		cmdArgs = append(cmdArgs, "--resume", sessionID)
	}

	log.Printf("[ClaudeService] Executing stream: claude %v", cmdArgs)

	// claude コマンドを実行
	cmd := exec.CommandContext(ctx, "claude", cmdArgs...)
	cmd.Dir = project

	// stdoutをパイプで取得
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		eventCh <- StreamEvent{Type: EventTypeError, Message: "Failed to create stdout pipe: " + err.Error()}
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// stderrもリアルタイムで出力
	cmd.Stderr = &stderrLogger{}

	// コマンド開始
	if err := cmd.Start(); err != nil {
		eventCh <- StreamEvent{Type: EventTypeError, Message: "Failed to start command: " + err.Error()}
		return fmt.Errorf("failed to start command: %w", err)
	}

	// init イベントを送信
	eventCh <- StreamEvent{Type: EventTypeInit, Message: "Claude CLI started"}

	// stdoutを行ごとに読み取り
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	var finalResult *CommandResult
	var currentSessionID string

	for scanner.Scan() {
		// コンテキストキャンセルをチェック（クライアント切断時にループを早期終了）
		if ctx.Err() != nil {
			log.Printf("[ClaudeService] Context canceled during scanning, stopping stream")
			break
		}

		line := scanner.Text()
		log.Printf("[ClaudeService] Stream line received: %s", truncateLog(line, 200))
		if line == "" {
			continue
		}

		events := s.parseStreamLine(line)
		for _, event := range events {
			// セッションIDを保持
			if event.SessionID != "" {
				currentSessionID = event.SessionID
			}

			// 最終結果を保持
			if event.Type == EventTypeComplete && event.Result != nil {
				finalResult = event.Result
			}

			// AskUserQuestion検出時はプロセスを停止してユーザー入力を待つ
			if event.Type == EventTypeQuestion {
				// セッションIDを設定
				if event.SessionID == "" {
					event.SessionID = currentSessionID
				}
				if event.Result != nil {
					event.Result.SessionID = currentSessionID
				}
				select {
				case eventCh <- event:
				case <-ctx.Done():
					log.Printf("[ClaudeService] Context canceled while sending question event")
					return nil
				}
				log.Printf("[ClaudeService] AskUserQuestion detected, killing process to wait for user input: sessionID=%s", currentSessionID)
				cmd.Process.Kill()
				return nil
			}

			select {
			case eventCh <- event:
			case <-ctx.Done():
				log.Printf("[ClaudeService] Context canceled while sending event, stopping stream")
				return nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[ClaudeService] Scanner error: %v", err)
		select {
		case eventCh <- StreamEvent{Type: EventTypeError, Message: "Stream read error: " + err.Error()}:
		case <-ctx.Done():
			log.Printf("[ClaudeService] Context canceled, skipping scanner error event")
		}
	}

	// コマンド終了を待つ
	err = cmd.Wait()

	if err != nil {
		// コンテキストキャンセルの場合
		if ctx.Err() == context.DeadlineExceeded {
			select {
			case eventCh <- StreamEvent{Type: EventTypeError, Message: "Execution timeout"}:
			default:
			}
			return fmt.Errorf("execution timeout after %v", s.timeout)
		}
		if ctx.Err() == context.Canceled {
			log.Printf("[ClaudeService] Context canceled detected: client likely disconnected")
			select {
			case eventCh <- StreamEvent{Type: EventTypeError, Message: "Client disconnected"}:
			default:
			}
			return fmt.Errorf("execution canceled: client disconnected")
		}

		// エラーでも結果がある場合は返す
		if finalResult != nil {
			return nil
		}

		log.Printf("[ClaudeService] Command failed: error=%v", err)
		select {
		case eventCh <- StreamEvent{Type: EventTypeError, Message: "Command failed: " + err.Error()}:
		default:
		}
		return fmt.Errorf("claude cli execution failed: %w", err)
	}

	// 最終結果がない場合は完了イベントを送信
	if finalResult == nil {
		eventCh <- StreamEvent{
			Type:      EventTypeComplete,
			SessionID: currentSessionID,
			Result: &CommandResult{
				SessionID: currentSessionID,
				Completed: true,
			},
		}
	}

	log.Printf("[ClaudeService] Stream completed: sessionID=%s", currentSessionID)
	return nil
}

// parseStreamLine はstream-jsonの1行をパースしてStreamEvent配列を返します
func (s *claudeServiceImpl) parseStreamLine(line string) []StreamEvent {
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		log.Printf("[ClaudeService] Failed to parse stream line: %v, line=%s", err, line)
		return nil
	}

	msgType := getStringValue(msg, "type")

	switch msgType {
	case "system":
		// システムメッセージ（セッション開始など）
		sessionID := getStringValue(msg, "session_id")
		return []StreamEvent{{
			Type:      EventTypeInit,
			SessionID: sessionID,
			Message:   "Session started",
		}}

	case "assistant":
		// アシスタントのメッセージ - 全てのcontent要素を処理
		var events []StreamEvent
		if messageData, ok := msg["message"].(map[string]interface{}); ok {
			// content配列をチェック
			if content, ok := messageData["content"].([]interface{}); ok {
				for _, c := range content {
					if contentMap, ok := c.(map[string]interface{}); ok {
						contentType := getStringValue(contentMap, "type")

						switch contentType {
						case "tool_use":
							toolName := getStringValue(contentMap, "name")
							toolInput := contentMap["input"]

							// AskUserQuestionの場合は質問イベント
							if toolName == "AskUserQuestion" {
								if inputMap, ok := toolInput.(map[string]interface{}); ok {
									questions := s.extractQuestions(inputMap)
									if len(questions) > 0 {
										events = append(events, StreamEvent{
											Type:      EventTypeQuestion,
											ToolName:  toolName,
											ToolInput: toolInput,
											Result: &CommandResult{
												Questions: questions,
												Completed: false,
											},
										})
										continue
									}
								}
							}

							events = append(events, StreamEvent{
								Type:      EventTypeToolUse,
								ToolName:  toolName,
								ToolInput: toolInput,
								Message:   formatToolMessage(toolName, toolInput),
							})

						case "text":
							text := getStringValue(contentMap, "text")
							if text != "" {
								events = append(events, StreamEvent{
									Type:    EventTypeText,
									Message: text,
								})
							}

						case "thinking":
							// 思考中
							events = append(events, StreamEvent{
								Type:    EventTypeThinking,
								Message: "Thinking...",
							})
						}
					}
				}
			}
		}
		return events

	case "result":
		// 最終結果
		resultData := msg["result"]
		if resultMap, ok := resultData.(map[string]interface{}); ok {
			sessionID := getStringValue(msg, "session_id")
			if sessionID == "" {
				sessionID = getStringValue(resultMap, "session_id")
			}

			result := &CommandResult{
				SessionID: sessionID,
				Output:    getStringValue(resultMap, "result"),
				Completed: true,
			}

			if costUSD, ok := resultMap["cost_usd"].(float64); ok {
				result.CostUSD = costUSD
			}

			// permission_denialsをチェック
			if denials, ok := resultMap["permission_denials"].([]interface{}); ok {
				for _, d := range denials {
					if denial, ok := d.(map[string]interface{}); ok {
						toolName := getStringValue(denial, "tool_name")
						if toolName == "AskUserQuestion" {
							if toolInput, ok := denial["tool_input"].(map[string]interface{}); ok {
								questions := s.extractQuestions(toolInput)
								if len(questions) > 0 {
									result.Questions = questions
									result.Completed = false
								}
							}
						}
					}
				}
			}

			return []StreamEvent{{
				Type:      EventTypeComplete,
				SessionID: sessionID,
				Result:    result,
			}}
		}

	case "content_block_start", "content_block_delta", "content_block_stop":
		// ストリーミングコンテンツブロック（必要に応じて処理）
		return nil

	default:
		log.Printf("[ClaudeService] Unknown message type: %s", msgType)
	}

	return nil
}

// formatToolMessage はツール使用の表示メッセージを生成します
func formatToolMessage(toolName string, toolInput interface{}) string {
	switch toolName {
	case "Read":
		if input, ok := toolInput.(map[string]interface{}); ok {
			filePath := getStringValue(input, "file_path")
			if filePath != "" {
				// ファイルパスを短縮
				parts := strings.Split(filePath, "/")
				if len(parts) > 3 {
					filePath = ".../" + strings.Join(parts[len(parts)-3:], "/")
				}
				return fmt.Sprintf("Reading: %s", filePath)
			}
		}
		return "Reading file..."

	case "Write":
		if input, ok := toolInput.(map[string]interface{}); ok {
			filePath := getStringValue(input, "file_path")
			if filePath != "" {
				parts := strings.Split(filePath, "/")
				if len(parts) > 3 {
					filePath = ".../" + strings.Join(parts[len(parts)-3:], "/")
				}
				return fmt.Sprintf("Writing: %s", filePath)
			}
		}
		return "Writing file..."

	case "Edit":
		if input, ok := toolInput.(map[string]interface{}); ok {
			filePath := getStringValue(input, "file_path")
			if filePath != "" {
				parts := strings.Split(filePath, "/")
				if len(parts) > 3 {
					filePath = ".../" + strings.Join(parts[len(parts)-3:], "/")
				}
				return fmt.Sprintf("Editing: %s", filePath)
			}
		}
		return "Editing file..."

	case "Glob":
		if input, ok := toolInput.(map[string]interface{}); ok {
			pattern := getStringValue(input, "pattern")
			if pattern != "" {
				return fmt.Sprintf("Searching: %s", pattern)
			}
		}
		return "Searching files..."

	case "Grep":
		if input, ok := toolInput.(map[string]interface{}); ok {
			pattern := getStringValue(input, "pattern")
			if pattern != "" {
				if len(pattern) > 30 {
					pattern = pattern[:30] + "..."
				}
				return fmt.Sprintf("Grep: %s", pattern)
			}
		}
		return "Searching content..."

	case "Bash":
		if input, ok := toolInput.(map[string]interface{}); ok {
			command := getStringValue(input, "command")
			if command != "" {
				if len(command) > 50 {
					command = command[:50] + "..."
				}
				return fmt.Sprintf("Running: %s", command)
			}
		}
		return "Running command..."

	case "TodoWrite":
		return "Updating task list..."

	case "Task":
		return "Running sub-agent..."

	case "WebFetch":
		if input, ok := toolInput.(map[string]interface{}); ok {
			url := getStringValue(input, "url")
			if url != "" {
				if len(url) > 40 {
					url = url[:40] + "..."
				}
				return fmt.Sprintf("Fetching: %s", url)
			}
		}
		return "Fetching web content..."

	default:
		return fmt.Sprintf("Using: %s", toolName)
	}
}

// executeCommand はCLIコマンドを実行し、結果をパースします
func (s *claudeServiceImpl) executeCommand(ctx context.Context, project, prompt, sessionID string, imagePaths []string) (*CommandResult, error) {
	// タイムアウト付きコンテキストを作成
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// コマンド引数を構築
	// bypassPermissionsを使用して全ての許可をバイパスする
	cmdArgs := []string{"-p", prompt, "--output-format", "json", "--permission-mode", "bypassPermissions"}
	if sessionID != "" {
		cmdArgs = append(cmdArgs, "--resume", sessionID)
	}

	log.Printf("[ClaudeService] Executing: claude %v", cmdArgs)

	// claude コマンドを実行
	cmd := exec.CommandContext(ctx, "claude", cmdArgs...)
	cmd.Dir = project

	// stdout/stderrをキャプチャ
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// コマンド実行
	err := cmd.Run()

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	log.Printf("[ClaudeService] Command completed: stdout_len=%d, stderr_len=%d, err=%v", len(stdoutStr), len(stderrStr), err)

	if err != nil {
		// コンテキストキャンセルの場合は特別なエラーメッセージ
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[ClaudeService] Command timeout: project=%s", project)
			return nil, fmt.Errorf("execution timeout after %v: %w", s.timeout, err)
		}
		if ctx.Err() == context.Canceled {
			log.Printf("[ClaudeService] Command canceled: project=%s", project)
			return nil, fmt.Errorf("execution canceled: %w", err)
		}

		// JSON出力がある場合はパースを試みる（エラー時でもJSONが返る場合がある）
		if stdoutStr != "" {
			result, parseErr := s.parseResponse(stdoutStr)
			if parseErr == nil {
				log.Printf("[ClaudeService] Parsed response from error state: sessionID=%s, questions=%d", result.SessionID, len(result.Questions))
				return result, nil
			}
		}

		log.Printf("[ClaudeService] Command failed: project=%s, error=%v, stdout=%s, stderr=%s", project, err, stdoutStr, stderrStr)
		return nil, fmt.Errorf("claude cli execution failed: %w", err)
	}

	// JSONレスポンスをパース
	result, err := s.parseResponse(stdoutStr)
	if err != nil {
		log.Printf("[ClaudeService] Failed to parse response: error=%v, stdout=%s", err, stdoutStr)
		// パースできない場合はテキストとして返す
		return &CommandResult{
			Output:    stdoutStr,
			Completed: true,
		}, nil
	}

	log.Printf("[ClaudeService] Command completed: sessionID=%s, questions=%d, completed=%v", result.SessionID, len(result.Questions), result.Completed)
	return result, nil
}

// parseResponse はCLIのJSON出力をパースします
func (s *claudeServiceImpl) parseResponse(output string) (*CommandResult, error) {
	var resp ClaudeResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := &CommandResult{
		SessionID: resp.SessionID,
		Output:    resp.Result,
		Completed: true,
		CostUSD:   resp.CostUSD,
	}

	// permission_denialsからAskUserQuestionを探す
	for _, denial := range resp.PermissionDenials {
		if denial.ToolName == "AskUserQuestion" {
			questions := s.extractQuestions(denial.ToolInput)
			if len(questions) > 0 {
				result.Questions = questions
				result.Completed = false
			}
		}
	}

	return result, nil
}

// extractQuestions はtool_inputからQuestion配列を抽出します
func (s *claudeServiceImpl) extractQuestions(toolInput map[string]interface{}) []Question {
	questionsRaw, ok := toolInput["questions"]
	if !ok {
		return nil
	}

	questionsSlice, ok := questionsRaw.([]interface{})
	if !ok {
		return nil
	}

	var questions []Question
	for _, qRaw := range questionsSlice {
		qMap, ok := qRaw.(map[string]interface{})
		if !ok {
			continue
		}

		q := Question{
			Question:    getStringValue(qMap, "question"),
			Header:      getStringValue(qMap, "header"),
			MultiSelect: getBoolValue(qMap, "multiSelect"),
		}

		// オプションを抽出
		if optionsRaw, ok := qMap["options"].([]interface{}); ok {
			for _, optRaw := range optionsRaw {
				if optMap, ok := optRaw.(map[string]interface{}); ok {
					opt := Option{
						Label:       getStringValue(optMap, "label"),
						Description: getStringValue(optMap, "description"),
					}
					q.Options = append(q.Options, opt)
				}
			}
		}

		questions = append(questions, q)
	}

	return questions
}

// getStringValue はmapから文字列を安全に取得します
func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getBoolValue はmapから真偽値を安全に取得します
func getBoolValue(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// truncateLog はログ出力用に文字列を切り詰めます
func truncateLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// stderrLogger はstderrをログに出力するio.Writer実装
type stderrLogger struct{}

func (l *stderrLogger) Write(p []byte) (n int, err error) {
	log.Printf("[ClaudeService] stderr: %s", string(p))
	return len(p), nil
}

// saveImagesToTemp は画像データを一時ファイルに保存し、パスの配列とクリーンアップ関数を返します
func saveImagesToTemp(images []ImageData) ([]string, func(), error) {
	if len(images) == 0 {
		return nil, func() {}, nil
	}

	var paths []string
	cleanup := func() {
		for _, p := range paths {
			if err := os.Remove(p); err != nil {
				log.Printf("[ClaudeService] Failed to remove temp image: path=%s, error=%v", p, err)
			}
		}
	}

	for i, img := range images {
		// Base64デコード
		decoded, err := base64.StdEncoding.DecodeString(img.Data)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("failed to decode image %d: %w", i+1, err)
		}

		// 拡張子を取得
		ext := mimeTypeToExt(img.MimeType)

		// 一時ファイルを作成
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("claude-image-%d-*%s", i, ext))
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("failed to create temp file for image %d: %w", i+1, err)
		}

		// データを書き込み
		if _, err := tmpFile.Write(decoded); err != nil {
			tmpFile.Close()
			cleanup()
			return nil, nil, fmt.Errorf("failed to write image %d: %w", i+1, err)
		}
		tmpFile.Close()

		// 絶対パスを取得
		absPath, err := filepath.Abs(tmpFile.Name())
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("failed to get absolute path for image %d: %w", i+1, err)
		}

		paths = append(paths, absPath)
		log.Printf("[ClaudeService] Saved temp image: path=%s, size=%d", absPath, len(decoded))
	}

	return paths, cleanup, nil
}

// mimeTypeToExt はMIMEタイプから拡張子を返します
func mimeTypeToExt(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}

// buildPromptWithImages は画像情報を含むプロンプトを構築します
func buildPromptWithImages(command, args string, imagePaths []string) string {
	basePrompt := fmt.Sprintf("/%s %s", command, args)

	if len(imagePaths) == 0 {
		return basePrompt
	}

	imageInfo := "\n\nAttached images:\n"
	for i, path := range imagePaths {
		imageInfo += fmt.Sprintf("%d. %s\n", i+1, path)
	}

	return basePrompt + imageInfo
}
