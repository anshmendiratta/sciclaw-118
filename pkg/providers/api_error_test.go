package providers

import (
	"net/http"
	"testing"
)

func TestNewAPIErrorParsesOpenAIEnvelope(t *testing.T) {
	response := &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"X-Request-Id": []string{"req-123"}}}
	err := NewAPIError("OpenAI", response, []byte(`{"error":{"code":"model_not_found","message":"internal model name"}}`))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type=%T", err)
	}
	if apiErr.Code != "model_not_found" || apiErr.Message != "internal model name" || apiErr.RequestID != "req-123" {
		t.Fatalf("unexpected API error: %#v", apiErr)
	}
}
