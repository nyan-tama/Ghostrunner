package grrun

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// Notifier は通知送信のインターフェースです。
// service.NtfyService と同じシグネチャだが、直接依存を避けるために再定義しています。
type Notifier interface {
	Notify(title, message string)
	NotifyError(title, message string)
}

// CommandExecutor はClaude CLIの実行を抽象化する関数型です。
// テスト時に差し替え可能にするために型定義しています。
type CommandExecutor func(ctx context.Context, projectPath, taskFile string) (exitCode int, err error)

// Runner はgr-runのメイン実行ロジックを保持します
type Runner struct {
	cfg      Config
	notifier Notifier
	executor CommandExecutor
	lockFile *os.File // flockのfdをGC防止のため保持
}

// NewRunner は新しいRunnerを生成します。
// notifierはnilを許容します（通知なしで動作）。
func NewRunner(cfg Config, notifier Notifier, executor CommandExecutor) *Runner {
	return &Runner{
		cfg:      cfg,
		notifier: notifier,
		executor: executor,
	}
}

// DefaultExecutor はClaude CLIを実行するデフォルトのCommandExecutorを返します
func DefaultExecutor() CommandExecutor {
	return func(ctx context.Context, projectPath, taskFile string) (int, error) {
		prompt := fmt.Sprintf("/coding @%s", filepath.Join(RelRunning, taskFile))
		args := []string{
			"-p", prompt,
			"--permission-mode", "bypassPermissions",
		}

		cmd := exec.CommandContext(ctx, "claude", args...)
		cmd.Dir = projectPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return -1, fmt.Errorf("failed to start claude process: %w", err)
		}

		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr.ExitCode(), nil
			}
			return -1, fmt.Errorf("failed to wait for claude process: %w", err)
		}

		return 0, nil
	}
}

// Run はgr-runのメイン処理を実行します。
// ロック取得 -> タスククレーム -> Claude実行 -> 結果分類 -> 通知 の順に処理します。
func (r *Runner) Run(ctx context.Context) RunResult {
	ctx, cancel := context.WithTimeout(ctx, ClaudeTimeout)
	defer cancel()

	projectPath := r.cfg.ProjectPath
	taskFile := r.cfg.TaskFile

	log.Printf("[gr-run] started: project=%s, task=%s", projectPath, taskFile)

	// ロック取得
	lockFile, acquired, err := AcquireLock(r.cfg.LocksDir, projectPath)
	if err != nil {
		msg := fmt.Sprintf("ロック取得に失敗: %v", err)
		log.Printf("[gr-run] lock error: %v", err)
		r.notifyError("gr-run: ロック失敗", msg)
		return RunResult{Outcome: OutcomeAbnormal, Message: msg}
	}
	if !acquired {
		msg := fmt.Sprintf("他のプロセスが実行中です: %s", projectPath)
		log.Printf("[gr-run] lock busy: %s", projectPath)
		return RunResult{Outcome: OutcomeLockBusy, Message: msg}
	}
	r.lockFile = lockFile
	defer func() {
		r.lockFile.Close()
		r.lockFile = nil
	}()

	// タスクをクレーム（実装待ち -> 実行中）
	if err := ClaimTask(projectPath, taskFile); err != nil {
		msg := fmt.Sprintf("タスクの移動に失敗: %v", err)
		log.Printf("[gr-run] claim failed: %v", err)
		r.notifyError("gr-run: タスク移動失敗", msg)
		return RunResult{Outcome: OutcomeAbnormal, Message: msg}
	}
	log.Printf("[gr-run] task claimed: %s -> %s", RelWaiting, RelRunning)

	// Claude実行
	exitCode, err := r.executor(ctx, projectPath, taskFile)
	if err != nil {
		log.Printf("[gr-run] executor error: %v (exitCode=%d)", err, exitCode)
		if exitCode == -1 {
			msg := fmt.Sprintf("Claude起動に失敗: %v", err)
			r.notifyError("gr-run: Claude起動失敗", msg)
			return RunResult{Outcome: OutcomeAbnormal, Message: msg}
		}
	}
	log.Printf("[gr-run] claude finished: exitCode=%d", exitCode)

	// 結果分類
	outcome := ClassifyResult(projectPath, taskFile, exitCode)
	result := r.buildResult(outcome, taskFile)

	// 通知
	r.sendNotification(outcome, taskFile, result.Message)

	log.Printf("[gr-run] completed: outcome=%s, task=%s", outcome, taskFile)
	return result
}

// buildResult はOutcomeからRunResultを構築します
func (r *Runner) buildResult(outcome Outcome, taskFile string) RunResult {
	switch outcome {
	case OutcomeCompleted:
		return RunResult{
			Outcome: outcome,
			Message: fmt.Sprintf("タスク完了: %s", taskFile),
		}
	case OutcomeWaitingAnswer:
		return RunResult{
			Outcome: outcome,
			Message: fmt.Sprintf("確認事項あり（未回答）: %s", taskFile),
		}
	case OutcomeAbnormal:
		return RunResult{
			Outcome: outcome,
			Message: fmt.Sprintf("異常終了: %s", taskFile),
		}
	case OutcomeNeedsCheck:
		return RunResult{
			Outcome: outcome,
			Message: fmt.Sprintf("完了ディレクトリ未移動（フォーマット不一致の可能性）: %s", taskFile),
		}
	default:
		return RunResult{
			Outcome: OutcomeAbnormal,
			Message: fmt.Sprintf("不明な結果: %s", taskFile),
		}
	}
}

// sendNotification はOutcomeに応じた通知を送信します
func (r *Runner) sendNotification(outcome Outcome, taskFile, message string) {
	if r.notifier == nil {
		return
	}

	title := fmt.Sprintf("gr-run: %s", taskFile)
	switch outcome {
	case OutcomeAbnormal:
		r.notifier.NotifyError(title, message)
	case OutcomeLockBusy:
		// 通知しない
	default:
		r.notifier.Notify(title, message)
	}
}

// notifyError はエラー通知を送信するヘルパーです
func (r *Runner) notifyError(title, message string) {
	if r.notifier == nil {
		return
	}
	r.notifier.NotifyError(title, message)
}
