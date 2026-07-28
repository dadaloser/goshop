package errors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type formatInfo struct {
	code    int
	message string
	err     string
	stack   *stack
}

// Format renders an error specification safely by default and includes
// diagnostics only when an explicit formatting flag is requested.
func (w *withSpec) Format(state fmt.State, verb rune) {
	switch verb {
	case 'v':
		str := bytes.NewBuffer(nil)
		jsonData := []map[string]interface{}{}
		flagDetail := state.Flag('-')
		flagTrace := state.Flag('+')
		modeJSON := state.Flag('#')

		sep := ""
		errs := list(w)
		for k, err := range errs {
			finfo := buildFormatInfo(err)
			jsonData, str = format(len(errs)-k-1, jsonData, str, finfo, sep, flagDetail, flagTrace, modeJSON)
			sep = "; "
			if !flagTrace {
				break
			}
		}
		if modeJSON {
			byts, _ := json.Marshal(jsonData)
			str.Write(byts)
		}
		_, _ = fmt.Fprint(state, strings.Trim(str.String(), "\r\n\t"))
	default:
		_, _ = fmt.Fprint(state, w.spec.Message)
	}
}

func format(k int, jsonData []map[string]interface{}, str *bytes.Buffer, finfo *formatInfo, sep string, detail, trace, modeJSON bool) ([]map[string]interface{}, *bytes.Buffer) {
	if modeJSON {
		data := map[string]interface{}{"error": finfo.message}
		if detail || trace {
			data = map[string]interface{}{"message": finfo.message, "code": finfo.code, "error": finfo.err}
			caller := fmt.Sprintf("#%d", k)
			if finfo.stack != nil {
				f := Frame((*finfo.stack)[0])
				caller = fmt.Sprintf("%s %s:%d (%s)", caller, f.file(), f.line(), f.name())
			}
			data["caller"] = caller
		}
		return append(jsonData, data), str
	}

	if detail || trace {
		if finfo.stack != nil {
			f := Frame((*finfo.stack)[0])
			fmt.Fprintf(str, "%s%s - #%d [%s:%d (%s)] (%d) %s", sep, finfo.err, k, f.file(), f.line(), f.name(), finfo.code, finfo.message)
		} else {
			fmt.Fprintf(str, "%s%s - #%d %s", sep, finfo.err, k, finfo.message)
		}
	} else {
		fmt.Fprint(str, finfo.message)
	}
	return jsonData, str
}

func list(err error) []error {
	if err == nil {
		return nil
	}
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		return append([]error{err}, list(unwrapper.Unwrap())...)
	}
	return []error{err}
}

func buildFormatInfo(err error) *formatInfo {
	switch value := err.(type) {
	case *fundamental:
		return &formatInfo{code: unknownCoder.Code(), message: value.msg, err: value.msg, stack: value.stack}
	case *withStack:
		return &formatInfo{code: unknownCoder.Code(), message: value.Error(), err: value.Error(), stack: value.stack}
	case *withSpec:
		message := value.spec.Message
		if message == "" {
			message = value.Error()
		}
		return &formatInfo{code: value.spec.Code, message: message, err: value.diagnostic(), stack: value.stack}
	default:
		return &formatInfo{code: unknownCoder.Code(), message: err.Error(), err: err.Error()}
	}
}
