package logging

import "testing"

func TestSanitizeFieldsRedactsSecretsAndPayloads(t *testing.T) {
	fields := map[string]any{
		"request_id":   "req_123",
		"api_key":      "secret",
		"environment":  map[string]any{"TOKEN": "nested-secret"},
		"fileContents": "source code",
		"details":      []any{map[string]any{"authorization": "bearer secret"}},
	}
	clean := SanitizeFields(fields)
	if clean["request_id"] != "req_123" || clean["api_key"] != Redacted || clean["environment"] != Redacted || clean["fileContents"] != Redacted {
		t.Fatalf("unexpected sanitized fields: %#v", clean)
	}
	details := clean["details"].([]any)[0].(map[string]any)
	if details["authorization"] != Redacted {
		t.Fatalf("nested authorization was not redacted: %#v", details)
	}
}
