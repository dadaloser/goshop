package log

import (
	stderrors "errors"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestErrorDetailRedactsDiagnosticContent(t *testing.T) {
	secret := "top-secret-token"
	detail := ErrorDetail(stderrors.New(`request failed: authorization=Bearer top-secret-token password="db-password" dsn=postgres://user:url-password@db.example.test/app email=alice@example.com phone=13800138000`))

	if strings.Contains(detail.String, secret) || strings.Contains(detail.String, "db-password") || strings.Contains(detail.String, "url-password") || strings.Contains(detail.String, "alice@example.com") || strings.Contains(detail.String, "13800138000") {
		t.Fatalf("ErrorDetail() = %q, want sensitive diagnostic content redacted", detail.String)
	}
	if got := strings.Count(detail.String, redactedFieldValue); got < 5 {
		t.Fatalf("ErrorDetail() redactions = %d, want at least 5: %q", got, detail.String)
	}
}

func TestSanitizeFieldRedactsDirectErrorDetail(t *testing.T) {
	field := sanitizeField(zap.String(KeyErrorDetail, `token: direct-secret`))
	if strings.Contains(field.String, "direct-secret") {
		t.Fatalf("sanitizeField(error_detail) = %q, want secret redacted", field.String)
	}
}
