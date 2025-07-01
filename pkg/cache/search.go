package cache

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

)

// SetSearchResult caches a list of property IDs for a search key with an expiration time.
// It also associates the search key with each property ID for invalidation purposes.
func SetSearchResult(ctx context.Context, key string, propertyIDs []string, expiration time.Duration) error {
	start := time.Now()
	propertyIDsJSON, err := json.Marshal(propertyIDs)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set_search_marshal").Inc()
		logger.GlobalLogger.Errorf("failed to marshal property IDs for key %s: %v", key, err)
		return NewCacheError("set_search_marshal", err, true)
	}

	args := []interface{}{key, string(propertyIDsJSON), strconv.Itoa(int(expiration.Seconds()))}
	for _, id := range propertyIDs {
		args = append(args, id)
	}

	_, err = setSearchResultScript.Run(ctx, RedisClient, []string{}, args...).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("set_search_result").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set_search_result").Inc()
		logger.GlobalLogger.Errorf("failed to execute set search result script for key %s: %v", key, err)
		return NewCacheError("set_search_result", err, false)
	}
	return nil
}

// GetSearchResult retrieves a cached list of property IDs for a search key.
func GetSearchResult(ctx context.Context, key string) ([]string, error) {
	start := time.Now()
	val, err := RedisClient.Get(ctx, key).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("get_search_result").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("get_search_result").Inc()
		logger.GlobalLogger.Errorf("failed to get search result for key %s: %v", key, err)
		return nil, NewCacheError("get_search_result", err, false)
	}
	var propertyIDs []string
	if err := json.Unmarshal([]byte(val), &propertyIDs); err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("get_search_unmarshal").Inc()
		logger.GlobalLogger.Errorf("failed to unmarshal property IDs for key %s: %v", key, err)
		return nil, NewCacheError("get_search_unmarshal", err, true)
	}
	return propertyIDs, nil
}
