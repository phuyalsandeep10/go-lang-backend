package repositories

import (
	"context"
	"fmt"
	"time"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/database"
	"homeinsight-properties/pkg/metrics"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type propertyRepository struct {
	collection *mongo.Collection
}

func NewPropertyRepository() PropertyRepository {
	return &propertyRepository{
		collection: database.DB.Collection("properties"),
	}
}

func (r *propertyRepository) FindByID(ctx context.Context, id string) (*models.Property, error) {
	start := time.Now()
	var property models.Property
	err := r.collection.FindOne(ctx, bson.M{"propertyId": id}).Decode(&property)
	metrics.MongoOperationDuration.WithLabelValues("find_one", "properties").Observe(time.Since(start).Seconds())
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found
		}
		metrics.MongoErrorsTotal.WithLabelValues("find_one", "properties").Inc()
		return nil, err
	}
	return &property, nil
}

func (r *propertyRepository) FindByAddress(ctx context.Context, street, city, state, zip string) (*models.Property, error) {
	filter := bson.M{
		"address.streetAddress": street,
		"address.city":         city,
	}
	if state != "" {
		filter["address.state"] = state
	}
	if zip != "" {
		filter["address.zipCode"] = zip
	}
	start := time.Now()
	var property models.Property
	err := r.collection.FindOne(ctx, filter).Decode(&property)
	metrics.MongoOperationDuration.WithLabelValues("find_one", "properties").Observe(time.Since(start).Seconds())
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found
		}
		metrics.MongoErrorsTotal.WithLabelValues("find_one", "properties").Inc()
		return nil, err
	}
	return &property, nil
}

func (r *propertyRepository) FindWithPagination(ctx context.Context, offset, limit int) ([]models.Property, int64, error) {
	start := time.Now()
	total, err := r.collection.CountDocuments(ctx, bson.M{})
	metrics.MongoOperationDuration.WithLabelValues("count_documents", "properties").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("count_documents", "properties").Inc()
		return nil, 0, err
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "address.streetAddress", Value: 1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))

	start = time.Now()
	cursor, err := r.collection.Find(ctx, bson.M{}, findOptions)
	metrics.MongoOperationDuration.WithLabelValues("find", "properties").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("find", "properties").Inc()
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var properties []models.Property
	start = time.Now()
	err = cursor.All(ctx, &properties)
	metrics.MongoOperationDuration.WithLabelValues("cursor_all", "properties").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("cursor_all", "properties").Inc()
		return nil,

0, err
	}
	return properties, total, nil
}

func (r *propertyRepository) Create(ctx context.Context, property *models.Property) error {
	property.ID = primitive.NewObjectID()
	start := time.Now()
	_, err := r.collection.InsertOne(ctx, property)
	metrics.MongoOperationDuration.WithLabelValues("insert", "properties").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("insert", "properties").Inc()
		return err
	}
	return nil
}

func (r *propertyRepository) Update(ctx context.Context, property *models.Property) error {
	update := bson.M{
		"$set": bson.M{
			"avmPropertyId":    property.AVMPropertyID,
			"address":          property.Address,
			"location":         property.Location,
			"lot":              property.Lot,
			"landUseAndZoning": property.LandUseAndZoning,
			"utilities":        property.Utilities,
			"building":         property.Building,
			"ownership":        property.Ownership,
			"taxAssessment":    property.TaxAssessment,
			"lastMarketSale":   property.LastMarketSale,
		},
	}
	start := time.Now()
	result, err := r.collection.UpdateOne(ctx, bson.M{"propertyId": property.PropertyID}, update)
	metrics.MongoOperationDuration.WithLabelValues("update_one", "properties").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("update_one", "properties").Inc()
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("property not found")
	}
	return nil
}

func (r *propertyRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	result, err := r.collection.DeleteOne(ctx, bson.M{"propertyId": id})
	metrics.MongoOperationDuration.WithLabelValues("delete_one", "properties").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("delete_one", "properties").Inc()
		return err
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("property not found")
	}
	return nil
}

func (r *propertyRepository) FindAll(ctx context.Context) ([]models.Property, error) {
	start := time.Now()
	cursor, err := r.collection.Find(ctx, bson.M{})
	metrics.MongoOperationDuration.WithLabelValues("find", "properties").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("find", "properties").Inc()
		return nil, err
	}
	defer cursor.Close(ctx)

	var properties []models.Property
	start = time.Now()
	err = cursor.All(ctx, &properties)
	metrics.MongoOperationDuration.WithLabelValues("cursor_all", "properties").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("cursor_all", "properties").Inc()
		return nil, err
	}
	return properties, nil
}
