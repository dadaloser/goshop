package errors

import (
	stderrors "errors"
	"testing"
)

func TestRegisterAllowsIdenticalDuplicate(t *testing.T) {
	restoreCodes(t)

	coder := defaultCoder{
		C:   990001,
		K:   KindInvalidArgument,
		Ext: "duplicate coder",
		Ref: "https://example.test/duplicate",
	}

	Register(coder)

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

func TestCatalogRegisterAllPanicsForInvalidSpec(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Catalog.RegisterAll() did not panic for invalid specification")
		}
	}()
	Catalog{{Code: 0, Kind: KindInternal, Message: "invalid"}}.RegisterAll()
}

type cyclicError struct{}

func (cyclicError) Error() string { return "cyclic error" }
func (cyclicError) Unwrap() error { return cyclicError{} }

func TestIsCodeStopsOnCyclicError(t *testing.T) {
	if IsCode(cyclicError{}, 991003) {
		t.Fatal("IsCode() = true for cyclic error without code")
	}
}

func TestIsCodeFindsCodesInJoinedAndAggregatedErrors(t *testing.T) {
	const code = 991002
	err := NewSpec(Spec{Code: code, Kind: KindUnavailable, Message: "dependency unavailable"}, "dial tcp: refused")

	if !IsCode(stderrors.Join(stderrors.New("other failure"), err), code) {
		t.Fatal("IsCode() did not find code in errors.Join branch")
	}
	if !IsCode(NewAggregate([]error{stderrors.New("other failure"), err}), code) {
		t.Fatal("IsCode() did not find code in Aggregate branch")
	}
}

func TestRegisterPanicsOnConflictingDuplicate(t *testing.T) {
	restoreCodes(t)

	Register(defaultCoder{
		C:   990002,
		K:   KindInvalidArgument,
		Ext: "first coder",
		Ref: "https://example.test/first",
	})

	defer func() {
		if recover() == nil {
			t.Fatal("Register() did not panic for conflicting duplicate coder")
		}
	}()

	Register(defaultCoder{
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
