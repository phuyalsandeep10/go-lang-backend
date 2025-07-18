package corelogic

import (
	"net/http"
	"time"

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
