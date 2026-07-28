package errors

import "testing"

func TestMustRegisterAllowsIdenticalDuplicate(t *testing.T) {
	restoreCodes(t)

	coder := defaultCoder{
		C:   990001,
		K:   KindInvalidArgument,
		Ext: "duplicate coder",
		Ref: "https://example.test/duplicate",
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

func TestCatalogRegisterAllIsIdempotent(t *testing.T) {
	const code = 991001
	catalog := Catalog{{
		Code:      code,
		Kind:      KindUnavailable,
		Message:   "catalog dependency unavailable",
		Reference: "https://example.test/errors/dependency-unavailable",
	}}

	catalog.RegisterAll()
	catalog.RegisterAll()

	got := SpecForCode(code)
	want := catalog[0]
	if got != want {
		t.Fatalf("SpecForCode(%d) = %#v, want %#v", code, got, want)
	}
}

func TestMustRegisterPanicsOnConflictingDuplicate(t *testing.T) {
	restoreCodes(t)

	MustRegister(defaultCoder{
		C:   990002,
		K:   KindInvalidArgument,
		Ext: "first coder",
		Ref: "https://example.test/first",
	})

	defer func() {
		if recover() == nil {
			t.Fatal("MustRegister() did not panic for conflicting duplicate coder")
		}
	}()

	MustRegister(defaultCoder{
		C:   990002,
		K:   KindInternal,
		Ext: "second coder",
		Ref: "https://example.test/second",
	})
}

func TestSameCoder(t *testing.T) {
	left := defaultCoder{
		C:   990003,
		K:   KindPermissionDenied,
		Ext: "same coder",
		Ref: "https://example.test/same",
	}
	right := defaultCoder{
		C:   990003,
		K:   KindPermissionDenied,
		Ext: "same coder",
		Ref: "https://example.test/same",
	}
	other := defaultCoder{
		C:   990003,
		K:   KindPermissionDenied,
		Ext: "different coder",
		Ref: "https://example.test/same",
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
