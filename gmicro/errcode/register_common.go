package errcode

import "goshop/pkg/errors"

// Catalog contains the framework-level public error contracts. Registration is
// explicit: applications must call RegisterAll during startup.
var commonCatalog = errors.Catalog{
	{Code: ErrUnknown, Kind: errors.KindInternal, Message: "Internal server error"},
	{Code: ErrBind, Kind: errors.KindInvalidArgument, Message: "Error occurred while binding the request body to the struct"},
	{Code: ErrValidation, Kind: errors.KindInvalidArgument, Message: "Validation failed"},
	{Code: ErrTokenInvalid, Kind: errors.KindUnauthenticated, Message: "Token invalid"},
	{Code: ErrPageNotFound, Kind: errors.KindNotFound, Message: "Page not found"},
	{Code: ErrConflict, Kind: errors.KindConflict, Message: "Request conflicts with the current resource state"},
	{Code: ErrServiceUnavailable, Kind: errors.KindUnavailable, Message: "Required service is temporarily unavailable"},
	{Code: ErrTimeout, Kind: errors.KindTimeout, Message: "Required service response timed out"},
}

// Catalog is the complete framework error-code catalog.
var Catalog = append(commonCatalog,
	errors.Spec{Code: ErrDatabase, Kind: errors.KindUnavailable, Message: "Database error"},
	errors.Spec{Code: ErrEncrypt, Kind: errors.KindInternal, Message: "Error occurred while encrypting the user password"},
	errors.Spec{Code: ErrSignatureInvalid, Kind: errors.KindUnauthenticated, Message: "Signature is invalid"},
	errors.Spec{Code: ErrExpired, Kind: errors.KindUnauthenticated, Message: "Token expired"},
	errors.Spec{Code: ErrInvalidAuthHeader, Kind: errors.KindUnauthenticated, Message: "Invalid authorization header"},
	errors.Spec{Code: ErrMissingHeader, Kind: errors.KindUnauthenticated, Message: "The `Authorization` header was empty"},
	errors.Spec{Code: ErrPasswordIncorrect, Kind: errors.KindUnauthenticated, Message: "Password was incorrect"},
	errors.Spec{Code: ErrPermissionDenied, Kind: errors.KindPermissionDenied, Message: "Permission denied"},
	errors.Spec{Code: ErrEncodingFailed, Kind: errors.KindInternal, Message: "Encoding failed due to an error with the data"},
	errors.Spec{Code: ErrDecodingFailed, Kind: errors.KindInternal, Message: "Decoding failed due to an error with the data"},
	errors.Spec{Code: ErrInvalidJSON, Kind: errors.KindInvalidArgument, Message: "Data is not valid JSON"},
	errors.Spec{Code: ErrEncodingJSON, Kind: errors.KindInternal, Message: "JSON data could not be encoded"},
	errors.Spec{Code: ErrDecodingJSON, Kind: errors.KindInvalidArgument, Message: "JSON data could not be decoded"},
	errors.Spec{Code: ErrInvalidYaml, Kind: errors.KindInvalidArgument, Message: "Data is not valid Yaml"},
	errors.Spec{Code: ErrEncodingYaml, Kind: errors.KindInternal, Message: "Yaml data could not be encoded"},
	errors.Spec{Code: ErrDecodingYaml, Kind: errors.KindInvalidArgument, Message: "Yaml data could not be decoded"},
)

// RegisterAll explicitly adds every framework error contract to the shared
// errors catalog. It is safe to call more than once.
func RegisterAll() { Catalog.RegisterAll() }
