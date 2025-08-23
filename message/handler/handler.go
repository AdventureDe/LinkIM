package handler

import (
	"log"
	"net/http"

	"github.com/AdventureDe/LinkIM/message/dto"
	"github.com/AdventureDe/LinkIM/message/service"
	"github.com/google/uuid"

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

func (h *MessageHandler) SendMessageToGroup(c *gin.Context) {
	var input struct {
		UserId   int64     `json:"user_id"`
		GroupId  uuid.UUID `json:"group_id"`
		Text     string    `json:"text"`
		Platform int       `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if len(input.Text) > 200 {
		c.JSON(416, gin.H{"code": 1, "error": "文本长度超过200!"})
		return
	}
	var lastMsgId *int64
	var err error
	lastMsgId, err = h.service.SendMessageToGroup(c.Request.Context(), input.UserId, input.GroupId, input.Text)
	if err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "send message ok", "lastMsgId": lastMsgId})
}

func (h *MessageHandler) GetConversationMessagesSingle(c *gin.Context) {
	var input struct {
		UserId           int64 `json:"user_id"`
		TheOtherPersonId int64 `json:"the_other_person_id"`
		Platform         int   `json:"platform"`
		LastMsgId        int64 `json:"last_msg_id"`
		PageNum          int   `json:"page_num"`
		PageSize         int   `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	messages, err := h.service.GetConversationMessagesSingle(c.Request.Context(),
		input.UserId, input.TheOtherPersonId, input.LastMsgId, input.PageNum, input.PageSize)
	if err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
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

func (h *MessageHandler) GetConversationMessagesGroup(c *gin.Context) {
	var input struct {
		UserID    int64     `json:"user_id"`
		GroupID   uuid.UUID `json:"group_id"`
		Platform  int       `json:"platform"`
		LastMsgID int64     `json:"last_msg_id"`
		PageNum   int       `json:"page_num"`
		PageSize  int       `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "msg": "invalid request: " + err.Error()})
		return
	}

	messages, err := h.service.GetConversationMessagesGroup(
		c.Request.Context(),
		input.UserID, input.GroupID, input.LastMsgID,
		input.PageNum, input.PageSize,
	)
	if err != nil {
		c.JSON(500, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	// 确保返回非 nil
	if messages == nil {
		messages = &dto.ConversationMessagesDTO{}
	}
	//var messages []*dto.ConversationMessagesDTO
	// messages = []*dto.ConversationMessagesDTO{} // ✅ 空切片
	// return messages, nil

	// var message *dto.ConversationMessagesDTO
	// message = &dto.ConversationMessagesDTO{} // ✅ 单个对象指针
	// return message, nil

	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "load messages successfully",
		"data": messages,
	})
}

func (h *MessageHandler) WithdrawMessageSingle(c *gin.Context) {
	var input struct {
		UserId           int64 `json:"user_id"`
		TheOtherPersonId int64 `json:"the_other_person_id"`
		Platform         int   `json:"platform"`
		MessageId        int64 `json:"message_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	var lastMsgId int64
	var err error
	lastMsgId, err = h.service.WithdrawMessageSingle(c.Request.Context(), input.UserId, input.TheOtherPersonId, input.MessageId)
	if err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if lastMsgId == -1 {
		c.JSON(502, gin.H{"code": 1, "error": "load wrong"})
	}
	c.JSON(200, gin.H{
		"code":      0,
		"message":   "withdraw ok",
		"lastMsgId": lastMsgId,
	})
}

func (h *MessageHandler) WithdrawMessageGroup(c *gin.Context) {
	var input struct {
		UserID    int64     `json:"user_id"`
		GroupID   uuid.UUID `json:"group_id"`
		Platform  int       `json:"platform"`
		MessageID int64     `json:"message_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	var lastMsgID int64
	var err error
	lastMsgID, err = h.service.WithdrawMessageGroup(c.Request.Context(), input.UserID, input.GroupID, input.MessageID)
	if err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}

	if lastMsgID == -1 {
		c.JSON(502, gin.H{"code": 1, "error": "load wrong"})
	}

	c.JSON(200, gin.H{
		"code":      0,
		"message":   "withdraw ok",
		"lastMsgId": lastMsgID,
	})
}

func (h *MessageHandler) UnWithdrawMessageSingle(c *gin.Context) {
	var input struct {
		UserID           int64  `json:"user_id"`
		TheOtherPersonID int64  `json:"the_other_person_id"`
		Platform         int    `json:"platform"`
		MessageID        int64  `json:"message_id"`
		NewText          string `json:"new_text"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(500, gin.H{"code": 1, "error": err.Error()})
		return
	}
	var lastMsgID int64
	var err error
	lastMsgID, err = h.service.UnWithdrawMessageSingle(c.Request.Context(), input.UserID,
		input.TheOtherPersonID, input.MessageID, input.NewText)
	if err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if lastMsgID == -1 {
		c.JSON(502, gin.H{"code": 1, "error": "load wrong"})
	}
	c.JSON(200, gin.H{
		"code":      0,
		"message":   "unwithdraw ok",
		"lastMsgId": lastMsgID,
	})
}

func (h *MessageHandler) UnWithdrawMessageGroup(c *gin.Context) {
	var input struct {
		UserID    int64     `json:"user_id"`
		GroupID   uuid.UUID `json:"group_id"`
		Platform  int       `json:"platform"`
		MessageID int64     `json:"message_id"`
		NewText   string    `json:"new_text"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(500, gin.H{"code": 1, "error": err.Error()})
		return
	}
	var lastMsgID int64
	var err error
	lastMsgID, err = h.service.UnWithdrawMessageGroup(c.Request.Context(), input.UserID,
		input.GroupID, input.MessageID, input.NewText)
	if err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if lastMsgID == -1 {
		c.JSON(502, gin.H{"code": 1, "error": "load wrong"})
	}
	c.JSON(200, gin.H{
		"code":      0,
		"message":   "unwithdraw ok",
		"lastMsgId": lastMsgID,
	})
}

func (h *MessageHandler) UpdateUnread(c *gin.Context) {
	var input struct {
		UserID   int64 `json:"user_id"`
		ThreadID int64 `json:"thread_id"`
		Platform int   `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.UpdateUnread(c.Request.Context(), input.UserID, input.ThreadID); err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "update unread ok",
	})
}

func (h *MessageHandler) GetConversations(c *gin.Context) {
	var input struct {
		UserId   int64 `form:"user_id"`
		Platform int   `form:"platform"`
	}
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	conversations, err := h.service.GetConversations(c.Request.Context(), input.UserId)

	if err != nil {
		log.Printf("GetConversations error: %v", err)
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}

	if conversations == nil {
		log.Printf("GetConversations error: %v", err)
		c.JSON(502, gin.H{"code": 1, "error": "conversation is nil"})
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "get conversations ok",
		"detail":  conversations,
	})
}
