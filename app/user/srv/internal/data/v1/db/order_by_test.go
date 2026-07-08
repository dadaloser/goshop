package db

import "testing"

func TestParseOrderBy(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantColumn string
		wantDesc   bool
		wantOK     bool
	}{
		{
			name:       "allowed ascending",
			value:      "mobile asc",
			wantColumn: "mobile",
			wantOK:     true,
		},
		{
			name:       "allowed descending uppercase",
			value:      "add_time DESC",
			wantColumn: "add_time",
			wantDesc:   true,
			wantOK:     true,
		},
		{
			name:       "allows backtick wrapped column",
			value:      "`username` asc",
			wantColumn: "username",
			wantOK:     true,
		},
		{
			name:  "rejects unknown column",
			value: "password asc",
		},
		{
			name:  "rejects injected clause",
			value: "id desc; drop table user",
		},
		{
			name:  "rejects invalid direction",
			value: "id sideways",
		},
		{
			name:  "rejects empty",
			value: " ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotColumn, gotDesc, gotOK := parseOrderBy(tt.value, userOrderColumns)
			if gotColumn != tt.wantColumn || gotDesc != tt.wantDesc || gotOK != tt.wantOK {
				t.Fatalf("parseOrderBy() = (%q, %t, %t), want (%q, %t, %t)",
					gotColumn, gotDesc, gotOK, tt.wantColumn, tt.wantDesc, tt.wantOK)
			}
		})
	}
}
