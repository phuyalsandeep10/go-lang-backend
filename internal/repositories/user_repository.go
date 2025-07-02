package repositories

import (
	"context"
	"time"
	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/database"
	"homeinsight-properties/pkg/metrics"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type userRepository struct {
	db *mongo.Database
}

func NewUserRepository() UserRepository {
	return &userRepository{
		db: database.DB,
	}
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	collection := r.db.Collection("users")
	start := time.Now()
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("find_one", "users").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("find_one", "users").Inc()
		if err == mongo.ErrNoDocuments {
			return nil, err
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	collection := r.db.Collection("users")
	start := time.Now()
	_, err := collection.InsertOne(ctx, user)
	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("insert", "users").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("insert", "users").Inc()
		return err
	}
	return nil
}
