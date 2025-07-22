package services

import (
	"context"
	"net/url"
	"strconv"
	"time"

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
	ginCtx.Set("query", "offset="+strconv.Itoa(offset)+",limit="+strconv.Itoa(limit))

	// Query database
	var properties []models.Property
	var total int64
	var err error
	for attempt := 1; attempt <= s.config.ErrorHandling.RetryAttempts; attempt++ {
		properties, total, err = s.repo.FindWithPagination(ctx, offset, limit)
		if err == nil || !utils.IsRetryableError(err) {
			break
		}
		logger.GlobalLogger.Warnf("Database query attempt %d/%d failed: offset=%d, limit=%d, error=%v", attempt, s.config.ErrorHandling.RetryAttempts, offset, limit, err)
		time.Sleep(time.Duration(s.config.ErrorHandling.RetryDelayMS) * time.Millisecond)
	}
	if err != nil {
		return nil, utils.LogAndMapError(ctx, err, "list properties",
			"offset", offset,
			"limit", limit)
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
