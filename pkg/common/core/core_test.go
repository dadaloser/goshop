package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

func TestWriteResponseUsesSpecPublicFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	WriteResponse(ctx, apperrors.NewSpec(apperrors.Spec{
		Code:      990301,
		Kind:      apperrors.KindNotFound,
		Message:   "user not found",
		Reference: "https://example.test/errors/user-not-found",
	}, "select user: database connection string=secret"), nil)

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
