package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerPort      string
	MinIOEndpoint   string
	MinIOAccessKey  string
	MinIOSecretKey  string
	MinIOBucket     string
	MinIOUseSSL     bool
	CacheTTL        time.Duration
	PresignedURLTTL time.Duration
	Environment     string
	SourceBucket    string // Бакет для исходных XLSX файлов
	TargetBucket    string // Бакет для обработанных JSON файлов
	FilePathPattern string // Паттерн пути к файлам
}

func Load() *Config {
	cacheMinutes, _ := strconv.Atoi(getEnv("CACHE_TTL_MINUTES", "10"))
	presignedMinutes, _ := strconv.Atoi(getEnv("PRESIGNED_URL_TTL_MINUTES", "15"))
	useSSL, _ := strconv.ParseBool(getEnv("MINIO_USE_SSL", "false"))

	return &Config{
		ServerPort:      getEnv("SERVER_PORT", "8080"),
		MinIOEndpoint:   getEnv("MINIO_ENDPOINT", "minio:9000"),
		MinIOAccessKey:  getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:  getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:     getEnv("MINIO_BUCKET", "university-schedules"),
		MinIOUseSSL:     useSSL,
		CacheTTL:        time.Duration(cacheMinutes) * time.Minute,
		PresignedURLTTL: time.Duration(presignedMinutes) * time.Minute,
		Environment:     getEnv("ENVIRONMENT", "development"),
		SourceBucket:    getEnv("SOURCE_BUCKET", "file-upload"),
		TargetBucket:    getEnv("TARGET_BUCKET", "university-schedules"),
		FilePathPattern: getEnv("FILE_PATH_PATTERN", "universities/%s/courses/%s/types/%s/files/%s"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
