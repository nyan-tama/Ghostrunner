package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"{{PROJECT_NAME}}/backend/internal/handler"
)

func main() {
	// ハンドラーの初期化
	healthHandler := handler.NewHealthHandler()
	helloHandler := handler.NewHelloHandler()

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

	// ルーティング
	api := r.Group("/api")
	{
		api.GET("/health", healthHandler.Handle)
		api.GET("/hello", helloHandler.Handle)
	}

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
