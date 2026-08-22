// Package errcode defines protocol-independent framework error codes.
package errcode

//go:generate codegen -type=int -doc -output ./error_code_generated.md

// Common: basic errors.
// Code must start with 1xxxxx.
const (
	// ErrSuccess - 200: OK.
	ErrSuccess = iota + 100001
	// ErrUnknown - 500: Internal server error.
	ErrUnknown
	// ErrBind - 400: Error occurred while binding the request body to the struct.
	ErrBind
	// ErrValidation - 400: Validation failed.
	ErrValidation
	// ErrTokenInvalid - 401: Token invalid.
	ErrTokenInvalid
	// ErrPageNotFound - 404: Page not found.
	ErrPageNotFound
	// ErrConflict - 409: Request conflicts with the current resource state.
	ErrConflict
	// ErrServiceUnavailable - 503: A required downstream service is unavailable.
	ErrServiceUnavailable
	// ErrTimeout - 504: A required downstream service did not respond in time.
	ErrTimeout
)

const (
	// ErrDatabase - 500: Database error.
	ErrDatabase int = iota + 100101
)

const (
	// ErrEncrypt - 401: Error occurred while encrypting the user password.
	ErrEncrypt int = iota + 100201
	// ErrSignatureInvalid - 401: Signature is invalid.
	ErrSignatureInvalid
	// ErrExpired - 401: Token expired.
	ErrExpired
	// ErrInvalidAuthHeader - 401: Invalid authorization header.
	ErrInvalidAuthHeader
	// ErrMissingHeader - 401: The `Authorization` header was empty.
	ErrMissingHeader
	// ErrPasswordIncorrect - 401: Password was incorrect.
	ErrPasswordIncorrect
	// ErrPermissionDenied - 403: Permission denied.
	ErrPermissionDenied
)

const (
	// ErrEncodingFailed - 500: Encoding failed due to an error with the data.
	ErrEncodingFailed int = iota + 100301
	// ErrDecodingFailed - 500: Decoding failed due to an error with the data.
	ErrDecodingFailed
	// ErrInvalidJSON - 500: Data is not valid JSON.
	ErrInvalidJSON
	// ErrEncodingJSON - 500: JSON data could not be encoded.
	ErrEncodingJSON
	// ErrDecodingJSON - 500: JSON data could not be decoded.
	ErrDecodingJSON
	// ErrInvalidYaml - 500: Data is not valid Yaml.
	ErrInvalidYaml
	// ErrEncodingYaml - 500: Yaml data could not be encoded.
	ErrEncodingYaml
	// ErrDecodingYaml - 500: Yaml data could not be decoded.
	ErrDecodingYaml
)
