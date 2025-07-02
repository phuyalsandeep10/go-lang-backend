package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		URI    string `yaml:"uri"`
		DBName string `yaml:"dbname"`
	} `yaml:"database"`
	Redis struct {
		Host        string `yaml:"host" validate:"required,hostname"`
		Port        int    `yaml:"port" validate:"required,gt=0,lte=65535"`
		Password    string `yaml:"password"`
		DB          int    `yaml:"db" validate:"gte=0"`
		TLSEnabled  bool   `yaml:"tls_enabled"`
		TLSCertFile string `yaml:"tls_cert_file"`
	} `yaml:"redis"`
	JWT struct {
		Secret string `yaml:"secret"`
	} `yaml:"jwt"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// Override with environment variables if set
	if uri := os.Getenv("MONGO_URI"); uri != "" {
		cfg.Database.URI = uri
	}
	if dbname := os.Getenv("DB_NAME"); dbname != "" {
		cfg.Database.DBName = dbname
	}
	if host := os.Getenv("REDIS_HOST"); host != "" {
		cfg.Redis.Host = host
	}
	if port := os.Getenv("REDIS_PORT"); port != "" {
		portNum, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_PORT value: %v", err)
		}
		cfg.Redis.Port = portNum
	}
	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		cfg.Redis.Password = password
	}
	if db := os.Getenv("REDIS_DB"); db != "" {
		dbNum, err := strconv.Atoi(db)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB value: %v", err)
		}
		cfg.Redis.DB = dbNum
	}
	if tlsEnabled := os.Getenv("REDIS_TLS_ENABLED"); tlsEnabled != "" {
		cfg.Redis.TLSEnabled = tlsEnabled == "true"
	}
	if tlsCertFile := os.Getenv("REDIS_TLS_CERT_FILE"); tlsCertFile != "" {
		cfg.Redis.TLSCertFile = tlsCertFile
	}
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.JWT.Secret = secret
	}

	// Set default values
	if cfg.Redis.Host == "" {
		cfg.Redis.Host = "localhost"
	}
	if cfg.Redis.Port == 0 {
		cfg.Redis.Port = 6379
	}
	if cfg.Redis.DB < 0 {
		cfg.Redis.DB = 0
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
	if cfg.Redis.TLSEnabled && cfg.Redis.TLSCertFile != "" {
		if _, err := os.Stat(cfg.Redis.TLSCertFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("TLS certificate file does not exist: %s", cfg.Redis.TLSCertFile)
		}
	}

	return &cfg, nil
}
