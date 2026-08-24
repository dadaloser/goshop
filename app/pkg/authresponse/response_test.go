package authresponse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goshop/app/pkg/errorcatalog"
	"goshop/gmicro/server/restserver/middlewares/auth"
	"goshop/pkg/errcode"
	core "goshop/pkg/transport/httperror"

	"github.com/gin-gonic/gin"
)

func TestWriteMapsAuthenticationFailuresToBusinessCodes(t *testing.T) {
	errorcatalog.RegisterAll()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{name: "missing credentials", err: auth.ErrMissingCredentials, wantCode: errcode.ErrMissingHeader},
		{name: "invalid authorization", err: auth.ErrInvalidAuthorization, wantCode: errcode.ErrInvalidAuthHeader},
		{name: "expired credentials", err: auth.ErrExpiredCredentials, wantCode: errcode.ErrExpired},
		{name: "invalid token", err: auth.ErrInvalidToken, wantCode: errcode.ErrSignatureInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			Write(ctx, tt.err)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("Write(%v) status = %d, want %d", tt.err, recorder.Code, http.StatusUnauthorized)
			}
			var response core.ErrResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("Write(%v) decode response = %v", tt.err, err)
			}
			if response.Code != tt.wantCode {
				t.Errorf("Write(%v) code = %d, want %d", tt.err, response.Code, tt.wantCode)
			}
		})
	}
}
