package model

import "time"

// MeetingChat 会议内文字聊天消息模型，对应 meeting_chats 表
// 独立于 im_messages，不进入常规 IM 消息流
// 清理策略：会议结束（status=2）后 24 小时由定时任务批量 DELETE
type MeetingChat struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`                                                // 消息唯一标识
	RoomID    int64     `json:"room_id" gorm:"not null;index:idx_meeting_chats_room_created,priority:1"`           // 所属房间 ID
	UserID    int64     `json:"user_id" gorm:"not null"`                                                           // 发送者用户 ID
	Content   string    `json:"content" gorm:"type:text;not null"`                                                 // 消息内容，纯文本
	CreatedAt time.Time `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0);index:idx_meeting_chats_room_created,priority:2,sort:asc"` // 发送时间
}

// TableName 指定数据库表名
func (MeetingChat) TableName() string {
	return "meeting_chats"
}
