package repo

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
)

var RDB *redis.Client

// 初始化 Redis 客户端
func InitRedis() (*redis.Client, error) {
	// 配置 Redis 连接
	RDB = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis 服务器地址 后端在docker运行 改为  redis:6379
		Password: "12345678",       // Redis 密码
		DB:       0,                // 使用默认的 DB（0）
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

// 关闭 Redis 客户端连接
func CloseRedis() {
	if RDB != nil {
		err := RDB.Close() // 关闭 Redis 连接
		if err != nil {
			log.Println("关闭 Redis 连接失败：", err)
		}
	}
}
