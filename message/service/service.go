package service

import (
	"context"
	"fmt"

	"github.com/AdventureDe/tempName/message/repo"
)

type MessageService struct {
	repo  repo.MessageRepo
	redis repo.MessageRedis
}

func NewMessageService(r repo.MessageRepo, u repo.MessageRedis) *MessageService {
	return &MessageService{
		repo:  r,
		redis: u,
	}
}

func (s *MessageService) SendMessageToSingle(ctx context.Context, senderid int64, targetid int64, text string) (lastMsgId *int64, err error) {
	if lastMsgId, err = s.repo.SendMessageToSingle(ctx, senderid, targetid, text); err != nil {
		return
	}
	return
}

func (s *MessageService) GetConversationMessages(ctx context.Context,
	senderid, targetid int64, lastMsgid int64, pageSize int) (*repo.ConversationMessages, error) {

	messages, err := s.repo.GetConversationMessages(ctx, senderid, targetid, lastMsgid, pageSize)
	if err != nil {
		return nil, fmt.Errorf("fail to load messages:%w", err)
	}
	return messages, nil
}

func (s *MessageService) WithdrawMessage(ctx context.Context, senderid int64, targetid int64, messageid int64) (int64, error) {
	lastMsgID, err := s.repo.WithdrawMessage(ctx, senderid, targetid, messageid)
	if err != nil {
		return -1, fmt.Errorf("fail to withdraw this message:%d,error:%w", messageid, err)
	}
	return lastMsgID, nil
}

func (s *MessageService) UnWithdrawMessage(ctx context.Context, senderid int64, targetid int64,
	messageid int64, newtext string) (lastMessageID int64, err error) {
	lastMsgID, err := s.repo.UnWithdrawMessage(ctx, senderid, targetid, messageid, newtext)
	if err != nil {
		return -1, fmt.Errorf("fail to unwithdraw this message:%d,error:%w", messageid, err)
	}
	return lastMsgID, nil
}

func (s *MessageService) GetConversations(ctx context.Context, userID int64) ([]*repo.ConversationWithUser, error) {
	conversations, err := s.repo.GetConversations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fail to get conversations:%w", err)
	}
	return conversations, nil
}
