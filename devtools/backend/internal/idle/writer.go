package idle

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Writer は質問待ちマーカーへの要約書き戻しを提供します。
// 書き込みは既存マーカーへの summary / summarizedAt の付与のみで、
// マーカーの新規作成・削除はフック側の責務であり本Writerは行いません。
type Writer interface {
	// WriteSummary は sessionID のマーカーに要約を付与します。
	// expectedTimestamp は要約対象を List した時点（T0）のマーカー timestamp です。
	// 書き込みは現行マーカーを1回だけreadし、session_id と timestamp が
	// expectedTimestamp と一致し、ファイルが存在することを確認したうえで
	// temp+rename で行います（compare-and-swap ガード・C3）。
	// 要約実行中（数十秒）にユーザー回答→マーカー削除→同session新timestampで再生成が
	// 起きても、基準が T0 のため新マーカーへ旧要約を上書きしません。
	// 不在/不一致の場合は書き戻しを破棄し、解消済みマーカーを復活させません。
	WriteSummary(sessionID string, expectedTimestamp int64, summary string, at time.Time) error
}

type fileWriter struct {
	markerDir string
}

// NewWriter は markerDir 配下のマーカーに要約を書き戻すWriterを生成します
func NewWriter(markerDir string) Writer {
	return &fileWriter{markerDir: markerDir}
}

// WriteSummary は sessionID のマーカーに summary / summarizedAt を付与します。
func (w *fileWriter) WriteSummary(sessionID string, expectedTimestamp int64, summary string, at time.Time) error {
	path := filepath.Join(w.markerDir, sessionID+".idle")

	// compare-and-swap ガード（C3）:
	// 現行マーカーを1回だけreadし、List時点（T0）の基準と照合する。
	cur, err := readMarker(path)
	if err != nil {
		// 不在（削除済み）または読み取り不能: 解消済みとみなし書き戻しを破棄。
		// 正常な解決フローのためerror扱いにしない。
		log.Printf("[idle] discard summary write (marker gone): session=%s", sessionID)
		return nil
	}
	if cur.SessionID != sessionID || cur.Timestamp != expectedTimestamp {
		// 別の待機に置き換わっている（同session新timestamp含む）: 古い要約で上書きしない
		log.Printf("[idle] discard summary write (marker changed): session=%s", sessionID)
		return nil
	}

	updated := cur
	updated.Summary = summary
	updated.SummarizedAt = at.Format(time.RFC3339)

	data, err := json.MarshalIndent(&updated, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal marker %s: %w", path, err)
	}

	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp marker %s: %w", tmpFile, err)
	}
	if err := os.Rename(tmpFile, path); err != nil {
		if rmErr := os.Remove(tmpFile); rmErr != nil {
			log.Printf("[idle] failed to remove temp marker: path=%s, error=%v", tmpFile, rmErr)
		}
		return fmt.Errorf("failed to rename marker %s: %w", path, err)
	}

	return nil
}

// readMarker は1つのマーカーファイルを読み取りパースします
func readMarker(path string) (Marker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Marker{}, err
	}
	var m Marker
	if err := json.Unmarshal(data, &m); err != nil {
		return Marker{}, fmt.Errorf("invalid marker JSON: %w", err)
	}
	return m, nil
}
