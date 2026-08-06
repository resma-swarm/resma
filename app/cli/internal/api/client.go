package api

import (
	"net/http"
)

// Client is the RESMA API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
	apiKey     string
}

// NewClient creates a new API client for the given base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// Get performs a GET request to the given path.
func (c *Client) Get(path string, body ...any) (*http.Response, error) {
	return nil, nil
}

// Post performs a POST request to the given path with an optional body.
func (c *Client) Post(path string, body ...any) (*http.Response, error) {
	return nil, nil
}

// Put performs a PUT request to the given path with an optional body.
func (c *Client) Put(path string, body ...any) (*http.Response, error) {
	return nil, nil
}

// Patch performs a PATCH request to the given path with an optional body.
func (c *Client) Patch(path string, body ...any) (*http.Response, error) {
	return nil, nil
}

// Delete performs a DELETE request to the given path.
func (c *Client) Delete(path string, body ...any) (*http.Response, error) {
	return nil, nil
}
