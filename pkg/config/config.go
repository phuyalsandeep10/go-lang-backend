package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		URI      string `yaml:"uri"`
		DBName   string `yaml:"dbname"`
	} `yaml:"database"`
	Redis struct {
		Host        string `yaml:"host" validate:"required,hostname"`
		Port        int    `yaml:"port" validate:"required,gt=0,lte=65535"`
		Password    string `yaml:"password"`
		DB          int    `yaml:"db" validate:"gte=0"`
		TLSEnabled  bool   `yaml:"tls_enabled"`
	} `yaml:"redis"`
	JWT struct {
		Secret string `yaml:"secret"`
	} `yaml:"jwt"`
}

func LoadConfig(path string) (*Config, error) {
	// Initialize config with hardcoded values for testing
	cfg := &Config{
		Server: struct {
			Port int `yaml:"port"`
		}{
			Port: 8000, // SERVER_PORT
		},
		Database: struct {
			URI      string `yaml:"uri"`
			DBName   string `yaml:"dbname"`

		}{
			URI:      "mongodb+srv://homeinsightcore:Zj3l6zfaM43K3PpG@cluster0.9kecynk.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0", // MONGO_URI
			DBName:   "homeinsight",                                                                                                                     // DB_NAME
		                                                                                                                  // DB_PASSWORD
		},
		Redis: struct {
			Host        string `yaml:"host" validate:"required,hostname"`
			Port        int    `yaml:"port" validate:"required,gt=0,lte=65535"`
			Password    string `yaml:"password"`
			DB          int    `yaml:"db" validate:"gte=0"`
			TLSEnabled  bool   `yaml:"tls_enabled"`
		}{
			Host:       "clustercfg.homeinsight-core-cache.dxz4rf.use1.cache.amazonaws.com", // REDIS_HOST
			Port:       6379,                                                                // REDIS_PORT
			Password:   "",                                                                  // REDIS_PASSWORD
			DB:         0,                                                                   // REDIS_DB
			TLSEnabled: true,                                                                // REDIS_TLS_ENABLED
		},
		JWT: struct {
			Secret string `yaml:"secret"`
		}{
			Secret: "your_jwt_secret_key", // JWT_SECRET
		},
	}

	// Optionally load from YAML file if provided
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %v", err)
		}
	}

	// Validation
	if cfg.Redis.Host == "" {
		return nil, fmt.Errorf("REDIS_HOST is required")
	}
	if cfg.Redis.Port <= 0 || cfg.Redis.Port > 65535 {
		return nil, fmt.Errorf("REDIS_PORT must be between 1 and 65535")
	}
	if cfg.Redis.DB < 0 {
		return nil, fmt.Errorf("REDIS_DB must be non-negative")
	}

	return cfg, nil
}
