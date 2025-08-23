package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AdventureDe/LinkIM/message/dto"
	"github.com/AdventureDe/LinkIM/message/repo"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MessageService struct {
	repo   repo.MessageRepo
	rdb    *redis.Client
	logger *zap.Logger
}

func NewMessageService(r repo.MessageRepo, u *redis.Client, l *zap.Logger) *MessageService {
	return &MessageService{
		repo:   r,
		rdb:    u,
		logger: l,
	}
}

func (s *MessageService) SendMessageToSingle(ctx context.Context, senderID, targetID int64, text string) (*int64, error) {
	// 参数校验
	if senderID <= 0 || targetID <= 0 || senderID == targetID {
		return nil, errors.New("invalid senderID or targetID")
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("message text cannot be empty")
	}

	// Step 1. 消息入库
	msgID, err := s.repo.SendMessageToSingle(ctx, senderID, targetID, text)
	if err != nil {
		s.logger.Error("failed to persist message",
			zap.Int64("senderID", senderID),
			zap.Int64("targetID", targetID),
			zap.String("text", text),
			zap.Error(err),
		)
		return nil, fmt.Errorf("persist message failed: %w", err)
	}

	// Step 2. 消息推送 (Redis Pub/Sub)
	channel := fmt.Sprintf("user:%d:messages", targetID)
	payload := fmt.Sprintf(`{"msgID":%d,"from":%d,"to":%d,"text":"%s"}`, *msgID, senderID, targetID, text)
	if err := s.rdb.Publish(ctx, channel, payload).Err(); err != nil {
		s.logger.Warn("failed to push message via redis",
			zap.String("channel", channel),
			zap.String("payload", payload),
			zap.Error(err),
		)
	}

	s.logger.Info("message sent successfully",
		zap.Int64("senderID", senderID),
		zap.Int64("targetID", targetID),
		zap.Int64("msgID", *msgID),
	)

	return msgID, nil
}

func (s *MessageService) SendMessageToGroup(ctx context.Context, senderID int64, groupID uuid.UUID,
	text string) (lastMsgId *int64, err error) {
	if senderID <= 0 || groupID == uuid.Nil {
		return nil, errors.New("invalid senderID or groupID")
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("text cannot be empty")
	}

	// 持久化
	msgID, err := s.repo.SendMessageToGroup(ctx, senderID, groupID, text)
	if err != nil {
		s.logger.Error("failed to persist message",
			zap.Int64("senderID", senderID),
			zap.String(" groupID", groupID.String()),
			zap.String("text", text),
			zap.Error(err),
		)
		return nil, fmt.Errorf("persist message failed: %w", err)
	}

	//实时推送, 到redis
	channel := fmt.Sprintf("groupID:%s:messages", groupID.String())
	payload := fmt.Sprintf(`{"msgID":%d,"from":%d,"to":%s,"text":"%s"}`, *msgID, senderID, groupID.String(), text)
	if err := s.rdb.Publish(ctx, channel, payload).Err(); err != nil {
		s.logger.Warn("failed to push message via redis",
			zap.String("channel", channel),
			zap.String("payload", payload),
			zap.Error(err),
		)
	}

	s.logger.Info("message sent successfully",
		zap.Int64("senderID", senderID),
		zap.String("groupID", groupID.String()),
		zap.Int64("msgID", *msgID),
	)

	return msgID, nil

}

func (s *MessageService) GetConversationMessagesSingle(ctx context.Context,
	senderID, targetID int64, lastMsgID int64, pageNum int, pageSize int) (*dto.ConversationMessagesDTO, error) {
	// 只缓存第一页（pageNum == 1）
	useCache := (pageNum == 1) //缓存3页  useCache := (pageNum <= 3)

	var cacheKey string
	if useCache {
		cacheKey = fmt.Sprintf("conv:%d:tar:%d:page:1", senderID, targetID)
		// 1. 尝试读缓存
		if val, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var cached dto.ConversationMessagesDTO
			if jsonErr := json.Unmarshal([]byte(val), &cached); jsonErr == nil { //作为json存入redis
				return &cached, nil
			}
			_ = s.rdb.Del(ctx, cacheKey).Err() // 删除坏数据
		} else if err != redis.Nil {
			s.logger.Warn("redis get failed", zap.Error(err))
		}
	}

	// 2. 查数据库
	cm, err := s.repo.GetConversationMessagesSingle(ctx, senderID, targetID, lastMsgID, pageSize)
	if err != nil {
		return nil, err
	}

	msgs := make([]*dto.MessageDTO, len(cm.Messages))
	for i, m := range cm.Messages {
		msgs[i] = &dto.MessageDTO{
			ID:         m.Message.ID,
			Content:    m.Message.Content,
			Sender:     m.Message.SenderID,
			CreateTime: m.Message.CreatedAt,
			UserInfo: &dto.UserInfoDTO{
				UserID:       m.User.UserID,
				SelfNickname: m.User.Nickname,
				Avatar:       m.User.Avatar,
			},
		}
	}

	dtoResult := &dto.ConversationMessagesDTO{
		ThreadID: cm.Thread.ID,
		Messages: msgs,
		HasMore:  cm.HasMore,
		Unread:   cm.Unread,
	}

	// 3. 如果是第一页，写入缓存
	if useCache {
		if bytes, err := json.Marshal(dtoResult); err == nil {
			if err := s.rdb.Set(ctx, cacheKey, bytes, 5*time.Minute).Err(); err != nil {
				s.logger.Warn("redis set failed", zap.Error(err))
			}
		}
	}

	return dtoResult, nil
}

func (s *MessageService) GetConversationMessagesGroup(ctx context.Context, senderID int64, groupID uuid.UUID,
	lastMsgID int64, pageNum int, pageSize int,
) (*dto.ConversationMessagesDTO, error) {
	// 只缓存第一页（pageNum == 1）
	useCache := (pageNum == 1) //缓存3页  useCache := (pageNum <= 3)

	var cacheKey string
	if useCache {
		cacheKey = fmt.Sprintf("conv:%d:grp:%s:page:1", senderID, groupID.String())
		// 1. 尝试读缓存
		if val, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var cached dto.ConversationMessagesDTO
			if jsonErr := json.Unmarshal([]byte(val), &cached); jsonErr == nil { //作为json存入redis
				return &cached, nil
			}
			_ = s.rdb.Del(ctx, cacheKey).Err() // 删除坏数据
		} else if err != redis.Nil {
			s.logger.Warn("redis get failed", zap.Error(err))
		}
	}

	// 2. 查数据库
	cm, err := s.repo.GetConversationMessagesGroup(ctx, senderID, groupID, lastMsgID, pageSize)
	if err != nil {
		return nil, err
	}

	msgs := make([]*dto.MessageDTO, len(cm.Messages))
	for i, m := range cm.Messages {
		msgs[i] = &dto.MessageDTO{
			ID:            m.Message.ID,
			Content:       m.Message.Content,
			Sender:        m.Message.SenderID,
			CreateTime:    m.Message.CreatedAt,
			GroupNickname: m.GroupNickname,
			UserInfo: &dto.UserInfoDTO{
				UserID:       m.User.UserID,
				SelfNickname: m.User.Nickname,
				Avatar:       m.User.Avatar,
			},
		}
	}

	dtoResult := &dto.ConversationMessagesDTO{
		ThreadID: cm.Thread.ID,
		Messages: msgs,
		HasMore:  cm.HasMore,
		Unread:   cm.Unread,
	}

	// 3. 如果是第一页，写入缓存
	if useCache {
		if bytes, err := json.Marshal(dtoResult); err == nil {
			if err := s.rdb.Set(ctx, cacheKey, bytes, 5*time.Minute).Err(); err != nil {
				s.logger.Warn("redis set failed", zap.Error(err))
			}
		}
	}

	return dtoResult, nil
}

func (s *MessageService) WithdrawMessageSingle(ctx context.Context, senderID int64, targetID int64, messageID int64) (int64, error) {
	// 参数校验
	if senderID <= 0 || targetID <= 0 || senderID == targetID {
		return -1, errors.New("invalid senderID or targetID")
	}

	lastMsgID, err := s.repo.WithdrawMessageSingle(ctx, senderID, targetID, messageID)
	if err != nil {
		return -1, fmt.Errorf("fail to withdraw this message:%d,error:%w", messageID, err)
	}
	return lastMsgID, nil
}

func (s *MessageService) UnWithdrawMessageSingle(ctx context.Context, senderID int64, targetID int64,
	messageID int64, newText string) (lastMessageID int64, err error) {
	// 参数校验
	if senderID <= 0 || targetID <= 0 || senderID == targetID {
		return -1, errors.New("invalid senderID or targetID")
	}
	if strings.TrimSpace(newText) == "" {
		return -1, errors.New("message text cannot be empty")
	}
	// 持久化
	lastMsgID, err := s.repo.UnWithdrawMessageSingle(ctx, senderID, targetID, messageID, newText)
	if err != nil {
		s.logger.Error("fail to persist message",
			zap.Int64("senderID", senderID),
			zap.Int64("targetID", targetID),
			zap.Int64("messageID", messageID),
			zap.String("newText", newText),
			zap.Error(err),
		)
	}
	//实时推送, 到redis
	channel := fmt.Sprintf("targetID:%d,messages", targetID)
	payload := fmt.Sprintf(`{"msgID":%d,"from":%d,"to":%d,"text":"%s"}`, messageID, senderID, targetID, newText)
	if err := s.rdb.Publish(ctx, channel, payload).Err(); err != nil {
		s.logger.Warn("failed to push message via redis",
			zap.String("channel", channel),
			zap.String("payload", payload),
			zap.Error(err),
		)
	}

	s.logger.Info("message sent successfully",
		zap.Int64("senderID", senderID),
		zap.Int64("targetID", targetID),
		zap.Int64("msgID", messageID),
	)

	return lastMsgID, nil
}

func (s *MessageService) WithdrawMessageGroup(ctx context.Context, senderID int64,
	groupID uuid.UUID, messageID int64) (int64, error) {
	if senderID <= 0 || groupID == uuid.Nil {
		return -1, errors.New("invalid senderID or groupID")
	}
	lastMsgID, err := s.repo.WithdrawMessageGroup(ctx, senderID, groupID, messageID)
	if err != nil {
		s.logger.Error("withdraw message error",
			zap.Int64("senderID", senderID),
			zap.String("groupID", groupID.String()),
			zap.Int64("messageID", messageID),
			zap.Error(err),
		)
	}
	s.logger.Info("message withdraw successfully",
		zap.Int64("senderID", senderID),
		zap.String("groupID", groupID.String()),
		zap.Int64("messageID", messageID),
	)
	return lastMsgID, nil
}

func (s *MessageService) UnWithdrawMessageGroup(ctx context.Context, senderID int64, groupID uuid.UUID,
	messageID int64, newText string) (lastMessageID int64, err error) {
	if senderID <= 0 || groupID == uuid.Nil {
		return -1, errors.New("invalid senderID or groupID")
	}
	if strings.TrimSpace(newText) == "" {
		return -1, errors.New("message text cannot be empty")
	}
	// 持久化
	lastMsgID, err := s.repo.UnWithdrawMessageGroup(ctx, senderID, groupID, messageID, newText)
	if err != nil {
		s.logger.Error("fail to persist message",
			zap.Int64("senderID", senderID),
			zap.String("groupID", groupID.String()),
			zap.Int64("messageID", messageID),
			zap.String("newText", newText),
			zap.Error(err),
		)
		return -1, fmt.Errorf("update unread failed: %w", err) // 返回给上层
	}
	//实时推送, 到redis
	channel := fmt.Sprintf("groupID:%s,messages", groupID.String())
	payload := fmt.Sprintf(`{"msgID":%d,"from":%d,"to":%s,"text":"%s"}`, messageID, senderID, groupID.String(), newText)
	if err := s.rdb.Publish(ctx, channel, payload).Err(); err != nil {
		s.logger.Warn("failed to push message via redis",
			zap.String("channel", channel),
			zap.String("payload", payload),
			zap.Error(err),
		)
	}

	s.logger.Info("message sent successfully",
		zap.Int64("senderID", senderID),
		zap.String("groupID", groupID.String()),
		zap.Int64("msgID", messageID),
	)
	return lastMsgID, nil
}

func (s *MessageService) UpdateUnread(ctx context.Context, userID, threadID int64) error {
	if userID <= 0 || threadID <= 0 {
		return errors.New("invalid userID or threadID")
	}
	if err := s.repo.UpdateUnread(ctx, userID, threadID); err != nil {
		s.logger.Error("update unread fail",
			zap.Int64("userID", userID),
			zap.Int64("threadID", threadID),
			zap.Error(err),
		)
		return fmt.Errorf("update unread failed: %w", err) // 返回给上层
	}
	s.logger.Info("update unread successfully",
		zap.Int64("userID", userID),
		zap.Int64("threadID", threadID),
	)
	return nil
}

func (s *MessageService) GetConversations(ctx context.Context, userID int64) ([]*dto.ConversationDTO, error) {
	if userID <= 0 {
		return nil, errors.New("invalid userID")
	}

	conversations, err := s.repo.GetConversations(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get conversations",
			zap.Int64("userID", userID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("fail to get conversations: %w", err)
	}

	if conversations == nil {
		conversations = []*repo.ConversationWithUser{}
	}

	c := make([]*dto.ConversationDTO, 0, len(conversations))
	for _, conv := range conversations {
		ty := "single"
		userInfo := &dto.UserInfo{}
		groupInfo := &dto.GroupInfo{}
		if conv.UserInfo == nil {
			ty = "group"
			groupInfo = &dto.GroupInfo{
				GroupID:   conv.GroupInfo.GroupID,
				GroupName: conv.GroupInfo.GroupName,
				Avatar:    conv.GroupInfo.Avatar,
			}
		} else {
			userInfo = &dto.UserInfo{
				UserID:   conv.UserInfo.UserID,
				Nickname: conv.UserInfo.Nickname,
				Avatar:   conv.UserInfo.Avatar,
			}
		}

		c = append(c, &dto.ConversationDTO{
			Type:     ty,
			ThreadID: conv.ThreadID,
			LastMessage: &dto.Message{
				ID:        conv.LastMessage.ID,
				SenderID:  conv.LastMessage.SenderID,
				Kind:      conv.LastMessage.Kind,
				Content:   conv.LastMessage.Content,
				CreatedAt: conv.LastMessage.CreatedAt,
			},
			UnreadCount: conv.UnreadCount,
			UserInfo:    userInfo,
			GroupInfo:   groupInfo,
			UpdateTime:  conv.UpdateTime,
		})
	}

	return c, nil
}
