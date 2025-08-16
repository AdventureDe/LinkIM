package handler

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"user/repo"
	"user/service"

	"github.com/gin-gonic/gin"
)

// UserHandler 处理用户相关的 HTTP 请求
// 这里的 UserHandler 结构体包含了 UserService 的实例
type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{service: s}
}

// VerificationHandler 处理验证码相关的 HTTP 请求
// 这里的 VerificationHandler 结构体包含了 VerificationService 的实例
type VerificationHandler struct {
	service *service.VerificationService
}

func NewVerificationHandler(s *service.VerificationService) *VerificationHandler {
	return &VerificationHandler{service: s}
}

/* ----------------------------------------------------- */
// 验证码部分
func (h *VerificationHandler) SendCode(c *gin.Context) {
	var input struct {
		Phone          string `json:"phoneNumber" binding:"required"`
		Area           string `json:"areaCode" binding:"required"`
		UsedFor        int    `json:"usedFor" binding:"required"`
		InvitationCode string `json:"invitationCode"`
	}
	//{"phoneNumber":"12345678901","areaCode":"+86","usedFor":1,"invitationCode":""}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		fmt.Println("Error binding JSON:", err)
		return
	}

	if err := h.service.SendCode(c.Request.Context(), input.Area, input.Phone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send code"})
		fmt.Println("Error sending code:", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "code sent"})
}

func (h *VerificationHandler) VerifyCode(c *gin.Context) {
	var input struct {
		Area    string `json:"areaCode" binding:"required"`
		Phone   string `json:"phoneNumber" binding:"required"`
		UsedFor int    `json:"usedFor" binding:"required"`
		Code    string `json:"verifyCode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := h.service.VerifyCode(c.Request.Context(), input.Area, input.Phone, input.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "code verified"})
}

/* ----------------------------------------------------- */
// 个人信息管理部分
func (h *UserHandler) Register(c *gin.Context) {
	var input struct {
		VerifyCode string `json:"verifyCode" binding:"required"`
		Platform   int    `json:"platform" binding:"required"`
		AutoLogin  bool   `json:"autoLogin"`
		User       struct {
			PhoneNumber     string `json:"phoneNumber" binding:"required"`
			AreaCode        string `json:"areaCode" binding:"required"`
			Nickname        string `json:"nickname"`
			Password        string `json:"password" binding:"required"`
			ConfirmPassword string `json:"confirmPassword" binding:"required"`
			Email           string `json:"email"`
			InvitationCode  string `json:"invitationCode"`
		} `json:"user" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.Register(c.Request.Context(), input.User.Nickname, input.User.Password,
		input.User.AreaCode, input.User.PhoneNumber, input.User.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "register success"})
}

func md5String(str string) string {
	// 创建 MD5 哈希对象
	hash := md5.New()
	// 写入要加密的数据
	hash.Write([]byte(str))
	// 计算哈希值，返回字节切片
	bytes := hash.Sum(nil)
	// 将字节切片转换为十六进制字符串
	return hex.EncodeToString(bytes)
}

func (h *UserHandler) Login(c *gin.Context) {
	var input struct {
		PhoneNumber string `json:"phoneNumber" binding:"required"`
		Email       string `json:"email"`
		AreaCode    string `json:"areaCode" binding:"required"`
		Password    string `json:"password" binding:"required"`
		Platform    int    `json:"platform" binding:"required"`
		VerifyCode  string `json:"verifyCode"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash := md5String(input.Password)
	userid, imToken, err := h.service.LoginByPhone(c.Request.Context(), input.PhoneNumber, hash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "login success",
		"data": gin.H{
			"userID":     userid,
			"imToken":    imToken,
			"platformID": input.Platform,
		},
	})

}

func (h *UserHandler) UpdatePassWord(c *gin.Context) {
	var input struct {
		UserId      int64  `json:"userId" binding:"required"`
		PhoneNumber string `json:"phoneNumber" binding:"required"`
		AreaCode    string `json:"areaCode" binding:"required"`
		VerifyCode  string `json:"verifyCode" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
		Email       string `json:"email"`
		Platform    int    `json:"platform" binding:"required"`
	}
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdatePassWord(c.Request.Context(), input.UserId, input.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "update success"})
}

func (h *UserHandler) Logout(c *gin.Context) {
	var input struct {
		UserID int64  `json:"userID" binding:"required"`
		Token  string `json:"token" binding:"required"`
	}
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Logout(c.Request.Context(), input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "logout success"})
}

// TODO: 已经将Session存入了redis，可以使用session中的token，进行 Authorization: Bearer <token>
func (h *UserHandler) UpdatePhone(c *gin.Context) {
	var input struct {
		UserID      int64  `json:"userID" binding:"required"`
		PhoneNumber string `json:"phoneNumber" binding:"required"`
		AreaCode    string `json:"areaCode" binding:"required"`
		Platform    int    `json:"platform" binding:"required"`
		VerifyCode  string `json:"verifyCode" binding:"required"`
		// add more
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdatePhone(c.Request.Context(), input.UserID, input.PhoneNumber, input.AreaCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "phone updated"})
}

func (h *UserHandler) UpdateEmail(c *gin.Context) {
	var input struct {
		UserID     int64  `json:"userID" binding:"required"`
		Platform   int    `json:"platform" binding:"required"`
		VerifyCode string `json:"verifyCode" binding:"required"`
		Email      string `json:"email" binding:"required"`
		// add more
	}
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdateEmail(c.Request.Context(), input.UserID, input.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "email updated"})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var input struct {
		UserID     int64  `json:"userID" binding:"required"`
		Platform   int    `json:"platform" binding:"required"`
		ProfileUrl string `json:"profileUrl" binding:"required"`
	}
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdateProfile(c.Request.Context(), input.UserID, input.ProfileUrl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "profile updated"})
}

func (h *UserHandler) UpdateNickName(c *gin.Context) {
	var input struct {
		UserID   int64  `json:"userID" binding:"required"`
		Platform int    `json:"platform" binding:"required"`
		Nickname string `json:"nickname" binding:"required"`
	}
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdateNickName(c.Request.Context(), input.UserID, input.Nickname); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "nickname updated"})
}

// 前端接口应使用  query查询  ?userID=5&platform=2
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	var input struct {
		UserID   int64 `form:"userID" binding:"required"`
		Platform int   `form:"platform" binding:"required"`
	}
	//------- test
	// body, _ := io.ReadAll(c.Request.Body)
	// fmt.Println("Raw body:", string(body))
	// c.Request.Body = io.NopCloser(bytes.NewBuffer(body)) // 重新放回去

	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	user, err := h.service.GetUserInfo(c.Request.Context(), input.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"message":    "load ok",
		"nickname":   user.Nickname,
		"avatar_url": user.AvatarUrl,
		"phone":      user.Phone,
		"email":      user.Email,
		"signature":  user.Signature,
	})
}

func (h *UserHandler) UpdateSignature(c *gin.Context) {
	var input struct {
		UserID    int64  `json:"userID" binding:"required"`
		Platform  int    `json:"platform" binding:"required"`
		Signature string `json:"signature" binding:"required"`
	}
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdateSignature(c.Request.Context(), input.UserID, input.Signature); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "signature updated"})
}

/* --------------------------------------------------------------- */
// 好友管理部分
// 添加好友
func (h *UserHandler) CreateFriendShip(c *gin.Context) {
	var input struct {
		UserID         int64  `json:"user_id" binding:"required"`
		Platform       int    `json:"platform" binding:"required,min=1"`
		FriendID       int64  `json:"friend_id" binding:"required"`
		RequestMessage string `json:"request_message" binding:"omitempty,max=500"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.CreateFriendShip(c.Request.Context(), input.UserID, input.FriendID, input.RequestMessage); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "message has present",
	})
}

// 接受好友请求
func (h *UserHandler) AcceptFriend(c *gin.Context) {
	var input struct {
		UserID   int64  `json:"user_id" binding:"required"`
		Platform int    `json:"platform" binding:"required,min=1"`
		FriendID int64  `json:"friend_id" binding:"required"`
		Statusp  string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.AcceptFriend(c.Request.Context(), input.UserID, input.FriendID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "accept the friend",
	})
}

// 拒绝好友请求
func (h *UserHandler) RejectFriend(c *gin.Context) {
	var input struct {
		UserID   int64  `json:"user_id" binding:"required"`
		Platform int    `json:"platform" binding:"required,min=1"`
		FriendID int64  `json:"friend_id" binding:"required"`
		Statusp  string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.RejectFriend(c.Request.Context(), input.UserID, input.FriendID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "reject the friend",
	})
}

// 获取好友列表
func (h *UserHandler) GetFriendLists(c *gin.Context) {
	var input struct {
		UserID   int64 `form:"user_id" binding:"required"`
		Platform int   `form:"platform" binding:"required,min=1"`
	}
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	lists, err := h.service.GetFriendLists(c.Request.Context(), input.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "get lists",
		"detail":  lists,
	})
}

// 删除好友
func (h *UserHandler) DelFriend(c *gin.Context) {
	var input struct {
		UserID   int64 `json:"user_id" binding:"required"`
		Platform int   `json:"platform" binding:"required,min=1"`
		FriendID int64 `json:"friend_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.DelFriend(c.Request.Context(), input.UserID, input.FriendID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "delete ok",
	})
}

// 创建关系 针对好友
func (h *UserHandler) CreateRelationShip(c *gin.Context) {
	var input struct {
		UserID           int64  `json:"user_id" binding:"required"`
		Platform         int    `json:"platform" binding:"required,min=1"`
		RelationShipName string `json:"relation_ship_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.CreateRelationShip(c.Request.Context(), input.UserID, input.RelationShipName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "create relationShip ok",
	})
}

// 删除关系 针对好友
func (h *UserHandler) DelRelationShip(c *gin.Context) {
	var input struct {
		UserID           int64  `json:"user_id" binding:"required"`
		Platform         int    `json:"platform" binding:"required,min=1"`
		RelationShipName string `json:"relation_ship_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.DelRelationShip(c.Request.Context(), input.UserID, input.RelationShipName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Delete relationShip ok",
	})
}

// 获取所有以及创建了的关系
func (h *UserHandler) GetAllRelationShips(c *gin.Context) {
	var input struct {
		UserID   int64 `form:"user_id" binding:"required"`
		Platform int   `form:"platform" binding:"required,min=1"`
	}
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	var relationShips []string
	var err error
	if relationShips, err = h.service.GetAllRelationShips(c.Request.Context(), input.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "load relationships ok",
		"detail":  relationShips,
	})
}

// 添加好友到已经创建了的关系
func (h *UserHandler) AddFriendtoRelationShip(c *gin.Context) {
	var input struct {
		UserID           int64  `json:"user_id" binding:"required"`
		Platform         int    `json:"platform" binding:"required,min=1"`
		RelationShipName string `json:"relation_ship_name" binding:"required"`
		FriendID         int64  `json:"friend_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.AddFriendToRelationShip(c.Request.Context(), input.UserID, input.RelationShipName, input.FriendID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "add friend to relationShip ok",
	})
}

// 删除好友从已经创建了的关系
func (h *UserHandler) DelFriendFromRelationShip(c *gin.Context) {
	var input struct {
		UserID           int64  `json:"user_id" binding:"required"`
		Platform         int    `json:"platform" binding:"required,min=1"`
		RelationShipName string `json:"relation_ship_name" binding:"required"`
		FriendID         int64  `json:"friend_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.DelFriendFromRelationShip(c.Request.Context(), input.UserID, input.RelationShipName, input.FriendID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "delete friend from relationship ok",
	})
}

// 获取好友信息从已经创建了的关系
func (h *UserHandler) GetFriendsInfoFromRelationShip(c *gin.Context) {
	var input struct {
		UserID           int64  `json:"user_id" binding:"required"`
		Platform         int    `json:"platform" binding:"required,min=1"`
		RelationShipName string `json:"relation_ship_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	friendsInfo, err := h.service.GetFriendsInfoFromRelationShip(c.Request.Context(), input.UserID, input.RelationShipName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "get all friend's info",
		"detail":  friendsInfo,
	})
}

// 拉黑
func (h *UserHandler) BlockaFriend(c *gin.Context) {
	var input struct {
		UserID   int64 `json:"user_id" binding:"required"`
		Platform int   `json:"platform" binding:"required,min=1"`
		FriendID int64 `json:"friend_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.BlockFriend(c.Request.Context(), input.UserID, input.FriendID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success blocked",
	})
}

// 取消拉黑
func (h *UserHandler) UnblockaFriend(c *gin.Context) {
	var input struct {
		UserID   int64 `json:"user_id" binding:"required"`
		Platform int   `json:"platform" binding:"required,min=1"`
		FriendID int64 `json:"friend_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	if err := h.service.UnblockFriend(c.Request.Context(), input.UserID, input.FriendID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success unblocked",
	})
}

// 获取
func (h *UserHandler) GetBlockedFriends(c *gin.Context) {
	var input struct {
		UserID   int64 `form:"user_id" binding:"required"`
		Platform int   `form:"platform" binding:"required,min=1"`
	}
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "error": err.Error()})
		return
	}
	var info []*repo.User
	info, err := h.service.GetBlockedFriends(c.Request.Context(), input.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success unblocked",
		"detail":  info,
	})
}
