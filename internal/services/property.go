// internal/services/property_service.go
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"net/url"
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

// Helper function to build pagination URLs
func (s *PropertyService) buildPaginationURL(baseURL string, offset, limit int, params url.Values) string {
	u, _ := url.Parse(baseURL)
	q := u.Query()

	// Add pagination parameters
	q.Set("offset", fmt.Sprintf("%d", offset))
	q.Set("limit", fmt.Sprintf("%d", limit))

	// Add any additional query parameters
	for key, values := range params {
		if key != "offset" && key != "limit" {
			for _, value := range values {
				q.Add(key, value)
			}
		}
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func (s *PropertyService) GetPropertiesWithPagination(ginCtx *gin.Context, offset, limit int) (*models.PaginatedPropertiesResponse, error) {
	ctx := context.Background()

	// Create cache key that includes pagination parameters
	listKey := cache.PropertyListPaginatedKey(offset, limit)

	// Try to get from cache first
	var cachedResponse models.PaginatedPropertiesResponse
	err := cache.Get(ctx, listKey, &cachedResponse)
	if err == nil {
		fmt.Printf("Paginated properties (offset=%d, limit=%d) loaded from cache\n", offset, limit)
		s.setDataSource(ginCtx, "REDIS_CACHE", true)
		return &cachedResponse, nil
	}

	if err != redis.Nil {
		fmt.Printf("Cache error for paginated properties: %v\n", err)
	}

	// Get total count
	var total int64
	countQuery := "SELECT COUNT(*) FROM properties"
	err = database.DB.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %v", err)
	}

	// Get paginated properties
	query := "SELECT id, name, description, price FROM properties ORDER BY name LIMIT ? OFFSET ?"
	rows, err := database.DB.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query properties: %v", err)
	}
	defer rows.Close()

	properties := []models.Property{}
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

	// Build pagination metadata
	meta := models.PaginationMeta{
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}

	// Generate next/prev URLs
	baseURL := "/api/properties"
	var queryParams url.Values
	if ginCtx != nil {
		queryParams = ginCtx.Request.URL.Query()
	}

	// Next page URL
	if int64(offset+limit) < total {
		nextURL := s.buildPaginationURL(baseURL, offset+limit, limit, queryParams)
		meta.Next = &nextURL
	}

	// Previous page URL
	if offset > 0 {
		prevOffset := offset - limit
		if prevOffset < 0 {
			prevOffset = 0
		}
		prevURL := s.buildPaginationURL(baseURL, prevOffset, limit, queryParams)
		meta.Prev = &prevURL
	}

	response := &models.PaginatedPropertiesResponse{
		Data: properties,
		Meta: meta,
	}

	// Cache the results for 10 minutes (shorter than full list due to pagination variety)
	if err := cache.Set(ctx, listKey, response, 10*time.Minute); err != nil {
		fmt.Printf("Failed to cache paginated properties: %v\n", err)
	}

	fmt.Printf("Paginated properties (offset=%d, limit=%d) loaded from database and cached\n", offset, limit)
	s.setDataSource(ginCtx, "DATABASE", false)
	return response, nil
}

// Keep the old method for backward compatibility or internal use
func (s *PropertyService) GetAllProperties(ginCtx *gin.Context) ([]models.Property, error) {
	// This can now call the paginated version with default values
	result, err := s.GetPropertiesWithPagination(ginCtx, 0, 1000) // Large limit for "all"
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// Cache invalidation helper
func (s *PropertyService) invalidatePropertiesCache(ctx context.Context) {
	// Invalidate the basic list cache
	listKey := cache.PropertyListKey()
	if err := cache.Delete(ctx, listKey); err != nil {
		fmt.Printf("Failed to invalidate properties list cache: %v\n", err)
	}

	fmt.Println("Paginated caches will expire naturally due to TTL")
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
		fmt.Printf("Failed to cache property: %v\n", err)
	}

	// Invalidate all list caches
	s.invalidatePropertiesCache(ctx)

	s.setDataSource(ginCtx, "DATABASE_INSERT", false)
	return nil
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

	// Update individual property cache
	propertyKey := cache.PropertyKey(property.ID)
	if err := cache.Set(ctx, propertyKey, property, 1*time.Hour); err != nil {
		fmt.Printf("Failed to update property cache: %v\n", err)
	}

	// Invalidate all list caches
	s.invalidatePropertiesCache(ctx)

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

	// Remove individual property from cache
	propertyKey := cache.PropertyKey(id)
	if err := cache.Delete(ctx, propertyKey); err != nil {
		fmt.Printf("Failed to delete property from cache: %v\n", err)
	}

	// Invalidate all list caches
	s.invalidatePropertiesCache(ctx)

	s.setDataSource(ginCtx, "DATABASE_DELETE", false)
	return nil
}
