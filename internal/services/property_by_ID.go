package services

import (
	"context"
	"fmt"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"github.com/gin-gonic/gin"
)

func (s *PropertyService) GetPropertyByID(ctx context.Context, id string) (*models.Property, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		ginCtx = &gin.Context{}
	}

	propertyKey := cache.PropertyKey(id)
	ginCtx.Set("data_source", "REDIS")
	ginCtx.Set("property_id", id)

	// Check cache
	if property, err := s.cache.GetProperty(ctx, propertyKey); err == nil && property != nil {
		metrics.CacheHitsTotal.Inc()
		ginCtx.Set("cache_hit", true)
		return property, nil
	}

	metrics.CacheMissesTotal.Inc()
	ginCtx.Set("cache_hit", false)

	// Query database
	property, err := s.repo.FindByID(ctx, id)
	if err != nil {
		logger.GlobalLogger.Errorf("DB query failed: id=%s, error=%v", id, err)
		return nil, fmt.Errorf("failed to fetch property: %v", err)
	}
	if property == nil {
		logger.GlobalLogger.Errorf("Property not found: id=%s", id)
		return nil, fmt.Errorf("property with id %s not found", id)
	}

	ginCtx.Set("data_source", "DATABASE")

	// Cache the property
	_ = s.cache.SetProperty(ctx, propertyKey, property, Month)
	_ = s.cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, propertyKey)

	return property, nil
}
