package host

import (
	"net"
	"testing"
)

func TestChooseAdvertiseIPPrefersIPv4(t *testing.T) {
	ipv6 := net.ParseIP("2409:8a50:4a5:a310:2cf8:d300:2caf:b124")
	ipv4 := net.ParseIP("192.168.1.2")

	got := chooseAdvertiseIP(ipv6, ipv4)
	if !got.Equal(ipv4) {
		t.Fatalf("chooseAdvertiseIP() = %s, want %s", got, ipv4)
	}
}

func TestChooseAdvertiseIPKeepsExistingIPv4(t *testing.T) {
	current := net.ParseIP("192.168.1.2")
	candidate := net.ParseIP("2409:8a50:4a5:a310:2cf8:d300:2caf:b124")

	got := chooseAdvertiseIP(current, candidate)
	if !got.Equal(current) {
		t.Fatalf("chooseAdvertiseIP() = %s, want %s", got, current)
	}
}

func TestIsValidIPRejectsLoopbackAndLinkLocal(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "private IPv4", ip: "192.168.1.2", want: true},
		{name: "global IPv6", ip: "2409:8a50:4a5:a310:2cf8:d300:2caf:b124", want: true},
		{name: "loopback IPv4", ip: "127.0.0.1", want: false},
		{name: "loopback IPv6", ip: "::1", want: false},
		{name: "link local IPv6", ip: "fe80::1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidIP(tt.ip); got != tt.want {
				t.Fatalf("isValidIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}
