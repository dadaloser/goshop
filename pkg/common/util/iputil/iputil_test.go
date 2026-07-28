package iputil

import (
	"net"
	"net/http/httptest"
	"testing"
)

func TestRemoteIPIgnoresForwardedHeadersFromUntrustedPeer(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "203.0.113.10:8080"
	req.Header.Set(XForwardedFor, "198.51.100.8")
	req.Header.Set(XRealIP, "198.51.100.9")

	if got, want := RemoteIP(req), "203.0.113.10"; got != want {
		t.Fatalf("RemoteIP() = %q, want %q", got, want)
	}
}

func TestRemoteIPFromTrustedProxyUsesRightmostUntrustedForwardedIP(t *testing.T) {
	_, trustedProxy, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatalf("ParseCIDR() error = %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "10.1.2.3:8080"
	req.Header.Set(XForwardedFor, "not-an-ip, 198.51.100.8, 198.51.100.9")

	if got, want := RemoteIPFromTrustedProxy(req, []*net.IPNet{trustedProxy}), "198.51.100.9"; got != want {
		t.Fatalf("RemoteIPFromTrustedProxy() = %q, want %q", got, want)
	}
}

func TestRemoteIPFromTrustedProxySkipsTrustedForwardedHops(t *testing.T) {
	_, trustedProxy, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatalf("ParseCIDR() error = %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "10.1.2.3:8080"
	req.Header.Set(XForwardedFor, "198.51.100.8, 10.2.3.4")

	if got, want := RemoteIPFromTrustedProxy(req, []*net.IPNet{trustedProxy}), "198.51.100.8"; got != want {
		t.Fatalf("RemoteIPFromTrustedProxy() = %q, want %q", got, want)
	}
}

func TestRemoteIPFromTrustedProxyIgnoresHeadersForUntrustedPeer(t *testing.T) {
	_, trustedProxy, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatalf("ParseCIDR() error = %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "203.0.113.10:8080"
	req.Header.Set(XForwardedFor, "198.51.100.8")

	if got, want := RemoteIPFromTrustedProxy(req, []*net.IPNet{trustedProxy}), "203.0.113.10"; got != want {
		t.Fatalf("RemoteIPFromTrustedProxy() = %q, want %q", got, want)
	}
}
