package cache

import (
	"context"
	"encoding/json"
	"time"

	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

)

// store a value in the cache with the given key and expiration time.
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	start := time.Now()
	data, err := json.Marshal(value)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set_marshal").Inc()
		logger.GlobalLogger.Errorf("failed to marshal value for key %s: %v", key, err)
		return NewCacheError("marshal", err, true)
	}
	err = RedisClient.Set(ctx, key, data, expiration).Err()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("set").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set").Inc()
		logger.GlobalLogger.Errorf("failed to set key %s: %v", key, err)
		return NewCacheError("set", err, false)
	}
	return nil
}

// retrieve a value from the cache and unmarshals it into the provided destination.
func Get(ctx context.Context, key string, dest interface{}) error {
	start := time.Now()
	val, err := RedisClient.Get(ctx, key).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("get").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("get").Inc()
		logger.GlobalLogger.Errorf("failed to get key %s: %v", key, err)
		return NewCacheError("get", err, false)
	}
	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("get_unmarshal").Inc()
		logger.GlobalLogger.Errorf("failed to unmarshal value for key %s: %v", key, err)
		return NewCacheError("unmarshal", err, true)
	}
	return nil
}

// remove a exclusivement key from the cache.
func Delete(ctx context.Context, key string) error {
	start := time.Now()
	err := RedisClient.Del(ctx, key).Err()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("delete").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("delete").Inc()
		logger.GlobalLogger.Errorf("failed to delete key %s: %v", key, err)
		return NewCacheError("delete", err, false)
	}
	return nil
}

// check if a key exists in the cache.
func Exists(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	count, err := RedisClient.Exists(ctx, key).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("exists").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("exists").Inc()
		logger.GlobalLogger.Errorf("failed to check existence of key %s: %v", key, err)
		return false, NewCacheError("exists", err, false)
	}
	return count > 0, nil
}
