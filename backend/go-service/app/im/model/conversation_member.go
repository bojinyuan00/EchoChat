package model

import "time"

// ConversationMember 会话成员模型，对应 im_conversation_members 表
// 每个会话的每个成员一条记录，存储置顶/未读/软删除等个人视图
type ConversationMember struct {
	ID             int64     `json:"id" gorm:"primaryKey;autoIncrement"`                          // 记录唯一标识
	ConversationID int64     `json:"conversation_id" gorm:"not null;uniqueIndex:idx_conv_user"`   // 所属会话 ID
	UserID         int64     `json:"user_id" gorm:"not null;uniqueIndex:idx_conv_user"`           // 成员用户 ID
	IsPinned       bool      `json:"is_pinned" gorm:"default:false"`                              // 是否置顶该会话
	IsDeleted      bool      `json:"is_deleted" gorm:"default:false"`                             // 是否删除该会话（软删除，不影响对方）
	UnreadCount    int       `json:"unread_count" gorm:"default:0"`                               // 该成员在此会话中的未读消息数
	LastReadMsgID  int64     `json:"last_read_msg_id" gorm:"default:0"`                           // 该成员最后已读消息 ID
	CreatedAt      time.Time `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0)"` // 加入会话时间
	UpdatedAt      time.Time `json:"updated_at" gorm:"not null;autoUpdateTime;type:timestamp(0)"` // 最后更新时间
}

// TableName 指定数据库表名
func (ConversationMember) TableName() string {
	return "im_conversation_members"
}
