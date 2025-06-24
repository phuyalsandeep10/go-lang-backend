package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/database"
	"homeinsight-properties/pkg/logger"
	"homeinsight-properties/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const Month = 30 * 24 * time.Hour

type PropertyService struct{}

func NewPropertyService() *PropertyService {
	return &PropertyService{}
}

func (s *PropertyService) setDataSource(ginCtx *gin.Context, source string, cacheHit bool) {
	if ginCtx != nil {
		ginCtx.Set("data_source", source)
		ginCtx.Set("cache_hit", cacheHit)
	}
}

func (s *PropertyService) buildPaginationURL(baseURL string, offset, limit int, params url.Values) string {
	u, _ := url.Parse(baseURL)
	q := url.Values{}

	// Set offset and limit as query parameters
	q.Set("offset", fmt.Sprintf("%d", offset))
	q.Set("limit", fmt.Sprintf("%d", limit))

	// Copy other query parameters and ensure they are URL-encoded
	for key, values := range params {
		if key != "offset" && key != "limit" {
			for _, value := range values {
				q.Add(key, value) // Values are automatically encoded by url.Values
			}
		}
	}

	// Encode the query string
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *PropertyService) readMockData(_ string) (map[string]interface{}, error) {
	start := time.Now()
	filePath, err := filepath.Abs("data/coreLogic/property-detail.json")
	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("read_mock_file_path", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("read_mock_file_path", "").Inc()
		return nil, fmt.Errorf("failed to resolve mock data file path: %v", err)
	}

	start = time.Now()
	file, err := os.ReadFile(filePath)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("read_mock_file", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("read_mock_file", "").Inc()
		return nil, fmt.Errorf("failed to read mock data file %s: %v", filePath, err)
	}

	var data map[string]interface{}
	start = time.Now()
	err = json.Unmarshal(file, &data)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("unmarshal_mock_data", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("unmarshal_mock_data", "").Inc()
		return nil, fmt.Errorf("failed to parse mock data from %s: %v", filePath, err)
	}

	return data, nil
}

func (s *PropertyService) normalizeAddressComponent(input string) string {
	return strings.ToUpper(strings.TrimSpace(input))
}

func (s *PropertyService) parseAddress(search string) (streetAddress, city, state, zipCode string) {
	search = s.normalizeAddressComponent(search)
	if search == "" {
		return "", "", "", ""
	}

	// Primary regex for full address: street, city, state, zip
	re := regexp.MustCompile(`^(.*?),\s*([^,]+),\s*([A-Z]{2})\s*(\d{5})$`)
	matches := re.FindStringSubmatch(search)
	if len(matches) == 5 {
		return s.normalizeAddressComponent(matches[1]), s.normalizeAddressComponent(matches[2]),
			s.normalizeAddressComponent(matches[3]), s.normalizeAddressComponent(matches[4])
	}

	// Fallback for street, city
	parts := strings.Split(search, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	if len(parts) >= 2 {
		street := s.normalizeAddressComponent(parts[0])
		city := s.normalizeAddressComponent(parts[1])
		var state, zip string
		if len(parts) > 2 {
			stateZip := strings.Fields(parts[2])
			if len(stateZip) >= 2 {
				state = s.normalizeAddressComponent(stateZip[0])
				zip = s.normalizeAddressComponent(stateZip[1])
			} else if len(stateZip) == 1 {
				if regexp.MustCompile(`^[A-Z]{2}$`).MatchString(stateZip[0]) {
					state = s.normalizeAddressComponent(stateZip[0])
				} else if regexp.MustCompile(`^\d{5}$`).MatchString(stateZip[0]) {
					zip = s.normalizeAddressComponent(stateZip[0])
				}
			}
		}
		return street, city, state, zip
	}

	return s.normalizeAddressComponent(search), "", "", ""
}

func (s *PropertyService) SearchSpecificProperty(ginCtx *gin.Context, req *models.SearchRequest) (*models.Property, error) {
	ctx := context.Background()

	// Rely on handler for query validation; use req.Search directly
	req.Search = s.normalizeAddressComponent(req.Search)
	street, city, state, zip := s.parseAddress(req.Search)
	if street == "" || city == "" {
		return nil, fmt.Errorf("street address and city are required")
	}

	cacheKey := cache.PropertySpecificSearchKey(street, city)

	var propertyID string
	var property models.Property
	start := time.Now()
	err := cache.Get(ctx, cacheKey, &propertyID)
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("get_search_cache").Observe(duration)
	if err == nil {
		start = time.Now()
		err = cache.Get(ctx, cache.PropertyKey(propertyID), &property)
		duration = time.Since(start).Seconds()
		metrics.RedisOperationDuration.WithLabelValues("get_property_cache").Observe(duration)
		if err == nil {
			s.setDataSource(ginCtx, "REDIS_CACHE", true)
			return &property, nil
		}
	}
	if err != redis.Nil {
		metrics.RedisErrorsTotal.WithLabelValues("get_search_cache", "").Inc()
	}

	collection := database.DB.Collection("properties")
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

	start = time.Now()
	err = collection.FindOne(ctx, filter).Decode(&property)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("find_one", "properties").Observe(duration)
	if err == nil {
		propertyKey := cache.PropertyKey(property.PropertyID)
		start = time.Now()
		err = cache.Set(ctx, propertyKey, property, Month)
		duration = time.Since(start).Seconds()
		metrics.RedisOperationDuration.WithLabelValues("set_property").Observe(duration)
		if err != nil {
			metrics.RedisErrorsTotal.WithLabelValues("set_property", "").Inc()
		}
		start = time.Now()
		err = cache.Set(ctx, cacheKey, property.PropertyID, Month)
		duration = time.Since(start).Seconds()
		metrics.RedisOperationDuration.WithLabelValues("set_search_key").Observe(duration)
		if err != nil {
			metrics.RedisErrorsTotal.WithLabelValues("set_search_key", "").Inc()
		}
		start = time.Now()
		err = cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, cacheKey)
		duration = time.Since(start).Seconds()
		metrics.RedisOperationDuration.WithLabelValues("add_cache_key").Observe(duration)
		if err != nil {
			metrics.RedisErrorsTotal.WithLabelValues("add_cache_key", "").Inc()
		}
		start = time.Now()
		_, err = cache.RedisClient.Expire(ctx, cache.PropertyKeysSetKey(property.PropertyID), Month).Result()
		duration = time.Since(start).Seconds()
		metrics.RedisOperationDuration.WithLabelValues("expire").Observe(duration)
		if err != nil {
			metrics.RedisErrorsTotal.WithLabelValues("expire", "").Inc()
		}
		s.setDataSource(ginCtx, "MONGODB", false)
		return &property, nil
	}

	if err != mongo.ErrNoDocuments {
		metrics.MongoErrorsTotal.WithLabelValues("find_one", "properties").Inc()
		return nil, fmt.Errorf("failed to query property: %v", err)
	}

	logger.Logger.Printf("Property not found in MongoDB, attempting to load mock data")
	mockData, err := s.readMockData("default")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mock data: %v", err)
	}

	start = time.Now()
	propertyPtr, err := s.TransformAPIResponse(mockData)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("transform_mock_data", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("transform_mock_data", "").Inc()
		return nil, fmt.Errorf("failed to transform mock data: %v", err)
	}
	property = *propertyPtr

	// Override address fields to match search query
	property.Address.StreetAddress = street
	property.Address.City = city
	if state != "" {
		property.Address.State = state
	}
	if zip != "" {
		property.Address.ZipCode = zip
	}
	property.ID = primitive.NewObjectID()

	logger.Logger.Printf("Inserting mock property into MongoDB: %s", property.PropertyID)
	start = time.Now()
	_, err = collection.InsertOne(ctx, property)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("insert", "properties").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("insert", "properties").Inc()
		return nil, fmt.Errorf("failed to insert mock property: %v", err)
	}

	propertyKey := cache.PropertyKey(property.PropertyID)
	start = time.Now()
	err = cache.Set(ctx, propertyKey, property, Month)
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("set_property").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set_property", "").Inc()
	}
	start = time.Now()
	err = cache.Set(ctx, cacheKey, property.PropertyID, Month)
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("set_search_key").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set_search_key", "").Inc()
	}
	start = time.Now()
	err = cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, cacheKey)
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("add_cache_key").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("add_cache_key", "").Inc()
	}
	start = time.Now()
	_, err = cache.RedisClient.Expire(ctx, cache.PropertyKeysSetKey(property.PropertyID), Month).Result()
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("expire").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("expire", "").Inc()
	}

	s.setDataSource(ginCtx, "MOCK_DATA", false)
	return &property, nil
}

func (s *PropertyService) GetPropertiesWithPagination(ginCtx *gin.Context, offset, limit int) (*models.PaginatedPropertiesResponse, error) {
	ctx := context.Background()

	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	cacheKey := cache.PropertyListPaginatedKey(offset, limit)
	var response models.PaginatedPropertiesResponse
	start := time.Now()
	err := cache.Get(ctx, cacheKey, &response)
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("get_paginated_list").Observe(duration)
	if err == nil {
		s.setDataSource(ginCtx, "REDIS_CACHE", true)
		return &response, nil
	}
	if err != redis.Nil {
		metrics.RedisErrorsTotal.WithLabelValues("get_paginated_list", "").Inc()
	}

	collection := database.DB.Collection("properties")
	start = time.Now()
	total, err := collection.CountDocuments(ctx, bson.M{})
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("count_documents", "properties").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("count_documents", "properties").Inc()
		return nil, fmt.Errorf("failed to get total count")
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "address.streetAddress", Value: 1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))

	start = time.Now()
	cursor, err := collection.Find(ctx, bson.M{}, findOptions)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("find", "properties").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("find", "properties").Inc()
		return nil, fmt.Errorf("failed to query properties")
	}
	defer cursor.Close(ctx)

	properties := []models.Property{}
	start = time.Now()
	err = cursor.All(ctx, &properties)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("cursor_all", "properties").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("cursor_all", "properties").Inc()
		return nil, fmt.Errorf("failed to decode properties")
	}

	for _, property := range properties {
		propertyKey := cache.PropertyKey(property.PropertyID)
		start = time.Now()
		err = cache.Set(ctx, propertyKey, property, Month)
		duration = time.Since(start).Seconds()
		metrics.RedisOperationDuration.WithLabelValues("set_property").Observe(duration)
		if err != nil {
			metrics.RedisErrorsTotal.WithLabelValues("set_property", "").Inc()
		}
	}

	metadata := models.PaginationMeta{
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}
	if int64(offset+limit) < total {
		nextURL := s.buildPaginationURL("/api/properties", offset+limit, limit, ginCtx.Request.URL.Query())
		metadata.Next = &nextURL
	}
	if offset > 0 {
		prevOffset := offset - limit
		if prevOffset < 0 {
			prevOffset = 0
		}
		prevURL := s.buildPaginationURL("/api/properties", prevOffset, limit, ginCtx.Request.URL.Query())
		metadata.Prev = &prevURL
	}

	response = models.PaginatedPropertiesResponse{
		Data:     make([]models.PropertyResponse, len(properties)),
		Metadata: metadata,
	}
	for i, prop := range properties {
		response.Data[i] = models.PropertyResponse{Property: &prop}
	}

	start = time.Now()
	err = cache.Set(ctx, cacheKey, response, Month)
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("set_paginated_list").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set_paginated_list", "").Inc()
	}

	s.setDataSource(ginCtx, "MONGODB", false)
	return &response, nil
}

func (s *PropertyService) GetAllProperties(ginCtx *gin.Context) ([]models.Property, error) {
	start := time.Now()
	response, err := s.GetPropertiesWithPagination(ginCtx, 0, 1000)
	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("get_all_properties", "properties").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("get_all_properties", "properties").Inc()
		return nil, err
	}
	properties := make([]models.Property, len(response.Data))
	for i, prop := range response.Data {
		properties[i] = *prop.Property
	}
	return properties, nil
}

func (s *PropertyService) invalidatePropertiesCache(ctx context.Context, propertyID string) error {
	start := time.Now()
	err := cache.InvalidatePropertyCacheKeys(ctx, propertyID)
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("invalidate_cache").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("invalidate_cache", "").Inc()
		return fmt.Errorf("failed to invalidate cache keys for property %s", propertyID)
	}
	listKey := cache.PropertyListKey()
	start = time.Now()
	err = cache.Delete(ctx, listKey)
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("delete_list_cache").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("delete_list_cache", "").Inc()
	}
	return nil
}

func (s *PropertyService) GetPropertyByID(ginCtx *gin.Context, id string) (*models.Property, error) {
	ctx := context.Background()
	propertyKey := cache.PropertyKey(id)

	var property models.Property
	start := time.Now()
	err := cache.Get(ctx, propertyKey, &property)
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("get_property").Observe(duration)
	if err == nil {
		s.setDataSource(ginCtx, "REDIS_CACHE", true)
		return &property, nil
	}
	if err != redis.Nil {
		metrics.RedisErrorsTotal.WithLabelValues("get_property", "").Inc()
	}

	collection := database.DB.Collection("properties")
	start = time.Now()
	err = collection.FindOne(ctx, bson.M{"propertyId": id}).Decode(&property)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("find_one", "properties").Observe(duration)
	if err == nil {
		start = time.Now()
		err = cache.Set(ctx, propertyKey, property, Month)
		duration = time.Since(start).Seconds()
		metrics.RedisOperationDuration.WithLabelValues("set_property").Observe(duration)
		if err != nil {
			metrics.RedisErrorsTotal.WithLabelValues("set_property", "").Inc()
		}
		s.setDataSource(ginCtx, "MONGODB", false)
		return &property, nil
	}

	if err != mongo.ErrNoDocuments {
		metrics.MongoErrorsTotal.WithLabelValues("find_one", "properties").Inc()
		return nil, fmt.Errorf("failed to query property")
	}

	start = time.Now()
	mockData, err := s.readMockData(id)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("read_mock_data", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("read_mock_data", "").Inc()
		return nil, fmt.Errorf("failed to fetch mock data")
	}

	start = time.Now()
	tempProperty, err := s.TransformAPIResponse(mockData)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("transform_mock_data", "").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("transform_mock_data", "").Inc()
		return nil, fmt.Errorf("failed to transform mock data")
	}
	property = *tempProperty

	property.ID = primitive.NewObjectID()

	start = time.Now()
	_, err = collection.InsertOne(ctx, property)
	duration = time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("insert", "properties").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("insert", "properties").Inc()
		return nil, fmt.Errorf("failed to insert property to MongoDB")
	}

	start = time.Now()
	err = cache.Set(ctx, propertyKey, property, Month)
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("set_property").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set_property", "").Inc()
	}

	start = time.Now()
	err = s.invalidatePropertiesCache(ctx, id)
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("invalidate_cache").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("invalidate_cache", "").Inc()
	}

	s.setDataSource(ginCtx, "MOCK_DATA", false)
	return &property, nil
}

func (s *PropertyService) CreateProperty(ginCtx *gin.Context, property *models.Property) error {
	if property.PropertyID == "" || property.Address.StreetAddress == "" {
		return fmt.Errorf("property ID and street address are required")
	}

	property.Address.StreetAddress = s.normalizeAddressComponent(property.Address.StreetAddress)
	if property.Address.City != "" {
		property.Address.City = s.normalizeAddressComponent(property.Address.City)
	}
	if property.Address.State != "" {
		property.Address.State = s.normalizeAddressComponent(property.Address.State)
	}
	if property.Address.ZipCode != "" {
		property.Address.ZipCode = s.normalizeAddressComponent(property.Address.ZipCode)
	}

	property.ID = primitive.NewObjectID()
	ctx := context.Background()
	collection := database.DB.Collection("properties")

	start := time.Now()
	_, err := collection.InsertOne(ctx, property)
	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("insert", "properties").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("insert", "properties").Inc()
		return fmt.Errorf("failed to insert property")
	}

	propertyKey := cache.PropertyKey(property.PropertyID)
	start = time.Now()
	err = cache.Set(ctx, propertyKey, property, Month)
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("set_property").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set_property", "").Inc()
	}

	start = time.Now()
	err = s.invalidatePropertiesCache(ctx, property.PropertyID)
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("invalidate_cache").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("invalidate_cache", "").Inc()
	}

	s.setDataSource(ginCtx, "MONGODB_INSERT", false)
	return nil
}

func (s *PropertyService) UpdateProperty(ginCtx *gin.Context, property *models.Property) error {
	if property.PropertyID == "" || property.Address.StreetAddress == "" {
		return fmt.Errorf("property ID and street address are required")
	}

	property.Address.StreetAddress = s.normalizeAddressComponent(property.Address.StreetAddress)
	if property.Address.City != "" {
		property.Address.City = s.normalizeAddressComponent(property.Address.City)
	}
	if property.Address.State != "" {
		property.Address.State = s.normalizeAddressComponent(property.Address.State)
	}
	if property.Address.ZipCode != "" {
		property.Address.ZipCode = s.normalizeAddressComponent(property.Address.ZipCode)
	}

	ctx := context.Background()
	collection := database.DB.Collection("properties")

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
	result, err := collection.UpdateOne(ctx, bson.M{"propertyId": property.PropertyID}, update)
	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("update_one", "properties").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("update_one", "properties").Inc()
		return fmt.Errorf("failed to update property")
	}
	if result.MatchedCount == 0 {
		metrics.MongoErrorsTotal.WithLabelValues("update_one", "properties").Inc()
		return fmt.Errorf("property not found")
	}

	propertyKey := cache.PropertyKey(property.PropertyID)
	start = time.Now()
	err = cache.Set(ctx, propertyKey, property, Month)
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("set_property").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("set_property", "").Inc()
	}

	start = time.Now()
	err = s.invalidatePropertiesCache(ctx, property.PropertyID)
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("invalidate_cache").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("invalidate_cache", "").Inc()
	}

	s.setDataSource(ginCtx, "MONGODB_UPDATE", false)
	return nil
}

func (s *PropertyService) DeleteProperty(ginCtx *gin.Context, id string) error {
	ctx := context.Background()
	collection := database.DB.Collection("properties")

	start := time.Now()
	result, err := collection.DeleteOne(ctx, bson.M{"propertyId": id})
	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("delete_one", "properties").Observe(duration)
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("delete_one", "properties").Inc()
		return fmt.Errorf("failed to delete property")
	}
	if result.DeletedCount == 0 {
		metrics.MongoErrorsTotal.WithLabelValues("delete_one", "properties").Inc()
		return fmt.Errorf("property not found")
	}

	start = time.Now()
	err = s.invalidatePropertiesCache(ctx, id)
	duration = time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("invalidate_cache").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("invalidate_cache", "").Inc()
	}

	s.setDataSource(ginCtx, "MONGODB_DELETE", false)
	return nil
}

func (s *PropertyService) MigrateAddressesToUppercase(ctx context.Context) error {
	collection := database.DB.Collection("properties")
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("find", "properties").Inc()
		return fmt.Errorf("failed to query properties for migration: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var property models.Property
		if err := cursor.Decode(&property); err != nil {
			metrics.MongoErrorsTotal.WithLabelValues("cursor_decode", "properties").Inc()
			continue
		}

		// Normalize address fields to uppercase
		property.Address.StreetAddress = s.normalizeAddressComponent(property.Address.StreetAddress)
		if property.Address.City != "" {
			property.Address.City = s.normalizeAddressComponent(property.Address.City)
		}
		if property.Address.State != "" {
			property.Address.State = s.normalizeAddressComponent(property.Address.State)
		}
		if property.Address.ZipCode != "" {
			property.Address.ZipCode = s.normalizeAddressComponent(property.Address.ZipCode)
		}
		if property.Address.CarrierRoute != "" {
			property.Address.CarrierRoute = s.normalizeAddressComponent(property.Address.CarrierRoute)
		}
		if property.Address.StreetAddressParsed.HouseNumber != "" {
			property.Address.StreetAddressParsed.HouseNumber = s.normalizeAddressComponent(property.Address.StreetAddressParsed.HouseNumber)
		}
		if property.Address.StreetAddressParsed.StreetName != "" {
			property.Address.StreetAddressParsed.StreetName = s.normalizeAddressComponent(property.Address.StreetAddressParsed.StreetName)
		}
		if property.Address.StreetAddressParsed.StreetNameSuffix != "" {
			property.Address.StreetAddressParsed.StreetNameSuffix = s.normalizeAddressComponent(property.Address.StreetAddressParsed.StreetNameSuffix)
		}
		if property.Ownership.MailingAddress.StreetAddress != "" {
			property.Ownership.MailingAddress.StreetAddress = s.normalizeAddressComponent(property.Ownership.MailingAddress.StreetAddress)
		}
		if property.Ownership.MailingAddress.City != "" {
			property.Ownership.MailingAddress.City = s.normalizeAddressComponent(property.Ownership.MailingAddress.City)
		}
		if property.Ownership.MailingAddress.State != "" {
			property.Ownership.MailingAddress.State = s.normalizeAddressComponent(property.Ownership.MailingAddress.State)
		}
		if property.Ownership.MailingAddress.ZipCode != "" {
			property.Ownership.MailingAddress.ZipCode = s.normalizeAddressComponent(property.Ownership.MailingAddress.ZipCode)
		}
		if property.Ownership.MailingAddress.CarrierRoute != "" {
			property.Ownership.MailingAddress.CarrierRoute = s.normalizeAddressComponent(property.Ownership.MailingAddress.CarrierRoute)
		}

		// Update the document in MongoDB
		update := bson.M{
			"$set": bson.M{
				"address":   property.Address,
				"ownership": property.Ownership,
			},
		}
		start := time.Now()
		result, err := collection.UpdateOne(ctx, bson.M{"_id": property.ID}, update)
		duration := time.Since(start).Seconds()
		metrics.MongoOperationDuration.WithLabelValues("update_one", "properties").Observe(duration)
		if err != nil {
			metrics.MongoErrorsTotal.WithLabelValues("update_one", "properties").Inc()
			continue
		}
		if result.MatchedCount == 0 {
			continue
		}

		// Update cache
		propertyKey := cache.PropertyKey(property.PropertyID)
		start = time.Now()
		err = cache.Set(ctx, propertyKey, property, Month)
		duration = time.Since(start).Seconds()
		metrics.RedisOperationDuration.WithLabelValues("set_property").Observe(duration)
		if err != nil {
			metrics.RedisErrorsTotal.WithLabelValues("set_property", "").Inc()
		}

		// Invalidate related cache keys
		start = time.Now()
		err = s.invalidatePropertiesCache(ctx, property.PropertyID)
		duration = time.Since(start).Seconds()
		metrics.RedisOperationDuration.WithLabelValues("invalidate_cache").Observe(duration)
		if err != nil {
			metrics.RedisErrorsTotal.WithLabelValues("invalidate_cache", "").Inc()
		}
	}

	if err := cursor.Err(); err != nil {
		metrics.MongoErrorsTotal.WithLabelValues("cursor", "properties").Inc()
		return fmt.Errorf("cursor error during migration: %v", err)
	}

	return nil
}

func (s *PropertyService) ClearAllCache(ctx context.Context) error {
	start := time.Now()
	err := cache.RedisClient.FlushAll(ctx).Err()
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues("flush_all").Observe(duration)
	if err != nil {
		metrics.RedisErrorsTotal.WithLabelValues("flush_all", "").Inc()
		return fmt.Errorf("failed to clear Redis cache: %v", err)
	}
	return nil
}

func (s *PropertyService) TransformAPIResponse(apiResponse map[string]interface{}) (*models.Property, error) {
	start := time.Now()
	property := &models.Property{}

	if buildings, ok := apiResponse["buildings"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if clip, ok := buildings["clip"].(string); ok && clip != "" {
			property.PropertyID = clip
			property.AVMPropertyID = fmt.Sprintf("47149:%s", clip)
		} else {
			metrics.MongoErrorsTotal.WithLabelValues("transform_mock_data", "").Inc()
			return nil, fmt.Errorf("clip field is missing or invalid in mock data")
		}
	} else {
		metrics.MongoErrorsTotal.WithLabelValues("transform_mock_data", "").Inc()
		return nil, fmt.Errorf("buildings.data field is missing in mock data")
	}

	if ownership, ok := apiResponse["ownership"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if mailing, ok := ownership["currentOwnerMailingInfo"].(map[string]interface{})["mailingAddress"].(map[string]interface{}); ok {
			property.Address = models.Address{
				StreetAddress: s.normalizeAddressComponent(getString(mailing, "streetAddress")),
				City:          s.normalizeAddressComponent(getString(mailing, "city")),
				State:         s.normalizeAddressComponent(getString(mailing, "state")),
				ZipCode:       s.normalizeAddressComponent(getString(mailing, "zipCode")),
				CarrierRoute:  s.normalizeAddressComponent(getString(mailing, "carrierRoute")),
			}
			if parsed, ok := mailing["streetAddressParsed"].(map[string]interface{}); ok {
				property.Address.StreetAddressParsed = models.StreetAddressParsed{
					HouseNumber:      s.normalizeAddressComponent(getString(parsed, "houseNumber")),
					StreetName:       s.normalizeAddressComponent(getString(parsed, "streetName")),
					StreetNameSuffix: s.normalizeAddressComponent(getString(parsed, "mailingMode")),
				}
			}
		}
	}

	if siteLocation, ok := apiResponse["siteLocation"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		property.Location = models.Location{
			Coordinates: models.Coordinates{
				Parcel: models.CoordinatesPoint{
					Lat: getFloat64(siteLocation, "coordinatesParcel.lat"),
					Lng: getFloat64(siteLocation, "coordinatesParcel.lng"),
				},
				Block: models.CoordinatesPoint{
					Lat: getFloat64(siteLocation, "coordinatesBlock.lat"),
					Lng: getFloat64(siteLocation, "coordinatesBlock.lng"),
				},
			},
			Legal: models.Legal{
				SubdivisionName:           getString(siteLocation, "locationLegal.subdivisionName"),
				SubdivisionPlatBookNumber: getString(siteLocation, "locationLegal.subdivisionPlatBookNumber"),
				SubdivisionPlatPageNumber: getString(siteLocation, "locationLegal.subdivisionPlatPageNumber"),
			},
			CBSA: models.CBSA{
				Code: getString(siteLocation, "cbsa.code"),
				Type: getString(siteLocation, "cbsa.type"),
			},
			CensusTract: models.CensusTract{
				ID: getString(siteLocation, "censusTract.id"),
			},
		}
	}

	if siteLocation, ok := apiResponse["siteLocation"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		property.Lot = models.Lot{
			AreaAcres:            getFloat64(siteLocation, "lot.areaAcres"),
			AreaSquareFeet:       getInt(siteLocation, "lot.areaSquareFeet"),
			AreaSquareFeetUsable: getInt(siteLocation, "lot.areaSquareFeetUsable"),
			TopographyType:       getString(siteLocation, "lot.topographyType"),
		}
	}

	if siteLocation, ok := apiResponse["siteLocation"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		property.LandUseAndZoning = models.LandUseAndZoning{
			PropertyTypeCode:        getString(siteLocation, "landUseAndZoningCodes.propertyTypeCode"),
			LandUseCode:             getString(siteLocation, "landUseAndZoningCodes.landUseCode"),
			StateLandUseCode:        getString(siteLocation, "landUseAndZoningCodes.stateLandUseCode"),
			StateLandUseDescription: getString(siteLocation, "landUseAndZoningCodes.stateLandUseDescription"),
		}
	}

	if siteLocation, ok := apiResponse["siteLocation"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		property.Utilities = models.Utilities{
			FuelTypeCode:             getString(siteLocation, "utilities.fuelTypeCode"),
			ElectricityWiringTypeCode: getString(siteLocation, "utilities.electricityWiringTypeCode"),
			SewerTypeCode:            getString(siteLocation, "utilities.sewerTypeCode"),
			UtilitiesTypeCode:        getString(siteLocation, "utilities.utilitiesTypeCode"),
			WaterTypeCode:            getString(siteLocation, "utilities.waterTypeCode"),
		}
	}

	if buildings, ok := apiResponse["buildings"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		property.Building = models.Building{
			Summary: models.BuildingSummary{
				BuildingsCount:        getInt(buildings, "allBuildingsSummary.buildingsCount"),
				BathroomsCount:        getInt(buildings, "allBuildingsSummary.bathroomsCount"),
				FullBathroomsCount:    getInt(buildings, "allBuildingsSummary.fullBathroomsCount"),
				HalfBathroomsCount:    getInt(buildings, "allBuildingsSummary.halfBathroomsCount"),
				BathroomFixturesCount: getInt(buildings, "allBuildingsSummary.bathroomFixturesCount"),
				FireplacesCount:       getInt(buildings, "allBuildingsSummary.fireplacesCount"),
				LivingAreaSquareFeet:  getInt(buildings, "allBuildingsSummary.livingAreaSquareFeet"),
				TotalAreaSquareFeet:   getInt(buildings, "allBuildingsSummary.totalAreaSquareFeet"),
			},
		}
		if buildingList, ok := buildings["buildings"].([]interface{}); ok && len(buildingList) > 0 {
			if building, ok := buildingList[0].(map[string]interface{}); ok {
				property.Building.Details = models.BuildingDetails{
					StructureID: models.StructureID{
						SequenceNumber:              getInt(building, "structureId.sequenceNumber"),
						CompositeBuildingLinkageKey: getString(building, "structureId.compositeBuildingLinkageKey"),
						BuildingNumber:              getString(building, "structureId.buildingNumber"),
					},
					Classification: models.Classification{
						BuildingTypeCode: getString(building, "structureClassification.buildingTypeCode"),
						GradeTypeCode:    getString(building, "structureClassification.gradeTypeCode"),
					},
					VerticalProfile: models.VerticalProfile{
						StoriesCount: getInt(building, "structureVerticalProfile.storiesCount"),
					},
					Construction: models.Construction{
						YearBuilt:                       getInt(building, "constructionDetails.yearBuilt"),
						EffectiveYearBuilt:              getInt(building, "constructionDetails.effectiveYearBuilt"),
						BuildingQualityTypeCode:         getString(building, "constructionDetails.buildingQualityTypeCode"),
						FrameTypeCode:                   getString(building, "constructionDetails.frameTypeCode"),
						FoundationTypeCode:              getString(building, "constructionDetails.foundationTypeCode"),
						BuildingImprovementConditionCode: getString(building, "constructionDetails.buildingImprovementConditionCode"),
					},
					Exterior: models.Exterior{
						Patios: models.Patios{
							Count:          getInt(building, "structureExterior.patios.count"),
							TypeCode:       getString(building, "structureExterior.patios.typeCode"),
							AreaSquareFeet: getInt(building, "structureExterior.patios.areaSquareFeet"),
						},
						Porches: models.Porches{
							Count:          getInt(building, "structureExterior.porches.count"),
							TypeCode:       getString(building, "structureExterior.porches.typeCode"),
							AreaSquareFeet: getInt(building, "structureExterior.porches.areaSquareFeet"),
						},
						Pool: models.Pool{
							TypeCode:       getString(building, "structureExterior.pool.typeCode"),
							AreaSquareFeet: getInt(building, "structureExterior.pool.areaSquareFeet"),
						},
						Walls: models.Walls{
							TypeCode: getString(building, "structureExterior.walls.typeCode"),
						},
						Roof: models.Roof{
							TypeCode:      getString(building, "structureExterior.roof.typeCode"),
							CoverTypeCode: getString(building, "structureExterior.roof.coverTypeCode"),
						},
					},
					Interior: models.Interior{
						Area: models.InteriorArea{
							UniversalBuildingAreaSquareFeet:  getInt(building, "interiorArea.universalBuildingAreaSquareFeet"),
							LivingAreaSquareFeet:             getInt(building, "interiorArea.livingAreaSquareFeet"),
							AboveGradeAreaSquareFeet:         getInt(building, "interiorArea.aboveGradeAreaSquareFeet"),
							GroundFloorAreaSquareFeet:        getInt(building, "interiorArea.groundFloorAreaSquareFeet"),
							BasementAreaSquareFeet:           getInt(building, "interiorArea.basementAreaSquareFeet"),
							UnfinishedBasementAreaSquareFeet: getInt(building, "interiorArea.unfinishedBasementAreaSquareFeet"),
							AboveGroundFloorAreaSquareFeet:   getInt(building, "interiorArea.aboveGroundFloorAreaSquareFeet"),
							BuildingAdditionsAreaSquareFeet:  getInt(building, "interiorArea.buildingAdditionsAreaSquareFeet"),
						},
						Walls: models.Walls{
							TypeCode: getString(building, "structureInterior.walls.typeCode"),
						},
						Basement: models.Basement{
							TypeCode: getString(building, "structureInterior.basement.typeCode"),
						},
						Flooring: models.Flooring{
							CoverTypeCode: getString(building, "structureInterior.flooring.coverTypeCode"),
						},
						Features: models.Features{
							AirConditioning: models.AirConditioning{
								TypeCode: getString(building, "structureFeatures.airConditioning.typeCode"),
							},
							Heating: models.Heating{
								TypeCode: getString(building, "structureFeatures.heating.typeCode"),
							},
							Fireplaces: models.Fireplaces{
								TypeCode: getString(building, "structureFeatures.firePlaces.typeCode"),
								Count:    getInt(building, "structureFeatures.firePlaces.count"),
							},
						},
					},
				}
			}
		}
	}

	if ownership, ok := apiResponse["ownership"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if currentOwners, ok := ownership["currentOwners"].(map[string]interface{}); ok {
			property.Ownership = models.Ownership{
				RelationshipTypeCode: getString(currentOwners, "relationshipTypeCode"),
				OccupancyCode:        getString(currentOwners, "occupancyCode"),
			}
			if ownerNames, ok := currentOwners["ownerNames"].([]interface{}); ok {
				for _, owner := range ownerNames {
					if ownerMap, ok := owner.(map[string]interface{}); ok {
						property.Ownership.CurrentOwners = append(property.Ownership.CurrentOwners, models.Owner{
							SequenceNumber: getInt(ownerMap, "sequenceNumber"),
							FullName:       getString(ownerMap, "fullName"),
							FirstName:      getString(ownerMap, "firstName"),
							MiddleName:     getString(ownerMap, "middleName"),
							LastName:       getString(ownerMap, "lastName"),
							IsCorporate:    getBool(ownerMap, "isCorporate"),
						})
					}
				}
			}
			if mailing, ok := ownership["currentOwnerMailingInfo"].(map[string]interface{})["mailingAddress"].(map[string]interface{}); ok {
				property.Ownership.MailingAddress = models.MailingAddress{
					StreetAddress: s.normalizeAddressComponent(getString(mailing, "streetAddress")),
					City:          s.normalizeAddressComponent(getString(mailing, "city")),
					State:         s.normalizeAddressComponent(getString(mailing, "state")),
					ZipCode:       s.normalizeAddressComponent(getString(mailing, "zipCode")),
					CarrierRoute:  s.normalizeAddressComponent(getString(mailing, "carrierRoute")),
				}
			}
		}
	}

	if taxAssessment, ok := apiResponse["taxAssessment"].(map[string]interface{})["items"].([]interface{}); ok && len(taxAssessment) > 0 {
		if item, ok := taxAssessment[0].(map[string]interface{}); ok {
			property.TaxAssessment = models.TaxAssessment{
				Year:           getInt(item, "taxAmount.billedYear"),
				TotalTaxAmount: getInt(item, "taxAmount.totalTaxAmount"),
				CountyTaxAmount: getInt(item, "taxAmount.countyTaxAmount"),
				AssessedValue: models.AssessedValue{
					TotalValue:                 getInt(item, "assessedValue.calculatedTotalValue"),
					LandValue:                  getInt(item, "assessedValue.calculatedLandValue"),
					ImprovementValue:           getInt(item, "assessedValue.calculatedImprovementValue"),
					ImprovementValuePercentage: getInt(item, "assessedValue.calculatedImprovementValuePercentage"),
				},
				TaxRoll: models.TaxRoll{
					LastAssessorUpdateDate: getString(item, "taxrollUpdate.lastAssessorUpdateDate"),
					CertificationDate:      getString(item, "taxrollUpdate.taxrollCertificationDate"),
				},
				SchoolDistrict: models.SchoolDistrict{
					Code: getString(item, "schoolDistricts.code"),
					Name: getString(item, "schoolDistricts.name"),
				},
			}
		}
	}

	if lastMarketSale, ok := apiResponse["lastMarketSale"].(map[string]interface{})["items"].([]interface{}); ok && len(lastMarketSale) > 0 {
		if item, ok := lastMarketSale[0].(map[string]interface{}); ok {
			property.LastMarketSale = models.LastMarketSale{
				Date:                   getString(item, "transactionDetails.saleDateDerived"),
				RecordingDate:          getString(item, "transactionDetails.saleRecordingDateDerived"),
				Amount:                 getInt(item, "transactionDetails.saleAmount"),
				DocumentTypeCode:       getString(item, "transactionDetails.saleDocumentTypeCode"),
				DocumentNumber:         getString(item, "transactionDetails.saleDocumentNumber"),
				BookNumber:             getString(item, "transactionDetails.saleBookNumber"),
				PageNumber:             getString(item, "transactionDetails.salePageNumber"),
				MultiOrSplitParcelCode: getString(item, "transactionDetails.multiOrSplitParcelCode"),
				IsMortgagePurchase:     getBool(item, "transactionDetails.isMortgagePurchase"),
				IsResale:               getBool(item, "transactionDetails.isResale"),
				TitleCompany:           models.TitleCompany{
					Name: getString(item, "titleCompany.name"),
					Code: getString(item, "titleCompany.code"),
				},
			}
			if buyerNames, ok := item["buyerDetails"].(map[string]interface{})["buyerNames"].([]interface{}); ok {
				for _, buyer := range buyerNames {
					if buyerMap, ok := buyer.(map[string]interface{}); ok {
						property.LastMarketSale.Buyers = append(property.LastMarketSale.Buyers, models.Buyer{
							FullName:                  getString(buyerMap, "fullName"),
							LastName:                  getString(buyerMap, "lastName"),
							FirstNameAndMiddleInitial: getString(buyerMap, "firstNameAndMiddleInitial"),
						})
					}
				}
			}
			if sellerNames, ok := item["sellerDetails"].(map[string]interface{})["sellerNames"].([]interface{}); ok {
				for _, seller := range sellerNames {
					if sellerMap, ok := seller.(map[string]interface{}); ok {
						property.LastMarketSale.Sellers = append(property.LastMarketSale.Sellers, models.Seller{
							FullName: getString(sellerMap, "seller"),
						})
					}
				}
			}
		}
	}

	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues("transform_mock_data", "").Observe(duration)
	return property, nil
}

func getString(m map[string]interface{}, key string) string {
	keys := strings.Split(key, ".")
	current := m
	for _, k := range keys[:len(keys)-1] {
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			return ""
		}
	}
	if val, ok := current[keys[len(keys)-1]]; ok && val != nil {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	keys := strings.Split(key, ".")
	current := m
	for _, k := range keys[:len(keys)-1] {
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			return 0
		}
	}
	if val, ok := current[keys[len(keys)-1]]; ok && val != nil {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case int64:
			return int(v)
		}
	}
	return 0
}

func getFloat64(m map[string]interface{}, key string) float64 {
	keys := strings.Split(key, ".")
	current := m
	for _, k := range keys[:len(keys)-1] {
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			return 0
		}
	}
	if val, ok := current[keys[len(keys)-1]]; ok && val != nil {
		if v, ok := val.(float64); ok {
			return v
		}
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	keys := strings.Split(key, ".")
	current := m
	for _, k := range keys[:len(keys)-1] {
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			return false
		}
	}
	if val, ok := current[keys[len(keys)-1]]; ok && val != nil {
		if v, ok := val.(bool); ok {
			return v
		}
	}
	return false
}
