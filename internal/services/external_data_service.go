package services

import (
	"context"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/transformers"
	"homeinsight-properties/internal/utils"
	"homeinsight-properties/pkg/config"
	"homeinsight-properties/pkg/corelogic"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ExternalDataService struct {
	corelogic *corelogic.Client
	propTrans transformers.PropertyTransformer
	config    *config.Config
}

func NewExternalDataService(
	corelogicClient *corelogic.Client,
	propTrans transformers.PropertyTransformer,
	cfg *config.Config,
) *ExternalDataService {
	return &ExternalDataService{
		corelogic: corelogicClient,
		propTrans: propTrans,
		config:    cfg,
	}
}

func (s *ExternalDataService) FetchFromExternalSource(ctx context.Context, street, city, state, zip string, req *models.SearchRequest) (*models.Property, error) {
	ginCtx, _ := ctx.(*gin.Context)
	if ginCtx == nil {
		ginCtx = &gin.Context{}
	}

	// Request CoreLogic
	property, err := s.corelogic.RequestCoreLogic(ctx, street, city, state, zip)
	if err != nil {
		return nil, utils.WrapError(err, "CoreLogic fetch failed: query=%s", req.Search)
	}

	// Override address fields with search input
	property.Address.StreetAddress = street
	property.Address.City = city
	property.Address.State = state
	property.Address.ZipCode = zip

	// Generate a new ID
	property.ID = primitive.NewObjectID()

	return property, nil
}
