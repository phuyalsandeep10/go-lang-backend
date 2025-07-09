package corelogic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"homeinsight-properties/pkg/logger"
)

// GetPropertyDetails retrieves detailed property information using clip or propertyId
func (c *Client) GetPropertyDetails(propertyId string) (map[string]interface{}, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, err
	}

	detailsURL := fmt.Sprintf("https://property.corelogicapi.com/v2/properties/%s/property-detail", propertyId)
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", detailsURL, nil)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to create property details request (attempt %d/%d): url=%s, error=%v", attempt, maxRetries, detailsURL, err)
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to create property details request after %d attempts: %v", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("x-developer-email", c.developerEmail)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to send property details request (attempt %d/%d): url=%s, error=%v", attempt, maxRetries, detailsURL, err)
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to send property details request after %d attempts: %v", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to read property details response body (attempt %d/%d): url=%s, status=%s, error=%v", attempt, maxRetries, detailsURL, resp.Status, err)
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to read response body after %d attempts: %v", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			logger.GlobalLogger.Errorf("Property details request failed (attempt %d/%d): url=%s, status=%s, response=%s", attempt, maxRetries, detailsURL, resp.Status, string(body))
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to get property details after %d attempts: %s, response: %s", maxRetries, resp.Status, string(body))
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		var details map[string]interface{}
		if err := json.Unmarshal(body, &details); err != nil {
			logger.GlobalLogger.Errorf("Failed to decode property details response (attempt %d/%d): url=%s, response=%s, error=%v", attempt, maxRetries, detailsURL, string(body), err)
			return nil, fmt.Errorf("failed to decode property details response: %v", err)
		}

		logger.GlobalLogger.Printf("Property details retrieved successfully for property ID: %s", propertyId)
		return details, nil
	}

	return nil, fmt.Errorf("failed to get property details: max retries exceeded")
}

// GetPropertyDetailsByClip retrieves detailed property information using clip
func (c *Client) GetPropertyDetailsByClip(clip string) (map[string]interface{}, error) {
	return c.GetPropertyDetails(clip)
}

// GetPropertyDetailsByV1PropertyId retrieves detailed property information using v1PropertyId
func (c *Client) GetPropertyDetailsByV1PropertyId(v1PropertyId string) (map[string]interface{}, error) {
	return c.GetPropertyDetails(v1PropertyId)
}
