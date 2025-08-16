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

func SetupFriendRouter(r *gin.Engine, userHandler *handler.UserHandler) {
	r.POST("/account/addfriend", userHandler.CreateFriendShip)
	r.PUT("/account/acceptfriend", userHandler.AcceptFriend)
	r.PUT("/account/rejectfriend", userHandler.RejectFriend)
	r.GET("/account/friendlists", userHandler.GetFriendLists)
	r.DELETE("/account/delfriend", userHandler.DelFriend)
	r.POST("/account/addrelationship", userHandler.CreateRelationShip)
	r.DELETE("/account/delrelationship", userHandler.DelRelationShip)
	r.GET("/account/relationships", userHandler.GetAllRelationShips)
	r.POST("/account/relationship/addfriend", userHandler.AddFriendtoRelationShip)
	r.DELETE("/account/relationship/delfriend", userHandler.DelFriendFromRelationShip)
	r.GET("/account/relationship/friendlists", userHandler.GetFriendsInfoFromRelationShip)
	r.POST("/account/blacklist/add", userHandler.BlockaFriend)
	r.DELETE("/account/blacklist/delete", userHandler.UnblockaFriend)
	r.GET("account/blacklist/friendlists", userHandler.GetBlockedFriends)
}

func SetupVerificationRouter(r *gin.Engine, verificationHandler *handler.VerificationHandler) {
	r.POST("/account/code/send", verificationHandler.SendCode)
	r.POST("/account/code/verify", verificationHandler.VerifyCode)
}
