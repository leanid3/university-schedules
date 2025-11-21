package handlers

import (
	"fmt"
	"net/http"

	"schedule-api/models"
	"schedule-api/services"

	"github.com/gin-gonic/gin"
)

type ScheduleHandler struct {
	minioService *services.MinIOService
	cacheService *services.CacheService
}

func NewScheduleHandler(minio *services.MinIOService, cache *services.CacheService) *ScheduleHandler {
	return &ScheduleHandler{
		minioService: minio,
		cacheService: cache,
	}
}

// GetScheduleTypes возвращает список типов расписаний
func (h *ScheduleHandler) GetScheduleTypes(c *gin.Context) {
	university := c.Param("university")
	course := c.Param("course")

	if university == "" || course == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "university and course parameters are required",
		})
		return
	}

	cacheKey := fmt.Sprintf("types:%s:%s", university, course)

	// Проверяем кэш
	if cached, found := h.cacheService.Get(cacheKey); found {
		c.JSON(http.StatusOK, gin.H{
			"data":   cached,
			"cached": true,
		})
		return
	}

	// Получаем из MinIO
	prefix := fmt.Sprintf("%s/%s/", university, course)
	prefixes, err := h.minioService.ListPrefixes(c.Request.Context(), prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to list schedule types",
			Message: err.Error(),
		})
		return
	}

	types := make([]models.ScheduleType, 0, len(prefixes))
	for _, prefix := range prefixes {
		types = append(types, models.ScheduleType{
			Name:       prefix,
			University: university,
			Course:     course,
		})
	}

	// Сохраняем в кэш
	h.cacheService.Set(cacheKey, types, 0)

	c.JSON(http.StatusOK, gin.H{
		"data":   types,
		"cached": false,
	})
}

// GetScheduleFiles возвращает список файлов расписаний
func (h *ScheduleHandler) GetScheduleFiles(c *gin.Context) {
	university := c.Param("university")
	course := c.Param("course")
	scheduleType := c.Param("type")

	if university == "" || course == "" || scheduleType == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "all path parameters are required",
		})
		return
	}

	cacheKey := fmt.Sprintf("files:%s:%s:%s", university, course, scheduleType)
	// Проверяем кэш
	if cached, found := h.cacheService.Get(cacheKey); found {
		c.JSON(http.StatusOK, gin.H{
			"data":   cached,
			"cached": true,
		})
		return
	}

	// Получаем из MinIO
	prefix := fmt.Sprintf("%s/%s/%s/", university, course, scheduleType)

	files, err := h.minioService.ListFiles(c.Request.Context(), prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to list schedule files",
			Message: err.Error(),
		})
		return
	}

	// Сохраняем в кэш
	h.cacheService.Set(cacheKey, files, 0)

	c.JSON(http.StatusOK, gin.H{
		"data":   files,
		"cached": false,
	})
}

// GetPresignedDownloadURL возвращает presigned URL для скачивания
func (h *ScheduleHandler) GetPresignedDownloadURL(c *gin.Context) {
	university := c.Param("university")
	course := c.Param("course")
	scheduleType := c.Param("type")
	fileName := c.Param("filename")

	if university == "" || course == "" || scheduleType == "" || fileName == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "all parameters are required",
		})
		return
	}

	objectPath := fmt.Sprintf("%s/%s/%s/%s", university, course, scheduleType, fileName)

	// Проверяем существование файла
	exists, err := h.minioService.ObjectExists(c.Request.Context(), objectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to check file existence",
			Message: err.Error(),
		})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "file not found",
		})
		return
	}

	// Генерируем presigned URL
	urlResponse, err := h.minioService.GetPresignedURL(c.Request.Context(), objectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to generate download url",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, urlResponse)
}

// InvalidateCache удаляет кэш (для будущих webhook'ов)
func (h *ScheduleHandler) InvalidateCache(c *gin.Context) {
	h.cacheService.Flush()
	c.JSON(http.StatusOK, gin.H{
		"message": "cache invalidated successfully",
	})
}
