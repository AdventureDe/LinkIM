package main

import (
	"log"
	"net"

	userpb "github.com/AdventureDe/LinkIM/api/user"
	"github.com/AdventureDe/LinkIM/user/config"
	"github.com/AdventureDe/LinkIM/user/handler"
	"github.com/AdventureDe/LinkIM/user/repo"
	"github.com/AdventureDe/LinkIM/user/router"
	"github.com/AdventureDe/LinkIM/user/service"
	"google.golang.org/grpc"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// 测试账号：  12321412411
// 测试密码:   qwer12345678
// cd cmd      |  go run main.go
func main() {
	cfg := config.Load()
	db, err := repo.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	db = db.Debug()
	defer repo.CloseDB()

	rdb, err := repo.InitRedis()
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer repo.CloseRedis()

	// grpc 服务器
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 创建Gin引擎
	r := gin.Default()

	// 配置CORS（更安全的配置）
	r.Use(cors.New(config.CorsConfig))

	// 初始化仓库和服务
	userRepo := repo.NewUserRepo(db)
	userRepoRedis := repo.NewUserRedis(rdb)
	//------
	userService := service.NewUserService(userRepo, userRepoRedis)
	userHandler := handler.NewUserHandler(userService)
	router.SetupRouter(r, userHandler)
	router.SetupFriendRouter(r, userHandler)
	userServiceWithRedis := service.NewVerificationService(userRepoRedis)
	userHandlerWithRedis := handler.NewVerificationHandler(userServiceWithRedis)
	router.SetupVerificationRouter(r, userHandlerWithRedis)
	// 初始化grpc
	grpcServer := grpc.NewServer()
	userServer := repo.NewUserServiceServer(userRepo) //放入repo
	userpb.RegisterUserServiceServer(grpcServer, userServer)

	log.Println("UserService gRPC listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	// 启动 HTTP 服务
	log.Printf("User service started at http://localhost:%d", cfg.Port)
	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
