// Package main はGhostrunner APIサーバーのエントリーポイントです
package main

import (
	"log"

	"ghostrunner/internal/handler"
	"ghostrunner/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("[Server] Starting Ghostrunner API server...")

	// 依存性の組み立て
	claudeService := service.NewClaudeService()
	planHandler := handler.NewPlanHandler(claudeService)
	commandHandler := handler.NewCommandHandler(claudeService)

	// Ginエンジン初期化
	r := gin.Default()

	// APIルーティング
	api := r.Group("/api")
	{
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
	}

	// 静的ファイル配信
	r.StaticFile("/", "./web/index.html")
	r.Static("/web", "./web")

	// サーバー起動（0.0.0.0で全インターフェースからアクセス可能に）
	log.Println("[Server] Listening on 0.0.0.0:8080")
	if err := r.Run("0.0.0.0:8080"); err != nil {
		log.Fatalf("[Server] Failed to start server: %v", err)
	}
}
