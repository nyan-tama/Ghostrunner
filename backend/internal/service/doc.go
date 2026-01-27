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
// OpenAIService インターフェースがOpenAI Realtime API操作を抽象化する。
// 音声対話機能のためのエフェメラルキー発行を担当する。
//
// # ClaudeService
//
// Claude CLIの実行を担当するサービス。
//
// 主なメソッド:
//   - ExecuteCommand: 任意のスラッシュコマンドを同期実行
//   - ExecuteCommandStream: 任意のスラッシュコマンドをストリーミング実行（SSE用）
//   - ExecutePlan: /plan コマンドを同期実行（後方互換性）
//   - ExecutePlanStream: /plan コマンドをストリーミング実行（後方互換性）
//   - ContinueSession: セッションを継続して回答を送信
//   - ContinueSessionStream: セッション継続をストリーミング実行
//
// # AllowedCommands
//
// 実行可能なスラッシュコマンドのホワイトリスト。
// types.go で定義されている。
//
// 許可コマンド:
//   - plan: 実装計画の作成
//   - fullstack: バックエンド + フロントエンドの実装
//   - go: Go バックエンドのみの実装
//   - nextjs: Next.js フロントエンドのみの実装
//   - discuss: アイデアや構想の対話形式での深掘り
//   - research: 外部情報の調査・収集
//
// # OpenAIService
//
// OpenAI Realtime API用のエフェメラルキー発行を担当するサービス。
// OPENAI_API_KEY 環境変数が設定されていない場合は nil を返し、機能が無効になる。
//
// 主なメソッド:
//   - CreateRealtimeSession: Realtime API用のエフェメラルキーを発行
//
// セッション作成パラメータ:
//   - model: 使用するモデル（未指定時: gpt-4o-realtime-preview-2024-12-17）
//   - voice: 音声タイプ（未指定時: verse）
//
// # 画像サポート
//
// ExecuteCommandとExecuteCommandStreamは画像データを受け取り、
// Claude CLIの--imageオプションとして渡すことが可能。
//
// 画像制約:
//   - 最大枚数: 5枚
//   - 最大サイズ: 1枚あたり5MB
//   - 対応形式: JPEG, PNG, GIF, WebP
//
// 画像はBase64デコード後、一時ファイルとして保存され、
// コマンド実行完了後に自動的に削除される。
//
// # 機能
//
//   - 指定されたプロジェクトディレクトリでClaude CLIを実行
//   - 60分のタイムアウト制御
//   - コンテキストによるキャンセル対応
//   - JSON形式およびstream-json形式の出力パース
//   - AskUserQuestionの質問抽出とセッション継続
//   - コマンドホワイトリストによるバリデーション
//   - 画像データの一時ファイル保存とCLIへの引き渡し
//
// # ストリーミング
//
// ExecuteCommandStreamとContinueSessionStreamはチャンネル経由で
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
// AllowedCommandsによりホワイトリスト方式で実行可能なコマンドを制限する。
//
// # 使用例
//
// インポート:
//
//	import "ghostrunner/backend/internal/service"
//
// 同期実行（汎用コマンド、テキストのみ）:
//
//	svc := service.NewClaudeService()
//	result, err := svc.ExecuteCommand(ctx, "/path/to/project", "fullstack", "implement feature X", nil)
//	if err != nil {
//	    // エラーハンドリング
//	}
//	fmt.Println(result.Output)
//
// 同期実行（画像付き）:
//
//	images := []service.ImageData{
//	    {Name: "screenshot.png", Data: "Base64データ", MimeType: "image/png"},
//	}
//	result, err := svc.ExecuteCommand(ctx, "/path/to/project", "go", "この画像を参考に実装", images)
//	if err != nil {
//	    // エラーハンドリング
//	}
//	fmt.Println(result.Output)
//
// 同期実行（後方互換性）:
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
//	    err := svc.ExecuteCommandStream(ctx, project, "go", args, nil, eventCh)
//	    if err != nil {
//	        log.Printf("Error: %v", err)
//	    }
//	}()
//	for event := range eventCh {
//	    // イベント処理
//	}
package service
