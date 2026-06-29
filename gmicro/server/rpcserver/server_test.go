package rpcserver

import "testing"

func TestNewServerEReturnsListenError(t *testing.T) {
	_, err := NewServerE(WithAddress("127.0.0.1:-1"))
	if err == nil {
		t.Fatal("NewServerE() error = nil, want listen error")
	}
}
