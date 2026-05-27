package dashboard

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetState_EmptyConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	// ファイルが存在しない場合
	fixedNow := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	svc := NewServiceWithClock(configPath, dir, func() time.Time { return fixedNow })

	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(state.Projects))
	}

	if state.GeneratedAt != fixedNow.Format(time.RFC3339) {
		t.Errorf("unexpected generatedAt: %s", state.GeneratedAt)
	}
}

func TestGetState_WithProjects(t *testing.T) {
	dir := t.TempDir()

	// プロジェクトディレクトリを作成
	projA := filepath.Join(dir, "project-a")
	projB := filepath.Join(dir, "project-b")
	for _, p := range []string{projA, projB} {
		waitDir := filepath.Join(p, "開発", "実装", "実装待ち")
		if err := os.MkdirAll(waitDir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// project-aに未回答を配置 -> attention=required
	content := "### Q1: test\nquestion\n**ステータス**: 未回答\n"
	if err := os.WriteFile(filepath.Join(projA, "開発", "実装", "実装待ち", "plan.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// 設定ファイルを作成
	config := map[string]any{
		"projects": []map[string]string{
			{"path": projA, "name": "project-a"},
			{"path": projB, "name": "project-b"},
		},
	}
	configData, _ := json.Marshal(config)
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatal(err)
	}

	fixedNow := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	svc := NewServiceWithClock(configPath, "/other", func() time.Time { return fixedNow })

	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(state.Projects))
	}

	// ソート順の確認: required(project-a)が先
	if state.Projects[0].Attention != AttentionRequired {
		t.Errorf("expected first project attention=required, got %s", state.Projects[0].Attention)
	}
	if state.Projects[0].Name != "project-a" {
		t.Errorf("expected first project name=project-a, got %s", state.Projects[0].Name)
	}
}

func TestGetState_ContextCancellation(t *testing.T) {
	dir := t.TempDir()

	// 複数プロジェクトの設定を作成
	config := map[string]any{
		"projects": []map[string]string{
			{"path": "/tmp/nonexist-a", "name": "a"},
			{"path": "/tmp/nonexist-b", "name": "b"},
		},
	}
	configData, _ := json.Marshal(config)
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 即キャンセル

	svc := NewService(configPath, dir)

	_, err := svc.GetState(ctx)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}
