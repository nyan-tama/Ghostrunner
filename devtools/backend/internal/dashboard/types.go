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
}

// State はダッシュボード全体の状態を表します
type State struct {
	Projects    []ProjectState `json:"projects"`
	GeneratedAt string         `json:"generatedAt"`
}
