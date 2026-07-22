package log

import (
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const redactedFieldValue = "***REDACTED***"

var sensitiveFieldKeyParts = []string{
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

func isSensitiveFieldKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	if normalized == "key" || strings.HasSuffix(normalized, ".key") || strings.HasSuffix(normalized, "-key") {
		return true
	}
	for _, part := range sensitiveFieldKeyParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}

	return false
}

func sanitizeField(field zapcore.Field) zapcore.Field {
	if !isSensitiveFieldKey(field.Key) {
		return field
	}
	return zap.String(field.Key, redactedFieldValue)
}

func sanitizeFields(fields []zapcore.Field) []zapcore.Field {
	if len(fields) == 0 {
		return fields
	}

	sanitized := make([]zapcore.Field, len(fields))
	for i, field := range fields {
		sanitized[i] = sanitizeField(field)
	}
	return sanitized
}

func maskedAttribute(key string) attribute.KeyValue {
	return attribute.String(key, redactedFieldValue)
}
