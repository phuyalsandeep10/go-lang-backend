package services

import (
	"context"

	"homeinsight-properties/internal/repositories"
	"homeinsight-properties/internal/transformers"
	"homeinsight-properties/pkg/cache"
)

type PropertyMigrationService struct {
	repo      repositories.PropertyRepository
	cache     repositories.PropertyCache
	addrTrans transformers.AddressTransformer
}

func NewPropertyMigrationService(
	repo repositories.PropertyRepository,
	cache repositories.PropertyCache,
	addrTrans transformers.AddressTransformer,
) *PropertyMigrationService {
	return &PropertyMigrationService{
		repo:      repo,
		cache:     cache,
		addrTrans: addrTrans,
	}
}

func (s *PropertyMigrationService) MigrateAddressesToUppercase(ctx context.Context) error {
	properties, err := s.repo.FindAll(ctx)
	if err != nil {
		return err
	}

	for _, property := range properties {
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
		if property.Address.CarrierRoute != "" {
			property.Address.CarrierRoute = s.addrTrans.NormalizeAddressComponent(property.Address.CarrierRoute)
		}
		// Add other address fields as needed

		if err := s.repo.Update(ctx, &property); err != nil {
			continue
		}

		propertyKey := cache.PropertyKey(property.PropertyID)
		_ = s.cache.SetProperty(ctx, propertyKey, &property, Month)
		_ = s.cache.InvalidatePropertyCacheKeys(ctx, property.PropertyID)
	}
	return nil
}

func (s *PropertyMigrationService) ClearAllCache(ctx context.Context) error {
	return s.cache.ClearAll(ctx)
}
