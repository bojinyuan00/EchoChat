// Package model 定义 IM 模块的数据库模型
package model

import "time"

// Conversation 会话模型，对应 im_conversations 表
// Type: 1=单聊（预留 2=群聊）
type Conversation struct {
	ID              int64      `json:"id" gorm:"primaryKey;autoIncrement"`                          // 会话唯一标识，自增主键
	Type            int        `json:"type" gorm:"not null;default:1"`                              // 会话类型：1=单聊，2=群聊（预留）
	CreatorID       int64      `json:"creator_id" gorm:"not null"`                                  // 会话创建者用户 ID
	LastMessageID   *int64     `json:"last_message_id"`                                             // 最后一条消息 ID，用于快速定位
	LastMsgContent  string     `json:"last_msg_content" gorm:"type:text;default:''"`                // 最后消息预览文本，用于会话列表展示
	LastMsgTime     *time.Time `json:"last_msg_time" gorm:"type:timestamp(0)"`                      // 最后消息时间，用于排序
	LastMsgSenderID *int64     `json:"last_msg_sender_id"`                                          // 最后消息发送者 ID
	CreatedAt       time.Time  `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0)"` // 会话创建时间
	UpdatedAt       time.Time  `json:"updated_at" gorm:"not null;autoUpdateTime;type:timestamp(0)"` // 会话更新时间
}

// TableName 指定数据库表名
func (Conversation) TableName() string {
	return "im_conversations"
}
