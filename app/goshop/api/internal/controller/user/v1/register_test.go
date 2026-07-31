package user

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"goshop/app/pkg/bizcode"
	"goshop/gmicro/server/restserver/validation"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func TestRegisterWithoutSMSReturnsAccurateVerificationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registerUserMobileValidator(t)

	tests := []struct {
		name        string
		body        string
		wantCode    int
		wantMessage string
	}{
		{
			name:        "password confirmation mismatch",
			body:        `{"username":"user_001","mobile":"13800138000","password":"password1","confirm_password":"password2"}`,
			wantCode:    bizcode.ErrPasswordConfirmationMismatch,
			wantMessage: "两次输入的密码不一致",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/user/register", strings.NewReader(tt.body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			(&userServer{}).Register(ctx)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("Register() status = %d, want %d body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
			}

			var response struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.Code != tt.wantCode || response.Msg != tt.wantMessage {
				t.Fatalf("response = %#v, want code=%d msg=%q", response, tt.wantCode, tt.wantMessage)
			}
		})
	}
}

var registerUserMobileOnce sync.Once

func registerUserMobileValidator(t *testing.T) {
	t.Helper()

	var err error
	registerUserMobileOnce.Do(func() {
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			err = v.RegisterValidation("mobile", validation.ValidateMobile)
		}
	})
	if err != nil {
		t.Fatalf("register mobile validator: %v", err)
	}
}
