package api

// AuthMethod represents the authentication method used by the client.
type AuthMethod string

const (
	// AuthJWT authenticates using a JWT bearer token.
	AuthJWT AuthMethod = "jwt"
	// AuthAPIKey authenticates using an API key.
	AuthAPIKey AuthMethod = "apikey"
	// AuthNone disables authentication.
	AuthNone AuthMethod = "none"
)

// SetAuth configures the authentication header on the client.
func (c *Client) SetAuth(method AuthMethod, token, apiKey string) {
	// Stub implementation
	_ = method
	_ = token
	_ = apiKey
}
