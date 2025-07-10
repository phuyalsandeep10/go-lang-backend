package utils

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/internal/transformers"
	"homeinsight-properties/pkg/metrics"

	"github.com/gin-gonic/gin"
)

func ReadMockData(ctx context.Context, filename string, propTrans transformers.PropertyTransformer) (*models.Property, error) {

	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		ginCtx = &gin.Context{}
	}

	ginCtx.Set("data_source", "MOCK_DATA")
	start := time.Now()
	filePath, err := filepath.Abs("data/coreLogic/" + filename)
	metrics.MongoOperationDuration.WithLabelValues("read_mock_file_path", "").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("read_mock_file_path", "").Inc()
		return nil, err
	}

	start = time.Now()
	data, err := os.ReadFile(filePath)
	metrics.MongoOperationDuration.WithLabelValues("read_mock_file", "").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("read_mock_file", "").Inc()
		return nil, err
	}

	var result map[string]interface{}
	start = time.Now()
	err = json.Unmarshal(data, &result)
	metrics.MongoOperationDuration.WithLabelValues("unmarshal_mock_data", "").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("unmarshal_mock_data", "").Inc()
		return nil, err
	}

	// Transform mock data to models.Property
	property, err := propTrans.TransformAPIResponse(result)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("transform_mock_data", "").Inc()
		return nil, err
	}

	return property, nil
}
