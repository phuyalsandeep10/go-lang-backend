package corelogic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"homeinsight-properties/pkg/logger"
)

// structure for the detail task payload.
type DetailRequest struct {
	Task   string `json:"task"`
	ClipId string `json:"clipId"`
}

// retrieve detailed property information using the cloud function proxy.
func (c *Client) GetPropertyDetails(propertyId string) (map[string]interface{}, error) {
	proxyURL := os.Getenv("CORELOGIC_PROXY_URL")
	if proxyURL == "" {
		return nil, fmt.Errorf("CORELOGIC_PROXY_URL environment variable is not set")
	}

	// Get the authentication token
	token, err := c.getToken()
	if err != nil {
		logger.GlobalLogger.Errorf("Failed to get token: error=%v", err)
		return nil, err
	}

	// Create the request body for the detail task
	requestBody := DetailRequest{
		Task:   "detail",
		ClipId: propertyId,
	}

	// Marshal the request body to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		logger.GlobalLogger.Errorf("Failed to marshal detail request body: error=%v", err)
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Create the HTTP POST request
	req, err := http.NewRequest("POST", proxyURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		logger.GlobalLogger.Errorf("Failed to create detail request: error=%v", err)
		return nil, err
	}

	// Set headers (Authorization and Content-Type;)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Retry loop for HTTP request
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to send detail request to proxy (attempt %d/%d): url=%s, error=%v", attempt, maxRetries, proxyURL, err)
		 if attempt == maxRetries {
			return nil, fmt.Errorf("failed to send detail request to proxy after %d attempts: %v", maxRetries, err)
		 }
		 time.Sleep(time.Duration(attempt) * time.Second)
		 continue
		}
		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to read detail response body (attempt %d/%d): url=%s, status=%s, error=%v", attempt, maxRetries, proxyURL, resp.Status, err)
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to read response body after %d attempts: %v", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		// Check the response status
		if resp.StatusCode != http.StatusOK {
			logger.GlobalLogger.Errorf("Detail request to proxy failed (attempt %d/%d): url=%s, status=%s, response=%s", attempt, maxRetries, proxyURL, resp.Status, string(body))
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to get property details after %d attempts: %s, response: %s", maxRetries, resp.Status, string(body))
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		// Parse the response
		var details map[string]interface{}
		if err := json.Unmarshal(body, &details); err != nil {
			logger.GlobalLogger.Errorf("Failed to decode detail response: url=%s, response=%s, error=%v", proxyURL, string(body), err)
			return nil, fmt.Errorf("failed to decode property details response: %v", err)
		}

		logger.GlobalLogger.Printf("Property details retrieved successfully for property ID: %s", propertyId)
		return details, nil
	}

	return nil, fmt.Errorf("failed to get property details: max retries exceeded")
}

// retrieve detailed property information using clip.
func (c *Client) GetPropertyDetailsByClip(clip string) (map[string]interface{}, error) {
	return c.GetPropertyDetails(clip)
}

// retrieve detailed property information using v1PropertyId.
func (c *Client) GetPropertyDetailsByV1PropertyId(v1PropertyId string) (map[string]interface{}, error) {
	return c.GetPropertyDetails(v1PropertyId)
}
