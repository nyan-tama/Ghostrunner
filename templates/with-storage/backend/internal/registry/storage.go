package registry

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"{{PROJECT_NAME}}/backend/internal/handler"
	"{{PROJECT_NAME}}/backend/internal/infrastructure"
)

func init() {
	var storage *infrastructure.Storage

	OnInit(func() error {
		accessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
		accessKeySecret := os.Getenv("R2_ACCESS_KEY_SECRET")
		bucketName := os.Getenv("R2_BUCKET_NAME")

		if accessKeyID == "" || accessKeySecret == "" || bucketName == "" {
			log.Println("[Server] Storage: environment variables not set, skipping initialization")
			return nil
		}

		// STORAGE_ENDPOINT が設定されている場合はそれを使用（MinIO 等の S3 互換ストレージ）
		// 未設定の場合は R2_ACCOUNT_ID から Cloudflare R2 エンドポイントを構築
		endpoint := os.Getenv("STORAGE_ENDPOINT")
		usePathStyle := false
		if endpoint == "" {
			accountID := os.Getenv("R2_ACCOUNT_ID")
			if accountID == "" {
				log.Println("[Server] Storage: STORAGE_ENDPOINT or R2_ACCOUNT_ID required, skipping initialization")
				return nil
			}
			endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
		} else {
			usePathStyle = true
		}

		var err error
		storage, err = infrastructure.NewStorage(endpoint, accessKeyID, accessKeySecret, bucketName, usePathStyle)
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		log.Println("[Server] Storage initialized")
		return nil
	})

	OnRoute(func(api *gin.RouterGroup) {
		if storage == nil {
			return
		}

		storageHandler := handler.NewStorageHandler(storage)

		storageGroup := api.Group("/storage")
		storageGroup.POST("/upload", storageHandler.Upload)
		storageGroup.GET("/files", storageHandler.List)
		storageGroup.GET("/files/:key", storageHandler.Download)
		storageGroup.DELETE("/files/:key", storageHandler.Delete)
	})
}
