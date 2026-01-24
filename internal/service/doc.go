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
//
// 主なメソッド:
//   - ExecutePlan: /plan コマンドを同期実行
//   - ExecutePlanStream: /plan コマンドをストリーミング実行（SSE用）
//   - ContinueSession: セッションを継続して回答を送信
//   - ContinueSessionStream: セッション継続をストリーミング実行
//
// # 機能
//
//   - 指定されたプロジェクトディレクトリでClaude CLIを実行
//   - 60分のタイムアウト制御
//   - コンテキストによるキャンセル対応
//   - JSON形式およびstream-json形式の出力パース
//   - AskUserQuestionの質問抽出とセッション継続
//
// # ストリーミング
//
// ExecutePlanStreamとContinueSessionStreamはチャンネル経由で
// StreamEventを送信する。イベントタイプ:
//   - init: セッション開始
//   - thinking: 思考中
//   - tool_use: ツール使用
//   - text: テキスト出力
//   - question: 質問
//   - complete: 完了
//   - error: エラー
//
// # セキュリティ
//
// シェル経由ではなく直接exec.Commandを使用することで、
// コマンドインジェクション攻撃を防止する。
//
// # 使用例
//
// 同期実行:
//
//	svc := service.NewClaudeService()
//	result, err := svc.ExecutePlan(ctx, "/path/to/project", "implement feature X")
//	if err != nil {
//	    // エラーハンドリング
//	}
//	fmt.Println(result.Output)
//
// ストリーミング実行:
//
//	eventCh := make(chan service.StreamEvent, 100)
//	go func() {
//	    err := svc.ExecutePlanStream(ctx, project, args, eventCh)
//	    if err != nil {
//	        log.Printf("Error: %v", err)
//	    }
//	}()
//	for event := range eventCh {
//	    // イベント処理
//	}
package service
