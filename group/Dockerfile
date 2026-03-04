FROM golang:1.24 AS builder

WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct

COPY go.work ./
COPY go.work.sum ./ 

COPY group/go.mod ./group/
COPY api/go.mod ./api/
COPY message/go.mod ./message/
COPY user/go.mod ./user/

COPY group/go.sum ./group/
COPY api/go.sum ./api/
COPY message/go.sum ./message/
COPY user/go.sum ./user/

WORKDIR /app/message
RUN go mod download

WORKDIR /app
COPY api/ ./api/
COPY message/ ./message/
COPY user/ ./user/
COPY group/ ./group/

WORKDIR /app/message
RUN CGO_ENABLED=0 GOOS=linux go build -o message-service ./cmd/main.go

FROM alpine:latest
WORKDIR /root/
RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone
COPY --from=builder /app/message/message-service .
CMD ["./message-service"]