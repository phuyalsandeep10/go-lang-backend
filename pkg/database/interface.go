package database

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

// interface for MongoDB operations.
type Database interface {
	GetCollection(name string) *mongo.Collection
	CreatePropertyIndexes(ctx context.Context) error
}

// Database interface using a MongoDB database.
type MongoDatabase struct {
	db *mongo.Database
}

// create a new MongoDatabase instance.
func NewMongoDatabase(db *mongo.Database) *MongoDatabase {
	return &MongoDatabase{db: db}
}

// return a MongoDB collection by name.
func (m *MongoDatabase) GetCollection(name string) *mongo.Collection {
	return m.db.Collection(name)
}

// create indexes for the properties collection.
func (m *MongoDatabase) CreatePropertyIndexes(ctx context.Context) error {
	return CreatePropertyIndexes(m.db)
}
