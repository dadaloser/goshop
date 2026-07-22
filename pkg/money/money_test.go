package money

import (
	"math"
	"testing"
)

func TestParseYuan(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Fen
		wantErr bool
	}{
		{name: "zero", input: "0", want: 0},
		{name: "integer", input: "12", want: 1200},
		{name: "one decimal", input: "12.3", want: 1230},
		{name: "two decimals", input: "12.34", want: 1234},
		{name: "leading dot not allowed", input: ".34", want: 34},
		{name: "trim space", input: " 99.01 ", want: 9901},
		{name: "negative", input: "-1.23", want: -123},
		{name: "too many decimals", input: "1.234", wantErr: true},
		{name: "invalid letters", input: "12a", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseYuan(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseYuan(%q) error = nil, want error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseYuan(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("ParseYuan(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestFenYuanString(t *testing.T) {
	tests := []struct {
		input Fen
		want  string
	}{
		{input: 0, want: "0.00"},
		{input: 1, want: "0.01"},
		{input: 1200, want: "12.00"},
		{input: 1234, want: "12.34"},
		{input: -1234, want: "-12.34"},
	}

	for _, tt := range tests {
		if got := tt.input.YuanString(); got != tt.want {
			t.Fatalf("Fen(%d).YuanString() = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFenAdd(t *testing.T) {
	got, err := Fen(123).Add(Fen(77))
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if got != 200 {
		t.Fatalf("Add() = %d, want 200", got)
	}

	if _, err := Fen(math.MaxInt64).Add(1); err == nil {
		t.Fatalf("Add() overflow error = nil, want error")
	}
}

func TestFenMultiply(t *testing.T) {
	got, err := Fen(1234).Multiply(3)
	if err != nil {
		t.Fatalf("Multiply() error = %v", err)
	}
	if got != 3702 {
		t.Fatalf("Multiply() = %d, want 3702", got)
	}

	if _, err := Fen(math.MaxInt64).Multiply(2); err == nil {
		t.Fatalf("Multiply() overflow error = nil, want error")
	}
}
