package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"
	"user/dto"
	"user/repo"
	"user/repo/model"

	"github.com/golang-jwt/jwt/v5"
)

type UserService struct {
	repo  repo.UserRepo
	redis repo.UserRedis
}

/*
1. userService 结构体组成
userService 一般应该包含：

repo 接口（依赖倒置，不直接依赖 userRepo 结构体）
（可选）其他外部依赖，比如短信服务、邮件服务、缓存、日志组件等
正确写法：

	type UserService struct {
	    repo repo.UserRepo // 依赖接口，不依赖具体实现
	}

这样更符合 SOLID 原则 中的依赖倒置原则。
*/
func NewUserService(r repo.UserRepo, u repo.UserRedis) *UserService {
	return &UserService{
		repo:  r,
		redis: u,
	}
}

type VerificationService struct { //依赖注入
	rdb repo.UserRedis
}

func NewVerificationService(rdb repo.UserRedis) *VerificationService {
	return &VerificationService{rdb: rdb}
}

func generateNumericCode() string {
	// 使用本地随机数生成器，避免使用已弃用的 rand.Seed
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 生成100000到999999之间的随机数
	code := r.Intn(900000) + 100000
	return fmt.Sprintf("%d", code)
}

func (s *VerificationService) SendCode(ctx context.Context, areaCode string, phone string) error {
	// 这里可以集成短信服务，发送验证码
	// 目前仅模拟发送成功
	code := generateNumericCode()
	fmt.Printf("Sending code %s to phone %s\n", code, phone)
	CaptchaStore := &dto.CaptchaStore{
		Code: code,
	}
	//存储验证码到redis
	err := s.rdb.SetCaptcha(ctx, areaCode, phone, CaptchaStore)
	if err != nil {
		return fmt.Errorf("failed to set captcha: %w", err)
	}

	return nil
}

func (s *VerificationService) VerifyCode(ctx context.Context, area, phone, code string) (bool, error) {
	// 从 Redis 获取验证码
	captchaStore, err := s.rdb.GetCaptcha(ctx, area, phone)
	if err != nil {
		return false, fmt.Errorf("failed to get captcha: %w", err)
	}

	// 验证码不存在或已过期
	if captchaStore == nil || captchaStore.Code != code {
		return false, errors.New("invalid or expired captcha")
	}

	// 验证成功，删除验证码
	err = s.rdb.DeleteCaptcha(ctx, area, phone)
	if err != nil {
		return false, fmt.Errorf("failed to delete captcha: %w", err)
	}

	return true, nil
}

func (s *UserService) Register(ctx context.Context, nickname, password, area, phone, email string) error {
	// 检查手机号是否已存在
	existing, _ := s.repo.GetUserByUserPhone(ctx, phone)
	if existing != nil {
		return errors.New("该手机号已被注册")
	}
	// 创建用户
	user := &model.User{
		Nickname:     nickname,
		PasswordHash: password,
		Area:         area,
		Phone:        phone,
		Email:        email,
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func GenerateToken(userID string, secret string) (string, error) {
	claims := jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(24 * time.Hour).Unix(), // 24 小时过期
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *UserService) LoginByPhone(ctx context.Context, phone, password_hash string) (int64, string, error) {
	password, err := s.repo.GetPasswordHash_type1(ctx, phone)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get password hash: %w", err)
	}
	if password != password_hash {
		return 0, "", errors.New("密码错误")
	}

	userid, err := s.repo.GetUserIdByUserPhone(ctx, phone)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get user by phone: %w", err)
	}
	token, err := GenerateToken(strconv.FormatInt(userid, 10), password_hash)
	if err != nil {
		return 0, "", fmt.Errorf("%w", err)
	}
	userSession := &dto.UserSession{
		UserID:    userid,
		Token:     token,
		LoginTime: time.Now(),
	}
	s.repo.UpdateLoginTime(ctx, userid)
	s.redis.SetSession(ctx, userSession)
	return userid, token, nil
}

func (s *UserService) Logout(ctx context.Context, req dto.LogoutRequest) error {
	// 获取当前用户的会话信息
	storedSession, err := s.redis.GetSession(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to retrieve session from Redis: %w", err)
	}

	// 校验 token 是否匹配
	if storedSession.Token != req.Token {
		return errors.New("invalid token")
	}

	// 删除 Redis 中的会话
	err = s.redis.DelSession(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to delete session from Redis: %w", err)
	}

	return nil
}

// 用于修改密码 或者 找回密码
func (s *UserService) UpdatePassWord(ctx context.Context, userid int64, newPassword string) error {
	err := s.repo.UpdatePassWord(ctx, userid, newPassword)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

// 用于更改手机号
func (s *UserService) UpdatePhone(ctx context.Context, userid int64, phone, areaCode string) error {
	err := s.repo.UpdatePhone(ctx, userid, phone, areaCode)
	if err != nil {
		return fmt.Errorf("fail to update phone:%w", err)
	}
	return nil
}

// 用于更改邮箱
func (s *UserService) UpdateEmail(ctx context.Context, userid int64, email string) error {
	err := s.repo.UpdateEmail(ctx, userid, email)
	if err != nil {
		return fmt.Errorf("fail to update email:%w", err)
	}
	return nil
}

// 用于更新头像
func (s *UserService) UpdateProfile(ctx context.Context, userid int64, url string) error {
	err := s.repo.UpdateProfile(ctx, userid, url)
	if err != nil {
		return fmt.Errorf("fail to update profile:%w", err)
	}
	return nil
}

// 用于更新昵称
func (s *UserService) UpdateNickName(ctx context.Context, userid int64, nickname string) error {
	err := s.repo.UpdateNickName(ctx, userid, nickname)
	if err != nil {
		return fmt.Errorf("fail to update nickname:%w", err)
	}
	return nil
}

// 获取用户信息
func (s *UserService) GetUserInfo(ctx context.Context, userid int64) (*repo.User, error) {
	user, err := s.repo.GetUserInfo(ctx, userid)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) UpdateSignature(ctx context.Context, userid int64, newsignature string) error {
	err := s.repo.UpdateSignature(ctx, userid, newsignature)
	if err != nil {
		return fmt.Errorf("fail to update signature: %w", err)
	}
	return nil
}
