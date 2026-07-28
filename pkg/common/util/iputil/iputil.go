package iputil

import (
	"net"
	"net/http"
	"strings"
)

// Define http headers.
const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-IP"
	XClientIP     = "x-client-ip"
)

// GetLocalIP returns the non loopback local IP of the host.
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

// RemoteIP returns the direct peer IP of the request. Forwarded headers are
// intentionally ignored because they can be supplied by an untrusted client.
func RemoteIP(req *http.Request) string {
	return directRemoteIP(req)
}

// RemoteIPFromTrustedProxy returns the client IP from forwarding headers only
// when the direct peer belongs to trustedProxies. X-Forwarded-For is evaluated
// from right to left, skipping trusted proxies, so appended proxy hops cannot
// cause a client-supplied leftmost address to be trusted.
func RemoteIPFromTrustedProxy(req *http.Request, trustedProxies []*net.IPNet) string {
	peerIP := directRemoteIP(req)
	if !isTrustedProxy(peerIP, trustedProxies) {
		return peerIP
	}

	if forwardedIP := clientIPFromForwardedFor(req.Header.Get(XForwardedFor), trustedProxies); forwardedIP != "" {
		return forwardedIP
	}
	for _, header := range []string{XRealIP, XClientIP} {
		if forwardedIP := firstValidIP(req.Header.Get(header)); forwardedIP != "" {
			return forwardedIP
		}
	}

	return peerIP
}

func directRemoteIP(req *http.Request) string {
	remoteAddr, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		remoteAddr = req.RemoteAddr
	}
	if remoteAddr == "::1" {
		return "127.0.0.1"
	}

	return remoteAddr
}

func isTrustedProxy(remoteAddr string, trustedProxies []*net.IPNet) bool {
	ip := net.ParseIP(remoteAddr)
	if ip == nil {
		return false
	}
	for _, trustedProxy := range trustedProxies {
		if trustedProxy != nil && trustedProxy.Contains(ip) {
			return true
		}
	}

	return false
}

func firstValidIP(header string) string {
	for _, value := range strings.Split(header, ",") {
		value = strings.TrimSpace(value)
		if net.ParseIP(value) != nil {
			return value
		}
	}

	return ""
}

func clientIPFromForwardedFor(header string, trustedProxies []*net.IPNet) string {
	values := strings.Split(header, ",")
	for i := len(values) - 1; i >= 0; i-- {
		value := strings.TrimSpace(values[i])
		if net.ParseIP(value) == nil || isTrustedProxy(value, trustedProxies) {
			continue
		}
		return value
	}

	return ""
}
