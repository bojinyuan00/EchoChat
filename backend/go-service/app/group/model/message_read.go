package model

import "time"

// MessageRead 群聊消息已读记录模型，对应 im_message_reads 表
// 复合主键 (message_id, user_id)，记录每条群消息的已读用户
type MessageRead struct {
	MessageID int64     `json:"message_id" gorm:"primaryKey"`                                // 消息 ID
	UserID    int64     `json:"user_id" gorm:"primaryKey"`                                   // 已读用户 ID
	ReadAt    time.Time `json:"read_at" gorm:"not null;autoCreateTime;type:timestamp(0)"`    // 已读时间
}

// TableName 指定数据库表名
func (MessageRead) TableName() string {
	return "im_message_reads"
}
