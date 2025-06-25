package transformers

import (
	"homeinsight-properties/internal/models"
)

type PropertyTransformer interface {
	TransformAPIResponse(apiResponse map[string]interface{}) (*models.Property, error)
}

type AddressTransformer interface {
	NormalizeAddressComponent(input string) string
	ParseAddress(search string) (street, city, state, zip string)
}
