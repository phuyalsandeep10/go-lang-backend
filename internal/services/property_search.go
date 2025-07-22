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
	"homeinsight-properties/pkg/config"
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
	config              *config.Config
}

func NewPropertySearchService(
	repo repositories.PropertyRepository,
	cache repositories.PropertyCache,
	addrTrans transformers.AddressTransformer,
	propTrans transformers.PropertyTransformer,
	validator validators.PropertyValidator,
	corelogicClient *corelogic.Client,
	cfg *config.Config,
) *PropertySearchService {
	return &PropertySearchService{
		repo:                repo,
		cache:               cache,
		addrTrans:           addrTrans,
		propTrans:           propTrans,
		validator:           validator,
		externalDataService: NewExternalDataService(corelogicClient, propTrans),
		config:              cfg,
	}
}

// cacheProperty stores a property and its search key in the cache.
func (s *PropertySearchService) cacheProperty(ctx context.Context, property *models.Property, cacheKey string) error {
	propertyKey := cache.PropertyKey(property.PropertyID)
	cacheTTL := time.Duration(s.config.Redis.CacheTTLDays) * 24 * time.Hour
	if err := s.cache.SetProperty(ctx, propertyKey, property, cacheTTL); err != nil {
		return fmt.Errorf("failed to cache property: %v", err)
	}
	if err := s.cache.SetSearchKey(ctx, cacheKey, property.PropertyID, cacheTTL); err != nil {
		return fmt.Errorf("failed to cache search key: %v", err)
	}
	if err := s.cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, cacheKey); err != nil {
		return fmt.Errorf("failed to add cache key to property set: %v", err)
	}
	return nil
}

// isPropertyStale checks if a property's UpdatedAt timestamp is older than the staleness threshold.
func (s *PropertySearchService) isPropertyStale(updatedAt time.Time) bool {
	threshold := time.Now().AddDate(0, 0, -s.config.Database.StaleThresholdDays)
	return !updatedAt.After(threshold)
}

func (s *PropertySearchService) SearchSpecificProperty(ctx context.Context, req *models.SearchRequest) (*models.Property, error) {
	ginCtx, _ := ctx.(*gin.Context)
	if ginCtx == nil {
		ginCtx = &gin.Context{}
	}

	// Validate search request
	if err := s.validator.ValidateSearch(req); err != nil {
		logger.GlobalLogger.Errorf("Invalid search request: query=%s, error=%v", req.Search, err)
		return nil, fmt.Errorf("invalid search request: %v", err)
	}

	// Parse address
	street, city, state, zip := s.addrTrans.ParseAddress(req.Search)
	if street == "" || city == "" {
		logger.GlobalLogger.Errorf("Missing address fields: query=%s", req.Search)
		return nil, fmt.Errorf("street address and city are required")
	}

	// Generate cache key and set initial metadata
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

	// Cache miss
	metrics.CacheMissesTotal.Inc()
	ginCtx.Set("cache_hit", false)

	// Query database
	property, err := s.repo.FindByAddress(ctx, street, city, state, zip)
	if err != nil {
		logger.GlobalLogger.Errorf("Database query failed: query=%s, error=%v", req.Search, err)
		return nil, fmt.Errorf("database query failed: %v", err)
	}

	// Handle existing property
	if property != nil {
		ginCtx.Set("property_id", property.PropertyID)
		if !s.isPropertyStale(property.UpdatedAt) {
			ginCtx.Set("data_source", "DATABASE")
			if err := s.cacheProperty(ctx, property, cacheKey); err != nil {
				logger.GlobalLogger.Errorf("Cache update failed: propertyID=%s, error=%v", property.PropertyID, err)
			}
			return property, nil
		}

		// Property is stale, fetch from external source
		newProperty, err := s.externalDataService.FetchFromExternalSource(ctx, street, city, state, zip, req)
		if err != nil {
			logger.GlobalLogger.Errorf("External data fetch failed: query=%s, error=%v", req.Search, err)
			return nil, fmt.Errorf("failed to fetch from external source: %v", err)
		}

		// Update existing property
		newProperty.ID = property.ID
		newProperty.PropertyID = property.PropertyID
		newProperty.UpdatedAt = time.Now()

		if err := s.repo.Update(ctx, newProperty); err != nil {
			logger.GlobalLogger.Errorf("Failed to update property: propertyID=%s, error=%v", newProperty.PropertyID, err)
			return nil, fmt.Errorf("failed to update property: %v", err)
		}

		// Cache updated property
		if err := s.cacheProperty(ctx, newProperty, cacheKey); err != nil {
			logger.GlobalLogger.Errorf("Cache update failed: propertyID=%s, error=%v", newProperty.PropertyID, err)
		}
		ginCtx.Set("data_source", "CORELOGIC_API")
		return newProperty, nil
	}

	// No property found, fetch from external source
	newProperty, err := s.externalDataService.FetchFromExternalSource(ctx, street, city, state, zip, req)
	if err != nil {
		logger.GlobalLogger.Errorf("External data fetch failed: query=%s, error=%v", req.Search, err)
		return nil, fmt.Errorf("failed to fetch from external source: %v", err)
	}

	// Check for race condition
	existingProperty, err := s.repo.FindByID(ctx, newProperty.PropertyID)
	if err != nil {
		logger.GlobalLogger.Errorf("Failed to check existing property: propertyID=%s, error=%v", newProperty.PropertyID, err)
		return nil, fmt.Errorf("failed to check existing property: %v", err)
	}
	if existingProperty != nil {
		newProperty.ID = existingProperty.ID
		newProperty.PropertyID = existingProperty.PropertyID
		newProperty.UpdatedAt = time.Now()

		if err := s.repo.Update(ctx, newProperty); err != nil {
			logger.GlobalLogger.Errorf("Failed to update property: propertyID=%s, error=%v", newProperty.PropertyID, err)
			return nil, fmt.Errorf("failed to update property: %v", err)
		}

		if err := s.cacheProperty(ctx, newProperty, cacheKey); err != nil {
			logger.GlobalLogger.Errorf("Cache update failed: propertyID=%s, error=%v", newProperty.PropertyID, err)
		}
		ginCtx.Set("data_source", "CORELOGIC_API")
		ginCtx.Set("property_id", newProperty.PropertyID)
		return newProperty, nil
	}

	// Create new property
	newProperty.ID = primitive.NewObjectID()
	newProperty.UpdatedAt = time.Now()

	if err := s.repo.Create(ctx, newProperty); err != nil {
		logger.GlobalLogger.Errorf("Failed to create property: propertyID=%s, error=%v", newProperty.PropertyID, err)
		return nil, fmt.Errorf("failed to create property: %v", err)
	}

	// Cache new property
	if err := s.cacheProperty(ctx, newProperty, cacheKey); err != nil {
		logger.GlobalLogger.Errorf("Cache update failed: propertyID=%s, error=%v", newProperty.PropertyID, err)
	}
	ginCtx.Set("data_source", "CORELOGIC_API")
	ginCtx.Set("property_id", newProperty.PropertyID)
	return newProperty, nil
}
