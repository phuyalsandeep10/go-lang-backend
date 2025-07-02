package database

import (
	"context"
	"time"

	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// create indexes for the properties collection to optimize search performance.
func CreatePropertyIndexes(db *mongo.Database) error {
	collection := db.Collection("properties")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	_, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "propertyId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "address.streetAddress", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "address.city", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "address.state", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "address.zipCode", Value: 1}},
		},
	})
	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("create_indexes", "properties").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("create_indexes", "properties").Inc()
		logger.GlobalLogger.Errorf("Failed to create indexes: %v", err)
		return err
	}

	logger.GlobalLogger.Println("MongoDB indexes created successfully.")
	return nil
}
