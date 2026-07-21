package idle

import "context"

// RawTail はマーカー書き込み時点の会話末尾（要約前の生テキスト）を表します
type RawTail struct {
	LastAssistant string `json:"lastAssistant"`
	LastPrompt    string `json:"lastPrompt"`
}

// Status はセッションの状態種別を表します。
// reader（transcript）が mtime 鮮度を合成して確定し、下流 dashboard が
// waiting→IdleState / running→RunningState へディスパッチします。
type Status string

const (
	// StatusWaiting は質問待ち（未応答 assistant が滞留）を表します。
	StatusWaiting Status = "waiting"
	// StatusRunning は動作中（Claude が処理中の代表セッション）を表します。
	StatusRunning Status = "running"
)

// Marker は1セッションの代表状態マーカーを表します。
// reader がプロジェクト毎に最新 mtime の代表1件へ collapse して返します。
// Timestamp の意味は Status で分岐します: waiting は待機開始 entry-time（要約 key の同一性用・C1）、
// running は代表セッションの mtime です（epoch 秒）。
type Marker struct {
	Cwd          string  `json:"cwd"`
	SessionID    string  `json:"session_id"`
	Timestamp    int64   `json:"timestamp"`
	Status       Status  `json:"status,omitempty"`
	SessionCount int     `json:"sessionCount,omitempty"`
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
