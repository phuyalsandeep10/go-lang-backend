package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// interface for Redis client operations.
type CacheClient interface {
	Ping(ctx context.Context) *redis.StatusCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	ScriptRun(ctx context.Context, script *redis.Script, keys []string, args ...interface{}) *redis.Cmd
	Close() error
}

// interface for basic cache operations.
type CacheOperations interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// interface for search-specific cache operations.
type SearchOperations interface {
	SetSearchResult(ctx context.Context, key string, propertyIDs []string, expiration time.Duration) error
	GetSearchResult(ctx context.Context, key string) ([]string, error)
}

// interface for property-specific cache operations.
type PropertyOperations interface {
	AddCacheKeyToPropertySet(ctx context.Context, propertyID, cacheKey string) error
	GetCacheKeysForProperty(ctx context.Context, propertyID string) ([]string, error)
	InvalidatePropertyCacheKeys(ctx context.Context, propertyID string) error
}
