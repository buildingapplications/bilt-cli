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

// User represents the current authenticated user from GET /api/cli/me.
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Plan  string `json:"plan"`
}

// Project represents a Bilt project from GET /api/cli/projects.
type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	GitURL      string `json:"git_url"`
	Visibility  string `json:"visibility"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
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

// AuthStartResponse is returned by POST /api/cli/auth/start.
type AuthStartResponse struct {
	Code      string `json:"code"`
	ExpiresIn int    `json:"expires_in"`
}

// AuthPollResponse is returned by GET /api/cli/auth/poll.
type AuthPollResponse struct {
	Status string `json:"status"` // "pending" or "complete"
	APIKey string `json:"api_key,omitempty"`
	Name   string `json:"name,omitempty"`
	Email  string `json:"email,omitempty"`
}

// projectsResponse wraps the list response.
type projectsResponse struct {
	Projects []Project `json:"projects"`
}

// APIError represents a structured error from the API.
type APIError struct {
	Error      string `json:"error"`
	RetryAfter int    `json:"retry_after,omitempty"`
}

// StartAuth begins the device auth flow. No API key required.
func (c *Client) StartAuth() (*AuthStartResponse, error) {
	req, err := http.NewRequest("POST", c.BaseURL+"/api/cli/auth/start", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	var resp AuthStartResponse
	if err := c.doJSONNoAuth(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PollAuth checks the status of a device auth flow. No API key required.
func (c *Client) PollAuth(code string) (*AuthPollResponse, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/api/cli/auth/poll?code="+code, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	var resp AuthPollResponse
	if err := c.doJSONNoAuth(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Me returns the current authenticated user.
func (c *Client) Me() (*User, error) {
	var user User
	if err := c.get("/api/cli/me", &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ListProjects returns all projects for the authenticated user.
func (c *Client) ListProjects() ([]Project, error) {
	var resp projectsResponse
	if err := c.get("/api/cli/projects", &resp); err != nil {
		return nil, err
	}
	return resp.Projects, nil
}

// GetProject returns full details for a specific project, including clone URL.
func (c *Client) GetProject(id string) (*ProjectDetail, error) {
	var project ProjectDetail
	if err := c.get("/api/cli/projects/"+id, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

func (c *Client) get(path string, result interface{}) error {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setAuth(req)
	return c.doJSON(req, result)
}

func (c *Client) post(path string, body any, result any) error { //nolint:unused // kept for future use
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = strings.NewReader(string(data))
	}

	req, err := http.NewRequest("POST", c.BaseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.setAuth(req)
	return c.doJSON(req, result)
}

// doJSONNoAuth executes a request without auth headers (for public endpoints).
func (c *Client) doJSONNoAuth(req *http.Request, result interface{}) error {
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
		return fmt.Errorf("invalid or expired API key — run `bilt auth login` to update")
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
