// Package service はビジネスロジックを提供します
package service

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"
)

// ClaudeService はClaude CLI操作のインターフェースを定義します
type ClaudeService interface {
	// ExecutePlan は/planコマンドを実行します
	ExecutePlan(ctx context.Context, project, args string) (output string, err error)
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

// ExecutePlan はClaude CLIの/planコマンドを実行します
func (s *claudeServiceImpl) ExecutePlan(ctx context.Context, project, args string) (string, error) {
	log.Printf("[ClaudeService] ExecutePlan started: project=%s, args=%s", project, args)

	// タイムアウト付きコンテキストを作成
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// /plan コマンドを構築
	prompt := fmt.Sprintf("/plan %s", args)

	// claude -p "prompt" --cwd project を実行
	// シェル経由せず直接実行することでコマンドインジェクションを防止
	cmd := exec.CommandContext(ctx, "claude", "-p", prompt, "--cwd", project)

	// stdout/stderrをキャプチャ
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// コマンド実行
	err := cmd.Run()

	// stdout/stderrを結合
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if err != nil {
		// コンテキストキャンセルの場合は特別なエラーメッセージ
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[ClaudeService] ExecutePlan timeout: project=%s, args=%s", project, args)
			return output, fmt.Errorf("execution timeout after %v: %w", s.timeout, err)
		}
		if ctx.Err() == context.Canceled {
			log.Printf("[ClaudeService] ExecutePlan canceled: project=%s, args=%s", project, args)
			return output, fmt.Errorf("execution canceled: %w", err)
		}

		log.Printf("[ClaudeService] ExecutePlan failed: project=%s, args=%s, error=%v", project, args, err)
		return output, fmt.Errorf("claude cli execution failed: %w", err)
	}

	log.Printf("[ClaudeService] ExecutePlan completed: project=%s, args=%s", project, args)
	return output, nil
}
