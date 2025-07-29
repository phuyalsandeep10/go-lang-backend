package main

import (
	"context"
	"net/http"
	"os"
	"strconv"

	"homeinsight-properties/internal/handlers"
	"homeinsight-properties/internal/middleware"
	"homeinsight-properties/internal/repositories"
	"homeinsight-properties/internal/services"
	"homeinsight-properties/internal/transformers"
	"homeinsight-properties/internal/validators"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/config"
	"homeinsight-properties/pkg/corelogic"
	"homeinsight-properties/pkg/database"
	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

type App struct {
	Config          *config.Config
	Router          *gin.Engine
	PropertyHandler *handlers.PropertyHandler
	UserHandler     *handlers.UserHandler
	RateLimiter     *middleware.RateLimiter
	Server          *http.Server
	RedisClient     *redis.Client
}

// create and initialize a new App instance
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

// database connection
func (a *App) initializeDatabase() {
	if err := database.InitDB(a.Config); err != nil {
		logger.GlobalLogger.Errorf("Failed to initialize database: %v", err)
		os.Exit(1)
	}
	if err := database.CreatePropertyIndexes(database.DB); err != nil {
		logger.GlobalLogger.Errorf("Failed to create database indexes: %v", err)
		os.Exit(1)
	}
}

// Redis cache
func (a *App) initializeCache() {
	addr := a.Config.Redis.Host + ":" + strconv.Itoa(a.Config.Redis.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.GlobalLogger.Errorf("Failed to initialize Redis: %v", err)
		os.Exit(1)
	}

	a.RedisClient = rdb
}

// Prometheus metrics
func (a *App) initializeMetrics() {
	metrics.Init()
}

// rate limiter
func (a *App) initializeRateLimiter() {
	a.RateLimiter = middleware.NewRateLimiter(rate.Limit(100/60.0), 10)
	go a.RateLimiter.Cleanup()
}

// set up all dependencies
func (a *App) initializeDependencies() {
	// Repositories
	propertyRepo := repositories.NewPropertyRepository()
	propertyCache := repositories.NewPropertyCache()
	userRepo := repositories.NewUserRepository()

	// Transformers
	addrTrans := transformers.NewAddressTransformer()
	propTrans := transformers.NewPropertyTransformer()

	// Validators
	propertyValidator := validators.NewPropertyValidator()
	userValidator := validators.NewUserValidator()

	// CoreLogic client
	corelogicClient := corelogic.NewClient(
		a.Config.CoreLogic.ClientKey,
		a.Config.CoreLogic.ClientSecret,
		a.Config.CoreLogic.DeveloperEmail,
	)

	// Services
	propertyService := services.NewPropertyService(propertyRepo, propertyCache, propTrans, addrTrans, propertyValidator, corelogicClient, a.Config)
	searchService := services.NewPropertySearchService(propertyRepo, propertyCache, addrTrans, propTrans, propertyValidator, corelogicClient, a.Config)
	userService := services.NewUserService(userRepo, userValidator)

	// Handlers
	a.PropertyHandler = handlers.NewPropertyHandler(propertyService, searchService)
	a.UserHandler = handlers.NewUserHandler(userService)
}

// Gin router with middleware and routes
// func (a *App) initializeRouter() {
// 	a.Router = gin.New()
// 	a.setupMiddleware()
// 	a.setupRoutes()
// }
func (a *App) initializeRouter() {
	a.Router = gin.New()
	a.setupMiddleware()
	a.setupRoutes()

	// Add this to handle "/"
	a.Router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Welcome to HomeInsight API"})
	})
}


// cleanup operations
func (a *App) cleanup() {
	database.CloseDB()
	cache.CloseRedis()
}
