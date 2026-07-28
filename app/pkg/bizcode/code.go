package bizcode

import (
	"goshop/pkg/errors"
)

type ErrCode struct {
	//错误码
	C int

	//扩展字段
	Ext string

	//引用文档
	Ref string

	// Kind is the protocol-independent public classification.
	K errors.Kind
}

func (e ErrCode) String() string { return e.Ext }

func (e ErrCode) Reference() string { return e.Ref }

func (e ErrCode) Code() int { return e.C }

// Kind returns the protocol-independent public classification.
func (e ErrCode) Kind() errors.Kind { return e.K }

func register(code int, kind errors.Kind, message string, refs ...string) {
	var ref string
	if len(refs) > 0 {
		ref = refs[0]
	}
	coder := ErrCode{
		C:   code,
		K:   kind,
		Ext: message,
		Ref: ref,
	}

	errors.MustRegister(coder)
}

var _ errors.Coder = (*ErrCode)(nil)
