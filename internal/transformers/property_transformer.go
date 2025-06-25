package transformers

import (
	"fmt"
	"strings"
	"time"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/metrics"
)

type propertyTransformer struct{}

func NewPropertyTransformer() PropertyTransformer {
	return &propertyTransformer{}
}

func (t *propertyTransformer) TransformAPIResponse(apiResponse map[string]interface{}) (*models.Property, error) {
	start := time.Now()
	defer func() {
		metrics.MongoOperationDuration.WithLabelValues("transform_api_response", "").Observe(time.Since(start).Seconds())
	}()

	property := &models.Property{}

	if buildings, ok := apiResponse["buildings"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if clip, ok := buildings["clip"].(string); ok && clip != "" {
			property.PropertyID = clip
			property.AVMPropertyID = fmt.Sprintf("47149:%s", clip)
		} else {
			metrics.MongoErrorsTotal.WithLabelValues("transform_api_response", "").Inc()
			return nil, fmt.Errorf("clip field is missing or invalid")
		}
	} else {
		metrics.MongoErrorsTotal.WithLabelValues("transform_api_response", "").Inc()
		return nil, fmt.Errorf("buildings.data field is missing")
	}

	// Address transformation (assuming AddressTransformer is injected if needed)
	if ownership, ok := apiResponse["ownership"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if mailing, ok := ownership["currentOwnerMailingInfo"].(map[string]interface{})["mailingAddress"].(map[string]interface{}); ok {
			property.Address = models.Address{
				StreetAddress: getString(mailing, "streetAddress"),
				City:          getString(mailing, "city"),
				State:         getString(mailing, "state"),
				ZipCode:       getString(mailing, "zipCode"),
				CarrierRoute:  getString(mailing, "carrierRoute"),
			}
			if parsed, ok := mailing["streetAddressParsed"].(map[string]interface{}); ok {
				property.Address.StreetAddressParsed = models.StreetAddressParsed{
					HouseNumber:      getString(parsed, "houseNumber"),
					StreetName:       getString(parsed, "streetName"),
					StreetNameSuffix: getString(parsed, "mailingMode"),
				}
			}
		}
	}


	//  Location, Lot, LandUseAndZoning, Utilities, Building, Ownership, TaxAssessment, LastMarketSale
	//  getString, getInt, getFloat64, getBool helpers as in original code

	return property, nil
}

func getString(m map[string]interface{}, key string) string {
	keys := strings.Split(key, ".")
	current := m
	for _, k := range keys[:len(keys)-1] {
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			return ""
		}
	}
	if val, ok := current[keys[len(keys)-1]]; ok && val != nil {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// getInt, getFloat64, getBool implementations
