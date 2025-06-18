package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

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

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
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

	log.Println("Redis connected successfully")
	return nil
}

func CloseRedis() {
	if RedisClient != nil {
		if err := RedisClient.Close(); err != nil {
			log.Printf("Error closing Redis: %v", err)
		} else {
			log.Println("Redis connection closed")
		}
	}
}

func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %v", err)
	}
	if err := RedisClient.Set(ctx, key, data, expiration).Err(); err != nil {
		return err
	}
	return nil
}

func Get(ctx context.Context, key string, dest interface{}) error {
	val, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

func Delete(ctx context.Context, key string) error {
	return RedisClient.Del(ctx, key).Err()
}

func Exists(ctx context.Context, key string) (bool, error) {
	count, err := RedisClient.Exists(ctx, key).Result()
	return count > 0, err
}

func SetSearchResult(ctx context.Context, key string, propertyIDs []string, expiration time.Duration) error {
	propertyIDsJSON, err := json.Marshal(propertyIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal property IDs: %v", err)
	}

	args := []interface{}{key, string(propertyIDsJSON), strconv.Itoa(int(expiration.Seconds()))}
	for _, id := range propertyIDs {
		args = append(args, id)
	}

	_, err = setSearchResultScript.Run(ctx, RedisClient, []string{}, args...).Result()
	if err != nil {
		return fmt.Errorf("failed to execute set search result script: %v", err)
	}
	return nil
}

func GetSearchResult(ctx context.Context, key string) ([]string, error) {
	val, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var propertyIDs []string
	if err := json.Unmarshal([]byte(val), &propertyIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal property IDs: %v", err)
	}
	return propertyIDs, nil
}

func AddCacheKeyToPropertySet(ctx context.Context, propertyID, cacheKey string) error {
	setKey := PropertyKeysSetKey(propertyID)
	_, err := RedisClient.SAdd(ctx, setKey, cacheKey).Result()
	if err != nil {
		return fmt.Errorf("failed to add cache key %s to set %s: %v", cacheKey, setKey, err)
	}
	return nil
}

func GetCacheKeysForProperty(ctx context.Context, propertyID string) ([]string, error) {
	setKey := PropertyKeysSetKey(propertyID)
	cacheKeys, err := RedisClient.SMembers(ctx, setKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache keys for property %s: %v", propertyID, err)
	}
	return cacheKeys, nil
}

func InvalidatePropertyCacheKeys(ctx context.Context, propertyID string) error {
	_, err := invalidatePropertyCacheScript.Run(ctx, RedisClient, []string{}, propertyID).Result()
	if err != nil {
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
