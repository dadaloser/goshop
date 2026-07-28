package errors

/*
WARNING - changing the line numbers in this file will break the
examples.
*/

import (
	"fmt"
)

const (
	// Error codes below 1000 are reserved future use by the
	// "github.com/bdlm/errors" package.
	ConfigurationNotValid int = iota + 1000
	ErrInvalidJSON
	ErrEOF
	ErrLoadConfigFailed
)

func init() {
	MustRegister(Spec{Code: ConfigurationNotValid, Kind: KindInternal, Message: "ConfigurationNotValid error"})
	MustRegister(Spec{Code: ErrInvalidJSON, Kind: KindInternal, Message: "Data is not valid JSON"})
	MustRegister(Spec{Code: ErrEOF, Kind: KindInternal, Message: "End of input"})
	MustRegister(Spec{Code: ErrLoadConfigFailed, Kind: KindInternal, Message: "Load configuration file failed"})
}

func loadConfig() error {
	err := decodeConfig()
	return WrapCode(err, ConfigurationNotValid, "service configuration could not be loaded")
}

func decodeConfig() error {
	err := readConfig()
	return WrapCode(err, ErrInvalidJSON, "could not decode configuration data")
}

func readConfig() error {
	err := fmt.Errorf("read: end of input")
	return WrapCode(err, ErrEOF, "could not read configuration file")
}
