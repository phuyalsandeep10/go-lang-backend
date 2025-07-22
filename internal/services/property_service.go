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
)

type PropertyService struct {
	repo      repositories.PropertyRepository
	cache     repositories.PropertyCache
	trans     transformers.PropertyTransformer
	addrTrans transformers.AddressTransformer
	validator validators.PropertyValidator
	corelogic *corelogic.Client
	config    *config.Config
	cacheTTL  time.Duration
}

func NewPropertyService(
	repo repositories.PropertyRepository,
	cache repositories.PropertyCache,
	trans transformers.PropertyTransformer,
	addrTrans transformers.AddressTransformer,
	validator validators.PropertyValidator,
	corelogicClient *corelogic.Client,
	cfg *config.Config,
) *PropertyService {
	return &PropertyService{
		repo:      repo,
		cache:     cache,
		trans:     trans,
		addrTrans: addrTrans,
		validator: validator,
		corelogic: corelogicClient,
		config:    cfg,
		cacheTTL:  time.Duration(cfg.Redis.CacheTTLDays) * 24 * time.Hour,
	}
}

func (s *PropertyService) GetPropertyByID(ctx context.Context, id string) (*models.Property, error) {
	ginCtx, _ := ctx.(*gin.Context)
	if ginCtx == nil {
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
	if err := s.cache.SetProperty(ctx, propertyKey, property, s.cacheTTL); err != nil {
		logger.GlobalLogger.Errorf("Failed to cache property: id=%s, error=%v", id, err)
	}
	if err := s.cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, propertyKey); err != nil {
		logger.GlobalLogger.Errorf("Failed to add cache key to property set: id=%s, key=%s, error=%v", id, propertyKey, err)
	}

	return property, nil
}

func (s *PropertyService) CreateProperty(ctx context.Context, property *models.Property) error {
	if err := s.validator.ValidateCreate(property); err != nil {
		return err
	}

	s.normalizeAddress(property)
	if err := s.repo.Create(ctx, property); err != nil {
		return err
	}

	propertyKey := cache.PropertyKey(property.PropertyID)
	if err := s.cache.SetProperty(ctx, propertyKey, property, s.cacheTTL); err != nil {
		logger.GlobalLogger.Errorf("Failed to cache property: id=%s, error=%v", property.PropertyID, err)
	}
	if err := s.cache.InvalidatePropertyCacheKeys(ctx, property.PropertyID); err != nil {
		logger.GlobalLogger.Errorf("Failed to invalidate cache keys: id=%s, error=%v", property.PropertyID, err)
	}
	return nil
}

func (s *PropertyService) UpdateProperty(ctx context.Context, property *models.Property) error {
	if err := s.validator.ValidateUpdate(property); err != nil {
		return err
	}

	s.normalizeAddress(property)
	if err := s.repo.Update(ctx, property); err != nil {
		return err
	}

	propertyKey := cache.PropertyKey(property.PropertyID)
	if err := s.cache.SetProperty(ctx, propertyKey, property, s.cacheTTL); err != nil {
		logger.GlobalLogger.Errorf("Failed to cache property: id=%s, error=%v", property.PropertyID, err)
	}
	if err := s.cache.InvalidatePropertyCacheKeys(ctx, property.PropertyID); err != nil {
		logger.GlobalLogger.Errorf("Failed to invalidate cache keys: id=%s, error=%v", property.PropertyID, err)
	}
	return nil
}

func (s *PropertyService) DeleteProperty(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if err := s.cache.InvalidatePropertyCacheKeys(ctx, id); err != nil {
		logger.GlobalLogger.Errorf("Failed to invalidate cache keys: id=%s, error=%v", id, err)
	}
	return nil
}

func (s *PropertyService) normalizeAddress(property *models.Property) {
	property.Address.StreetAddress = s.addrTrans.NormalizeAddressComponent(property.Address.StreetAddress)
	if property.Address.City != "" {
		property.Address.City = s.addrTrans.NormalizeAddressComponent(property.Address.City)
	}
	if property.Address.State != "" {
		property.Address.State = s.addrTrans.NormalizeAddressComponent(property.Address.State)
	}
	if property.Address.ZipCode != "" {
		property.Address.ZipCode = s.addrTrans.NormalizeAddressComponent(property.Address.ZipCode)
	}
}
