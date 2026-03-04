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
	"github.com/IBM/sarama"
	"github.com/bwmarrin/snowflake" // 假设使用了此包作为 idGen
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// 定义传输到 Kafka 的消息结构 (DTO)
type AsyncMessage struct {
	MsgID     int64     `json:"msg_id"`
	SeqID     int64     `json:"seq_id"`
	SenderID  int64     `json:"sender_id"`
	TargetID  int64     `json:"target_id"`
	GroupID   uuid.UUID `json:"group_id"`
	Text      string    `json:"text"`
	Timestamp int64     `json:"timestamp"`
	Type      int       `json:"type"` // 1:单聊 2:群聊
}

type MessageService struct {
	repo          repo.MessageRepo
	rdb           *redis.Client
	logger        *zap.Logger
	kafkaProducer sarama.AsyncProducer // Kafka 生产者
	idGen         *snowflake.Node      // 分布式 ID 生成器
}

// 在初始化时注入 kafkaProducer 和 idGen
func NewMessageService(r repo.MessageRepo, u *redis.Client, l *zap.Logger, kp sarama.AsyncProducer, idGen *snowflake.Node) *MessageService {
	return &MessageService{
		repo:          r,
		rdb:           u,
		logger:        l,
		kafkaProducer: kp,
		idGen:         idGen,
	}
}

// SendMessageToSingle 发送单聊消息
func (s *MessageService) SendMessageToSingle(ctx context.Context, senderID, targetID int64, text string) (*int64, error) {
	// 1. 参数校验
	if senderID <= 0 || targetID <= 0 || senderID == targetID {
		return nil, errors.New("invalid senderID or targetID")
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("message text cannot be empty")
	}

	// 2. 【核心新增】利用 Redis 生成会话级的连续自增 ID (SeqID)
	// 使用 min 和 max 保证 A发给B 和 B发给A 共享同一个计数器
	redisSeqKey := fmt.Sprintf("linkim:seq:single:%d_%d", min(senderID, targetID), max(senderID, targetID))

	// 调用 Redis 的 INCR 命令，每次调用都会严格 +1
	seqID, err := s.rdb.Incr(ctx, redisSeqKey).Result()
	if err != nil {
		s.logger.Error("failed to generate seq_id from redis", zap.Error(err))
		return nil, err
	}

	// 3. 提前生成分布式 ID (Snowflake)，用于对外暴露
	msgID := s.idGen.Generate().Int64()

	// 4. 组装消息体
	msgPayload := AsyncMessage{
		MsgID:     msgID, // 乱序的安全 ID
		SeqID:     seqID, // 严格连续的内部序号 (1, 2, 3...)
		SenderID:  senderID,
		TargetID:  targetID,
		Text:      text,
		Timestamp: time.Now().UnixMilli(),
		Type:      1, // 单聊
	}

	// 5. 序列化
	val, err := json.Marshal(msgPayload)
	if err != nil {
		s.logger.Error("failed to marshal message", zap.Error(err))
		return nil, err
	}

	// 6. 发送给 Kafka
	partitionKey := fmt.Sprintf("%d_%d", min(senderID, targetID), max(senderID, targetID))

	msg := &sarama.ProducerMessage{
		Topic: "im_message_persistence_topic",
		Key:   sarama.StringEncoder(partitionKey),
		Value: sarama.ByteEncoder(val),
	}

	s.kafkaProducer.Input() <- msg

	s.logger.Info("message queued for persistence",
		zap.Int64("msgID", msgID),
		zap.Int64("seqID", seqID),
		zap.String("partitionKey", partitionKey),
	)

	// 7. 直接返回 Snowflake ID 给前端
	return &msgID, nil
}

// 辅助函数 (注：Go 1.21 以后自带 min/max，如果是旧版本保留这两个函数即可)
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// ConsumerHandler 实现 sarama.ConsumerGroupHandler 接口
type ConsumerHandler struct {
	repo   repo.MessageRepo // 修复：将 db *gorm.DB 改为 repo，方便调用持久化方法
	rdb    *redis.Client
	logger *zap.Logger
}

// NewConsumerHandler 初始化
func NewConsumerHandler(r repo.MessageRepo, rdb *redis.Client, logger *zap.Logger) *ConsumerHandler {
	return &ConsumerHandler{
		repo:   r,
		rdb:    rdb,
		logger: logger,
	}
}

func (h *ConsumerHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *ConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *ConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		// 1. 反序列化
		var payload AsyncMessage
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			h.logger.Error("failed to unmarshal message", zap.Error(err))
			session.MarkMessage(msg, "")
			continue
		}

		h.logger.Debug("received message from kafka", zap.Int64("msgID", payload.MsgID))

		// ================= 业务逻辑开始 =================

		// 2. 写入数据库 (Step 1)
		err := h.persistMessageToDB(session.Context(), &payload)
		if err != nil {
			h.logger.Error("failed to persist message DB", zap.Error(err), zap.Int64("msgID", payload.MsgID))
			// 如果数据库挂了，不 MarkMessage，让 Kafka 稍后重试
			continue
		}

		// 3. 推送 Redis Pub/Sub (Step 2)
		h.pushToRedisPubSub(session.Context(), &payload)

		// 4. 写入 Redis 缓存 (Step 3)
		h.saveToRedisCache(session.Context(), &payload)

		// ================= 业务逻辑结束 =================

		// 5. 标记消息已处理
		session.MarkMessage(msg, "")
	}
	return nil
}

// 修复：彻底重写此函数，解决作用域和参数错误
func (h *ConsumerHandler) persistMessageToDB(ctx context.Context, msg *AsyncMessage) error {
	// msg 已经是 *AsyncMessage 结构体，直接取值调用 repo
	_, err := h.repo.SendMessageToSingle(ctx, msg.MsgID, msg.SeqID, msg.SenderID, msg.TargetID, msg.Text)
	if err != nil {
		h.logger.Error("failed to persist message",
			zap.Int64("senderID", msg.SenderID),
			zap.Int64("targetID", msg.TargetID),
			zap.String("text", msg.Text),
			zap.Error(err),
		)
		return fmt.Errorf("persist message failed: %w", err)
	}
	return nil
}

func (h *ConsumerHandler) pushToRedisPubSub(ctx context.Context, msg *AsyncMessage) {
	channel := fmt.Sprintf("user:%d:messages", msg.TargetID)
	payload, _ := json.Marshal(msg)

	if err := h.rdb.Publish(ctx, channel, payload).Err(); err != nil {
		h.logger.Warn("failed to publish redis", zap.Error(err))
	}
}

func (h *ConsumerHandler) saveToRedisCache(ctx context.Context, msg *AsyncMessage) {
	key := fmt.Sprintf("user:%d:msg_cache", msg.TargetID)
	payload, _ := json.Marshal(msg)

	z := &redis.Z{
		Score:  float64(msg.MsgID),
		Member: string(payload), // 修复：存入 ZSet 时最好转为 string
	}

	pipe := h.rdb.Pipeline()
	pipe.ZAdd(ctx, key, z)
	pipe.ZRemRangeByRank(ctx, key, 0, -101)
	pipe.Expire(ctx, key, 7*24*time.Hour)

	if _, err := pipe.Exec(ctx); err != nil {
		h.logger.Warn("failed to update redis cache", zap.Error(err))
	}
}

// SendMessageToGroup 发送群聊消息 (异步改造版)
func (s *MessageService) SendMessageToGroup(ctx context.Context, senderID int64, groupID uuid.UUID, text string) (*int64, error) {
	// 1. 参数校验
	if senderID <= 0 || groupID == uuid.Nil {
		return nil, errors.New("invalid senderID or groupID")
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("text cannot be empty")
	}
	// 伪代码演示群聊的 Key 生成
	redisSeqKey := fmt.Sprintf("linkim:seq:group:%s", groupID.String())

	// 然后一样去 INCR
	seqID, err := s.rdb.Incr(ctx, redisSeqKey).Result()
	// 2. 【核心】在这里生成全局唯一的 MessageID！
	msgID := s.idGen.Generate().Int64()

	// 3. 组装发给 Kafka 的消息体
	msgPayload := AsyncMessage{
		MsgID:     msgID, // 带上刚生成的 ID
		SeqID:     seqID,
		SenderID:  senderID,
		TargetID:  0,       // 群聊没有单一 TargetID，可以用 0 或扩展结构体
		GroupID:   groupID, // 建议在 AsyncMessage 结构体里加一个 GroupID 字段
		Text:      text,
		Timestamp: time.Now().UnixMilli(),
		Type:      2, // 2: 群聊
	}

	// 4. 序列化
	val, err := json.Marshal(msgPayload)
	if err != nil {
		s.logger.Error("failed to marshal group message", zap.Error(err))
		return nil, err
	}

	// 5. 发送给 Kafka
	// 群聊的 PartitionKey 用群 ID，保证同一个群的消息按顺序处理
	partitionKey := groupID.String()

	msg := &sarama.ProducerMessage{
		Topic: "im_message_persistence_topic",
		Key:   sarama.StringEncoder(partitionKey),
		Value: sarama.ByteEncoder(val),
	}

	s.kafkaProducer.Input() <- msg

	s.logger.Info("group message queued for persistence",
		zap.Int64("msgID", msgID),
		zap.String("groupID", groupID.String()),
	)

	// 6. 核心：不等待数据库，直接把刚才生成的 ID 返回给 Handler！
	return &msgID, nil
}

func (s *MessageService) GetConversationMessagesSingle(ctx context.Context, senderID, targetID int64, lastMsgID int64, pageNum int, pageSize int) (*dto.ConversationMessagesDTO, error) {
	useCache := (pageNum == 1)

	var cacheKey string
	if useCache {
		cacheKey = fmt.Sprintf("conv:%d:tar:%d:page:1", senderID, targetID)
		if val, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var cached dto.ConversationMessagesDTO
			if jsonErr := json.Unmarshal([]byte(val), &cached); jsonErr == nil {
				return &cached, nil
			}
			_ = s.rdb.Del(ctx, cacheKey).Err()
		}
	}

	cm, err := s.repo.GetConversationMessagesSingle(ctx, senderID, targetID, lastMsgID, pageSize)
	if err != nil {
		return nil, err
	}

	msgs := make([]*dto.MessageDTO, len(cm.Messages))
	for i, m := range cm.Messages {
		msgs[i] = &dto.MessageDTO{
			ID:         m.Message.MsgID,
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

	if useCache {
		if bytes, err := json.Marshal(dtoResult); err == nil {
			s.rdb.Set(ctx, cacheKey, bytes, 5*time.Minute)
		}
	}

	return dtoResult, nil
}

func (s *MessageService) GetConversationMessagesGroup(ctx context.Context, senderID int64, groupID uuid.UUID, lastMsgID int64, pageNum int, pageSize int) (*dto.ConversationMessagesDTO, error) {
	useCache := (pageNum == 1)

	var cacheKey string
	if useCache {
		cacheKey = fmt.Sprintf("conv:%d:grp:%s:page:1", senderID, groupID.String())
		if val, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var cached dto.ConversationMessagesDTO
			if jsonErr := json.Unmarshal([]byte(val), &cached); jsonErr == nil {
				return &cached, nil
			}
			_ = s.rdb.Del(ctx, cacheKey).Err()
		}
	}

	cm, err := s.repo.GetConversationMessagesGroup(ctx, senderID, groupID, lastMsgID, pageSize)
	if err != nil {
		return nil, err
	}

	msgs := make([]*dto.MessageDTO, len(cm.Messages))
	for i, m := range cm.Messages {
		msgs[i] = &dto.MessageDTO{
			ID:            m.Message.MsgID,
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

	if useCache {
		if bytes, err := json.Marshal(dtoResult); err == nil {
			s.rdb.Set(ctx, cacheKey, bytes, 5*time.Minute)
		}
	}

	return dtoResult, nil
}

func (s *MessageService) WithdrawMessageSingle(ctx context.Context, senderID int64, targetID int64, messageID int64) (int64, error) {
	if senderID <= 0 || targetID <= 0 || senderID == targetID {
		return -1, errors.New("invalid senderID or targetID")
	}

	lastMsgID, err := s.repo.WithdrawMessageSingle(ctx, senderID, targetID, messageID)
	if err != nil {
		return -1, fmt.Errorf("fail to withdraw this message:%d,error:%w", messageID, err)
	}
	return lastMsgID, nil
}

func (s *MessageService) UnWithdrawMessageSingle(ctx context.Context, senderID int64, targetID int64, messageID int64, newText string) (lastMessageID int64, err error) {
	if senderID <= 0 || targetID <= 0 || senderID == targetID {
		return -1, errors.New("invalid senderID or targetID")
	}
	if strings.TrimSpace(newText) == "" {
		return -1, errors.New("message text cannot be empty")
	}
	lastMsgID, err := s.repo.UnWithdrawMessageSingle(ctx, senderID, targetID, messageID, newText)
	if err != nil {
		s.logger.Error("fail to persist message", zap.Error(err))
	}

	channel := fmt.Sprintf("targetID:%d,messages", targetID)
	payload := fmt.Sprintf(`{"msgID":%d,"from":%d,"to":%d,"text":"%s"}`, messageID, senderID, targetID, newText)
	if err := s.rdb.Publish(ctx, channel, payload).Err(); err != nil {
		s.logger.Warn("failed to push message via redis", zap.Error(err))
	}

	return lastMsgID, nil
}

func (s *MessageService) WithdrawMessageGroup(ctx context.Context, senderID int64, groupID uuid.UUID, messageID int64) (int64, error) {
	if senderID <= 0 || groupID == uuid.Nil {
		return -1, errors.New("invalid senderID or groupID")
	}
	lastMsgID, err := s.repo.WithdrawMessageGroup(ctx, senderID, groupID, messageID)
	if err != nil {
		s.logger.Error("withdraw message error", zap.Error(err))
	}
	return lastMsgID, nil
}

func (s *MessageService) UnWithdrawMessageGroup(ctx context.Context, senderID int64, groupID uuid.UUID, messageID int64, newText string) (lastMessageID int64, err error) {
	if senderID <= 0 || groupID == uuid.Nil {
		return -1, errors.New("invalid senderID or groupID")
	}
	if strings.TrimSpace(newText) == "" {
		return -1, errors.New("message text cannot be empty")
	}

	lastMsgID, err := s.repo.UnWithdrawMessageGroup(ctx, senderID, groupID, messageID, newText)
	if err != nil {
		s.logger.Error("fail to persist message", zap.Error(err))
		return -1, fmt.Errorf("unwithdraw message failed: %w", err) // 修复：改掉了文案 "update unread failed"
	}

	channel := fmt.Sprintf("groupID:%s,messages", groupID.String())
	payload := fmt.Sprintf(`{"msgID":%d,"from":%d,"to":"%s","text":"%s"}`, messageID, senderID, groupID.String(), newText) // 修复：to 加上双引号以防 JSON 崩溃
	if err := s.rdb.Publish(ctx, channel, payload).Err(); err != nil {
		s.logger.Warn("failed to push message via redis", zap.Error(err))
	}

	return lastMsgID, nil
}

func (s *MessageService) UpdateUnread(ctx context.Context, userID, threadID int64) error {
	if userID <= 0 || threadID <= 0 {
		return errors.New("invalid userID or threadID")
	}
	if err := s.repo.UpdateUnread(ctx, userID, threadID); err != nil {
		s.logger.Error("update unread fail", zap.Error(err))
		return fmt.Errorf("update unread failed: %w", err)
	}
	return nil
}

func (s *MessageService) GetConversations(ctx context.Context, userID int64) ([]*dto.ConversationDTO, error) {
	if userID <= 0 {
		return nil, errors.New("invalid userID")
	}

	conversations, err := s.repo.GetConversations(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get conversations", zap.Error(err))
		return nil, fmt.Errorf("fail to get conversations: %w", err)
	}

	if conversations == nil {
		return []*dto.ConversationDTO{}, nil // 优化：直接返回空切片
	}

	c := make([]*dto.ConversationDTO, 0, len(conversations))
	for _, conv := range conversations {
		ty := "single"
		userInfo := &dto.UserInfo{}
		groupInfo := &dto.GroupInfo{}
		if conv.UserInfo == nil {
			ty = "group"
			groupInfo = &dto.GroupInfo{
				GroupID:   &conv.GroupInfo.GroupID,
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
				ID:        conv.LastMessage.MsgID,
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
