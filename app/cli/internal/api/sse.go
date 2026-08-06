package api

import "net/http"

// SSEClient is a server-sent events client.
type SSEClient struct {
	baseURL    string
	topic      string
	httpClient *http.Client
}

// NewSSEClient creates a new SSE client for the given base URL and topic.
func NewSSEClient(baseURL, topic string) *SSEClient {
	return &SSEClient{
		baseURL:    baseURL,
		topic:      topic,
		httpClient: &http.Client{},
	}
}

// Connect establishes a connection to the SSE endpoint.
func (s *SSEClient) Connect() error {
	// Stub implementation
	return nil
}

// Events returns a channel of SSE events.
func (s *SSEClient) Events() <-chan SSEEvent {
	// Stub implementation
	return nil
}
