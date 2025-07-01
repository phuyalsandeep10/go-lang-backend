package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

// initialize the Redis client with the provided configuration.
func InitRedis() error {
	cfg, err := LoadRedisConfig()
	if err != nil {
		return fmt.Errorf("failed to load Redis config: %v", err)
	}

	var tlsConfig *tls.Config
	if cfg.TLSEnabled {
		if cfg.TLSCertFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, "")
			if err != nil {
				logger.GlobalLogger.Errorf("failed to load TLS certificate: %v", err)
				return fmt.Errorf("failed to load TLS certificate: %v", err)
			}
			tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
		} else {
			tlsConfig = &tls.Config{}
		}
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     10,
		MinIdleConns: 5,
		TLSConfig:    tlsConfig,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	_, err = RedisClient.Ping(ctx).Result()
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

// close the Redis client connection.
func CloseRedis() {
	if RedisClient != nil {
		if err := RedisClient.Close(); err != nil {
			logger.GlobalLogger.Errorf("error closing Redis: %v", err)
		} else {
			logger.GlobalLogger.Println("Redis connection closed")
		}
	}
}
