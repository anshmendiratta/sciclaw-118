package agent

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type providerFailureError struct {
	cause   error
	error   bus.OutboundError
	message string
}

func (e *providerFailureError) Error() string       { return e.cause.Error() }
func (e *providerFailureError) Unwrap() error       { return e.cause }
func (e *providerFailureError) UserMessage() string { return e.message }

func newProviderFailureError(cause error, referenceID string) error {
	message, details := classifyProviderFailure(cause)
	return &providerFailureError{
		cause:   cause,
		message: message + "\n\nReference ID: `" + referenceID + "`",
		error: bus.OutboundError{
			TechnicalDetails: details,
			ReferenceID:      referenceID,
		},
	}
}

func providerErrorReference(turnID string) string {
	return "ERR-" + strings.TrimPrefix(turnID, "turn-")
}

// UserError returns the safe presentation attached to an error, if any.
func UserError(err error) (string, *bus.OutboundError, bool) {
	var providerErr *providerFailureError
	if errors.As(err, &providerErr) {
		return providerErr.message, &providerErr.error, true
	}
	var visible userVisibleError
	if errors.As(err, &visible) && strings.TrimSpace(visible.UserMessage()) != "" {
		return visible.UserMessage(), nil, true
	}
	return "", nil, false
}

func classifyProviderFailure(err error) (string, string) {
	var apiErr *providers.APIError
	status, code := 0, ""
	if errors.As(err, &apiErr) {
		status, code = apiErr.StatusCode, strings.ToLower(apiErr.Code)
	}
	lower := strings.ToLower(err.Error())
	message, detail := "The AI service could not complete this request. Please try again, then contact an administrator if it continues.", "The provider did not return a recognized error category."
	switch {
	case status == 401 || status == 403 || containsAny(lower, "invalid api key", "invalid bearer token", "unauthorized", "authentication"):
		message, detail = "The AI service needs administrator attention before it can respond.", "Authentication or account access was rejected."
	case containsAny(code, "insufficient_quota", "billing", "quota") || containsAny(lower, "insufficient quota", "billing", "quota exceeded", "token usage"):
		message, detail = "The AI service has reached its usage limit. Please contact an administrator.", "The provider rejected the request because of an account usage limit."
	case status == 429 || containsAny(lower, "rate limit", "rate_limit"):
		message, detail = "The AI service is busy. Please wait a moment and try again.", "The provider rate-limited this request."
	case status == 400 && containsAny(lower, "model", "not supported", "not available"):
		message, detail = "The selected AI model is not available. Choose another model or contact an administrator.", "The provider rejected the selected model."
	case status == 400 && containsAny(lower, "context length", "maximum context", "too many tokens"):
		message, detail = "This conversation is too long for the selected AI model. Start a new conversation or shorten your request.", "The provider rejected the request because it exceeds the model context limit."
	case status >= 500 || containsAny(lower, "timeout", "deadline exceeded", "temporarily unavailable"):
		message, detail = "The AI service is temporarily unavailable. Please try again shortly.", "The provider or network did not complete the request."
	}
	technical := []string{detail}
	if status != 0 {
		technical = append(technical, fmt.Sprintf("Status: %d", status))
	}
	if code != "" {
		technical = append(technical, "Code: "+code)
	}
	if apiErr != nil && strings.TrimSpace(apiErr.RequestID) != "" {
		technical = append(technical, "Provider request ID: "+apiErr.RequestID)
	}
	return message, strings.Join(technical, "\n")
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
