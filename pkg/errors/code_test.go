package errors

import (
	"net/http"
	"testing"
)

func TestMustRegisterAllowsIdenticalDuplicate(t *testing.T) {
	restoreCodes(t)

	coder := defaultCoder{
		C:    990001,
		HTTP: http.StatusBadRequest,
		Ext:  "duplicate coder",
		Ref:  "https://example.test/duplicate",
	}

	MustRegister(coder)
	MustRegister(coder)

	got, ok := codes[coder.Code()]
	if !ok {
		t.Fatalf("codes[%d] missing after duplicate registration", coder.Code())
	}
	if !sameCoder(got, coder) {
		t.Fatalf("codes[%d] = %#v, want %#v", coder.Code(), got, coder)
	}
}

func TestMustRegisterPanicsOnConflictingDuplicate(t *testing.T) {
	restoreCodes(t)

	MustRegister(defaultCoder{
		C:    990002,
		HTTP: http.StatusBadRequest,
		Ext:  "first coder",
		Ref:  "https://example.test/first",
	})

	defer func() {
		if recover() == nil {
			t.Fatal("MustRegister() did not panic for conflicting duplicate coder")
		}
	}()

	MustRegister(defaultCoder{
		C:    990002,
		HTTP: http.StatusInternalServerError,
		Ext:  "second coder",
		Ref:  "https://example.test/second",
	})
}

func TestSameCoder(t *testing.T) {
	left := defaultCoder{
		C:    990003,
		HTTP: http.StatusForbidden,
		Ext:  "same coder",
		Ref:  "https://example.test/same",
	}
	right := defaultCoder{
		C:    990003,
		HTTP: http.StatusForbidden,
		Ext:  "same coder",
		Ref:  "https://example.test/same",
	}
	other := defaultCoder{
		C:    990003,
		HTTP: http.StatusForbidden,
		Ext:  "different coder",
		Ref:  "https://example.test/same",
	}

	if !sameCoder(left, right) {
		t.Fatal("sameCoder() = false, want true for identical coders")
	}
	if sameCoder(left, other) {
		t.Fatal("sameCoder() = true, want false for different coders")
	}
}

func restoreCodes(t *testing.T) {
	t.Helper()

	codeMux.Lock()
	snapshot := make(map[int]Coder, len(codes))
	for k, v := range codes {
		snapshot[k] = v
	}
	codeMux.Unlock()

	t.Cleanup(func() {
		codeMux.Lock()
		codes = make(map[int]Coder, len(snapshot))
		for k, v := range snapshot {
			codes[k] = v
		}
		codeMux.Unlock()
	})
}
