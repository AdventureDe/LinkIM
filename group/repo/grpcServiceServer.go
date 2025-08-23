package repo

import (
	"context"
	"fmt"
	"log"

	grouppb "github.com/AdventureDe/LinkIM/api/group"
	"github.com/AdventureDe/LinkIM/group/repo/model"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 用 protoc + Go 插件生成代码以后，会得到：
// 接口 GroupServiceServer
// 空实现 UnimplementedGroupServiceServer
// 客户端 GroupServiceClient

// 需要自己写一个 struct 去实现 GroupServiceServer 接口：

type GroupServiceServer struct {
	grouppb.UnimplementedGroupServiceServer
	repo GroupRepo
}

func NewGroupServiceServer(r GroupRepo) *GroupServiceServer {
	return &GroupServiceServer{
		repo: r,
	}
}

func toProtoRole(r model.GroupRole) grouppb.Role {
	switch r {
	case model.Member:
		return grouppb.Role_ROLE_MEMBER
	case model.Admin:
		return grouppb.Role_ROLE_ADMIN
	case model.Owner:
		return grouppb.Role_ROLE_OWNER
	default:
		return grouppb.Role_ROLE_UNSPECIFIED
	}
}

// for grpc
func (s *GroupServiceServer) ListGroupMembers(ctx context.Context, req *grouppb.ListGroupMembersRequest,
) (*grouppb.ListGroupMembersResponse, error) {

	groupID := req.GetGroupId()
	if groupID == "" {
		return nil, status.Error(codes.InvalidArgument, "group_id is required")
	}

	// 调用 repo 获取数据
	members, err := s.repo.GetGroupMembers(ctx, uuid.MustParse(groupID))
	if err != nil {
		log.Printf("failed to get group members: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get members")
	}

	// 转换成 pb 类型
	pbMembers := make([]*grouppb.GroupMember, 0, len(members))
	for _, m := range members {
		pbMembers = append(pbMembers, &grouppb.GroupMember{
			UserId:   m.UserID,
			Role:     toProtoRole(m.Role), // 转换为proto里面的Role 枚举
			Nickname: m.Nickname,
			JoinTime: m.JoinTime.GoString(),
		})
	}

	return &grouppb.ListGroupMembersResponse{
		Members: pbMembers,
	}, nil
}

func (s GroupServiceServer) ListGroupInfos(ctx context.Context, req *grouppb.ListGroupInfosRequest) (
	*grouppb.ListGroupInfosResponse, error,
) {
	groupIDs := req.GetGroupId()
	if len(groupIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "group_ids is required")
	}

	var groupIDu []uuid.UUID
	for _, str := range groupIDs {
		temp, err := uuid.Parse(str)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "group_ids is invalid")
		}
		groupIDu = append(groupIDu, temp)
	}

	infos, err := s.repo.GetGroupInfos(ctx, groupIDu)
	if err != nil {
		log.Printf("failed to get group infos: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get infos")
	}
	fmt.Printf("infos: %+v\n", infos) // %+v 打印字段名和值
	pbInfos := make([]*grouppb.GroupInfo, 0, len(infos))
	for _, i := range infos {
		pbInfos = append(pbInfos, &grouppb.GroupInfo{
			GroupId:   i.GroupID.String(),
			GroupName: i.GroupName,
			Avatar:    i.Avatar,
		})
	}
	fmt.Printf("pbInfos: %+v\n", pbInfos)
	fmt.Printf("type of GroupID field: %T\n", infos[0].GroupID)

	return &grouppb.ListGroupInfosResponse{Groups: pbInfos}, nil
}
