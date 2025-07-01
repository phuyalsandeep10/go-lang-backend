// Package cache provides Redis caching functionality for the homeinsight-properties application.
package cache

import (
	"fmt"
	"os"
	"strconv"
)

// configuration settings for connecting to a Redis instance.
type RedisConfig struct {
	Host        string `validate:"required,hostname"`
	Port        int    `validate:"required,gt=0,lte=65535"`
	Password    string
	DB          int    `validate:"gte=0"`
	TLSEnabled  bool
	TLSCertFile string
}

// load and validate Redis configuration from environment variables.
func LoadRedisConfig() (*RedisConfig, error) {
	config := &RedisConfig{
		Host:        os.Getenv("REDIS_HOST"),
		Port:        6379, // Default port
		Password:    os.Getenv("REDIS_PASSWORD"),
		DB:          0,    // Default DB
		TLSEnabled:  os.Getenv("REDIS_TLS_ENABLED") == "true",
		TLSCertFile: os.Getenv("REDIS_TLS_CERT_FILE"),
	}

	// Override port if set
	if portStr := os.Getenv("REDIS_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_PORT value: %v", err)
		}
		config.Port = port
	}

	// Override DB if set
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		db, err := strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB value: %v", err)
		}
		config.DB = db
	}

	// Set default host if not provided
	if config.Host == "" {
		config.Host = "localhost"
	}

	// Basic validation
	if config.Host == "" {
		return nil, fmt.Errorf("REDIS_HOST is required")
	}
	if config.Port <= 0 || config.Port > 65535 {
		return nil, fmt.Errorf("REDIS_PORT must be between 1 and 65535")
	}
	if config.DB < 0 {
		return nil, fmt.Errorf("REDIS_DB must be non-negative")
	}
	if config.TLSEnabled && config.TLSCertFile != "" {
		if _, err := os.Stat(config.TLSCertFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("TLS certificate file does not exist: %s", config.TLSCertFile)
		}
	}

	return config, nil
}
