package services

import (
	"fmt"
	"github.com/google/uuid"
	"homeinsight-properties/internal/models"
)

func (s *PropertyService) TransformAPIResponse(apiResponse map[string]interface{}) (*models.Property, error) {

	// Ownership
	if ownership, ok := apiResponse["ownership"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if currentOwners, ok := ownership["currentOwners"].(map[string]interface{}); ok {
			owners := []models.Owner{}
			for _, owner := range currentOwners["ownerNames"].([]interface{}) {
				o := owner.(map[string]interface{})
				owners = append(owners, models.Owner{
					SequenceNumber: int(o["sequenceNumber"].(float64)),
					FullName:       o["fullName"].(string),
					IsCorporate:    o["isCorporate"].(bool),
				})
			}
			property.Ownership.CurrentOwners = owners
			property.Ownership.OccupancyCode = currentOwners["occupancyCode"].(string)
		}
		if mailing, ok := ownership["currentOwnerMailingInfo"].(map[string]interface{})["mailingAddress"].(map[string]interface{}); ok {
			property.Ownership.MailingAddress = models.MailingAddress{
				StreetAddress: mailing["streetAddress"].(string),
				City:         mailing["city"].(string),
				State:        mailing["state"].(string),
				ZipCode:      mailing["zipCode"].(string),
				CarrierRoute: mailing["carrierRoute"].(string),
			}
			if parsed, ok := mailing["streetAddressParsed"].(map[string]interface{}); ok {
				property.Address.StreetAddressParsed = models.StreetAddressParsed{
					HouseNumber:      parsed["houseNumber"].(string),
					StreetName:       parsed["streetName"].(string),
					StreetNameSuffix: parsed["mailingMode"].(string),
				}
			}
		}
	}

	// Site Location
	if site, ok := apiResponse["siteLocation"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if coords, ok := site["coordinatesParcel"].(map[string]interface{}); ok {
			property.Location.Coordinates.Parcel = models.CoordinatesPoint{
				Lat: coords["lat"].(float64),
				Lng: coords["lng"].(float64),
			}
		}
		if coords, ok := site["coordinatesBlock"].(map[string]interface{}); ok {
			property.Location.Coordinates.Block = models.CoordinatesPoint{
				Lat: coords["lat"].(float64),
				Lng: coords["lng"].(float64),
			}
		}
		if legal, ok := site["locationLegal"].(map[string]interface{}); ok {
			property.Location.Legal = models.Legal{
				SubdivisionName:          legal["subdivisionName"].(string),
				SubdivisionPlatBookNumber: legal["subdivisionPlatBookNumber"].(string),
				SubdivisionPlatPageNumber: legal["subdivisionPlatPageNumber"].(string),
			}
		}
		if cbsa, ok := site["cbsa"].(map[string]interface{}); ok {
			property.Location.CBSA = models.CBSA{
				Code: cbsa["code"].(string),
				Type: cbsa["type"].(string),
			}
		}
		if census, ok := site["censusTract"].(map[string]interface{}); ok {
			property.Location.CensusTract = models.CensusTract{
				ID: census["id"].(string),
			}
		}
		if lot, ok := site["lot"].(map[string]interface{}); ok {
			property.Lot = models.Lot{
				AreaAcres:      lot["areaAcres"].(float64),
				AreaSquareFeet: int(lot["areaSquareFeet"].(float64)),
			}
		}
		if utilities, ok := site["utilities"].(map[string]interface{}); ok {
			property.Utilities = models.Utilities{
				FuelTypeCode:         utilities["fuelTypeCode"].(string),
				ElectricityWiringTypeCode: utilities["electricityWiringTypeCode"].(string),
				SewerTypeCode:        utilities["sewerTypeCode"].(string),
				UtilitiesTypeCode:    utilities["utilitiesTypeCode"].(string),
				WaterTypeCode:        utilities["waterTypeCode"].(string),
			}
		}
	}

	// Tax Assessment
	if tax, ok := apiResponse["taxAssessment"].(map[string]interface{})["items"].([]interface{}); ok && len(tax) > 0 {
		t := tax[0].(map[string]interface{})
		ta := t["taxAmount"].(map[string]interface{})
		av := t["assessedValue"].(map[string]interface{})
		tr := t["taxrollUpdate"].(map[string]interface{})
		sd := t["schoolDistricts"].(map[string]interface{})["school"].(map[string]interface{})
		property.TaxAssessment = models.TaxAssessment{
			Year:           int(ta["billedYear"].(float64)),
			TotalTaxAmount: int(ta["totalTaxAmount"].(float64)),
			AssessedValue: models.AssessedValue{
				TotalValue:            int(av["calculatedTotalValue"].(float64)),
				LandValue:             int(av["calculatedLandValue"].(float64)),
				ImprovementValue:      int(av["calculatedImprovementValue"].(float64)),
				ImprovementValuePercentage: int(av["calculatedImprovementValuePercentage"].(float64)),
			},
			TaxRoll: models.TaxRoll{
				LastAssessorUpdateDate: tr["lastAssessorUpdateDate"].(string),
				CertificationDate:      tr["taxrollCertificationDate"].(string),
			},
			SchoolDistrict: models.SchoolDistrict{
				Name: sd["name"].(string),
			},
		}
	}

	// Last Market Sale
	if sale, ok := apiResponse["lastMarketSale"].(map[string]interface{})["items"].([]interface{}); ok && len(sale) > 0 {
		s := sale[0].(map[string]interface{})
		td := s["transactionDetails"].(map[string]interface{})
		tc := s["titleCompany"].(map[string]interface{})
		buyers := []models.Buyer{}
		for _, buyer := range s["buyerDetails"].(map[string]interface{})["buyerNames"].([]interface{}) {
			b := buyer.(map[string]interface{})
			buyers = append(buyers, models.Buyer{
				FullName:             b["fullName"].(string),
				LastName:             b["lastName"].(string),
				FirstNameAndMiddleInitial: b["firstNameAndMiddleInitial"].(string),
			})
		}
		sellers := []models.Seller{}
		for _, seller := range s["sellerDetails"].(map[string]interface{})["sellerNames"].([]interface{}) {
			sellers = append(sellers, models.Seller{
				FullName: seller.(map[string]interface{})["fullName"].(string),
			})
		}
		property.LastMarketSale = models.LastMarketSale{
			Date:               td["saleDateDerived"].(string),
			RecordingDate:      td["saleRecordingDateDerived"].(string),
			Amount:             int(td["saleAmount"].(float64)),
			DocumentTypeCode:   td["saleDocumentTypeCode"].(string),
			DocumentNumber:     td["saleDocumentNumber"].(string),
			BookNumber:         td["saleBookNumber"].(string),
			PageNumber:         td["salePageNumber"].(string),
			MultiOrSplitParcelCode: td["multiOrSplitParcelCode"].(string),
			IsMortgagePurchase: td["isMortgagePurchase"].(bool),
			IsResale:           td["isResale"].(bool),
			Buyers:             buyers,
			Sellers:            sellers,
			TitleCompany: models.TitleCompany{
				Name: tc["name"].(string),
				Code: tc["code"].(string),
			},
		}
	}

	return property, nil
}
	property := &models.Property{
		PropertyID: uuid.New().String(),
	}

	// Buildings
	if buildings, ok := apiResponse["buildings"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		clip := buildings["clip"].(string)
		property.AVMPropertyID = fmt.Sprintf("47149:%s", clip)

		if summary, ok := buildings["allBuildingsSummary"].(map[string]interface{}); ok {
			property.Building.Summary = models.BuildingSummary{
				BuildingsCount:       int(summary["buildingsCount"].(float64)),
				LivingAreaSquareFeet: int(summary["livingAreaSquareFeet"].(float64)),
				TotalAreaSquareFeet:  int(summary["totalAreaSquareFeet"].(float64)),
			}
		}

		if buildingsArray, ok := buildings["buildings"].([]interface{}); ok && len(buildingsArray) > 0 {
			b := buildingsArray[0].(map[string]interface{})
			details := models.BuildingDetails{
				StructureID: models.StructureID{
					SequenceNumber:         int(b["structureId"].(map[string]interface{})["sequenceNumber"].(float64)),
					CompositeBuildingLinkageKey: b["structureId"].(map[string]interface{})["compositeBuildingLinkageKey"].(string),
					BuildingNumber:         b["structureId"].(map[string]interface{})["buildingNumber"].(string),
				},
				Classification: models.Classification{
					BuildingTypeCode: b["structureClassification"].(map[string]interface{})["buildingTypeCode"].(string),
				},
				Construction: models.Construction{
					YearBuilt:               int(b["constructionDetails"].(map[string]interface{})["yearBuilt"].(float64)),
					FoundationTypeCode:      b["constructionDetails"].(map[string]interface{})["foundationTypeCode"].(string),
					BuildingQualityTypeCode: b["constructionDetails"].(map[string]interface{})["buildingQualityTypeCode"].(string),
				},
				Exterior: models.Exterior{
					Walls: models.Walls{TypeCode: b["structureExterior"].(map[string]interface{})["walls"].(map[string]interface{})["typeCode"].(string)},
					Roof:  models.Roof{
						TypeCode:      b["structureExterior"].(map[string]interface{})["roof"].(map[string]interface{})["typeCode"].(string),
						CoverTypeCode: b["structureExterior"].(map[string]interface{})["roof"].(map[string]interface{})["coverTypeCode"].(string),
					},
				},
				Interior: models.Interior{
					Walls: models.Walls{TypeCode: b["structureInterior"].(map[string]interface{})["walls"].(map[string]interface{})["typeCode"].(string)},
					Flooring: models.Flooring{CoverTypeCode: b["structureInterior"].(map[string]interface{})["flooring"].(map[string]interface{})["typeCode"].(string)},
					Area: models.InteriorArea{
						UniversalBuildingAreaSquareFeet: int(b["interiorArea"].(map[string]interface{})["universalBuildingAreaSquareFeet"].(float64)),
						LivingAreaSquareFeet:           int(b["interiorArea"].(map[string]interface{})["livingAreaSquareFeet"].(float64)),
						GroundFloorAreaSquareFeet:      int(b["interiorArea"].(map[string]interface{})["groundFloorAreaSquareFeet"].(float64)),
						AboveGroundFloorAreaSquareFeet: int(b["interiorArea"].(map[string]interface{})["aboveGroundFloorAreaSquareFeet"].(float64)),
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
			for _, owner := range currentOwners["ownerNames"].([]interface{}) {
				o := owner.(map[string]interface{})
				owners = append(owners, models.Owner{
					SequenceNumber: int(o["sequenceNumber"].(float64)),
					FullName:       o["fullName"].(string),
					IsCorporate:    o["isCorporate"].(bool),
				})
			}
			property.Ownership.CurrentOwners = owners
			property.Ownership.OccupancyCode = currentOwners["occupancyCode"].(string)
		}
		if mailing, ok := ownership["currentOwnerMailingInfo"].(map[string]interface{})["mailingAddress"].(map[string]interface{}); ok {
			property.Ownership.MailingAddress = models.MailingAddress{
				StreetAddress: mailing["streetAddress"].(string),
				City:         mailing["city"].(string),
				State:        mailing["state"].(string),
				ZipCode:      mailing["zipCode"].(string),
				CarrierRoute: mailing["carrierRoute"].(string),
			}
			if parsed, ok := mailing["streetAddressParsed"].(map[string]interface{}); ok {
				property.Address.StreetAddressParsed = models.StreetAddressParsed{
					HouseNumber:      parsed["houseNumber"].(string),
					StreetName:       parsed["streetName"].(string),
					StreetNameSuffix: parsed["mailingMode"].(string),
				}
			}
		}
	}

	// Site Location
	if site, ok := apiResponse["siteLocation"].(map[string]interface{})["data"].(map[string]interface{}); ok {
		if coords, ok := site["coordinatesParcel"].(map[string]interface{}); ok {
			property.Location.Coordinates.Parcel = models.CoordinatesPoint{
				Lat: coords["lat"].(float64),
				Lng: coords["lng"].(float64),
			}
		}
		if coords, ok := site["coordinatesBlock"].(map[string]interface{}); ok {
			property.Location.Coordinates.Block = models.CoordinatesPoint{
				Lat: coords["lat"].(float64),
				Lng: coords["lng"].(float64),
			}
		}
		if legal, ok := site["locationLegal"].(map[string]interface{}); ok {
			property.Location.Legal = models.Legal{
				SubdivisionName: legal["subdivisionName"].(string),
				SubdivisionPlatBookNumber: legal["subdivisionPlatBookNumber"].(string),
				SubdivisionPlatPageNumber: legal["subdivisionPlatPageNumber"].(string),
			}
		}
		if cbsa, ok := site["cbsa"].(map[string]interface{}); ok {
			property.Location.CBSA = models.CBSA{
				Code: cbsa["code"].(string),
				Type: cbsa["type"].(string),
			}
		}
		if census, ok := site["censusTract"].(map[string]interface{}); ok {
			property.Location.CensusTract = models.ID{
				ID: census["id"].(string),
			}
		}
		if lot, ok := site["lot"].(map[string]interface{}); ok {
			property.Lot = models.Lot{
				AreaAcres: lot["areaAcres"].(float64),
				AreaSquareFeet: int(lot["areaSquareFeet"].(float64)),
			}
		}
		if utilities, ok := site["utilities"].(map[string]interface{}); ok {
			property.Utilities = models.Utilities{
				FuelTypeCode: utilities["fuelTypeCode"].(string),
				ElectricityWiringTypeCode: utilities["electricityWiringTypeCode"].(string),
				SewerTypeCode: utilities["sewerTypeCode"].(string),
				UtilitiesTypeCode: utilities["utilitiesTypeCode"].(string),
				WaterTypeCode: utilities["waterTypeCode"].(string)),
			}
		}
	}

	// Tax Assessment
	if tax, ok := apiResponse["taxAssessment"].(map[string]interface{})["items"].([]interface{}); ok && len(tax) > 0 {
		t := tax[0].(map[string]interface{}))
		ta := t["taxAmount"].(map[string]interface{})
		av := t["assessedValue"].(map[string]interface{})
		tr := t["taxrollUpdate"].(map[string]interface{})
		sd := t["schoolDistricts"].(map[string]interface{})["school"].(map(string]interface{})) {
			property.TaxAssessment = models.TaxAssessment{
				Year: int(ta["billedYear"].(float64)),
				TotalTaxAmount: int(ta["totalTaxAmount"].(float64)),
				AssessedValue: models.AssessedValue{
					TotalValue: int(av["calculatedTotalValue"].(float64)),
					LandValue: int(av["calculatedLandValue"].(float64)),
					ImprovementValue: int(av["calculatedImprovementValue"].(float64)),
					ImprovementValuePercentage: int(av["calculatedImprovementValuePercentage"].(float64)),
				},
				{
					TaxRoll: models.TaxRoll{
						LandLastAssessorUpdateDate: tr["lastAssessorUpdateDate"].(string),
						CertificationDate: tr["taxrollCertification"].(string),
					},
					SchoolDistrict: models.SchoolDistrict{
						Name: sd["name"].(string),
					},
				}
			}
		}

	// Last Market Sale
		if sale, ok := apiResponse["lastMarketSale"].(map[string]interface{})["items"].([]interface{}); ok && len(sale) > 0 {
			s := sale[0].([]interface{})(map[string]interface{})
			td := s["transactionDetails"].(map[string]interface{})
			tc := td(s["titleCompany"].(map[string]interface{})),
			buyers := []interface{}models.Buyer{}
			for _, buyer := range buyers {
				s["buyersDetails"].(map[string]interface{})["buyerNames"].([]interface{})) {
				b := buyer.(map[string]interface{}))
				buyers = append(buyers, map[string]interface{}{
					"FullName": string(b["fullName"].(string)),
					"LastName": string(b["lastName"].(string)),
					"FirstNameAndMiddleInitial": string(b["firstNameAndMiddleInitial"].(string)),
				})
				}
			}
			sellers := []interface{}models.Seller{}
			for _, seller := range sellers {
				s["sellerDetails"].(map[string]interface{})["sellerNames"].([]interface{})) {
					sellers := append(sellers, map[string]interface{}{
						"FullName": string(seller["fullName"].(string)),
					})
				}
			}
			properties.lastMarketSale := models.LastMarketSale{
				Date: string(td["saleDateDerived"].(string)),
				RecordingDate: string(td["saleRecordingDateDerived"].(string)),
				Amount: int(td["saleAmount"].(float64)),
				DocumentTypeCode: string(td["saleDocumentTypeCode"].(string)),
				DocumentNumber: string(td["saleDocumentNumber"].(string)),
				BookNumber: string(td["saleBookNumber"].(string)),
				PageNumber: string(td["salePageNumber"].(string)),
				MultiOrSplitParcelCode: string(td["multiOrSplitParcelCode"].(string)),
				IsMortgagePurchase: bool(td["isMortgagePurchase"].(bool)),
				IsResale: bool(td["isResale"].(bool)),
				Buyers: buyers,
				Sellers: sellers,
				TitleCompany: models.TitleCompany{
					Name: string(tc["name"].(string)),
					Code: string(tc["code"].(string)),
				},
			}
		}
	}

		return property, nil
	}
