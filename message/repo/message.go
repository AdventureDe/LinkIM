package repo

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/AdventureDe/LinkIM/message/repo/model"

	userpb "github.com/AdventureDe/LinkIM/api/user"
	"gorm.io/gorm"
)

// 会话消息返回结构
type ConversationMessages struct {
	Thread   *model.Thread    `json:"thread"`   // 会话信息（顶层返回一次即可）
	Messages []*model.Message `json:"messages"` // 消息列表
	HasMore  bool             `json:"has_more"`
	Unread   int              `json:"unread"`
}

// 从 DB 查出来的会话信息
type DBConversation struct {
	ThreadID    int64
	PeerID      int64
	LastMessage *model.Message
	UnreadCount int
}

// 最终返回给前端的结构（包含用户信息）
type ConversationWithUser struct {
	ThreadID    int64          `json:"thread_id"`
	LastMessage *model.Message `json:"last_message"`
	UnreadCount int            `json:"unread_count"`
	UserInfo    *UserInfo      `json:"user_info"`
}

type UserInfo struct {
	UserID   int64  `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type MessageRepo interface {
	SendMessageToSingle(ctx context.Context, senderid int64, targetid int64, text string) (*int64, error)                                //单聊
	GetConversationMessages(ctx context.Context, senderid, targetid int64, lastMsgid int64, pageSize int) (*ConversationMessages, error) //获取对话的消息
	WithdrawMessage(ctx context.Context, senderid int64, targetid int64, messageid int64) (int64, error)
	UnWithdrawMessage(ctx context.Context, senderID int64, targetID int64,
		messageID int64, newtext string) (lastMessageID int64, err error)
	UpdateUnread(ctx context.Context, userID, threadID int64) error
	GetConversationsFromDB(ctx context.Context, userID int64) ([]*DBConversation, error)
	GetConversations(ctx context.Context, userID int64) ([]*ConversationWithUser, error)
}

type messageRepo struct {
	db         *gorm.DB
	userClient userpb.UserServiceClient
}

func NewMessageRepo(db *gorm.DB, m *messageService) MessageRepo {
	return &messageRepo{
		db:         db,
		userClient: m.userClient,
	}
}

func (r *messageRepo) SendMessageToSingle(ctx context.Context, senderID int64, targetID int64, text string) (lastMsgId *int64, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 查找或创建 thread (单聊)
		var thread model.Thread
		if err := tx.Model(model.Thread{}).Where(
			"(peer_a = ? AND peer_b = ?) OR (peer_a = ? AND peer_b = ?)",
			senderID, targetID, targetID, senderID,
		).First(&thread).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 新建 thread
				thread = model.Thread{
					Type:  1,
					PeerA: &senderID,
					PeerB: &targetID,
				}
				if err := tx.Create(&thread).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// 2. 插入消息
		msg := model.Message{
			ThreadID: thread.ID,
			SenderID: senderID,
			Kind:     1, // text
			Content:  text,
		}
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}

		// 3. 更新 Conversation
		// (a) 发送者
		if err = upsertConversation(tx, senderID, thread.ID, msg.ID, 0); err != nil {
			return err
		}
		// (b) 接收者（未读 +1）
		if err = upsertConversation(tx, targetID, thread.ID, msg.ID, 1); err != nil {
			return err
		}

		// 4. 写 message_status（接收者未读）
		status := model.MessageStatus{
			MessageID: msg.ID,
			UserID:    targetID,
			Status:    0, // 未读
		}
		if err := tx.Create(&status).Error; err != nil {
			return err
		}
		lastMsgId, err = GetLastMessageID(tx, ctx)
		return nil
	})
	return
}

// 更新/插入 conversation
func upsertConversation(tx *gorm.DB, ownerID, threadID, lastMsgID int64, unreadDelta int) error {
	var conv model.Conversation
	err := tx.Where("owner_id = ? AND thread_id = ?", ownerID, threadID).First(&conv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { // 不存在才会创建,创建时lastMsgID=1
		conv = model.Conversation{
			OwnerID:       ownerID,
			ThreadID:      threadID,
			LastMessageID: &lastMsgID,
			UnreadCount:   unreadDelta,
		}
		return tx.Create(&conv).Error
	} else if err != nil {
		return err
	}

	// 已存在则更新
	conv.LastMessageID = &lastMsgID
	if unreadDelta > 0 {
		conv.UnreadCount += unreadDelta
	}
	return tx.Save(&conv).Error
}

func GetLastMessageID(tx *gorm.DB, ctx context.Context) (lastMessageID *int64, err error) {
	err = tx.WithContext(ctx).Model(&model.Message{}).Select("MAX(id)").Scan(&lastMessageID).Error
	return
}

// 删除两边的消息
func (r *messageRepo) WithdrawMessage(ctx context.Context, senderID int64, targetID int64, messageID int64) (lastMessageID int64, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var thread model.Thread
		// 找到对应的单聊线程
		if err := tx.Where("(peer_a = ? AND peer_b = ?) OR (peer_a = ? AND peer_b = ?)",
			senderID, targetID, targetID, senderID).
			First(&thread).Error; err != nil {
			return err
		}

		// 查找要撤回的消息
		var message model.Message
		// 保证安全性
		if err := tx.Where("thread_id = ? AND sender_id = ? AND id = ?", thread.ID, senderID, messageID).
			First(&message).Error; err != nil {
			return err
		}

		// 时间限制 查验
		if time.Since(message.CreatedAt) > 3*time.Minute {
			return errors.New("超过撤回时间限制")
		}

		// 逻辑撤回：更新 Message 表 IsWithdrawed
		if err := tx.Model(&model.Message{}).
			Where("id = ?", messageID).
			Update("is_withdrawed", true).Error; err != nil {
			return err
		}

		// 更新 Conversation.last_message_id
		for _, userID := range []int64{senderID, targetID} {
			var lastMsg model.Message
			err := tx.Where("thread_id = ? AND is_withdrawed = ?", thread.ID, false).
				Order("id DESC").
				First(&lastMsg).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			lastMessageID = lastMsg.ID // 保存最后一条消息 ID

			// 更新每个用户的会话
			err = tx.Model(&model.Conversation{}).
				Where("owner_id = ? AND thread_id = ?", userID, thread.ID).
				Update("last_message_id", lastMsg.ID).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	return
}

func (r *messageRepo) UnWithdrawMessage(ctx context.Context, senderID, targetID, messageID int64, newtext string) (lastMessageID int64, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var thread model.Thread
		if err := tx.Where("(peer_a = ? AND peer_b = ?) OR (peer_a = ? AND peer_b = ?)",
			senderID, targetID, targetID, senderID).First(&thread).Error; err != nil {
			return err
		}

		var message model.Message
		if err := tx.Where("thread_id = ? AND sender_id = ? AND id = ?", thread.ID, senderID, messageID).
			First(&message).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.Message{}).Where("id = ?", messageID).
			Updates(map[string]interface{}{ //批量更新消息一次完成
				"content":       newtext,
				"is_withdrawed": false,
			}).Error; err != nil {
			return err
		}

		var lastMsg model.Message
		err := tx.Where("thread_id = ? AND is_withdrawed = ?", thread.ID, false).
			Order("id DESC").First(&lastMsg).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// 处理了没有未撤回消息的情况
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			lastMessageID = lastMsg.ID
		}

		for _, userID := range []int64{senderID, targetID} {
			if err := tx.Model(&model.Conversation{}).
				Where("owner_id = ? AND thread_id = ?", userID, thread.ID).
				Update("last_message_id", lastMessageID).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return
}

func (r *messageRepo) GetConversationMessages(
	ctx context.Context, senderID, targetID int64, lastMsgID int64, pageSize int) (*ConversationMessages, error) {
	// 查找单聊 thread
	var thread model.Thread
	if err := r.db.WithContext(ctx).
		Where("(peer_a = ? AND peer_b = ?) OR (peer_a = ? AND peer_b = ?)",
			senderID, targetID, targetID, senderID).
		First(&thread).Error; err != nil {
		return nil, err
	}

	// 查询分页消息
	var messages []*model.Message
	db := r.db.WithContext(ctx).
		Where("thread_id = ? AND is_withdrawed = ?", thread.ID, false).
		Order("id DESC").
		Limit(pageSize + 1) // 多拉一条用于判断 HasMore

	if lastMsgID > 0 {
		db = db.Where("id < ?", lastMsgID+1) // 游标分页
	}

	if err := db.Find(&messages).Error; err != nil {
		return nil, err
	}

	// 判断是否还有更多历史消息
	hasMore := false
	if len(messages) > pageSize {
		hasMore = true
		messages = messages[:pageSize]
	}

	// 获取未读计数
	var conv model.Conversation
	unreadCount := 0
	if err := r.db.WithContext(ctx).
		Where("owner_id = ? AND thread_id = ?", senderID, thread.ID).
		First(&conv).Error; err == nil {
		unreadCount = conv.UnreadCount
	}

	// 返回封装结果
	return &ConversationMessages{
		Thread:   &thread,
		Messages: messages,
		HasMore:  hasMore,
		Unread:   unreadCount,
	}, nil
}

// 当前哪个用户在读，在哪个thread读，同时获取最后已读messageid
func (r *messageRepo) UpdateUnread(ctx context.Context, userID, threadID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新 MessageStatus 已读状态
		query := tx.Model(&model.MessageStatus{}).
			Joins("JOIN messages ON message_statuses.message_id = messages.id").
			Where("message_statuses.user_id = ? AND messages.thread_id = ?", userID, threadID)

		if err := query.Update("status", 1).Error; err != nil {
			return err
		}

		// 更新 Conversation.unread_count
		if err := tx.Model(&model.Conversation{}).
			Where("owner_id = ? AND thread_id = ?", userID, threadID).
			Update("unread_count", 0).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *messageRepo) GetConversationsFromDB(ctx context.Context, userID int64) ([]*DBConversation, error) {
	var convs []*DBConversation

	// 1. 查询 Conversation + Thread
	var conversations []model.Conversation
	if err := r.db.WithContext(ctx).
		Preload("Thread").
		Where("owner_id = ? AND is_deleted = false", userID).
		Order("updated_at DESC").
		Find(&conversations).Error; err != nil {
		return nil, err
	}

	// 2. 收集 LastMessageID
	var msgIDs []int64
	for _, c := range conversations {
		if c.LastMessageID != nil {
			msgIDs = append(msgIDs, *c.LastMessageID)
		}
	}

	// 3. 批量查询消息
	msgMap := make(map[int64]*model.Message)
	if len(msgIDs) > 0 {
		var msgs []model.Message
		if err := r.db.WithContext(ctx).Where("id IN ?", msgIDs).Find(&msgs).Error; err != nil {
			return nil, err
		}
		for _, m := range msgs {
			msgMap[m.ID] = &m
		}
	}

	// 4. 拼接 DBConversation
	for _, c := range conversations {
		peerID := int64(0)
		switch c.Thread.Type {
		case 1: // 单聊
			if c.Thread.PeerA != nil && *c.Thread.PeerA != userID {
				peerID = *c.Thread.PeerA
			} else if c.Thread.PeerB != nil && *c.Thread.PeerB != userID {
				peerID = *c.Thread.PeerB
			}
		case 2: // 群聊
			if c.Thread.GroupID != nil {
				peerID = *c.Thread.GroupID
			}
		}

		var lastMsg *model.Message
		if c.LastMessageID != nil {
			lastMsg = msgMap[*c.LastMessageID]
		}

		convs = append(convs, &DBConversation{
			ThreadID:    c.ThreadID,
			PeerID:      peerID,
			LastMessage: lastMsg,
			UnreadCount: c.UnreadCount,
		})
	}

	return convs, nil
}

func (r *messageRepo) GetConversations(ctx context.Context, userID int64) ([]*ConversationWithUser, error) {
	// 1. 从数据库查会话
	conversations, err := r.GetConversationsFromDB(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 2. 收集 PeerID
	var peerIDs []int64
	for _, conv := range conversations {
		peerIDs = append(peerIDs, conv.PeerID)
	}

	// 3. 调用 UserService 批量获取用户信息
	resp, err := r.userClient.GetUserInfos(ctx, &userpb.GetUserInfosRequest{
		UserIds: peerIDs,
	})
	if err != nil {
		return nil, err
	}

	userMap := make(map[int64]*userpb.UserInfo)
	for _, u := range resp.Users {
		userMap[u.UserId] = u
	}

	// 4. 拼接返回
	var result []*ConversationWithUser
	for _, conv := range conversations {
		u, ok := userMap[conv.PeerID]
		if !ok || u == nil {
			log.Printf("user info not found for peerID=%d", conv.PeerID)
			continue
		}
		result = append(result, &ConversationWithUser{
			ThreadID:    conv.ThreadID,
			LastMessage: conv.LastMessage,
			UnreadCount: conv.UnreadCount,
			UserInfo: &UserInfo{
				UserID:   u.UserId,
				Nickname: u.Nickname,
				Avatar:   u.Avatar,
			},
		})
	}

	return result, nil
}
