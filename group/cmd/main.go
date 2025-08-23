package main

import (
	"log"
	"net"

	grouppb "github.com/AdventureDe/LinkIM/api/group"
	"github.com/AdventureDe/LinkIM/group/config"
	"github.com/AdventureDe/LinkIM/group/handler"
	"github.com/AdventureDe/LinkIM/group/repo"
	"github.com/AdventureDe/LinkIM/group/router"
	"github.com/AdventureDe/LinkIM/group/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

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
	// grpc 服务器 group-service 在50053启动
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	//grpc 客户端
	m, err := repo.NewGroupService("localhost:50051") // 镜像需要改为user-service
	if err != nil {
		log.Fatalf("Fail to initialize Grpc:%v", err)
	}
	defer m.Close()

	r := gin.Default()
	r.Use(cors.New(config.CorsConfig))

	groupRepo := repo.NewGroupRepo(db, m)
	groupRedis := repo.NewGroupRedis(rdb)
	groupService := service.NewGroupService(groupRepo, groupRedis)
	groupHandler := handler.NewGroupHandler(groupService)
	router.SetGroupRouter(r, groupHandler)

	// 初始化grpc
	grpcServer := grpc.NewServer()
	groupServer := repo.NewGroupServiceServer(groupRepo) //放入repo
	grouppb.RegisterGroupServiceServer(grpcServer, groupServer)
	reflection.Register(grpcServer)
	// grpc服务器
	go func() {
		log.Println("GroupService gRPC listening on :50053")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 启动 HTTP 服务
	log.Printf("Group service started at http://localhost:%d", cfg.Port)
	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
