package services

import (
	"errors"
	"sync"
	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/cache"
	"github.com/google/uuid"
)

type PropertyService struct {
	cache *cache.Cache
	mu    sync.RWMutex
}

func NewPropertyService() *PropertyService {
	return &PropertyService{
		cache: cache.NewCache(),
	}
}

func (s *PropertyService) CreateProperty(property *models.Property) error {
	if property.Name == "" {
		return errors.New("property name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate unique ID
	property.ID = uuid.New().String()

	// Store in cache
	return s.cache.Set(property.ID, *property)
}

func (s *PropertyService) GetAllProperties() ([]models.Property, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.cache.GetAll()
	properties := make([]models.Property, 0, len(items))
	for _, item := range items {
		if prop, ok := item.(models.Property); ok {
			properties = append(properties, prop)
		}
	}

	return properties, nil
}
