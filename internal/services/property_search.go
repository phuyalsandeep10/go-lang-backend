package services

import (
	"context"
	"fmt"
	"time"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/repositories"
	"homeinsight-properties/internal/transformers"
	"homeinsight-properties/internal/validators"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/corelogic"
	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PropertySearchService struct {
	repo                repositories.PropertyRepository
	cache               repositories.PropertyCache
	addrTrans           transformers.AddressTransformer
	propTrans           transformers.PropertyTransformer
	validator           validators.PropertyValidator
	externalDataService *ExternalDataService
}

func NewPropertySearchService(
	repo repositories.PropertyRepository,
	cache repositories.PropertyCache,
	addrTrans transformers.AddressTransformer,
	propTrans transformers.PropertyTransformer,
	validator validators.PropertyValidator,
	corelogicClient *corelogic.Client,
) *PropertySearchService {
	return &PropertySearchService{
		repo:                repo,
		cache:               cache,
		addrTrans:           addrTrans,
		propTrans:           propTrans,
		validator:           validator,
		externalDataService: NewExternalDataService(corelogicClient, propTrans),
	}
}

func (s *PropertySearchService) SearchSpecificProperty(ctx context.Context, req *models.SearchRequest) (*models.Property, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		ginCtx = &gin.Context{}
	}

	if err := s.validator.ValidateSearch(req); err != nil {
		logger.GlobalLogger.Errorf("Invalid search: query=%s, error=%v", req.Search, err)
		return nil, err
	}

	street, city, state, zip := s.addrTrans.ParseAddress(req.Search)
	if street == "" || city == "" {
		logger.GlobalLogger.Errorf("Missing address fields: query=%s", req.Search)
		return nil, fmt.Errorf("street address and city are required")
	}

	cacheKey := cache.PropertySpecificSearchKey(street, city)
	ginCtx.Set("data_source", "REDIS")
	ginCtx.Set("query", req.Search)

	// Check cache
	if propertyID, err := s.cache.GetSearchKey(ctx, cacheKey); err == nil && propertyID != "" {
		if property, err := s.cache.GetProperty(ctx, cache.PropertyKey(propertyID)); err == nil && property != nil {
			metrics.CacheHitsTotal.Inc()
			ginCtx.Set("cache_hit", true)
			ginCtx.Set("property_id", propertyID)
			return property, nil
		}
	}

	metrics.CacheMissesTotal.Inc()
	ginCtx.Set("cache_hit", false)

	// Query database
	property, err := s.repo.FindByAddress(ctx, street, city, state, zip)
	if err != nil {
		logger.GlobalLogger.Errorf("DB query failed: query=%s, error=%v", req.Search, err)
		return nil, err
	}

	// Check if property exists and if updatedAt is older than 2 months
	if property != nil {
		twoMonthsAgo := time.Now().AddDate(0, -2, 0)
		if property.UpdatedAt.After(twoMonthsAgo) {
			ginCtx.Set("data_source", "DATABASE")
			ginCtx.Set("property_id", property.PropertyID)
			propertyKey := cache.PropertyKey(property.PropertyID)
			_ = s.cache.SetProperty(ctx, propertyKey, property, Month)
			_ = s.cache.SetSearchKey(ctx, cacheKey, property.PropertyID, Month)
			_ = s.cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, cacheKey)
			return property, nil
		}
	}

	// Fallback to external data source
	newProperty, err := s.externalDataService.FetchFromExternalSource(ctx, street, city, state, zip, req)
	if err != nil {
		return nil, err
	}

	ginCtx.Set("property_id", newProperty.PropertyID)

	// If property exists, update it; otherwise, create a new one
	if property != nil {
		// Preserve existing ID and propertyId
		newProperty.ID = property.ID
		newProperty.PropertyID = property.PropertyID
		newProperty.UpdatedAt = time.Now()

		if err := s.repo.Update(ctx, newProperty); err != nil {
			logger.GlobalLogger.Errorf("Failed to update property: propertyID=%s, error=%v", newProperty.PropertyID, err)
			return nil, fmt.Errorf("failed to update property: %v", err)
		}
	} else {
		// Generate a new ID for a new property
		newProperty.ID = primitive.NewObjectID()
		newProperty.UpdatedAt = time.Now()

		if err := s.repo.Create(ctx, newProperty); err != nil {
			logger.GlobalLogger.Errorf("Failed to create property: propertyID=%s, error=%v", newProperty.PropertyID, err)
			return nil, fmt.Errorf("failed to create property: %v", err)
		}
	}

	// Cache the property
	propertyKey := cache.PropertyKey(newProperty.PropertyID)
	_ = s.cache.SetProperty(ctx, propertyKey, newProperty, Month)
	_ = s.cache.SetSearchKey(ctx, cacheKey, newProperty.PropertyID, Month)
	_ = s.cache.AddCacheKeyToPropertySet(ctx, newProperty.PropertyID, cacheKey)

	return newProperty, nil
}
