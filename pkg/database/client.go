package database

import (
	"context"
	"fmt"
	"time"

	"homeinsight-properties/pkg/config"
	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client
var DB *mongo.Database

// initialize the MongoDB client and database connection.
func InitDB(cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.Database.URI).
		SetConnectTimeout(10 * time.Second).
		SetMaxPoolSize(100)

	start := time.Now()
	client, err := mongo.Connect(ctx, clientOptions)
	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("connect", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("connect", "").Inc()
		logger.GlobalLogger.Errorf("failed to connect to MongoDB: %v", err)
		return fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(ctx, nil)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("ping", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("ping", "").Inc()
		client.Disconnect(ctx)
		logger.GlobalLogger.Errorf("failed to ping MongoDB: %v", err)
		return fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	MongoClient = client
	DB = client.Database(cfg.Database.DBName)

	logger.GlobalLogger.Println("MongoDB connected successfully.")
	return nil
}

// close the MongoDB client connection.
func CloseDB() {
	if MongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		start := time.Now()
		err := MongoClient.Disconnect(ctx)
		duration := time.Since(start).Seconds()
		metrics.MongoOperationDuration.WithLabelValues("disconnect", "").Observe(duration)
		if err != nil {
			metrics.MongoErrorsTotal.WithLabelValues("disconnect", "").Inc()
			logger.GlobalLogger.Errorf("Error closing MongoDB: %v", err)
		} else {
			logger.GlobalLogger.Println("MongoDB connection closed")
		}
	}
}
