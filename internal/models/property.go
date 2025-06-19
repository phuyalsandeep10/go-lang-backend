package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Property struct {
	ID                 primitive.ObjectID `json:"_id" bson:"_id"`
	PropertyID         string             `json:"propertyId" bson:"propertyId"`
	AVMPropertyID      string             `json:"avmPropertyId" bson:"avmPropertyId"`
	Address            Address            `json:"address" bson:"address"`
	Location           Location           `json:"location" bson:"location"`
	Lot                Lot                `json:"lot" bson:"lot"`
	LandUseAndZoning   LandUseAndZoning   `json:"landUseAndZoning" bson:"landUseAndZoning"`
	Utilities          Utilities          `json:"utilities" bson:"utilities"`
	Building           Building           `json:"building" bson:"building"`
	Ownership          Ownership          `json:"ownership" bson:"ownership"`
	TaxAssessment      TaxAssessment      `json:"taxAssessment" bson:"taxAssessment"`
	LastMarketSale     LastMarketSale     `json:"lastMarketSale" bson:"lastMarketSale"`
}

type Address struct {
	StreetAddress      string             `json:"streetAddress" bson:"streetAddress"`
	StreetAddressParsed StreetAddressParsed `json:"streetAddressParsed" bson:"streetAddressParsed"`
	City               string             `json:"city" bson:"city"`
	State              string             `json:"state" bson:"state"`
	ZipCode            string             `json:"zipCode" bson:"zipCode"`
	ZipPlus4           string             `json:"zipPlus4" bson:"zipPlus4"`
	County             string             `json:"county" bson:"county"`
	CarrierRoute       string             `json:"carrierRoute" bson:"carrierRoute"`
}

type StreetAddressParsed struct {
	HouseNumber      string `json:"houseNumber" bson:"houseNumber"`
	StreetName       string `json:"streetName" bson:"streetName"`
	StreetNameSuffix string `json:"streetNameSuffix" bson:"streetNameSuffix"`
}

type Location struct {
	Coordinates Coordinates `json:"coordinates" bson:"coordinates"`
	Legal       Legal       `json:"legal" bson:"legal"`
	CBSA        CBSA        `json:"cbsa" bson:"cbsa"`
	CensusTract CensusTract `json:"censusTract" bson:"censusTract"`
}

type Coordinates struct {
	Parcel CoordinatesPoint `json:"parcel" bson:"parcel"`
	Block  CoordinatesPoint `json:"block" bson:"block"`
}

type CoordinatesPoint struct {
	Lat float64 `json:"lat" bson:"lat"`
	Lng float64 `json:"lng" bson:"lng"`
}

type Legal struct {
	SubdivisionName          string `json:"subdivisionName" bson:"subdivisionName"`
	SubdivisionPlatBookNumber string `json:"subdivisionPlatBookNumber" bson:"subdivisionPlatBookNumber"`
	SubdivisionPlatPageNumber string `json:"subdivisionPlatPageNumber" bson:"subdivisionPlatPageNumber"`
}

type CBSA struct {
	Code string `json:"code" bson:"code"`
	Type string `json:"type" bson:"type"`
}

type CensusTract struct {
	ID string `json:"id" bson:"id"`
}

type Lot struct {
	AreaAcres          float64 `json:"areaAcres" bson:"areaAcres"`
	AreaSquareFeet     int     `json:"areaSquareFeet" bson:"areaSquareFeet"`
	AreaSquareFeetUsable int   `json:"areaSquareFeetUsable" bson:"areaSquareFeetUsable"`
	TopographyType     string  `json:"topographyType" bson:"topographyType"`
}

type LandUseAndZoning struct {
	PropertyTypeCode      string `json:"propertyTypeCode" bson:"propertyTypeCode"`
	LandUseCode           string `json:"landUseCode" bson:"landUseCode"`
	StateLandUseCode      string `json:"stateLandUseCode" bson:"stateLandUseCode"`
	StateLandUseDescription string `json:"stateLandUseDescription" bson:"stateLandUseDescription"`
}

type Utilities struct {
	FuelTypeCode          string `json:"fuelTypeCode" bson:"fuelTypeCode"`
	ElectricityWiringTypeCode string `json:"electricityWiringTypeCode" bson:"electricityWiringTypeCode"`
	SewerTypeCode         string `json:"sewerTypeCode" bson:"sewerTypeCode"`
	UtilitiesTypeCode     string `json:"utilitiesTypeCode" bson:"utilitiesTypeCode"`
	WaterTypeCode         string `json:"waterTypeCode" bson:"waterTypeCode"`
}

type Building struct {
	Summary  BuildingSummary  `json:"summary" bson:"summary"`
	Details  BuildingDetails  `json:"details" bson:"details"`
}

type BuildingSummary struct {
	BuildingsCount      int `json:"buildingsCount" bson:"buildingsCount"`
	BathroomsCount      int `json:"bathroomsCount" bson:"bathroomsCount"`
	FullBathroomsCount  int `json:"fullBathroomsCount" bson:"fullBathroomsCount"`
	HalfBathroomsCount  int `json:"halfBathroomsCount" bson:"halfBathroomsCount"`
	BathroomFixturesCount int `json:"bathroomFixturesCount" bson:"bathroomFixturesCount"`
	FireplacesCount     int `json:"fireplacesCount" bson:"fireplacesCount"`
	LivingAreaSquareFeet int `json:"livingAreaSquareFeet" bson:"livingAreaSquareFeet"`
	TotalAreaSquareFeet int  `json:"totalAreaSquareFeet" bson:"totalAreaSquareFeet"`
}

type BuildingDetails struct {
	StructureID   StructureID   `json:"structureId" bson:"structureId"`
	Classification Classification `json:"classification" bson:"classification"`
	VerticalProfile VerticalProfile `json:"verticalProfile" bson:"verticalProfile"`
	Construction  Construction  `json:"construction" bson:"construction"`
	Exterior      Exterior      `json:"exterior" bson:"exterior"`
	Interior      Interior      `json:"interior" bson:"interior"`
}

type StructureID struct {
	SequenceNumber         int    `json:"sequenceNumber" bson:"sequenceNumber"`
	CompositeBuildingLinkageKey string `json:"compositeBuildingLinkageKey" bson:"compositeBuildingLinkageKey"`
	BuildingNumber         string `json:"buildingNumber" bson:"buildingNumber"`
}

type Classification struct {
	BuildingTypeCode string `json:"buildingTypeCode" bson:"buildingTypeCode"`
	GradeTypeCode    string `json:"gradeTypeCode" bson:"gradeTypeCode"`
}

type VerticalProfile struct {
	StoriesCount int `json:"storiesCount" bson:"storiesCount"`
}

type Construction struct {
	YearBuilt                int    `json:"yearBuilt" bson:"yearBuilt"`
	EffectiveYearBuilt       int    `json:"effectiveYearBuilt" bson:"effectiveYearBuilt"`
	BuildingQualityTypeCode  string `json:"buildingQualityTypeCode" bson:"buildingQualityTypeCode"`
	FrameTypeCode            string `json:"frameTypeCode" bson:"frameTypeCode"`
	FoundationTypeCode       string `json:"foundationTypeCode" bson:"foundationTypeCode"`
	BuildingImprovementConditionCode string `json:"buildingImprovementConditionCode" bson:"buildingImprovementConditionCode"`
}

type Exterior struct {
	Patios Patios `json:"patios" bson:"patios"`
	Porches Porches `json:"porches" bson:"porches"`
	Pool   Pool   `json:"pool" bson:"pool"`
	Walls  Walls  `json:"walls" bson:"walls"`
	Roof   Roof   `json:"roof" bson:"roof"`
}

type Patios struct {
	Count         int    `json:"count" bson:"count"`
	TypeCode      string `json:"typeCode" bson:"typeCode"`
	AreaSquareFeet int    `json:"areaSquareFeet" bson:"areaSquareFeet"`
}

type Porches struct {
	Count         int    `json:"count" bson:"count"`
	TypeCode      string `json:"typeCode" bson:"typeCode"`
	AreaSquareFeet int    `json:"areaSquareFeet" bson:"areaSquareFeet"`
}

type Pool struct {
	TypeCode      string `json:"typeCode" bson:"typeCode"`
	AreaSquareFeet int    `json:"areaSquareFeet" bson:"areaSquareFeet"`
}

type Walls struct {
	TypeCode string `json:"typeCode" bson:"typeCode"`
}

type Roof struct {
	TypeCode     string `json:"typeCode" bson:"typeCode"`
	CoverTypeCode string `json:"coverTypeCode" bson:"coverTypeCode"`
}

type Interior struct {
	Area    InteriorArea `json:"area" bson:"area"`
	Walls   Walls        `json:"walls" bson:"walls"`
	Basement Basement     `json:"basement" bson:"basement"`
	Flooring Flooring     `json:"flooring" bson:"flooring"`
	Features Features     `json:"features" bson:"features"`
}

type InteriorArea struct {
	UniversalBuildingAreaSquareFeet int `json:"universalBuildingAreaSquareFeet" bson:"universalBuildingAreaSquareFeet"`
	LivingAreaSquareFeet           int `json:"livingAreaSquareFeet" bson:"livingAreaSquareFeet"`
	AboveGradeAreaSquareFeet       int `json:"aboveGradeAreaSquareFeet" bson:"aboveGradeAreaSquareFeet"`
	GroundFloorAreaSquareFeet      int `json:"groundFloorAreaSquareFeet" bson:"groundFloorAreaSquareFeet"`
	BasementAreaSquareFeet         int `json:"basementAreaSquareFeet" bson:"basementAreaSquareFeet"`
	UnfinishedBasementAreaSquareFeet int `json:"unfinishedBasementAreaSquareFeet" bson:"unfinishedBasementAreaSquareFeet"`
	AboveGroundFloorAreaSquareFeet  int `json:"aboveGroundFloorAreaSquareFeet" bson:"aboveGroundFloorAreaSquareFeet"`
	BuildingAdditionsAreaSquareFeet int `json:"buildingAdditionsAreaSquareFeet" bson:"buildingAdditionsAreaSquareFeet"`
}

type Basement struct {
	TypeCode string `json:"typeCode" bson:"typeCode"`
}

type Flooring struct {
	CoverTypeCode string `json:"coverTypeCode" bson:"coverTypeCode"`
}

type Features struct {
	AirConditioning AirConditioning `json:"airConditioning" bson:"airConditioning"`
	Heating        Heating         `json:"heating" bson:"heating"`
	Fireplaces     Fireplaces      `json:"fireplaces" bson:"fireplaces"`
}

type AirConditioning struct {
	TypeCode string `json:"typeCode" bson:"typeCode"`
}

type Heating struct {
	TypeCode string `json:"typeCode" bson:"typeCode"`
}

type Fireplaces struct {
	TypeCode string `json:"typeCode" bson:"typeCode"`
	Count    int    `json:"count" bson:"count"`
}

type Ownership struct {
	CurrentOwners []Owner `json:"currentOwners" bson:"currentOwners"`
	RelationshipTypeCode string `json:"relationshipTypeCode" bson:"relationshipTypeCode"`
	OccupancyCode       string `json:"occupancyCode" bson:"occupancyCode"`
	MailingAddress      MailingAddress `json:"mailingAddress" bson:"mailingAddress"`
}

type Owner struct {
	SequenceNumber int    `json:"sequenceNumber" bson:"sequenceNumber"`
	FullName       string `json:"fullName" bson:"fullName"`
	FirstName      string `json:"firstName" bson:"firstName"`
	MiddleName     string `json:"middleName" bson:"middleName"`
	LastName       string `json:"lastName" bson:"lastName"`
	IsCorporate    bool   `json:"isCorporate" bson:"isCorporate"`
}

type MailingAddress struct {
	StreetAddress string `json:"streetAddress" bson:"streetAddress"`
	City         string `json:"city" bson:"city"`
	State        string `json:"state" bson:"state"`
	ZipCode      string `json:"zipCode" bson:"zipCode"`
	CarrierRoute string `json:"carrierRoute" bson:"carrierRoute"`
}

type TaxAssessment struct {
	Year         int         `json:"year" bson:"year"`
	TotalTaxAmount int       `json:"totalTaxAmount" bson:"totalTaxAmount"`
	CountyTaxAmount int      `json:"countyTaxAmount" bson:"countyTaxAmount"`
	AssessedValue AssessedValue `json:"assessedValue" bson:"assessedValue"`
	TaxRoll       TaxRoll      `json:"taxRoll" bson:"taxRoll"`
	SchoolDistrict SchoolDistrict `json:"schoolDistrict" bson:"schoolDistrict"`
}

type AssessedValue struct {
	TotalValue            int `json:"totalValue" bson:"totalValue"`
	LandValue             int `json:"landValue" bson:"landValue"`
	ImprovementValue      int `json:"improvementValue" bson:"improvementValue"`
	ImprovementValuePercentage int `json:"improvementValuePercentage" bson:"improvementValuePercentage"`
}

type TaxRoll struct {
	LastAssessorUpdateDate string `json:"lastAssessorUpdateDate" bson:"lastAssessorUpdateDate"`
	CertificationDate     string `json:"certificationDate" bson:"certificationDate"`
}

type SchoolDistrict struct {
	Code string `json:"code" bson:"code"`
	Name string `json:"name" bson:"name"`
}

type LastMarketSale struct {
	Date               string         `json:"date" bson:"date"`
	RecordingDate      string         `json:"recordingDate" bson:"recordingDate"`
	Amount             int            `json:"amount" bson:"amount"`
	DocumentTypeCode   string         `json:"documentTypeCode" bson:"documentTypeCode"`
	DocumentNumber     string         `json:"documentNumber" bson:"documentNumber"`
	BookNumber         string         `json:"bookNumber" bson:"bookNumber"`
	PageNumber         string         `json:"pageNumber" bson:"pageNumber"`
	MultiOrSplitParcelCode string     `json:"multiOrSplitParcelCode" bson:"multiOrSplitParcelCode"`
	IsMortgagePurchase bool           `json:"isMortgagePurchase" bson:"isMortgagePurchase"`
	IsResale           bool           `json:"isResale" bson:"isResale"`
	Buyers             []Buyer        `json:"buyers" bson:"buyers"`
	Sellers            []Seller       `json:"sellers" bson:"sellers"`
	TitleCompany       TitleCompany   `json:"titleCompany" bson:"titleCompany"`
}

type Buyer struct {
	FullName             string `json:"fullName" bson:"fullName"`
	LastName             string `json:"lastName" bson:"lastName"`
	FirstNameAndMiddleInitial string `json:"firstNameAndMiddleInitial" bson:"firstNameAndMiddleInitial"`
}

type Seller struct {
	FullName string `json:"fullName" bson:"fullName"`
}

type TitleCompany struct {
	Name string `json:"name" bson:"name"`
	Code string `json:"code" bson:"code"`
}

type SearchRequest struct {
	Search        string `json:"search" bson:"search"`
	StreetAddress string `json:"streetAddress" bson:"streetAddress"`
	City          string `json:"city" bson:"city"`
	State         string `json:"state" bson:"state"`
	ZipCode       string `json:"zipCode" bson:"zipCode"`
	Offset        int    `json:"offset" bson:"offset"`
	Limit         int    `json:"limit" bson:"limit"`
}

type PropertyResponse struct {
	Property *Property `json:"property" bson:"property"`
}

type PaginationMeta struct {
	Total  int64   `json:"total" bson:"total"`
	Offset int     `json:"offset" bson:"offset"`
	Limit  int     `json:"limit" bson:"limit"`
	Next   *string `json:"next,omitempty" bson:"next,omitempty"`
	Prev   *string `json:"prev,omitempty" bson:"prev,omitempty"`
}

type PaginatedPropertiesResponse struct {
	Data     []PropertyResponse `json:"data" bson:"data"`
	Metadata PaginationMeta     `json:"metadata" bson:"metadata"`
}
