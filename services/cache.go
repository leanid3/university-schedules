package services

import (
	"time"

	"github.com/patrickmn/go-cache"
)

type CacheService struct {
	cache *cache.Cache
}

func NewCacheService(defaultExpiration, cleanupInterval time.Duration) *CacheService {
	return &CacheService{
		cache: cache.New(defaultExpiration, cleanupInterval),
	}
}

func (s *CacheService) Get(key string) (interface{}, bool) {
	return s.cache.Get(key)
}

func (s *CacheService) Set(key string, value interface{}, duration time.Duration) {
	s.cache.Set(key, value, duration)
}

func (s *CacheService) Delete(key string) {
	s.cache.Delete(key)
}

func (s *CacheService) Flush() {
	s.cache.Flush()
}
