package captcha

import "github.com/mojocn/base64Captcha"

var store = base64Captcha.DefaultMemStore

func NewDigitCaptcha() *base64Captcha.Captcha {
	driver := base64Captcha.NewDriverDigit(80, 240, 5, 0.7, 80)
	return base64Captcha.NewCaptcha(driver, store)
}

func Verify(id, answer string, clear bool) bool {
	return store.Verify(id, answer, clear)
}
