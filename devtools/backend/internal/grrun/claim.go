package grrun

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

// unansweredRe は確認事項の未回答パターンを検出する正規表現です
var unansweredRe = regexp.MustCompile(UnansweredPattern)

// ClaimTask はタスクファイルを実装待ちから実行中へ移動します。
// gitコミットは行いません（/coding ブランチとの干渉を避けるため）。
func ClaimTask(projectPath, taskFile string) error {
	runningDir := filepath.Join(projectPath, RelRunning)
	if err := os.MkdirAll(runningDir, 0755); err != nil {
		return fmt.Errorf("failed to create running directory %s: %w", runningDir, err)
	}

	src := filepath.Join(projectPath, RelWaiting, taskFile)
	dst := filepath.Join(runningDir, taskFile)

	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("failed to move task %s to running: %w", taskFile, err)
	}

	return nil
}

// ClassifyResult はClaude実行後のタスク状態を分類します。
// ワーキングツリーのファイル位置と内容に基づいて判定します。
func ClassifyResult(projectPath, taskFile string, exitCode int) Outcome {
	donePath := filepath.Join(projectPath, RelDone, taskFile)
	runningPath := filepath.Join(projectPath, RelRunning, taskFile)

	// 完了ディレクトリに移動済み
	if fileExists(donePath) {
		return OutcomeCompleted
	}

	// 実行中ディレクトリに残っている場合
	if fileExists(runningPath) {
		// 未回答の確認事項があればWaitingAnswer（exitCodeより優先）
		if hasUnansweredQuestion(runningPath) {
			return OutcomeWaitingAnswer
		}
		if exitCode != 0 {
			return OutcomeAbnormal
		}
		// exitCode==0 だが完了ディレクトリに移動されていない
		return OutcomeNeedsCheck
	}

	// どちらのディレクトリにもない（予期しない状態）
	return OutcomeAbnormal
}

// hasUnansweredQuestion はプランファイルに未回答の確認事項があるかを判定します。
// ファイルが存在しない場合はfalseを返します（前方互換性のため）。
func hasUnansweredQuestion(planPath string) bool {
	data, err := os.ReadFile(planPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[gr-run] failed to read plan file %s: %v", planPath, err)
		}
		return false
	}
	return unansweredRe.Match(data)
}

// fileExists はファイルが存在するかを返します
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
