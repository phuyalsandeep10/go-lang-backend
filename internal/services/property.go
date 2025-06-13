// Updated PropertyService methods with context data source tracking
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/database"
)

type PropertyService struct {
}

func NewPropertyService() *PropertyService {
	return &PropertyService{}
}

// Helper function to set data source in gin context
func (s *PropertyService) setDataSource(ginCtx *gin.Context, source string, cacheHit bool) {
	if ginCtx != nil {
		ginCtx.Set("data_source", source)
		ginCtx.Set("cache_hit", cacheHit)
	}
}

func (s *PropertyService) CreateProperty(ginCtx *gin.Context, property *models.Property) error {
	if property.Name == "" {
		return errors.New("property name is required")
	}

	property.ID = uuid.New().String()

	query := "INSERT INTO properties (id, name, description, price) VALUES (?, ?, ?, ?)"
	_, err := database.DB.Exec(query, property.ID, property.Name, property.Description, property.Price)
	if err != nil {
		return fmt.Errorf("failed to insert property: %v", err)
	}

	ctx := context.Background()

	// Cache the new property
	propertyKey := cache.PropertyKey(property.ID)
	if err := cache.Set(ctx, propertyKey, property, 1*time.Hour); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to cache property: %v\n", err)
	}

	// Invalidate properties list cache
	listKey := cache.PropertyListKey()
	if err := cache.Delete(ctx, listKey); err != nil {
		fmt.Printf("Failed to invalidate properties list cache: %v\n", err)
	}

	// Set data source for logging
	s.setDataSource(ginCtx, "DATABASE_INSERT", false)

	return nil
}

func (s *PropertyService) GetAllProperties(ginCtx *gin.Context) ([]models.Property, error) {
	ctx := context.Background()
	listKey := cache.PropertyListKey()

	// Try to get from cache first
	var properties []models.Property
	err := cache.Get(ctx, listKey, &properties)
	if err == nil {
		fmt.Println("Properties loaded from cache")
		s.setDataSource(ginCtx, "REDIS_CACHE", true)
		return properties, nil
	}

	// If not in cache or cache error, get from database
	if err != redis.Nil {
		fmt.Printf("Cache error (proceeding with DB query): %v\n", err)
	}

	rows, err := database.DB.Query("SELECT id, name, description, price FROM properties")
	if err != nil {
		return nil, fmt.Errorf("failed to query properties: %v", err)
	}
	defer rows.Close()

	properties = []models.Property{}
	for rows.Next() {
		var prop models.Property
		var description sql.NullString
		if err := rows.Scan(&prop.ID, &prop.Name, &description, &prop.Price); err != nil {
			return nil, fmt.Errorf("failed to scan property: %v", err)
		}
		prop.Description = description.String
		properties = append(properties, prop)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	// Cache the results for 30 minutes
	if err := cache.Set(ctx, listKey, properties, 30*time.Minute); err != nil {
		fmt.Printf("Failed to cache properties list: %v\n", err)
	}

	fmt.Println("Properties loaded from database and cached")
	s.setDataSource(ginCtx, "DATABASE", false)
	return properties, nil
}

func (s *PropertyService) GetPropertyByID(ginCtx *gin.Context, id string) (*models.Property, error) {
	ctx := context.Background()
	propertyKey := cache.PropertyKey(id)

	// Try cache first
	var property models.Property
	err := cache.Get(ctx, propertyKey, &property)
	if err == nil {
		fmt.Printf("Property %s loaded from cache\n", id)
		s.setDataSource(ginCtx, "REDIS_CACHE", true)
		return &property, nil
	}

	if err != redis.Nil {
		fmt.Printf("Cache error for property %s: %v\n", id, err)
	}

	// Get from database
	query := "SELECT id, name, description, price FROM properties WHERE id = ?"
	row := database.DB.QueryRow(query, id)

	var description sql.NullString
	err = row.Scan(&property.ID, &property.Name, &description, &property.Price)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("property not found")
		}
		return nil, fmt.Errorf("failed to scan property: %v", err)
	}
	property.Description = description.String

	// Cache for 1 hour
	if err := cache.Set(ctx, propertyKey, &property, 1*time.Hour); err != nil {
		fmt.Printf("Failed to cache property %s: %v\n", id, err)
	}

	fmt.Printf("Property %s loaded from database and cached\n", id)
	s.setDataSource(ginCtx, "DATABASE", false)
	return &property, nil
}

func (s *PropertyService) UpdateProperty(ginCtx *gin.Context, property *models.Property) error {
	if property.ID == "" || property.Name == "" {
		return errors.New("property ID and name are required")
	}

	query := "UPDATE properties SET name = ?, description = ?, price = ? WHERE id = ?"
	result, err := database.DB.Exec(query, property.Name, property.Description, property.Price, property.ID)
	if err != nil {
		return fmt.Errorf("failed to update property: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("property not found")
	}

	ctx := context.Background()

	// Update cache
	propertyKey := cache.PropertyKey(property.ID)
	if err := cache.Set(ctx, propertyKey, property, 1*time.Hour); err != nil {
		fmt.Printf("Failed to update property cache: %v\n", err)
	}

	// Invalidate list cache
	listKey := cache.PropertyListKey()
	if err := cache.Delete(ctx, listKey); err != nil {
		fmt.Printf("Failed to invalidate properties list cache: %v\n", err)
	}

	s.setDataSource(ginCtx, "DATABASE_UPDATE", false)
	return nil
}

func (s *PropertyService) DeleteProperty(ginCtx *gin.Context, id string) error {
	query := "DELETE FROM properties WHERE id = ?"
	result, err := database.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete property: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("property not found")
	}

	ctx := context.Background()

	// Remove from cache
	propertyKey := cache.PropertyKey(id)
	if err := cache.Delete(ctx, propertyKey); err != nil {
		fmt.Printf("Failed to delete property from cache: %v\n", err)
	}

	// Invalidate list cache
	listKey := cache.PropertyListKey()
	if err := cache.Delete(ctx, listKey); err != nil {
		fmt.Printf("Failed to invalidate properties list cache: %v\n", err)
	}

	s.setDataSource(ginCtx, "DATABASE_DELETE", false)
	return nil
}
