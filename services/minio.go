package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	"schedule-api/config"
	"schedule-api/models"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOService struct {
	client *minio.Client
	bucket string
	urlTTL time.Duration
}

func NewMinIOService(cfg *config.Config) (*MinIOService, error) {
	client, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinIOService{
		client: client,
		bucket: cfg.MinIOBucket,
		urlTTL: cfg.PresignedURLTTL,
	}, nil
}

// ListPrefixes возвращает список "папок" на указанном уровне
func (s *MinIOService) ListPrefixes(ctx context.Context, prefix string) ([]string, error) {
	log.Println("MinIOService - ListPrefixes")
	var prefixes []string

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	}

	for object := range s.client.ListObjects(ctx, s.bucket, opts) {
		if object.Err != nil {
			return nil, object.Err
		}
		// Извлекаем имя директории из prefix
		if strings.HasSuffix(object.Key, "/") {
			parts := strings.Split(strings.TrimSuffix(object.Key, "/"), "/")
			if len(parts) > 0 {
				name := parts[len(parts)-1]
				if name != "" && !contains(prefixes, name) {
					prefixes = append(prefixes, name)
				}
			}
		}
	}

	return prefixes, nil
}

// ListFiles возвращает список файлов в указанном префиксе
func (s *MinIOService) ListFiles(ctx context.Context, prefix string) ([]models.ScheduleFile, error) {
	var files []models.ScheduleFile

	opts := minio.ListObjectsOptions{
		Prefix:       prefix,
		Recursive:    false,
		WithVersions: true,
	}

	for object := range s.client.ListObjects(ctx, s.bucket, opts) {
		if object.Err != nil {
			return nil, object.Err
		}

		// Игнорируем директории
		if strings.HasSuffix(object.Key, "/") {
			continue
		}

		// Проверяем что файл xlsx
		if !strings.HasSuffix(strings.ToLower(object.Key), ".xlsx") {
			continue
		}

		fileName := extractFileName(object.Key)
		files = append(files, models.ScheduleFile{
			Name:         fileName,
			Path:         object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
			ETag:         object.ETag,
			Version:      object.VersionID,
		})
	}

	return files, nil
}

// GetPresignedURL генерирует presigned URL для скачивания
func (s *MinIOService) GetPresignedURL(ctx context.Context, objectPath string) (*models.PresignedURLResponse, error) {
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", extractFileName(objectPath)))

	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucket, objectPath, s.urlTTL, reqParams)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned url: %w", err)
	}

	return &models.PresignedURLResponse{
		URL:       presignedURL.String(),
		ExpiresAt: time.Now().Add(s.urlTTL),
		FileName:  extractFileName(objectPath),
	}, nil
}

// GetPresignedUploadURL генерирует presigned URL для загрузки (для будущего админа)
func (s *MinIOService) GetPresignedUploadURL(ctx context.Context, objectPath string) (*models.PresignedURLResponse, error) {
	presignedURL, err := s.client.PresignedPutObject(ctx, s.bucket, objectPath, s.urlTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload url: %w", err)
	}

	return &models.PresignedURLResponse{
		URL:       presignedURL.String(),
		ExpiresAt: time.Now().Add(s.urlTTL),
		FileName:  extractFileName(objectPath),
	}, nil
}

// ObjectExists проверяет существование объекта
func (s *MinIOService) ObjectExists(ctx context.Context, objectPath string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, objectPath, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ObjectExistsInBucket проверяет существование объекта в указанном бакете
func (s *MinIOService) ObjectExistsInBucket(ctx context.Context, bucket, objectPath string) (bool, error) {
	_, err := s.client.StatObject(ctx, bucket, objectPath, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListAllObjectsInBucket возвращает список всех объектов в указанном бакете с префиксом
func (s *MinIOService) ListAllObjectsInBucket(ctx context.Context, bucket, prefix string) ([]string, error) {
	var objects []string

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for object := range s.client.ListObjects(ctx, bucket, opts) {
		if object.Err != nil {
			return nil, object.Err
		}
		objects = append(objects, object.Key)
	}

	return objects, nil
}

// DownloadFile скачивает файл из указанного бакета
func (s *MinIOService) DownloadFile(ctx context.Context, bucket, objectPath string) ([]byte, error) {
	object, err := s.client.GetObject(ctx, bucket, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	// Читаем все данные
	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return data, nil
}

// UploadFile загружает файл в указанный бакет
func (s *MinIOService) UploadFile(ctx context.Context, bucket, objectPath string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, bucket, objectPath, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}
	return nil
}

// Вспомогательные функции
func extractFileName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
