package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"homeinsight-properties/internal/handlers"
	"homeinsight-properties/internal/middleware"
	"homeinsight-properties/pkg/config"
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

	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

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

	// Initialize handlers
	propertyHandler := handlers.NewPropertyHandler(database.DB)
	userHandler := handlers.NewUserHandler(database.DB)

	// Define routes
	api := r.Group("/api")
	{
		// Public routes
		log.Println("Registering route: POST /api/register")
		api.POST("/register", userHandler.Register)
		log.Println("Registering route: POST /api/login")
		api.POST("/login", userHandler.Login)

		// Protected routes
		protected := api.Group("/properties")
		protected.Use(middleware.AuthMiddleware())
		{
			log.Println("Registering route: GET /api/properties")
			protected.GET("", propertyHandler.ListProperties)
			log.Println("Registering route: POST /api/properties")
			protected.POST("", propertyHandler.CreateProperty)
		}
	}

	// Set up server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}
