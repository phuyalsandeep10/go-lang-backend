package services

import (
	"context"
	"fmt"
	"net/url"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/repositories"
	"homeinsight-properties/internal/transformers"
	"homeinsight-properties/internal/utils"
	"homeinsight-properties/internal/validators"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/corelogic"
	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PropertySearchService struct {
	repo      repositories.PropertyRepository
	cache     repositories.PropertyCache
	addrTrans transformers.AddressTransformer
	propTrans transformers.PropertyTransformer
	validator validators.PropertyValidator
	corelogic *corelogic.Client
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
		repo:      repo,
		cache:     cache,
		addrTrans: addrTrans,
		propTrans: propTrans,
		validator: validator,
		corelogic: corelogicClient,
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
	if property != nil {
		ginCtx.Set("data_source", "DATABASE")
		ginCtx.Set("property_id", property.PropertyID)
		propertyKey := cache.PropertyKey(property.PropertyID)
		_ = s.cache.SetProperty(ctx, propertyKey, property, Month)
		_ = s.cache.SetSearchKey(ctx, cacheKey, property.PropertyID, Month)
		_ = s.cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, cacheKey)
		return property, nil
	}

	// Fallback to external data source

	// Option 1: Use CoreLogic API
	// property, err = s.corelogic.RequestCoreLogic(ctx, street, city, state, zip)
	// if err != nil {
	//     logger.GlobalLogger.Errorf("CoreLogic failed: query=%s, error=%v", req.Search, err)
	//     return nil, fmt.Errorf("failed to fetch from CoreLogic: %v", err)
	// }

	// Option 2: Use Mock Data
	property, err = utils.ReadMockData(ctx,"property-detail.json", s.propTrans)
	if err != nil {
		logger.GlobalLogger.Errorf("Mock data read failed: query=%s, error=%v", req.Search, err)
		return nil, fmt.Errorf("failed to read mock data: %v", err)
	}

	// Override address fields with search input
	property.Address.StreetAddress = street
	property.Address.City = city
	property.Address.State = state
	property.Address.ZipCode = zip

	// Generate a new ID
	property.ID = primitive.NewObjectID()
	ginCtx.Set("property_id", property.PropertyID)

	// Insert into database
	if err := s.repo.Create(ctx, property); err != nil {
		logger.GlobalLogger.Errorf("Failed to create property: propertyID=%s, error=%v", property.PropertyID, err)
		return nil, fmt.Errorf("failed to create property: %v", err)
	}

	// Cache the property
	propertyKey := cache.PropertyKey(property.PropertyID)
	_ = s.cache.SetProperty(ctx, propertyKey, property, Month)
	_ = s.cache.SetSearchKey(ctx, cacheKey, property.PropertyID, Month)
	_ = s.cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, cacheKey)

	return property, nil
}

func (s *PropertySearchService) GetPropertiesWithPagination(ctx context.Context, offset, limit int, baseURL string, params url.Values) (*models.PaginatedPropertiesResponse, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		ginCtx = &gin.Context{}
	}

	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	ginCtx.Set("data_source", "DATABASE")
	ginCtx.Set("query", fmt.Sprintf("offset=%d,limit=%d", offset, limit))

	// Query database
	properties, total, err := s.repo.FindWithPagination(ctx, offset, limit)
	if err != nil {
		logger.GlobalLogger.Errorf("DB query failed: offset=%d, limit=%d, error=%v", offset, limit, err)
		return nil, err
	}

	metadata := models.PaginationMeta{
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}
	if int64(offset+limit) < total {
		nextURL := utils.BuildPaginationURL(baseURL, offset+limit, limit, params)
		metadata.Next = &nextURL
	}
	if offset > 0 {
		prevOffset := offset - limit
		if prevOffset < 0 {
			prevOffset = 0
		}
		prevURL := utils.BuildPaginationURL(baseURL, prevOffset, limit, params)
		metadata.Prev = &prevURL
	}

	response := &models.PaginatedPropertiesResponse{
		Data:     make([]models.PropertyResponse, len(properties)),
		Metadata: metadata,
	}
	for i, prop := range properties {
		response.Data[i] = models.PropertyResponse{Property: &prop}
	}

	return response, nil
}
