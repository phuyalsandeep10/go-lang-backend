package cache

import (
	"context"
	"time"

	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

)

// add a cache key to the set of keys associated with a property ID.
func AddCacheKeyToPropertySet(ctx context.Context, propertyID, cacheKey string) error {
	start := time.Now()
	setKey := PropertyKeysSetKey(propertyID)
	_, err := RedisClient.SAdd(ctx, setKey, cacheKey).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("sadd").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("sadd").Inc()
		logger.GlobalLogger.Errorf("failed to add cache key %s to set %s: %v", cacheKey, setKey, err)
		return NewCacheError("sadd", err, false)
	}
	return nil
}

// retrieve all cache keys associated with a property ID.
func GetCacheKeysForProperty(ctx context.Context, propertyID string) ([]string, error) {
	start := time.Now()
	setKey := PropertyKeysSetKey(propertyID)
	cacheKeys, err := RedisClient.SMembers(ctx, setKey).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("smembers").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("smembers").Inc()
		logger.GlobalLogger.Errorf("failed to get cache keys for property %s: %v", propertyID, err)
		return nil, NewCacheError("smembers", err, false)
	}
	return cacheKeys, nil
}

// invalidate all cache keys associated with a property ID using a Lua script.
func InvalidatePropertyCacheKeys(ctx context.Context, propertyID string) error {
	start := time.Now()
	_, err := invalidatePropertyCacheScript.Run(ctx, RedisClient, []string{}, propertyID).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("invalidate_cache").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("invalidate_cache").Inc()
		logger.GlobalLogger.Errorf("failed to execute invalidate property cache script for property %s: %v", propertyID, err)
		return NewCacheError("invalidate_cache", err, false)
	}
	return nil
}
