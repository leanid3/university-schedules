package handlers

import (
	"log"
	"net/http"

	"schedule-api/models"
	"schedule-api/services"

	"github.com/gin-gonic/gin"
)

type UniversityHandler struct {
	minioService *services.MinIOService
	cacheService *services.CacheService
}

func NewUniversityHandler(minio *services.MinIOService, cache *services.CacheService) *UniversityHandler {
	return &UniversityHandler{
		minioService: minio,
		cacheService: cache,
	}
}

// GetUniversities возвращает список университетов
func (h *UniversityHandler) GetUniversities(c *gin.Context) {
	log.Println("UniversityHandler - GetUniversities")
	cacheKey := "universities"

	// Проверяем кэш
	if cached, found := h.cacheService.Get(cacheKey); found {
		c.JSON(http.StatusOK, gin.H{
			"data":   cached,
			"cached": true,
		})
		return
	}

	// Получаем из MinIO
	prefixes, err := h.minioService.ListPrefixes(c.Request.Context(), "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to list universities",
			Message: err.Error(),
		})
		return
	}

	universities := make([]models.University, 0, len(prefixes))
	for _, prefix := range prefixes {
		universities = append(universities, models.University{Name: prefix})
	}

	// Сохраняем в кэш
	h.cacheService.Set(cacheKey, universities, 0)

	c.JSON(http.StatusOK, gin.H{
		"data":   universities,
		"cached": false,
	})
}
