package user

import (
	"net/http"

	"goshop/app/goshop/api/internal/captcha"
	"goshop/pkg/log"

	"github.com/gin-gonic/gin"
)

func GetCaptcha(ctx *gin.Context) {
	cp := captcha.NewDigitCaptcha()
	id, b64s, _, err := cp.Generate()
	if err != nil {
		log.Errorf("生成验证码错误: %s", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"msg": "生成验证码错误",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"captchaId": id,
		"picPath":   b64s,
	})
}
