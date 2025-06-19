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
	"homeinsight-properties/internal/services"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/config"
	"homeinsight-properties/pkg/database"
	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
	_ "net/http/pprof"

	// Swagger imports
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "homeinsight-properties/docs"
)

// @title HomeInsight Properties API
// @version 1.0
// @description A comprehensive property management API for real estate data
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8000
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

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

	// Configure CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true // Temporary for debugging; restrict in production
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With"}
	corsConfig.AllowCredentials = true
	corsConfig.ExposeHeaders = []string{"Content-Length"}
	corsConfig.MaxAge = 12 * time.Hour

	// Log requests for debugging
	r.Use(func(c *gin.Context) {
		log.Printf("Handling request: %s %s, Origin: %s", c.Request.Method, c.Request.URL.Path, c.Request.Header.Get("Origin"))
		c.Next()
	})
	r.Use(cors.New(corsConfig))

	// Apply other middleware
	r.Use(middleware.MetricsMiddleware())
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.RateLimitMiddleware(rl))
	r.Use(gin.Recovery())

	// Serve Redoc UI
	r.Static("/redoc", "./static/redoc")
	r.StaticFile("/favicon.png", "./static/redoc/favicon.png")
	r.GET("/redoc", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/redoc/index.html")
	})

	// Serve Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Serve swagger.json
	r.StaticFile("/swagger.json", "./docs/swagger.json")

	// Expose pprof profiling endpoints
	r.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))

	// Expose Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := database.MongoClient.Ping(ctx, nil); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "MongoDB unavailable"})
			return
		}
		if _, err := cache.RedisClient.Ping(ctx).Result(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Redis unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Initialize services
	propertyService := services.NewPropertyService()

	// Initialize handlers
	propertyHandler := handlers.NewPropertyHandler(propertyService)
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
			protected.GET("", propertyHandler.GetProperties)
			protected.GET("/property-search", propertyHandler.SearchProperty)
			protected.GET("/:id", propertyHandler.GetPropertyByID)
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
		log.Printf("Redoc documentation available at: http://localhost%s/redoc", addr)
		log.Printf("Swagger UI available at: http://localhost%s/swagger/index.html", addr)
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
