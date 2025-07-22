
package corelogic

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"

    "homeinsight-properties/pkg/logger"
)

// structure for the detail task payload.
type DetailRequest struct {
    Task   string `json:"task"`
    ClipId string `json:"clipId"`
}

// retrieve detailed property information using the cloud function proxy.
func (c *Client) GetPropertyDetails(token, propertyId string) (map[string]interface{}, error) {
    proxyURL := os.Getenv("CORELOGIC_PROXY_URL")
    if proxyURL == "" {
        return nil, fmt.Errorf("CORELOGIC_PROXY_URL environment variable is not set")
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

    // Set headers (Authorization and Content-Type)
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")

    // Send the HTTP request
    resp, err := c.httpClient.Do(req)
    if err != nil {
        logger.GlobalLogger.Errorf("Failed to send detail request to proxy: url=%s, error=%v", proxyURL, err)
        return nil, fmt.Errorf("failed to send detail request to proxy: %v", err)
    }
    defer resp.Body.Close()

    // Read the response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        logger.GlobalLogger.Errorf("Failed to read detail response body: url=%s, status=%s, error=%v", proxyURL, resp.Status, err)
        return nil, fmt.Errorf("failed to read response body: %v", err)
    }

    // Check the response status
    if resp.StatusCode != http.StatusOK {
        logger.GlobalLogger.Errorf("Detail request to proxy failed: url=%s, status=%s, response=%s", proxyURL, resp.Status, string(body))
        return nil, fmt.Errorf("failed to get property details: %s, response: %s", resp.Status, string(body))
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

// retrieve detailed property information using clip.
func (c *Client) GetPropertyDetailsByClip(token, clip string) (map[string]interface{}, error) {
    return c.GetPropertyDetails(token, clip)
}

// retrieve detailed property information using v1PropertyId.
func (c *Client) GetPropertyDetailsByV1PropertyId(token, v1PropertyId string) (map[string]interface{}, error) {
    return c.GetPropertyDetails(token, v1PropertyId)
}
