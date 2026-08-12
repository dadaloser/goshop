package errors

import (
	stderrors "errors"
	"testing"
)

func TestRegisterAllowsIdenticalDuplicate(t *testing.T) {
	restoreCodes(t)

	coder := Spec{
		Code:      990001,
		Kind:      KindInvalidArgument,
		Message:   "duplicate coder",
		Reference: "https://example.test/duplicate",
	}

	MustRegister(coder)

	got, ok := codes[coder.Code]
	if !ok {
		t.Fatalf("codes[%d] missing after duplicate registration", coder.Code)
	}
	gotSpec := Spec{Code: got.Code(), Kind: got.Kind(), Message: got.String(), Reference: got.Reference()}
	if gotSpec != coder {
		t.Fatalf("codes[%d] = %#v, want %#v", coder.Code, gotSpec, coder)
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

func TestMustRegisterPanicsForInvalidSpec(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("MustRegister() did not panic for invalid specification")
		}
	}()
	MustRegister(Spec{Code: 991004, Message: "missing kind"})
}

type cyclicError struct{}

func (cyclicError) Error() string { return "cyclic error" }
func (cyclicError) Unwrap() error { return cyclicError{} }

func TestIsCodeStopsOnCyclicError(t *testing.T) {
	if IsCode(cyclicError{}, 991003) {
		t.Fatal("IsCode() = true for cyclic error without code")
	}
}

func TestWalkErrorsCapsEntireJoinedTree(t *testing.T) {
	branches := make([]error, maxErrorTreeNodes+10)
	for i := range branches {
		branches[i] = stderrors.New("branch")
	}

	if got := len(list(stderrors.Join(branches...))); got != maxErrorTreeNodes {
		t.Fatalf("list(joined tree) length = %d, want %d", got, maxErrorTreeNodes)
	}
}

func TestIsCodeFindsCodesInJoinedAndAggregatedErrors(t *testing.T) {
	const code = 991002
	spec := Spec{Code: code, Kind: KindUnavailable, Message: "dependency unavailable"}
	MustRegister(spec)
	err := NewSpec(spec, "dial tcp: refused")

	if !IsCode(stderrors.Join(stderrors.New("other failure"), err), code) {
		t.Fatal("IsCode() did not find code in errors.Join branch")
	}
	if !IsCode(NewAggregate([]error{stderrors.New("other failure"), err}), code) {
		t.Fatal("IsCode() did not find code in Aggregate branch")
	}
}

func TestRegisterPanicsOnConflictingDuplicate(t *testing.T) {
	restoreCodes(t)

	MustRegister(Spec{
		Code:      990002,
		Kind:      KindInvalidArgument,
		Message:   "first coder",
		Reference: "https://example.test/first",
	})

	defer func() {
		if recover() == nil {
			t.Fatal("MustRegister() did not panic for conflicting duplicate coder")
		}
	}()

	MustRegister(Spec{
		Code:      990002,
		Kind:      KindInternal,
		Message:   "second coder",
		Reference: "https://example.test/second",
	})
}

func TestCatalogValidateRejectsDuplicateCode(t *testing.T) {
	catalog := Catalog{
		{Code: 991003, Kind: KindInternal, Message: "first"},
		{Code: 991003, Kind: KindInternal, Message: "first"},
	}
	if err := catalog.Validate(); err == nil {
		t.Fatal("Catalog.Validate() = nil, want duplicate-code error")
	}
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
