// Package service はビジネスロジックを提供します
package service

import "time"

// 巡回の並列実行数とポーリング間隔
const (
	// MaxParallelSlots は同時実行可能なプロジェクト数
	MaxParallelSlots = 5
	// PollingInterval はポーリングの間隔
	PollingInterval = 5 * time.Minute
)

// PatrolStatus はプロジェクトの巡回状態を表します
type PatrolStatus string

const (
	StatusIdle            PatrolStatus = "idle"
	StatusRunning         PatrolStatus = "running"
	StatusWaitingApproval PatrolStatus = "waiting_approval"
	StatusQueued          PatrolStatus = "queued"
	StatusCompleted       PatrolStatus = "completed"
	StatusError           PatrolStatus = "error"
)

// PatrolProject は巡回対象のプロジェクトを表します
type PatrolProject struct {
	Path string `json:"path"` // プロジェクトの絶対パス
	Name string `json:"name"` // プロジェクト名（ディレクトリ名）
}

// ProjectState はプロジェクトの実行状態を表します
type ProjectState struct {
	Project      PatrolProject `json:"project"`                // プロジェクト情報
	Status       PatrolStatus  `json:"status"`                 // 現在の状態
	SessionID    string        `json:"sessionId,omitempty"`    // Claude CLIのセッションID
	Question     *Question     `json:"question,omitempty"`     // 承認待ちの質問（単数）
	GitLog       string        `json:"gitLog,omitempty"`       // 直近のgit log
	PendingTasks []string      `json:"pendingTasks,omitempty"` // 未処理タスクのファイル名一覧
	Error        string        `json:"error,omitempty"`        // エラーメッセージ
	StartedAt    *time.Time    `json:"startedAt,omitempty"`    // 実行開始時刻
	UpdatedAt    *time.Time    `json:"updatedAt,omitempty"`    // 最終更新時刻
}

// PatrolEvent はSSE配信用のイベントを表します
type PatrolEvent struct {
	Type        string        `json:"type"`                  // イベントタイプ
	ProjectPath string        `json:"projectPath,omitempty"` // 対象プロジェクトのパス
	State       *ProjectState `json:"state,omitempty"`       // プロジェクトの状態
	Message     string        `json:"message,omitempty"`     // メッセージ
}

// PatrolEventType はSSEイベントのタイプを定義します
const (
	PatrolEventProjectStarted   = "project_started"
	PatrolEventProjectQuestion  = "project_question"
	PatrolEventProjectCompleted = "project_completed"
	PatrolEventProjectError     = "project_error"
	PatrolEventScanCompleted    = "scan_completed"
)

// ScanResult はプロジェクトスキャン結果を表します
type ScanResult struct {
	Project      PatrolProject `json:"project"`                // プロジェクト情報
	GitLog       string        `json:"gitLog,omitempty"`       // 直近のgit log
	PendingTasks []string      `json:"pendingTasks,omitempty"` // 未処理タスクのファイル名一覧
}

// PatrolConfig は巡回設定の永続化用構造体です
type PatrolConfig struct {
	Projects []PatrolProject `json:"projects"`
}
