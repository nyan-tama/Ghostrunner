package idle

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SummaryCacheEntry は要約キャッシュ1件の永続形式です。
// key（ファイル名）の timestamp は待機 episode の安定同一性（entry-time・C1）で、
// 待機が変われば別 key になるため旧要約が新待機へ復活しません。
type SummaryCacheEntry struct {
	SessionID    string `json:"sessionId"`
	Timestamp    int64  `json:"timestamp"`
	Summary      string `json:"summary"`
	SummarizedAt string `json:"summarizedAt"`
}

// summaryCacheWriter は要約を独立キャッシュ（~/.claude/gr-idle-summaries）へ書き戻す
// idle.Writer 実装です。marker への書き戻し（fileWriter）とは別実装として並存します。
// transcript 方式のセッションには .idle マーカーが無いため、要約は本キャッシュに保存し、
// reader の List が MergeSummaries で読み戻して Marker に反映します（C3）。
type summaryCacheWriter struct {
	cacheDir string
}

// NewSummaryCacheWriter は cacheDir 配下へ要約を書き戻す Writer を生成します。
func NewSummaryCacheWriter(cacheDir string) Writer {
	return &summaryCacheWriter{cacheDir: cacheDir}
}

// WriteSummary は expectedTimestamp（=待機開始 entry-time）を key に埋めて要約を保存します。
// compare-and-swap は key の timestamp で担保します。待機が新質問へ変われば entry-time が
// 前進して別 key になるため、旧要約を新待機へ上書きしません（C1）。書き込みは temp+rename で
// 原子的に行います。
func (w *summaryCacheWriter) WriteSummary(sessionID string, expectedTimestamp int64, summary string, at time.Time) error {
	if err := os.MkdirAll(w.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create summary cache dir %s: %w", w.cacheDir, err)
	}

	path := filepath.Join(w.cacheDir, CacheKey(sessionID, expectedTimestamp)+".json")
	entry := SummaryCacheEntry{
		SessionID:    sessionID,
		Timestamp:    expectedTimestamp,
		Summary:      summary,
		SummarizedAt: at.Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(&entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary cache entry %s: %w", path, err)
	}

	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp summary cache %s: %w", tmpFile, err)
	}
	if err := os.Rename(tmpFile, path); err != nil {
		if rmErr := os.Remove(tmpFile); rmErr != nil {
			log.Printf("[idle] failed to remove temp summary cache: path=%s, error=%v", tmpFile, rmErr)
		}
		return fmt.Errorf("failed to rename summary cache %s: %w", path, err)
	}

	return nil
}

// MergeSummaries は各 marker に対応するキャッシュ（<sessionID>_<timestamp>.json）を読み、
// Summary / SummarizedAt を反映した新しいスライスをイミュータブルに返します（元 markers は変更しません）。
// キャッシュ不在・壊れ JSON は Summary 空のまま skip します（保守的フォールバック）。
// List 内で呼ぶことで reader は Summary 込みの完成 Marker を返す契約を満たします（C3）。
func MergeSummaries(markers []Marker, cacheDir string) []Marker {
	out := make([]Marker, len(markers))
	for i, m := range markers {
		out[i] = m
		entry, ok := readSummaryCache(cacheDir, m.SessionID, m.Timestamp)
		if !ok {
			continue
		}
		out[i].Summary = entry.Summary
		out[i].SummarizedAt = entry.SummarizedAt
	}
	return out
}

// PruneSummaryCache は現存 marker（aliveKeys）に対応しないキャッシュファイルを削除します（W5）。
// aliveKeys は CacheKey(sessionID, timestamp) の集合で、reader が今回返す marker から構築します。
// 待機が解消・変化して孤児化したキャッシュを掃除し、ディスク肥大と古い要約の残留を防ぎます。
func PruneSummaryCache(cacheDir string, aliveKeys map[string]bool) error {
	paths, err := filepath.Glob(filepath.Join(cacheDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to glob summary cache %s: %w", cacheDir, err)
	}
	for _, path := range paths {
		key := strings.TrimSuffix(filepath.Base(path), ".json")
		if aliveKeys[key] {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Printf("[idle] failed to prune summary cache: path=%s, error=%v", path, err)
		}
	}
	return nil
}

// CacheKey は要約キャッシュのファイル名（拡張子除く）を返します。
// sessionID は UUID（英数+ハイフン）想定ですが、パス安全化のため念のため sanitize します。
// 書き込み（WriteSummary）・読み取り（MergeSummaries）・掃除（PruneSummaryCache）で
// 同一のキー生成を共有し、reader が aliveKeys を組む際にも本関数を使います。
func CacheKey(sessionID string, timestamp int64) string {
	return fmt.Sprintf("%s_%d", sanitizeSessionID(sessionID), timestamp)
}

// sanitizeSessionID はファイル名に使えない文字を "-" に置換します。
func sanitizeSessionID(sessionID string) string {
	var b strings.Builder
	b.Grow(len(sessionID))
	for _, r := range sessionID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}

// readSummaryCache は marker に対応するキャッシュを読みます。
// 不在・壊れ JSON は (空, false) を返し、呼び出し側は Summary 空のまま扱います。
func readSummaryCache(cacheDir, sessionID string, timestamp int64) (SummaryCacheEntry, bool) {
	path := filepath.Join(cacheDir, CacheKey(sessionID, timestamp)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return SummaryCacheEntry{}, false
	}
	var entry SummaryCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		log.Printf("[idle] skip summary cache (invalid JSON): path=%s, error=%v", path, err)
		return SummaryCacheEntry{}, false
	}
	return entry, true
}
