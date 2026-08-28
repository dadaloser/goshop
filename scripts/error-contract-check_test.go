package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestCreatesDatabaseCodeWithoutCause(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{
			name: "direct error text",
			src:  "errors.NewCode(errcode.ErrDatabase, err.Error())",
			want: true,
		},
		{
			name: "multiline formatted error text",
			src: "errors.NewCode(\n" +
				"errcode.ErrDatabase,\n" +
				"fmt.Sprintf(\"query: %v\", err),\n" +
				")",
			want: true,
		},
		{
			name: "static diagnostic",
			src:  "errors.NewCode(errcode.ErrDatabase, \"database unavailable\")",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createsDatabaseCodeWithoutCause(parseCall(t, tt.src)); got != tt.want {
				t.Errorf("createsDatabaseCodeWithoutCause(%s) = %t, want %t", tt.src, got, tt.want)
			}
		})
	}
}

func TestClassifiesErrorByText(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{
			name: "wrapped error text",
			src:  "strings.Contains(strings.ToLower(err.Error()), \"unique\")",
			want: true,
		},
		{
			name: "ordinary string",
			src:  "strings.Contains(value, \"unique\")",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifiesErrorByText(parseCall(t, tt.src)); got != tt.want {
				t.Errorf("classifiesErrorByText(%s) = %t, want %t", tt.src, got, tt.want)
			}
		})
	}
}

func parseCall(t *testing.T, expression string) *ast.CallExpr {
	t.Helper()
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "check.go", "package check\nfunc check() { "+expression+" }", 0)
	if err != nil {
		t.Fatalf("parseCall(%s) error = %v", expression, err)
	}

	var call *ast.CallExpr
	ast.Inspect(file, func(node ast.Node) bool {
		if candidate, ok := node.(*ast.CallExpr); ok && call == nil {
			call = candidate
		}
		return call == nil
	})
	if call == nil {
		t.Fatalf("parseCall(%s) = nil", expression)
	}
	return call
}
