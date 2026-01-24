// Package service はGhostrunner APIサーバーのビジネスロジックを提供する。
//
// # 概要
//
// このパッケージはClaude CLIとの連携機能を提供する。
// 外部プロセスとしてClaude CLIを実行し、その結果を返却する。
//
// # 主要なコンポーネント
//
// ClaudeService インターフェースがClaude CLI操作を抽象化する。
// handlerパッケージから利用され、依存性の注入を可能にする。
//
// # ClaudeService
//
// Claude CLIの実行を担当するサービス。
// ExecutePlanメソッドで /plan コマンドを実行する。
//
// 主な機能:
//   - 指定されたプロジェクトディレクトリでClaude CLIを実行
//   - 60分のタイムアウト制御
//   - コンテキストによるキャンセル対応
//   - 標準出力と標準エラー出力の結合
//
// # セキュリティ
//
// シェル経由ではなく直接exec.Commandを使用することで、
// コマンドインジェクション攻撃を防止する。
//
// # 使用例
//
//	svc := service.NewClaudeService()
//	output, err := svc.ExecutePlan(ctx, "/path/to/project", "implement feature X")
//	if err != nil {
//	    // エラーハンドリング
//	}
//	fmt.Println(output)
package service
