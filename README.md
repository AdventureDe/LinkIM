# LinkIM

[![Version](https://img.shields.io/badge/Version-1.0.0-brightgreen)](https://github.com/AdventureDe/LinkIM) [![Go Report Card](https://goreportcard.com/badge/github.com/AdventureDe/LinkIM)](https://goreportcard.com/report/github.com/AdventureDe/LinkIM) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**LinkIM** 是一个高性能、模块化的分布式即时通讯（IM）系统后端项目。采用微服务架构，集成了gRPC、Redis、Kafka等现代技术栈，提供了完整的用户管理、实时消息推送、群组聊天以及消息持久化等核心功能。

## 🚀 功能特性

### 1️⃣ 用户服务 (User Service)
负责用户身份认证与关系管理。
- **用户认证** - 注册、登录、登出与JWT Token验证
- **资料管理** - 维护用户个人资料，如昵称、头像、签名、手机和邮箱
- **关系链** - 好友的添加、删除、列表查询
- **黑名单** - 屏蔽与解除屏蔽用户
- **会话管理** - 用户会话列表维护

### 2️⃣ 消息服务 (Message Service)
负责消息的存储、状态管理与历史查询。
- **消息处理** - 可靠存储单聊与群聊消息
- **状态同步** - 管理消息的发送、送达和已读状态
- **历史记录** - 支持按会话分页拉取聊天记录
- **消息队列** - 使用Kafka进行异步消息处理
- **实时推送** - 通过Redis Pub/Sub实现消息实时推送

### 3️⃣ 群组服务 (Group Service)
负责群组与成员管理。
- **群组管理** - 创建和解散群组
- **成员管理** - 邀请成员、踢出成员、设置管理员
- **群组信息** - 编辑群名称、群公告、群头像
- **身份管理** - 灵活的群主、管理员、普通成员角色体系

## 📦 系统架构

LinkIM 采用微服务架构，使用gRPC进行服务间通信，Redis Pub/Sub实现实时消息推送，Kafka处理异步消息任务。

### 技术栈
| 组件 | 技术选型 | 用途 |
| :--- | :--- | :--- |
| **API网关** | Gin | HTTP请求路由、认证、负载均衡 |
| **服务通信** | gRPC | 微服务间高性能通信 |
| **数据库** | PostgreSQL | 核心数据持久化存储 |
| **缓存** | Redis | 会话缓存、Pub/Sub消息推送 |
| **消息队列** | Kafka | 异步消息处理、削峰填谷 |
| **部署** | Docker, Docker Compose | 容器化部署与管理 |

### 服务部署
| 服务 | 端口 | 协议 | 描述 |
| :--- | :--- | :--- | :--- |
| **User Service** | 10008 | gRPC | 用户服务，处理用户相关功能 |
| **Message Service** | 10009 | gRPC | 消息服务，处理消息存储与推送 |
| **Group Service** | 10010 | gRPC | 群组服务，管理群组相关功能 |
| **API Gateway** | 8080 | HTTP | 统一API入口，对外提供服务 |

### 架构图
```
客户端 → API Gateway (HTTP) → 微服务集群 (gRPC)
                             │
                             ├─ User Service (10008) → PostgreSQL → Redis
                             ├─ Message Service (10009) → PostgreSQL → Redis → Kafka
                             └─ Group Service (10010) → PostgreSQL → Redis
```

## 🔌 API 接口文档

- User Service Module: https://documenter.getpostman.com/view/47474975/2sB3BHkUXd
- Message Service Module: https://documenter.getpostman.com/view/47474975/2sB3BLk7y2
- Group Service Module: https://documenter.getpostman.com/view/47474975/2sB3BLk7y5

## 🛠️ 快速开始

### 前置要求
- Go 1.18+
- Docker & Docker Compose
- Git

### 使用Docker一键部署

1. **克隆项目**
   ```bash
   git clone https://github.com/AdventureDe/LinkIM.git
   cd LinkIM
   ```

2. **启动所有服务**
   ```bash
   # 使用docker-compose一键启动所有服务
   docker-compose up -d
   
   # 或者使用Makefile
   make start
   ```

   这将启动以下容器：
   - PostgreSQL数据库
   - Redis缓存
   - Kafka消息队列
   - User Service (端口: 10008)
   - Message Service (端口: 10009)
   - Group Service (端口: 10010)
   - API Gateway (端口: 8080)

3. **验证服务状态**
   ```bash
   docker-compose ps
   ```

### 手动开发环境部署

1. **启动依赖服务**
   ```bash
   # 只启动数据库、Redis和Kafka
   docker-compose up -d postgres redis kafka zookeeper
   ```

2. **配置环境变量**
   复制并修改配置文件：
   ```bash
   cp config.example.yaml config.yaml
   ```
   更新数据库和Redis连接信息。

3. **初始化数据库**
   ```bash
   # 各服务会自动创建表结构
   make migrate
   ```

4. **启动各个服务**
   ```bash
   # 启动用户服务
   cd user && go run cmd/main.go
   
   # 启动消息服务 (新终端)
   cd message && go run cmd/main.go
   
   # 启动群组服务 (新终端)
   cd group && go run cmd/main.go
   
   # 启动API网关 (新终端)
   cd gateway && go run main.go
   ```

### 使用示例

以下是一个用户登录并发送消息的示例流程：

1. **用户注册**
   ```bash
    curl --location 'http://localhost:10008/account/register' \
    --header 'Content-Type: application/json' \
    --data-raw '{
    "verifyCode": "123456",
    "platform": 1,
    "autoLogin": true,
    "user": {
        "phoneNumber": "13800138000",
        "areaCode": "+86",
        "nickname": "testuser",
        "password": "Password123!",
        "confirmPassword": "Password123!",
        "email": "test@example.com",
        "invitationCode": "ABC123"
    }
    }'
   ```

2. **用户登录**
   ```bash
    curl --location 'http://localhost:10008/account/login' \
    --header 'Content-Type: application/json' \
    --data-raw '{
    "phoneNumber": "12321412411",
    "email": "test@example.com",
    "areaCode": "+86",
    "password": "qwer12345678",
    "platform": 2,
    "verifyCode": "654321"
    }'
   ```
   *响应中将包含一个JWT token，用于后续请求的认证。*

3. **发送消息**
   ```bash
    curl --location -g 'http://localhost:10009/message/send' \
    --header 'Content-Type: application/json' \
    --data '{
        "user_id":18,
        "platform":2,
        "the_other_person_id":5,
        "text":"为什么要这样做呢"
    }'
   ```

## 📁 项目结构

```
LinkIM/
├── api/                          # Protobuf定义和生成的gRPC代码
│   ├── group/                    # 群组服务gRPC定义
│   ├── message/                  # 消息服务gRPC定义
│   └── user/                     # 用户服务gRPC定义
├── gateway/                      # API网关
│   ├── config/                   # 网关配置
│   ├── handler/                  # HTTP处理器
│   ├── service/                  # 网关服务层
│   └── main.go                   # 网关入口
├── group/                        # 群组服务
│   ├── cmd/main.go               # 服务入口
│   ├── config/                   # 配置管理
│   ├── handler/                  # gRPC处理器
│   ├── repo/                     # 数据访问层
│   ├── service/                  # 业务逻辑层
│   └── Dockerfile                # 容器化配置
├── message/                      # 消息服务
│   ├── cmd/main.go               # 服务入口
│   ├── config/                   # 配置管理
│   ├── handler/                  # gRPC处理器
│   ├── repo/                     # 数据访问层
│   ├── service/                  # 业务逻辑层
│   └── Dockerfile                # 容器化配置
├── user/                         # 用户服务
│   ├── cmd/main.go               # 服务入口
│   ├── config/                   # 配置管理
│   ├── handler/                  # gRPC处理器
│   ├── repo/                     # 数据访问层
│   ├── service/                  # 业务逻辑层
│   └── Dockerfile                # 容器化配置
├── docker-compose.yaml           # 容器编排配置
├── Makefile                      # 项目构建工具
└── go.work                       # Go工作区配置
```

## 🧪 测试

```bash
# 运行所有单元测试
make test

# 运行特定服务的测试
make test-user
make test-message
make test-group

# 构建所有Docker镜像
make build

# 启动所有服务
make start

# 停止所有服务
make stop
```

## 🤝 如何贡献

我们欢迎任何形式的贡献！
1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交您的更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开一个 Pull Request

## 📄 许可证

该项目基于 MIT 许可证开源。详情请参阅 [LICENSE](LICENSE) 文件。

## 🚧 版本管理

本项目使用 [Semantic Versioning](https://semver.org/) 进行版本编号。
- **当前版本：** `1.0.0`
- **最新更新：** `2025-08-14`

## 🔮 未来规划

- [x] 集成gRPC实现服务间通信
- [x] 使用Redis Pub/Sub实现消息实时推送
- [x] 容器化部署与Docker Compose编排
- [ ] 消息已读回执功能完善
- [ ] 分布式会话管理
- [ ] 消息历史记录云端同步
- [ ] 移动端SDK开发
- [ ] 管理后台与数据统计

---

**如有问题或建议，请随时提出 [Issue](https://github.com/AdventureDe/LinkIM/issues)。**