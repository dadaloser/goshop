package log

import (
	"regexp"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const redactedFieldValue = "***REDACTED***"

var (
	diagnosticSecretPattern = regexp.MustCompile(`(?i)(["']?(?:access[_-]?key|api[_-]?key|api[_-]?secret|authorization|credential|passwd|password|private[_-]?key|profiling[_-]?token|secret|token)["']?\s*[:=]\s*(?:bearer\s+)?["']?)([^\s,"'\}\]]+)(["']?)`)
	diagnosticBearerPattern = regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/=-]+`)
	diagnosticURLPassword   = regexp.MustCompile(`(?i)(://[^:\s/@]+:)([^@\s/]+)(@)`)
	diagnosticEmailPattern  = regexp.MustCompile(`(?i)\b[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}\b`)
	diagnosticPhonePattern  = regexp.MustCompile(`\b1[3-9][0-9]{9}\b`)
)

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
	if isSensitiveFieldKey(field.Key) {
		return zap.String(field.Key, redactedFieldValue)
	}
	if field.Key == KeyErrorDetail && field.Type == zapcore.StringType {
		return zap.String(field.Key, sanitizeDiagnostic(field.String))
	}
	return field
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

// sanitizeDiagnostic applies a best-effort content filter to controlled error
// diagnostics. It is a defense in depth measure, not permission to include
// complete external response bodies, credentials, or raw SQL parameters in
// diagnostic messages.
func sanitizeDiagnostic(detail string) string {
	detail = diagnosticSecretPattern.ReplaceAllString(detail, "${1}"+redactedFieldValue+"${3}")
	detail = diagnosticBearerPattern.ReplaceAllString(detail, "Bearer "+redactedFieldValue)
	detail = diagnosticURLPassword.ReplaceAllString(detail, "${1}"+redactedFieldValue+"${3}")
	detail = diagnosticEmailPattern.ReplaceAllString(detail, redactedFieldValue)
	return diagnosticPhonePattern.ReplaceAllString(detail, redactedFieldValue)
}
