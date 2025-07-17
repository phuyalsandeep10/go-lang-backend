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

// structure for the search task payload.
type SearchRequest struct {
	Task        string `json:"task"`
	FullAddress string `json:"fullAddress"`
}

// structure of the search response from the proxy.
type PropertySearchResponse struct {
	Items []struct {
		Clip         string `json:"clip"`
		V1PropertyId string `json:"v1PropertyId"`
	} `json:"items"`
}

// searche for a property by address using the cloud function proxy.
func (c *Client) SearchPropertyByAddress(street, city, state, zip string) (string, string, error) {
	proxyURL := os.Getenv("CORELOGIC_PROXY_URL")
	if proxyURL == "" {
		return "", "", fmt.Errorf("CORELOGIC_PROXY_URL environment variable is not set")
	}

	// Get the authentication token
	token, err := c.getToken()
	if err != nil {
		logger.GlobalLogger.Errorf("Failed to get token: error=%v", err)
		return "", "", err
	}

	// Construct the full address in the format expected by the proxy: "street, city, state zip"
	fullAddress := fmt.Sprintf("%s, %s, %s %s", street, city, state, zip)
	requestBody := SearchRequest{
		Task:        "search",
		FullAddress: fullAddress,
	}

	// Marshal the request body to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		logger.GlobalLogger.Errorf("Failed to marshal search request body: error=%v", err)
		return "", "", fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Create the HTTP POST request
	req, err := http.NewRequest("POST", proxyURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		logger.GlobalLogger.Errorf("Failed to create search request: error=%v", err)
		return "", "", err
	}

	// Set headers (Authorization and Content-Type;)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Retry loop for HTTP request
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to send search request to proxy (attempt %d/%d): url=%s, error=%v", attempt, maxRetries, proxyURL, err)
			if attempt == maxRetries {
				return "", "", fmt.Errorf("failed to send search request to proxy after %d attempts: %v", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to read search response body (attempt %d/%d): url=%s, status=%s, error=%v", attempt, maxRetries, proxyURL, resp.Status, err)
			if attempt == maxRetries {
				return "", "", fmt.Errorf("failed to read response body after %d attempts: %v", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		// Check the response status
		if resp.StatusCode != http.StatusOK {
			logger.GlobalLogger.Errorf("Search request to proxy failed (attempt %d/%d): url=%s, status=%s, response=%s", attempt, maxRetries, proxyURL, resp.Status, string(body))
			if attempt == maxRetries {
				return "", "", fmt.Errorf("search failed after %d attempts: %s, response: %s", maxRetries, resp.Status, string(body))
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		// Parse the response
		var searchResp PropertySearchResponse
		if err := json.Unmarshal(body, &searchResp); err != nil {
			logger.GlobalLogger.Errorf("Failed to decode search response: url=%s, response=%s, error=%v", proxyURL, string(body), err)
			return "", "", fmt.Errorf("failed to decode search response: %v", err)
		}

		if len(searchResp.Items) == 0 {
			logger.GlobalLogger.Errorf("No property found: fullAddress=%s", fullAddress)
			return "", "", fmt.Errorf("no property found for address: %s", fullAddress)
		}

		return searchResp.Items[0].Clip, searchResp.Items[0].V1PropertyId, nil
	}

	logger.GlobalLogger.Errorf("Failed to search property: max retries exceeded for fullAddress: %s", fullAddress)
	return "", "", fmt.Errorf("failed to search property: max retries exceeded")
}
