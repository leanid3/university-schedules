package handlers

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"schedule-api/models"
	"schedule-api/services"
	"strings"

	"github.com/gin-gonic/gin"
)

type UploadFileHandler struct {
	minioService    *services.MinIOService
	parserService   *services.ParserService
	cacheService    *services.CacheService
	sourceBucket    string
	targetBucket    string
	filePathPattern string
}

func NewUploadFileHandler(minio *services.MinIOService, cache *services.CacheService, sourceBucket, targetBucket, filePathPattern string) *UploadFileHandler {
	return &UploadFileHandler{
		minioService:    minio,
		parserService:   services.NewParserService(),
		cacheService:    cache,
		sourceBucket:    sourceBucket,
		targetBucket:    targetBucket,
		filePathPattern: filePathPattern,
	}
}

type FileItem struct {
	University   string `json:"university" binding:"required"`
	Course       string `json:"course" binding:"required"`
	ScheduleType string `json:"schedule_type" binding:"required"`
	FileName     string `json:"file_name" binding:"required"`
}

type ProcessFilesRequest struct {
	Files []FileItem `json:"files" binding:"required,min=1"`
}

type ProcessFileResult struct {
	FileName   string `json:"file_name"`
	SourceFile string `json:"source_file"`
	TargetFile string `json:"target_file"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

func (h *UploadFileHandler) ProcessFile(c *gin.Context) {
	log.Println("UploadFileHandler - ProcessFile")

	var req ProcessFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request body",
			Message: err.Error(),
		})
		return
	}

	results := make([]ProcessFileResult, 0, len(req.Files))
	successCount := 0
	failureCount := 0

	// Обрабатываем каждый файл
	for _, fileItem := range req.Files {
		result := h.processOneFile(c, fileItem)
		results = append(results, result)

		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	// Формируем итоговый ответ
	statusCode := http.StatusOK
	if failureCount > 0 && successCount == 0 {
		statusCode = http.StatusInternalServerError
	} else if failureCount > 0 {
		statusCode = http.StatusMultiStatus
	}

	c.JSON(statusCode, gin.H{
		"message":       fmt.Sprintf("processed %d files: %d succeeded, %d failed", len(req.Files), successCount, failureCount),
		"total":         len(req.Files),
		"succeeded":     successCount,
		"failed":        failureCount,
		"results":       results,
		"source_bucket": h.sourceBucket,
		"target_bucket": h.targetBucket,
	})
}

func (h *UploadFileHandler) processOneFile(c *gin.Context, fileItem FileItem) ProcessFileResult {
	result := ProcessFileResult{
		FileName: fileItem.FileName,
		Success:  false,
	}

	// Формируем путь к XLSX файлу в бакете file-upload
	xlsxPath := fmt.Sprintf(h.filePathPattern, fileItem.University, fileItem.Course, fileItem.ScheduleType, fileItem.FileName)
	result.SourceFile = xlsxPath

	// Проверяем существование файла перед скачиванием
	log.Printf("Проверка существования файла в %s: %s", h.sourceBucket, xlsxPath)
	exists, err := h.minioService.ObjectExistsInBucket(c.Request.Context(), h.sourceBucket, xlsxPath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to check file existence: %v", err)
		log.Printf("Ошибка проверки существования %s: %v", xlsxPath, err)
		return result
	}
	if !exists {
		result.Error = fmt.Sprintf("file not found in bucket: %s", xlsxPath)
		log.Printf("Файл не найден в %s: %s", h.sourceBucket, xlsxPath)
		return result
	}

	// Скачиваем XLSX файл из source bucket
	log.Printf("Скачивание файла из %s: %s", h.sourceBucket, xlsxPath)
	xlsxData, err := h.minioService.DownloadFile(c.Request.Context(), h.sourceBucket, xlsxPath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to download file: %v", err)
		log.Printf("Ошибка скачивания %s: %v", xlsxPath, err)
		return result
	}

	// Валидируем XLSX файл
	reader := bytes.NewReader(xlsxData)
	valid, err := h.parserService.ValidateScheduleFile(reader, fileItem.ScheduleType)
	if err != nil || !valid {
		result.Error = fmt.Sprintf("invalid schedule file: %v", err)
		log.Printf("Ошибка валидации %s: %v", xlsxPath, err)
		return result
	}

	// Возвращаемся в начало после валидации
	reader.Seek(0, 0)

	// Парсим XLSX в JSON
	log.Printf("Парсинг файла: %s", fileItem.FileName)
	jsonData, err := h.parserService.ParseXLSXToJSON(reader, fileItem.ScheduleType)
	if err != nil {
		result.Error = fmt.Sprintf("failed to parse file: %v", err)
		log.Printf("Ошибка парсинга %s: %v", xlsxPath, err)
		return result
	}

	// Формируем путь для JSON файла в целевом бакете
	jsonFileName := strings.TrimSuffix(fileItem.FileName, ".xlsx") + ".json"
	jsonPath := fmt.Sprintf(h.filePathPattern, fileItem.University, fileItem.Course, fileItem.ScheduleType, jsonFileName)
	result.TargetFile = jsonPath

	// Загружаем JSON в target bucket
	log.Printf("Загрузка JSON в %s: %s", h.targetBucket, jsonPath)
	err = h.minioService.UploadFile(c.Request.Context(), h.targetBucket, jsonPath, bytes.NewReader(jsonData), int64(len(jsonData)), "application/json")
	if err != nil {
		result.Error = fmt.Sprintf("failed to upload json: %v", err)
		log.Printf("Ошибка загрузки %s: %v", jsonPath, err)
		return result
	}

	// Инвалидируем кэш для этого расписания
	cacheKey := fmt.Sprintf("files:%s:%s:%s", fileItem.University, fileItem.Course, fileItem.ScheduleType)
	h.cacheService.Delete(cacheKey)

	log.Printf("Файл успешно обработан: %s -> %s", xlsxPath, jsonPath)
	result.Success = true
	return result
}
