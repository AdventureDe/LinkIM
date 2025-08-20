package model

import "time"

// 逻辑会话（Thread）
type Thread struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Type      int16     `gorm:"not null"` // 1=single, 2=group
	GroupID   *int64    `gorm:"index"`    // 群聊ID（如果是群聊）
	PeerA     *int64    `gorm:"index"`    // 单聊 A
	PeerB     *int64    `gorm:"index"`    // 单聊 B
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// 用户会话条目（Conversation）
type Conversation struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	OwnerID       int64     `gorm:"not null;index"` // 会话所属用户
	ThreadID      int64     `gorm:"not null;index"` // 关联 Thread
	Thread        Thread    `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE"`
	LastMessageID *int64    `gorm:"index"` // 最近一条消息
	UnreadCount   int       `gorm:"default:0"`
	Pinned        bool      `gorm:"default:false"`
	Mute          bool      `gorm:"default:false"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
	IsDeleted     bool      `gorm:"is_deleted"`
	// 保证每个用户同一个 thread 只会有一条记录
	// UNIQUE(owner_id, thread_id) -> gorm 里用 index+uniqueConstraint
}

// 消息（Message）
type Message struct {
	ID       int64 `gorm:"primaryKey;autoIncrement"`
	ThreadID int64 `gorm:"not null;index"`
	//Thread       Thread    `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE"`
	SenderID     int64     `gorm:"not null;index"`
	Kind         int16     `gorm:"not null"` // 消息类型 1. text 2. image 3. file
	Content      string    `gorm:"type:text;not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	IsWithdrawed bool      `gorm:"is_withdrawed"`
}

// 每条消息针对每个用户的读状态（MessageStatus）
type MessageStatus struct {
	MessageID int64     `gorm:"primaryKey"`
	Message   Message   `gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE"`
	UserID    int64     `gorm:"primaryKey;index"`
	Status    int16     `gorm:"not null"` // 0=未读,1=已读
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// 会话内消息索引（MessageIndex）
type MessageIndex struct {
	ThreadID  int64     `gorm:"primaryKey"`
	MsgSeq    int64     `gorm:"primaryKey;autoIncrement:false"` // 会话内序号
	MessageID int64     `gorm:"not null;index"`
	Message   Message   `gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
