package dto

import (
	"time"

	"github.com/google/uuid"
)

// 业务逻辑层 用于service层获取对应想要的数据
type ConversationMessagesDTO struct {
	ThreadID int64
	Messages []*MessageDTO
	HasMore  bool
	Unread   int
}

type MessageDTO struct {
	ID            int64
	Content       string
	Sender        int64
	GroupNickname string
	UserInfo      *UserInfoDTO
	CreateTime    time.Time
}

type UserInfoDTO struct {
	UserID       int64
	SelfNickname string
	Avatar       string
}

type ConversationDTO struct {
	Type        string     `json:"type"`
	ThreadID    int64      `json:"thread_id"`
	LastMessage *Message   `json:"last_message"`
	UnreadCount int        `json:"unread_count"`
	UserInfo    *UserInfo  `json:"user_info"`
	GroupInfo   *GroupInfo `json:"group_info"`
	UpdateTime  time.Time  `json:"update_time"`
}

type Message struct {
	ID        int64     `json:"id"`
	SenderID  int64     `json:"sender_id"`
	Kind      int16     `json:"kind"` // 消息类型 1. text 2. image 3. file
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type UserInfo struct {
	UserID   int64  `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type GroupInfo struct {
	GroupID   uuid.UUID `json:"group_id"`
	GroupName string    `json:"group_name"`
	Avatar    string    `json:"avatar"`
}
