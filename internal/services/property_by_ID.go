package services

import (
	"context"
	"fmt"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/utils"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *PropertyService) GetPropertyByID(ctx context.Context, id string) (*models.Property, error) {
	propertyKey := cache.PropertyKey(id)
	if property, err := s.cache.GetProperty(ctx, propertyKey); err == nil && property != nil {
		metrics.CacheHitsTotal.Inc()
		return property, nil
	}
	metrics.CacheMissesTotal.Inc()

	property, err := s.repo.FindByID(ctx, id)
	if err != nil {
		logger.GlobalLogger.Errorf("DB query failed: id=%s, error=%v", id, err)
		return nil, err
	}
	if property != nil {
		_ = s.cache.SetProperty(ctx, propertyKey, property, Month)
		return property, nil
	}

	// Fallback to external data source
	// Option 1: Use CoreLogic API
	// property, err = s.corelogic.RequestCoreLogic(ctx, "", "", "", "")
	// if err != nil {
	// 	logger.GlobalLogger.Errorf("CoreLogic failed: id=%s, error=%v", id, err)
	// 	return nil, fmt.Errorf("failed to fetch from CoreLogic: %v", err)
	// }

	// Option 2: Use Mock Data
	property, err = utils.ReadMockData(ctx, "property-detail.json", s.trans)
	if err != nil {
		logger.GlobalLogger.Errorf("Mock data read failed: id=%s, error=%v", id, err)
		return nil, fmt.Errorf("failed to read mock data: %v", err)
	}

	// Override ID to match the requested ID
	property.ID = primitive.NewObjectID()
	property.PropertyID = id

	if err := s.repo.Create(ctx, property); err != nil {
		logger.GlobalLogger.Errorf("Failed to create property: id=%s, error=%v", id, err)
		return nil, err
	}

	_ = s.cache.SetProperty(ctx, propertyKey, property, Month)
	_ = s.cache.InvalidatePropertyCacheKeys(ctx, property.PropertyID)
	return property, nil
}
