package client

import "testing"

func TestServiceEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		service string
		want    string
	}{
		{
			name:    "service name",
			service: ServiceUser,
			want:    "discovery:///goshop-user-srv",
		},
		{
			name:    "existing endpoint",
			service: "dns:///user-srv:8021",
			want:    "dns:///user-srv:8021",
		},
		{
			name:    "trim spaces",
			service: " goshop-goods-srv ",
			want:    "discovery:///goshop-goods-srv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ServiceEndpoint(tt.service); got != tt.want {
				t.Fatalf("ServiceEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}
