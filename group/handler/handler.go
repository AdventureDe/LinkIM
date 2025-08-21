package handler

import (
	"github.com/AdventureDe/tempName/group/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GroupHandler struct {
	service *service.GroupService
}

func NewGroupHandler(s *service.GroupService) *GroupHandler {
	return &GroupHandler{
		service: s,
	}
}

func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var input struct {
		OwnerID   int64   `json:"owner_id"`
		UserIDs   []int64 `json:"user_ids"`
		GroupName string  `json:"group_name"`
		Platform  int     `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	groupID, err := h.service.CreateGroup(c.Request.Context(), input.OwnerID, input.UserIDs, input.GroupName)
	if err != nil {
		c.JSON(502, gin.H{
			"code":  1,
			"error": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "Create group ok",
		"groupID": groupID,
	})
}

func (h *GroupHandler) AddGroupMember(c *gin.Context) {
	var input struct {
		GroupID  uuid.UUID `json:"group_id"`
		UserIDs  []int64   `json:"user_ids"`
		Platform int       `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.AddGroupMember(c.Request.Context(), input.GroupID, input.UserIDs); err != nil {
		c.JSON(502, gin.H{
			"code":  1,
			"error": err.Error(),
		})
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "invite successfullu",
	})
}

func (h *GroupHandler) KickOutGroupMember(c *gin.Context) {
	var input struct {
		GroupID    uuid.UUID `json:"group_id"`
		ExecutorID int64     `json:"executor_id"`
		UserIDs    []int64   `json:"user_ids"`
		Platform   int       `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.KickOutGroupMember(c.Request.Context(), input.GroupID, input.ExecutorID, input.UserIDs); err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "kick out successfully",
	})
}

func (h *GroupHandler) PromoteToAdmin(c *gin.Context) {
	var input struct {
		GroupID    uuid.UUID `json:"group_id"`
		ExecutorID int64     `json:"executor_id"`
		UserID     int64     `json:"user_ids"`
		Platform   int       `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.PromoteToAdmin(c.Request.Context(), input.GroupID, input.ExecutorID, input.UserID); err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "promote to admin ok",
		"userid":  input.UserID,
	})
}

func (h *GroupHandler) TransferGroupOwner(c *gin.Context) {
	var input struct {
		GroupID    uuid.UUID `json:"group_id"`
		ExecutorID int64     `json:"executor_id"`
		UserID     int64     `json:"user_ids"`
		Platform   int       `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.TransferGroupOwner(c.Request.Context(), input.GroupID, input.ExecutorID, input.UserID); err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "transfer group owner ok",
		"userid":  input.UserID,
	})
}

func (h *GroupHandler) DemotedToMember(c *gin.Context) {
	var input struct {
		GroupID    uuid.UUID `json:"group_id"`
		ExecutorID int64     `json:"executor_id"`
		UserID     int64     `json:"user_ids"`
		Platform   int       `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.DemotedToMember(c.Request.Context(), input.GroupID, input.ExecutorID, input.UserID); err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "demote to member ok",
		"userid":  input.UserID,
	})
}

func (h *GroupHandler) UpdateNotice(c *gin.Context) {
	var input struct {
		GroupID    uuid.UUID `json:"group_id"`
		ExecutorID int64     `json:"executor_id"`
		NewNotice  string    `json:"new_notice"`
		Platform   int       `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.UpdateNotice(c.Request.Context(), input.GroupID, input.ExecutorID, input.NewNotice); err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "update notice ok",
	})
}

func (h *GroupHandler) GetNotice(c *gin.Context) {
	var input struct {
		GroupID  uuid.UUID `json:"group_id"`
		Platform int       `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	notice, err := h.service.GetNotice(c.Request.Context(), input.GroupID)
	if err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "get notice ok",
		"notice":  notice,
	})
}

func (h *GroupHandler) UpdateGroupName(c *gin.Context) {
	var input struct {
		GroupID    uuid.UUID `json:"group_id"`
		ExecutorID int64     `json:"executor_id"`
		NewName    string    `json:"new_name"`
		Platform   int       `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.UpdateGroupName(c.Request.Context(), input.GroupID, input.ExecutorID, input.NewName); err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "group name update ok",
	})
}

func (h *GroupHandler) GetGroupName(c *gin.Context) {
	var input struct {
		GroupID  uuid.UUID `json:"group_id"`
		Platform int       `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	name, err := h.service.GetGroupName(c.Request.Context(), input.GroupID)
	if err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "get group name ok",
		"name":    name,
	})
}

func (h *GroupHandler) GetGroupAvatar(c *gin.Context) {
	var input struct {
		GroupID  uuid.UUID `json:"group_id"`
		Platform int       `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	set, err := h.service.GetGroupAvatar(c.Request.Context(), input.GroupID)
	if err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "get group avatar ok",
		"detail":  set,
	})
}

func (h *GroupHandler) UpdateSelfName(c *gin.Context) {
	var input struct {
		GroupID  uuid.UUID `json:"group_id"`
		UserID   int64     `json:"user_id"`
		NewName  string    `json:"new_name"`
		Platform int       `json:"platform"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.UpdateSelfName(c.Request.Context(), input.GroupID, input.UserID, input.NewName); err != nil {
		c.JSON(502, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "update self nickname ok",
		"userid":  input.UserID,
	})
}
