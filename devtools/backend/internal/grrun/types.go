package grrun

import "time"

// Config はgr-runの実行設定を保持します
type Config struct {
	// ProjectPath は対象プロジェクトの絶対パス
	ProjectPath string
	// TaskFile はタスクファイル名（実装待ちディレクトリ内のファイル名）
	TaskFile string
	// LocksDir はflockファイルの格納ディレクトリ（デフォルト: ~/.ghostrunner/locks）
	LocksDir string
}

// Outcome はClaude実行後の結果分類を表します
type Outcome string

const (
	// OutcomeCompleted はタスクが正常に完了したことを示します
	OutcomeCompleted Outcome = "completed"
	// OutcomeWaitingAnswer は確認事項が未回答であることを示します
	OutcomeWaitingAnswer Outcome = "waiting_answer"
	// OutcomeAbnormal は異常終了を示します
	OutcomeAbnormal Outcome = "abnormal"
	// OutcomeNeedsCheck はフォーマット不一致などで人手確認が必要なことを示します
	OutcomeNeedsCheck Outcome = "needs_check"
	// OutcomeLockBusy は既に他のプロセスが実行中であることを示します
	OutcomeLockBusy Outcome = "lock_busy"
)

// RunResult はgr-run実行の結果を保持します
type RunResult struct {
	// Outcome は結果分類
	Outcome Outcome
	// Message は結果の詳細メッセージ
	Message string
}

// カンバンディレクトリのパス（プロジェクトルートからの相対パス）
const (
	RelWaiting = "開発/実装/実装待ち"
	RelRunning = "開発/実装/実行中"
	RelDone    = "開発/実装/完了"
)

// UnansweredPattern は確認事項の未回答を検出する正規表現パターンです。
// chief-director.md の SSOT パターンと一致させること。
const UnansweredPattern = `\*\*ステータス\*\*:\s*未回答`

// ClaudeTimeout はClaude実行のタイムアウト時間です
const ClaudeTimeout = 60 * time.Minute
