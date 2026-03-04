package model

import "time"

// ConversationMember 会话成员模型，对应 im_conversation_members 表
// 每个会话的每个成员一条记录，存储置顶/未读/软删除等个人视图
// 群聊场景下额外使用 Role/Nickname/IsMuted/IsDoNotDisturb/JoinedAt/AtMeCount 字段
type ConversationMember struct {
	ID               int64      `json:"id" gorm:"primaryKey;autoIncrement"`                          // 记录唯一标识
	ConversationID   int64      `json:"conversation_id" gorm:"not null;uniqueIndex:idx_conv_user"`   // 所属会话 ID
	UserID           int64      `json:"user_id" gorm:"not null;uniqueIndex:idx_conv_user"`           // 成员用户 ID
	IsPinned         bool       `json:"is_pinned" gorm:"default:false"`                              // 是否置顶该会话
	IsDeleted        bool       `json:"is_deleted" gorm:"default:false"`                             // 是否删除该会话（软删除，不影响对方）
	UnreadCount      int        `json:"unread_count" gorm:"default:0"`                               // 该成员在此会话中的未读消息数
	LastReadMsgID    int64      `json:"last_read_msg_id" gorm:"default:0"`                           // 该成员最后已读消息 ID
	ClearBeforeMsgID int64      `json:"clear_before_msg_id" gorm:"default:0"`                        // 清空记录时的消息截止 ID（个人视图，不影响对方）
	Role             int        `json:"role" gorm:"not null;default:0"`                              // 成员角色：0=普通成员，1=管理员，2=群主
	Nickname         string     `json:"nickname" gorm:"size:50;default:''"`                          // 群内昵称（仅群聊有效）
	IsMuted          bool       `json:"is_muted" gorm:"not null;default:false"`                      // 是否被禁言
	IsDoNotDisturb   bool       `json:"is_do_not_disturb" gorm:"not null;default:false"`             // 是否消息免打扰
	JoinedAt         *time.Time `json:"joined_at" gorm:"type:timestamp(0)"`                          // 加入群聊时间
	AtMeCount        int        `json:"at_me_count" gorm:"default:0"`                                // 被@提醒未读计数
	CreatedAt        time.Time  `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0)"` // 加入会话时间
	UpdatedAt        time.Time  `json:"updated_at" gorm:"not null;autoUpdateTime;type:timestamp(0)"` // 最后更新时间
}

// TableName 指定数据库表名
func (ConversationMember) TableName() string {
	return "im_conversation_members"
}
