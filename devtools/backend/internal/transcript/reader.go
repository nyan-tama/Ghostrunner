package transcript

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"ghostrunner/backend/internal/idle"
	"ghostrunner/backend/internal/projects"
)

// transcriptReader は会話ログ JSONL を直読みして質問待ち/動作中を検出する idle.Reader 実装です。
type transcriptReader struct {
	homeDir          string
	projectsProvider func() ([]projects.Project, error)
	now              func() time.Time
	cacheDir         string
	cache            *parseCache
}

// NewReader は会話ログ直読みの idle.Reader を生成します。
// projectsProvider は走査対象の登録プロジェクトを都度取得します。
// cacheDir は要約キャッシュ（~/.claude/gr-idle-summaries）の格納先で、List が
// MergeSummaries / PruneSummaryCache に用います。
// now が nil の場合は time.Now を使います。
func NewReader(homeDir string, projectsProvider func() ([]projects.Project, error), now func() time.Time, cacheDir string) idle.Reader {
	if now == nil {
		now = time.Now
	}
	return &transcriptReader{
		homeDir:          homeDir,
		projectsProvider: projectsProvider,
		now:              now,
		cacheDir:         cacheDir,
		cache:            newParseCache(),
	}
}

// classifiedSession は1セッションの分類結果です（代表選定・SessionCount 集計用）。
type classifiedSession struct {
	sf     sessionFile
	tail   transcriptTail
	status idle.Status // "" は none（マーカー化しない）
}

// classifyRepresentative は種別（内容）と mtime 鮮度を合成して最終 status を確定する純粋関数です（C-2）。
// kind ごとに境界を変え、45〜60秒のデッドゾーンを作りません。
//   - midTurn: mtimeAge < RunningMaxAge なら running、超なら none（固まった tool_use の 6h 青表示を防ぐ・W-1）
//   - waiting: mtimeAge < MinAge(60s) なら running、>= 60s なら waiting（境界は 60s で一本化）
//   - none:    mtimeAge < BusyThreshold(45s) なら running、それ以外 none
func classifyRepresentative(kind tailKind, mtimeAge time.Duration) idle.Status {
	switch kind {
	case kindMidTurn:
		if mtimeAge < idle.RunningMaxAge {
			return idle.StatusRunning
		}
		return ""
	case kindWaiting:
		if mtimeAge < idle.MinAge {
			return idle.StatusRunning
		}
		return idle.StatusWaiting
	default: // kindNone
		if mtimeAge < idle.BusyThreshold {
			return idle.StatusRunning
		}
		return ""
	}
}

// List は登録プロジェクトの会話ログを走査し、プロジェクト毎に最新 mtime の代表セッション1件を
// 分類（running/waiting/none）して idle.Marker を返します。none 代表はマーカー化しません。
// 帰属は各セッションの実 cwd + idle.MatchProject で判定します（C2）。Marker.Timestamp は
// waiting なら代表の assistant entry-time（C1）、running なら代表の mtime です。
// Marker.SessionCount は代表と同一 status のセッション数です。
func (r *transcriptReader) List(ctx context.Context) ([]idle.Marker, error) {
	projs, err := r.projectsProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to load projects: %w", err)
	}
	if len(projs) == 0 {
		return []idle.Marker{}, nil
	}

	sessions, err := discoverSessions(r.homeDir, projs)
	if err != nil {
		return nil, fmt.Errorf("failed to discover sessions: %w", err)
	}

	now := r.now()
	alive := make(map[string]struct{}, len(sessions))
	byProject := make(map[string][]classifiedSession)

	for _, sf := range sessions {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		alive[sf.path] = struct{}{}

		// 粗い liveness ゲート: mtime が TTL より古いセッションは終了扱いで除外。
		if now.Sub(sf.modTime) > idle.TTL {
			continue
		}

		tail, ok := r.tailFor(sf)
		if !ok {
			continue
		}

		// 帰属は実 cwd + MatchProject（C2）。glob の前方一致は帰属に使わない。
		matched, ok := idle.MatchProject(tail.Cwd, projs)
		if !ok {
			continue
		}

		status := classifyRepresentative(tail.Kind, now.Sub(sf.modTime))
		byProject[matched] = append(byProject[matched], classifiedSession{sf: sf, tail: tail, status: status})
	}

	r.cache.prune(alive)

	markers := make([]idle.Marker, 0, len(byProject))
	for _, sessList := range byProject {
		rep := representative(sessList)
		if rep.status == "" {
			// 代表が none（応答済/古い midTurn 等）→ マーカー化しない
			continue
		}
		count := countByStatus(sessList, rep.status)
		markers = append(markers, r.buildMarker(rep, count, now))
	}

	// W5/W-2: 孤児キャッシュ掃除の aliveKeys は waiting marker のみで構築する。
	// running marker は要約対象外（summarizer が waiting 限定）で要約キャッシュを持たないため、
	// running の CacheKey を alive に含めると本来掃除すべき孤児判定を汚す。
	aliveKeys := make(map[string]bool, len(markers))
	for _, m := range markers {
		if m.Status == idle.StatusWaiting {
			aliveKeys[idle.CacheKey(m.SessionID, m.Timestamp)] = true
		}
	}
	if err := idle.PruneSummaryCache(r.cacheDir, aliveKeys); err != nil {
		log.Printf("[transcript] prune summary cache failed: error=%v", err)
	}

	// C3: 要約マージは List 内で実行し、reader は Summary 込みの完成 Marker を返す契約にする。
	// running marker にも走るが対応キャッシュが無く cache ヒットしないため無害（W-2）。
	markers = idle.MergeSummaries(markers, r.cacheDir)

	return markers, nil
}

// representative は最新 mtime のセッションを代表として返します（同着は先着優先）。
func representative(sessions []classifiedSession) classifiedSession {
	rep := sessions[0]
	for _, s := range sessions[1:] {
		if s.sf.modTime.After(rep.sf.modTime) {
			rep = s
		}
	}
	return rep
}

// countByStatus は同一 status のセッション数を返します（代表1件＋件数の集計）。
func countByStatus(sessions []classifiedSession, status idle.Status) int {
	n := 0
	for _, s := range sessions {
		if s.status == status {
			n++
		}
	}
	return n
}

// buildMarker は代表セッションから idle.Marker を組み立てます。
// Timestamp は waiting なら entry-time（C1・要約 key の同一性）、running なら mtime です。
func (r *transcriptReader) buildMarker(rep classifiedSession, count int, now time.Time) idle.Marker {
	var ts int64
	if rep.status == idle.StatusWaiting {
		ts = rep.tail.LastAssistantAt
		if ts == 0 {
			// entry-time 欠落版: 本文署名の初回検出時刻でキーを安定化（raw mtime fallback 禁止）
			ts = r.cache.stableTimestamp(rep.tail.ContentHash, now)
		}
	} else {
		ts = rep.sf.modTime.Unix()
	}

	return idle.Marker{
		Cwd:          rep.tail.Cwd,
		SessionID:    rep.sf.sessionID,
		Timestamp:    ts,
		Status:       rep.status,
		SessionCount: count,
		RawTail:      idle.RawTail{LastAssistant: rep.tail.LastAssistant, LastPrompt: rep.tail.LastPrompt},
		Summary:      "",
		SummarizedAt: "",
	}
}

// tailFor は parseCache を用いてセッションの待機判定を取得します。
// IO（parseTail）はロック外で行い、並行 List を直列化しません（W4）:
// check(lock)→miss判定→unlock→parseTail(IO)→lock→store。
func (r *transcriptReader) tailFor(sf sessionFile) (transcriptTail, bool) {
	if tail, ok := r.cache.get(sf.path, sf.modTime); ok {
		return tail, true
	}
	tail, err := parseTail(sf.path)
	if err != nil {
		log.Printf("[transcript] skip session (parse failed): path=%s, error=%v", sf.path, err)
		return transcriptTail{}, false
	}
	r.cache.store(sf.path, sf.modTime, tail)
	return tail, true
}

// cachedParse は1セッションのパース結果キャッシュです。
type cachedParse struct {
	modTime time.Time
	tail    transcriptTail
}

// parseCache は mtime 不変時の再パース抑制と、entry-time 欠落版の署名→初回検出時刻の保持を担います。
// IO はロック外で行う前提で、本キャッシュは in-memory の読み書きのみをロックで守ります（W4）。
type parseCache struct {
	mu       sync.Mutex
	entries  map[string]cachedParse
	hashSeen map[string]int64
}

func newParseCache() *parseCache {
	return &parseCache{
		entries:  make(map[string]cachedParse),
		hashSeen: make(map[string]int64),
	}
}

// get は mtime が一致するキャッシュを返します（IO はしない）。
func (c *parseCache) get(path string, mod time.Time) (transcriptTail, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[path]
	if ok && e.modTime.Equal(mod) {
		return e.tail, true
	}
	return transcriptTail{}, false
}

// store はパース結果をキャッシュします（IO はしない）。
func (c *parseCache) store(path string, mod time.Time, tail transcriptTail) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[path] = cachedParse{modTime: mod, tail: tail}
}

// stableTimestamp は本文署名 hash に対する初回検出時刻（epoch秒）を返します。
// 同一署名なら以後も同じ値を返し、待機 episode のキーを安定化します。
func (c *parseCache) stableTimestamp(hash string, now time.Time) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.hashSeen[hash]; ok {
		return t
	}
	t := now.Unix()
	c.hashSeen[hash] = t
	return t
}

// prune は現存しないセッションのキャッシュを掃除しメモリ肥大を防ぎます。
func (c *parseCache) prune(alive map[string]struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for path := range c.entries {
		if _, ok := alive[path]; !ok {
			delete(c.entries, path)
		}
	}
}
