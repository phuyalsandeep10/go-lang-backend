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
	if err := cache.Get(ctx, cacheKey, &propertyID); err != redis.Nil {
		fmt.Printf("Cache error for search key %s: %v\n", cacheKey, err)
	}

	// Query MongoDB
	collection := database.DB.Collection("properties")
	filter := bson.M{
		"address.streetAddress": street,
		"address.city":          city,
	}
	if state != "" {
		filter["address.state"] = state
	}
	if zip != "" {
		filter["address.zipCode"] = zip
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
		if err := cache.Set(ctx, propertyKey, property, 1*Month); err != nil {
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

	// Map Address
	if ownership, ok := apiResponse["ownership"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if mailing, ok := ownership["currentOwnerMailingInfo"].(map[string]interface{})["mailingAddress"].(map[string]interface{}); ok {
			property.Address = models.Address{
				StreetAddress: s.normalizeAddressComponent(getString(mailing, "streetAddress")),
				City:          s.normalizeAddressComponent(getString(mailing, "city")),
				State:         s.normalizeAddressComponent(getString(mailing, "state")),
				ZipCode:       s.normalizeAddressComponent(getString(mailing, "zipCode")),
				CarrierRoute:  getString(mailing, "carrierRoute"),
			}
			if parsed, ok := mailing["streetAddressParsed"].(map[string]interface{}); ok {
				property.Address.StreetAddressParsed = models.StreetAddressParsed{
					HouseNumber:      getString(parsed, "houseNumber"),
					StreetName:       getString(parsed, "streetName"),
					StreetNameSuffix: getString(parsed, "mailingMode"),
				}
			}
		}
	}

	// Map Location
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
				SubdivisionName:          getString(siteLocation, "locationLegal.subdivisionName"),
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

	// Map Lot
	if siteLocation, ok := apiResponse["siteLocation"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		property.Lot = models.Lot{
			AreaAcres:          getFloat64(siteLocation, "lot.areaAcres"),
			AreaSquareFeet:     getInt(siteLocation, "lot.areaSquareFeet"),
			AreaSquareFeetUsable: getInt(siteLocation, "lot.areaSquareFeetUsable"),
			TopographyType:     getString(siteLocation, "lot.topographyType"),
		}
	}

	// Map LandUseAndZoning
	if siteLocation, ok := apiResponse["siteLocation"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		property.LandUseAndZoning = models.LandUseAndZoning{
			PropertyTypeCode:      getString(siteLocation, "landUseAndZoningCodes.propertyTypeCode"),
			LandUseCode:           getString(siteLocation, "landUseAndZoningCodes.landUseCode"),
			StateLandUseCode:      getString(siteLocation, "landUseAndZoningCodes.stateLandUseCode"),
			StateLandUseDescription: getString(siteLocation, "landUseAndZoningCodes.stateLandUseDescription"),
		}
	}

	// Map Utilities
	if siteLocation, ok := apiResponse["siteLocation"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		property.Utilities = models.Utilities{
			FuelTypeCode:          getString(siteLocation, "utilities.fuelTypeCode"),
			ElectricityWiringTypeCode: getString(siteLocation, "utilities.electricityWiringTypeCode"),
			SewerTypeCode:         getString(siteLocation, "utilities.sewerTypeCode"),
			UtilitiesTypeCode:     getString(siteLocation, "utilities.utilitiesTypeCode"),
			WaterTypeCode:         getString(siteLocation, "utilities.waterTypeCode"),
		}
	}

	// Map Building
	if buildings, ok := apiResponse["buildings"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		property.Building = models.Building{
			Summary: models.BuildingSummary{
				BuildingsCount:      getInt(buildings, "allBuildingsSummary.buildingsCount"),
				BathroomsCount:      getInt(buildings, "allBuildingsSummary.bathroomsCount"),
				FullBathroomsCount:  getInt(buildings, "allBuildingsSummary.fullBathroomsCount"),
				HalfBathroomsCount:  getInt(buildings, "allBuildingsSummary.halfBathroomsCount"),
				BathroomFixturesCount: getInt(buildings, "allBuildingsSummary.bathroomFixturesCount"),
				FireplacesCount:     getInt(buildings, "allBuildingsSummary.fireplacesCount"),
				LivingAreaSquareFeet: getInt(buildings, "allBuildingsSummary.livingAreaSquareFeet"),
				TotalAreaSquareFeet: getInt(buildings, "allBuildingsSummary.totalAreaSquareFeet"),
			},
		}
		if buildingList, ok := buildings["buildings"].([]interface{}); ok && len(buildingList) > 0 {
			if building, ok := buildingList[0].(map[string]interface{}); ok {
				property.Building.Details = models.BuildingDetails{
					StructureID: models.StructureID{
						SequenceNumber:         getInt(building, "structureId.sequenceNumber"),
						CompositeBuildingLinkageKey: getString(building, "structureId.compositeBuildingLinkageKey"),
						BuildingNumber:         getString(building, "structureId.buildingNumber"),
					},
					Classification: models.Classification{
						BuildingTypeCode: getString(building, "structureClassification.buildingTypeCode"),
						GradeTypeCode:    getString(building, "structureClassification.gradeTypeCode"),
					},
					VerticalProfile: models.VerticalProfile{
						StoriesCount: getInt(building, "structureVerticalProfile.storiesCount"),
					},
					Construction: models.Construction{
						YearBuilt:                getInt(building, "constructionDetails.yearBuilt"),
						EffectiveYearBuilt:       getInt(building, "constructionDetails.effectiveYearBuilt"),
						BuildingQualityTypeCode:  getString(building, "constructionDetails.buildingQualityTypeCode"),
						FrameTypeCode:            getString(building, "constructionDetails.frameTypeCode"),
						FoundationTypeCode:       getString(building, "constructionDetails.foundationTypeCode"),
						BuildingImprovementConditionCode: getString(building, "constructionDetails.buildingImprovementConditionCode"),
					},
					Exterior: models.Exterior{
						Patios: models.Patios{
							Count:         getInt(building, "structureExterior.patios.count"),
							TypeCode:      getString(building, "structureExterior.patios.typeCode"),
							AreaSquareFeet: getInt(building, "structureExterior.patios.areaSquareFeet"),
						},
						Porches: models.Porches{
							Count:         getInt(building, "structureExterior.porches.count"),
							TypeCode:      getString(building, "structureExterior.porches.typeCode"),
							AreaSquareFeet: getInt(building, "structureExterior.porches.areaSquareFeet"),
						},
						Pool: models.Pool{
							TypeCode:      getString(building, "structureExterior.pool.typeCode"),
							AreaSquareFeet: getInt(building, "structureExterior.pool.areaSquareFeet"),
						},
						Walls: models.Walls{
							TypeCode: getString(building, "structureExterior.walls.typeCode"),
						},
						Roof: models.Roof{
							TypeCode:     getString(building, "structureExterior.roof.typeCode"),
							CoverTypeCode: getString(building, "structureExterior.roof.coverTypeCode"),
						},
					},
					Interior: models.Interior{
						Area: models.InteriorArea{
							UniversalBuildingAreaSquareFeet: getInt(building, "interiorArea.universalBuildingAreaSquareFeet"),
							LivingAreaSquareFeet:           getInt(building, "interiorArea.livingAreaSquareFeet"),
							AboveGradeAreaSquareFeet:       getInt(building, "interiorArea.aboveGradeAreaSquareFeet"),
							GroundFloorAreaSquareFeet:      getInt(building, "interiorArea.groundFloorAreaSquareFeet"),
							BasementAreaSquareFeet:         getInt(building, "interiorArea.basementAreaSquareFeet"),
							UnfinishedBasementAreaSquareFeet: getInt(building, "interiorArea.unfinishedBasementAreaSquareFeet"),
							AboveGroundFloorAreaSquareFeet:  getInt(building, "interiorArea.aboveGroundFloorAreaSquareFeet"),
							BuildingAdditionsAreaSquareFeet: getInt(building, "interiorArea.buildingAdditionsAreaSquareFeet"),
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

	// Map Ownership
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
					StreetAddress: getString(mailing, "streetAddress"),
					City:          getString(mailing, "city"),
					State:         getString(mailing, "state"),
					ZipCode:       getString(mailing, "zipCode"),
					CarrierRoute:  getString(mailing, "carrierRoute"),
				}
			}
		}
	}

	// Map TaxAssessment
	if taxAssessment, ok := apiResponse["taxAssessment"].(map[string]interface{})["items"].([]interface{}); ok && len(taxAssessment) > 0 {
		if item, ok := taxAssessment[0].(map[string]interface{}); ok {
			property.TaxAssessment = models.TaxAssessment{
				Year:           getInt(item, "taxAmount.billedYear"),
				TotalTaxAmount: getInt(item, "taxAmount.totalTaxAmount"),
				CountyTaxAmount: getInt(item, "taxAmount.countyTaxAmount"),
				AssessedValue: models.AssessedValue{
					TotalValue:            getInt(item, "assessedValue.calculatedTotalValue"),
					LandValue:             getInt(item, "assessedValue.calculatedLandValue"),
					ImprovementValue:      getInt(item, "assessedValue.calculatedImprovementValue"),
					ImprovementValuePercentage: getInt(item, "assessedValue.calculatedImprovementValuePercentage"),
				},
				TaxRoll: models.TaxRoll{
					LastAssessorUpdateDate: getString(item, "taxrollUpdate.lastAssessorUpdateDate"),
					CertificationDate:     getString(item, "taxrollUpdate.taxrollCertificationDate"),
				},
				SchoolDistrict: models.SchoolDistrict{
					Code: getString(item, "schoolDistricts.school.code"),
					Name: getString(item, "schoolDistricts.school.name"),
				},
			}
		}
	}

	// Map LastMarketSale
	if lastMarketSale, ok := apiResponse["lastMarketSale"].(map[string]interface{})["items"].([]interface{}); ok && len(lastMarketSale) > 0 {
		if item, ok := lastMarketSale[0].(map[string]interface{}); ok {
			property.LastMarketSale = models.LastMarketSale{
				Date:               getString(item, "transactionDetails.saleDateDerived"),
				RecordingDate:      getString(item, "transactionDetails.saleRecordingDateDerived"),
				Amount:             getInt(item, "transactionDetails.saleAmount"),
				DocumentTypeCode:   getString(item, "transactionDetails.saleDocumentTypeCode"),
				DocumentNumber:     getString(item, "transactionDetails.saleDocumentNumber"),
				BookNumber:         getString(item, "transactionDetails.saleBookNumber"),
				PageNumber:         getString(item, "transactionDetails.salePageNumber"),
				MultiOrSplitParcelCode: getString(item, "transactionDetails.multiOrSplitParcelCode"),
				IsMortgagePurchase: getBool(item, "transactionDetails.isMortgagePurchase"),
				IsResale:           getBool(item, "transactionDetails.isResale"),
				TitleCompany: models.TitleCompany{
					Name: getString(item, "titleCompany.name"),
					Code: getString(item, "titleCompany.code"),
				},
			}
			if buyerNames, ok := item["buyerDetails"].(map[string]interface{})["buyerNames"].([]interface{}); ok {
				for _, buyer := range buyerNames {
					if buyerMap, ok := buyer.(map[string]interface{}); ok {
						property.LastMarketSale.Buyers = append(property.LastMarketSale.Buyers, models.Buyer{
							FullName:             getString(buyerMap, "fullName"),
							LastName:             getString(buyerMap, "lastName"),
							FirstNameAndMiddleInitial: getString(buyerMap, "firstNameAndMiddleInitial"),
						})
					}
				}
			}
			if sellerNames, ok := item["sellerDetails"].(map[string]interface{})["sellerNames"].([]interface{}); ok {
				for _, seller := range sellerNames {
					if sellerMap, ok := seller.(map[string]interface{}); ok {
						property.LastMarketSale.Sellers = append(property.LastMarketSale.Sellers, models.Seller{
							FullName: getString(sellerMap, "fullName"),
						})
					}
				}
			}
		}
	}

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
