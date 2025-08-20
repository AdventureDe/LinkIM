package router

import (
	"message/handler"

	"github.com/gin-gonic/gin"
)

func SetMessageRouter(r *gin.Engine, m *handler.MessageHandler) {
	r.POST("/message/send", m.SendMessageToSingle)
	r.GET("/conversation/get", m.GetConversationMessages)
	r.PUT("/message/withdraw", m.WithdrawMessage)
	r.PUT("/message/unwithdraw", m.UnWithdrawMessage)
}
