package config

import (
	"strconv"

	"github.com/gin-contrib/cors"
)

type Config struct {
	Port int
}

var CorsConfig = cors.Config{
	AllowOrigins:     []string{"http://localhost:8080"}, //跨域
	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
	AllowHeaders:     []string{"*"},
	ExposeHeaders:    []string{"X-My-Custom-Header"},
	AllowCredentials: true,
}

func Load() *Config {
	return &Config{
		Port: 10008, // 默认端口
	}
}

// Addr 返回监听地址
func (c *Config) Addr() string {
	return ":" + strconv.Itoa(c.Port)
}
