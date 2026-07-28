package errors

// walkErrors visits every leaf and wrapper in an error tree. It supports both
// Go error wrapping forms and this package's legacy Aggregate type.
func walkErrors(err error, visit func(error) bool) bool {
	if err == nil {
		return false
	}

	if aggregate, ok := err.(Aggregate); ok {
		for _, nested := range aggregate.Errors() {
			if walkErrors(nested, visit) {
				return true
			}
		}
		return false
	}

	if visit(err) {
		return true
	}

	switch unwrapper := err.(type) {
	case interface{ Unwrap() []error }:
		for _, nested := range unwrapper.Unwrap() {
			if walkErrors(nested, visit) {
				return true
			}
		}
	case interface{ Unwrap() error }:
		return walkErrors(unwrapper.Unwrap(), visit)
	}

	return false
}
