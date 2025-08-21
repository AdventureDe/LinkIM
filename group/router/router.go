package router

import (
	"github.com/AdventureDe/tempName/group/handler"

	"github.com/gin-gonic/gin"
)

func SetGroupRouter(r *gin.Engine, g *handler.GroupHandler) {
	r.POST("/group/create", g.CreateGroup)
	r.POST("/group/invite", g.AddGroupMember)
	r.DELETE("/group/kickout", g.KickOutGroupMember)
	r.PUT("/group/promote/admin", g.PromoteToAdmin)
	r.PUT("/group/demote/member", g.DemotedToMember)
	r.PUT("/group/promote/owner", g.TransferGroupOwner)
	r.PUT("/group/notice", g.UpdateNotice)
	r.GET("/group/notice", g.GetNotice)
	r.PUT("/group/name", g.UpdateGroupName)
	r.GET("/group/name", g.GetGroupName)
	r.GET("/group/avatar", g.GetGroupAvatar)
	r.PUT("/group/nickname", g.UpdateSelfName)
}
