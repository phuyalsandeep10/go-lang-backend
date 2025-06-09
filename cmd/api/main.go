package main

import (
	"fmt"
	"homeinsight-properties/internal/handlers"
	"homeinsight-properties/internal/middleware"
	"homeinsight-properties/pkg/config"
	"homeinsight-properties/pkg/logger"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err :=config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger.InitLogger()

	// Set up Gin router
	r := gin.New()

	// Apply middleware
	r.Use(middleware.LoggingMiddleware())
	r.Use(gin.Recovery())

	// Initialize handlers
	propertyHandler := handlers.NewPropertyHandler()

	// Define routes
	r.GET("/properties", propertyHandler.ListProperties)
	r.POST("/properties", propertyHandler.CreateProperty)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting server on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
