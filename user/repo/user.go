package repo

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
	"user/repo/model"

	"gorm.io/gorm"
)

// User 代表一个用户实体 用于数据访问操作 用于简单的函数返回 不要用于数据库操作
type User struct {
	ID           int64
	Nickname     string
	PasswordHash string
	Email        string
	Area         string
	Phone        string
	AvatarUrl    string
	Signature    string
}

// UserRepo 接口定义
type UserRepo interface {
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
