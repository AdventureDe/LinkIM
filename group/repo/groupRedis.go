package repo

import "github.com/go-redis/redis/v8"

type GroupRedis interface {
}

type groupRedis struct {
	redis *redis.Client
}

func NewGroupRedis(r *redis.Client) GroupRedis {
	return &groupRedis{
		redis: r,
	}
}
