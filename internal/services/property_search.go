
package services

import (
	"context"
	"fmt"
	"time"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/repositories"
	"homeinsight-properties/internal/transformers"
	"homeinsight-properties/internal/utils"
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
		externalDataService: NewExternalDataService(corelogicClient, propTrans, cfg),
		config:              cfg,
	}
}

// cacheProperty stores a property and its search key in the cache.
func (s *PropertySearchService) cacheProperty(ctx context.Context, property *models.Property, cacheKey string) error {
	propertyKey := cache.PropertyKey(property.PropertyID)
	cacheTTL := time.Duration(s.config.Redis.CacheTTLDays) * 24 * time.Hour
	if err := s.cache.SetProperty(ctx, propertyKey, property, cacheTTL); err != nil {
		logger.GlobalLogger.Warnf("Failed to cache property: propertyID=%s, error=%v", property.PropertyID, err)
		return nil
	}
	if err := s.cache.SetSearchKey(ctx, cacheKey, property.PropertyID, cacheTTL); err != nil {
		logger.GlobalLogger.Warnf("Failed to cache search key: propertyID=%s, error=%v", property.PropertyID, err)
		return nil
	}
	if err := s.cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, cacheKey); err != nil {
		logger.GlobalLogger.Warnf("Failed to add cache key to property set: propertyID=%s, error=%v", property.PropertyID, err)
		return nil
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
		return nil, utils.LogAndMapError(ctx, err, "validate search request", "query", req.Search)
	}

	// Parse address
	street, city, state, zip := s.addrTrans.ParseAddress(req.Search)
	if street == "" || city == "" {
		err := fmt.Errorf("street address and city are required")
		return nil, utils.LogAndMapError(ctx, err, "parse address", "query", req.Search)
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
		logger.GlobalLogger.Warnf("Cache miss for property: cacheKey=%s, error=%v", cacheKey, err)
	}

	// Cache miss
	metrics.CacheMissesTotal.Inc()
	ginCtx.Set("cache_hit", false)

	// Query database
	var property *models.Property
	var err error
	for attempt := 1; attempt <= s.config.ErrorHandling.RetryAttempts; attempt++ {
		property, err = s.repo.FindByAddress(ctx, street, city, state, zip)
		if err == nil || !utils.IsRetryableError(err) {
			break
		}
		logger.GlobalLogger.Warnf("Database query attempt %d/%d failed: query=%s, error=%v", attempt, s.config.ErrorHandling.RetryAttempts, req.Search, err)
		time.Sleep(time.Duration(s.config.ErrorHandling.RetryDelayMS) * time.Millisecond)
	}
	if err != nil {
		return nil, utils.LogAndMapError(ctx, utils.WrapError(err, "database query failed: query=%s", req.Search),
			"database query",
			"query", req.Search,
			"street", street,
			"city", city,
			"state", state,
			"zip", zip)
	}

	// Handle existing property
	if property != nil {
		ginCtx.Set("property_id", property.PropertyID)
		if !s.isPropertyStale(property.UpdatedAt) {
			ginCtx.Set("data_source", "DATABASE")
			if err := s.cacheProperty(ctx, property, cacheKey); err != nil {
				logger.GlobalLogger.Warnf("Cache update failed: propertyID=%s, error=%v", property.PropertyID, err)
			}
			return property, nil
		}

		// Property is stale, fetch from external source
		newProperty, err := s.externalDataService.FetchFromExternalSource(ctx, street, city, state, zip, req)
		if err != nil {
			return nil, utils.WrapError(err, "fetch external data failed: query=%s", req.Search)
		}

		// Update existing property
		newProperty.ID = property.ID
		newProperty.PropertyID = property.PropertyID
		newProperty.UpdatedAt = time.Now()

		if err := s.repo.Update(ctx, newProperty); err != nil {
			return nil, utils.LogAndMapError(ctx, utils.WrapError(err, "update property failed: propertyID=%s", newProperty.PropertyID),
				"update property",
				"propertyID", newProperty.PropertyID)
		}

		// Cache updated property
		if err := s.cacheProperty(ctx, newProperty, cacheKey); err != nil {
			logger.GlobalLogger.Warnf("Cache update failed: propertyID=%s, error=%v", newProperty.PropertyID, err)
		}
		ginCtx.Set("data_source", "CORELOGIC_API")
		return newProperty, nil
	}

	// No property found, fetch from external source
	newProperty, err := s.externalDataService.FetchFromExternalSource(ctx, street, city, state, zip, req)
	if err != nil {
		return nil, utils.WrapError(err, "fetch external data failed: query=%s", req.Search)
	}

	// Check for race condition
	existingProperty, err := s.repo.FindByID(ctx, newProperty.PropertyID)
	if err != nil {
		return nil, utils.LogAndMapError(ctx, utils.WrapError(err, "check existing property failed: propertyID=%s", newProperty.PropertyID),
			"check existing property",
			"propertyID", newProperty.PropertyID)
	}
	if existingProperty != nil {
		newProperty.ID = existingProperty.ID
		newProperty.PropertyID = existingProperty.PropertyID
		newProperty.UpdatedAt = time.Now()

		if err := s.repo.Update(ctx, newProperty); err != nil {
			return nil, utils.LogAndMapError(ctx, utils.WrapError(err, "update property failed: propertyID=%s", newProperty.PropertyID),
				"update property",
				"propertyID", newProperty.PropertyID)
		}

		if err := s.cacheProperty(ctx, newProperty, cacheKey); err != nil {
			logger.GlobalLogger.Warnf("Cache update failed: propertyID=%s, error=%v", newProperty.PropertyID, err)
		}
		ginCtx.Set("data_source", "CORELOGIC_API")
		ginCtx.Set("property_id", newProperty.PropertyID)
		return newProperty, nil
	}

	// Create new property
	newProperty.ID = primitive.NewObjectID()
	newProperty.UpdatedAt = time.Now()

	if err := s.repo.Create(ctx, newProperty); err != nil {
		return nil, utils.LogAndMapError(ctx, utils.WrapError(err, "create property failed: propertyID=%s", newProperty.PropertyID),
			"create property",
			"propertyID", newProperty.PropertyID)
	}

	// Cache new property
	if err := s.cacheProperty(ctx, newProperty, cacheKey); err != nil {
		logger.GlobalLogger.Warnf("Cache update failed: propertyID=%s, error=%v", newProperty.PropertyID, err)
	}
	ginCtx.Set("data_source", "CORELOGIC_API")
	ginCtx.Set("property_id", newProperty.PropertyID)
	return newProperty, nil
}
