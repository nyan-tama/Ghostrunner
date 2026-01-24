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

	// Ginエンジン初期化
	r := gin.Default()

	// APIルーティング
	api := r.Group("/api")
	{
		api.POST("/plan", planHandler.Handle)
	}

	// 静的ファイル配信
	r.StaticFile("/", "./web/index.html")
	r.Static("/web", "./web")

	// サーバー起動
	log.Println("[Server] Listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("[Server] Failed to start server: %v", err)
	}
}
