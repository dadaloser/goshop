package controller

import (
	"goshop/pkg/log"

	"github.com/gin-gonic/gin"
)

func (us *userServer) List(ctx *gin.Context) {
	log.Info("GetUserList is called")
}
