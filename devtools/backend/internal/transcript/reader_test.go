package transcript

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"ghostrunner/backend/internal/idle"
	"ghostrunner/backend/internal/projects"
)

// writeSession は homeDir/.claude/projects/<projID>/<sessionID>.jsonl に JSONL を書き、パスを返します。
func writeSession(t *testing.T, home, projID, sessionID string, lines ...string) string {
	t.Helper()
	dir := filepath.Join(home, ".claude", "projects", projID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	path := filepath.Join(dir, sessionID+".jsonl")
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write session: %v", err)
	}
	return path
}

func provider(projs ...projects.Project) func() ([]projects.Project, error) {
	return func() ([]projects.Project, error) { return projs, nil }
}

func fixedNow(ts string) func() time.Time {
	parsed, _ := time.Parse(time.RFC3339, ts)
	return func() time.Time { return parsed }
}

// TestTranscriptReaderList_WaitingSessionToMarker は待機セッションが正しい Marker になることを検証します（section3 case1/8）。
func TestTranscriptReaderList_WaitingSessionToMarker(t *testing.T) {
	home := t.TempDir()
	appPath := "/Users/x/app"
	ts := "2026-07-20T10:00:00Z"
	writeSession(t, home, "-Users-x-app", "sess-1", asstAsk(ts, appPath, "案Aと案Bどちら?"))

	r := NewReader(home, provider(projects.Project{Path: appPath, Name: "app"}), fixedNow("2026-07-20T10:30:00Z"), filepath.Join(home, "summaries"))
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(markers) != 1 {
		t.Fatalf("markers = %d, want 1", len(markers))
	}
	m := markers[0]
	if m.Cwd != appPath {
		t.Errorf("Cwd = %q, want %q", m.Cwd, appPath)
	}
	if m.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want sess-1", m.SessionID)
	}
	if m.Timestamp != epoch(t, ts) {
		t.Errorf("Timestamp = %d, want entry-time %d", m.Timestamp, epoch(t, ts))
	}
	if m.RawTail.LastAssistant != "案Aと案Bどちら?" {
		t.Errorf("RawTail.LastAssistant = %q", m.RawTail.LastAssistant)
	}
	// キャッシュ無しなら Summary は空（MergeSummaries が該当キャッシュを見つけられない）
	if m.Summary != "" || m.SummarizedAt != "" {
		t.Errorf("no-cache expects empty summary, got Summary=%q SummarizedAt=%q", m.Summary, m.SummarizedAt)
	}
}

// TestTranscriptReaderList_MergesSummaryFromCache は cacheDir に対応キャッシュを置いた待機セッションが、
// List の返す Marker に Summary/SummarizedAt 込みで返ること（C3・MergeSummaries 内包・契約3-8）を検証します。
// キャッシュ無しの別セッションは Summary 空のままであることも同時に確認します。
func TestTranscriptReaderList_MergesSummaryFromCache(t *testing.T) {
	home := t.TempDir()
	cacheDir := filepath.Join(home, "summaries")
	appPath := "/Users/x/app"
	ts := "2026-07-20T10:00:00Z"
	entryTime := epoch(t, ts)
	at := time.Date(2026, 7, 20, 10, 5, 0, 0, time.UTC)

	// 待機セッション2つ: withcache にだけ要約キャッシュを用意する。
	writeSession(t, home, "-Users-x-app", "withcache", asstAsk(ts, appPath, "案Aと案Bどちら?"))
	writeSession(t, home, "-Users-x-app", "nocache", asstText(ts, appPath, "no cache waiting"))

	// entry-time(=marker.Timestamp) を key にキャッシュを書く。
	cw := idle.NewSummaryCacheWriter(cacheDir)
	if err := cw.WriteSummary("withcache", entryTime, "A案かB案の選択を待っている", at); err != nil {
		t.Fatalf("WriteSummary: %v", err)
	}

	r := NewReader(home, provider(projects.Project{Path: appPath, Name: "app"}), fixedNow("2026-07-20T10:30:00Z"), cacheDir)
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(markers) != 2 {
		t.Fatalf("markers = %d, want 2", len(markers))
	}

	got := map[string]idle.Marker{}
	for _, m := range markers {
		got[m.SessionID] = m
	}
	if got["withcache"].Summary != "A案かB案の選択を待っている" {
		t.Errorf("withcache Summary = %q, want 反映済み", got["withcache"].Summary)
	}
	if got["withcache"].SummarizedAt != at.Format(time.RFC3339) {
		t.Errorf("withcache SummarizedAt = %q", got["withcache"].SummarizedAt)
	}
	if got["nocache"].Summary != "" || got["nocache"].SummarizedAt != "" {
		t.Errorf("nocache は空のはず: Summary=%q SummarizedAt=%q", got["nocache"].Summary, got["nocache"].SummarizedAt)
	}
}

// TestTranscriptReaderList_PrunesOrphanCache は List が孤児キャッシュ（現存 marker が指さない *.json）を
// prune し、現存 marker のキャッシュは保持することを検証します（W5・List が PruneSummaryCache を内包）。
func TestTranscriptReaderList_PrunesOrphanCache(t *testing.T) {
	home := t.TempDir()
	cacheDir := filepath.Join(home, "summaries")
	appPath := "/Users/x/app"
	ts := "2026-07-20T10:00:00Z"
	entryTime := epoch(t, ts)
	at := time.Date(2026, 7, 20, 10, 5, 0, 0, time.UTC)

	writeSession(t, home, "-Users-x-app", "live", asstText(ts, appPath, "waiting"))

	cw := idle.NewSummaryCacheWriter(cacheDir)
	// 現存 marker(live, entryTime) のキャッシュ + 孤児キャッシュ
	if err := cw.WriteSummary("live", entryTime, "生きている要約", at); err != nil {
		t.Fatalf("WriteSummary live: %v", err)
	}
	if err := cw.WriteSummary("gone", 12345, "孤児要約", at); err != nil {
		t.Fatalf("WriteSummary gone: %v", err)
	}

	r := NewReader(home, provider(projects.Project{Path: appPath, Name: "app"}), fixedNow("2026-07-20T10:30:00Z"), cacheDir)
	if _, err := r.List(context.Background()); err != nil {
		t.Fatalf("List: %v", err)
	}

	livePath := filepath.Join(cacheDir, idle.CacheKey("live", entryTime)+".json")
	if _, err := os.Stat(livePath); err != nil {
		t.Errorf("現存 marker のキャッシュが誤削除された: %v", err)
	}
	orphanPath := filepath.Join(cacheDir, idle.CacheKey("gone", 12345)+".json")
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Errorf("孤児キャッシュが prune されていない: err=%v", err)
	}
}

// TestTranscriptReaderList_SiblingProjectNoMismatch は兄弟 projid の誤マッチ防止を検証します（C2・section3 case2）。
func TestTranscriptReaderList_SiblingProjectNoMismatch(t *testing.T) {
	home := t.TempDir()
	appPath := "/Users/x/app"
	backendPath := "/Users/x/app-backend"
	ts := "2026-07-20T10:00:00Z"

	// app と app-backend の別 projid dir。両セッションとも待機中。登録は app のみ。
	writeSession(t, home, "-Users-x-app", "app-sess", asstText(ts, appPath, "app waiting"))
	writeSession(t, home, "-Users-x-app-backend", "backend-sess", asstText(ts, backendPath, "backend waiting"))

	r := NewReader(home, provider(projects.Project{Path: appPath, Name: "app"}), fixedNow("2026-07-20T10:30:00Z"), filepath.Join(home, "summaries"))
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(markers) != 1 {
		t.Fatalf("markers = %d, want 1 (app-backend must not leak into app)", len(markers))
	}
	if markers[0].Cwd != appPath {
		t.Errorf("Cwd = %q, want %q (app-backend leaked)", markers[0].Cwd, appPath)
	}
}

// TestTranscriptReaderList_Subdirectory はサブディレクトリ起動が親プロジェクトに帰属することを検証します（C2・section3 case3）。
func TestTranscriptReaderList_Subdirectory(t *testing.T) {
	home := t.TempDir()
	appPath := "/Users/x/app"
	subCwd := "/Users/x/app/sub"
	ts := "2026-07-20T10:00:00Z"
	writeSession(t, home, "-Users-x-app-sub", "sub-sess", asstText(ts, subCwd, "waiting in sub"))

	r := NewReader(home, provider(projects.Project{Path: appPath, Name: "app"}), fixedNow("2026-07-20T10:30:00Z"), filepath.Join(home, "summaries"))
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(markers) != 1 {
		t.Fatalf("markers = %d, want 1", len(markers))
	}
	if markers[0].Cwd != subCwd {
		t.Errorf("Cwd = %q, want %q (実cwd由来)", markers[0].Cwd, subCwd)
	}
}

// TestTranscriptReaderList_MultipleWaiting は同一プロジェクトの複数待機がセッション毎に Marker 化されることを検証します（section3 case4）。
// reader は1セッション1マーカーを返し、代表選定/SessionCount は下流 attachIdleState に委ねる契約（reader.go doc）。
func TestTranscriptReaderList_MultipleWaiting(t *testing.T) {
	home := t.TempDir()
	appPath := "/Users/x/app"
	writeSession(t, home, "-Users-x-app", "old-sess", asstText("2026-07-20T09:00:00Z", appPath, "older"))
	writeSession(t, home, "-Users-x-app", "new-sess", asstText("2026-07-20T10:00:00Z", appPath, "newer"))

	r := NewReader(home, provider(projects.Project{Path: appPath, Name: "app"}), fixedNow("2026-07-20T10:30:00Z"), filepath.Join(home, "summaries"))
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(markers) != 2 {
		t.Fatalf("markers = %d, want 2 (1セッション1マーカー)", len(markers))
	}
	// entry-time が各セッションの assistant timestamp を反映していること
	got := map[string]int64{}
	for _, m := range markers {
		got[m.SessionID] = m.Timestamp
	}
	if got["old-sess"] != epoch(t, "2026-07-20T09:00:00Z") {
		t.Errorf("old-sess Timestamp = %d", got["old-sess"])
	}
	if got["new-sess"] != epoch(t, "2026-07-20T10:00:00Z") {
		t.Errorf("new-sess Timestamp = %d", got["new-sess"])
	}
}

// TestTranscriptReaderList_TTLExcluded は mtime が idleTTL(6h) 超過のセッションを除外することを検証します（section3 case5）。
func TestTranscriptReaderList_TTLExcluded(t *testing.T) {
	home := t.TempDir()
	appPath := "/Users/x/app"
	path := writeSession(t, home, "-Users-x-app", "stale", asstText("2026-07-20T02:00:00Z", appPath, "waiting"))

	now, _ := time.Parse(time.RFC3339, "2026-07-20T10:30:00Z")
	// mtime を now-7h に設定（idleTTL=6h 超過）
	stale := now.Add(-7 * time.Hour)
	if err := os.Chtimes(path, stale, stale); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	r := NewReader(home, provider(projects.Project{Path: appPath, Name: "app"}), func() time.Time { return now }, filepath.Join(home, "summaries"))
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(markers) != 0 {
		t.Fatalf("markers = %d, want 0 (TTL超過は除外)", len(markers))
	}
}

// TestTranscriptReaderList_NonWaitingNoMarker は非待機セッションが Marker 化されないことを検証します（section3 case6）。
func TestTranscriptReaderList_NonWaitingNoMarker(t *testing.T) {
	home := t.TempDir()
	appPath := "/Users/x/app"
	// busy(Bash)・応答済(user末尾)・last-prompt末尾 の3セッション
	writeSession(t, home, "-Users-x-app", "busy", asstBash("2026-07-20T10:00:00Z", appPath))
	writeSession(t, home, "-Users-x-app", "answered", asstAsk("2026-07-20T10:00:00Z", appPath, "Q?"), userEntry(appPath))
	writeSession(t, home, "-Users-x-app", "prompt", asstText("2026-07-20T10:00:00Z", appPath, "hi"), lastPromptEntry(appPath, "user replied"))

	r := NewReader(home, provider(projects.Project{Path: appPath, Name: "app"}), fixedNow("2026-07-20T10:30:00Z"), filepath.Join(home, "summaries"))
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(markers) != 0 {
		t.Fatalf("markers = %d, want 0 (全て非待機)", len(markers))
	}
}

// TestTranscriptReaderList_GlobFallback は glob 絞り込みが外れても全走査 fallback で拾えることを検証します（W6・section3 case7）。
func TestTranscriptReaderList_GlobFallback(t *testing.T) {
	home := t.TempDir()
	appPath := "/Users/x/app"
	// deriveProjectID(appPath)=-Users-x-app と一致しない dir 名に配置（lossy 変換ミスマッチ想定）
	writeSession(t, home, "totally-unrelated-dirname", "sess", asstText("2026-07-20T10:00:00Z", appPath, "waiting"))

	r := NewReader(home, provider(projects.Project{Path: appPath, Name: "app"}), fixedNow("2026-07-20T10:30:00Z"), filepath.Join(home, "summaries"))
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(markers) != 1 {
		t.Fatalf("markers = %d, want 1 (全走査fallbackで拾う)", len(markers))
	}
	if markers[0].Cwd != appPath {
		t.Errorf("Cwd = %q, want %q", markers[0].Cwd, appPath)
	}
}

// TestTranscriptReaderList_BrokenSessionSkipped は壊れたセッションを skip し他は継続することを検証します（section3 case9）。
func TestTranscriptReaderList_BrokenSessionSkipped(t *testing.T) {
	home := t.TempDir()
	appPath := "/Users/x/app"
	writeSession(t, home, "-Users-x-app", "broken", `{"type":"assist`, `not json at all`)
	writeSession(t, home, "-Users-x-app", "good", asstText("2026-07-20T10:00:00Z", appPath, "waiting"))

	r := NewReader(home, provider(projects.Project{Path: appPath, Name: "app"}), fixedNow("2026-07-20T10:30:00Z"), filepath.Join(home, "summaries"))
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List should not fail wholesale: %v", err)
	}
	if len(markers) != 1 {
		t.Fatalf("markers = %d, want 1 (壊れは skip・good は返る)", len(markers))
	}
	if markers[0].SessionID != "good" {
		t.Errorf("SessionID = %q, want good", markers[0].SessionID)
	}
}

// TestTranscriptReaderList_NoProjects はプロジェクト0件で空を返すことを検証します。
func TestTranscriptReaderList_NoProjects(t *testing.T) {
	home := t.TempDir()
	r := NewReader(home, provider(), fixedNow("2026-07-20T10:30:00Z"), filepath.Join(home, "summaries"))
	markers, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(markers) != 0 {
		t.Fatalf("markers = %d, want 0", len(markers))
	}
}

// TestParseCache_MtimeUnchangedSkipsReparse は mtime 不変時に再パースが skip される（キャッシュ命中）ことを
// 観測的に検証します（W4・section5 case2）。mtime を据え置いたまま内容を変えても結果が変わらないこと＝再パースしていないこと。
func TestParseCache_MtimeUnchangedSkipsReparse(t *testing.T) {
	home := t.TempDir()
	appPath := "/Users/x/app"
	path := writeSession(t, home, "-Users-x-app", "sess", asstText("2026-07-20T10:00:00Z", appPath, "waiting-A"))

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	origMod := info.ModTime()

	r := NewReader(home, provider(projects.Project{Path: appPath, Name: "app"}), fixedNow("2026-07-20T10:30:00Z"), filepath.Join(home, "summaries"))

	first, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List 1: %v", err)
	}
	if len(first) != 1 || first[0].RawTail.LastAssistant != "waiting-A" {
		t.Fatalf("first List unexpected: %+v", first)
	}

	// 内容を非待機に書き換えるが mtime は原状に戻す → キャッシュ命中で旧結果が返るはず
	if err := os.WriteFile(path, []byte(userEntry(appPath)+"\n"), 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if err := os.Chtimes(path, origMod, origMod); err != nil {
		t.Fatalf("chtimes restore: %v", err)
	}

	cached, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List 2: %v", err)
	}
	if len(cached) != 1 || cached[0].RawTail.LastAssistant != "waiting-A" {
		t.Fatalf("mtime不変で再パースされた（キャッシュ未命中）: %+v", cached)
	}

	// mtime を進めるとキャッシュ無効化 → 再パースで非待機を反映
	newMod := origMod.Add(time.Minute)
	if err := os.Chtimes(path, newMod, newMod); err != nil {
		t.Fatalf("chtimes advance: %v", err)
	}
	after, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List 3: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("mtime変化後も再パースされていない: %+v", after)
	}
}

// TestTranscriptReaderList_ConcurrentRace は並行 List が -race clean であることを検証します（W4・section5 case1）。
func TestTranscriptReaderList_ConcurrentRace(t *testing.T) {
	home := t.TempDir()
	appPath := "/Users/x/app"
	writeSession(t, home, "-Users-x-app", "s1", asstText("2026-07-20T10:00:00Z", appPath, "waiting"))
	writeSession(t, home, "-Users-x-app", "s2", asstAsk("2026-07-20T10:01:00Z", appPath, "Q?"))

	r := NewReader(home, provider(projects.Project{Path: appPath, Name: "app"}), fixedNow("2026-07-20T10:30:00Z"), filepath.Join(home, "summaries"))

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < 5; k++ {
				markers, err := r.List(context.Background())
				if err != nil {
					t.Errorf("concurrent List: %v", err)
					return
				}
				if len(markers) != 2 {
					t.Errorf("concurrent List markers = %d, want 2", len(markers))
					return
				}
			}
		}()
	}
	wg.Wait()
}
