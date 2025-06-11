package services

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"homeinsight-properties/internal/models"
	"homeinsight-properties/pkg/database"
)

// PropertyService manages property-related operations using the database.
type PropertyService struct {
}

// NewPropertyService creates a new instance of PropertyService.
func NewPropertyService() *PropertyService {
	return &PropertyService{}
}

// CreateProperty adds a new property to the database.
func (s *PropertyService) CreateProperty(property *models.Property) error {
	// Validate required fields
	if property.Name == "" {
		return errors.New("property name is required")
	}

	// Generate a unique ID using UUID
	property.ID = uuid.New().String()

	// SQL query to insert the property
	query := "INSERT INTO properties (id, name, description, price) VALUES (?, ?, ?, ?)"
	_, err := database.DB.Exec(query, property.ID, property.Name, property.Description, property.Price)
	if err != nil {
		return fmt.Errorf("failed to insert property: %v", err)
	}

	return nil
}

// GetAllProperties retrieves all properties from the database.
func (s *PropertyService) GetAllProperties() ([]models.Property, error) {
	// SQL query to select all properties
	rows, err := database.DB.Query("SELECT id, name, description, price FROM properties")
	if err != nil {
		return nil, fmt.Errorf("failed to query properties: %v", err)
	}
	defer rows.Close()

	// Collect properties into a slice
	var properties []models.Property
	for rows.Next() {
		var prop models.Property
		if err := rows.Scan(&prop.ID, &prop.Name, &prop.Description, &prop.Price); err != nil {
			return nil, fmt.Errorf("failed to scan property: %v", err)
		}
		properties = append(properties, prop)
	}

	// Check for errors encountered during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return properties, nil
}
