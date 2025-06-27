package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"homeinsight-properties/pkg/logger" // Import the custom logger
	"homeinsight-properties/pkg/metrics"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client
var setSearchResultScript *redis.Script
var invalidatePropertyCacheScript *redis.Script

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func LoadRedisConfig() *RedisConfig {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}

	portStr := os.Getenv("REDIS_PORT")
	if portStr == "" {
		portStr = "6379"
	}
	port, _ := strconv.Atoi(portStr)

	password := os.Getenv("REDIS_PASSWORD")

	dbStr := os.Getenv("REDIS_DB")
	if dbStr == "" {
		dbStr = "0"
	}
	db, _ := strconv.Atoi(dbStr)

	return &RedisConfig{
		Host:     host,
		Port:     port,
		Password: password,
		DB:       db,
	}
}

func InitRedis() error {
	cfg := LoadRedisConfig()

	RedisClient = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	_, err := RedisClient.Ping(ctx).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("ping").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("ping").Inc()
		logger.GlobalLogger.Errorf("failed to connect to Redis: %v", err)
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}

	setSearchResultScript = redis.NewScript(`
		local search_key = ARGV[1]
		local property_ids_json = ARGV[2]
		local search_expiration = tonumber(ARGV[3])
		redis.call('SET', search_key, property_ids_json)
		redis.call('EXPIRE', search_key, search_expiration)
		for i = 4, #ARGV do
			local property_id = ARGV[i]
			local set_key = 'property:keys:' .. property_id
			redis.call('SADD', set_key, search_key)
			redis.call('EXPIRE', set_key, 3600)
		end
		return 1
	`)

	invalidatePropertyCacheScript = redis.NewScript(`
		local set_key = 'property:keys:' .. ARGV[1]
		local cache_keys = redis.call('SMEMBERS', set_key)
		if #cache_keys > 0 then
			redis.call('DEL', unpack(cache_keys))
		end
		redis.call('DEL', set_key)
		return 1
	`)

	logger.GlobalLogger.Println("Redis connected successfully")
	return nil
}

func CloseRedis() {
	if RedisClient != nil {
		if err := RedisClient.Close(); err != nil {
			logger.GlobalLogger.Errorf("Error closing Redis: %v", err)
		} else {
			logger.GlobalLogger.Println("Redis connection closed")
		}
	}
}

func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	start := time.Now()
	data, err := json.Marshal(value)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set_marshal").Inc()
		logger.GlobalLogger.Errorf("failed to marshal value for key %s: %v", key, err)
		return fmt.Errorf("failed to marshal value: %v", err)
	}
	err = RedisClient.Set(ctx, key, data, expiration).Err()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("set").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set").Inc()
		logger.GlobalLogger.Errorf("failed to set key %s: %v", key, err)
		return err
	}
	return nil
}

func Get(ctx context.Context, key string, dest interface{}) error {
	start := time.Now()
	val, err := RedisClient.Get(ctx, key).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("get").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("get").Inc()
		logger.GlobalLogger.Errorf("failed to get key %s: %v", key, err)
		return err
	}
	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("get_unmarshal").Inc()
		logger.GlobalLogger.Errorf("failed to unmarshal value for key %s: %v", key, err)
		return fmt.Errorf("failed to unmarshal value: %v", err)
	}
	return nil
}

func Delete(ctx context.Context, key string) error {
	start := time.Now()
	err := RedisClient.Del(ctx, key).Err()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("delete").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("delete").Inc()
		logger.GlobalLogger.Errorf("failed to delete key %s: %v", key, err)
		return err
	}
	return nil
}

func Exists(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	count, err := RedisClient.Exists(ctx, key).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("exists").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("exists").Inc()
		logger.GlobalLogger.Errorf("failed to check existence of key %s: %v", key, err)
		return false, err
	}
	return count > 0, nil
}

func SetSearchResult(ctx context.Context, key string, propertyIDs []string, expiration time.Duration) error {
	start := time.Now()
	propertyIDsJSON, err := json.Marshal(propertyIDs)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set_search_marshal").Inc()
		logger.GlobalLogger.Errorf("failed to marshal property IDs for key %s: %v", key, err)
		return fmt.Errorf("failed to marshal property IDs: %v", err)
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
		return fmt.Errorf("failed to execute set search result script: %v", err)
	}
	return nil
}

func GetSearchResult(ctx context.Context, key string) ([]string, error) {
	start := time.Now()
	val, err := RedisClient.Get(ctx, key).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("get_search_result").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("get_search_result").Inc()
		logger.GlobalLogger.Errorf("failed to get search result for key %s: %v", key, err)
		return nil, err
	}
	var propertyIDs []string
	if err := json.Unmarshal([]byte(val), &propertyIDs); err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("get_search_unmarshal").Inc()
		logger.GlobalLogger.Errorf("failed to unmarshal property IDs for key %s: %v", key, err)
		return nil, fmt.Errorf("failed to unmarshal property IDs: %v", err)
	}
	return propertyIDs, nil
}

func AddCacheKeyToPropertySet(ctx context.Context, propertyID, cacheKey string) error {
	start := time.Now()
	setKey := PropertyKeysSetKey(propertyID)
	_, err := RedisClient.SAdd(ctx, setKey, cacheKey).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("sadd").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("sadd").Inc()
		logger.GlobalLogger.Errorf("failed to add cache key %s to set %s: %v", cacheKey, setKey, err)
		return fmt.Errorf("failed to add cache key %s to set %s: %v", cacheKey, setKey, err)
	}
	return nil
}

func GetCacheKeysForProperty(ctx context.Context, propertyID string) ([]string, error) {
	start := time.Now()
	setKey := PropertyKeysSetKey(propertyID)
	cacheKeys, err := RedisClient.SMembers(ctx, setKey).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("smembers").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("smembers").Inc()
		logger.GlobalLogger.Errorf("failed to get cache keys for property %s: %v", propertyID, err)
		return nil, fmt.Errorf("failed to get cache keys for property %s: %v", propertyID, err)
	}
	return cacheKeys, nil
}

func InvalidatePropertyCacheKeys(ctx context.Context, propertyID string) error {
	start := time.Now()
	_, err := invalidatePropertyCacheScript.Run(ctx, RedisClient, []string{}, propertyID).Result()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("invalidate_cache").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("invalidate_cache").Inc()
		logger.GlobalLogger.Errorf("failed to execute invalidate property cache script for property %s: %v", propertyID, err)
		return fmt.Errorf("failed to execute invalidate property cache script: %v", err)
	}
	return nil
}

func PropertyListKey() string {
	return "properties:list"
}

func PropertyListPaginatedKey(offset, limit int) string {
	return fmt.Sprintf("properties:list:offset:%d:limit:%d", offset, limit)
}

func NormalizeAddressComponent(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacements := map[string]string{
		"drive":     "dr",
		"street":    "st",
		"avenue":    "ave",
		"road":      "rd",
		"boulevard": "blvd",
		"lane":      "ln",
		"circle":    "cir",
		"court":     "ct",
		"terrace":   "ter",
		"place":     "pl",
		"highway":   "hwy",
	}
	for full, abbr := range replacements {
		s = strings.ReplaceAll(s, " "+full, " "+abbr)
	}
	return s
}

func PropertySpecificSearchKey(street, city string) string {
	return fmt.Sprintf("properties:search-specific:street:%s:city:%s", street, city)
}

func PropertyKey(id string) string {
	return fmt.Sprintf("property:%s", id)
}

func PropertyKeysSetKey(propertyID string) string {
	return fmt.Sprintf("property:keys:%s", propertyID)
}

func UserKey(id string) string {
	return fmt.Sprintf("user:%s", id)
}
