package dashboard

// Attention はプロジェクトの注目度を表します
type Attention string

const (
	// AttentionRequired はユーザーの対応が必要な状態を示します
	AttentionRequired Attention = "required"
	// AttentionProgress は処理が進行中の状態を示します
	AttentionProgress Attention = "progress"
	// AttentionWatching は監視のみの状態を示します
	AttentionWatching Attention = "watching"
)

// KanbanCounts はカンバンの各ステータスの件数を表します
type KanbanCounts struct {
	Reviewing int `json:"reviewing"`
	Waiting   int `json:"waiting"`
	Running   int `json:"running"`
	Done      int `json:"done"`
}

// UnansweredQuestion は未回答の確認事項を表します
type UnansweredQuestion struct {
	PlanPath     string `json:"planPath"`
	LineStart    int    `json:"lineStart"`
	LineEnd      int    `json:"lineEnd"`
	QuestionText string `json:"questionText"`
	Heading      string `json:"heading"`
}

// OpsProgress は運用処理の進捗を表します
type OpsProgress struct {
	Index int `json:"index"`
	Total int `json:"total"`
}

// OpsToday は本日の運用実績を表します
type OpsToday struct {
	Count  int `json:"count"`
	Target int `json:"target"`
}

// OpsStats は運用処理の統計を表します
type OpsStats struct {
	Followed int `json:"followed"`
	Already  int `json:"already"`
	Skipped  int `json:"skipped"`
	Error    int `json:"error"`
}

// OpsEntry は1つの運用状態エントリを表します
type OpsEntry struct {
	Account           string         `json:"account"`
	Kind              string         `json:"kind"`
	Status            string         `json:"status"`
	Progress          *OpsProgress   `json:"progress,omitempty"`
	Today             *OpsToday      `json:"today,omitempty"`
	Stats             *OpsStats      `json:"stats,omitempty"`
	ConsecutiveErrors int            `json:"consecutiveErrors"`
	UpdatedAt         string         `json:"updatedAt"`
	Stale             bool           `json:"stale"`
	StaleHours        int            `json:"staleHours"`
	SourceFile        string         `json:"sourceFile"`
	RawExtra          map[string]any `json:"rawExtra,omitempty"`
}

// IdleState は1プロジェクトの質問待ち状態を表します。
// このフィールドが存在すること自体が「質問待ち」を意味します
// （非待機時はキーごと欠落し、waiting bool は持ちません）。
// 経過分はサーバーに載せず、Timestamp からフロントが算出します。
type IdleState struct {
	Timestamp    string `json:"timestamp"`    // RFC3339。バッジの「N分」はフロントが now - timestamp で算出
	Preview      string `json:"preview"`      // rawTail.lastAssistant 先頭80字（要約前の暫定）
	SessionCount int    `json:"sessionCount"` // 同プロジェクトの質問待ちセッション数（代表1件＋件数）
	Summary      string `json:"summary"`      // 「何を待っているか」の日本語1行要約（Phase 1a では空）
	SummarizedAt string `json:"summarizedAt"` // 要約生成時刻（RFC3339・Phase 1a では空）
}

// RunningState は1プロジェクトの動作中（ランタイム）セッション状態を表します。
// このフィールドが存在すること自体が「動作中（青）」を意味します（IdleState=質問待ちとは排他扱い）。
//
// 注意: 名前は同じでも以下の3つの running は別概念です。
//   - KanbanCounts.Running: 開発/実装/実行中/ 配下の .md ファイル件数（カンバンのレーン）
//   - OpsEntry.Status == "running": 運用ジョブが稼働中
//   - ProjectState.Running（本型）: 会話ログ上で Claude が今まさに処理中の代表セッション
//
// 動作中は内容が刻々変わるため要約せず、生 preview のみ保持します（Summary/Timestamp は持たない・W-6）。
type RunningState struct {
	Preview      string `json:"preview"`      // rawTail.lastAssistant 先頭80字（要約前の生text）
	SessionCount int    `json:"sessionCount"` // 同プロジェクトの動作中セッション数（代表1件＋件数）
}

// ProjectState は1つのプロジェクトの集約状態を表します
type ProjectState struct {
	Name       string               `json:"name"`
	Path       string               `json:"path"`
	IsSelf     bool                 `json:"isSelf"`
	Attention  Attention            `json:"attention"`
	Kanban     KanbanCounts         `json:"kanban"`
	Unanswered []UnansweredQuestion `json:"unanswered"`
	Ops        []OpsEntry           `json:"ops"`
	OpsOptedIn bool                 `json:"opsOptedIn"`
	Warnings   []string             `json:"warnings"`
	Idle       *IdleState           `json:"idle,omitempty"`
	Running    *RunningState        `json:"running,omitempty"`
}

// State はダッシュボード全体の状態を表します
type State struct {
	Projects    []ProjectState `json:"projects"`
	GeneratedAt string         `json:"generatedAt"`
}
