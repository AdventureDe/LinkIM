package main

import (
	"log"

	"github.com/AdventureDe/LinkIM/message/config"
	"github.com/AdventureDe/LinkIM/message/handler"
	"github.com/AdventureDe/LinkIM/message/repo"
	"github.com/AdventureDe/LinkIM/message/router"
	"github.com/AdventureDe/LinkIM/message/service"

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
	//grpc
	m, err := repo.NewMessageService("localhost:50051") //镜像需要改为user-service
	if err != nil {
		log.Fatalf("Fail to initialize Grpc:%v", err)
	}
	defer m.Close()

	r := gin.Default()
	r.Use(cors.New(config.CorsConfig))

	messageRepo := repo.NewMessageRepo(db, m)
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
