package repo

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	grouppb "github.com/AdventureDe/LinkIM/api/group"
	userpb "github.com/AdventureDe/LinkIM/api/user"
)

type messageService struct {
	userConn    *grpc.ClientConn
	groupConn   *grpc.ClientConn
	userClient  userpb.UserServiceClient
	groupClient grouppb.GroupServiceClient
}

func NewMessageService(userAddr string, groupAddr string) (*messageService, error) {
	userConn, err := grpc.NewClient(
		userAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	groupConn, err := grpc.NewClient(
		groupAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	userClient := userpb.NewUserServiceClient(userConn)
	groupClient := grouppb.NewGroupServiceClient(groupConn)
	return &messageService{
		userConn:    userConn,
		groupConn:   groupConn,
		userClient:  userClient,
		groupClient: groupClient,
	}, nil
}

// 记得增加一个关闭方法
func (s *messageService) Close() {
	if s.userConn != nil {
		_ = s.userConn.Close()
	}
	if s.groupConn != nil {
		_ = s.groupConn.Close()
	}
}
