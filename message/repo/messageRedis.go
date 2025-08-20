package repo

import "github.com/go-redis/redis/v8"

type MessageRedis interface {
}

type messageRedis struct {
	redis *redis.Client
}

func NewMessageRedis(r *redis.Client) MessageRedis {
	return &messageRedis{
		redis: r,
	}
}
