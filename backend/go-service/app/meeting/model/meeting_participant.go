package model

import "time"

// MeetingParticipant 会议参与者模型，对应 meeting_participants 表
// 一次入会对应一条记录，重入（left_at 已填的情况下再次加入）通过 UPDATE 复用
// 主持人转让通过事务：旧 host.role=0 + 新 host.role=1
type MeetingParticipant struct {
	ID         int64      `json:"id" gorm:"primaryKey;autoIncrement"`                                         // 参与者记录唯一标识
	RoomID     int64      `json:"room_id" gorm:"not null;uniqueIndex:uk_meeting_participants_room_user,priority:1;index:idx_meeting_participants_room,priority:1"`  // 所属房间 ID，与 user_id 组合唯一
	UserID     int64      `json:"user_id" gorm:"not null;uniqueIndex:uk_meeting_participants_room_user,priority:2;index:idx_meeting_participants_user,priority:1"`  // 用户 ID
	Role       int        `json:"role" gorm:"not null;default:0"`                                             // 0=普通，1=主持人（MVP 仅此两档），2=联合主持人
	JoinedAt   time.Time  `json:"joined_at" gorm:"not null;autoCreateTime;type:timestamp(0);index:idx_meeting_participants_room,priority:2,sort:asc;index:idx_meeting_participants_user,priority:2,sort:desc"` // 加入时间
	LeftAt     *time.Time `json:"left_at" gorm:"type:timestamp(0)"`                                           // 离开时间，NULL 表示仍在会议中
	LeftReason *string    `json:"left_reason" gorm:"size:20"`                                                 // 离会原因：self / kicked / host_end / empty_ttl / disconnect
	Duration   int        `json:"duration" gorm:"not null;default:0"`                                         // 本次参会时长（秒），left_at 写入时同步计算
}

// TableName 指定数据库表名
func (MeetingParticipant) TableName() string {
	return "meeting_participants"
}

// IsActive 参与者当前是否仍在会议中（left_at 为 NULL）
func (p *MeetingParticipant) IsActive() bool {
	return p.LeftAt == nil
}
