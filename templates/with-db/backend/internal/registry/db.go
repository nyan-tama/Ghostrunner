package registry

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"{{PROJECT_NAME}}/backend/internal/domain/model"
	"{{PROJECT_NAME}}/backend/internal/handler"
	"{{PROJECT_NAME}}/backend/internal/infrastructure"
)

func init() {
	var database *infrastructure.Database

	OnInit(func() error {
		databaseURL := os.Getenv("DATABASE_URL")
		if databaseURL == "" {
			return fmt.Errorf("DATABASE_URL is required")
		}

		var err error
		database, err = infrastructure.NewDatabase(databaseURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}

		if err := database.AutoMigrate(&model.Sample{}); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
		log.Println("[Server] Database migration completed")
		return nil
	})

	OnRoute(func(api *gin.RouterGroup) {
		sampleHandler := handler.NewSampleHandler(database)

		api.GET("/samples", sampleHandler.List)
		api.GET("/samples/:id", sampleHandler.Get)
		api.POST("/samples", sampleHandler.Create)
		api.PUT("/samples/:id", sampleHandler.Update)
		api.DELETE("/samples/:id", sampleHandler.Delete)
	})

	OnCleanup(func() {
		if database != nil {
			database.Close()
		}
	})
}
