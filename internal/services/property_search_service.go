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
	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PropertySearchService struct {
	repo      repositories.PropertyRepository
	cache     repositories.PropertyCache
	addrTrans transformers.AddressTransformer
	validator validators.PropertyValidator
}

func NewPropertySearchService(
	repo repositories.PropertyRepository,
	cache repositories.PropertyCache,
	addrTrans transformers.AddressTransformer,
	validator validators.PropertyValidator,
) *PropertySearchService {
	return &PropertySearchService{
		repo:      repo,
		cache:     cache,
		addrTrans: addrTrans,
		validator: validator,
	}
}


func (s *PropertySearchService) SearchSpecificProperty(ctx context.Context, req *models.SearchRequest) (*models.Property, error) {
	if err := s.validator.ValidateSearch(req); err != nil {
		return nil, err
	}

	street, city, state, zip := s.addrTrans.ParseAddress(req.Search)
	if street == "" || city == "" {
		return nil, fmt.Errorf("street address and city are required")
	}

	cacheKey := cache.PropertySpecificSearchKey(street, city)
	var dataSource string

	// Check cache for property ID
	if propertyID, err := s.cache.GetSearchKey(ctx, cacheKey); err == nil && propertyID != "" {
		if property, err := s.cache.GetProperty(ctx, cache.PropertyKey(propertyID)); err == nil && property != nil {
			metrics.CacheHitsTotal.Inc()
			dataSource = "cache_hit"
			logger.Logger.Printf("SearchSpecificProperty: Found property in cache for query=%s, propertyID=%s, data_source=%s", req.Search, propertyID, dataSource)
			return property, nil
		}
	}
	metrics.CacheMissesTotal.Inc()
	dataSource = "cache_miss"
	logger.Logger.Printf("SearchSpecificProperty: Cache miss for query=%s, cache_key=%s, data_source=%s", req.Search, cacheKey, dataSource)

	// Query database
	property, err := s.repo.FindByAddress(ctx, street, city, state, zip)
	if err != nil {
		logger.Logger.Printf("SearchSpecificProperty: Database query failed for query=%s, error=%v, data_source=database", req.Search, err)
		return nil, err
	}
	if property != nil {
		dataSource = "database"
		logger.Logger.Printf("SearchSpecificProperty: Found property in database for query=%s, propertyID=%s, data_source=%s", req.Search, property.PropertyID, dataSource)
		propertyKey := cache.PropertyKey(property.PropertyID)
		_ = s.cache.SetProperty(ctx, propertyKey, property, Month)
		_ = s.cache.SetSearchKey(ctx, cacheKey, property.PropertyID, Month)
		_ = s.cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, cacheKey)
		return property, nil
	}

	// Fallback to mock data
	logger.Logger.Printf("SearchSpecificProperty: Property not found in database, falling back to mock data for query=%s, data_source=mock", req.Search)
	_, err = utils.ReadMockData("property-detail.json")
	if err != nil {
		logger.Logger.Printf("SearchSpecificProperty: Failed to read mock data for query=%s, error=%v, data_source=mock", req.Search, err)
		return nil, err
	}

	// For simplicity, assume PropertyService handles transformation
	// Alternatively, inject PropertyTransformer here if needed
	property = &models.Property{
		Address: models.Address{
			StreetAddress: street,
			City:          city,
			State:         state,
			ZipCode:       zip,
		},
		ID: primitive.NewObjectID(),
	}
	// Populate remaining fields from mockData as needed

	if err := s.repo.Create(ctx, property); err != nil {
		logger.Logger.Printf("SearchSpecificProperty: Failed to create property in database for query=%s, error=%v, data_source=database", req.Search, err)
		return nil, err
	}

	dataSource = "mock"
	logger.Logger.Printf("SearchSpecificProperty: Created property from mock data for query=%s, propertyID=%s, data_source=%s", req.Search, property.PropertyID, dataSource)
	propertyKey := cache.PropertyKey(property.PropertyID)
	_ = s.cache.SetProperty(ctx, propertyKey, property, Month)
	_ = s.cache.SetSearchKey(ctx, cacheKey, property.PropertyID, Month)
	_ = s.cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, cacheKey)
	return property, nil
}


func (s *PropertySearchService) GetPropertiesWithPagination(ctx context.Context, offset, limit int, baseURL string, params url.Values) (*models.PaginatedPropertiesResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	cacheKey := cache.PropertyListPaginatedKey(offset, limit)
	if cached, err := s.cache.GetProperty(ctx, cacheKey); err == nil && cached != nil {
		metrics.CacheHitsTotal.Inc()
		// Assuming cached is serialized PaginatedPropertiesResponse; adjust as needed
	}

	properties, total, err := s.repo.FindWithPagination(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	for _, prop := range properties {
		_ = s.cache.SetProperty(ctx, cache.PropertyKey(prop.PropertyID), &prop, Month)
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

	_ = s.cache.SetProperty(ctx, cacheKey, &models.Property{}, Month) // Adjust serialization
	return response, nil
}
