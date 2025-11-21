package handlers

import (
	"fmt"
	"log"
	"net/http"

	"schedule-api/models"
	"schedule-api/services"

	"github.com/gin-gonic/gin"
)

type CourseHandler struct {
	minioService *services.MinIOService
	cacheService *services.CacheService
}

func NewCourseHandler(minio *services.MinIOService, cache *services.CacheService) *CourseHandler {
	return &CourseHandler{
		minioService: minio,
		cacheService: cache,
	}
}

// GetCourses возвращает список курсов для университета
func (h *CourseHandler) GetCourses(c *gin.Context) {
	log.Println("CourseHandler - GetCourses")
	university := c.Param("university")
	if university == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "university parameter is required",
		})
		return
	}

	cacheKey := fmt.Sprintf("courses:%s", university)

	// Проверяем кэш
	if cached, found := h.cacheService.Get(cacheKey); found {
		c.JSON(http.StatusOK, gin.H{
			"data":   cached,
			"cached": true,
		})
		return
	}

	// Получаем из MinIO
	prefix := fmt.Sprintf("%s/", university)
	prefixes, err := h.minioService.ListPrefixes(c.Request.Context(), prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to list courses",
			Message: err.Error(),
		})
		return
	}

	courses := make([]models.Course, 0, len(prefixes))
	for _, prefix := range prefixes {
		courses = append(courses, models.Course{
			Name:       prefix,
			University: university,
		})
	}

	// Сохраняем в кэш
	h.cacheService.Set(cacheKey, courses, 0)

	c.JSON(http.StatusOK, gin.H{
		"data":   courses,
		"cached": false,
	})
}
