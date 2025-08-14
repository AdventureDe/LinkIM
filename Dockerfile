FROM golang:1.24 as builder

WORKDIR /app

COPY user/go.mod ./ 
COPY user/go.sum ./

RUN go mod download

# 把 user 目录下的代码全部拷贝进来
COPY user/. .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ./cmd

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/app .

CMD ["./app"]