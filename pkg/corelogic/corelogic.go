package corelogic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"homeinsight-properties/pkg/logger"
)

// Client manages CoreLogic API authentication and requests
type Client struct {
	username       string
	password       string
	developerEmail string
	baseURL        string
	token          string
	tokenExpiry    time.Time
	httpClient     *http.Client
}

// TokenResponse represents the OAuth token response from CoreLogic
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

// NewClient creates a new CoreLogic client
func NewClient(username, password, baseURL, developerEmail string) *Client {
	return &Client{
		username:       username,
		password:       password,
		developerEmail: developerEmail,
		baseURL:        baseURL,
		httpClient:     &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetDeveloperEmail returns the developer email for the client
func (c *Client) GetDeveloperEmail() string {
	return c.developerEmail
}

// GetUsername returns the username for the client
func (c *Client) GetUsername() string {
	return c.username
}

// getToken retrieves or refreshes the access token
func (c *Client) getToken() (string, error) {
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	tokenURL := "https://api-prod.corelogic.com/oauth/token?" + data.Encode()
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", tokenURL, nil)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to create token request (attempt %d/%d): url=%s, error=%v", attempt, maxRetries, tokenURL, err)
			return "", fmt.Errorf("failed to create token request: %v", err)
		}

		req.SetBasicAuth(c.username, c.password)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to send token request (attempt %d/%d): url=%s, error=%v", attempt, maxRetries, tokenURL, err)
			if attempt == maxRetries {
				return "", fmt.Errorf("failed to send token request after %d attempts: %v", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to read token response body (attempt %d/%d): url=%s, status=%s, error=%v", attempt, maxRetries, tokenURL, resp.Status, err)
			if attempt == maxRetries {
				return "", fmt.Errorf("failed to read token response body: %s", resp.Status)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			logger.GlobalLogger.Errorf("Token request failed (attempt %d/%d): url=%s, status=%s, response=%s", attempt, maxRetries, tokenURL, resp.Status, string(body))
			if attempt == maxRetries {
				return "", fmt.Errorf("failed to get token after %d attempts: %s, response: %s", maxRetries, resp.Status, string(body))
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		var tokenResp TokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			logger.GlobalLogger.Errorf("Failed to decode token response (attempt %d/%d): url=%s, response=%s, error=%v", attempt, maxRetries, tokenURL, string(body), err)
			return "", fmt.Errorf("failed to decode token response: %v", err)
		}

		expiresIn, err := strconv.Atoi(tokenResp.ExpiresIn)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to parse expires_in as integer: url=%s, expires_in=%s, error=%v", tokenURL, tokenResp.ExpiresIn, err)
			return "", fmt.Errorf("failed to parse expires_in: %v", err)
		}

		c.token = tokenResp.AccessToken
		c.tokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
		logger.GlobalLogger.Printf("Successfully retrieved CoreLogic token: expires_in=%d seconds", expiresIn)
		return c.token, nil
	}

	return "", fmt.Errorf("failed to get token: max retries exceeded")
}
