package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Property struct {
	ID                 primitive.ObjectID `json:"_id" bson:"_id"`
	PropertyID         string             `json:"propertyId" bson:"propertyId" validate:"required"`
	AVMPropertyID      string             `json:"avmPropertyId" bson:"avmPropertyId" validate:"required"`
	Address            Address            `json:"address" bson:"address" validate:"required,dive"`
	Location           Location           `json:"location" bson:"location"`
	Lot                Lot                `json:"lot" bson:"lot"`
	LandUseAndZoning   LandUseAndZoning   `json:"landUseAndZoning" bson:"landUseAndZoning"`
	Utilities          Utilities          `json:"utilities" bson:"utilities"`
	Building           Building           `json:"building" bson:"building"`
	Ownership          Ownership          `json:"ownership" bson:"ownership"`
	TaxAssessment      TaxAssessment      `json:"taxAssessment" bson:"taxAssessment"`
	LastMarketSale     LastMarketSale     `json:"lastMarketSale" bson:"lastMarketSale"`
	UpdatedAt          time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type Address struct {
	StreetAddress       string             `json:"streetAddress" bson:"streetAddress" validate:"required"`
	StreetAddressParsed StreetAddressParsed `json:"streetAddressParsed" bson:"streetAddressParsed"`
	City                string             `json:"city" bson:"city" validate:"required"`
	State               string             `json:"state" bson:"state" validate:"required,len=2"`
	ZipCode             string             `json:"zipCode" bson:"zipCode" validate:"required,regex=^[0-9]{5}$"`
	ZipPlus4            string             `json:"zipPlus4" bson:"zipPlus4"`
	County              string             `json:"county" bson:"county"`
	CarrierRoute        string             `json:"carrierRoute" bson:"carrierRoute"`
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
	Lat float64 `json:"lat" bson:"lat" validate:"gte=-90,lte=90"`
	Lng float64 `json:"lng" bson:"lng" validate:"gte=-180,lte=180"`
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
	AreaAcres          float64 `json:"areaAcres" bson:"areaAcres" validate:"gte=0"`
	AreaSquareFeet     int     `json:"areaSquareFeet" bson:"areaSquareFeet" validate:"gte=0"`
	AreaSquareFeetUsable int   `json:"areaSquareFeetUsable" bson:"areaSquareFeetUsable" validate:"gte=0"`
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
	Summary BuildingSummary `json:"summary" bson:"summary"`
	Details BuildingDetails `json:"details" bson:"details"`
}

type BuildingSummary struct {
	BuildingsCount      int `json:"buildingsCount" bson:"buildingsCount" validate:"gte=0"`
	BathroomsCount      int `json:"bathroomsCount" bson:"bathroomsCount" validate:"gte=0"`
	FullBathroomsCount  int `json:"fullBathroomsCount" bson:"fullBathroomsCount" validate:"gte=0"`
	HalfBathroomsCount  int `json:"halfBathroomsCount" bson:"halfBathroomsCount" validate:"gte=0"`
	BathroomFixturesCount int `json:"bathroomFixturesCount" bson:"bathroomFixturesCount" validate:"gte=0"`
	BedroomsCount       int `json:"bedroomsCount" bson:"bedroomsCount" validate:"gte=0"`
	KitchensCount       int `json:"kitchensCount" bson:"kitchensCount" validate:"gte=0"`
	FamilyRoomsCount    int `json:"familyRoomsCount" bson:"familyRoomsCount" validate:"gte=0"`
	LivingRoomsCount    int `json:"livingRoomsCount" bson:"livingRoomsCount" validate:"gte=0"`
	FireplacesCount     int `json:"fireplacesCount" bson:"fireplacesCount" validate:"gte=0"`
	LivingAreaSquareFeet int `json:"livingAreaSquareFeet" bson:"livingAreaSquareFeet" validate:"gte=0"`
	TotalAreaSquareFeet int `json:"totalAreaSquareFeet" bson:"totalAreaSquareFeet" validate:"gte=0"`
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
	SequenceNumber         int    `json:"sequenceNumber" bson:"sequenceNumber" validate:"gte=0"`
	CompositeBuildingLinkageKey string `json:"compositeBuildingLinkageKey" bson:"compositeBuildingLinkageKey"`
	BuildingNumber         string `json:"buildingNumber" bson:"buildingNumber"`
}

type Classification struct {
	BuildingTypeCode string `json:"buildingTypeCode" bson:"buildingTypeCode"`
	GradeTypeCode    string `json:"gradeTypeCode" bson:"gradeTypeCode"`
}

type VerticalProfile struct {
	StoriesCount int `json:"storiesCount" bson:"storiesCount" validate:"gte=0"`
}

type Construction struct {
	YearBuilt                int    `json:"yearBuilt" bson:"yearBuilt" validate:"gte=0"`
	EffectiveYearBuilt       int    `json:"effectiveYearBuilt" bson:"effectiveYearBuilt" validate:"gte=0"`
	BuildingQualityTypeCode  string `json:"buildingQualityTypeCode" bson:"buildingQualityTypeCode"`
	FrameTypeCode            string `json:"frameTypeCode" bson:"frameTypeCode"`
	FoundationTypeCode       string `json:"foundationTypeCode" bson:"foundationTypeCode"`
	BuildingImprovementConditionCode string `json:"buildingImprovementConditionCode" bson:"buildingImprovementConditionCode"`
}

type Exterior struct {
	Patios  Patios  `json:"patios" bson:"patios"`
	Porches Porches `json:"porches" bson:"porches"`
	Pool    Pool    `json:"pool" bson:"pool"`
	Walls   Walls   `json:"walls" bson:"walls"`
	Roof    Roof    `json:"roof" bson:"roof"`
	Parking Parking `json:"parking" bson:"parking"`
}

type Patios struct {
	Count         int    `json:"count" bson:"count" validate:"gte=0"`
	TypeCode      string `json:"typeCode" bson:"typeCode"`
	AreaSquareFeet int    `json:"areaSquareFeet" bson:"areaSquareFeet" validate:"gte=0"`
}

type Porches struct {
	Count         int    `json:"count" bson:"count" validate:"gte=0"`
	TypeCode      string `json:"typeCode" bson:"typeCode"`
	AreaSquareFeet int    `json:"areaSquareFeet" bson:"areaSquareFeet" validate:"gte=0"`
}

type Pool struct {
	TypeCode      string `json:"typeCode" bson:"typeCode"`
	AreaSquareFeet int    `json:"areaSquareFeet" bson:"areaSquareFeet" validate:"gte=0"`
}

type Walls struct {
	TypeCode string `json:"typeCode" bson:"typeCode"`
}

type Roof struct {
	TypeCode     string `json:"typeCode" bson:"typeCode"`
	CoverTypeCode string `json:"coverTypeCode" bson:"coverTypeCode"`
}

type Parking struct {
	TypeCode           string `json:"typeCode" bson:"typeCode"`
	ParkingSpacesCount int    `json:"parkingSpacesCount" bson:"parkingSpacesCount" validate:"gte=0"`
}

type Interior struct {
	Area    InteriorArea `json:"area" bson:"area"`
	Walls   Walls        `json:"walls" bson:"walls"`
	Basement Basement     `json:"basement" bson:"basement"`
	Flooring Flooring     `json:"flooring" bson:"flooring"`
	Features Features     `json:"features" bson:"features"`
}

type InteriorArea struct {
	UniversalBuildingAreaSquareFeet int `json:"universalBuildingAreaSquareFeet" bson:"universalBuildingAreaSquareFeet" validate:"gte=0"`
	LivingAreaSquareFeet           int `json:"livingAreaSquareFeet" bson:"livingAreaSquareFeet" validate:"gte=0"`
	AboveGradeAreaSquareFeet       int `json:"aboveGradeAreaSquareFeet" bson:"aboveGradeAreaSquareFeet" validate:"gte=0"`
	GroundFloorAreaSquareFeet      int `json:"groundFloorAreaSquareFeet" bson:"groundFloorAreaSquareFeet" validate:"gte=0"`
	BasementAreaSquareFeet         int `json:"basementAreaSquareFeet" bson:"basementAreaSquareFeet" validate:"gte=0"`
	UnfinishedBasementAreaSquareFeet int `json:"unfinishedBasementAreaSquareFeet" bson:"unfinishedBasementAreaSquareFeet" validate:"gte=0"`
	AboveGroundFloorAreaSquareFeet  int `json:"aboveGroundFloorAreaSquareFeet" bson:"aboveGroundFloorAreaSquareFeet" validate:"gte=0"`
	BuildingAdditionsAreaSquareFeet int `json:"buildingAdditionsAreaSquareFeet" bson:"buildingAdditionsAreaSquareFeet" validate:"gte=0"`
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
	Count    int    `json:"count" bson:"count" validate:"gte=0"`
}

type Ownership struct {
	CurrentOwners []Owner `json:"currentOwners" bson:"currentOwners"`
	RelationshipTypeCode string `json:"relationshipTypeCode" bson:"relationshipTypeCode"`
	OccupancyCode       string `json:"occupancyCode" bson:"occupancyCode"`
	MailingAddress      MailingAddress `json:"mailingAddress" bson:"mailingAddress"`
}

type Owner struct {
	SequenceNumber int    `json:"sequenceNumber" bson:"sequenceNumber" validate:"gte=0"`
	FullName       string `json:"fullName" bson:"fullName"`
	FirstName      string `json:"firstName" bson:"firstName"`
	MiddleName     string `json:"middleName" bson:"middleName"`
	LastName       string `json:"lastName" bson:"lastName"`
	IsCorporate    bool   `json:"isCorporate" bson:"isCorporate"`
}

type MailingAddress struct {
	StreetAddress string `json:"streetAddress" bson:"streetAddress"`
	City         string `json:"city" bson:"city"`
	State        string `json:"state" bson:"state" validate:"len=2"`
	ZipCode      string `json:"zipCode" bson:"zipCode" validate:"regex=^[0-9]{5}$"`
	CarrierRoute string `json:"carrierRoute" bson:"carrierRoute"`
}

type TaxAssessment struct {
	Year         int         `json:"year" bson:"year" validate:"gte=0"`
	TotalTaxAmount int       `json:"totalTaxAmount" bson:"totalTaxAmount" validate:"gte=0"`
	CountyTaxAmount int      `json:"countyTaxAmount" bson:"countyTaxAmount" validate:"gte=0"`
	AssessedValue AssessedValue `json:"assessedValue" bson:"assessedValue"`
	TaxRoll       TaxRoll      `json:"taxRoll" bson:"taxRoll"`
	SchoolDistrict SchoolDistrict `json:"schoolDistrict" bson:"schoolDistrict"`
}

type AssessedValue struct {
	TotalValue            int `json:"totalValue" bson:"totalValue" validate:"gte=0"`
	LandValue             int `json:"landValue" bson:"landValue" validate:"gte=0"`
	ImprovementValue      int `json:"improvementValue" bson:"improvementValue" validate:"gte=0"`
	ImprovementValuePercentage int `json:"improvementValuePercentage" bson:"improvementValuePercentage" validate:"gte=0,lte=100"`
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
	Amount             int            `json:"amount" bson:"amount" validate:"gte=0"`
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
	Search        string `json:"search" bson:"search" validate:"required"`
	StreetAddress string `json:"streetAddress" bson:"streetAddress"`
	City          string `json:"city" bson:"city"`
	State         string `json:"state" bson:"state"`
	ZipCode       string `json:"zipCode" bson:"zipCode"`
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
