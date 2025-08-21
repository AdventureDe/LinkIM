package handler

import (
	"log"
	"net/http"

	"github.com/AdventureDe/LinkIM/message/repo"
	"github.com/AdventureDe/LinkIM/message/service"

	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	service *service.MessageService
}

func NewMessageHandler(s *service.MessageService) *MessageHandler {
	return &MessageHandler{
		service: s,
	}
}

func (h *MessageHandler) SendMessageToSingle(c *gin.Context) {
	var input struct {
		UserId           int64  `gorm:"column:user_id" json:"user_id"`
		TheOtherPersonId int64  `gorm:"column:the_other_person_id" json:"the_other_person_id"`
		Text             string `gorm:"column:text" json:"text"`
		Platform         int    `gorm:"column:platform" json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if len(input.Text) > 200 {
		c.JSON(416, gin.H{"code": 1, "error": "文本长度超过200！"})
		return
	}
	var lastMsgId *int64
	var err error
	lastMsgId, err = h.service.SendMessageToSingle(c.Request.Context(), input.UserId, input.TheOtherPersonId, input.Text)
	if err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "send message ok", "lastMsgId": lastMsgId})
}

func (h *MessageHandler) GetConversationMessages(c *gin.Context) {
	var input struct {
		UserId           int64 `gorm:"column:user_id" json:"user_id"`
		TheOtherPersonId int64 `gorm:"column:the_other_person_id" json:"the_other_person_id"`
		Platform         int   `gorm:"column:platform" json:"platform"`
		LastMsgId        int64 `gorm:"column:last_msg_id" json:"last_msg_id"`
		PageSize         int   `gorm:"column:page_size" json:"page_size"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	var messages *repo.ConversationMessages
	var err error
	messages, err = h.service.GetConversationMessages(c.Request.Context(), input.UserId, input.TheOtherPersonId, input.LastMsgId, input.PageSize)
	if err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if messages == nil {
		// 可以返回空切片或空对象，避免 panic
		c.JSON(200, gin.H{
			"code":    0,
			"message": "load message ok, message is null",
			// 假设是切片类型，返回空切片
		})
		return
	}
	c.JSON(200, gin.H{
		"code":   0,
		"messge": "load message ok",
		"detail": messages,
	})
}

func (h *MessageHandler) WithdrawMessage(c *gin.Context) {
	var input struct {
		UserId           int64 `gorm:"column:user_id" json:"user_id"`
		TheOtherPersonId int64 `gorm:"column:the_other_person_id" json:"the_other_person_id"`
		Platform         int   `gorm:"column:platform" json:"platform"`
		MessageId        int64 `gorm:"column:id" json:"message_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	var lastMsgId int64
	var err error
	lastMsgId, err = h.service.WithdrawMessage(c.Request.Context(), input.UserId, input.TheOtherPersonId, input.MessageId)
	if err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if lastMsgId == -1 {
		c.JSON(400, gin.H{"code": 1, "error": "load wrong"})
	}
	c.JSON(200, gin.H{
		"code":      0,
		"message":   "withdraw ok",
		"lastMsgId": lastMsgId,
	})
}

func (h *MessageHandler) UnWithdrawMessage(c *gin.Context) {
	var input struct {
		UserId           int64  `gorm:"column:user_id" json:"user_id"`
		TheOtherPersonId int64  `gorm:"column:the_other_person_id" json:"the_other_person_id"`
		Platform         int    `gorm:"column:platform" json:"platform"`
		MessageId        int64  `gorm:"column:id" json:"message_id"`
		NewText          string `gorm:"column:text" json:"new_text"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	var lastMsgId int64
	var err error
	lastMsgId, err = h.service.UnWithdrawMessage(c.Request.Context(), input.UserId, input.TheOtherPersonId, input.MessageId, input.NewText)
	if err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if lastMsgId == -1 {
		c.JSON(400, gin.H{"code": 1, "error": "load wrong"})
	}
	c.JSON(200, gin.H{
		"code":      0,
		"message":   "unwithdraw ok",
		"lastMsgId": lastMsgId,
	})
}

func (h *MessageHandler) GetConversations(c *gin.Context) {
	var input struct {
		UserId   int64 `gorm:"column:user_id" form:"user_id"`
		Platform int   `gorm:"column:platform" form:"platform"`
	}
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	conversations, err := h.service.GetConversations(c.Request.Context(), input.UserId)

	if err != nil {
		log.Printf("GetConversations error: %v", err)
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}

	if conversations == nil {
		log.Printf("GetConversations error: %v", err)
		c.JSON(400, gin.H{"code": 1, "error": "conversation is nil"})
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "get conversations ok",
		"detail":  conversations,
	})
}
