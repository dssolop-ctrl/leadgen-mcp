package common

import "fmt"

// APIError represents an upstream API error (non-auth).
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, truncate(e.Message, 300))
}

// AuthError represents an authentication/authorization error.
type AuthError struct {
	StatusCode int
	Message    string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("auth error %d: %s", e.StatusCode, truncate(e.Message, 300))
}
