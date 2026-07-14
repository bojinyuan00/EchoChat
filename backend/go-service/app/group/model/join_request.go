package model

import "time"

// GroupJoinRequest 入群申请 / 群邀请模型，对应 im_group_join_requests 表
// Status: 0=待审批，1=通过，2=拒绝
//
// 两种场景复用同一张表，通过 InviterID 字段区分：
//   - InviterID == nil：用户主动申请入群（原有语义，由群主/管理员审批）
//   - InviterID != nil：群主/管理员主动邀请用户入群（由被邀请者 UserID 本人审批）
// 这样可以复用 pending/approved/rejected 状态机与既有通知管道。
type GroupJoinRequest struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`                          // 申请唯一标识
	GroupID    int64     `json:"group_id" gorm:"not null;index:idx_group_join_req_group"`     // 目标群 ID
	UserID     int64     `json:"user_id" gorm:"not null;index:idx_group_join_req_user"`       // 申请人用户 ID（或被邀请者）
	Message    string    `json:"message" gorm:"type:text;default:''"`                         // 申请附言
	InviterID  *int64    `json:"inviter_id,omitempty" gorm:"index:idx_group_join_req_inviter"`// 邀请方用户 ID（管理员邀请场景；NULL=用户主动申请）
	ReviewerID *int64    `json:"reviewer_id"`                                                 // 审批人用户 ID
	Status     int       `json:"status" gorm:"not null;default:0"`                            // 状态：0=待审批，1=通过，2=拒绝
	CreatedAt  time.Time `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0)"` // 申请时间
	UpdatedAt  time.Time `json:"updated_at" gorm:"not null;autoUpdateTime;type:timestamp(0)"` // 更新时间
}

// IsInvitation 是否为管理员邀请记录（非用户主动申请）
func (r *GroupJoinRequest) IsInvitation() bool {
	return r.InviterID != nil && *r.InviterID > 0
}

// TableName 指定数据库表名
func (GroupJoinRequest) TableName() string {
	return "im_group_join_requests"
}
