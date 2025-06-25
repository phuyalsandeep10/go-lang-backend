package services

import (
	"context"
	"time"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/repositories"
	"homeinsight-properties/internal/transformers"
	"homeinsight-properties/internal/utils"
	"homeinsight-properties/internal/validators"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/metrics"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const Month = 30 * 24 * time.Hour

type PropertyService struct {
	repo      repositories.PropertyRepository
	cache     repositories.PropertyCache
	trans     transformers.PropertyTransformer
	addrTrans transformers.AddressTransformer
	validator validators.PropertyValidator
}

func NewPropertyService(
	repo repositories.PropertyRepository,
	cache repositories.PropertyCache,
	trans transformers.PropertyTransformer,
	addrTrans transformers.AddressTransformer,
	validator validators.PropertyValidator,
) *PropertyService {
	return &PropertyService{
		repo:      repo,
		cache:     cache,
		trans:     trans,
		addrTrans: addrTrans,
		validator: validator,
	}
}

func (s *PropertyService) GetPropertyByID(ctx context.Context, id string) (*models.Property, error) {
	propertyKey := cache.PropertyKey(id)
	if property, err := s.cache.GetProperty(ctx, propertyKey); err == nil && property != nil {
		metrics.CacheHitsTotal.Inc()
		return property, nil
	}
	metrics.CacheMissesTotal.Inc()

	property, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if property != nil {
		_ = s.cache.SetProperty(ctx, propertyKey, property, Month)
		return property, nil
	}

	mockData, err := utils.ReadMockData("property-detail.json")
	if err != nil {
		return nil, err
	}

	property, err = s.trans.TransformAPIResponse(mockData)
	if err != nil {
		return nil, err
	}
	property.ID = primitive.NewObjectID()

	if err := s.repo.Create(ctx, property); err != nil {
		return nil, err
	}

	_ = s.cache.SetProperty(ctx, propertyKey, property, Month)
	_ = s.cache.InvalidatePropertyCacheKeys(ctx, property.PropertyID)
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
	_ = s.cache.SetProperty(ctx, propertyKey, property, Month)
	_ = s.cache.InvalidatePropertyCacheKeys(ctx, property.PropertyID)
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
	_ = s.cache.SetProperty(ctx, propertyKey, property, Month)
	_ = s.cache.InvalidatePropertyCacheKeys(ctx, property.PropertyID)
	return nil
}

func (s *PropertyService) DeleteProperty(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.cache.InvalidatePropertyCacheKeys(ctx, id)
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
