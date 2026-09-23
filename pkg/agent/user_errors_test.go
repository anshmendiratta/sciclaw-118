package agent

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestProviderFailureErrorDoesNotExposeRawBody(t *testing.T) {
	rawBody := `{"error":{"code":"model_not_found","message":"private provider details"}}`
	err := newProviderFailureError(&providers.APIError{
		StatusCode: 400,
		Code:       "model_not_found",
		Message:    "private provider details",
		RequestID:  "req-private",
		Body:       rawBody,
	}, "ERR-123")

	message, details, ok := UserError(err)
	if !ok || details == nil {
		t.Fatal("expected safe error presentation")
	}
	if strings.Contains(message, "private provider details") || strings.Contains(details.TechnicalDetails, "private provider details") || strings.Contains(details.TechnicalDetails, rawBody) {
		t.Fatalf("raw provider content leaked: message=%q details=%q", message, details.TechnicalDetails)
	}
	if !strings.Contains(message, "selected AI model") || !strings.Contains(message, "ERR-123") {
		t.Fatalf("unexpected message: %q", message)
	}
	if !strings.Contains(details.TechnicalDetails, "Status: 400") || !strings.Contains(details.TechnicalDetails, "Code: model_not_found") || !strings.Contains(details.TechnicalDetails, "req-private") {
		t.Fatalf("unexpected details: %q", details.TechnicalDetails)
	}
}

func TestProviderFailureErrorClassifiesQuotaBeforeRateLimit(t *testing.T) {
	err := newProviderFailureError(&providers.APIError{StatusCode: 429, Code: "insufficient_quota"}, "ERR-456")
	message, _, ok := UserError(err)
	if !ok || !strings.Contains(message, "usage limit") {
		t.Fatalf("expected quota guidance, got %q", message)
	}
}
