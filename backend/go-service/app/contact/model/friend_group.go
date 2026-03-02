package model

import "time"

// FriendGroup 好友分组模型，对应 contact_groups 表
type FriendGroup struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`         // 分组唯一标识
	UserID    int64     `gorm:"not null;index:idx_contact_groups_user" json:"user_id"` // 所属用户 ID
	Name      string    `gorm:"size:50;not null" json:"name"`               // 分组名称
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`       // 排序权重，越小越靠前
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`  // 创建时间
}

func (FriendGroup) TableName() string {
	return "contact_groups"
}
