// Package main はgr-run CLIのエントリーポイントです。
// 一括実装（bulk-coding）のワンショット実行を担当します。
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"ghostrunner/backend/internal/grrun"
	"ghostrunner/backend/internal/service"
)

func main() {
	var (
		project  = flag.String("project", "", "対象プロジェクトの絶対パス（必須）")
		task     = flag.String("task", "", "タスクファイル名（必須）")
		locksDir = flag.String("locks-dir", "", "ロックファイルの格納ディレクトリ（デフォルト: ~/.ghostrunner/locks）")
	)
	flag.Parse()

	if *project == "" || *task == "" {
		log.Fatal("[gr-run] --project と --task は必須です")
	}

	// ロックディレクトリのデフォルト値を解決
	if *locksDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("[gr-run] ホームディレクトリの取得に失敗: %v", err)
		}
		*locksDir = filepath.Join(home, ".ghostrunner", "locks")
	}

	cfg := grrun.Config{
		ProjectPath: *project,
		TaskFile:    *task,
		LocksDir:    *locksDir,
	}

	// 通知サービスの初期化（NTFY_TOPIC未設定時はnil）
	var notifier grrun.Notifier
	ntfySvc := service.NewNtfyService()
	if ntfySvc != nil {
		notifier = ntfySvc
	}

	executor := grrun.DefaultExecutor()
	runner := grrun.NewRunner(cfg, notifier, executor)
	result := runner.Run(context.Background())

	log.Printf("[gr-run] result: outcome=%s, message=%s", result.Outcome, result.Message)

	switch result.Outcome {
	case grrun.OutcomeAbnormal:
		os.Exit(1)
	default:
		os.Exit(0)
	}
}
