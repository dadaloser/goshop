package errors

import "reflect"

const maxErrorTreeNodes = 1_000

// walkErrors visits every leaf and wrapper in an error tree. It supports both
// Go error wrapping forms and this package's legacy Aggregate type.
func walkErrors(err error, visit func(error) bool) bool {
	return walkErrorTree(err, visit, make(map[error]struct{}), 0)
}

func walkErrorTree(err error, visit func(error) bool, seen map[error]struct{}, visited int) bool {
	if err == nil {
		return false
	}
	if visited >= maxErrorTreeNodes {
		return false
	}
	visited++
	if reflect.TypeOf(err).Comparable() {
		if _, ok := seen[err]; ok {
			return false
		}
		seen[err] = struct{}{}
	}

	if aggregate, ok := err.(Aggregate); ok {
		for _, nested := range aggregate.Errors() {
			if walkErrorTree(nested, visit, seen, visited) {
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
			if walkErrorTree(nested, visit, seen, visited) {
				return true
			}
		}
	case interface{ Unwrap() error }:
		return walkErrorTree(unwrapper.Unwrap(), visit, seen, visited)
	}

	return false
}
