package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"homeinsight-properties/pkg/metrics"
)

func ReadMockData(filename string) (map[string]interface{}, error) {
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
	return result, nil
}
