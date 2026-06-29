package restserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadyzReturnsUnavailableAfterStop(t *testing.T) {
	srv := NewServer(WithHealthCheck(true))
	srv.registerHealthRoutes()
	srv.readyOnce.Do(func() {
		close(srv.ready)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("readyz before stop status = %d, want 200", rec.Code)
	}

	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v, want nil", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz after stop status = %d, want 503", rec.Code)
	}
}

func TestDefaultJWTKeyIsEmpty(t *testing.T) {
	srv := NewServer()
	if srv.jwt.Key != "" {
		t.Fatal("default JWT key should be empty and configured explicitly")
	}
}
