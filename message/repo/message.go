package repo

import (
	"context"
	"errors"
	"message/repo/model"
	"time"

	"gorm.io/gorm"
	"github.com/AdventureDe/tempName/api/user"	
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
	GetConversations(ctx context.Context, userID int64) ([]*ConversationWithUser, error)
}

type messageRepo struct {
	db *gorm.DB
}

func NewMessageRepo(db *gorm.DB) MessageRepo {
	return &messageRepo{db: db}
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

func (r *messageRepo) UnWithdrawMessage(ctx context.Context, senderID int64, targetID int64,
	messageID int64, newtext string) (lastMessageID int64, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var thread model.Thread
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
		if err := tx.Model(&model.Message{}).
			Where("id = ?", messageID).
			Update("content", newtext).
			Update("is_withdrawed", true).Error; err != nil {
			return err
		}

		for _, userID := range []int64{senderID, targetID} {
			var lastMsg model.Message
			err := tx.Where("thread_id = ? AND is_withdrawed = ?", thread.ID, false).
				Order("is DESC").
				First(&lastMsg).Error
			if err != nil {
				return err
			}
			lastMessageID = lastMsg.ID
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

func (r *messageRepo) GetConversationMessages(
	ctx context.Context, senderID, targetID int64, lastMsgID int64, pageSize int) (*ConversationMessages, error) {
	// 1. 查找单聊 thread
	var thread model.Thread
	if err := r.db.WithContext(ctx).
		Where("(peer_a = ? AND peer_b = ?) OR (peer_a = ? AND peer_b = ?)",
			senderID, targetID, targetID, senderID).
		First(&thread).Error; err != nil {
		return nil, err
	}

	// 2. 查询分页消息
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

	// 3. 判断是否还有更多历史消息
	hasMore := false
	if len(messages) > pageSize {
		hasMore = true
		messages = messages[:pageSize]
	}

	// 4. 获取未读计数
	var conv model.Conversation
	unreadCount := 0
	if err := r.db.WithContext(ctx).
		Where("owner_id = ? AND thread_id = ?", senderID, thread.ID).
		First(&conv).Error; err == nil {
		unreadCount = conv.UnreadCount
	}

	// 5. 返回封装结果
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

func GetConversationsFromDB(ctx context.Context, userID int64) ([]*DBConversation, error) {

	return nil, nil
}

func (r *messageRepo) GetConversations(ctx context.Context, userID int64) ([]*ConversationWithUser, error) {
	// 1. 从数据库查会话
	conversations, err := GetConversationsFromDB(ctx, userID)
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
		u := userMap[conv.PeerID]
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
