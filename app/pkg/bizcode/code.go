package bizcode

import (
	"goshop/pkg/errors"
	"net/http"

	"github.com/novalagung/gubrak"
)

type ErrCode struct {
	//错误码
	C int

	//http的状态码
	HTTP int

	//扩展字段
	Ext string

	//引用文档
	Ref string

	// Kind is the protocol-independent public classification.
	K errors.Kind
}

func (e ErrCode) HTTPStatus() int {
	return e.HTTP
}

func (e ErrCode) String() string {
	return e.Ext
}

func (e ErrCode) Reference() string {
	return e.Ref
}

func (e ErrCode) Code() int {
	if e.C == 0 {
		return http.StatusInternalServerError
	}
	return e.C
}

// Kind returns the protocol-independent classification of the legacy code.
func (e ErrCode) Kind() errors.Kind {
	return e.K
}

func register(code int, httpStatus int, kind errors.Kind, message string, refs ...string) {
	found, _ := gubrak.Includes([]int{200, 400, 401, 403, 404, 409, 429, 500, 503}, httpStatus)
	if !found {
		panic("http code not in `200, 400, 401, 403, 404, 500`")
	}
	var ref string
	if len(refs) > 0 {
		ref = refs[0]
	}
	coder := ErrCode{
		C:    code,
		HTTP: httpStatus,
		K:    kind,
		Ext:  message,
		Ref:  ref,
	}

	errors.MustRegister(coder)
}

var _ errors.Coder = (*ErrCode)(nil)
