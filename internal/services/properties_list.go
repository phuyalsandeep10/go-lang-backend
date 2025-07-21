package services

import (
	"context"
	"fmt"
	"net/url"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/utils"
	"homeinsight-properties/pkg/logger"

	"github.com/gin-gonic/gin"
)

func (s *PropertySearchService) ListProperties(ctx context.Context, offset, limit int, baseURL string, params url.Values) (*models.PaginatedPropertiesResponse, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		ginCtx = &gin.Context{}
	}

	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	ginCtx.Set("data_source", "DATABASE")
	ginCtx.Set("query", fmt.Sprintf("offset=%d,limit=%d", offset, limit))

	// Query database
	properties, total, err := s.repo.FindWithPagination(ctx, offset, limit)
	if err != nil {
		logger.GlobalLogger.Errorf("DB query failed: offset=%d, limit=%d, error=%v", offset, limit, err)
		return nil, err
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
		Data:     properties,
		Metadata: metadata,
	}

	return response, nil
}
