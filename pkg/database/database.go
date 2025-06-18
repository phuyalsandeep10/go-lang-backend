package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
)

var MongoClient *mongo.Client
var DB *mongo.Database

type Config struct {
	URI      string
	DBName   string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	uri := os.Getenv("MONGO_URI")
	dbName := os.Getenv("DB_NAME")

	if uri == "" || dbName == "" {
		return nil, fmt.Errorf("missing required environment variables: MONGO_URI or DB_NAME")
	}

	return &Config{
		URI:    uri,
		DBName: dbName,
	}, nil
}

func InitDB() error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.URI).
		SetConnectTimeout(10 * time.Second).
		SetMaxPoolSize(100)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		client.Disconnect(ctx)
		return fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	MongoClient = client
	DB = client.Database(cfg.DBName)

	// Create indexes for search performance
	collection := DB.Collection("properties")
	_, err = collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "propertyId", Value: 1}},
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
	if err != nil {
		log.Printf("Failed to create indexes: %v", err)
	}

	log.Println("MongoDB connected successfully.")
	return nil
}

func CloseDB() {
	if MongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := MongoClient.Disconnect(ctx); err != nil {
			log.Printf("Error closing MongoDB: %v", err)
		} else {
			log.Println("MongoDB connection closed")
		}
	}
}
