package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/AdventureDe/LinkIM/message/config"
	"github.com/AdventureDe/LinkIM/message/handler"
	"github.com/AdventureDe/LinkIM/message/repo"
	"github.com/AdventureDe/LinkIM/message/router"
	"github.com/AdventureDe/LinkIM/message/service"
	"github.com/go-redis/redis/v8"

	"github.com/IBM/sarama"
	"github.com/bwmarrin/snowflake"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 加载配置（自动决定是用本地地址还是 Docker 云端地址）
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

	// 3. 初始化 gRPC 客户端
	// TODO: 部署到 Docker 后，这里的 localhost 也要改成 user-service 的容器名
	m, err := repo.NewMessageService("localhost:50051", "localhost:50053")
	if err != nil {
		log.Fatalf("Fail to initialize Grpc:%v", err)
	}
	defer m.Close()

	// 4. 初始化 Logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// 5. 初始化分布式 ID 生成器
	idGen, err := snowflake.NewNode(1)
	if err != nil {
		log.Fatalf("Fail to initialize Snowflake Node:%v", err)
	}

	// 6. 初始化 Kafka 异步生产者 (使用配置里的 Kafka 地址)
	kafkaBrokers := []string{cfg.KafkaHost}
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Return.Errors = true
	kafkaProducer, err := sarama.NewAsyncProducer(kafkaBrokers, producerConfig)
	if err != nil {
		log.Fatalf("Fail to initialize Kafka Producer:%v", err)
	}
	defer kafkaProducer.Close()

	go func() {
		for err := range kafkaProducer.Errors() {
			logger.Error("Kafka producer async error", zap.Error(err))
		}
	}()

	// 7. 初始化核心架构层
	messageRepo := repo.NewMessageRepo(db, m)
	messageService := service.NewMessageService(messageRepo, rdb, logger, kafkaProducer, idGen)
	messageHandler := handler.NewMessageHandler(messageService)

	// 8. 启动 Kafka 消费者
	consumerGroupID := "im_message_group"
	consumerClient := StartMessageConsumer(kafkaBrokers, consumerGroupID, messageRepo, rdb, logger)
	defer consumerClient.Close()

	// 9. 启动 HTTP 路由与服务
	r := gin.Default()
	r.Use(cors.New(config.CorsConfig))
	router.SetMessageRouter(r, messageHandler)

	log.Printf("Message service started at http://0.0.0.0:%d", cfg.Port)
	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// StartMessageConsumer 保持不变
func StartMessageConsumer(brokers []string, groupID string, messageRepo repo.MessageRepo, rdb *redis.Client, logger *zap.Logger) sarama.ConsumerGroup {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategySticky()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	client, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		panic(fmt.Sprintf("Error creating consumer group client: %v", err))
	}

	handler := service.NewConsumerHandler(messageRepo, rdb, logger)
	ctx := context.Background()

	go func() {
		for {
			topics := []string{"im_message_persistence_topic"}
			if err := client.Consume(ctx, topics, handler); err != nil {
				logger.Error("Error from Kafka consumer", zap.Error(err))
				time.Sleep(2 * time.Second)
			}
		}
	}()

	logger.Info("Kafka Consumer started successfully", zap.Strings("brokers", brokers), zap.String("topic", "im_message_persistence_topic"))

	return client
}
