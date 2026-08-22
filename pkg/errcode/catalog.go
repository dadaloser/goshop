package errcode

// Contract describes a framework-level public error contract without coupling
// error codes to an application error implementation.
type Contract struct {
	Code    int
	Kind    string
	Message string
}

// Catalog is the complete framework error-code catalog.
var Catalog = []Contract{
	{Code: ErrUnknown, Kind: "internal", Message: "Internal server error"},
	{Code: ErrBind, Kind: "invalid_argument", Message: "Error occurred while binding the request body to the struct"},
	{Code: ErrValidation, Kind: "invalid_argument", Message: "Validation failed"},
	{Code: ErrTokenInvalid, Kind: "unauthenticated", Message: "Token invalid"},
	{Code: ErrPageNotFound, Kind: "not_found", Message: "Page not found"},
	{Code: ErrConflict, Kind: "conflict", Message: "Request conflicts with the current resource state"},
	{Code: ErrServiceUnavailable, Kind: "unavailable", Message: "Required service is temporarily unavailable"},
	{Code: ErrTimeout, Kind: "timeout", Message: "Required service response timed out"},
	{Code: ErrDatabase, Kind: "unavailable", Message: "Database error"},
	{Code: ErrEncrypt, Kind: "internal", Message: "Error occurred while encrypting the user password"},
	{Code: ErrSignatureInvalid, Kind: "unauthenticated", Message: "Signature is invalid"},
	{Code: ErrExpired, Kind: "unauthenticated", Message: "Token expired"},
	{Code: ErrInvalidAuthHeader, Kind: "unauthenticated", Message: "Invalid authorization header"},
	{Code: ErrMissingHeader, Kind: "unauthenticated", Message: "The `Authorization` header was empty"},
	{Code: ErrPasswordIncorrect, Kind: "unauthenticated", Message: "Password was incorrect"},
	{Code: ErrPermissionDenied, Kind: "permission_denied", Message: "Permission denied"},
	{Code: ErrEncodingFailed, Kind: "internal", Message: "Encoding failed due to an error with the data"},
	{Code: ErrDecodingFailed, Kind: "internal", Message: "Encoding failed due to an error with the data"},
	{Code: ErrInvalidJSON, Kind: "invalid_argument", Message: "Data is not valid JSON"},
	{Code: ErrEncodingJSON, Kind: "internal", Message: "JSON data could not be encoded"},
	{Code: ErrDecodingJSON, Kind: "invalid_argument", Message: "JSON data could not be decoded"},
	{Code: ErrInvalidYaml, Kind: "invalid_argument", Message: "Data is not valid Yaml"},
	{Code: ErrEncodingYaml, Kind: "internal", Message: "Yaml data could not be encoded"},
	{Code: ErrDecodingYaml, Kind: "invalid_argument", Message: "Yaml data could not be decoded"},
}
