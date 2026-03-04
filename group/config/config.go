package config

import (
	"os"
	"strconv"

	"github.com/gin-contrib/cors"
)

type Config struct {
	Port            int
	DBHost          string // 数据库地址
	RedisHost       string // Redis地址
	UserServiceAddr string // 新增：User 服务 gRPC 地址！(刚才结构体里漏了这行)
	// KafkaHost    string // 如果 group 暂未用到 Kafka，这行可以注释掉或删掉
}

var CorsConfig = cors.Config{
	AllowOrigins:     []string{"http://localhost:8080", "*"}, // 测试阶段可以加上 "*" 放行
	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
	AllowHeaders:     []string{"*"},
	ExposeHeaders:    []string{"X-My-Custom-Header"},
	AllowCredentials: true,
}

// 辅助函数：优先读取环境变量，如果没有就用 fallback 默认值
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func Load() *Config {
	port := 10009 // Group 服务的默认端口
	// 允许通过环境变量修改端口
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	return &Config{
		Port: port,
		// 核心秘诀：默认值写本地的，部署时通过 Docker 注入环境变量覆盖它！
		DBHost:          getEnv("DB_HOST", "localhost"),
		RedisHost:       getEnv("REDIS_HOST", "localhost"),
		UserServiceAddr: getEnv("USER_HOST", "localhost:50051"), // 指向 User 服务的 gRPC 端口
	}
}

// Addr 返回监听地址
func (c *Config) Addr() string {
	return ":" + strconv.Itoa(c.Port)
}
