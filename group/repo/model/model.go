package model

import (
	"time"

	"github.com/google/uuid"
)

// 群组状态枚举
type GroupStatus string

const (
	GroupActive   GroupStatus = "active"
	GroupArchived GroupStatus = "archived"
	GroupDeleted  GroupStatus = "deleted"
)

type Group struct {
	ID        uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"` // 主键 UUID
	Name      string      `gorm:"type:varchar(100);not null"`
	OwnerID   int64       `gorm:"not null;index"`    // 建索引方便查询
	Avatar    string      `gorm:"type:varchar(255)"` //url
	Notice    string      `gorm:"type:text"`
	Status    GroupStatus `gorm:"type:group_status;not null;default:'active'"`
	IsBanned  bool        `gorm:"not null;default:false"` //是否禁言
	CreatedAt time.Time   `gorm:"autoCreateTime"`
	UpdatedAt time.Time   `gorm:"autoUpdateTime"`
}

// 群成员角色枚举
type GroupRole string

const (
	Member GroupRole = "member"
	Admin  GroupRole = "admin"
	Owner  GroupRole = "owner"
)

type GroupMember struct {
	GroupID  uuid.UUID `gorm:"type:uuid;primaryKey"` // 复合主键
	UserID   int64     `gorm:"primaryKey;index"`
	Role     GroupRole `gorm:"type:group_role;not null;default:'member'"`
	JoinTime time.Time `gorm:"autoCreateTime"`
	Nickname string    `gorm:"type:varchar(50)"`
	IsOwner  bool      `gorm:"not null;default:false"`
}
