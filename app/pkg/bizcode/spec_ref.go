package bizcode

import "goshop/pkg/errors"

// codeSpec is a code-only reference. NewSpec resolves it through the
// application catalog and never uses local Kind or Message metadata.
func codeSpec(code int) errors.Spec { return errors.Spec{Code: code} }
