
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

// search for a property by address using the cloud function proxy.
func (c *Client) SearchPropertyByAddress(token, street, city, state, zip string) (string, string, error) {
    proxyURL := os.Getenv("CORELOGIC_PROXY_URL")
    if proxyURL == "" {
        return "", "", fmt.Errorf("CORELOGIC_PROXY_URL environment variable is not set")
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

    // Set headers (Authorization and Content-Type)
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")

    // Send the HTTP request
    resp, err := c.httpClient.Do(req)
    if err != nil {
        logger.GlobalLogger.Errorf("Failed to send search request to proxy: url=%s, error=%v", proxyURL, err)
        return "", "", fmt.Errorf("failed to send search request to proxy: %v", err)
    }
    defer resp.Body.Close()

    // Read the response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        logger.GlobalLogger.Errorf("Failed to read search response body: url=%s, status=%s, error=%v", proxyURL, resp.Status, err)
        return "", "", fmt.Errorf("failed to read response body: %v", err)
    }

    // Check the response status
    if resp.StatusCode != http.StatusOK {
        return "", "", fmt.Errorf("search failed: %s, response: %s", resp.Status, string(body))
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
