package restserver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

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
	req.RemoteAddr = "10.0.0.2:1234"
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("readyz before stop status = %d, want 200", rec.Code)
	}

	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v, want nil", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz after stop status = %d, want 503", rec.Code)
	}
}

func TestInitTransSupportsConcurrentServers(t *testing.T) {
	servers := []*Server{
		NewServer(WithTransNames("zh")),
		NewServer(WithTransNames("en")),
	}

	errs := make(chan error, len(servers))
	var wg sync.WaitGroup
	for _, srv := range servers {
		srv := srv
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- srv.initTrans(srv.transName)
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("initTrans() error = %v, want nil", err)
		}
	}
	for _, srv := range servers {
		if srv.Translator() == nil {
			t.Errorf("initTrans(%q) translator = nil, want initialized translator", srv.transName)
		}
	}
}

func TestReadyzReturnsUnavailableWhenDependencyFails(t *testing.T) {
	srv := NewServer(WithHealthCheck(true), WithReadinessCheck(func() error {
		return errors.New("redis unavailable")
	}))
	srv.registerHealthRoutes()
	srv.readyOnce.Do(func() {
		close(srv.ready)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz dependency failure status = %d, want 503", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/livez", nil)
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("livez dependency failure status = %d, want 200", rec.Code)
	}
}

func TestRegisterBuiltInRoutesIsIdempotent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := NewServer(
		WithMode(gin.TestMode),
		WithHealthCheck(true),
		WithEnableProfiling(true),
		WithProfilingToken("secret-token"),
	)

	srv.registerBuiltInRoutes()
	srv.registerBuiltInRoutes()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("livez status = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	req.Header.Set("Authorization", "Bearer secret-token")
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("pprof status = %d, want 200", rec.Code)
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
	req.RemoteAddr = "10.0.0.2:1234"
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("pprof without token status = %d, want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("pprof with token status = %d, want 200", rec.Code)
	}
}

func TestValidateStartupConfigRejectsInvalidBuiltInRouteCIDR(t *testing.T) {
	srv := NewServer(WithBuiltInRouteCIDRs([]string{"not-a-cidr"}))

	if err := srv.ValidateStartupConfig(); err == nil {
		t.Fatal("ValidateStartupConfig() error = nil, want invalid built-in route cidr error")
	}
}

func TestValidateStartupConfigRejectsUnknownMiddleware(t *testing.T) {
	srv := NewServer(WithMiddlewares([]string{"does-not-exist"}))

	if err := srv.ValidateStartupConfig(); err == nil {
		t.Fatal("ValidateStartupConfig() error = nil, want unknown middleware error")
	}
}

func TestValidateStartupConfigRejectsDuplicateMiddleware(t *testing.T) {
	srv := NewServer(WithMiddlewares([]string{"recovery", "recovery"}))

	if err := srv.ValidateStartupConfig(); err == nil {
		t.Fatal("ValidateStartupConfig() error = nil, want duplicate middleware error")
	}
}

func TestValidateStartupConfigRejectsBuiltInRecoveryMiddleware(t *testing.T) {
	srv := NewServer(WithMiddlewares([]string{"recovery"}))

	if err := srv.ValidateStartupConfig(); err == nil {
		t.Fatal("ValidateStartupConfig() error = nil, want built-in middleware error")
	}
}

func TestWithNamedMiddlewareIsServerLocal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	custom := func(c *gin.Context) {
		c.Header("X-Custom-Middleware", "installed")
		c.Next()
	}
	configured := NewServer(
		WithMode(gin.TestMode),
		WithNamedMiddleware("custom", custom),
		WithMiddlewares([]string{"custom"}),
	)
	configured.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	rec := httptest.NewRecorder()
	configured.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := rec.Header().Get("X-Custom-Middleware"); got != "installed" {
		t.Fatalf("custom middleware header = %q, want installed", got)
	}

	unconfigured := NewServer(WithMode(gin.TestMode), WithMiddlewares([]string{"custom"}))
	if err := unconfigured.ValidateStartupConfig(); err == nil {
		t.Fatal("custom middleware leaked into another server instance")
	}
}

func TestServerProvidesRequestIDAndRecoversPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := NewServer(WithMode(gin.TestMode))
	srv.GET("/panic", func(*gin.Context) { panic("boom") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic status = %d, want 500", rec.Code)
	}
	if got := rec.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("X-Request-ID is empty")
	}
}

func TestRequestBodyLimitRejectsKnownOversizedBody(t *testing.T) {
	srv := NewServer(WithMode(gin.TestMode), WithMaxRequestBodyBytes(4))
	called := false
	srv.POST("/body", func(c *gin.Context) { called = true; c.Status(http.StatusNoContent) })

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/body", bytes.NewBufferString("12345")))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status = %d, want 413", rec.Code)
	}
	if called {
		t.Fatal("oversized body handler called = true, want false")
	}
}

func TestRequestBodyLimitCapsUnknownLengthBody(t *testing.T) {
	srv := NewServer(WithMode(gin.TestMode), WithMaxRequestBodyBytes(4))
	var readErr error
	srv.POST("/body", func(c *gin.Context) {
		_, readErr = io.ReadAll(c.Request.Body)
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/body", bytes.NewBufferString("12345"))
	req.ContentLength = -1
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var maxBytesErr *http.MaxBytesError
	if !errors.As(readErr, &maxBytesErr) {
		t.Fatalf("io.ReadAll(over-limit body) error = %v, want *http.MaxBytesError", readErr)
	}
}

func TestRequestDeadlinePropagatesAndReturnsGatewayTimeout(t *testing.T) {
	srv := NewServer(WithMode(gin.TestMode), WithHandlerTimeout(10*time.Millisecond))
	srv.GET("/slow", func(c *gin.Context) { <-c.Request.Context().Done() })
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/slow", nil))
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("deadline status = %d, want 504", rec.Code)
	}
}

func TestClientRouteRateLimitUsesForwardedIPOnlyFromTrustedProxy(t *testing.T) {
	request := func(srv *Server, forwarded string) int {
		req := httptest.NewRequest(http.MethodGet, "/limited", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		req.Header.Set("X-Forwarded-For", forwarded)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		return rec.Code
	}

	trusted := NewServer(WithMode(gin.TestMode), WithTrustedProxies([]string{"10.0.0.0/8"}), WithClientRouteRateLimit(0.0001, 1, 10))
	trusted.GET("/limited", func(c *gin.Context) { c.Status(http.StatusOK) })
	if got := request(trusted, "192.0.2.1"); got != http.StatusOK {
		t.Fatalf("trusted proxy first client status = %d, want 200", got)
	}
	if got := request(trusted, "192.0.2.2"); got != http.StatusOK {
		t.Fatalf("trusted proxy second client status = %d, want 200", got)
	}

	untrusted := NewServer(WithMode(gin.TestMode), WithClientRouteRateLimit(0.0001, 1, 10))
	untrusted.GET("/limited", func(c *gin.Context) { c.Status(http.StatusOK) })
	if got := request(untrusted, "192.0.2.1"); got != http.StatusOK {
		t.Fatalf("untrusted proxy first request status = %d, want 200", got)
	}
	if got := request(untrusted, "192.0.2.2"); got != http.StatusTooManyRequests {
		t.Fatalf("untrusted proxy spoofed request status = %d, want 429", got)
	}
}

func TestPanicIsRecordedAsHTTP500Metric(t *testing.T) {
	service := "panic-metrics-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	srv := NewServer(WithMode(gin.TestMode), WithServiceName(service), WithMetricsCollection(true))
	srv.GET("/panic", func(*gin.Context) { panic("boom") })
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic response status = %d, want 500", rec.Code)
	}
	if got := httpCounterValue(t, service, "/panic", "500"); got != 1 {
		t.Fatalf("panic HTTP counter = %v, want 1", got)
	}
	if got := httpHistogramCount(t, service, "/panic"); got != 1 {
		t.Fatalf("panic HTTP latency count = %v, want 1", got)
	}
}

func TestClientRouteRateLimitIsolatesClientsAndRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := NewServer(WithMode(gin.TestMode), WithClientRouteRateLimit(0.0001, 1, 100))
	srv.GET("/users/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	srv.GET("/orders/:id", func(c *gin.Context) { c.Status(http.StatusOK) })

	request := func(path, remoteAddr string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = remoteAddr
		srv.ServeHTTP(rec, req)
		return rec
	}

	if got := request("/users/1", "192.0.2.1:1234").Code; got != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", got)
	}
	limited := request("/users/2", "192.0.2.1:1234")
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("same client and route template status = %d, want 429", limited.Code)
	}
	if got := limited.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("Retry-After = %q, want 1", got)
	}
	if got := request("/orders/1", "192.0.2.1:1234").Code; got != http.StatusOK {
		t.Fatalf("different route status = %d, want 200", got)
	}
	if got := request("/users/3", "192.0.2.2:1234").Code; got != http.StatusOK {
		t.Fatalf("different client status = %d, want 200", got)
	}
}

func TestClientRouteLimiterBoundsKeysWithLRUEviction(t *testing.T) {
	limiter := newClientRouteLimiter(1, 1, 2)
	now := time.Now()

	limiter.allow("first", now)
	limiter.allow("second", now)
	limiter.allow("first", now)
	limiter.allow("third", now)

	if got := len(limiter.entries); got != 2 {
		t.Fatalf("limiter entries = %d, want 2", got)
	}
	if _, ok := limiter.entries["second"]; ok {
		t.Fatal("least recently used entry was not evicted")
	}
}

func TestReadyzAllowsInternalAndRejectsPublicClients(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := NewServer(WithHealthCheck(true))
	srv.registerHealthRoutes()
	srv.readyOnce.Do(func() {
		close(srv.ready)
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	req.RemoteAddr = "8.8.8.8:1234"
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("readyz public client status = %d, want 403", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	req.RemoteAddr = "10.1.2.3:1234"
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("readyz internal client status = %d, want 200", rec.Code)
	}
}

func TestMetricsAllowInternalAndRejectPublicClients(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := NewServer(WithMetrics(true))
	srv.registerBuiltInRoutes()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("metrics public client status = %d, want 403", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "192.168.1.10:5678"
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics internal client status = %d, want 200", rec.Code)
	}
}

func TestMetricsCollectionDoesNotRequireMetricsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := "metrics-collection-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	srv := NewServer(
		WithMode(gin.TestMode),
		WithServiceName(service),
		WithMetricsCollection(true),
		WithMetricsEndpoint(false),
	)
	srv.GET("/orders/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	srv.registerBuiltInRoutes()

	req := httptest.NewRequest(http.MethodGet, "/orders/42", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("business request status = %d, want 204", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("metrics endpoint status = %d, want 404", rec.Code)
	}

	if !hasHTTPMetric(t, service, "/orders/:id") {
		t.Fatal("business request metric was not collected")
	}
}

func TestMetricsEndpointDoesNotCollectManagementTraffic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := "metrics-management-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	srv := NewServer(
		WithMode(gin.TestMode),
		WithServiceName(service),
		WithMetricsCollection(false),
		WithMetricsEndpoint(true),
	)
	srv.registerBuiltInRoutes()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics endpoint status = %d, want 200", rec.Code)
	}
	if hasHTTPMetric(t, service, "/metrics") {
		t.Fatal("management metrics scrape was recorded as business traffic")
	}
}

func hasHTTPMetric(t *testing.T, service, route string) bool {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	for _, family := range families {
		if family.GetName() != "goshop_http_server_requests_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := make(map[string]string, len(metric.GetLabel()))
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["service"] == service && labels["route"] == route {
				return true
			}
		}
	}
	return false
}

func httpCounterValue(t *testing.T, service, route, code string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("prometheus Gather() error = %v", err)
	}
	for _, family := range families {
		if family.GetName() != "goshop_http_server_requests_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := make(map[string]string, len(metric.GetLabel()))
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["service"] == service && labels["route"] == route && labels["code"] == code {
				return metric.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func httpHistogramCount(t *testing.T, service, route string) uint64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("prometheus Gather() error = %v", err)
	}
	for _, family := range families {
		if family.GetName() != "goshop_http_server_request_duration_seconds" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := make(map[string]string, len(metric.GetLabel()))
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["service"] == service && labels["route"] == route {
				return metric.GetHistogram().GetSampleCount()
			}
		}
	}
	return 0
}

func TestProfilingRequiresInternalClientAndBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := NewServer(
		WithMode(gin.TestMode),
		WithEnableProfiling(true),
		WithProfilingToken("secret-token"),
	)
	srv.registerProfilingRoutes()

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.RemoteAddr = "203.0.113.5:4321"
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("pprof public client status = %d, want 403", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.RemoteAddr = "10.0.0.8:4321"
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("pprof internal client without token status = %d, want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.RemoteAddr = "10.0.0.8:4321"
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("pprof internal client with token status = %d, want 200", rec.Code)
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
