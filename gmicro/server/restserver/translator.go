package restserver

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"

	"reflect"
	"strings"

	"goshop/gmicro/server/restserver/validation"

	en_translations "github.com/go-playground/validator/v10/translations/en"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

var globalTranslations struct {
	sync.Once
	translators map[string]ut.Translator
	err         error
}

func (s *Server) initTrans(locale string) error {
	globalTranslations.Do(initGlobalTranslations)
	if globalTranslations.err != nil {
		return globalTranslations.err
	}
	translator, ok := globalTranslations.translators[locale]
	if !ok {
		return fmt.Errorf("unsupported translator locale %q", locale)
	}
	s.trans = translator
	return nil
}

func initGlobalTranslations() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		globalTranslations.err = fmt.Errorf("gin validator engine is unavailable")
		return
	}

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	zhT := zh.New()
	enT := en.New()
	uni := ut.New(enT, zhT, enT)
	zhTranslator, ok := uni.GetTranslator("zh")
	if !ok {
		globalTranslations.err = fmt.Errorf("get zh translator")
		return
	}
	enTranslator, ok := uni.GetTranslator("en")
	if !ok {
		globalTranslations.err = fmt.Errorf("get en translator")
		return
	}
	if err := zh_translations.RegisterDefaultTranslations(v, zhTranslator); err != nil {
		globalTranslations.err = err
		return
	}
	if err := en_translations.RegisterDefaultTranslations(v, enTranslator); err != nil {
		globalTranslations.err = err
		return
	}
	validation.RegisterMobile(zhTranslator)
	validation.RegisterMobile(enTranslator)
	globalTranslations.translators = map[string]ut.Translator{
		"zh": zhTranslator,
		"en": enTranslator,
	}
}
