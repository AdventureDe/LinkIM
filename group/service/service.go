package service

import (
	"context"

	"github.com/AdventureDe/tempName/group/repo"
	"github.com/google/uuid"
)

type GroupService struct {
	repo  repo.GroupRepo
	redis repo.GroupRedis
}

func NewGroupService(r repo.GroupRepo, u repo.GroupRedis) *GroupService {
	return &GroupService{
		repo:  r,
		redis: u,
	}
}

func (s *GroupService) CreateGroup(ctx context.Context, ownerID int64, userIDs []int64, groupName string) (uuid.UUID, error) {
	groupID, err := s.repo.CreateGroup(ctx, ownerID, userIDs, groupName)
	if err != nil {
		return uuid.Nil, err
	}
	return groupID, nil
}

func (s *GroupService) AddGroupMember(ctx context.Context, groupID uuid.UUID, userIDs []int64) error {
	return s.repo.AddGroupMember(ctx, groupID, userIDs)
}

func (s *GroupService) KickOutGroupMember(ctx context.Context, groupID uuid.UUID,
	executorID int64, userIDs []int64) error {
	return s.repo.KickOutGroupMember(ctx, groupID, executorID, userIDs)
}

func (s *GroupService) PromoteToAdmin(ctx context.Context, groupID uuid.UUID,
	executorID int64, userID int64) error {
	return s.repo.PromoteToAdmin(ctx, groupID, executorID, userID)
}

func (s *GroupService) TransferGroupOwner(ctx context.Context, groupID uuid.UUID,
	executorID int64, userID int64) error {
	return s.repo.TransferGroupOwner(ctx, groupID, executorID, userID)
}

func (s *GroupService) DemotedToMember(ctx context.Context, groupID uuid.UUID,
	executorID int64, userID int64) error {
	return s.repo.DemotedToMember(ctx, groupID, executorID, userID)
}

func (s *GroupService) UpdateNotice(ctx context.Context, groupID uuid.UUID,
	executorID int64, newNoticeText string) error {
	return s.repo.UpdateNotice(ctx, groupID, executorID, newNoticeText)
}

func (s *GroupService) GetNotice(ctx context.Context, groupID uuid.UUID) (string, error) {
	return s.repo.GetNotice(ctx, groupID)
}

func (s *GroupService) UpdateGroupName(ctx context.Context, groupID uuid.UUID,
	executorID int64, newGroupName string) error {
	return s.repo.UpdateGroupName(ctx, groupID, executorID, newGroupName)
}

func (s *GroupService) GetGroupName(ctx context.Context, groupID uuid.UUID) (string, error) {
	return s.repo.GetGroupName(ctx, groupID)
}

func (s *GroupService) GetGroupAvatar(ctx context.Context, groupID uuid.UUID) (*repo.GroupAvatarSet, error) {
	return s.repo.GetGroupAvatar(ctx, groupID)
}

func (s *GroupService) UpdateSelfName(ctx context.Context, groupID uuid.UUID,
	userID int64, newName string) error {
	return s.repo.UpdateSelfName(ctx, groupID, userID, newName)
}
