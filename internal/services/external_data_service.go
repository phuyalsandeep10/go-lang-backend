
package services

import (
	"context"
	"fmt"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/transformers"
	"homeinsight-properties/pkg/corelogic"
	"homeinsight-properties/pkg/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ExternalDataService struct {
	corelogic *corelogic.Client
	propTrans transformers.PropertyTransformer
}

func NewExternalDataService(
	corelogicClient *corelogic.Client,
	propTrans transformers.PropertyTransformer,
) *ExternalDataService {
	return &ExternalDataService{
		corelogic: corelogicClient,
		propTrans: propTrans,
	}
}

func (s *ExternalDataService) FetchFromExternalSource(ctx context.Context, street, city, state, zip string, req *models.SearchRequest) (*models.Property, error) {
	// Option 1: Use CoreLogic API
	property, err := s.corelogic.RequestCoreLogic(ctx, street, city, state, zip)
	if err != nil {
		logger.GlobalLogger.Errorf("CoreLogic failed: query=%s, error=%v", req.Search, err)
		return nil, fmt.Errorf("failed to fetch from CoreLogic: %v", err)
	}

	// Option 2: Use Mock Data
	// property, err = utils.ReadMockData(ctx, "property-detail.json", s.propTrans)
	// if err != nil {
	// 	logger.GlobalLogger.Errorf("Mock data read failed: query=%s, error=%v", req.Search, err)
	// 	return nil, fmt.Errorf("failed to read mock data: %v", err)
	// }

	// Override address fields with search input
	property.Address.StreetAddress = street
	property.Address.City = city
	property.Address.State = state
	property.Address.ZipCode = zip

	// Generate a new ID
	property.ID = primitive.NewObjectID()

	return property, nil
}
