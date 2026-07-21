package dashboard

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ghostrunner/backend/internal/idle"
)

// fakeIdleReader は idle.Reader を満たすテスト用スタブです
type fakeIdleReader struct {
	markers []idle.Marker
	err     error
}

func (f *fakeIdleReader) List(ctx context.Context) ([]idle.Marker, error) {
	return f.markers, f.err
}

// makeConfig はproject設定JSONを書き込みconfigPathを返します
func makeConfig(t *testing.T, dir string, entries []map[string]string) string {
	t.Helper()
	config := map[string]any{"projects": entries}
	configData, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatal(err)
	}
	return configPath
}

// mkProjectDir はプロジェクトの実装待ちディレクトリを作成し、pathを返します
func mkProjectDir(t *testing.T, base, name string) string {
	t.Helper()
	p := filepath.Join(base, name)
	if err := os.MkdirAll(filepath.Join(p, "開発", "実装", "実装待ち"), 0755); err != nil {
		t.Fatal(err)
	}
	return p
}

// writeUnanswered はプロジェクトに未回答確認事項の計画書を配置します（attention=required化）
func writeUnanswered(t *testing.T, projPath string) {
	t.Helper()
	content := "### Q1: test\nquestion\n**ステータス**: 未回答\n"
	f := filepath.Join(projPath, "開発", "実装", "実装待ち", "plan.md")
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

var fixedNow = time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

func epochAgo(d time.Duration) int64 {
	return fixedNow.Add(-d).Unix()
}

func TestGetState_EmptyConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	// ファイルが存在しない場合
	fixedNow := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	svc := NewServiceWithClock(configPath, dir, nil, func() time.Time { return fixedNow })

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
	svc := NewServiceWithClock(configPath, "/other", nil, func() time.Time { return fixedNow })

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

	svc := NewService(configPath, dir, nil)

	_, err := svc.GetState(ctx)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

// findProject はstate.Projectsから名前で探します
func findProject(t *testing.T, state State, name string) ProjectState {
	t.Helper()
	for _, p := range state.Projects {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("project %q not found in state", name)
	return ProjectState{}
}

// TestGetState_IdleAttachedAndAttentionReeval は、マッチしたマーカーがIdleとして付与され、
// 付与後にAttentionがrequiredへ再評価されることを検証します（C1）。
func TestGetState_IdleAttachedAndAttentionReeval(t *testing.T) {
	dir := t.TempDir()
	projA := mkProjectDir(t, dir, "project-a")

	configPath := makeConfig(t, dir, []map[string]string{
		{"path": projA, "name": "project-a"},
	})

	reader := &fakeIdleReader{markers: []idle.Marker{
		{
			Cwd:          projA,
			SessionID:    "s1",
			Timestamp:    epochAgo(12 * time.Minute),
			Status:       idle.StatusWaiting,
			SessionCount: 1,
			RawTail:      idle.RawTail{LastAssistant: "この方針で進めてよいですか"},
		},
	}}

	svc := NewServiceWithClock(configPath, "/other", reader, func() time.Time { return fixedNow })
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p := findProject(t, state, "project-a")
	if p.Idle == nil {
		t.Fatal("expected Idle to be attached")
	}
	if p.Attention != AttentionRequired {
		t.Errorf("expected attention=required after idle attach, got %s", p.Attention)
	}
	if p.Idle.SessionCount != 1 {
		t.Errorf("expected sessionCount=1, got %d", p.Idle.SessionCount)
	}
	if p.Idle.Preview != "この方針で進めてよいですか" {
		t.Errorf("unexpected preview: %q", p.Idle.Preview)
	}
	// Timestamp は RFC3339 文字列であること
	if _, err := time.Parse(time.RFC3339, p.Idle.Timestamp); err != nil {
		t.Errorf("Idle.Timestamp is not RFC3339: %q (%v)", p.Idle.Timestamp, err)
	}
}

// TestGetState_IdleYoungerThanMinAgeExcluded は、滞留が idleMinAge(60秒)未満のマーカー（応答直後で
// まだユーザーが読んでいる可能性が高い）は質問待ちに含めない（ノイズ抑制）ことを検証します。
func TestGetState_IdleYoungerThanMinAgeExcluded(t *testing.T) {
	dir := t.TempDir()
	projA := mkProjectDir(t, dir, "project-a")

	configPath := makeConfig(t, dir, []map[string]string{
		{"path": projA, "name": "project-a"},
	})

	reader := &fakeIdleReader{markers: []idle.Marker{
		{
			Cwd:          projA,
			SessionID:    "s1",
			Timestamp:    epochAgo(30 * time.Second), // 閾値60秒未満
			Status:       idle.StatusWaiting,
			SessionCount: 1,
			RawTail:      idle.RawTail{LastAssistant: "応答直後"},
		},
	}}

	svc := NewServiceWithClock(configPath, "/other", reader, func() time.Time { return fixedNow })
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p := findProject(t, state, "project-a")
	if p.Idle != nil {
		t.Errorf("expected no Idle for marker younger than idleMinAge, got %+v", p.Idle)
	}
}

// TestGetState_IdleSortsAboveRequired は、idle有りプロジェクトがidle無しのrequiredより
// 上位にソートされることを検証します（C2・idle存在が第1キー）。
func TestGetState_IdleSortsAboveRequired(t *testing.T) {
	dir := t.TempDir()
	// 名前はidle側をアルファベット後方にし、idle第1キーがname副キーに勝つことを示す
	projReq := mkProjectDir(t, dir, "a-required")
	projIdle := mkProjectDir(t, dir, "z-idle")
	writeUnanswered(t, projReq) // a-required は未回答由来required

	configPath := makeConfig(t, dir, []map[string]string{
		{"path": projReq, "name": "a-required"},
		{"path": projIdle, "name": "z-idle"},
	})

	reader := &fakeIdleReader{markers: []idle.Marker{
		{Cwd: projIdle, SessionID: "s1", Timestamp: epochAgo(3 * time.Minute), Status: idle.StatusWaiting, SessionCount: 1, RawTail: idle.RawTail{LastAssistant: "待機中"}},
	}}

	svc := NewServiceWithClock(configPath, "/other", reader, func() time.Time { return fixedNow })
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(state.Projects))
	}
	if state.Projects[0].Name != "z-idle" {
		t.Errorf("expected idle project first, got %s", state.Projects[0].Name)
	}
	// 両者ともrequiredだがidleが独立第1キーで先
	if state.Projects[0].Attention != AttentionRequired || state.Projects[1].Attention != AttentionRequired {
		t.Errorf("both should be required: %s, %s", state.Projects[0].Attention, state.Projects[1].Attention)
	}
}

// TestGetState_IdleSortByElapsedDesc は、idle同士は経過時間（timestamp古い=待機長い）が
// 上位になることを検証します（C2・同点経過降順）。
func TestGetState_IdleSortByElapsedDesc(t *testing.T) {
	dir := t.TempDir()
	projShort := mkProjectDir(t, dir, "short-wait")
	projLong := mkProjectDir(t, dir, "long-wait")

	configPath := makeConfig(t, dir, []map[string]string{
		{"path": projShort, "name": "short-wait"},
		{"path": projLong, "name": "long-wait"},
	})

	reader := &fakeIdleReader{markers: []idle.Marker{
		{Cwd: projShort, SessionID: "s1", Timestamp: epochAgo(10 * time.Minute), Status: idle.StatusWaiting, SessionCount: 1, RawTail: idle.RawTail{LastAssistant: "10分"}},
		{Cwd: projLong, SessionID: "s2", Timestamp: epochAgo(30 * time.Minute), Status: idle.StatusWaiting, SessionCount: 1, RawTail: idle.RawTail{LastAssistant: "30分"}},
	}}

	svc := NewServiceWithClock(configPath, "/other", reader, func() time.Time { return fixedNow })
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Projects[0].Name != "long-wait" {
		t.Errorf("expected longest-waiting (30min) first, got %s", state.Projects[0].Name)
	}
}

// TestGetState_IdleUsesRepSessionCount は、reader が既に代表1件へ collapse 済みの契約下で、
// attachIdleState が rep.SessionCount をそのまま採用し（len(ms) を捨てる）、代表の内容を Idle へ
// 反映することを検証します（C-1）。
func TestGetState_IdleUsesRepSessionCount(t *testing.T) {
	dir := t.TempDir()
	projA := mkProjectDir(t, dir, "project-a")

	configPath := makeConfig(t, dir, []map[string]string{
		{"path": projA, "name": "project-a"},
	})

	// reader は per-project 代表1件を返す。SessionCount は代表と同一 status のセッション数。
	reader := &fakeIdleReader{markers: []idle.Marker{
		{Cwd: projA, SessionID: "rep", Timestamp: epochAgo(40 * time.Minute), Status: idle.StatusWaiting, SessionCount: 3, RawTail: idle.RawTail{LastAssistant: "代表"}},
	}}

	svc := NewServiceWithClock(configPath, "/other", reader, func() time.Time { return fixedNow })
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p := findProject(t, state, "project-a")
	if p.Idle == nil {
		t.Fatal("expected Idle attached")
	}
	if p.Idle.SessionCount != 3 {
		t.Errorf("expected sessionCount=3 (rep.SessionCount), got %d", p.Idle.SessionCount)
	}
	if p.Idle.Preview != "代表" {
		t.Errorf("expected representative preview, got %q", p.Idle.Preview)
	}
	want := time.Unix(epochAgo(40*time.Minute), 0).Format(time.RFC3339)
	if p.Idle.Timestamp != want {
		t.Errorf("expected representative timestamp %q, got %q", want, p.Idle.Timestamp)
	}
}

// TestGetState_IdleExpiredExcluded は、TTL(6h)を超えた失効マーカーがIdle付与されない
// （除外・削除しない）ことを検証します。
func TestGetState_IdleExpiredExcluded(t *testing.T) {
	dir := t.TempDir()
	projA := mkProjectDir(t, dir, "project-a")

	configPath := makeConfig(t, dir, []map[string]string{
		{"path": projA, "name": "project-a"},
	})

	reader := &fakeIdleReader{markers: []idle.Marker{
		{Cwd: projA, SessionID: "expired", Timestamp: epochAgo(7 * time.Hour), Status: idle.StatusWaiting, SessionCount: 1, RawTail: idle.RawTail{LastAssistant: "失効"}},
	}}

	svc := NewServiceWithClock(configPath, "/other", reader, func() time.Time { return fixedNow })
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p := findProject(t, state, "project-a")
	if p.Idle != nil {
		t.Errorf("expected expired marker to be excluded, but Idle attached: %+v", p.Idle)
	}
	// Attentionは既存挙動(watching)のまま
	if p.Attention != AttentionWatching {
		t.Errorf("expected attention=watching (no idle), got %s", p.Attention)
	}
}

// TestGetState_PreviewTruncatedToRunes は、previewがLastAssistantの先頭80字(rune境界)に
// 切り詰められることを検証します。
func TestGetState_PreviewTruncatedToRunes(t *testing.T) {
	dir := t.TempDir()
	projLong := mkProjectDir(t, dir, "long-preview")
	projShort := mkProjectDir(t, dir, "short-preview")

	configPath := makeConfig(t, dir, []map[string]string{
		{"path": projLong, "name": "long-preview"},
		{"path": projShort, "name": "short-preview"},
	})

	longText := strings.Repeat("あ", 100) // 100 runes
	shortText := "短い本文"

	reader := &fakeIdleReader{markers: []idle.Marker{
		{Cwd: projLong, SessionID: "s1", Timestamp: epochAgo(2 * time.Minute), Status: idle.StatusWaiting, SessionCount: 1, RawTail: idle.RawTail{LastAssistant: longText}},
		{Cwd: projShort, SessionID: "s2", Timestamp: epochAgo(2 * time.Minute), Status: idle.StatusWaiting, SessionCount: 1, RawTail: idle.RawTail{LastAssistant: shortText}},
	}}

	svc := NewServiceWithClock(configPath, "/other", reader, func() time.Time { return fixedNow })
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pl := findProject(t, state, "long-preview")
	if pl.Idle == nil {
		t.Fatal("expected Idle attached (long)")
	}
	gotRunes := []rune(pl.Idle.Preview)
	if len(gotRunes) != 80 {
		t.Errorf("expected preview truncated to 80 runes, got %d", len(gotRunes))
	}
	if pl.Idle.Preview != strings.Repeat("あ", 80) {
		t.Errorf("unexpected truncated preview: %q", pl.Idle.Preview)
	}

	ps := findProject(t, state, "short-preview")
	if ps.Idle == nil {
		t.Fatal("expected Idle attached (short)")
	}
	if ps.Idle.Preview != shortText {
		t.Errorf("short preview should be unchanged, got %q", ps.Idle.Preview)
	}
}

// TestGetState_NilIdleReader は、NewServiceにnilを渡してもpanicせず、idle付与をスキップし
// 既存挙動が保たれることを検証します（W5）。
func TestGetState_NilIdleReader(t *testing.T) {
	dir := t.TempDir()
	projA := mkProjectDir(t, dir, "project-a")
	writeUnanswered(t, projA)

	configPath := makeConfig(t, dir, []map[string]string{
		{"path": projA, "name": "project-a"},
	})

	// idleReader = nil
	svc := NewServiceWithClock(configPath, "/other", nil, func() time.Time { return fixedNow })
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error (nil reader must not panic): %v", err)
	}

	p := findProject(t, state, "project-a")
	if p.Idle != nil {
		t.Errorf("expected no Idle with nil reader, got %+v", p.Idle)
	}
	// 未回答由来のrequiredは既存どおり
	if p.Attention != AttentionRequired {
		t.Errorf("expected attention=required (unanswered), got %s", p.Attention)
	}
}

// TestGetState_RunningAttachedNotDroppedByMinAge は、Status=running のマーカーが age<60s でも
// idleMinAge ゲートで落ちず ProjectState.Running へ付与されることを検証します（C-1・最重要）。
// running にゲートを効かせると fresh な動作中が一切表示されなくなる回帰を防ぎます。
func TestGetState_RunningAttachedNotDroppedByMinAge(t *testing.T) {
	dir := t.TempDir()
	projA := mkProjectDir(t, dir, "project-a")

	configPath := makeConfig(t, dir, []map[string]string{
		{"path": projA, "name": "project-a"},
	})

	reader := &fakeIdleReader{markers: []idle.Marker{
		{
			Cwd:          projA,
			SessionID:    "run",
			Timestamp:    epochAgo(30 * time.Second), // idleMinAge(60s)未満
			Status:       idle.StatusRunning,
			SessionCount: 2,
			RawTail:      idle.RawTail{LastAssistant: "ビルド中"},
		},
	}}

	svc := NewServiceWithClock(configPath, "/other", reader, func() time.Time { return fixedNow })
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p := findProject(t, state, "project-a")
	if p.Running == nil {
		t.Fatal("expected Running attached (idleMinAge gate must not apply to running)")
	}
	if p.Idle != nil {
		t.Errorf("running marker must not set Idle, got %+v", p.Idle)
	}
	if p.Running.SessionCount != 2 {
		t.Errorf("expected sessionCount=2 (rep.SessionCount), got %d", p.Running.SessionCount)
	}
	if p.Running.Preview != "ビルド中" {
		t.Errorf("unexpected preview: %q", p.Running.Preview)
	}
	// 付与後 attention は progress（required 要因なし）
	if p.Attention != AttentionProgress {
		t.Errorf("expected attention=progress after running attach, got %s", p.Attention)
	}
}

// TestGetState_RunningOmitemptyInJSON は、running無しプロジェクトはJSONにrunningキーが出ないこと
// （omitempty・後方互換）を検証します。
func TestGetState_RunningOmitemptyInJSON(t *testing.T) {
	dir := t.TempDir()
	projA := mkProjectDir(t, dir, "project-a")

	configPath := makeConfig(t, dir, []map[string]string{
		{"path": projA, "name": "project-a"},
	})

	reader := &fakeIdleReader{markers: []idle.Marker{}}
	svc := NewServiceWithClock(configPath, "/other", reader, func() time.Time { return fixedNow })
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "\"running\":{") {
		t.Errorf("expected no running object for non-running project, got: %s", data)
	}
}

// TestGetState_SortSecondKeyRunning は複合ソートの第2キー（Running）を固定します（section6 case3）。
// idle(waiting)有り > running有り > idle/running無しの required、の順。
// running が「未回答由来の required」より上位（第2キー Running が第3キー attention に勝つ）ことを明示します。
func TestGetState_SortSecondKeyRunning(t *testing.T) {
	dir := t.TempDir()
	// 名前で並ばないよう、期待順と逆のアルファベットにする
	projIdle := mkProjectDir(t, dir, "z-idle")
	projRunning := mkProjectDir(t, dir, "y-running")
	projRequired := mkProjectDir(t, dir, "a-required")
	writeUnanswered(t, projRequired) // a-required は未回答由来 required（idle/running 無し）

	configPath := makeConfig(t, dir, []map[string]string{
		{"path": projRequired, "name": "a-required"},
		{"path": projRunning, "name": "y-running"},
		{"path": projIdle, "name": "z-idle"},
	})

	reader := &fakeIdleReader{markers: []idle.Marker{
		{Cwd: projIdle, SessionID: "i1", Timestamp: epochAgo(3 * time.Minute), Status: idle.StatusWaiting, SessionCount: 1, RawTail: idle.RawTail{LastAssistant: "質問待ち"}},
		{Cwd: projRunning, SessionID: "r1", Timestamp: epochAgo(20 * time.Second), Status: idle.StatusRunning, SessionCount: 1, RawTail: idle.RawTail{LastAssistant: "動作中"}},
	}}

	svc := NewServiceWithClock(configPath, "/other", reader, func() time.Time { return fixedNow })
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Projects) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(state.Projects))
	}
	wantOrder := []string{"z-idle", "y-running", "a-required"}
	for i, want := range wantOrder {
		if state.Projects[i].Name != want {
			t.Errorf("sort position %d = %s, want %s (idle > running > required)", i, state.Projects[i].Name, want)
		}
	}
	// running が未回答 required より上位であること（第2キー Running > 第3キー attention）
	running := findProject(t, state, "y-running")
	if running.Running == nil {
		t.Error("expected y-running to have Running attached")
	}
}

// TestGetState_IdleOmitemptyInJSON は、idle無しプロジェクトはJSONにidleキーが出ないこと
// （omitempty・後方互換）を検証します。
func TestGetState_IdleOmitemptyInJSON(t *testing.T) {
	dir := t.TempDir()
	projA := mkProjectDir(t, dir, "project-a")

	configPath := makeConfig(t, dir, []map[string]string{
		{"path": projA, "name": "project-a"},
	})

	reader := &fakeIdleReader{markers: []idle.Marker{}}
	svc := NewServiceWithClock(configPath, "/other", reader, func() time.Time { return fixedNow })
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "\"idle\"") {
		t.Errorf("expected no idle key for non-waiting project, got: %s", data)
	}
}
