package main

import (
	"log"
	"time"

	"schedule-api/config"
	"schedule-api/handlers"
	"schedule-api/middleware"
	"schedule-api/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("Start service")
	// Загружаем .env файл (игнорируем ошибку для продакшн)
	_ = godotenv.Load()

	// Загружаем конфигурацию
	cfg := config.Load()

	log.Println("init services")
	// Инициализируем сервисы
	minioService, err := services.NewMinIOService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO service: %v", err)
	}

	cacheService := services.NewCacheService(cfg.CacheTTL, 2*cfg.CacheTTL)

	log.Println("init handlers")
	// Инициализируем handlers
	universityHandler := handlers.NewUniversityHandler(minioService, cacheService)
	courseHandler := handlers.NewCourseHandler(minioService, cacheService)
	scheduleHandler := handlers.NewScheduleHandler(minioService, cacheService)
	uploadFileHandler := handlers.NewUploadFileHandler(minioService, cacheService, cfg.SourceBucket, cfg.TargetBucket, cfg.FilePathPattern)

	// Настраиваем Gin
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	log.Println("init router")
	router := gin.New()
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())
	router.Use(gin.Recovery())

	// API routes
	api := router.Group("/api/v1")
	{
		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
				"time":   time.Now(),
			})
		})

		// Universities
		api.GET("/universities", universityHandler.GetUniversities)

		// Courses
		api.GET("/universities/:university/courses", courseHandler.GetCourses)

		// Schedule types
		api.GET("/universities/:university/courses/:course/types", scheduleHandler.GetScheduleTypes)

		// Schedule files
		api.GET("/universities/:university/courses/:course/types/:type/files", scheduleHandler.GetScheduleFiles)

		// Download presigned URL
		api.GET("/universities/:university/courses/:course/types/:type/files/:filename/download", scheduleHandler.GetPresignedDownloadURL)

		// Cache management
		api.POST("/cache/invalidate", scheduleHandler.InvalidateCache)

		// File processing
		api.POST("/files_uploaded", uploadFileHandler.ProcessFile)
	}

	// Запускаем сервер
	log.Printf("Starting server on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
