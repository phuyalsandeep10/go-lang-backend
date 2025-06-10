package main

import (
	"fmt"
	"log"
	"os"

	"homeinsight-properties/internal/handlers"
	"homeinsight-properties/internal/middleware"
	"homeinsight-properties/pkg/database"
	"homeinsight-properties/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file for local development
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, relying on system environment variables: %v", err)
	}

	// Initialize logger
	logger.InitLogger()

	// Initialize database
	if err := database.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDB()

	// Set up Gin router
	r := gin.New()

	// Apply middleware
	r.Use(middleware.LoggingMiddleware())
	r.Use(gin.Recovery())

	// Initialize handlers with database dependency
	propertyHandler := handlers.NewPropertyHandler(database.DB)

	// Define routes
	r.GET("/properties", propertyHandler.ListProperties)
	r.POST("/properties", propertyHandler.CreateProperty)

	// Start server
	addr := ":8000" // Default port, override with environment variable if needed
	if port := os.Getenv("SERVER_PORT"); port != "" {
		addr = fmt.Sprintf(":%s", port)
	}
	log.Printf("Starting server on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
