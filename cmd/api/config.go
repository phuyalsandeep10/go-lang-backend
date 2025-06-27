package main

import (
	"log"
	"os"

	"homeinsight-properties/pkg/config"
	"homeinsight-properties/pkg/logger"

	"github.com/joho/godotenv"
)

// load environment variables and configuration
func LoadConfiguration() *config.Config {
	loadEnvironment()
	logger.InitLogger()
	return loadConfigFile()
}

// load environment variables from .env file
func loadEnvironment() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, relying on system environment variables: %v", err)
	}
}

// load the application configuration from a YAML file
func loadConfigFile() *config.Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Logger.Fatalf("Failed to load config: %v", err)
	}

	return cfg
}
