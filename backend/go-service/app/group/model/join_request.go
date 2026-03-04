package model

import "time"

// GroupJoinRequest 入群申请模型，对应 im_group_join_requests 表
// Status: 0=待审批，1=通过，2=拒绝
type GroupJoinRequest struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`                          // 申请唯一标识
	GroupID    int64     `json:"group_id" gorm:"not null;index:idx_group_join_req_group"`     // 目标群 ID
	UserID     int64     `json:"user_id" gorm:"not null;index:idx_group_join_req_user"`       // 申请人用户 ID
	Message    string    `json:"message" gorm:"type:text;default:''"`                         // 申请附言
	ReviewerID *int64    `json:"reviewer_id"`                                                 // 审批人用户 ID
	Status     int       `json:"status" gorm:"not null;default:0"`                            // 状态：0=待审批，1=通过，2=拒绝
	CreatedAt  time.Time `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0)"` // 申请时间
	UpdatedAt  time.Time `json:"updated_at" gorm:"not null;autoUpdateTime;type:timestamp(0)"` // 更新时间
}

// TableName 指定数据库表名
func (GroupJoinRequest) TableName() string {
	return "im_group_join_requests"
}
