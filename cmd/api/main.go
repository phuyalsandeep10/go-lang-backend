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
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/config"
	"homeinsight-properties/pkg/database"
	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
	_ "net/http/pprof"
)

func main() {
	// Load .env file
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

	// Initialize Redis cache
	if err := cache.InitRedis(); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer cache.CloseRedis()

	// Initialize rate limiter
	rl := middleware.NewRateLimiter(rate.Limit(100/60.0), 10)
	go rl.Cleanup()

	// Initialize Prometheus metrics
	metrics.Init()

	// Set up Gin router
	r := gin.New()

	// Apply middleware
	r.Use(middleware.MetricsMiddleware()) // Add metrics middleware
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.RateLimitMiddleware(rl))
	r.Use(gin.Recovery())

	// Expose pprof profiling endpoints
	r.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))

	// Expose Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		// Check database connectivity
		if err := database.DB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Database unavailable"})
			return
		}
		// Check Redis connectivity
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := cache.RedisClient.Ping(ctx).Result(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Redis unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Initialize handlers
	propertyHandler := handlers.NewPropertyHandler(database.DB)
	userHandler := handlers.NewUserHandler(database.DB)

	// Define routes
	api := r.Group("/api")
	{
		// Public routes
		api.POST("/register", userHandler.Register)
		api.POST("/login", userHandler.Login)

		// Protected routes
		protected := api.Group("/properties")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("", propertyHandler.ListProperties)
			protected.GET("/:id", propertyHandler.GetProperty)
			protected.POST("", propertyHandler.CreateProperty)
			protected.PUT("/:id", propertyHandler.UpdateProperty)
			protected.DELETE("/:id", propertyHandler.DeleteProperty)
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
