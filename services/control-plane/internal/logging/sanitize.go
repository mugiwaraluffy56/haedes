package logging

import (
	"strings"
)

const Redacted = "[REDACTED]"

// SanitizeFields returns a new map suitable for structured logs. Sensitive
// fields are replaced before a logger can serialize them, including nested
// maps and slices. Callers should pass request metadata only, not raw request
// bodies.
func SanitizeFields(fields map[string]any) map[string]any {
	clean := make(map[string]any, len(fields))
	for key, value := range fields {
		if sensitiveKey(key) {
			clean[key] = Redacted
			continue
		}
		clean[key] = sanitizeValue(value)
	}
	return clean
}

func sanitizeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return SanitizeFields(typed)
	case []any:
		clean := make([]any, len(typed))
		for i, item := range typed {
			clean[i] = sanitizeValue(item)
		}
		return clean
	default:
		return value
	}
}

func sensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(key))
	if normalized == "environment" || normalized == "env" || normalized == "content" || normalized == "filecontents" || normalized == "filecontent" {
		return true
	}
	for _, fragment := range []string{"apikey", "authorization", "credential", "password", "secret", "token", "privatekey"} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}
