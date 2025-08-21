package repo

import (
	"context"
	"log"

	userpb "github.com/AdventureDe/LinkIM/api/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserServiceServer struct {
	userpb.UnimplementedUserServiceServer
	repo UserRepo
}

func NewUserServiceServer(r UserRepo) *UserServiceServer {
	return &UserServiceServer{
		repo: r,
	}
}

// for grpc
// 实现批量获取的逻辑，使用批量查询数据库
func (s *UserServiceServer) GetUserInfos(ctx context.Context, req *userpb.GetUserInfosRequest) (*userpb.GetUserInfosResponse, error) {
	// 1. 从请求中获取用户ID列表
	userIDs := req.GetUserIds()
	if len(userIDs) == 0 {
		log.Printf("there is not userid")
		return &userpb.GetUserInfosResponse{Users: []*userpb.UserInfo{}}, nil
	}
	// 2. 调用批量查询方法获取所有用户信息
	users, err := s.repo.GetUserInfos(ctx, userIDs)
	if err != nil {
		log.Printf("Failed to get users: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to retrieve user information")
	}

	// 3. 创建映射以便快速查找特定用户
	userMap := make(map[int64]User)
	for _, user := range users {
		userMap[user.ID] = user
	}

	// 4. 准备响应数据
	userProtos := make([]*userpb.UserInfo, 0, len(userIDs))
	for _, id := range userIDs {
		if user, exists := userMap[id]; exists {
			// 转换 *repo.User -> *userpb.UserInfo
			userProto := &userpb.UserInfo{
				UserId:   user.ID,
				Nickname: user.Nickname,
				Avatar:   user.AvatarUrl,
			}
			userProtos = append(userProtos, userProto)
		} else {
			// 可以选择记录日志或跳过不存在的用户
			log.Printf("User with ID %d not found", id)
			// 也可以选择返回一个带有默认值的用户信息
			// userProtos = append(userProtos, &userpb.UserInfo{UserId: id})
		}
	}

	// 5. 组装响应
	return &userpb.GetUserInfosResponse{
		Users: userProtos,
	}, nil
}
