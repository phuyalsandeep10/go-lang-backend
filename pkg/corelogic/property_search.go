package corelogic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"homeinsight-properties/pkg/logger"
)

type PropertySearchResponse struct {
	Items []struct {
		Clip         string `json:"clip"`
		V1PropertyId string `json:"v1PropertyId"`
	} `json:"items"`
}

func (c *Client) SearchPropertyByAddress(street, city, state, zip string) (string, string, error) {
	token, err := c.getToken()
	if err != nil {
		logger.GlobalLogger.Errorf("Failed to get token: error=%v", err)
		return "", "", err
	}

	// Base URL
	baseURL := "https://property.corelogicapi.com/v2/properties/search"

	// Build query parameters
	var queryParts []string
	queryParts = append(queryParts, "streetAddress="+url.PathEscape(street))

	if city != "" {
		queryParts = append(queryParts, "city="+url.PathEscape(city))
	}

	if state != "" {
		queryParts = append(queryParts, "state="+url.PathEscape(state))
	}

	if zip != "" {
		queryParts = append(queryParts, "zipCode="+url.PathEscape(zip))
	}

	// Construct final URL
	searchURL := baseURL + "?" + strings.Join(queryParts, "&")

	// Retry loop for HTTP request
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", searchURL, nil)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to create property search request (attempt %d/%d): url=%s, error=%v", attempt, maxRetries, searchURL, err)
			if attempt == maxRetries {
				return "", "", fmt.Errorf("failed to create property search request after %d attempts: %v", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		// Set headers
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("x-developer-email", c.developerEmail)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to send property search request (attempt %d/%d): url=%s, error=%v", attempt, maxRetries, searchURL, err)
			if attempt == maxRetries {
				return "", "", fmt.Errorf("failed to send property search request after %d attempts: %v", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to read property search response body (attempt %d/%d): url=%s, status=%s, error=%v", attempt, maxRetries, searchURL, resp.Status, err)
			if attempt == maxRetries {
				return "", "", fmt.Errorf("failed to read response body after %d attempts: %v", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			logger.GlobalLogger.Errorf("Property search failed (attempt %d/%d): url=%s, status=%s, response=%s", attempt, maxRetries, searchURL, resp.Status, string(body))
			if attempt == maxRetries {
				return "", "", fmt.Errorf("search failed after %d attempts: %s, response: %s", maxRetries, resp.Status, string(body))
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		var searchResp PropertySearchResponse
		if err := json.Unmarshal(body, &searchResp); err != nil {
			logger.GlobalLogger.Errorf("Failed to decode property search response (attempt %d/%d): url=%s, response=%s, error=%v", attempt, maxRetries, searchURL, string(body), err)
			return "", "", fmt.Errorf("failed to decode search response: %v", err)
		}

		if len(searchResp.Items) == 0 {
			logger.GlobalLogger.Errorf("No property found: url=%s", searchURL)
			return "", "", fmt.Errorf("no property found for address: %s, %s, %s %s", street, city, state, zip)
		}

		return searchResp.Items[0].Clip, searchResp.Items[0].V1PropertyId, nil
	}

	logger.GlobalLogger.Errorf("Failed to search property: max retries exceeded for URL: %s", searchURL)
	return "", "", fmt.Errorf("failed to search property: max retries exceeded")
}
