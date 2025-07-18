package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"homeinsight-properties/pkg/config"
	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

// Initialize the Redis client with the provided configuration.
func InitRedis(cfg *config.Config) error {
	var tlsConfig *tls.Config
	if cfg.Redis.TLSEnabled {
		tlsConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12, // Required for AWS ElastiCache
			InsecureSkipVerify: true,             // Skip verification for AWS self-signed certificates
		}
	}

	// Default to port 6379 if not specified
	port := cfg.Redis.Port
	if port == 0 {
		port = 6379
	}

	// Configure Redis client options
	options := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, port),
		DB:           cfg.Redis.DB,
		PoolSize:     10,
		MinIdleConns: 5,
		TLSConfig:    tlsConfig,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	// Only set password if non-empty
	if cfg.Redis.Password != "" {
		options.Password = cfg.Redis.Password
	}

	RedisClient = redis.NewClient(options)

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

	logger.GlobalLogger.Println("Redis connected successfully")
	return nil
}

// Close the Redis client connection.
func CloseRedis() {
	if RedisClient != nil {
		if err := RedisClient.Close(); err != nil {
			logger.GlobalLogger.Errorf("error closing Redis: %v", err)
		} else {
			logger.GlobalLogger.Println("Redis connection closed")
		}
	}
}
