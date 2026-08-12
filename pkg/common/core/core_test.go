package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goshop/app/pkg/errorcatalog"
	apperrors "goshop/pkg/errors"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWriteResponseUsesSpecPublicFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	spec := apperrors.Spec{
		Code:      990301,
		Kind:      apperrors.KindNotFound,
		Message:   "user not found",
		Reference: "https://example.test/errors/user-not-found",
	}
	apperrors.MustRegister(spec)
	WriteResponse(ctx, apperrors.NewSpec(spec, "select user: database connection string=secret"), nil)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("WriteResponse() status = %d, want %d", recorder.Code, http.StatusNotFound)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["msg"] != "user not found" {
		t.Fatalf("WriteResponse() msg = %v, want user not found", response["msg"])
	}
	if _, ok := response["detail"]; ok {
		t.Fatalf("WriteResponse() exposed detail: %v", response["detail"])
	}
	if _, ok := response["reference"]; ok {
		t.Fatalf("WriteResponse() exposed reference: %v", response["reference"])
	}
}

func TestWriteResponseMapsKnownGRPCServiceErrorsToPublicCodes(t *testing.T) {
	errorcatalog.RegisterAll()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "user validation", err: status.Error(codes.InvalidArgument, "password must contain a digit"), wantStatus: http.StatusBadRequest},
		{name: "goods conflict", err: status.Error(codes.Aborted, "version conflict"), wantStatus: http.StatusConflict},
		{name: "inventory not found", err: status.Error(codes.NotFound, "stock not found"), wantStatus: http.StatusNotFound},
		{name: "order dependency unavailable", err: status.Error(codes.Unavailable, "inventory unavailable"), wantStatus: http.StatusServiceUnavailable},
		{name: "review validation", err: status.Error(codes.FailedPrecondition, "order cannot be reviewed"), wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			WriteResponse(ctx, tt.err, nil)

			if recorder.Code != tt.wantStatus {
				t.Errorf("WriteResponse(%s) status = %d, want %d", tt.name, recorder.Code, tt.wantStatus)
			}
			var response ErrResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("WriteResponse(%s) decode response = %v", tt.name, err)
			}
			if response.Code == 1 {
				t.Errorf("WriteResponse(%s) code = %d, want a public non-unknown code", tt.name, response.Code)
			}
		})
	}
}
