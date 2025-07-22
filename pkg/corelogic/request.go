package corelogic

import (
    "context"
    "fmt"

    "homeinsight-properties/internal/models"
    "homeinsight-properties/internal/transformers"
    "homeinsight-properties/pkg/logger"

    "github.com/gin-gonic/gin"
)

// RequestCoreLogic handles the actual CoreLogic API call
func (c *Client) RequestCoreLogic(ctx context.Context, street, city, state, zip string) (*models.Property, error) {
    ginCtx, ok := ctx.(*gin.Context)
    if !ok {
        ginCtx = &gin.Context{}
    }

    ginCtx.Set("data_source", "CORELOGIC_API")

    // Get the authentication token
    token, err := c.getToken()
    if err != nil {
        logger.GlobalLogger.Errorf("Failed to get token: error=%v", err)
        return nil, fmt.Errorf("failed to get authentication token: %v", err)
    }

    // Search for property by address
    clip, v1PropertyId, err := c.SearchPropertyByAddress(token, street, city, state, zip)
    if err != nil {
        return nil, fmt.Errorf("failed to search property: %v", err)
    }

    // Get property details
    details, err := c.GetPropertyDetails(token, clip)
    if err != nil {
        logger.GlobalLogger.Errorf("CoreLogic details failed: clip=%s, error=%v", clip, err)
        return nil, fmt.Errorf("failed to get property details: %v", err)
    }

    // Transform API response
    propTrans := transformers.NewPropertyTransformer()
    property, err := propTrans.TransformAPIResponse(details)
    if err != nil {
        logger.GlobalLogger.Errorf("Failed to transform CoreLogic data: clip=%s, error=%v", clip, err)
        return nil, fmt.Errorf("failed to transform property data: %v", err)
    }

    // Set PropertyID and AVMPropertyID
    property.PropertyID = clip
    property.AVMPropertyID = v1PropertyId

    return property, nil
}
