package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"
	"github.com/AdventureDe/LinkIM/user/dto"

	"github.com/go-redis/redis/v8"
)

type UserRedis interface {
	SetCaptcha(ctx context.Context, areaCode, phone string, store *dto.CaptchaStore) error
	GetCaptcha(ctx context.Context, areaCode, phone string) (*dto.CaptchaStore, error)
	DeleteCaptcha(ctx context.Context, areaCode, phone string) error
	SetSession(ctx context.Context, session *dto.UserSession) error
	GetSession(ctx context.Context, userId int64) (*dto.UserSession, error)
	DelSession(ctx context.Context, userId int64) error
}

type userRedis struct {
	rdb *redis.Client
}

func NewUserRedis(rdb *redis.Client) UserRedis {
	return &userRedis{rdb: rdb}
}

// 验证码服务
func (r *userRedis) SetCaptcha(ctx context.Context, areaCode, phone string, store *dto.CaptchaStore) error {
	data, err := json.Marshal(store)
	if err != nil {
		return err
	}
	Default_Time := 5 * time.Minute
	// 使用 areaCode 和 phone 作为键，存储验证码
	p := areaCode + phone
	return r.rdb.Set(ctx, "Captcha:"+p, data, Default_Time).Err()
}

func (r *userRedis) GetCaptcha(ctx context.Context, areaCode, phone string) (*dto.CaptchaStore, error) {
	p := areaCode + phone
	val, err := r.rdb.Get(ctx, "Captcha:"+p).Result()
	if err != nil {
		return nil, err
	}
	var store dto.CaptchaStore
	if err := json.Unmarshal([]byte(val), &store); err != nil {
		return nil, err
	}
	return &store, nil
}

func (r *userRedis) DeleteCaptcha(ctx context.Context, areaCode, phone string) error {
	p := areaCode + phone
	return r.rdb.Del(ctx, "Captcha:"+p).Err()
}

func (r *userRedis) SetSession(ctx context.Context, session *dto.UserSession) error {
	key := "session:" + strconv.FormatInt(session.UserID, 10)
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, key, data, 10*time.Hour).Err()
}

func (r *userRedis) GetSession(ctx context.Context, userId int64) (*dto.UserSession, error) {
	key := "session:" + strconv.FormatInt(userId, 10)
	var res, err = r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, errors.New("session not found")
	}
	if err != nil {
		return nil, err
	}
	var session dto.UserSession
	err = json.Unmarshal([]byte(res), &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *userRedis) DelSession(ctx context.Context, userId int64) error {
	key := "session:" + strconv.FormatInt(userId, 10)
	return r.rdb.Del(ctx, key).Err()
}
