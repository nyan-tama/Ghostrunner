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
// PlanHandler が /api/plan 関連のエンドポイントを処理する。
// ClaudeServiceへの依存性注入によりテスタビリティを確保する。
//
// # PlanHandler
//
// Claude CLIの /plan コマンドを実行するエンドポイント群。
//
// # エンドポイント一覧
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
// リクエスト: /api/plan と同じ
// レスポンス: Server-Sent Events形式でStreamEventを送信
//
// POST /api/plan/continue - セッション継続
//
// リクエスト:
//
//	{
//	    "project": "/path/to/project",  // 対象プロジェクトの絶対パス (必須)
//	    "session_id": "session-xxx",    // セッションID (必須)
//	    "answer": "yes"                 // ユーザーの回答 (必須)
//	}
//
// POST /api/plan/continue/stream - セッション継続のストリーミング実行 (SSE)
//
// リクエスト: /api/plan/continue と同じ
// レスポンス: Server-Sent Events形式でStreamEventを送信
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
// # HTTPステータスコード
//
//   - 200 OK: 正常完了
//   - 400 Bad Request: リクエスト不正、バリデーションエラー
//   - 500 Internal Server Error: Claude CLI実行エラー
//
// # 使用例
//
//	claudeService := service.NewClaudeService()
//	planHandler := handler.NewPlanHandler(claudeService)
//	api := router.Group("/api")
//	api.POST("/plan", planHandler.Handle)
//	api.POST("/plan/stream", planHandler.HandleStream)
//	api.POST("/plan/continue", planHandler.HandleContinue)
//	api.POST("/plan/continue/stream", planHandler.HandleContinueStream)
package handler
