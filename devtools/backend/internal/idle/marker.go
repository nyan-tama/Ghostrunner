package idle

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// RawTail はマーカー書き込み時点の会話末尾（要約前の生テキスト）を表します
type RawTail struct {
	LastAssistant string `json:"lastAssistant"`
	LastPrompt    string `json:"lastPrompt"`
}

// Marker は1セッションの質問待ちマーカーを表します。
// Timestamp はフックが date +%s で書き込むため epoch 秒（JSON number）です。
type Marker struct {
	Cwd          string  `json:"cwd"`
	SessionID    string  `json:"session_id"`
	Timestamp    int64   `json:"timestamp"`
	RawTail      RawTail `json:"rawTail"`
	Summary      string  `json:"summary"`
	SummarizedAt string  `json:"summarizedAt"`
}

// Reader は質問待ちマーカーの読み取りを提供します
type Reader interface {
	// List は全マーカーを読み取ります。壊れたファイルはスキップします。
	List(ctx context.Context) ([]Marker, error)
}

type fileReader struct {
	markerDir string
}

// NewReader は markerDir 配下の *.idle を読むReaderを生成します
func NewReader(markerDir string) Reader {
	return &fileReader{markerDir: markerDir}
}

// List は markerDir/*.idle を Glob→ReadFile→Unmarshal で読み取ります。
// markerDir が存在しない場合は空スライスを返します。
// 壊れたJSON・読み取り失敗のファイルは warning ログを出してスキップし、
// 全体を失敗させません。
func (r *fileReader) List(ctx context.Context) ([]Marker, error) {
	pattern := filepath.Join(r.markerDir, "*.idle")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob markers %s: %w", pattern, err)
	}

	markers := make([]Marker, 0, len(paths))
	for _, path := range paths {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[idle] skip marker (read failed): path=%s, error=%v", path, err)
			continue
		}

		var m Marker
		if err := json.Unmarshal(data, &m); err != nil {
			log.Printf("[idle] skip marker (invalid JSON): path=%s, error=%v", path, err)
			continue
		}

		markers = append(markers, m)
	}

	return markers, nil
}
