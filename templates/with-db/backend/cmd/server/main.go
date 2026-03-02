package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"{{PROJECT_NAME}}/backend/internal/domain/model"
	"{{PROJECT_NAME}}/backend/internal/handler"
	"{{PROJECT_NAME}}/backend/internal/infrastructure"
)

func main() {
	// DB接続
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("[Server] DATABASE_URL is required")
	}

	database, err := infrastructure.NewDatabase(databaseURL)
	if err != nil {
		log.Fatalf("[Server] Failed to connect to database: %v", err)
	}
	defer database.Close()

	// AutoMigrate
	if err := database.AutoMigrate(&model.Sample{}); err != nil {
		log.Fatalf("[Server] Failed to run migrations: %v", err)
	}
	log.Println("[Server] Database migration completed")

	// ハンドラーの初期化
	healthHandler := handler.NewHealthHandler()
	helloHandler := handler.NewHelloHandler()
	sampleHandler := handler.NewSampleHandler(database)

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

		// Sample CRUD
		api.GET("/samples", sampleHandler.List)
		api.GET("/samples/:id", sampleHandler.Get)
		api.POST("/samples", sampleHandler.Create)
		api.PUT("/samples/:id", sampleHandler.Update)
		api.DELETE("/samples/:id", sampleHandler.Delete)
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
