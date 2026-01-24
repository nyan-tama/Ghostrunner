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
// PlanHandler が /api/plan エンドポイントを処理する。
// ClaudeServiceへの依存性注入によりテスタビリティを確保する。
//
// # PlanHandler
//
// Claude CLIの /plan コマンドを実行するエンドポイント。
//
// エンドポイント: POST /api/plan
//
// リクエスト:
//
//	{
//	    "project": "/path/to/project",  // 対象プロジェクトの絶対パス (必須)
//	    "args": "implement feature X"   // /planコマンドの引数 (必須)
//	}
//
// レスポンス (成功時):
//
//	{
//	    "success": true,
//	    "output": "Claude CLIの出力結果"
//	}
//
// レスポンス (エラー時):
//
//	{
//	    "success": false,
//	    "output": "部分的な出力 (あれば)",
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
//	router.POST("/api/plan", planHandler.Handle)
package handler
