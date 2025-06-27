package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"homeinsight-properties/pkg/logger"
)

// create the HTTP server
func (a *App) InitializeServer() {
	addr := fmt.Sprintf(":%d", a.Config.Server.Port)
	a.Server = &http.Server{
		Addr:    addr,
		Handler: a.Router,
	}
}

// start the HTTP server with graceful shutdown
func (a *App) StartServer() {
	go func() {
		addr := fmt.Sprintf(":%d", a.Config.Server.Port)
		logger.Logger.Printf("Starting server on %s", addr)
		logger.Logger.Printf("Redoc documentation available at: http://localhost%s/redoc", addr)
		logger.Logger.Printf("Swagger UI available at: http://localhost%s/swagger/index.html", addr)

		if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	a.shutdownServer()
}

// shutdown of the server
func (a *App) shutdownServer() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Logger.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.Server.Shutdown(ctx); err != nil {
		logger.Logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Logger.Println("Server exited")
}
