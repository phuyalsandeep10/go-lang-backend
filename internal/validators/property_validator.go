package validators

import (
	"fmt"

	"homeinsight-properties/internal/models"
)

type propertyValidator struct{}

func NewPropertyValidator() PropertyValidator {
	return &propertyValidator{}
}

func (v *propertyValidator) ValidateCreate(property *models.Property) error {
	if property.PropertyID == "" || property.Address.StreetAddress == "" {
		return fmt.Errorf("property ID and street address are required")
	}
	return nil
}

func (v *propertyValidator) ValidateUpdate(property *models.Property) error {
	if property.PropertyID == "" || property.Address.StreetAddress == "" {
		return fmt.Errorf("property ID and street address are required")
	}
	return nil
}

func (v *propertyValidator) ValidateSearch(req *models.SearchRequest) error {
	if req.Search == "" {
		return fmt.Errorf("search query is required")
	}
	return nil
}
