package repo

import (
	"context"
	"fmt"

	userpb "github.com/AdventureDe/LinkIM/api/user"
	"github.com/AdventureDe/LinkIM/group/repo/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroupAvatarSet struct {
	GroupId  uuid.UUID
	UserInfo []*UserInfo
}

type UserInfo struct {
	UserID   int64  `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type GroupRepo interface {
	CreateGroup(ctx context.Context, ownerID int64, userIDs []int64, groupName string) (groupID uuid.UUID, err error)
	AddGroupMember(ctx context.Context, groupID uuid.UUID, userIDs []int64) error // 可进一步拓展
	KickOutGroupMember(ctx context.Context, groupID uuid.UUID, executorID int64, userIDs []int64) error
	PromoteToAdmin(ctx context.Context, groupID uuid.UUID, executorID int64, userID int64) error
	TransferGroupOwner(ctx context.Context, groupID uuid.UUID, executorID int64, userID int64) error
	DemotedToMember(ctx context.Context, groupID uuid.UUID, executorID int64, userID int64) error
	UpdateNotice(ctx context.Context, groupID uuid.UUID, executorID int64, newNoticeText string) error
	GetNotice(ctx context.Context, groupID uuid.UUID) (string, error)
	UpdateGroupName(ctx context.Context, groupID uuid.UUID, executorID int64, newGroupName string) error
	GetGroupName(ctx context.Context, groupID uuid.UUID) (string, error)
	GetGroupAvatar(ctx context.Context, groupID uuid.UUID) (*GroupAvatarSet, error)
	UpdateSelfName(ctx context.Context, groupID uuid.UUID, userID int64, newName string) error
}

type groupRepo struct {
	db         *gorm.DB
	userClient userpb.UserServiceClient
}

func NewGroupRepo(db *gorm.DB, m *groupService) GroupRepo {
	return &groupRepo{
		db:         db,
		userClient: m.userClient,
	}
}

func (r *groupRepo) CreateGroup(ctx context.Context, ownerID int64, userIDs []int64, groupName string) (groupID uuid.UUID, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(userIDs) <= 1 {
			return fmt.Errorf("Number of people is less than three")
		}
		// 创建群组
		group := model.Group{
			Name:    groupName,
			OwnerID: ownerID,
		}
		if err := tx.Create(&group).Error; err != nil {
			return err
		}
		groupID = group.ID // 拿到数据库生成的 groupID

		// 插入群主
		ownerMember := model.GroupMember{
			GroupID: group.ID,
			UserID:  ownerID,
			Role:    model.Owner,
			IsOwner: true,
		}
		if err := tx.Create(&ownerMember).Error; err != nil {
			return err
		}

		// 插入其他成员
		var members []model.GroupMember
		for _, id := range userIDs {
			// 跳过群主，避免重复
			if id == ownerID {
				continue
			}
			members = append(members, model.GroupMember{
				GroupID: group.ID,
				UserID:  id,
				Role:    model.Member,
				IsOwner: false,
			})
		}
		// 避免空指针
		if len(members) > 0 { // 使用CreateInBatches批量插入数据，减少数据库访问，每次插入100个
			if err := tx.CreateInBatches(&members, 100).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return
}

func (r *groupRepo) AddGroupMember(ctx context.Context, groupid uuid.UUID, userIDs []int64) error {
	if len(userIDs) == 0 {
		return fmt.Errorf("fail to add groupMember, num == 0")
	}
	var members []model.GroupMember
	for _, id := range userIDs {
		members = append(members, model.GroupMember{
			GroupID: groupid,
			UserID:  id,
			Role:    model.Member,
			IsOwner: false,
		})
	}
	// 避免空指针
	if len(members) > 0 {
		if err := r.db.WithContext(ctx).CreateInBatches(&members, 100).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *groupRepo) KickOutGroupMember(ctx context.Context, groupid uuid.UUID, executorid int64,
	userIDs []int64) error {
	if len(userIDs) == 0 {
		return fmt.Errorf("fail to kick out groupMember, num == 0")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 查询执行者的角色
		var executor model.GroupMember
		if err := tx.Select("role").
			Where("group_id = ? AND user_id = ?", groupid, executorid).
			First(&executor).Error; err != nil {
			return fmt.Errorf("executor not in group: %w", err)
		}
		// First(&executor)
		// GORM 会把查询结果映射到 executor 的 结构体字段上。
		// Select("role")
		// 生成的 SQL 就只会查 role 列，所以最后 executor.Role 会有值，而 executor 里其它字段因为根
		// 本没被查出来，就保持 Go 的 零值（不是 nil，而是每个字段的默认值，比如 int 就是 0，string 就
		// 是 ""，time.Time 是 0001-01-01）。
		// ⚠️ 除非字段类型是指针，才可能得到 nil。
		// 权限校验
		if executor.Role == model.Member {
			return fmt.Errorf("insufficient permissions")
		}

		// 防止把管理员/群主踢掉
		// 规则： 权限 群主 > 管理员 > 普通成员
		// 只有管理员/群主可以踢人，踢人只能踢普通成员，否则将其权限降为普通成员再踢出
		var admins []int64
		if err := tx.Model(&model.GroupMember{}).
			Where("group_id = ? AND role IN ?", groupid, []model.GroupRole{model.Owner, model.Admin}).
			Pluck("user_id", &admins).Error; err != nil {
			return err
		}

		for _, id := range userIDs {
			for _, adminID := range admins {
				if id == adminID {
					return fmt.Errorf("cannot kick out group admin: %d", id)
				}
			}
		}

		// 执行删除
		if err := tx.Where("group_id = ? AND user_id IN ?", groupid, userIDs).
			Delete(&model.GroupMember{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *groupRepo) PromoteToAdmin(ctx context.Context, groupid uuid.UUID, executorid int64, userid int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target []model.GroupMember
		if err := tx.Select("user_id", "role").Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND user_id IN ?", groupid, []int64{executorid, userid}).
			Find(&target).Error; err != nil {
			return err
		}
		if len(target) != 2 {
			return fmt.Errorf("either executor or target not in group")
		}
		var executorRole, userRole model.GroupRole
		for _, m := range target { //IN搜索不保证顺序，遍历搜索
			if m.UserID == executorid {
				executorRole = m.Role
			} else if m.UserID == userid {
				userRole = m.Role
			}
		}

		if executorRole != model.Owner {
			return fmt.Errorf("executor must be owner")
		}

		if userRole != model.Member {
			return fmt.Errorf("target must be member")
		}

		if err := tx.Model(&model.GroupMember{}).
			Where("group_id = ? AND user_id = ?", groupid, userid).
			Update("role", model.Admin).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *groupRepo) TransferGroupOwner(ctx context.Context, groupid uuid.UUID,
	executorid int64, userid int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 查询执行者，并加行锁
		var executor model.GroupMember
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}). //Clauses 用来给 SQL 语句增加一些额外的 SQL 子句。
			//FOR UPDATE → 给选中的行加排他锁，阻止其它事务修改这些行。
			//FOR SHARE → 给选中的行加共享锁，允许别人读，但不允许写。
			//1. FOR SHARE （共享锁）
			// 作用：允许多个事务同时读，但不允许修改（写）。
			// 行为：
			// 多个事务可以同时 SELECT ... FOR SHARE。
			// 但别的事务不能对这些行做 UPDATE 或 DELETE，直到锁释放。
			// 👉 适合 只读校验 的场景，比如我只是要保证没人删掉某一行。
			// 2. FOR UPDATE （排他锁）
			// 作用：同一时间只有一个事务能锁住这些行，别人既不能写也不能读（加锁的 SELECT）。
			// 行为：
			// 其它事务如果也试图 FOR UPDATE 同一行，会被阻塞，直到前一个事务结束。
			// 确保 检查 + 修改 在同一个事务里是安全的。
			// 👉 适合 读后要写 的场景。
			Select("role").
			Where("group_id = ? AND user_id = ?", groupid, executorid).
			First(&executor).Error; err != nil {
			return fmt.Errorf("executor not in group: %w", err)
		}
		if executor.Role != model.Owner {
			return fmt.Errorf("insufficient permissions")
		}

		// 查询目标用户，并加行锁，确保不会被别的事务同时修改
		var target model.GroupMember
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("role").
			Where("group_id = ? AND user_id = ?", groupid, userid).
			First(&target).Error; err != nil {
			return fmt.Errorf("target not in group: %w", err)
		}

		// 升级目标用户为群主
		if err := tx.Model(&model.GroupMember{}).
			Where("group_id = ? AND user_id = ?", groupid, userid).
			Updates(map[string]interface{}{
				"role":     model.Owner,
				"is_owner": true,
			}).Error; err != nil {
			return err
		}

		// 更新群信息里的 owner_id（这里最好也加锁）
		if err := tx.Model(&model.Group{}).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", groupid).
			Update("owner_id", userid).Error; err != nil {
			return err
		}

		// 原群主降级为管理员
		if err := tx.Model(&model.GroupMember{}).
			Where("group_id = ? AND user_id = ?", groupid, executorid).
			Updates(map[string]interface{}{
				"role":     model.Admin,
				"is_owner": false,
			}).Error; err != nil {
			return err
		}

		return nil
	})
}

// 撤销管理员
func (r *groupRepo) DemotedToMember(ctx context.Context, groupid uuid.UUID, executorid int64, userid int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target []model.GroupMember
		if err := tx.Select("user_id", "role").Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND user_id IN ?", groupid, []int64{executorid, userid}).
			Find(&target).Error; err != nil {
			return err
		}
		if len(target) != 2 {
			return fmt.Errorf("either executor or target not in group")
		}
		var executorRole, userRole model.GroupRole
		for _, m := range target { // IN搜索不保证顺序，遍历搜索
			switch m.UserID {
			case executorid:
				executorRole = m.Role
			case userid:
				userRole = m.Role
			}
		}

		if executorRole != model.Owner {
			return fmt.Errorf("executor must be owner")
		}

		if userRole != model.Admin {
			return fmt.Errorf("target must be admin")
		}

		if err := tx.Model(&model.GroupMember{}).
			Where("group_id = ? AND user_id = ?", groupid, userid).
			Update("role", model.Member).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *groupRepo) UpdateNotice(ctx context.Context, groupid uuid.UUID, executorid int64,
	newnoticetext string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var executor model.GroupMember
		if err := tx.Select("role").Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND user_id = ?", groupid, executorid).
			First(&executor).Error; err != nil {
			return err
		}
		if executor.Role != model.Owner && executor.Role != model.Admin {
			return fmt.Errorf("insufficient permissions")
		}
		var g model.Group
		if err := tx.Select("notice").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", groupid).
			First(&g).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.Group{}).
			Where("id = ?", groupid).
			Update("notice", newnoticetext).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *groupRepo) GetNotice(ctx context.Context, groupid uuid.UUID) (string, error) {
	var g model.Group
	if err := r.db.WithContext(ctx).
		Select("notice").
		Where("id = ?", groupid).
		First(&g).Error; err != nil {
		return "", err
	}
	return g.Notice, nil
}

func (r *groupRepo) UpdateGroupName(ctx context.Context, groupid uuid.UUID, executorid int64, newgroupname string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var executor model.GroupMember
		if err := tx.Select("role").Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND user_id = ?", groupid, executorid).
			First(&executor).Error; err != nil {
			return err
		}
		if executor.Role != model.Admin && executor.Role != model.Owner {
			return fmt.Errorf("insufficient permissions")
		}
		var g model.Group
		if err := tx.Select("name").
			Clauses(clause.Locking{Strength: "UPDATE"}). //先锁住再进行更新
			Where("id = ?", groupid).
			First(&g).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Group{}).
			Where("id = ?", groupid).
			Update("name", newgroupname).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *groupRepo) GetGroupName(ctx context.Context, groupid uuid.UUID) (string, error) {
	var g model.Group
	if err := r.db.WithContext(ctx).
		Select("name").
		Where("id = ?", groupid).
		First(&g).Error; err != nil {
		return "", err
	}
	return g.Name, nil
}

func (r *groupRepo) GetGroupAvatar(ctx context.Context, groupid uuid.UUID) (gas *GroupAvatarSet, err error) {
	var userids []int64
	err = r.db.WithContext(ctx).Model(&model.GroupMember{}).
		Select("user_id").
		Where("group_id = ?", groupid).
		Order("join_time ASC").
		Limit(9).
		Find(&userids).Error

	resp, err := r.userClient.GetUserInfos(ctx, &userpb.GetUserInfosRequest{
		UserIds: userids,
	})

	userMap := make(map[int64]*userpb.UserInfo)
	for _, u := range resp.Users {
		userMap[u.UserId] = u
	}

	res := make([]*UserInfo, 0, len(userids))
	for _, userid := range userids {
		u, ok := userMap[userid]
		if !ok || u == nil {
			err = fmt.Errorf("user info not found for userid: %d", userid)
		}
		userinfo := &UserInfo{
			UserID:   u.UserId,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
		}
		res = append(res, userinfo)
	}
	gas = &GroupAvatarSet{
		GroupId:  groupid,
		UserInfo: res,
	}

	return
}

func (r *groupRepo) UpdateSelfName(ctx context.Context, groupid uuid.UUID, userid int64, newname string) error {
	if len(newname) == 0 || len(newname) > 64 {
		return fmt.Errorf("invalid nickname length")
	}
	// 严格检查用户是否存在
	res := r.db.WithContext(ctx).Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupid, userid).
		Update("nickname", newname)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("user not found in group")
	}
	return nil
}
