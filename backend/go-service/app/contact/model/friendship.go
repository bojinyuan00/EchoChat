package model

import "time"

// Friendship 好友关系模型，对应 contact_friendships 表
// 双向存储：A→B 和 B→A 各一条记录
type Friendship struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`              // 记录唯一标识
	UserID    int64     `gorm:"not null;index:idx_friendships_user_status" json:"user_id"`   // 发起方用户 ID
	FriendID  int64     `gorm:"not null;index:idx_friendships_friend_status" json:"friend_id"` // 好友用户 ID
	Remark    string    `gorm:"size:50;default:''" json:"remark"`                // 好友备注名
	GroupID   *int64    `gorm:"default:null" json:"group_id"`                    // 所属好友分组 ID
	Status    int       `gorm:"not null;default:0;index:idx_friendships_user_status" json:"status"` // 状态：0=待确认，1=已接受，2=已拒绝，3=已拉黑
	Message   string    `gorm:"size:200;default:''" json:"message"`              // 好友申请附言
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`       // 创建时间
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`       // 更新时间
}

func (Friendship) TableName() string {
	return "contact_friendships"
}
