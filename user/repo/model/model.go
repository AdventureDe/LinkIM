package model

import (
	"time"
)

type Status string

const (
	Pending  Status = "pending"
	Accepted Status = "accepted"
	Rejected Status = "rejected"
)

// 测试账号：  12321412411
// 测试密码:   qwer12345678
type User struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Nickname     string    `gorm:"default:'momo'" json:"nickname"`
	PasswordHash string    `gorm:"not null" json:"password_hash"`
	Email        string    `gorm:"uniqueIndex:users_email_key" json:"email"`
	Area         string    `gorm:"default:'+86'" json:"area"`
	Phone        string    `gorm:"uniqueIndex:users_phone_key" json:"phone"`
	AvatarUrl    string    `gorm:"default:''" json:"avatar:_url"`
	Signature    string    `gorm:"default:''" json:"signature"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastLoginAt  time.Time `json:"last_login_at "`
}

type Friendship struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         int64     `gorm:"not null;index" json:"user_id"`
	FriendID       int64     `gorm:"not null;index" json:"friend_id"`
	Status         Status    `gorm:"type:enum('pending','accepted','rejected');default:'pending'" json:"status"`
	RequestMessage string    `gorm:"type:text" json:"request_message"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// relationShip
type FriendGroup struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"not null;index" json:"user_id"`
	Name      string    `gorm:"size:50;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FriendGroupMember struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID   int64     `gorm:"not null;index" json:"group_id"`
	FriendID  int64     `gorm:"not null;index" json:"friend_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Blacklist struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        int64     `gorm:"not null;index:user_block_unique" json:"user_id"`         // 拉黑人
	BlockedUserID int64     `gorm:"not null;index:user_block_unique" json:"blocked_user_id"` // 被拉黑的人
	CreatedAt     time.Time `gorm:"not null;default:now()" json:"created_at"`
}
