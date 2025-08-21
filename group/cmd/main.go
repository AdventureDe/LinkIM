package main

import (
	"log"

	"github.com/AdventureDe/tempName/group/config"
	"github.com/AdventureDe/tempName/group/handler"
	"github.com/AdventureDe/tempName/group/repo"
	"github.com/AdventureDe/tempName/group/router"
	"github.com/AdventureDe/tempName/group/service"
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
	m, err := repo.NewGroupService("localhost:50051") //镜像需要改为user-service
	if err != nil {
		log.Fatalf("Fail to initialize Grpc:%v", err)
	}
	defer m.Close()
	r := gin.Default()
	r.Use(cors.New(config.CorsConfig))

	groupRepo := repo.NewGroupRepo(db,m)
	groupRedis := repo.NewGroupRedis(rdb)
	groupService := service.NewGroupService(groupRepo, groupRedis)
	groupHandler := handler.NewGroupHandler(groupService)
	router.SetGroupRouter(r, groupHandler)
	//
	// 启动 HTTP 服务
	log.Printf("User service started at http://localhost:%d", cfg.Port)
	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
