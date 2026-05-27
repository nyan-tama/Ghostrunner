package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ghostrunner/backend/internal/grrun"
)

func TestGetPatternForTest_SSOT(t *testing.T) {
	// grrunパッケージのパターンと一致することを確認
	if GetPatternForTest() != grrun.UnansweredPattern {
		t.Errorf("pattern mismatch: dashboard=%q, grrun=%q", GetPatternForTest(), grrun.UnansweredPattern)
	}
}

func TestScanProject_KanbanCounts(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	// カンバンディレクトリを作成
	for _, sub := range []string{
		filepath.Join("開発", "実装", "実装待ち"),
		filepath.Join("開発", "実装", "実行中"),
		filepath.Join("開発", "実装", "完了"),
	} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// テストファイルを配置
	writeFile(t, dir, filepath.Join("開発", "実装", "実装待ち", "task1.md"), "# Task 1")
	writeFile(t, dir, filepath.Join("開発", "実装", "実装待ち", "task2.md"), "# Task 2")
	writeFile(t, dir, filepath.Join("開発", "実装", "実行中", "task3.md"), "# Task 3")
	writeFile(t, dir, filepath.Join("開発", "実装", "完了", "task4.md"), "# Task 4")
	writeFile(t, dir, filepath.Join("開発", "実装", "完了", "task5.md"), "# Task 5")
	writeFile(t, dir, filepath.Join("開発", "実装", "完了", "task6.md"), "# Task 6")

	state, err := ScanProject(dir, "/other/root", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Kanban.Waiting != 2 {
		t.Errorf("expected waiting=2, got %d", state.Kanban.Waiting)
	}
	if state.Kanban.Running != 1 {
		t.Errorf("expected running=1, got %d", state.Kanban.Running)
	}
	if state.Kanban.Done != 3 {
		t.Errorf("expected done=3, got %d", state.Kanban.Done)
	}
	if state.Kanban.Reviewing != 0 {
		t.Errorf("expected reviewing=0, got %d", state.Kanban.Reviewing)
	}
}

func TestScanProject_Unanswered(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	waitDir := filepath.Join(dir, "開発", "実装", "実装待ち")
	if err := os.MkdirAll(waitDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `# 実装計画

### Q1: DBスキーマについて
テーブル構成はどうしますか？
**ステータス**: 未回答

### Q2: APIエンドポイント
RESTとGraphQLどちらにしますか？
**ステータス**: 回答済
**回答**: RESTで
`
	writeFile(t, dir, filepath.Join("開発", "実装", "実装待ち", "plan.md"), content)

	state, err := ScanProject(dir, "/other", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Unanswered) != 1 {
		t.Fatalf("expected 1 unanswered, got %d", len(state.Unanswered))
	}

	q := state.Unanswered[0]
	if q.Heading != "### Q1: DBスキーマについて" {
		t.Errorf("unexpected heading: %s", q.Heading)
	}
	if q.PlanPath != filepath.Join("開発", "実装", "実装待ち", "plan.md") {
		t.Errorf("unexpected planPath: %s", q.PlanPath)
	}
}

func TestScanProject_Attention(t *testing.T) {
	tests := []struct {
		name     string
		state    ProjectState
		expected Attention
	}{
		{
			name: "required: 未回答あり",
			state: ProjectState{
				Unanswered: []UnansweredQuestion{{PlanPath: "test.md"}},
			},
			expected: AttentionRequired,
		},
		{
			name: "required: ops blocked",
			state: ProjectState{
				Unanswered: []UnansweredQuestion{},
				Ops:        []OpsEntry{{Status: "blocked"}},
			},
			expected: AttentionRequired,
		},
		{
			name: "required: ops stale",
			state: ProjectState{
				Unanswered: []UnansweredQuestion{},
				Ops:        []OpsEntry{{Status: "running", Stale: true}},
			},
			expected: AttentionRequired,
		},
		{
			name: "required: ops連続エラー3以上",
			state: ProjectState{
				Unanswered: []UnansweredQuestion{},
				Ops:        []OpsEntry{{ConsecutiveErrors: 3}},
			},
			expected: AttentionRequired,
		},
		{
			name: "progress: running > 0",
			state: ProjectState{
				Unanswered: []UnansweredQuestion{},
				Kanban:     KanbanCounts{Running: 1},
			},
			expected: AttentionProgress,
		},
		{
			name: "progress: ops running（正常）",
			state: ProjectState{
				Unanswered: []UnansweredQuestion{},
				Ops:        []OpsEntry{{Status: "running", Stale: false}},
			},
			expected: AttentionProgress,
		},
		{
			name: "watching: 何もなし",
			state: ProjectState{
				Unanswered: []UnansweredQuestion{},
			},
			expected: AttentionWatching,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineAttention(tt.state)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestScanProject_IsSelf(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	state, err := ScanProject(dir, dir, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.IsSelf {
		t.Error("expected isSelf=true")
	}

	state2, err := ScanProject(dir, "/other/path", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state2.IsSelf {
		t.Error("expected isSelf=false")
	}
}

func TestScanProject_OpsOptedIn(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	// 運用ディレクトリなし
	state, err := ScanProject(dir, "/other", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.OpsOptedIn {
		t.Error("expected opsOptedIn=false when no 運用/ dir")
	}

	// 運用ディレクトリあり
	opsDir := filepath.Join(dir, "運用", "状態")
	if err := os.MkdirAll(opsDir, 0755); err != nil {
		t.Fatal(err)
	}

	state2, err := ScanProject(dir, "/other", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state2.OpsOptedIn {
		t.Error("expected opsOptedIn=true when 運用/ dir exists")
	}
}

func TestScanProject_OpsStale(t *testing.T) {
	dir := t.TempDir()
	opsDir := filepath.Join(dir, "運用", "状態")
	if err := os.MkdirAll(opsDir, 0755); err != nil {
		t.Fatal(err)
	}

	opsData := map[string]any{
		"account":           "test-account",
		"kind":              "follower",
		"status":            "running",
		"consecutiveErrors": 0,
		"updatedAt":         "2026-05-26T10:00:00+09:00",
	}
	data, _ := json.Marshal(opsData)
	opsFile := filepath.Join(opsDir, "test.json")
	if err := os.WriteFile(opsFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// ファイルのModTimeを4時間前に設定
	oldTime := time.Now().Add(-4 * time.Hour)
	if err := os.Chtimes(opsFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	state, err := ScanProject(dir, "/other", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Ops) != 1 {
		t.Fatalf("expected 1 ops entry, got %d", len(state.Ops))
	}

	if !state.Ops[0].Stale {
		t.Error("expected stale=true for 4-hour-old running ops")
	}
	if state.Ops[0].StaleHours < 3 {
		t.Errorf("expected staleHours >= 3, got %d", state.Ops[0].StaleHours)
	}
}

func writeFile(t *testing.T, base, rel, content string) {
	t.Helper()
	path := filepath.Join(base, rel)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
