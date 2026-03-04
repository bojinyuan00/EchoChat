package model

import (
	"time"

	"github.com/echochat/backend/pkg/utils"
)

// Message 消息模型，对应 im_messages 表
// Type: 1=文本（预留 2=图片 3=语音 4=视频 5=文件），10=系统消息
// Status: 1=正常 2=已撤回 3=已删除
type Message struct {
	ID             int64            `json:"id" gorm:"primaryKey;autoIncrement"`                                       // 消息唯一标识，自增主键
	ConversationID int64            `json:"conversation_id" gorm:"not null;index:idx_msg_conv_time;index:idx_msg_conv_id"` // 所属会话 ID
	SenderID       int64            `json:"sender_id" gorm:"not null"`                                                // 发送者用户 ID（系统消息为 0）
	Type           int              `json:"type" gorm:"not null;default:1"`                                           // 消息类型：1=文本，10=系统消息
	Content        string           `json:"content" gorm:"type:text;not null"`                                        // 消息内容
	Extra          *string          `json:"extra" gorm:"type:jsonb"`                                                  // 扩展数据（JSON 格式，预留图片/语音等元信息）
	Status         int              `json:"status" gorm:"not null;default:1"`                                         // 消息状态：1=正常，2=已撤回，3=已删除
	ClientMsgID    string           `json:"client_msg_id" gorm:"size:64;default:''"`                                  // 客户端消息唯一 ID，用于幂等去重
	AtUserIDs      utils.Int64Array `json:"at_user_ids" gorm:"type:bigint[]"`                                         // @提醒用户 ID 列表，nil=无@，含 0 表示 @所有人
	CreatedAt      time.Time        `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0)"`              // 消息发送时间
}

// TableName 指定数据库表名
func (Message) TableName() string {
	return "im_messages"
}
