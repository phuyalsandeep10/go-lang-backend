package transformers

import (
	"fmt"
	"strings"
	"time"

	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/metrics"
)


type propertyTransformer struct{}

func NewPropertyTransformer() PropertyTransformer {
	return &propertyTransformer{}
}

func (t *propertyTransformer) TransformAPIResponse(apiResponse map[string]interface{}) (*models.Property, error) {
	start := time.Now()
	defer func() {
		metrics.MongoOperationDuration.WithLabelValues("transform_api_response", "").Observe(time.Since(start).Seconds())
	}()

	property := &models.Property{}

	if buildings, ok := apiResponse["buildings"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if clip, ok := buildings["clip"].(string); ok && clip != "" {
			property.PropertyID = clip
			property.AVMPropertyID = fmt.Sprintf("47149:%s", clip)
		} else {
			metrics.MongoErrorsTotal.WithLabelValues("transform_api_response", "").Inc()
			return nil, fmt.Errorf("clip field is missing or invalid")
		}
	} else {
		metrics.MongoErrorsTotal.WithLabelValues("transform_api_response", "").Inc()
		return nil, fmt.Errorf("buildings.data field is missing")
	}

	if ownership, ok := apiResponse["ownership"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if mailing, ok := ownership["currentOwnerMailingInfo"].(map[string]interface{})["mailingAddress"].(map[string]interface{}); ok {
			property.Address = models.Address{
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
			FuelTypeCode:              getString(siteLocation, "utilities.fuelTypeCode"),
			ElectricityWiringTypeCode: getString(siteLocation, "utilities.electricityWiringTypeCode"),
			SewerTypeCode:             getString(siteLocation, "utilities.sewerTypeCode"),
			UtilitiesTypeCode:         getString(siteLocation, "utilities.utilitiesTypeCode"),
			WaterTypeCode:             getString(siteLocation, "utilities.waterTypeCode"),
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
				BedroomsCount:         getInt(buildings, "allBuildingsSummary.bedroomsCount"),
				KitchensCount:         getInt(buildings, "allBuildingsSummary.kitchensCount"),
				FamilyRoomsCount:      getInt(buildings, "allBuildingsSummary.familyRoomsCount"),
				LivingRoomsCount:      getInt(buildings, "allBuildingsSummary.livingRoomsCount"),
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
						YearBuilt:                        getInt(building, "constructionDetails.yearBuilt"),
						EffectiveYearBuilt:               getInt(building, "constructionDetails.effectiveYearBuilt"),
						BuildingQualityTypeCode:          getString(building, "constructionDetails.buildingQualityTypeCode"),
						FrameTypeCode:                    getString(building, "constructionDetails.frameTypeCode"),
						FoundationTypeCode:               getString(building, "constructionDetails.foundationTypeCode"),
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
						Parking: models.Parking{
							TypeCode:           getString(building, "structureExterior.parking.typeCode"),
							ParkingSpacesCount: getInt(building, "structureExterior.parking.parkingSpacesCount"),
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
					StreetAddress: getString(mailing, "streetAddress"),
					City:          getString(mailing, "city"),
					State:         getString(mailing, "state"),
					ZipCode:       getString(mailing, "zipCode"),
					CarrierRoute:  getString(mailing, "carrierRoute"),
				}
			}
		}
	}

	if taxAssessment, ok := apiResponse["taxAssessment"].(map[string]interface{})["items"].([]interface{}); ok && len(taxAssessment) > 0 {
		if item, ok := taxAssessment[0].(map[string]interface{}); ok {
			property.TaxAssessment = models.TaxAssessment{
				Year:            getInt(item, "taxAmount.billedYear"),
				TotalTaxAmount:  getInt(item, "taxAmount.totalTaxAmount"),
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
					Code: getString(item, "schoolDistricts.school.code"),
					Name: getString(item, "schoolDistricts.school.name"),
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
				TitleCompany: models.TitleCompany{
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
