package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxErrorResponseBytes = 64 << 10

// APIError preserves provider diagnostics for server-side logs without making
// callers parse formatted response bodies.
type APIError struct {
	Provider   string
	StatusCode int
	Code       string
	Message    string
	RequestID  string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s API request failed: status=%d code=%q message=%q body=%s", e.Provider, e.StatusCode, e.Code, e.Message, e.Body)
}

func NewAPIError(provider string, response *http.Response, body []byte) error {
	code, message := parseAPIErrorBody(body)
	requestID := response.Header.Get("x-request-id")
	if requestID == "" {
		requestID = response.Header.Get("request-id")
	}
	return &APIError{
		Provider:   provider,
		StatusCode: response.StatusCode,
		Code:       code,
		Message:    message,
		RequestID:  requestID,
		Body:       string(body),
	}
}

func readProviderResponse(response *http.Response) ([]byte, error) {
	if response.StatusCode == http.StatusOK {
		return io.ReadAll(response.Body)
	}
	return io.ReadAll(io.LimitReader(response.Body, maxErrorResponseBytes))
}

func parseAPIErrorBody(body []byte) (string, string) {
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
		Code    string `json:"code"`
		Type    string `json:"type"`
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return "", ""
	}
	code := firstNonEmpty(payload.Error.Code, payload.Error.Type, payload.Code, payload.Type)
	message := firstNonEmpty(payload.Error.Message, payload.Message, payload.Detail)
	return strings.TrimSpace(code), strings.TrimSpace(message)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
