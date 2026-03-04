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
	"google.golang.org/grpc/reflection"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// 测试账号：  12321412411
// 测试密码:   qwer12345678
// cd cmd      |  go run main.go
func main() {
	cfg := config.Load()

	// 1. 初始化数据库 (传入动态 Host)
	db, err := repo.InitDB(cfg.DBHost)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	db = db.Debug()
	defer repo.CloseDB()

	// 2. 初始化 Redis (传入动态 Host)
	rdb, err := repo.InitRedis(cfg.RedisHost)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer repo.CloseRedis()

	// 3. gRPC 服务器监听配置 (对外提供服务，端口保持写死或移入 cfg 均可)
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 4. 创建 Gin 引擎并配置 CORS
	r := gin.Default()
	r.Use(cors.New(config.CorsConfig))

	// 5. 初始化核心架构层
	userRepo := repo.NewUserRepo(db)
	userRepoRedis := repo.NewUserRedis(rdb)

	userService := service.NewUserService(userRepo, userRepoRedis)
	userHandler := handler.NewUserHandler(userService)
	router.SetupRouter(r, userHandler)
	router.SetupFriendRouter(r, userHandler)

	userServiceWithRedis := service.NewVerificationService(userRepoRedis)
	userHandlerWithRedis := handler.NewVerificationHandler(userServiceWithRedis)
	router.SetupVerificationRouter(r, userHandlerWithRedis)

	// 6. 初始化并注册 gRPC 服务
	grpcServer := grpc.NewServer()
	userServer := repo.NewUserServiceServer(userRepo)
	userpb.RegisterUserServiceServer(grpcServer, userServer)
	reflection.Register(grpcServer)

	// 7. 并发启动 gRPC 服务器
	go func() {
		log.Println("UserService gRPC listening on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// 8. 启动 HTTP 服务
	log.Printf("User service started at http://0.0.0.0:%d", cfg.Port)
	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
