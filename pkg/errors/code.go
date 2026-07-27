package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"sync"
)

var (
	unknownCoder defaultCoder = defaultCoder{1, http.StatusInternalServerError, "An internal server error occurred", "http://goshop/pkg/errors/README.md"}
)

// Coder defines an interface for an error code detail information.
type Coder interface {
	// HTTP status that should be used for the associated error code.
	HTTPStatus() int

	// External (user) facing error text.
	String() string

	// Reference returns the detail documents for user.
	Reference() string

	// Code returns the code of the coder
	Code() int
}

type defaultCoder struct {
	// C refers to the integer code of the ErrCode.
	C int

	// HTTP status that should be used for the associated error code.
	HTTP int

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

// HTTPStatus returns the associated HTTP status code, if any. Otherwise,
// returns 200.
func (coder defaultCoder) HTTPStatus() int {
	if coder.HTTP == 0 {
		return 500
	}

	return coder.HTTP
}

// Reference returns the reference document.
func (coder defaultCoder) Reference() string {
	return coder.Ref
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

	var coded *withCode
	if stderrors.As(err, &coded) {
		if coder, ok := lookupCoder(coded.code); ok {
			return coder
		}
	}

	return unknownCoder
}

// IsCode reports whether any error in err's chain contains the given error code.
func IsCode(err error, code int) bool {
	for err != nil {
		if coded, ok := err.(*withCode); ok && coded.code == code {
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
		left.HTTPStatus() == right.HTTPStatus() &&
		left.String() == right.String() &&
		left.Reference() == right.Reference()
}

func lookupCoder(code int) (Coder, bool) {
	codeMux.RLock()
	defer codeMux.RUnlock()

	coder, ok := codes[code]
	return coder, ok
}
