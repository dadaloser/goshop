package restserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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

func TestValidateStartupConfigRejectsProductionProfiling(t *testing.T) {
	srv := NewServer(
		WithMode(gin.ReleaseMode),
		WithEnableProfiling(true),
	)

	err := srv.ValidateStartupConfig()
	if err == nil {
		t.Fatal("ValidateStartupConfig() error = nil, want profiling error")
	}
}

func TestValidateStartupConfigRunsCustomValidator(t *testing.T) {
	wantErr := errors.New("custom config rejected")
	srv := NewServer(
		WithStartupValidator(func(*Server) error {
			return wantErr
		}),
	)

	err := srv.ValidateStartupConfig()
	if !errors.Is(err, wantErr) {
		t.Fatalf("ValidateStartupConfig() error = %v, want %v", err, wantErr)
	}
}

func TestValidateStartupConfigAllowsProtectedProductionProfiling(t *testing.T) {
	srv := NewServer(
		WithMode(gin.ReleaseMode),
		WithEnableProfiling(true),
		WithProfilingToken("secret-token"),
	)

	if err := srv.ValidateStartupConfig(); err != nil {
		t.Fatalf("ValidateStartupConfig() error = %v, want nil", err)
	}
}

func TestRegisterProfilingRequiresBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := NewServer(
		WithMode(gin.TestMode),
		WithEnableProfiling(true),
		WithProfilingToken("secret-token"),
	)
	srv.registerProfilingRoutes()

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("pprof without token status = %d, want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("pprof with token status = %d, want 200", rec.Code)
	}
}

func TestRateLimiterRejectsRequestsBeyondBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := NewServer(WithRateLimit(1, 1))
	srv.GET("/limited", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("first request status = %d, want 204", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/limited", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", rec.Code)
	}
}

func TestMaxConcurrentRequestsRejectsWhenSaturated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := NewServer(WithMaxConcurrentRequests(1))
	block := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	srv.GET("/work", func(c *gin.Context) {
		once.Do(func() { close(started) })
		<-block
		c.Status(http.StatusNoContent)
	})

	firstDone := make(chan int, 1)
	go func() {
		req := httptest.NewRequest(http.MethodGet, "/work", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		firstDone <- rec.Code
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first request did not start")
	}

	req := httptest.NewRequest(http.MethodGet, "/work", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("second request status = %d, want 503", rec.Code)
	}

	close(block)
	select {
	case code := <-firstDone:
		if code != http.StatusNoContent {
			t.Fatalf("first request status = %d, want 204", code)
		}
	case <-time.After(time.Second):
		t.Fatal("first request did not finish")
	}
}
