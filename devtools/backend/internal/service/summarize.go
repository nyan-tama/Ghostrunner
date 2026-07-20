// Package service はビジネスロジックを提供します
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// summarizeTimeout は要約CLI実行の専用短タイムアウトです。
// 要約は滞留マーカーの後付け処理のため、通常のコマンド実行（60分）とは分離した
// 短いタイムアウトで打ち切ります（W4）。
const summarizeTimeout = 45 * time.Second

// SummarizeService は質問待ちマーカーの会話末尾を日本語1行に要約するサービスです
type SummarizeService interface {
	// SummarizeIdle は会話末尾（直近のアシスタント発言とユーザー発言）から
	// 「ユーザーが今何を答えれば/決めれば良いか」の日本語1行要約を返します。
	// 双方が空の場合はCLIを呼ばず空文字を返します。
	SummarizeIdle(ctx context.Context, lastAssistant, lastPrompt string) (string, error)
}

// summarizeExecFunc は要約用CLIの実行を抽象化します（テストでモック可能）
type summarizeExecFunc func(ctx context.Context, prompt string) (string, error)

type summarizeServiceImpl struct {
	exec    summarizeExecFunc
	timeout time.Duration
}

// NewSummarizeService は claude CLI(haikuモデル)を用いる本番用SummarizeServiceを生成します
func NewSummarizeService() SummarizeService {
	s := &summarizeServiceImpl{timeout: summarizeTimeout}
	s.exec = runClaudeSummarize
	return s
}

// newSummarizeServiceWithExec はCLI実行を差し替えたSummarizeServiceを生成します（テスト用）
func newSummarizeServiceWithExec(exec summarizeExecFunc, timeout time.Duration) SummarizeService {
	return &summarizeServiceImpl{exec: exec, timeout: timeout}
}

// SummarizeIdle は会話末尾から日本語1行要約を返します
func (s *summarizeServiceImpl) SummarizeIdle(ctx context.Context, lastAssistant, lastPrompt string) (string, error) {
	// 双方空はCLIを無駄打ちしない
	if strings.TrimSpace(lastAssistant) == "" && strings.TrimSpace(lastPrompt) == "" {
		return "", nil
	}

	prompt := buildSummarizePrompt(lastAssistant, lastPrompt)

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	log.Printf("[SummarizeService] SummarizeIdle started")
	out, err := s.exec(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to summarize idle: %w", err)
	}

	summary := firstLine(out)
	log.Printf("[SummarizeService] SummarizeIdle completed: summary=%s", summary)
	return summary, nil
}

// runClaudeSummarize は claude -p <prompt> --model haiku --output-format json を実行し
// JSON の .result を返します。cmd.Dir はユーザーhomeとしプロジェクト非依存にします。
func runClaudeSummarize(ctx context.Context, prompt string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "claude", "-p", prompt, "--model", "haiku", "--output-format", "json")
	cmd.Dir = home

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("claude summarize execution failed: %w", err)
	}

	var resp struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("failed to parse summarize response: %w", err)
	}

	return resp.Result, nil
}

// buildSummarizePrompt は要約プロンプトを構築します
func buildSummarizePrompt(lastAssistant, lastPrompt string) string {
	var b strings.Builder
	b.WriteString("次のやりとりで、ユーザーが今何を答えれば/決めれば良いかを日本語1行(30字程度)で要約。前置き不要。\n\n")
	if lastPrompt != "" {
		b.WriteString("ユーザーの直近の発言:\n")
		b.WriteString(lastPrompt)
		b.WriteString("\n\n")
	}
	if lastAssistant != "" {
		b.WriteString("アシスタントの最後の発言:\n")
		b.WriteString(lastAssistant)
		b.WriteString("\n")
	}
	return b.String()
}

// firstLine は文字列を先頭1行にトリムします（前後空白除去・改行以降を切り落とす）
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexAny(s, "\r\n"); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}
