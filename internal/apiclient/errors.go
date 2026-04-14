// Package apiclient is the CLI-side HTTP client that signs every request
// with the user's SSH key.
package apiclient

import "fmt"

// APIError is returned for any non-2xx response.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error %d %s: %s", e.Status, e.Code, e.Message)
}

// errorEnvelope mirrors server-side respond.ErrorEnvelope.
type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"error"`
}
