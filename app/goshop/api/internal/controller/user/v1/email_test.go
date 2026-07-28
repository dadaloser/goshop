package user

import (
	"bytes"
	"context"
	stderrors "errors"
	"goshop/app/pkg/bizcode"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSendEmailCodeMapsSpecErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		sender     failingEmailSender
		body       string
		wantStatus int
		wantCode   int
	}{
		{
			name:       "invalid request",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   bizcode.ErrCodeInCorrect,
		},
		{
			name:       "sender unavailable",
			sender:     failingEmailSender{err: stderrors.New("smtp unavailable")},
			body:       `{"email":"user@example.com","purpose":"login"}`,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   bizcode.ErrEmailVerificationUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/email-code", SendEmailCode(tt.sender))

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/email-code", bytes.NewBufferString(tt.body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if !bytes.Contains(recorder.Body.Bytes(), []byte(`"code":`+strconv.Itoa(tt.wantCode))) {
				t.Fatalf("body = %s, want code %d", recorder.Body.String(), tt.wantCode)
			}
		})
	}
}

type failingEmailSender struct {
	err error
}

func (s failingEmailSender) Send(context.Context, string, string) error {
	return s.err
}
