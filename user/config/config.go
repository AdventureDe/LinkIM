package config

import (
	"os"
	"strconv"

	"github.com/gin-contrib/cors"
)

type Config struct {
	Port      int
	DBHost    string // 新增：数据库地址
	RedisHost string // 新增：Redis地址
	KafkaHost string // 新增：Kafka地址
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
	port := 10008 // 默认端口
	// 允许通过环境变量修改端口
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	return &Config{
		Port: port,
		// 核心秘诀在这里：默认值写本地的，部署时通过 Docker 注入环境变量覆盖它！
		DBHost:    getEnv("DB_HOST", "localhost"),
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		KafkaHost: getEnv("KAFKA_HOST", "localhost:19092"), // 本地默认用外部映射端口
	}
}

// Addr 返回监听地址
func (c *Config) Addr() string {
	return ":" + strconv.Itoa(c.Port)
}
