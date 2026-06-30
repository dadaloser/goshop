package app

import (
	"encoding/json"
	"strings"
)

const redactedValue = "***REDACTED***"

var sensitiveKeyParts = []string{
	"access-key",
	"apikey",
	"api-key",
	"api_secret",
	"apisecret",
	"api-secret",
	"authorization",
	"credential",
	"passwd",
	"password",
	"private-key",
	"profiling-token",
	"secret",
	"token",
}

// RedactJSON redacts sensitive fields from a JSON object string.
func RedactJSON(raw string) string {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}

	redactValue(value, "")
	data, err := json.Marshal(value)
	if err != nil {
		return raw
	}

	return string(data)
}

func redactValue(value any, key string) {
	switch typed := value.(type) {
	case map[string]any:
		for childKey, childValue := range typed {
			if isSensitiveKey(childKey) {
				typed[childKey] = redactedValue
				continue
			}
			redactValue(childValue, childKey)
		}
	case []any:
		for _, childValue := range typed {
			redactValue(childValue, key)
		}
	}
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	if normalized == "key" || strings.HasSuffix(normalized, ".key") || strings.HasSuffix(normalized, "-key") {
		return true
	}
	for _, part := range sensitiveKeyParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}

	return false
}
