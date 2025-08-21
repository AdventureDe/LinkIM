package repo

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	userpb "github.com/AdventureDe/tempName/api/user"
)

type groupService struct {
	conn       *grpc.ClientConn
	userClient userpb.UserServiceClient
}

func NewGroupService(userAddr string) (*groupService, error) {
	conn, err := grpc.NewClient(
		userAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	client := userpb.NewUserServiceClient(conn)
	return &groupService{
		conn:       conn,
		userClient: client,
	}, nil
}

// 记得增加一个关闭方法
func (s *groupService) Close() {
	if s.conn != nil {
		_ = s.conn.Close()
	}
}
