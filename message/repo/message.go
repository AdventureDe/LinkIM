package repo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/AdventureDe/LinkIM/message/repo/model"
	"github.com/google/uuid"
	"go.uber.org/zap"

	grouppb "github.com/AdventureDe/LinkIM/api/group"
	userpb "github.com/AdventureDe/LinkIM/api/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 会话消息返回结构
type ConversationMessages struct {
	Thread   *model.Thread      `json:"thread"`   // 会话信息（顶层返回一次即可）
	Messages []*MessageWithUser `json:"messages"` // 消息列表
	HasMore  bool               `json:"has_more"`
	Unread   int                `json:"unread"`
}

type ConversationGroupMessages struct {
	Thread   *model.Thread      `json:"thread"`   // 会话信息（顶层返回一次即可）
	Messages []*MessageWithUser `json:"messages"` // 消息列表
	HasMore  bool               `json:"has_more"`
	Unread   int                `json:"unread"`
}

type MessageWithUser struct {
	Message       model.Message
	User          UserInfo
	GroupNickname string `json:"group_nickname"`
}

// 单聊会话信息
type SingleConversation struct {
	ThreadID    int64
	PeerID      int64
	LastMessage *model.Message
	UnreadCount int
	UpdateTime  time.Time
}

// 群聊会话信息
type GroupConversation struct {
	ThreadID    int64
	GroupID     uuid.UUID
	LastMessage *model.Message
	UnreadCount int
	UpdateTime  time.Time
}

// 最终返回给前端的结构（包含用户信息）
type ConversationWithUser struct {
	ThreadID    int64          `json:"thread_id"`
	LastMessage *model.Message `json:"last_message"`
	UnreadCount int            `json:"unread_count"`
	UserInfo    *UserInfo      `json:"user_info"`
	GroupInfo   *GroupInfo     `json:"group_info"`
	UpdateTime  time.Time      `json:"update_time"`
}

type UserInfo struct {
	UserID   int64  `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type GroupInfo struct {
	GroupID   uuid.UUID `json:"group_id"`
	GroupName string    `json:"group_name"`
	Avatar    string    `json:"avatar"`
}

type MessageRepo interface {
	SendMessageToSingle(ctx context.Context, message_id, seq_id, senderid, targetid int64,
		text string) (lastMsgId *int64, err error)
	SendMessageToGroup(ctx context.Context, message_id, seq_id, senderID int64, groupID uuid.UUID,
		text string) (lastMsgId *int64, err error)
	GetConversationMessagesSingle(ctx context.Context, senderID, targetID int64,
		lastMsgID int64, pageSize int) (*ConversationMessages, error)
	GetConversationMessagesGroup(ctx context.Context, senderID int64, groupID uuid.UUID,
		lastMsgID int64, pageSize int) (*ConversationGroupMessages, error)
	WithdrawMessageSingle(ctx context.Context, senderID int64, targetID int64, messageID int64) (int64, error)
	UnWithdrawMessageSingle(ctx context.Context, senderID int64, targetID int64,
		messageID int64, newText string) (lastMessageID int64, err error)
	WithdrawMessageGroup(ctx context.Context, senderID int64, groupID uuid.UUID, messageID int64) (int64, error)
	UnWithdrawMessageGroup(ctx context.Context, senderID int64, groupID uuid.UUID, messageID int64,
		newText string) (lastMessageID int64, err error)
	UpdateUnread(ctx context.Context, userID, threadID int64) error
	GetSingleConversationsFromDB(ctx context.Context, userID int64) ([]*SingleConversation, error) //辅助函数
	GetGroupConversationsFromDB(ctx context.Context, userID int64) ([]*GroupConversation, error)   //辅助函数
	GetConversations(ctx context.Context, userID int64) ([]*ConversationWithUser, error)
}

type messageRepo struct {
	db          *gorm.DB
	userClient  userpb.UserServiceClient
	groupClient grouppb.GroupServiceClient
}

func NewMessageRepo(db *gorm.DB, m *messageService) MessageRepo {
	return &messageRepo{
		db:          db,
		userClient:  m.userClient,
		groupClient: m.groupClient,
	}
}

func (r *messageRepo) SendMessageToSingle(ctx context.Context, message_id, seq_id, senderID, targetID int64, text string) (lastMsgId *int64, err error) {
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
			MsgID:    message_id,
			SeqID:    seq_id,
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
		if err = upsertConversation(tx, senderID, thread.ID, msg.MsgID, 0); err != nil {
			return err
		}
		// (b) 接收者（未读 +1）
		if err = upsertConversation(tx, targetID, thread.ID, msg.MsgID, 1); err != nil {
			return err
		}

		// 4. 写 message_status（接收者未读）
		status := model.MessageStatus{
			MessageID: msg.MsgID,
			UserID:    targetID,
			Status:    0, // 未读
		}
		if err := tx.Create(&status).Error; err != nil {
			return err
		}
		var temp int64
		temp, err = GetLastMessageID(tx, ctx)
		if err != nil {
			return err
		}
		lastMsgId = &temp
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

func upsertConversationsBatch(tx *gorm.DB, ownerIDs []int64, threadID, lastMsgID int64, unreadDelta int) error {
	var convs []model.Conversation //转换为conversation数组
	for _, ownerID := range ownerIDs {
		convs = append(convs, model.Conversation{
			OwnerID:       ownerID,
			ThreadID:      threadID,
			LastMessageID: &lastMsgID,
			UnreadCount:   unreadDelta,
		})
	}
	//一次性更新
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "owner_id"}, {Name: "thread_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"last_message_id": lastMsgID,
			"unread_count":    gorm.Expr("conversations.unread_count + ?", unreadDelta),
		}),
	}).Create(&convs).Error
}

func GetLastMessageID(tx *gorm.DB, ctx context.Context) (lastMessageID int64, err error) {
	err = tx.WithContext(ctx).Model(&model.Message{}).Select("MAX(id)").Scan(&lastMessageID).Error
	return
}

// SendMessageToGroup 负责发送群消息
// 设计原则：
// 1. 远程调用放事务外，避免长事务
// 2. 数据写入保证强一致性
// 3. 批量写入提高性能
func (r *messageRepo) SendMessageToGroup(
	ctx context.Context,
	messageID,
	seq_id,
	senderID int64,
	groupID uuid.UUID,
	text string,
) (*int64, error) {

	// ====== 第一阶段：事务外调用远程服务 ======
	// 获取群成员（避免在事务内调用 gRPC）
	res, err := r.groupClient.ListGroupMembers(ctx, &grouppb.ListGroupMembersRequest{
		GroupId: groupID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("list group members failed: %w", err)
	}

	// 过滤发送者本人，生成需要更新未读的用户列表
	memberIDs := make([]int64, 0, len(res.Members))
	for _, m := range res.Members {
		if m.UserId != senderID {
			memberIDs = append(memberIDs, m.UserId)
		}
	}

	var lastMsgID int64

	// ====== 第二阶段：事务内保证强一致性 ======
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// 1️⃣ 获取或创建群会话线程
		thread, err := getOrCreateGroupThread(tx, groupID)
		if err != nil {
			return err
		}

		// 2️⃣ 创建消息记录
		msg := model.Message{
			MsgID:    messageID,
			SeqID:    seq_id,
			ThreadID: thread.ID,
			SenderID: senderID,
			Kind:     1, // 1 = 文本消息
			Content:  text,
		}
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}

		// 3️⃣ 更新发送者会话（未读数=0）
		if err := upsertConversation(tx, senderID, thread.ID, msg.MsgID, 0); err != nil {
			return err
		}

		// 4️⃣ 批量更新群成员会话（未读数+1）
		if len(memberIDs) > 0 {
			if err := upsertConversationsBatch(tx, memberIDs, thread.ID, msg.MsgID, 1); err != nil {
				return err
			}
		}

		// 5️⃣ 批量创建未读消息状态
		if len(memberIDs) > 0 {
			statuses := make([]model.MessageStatus, 0, len(memberIDs))
			for _, uid := range memberIDs {
				statuses = append(statuses, model.MessageStatus{
					MessageID: msg.MsgID,
					UserID:    uid,
					Status:    0, // 0 = 未读
				})
			}

			if err := tx.Create(&statuses).Error; err != nil {
				return err
			}
		}

		// 6️⃣ 直接使用当前消息ID作为返回值
		lastMsgID = msg.MsgID

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &lastMsgID, nil
}

// 获取或创建 Thread
func getOrCreateGroupThread(tx *gorm.DB, groupID uuid.UUID) (*model.Thread, error) {
	var thread model.Thread
	err := tx.Where("group_id = ?", groupID).First(&thread).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		thread = model.Thread{
			Type:    2,
			GroupID: &groupID,
		}
		if err := tx.Create(&thread).Error; err != nil {
			return nil, err
		}
		return &thread, nil
	} else if err != nil {
		return nil, err
	}
	return &thread, nil
}

// 删除两边的消息 单聊 撤回
func (r *messageRepo) WithdrawMessageSingle(ctx context.Context, senderID int64, targetID int64, messageID int64) (lastMessageID int64, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var thread model.Thread
		// 1. 找到对应的单聊线程
		if err := tx.Where("(peer_a = ? AND peer_b = ?) OR (peer_a = ? AND peer_b = ?)",
			senderID, targetID, targetID, senderID).
			First(&thread).Error; err != nil {
			return err
		}

		// 2. 查找要撤回的消息实体 (注意：主键字段名是 msg_id)
		var message model.Message
		if err := tx.Where("thread_id = ? AND sender_id = ? AND id = ?", thread.ID, senderID, messageID).
			First(&message).Error; err != nil {
			return err
		}

		// 3. 时间限制查验
		if time.Since(message.CreatedAt) > 3*time.Minute {
			return errors.New("超过撤回时间限制")
		}

		// 4. 逻辑撤回：更新 Message 表
		if err := tx.Model(&model.Message{}).
			Where("msg_id = ?", messageID). // 修正字段名
			Update("is_withdrawed", true).Error; err != nil {
			return err
		}

		// 5. 寻找撤回后的“最新一条有效消息”
		var lastMsg model.Message
		err := tx.Where("thread_id = ? AND is_withdrawed = ?", thread.ID, false).
			Order("seq_id DESC"). // 【核心精髓】利用严格连续的 SeqID 倒序，找出来的绝对是真实的最后一条！
			First(&lastMsg).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 如果整个会话的消息都被撤回了，会话可能就没有 last message 了
				// 根据你的业务逻辑，这里可以将 Conversation.last_message_id 置为 NULL 或 0
				lastMessageID = 0
			} else {
				return err
			}
		} else {
			// 保存最后一条消息的雪花 ID
			lastMessageID = lastMsg.MsgID
		}

		// 6. 批量更新会话的 LastMessageID
		// 修正：切片对应的是 IN 查询
		if lastMessageID != 0 {
			err = tx.Model(&model.Conversation{}).
				Where("owner_id IN ? AND thread_id = ?", []int64{senderID, targetID}, thread.ID).
				Update("last_message_id", lastMessageID).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
	return
}
func (r *messageRepo) UnWithdrawMessageSingle(ctx context.Context, senderID, targetID, messageID int64, newtext string) (lastMessageID int64, err error) {
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
			Order("seq_id DESC").First(&lastMsg).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// 处理了没有未撤回消息的情况
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			lastMessageID = lastMsg.MsgID
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

// 撤回 群聊
func (r *messageRepo) WithdrawMessageGroup(ctx context.Context, senderID int64, groupID uuid.UUID,
	messageID int64) (lastMessageID int64, err error) {
	// 事务外先查群成员，避免长事务阻塞
	res, err := r.groupClient.ListGroupMembers(ctx, &grouppb.ListGroupMembersRequest{
		GroupId: groupID.String(),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list group members: %w", err)
	}
	var mem []int64 // 所有的成员
	for _, m := range res.Members {
		mem = append(mem, m.UserId)
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 找 thread
		var thread model.Thread
		if err := tx.Where("group_id = ?", groupID).First(&thread).Error; err != nil {
			return err
		}

		// 查找要撤回的消息
		var message model.Message
		if err := tx.Where("thread_id = ? AND sender_id = ? AND id = ?",
			thread.ID, senderID, messageID).First(&message).Error; err != nil {
			return err
		}

		// 时间限制
		if time.Since(message.CreatedAt) > 3*time.Minute {
			return errors.New("超过撤回时间限制")
		}

		// 逻辑撤回
		if err := tx.Model(&model.Message{}).
			Where("id = ?", messageID).
			Update("is_withdrawed", true).Error; err != nil {
			return err
		}

		// 找 thread 下的最后一条未撤回消息
		var lastMsg model.Message
		err := tx.Where("thread_id = ? AND is_withdrawed = ?", thread.ID, false).
			Order("id DESC").First(&lastMsg).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		lastMessageID = lastMsg.MsgID

		// 一次性更新所有成员的会话
		if len(mem) > 0 {
			err = tx.Model(&model.Conversation{}).
				Where("owner_id IN ? AND thread_id = ?", mem, thread.ID).
				Update("last_message_id", lastMsg.MsgID).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	return
}

func (r *messageRepo) UnWithdrawMessageGroup(ctx context.Context, senderID int64, groupID uuid.UUID, messageID int64,
	newText string) (lastMessageID int64, err error) {
	// 事务外先查群成员，避免长事务阻塞
	res, err := r.groupClient.ListGroupMembers(ctx, &grouppb.ListGroupMembersRequest{
		GroupId: groupID.String(),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list group members: %w", err)
	}
	mem := make([]int64, 0, len(res.Members)) // 所有的成员
	for _, m := range res.Members {
		mem = append(mem, m.UserId)
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var thread model.Thread
		if err := tx.Where("group_id = ?", groupID).
			First(&thread).Error; err != nil {
			return err
		}
		//检查是否存在改条记录
		var dummy int
		if err := tx.Model(&model.Message{}).
			Where("id = ? AND thread_id = ? AND sender_id = ?", messageID, thread.ID, senderID).
			Select("1").First(&dummy).Error; err != nil {
			return err
		}

		// 批量更新  使用Updates
		if err := tx.Model(&model.Message{}).Where("id = ?", messageID).
			Updates(map[string]interface{}{
				"content":       newText,
				"is_withdrawed": false,
			}).Error; err != nil {
			return err
		}

		var lastMsg model.Message
		if err := tx.Where("thread_id = ? AND is_withdrawed = ?", thread.ID, false).
			Order("id DESC").First(&lastMsg).Error; err == nil {
			lastMessageID = lastMsg.MsgID
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// 处理所有会话的更新   批量更新： UpdateColumn
		if err := tx.Model(&model.Conversation{}).
			Where("owner_id IN ? AND thread_id = ?", mem, thread.ID).
			UpdateColumn("last_message_id", lastMessageID).Error; err != nil {
			return err
		}
		return nil
	})
	return
}

func (r *messageRepo) GetConversationMessagesSingle(
	ctx context.Context,
	senderID, targetID int64,
	lastMsgID int64,
	pageSize int,
) (*ConversationMessages, error) {
	db := r.db.WithContext(ctx)

	// 1. 查找单聊 thread
	var thread model.Thread
	if err := db.Where(
		"(peer_a = ? AND peer_b = ?) OR (peer_a = ? AND peer_b = ?)",
		senderID, targetID, targetID, senderID,
	).First(&thread).Error; err != nil {
		return nil, err
	}

	// 2. 游标转换：把前端传来的雪花 ID (lastMsgID) 翻译成内部的连续序号 (SeqID)
	var cursorSeqID int64
	if lastMsgID > 0 {
		// 去数据库里查这条消息对应的 seq_id
		if err := db.Model(&model.Message{}).
			Where("thread_id = ? AND id = ?", thread.ID, lastMsgID).
			Pluck("seq_id", &cursorSeqID).Error; err != nil {
			return nil, fmt.Errorf("invalid cursor message id: %w", err)
		}
	}

	// 3. 查询消息（使用 SeqID 进行严格时序分页）
	messages := make([]*model.Message, 0, pageSize+1)
	query := db.Model(&model.Message{}).
		Where("thread_id = ? AND is_withdrawed = ?", thread.ID, false).
		Order("seq_id DESC"). // 【修改】必须用 seq_id 倒序，保证时序绝对正确
		Limit(pageSize + 1)

	if cursorSeqID > 0 {
		// 【修改】使用 seq_id < ?
		// 注意：这里用 `<` 而不是原来的 `<=`，因为正常的下拉刷新不应该把刚才那条基准消息再查出来一遍
		query = query.Where("seq_id < ?", cursorSeqID)
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}

	hasMore := false
	if len(messages) > pageSize {
		hasMore = true
		messages = messages[:pageSize] // 丢弃最后一条，保持 pageSize
	}

	// 4. 获取未读计数
	var unreadCount int
	_ = db.Model(&model.Conversation{}).
		Where("owner_id = ? AND thread_id = ?", senderID, thread.ID).
		Pluck("unread_count", &unreadCount).Error

	// 5. 调用 user-service 获取用户信息
	userIDs := []int64{senderID, targetID}
	userResp, err := r.userClient.GetUserInfos(ctx, &userpb.GetUserInfosRequest{
		UserIds: userIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("fail to get UserInfos: %w", err)
	}

	// 6. 建立 userMap
	userMap := make(map[int64]UserInfo, len(userResp.Users))
	for _, u := range userResp.Users {
		userMap[u.UserId] = UserInfo{
			UserID:   u.UserId,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
		}
	}

	// 7. 组装返回数据
	messageWithUserInfos := make([]*MessageWithUser, 0, len(messages))
	for _, m := range messages {
		mwu := &MessageWithUser{
			Message: *m,
			User:    userMap[m.SenderID], // 直接 O(1) 查
		}
		messageWithUserInfos = append(messageWithUserInfos, mwu)
	}

	return &ConversationMessages{
		Thread:   &thread,
		Messages: messageWithUserInfos,
		HasMore:  hasMore,
		Unread:   unreadCount,
	}, nil
}

func (r *messageRepo) GetConversationMessagesGroup(
	ctx context.Context,
	senderID int64,
	groupID uuid.UUID,
	lastMsgID int64,
	pageSize int,
) (*ConversationGroupMessages, error) {
	db := r.db.WithContext(ctx)

	// 1. 获取群聊 thread
	var thread model.Thread
	if err := db.Where("group_id = ?", groupID).First(&thread).Error; err != nil {
		return nil, err
	}

	// 2. 游标转换：将外部传递的雪花 MsgID 转换为内部严格连续的 SeqID
	var cursorSeqID int64
	if lastMsgID > 0 {
		if err := db.Model(&model.Message{}).
			Where("thread_id = ? AND id = ?", thread.ID, lastMsgID).
			Pluck("seq_id", &cursorSeqID).Error; err != nil {
			return nil, fmt.Errorf("invalid cursor message id: %w", err)
		}
	}

	// 3. 查询消息（分页）：使用 SeqID 保证绝对时序
	messages := make([]*model.Message, 0, pageSize+1)
	query := db.Model(&model.Message{}).
		Where("thread_id = ? AND is_withdrawed = ?", thread.ID, false).
		Order("seq_id DESC"). // 【修改】使用 seq_id 倒序
		Limit(pageSize + 1)

	if cursorSeqID > 0 {
		// 【修改】使用 seq_id < ? 进行游标分页，完美规避重复数据
		query = query.Where("seq_id < ?", cursorSeqID)
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}

	hasMore := false
	if len(messages) > pageSize {
		hasMore = true
		messages = messages[:pageSize] // 丢弃最后一条，保持 pageSize
	}

	// 4. 查询未读数
	var unreadCount int
	err := db.Model(&model.Conversation{}).
		Where("owner_id = ? AND thread_id = ?", senderID, thread.ID).
		Pluck("unread_count", &unreadCount).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 5. 收集并去重 senderIDs (优化网络开销)
	senderIDMap := make(map[int64]struct{})
	for _, m := range messages {
		senderIDMap[m.SenderID] = struct{}{}
	}
	uniqueSenderIDs := make([]int64, 0, len(senderIDMap))
	for id := range senderIDMap {
		uniqueSenderIDs = append(uniqueSenderIDs, id)
	}

	// 6. 调用 user-service 获取用户信息
	userResp, err := r.userClient.GetUserInfos(ctx, &userpb.GetUserInfosRequest{
		UserIds: uniqueSenderIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("fail to get user infos: %w", err)
	}

	// 7. 调用 group-service 获取群昵称 (群聊专属)
	groupResp, err := r.groupClient.ListGroupMembers(ctx, &grouppb.ListGroupMembersRequest{
		GroupId: groupID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("fail to get group members: %w", err)
	}

	// 8. 建立查找表
	userMap := make(map[int64]UserInfo, len(userResp.Users))
	for _, u := range userResp.Users {
		userMap[u.UserId] = UserInfo{ // 直接存值而不是指针，避免后续查空引发 Panic
			UserID:   u.UserId,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
		}
	}

	groupNicknameMap := make(map[int64]string, len(groupResp.Members))
	for _, gm := range groupResp.Members {
		groupNicknameMap[gm.UserId] = gm.Nickname
	}

	// 9. 组装返回数据 (包含安全防御)
	messageWithUserInfos := make([]*MessageWithUser, 0, len(messages))
	for _, m := range messages {
		// 安全获取用户信息，如果没有拿到，给个默认兜底，防止前端渲染报错
		uInfo, exists := userMap[m.SenderID]
		if !exists {
			uInfo = UserInfo{UserID: m.SenderID, Nickname: "未知用户"}
		}

		mwu := &MessageWithUser{
			Message:       *m,
			GroupNickname: groupNicknameMap[m.SenderID],
			User:          uInfo,
		}
		messageWithUserInfos = append(messageWithUserInfos, mwu)
	}

	return &ConversationGroupMessages{
		Thread:   &thread,
		Messages: messageWithUserInfos,
		HasMore:  hasMore,
		Unread:   unreadCount,
	}, nil
}

// 当前哪个用户在读，在哪个thread读，同时获取最后已读messageid
func (r *messageRepo) UpdateUnread(ctx context.Context, userID, threadID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新 MessageStatus 已读状态
		if err := tx.Model(&model.MessageStatus{}).
			Where("user_id = ? AND message_id IN (?)", userID,
				tx.Model(&model.Message{}).Select("id").Where("thread_id = ?", threadID),
			).
			Update("status", 1).Error; err != nil {
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

func (r *messageRepo) GetSingleConversationsFromDB(ctx context.Context, userID int64) ([]*SingleConversation, error) {
	// 1. 查询 Conversation + Thread
	logger, _ := zap.NewDevelopment() //日志调试
	defer logger.Sync()
	var conversations []model.Conversation
	// 单聊
	if err := r.db.WithContext(ctx). // 需要进行group_id的判断,否则可能会混淆
						Preload("Thread").
						Where("owner_id = ? AND is_deleted = false", userID).
						Joins("JOIN threads ON conversations.thread_id = threads.id").
						Where("threads.group_id IS NULL").
						Order("updated_at DESC").
						Find(&conversations).Error; err != nil {
		return nil, err
	}
	logger.Info("conversations", zap.Any("data", conversations))
	convs := make([]*SingleConversation, 0, len(conversations))
	// 2. 收集 LastMessageID
	msgIDs := make([]int64, 0, len(conversations))
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
			msgMap[m.MsgID] = &m
		}
	}

	// 4. 拼接 Conversation
	for _, c := range conversations {
		peerID := int64(0)
		if c.Thread.PeerA != nil && *c.Thread.PeerA != userID {
			peerID = *c.Thread.PeerA
		} else if c.Thread.PeerB != nil && *c.Thread.PeerB != userID {
			peerID = *c.Thread.PeerB
		}

		var lastMsg *model.Message
		if c.LastMessageID != nil {
			lastMsg = msgMap[*c.LastMessageID]
		}

		convs = append(convs, &SingleConversation{
			ThreadID:    c.ThreadID,
			PeerID:      peerID,
			LastMessage: lastMsg,
			UnreadCount: c.UnreadCount,
			UpdateTime:  c.UpdatedAt,
		})
	}
	return convs, nil
}

func (r *messageRepo) GetGroupConversationsFromDB(ctx context.Context, userID int64) ([]*GroupConversation, error) {
	// 查询 Conversation + Thread
	var conversations []model.Conversation
	// 群聊
	if err := r.db.WithContext(ctx).
		Preload("Thread").
		Where("owner_id = ? AND is_deleted = false", userID).
		Joins("JOIN threads ON conversations.thread_id = threads.id").
		Where("threads.group_id IS NOT NULL").
		Order("updated_at DESC").
		Find(&conversations).Error; err != nil {
		return nil, err
	}

	// 收集 LastMessageID
	convs := make([]*GroupConversation, 0, len(conversations))
	msgIDs := make([]int64, 0, len(conversations))
	for _, c := range conversations {
		if c.LastMessageID != nil {
			msgIDs = append(msgIDs, *c.LastMessageID)
		}
	}
	// 批量查询最后一条消息 用于显示会话
	msgMap := make(map[int64]*model.Message)
	if len(msgIDs) > 0 {
		var msgs []model.Message
		if err := r.db.WithContext(ctx).Where("id IN ?", msgIDs).Find(&msgs).Error; err != nil {
			return nil, err
		}
		for _, m := range msgs {
			msgMap[m.MsgID] = &m
		}
	}
	// 拼接 Conversation
	for _, c := range conversations {
		groupID := c.Thread.GroupID
		var lastMsg *model.Message
		if c.LastMessageID != nil {
			lastMsg = msgMap[*c.LastMessageID]
		}
		convs = append(convs, &GroupConversation{
			ThreadID:    c.ThreadID,
			GroupID:     *groupID,
			LastMessage: lastMsg,
			UnreadCount: c.UnreadCount,
			UpdateTime:  c.UpdatedAt,
		})
	}
	return convs, nil
}

// TODO: 添加置顶功能
func (r *messageRepo) GetConversations(ctx context.Context, userID int64) ([]*ConversationWithUser, error) {
	// 从数据库查单聊会话
	singleConversations, err := r.GetSingleConversationsFromDB(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 收集 PeerID
	var peerIDs []int64
	for _, conv := range singleConversations {
		peerIDs = append(peerIDs, conv.PeerID)
	}

	// 查群聊会话
	groupConversations, err := r.GetGroupConversationsFromDB(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 获取 GroupID
	var groupIDs []string // grpc 的uuid 应为 string
	for _, conv := range groupConversations {
		groupIDs = append(groupIDs, conv.GroupID.String())
	}

	// 调用 UserService 批量获取用户信息
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
	// 调用 GroupService
	respq, err := r.groupClient.ListGroupInfos(ctx, &grouppb.ListGroupInfosRequest{
		GroupId: groupIDs,
	})
	if err != nil {
		return nil, err
	}
	groupMap := make(map[uuid.UUID]*GroupInfo)
	for _, g := range respq.Groups {
		u, err := uuid.Parse(g.GroupId)
		if err != nil {
			return nil, fmt.Errorf("invalid group_id")
		}
		groupMap[u] = &GroupInfo{
			GroupID:   u,
			GroupName: g.GroupName,
			Avatar:    g.Avatar,
		}
	}

	// 单聊
	result := make([]*ConversationWithUser, 0, len(singleConversations)+len(groupConversations))
	for _, conv := range singleConversations {
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
			UpdateTime: conv.UpdateTime,
		})
	}
	//群聊
	for _, conv := range groupConversations {
		g, ok := groupMap[conv.GroupID]
		if !ok || g == nil {
			log.Println("group info not found for groupID=", conv.GroupID)
			continue
		}
		result = append(result, &ConversationWithUser{
			ThreadID:    conv.ThreadID,
			LastMessage: conv.LastMessage,
			UnreadCount: conv.UnreadCount,
			GroupInfo: &GroupInfo{
				GroupID:   g.GroupID,
				GroupName: g.GroupName,
				Avatar:    g.Avatar,
			},
			UpdateTime: conv.UpdateTime,
		})
	}
	// 最终按更新时间排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdateTime.After(result[j].UpdateTime)
	})
	return result, nil
}
