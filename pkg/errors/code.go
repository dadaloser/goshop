package errors

import (
	"fmt"
	"sync"
)

var (
	unknownCoder defaultCoder = defaultCoder{1, KindInternal, "An internal server error occurred", "http://goshop/pkg/errors/README.md"}
)

// Coder defines an interface for an error code detail information.
type Coder interface {
	// External (user) facing error text.
	String() string

	// Reference returns the detail documents for user.
	Reference() string

	// Code returns the code of the coder
	Code() int

	// Kind returns the protocol-independent public classification.
	Kind() Kind
}

// Catalog is a declarative collection of public error specifications. Error
// domains own their Catalog values; this package owns the only registration
// implementation and the process-wide registry.
type Catalog []Spec

// Validate verifies that every contract is valid and that no code is declared
// more than once within the catalog.
func (catalog Catalog) Validate() error {
	seen := make(map[int]Spec, len(catalog))
	for _, spec := range catalog {
		if err := spec.Validate(); err != nil {
			return err
		}
		if _, ok := seen[spec.Code]; ok {
			return fmt.Errorf("duplicate error specification for code %d", spec.Code)
		}
		seen[spec.Code] = spec
	}
	return nil
}

// RegisterAll registers every specification in the catalog. Re-registering an
// identical specification is safe, which lets multiple application components
// explicitly request their required catalogs without depending on init order.
func (catalog Catalog) RegisterAll() {
	if err := catalog.Validate(); err != nil {
		panic(fmt.Sprintf("invalid error catalog: %v", err))
	}
	for _, spec := range catalog {
		MustRegister(spec)
	}
}

type specCoder struct{ spec Spec }

func (coder specCoder) String() string    { return coder.spec.Message }
func (coder specCoder) Reference() string { return coder.spec.Reference }
func (coder specCoder) Code() int         { return coder.spec.Code }
func (coder specCoder) Kind() Kind        { return coder.spec.Kind }

type defaultCoder struct {
	// C refers to the integer code of the ErrCode.
	C int

	// K is the protocol-independent public classification.
	K Kind

	// External (user) facing error text.
	Ext string

	// Ref specify the reference document.
	Ref string
}

// Code returns the integer code of the coder.
func (coder defaultCoder) Code() int {
	return coder.C

}

// String implements stringer. String returns the external error message,
// if any.
func (coder defaultCoder) String() string {
	return coder.Ext
}

// Reference returns the reference document.
func (coder defaultCoder) Reference() string {
	return coder.Ref
}

// Kind classifies unknown or unregistered codes as internal errors. It keeps
// legacy WithCode callers safe when a code has not yet been registered.
func (coder defaultCoder) Kind() Kind {
	return coder.K
}

// codes contains a map of error codes to metadata.
var codes = map[int]Coder{unknownCoder.Code(): unknownCoder}
var codeMux sync.RWMutex

// MustRegister registers a validated public error specification.
// It will panic for an invalid specification or a conflicting duplicate code.
//
// Error domains should normally declare a Catalog and register it at application
// startup. MustRegister exists for narrowly scoped bootstrap and test setup.
func MustRegister(spec Spec) {
	if err := spec.Validate(); err != nil {
		panic(fmt.Sprintf("invalid error specification: %v", err))
	}
	coder := specCoder{spec: spec}

	codeMux.Lock()
	defer codeMux.Unlock()

	if existing, ok := codes[coder.Code()]; ok {
		if sameCoder(existing, coder) {
			return
		}
		panic(fmt.Sprintf("code: %d already exist", coder.Code()))
	}

	codes[coder.Code()] = coder
}

// ParseCoder returns the registered coder associated with any error in err's chain.
// A nil error returns nil. Errors without a registered code return the unknown coder.
func ParseCoder(err error) Coder {
	if err == nil {
		return nil
	}

	if spec, ok := SpecOf(err); ok {
		if coder, ok := lookupCoder(spec.Code); ok {
			return coder
		}
	}

	return unknownCoder
}

// IsCode reports whether any error in err's chain contains the given error code.
func IsCode(err error, code int) bool {
	return walkErrors(err, func(err error) bool {
		if coded, ok := err.(Coded); ok && coded.Spec().Code == code {
			return true
		}
		return false
	})
}

func sameCoder(left, right Coder) bool {
	if left == nil || right == nil {
		return left == right
	}

	return left.Code() == right.Code() &&
		left.Kind() == right.Kind() &&
		left.String() == right.String() &&
		left.Reference() == right.Reference()
}

func lookupCoder(code int) (Coder, bool) {
	codeMux.RLock()
	defer codeMux.RUnlock()

	coder, ok := codes[code]
	return coder, ok
}

func registeredSpecForCode(code int) (Spec, bool) {
	if coder, ok := lookupCoder(code); ok {
		return Spec{
			Code:      coder.Code(),
			Kind:      coder.Kind(),
			Message:   coder.String(),
			Reference: coder.Reference(),
		}, true
	}
	return Spec{}, false
}
