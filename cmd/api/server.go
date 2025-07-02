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
		logger.GlobalLogger.Printf("Starting server on %s", addr)
		logger.GlobalLogger.Printf("Redoc documentation available at: http://localhost%s/redoc", addr)
		logger.GlobalLogger.Printf("Swagger UI available at: http://localhost%s/swagger/index.html", addr)

		if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.GlobalLogger.Errorf("Failed to start server: %v", err)
			os.Exit(1)
		}
	}()

	a.shutdownServer()
}

// shutdown of the server
func (a *App) shutdownServer() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.GlobalLogger.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.Server.Shutdown(ctx); err != nil {
		logger.GlobalLogger.Errorf("Server forced to shutdown: %v", err)
		os.Exit(1)
	}

	logger.GlobalLogger.Println("Server exited")
}
