package services

import (
	"context"
	"time"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/repositories"
	"homeinsight-properties/internal/transformers"
	"homeinsight-properties/internal/validators"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/corelogic"
)

const Month = 30 * 24 * time.Hour

type PropertyService struct {
	repo      repositories.PropertyRepository
	cache     repositories.PropertyCache
	trans     transformers.PropertyTransformer
	addrTrans transformers.AddressTransformer
	validator validators.PropertyValidator
	corelogic *corelogic.Client
}

func NewPropertyService(
	repo repositories.PropertyRepository,
	cache repositories.PropertyCache,
	trans transformers.PropertyTransformer,
	addrTrans transformers.AddressTransformer,
	validator validators.PropertyValidator,
	corelogicClient *corelogic.Client,
) *PropertyService {
	return &PropertyService{
		repo:      repo,
		cache:     cache,
		trans:     trans,
		addrTrans: addrTrans,
		validator: validator,
		corelogic: corelogicClient,
	}
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
