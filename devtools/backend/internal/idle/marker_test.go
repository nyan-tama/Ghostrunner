package idle

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// writeMarker はtmpDirにマーカーファイルを書き込むヘルパーです
func writeMarker(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestReaderList_Normal(t *testing.T) {
	dir := t.TempDir()

	writeMarker(t, dir, "sess-a.idle", `{
		"cwd": "/projects/a",
		"session_id": "sess-a",
		"timestamp": 1721476800,
		"rawTail": {"lastAssistant": "確認して良いですか", "lastPrompt": "進めて"},
		"summary": "",
		"summarizedAt": ""
	}`)
	writeMarker(t, dir, "sess-b.idle", `{
		"cwd": "/projects/b",
		"session_id": "sess-b",
		"timestamp": 1721476900,
		"rawTail": {"lastAssistant": "どちらにしますか", "lastPrompt": "A案で"},
		"summary": "",
		"summarizedAt": ""
	}`)

	r := NewReader(dir)
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(markers) != 2 {
		t.Fatalf("expected 2 markers, got %d", len(markers))
	}

	sort.Slice(markers, func(i, j int) bool { return markers[i].SessionID < markers[j].SessionID })
	if markers[0].SessionID != "sess-a" || markers[0].Cwd != "/projects/a" {
		t.Errorf("unexpected marker[0]: %+v", markers[0])
	}
	if markers[0].Timestamp != 1721476800 {
		t.Errorf("unexpected timestamp: %d", markers[0].Timestamp)
	}
	if markers[0].RawTail.LastAssistant != "確認して良いですか" {
		t.Errorf("unexpected lastAssistant: %q", markers[0].RawTail.LastAssistant)
	}
}

// TestReaderList_RealHookShape は実フック mark-idle.sh の出力形状（入れ子rawTail＋
// summary/summarizedAt＋timestampがepoch数値）が正しくパースされ、RawTail.LastAssistantが
// 非空で読めることを検証します。この形状ズレで本番previewが空になる回帰を防ぎます（reviewer申し送り）。
func TestReaderList_RealHookShape(t *testing.T) {
	dir := t.TempDir()

	// mark-idle.sh が実際に書き出す形状
	writeMarker(t, dir, "real-session.idle", `{
  "cwd": "/Users/user/myproject",
  "session_id": "real-session",
  "timestamp": 1721476800,
  "rawTail": {
    "lastAssistant": "この実装方針で進めてよいか確認させてください。認証はJWTを使いますか？",
    "lastPrompt": "認証機能を実装して"
  },
  "summary": "",
  "summarizedAt": ""
}`)

	r := NewReader(dir)
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(markers) != 1 {
		t.Fatalf("expected 1 marker, got %d", len(markers))
	}

	m := markers[0]
	if m.Cwd != "/Users/user/myproject" {
		t.Errorf("unexpected cwd: %q", m.Cwd)
	}
	if m.Timestamp != 1721476800 {
		t.Errorf("timestamp should parse as epoch number, got %d", m.Timestamp)
	}
	if m.RawTail.LastAssistant == "" {
		t.Fatal("RawTail.LastAssistant is empty: 形状ズレで本番previewが空になる回帰")
	}
	if m.RawTail.LastPrompt == "" {
		t.Error("RawTail.LastPrompt should be non-empty")
	}
}

func TestReaderList_SkipsCorruptJSON(t *testing.T) {
	dir := t.TempDir()

	writeMarker(t, dir, "good.idle", `{
		"cwd": "/projects/good",
		"session_id": "good",
		"timestamp": 1721476800,
		"rawTail": {"lastAssistant": "OK", "lastPrompt": "go"},
		"summary": "",
		"summarizedAt": ""
	}`)
	writeMarker(t, dir, "broken.idle", `{ this is not valid json `)

	r := NewReader(dir)
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("corrupt file must not fail the whole list, got error: %v", err)
	}
	if len(markers) != 1 {
		t.Fatalf("expected only the valid marker, got %d", len(markers))
	}
	if markers[0].SessionID != "good" {
		t.Errorf("unexpected surviving marker: %+v", markers[0])
	}
}

func TestReaderList_MissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")

	r := NewReader(dir)
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("missing markerDir must return nil error, got: %v", err)
	}
	if len(markers) != 0 {
		t.Errorf("expected empty slice, got %d markers", len(markers))
	}
}

func TestReaderList_IgnoresNonIdleFiles(t *testing.T) {
	dir := t.TempDir()

	writeMarker(t, dir, "sess.idle", `{
		"cwd": "/projects/x",
		"session_id": "sess",
		"timestamp": 1721476800,
		"rawTail": {"lastAssistant": "hi", "lastPrompt": "yo"},
		"summary": "",
		"summarizedAt": ""
	}`)
	// .idle 以外は無視される
	writeMarker(t, dir, "notes.txt", `not a marker`)
	writeMarker(t, dir, "sess.json", `{"cwd":"/projects/y"}`)

	r := NewReader(dir)
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(markers) != 1 {
		t.Fatalf("expected 1 marker (.idle only), got %d", len(markers))
	}
	if markers[0].SessionID != "sess" {
		t.Errorf("unexpected marker: %+v", markers[0])
	}
}
