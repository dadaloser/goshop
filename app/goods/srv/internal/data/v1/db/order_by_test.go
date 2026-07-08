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
			name:       "allowed ascending",
			value:      "id asc",
			allowed:    goodsOrderColumns,
			wantColumn: "id",
			wantOK:     true,
		},
		{
			name:       "allowed descending uppercase",
			value:      "shop_price DESC",
			allowed:    goodsOrderColumns,
			wantColumn: "shop_price",
			wantDesc:   true,
			wantOK:     true,
		},
		{
			name:       "allows backtick wrapped column",
			value:      "`index` asc",
			allowed:    bannerOrderColumns,
			wantColumn: "index",
			wantOK:     true,
		},
		{
			name:    "rejects unknown column",
			value:   "password asc",
			allowed: goodsOrderColumns,
		},
		{
			name:    "rejects injected clause",
			value:   "id desc; drop table goods",
			allowed: goodsOrderColumns,
		},
		{
			name:    "rejects invalid direction",
			value:   "id sideways",
			allowed: goodsOrderColumns,
		},
		{
			name:    "rejects empty",
			value:   " ",
			allowed: goodsOrderColumns,
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
