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

// TokenResponse represents the OAuth token response from CoreLogic
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

// isTokenValid checks if the current token is valid and unexpired
func (c *Client) isTokenValid() bool {
	return c.token != "" && time.Now().Before(c.tokenExpiry)
}

// buildTokenRequest constructs the HTTP request for the token endpoint
func (c *Client) buildTokenRequest(tokenURL string) (*http.Request, error) {
	req, err := http.NewRequest("POST", tokenURL, nil)
	if err != nil {
		logger.GlobalLogger.Errorf("Failed to create token request: url=%s, error=%v", tokenURL, err)
		return nil, fmt.Errorf("failed to create token request: %v", err)
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

// executeTokenRequest sends the HTTP request with retry logic
func (c *Client) executeTokenRequest(req *http.Request, tokenURL string, maxRetries int) (*http.Response, error) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			logger.GlobalLogger.Errorf("Failed to send token request (attempt %d/%d): url=%s, error=%v", attempt, maxRetries, tokenURL, err)
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to send token request after %d attempts: %v", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			logger.GlobalLogger.Errorf("Token request failed (attempt %d/%d): url=%s, status=%s, response=%s", attempt, maxRetries, tokenURL, resp.Status, string(body))
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to get token after %d attempts: %s, response: %s", maxRetries, resp.Status, string(body))
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("failed to get token: max retries exceeded")
}

// parseTokenResponse decodes the HTTP response into a TokenResponse
func (c *Client) parseTokenResponse(resp *http.Response, tokenURL string) (TokenResponse, error) {
	var tokenResp TokenResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.GlobalLogger.Errorf("Failed to read token response body: url=%s, status=%s, error=%v", tokenURL, resp.Status, err)
		return tokenResp, fmt.Errorf("failed to read token response body: %s", resp.Status)
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		logger.GlobalLogger.Errorf("Failed to decode token response: url=%s, response=%s, error=%v", tokenURL, string(body), err)
		return tokenResp, fmt.Errorf("failed to decode token response: %v", err)
	}
	return tokenResp, nil
}

// updateTokenState updates the client's token and expiry time
func (c *Client) updateTokenState(tokenResp TokenResponse, tokenURL string) error {
	expiresIn, err := strconv.Atoi(tokenResp.ExpiresIn)
	if err != nil {
		logger.GlobalLogger.Errorf("Failed to parse expires_in as integer: url=%s, expires_in=%s, error=%v", tokenURL, tokenResp.ExpiresIn, err)
		return fmt.Errorf("failed to parse expires_in: %v", err)
	}
	c.token = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
	logger.GlobalLogger.Printf("Successfully retrieved CoreLogic token: expires_in=%d seconds", expiresIn)
	return nil
}

// getToken retrieves or refreshes the access token
func (c *Client) getToken() (string, error) {
	if c.isTokenValid() {
		return c.token, nil
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	tokenURL := "https://api-prod.corelogic.com/oauth/token?" + data.Encode()
	maxRetries := 3

	req, err := c.buildTokenRequest(tokenURL)
	if err != nil {
		return "", err
	}

	resp, err := c.executeTokenRequest(req, tokenURL, maxRetries)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tokenResp, err := c.parseTokenResponse(resp, tokenURL)
	if err != nil {
		return "", err
	}

	if err := c.updateTokenState(tokenResp, tokenURL); err != nil {
		return "", err
	}

	return c.token, nil
}
