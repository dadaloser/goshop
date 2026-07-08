package db

import (
	"reflect"
	"testing"
)

func TestNormalizeIDs(t *testing.T) {
	tests := []struct {
		name string
		ids  []uint64
		want []uint64
	}{
		{
			name: "empty",
		},
		{
			name: "drops zero and duplicates",
			ids:  []uint64{0, 3, 2, 3, 0, 1, 2},
			want: []uint64{3, 2, 1},
		},
		{
			name: "all invalid",
			ids:  []uint64{0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeIDs(tt.ids)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeIDs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
