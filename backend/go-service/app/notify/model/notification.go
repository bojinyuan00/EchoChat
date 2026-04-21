// Package model 提供 notify 模块的数据库模型
package model

import (
	"time"
)

// Notification 通知记录，对应 notify_notifications 表
// 通知系统的核心持久化载体，覆盖好友 / 群聊 / 会议 / 系统广播四大类型
// 每条通知对应单个接收者，不做跨用户共享
type Notification struct {
	ID         int64      `json:"id" gorm:"primaryKey;autoIncrement"`                          // 通知唯一标识
	UserID     int64      `json:"user_id" gorm:"not null;index:idx_notify_user_time,priority:1;index:idx_notify_user_unread,priority:1"` // 接收者用户 ID
	Type       string     `json:"type" gorm:"size:50;not null"`                                // 通知类型常量，见 constants.NotifyType*
	Title      string     `json:"title" gorm:"size:100;not null;default:''"`                   // 通知标题（前端列表主文案）
	Content    string     `json:"content" gorm:"size:500;not null;default:''"`                 // 通知副文案
	Extra      *string    `json:"extra" gorm:"type:jsonb"`                                     // 扩展数据 JSON 字符串，保存类型相关的业务参数
	ActorID    *int64     `json:"actor_id"`                                                    // 触发主体用户 ID（申请人/邀请人等），系统广播为 nil
	TargetType string     `json:"target_type" gorm:"size:30;not null;default:''"`              // 业务对象类型：user / group / meeting / system
	TargetID   *int64     `json:"target_id"`                                                   // 业务对象 ID
	IsRead     bool       `json:"is_read" gorm:"not null;default:false"`                       // 是否已读：false=未读，true=已读
	ReadAt     *time.Time `json:"read_at"`                                                     // 已读时间
	CreatedAt  time.Time  `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0)"` // 通知生成时间
}

// TableName 指定数据库表名
func (Notification) TableName() string {
	return "notify_notifications"
}
