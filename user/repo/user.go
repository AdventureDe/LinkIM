package repo

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/AdventureDe/LinkIM/user/repo/model"

	"gorm.io/gorm"
)

// User 代表一个用户实体 用于数据访问操作 用于简单的函数返回 不要用于数据库操作
type User struct {
	ID        int64
	Nickname  string
	Email     string
	Area      string
	Phone     string
	AvatarUrl string
	Signature string
}

// UserRepo 接口定义
type UserRepo interface {
	// User
	CreateUser(ctx context.Context, user *model.User) error
	GetPasswordHash_type1(ctx context.Context, phone string) (string, error)
	GetPasswordHash_type2(ctx context.Context, email string) (string, error)
	GetUserByUserPhone(ctx context.Context, phone string) (*User, error)
	GetUserByUserEmail(ctx context.Context, email string) (*User, error)
	GetUserIdByUserPhone(ctx context.Context, phone string) (int64, error)
	GetUserIdByUserEmail(ctx context.Context, email string) (int64, error)
	UpdatePassWord(ctx context.Context, userid int64, newPassWord string) error
	UpdateLoginTime(ctx context.Context, userid int64) error
	UpdatePhone(ctx context.Context, userid int64, phone, areaCode string) error
	UpdateEmail(ctx context.Context, userid int64, email string) error
	UpdateProfile(ctx context.Context, userid int64, url string) error
	UpdateNickName(ctx context.Context, userid int64, nickname string) error
	UpdateSignature(ctx context.Context, userid int64, newSignature string) error
	GetUserInfo(ctx context.Context, userid int64) (*User, error)
	GetUserInfos(ctx context.Context, userid []int64) ([]User, error)
	// Friend
	CreateFriend(ctx context.Context, friendShip *model.Friendship) error
	UpdateFriendStatus(ctx context.Context, userid int64, friendid int64, status model.Status) error
	GetFriendLists(ctx context.Context, userid int64) ([]int64, error)
	GetUsersByIDs(ctx context.Context, userIDs []int64) ([]*User, error)
	GetUsersByIDsExceptBlacklist(ctx context.Context, userID int64, userIDs []int64) ([]*User, error)
	DelFriend(ctx context.Context, userid int64, friendid int64) error
	CreateRelationShip(ctx context.Context, relationShip *model.FriendGroup) error
	DelRelationShipByName(ctx context.Context, userId int64, relationShipName string) error
	DelRelationShipById(ctx context.Context, relationShipId int64) error
	GetRelationShipNameId(ctx context.Context, userid int64, relationShipName string) (int64, error)
	GetAllRelationShips(ctx context.Context, userid int64) ([]string, error)
	AddFriendToRelationShip(ctx context.Context, friendGroupMember *model.FriendGroupMember) error
	DelFriendFromRelationShip(ctx context.Context, groupid int64, friendid int64) error
	DelFriendsFromRelationShip(ctx context.Context, groupid int64, friendid []int64) error
	GetFriendListFromRelationShip(ctx context.Context, groupid int64) ([]int64, error)
	BlockFriend(ctx context.Context, blockedfriend *model.Blacklist) error
	UnblockFriend(ctx context.Context, userid int64, friendid int64) error
	GetBlockedFriends(ctx context.Context, userid int64) ([]int64, error)
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepo {
	return &userRepo{db: db}
}

func (r *userRepo) CreateUser(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepo) GetUserInfo(ctx context.Context, userid int64) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("id = ?", userid).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetUserInfos(ctx context.Context, userid []int64) ([]User, error) {
	if len(userid) == 0 {
		return []User{}, nil
	}

	var user []User
	if err := r.db.WithContext(ctx).Where("id IN ?", userid).Find(&user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (r *userRepo) GetUserByUserPhone(ctx context.Context, phone string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetUserByUserEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetPasswordHash_type1(ctx context.Context, phone string) (string, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error; err != nil {
		return "", err
	}
	return user.PasswordHash, nil
}

func (r *userRepo) GetPasswordHash_type2(ctx context.Context, email string) (string, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return "", err
	}
	return user.PasswordHash, nil
}

func (r *userRepo) GetUserIdByUserPhone(ctx context.Context, phone string) (int64, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error; err != nil {
		return 0, err
	}
	return user.ID, nil
}

func (r *userRepo) GetUserIdByUserEmail(ctx context.Context, email string) (int64, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return 0, err
	}
	return user.ID, nil
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

func (s *userRepo) UpdatePassWord(ctx context.Context, userid int64, newPassWord string) error {
	hash := md5String(newPassWord)
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userid).Update("password_hash", hash).Error; err != nil {
		return err
	}
	return nil
}

func (s *userRepo) UpdateLoginTime(ctx context.Context, userid int64) error {
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userid).Update("last_login_at", time.Now()).Error; err != nil {
		return err
	}
	return nil
}

func (s *userRepo) UpdatePhone(ctx context.Context, userid int64, phone, areaCode string) error {
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userid).Update("phone", phone).Error; err != nil {
		return err
	}
	return nil
}

func (s *userRepo) UpdateEmail(ctx context.Context, userid int64, email string) error {
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userid).Update("email", email).Error; err != nil {
		return err
	}
	return nil
}

func (s *userRepo) UpdateProfile(ctx context.Context, userid int64, url string) error {
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userid).Update("avatar_url", url).Error; err != nil {
		return err
	}
	return nil
}

func (s *userRepo) UpdateNickName(ctx context.Context, userid int64, nickname string) error {
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userid).Update("nickname", nickname).Error; err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (s *userRepo) UpdateSignature(ctx context.Context, userid int64, newSignature string) error {
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userid).Update("signature", newSignature).Error; err != nil {
		return err
	}
	return nil
}

func (s *userRepo) CreateFriend(ctx context.Context, friendShip *model.Friendship) error {
	return s.db.Debug().WithContext(ctx).Model(&model.Friendship{}).Create(friendShip).Error
}

func (s *userRepo) UpdateFriendStatus(ctx context.Context, userid int64, friendid int64, status model.Status) error {
	if err := s.db.Debug().WithContext(ctx).Model(&model.Friendship{}).Where("user_id = ? AND friend_id = ?", userid, friendid).Update("status", status).Error; err != nil {
		return err
	}
	return nil
}

// 获取好友列表
func (s *userRepo) GetFriendLists(ctx context.Context, userid int64) ([]int64, error) {
	var id1 []int64
	var id2 []int64
	if err := s.db.Debug().WithContext(ctx).Model(&model.Friendship{}).Select("user_id").Where("friend_id = ? AND status = ?", userid, model.Status("accepted")).Find(&id1).Error; err != nil {
		return nil, err
	}
	if err := s.db.Debug().WithContext(ctx).Model(&model.Friendship{}).Select("friend_id").Where("user_id = ? AND status = ?", userid, model.Status("accepted")).Find(&id2).Error; err != nil {
		return nil, err
	}
	ids := append(id1, id2...)
	return ids, nil
}

// 批量查询
func (s *userRepo) GetUsersByIDs(ctx context.Context, userIDs []int64) ([]*User, error) {
	var users []*User
	if err := s.db.WithContext(ctx).
		Where("id IN ?", userIDs).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// 排除了黑名单后的好友列表
func (s *userRepo) GetUsersByIDsExceptBlacklist(ctx context.Context, userID int64, userIDs []int64) ([]*User, error) {
	var users []*User
	if err := s.db.WithContext(ctx).
		Where("id IN ?", userIDs).
		Where("id NOT IN (?)", s.db.Model(&model.Blacklist{}).Select("blocked_user_id").Where("user_id = ?", userID)).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// 删除好友
func (s *userRepo) DelFriend(ctx context.Context, userid int64, friendid int64) error {
	if err := s.db.Debug().WithContext(ctx).
		Where("user_id = ? AND friend_id=?", userid, friendid).
		Delete(&model.Friendship{}).Error; err != nil {
		return err
	}
	return nil
}

// 创建关系
func (s *userRepo) CreateRelationShip(ctx context.Context, relationShip *model.FriendGroup) error {
	return s.db.Debug().WithContext(ctx).Model(&model.FriendGroup{}).Create(relationShip).Error
}

// 删除关系
func (s *userRepo) DelRelationShipByName(ctx context.Context, userId int64, relationShipName string) error {
	return s.db.WithContext(ctx).
		Where("user_id = ? AND name = ?", userId, relationShipName).
		Delete(&model.FriendGroup{}).Error
}

// 删除关系
func (s *userRepo) DelRelationShipById(ctx context.Context, relationShipId int64) error {
	return s.db.Debug().WithContext(ctx).Where("id = ?", relationShipId).Delete(&model.FriendGroup{}).Error
}

// 获取关系id
func (s *userRepo) GetRelationShipNameId(ctx context.Context, userid int64, relationShipName string) (int64, error) {
	var id int64
	if err := s.db.WithContext(ctx).
		Model(&model.FriendGroup{}).
		Select("id").
		Where("user_id = ? AND name = ?", userid, relationShipName).
		Pluck("id", &id).Error; err != nil {
		return -1, err
	}
	// First 获取的是Struct 即  select *
	// 而Pluck(&id)可以通过select id来获取id
	return id, nil
}

func (s *userRepo) GetAllRelationShips(ctx context.Context, userid int64) ([]string, error) {
	var names []string
	err := s.db.WithContext(ctx).
		Model(&model.FriendGroup{}).
		Where("user_id = ?", userid).
		Pluck("name", &names).Error
	if err != nil {
		return nil, err
	}
	return names, nil

}

func (s *userRepo) AddFriendToRelationShip(ctx context.Context, friendGroupMember *model.FriendGroupMember) error {
	return s.db.Debug().WithContext(ctx).Model(&model.FriendGroupMember{}).
		Create(friendGroupMember).Error
}

// 删除一个好友从关系中
func (s *userRepo) DelFriendFromRelationShip(ctx context.Context, groupid int64, friendid int64) error {
	return s.db.Debug().WithContext(ctx).Where("group_id = ? AND friend_id = ?", groupid, friendid).
		Delete(&model.FriendGroupMember{}).Error
}

// 删除多个好友从关系中
func (s *userRepo) DelFriendsFromRelationShip(ctx context.Context, groupid int64, friendid []int64) error {
	return s.db.Debug().WithContext(ctx).Where("group_id = ? AND friend_id IN ?", groupid, friendid).
		Delete(&model.FriendGroupMember{}).Error
}

func (s *userRepo) GetFriendListFromRelationShip(ctx context.Context, groupid int64) ([]int64, error) {
	var friendIds []int64
	if err := s.db.Debug().WithContext(ctx).
		Model(&model.FriendGroupMember{}).
		Where("group_id = ?", groupid).
		Pluck("friend_id", &friendIds).
		Error; err != nil {
		return nil, err
	}
	return friendIds, nil
}

// 拉黑一个好友
func (s *userRepo) BlockFriend(ctx context.Context, blockedfriend *model.Blacklist) error {
	return s.db.Debug().WithContext(ctx).
		Model(&model.Blacklist{}).
		Create(blockedfriend).Error
}

// 取消拉黑一个好友
func (s *userRepo) UnblockFriend(ctx context.Context, userid int64, friendid int64) error {
	return s.db.Debug().WithContext(ctx).
		Model(&model.Blacklist{}).
		Where("user_id = ? AND blocked_user_id = ?", userid, friendid).
		Delete(&model.Blacklist{}).Error
}

// 获取拉黑列表
func (s *userRepo) GetBlockedFriends(ctx context.Context, userid int64) ([]int64, error) {
	var friendIds []int64
	if err := s.db.Debug().WithContext(ctx).
		Model(&model.Blacklist{}).
		Where("user_id = ?", userid).
		Pluck("blocked_user_id", &friendIds).Error; err != nil {
		return nil, err
	}
	return friendIds, nil
}
