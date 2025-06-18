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

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TTL Period (30 days).
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
	q := u.Query()

	q.Set("offset", fmt.Sprintf("%d", offset))
	q.Set("limit", fmt.Sprintf("%d", limit))

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

func (s *PropertyService) readMockData(_ string) (map[string]interface{}, error) {
	filePath, err := filepath.Abs("data/coreLogic/property-detail.json")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve mock data file path: %v", err)
	}

	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mock data file %s: %v", filePath, err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(file, &data); err != nil {
		return nil, fmt.Errorf("failed to parse mock data from %s: %v", filePath, err)
	}

	return data, nil
}

func (s *PropertyService) normalizeAddressComponent(input string) string {
	return cache.NormalizeAddressComponent(input)
}

func (s *PropertyService) parseAddress(search string) (streetAddress, city, state, zipCode string) {
	search = strings.TrimSpace(search)
	if search == "" {
		return "", "", "", ""
	}

	// Regex for full address: street, city, state zip
	re := regexp.MustCompile(`^(.*?),\s*([^,]+),\s*([A-Z]{2})\s*(\d{5})$`)
	matches := re.FindStringSubmatch(search)
	if len(matches) == 5 {
		return s.normalizeAddressComponent(matches[1]), s.normalizeAddressComponent(matches[2]),
			s.normalizeAddressComponent(matches[3]), s.normalizeAddressComponent(matches[4])
	}

	// Try street, city
	parts := strings.Split(search, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	if len(parts) == 2 {
		return s.normalizeAddressComponent(parts[0]), s.normalizeAddressComponent(parts[1]), "", ""
	}

	// Try street, city, state
	if len(parts) == 3 {
		stateZip := strings.Split(parts[2], " ")
		if len(stateZip) >= 2 {
			return s.normalizeAddressComponent(parts[0]), s.normalizeAddressComponent(parts[1]),
				s.normalizeAddressComponent(stateZip[0]), s.normalizeAddressComponent(stateZip[1])
		}
		return s.normalizeAddressComponent(parts[0]), s.normalizeAddressComponent(parts[1]),
			s.normalizeAddressComponent(parts[2]), ""
	}

	return s.normalizeAddressComponent(search), "", "", ""
}

func (s *PropertyService) SearchSpecificProperty(ginCtx *gin.Context, req *models.SearchRequest) (*models.Property, error) {
	ctx := context.Background()

	// Normalize search query
	req.Search = s.normalizeAddressComponent(req.Search)

	// Parse address components
	street, city, state, zip := s.parseAddress(req.Search)
	if street == "" || city == "" {
		return nil, fmt.Errorf("street address and city are required")
	}

	// Generate cache key based on street and city only
	cacheKey := cache.PropertySpecificSearchKey(street, city)

	// Try cache first
	var propertyID string
	var property models.Property
	if err := cache.Get(ctx, cacheKey, &propertyID); err == nil {
		if err := cache.Get(ctx, cache.PropertyKey(propertyID), &property); err == nil {
			s.setDataSource(ginCtx, "REDIS_CACHE", true)
			fmt.Printf("Cache hit for search key %s, property ID %s\n", cacheKey, propertyID)
			return &property, nil
		}
	}
	if err := cache.Get(ctx, cacheKey, &propertyID); err != nil && err != redis.Nil {
		fmt.Printf("Cache error for search key %s: %v\n", cacheKey, err)
	}

	// Query MongoDB
	collection := database.DB.Collection("properties")
	filter := bson.M{
		"normalizedAddress.normalizedStreetAddress": street,
		"normalizedAddress.normalizedCity":          city,
	}
	if state != "" {
		filter["normalizedAddress.normalizedState"] = state
	}
	if zip != "" {
		filter["normalizedAddress.normalizedZipCode"] = zip
	}

	err := collection.FindOne(ctx, filter).Decode(&property)
	if err == nil {
		// Cache property
		propertyKey := cache.PropertyKey(property.PropertyID)
		if err := cache.Set(ctx, propertyKey, property, 1*Month); err != nil {
			fmt.Printf("Failed to cache property %s: %v\n", property.PropertyID, err)
		}
		// Cache search key with property ID
		if err := cache.Set(ctx, cacheKey, property.PropertyID, 1*Month); err != nil {
			fmt.Printf("Failed to cache search key %s: %v\n", cacheKey, err)
		}
		// Add search key to property:keys set
		if err := cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, cacheKey); err != nil {
			fmt.Printf("Failed to add search key %s to property set %s: %v\n", cacheKey, property.PropertyID, err)
		}
		// Set expiration for property:keys set
		cache.RedisClient.Expire(ctx, cache.PropertyKeysSetKey(property.PropertyID), 1*Month)
		s.setDataSource(ginCtx, "MONGODB", false)
		fmt.Printf("Cached property %s with search key %s\n", property.PropertyID, cacheKey)
		return &property, nil
	}

	if err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("failed to query property: %v", err)
	}

	// Try mock data
	mockData, err := s.readMockData("default")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mock data: %v", err)
	}

	propertyPtr, err := s.TransformAPIResponse(mockData)
	if err != nil {
		return nil, fmt.Errorf("failed to transform mock data: %v", err)
	}
	property = *propertyPtr

	property.ID = primitive.NewObjectID()
	property.Address.StreetAddress = street
	property.Address.City = city
	property.Address.State = state
	property.Address.ZipCode = zip
	property.NormalizedAddress = models.NormalizedAddress{
		StreetAddress: street,
		City:          city,
		State:         state,
		ZipCode:       zip,
	}

	_, err = collection.InsertOne(ctx, property)
	if err != nil {
		return nil, fmt.Errorf("failed to insert mock property: %v", err)
	}

	// Cache property
	propertyKey := cache.PropertyKey(property.PropertyID)
	if err := cache.Set(ctx, propertyKey, property, 1*Month); err != nil {
		fmt.Printf("Failed to cache property %s: %v\n", property.PropertyID, err)
	}
	// Cache search key
	if err := cache.Set(ctx, cacheKey, property.PropertyID, 1*Month); err != nil {
		fmt.Printf("Failed to cache search key %s: %v\n", cacheKey, err)
	}
	// Add search key to property:keys set
	if err := cache.AddCacheKeyToPropertySet(ctx, property.PropertyID, cacheKey); err != nil {
		fmt.Printf("Failed to add search key %s to property set %s: %v\n", cacheKey, property.PropertyID, err)
	}
	cache.RedisClient.Expire(ctx, cache.PropertyKeysSetKey(property.PropertyID), 1*Month)

	s.setDataSource(ginCtx, "MOCK_DATA", false)
	fmt.Printf("Inserted and cached mock property %s with search key %s\n", property.PropertyID, cacheKey)
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

	// Try cache
	cacheKey := cache.PropertyListPaginatedKey(offset, limit)
	var response models.PaginatedPropertiesResponse
	if err := cache.Get(ctx, cacheKey, &response); err == nil {
		s.setDataSource(ginCtx, "REDIS_CACHE", true)
		fmt.Printf("Cache hit for list key %s\n", cacheKey)
		return &response, nil
	}

	// Query MongoDB
	collection := database.DB.Collection("properties")
	total, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %v", err)
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "address.streetAddress", Value: 1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to query properties: %v", err)
	}
	defer cursor.Close(ctx)

	properties := []models.Property{}
	if err := cursor.All(ctx, &properties); err != nil {
		return nil, fmt.Errorf("failed to decode properties: %v", err)
	}

	// Cache properties
	for _, property := range properties {
		propertyKey := cache.PropertyKey(property.PropertyID)
		if err := cache.Set(ctx, propertyKey, property, 1*Month); err != nil {
			fmt.Printf("Failed to cache property %s: %v\n", property.PropertyID, err)
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

	// Cache response
	if err := cache.Set(ctx, cacheKey, response, 1*Month); err != nil {
		fmt.Printf("Failed to cache list key %s: %v\n", cacheKey, err)
	}

	s.setDataSource(ginCtx, "MONGODB", false)
	fmt.Printf("Retrieved %d properties for list key %s\n", len(properties), cacheKey)
	return &response, nil
}

func (s *PropertyService) GetAllProperties(ginCtx *gin.Context) ([]models.Property, error) {
	response, err := s.GetPropertiesWithPagination(ginCtx, 0, 1000)
	if err != nil {
		return nil, err
	}
	properties := make([]models.Property, len(response.Data))
	for i, prop := range response.Data {
		properties[i] = *prop.Property
	}
	return properties, nil
}

func (s *PropertyService) invalidatePropertiesCache(ctx context.Context, propertyID string) error {
	if err := cache.InvalidatePropertyCacheKeys(ctx, propertyID); err != nil {
		return fmt.Errorf("failed to invalidate cache keys for property %s: %v", propertyID, err)
	}
	listKey := cache.PropertyListKey()
	if err := cache.Delete(ctx, listKey); err != nil {
		fmt.Printf("Failed to invalidate properties list cache: %v\n", err)
	}
	return nil
}

func (s *PropertyService) GetPropertyByID(ginCtx *gin.Context, id string) (*models.Property, error) {
	ctx := context.Background()
	propertyKey := cache.PropertyKey(id)

	var property models.Property
	if err := cache.Get(ctx, propertyKey, &property); err == nil {
		s.setDataSource(ginCtx, "REDIS_CACHE", true)
		fmt.Printf("Cache hit for property %s\n", id)
		return &property, nil
	} else if err != redis.Nil {
		fmt.Printf("Cache error for property %s: %v\n", id, err)
	}

	collection := database.DB.Collection("properties")
	err := collection.FindOne(ctx, bson.M{"propertyId": id}).Decode(&property)
	if err == nil {
		if property.NormalizedAddress.StreetAddress == "" {
			property.NormalizedAddress = models.NormalizedAddress{
				StreetAddress: s.normalizeAddressComponent(property.Address.StreetAddress),
				City:          s.normalizeAddressComponent(property.Address.City),
				State:         s.normalizeAddressComponent(property.Address.State),
				ZipCode:       s.normalizeAddressComponent(property.Address.ZipCode),
			}
			_, err := collection.UpdateOne(ctx, bson.M{"propertyId": id}, bson.M{
				"$set": bson.M{"normalizedAddress": property.NormalizedAddress},
			})
			if err != nil {
				fmt.Printf("Failed to update normalized address for property %s: %v\n", id, err)
			}
		}
		if err := cache.Set(ctx, propertyKey, &property, 1*Month); err != nil {
			fmt.Printf("Failed to cache property %s: %v\n", id, err)
		}
		s.setDataSource(ginCtx, "MONGODB", false)
		fmt.Printf("Retrieved and cached property %s from MongoDB\n", id)
		return &property, nil
	}

	if err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("failed to query property: %v", err)
	}

	mockData, err := s.readMockData(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mock data: %v", err)
	}

	tempProperty, err := s.TransformAPIResponse(mockData)
	if err != nil {
		return nil, fmt.Errorf("failed to transform mock data: %v", err)
	}
	property = *tempProperty

	property.ID = primitive.NewObjectID()

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
	property.NormalizedAddress = models.NormalizedAddress{
		StreetAddress: property.Address.StreetAddress,
		City:          property.Address.City,
		State:         property.Address.State,
		ZipCode:       property.Address.ZipCode,
	}

	_, err = collection.InsertOne(ctx, property)
	if err != nil {
		return nil, fmt.Errorf("failed to insert property to MongoDB: %v", err)
	}

	if err := cache.Set(ctx, propertyKey, property, 1*Month); err != nil {
		fmt.Printf("Failed to cache property %s: %v\n", id, err)
	}

	s.invalidatePropertiesCache(ctx, id)
	s.setDataSource(ginCtx, "MOCK_DATA", false)
	fmt.Printf("Inserted and cached mock property %s\n", id)
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

	property.NormalizedAddress = models.NormalizedAddress{
		StreetAddress: property.Address.StreetAddress,
		City:          property.Address.City,
		State:         property.Address.State,
		ZipCode:       property.Address.ZipCode,
	}

	property.ID = primitive.NewObjectID()
	ctx := context.Background()
	collection := database.DB.Collection("properties")

	_, err := collection.InsertOne(ctx, property)
	if err != nil {
		return fmt.Errorf("failed to insert property: %v", err)
	}

	propertyKey := cache.PropertyKey(property.PropertyID)
	if err := cache.Set(ctx, propertyKey, property, 1*Month); err != nil {
		fmt.Printf("Failed to cache property %s: %v\n", property.PropertyID, err)
	}

	s.invalidatePropertiesCache(ctx, property.PropertyID)
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

	property.NormalizedAddress = models.NormalizedAddress{
		StreetAddress: property.Address.StreetAddress,
		City:          property.Address.City,
		State:         property.Address.State,
		ZipCode:       property.Address.ZipCode,
	}

	ctx := context.Background()
	collection := database.DB.Collection("properties")

	update := bson.M{
		"$set": bson.M{
			"avmPropertyId":     property.AVMPropertyID,
			"address":           property.Address,
			"normalizedAddress": property.NormalizedAddress,
			"location":          property.Location,
			"lot":               property.Lot,
			"landUseAndZoning":  property.LandUseAndZoning,
			"utilities":         property.Utilities,
			"building":          property.Building,
			"ownership":         property.Ownership,
			"taxAssessment":     property.TaxAssessment,
			"lastMarketSale":    property.LastMarketSale,
		},
	}

	result, err := collection.UpdateOne(ctx, bson.M{"propertyId": property.PropertyID}, update)
	if err != nil {
		return fmt.Errorf("failed to update property: %v", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("property not found")
	}

	propertyKey := cache.PropertyKey(property.PropertyID)
	if err := cache.Set(ctx, propertyKey, property, 1*Month); err != nil {
		fmt.Printf("Failed to update property cache %s: %v\n", property.PropertyID, err)
	}

	s.invalidatePropertiesCache(ctx, property.PropertyID)
	s.setDataSource(ginCtx, "MONGODB_UPDATE", false)
	return nil
}

func (s *PropertyService) DeleteProperty(ginCtx *gin.Context, id string) error {
	ctx := context.Background()
	collection := database.DB.Collection("properties")

	result, err := collection.DeleteOne(ctx, bson.M{"propertyId": id})
	if err != nil {
		return fmt.Errorf("failed to delete property: %v", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("property not found")
	}

	s.invalidatePropertiesCache(ctx, id)
	s.setDataSource(ginCtx, "MONGODB_DELETE", false)
	return nil
}

func (s *PropertyService) TransformAPIResponse(apiResponse map[string]interface{}) (*models.Property, error) {
	property := &models.Property{}

	// Extract clip and set PropertyID and AVMPropertyID
	if buildings, ok := apiResponse["buildings"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if clip, ok := buildings["clip"].(string); ok && clip != "" {
			property.PropertyID = clip
			property.AVMPropertyID = fmt.Sprintf("47149:%s", clip)
		} else {
			return nil, fmt.Errorf("clip field is missing or invalid in mock data")
		}
	} else {
		return nil, fmt.Errorf("buildings.data field is missing in mock data")
	}

	// Set address fields
	if ownership, ok := apiResponse["ownership"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if mailing, ok := ownership["currentOwnerMailingInfo"].(map[string]interface{})["mailingAddress"].(map[string]interface{}); ok {
			property.Address.StreetAddress = getString(mailing, "streetAddress")
			property.Address.City = getString(mailing, "city")
			property.Address.State = getString(mailing, "state")
			property.Address.ZipCode = getString(mailing, "zipCode")
			property.Address.StreetAddress = s.normalizeAddressComponent(property.Address.StreetAddress)
			property.Address.City = s.normalizeAddressComponent(property.Address.City)
			property.Address.State = s.normalizeAddressComponent(property.Address.State)
			property.Address.ZipCode = s.normalizeAddressComponent(property.Address.ZipCode)
			property.NormalizedAddress = models.NormalizedAddress{
				StreetAddress: property.Address.StreetAddress,
				City:          property.Address.City,
				State:         property.Address.State,
				ZipCode:       property.Address.ZipCode,
			}
		}
	}

	return property, nil
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok && val != nil {
		return fmt.Sprintf("%v", val)
	}
	return ""
}
