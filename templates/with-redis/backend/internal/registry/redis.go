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
	var cache *infrastructure.Cache

	OnInit(func() error {
		redisURL := os.Getenv("REDIS_URL")

		if redisURL == "" {
			log.Println("[Server] Redis: REDIS_URL not set, skipping initialization")
			return nil
		}

		var err error
		cache, err = infrastructure.NewCache(redisURL)
		if err != nil {
			return fmt.Errorf("failed to initialize Redis: %w", err)
		}

		log.Println("[Server] Redis connected")
		return nil
	})

	OnRoute(func(api *gin.RouterGroup) {
		if cache == nil {
			return
		}

		cacheHandler := handler.NewCacheHandler(cache)

		cacheGroup := api.Group("/cache")
		cacheGroup.POST("", cacheHandler.Set)
		cacheGroup.GET("", cacheHandler.List)
		cacheGroup.GET("/:key", cacheHandler.Get)
		cacheGroup.DELETE("/:key", cacheHandler.Delete)
	})

	OnCleanup(func() {
		if cache != nil {
			cache.Close()
		}
	})
}
