package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSummarizeIdle_正常はresultを1行トリムして返す(t *testing.T) {
	var gotPrompt string
	calls := 0
	exec := func(ctx context.Context, prompt string) (string, error) {
		calls++
		gotPrompt = prompt
		// runClaudeSummarize が .result を抽出した後の値を模す（末尾改行あり）
		return "認証情報を確認中\n", nil
	}
	svc := newSummarizeServiceWithExec(exec, time.Second)

	got, err := svc.SummarizeIdle(context.Background(), "どの認証を使いますか", "進めて")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "認証情報を確認中" {
		t.Errorf("summary: got %q, want %q", got, "認証情報を確認中")
	}
	if calls != 1 {
		t.Errorf("exec calls: got %d, want 1", calls)
	}
	if gotPrompt == "" {
		t.Errorf("prompt should be built and passed to exec")
	}
}

func TestSummarizeIdle_CLIエラーは伝播する(t *testing.T) {
	wantErr := errors.New("claude execution failed")
	exec := func(ctx context.Context, prompt string) (string, error) {
		return "", wantErr
	}
	svc := newSummarizeServiceWithExec(exec, time.Second)

	got, err := svc.SummarizeIdle(context.Background(), "質問", "")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error should wrap CLI error: got %v", err)
	}
	if got != "" {
		t.Errorf("summary should be empty on error, got %q", got)
	}
}

func TestSummarizeIdle_出力空は空文字を返す(t *testing.T) {
	tests := []struct {
		name string
		out  string
	}{
		{name: "完全に空", out: ""},
		{name: "空白のみ", out: "   \n  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := func(ctx context.Context, prompt string) (string, error) {
				return tt.out, nil
			}
			svc := newSummarizeServiceWithExec(exec, time.Second)
			got, err := svc.SummarizeIdle(context.Background(), "質問", "回答して")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != "" {
				t.Errorf("summary: got %q, want empty", got)
			}
		})
	}
}

func TestSummarizeIdle_双方空はCLIを叩かず空返し(t *testing.T) {
	calls := 0
	exec := func(ctx context.Context, prompt string) (string, error) {
		calls++
		return "叩いてはいけない", nil
	}
	svc := newSummarizeServiceWithExec(exec, time.Second)

	tests := []struct {
		name          string
		lastAssistant string
		lastPrompt    string
	}{
		{name: "両方空文字", lastAssistant: "", lastPrompt: ""},
		{name: "両方空白のみ", lastAssistant: "  ", lastPrompt: "\n\t"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.SummarizeIdle(context.Background(), tt.lastAssistant, tt.lastPrompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != "" {
				t.Errorf("summary: got %q, want empty", got)
			}
		})
	}
	if calls != 0 {
		t.Errorf("exec should not be called for empty rawTail, got %d calls", calls)
	}
}

// タイムアウトはモックで擬似（実CLIは叩かない）。
// exec が ctx 期限を尊重してブロックし、短タイムアウトで打ち切られることを検証する。
func TestSummarizeIdle_タイムアウトで打ち切りエラー(t *testing.T) {
	exec := func(ctx context.Context, prompt string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}
	svc := newSummarizeServiceWithExec(exec, 10*time.Millisecond)

	got, err := svc.SummarizeIdle(context.Background(), "長考中", "")
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if got != "" {
		t.Errorf("summary should be empty on timeout, got %q", got)
	}
}
