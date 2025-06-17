package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type ExternalAPIClient struct{}

// NewExternalAPIClient creates a new ExternalAPIClient.
func NewExternalAPIClient() *ExternalAPIClient {
	return &ExternalAPIClient{}
}

// FetchPropertyData simulates fetching data from an external API by reading property3.json.
func (c *ExternalAPIClient) FetchPropertyData(ctx context.Context) (map[string]interface{}, error) {
	filePath := "data/property3.json"

	// Read the mock JSON file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read property3.json: %v", err)
	}

	// Parse JSON into map[string]interface{}
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return apiResponse, nil
}
