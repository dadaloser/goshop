package errors

import "reflect"

const maxErrorTreeNodes = 1_000

// walkErrors visits every leaf and wrapper in an error tree. It supports both
// Go error wrapping forms and this package's legacy Aggregate type.
func walkErrors(err error, visit func(error) bool) bool {
	walker := errorTreeWalker{
		seen:      make(map[error]struct{}),
		remaining: maxErrorTreeNodes,
	}
	return walkErrorTree(err, visit, &walker)
}

type errorTreeWalker struct {
	seen      map[error]struct{}
	remaining int
}

func walkErrorTree(err error, visit func(error) bool, walker *errorTreeWalker) bool {
	if err == nil {
		return false
	}
	if walker.remaining == 0 {
		return false
	}
	walker.remaining--
	if reflect.TypeOf(err).Comparable() {
		if _, ok := walker.seen[err]; ok {
			return false
		}
		walker.seen[err] = struct{}{}
	}

	if aggregate, ok := err.(Aggregate); ok {
		for _, nested := range aggregate.Errors() {
			if walkErrorTree(nested, visit, walker) {
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
			if walkErrorTree(nested, visit, walker) {
				return true
			}
		}
	case interface{ Unwrap() error }:
		return walkErrorTree(unwrapper.Unwrap(), visit, walker)
	}

	return false
}
