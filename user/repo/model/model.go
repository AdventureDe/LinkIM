package model

import (
	"time"
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
