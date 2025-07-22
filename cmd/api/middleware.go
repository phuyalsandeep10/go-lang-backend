package main

import (
	"time"

	"homeinsight-properties/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// configure all middleware for the router
func (a *App) setupMiddleware() {
	// CORS middleware
	a.Router.Use(setupCORS())

	// Other middleware
	a.Router.Use(middleware.MetricsMiddleware())
	a.Router.Use(middleware.LoggingMiddleware())
	a.Router.Use(middleware.RateLimitMiddleware(a.RateLimiter))
	a.Router.Use(middleware.SecureHeaders())
	a.Router.Use(middleware.ErrorHandler())
	a.Router.Use(gin.Recovery())
}

// configure CORS middleware
func setupCORS() gin.HandlerFunc {
    corsConfig := cors.DefaultConfig()
    corsConfig.AllowAllOrigins = true // Allow all origins in all environments

    corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
    corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With"}
    corsConfig.AllowCredentials = true
    corsConfig.ExposeHeaders = []string{"Content-Length"}
    corsConfig.MaxAge = 12 * time.Hour

    return cors.New(corsConfig)
}
