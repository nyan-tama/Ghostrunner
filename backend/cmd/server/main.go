// Package main はGhostrunner APIサーバーのエントリーポイントです
package main

import (
	"log"

	"ghostrunner/backend/internal/handler"
	"ghostrunner/backend/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("[Server] Starting Ghostrunner API server...")

	// 依存性の組み立て
	claudeService := service.NewClaudeService()
	geminiService := service.NewGeminiService() // nil の場合がある（API キー未設定時）
	openaiService := service.NewOpenAIService() // nil の場合がある（API キー未設定時）
	planHandler := handler.NewPlanHandler(claudeService)
	commandHandler := handler.NewCommandHandler(claudeService)
	geminiHandler := handler.NewGeminiHandler(geminiService)
	openaiHandler := handler.NewOpenAIHandler(openaiService)
	filesHandler := handler.NewFilesHandler()
	projectsHandler := handler.NewProjectsHandler()
	healthHandler := handler.NewHealthHandler()

	// Ginエンジン初期化
	r := gin.Default()

	// CORS設定（ローカル開発時およびTailscale経由のアクセスを許可）
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// localhost を許可
			if origin == "http://localhost:3000" {
				return true
			}
			// Tailscale IP (100.x.x.x) を許可
			if len(origin) > 11 && origin[:11] == "http://100." {
				return true
			}
			// Tailscale Funnel ドメイン (*.ts.net) を許可
			if len(origin) > 7 && origin[len(origin)-7:] == ".ts.net" {
				return true
			}
			return false
		},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	// APIルーティング
	api := r.Group("/api")
	{
		// ヘルスチェックAPI
		api.GET("/health", healthHandler.Handle)

		// ファイル一覧API
		api.GET("/files", filesHandler.Handle)

		// プロジェクト一覧API
		api.GET("/projects", projectsHandler.Handle)

		// 汎用コマンドAPI（推奨）
		api.POST("/command", commandHandler.Handle)
		api.POST("/command/stream", commandHandler.HandleStream)
		api.POST("/command/continue", commandHandler.HandleContinue)
		api.POST("/command/continue/stream", commandHandler.HandleContinueStream)

		// 旧API（互換性維持）
		api.POST("/plan", planHandler.Handle)
		api.POST("/plan/stream", planHandler.HandleStream)
		api.POST("/plan/continue", planHandler.HandleContinue)
		api.POST("/plan/continue/stream", planHandler.HandleContinueStream)

		// Gemini API
		api.POST("/gemini/token", geminiHandler.HandleToken)

		// OpenAI Realtime API
		api.POST("/openai/realtime/session", openaiHandler.HandleSession)
	}

	// サーバー起動（0.0.0.0で全インターフェースからアクセス可能に）
	log.Println("[Server] Listening on 0.0.0.0:8080")
	if err := r.Run("0.0.0.0:8080"); err != nil {
		log.Fatalf("[Server] Failed to start server: %v", err)
	}
}
