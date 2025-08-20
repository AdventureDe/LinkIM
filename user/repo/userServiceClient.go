package repo

import (
    "context"
    "google.golang.org/grpc"
    userpb "D:/grpc/bin/proto"
)

type messageService struct {
    userClient userpb.UserServiceClient
}

func NewMessageService(userAddr string) (*messageService, error) {
    conn, err := grpc.Dial(userAddr, grpc.WithInsecure()) // 生产要加 TLS
    if err != nil {
        return nil, err
    }
    client := userpb.NewUserServiceClient(conn)
    return &messageService{userClient: client}, nil
}
