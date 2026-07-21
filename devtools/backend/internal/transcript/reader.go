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

// idleTTL はセッションを終了扱いとする mtime 上限です（dashboard.idleTTL と対応）。
// mtime がこれより古いセッションは走査から除外します。これは「終了したセッションを
// 拾わない」ための粗い liveness ゲート兼性能最適化であり、待機 episode の age/同一性は
// あくまで Marker.Timestamp=entry-time で判定します（C1）。
const idleTTL = 6 * time.Hour

// transcriptReader は会話ログ JSONL を直読みして質問待ちを検出する idle.Reader 実装です。
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

// List は登録プロジェクトの会話ログを走査し、質問待ちセッションを idle.Marker として返します。
// 帰属は各セッションの実 cwd + idle.MatchProject で判定し（C2）、Marker.Timestamp は
// 最後の assistant エントリの entry-time です（C1）。1セッション1マーカーで返し、
// プロジェクトごとの代表選定・SessionCount は下流 attachIdleState に委ねます（無改造流用）。
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
	markers := make([]idle.Marker, 0)
	alive := make(map[string]struct{}, len(sessions))

	for _, sf := range sessions {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		alive[sf.path] = struct{}{}

		// 粗い liveness ゲート: mtime が idleTTL より古いセッションは終了扱いで除外。
		// mtime は age/同一性には使わない（Marker.Timestamp=entry-time・C1）。
		if now.Sub(sf.modTime) > idleTTL {
			continue
		}

		tail, ok := r.tailFor(sf)
		if !ok {
			continue
		}
		if !tail.ParseOK || !tail.IsWaiting {
			continue
		}

		// 帰属は実 cwd + MatchProject（C2）。glob の前方一致は帰属に使わない。
		if _, ok := idle.MatchProject(tail.Cwd, projs); !ok {
			continue
		}

		ts := tail.LastAssistantAt
		if ts == 0 {
			// entry-time 欠落版: 本文署名の初回検出時刻でキーを安定化（raw mtime fallback 禁止）
			ts = r.cache.stableTimestamp(tail.ContentHash, now)
		}

		markers = append(markers, idle.Marker{
			Cwd:          tail.Cwd,
			SessionID:    sf.sessionID,
			Timestamp:    ts,
			RawTail:      idle.RawTail{LastAssistant: tail.LastAssistant, LastPrompt: tail.LastPrompt},
			Summary:      "",
			SummarizedAt: "",
		})
	}

	r.cache.prune(alive)

	// W5: 現存 marker に対応しない孤児キャッシュを掃除する。aliveKeys は今回返す marker の
	// CacheKey 集合。prune 失敗は致命でないため log のみ（要約反映は継続する）。
	aliveKeys := make(map[string]bool, len(markers))
	for _, m := range markers {
		aliveKeys[idle.CacheKey(m.SessionID, m.Timestamp)] = true
	}
	if err := idle.PruneSummaryCache(r.cacheDir, aliveKeys); err != nil {
		log.Printf("[transcript] prune summary cache failed: error=%v", err)
	}

	// C3: 要約マージは List 内で実行し、reader は Summary 込みの完成 Marker を返す契約にする。
	// 下流 attachIdleState/Summarizer は Summary 込み Marker を前提とするため、ここで反映する。
	// key の timestamp=entry-time（C1）なので、要約中にノイズ追記で mtime が動いても同じ key で
	// キャッシュを引ける（mtime ベースなら孤児化する所を回避）。
	markers = idle.MergeSummaries(markers, r.cacheDir)

	return markers, nil
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
