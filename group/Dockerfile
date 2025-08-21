FROM golang:1.24 as builder

WORKDIR /app

# 国内可以加代理
ENV GOPROXY=https://goproxy.cn,direct

# 先拷贝 go.mod 和 go.sum
COPY group/go.mod group/go.sum ./group/
COPY api/go.mod api/go.sum ./api/

# 进入 group 目录下载依赖
WORKDIR /app/group
RUN go mod download

# 再拷贝源码
COPY api/ ../api/
COPY group/ ./

# 编译 grpc 服务
RUN CGO_ENABLED=0 GOOS=linux go build -o group-service ./cmd

# 运行阶段
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/group/group-service .

CMD ["./group-service"]