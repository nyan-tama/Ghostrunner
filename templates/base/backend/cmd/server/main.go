package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"{{PROJECT_NAME}}/backend/internal/registry"
)

func main() {
	// 機能の初期化（DB接続など、各 registry で登録された処理を実行）
	if err := registry.RunInit(); err != nil {
		log.Fatalf("[Server] Init failed: %v", err)
	}
	defer registry.RunCleanup()

	// Gin エンジン初期化
	r := gin.Default()

	// CORS 設定
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{allowedOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	// ルーティング（各 registry で登録されたルートを一括設定）
	api := r.Group("/api")
	registry.SetupRoutes(api)

	// サーバー起動
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("[Server] Starting...")
	log.Printf("[Server] Listening on 0.0.0.0:%s", port)
	if err := r.Run("0.0.0.0:" + port); err != nil {
		log.Fatalf("[Server] Failed to start: %v", err)
	}
}
