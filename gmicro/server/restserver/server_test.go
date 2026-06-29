package restserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	mws "goshop/gmicro/server/restserver/middlewares"
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

func TestDefaultModeIsRelease(t *testing.T) {
	srv := NewServer()
	if srv.mode != gin.ReleaseMode {
		t.Fatalf("default mode = %q, want release", srv.mode)
	}
}

func TestStartRejectsProductionDebugMode(t *testing.T) {
	srv := NewServer(WithMode(gin.DebugMode))

	err := srv.validateProductionConfig()
	if err == nil {
		t.Fatal("validateProductionConfig() error = nil, want debug mode error")
	}
}

func TestStartRejectsProductionEmptyJWTKey(t *testing.T) {
	srv := NewServer(
		WithMode(gin.ReleaseMode),
		WithJwt(&JwtInfo{Realm: "JWT"}),
	)

	err := srv.validateProductionConfig()
	if err == nil {
		t.Fatal("validateProductionConfig() error = nil, want empty JWT key error")
	}
}

func TestStartAllowsReleaseModeWithoutJWTRequirement(t *testing.T) {
	srv := NewServer(WithMode(gin.ReleaseMode))

	if err := srv.validateProductionConfig(); err != nil {
		t.Fatalf("validateProductionConfig() error = %v, want nil", err)
	}
}

func TestStartAcceptsReleaseModeWithJWTKey(t *testing.T) {
	srv := NewServer(
		WithMode(gin.ReleaseMode),
		WithJwt(&JwtInfo{Realm: "JWT", Key: "test-secret"}),
	)

	if err := srv.validateProductionConfig(); err != nil {
		t.Fatalf("validateProductionConfig() error = %v, want nil", err)
	}
}

func TestStartRejectsProductionWildcardCors(t *testing.T) {
	srv := NewServer(
		WithMode(gin.ReleaseMode),
		WithMiddlewares([]string{"cors"}),
		WithCorsOptions(mws.CorsOptions{AllowOrigins: []string{"*"}}),
	)

	err := srv.validateProductionConfig()
	if err == nil {
		t.Fatal("validateProductionConfig() error = nil, want wildcard cors error")
	}
}

func TestStartAcceptsProductionExplicitCorsOrigins(t *testing.T) {
	srv := NewServer(
		WithMode(gin.ReleaseMode),
		WithMiddlewares([]string{"cors"}),
		WithCorsOptions(mws.CorsOptions{AllowOrigins: []string{"https://shop.example.com"}}),
	)

	if err := srv.validateProductionConfig(); err != nil {
		t.Fatalf("validateProductionConfig() error = %v, want nil", err)
	}
}
