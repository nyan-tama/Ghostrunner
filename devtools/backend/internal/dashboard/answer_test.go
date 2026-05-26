package dashboard

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ghostrunner/backend/internal/projects"
)

func TestAnswerQuestion_Validation(t *testing.T) {
	allowedProjects := []projects.Project{
		{Path: "/tmp/my-project", Name: "my-project"},
	}

	tests := []struct {
		name    string
		req     AnswerRequest
		wantErr bool
		errIs   error
	}{
		{
			name: "異常系: 許可外プロジェクト",
			req: AnswerRequest{
				ProjectPath: "/tmp/other-project",
				PlanPath:    filepath.Join("開発", "実装", "実行中", "plan.md"),
				LineStart:   1,
				Answer:      "answer",
			},
			wantErr: true,
			errIs:   ErrValidation,
		},
		{
			name: "異常系: 不正なディレクトリ",
			req: AnswerRequest{
				ProjectPath: "/tmp/my-project",
				PlanPath:    "other/dir/plan.md",
				LineStart:   1,
				Answer:      "answer",
			},
			wantErr: true,
			errIs:   ErrValidation,
		},
		{
			name: "異常系: .md以外の拡張子",
			req: AnswerRequest{
				ProjectPath: "/tmp/my-project",
				PlanPath:    filepath.Join("開発", "実装", "実行中", "plan.txt"),
				LineStart:   1,
				Answer:      "answer",
			},
			wantErr: true,
			errIs:   ErrValidation,
		},
		{
			name: "異常系: lineStart < 1",
			req: AnswerRequest{
				ProjectPath: "/tmp/my-project",
				PlanPath:    filepath.Join("開発", "実装", "実行中", "plan.md"),
				LineStart:   0,
				Answer:      "answer",
			},
			wantErr: true,
			errIs:   ErrValidation,
		},
		{
			name: "異常系: 空の回答",
			req: AnswerRequest{
				ProjectPath: "/tmp/my-project",
				PlanPath:    filepath.Join("開発", "実装", "実行中", "plan.md"),
				LineStart:   1,
				Answer:      "   ",
			},
			wantErr: true,
			errIs:   ErrValidation,
		},
		{
			name: "異常系: パストラバーサル",
			req: AnswerRequest{
				ProjectPath: "/tmp/my-project",
				PlanPath:    filepath.Join("開発", "実装", "実行中", "..", "..", "..", "etc", "passwd"),
				LineStart:   1,
				Answer:      "answer",
			},
			wantErr: true,
			errIs:   ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AnswerQuestion(tt.req, allowedProjects)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("expected error wrapping %v, got %v", tt.errIs, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAnswerQuestion_WriteBack(t *testing.T) {
	dir := t.TempDir()
	planDir := filepath.Join(dir, "開発", "実装", "実行中")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `# 計画書

### Q1: DBについて
テーブル構成はどうしますか？
**ステータス**: 未回答

### Q2: 別の質問
`
	planFile := filepath.Join(planDir, "plan.md")
	if err := os.WriteFile(planFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	allowed := []projects.Project{{Path: dir, Name: "test"}}

	req := AnswerRequest{
		ProjectPath: dir,
		PlanPath:    filepath.Join("開発", "実装", "実行中", "plan.md"),
		LineStart:   4, // **ステータス**: 未回答 の行
		Answer:      "正規化した設計で進めます",
	}

	if err := AnswerQuestion(req, allowed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ファイルを読み直して確認
	result, err := os.ReadFile(planFile)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}

	resultStr := string(result)

	// ステータスが回答済みに変更されている
	if !strings.Contains(resultStr, "**ステータス**: 回答済") {
		t.Error("expected status to be changed to 回答済")
	}

	// 未回答が残っていない
	if strings.Contains(resultStr, "未回答") {
		t.Error("expected no remaining 未回答")
	}

	// 回答が挿入されている
	if !strings.Contains(resultStr, "**回答**: 正規化した設計で進めます") {
		t.Error("expected answer to be inserted")
	}
}

func TestAnswerQuestion_AlreadyAnswered(t *testing.T) {
	dir := t.TempDir()
	planDir := filepath.Join(dir, "開発", "実装", "実行中")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `# 計画書

### Q1: DBについて
テーブル構成はどうしますか？
**ステータス**: 回答済
**回答**: 既存の回答
`
	planFile := filepath.Join(planDir, "plan.md")
	if err := os.WriteFile(planFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	allowed := []projects.Project{{Path: dir, Name: "test"}}

	req := AnswerRequest{
		ProjectPath: dir,
		PlanPath:    filepath.Join("開発", "実装", "実行中", "plan.md"),
		LineStart:   4,
		Answer:      "新しい回答",
	}

	err := AnswerQuestion(req, allowed)
	if !errors.Is(err, ErrAlreadyAnswered) {
		t.Errorf("expected ErrAlreadyAnswered, got %v", err)
	}
}

func TestAnswerQuestion_ExistingAnswerLine(t *testing.T) {
	dir := t.TempDir()
	planDir := filepath.Join(dir, "開発", "実装", "実行中")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 回答行が既にあるが空の場合
	content := `# 計画書

### Q1: DBについて
テーブル構成は？
**ステータス**: 未回答
**回答**:

### Q2: 次
`
	planFile := filepath.Join(planDir, "plan.md")
	if err := os.WriteFile(planFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	allowed := []projects.Project{{Path: dir, Name: "test"}}

	req := AnswerRequest{
		ProjectPath: dir,
		PlanPath:    filepath.Join("開発", "実装", "実行中", "plan.md"),
		LineStart:   5,
		Answer:      "A案で",
	}

	if err := AnswerQuestion(req, allowed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := os.ReadFile(planFile)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "**回答**: A案で") {
		t.Error("expected answer to replace existing answer line")
	}
}
