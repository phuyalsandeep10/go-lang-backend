package repositories

import (
	"context"
	"encoding/json"
	"time"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/metrics"

	"github.com/go-redis/redis/v8"
)

type propertyCache struct {
	client *redis.Client
}

func NewPropertyCache() PropertyCache {
	return &propertyCache{
		client: cache.RedisClient,
	}
}

func (c *propertyCache) GetProperty(ctx context.Context, key string) (*models.Property, error) {
	start := time.Now()
	data, err := c.client.Get(ctx, key).Result()
	metrics.RedisOperationDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("get", "").Inc()
		return nil, err
	}
	var property models.Property
	if err := json.Unmarshal([]byte(data), &property); err != nil {
		return nil, err
	}
	return &property, nil
}

func (c *propertyCache) SetProperty(ctx context.Context, key string, property *models.Property, expiration time.Duration) error {
	data, err := json.Marshal(property)
	if err != nil {
		return err
	}
	start := time.Now()
	err = c.client.Set(ctx, key, data, expiration).Err()
	metrics.RedisOperationDuration.WithLabelValues("set").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set", "").Inc()
		return err
	}
	return nil
}

func (c *propertyCache) GetSearchKey(ctx context.Context, key string) (string, error) {
	start := time.Now()
	result, err := c.client.Get(ctx, key).Result()
	metrics.RedisOperationDuration.WithLabelValues("get_search").Observe(time.Since(start).Seconds())
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("get_search", "").Inc()
		return "", err
	}
	return result, nil
}

func (c *propertyCache) SetSearchKey(ctx context.Context, key, propertyID string, expiration time.Duration) error {
	start := time.Now()
	err := c.client.Set(ctx, key, propertyID, expiration).Err()
	metrics.RedisOperationDuration.WithLabelValues("set_search").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set_search", "").Inc()
		return err
	}
	return nil
}

func (c *propertyCache) AddCacheKeyToPropertySet(ctx context.Context, propertyID, cacheKey string) error {
	start := time.Now()
	err := c.client.SAdd(ctx, cache.PropertyKeysSetKey(propertyID), cacheKey).Err()
	metrics.RedisOperationDuration.WithLabelValues("sadd").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("sadd", "").Inc()
		return err
 }
	return nil
}

func (c *propertyCache) InvalidatePropertyCacheKeys(ctx context.Context, propertyID string) error {
	start := time.Now()
	keys, err := c.client.SMembers(ctx, cache.PropertyKeysSetKey(propertyID)).Result()
	metrics.RedisOperationDuration.WithLabelValues("smembers").Observe(time.Since(start).Seconds())
	if err != nil && err != redis.Nil {
		metrics.RedisErrorsTotal.WithLabelValues("smembers", "").Inc()
		return err
	}
	for _, key := range keys {
		start := time.Now()
		err = c.client.Del(ctx, key).Err()
		metrics.RedisOperationDuration.WithLabelValues("del").Observe(time.Since(start).Seconds())
		if err != nil && err != redis.Nil {
			metrics.RedisErrorsTotal.WithLabelValues("del", "").Inc()
		}
	}
	start = time.Now()
	err = c.client.Del(ctx, cache.PropertyKeysSetKey(propertyID)).Err()
	metrics.RedisOperationDuration.WithLabelValues("del_set").Observe(time.Since(start).Seconds())
	if err != nil && err != redis.Nil {
		metrics.RedisErrorsTotal.WithLabelValues("del_set", "").Inc()
		return err
	}
	start = time.Now()
	err = c.client.Del(ctx, cache.PropertyListKey()).Err()
	metrics.RedisOperationDuration.WithLabelValues("del_list").Observe(time.Since(start).Seconds())
	if err != nil && err != redis.Nil {
		metrics.RedisErrorsTotal.WithLabelValues("del_list", "").Inc()
	}
	return nil
}

func (c *propertyCache) Delete(ctx context.Context, key string) error {
	start := time.Now()
	err := c.client.Del(ctx, key).Err()
	metrics.RedisOperationDuration.WithLabelValues("del").Observe(time.Since(start).Seconds())
	if err != nil && err != redis.Nil {
		metrics.RedisErrorsTotal.WithLabelValues("del", "").Inc()
		return err
	}
	return nil
}

func (c *propertyCache) ClearAll(ctx context.Context) error {
	start := time.Now()
	err := c.client.FlushAll(ctx).Err()
	metrics.RedisOperationDuration.WithLabelValues("flush_all").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("flush_all", "").Inc()
		return err
	}
	return nil
}
