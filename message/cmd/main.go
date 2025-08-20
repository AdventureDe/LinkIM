package main

import (
	"log"
	"message/config"
	"message/handler"
	"message/repo"
	"message/router"
	"message/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	db, err := repo.InitDB()
	if err != nil {
		log.Fatalf("Fail to initialize Database:%v", err)
	}
	db = db.Debug()
	defer repo.CloseDB()

	rdb, err := repo.InitRedis()
	if err != nil {
		log.Fatalf("Fail to initialize Redis:%v", err)
	}
	defer repo.CloseRedis()
	r := gin.Default()
	r.Use(cors.New(config.CorsConfig))

	messageRepo := repo.NewMessageRepo(db)
	messageRedis := repo.NewMessageRedis(rdb)
	messageService := service.NewMessageService(messageRepo, messageRedis)
	messageHandler := handler.NewMessageHandler(messageService)
	router.SetMessageRouter(r, messageHandler)
	//
	// 启动 HTTP 服务
	log.Printf("User service started at http://localhost:%d", cfg.Port)
	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}
