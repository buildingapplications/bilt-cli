package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "http://localhost:3000"

// Client is the Bilt API client. Uses API key authentication (bilt_live_... tokens).
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new API client with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		BaseURL: defaultBaseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProjectDetail represents full project details from GET /api/cli/projects/:id.
type ProjectDetail struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	GitURL      string `json:"git_url"`
	CloneURL    string `json:"clone_url"`
	BundleID    string `json:"bundle_id"`
	Visibility  string `json:"visibility"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// AuthExchangeResponse is returned by POST /api/cli/auth/exchange.
type AuthExchangeResponse struct {
	APIKey string `json:"api_key"`
	Name   string `json:"name,omitempty"`
	Email  string `json:"email,omitempty"`
}

// APIError represents a structured error from the API.
type APIError struct {
	Error      string `json:"error"`
	RetryAfter int    `json:"retry_after,omitempty"`
}

// GetProject returns full details for a specific project, including clone URL.
func (c *Client) GetProject(id string) (*ProjectDetail, error) {
	var project ProjectDetail
	if err := c.get("/api/cli/projects/"+id, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// ExchangeToken exchanges a one-time token for an API key. No API key required.
func (c *Client) ExchangeToken(token string) (*AuthExchangeResponse, error) {
	body := strings.NewReader(fmt.Sprintf(`{"token":"%s"}`, token))
	req, err := http.NewRequest("POST", c.BaseURL+"/api/cli/auth/exchange", body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var resp AuthExchangeResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) get(path string, result interface{}) error {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setAuth(req)
	return c.doJSON(req, result)
}

func (c *Client) doJSON(req *http.Request, result interface{}) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid or expired API key")
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		var apiErr APIError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.RetryAfter > 0 {
			return fmt.Errorf("rate limited — try again in %d seconds", apiErr.RetryAfter)
		}
		return fmt.Errorf("rate limited — try again later")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
			return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, apiErr.Error)
		}
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
	}
	return nil
}

func (c *Client) setAuth(req *http.Request) {
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
}
