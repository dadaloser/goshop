package httperror

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
	spec := apperrors.Spec{Code: 990301, Kind: apperrors.KindNotFound, Message: "user not found", Reference: "https://example.test/errors/user-not-found"}
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
}

func TestWriteResponseMapsKnownGRPCServiceErrorsToPublicCodes(t *testing.T) {
	errorcatalog.RegisterAll()
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name string
		err  error
		want int
	}{
		{name: "validation", err: status.Error(codes.InvalidArgument, "invalid"), want: http.StatusBadRequest},
		{name: "conflict", err: status.Error(codes.Aborted, "conflict"), want: http.StatusConflict},
		{name: "not found", err: status.Error(codes.NotFound, "missing"), want: http.StatusNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			WriteResponse(ctx, tt.err, nil)
			if recorder.Code != tt.want {
				t.Fatalf("WriteResponse() status = %d, want %d", recorder.Code, tt.want)
			}
		})
	}
}

func TestAbortWithErrorWritesSharedErrorResponse(t *testing.T) {
	errorcatalog.RegisterAll()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	AbortWithError(ctx, apperrors.NewCode(100207, "permission denied"))
	if !ctx.IsAborted() || recorder.Code != http.StatusForbidden {
		t.Fatalf("AbortWithError() aborted=%v status=%d, want true/%d", ctx.IsAborted(), recorder.Code, http.StatusForbidden)
	}
}
