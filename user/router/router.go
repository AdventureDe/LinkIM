package router

import (
	"user/handler"

	"github.com/gin-gonic/gin"
)

func SetupRouter(r *gin.Engine, userHandler *handler.UserHandler) {
	r.POST("/account/register", userHandler.Register)
	r.POST("/account/login", userHandler.Login)
	r.PUT("/account/password", userHandler.UpdatePassWord)
	r.POST("/account/logout", userHandler.Logout)
	r.PUT("/account/profile", userHandler.UpdateProfile)
	r.PUT("/account/nickname", userHandler.UpdateNickName)
	r.PUT("/account/phone", userHandler.UpdatePhone)
	r.PUT("/account/email", userHandler.UpdateEmail)
	r.PUT("/account/signature", userHandler.UpdateSignature)
	r.GET("/account/getinfo", userHandler.GetUserInfo)
}

func SetupVerificationRouter(r *gin.Engine, verificationHandler *handler.VerificationHandler) {
	r.POST("/account/code/send", verificationHandler.SendCode)
	r.POST("/account/code/verify", verificationHandler.VerifyCode)
}
