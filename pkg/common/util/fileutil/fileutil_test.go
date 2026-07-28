package fileutil

import "testing"

func TestGetParentEmptyPath(t *testing.T) {
	if got := GetParent(""); got != nil {
		t.Fatalf("GetParent(\"\") = %q, want nil", *got)
	}
}
