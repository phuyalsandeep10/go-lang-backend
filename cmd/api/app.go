package main

import (
	"net/http"
	"os"

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
	"golang.org/x/time/rate"
)

// App represents the application structure
type App struct {
	Config          *config.Config
	Router          *gin.Engine
	PropertyHandler *handlers.PropertyHandler
	UserHandler     *handlers.UserHandler
	RateLimiter     *middleware.RateLimiter
	Server          *http.Server
}

// Create and initialize a new App instance
func NewApp(cfg *config.Config) *App {
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

	return app
}

// initialize the database connection
func (a *App) initializeDatabase() {
	if err := database.InitDB(); err != nil {
		logger.GlobalLogger.Errorf("Failed to initialize database: %v", err)
		os.Exit(1)
	}
}

// initialize the Redis cache
func (a *App) initializeCache() {
	if err := cache.InitRedis(); err != nil {
		logger.GlobalLogger.Errorf("Failed to initialize Redis: %v", err)
		os.Exit(1)
	}
}

// initialize Prometheus metrics
func (a *App) initializeMetrics() {
	metrics.Init()
}

// initialize the rate limiter
func (a *App) initializeRateLimiter() {
	a.RateLimiter = middleware.NewRateLimiter(rate.Limit(100/60.0), 10)
	go a.RateLimiter.Cleanup()
}

// initialize all dependencies
func (a *App) initializeDependencies() {
	// repositories
	propertyRepo := repositories.NewPropertyRepository()
	propertyCache := repositories.NewPropertyCache()
	userRepo := repositories.NewUserRepository()

	// transformers
	addrTrans := transformers.NewAddressTransformer()
	propTrans := transformers.NewPropertyTransformer()

	// validators
	propertyValidator := validators.NewPropertyValidator()
	userValidator := validators.NewUserValidator()

	// services
	propertyService := services.NewPropertyService(propertyRepo, propertyCache, propTrans, addrTrans, propertyValidator)
	searchService := services.NewPropertySearchService(propertyRepo, propertyCache, addrTrans, propTrans, propertyValidator)
	userService := services.NewUserService(userRepo, userValidator)

	// handlers
	a.PropertyHandler = handlers.NewPropertyHandler(propertyService, searchService)
	a.UserHandler = handlers.NewUserHandler(userService)
}

// set up the Gin router with middleware and routes
func (a *App) initializeRouter() {
	a.Router = gin.New()
	a.setupMiddleware()
	a.setupRoutes()
}

// cleanup operations
func (a *App) cleanup() {
	database.CloseDB()
	cache.CloseRedis()
}
