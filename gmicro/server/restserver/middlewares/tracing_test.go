package middlewares

import "testing"

func TestLowNoiseManagementPaths(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/metrics", want: true},
		{path: "/livez", want: true},
		{path: "/readyz", want: true},
		{path: "/healthz", want: true},
		{path: "/debug/pprof/", want: false},
		{path: "/api/v1/healthz", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isLowNoiseManagementPath(tt.path); got != tt.want {
				t.Errorf("isLowNoiseManagementPath(%q) = %t, want %t", tt.path, got, tt.want)
			}
		})
	}
}
