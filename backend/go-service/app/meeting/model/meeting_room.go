// Package model 提供 meeting 模块的数据库模型
package model

import "time"

// MeetingRoom 会议房间模型，对应 meeting_rooms 表
// 记录会议的创建信息、主持人、容量、状态与生命周期时间点
// 对外使用 RoomCode（XXX-XXX-XXX）寻址，内部使用 ID（主键）寻址
type MeetingRoom struct {
	ID            int64      `json:"id" gorm:"primaryKey;autoIncrement"`                               // 会议房间唯一标识
	RoomCode      string     `json:"room_code" gorm:"size:20;not null;uniqueIndex"`                    // 用户可见的会议号 XXX-XXX-XXX，全局唯一
	Title         string     `json:"title" gorm:"size:200;not null"`                                   // 会议标题
	HostID        int64      `json:"host_id" gorm:"not null;index:idx_meeting_rooms_host_status,priority:1"` // 主持人用户 ID
	Type          int        `json:"type" gorm:"not null;default:1"`                                   // 会议类型：1=即时，2=预约
	PasswordHash  *string    `json:"-" gorm:"size:255"`                                                // 入会密码 bcrypt 哈希，NULL 表示无密码；JSON 始终剥离
	MaxMembers    int        `json:"max_members" gorm:"not null;default:50"`                           // 房间最大成员数，MVP 应用层强制 8
	Status        int        `json:"status" gorm:"not null;default:0;index:idx_meeting_rooms_host_status,priority:2"` // 状态：0=未开始，1=进行中，2=已结束
	ScheduledAt   *time.Time `json:"scheduled_at" gorm:"type:timestamp(0)"`                            // 预约开始时间，即时会议为 NULL
	StartedAt     *time.Time `json:"started_at" gorm:"type:timestamp(0)"`                              // 实际开始时间
	EndedAt       *time.Time `json:"ended_at" gorm:"type:timestamp(0)"`                                // 实际结束时间
	EndedReason   *string    `json:"ended_reason" gorm:"size:20"`                                      // 结束原因：host_ended / empty_ttl / admin_force / system_error
	Settings      string     `json:"settings" gorm:"type:jsonb;not null;default:'{}'"`                 // JSON 字符串，保存房间级开关
	CreatedAt     time.Time  `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0);index:idx_meeting_rooms_host_status,priority:3,sort:desc"` // 创建时间
	UpdatedAt     time.Time  `json:"updated_at" gorm:"not null;autoUpdateTime;type:timestamp(0)"`      // 更新时间
}

// TableName 指定数据库表名
func (MeetingRoom) TableName() string {
	return "meeting_rooms"
}
