package db

import "testing"

func TestParseOrderBy(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		allowed    map[string]string
		wantColumn string
		wantDesc   bool
		wantOK     bool
	}{
		{
			name:       "order allowed ascending",
			value:      "order_sn asc",
			allowed:    orderInfoOrderColumns,
			wantColumn: "order_sn",
			wantOK:     true,
		},
		{
			name:       "cart allowed descending uppercase",
			value:      "goods DESC",
			allowed:    shopCartOrderColumns,
			wantColumn: "goods",
			wantDesc:   true,
			wantOK:     true,
		},
		{
			name:       "allows backtick wrapped column",
			value:      "`add_time` asc",
			allowed:    orderInfoOrderColumns,
			wantColumn: "add_time",
			wantOK:     true,
		},
		{
			name:    "rejects unknown column",
			value:   "password asc",
			allowed: orderInfoOrderColumns,
		},
		{
			name:    "rejects injected clause",
			value:   "id desc; drop table orderinfo",
			allowed: orderInfoOrderColumns,
		},
		{
			name:    "rejects invalid direction",
			value:   "id sideways",
			allowed: orderInfoOrderColumns,
		},
		{
			name:    "rejects empty",
			value:   " ",
			allowed: shopCartOrderColumns,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotColumn, gotDesc, gotOK := parseOrderBy(tt.value, tt.allowed)
			if gotColumn != tt.wantColumn || gotDesc != tt.wantDesc || gotOK != tt.wantOK {
				t.Fatalf("parseOrderBy() = (%q, %t, %t), want (%q, %t, %t)",
					gotColumn, gotDesc, gotOK, tt.wantColumn, tt.wantDesc, tt.wantOK)
			}
		})
	}
}
