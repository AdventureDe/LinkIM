package repo

import (
	"context"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
)

var RDB *redis.Client

// InitRedis 初始化 Redis 客户端，接收动态 host 参数
func InitRedis(host string) (*redis.Client, error) {
	// 配置 Redis 连接
	RDB = redis.NewClient(&redis.Options{
		// 使用 fmt.Sprintf 动态拼接传进来的 host 和端口
		Addr:     fmt.Sprintf("%s:6379", host),
		Password: "12345678",
		DB:       0,
	})

	// 测试连接是否成功
	_, err := RDB.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("Redis 连接失败：", err)
		return nil, err
	}

	// 返回 Redis 客户端实例
	return RDB, nil
}

// CloseRedis 关闭 Redis 客户端连接
func CloseRedis() {
	if RDB != nil {
		err := RDB.Close() // 关闭 Redis 连接
		if err != nil {
			log.Println("关闭 Redis 连接失败：", err)
		}
	}
}
