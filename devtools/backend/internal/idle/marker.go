package idle

import "context"

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

// Reader は質問待ちマーカーの読み取りを提供します。
// 現行の実装は会話ログ直読み方式（transcript パッケージ）です。
type Reader interface {
	// List は全マーカーを読み取ります。壊れたファイルはスキップします。
	List(ctx context.Context) ([]Marker, error)
}
