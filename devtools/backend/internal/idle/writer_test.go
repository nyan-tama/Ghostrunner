package idle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeMarkerFile はマーカーをJSONファイルとして書き込みます（テスト補助）
func writeMarkerFile(t *testing.T, dir string, m Marker) string {
	t.Helper()
	path := filepath.Join(dir, m.SessionID+".idle")
	data, err := json.MarshalIndent(&m, "", "  ")
	if err != nil {
		t.Fatalf("marshal marker: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	return path
}

// readMarkerFile はマーカーファイルを読み取りパースします（テスト補助）
func readMarkerFile(t *testing.T, path string) Marker {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	var m Marker
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal marker: %v", err)
	}
	return m
}

// assertNoTempFiles は破棄時に *.tmp が残っていないことを検証します
func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	tmps, err := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if err != nil {
		t.Fatalf("glob tmp: %v", err)
	}
	if len(tmps) != 0 {
		t.Errorf("temp files should not remain, got %v", tmps)
	}
}

func TestWriteSummary_正常書き戻しで要約付与かつ他フィールド保持(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 7, 20, 12, 30, 0, 0, time.UTC)

	orig := Marker{
		Cwd:       "/Users/user/proj",
		SessionID: "sess-1",
		Timestamp: 1000,
		RawTail: RawTail{
			LastAssistant: "どちらの案にしますか",
			LastPrompt:    "実装を進めて",
		},
		Summary:      "",
		SummarizedAt: "",
	}
	path := writeMarkerFile(t, dir, orig)

	w := NewWriter(dir)
	if err := w.WriteSummary("sess-1", 1000, "A案かB案の選択を待っている", at); err != nil {
		t.Fatalf("WriteSummary returned error: %v", err)
	}

	got := readMarkerFile(t, path)
	if got.Summary != "A案かB案の選択を待っている" {
		t.Errorf("Summary: got %q, want %q", got.Summary, "A案かB案の選択を待っている")
	}
	if got.SummarizedAt != at.Format(time.RFC3339) {
		t.Errorf("SummarizedAt: got %q, want %q", got.SummarizedAt, at.Format(time.RFC3339))
	}
	// 他フィールドは保持される
	if got.Cwd != orig.Cwd {
		t.Errorf("Cwd: got %q, want %q", got.Cwd, orig.Cwd)
	}
	if got.Timestamp != orig.Timestamp {
		t.Errorf("Timestamp: got %d, want %d", got.Timestamp, orig.Timestamp)
	}
	if got.RawTail != orig.RawTail {
		t.Errorf("RawTail: got %+v, want %+v", got.RawTail, orig.RawTail)
	}
	assertNoTempFiles(t, dir)
}

// C3の核心: List時点(T0)のtimestampと現行マーカーのtimestampが異なる場合は破棄する。
// 「読んだ後にマーカーを別timestampで上書き」してからWriteSummaryを呼び、内容が変わらないことを検証。
func TestWriteSummary_timestamp変化で破棄し旧要約で上書きしない(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 7, 20, 12, 30, 0, 0, time.UTC)

	// T0: 要約対象を List した時点の timestamp=1000
	writeMarkerFile(t, dir, Marker{
		SessionID: "sess-1",
		Timestamp: 1000,
		RawTail:   RawTail{LastAssistant: "旧待機"},
	})

	// 要約中にユーザー回答→マーカー削除→同session新timestampで再生成された状態を再現。
	// 新マーカー(timestamp=2000, 新しい待機内容)で上書き。
	newMarker := Marker{
		SessionID: "sess-1",
		Timestamp: 2000,
		RawTail:   RawTail{LastAssistant: "新待機"},
	}
	path := writeMarkerFile(t, dir, newMarker)

	w := NewWriter(dir)
	// expectedTimestamp=1000（T0）で書き戻し。現行は 2000 なので不一致→破棄。
	if err := w.WriteSummary("sess-1", 1000, "旧待機に対する古い要約", at); err != nil {
		t.Fatalf("WriteSummary returned error: %v", err)
	}

	got := readMarkerFile(t, path)
	if got.Timestamp != 2000 {
		t.Errorf("Timestamp: got %d, want 2000 (新マーカー保持)", got.Timestamp)
	}
	if got.Summary != "" {
		t.Errorf("Summary: got %q, want empty (旧要約で上書きしない)", got.Summary)
	}
	if got.RawTail.LastAssistant != "新待機" {
		t.Errorf("RawTail: got %q, want 新待機", got.RawTail.LastAssistant)
	}
	assertNoTempFiles(t, dir)
}

func TestWriteSummary_マーカー不在で破棄しnil返し(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 7, 20, 12, 30, 0, 0, time.UTC)

	w := NewWriter(dir)
	// マーカーが存在しない（ユーザー回答で削除済み）→ 復活させず nil を返す
	if err := w.WriteSummary("no-such-session", 1000, "要約", at); err != nil {
		t.Fatalf("WriteSummary should return nil on missing marker, got %v", err)
	}

	// マーカーは作られていない（復活しない）
	path := filepath.Join(dir, "no-such-session.idle")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("marker should not be created, stat err=%v", err)
	}
	assertNoTempFiles(t, dir)
}

func TestWriteSummary_session_id不一致で破棄(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 7, 20, 12, 30, 0, 0, time.UTC)

	// ファイル名は sess-1 だが中身の session_id が別（想定外の不整合）
	path := writeMarkerFile(t, dir, Marker{
		SessionID: "different-session",
		Timestamp: 1000,
	})

	w := NewWriter(dir)
	if err := w.WriteSummary("sess-1", 1000, "要約", at); err != nil {
		t.Fatalf("WriteSummary should return nil on session_id mismatch, got %v", err)
	}

	got := readMarkerFile(t, path)
	if got.Summary != "" {
		t.Errorf("Summary: got %q, want empty (session_id不一致で破棄)", got.Summary)
	}
	assertNoTempFiles(t, dir)
}
