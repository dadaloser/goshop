package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goshop/app/pkg/errorcatalog"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWriteUserRPCErrorUsesPublicConflictResponse(t *testing.T) {
	errorcatalog.RegisterAll()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	writeUserRPCError(ctx, status.Error(codes.Aborted, "duplicate username"), "user already exists")

	if recorder.Code != http.StatusConflict {
		t.Fatalf("writeUserRPCError(Aborted) status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("writeUserRPCError(Aborted) response JSON = %v", err)
	}
	if got, ok := body["code"].(float64); !ok || got == 1 {
		t.Errorf("writeUserRPCError(Aborted) code = %#v, want public non-unknown code", body["code"])
	}
	if _, exists := body["detail"]; exists {
		t.Error("writeUserRPCError(Aborted) exposed detail")
	}
	if _, exists := body["reference"]; exists {
		t.Error("writeUserRPCError(Aborted) exposed reference")
	}
}
