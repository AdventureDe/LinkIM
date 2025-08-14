package dto

import "time"

type CaptchaStore struct {
	Code string `json:"code"`
}

type UserSession struct {
	UserID    int64     `json:"userID"`
	Token     string    `json:"token"`
	LoginTime time.Time `json:"loginTime"`
}

type LogoutRequest struct {
	UserID int64  `json:"userID" binding:"required"`
	Token  string `json:"token" binding:"required"`
}
