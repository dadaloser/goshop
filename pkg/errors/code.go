package errors

import (
	stderrors "errors"
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
var codes = map[int]Coder{}
var codeMux sync.RWMutex

// Register register a user define error code.
// It will overrid the exist code.
func Register(coder Coder) {
	if coder.Code() == 0 {
		panic("code `0` is reserved by `goshop/pkg/errors` as unknownCode error code")
	}

	codeMux.Lock()
	defer codeMux.Unlock()

	codes[coder.Code()] = coder
}

// MustRegister register a user define error code.
// It will panic when the same Code already exist.
func MustRegister(coder Coder) {
	if coder.Code() == 0 {
		panic("code '0' is reserved by 'goshop/pkg/errors' as ErrUnknown error code")
	}

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

	var specified Coded
	if stderrors.As(err, &specified) {
		if coder, ok := lookupCoder(specified.Spec().Code); ok {
			return coder
		}
	}

	return unknownCoder
}

// IsCode reports whether any error in err's chain contains the given error code.
func IsCode(err error, code int) bool {
	for err != nil {
		if coded, ok := err.(Coded); ok && coded.Spec().Code == code {
			return true
		}

		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrapper.Unwrap()
	}

	return false
}

func init() {
	codes[unknownCoder.Code()] = unknownCoder
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
