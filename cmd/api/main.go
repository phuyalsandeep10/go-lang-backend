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
	"homeinsight-properties/internal/repositories"
	"homeinsight-properties/internal/services"
	"homeinsight-properties/internal/transformers"
	"homeinsight-properties/internal/validators"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/config"
	"homeinsight-properties/pkg/database"
	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
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

// App represents the application structure
type App struct {
	Config          *config.Config
	Router          *gin.Engine
	PropertyHandler *handlers.PropertyHandler
	UserHandler     *handlers.UserHandler
	RateLimiter     *middleware.RateLimiter
	Server          *http.Server
}

func main() {
	loadEnvironment()
	logger.InitLogger()
	cfg := loadConfiguration()
	app := initializeApp(cfg)
	defer app.cleanup()
	startServer(app)
}

// loadEnvironment loads environment variables from .env file
func loadEnvironment() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, relying on system environment variables: %v", err)
	}
}

// loadConfiguration loads the application configuration
func loadConfiguration() *config.Config {
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

// initializeApp creates and initializes the application
func initializeApp(cfg *config.Config) *App {
	app := &App{Config: cfg}

	// Initialize infrastructure
	app.initializeDatabase()
	app.initializeCache()
	app.initializeMetrics()
	app.initializeRateLimiter()

	// Initialize business logic
	app.initializeDependencies()

	// Initialize web layer
	app.initializeRouter()
	app.initializeServer()

	return app
}

// initializeDatabase initializes the database connection
func (a *App) initializeDatabase() {
	if err := database.InitDB(); err != nil {
		logger.Logger.Fatalf("Failed to initialize database: %v", err)
	}
}

// initializeCache initializes the Redis cache
func (a *App) initializeCache() {
	if err := cache.InitRedis(); err != nil {
		logger.Logger.Fatalf("Failed to initialize Redis: %v", err)
	}
}

// initializeMetrics initializes Prometheus metrics
func (a *App) initializeMetrics() {
	metrics.Init()
}

// initializeRateLimiter initializes the rate limiter
func (a *App) initializeRateLimiter() {
	a.RateLimiter = middleware.NewRateLimiter(rate.Limit(100/60.0), 10)
	go a.RateLimiter.Cleanup()
}

// initializeDependencies initializes all dependencies (repositories, services, handlers)
func (a *App) initializeDependencies() {
	// Initialize repositories
	propertyRepo := repositories.NewPropertyRepository()
	propertyCache := repositories.NewPropertyCache()
	userRepo := repositories.NewUserRepository()

	// Initialize transformers
	addrTrans := transformers.NewAddressTransformer()
	propTrans := transformers.NewPropertyTransformer()

	// Initialize validators
	propertyValidator := validators.NewPropertyValidator()
	userValidator := validators.NewUserValidator()

	// Initialize services
	propertyService := services.NewPropertyService(propertyRepo, propertyCache, propTrans, addrTrans, propertyValidator)
	searchService := services.NewPropertySearchService(propertyRepo, propertyCache, addrTrans, propTrans, propertyValidator)
	userService := services.NewUserService(userRepo, userValidator)

	// Initialize handlers
	a.PropertyHandler = handlers.NewPropertyHandler(propertyService, searchService)
	a.UserHandler = handlers.NewUserHandler(userService)
}

// initializeRouter sets up the Gin router with middleware and routes
func (a *App) initializeRouter() {
	a.Router = gin.New()
	a.setupMiddleware()
	a.setupRoutes()
}

// setupMiddleware configures all middleware
func (a *App) setupMiddleware() {
	// CORS middleware
	a.Router.Use(setupCORS())

	// Other middleware
	a.Router.Use(middleware.MetricsMiddleware())
	a.Router.Use(middleware.LoggingMiddleware())
	a.Router.Use(middleware.RateLimitMiddleware(a.RateLimiter))
	a.Router.Use(middleware.SecureHeaders())
	a.Router.Use(gin.Recovery())
}

// initializeServer creates the HTTP server
func (a *App) initializeServer() {
	addr := fmt.Sprintf(":%d", a.Config.Server.Port)
	a.Server = &http.Server{
		Addr:    addr,
		Handler: a.Router,
	}
}

// startServer starts the HTTP server with graceful shutdown
func startServer(app *App) {
	// Start server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%d", app.Config.Server.Port)
		logger.Logger.Printf("Starting server on %s", addr)
		logger.Logger.Printf("Redoc documentation available at: http://localhost%s/redoc", addr)
		logger.Logger.Printf("Swagger UI available at: http://localhost%s/swagger/index.html", addr)

		if err := app.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	gracefulShutdown(app)
}

// gracefulShutdown handles graceful shutdown of the server
func gracefulShutdown(app *App) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Logger.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Server.Shutdown(ctx); err != nil {
		logger.Logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Logger.Println("Server exited")
}

// cleanup performs cleanup operations
func (a *App) cleanup() {
	database.CloseDB()
	cache.CloseRedis()
}
