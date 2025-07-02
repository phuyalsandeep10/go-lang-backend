package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"homeinsight-properties/internal/middleware"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/database"
	"homeinsight-properties/pkg/logger"

	_ "homeinsight-properties/docs"
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// configure all API routes
func (a *App) setupRoutes() {
	a.setupStaticRoutes()
	a.setupHealthCheck()
	a.setupAPIRoutes()
}

// static routes and documentation endpoints
func (a *App) setupStaticRoutes() {
	// Serve Redoc UI
	a.Router.Static("/redoc", "./static/redoc")
	a.Router.StaticFile("/favicon.ico", "./static/redoc/favicon.ico")
	a.Router.GET("/redoc", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/redoc/index.html")
	})

	// Serve Swagger UI
	a.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Serve swagger.json
	a.Router.StaticFile("/swagger.json", "./docs/swagger.json")

	// Expose pprof profiling endpoints (disable in production)
	if os.Getenv("ENV") != "production" {
		a.Router.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))
	}

	// Expose Prometheus metrics endpoint
	a.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

// health check endpoint
func (a *App) setupHealthCheck() {
	a.Router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := database.MongoClient.Ping(ctx, nil); err != nil {
			logger.GlobalLogger.Errorf("MongoDB ping failed: %v", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "MongoDB unavailable"})
			return
		}

		if _, err := cache.RedisClient.Ping(ctx).Result(); err != nil {
			logger.GlobalLogger.Errorf("Redis ping failed: %v", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Redis unavailable"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// API routes for user and property operations
func (a *App) setupAPIRoutes() {
	api := a.Router.Group("/api")
	{
		// Public routes
		api.POST("/register", a.UserHandler.Register)
		api.POST("/login", a.UserHandler.Login)

		// Protected routes
		protected := api.Group("/properties")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("", a.PropertyHandler.GetProperties)
			protected.GET("/property-search", a.PropertyHandler.SearchProperty)
			protected.GET("/:id", a.PropertyHandler.GetPropertyByID)
			protected.POST("", a.PropertyHandler.CreateProperty)
			protected.PUT("", a.PropertyHandler.UpdateProperty)
			protected.DELETE("/:id", a.PropertyHandler.DeleteProperty)
		}
	}
}
