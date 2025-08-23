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
	"go.uber.org/zap"
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
	m, err := repo.NewMessageService("localhost:50051", "localhost:50053") //镜像需要改为user-service,group-service
	if err != nil {
		log.Fatalf("Fail to initialize Grpc:%v", err)
	}
	defer m.Close()
	//logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync() // flush buffer, 避免丢日志
	r := gin.Default()
	r.Use(cors.New(config.CorsConfig))

	messageRepo := repo.NewMessageRepo(db, m)
	messageService := service.NewMessageService(messageRepo, rdb, logger)
	messageHandler := handler.NewMessageHandler(messageService)
	router.SetMessageRouter(r, messageHandler)
	//
	// 启动 HTTP 服务
	log.Printf("User service started at http://localhost:%d", cfg.Port)
	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}
