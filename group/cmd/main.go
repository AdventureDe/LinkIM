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

	// 1. 初始化数据库 (传入动态 Host)
	db, err := repo.InitDB(cfg.DBHost)
	if err != nil {
		log.Fatalf("Fail to initialize Database:%v", err)
	}
	db = db.Debug()
	defer repo.CloseDB()

	// 2. 初始化 Redis (传入动态 Host)
	rdb, err := repo.InitRedis(cfg.RedisHost)
	if err != nil {
		log.Fatalf("Fail to initialize Redis:%v", err)
	}
	defer repo.CloseRedis()

	// 3. gRPC 服务器监听配置 (在50053启动)
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 4. 初始化 gRPC 客户端去调用 user-service
	// ⚠️ 修复：将硬编码的 "localhost:50051" 替换为动态配置 cfg.UserServiceAddr
	m, err := repo.NewGroupService(cfg.UserServiceAddr)
	if err != nil {
		log.Fatalf("Fail to initialize Grpc client:%v", err)
	}
	defer m.Close()

	// 5. 初始化 HTTP 服务
	r := gin.Default()
	r.Use(cors.New(config.CorsConfig))

	// 6. 初始化核心架构层
	groupRepo := repo.NewGroupRepo(db, m)
	groupRedis := repo.NewGroupRedis(rdb)
	groupService := service.NewGroupService(groupRepo, groupRedis)
	groupHandler := handler.NewGroupHandler(groupService)
	router.SetGroupRouter(r, groupHandler)

	// 7. 初始化并注册 gRPC 服务端
	grpcServer := grpc.NewServer()
	groupServer := repo.NewGroupServiceServer(groupRepo)
	grouppb.RegisterGroupServiceServer(grpcServer, groupServer)
	reflection.Register(grpcServer)

	// 8. 并发启动 gRPC 服务器
	go func() {
		log.Println("GroupService gRPC listening on :50053")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 9. 启动 HTTP 服务
	log.Printf("Group service started at http://0.0.0.0:%d", cfg.Port)
	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
