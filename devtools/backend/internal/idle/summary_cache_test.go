package idle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// readCacheEntry はキャッシュファイルを読み SummaryCacheEntry にパースします（テスト補助）。
func readCacheEntry(t *testing.T, path string) SummaryCacheEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	var e SummaryCacheEntry
	if err := json.Unmarshal(data, &e); err != nil {
		t.Fatalf("unmarshal cache: %v", err)
	}
	return e
}

// assertNoCacheTemp は *.tmp が残っていないことを検証します。
func assertNoCacheTemp(t *testing.T, dir string) {
	t.Helper()
	tmps, err := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if err != nil {
		t.Fatalf("glob tmp: %v", err)
	}
	if len(tmps) != 0 {
		t.Errorf("temp files should not remain, got %v", tmps)
	}
}

// TestSummaryCache_WriteSummary_KeyIsEntryTimeAndRoundTrips は
// `<sessionID>_<expectedTimestamp>.json` に temp+rename で書かれ、CacheKey が
// Write/Read で完全一致し、SummaryCacheEntry が再読で一致することを検証します（section6 case1）。
func TestSummaryCache_WriteSummary_KeyIsEntryTimeAndRoundTrips(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 7, 20, 12, 30, 0, 0, time.UTC)

	w := NewSummaryCacheWriter(dir)
	if err := w.WriteSummary("sess-1", 1000, "A案かB案の選択を待っている", at); err != nil {
		t.Fatalf("WriteSummary: %v", err)
	}

	// ファイル名は CacheKey(sessionID, timestamp)+".json"
	wantName := CacheKey("sess-1", 1000) + ".json"
	if wantName != "sess-1_1000.json" {
		t.Fatalf("CacheKey unexpected: %q", wantName)
	}
	path := filepath.Join(dir, wantName)

	got := readCacheEntry(t, path)
	want := SummaryCacheEntry{
		SessionID:    "sess-1",
		Timestamp:    1000,
		Summary:      "A案かB案の選択を待っている",
		SummarizedAt: at.Format(time.RFC3339),
	}
	if got != want {
		t.Errorf("entry mismatch: got %+v, want %+v", got, want)
	}
	assertNoCacheTemp(t, dir)
}

// TestSummaryCache_CacheKey_MatchesAcrossWriteAndRead は CacheKey が Write/Read/Prune で
// 共有され完全一致すること（sessionID sanitize 込み）を検証します（section6 case1・sanitize）。
func TestSummaryCache_CacheKey_MatchesAcrossWriteAndRead(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		timestamp int64
		wantKey   string
	}{
		{"uuid風", "abc-123-DEF", 1000, "abc-123-DEF_1000"},
		{"スラッシュ混入はsanitize", "a/b", 42, "a-b_42"},
		{"ドット混入はsanitize", "a.b", 7, "a-b_7"},
		{"パス区切り混入はsanitize", "../evil", 1, "---evil_1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CacheKey(tt.sessionID, tt.timestamp); got != tt.wantKey {
				t.Errorf("CacheKey(%q,%d) = %q, want %q", tt.sessionID, tt.timestamp, got, tt.wantKey)
			}
		})
	}

	// sanitize されたキーで書いたファイルが、同じ引数の CacheKey で引ける（Write/Read 完全一致）。
	dir := t.TempDir()
	at := time.Date(2026, 7, 20, 12, 30, 0, 0, time.UTC)
	w := NewSummaryCacheWriter(dir)
	if err := w.WriteSummary("a/b", 42, "要約", at); err != nil {
		t.Fatalf("WriteSummary: %v", err)
	}
	path := filepath.Join(dir, CacheKey("a/b", 42)+".json")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("sanitize後のキーで引けない: %v", err)
	}
}

// TestSummaryCache_WriteSummary_DistinctKeyDoesNotOverwrite は待機が変わって entry-time が
// 前進すると別 key で書かれ、旧 key を上書きしないことを検証します（section6 case2/3）。
func TestSummaryCache_WriteSummary_DistinctKeyDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 7, 20, 12, 30, 0, 0, time.UTC)
	w := NewSummaryCacheWriter(dir)

	if err := w.WriteSummary("sess-1", 1000, "E0の要約", at); err != nil {
		t.Fatalf("WriteSummary E0: %v", err)
	}
	// entry-time 前進（待機が新質問へ変わった）
	if err := w.WriteSummary("sess-1", 2000, "E1の要約", at); err != nil {
		t.Fatalf("WriteSummary E1: %v", err)
	}

	e0 := readCacheEntry(t, filepath.Join(dir, "sess-1_1000.json"))
	if e0.Summary != "E0の要約" || e0.Timestamp != 1000 {
		t.Errorf("旧key(E0)が上書きされた: %+v", e0)
	}
	e1 := readCacheEntry(t, filepath.Join(dir, "sess-1_2000.json"))
	if e1.Summary != "E1の要約" || e1.Timestamp != 2000 {
		t.Errorf("新key(E1)が正しく書かれていない: %+v", e1)
	}
}

// TestSummaryCache_MergeSummaries_SurvivesMtimeChange は C1×C3 の核心（reviewer 必須）です。
// キャッシュ key は entry-time E0 のみに依存するため、会話ログの mtime が変わろうと
// Marker.Timestamp=E0 が同じなら要約を引ける。対比として、待機が新質問へ変わって
// Marker.Timestamp=E1 になると `<sid>_E1.json` は無く Summary は空のまま（旧要約が復活しない）。
func TestSummaryCache_MergeSummaries_SurvivesMtimeChange(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 7, 20, 12, 30, 0, 0, time.UTC)
	const sid = "sess-1"
	const e0 = int64(1000)
	const e1 = int64(2000)

	w := NewSummaryCacheWriter(dir)
	if err := w.WriteSummary(sid, e0, "何を待っているかの要約", at); err != nil {
		t.Fatalf("WriteSummary: %v", err)
	}

	// (d) 会話ログの mtime が動いても entry-time=E0 は不変な Marker（reader 契約）
	markerE0 := Marker{SessionID: sid, Timestamp: e0, RawTail: RawTail{LastAssistant: "旧待機"}}
	merged := MergeSummaries([]Marker{markerE0}, dir)
	if len(merged) != 1 {
		t.Fatalf("merged len = %d, want 1", len(merged))
	}
	if merged[0].Summary != "何を待っているかの要約" {
		t.Errorf("E0で要約が引けない（mtimeベースなら孤児化する所）: Summary=%q", merged[0].Summary)
	}
	if merged[0].SummarizedAt != at.Format(time.RFC3339) {
		t.Errorf("SummarizedAt = %q", merged[0].SummarizedAt)
	}

	// 対比: 待機が新質問へ変わり entry-time=E1 になった Marker には旧 E0 要約を出さない
	markerE1 := Marker{SessionID: sid, Timestamp: e1, RawTail: RawTail{LastAssistant: "新待機"}}
	mergedE1 := MergeSummaries([]Marker{markerE1}, dir)
	if mergedE1[0].Summary != "" {
		t.Errorf("待機変化後(E1)に旧要約が復活した: Summary=%q", mergedE1[0].Summary)
	}
}

// TestSummaryCache_MergeSummaries_Immutability は元 markers を変更せず新スライスを返すことを検証します（section7）。
func TestSummaryCache_MergeSummaries_Immutability(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 7, 20, 12, 30, 0, 0, time.UTC)
	w := NewSummaryCacheWriter(dir)
	if err := w.WriteSummary("sess-1", 1000, "要約", at); err != nil {
		t.Fatalf("WriteSummary: %v", err)
	}

	in := []Marker{{SessionID: "sess-1", Timestamp: 1000, RawTail: RawTail{LastAssistant: "待機"}}}
	out := MergeSummaries(in, dir)

	if in[0].Summary != "" {
		t.Errorf("元marker が変更された: Summary=%q", in[0].Summary)
	}
	if out[0].Summary != "要約" {
		t.Errorf("新スライスに要約が反映されていない: Summary=%q", out[0].Summary)
	}
	if &in[0] == &out[0] {
		t.Errorf("新スライスではなく同一配列を返している")
	}
}

// TestSummaryCache_MergeSummaries_MatchBySessionAndTimestamp は複数 marker/複数キャッシュで
// sessionID+timestamp が一致するものだけ Summary が反映され、該当無しは空のままであること、
// 壊れ JSON は skip し panic しないことを検証します（section7 case3/4）。
func TestSummaryCache_MergeSummaries_MatchBySessionAndTimestamp(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 7, 20, 12, 30, 0, 0, time.UTC)
	w := NewSummaryCacheWriter(dir)
	if err := w.WriteSummary("s1", 1000, "s1の要約", at); err != nil {
		t.Fatalf("WriteSummary s1: %v", err)
	}
	if err := w.WriteSummary("s3", 3000, "s3の要約", at); err != nil {
		t.Fatalf("WriteSummary s3: %v", err)
	}
	// 壊れ JSON のキャッシュ（s4_4000）: skip されるべき
	if err := os.WriteFile(filepath.Join(dir, "s4_4000.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write broken cache: %v", err)
	}

	markers := []Marker{
		{SessionID: "s1", Timestamp: 1000}, // キャッシュあり
		{SessionID: "s2", Timestamp: 2000}, // キャッシュ無し
		{SessionID: "s3", Timestamp: 3000}, // キャッシュあり
		{SessionID: "s1", Timestamp: 9999}, // 同session別timestamp→該当無し
		{SessionID: "s4", Timestamp: 4000}, // 壊れJSON→skip
	}
	out := MergeSummaries(markers, dir)

	want := []string{"s1の要約", "", "s3の要約", "", ""}
	for i, w := range want {
		if out[i].Summary != w {
			t.Errorf("markers[%d].Summary = %q, want %q", i, out[i].Summary, w)
		}
	}
}

// TestSummaryCache_MergeSummaries_EmptyCacheHarmless は該当キャッシュ無し・空 dir で
// marker がそのまま（Summary 空）返り、エラーにも panic にもならないことを検証します（section7 case4）。
func TestSummaryCache_MergeSummaries_EmptyCacheHarmless(t *testing.T) {
	dir := t.TempDir()
	markers := []Marker{{SessionID: "s1", Timestamp: 1000}}
	out := MergeSummaries(markers, dir)
	if len(out) != 1 || out[0].Summary != "" {
		t.Errorf("空キャッシュで無害でない: %+v", out)
	}
	// 存在しない dir でも安全
	out2 := MergeSummaries(markers, filepath.Join(dir, "nope"))
	if len(out2) != 1 || out2[0].Summary != "" {
		t.Errorf("存在しないdirで無害でない: %+v", out2)
	}
}

// TestSummaryCache_PruneSummaryCache_DeletesOrphansKeepsAlive は aliveKeys 以外の *.json を削除し、
// alive に含まれるキャッシュは保持することを検証します（W5・section8 case1/3）。
func TestSummaryCache_PruneSummaryCache_DeletesOrphansKeepsAlive(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 7, 20, 12, 30, 0, 0, time.UTC)
	w := NewSummaryCacheWriter(dir)
	// alive（現存 marker が指す）
	if err := w.WriteSummary("alive", 1000, "生きている要約", at); err != nil {
		t.Fatalf("WriteSummary alive: %v", err)
	}
	// orphan（現存 marker が指さない）
	if err := w.WriteSummary("orphan", 2000, "孤児要約", at); err != nil {
		t.Fatalf("WriteSummary orphan: %v", err)
	}

	aliveKeys := map[string]bool{CacheKey("alive", 1000): true}
	if err := PruneSummaryCache(dir, aliveKeys); err != nil {
		t.Fatalf("PruneSummaryCache: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "alive_1000.json")); err != nil {
		t.Errorf("alive キャッシュが誤削除された: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "orphan_2000.json")); !os.IsNotExist(err) {
		t.Errorf("orphan キャッシュが削除されていない: err=%v", err)
	}
}

// TestSummaryCache_PruneSummaryCache_SafeOnEmptyAndMissingDir は空 dir・存在しない dir で
// prune が安全（エラーにしない・panic しない）であることを検証します（section8）。
func TestSummaryCache_PruneSummaryCache_SafeOnEmptyAndMissingDir(t *testing.T) {
	dir := t.TempDir()
	if err := PruneSummaryCache(dir, map[string]bool{}); err != nil {
		t.Errorf("空dirで prune 失敗: %v", err)
	}
	if err := PruneSummaryCache(filepath.Join(dir, "missing"), map[string]bool{}); err != nil {
		t.Errorf("存在しないdirで prune 失敗: %v", err)
	}
}
