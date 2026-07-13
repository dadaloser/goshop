package resilience

import (
	"testing"
	"time"
)

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Options)
		wantErr bool
	}{
		{
			name:   "defaults",
			mutate: func(*Options) {},
		},
		{
			name: "invalid timeout",
			mutate: func(options *Options) {
				options.Timeout = 0
			},
			wantErr: true,
		},
		{
			name: "invalid error ratio",
			mutate: func(options *Options) {
				options.ErrorRatio = 1.1
			},
			wantErr: true,
		},
		{
			name: "invalid minimum requests",
			mutate: func(options *Options) {
				options.MinRequestAmount = 0
			},
			wantErr: true,
		},
		{
			name: "invalid statistic interval",
			mutate: func(options *Options) {
				options.StatInterval = time.Nanosecond
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := NewOptions()
			tt.mutate(options)
			if got := len(options.Validate()) > 0; got != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", options.Validate(), tt.wantErr)
			}
		})
	}
}
