FROM golang:1.24 as builder

WORKDIR /app

# 国内可以加代理
ENV GOPROXY=https://goproxy.cn,direct

# 先拷贝 go.mod 和 go.sum，利用缓存加速构建
COPY message/go.mod message/go.sum ./message/
COPY api/go.mod api/go.sum ./api/

# 进入 message 目录下载依赖
WORKDIR /app/message
RUN go mod download

# 再拷贝源码
COPY api/ ../api/
COPY message/ ./

# 编译 message 服务
RUN CGO_ENABLED=0 GOOS=linux go build -o message-service ./cmd

# 运行阶段 (使用极简的 alpine 镜像)
FROM alpine:latest
WORKDIR /root/

# 配置时区为上海（解决容器内时间相差8小时的问题）
RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 从 builder 拷贝编译好的二进制文件
COPY --from=builder /app/message/message-service .

CMD ["./message-service"]