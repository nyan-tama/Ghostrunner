package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateService_ValidateProjectName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		setup     func(t *testing.T, baseDir string) // 事前準備（ディレクトリ作成等）
		wantValid bool
		wantError string // 空文字の場合はエラーなしを期待
	}{
		{
			name:      "正常なプロジェクト名_ハイフン区切り",
			input:     "my-project",
			wantValid: true,
		},
		{
			name:      "正常なプロジェクト名_英数字のみ",
			input:     "test123",
			wantValid: true,
		},
		{
			name:      "正常なプロジェクト名_単一文字",
			input:     "a",
			wantValid: true,
		},
		{
			name:      "空文字列",
			input:     "",
			wantValid: false,
			wantError: "プロジェクト名を入力してください",
		},
		{
			name:      "大文字を含む",
			input:     "MyProject",
			wantValid: false,
			wantError: "プロジェクト名は小文字英数字とハイフンのみ使用できます（例: my-project）",
		},
		{
			name:      "スペースを含む",
			input:     "my project",
			wantValid: false,
			wantError: "プロジェクト名は小文字英数字とハイフンのみ使用できます（例: my-project）",
		},
		{
			name:      "特殊文字を含む_アンダースコア",
			input:     "my_project",
			wantValid: false,
			wantError: "プロジェクト名は小文字英数字とハイフンのみ使用できます（例: my-project）",
		},
		{
			name:      "特殊文字を含む_ドット",
			input:     "my.project",
			wantValid: false,
			wantError: "プロジェクト名は小文字英数字とハイフンのみ使用できます（例: my-project）",
		},
		{
			name:      "先頭がハイフン",
			input:     "-project",
			wantValid: false,
			wantError: "プロジェクト名は小文字英数字とハイフンのみ使用できます（例: my-project）",
		},
		{
			name:      "末尾がハイフン",
			input:     "project-",
			wantValid: false,
			wantError: "プロジェクト名は小文字英数字とハイフンのみ使用できます（例: my-project）",
		},
		{
			name:      "連続ハイフン",
			input:     "my--project",
			wantValid: false,
			wantError: "プロジェクト名は小文字英数字とハイフンのみ使用できます（例: my-project）",
		},
		{
			name:  "既存ディレクトリ名と重複",
			input: "existing-project",
			setup: func(t *testing.T, baseDir string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(baseDir, "existing-project"), 0755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
			},
			wantValid: false,
			wantError: "同名のディレクトリが既に存在します",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()

			if tt.setup != nil {
				tt.setup(t, baseDir)
			}

			svc := NewCreateService(nil, baseDir)

			result := svc.ValidateProjectName(tt.input)

			if result.Valid != tt.wantValid {
				t.Errorf("Valid: got %v, want %v", result.Valid, tt.wantValid)
			}

			if tt.wantError != "" {
				if result.Error != tt.wantError {
					t.Errorf("Error: got %q, want %q", result.Error, tt.wantError)
				}
			}

			// 正常時はPathが設定されていることを確認
			if tt.wantValid {
				expectedPath := filepath.Join(baseDir, tt.input)
				if result.Path != expectedPath {
					t.Errorf("Path: got %q, want %q", result.Path, expectedPath)
				}
			}
		})
	}
}

func TestCreateService_ProjectBaseDir(t *testing.T) {
	baseDir := "/tmp/test-projects"
	svc := NewCreateService(nil, baseDir)

	got := svc.ProjectBaseDir()
	if got != baseDir {
		t.Errorf("ProjectBaseDir(): got %q, want %q", got, baseDir)
	}
}

func TestCreateService_CreateProject_ContextCancel(t *testing.T) {
	baseDir := t.TempDir()
	svc := NewCreateService(nil, baseDir)

	ctx, cancel := context.WithCancel(context.Background())
	// 即座にキャンセルして、ループの最初の ctx.Err() チェックで中断させる
	cancel()

	eventCh := make(chan CreateEvent, 20)

	go svc.CreateProject(ctx, &CreateRequest{
		Name:     "test-project",
		Services: []string{},
	}, eventCh)

	// チャンネルからイベントを収集
	var events []CreateEvent
	for ev := range eventCh {
		events = append(events, ev)
	}

	if len(events) == 0 {
		t.Fatal("no events received, expected at least an error event")
	}

	lastEvent := events[len(events)-1]
	if lastEvent.Type != "error" {
		t.Errorf("last event type: got %q, want %q", lastEvent.Type, "error")
	}

	if lastEvent.Error == "" {
		t.Error("last event error message should not be empty")
	}
}

func TestCreateService_CreateProject_SendsProgressEvents(t *testing.T) {
	baseDir := t.TempDir()
	// テンプレートディレクトリを準備（CopyBase が参照する空ディレクトリ）
	ghostrunnerRoot := t.TempDir()
	templateBaseDir := filepath.Join(ghostrunnerRoot, "templates", "base")
	if err := os.MkdirAll(templateBaseDir, 0o755); err != nil {
		t.Fatalf("failed to create template base dir: %v", err)
	}

	templateSvc := NewTemplateService(ghostrunnerRoot)
	svc := NewCreateService(templateSvc, baseDir)

	// CopyBase は空ディレクトリのコピーで成功する
	// ReplacePlaceholders も空ディレクトリなので成功する
	// env_create ステップで .env.example がないのでエラーになり停止する
	ctx := context.Background()
	eventCh := make(chan CreateEvent, 20)

	go svc.CreateProject(ctx, &CreateRequest{
		Name:     "test-project",
		Services: []string{},
	}, eventCh)

	var events []CreateEvent
	for ev := range eventCh {
		events = append(events, ev)
	}

	if len(events) == 0 {
		t.Fatal("no events received")
	}

	// 最初のイベントは progress で step=template_copy であること
	firstEvent := events[0]
	if firstEvent.Type != "progress" {
		t.Errorf("first event type: got %q, want %q", firstEvent.Type, "progress")
	}
	if firstEvent.Step != "template_copy" {
		t.Errorf("first event step: got %q, want %q", firstEvent.Step, "template_copy")
	}

	// progress イベントが少なくとも1つ送信されていること
	progressCount := 0
	for _, ev := range events {
		if ev.Type == "progress" {
			progressCount++
		}
	}
	if progressCount == 0 {
		t.Error("no progress events received")
	}
}
