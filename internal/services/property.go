package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/cache"
	"homeinsight-properties/pkg/database"
)

type PropertyService struct{}

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

// Helper function to get data source
func (s *PropertyService) getDataSource(ginCtx *gin.Context) string {
	if ginCtx != nil {
		if source, exists := ginCtx.Get("data_source"); exists {
			return source.(string)
		}
	}
	return ""
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

// readMockData reads and parses the mock data from data/coreLogic/property-detail.json
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

func (s *PropertyService) GetPropertiesWithPagination(ginCtx *gin.Context, offset, limit int) (*models.PaginatedPropertiesResponse, error) {
	ctx := context.Background()

	// Create cache key that includes pagination parameters
	listKey := cache.PropertyListPaginatedKey(offset, limit)

	// Try to get from cache first
	var cachedResponse models.PaginatedPropertiesResponse
	err := cache.Get(ctx, listKey, &cachedResponse)
	if err == nil {
		s.setDataSource(ginCtx, "REDIS_CACHE", true)
		return &cachedResponse, nil
	}

	if err != redis.Nil {
		fmt.Printf("Cache error for paginated properties: %v\n", err)
	}

	// Get total count
	collection := database.DB.Collection("properties")
	total, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %v", err)
	}

	// Get paginated properties
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

	// If no properties found, fall back to mock data
	if len(properties) == 0 && total == 0 {
		mockData, err := s.readMockData("default")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch mock data: %v", err)
		}

		// Transform mock data
		property, err := s.TransformAPIResponse(mockData)
		if err != nil {
			return nil, fmt.Errorf("failed to transform mock data: %v", err)
		}

		// Assign a new PropertyID
		property.PropertyID = uuid.New().String()
		property.ID = primitive.NewObjectID()

		// Save to MongoDB
		_, err = collection.InsertOne(ctx, property)
		if err != nil {
			return nil, fmt.Errorf("failed to insert mock property to MongoDB: %v", err)
		}

		// Cache the individual property
		propertyKey := cache.PropertyKey(property.PropertyID)
		if err := cache.Set(ctx, propertyKey, property, 1*time.Hour); err != nil {
			fmt.Printf("Failed to cache property %s: %v\n", property.PropertyID, err)
		}

		// Add to properties list
		properties = append(properties, *property)
		total = 1 // Update total count

		// Invalidate all list caches
		s.invalidatePropertiesCache(ctx)

		s.setDataSource(ginCtx, "MOCK_DATA", false)
	}

	// Build pagination metadata
	metadata := models.PaginationMeta{
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
		metadata.Next = &nextURL
	}

	// Previous page URL
	if offset > 0 {
		prevOffset := offset - limit
		if prevOffset < 0 {
			prevOffset = 0
		}
		prevURL := s.buildPaginationURL(baseURL, prevOffset, limit, queryParams)
		metadata.Prev = &prevURL
	}

	response := &models.PaginatedPropertiesResponse{
		Data:     properties,
		Metadata: metadata,
	}

	// Cache the results for 15 minutes
	if err := cache.Set(ctx, listKey, response, 15*time.Minute); err != nil {
		fmt.Printf("Failed to cache paginated properties: %v\n", err)
	}

	if len(properties) > 0 && s.getDataSource(ginCtx) != "MOCK_DATA" {
		s.setDataSource(ginCtx, "MONGODB", false)
	}

	return response, nil
}

func (s *PropertyService) GetAllProperties(ginCtx *gin.Context) ([]models.Property, error) {
	result, err := s.GetPropertiesWithPagination(ginCtx, 0, 1000)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (s *PropertyService) invalidatePropertiesCache(ctx context.Context) {
	listKey := cache.PropertyListKey()
	if err := cache.Delete(ctx, listKey); err != nil {
		fmt.Printf("Failed to invalidate properties list cache: %v\n", err)
	}
}

func (s *PropertyService) CreateProperty(ginCtx *gin.Context, property *models.Property) error {
	if property.PropertyID == "" || property.Address.StreetAddress == "" {
		return fmt.Errorf("property ID and street address are required")
	}

	property.ID = primitive.NewObjectID()

	ctx := context.Background()
	collection := database.DB.Collection("properties")

	_, err := collection.InsertOne(ctx, property)
	if err != nil {
		return fmt.Errorf("failed to insert property: %v", err)
	}

	// Cache the new property
	propertyKey := cache.PropertyKey(property.PropertyID)
	if err := cache.Set(ctx, propertyKey, property, 1*time.Hour); err != nil {
		fmt.Printf("Failed to cache property %s: %v\n", property.PropertyID, err)
	}

	// Invalidate all list caches
	s.invalidatePropertiesCache(ctx)

	s.setDataSource(ginCtx, "MONGODB_INSERT", false)
	return nil
}

func (s *PropertyService) GetPropertyByID(ginCtx *gin.Context, id string) (*models.Property, error) {
	ctx := context.Background()
	propertyKey := cache.PropertyKey(id)

	// Try cache first
	var property models.Property
	err := cache.Get(ctx, propertyKey, &property)
	if err == nil {
		s.setDataSource(ginCtx, "REDIS_CACHE", true)
		return &property, nil
	}

	if err != redis.Nil {
		fmt.Printf("Cache error for property %s: %v\n", id, err)
	}

	// Get from MongoDB
	collection := database.DB.Collection("properties")
	err = collection.FindOne(ctx, bson.M{"propertyId": id}).Decode(&property)
	if err == nil {
		// Cache for 1 hour
		if err := cache.Set(ctx, propertyKey, &property, 1*time.Hour); err != nil {
			fmt.Printf("Failed to cache property %s: %v\n", id, err)
		}
		s.setDataSource(ginCtx, "MONGODB", false)
		return &property, nil
	}

	if err != mongo.ErrNoDocuments {
		fmt.Printf("Database error for property %s: %v\n", id, err)
		return nil, fmt.Errorf("failed to query property: %v", err)
	}

	// Property not found in cache or database, fetch from mock data
	fmt.Printf("Property %s not found in cache or database, attempting to fetch from mock data\n", id)
	mockData, err := s.readMockData(id)
	if err != nil {
		fmt.Printf("Error reading mock data for property %s: %v\n", id, err)
		return nil, fmt.Errorf("failed to fetch mock data: %v", err)
	}

	// Transform the mock data
	tempProperty, err := s.TransformAPIResponse(mockData)
	if err != nil {
		fmt.Printf("Error transforming mock data for property %s: %v\n", id, err)
		return nil, fmt.Errorf("failed to transform mock data: %v", err)
	}
	property = *tempProperty

	// Override the PropertyID to match the requested ID
	property.PropertyID = id
	property.ID = primitive.NewObjectID()

	// Save to MongoDB
	_, err = collection.InsertOne(ctx, property)
	if err != nil {
		fmt.Printf("Error inserting mock property %s to MongoDB: %v\n", id, err)
		return nil, fmt.Errorf("failed to insert property to MongoDB: %v", err)
	}

	// Cache for 1 hour
	if err := cache.Set(ctx, propertyKey, property, 1*time.Hour); err != nil {
		fmt.Printf("Failed to cache property %s: %v\n", id, err)
	}

	// Invalidate all list caches
	s.invalidatePropertiesCache(ctx)

	fmt.Printf("Property %s loaded from mock data, saved to MongoDB, and cached\n", id)
	s.setDataSource(ginCtx, "MOCK_DATA", false)
	return &property, nil
}

func (s *PropertyService) UpdateProperty(ginCtx *gin.Context, property *models.Property) error {
	if property.PropertyID == "" || property.Address.StreetAddress == "" {
		return fmt.Errorf("property ID and street address are required")
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

	// Update cache
	propertyKey := cache.PropertyKey(property.PropertyID)
	if err := cache.Set(ctx, propertyKey, property, 1*time.Hour); err != nil {
		fmt.Printf("Failed to update property cache %s: %v\n", property.PropertyID, err)
	}

	// Invalidate all list caches
	s.invalidatePropertiesCache(ctx)

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

	// Remove from cache
	propertyKey := cache.PropertyKey(id)
	if err := cache.Delete(ctx, propertyKey); err != nil {
		fmt.Printf("Failed to delete property from cache %s: %v\n", id, err)
	}

	// Invalidate all list caches
	s.invalidatePropertiesCache(ctx)

	s.setDataSource(ginCtx, "MONGODB_DELETE", false)
	return nil
}

func (s *PropertyService) TransformAPIResponse(apiResponse map[string]interface{}) (*models.Property, error) {
	property := &models.Property{
		PropertyID: uuid.New().String(),
	}

	// Buildings
	if buildings, ok := apiResponse["buildings"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if clip, ok := buildings["clip"].(string); ok {
			property.AVMPropertyID = fmt.Sprintf("47149:%s", clip)
		}

		if summary, ok := buildings["allBuildingsSummary"].(map[string]interface{}); ok {
			property.Building.Summary = models.BuildingSummary{
				BuildingsCount:       getInt(summary, "buildingsCount"),
				LivingAreaSquareFeet: getInt(summary, "livingAreaSquareFeet"),
				TotalAreaSquareFeet:  getInt(summary, "totalAreaSquareFeet"),
			}
		}

		if buildingsArray, ok := buildings["buildings"].([]interface{}); ok && len(buildingsArray) > 0 {
			b := buildingsArray[0].(map[string]interface{})
			details := models.BuildingDetails{
				StructureID: models.StructureID{
					SequenceNumber:             getInt(b["structureId"].(map[string]interface{}), "sequenceNumber"),
					CompositeBuildingLinkageKey: getString(b["structureId"].(map[string]interface{}), "compositeBuildingLinkageKey"),
					BuildingNumber:             getString(b["structureId"].(map[string]interface{}), "buildingNumber"),
				},
				Classification: models.Classification{
					BuildingTypeCode: getString(b["structureClassification"].(map[string]interface{}), "buildingTypeCode"),
				},
				Construction: models.Construction{
					YearBuilt:               getInt(b["constructionDetails"].(map[string]interface{}), "yearBuilt"),
					FoundationTypeCode:      getString(b["constructionDetails"].(map[string]interface{}), "foundationTypeCode"),
					BuildingQualityTypeCode: getString(b["constructionDetails"].(map[string]interface{}), "buildingQualityTypeCode"),
				},
				Exterior: models.Exterior{
					Walls: models.Walls{TypeCode: getString(b["structureExterior"].(map[string]interface{})["walls"].(map[string]interface{}), "typeCode")},
					Roof: models.Roof{
						TypeCode:      getString(b["structureExterior"].(map[string]interface{})["roof"].(map[string]interface{}), "typeCode"),
						CoverTypeCode: getString(b["structureExterior"].(map[string]interface{})["roof"].(map[string]interface{}), "coverTypeCode"),
					},
				},
				Interior: models.Interior{
					Walls:    models.Walls{TypeCode: getString(b["structureInterior"].(map[string]interface{})["walls"].(map[string]interface{}), "typeCode")},
					Flooring: models.Flooring{CoverTypeCode: getString(b["structureInterior"].(map[string]interface{})["flooring"].(map[string]interface{}), "typeCode")},
					Area: models.InteriorArea{
						UniversalBuildingAreaSquareFeet: getInt(b["interiorArea"].(map[string]interface{}), "universalBuildingAreaSquareFeet"),
						LivingAreaSquareFeet:           getInt(b["interiorArea"].(map[string]interface{}), "livingAreaSquareFeet"),
						GroundFloorAreaSquareFeet:      getInt(b["interiorArea"].(map[string]interface{}), "groundFloorAreaSquareFeet"),
						AboveGroundFloorAreaSquareFeet: getInt(b["interiorArea"].(map[string]interface{}), "aboveGroundFloorAreaSquareFeet"),
					},
				},
			}
			property.Building.Details = details
		}
	}

	// Ownership
	if ownership, ok := apiResponse["ownership"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if currentOwners, ok := ownership["currentOwners"].(map[string]interface{}); ok {
			owners := []models.Owner{}
			if ownerNames, ok := currentOwners["ownerNames"].([]interface{}); ok {
				for _, owner := range ownerNames {
					o := owner.(map[string]interface{})
					owners = append(owners, models.Owner{
						SequenceNumber: getInt(o, "sequenceNumber"),
						FullName:       getString(o, "fullName"),
						IsCorporate:    getBool(o, "isCorporate"),
					})
				}
			}
			property.Ownership.CurrentOwners = owners
			property.Ownership.OccupancyCode = getString(currentOwners, "occupancyCode")
		}
		if mailing, ok := ownership["currentOwnerMailingInfo"].(map[string]interface{})["mailingAddress"].(map[string]interface{}); ok {
			property.Ownership.MailingAddress = models.MailingAddress{
				StreetAddress: getString(mailing, "streetAddress"),
				City:          getString(mailing, "city"),
				State:         getString(mailing, "state"),
				ZipCode:       getString(mailing, "zipCode"),
				CarrierRoute:  getString(mailing, "carrierRoute"),
			}
			if parsed, ok := mailing["streetAddressParsed"].(map[string]interface{}); ok {
				property.Address.StreetAddressParsed = models.StreetAddressParsed{
					HouseNumber:      getString(parsed, "houseNumber"),
					StreetName:       getString(parsed, "streetName"),
					StreetNameSuffix: getString(parsed, "mailingMode"),
				}
				// Set StreetAddress for consistency
				property.Address.StreetAddress = getString(mailing, "streetAddress")
			}
		}
	}

	// Site Location
	if site, ok := apiResponse["siteLocation"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if coords, ok := site["coordinatesParcel"].(map[string]interface{}); ok {
			property.Location.Coordinates.Parcel = models.CoordinatesPoint{
				Lat: getFloat64(coords, "lat"),
				Lng: getFloat64(coords, "lng"),
			}
		}
		if coords, ok := site["coordinatesBlock"].(map[string]interface{}); ok {
			property.Location.Coordinates.Block = models.CoordinatesPoint{
				Lat: getFloat64(coords, "lat"),
				Lng: getFloat64(coords, "lng"),
			}
		}
		if legal, ok := site["locationLegal"].(map[string]interface{}); ok {
			property.Location.Legal = models.Legal{
				SubdivisionName:          getString(legal, "subdivisionName"),
				SubdivisionPlatBookNumber: getString(legal, "subdivisionPlatBookNumber"),
				SubdivisionPlatPageNumber: getString(legal, "subdivisionPlatPageNumber"),
			}
		}
		if cbsa, ok := site["cbsa"].(map[string]interface{}); ok {
			property.Location.CBSA = models.CBSA{
				Code: getString(cbsa, "code"),
				Type: getString(cbsa, "type"),
			}
		}
		if census, ok := site["censusTract"].(map[string]interface{}); ok {
			property.Location.CensusTract = models.CensusTract{
				ID: getString(census, "id"),
			}
		}
		if lot, ok := site["lot"].(map[string]interface{}); ok {
			property.Lot = models.Lot{
				AreaAcres:      getFloat64(lot, "areaAcres"),
				AreaSquareFeet: getInt(lot, "areaSquareFeet"),
			}
		}
		if utilities, ok := site["utilities"].(map[string]interface{}); ok {
			property.Utilities = models.Utilities{
				FuelTypeCode:             getString(utilities, "fuelTypeCode"),
				ElectricityWiringTypeCode: getString(utilities, "electricityWiringTypeCode"),
				SewerTypeCode:            getString(utilities, "sewerTypeCode"),
				UtilitiesTypeCode:        getString(utilities, "utilitiesTypeCode"),
				WaterTypeCode:            getString(utilities, "waterTypeCode"),
			}
		}
	}

	// Tax Assessment
	if tax, ok := apiResponse["taxAssessment"].(map[string]interface{})["items"].([]interface{}); ok && len(tax) > 0 {
		t := tax[0].(map[string]interface{})
		if ta, ok := t["taxAmount"].(map[string]interface{}); ok {
			if av, ok := t["assessedValue"].(map[string]interface{}); ok {
				if tr, ok := t["taxrollUpdate"].(map[string]interface{}); ok {
					if sd, ok := t["schoolDistricts"].(map[string]interface{})["school"].(map[string]interface{}); ok {
						property.TaxAssessment = models.TaxAssessment{
							Year:           getInt(ta, "billedYear"),
							TotalTaxAmount: getInt(ta, "totalTaxAmount"),
							AssessedValue: models.AssessedValue{
								TotalValue:                 getInt(av, "calculatedTotalValue"),
								LandValue:                  getInt(av, "calculatedLandValue"),
								ImprovementValue:           getInt(av, "calculatedImprovementValue"),
								ImprovementValuePercentage: getInt(av, "calculatedImprovementValuePercentage"),
							},
							TaxRoll: models.TaxRoll{
								LastAssessorUpdateDate: getString(tr, "lastAssessorUpdateDate"),
								CertificationDate:      getString(tr, "taxrollCertificationDate"),
							},
							SchoolDistrict: models.SchoolDistrict{
								Name: getString(sd, "name"),
							},
						}
					}
				}
			}
		}
	}

	// Last Market Sale
	if sale, ok := apiResponse["lastMarketSale"].(map[string]interface{})["items"].([]interface{}); ok && len(sale) > 0 {
		s := sale[0].(map[string]interface{})
		if td, ok := s["transactionDetails"].(map[string]interface{}); ok {
			if tc, ok := s["titleCompany"].(map[string]interface{}); ok {
				buyers := []models.Buyer{}
				if buyerDetails, ok := s["buyerDetails"].(map[string]interface{}); ok {
					if buyerNames, ok := buyerDetails["buyerNames"].([]interface{}); ok {
						for _, buyer := range buyerNames {
							b := buyer.(map[string]interface{})
							buyers = append(buyers, models.Buyer{
								FullName:                 getString(b, "fullName"),
								LastName:                 getString(b, "lastName"),
								FirstNameAndMiddleInitial: getString(b, "firstNameAndMiddleInitial"),
							})
						}
					}
				}
				sellers := []models.Seller{}
				if sellerDetails, ok := s["sellerDetails"].(map[string]interface{}); ok {
					if sellerNames, ok := sellerDetails["sellerNames"].([]interface{}); ok {
						for _, seller := range sellerNames {
							sellers = append(sellers, models.Seller{
								FullName: getString(seller.(map[string]interface{}), "fullName"),
							})
						}
					}
				}
				property.LastMarketSale = models.LastMarketSale{
					Date:                   getString(td, "saleDateDerived"),
					RecordingDate:          getString(td, "saleRecordingDateDerived"),
					Amount:                 getInt(td, "saleAmount"),
					DocumentTypeCode:       getString(td, "saleDocumentTypeCode"),
					DocumentNumber:         getString(td, "saleDocumentNumber"),
					BookNumber:             getString(td, "saleBookNumber"),
					PageNumber:             getString(td, "salePageNumber"),
					MultiOrSplitParcelCode: getString(td, "multiOrSplitParcelCode"),
					IsMortgagePurchase:     getBool(td, "isMortgagePurchase"),
					IsResale:               getBool(td, "isResale"),
					Buyers:                 buyers,
					Sellers:                sellers,
					TitleCompany: models.TitleCompany{
						Name: getString(tc, "name"),
						Code: getString(tc, "code"),
					},
				}
			}
		}
	}

	// Validate required fields (optional, log warning instead of error)
	if property.Address.StreetAddress == "" {
		fmt.Println("Warning: StreetAddress is empty after transformation, proceeding with empty address")
	}

	return property, nil
}

// Helper functions to safely extract values
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok && val != nil {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok && val != nil {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return 0
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok && val != nil {
		if v, ok := val.(float64); ok {
			return v
		}
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok && val != nil {
		if v, ok := val.(bool); ok {
			return v
		}
	}
	return false
}
