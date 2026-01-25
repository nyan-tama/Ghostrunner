// Package handler はGhostrunner APIサーバーのHTTPハンドラーを提供する。
//
// # 概要
//
// このパッケージはGin Frameworkを使用したHTTPリクエストハンドラーを提供する。
// 各ハンドラーはserviceパッケージのビジネスロジックを呼び出し、
// HTTPレスポンスを返却する。
//
// # 主要なコンポーネント
//
//   - PlanHandler: /api/plan 関連のエンドポイントを処理（後方互換性維持）
//   - CommandHandler: /api/command 関連のエンドポイントを処理（汎用コマンド実行）
//   - FilesHandler: /api/files 関連のエンドポイントを処理（ファイル一覧取得）
//
// ClaudeServiceへの依存性注入によりテスタビリティを確保する。
//
// # CommandHandler
//
// Claude CLIの任意のスラッシュコマンドを実行するエンドポイント群。
// 許可されたコマンドのみ実行可能。
//
// 許可コマンド:
//   - plan: 実装計画の作成
//   - fullstack: バックエンド + フロントエンドの実装
//   - go: Go バックエンドのみの実装
//   - nextjs: Next.js フロントエンドのみの実装
//   - discuss: アイデアや構想の対話形式での深掘り
//
// # FilesHandler
//
// プロジェクトの開発フォルダ内のmdファイル一覧を取得するエンドポイント。
// 外部サービスへの依存がなく、ローカルファイルシステムのみを参照する。
//
// スキャン対象フォルダ:
//   - 実装/実装待ち
//   - 実装/完了
//   - 検討中
//   - 資料
//   - アーカイブ
//
// # PlanHandler
//
// Claude CLIの /plan コマンドを実行するエンドポイント群。
// 内部的にCommandHandlerと同じサービスを使用する。
// 後方互換性のために維持している。
//
// # エンドポイント一覧
//
// ## Files API (ファイル一覧取得)
//
// GET /api/files - 開発フォルダ内のmdファイル一覧取得
//
// リクエスト:
//
//	GET /api/files?project=/path/to/project
//
// レスポンス:
//
//	{
//	    "success": true,
//	    "files": {
//	        "実装/実装待ち": [{"name": "plan.md", "path": "開発/実装/実装待ち/plan.md"}],
//	        "実装/完了": [],
//	        "検討中": [],
//	        "資料": [],
//	        "アーカイブ": []
//	    }
//	}
//
// ## Command API (汎用コマンド実行)
//
// POST /api/command - コマンドの同期実行
//
// リクエスト:
//
//	{
//	    "project": "/path/to/project",  // 対象プロジェクトの絶対パス (必須)
//	    "command": "fullstack",         // 実行するコマンド (必須)
//	    "args": "implement feature X"   // コマンドの引数 (必須)
//	}
//
// POST /api/command/stream - コマンドのストリーミング実行 (SSE)
//
// リクエスト: /api/command と同じ
// レスポンス: Server-Sent Events形式でStreamEventを送信
//
// POST /api/command/continue - セッション継続
//
// リクエスト:
//
//	{
//	    "project": "/path/to/project",  // 対象プロジェクトの絶対パス (必須)
//	    "session_id": "session-xxx",    // セッションID (必須)
//	    "answer": "yes"                 // ユーザーの回答 (必須)
//	}
//
// POST /api/command/continue/stream - セッション継続のストリーミング実行 (SSE)
//
// リクエスト: /api/command/continue と同じ
// レスポンス: Server-Sent Events形式でStreamEventを送信
//
// ## Plan API (後方互換性)
//
// POST /api/plan - /planコマンドの同期実行
//
// リクエスト:
//
//	{
//	    "project": "/path/to/project",  // 対象プロジェクトの絶対パス (必須)
//	    "args": "implement feature X"   // /planコマンドの引数 (必須)
//	}
//
// POST /api/plan/stream - /planコマンドのストリーミング実行 (SSE)
//
// POST /api/plan/continue - セッション継続
//
// POST /api/plan/continue/stream - セッション継続のストリーミング実行 (SSE)
//
// # レスポンス形式
//
// 同期エンドポイント (成功時):
//
//	{
//	    "success": true,
//	    "session_id": "session-xxx",
//	    "output": "Claude CLIの出力結果",
//	    "questions": [...],      // 質問がある場合
//	    "completed": true,       // 完了したかどうか
//	    "cost_usd": 0.01         // コスト
//	}
//
// 同期エンドポイント (エラー時):
//
//	{
//	    "success": false,
//	    "error": "エラーメッセージ"
//	}
//
// # バリデーション
//
// validateProjectPath 関数がプロジェクトパスを検証する。
//
// 検証項目:
//   - 空でないこと
//   - 絶対パスであること
//   - 存在するディレクトリであること
//
// CommandHandler はさらにコマンドのバリデーションを行う。
//
// コマンド検証項目:
//   - 空でないこと
//   - AllowedCommands に含まれること
//
// # HTTPステータスコード
//
//   - 200 OK: 正常完了
//   - 400 Bad Request: リクエスト不正、バリデーションエラー、許可されていないコマンド
//   - 404 Not Found: リソースが存在しない（/api/filesで開発ディレクトリが存在しない場合）
//   - 500 Internal Server Error: Claude CLI実行エラー、ファイルシステムエラー
//
// # 使用例
//
// インポート:
//
//	import (
//	    "ghostrunner/backend/internal/handler"
//	    "ghostrunner/backend/internal/service"
//	)
//
// ハンドラーの初期化とルーティング:
//
//	claudeService := service.NewClaudeService()
//
//	// CommandHandler
//	commandHandler := handler.NewCommandHandler(claudeService)
//	api := router.Group("/api")
//	api.POST("/command", commandHandler.Handle)
//	api.POST("/command/stream", commandHandler.HandleStream)
//	api.POST("/command/continue", commandHandler.HandleContinue)
//	api.POST("/command/continue/stream", commandHandler.HandleContinueStream)
//
//	// PlanHandler (後方互換性)
//	planHandler := handler.NewPlanHandler(claudeService)
//	api.POST("/plan", planHandler.Handle)
//	api.POST("/plan/stream", planHandler.HandleStream)
//	api.POST("/plan/continue", planHandler.HandleContinue)
//	api.POST("/plan/continue/stream", planHandler.HandleContinueStream)
//
//	// FilesHandler
//	filesHandler := handler.NewFilesHandler()
//	api.GET("/files", filesHandler.Handle)
package handler
