package router

import (
	"github.com/AdventureDe/LinkIM/message/handler"

	"github.com/gin-gonic/gin"
)

func SetMessageRouter(r *gin.Engine, m *handler.MessageHandler) {
	r.POST("/message/send", m.SendMessageToSingle)
	r.POST("/message/group/send", m.SendMessageToGroup)
	r.GET("/conversation/get", m.GetConversationMessagesSingle)
	r.GET("/conversation/group/get", m.GetConversationMessagesGroup)
	r.PUT("/message/withdraw", m.WithdrawMessageSingle)
	r.PUT("/message/group/withdraw", m.WithdrawMessageGroup)
	r.PUT("/message/unwithdraw", m.UnWithdrawMessageSingle)
	r.PUT("/message/group/unwithdraw", m.UnWithdrawMessageGroup)
	r.PUT("/conversation/unread", m.UpdateUnread)
	r.GET("/conversations", m.GetConversations)
}
