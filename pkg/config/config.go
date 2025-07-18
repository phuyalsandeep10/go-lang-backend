package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int `yaml:"port" validate:"required,gt=0,lte=65535"`
	} `yaml:"server"`
	Database struct {
		URI    string `yaml:"uri"`
		DBName string `yaml:"dbname" validate:"required"`
	} `yaml:"database"`
	Redis struct {
		Host       string `yaml:"host" validate:"required,hostname"`
		Port       int    `yaml:"port" validate:"required,gt=0,lte=65535"`
		Password   string `yaml:"password"`
		DB         int    `yaml:"db" validate:"gte=0"`
		TLSEnabled bool   `yaml:"tls_enabled"`
	} `yaml:"redis"`
	JWT struct {
		Secret string `yaml:"secret"`
	} `yaml:"jwt"`
	CoreLogic struct {
		ClientKey      string `yaml:"client_key"`
		ClientSecret   string `yaml:"client_secret"`
		DeveloperEmail string `yaml:"developer_email"`
	} `yaml:"corelogic"`
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{}

	// Load from YAML file if provided
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %v", err)
		}
	}

	// Override with environment variables for sensitive fields
	if mongoURI := os.Getenv("MONGO_URI"); mongoURI != "" {
		cfg.Database.URI = mongoURI
	}
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		cfg.Redis.Host = redisHost
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.JWT.Secret = jwtSecret
	}
	if corelogicUsername := os.Getenv("CORELOGIC_USERNAME"); corelogicUsername != "" {
		cfg.CoreLogic.ClientKey = corelogicUsername
	}
	if corelogicPassword := os.Getenv("CORELOGIC_PASSWORD"); corelogicPassword != "" {
		cfg.CoreLogic.ClientSecret = corelogicPassword
	}
	if corelogicDeveloperEmail := os.Getenv("CORELOGIC_DEVELOPER_EMAIL"); corelogicDeveloperEmail != "" {
		cfg.CoreLogic.DeveloperEmail = corelogicDeveloperEmail
	}

	// Set tls_enabled based on ENV
	if env := os.Getenv("ENV"); env == "production" {
		cfg.Redis.TLSEnabled = true
	} else {
		cfg.Redis.TLSEnabled = false
	}

	// Validation
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return nil, fmt.Errorf("SERVER_PORT must be between 1 and 65535")
	}
	if cfg.Database.URI == "" {
		return nil, fmt.Errorf("MONGO_URI is required")
	}
	if cfg.Database.DBName == "" {
		return nil, fmt.Errorf("DB_NAME is required")
	}
	if cfg.Redis.Host == "" {
		return nil, fmt.Errorf("REDIS_HOST is required")
	}
	if cfg.Redis.Port <= 0 || cfg.Redis.Port > 65535 {
		return nil, fmt.Errorf("REDIS_PORT must be between 1 and 65535")
	}
	if cfg.Redis.DB < 0 {
		return nil, fmt.Errorf("REDIS_DB must be non-negative")
	}
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.CoreLogic.ClientKey == "" {
		return nil, fmt.Errorf("CORELOGIC_USERNAME is required")
	}
	if cfg.CoreLogic.ClientSecret == "" {
		return nil, fmt.Errorf("CORELOGIC_PASSWORD is required")
	}
	if cfg.CoreLogic.DeveloperEmail == "" {
		return nil, fmt.Errorf("CORELOGIC_DEVELOPER_EMAIL is required")
	}

	return cfg, nil
}
