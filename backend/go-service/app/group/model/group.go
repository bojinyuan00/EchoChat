// Package model 定义 group 模块的数据库模型
package model

import "time"

// Group 群聊信息模型，对应 im_groups 表
// 与 im_conversations (type=2) 一对一关联
// Status: 1=正常，2=已解散
type Group struct {
	ID             int64     `json:"id" gorm:"primaryKey;autoIncrement"`                          // 群唯一标识
	ConversationID int64     `json:"conversation_id" gorm:"not null;uniqueIndex"`                 // 关联 im_conversations.id
	Name           string    `json:"name" gorm:"size:100;not null;default:''"`                    // 群名称
	Avatar         string    `json:"avatar" gorm:"size:500;default:''"`                           // 群头像 URL（MinIO）
	OwnerID        int64     `json:"owner_id" gorm:"not null;index:idx_im_groups_owner"`          // 群主用户 ID
	Notice         string    `json:"notice" gorm:"type:text;default:''"`                          // 群公告内容
	MaxMembers     int       `json:"max_members" gorm:"not null;default:200"`                     // 最大成员数
	IsSearchable   bool      `json:"is_searchable" gorm:"not null;default:true"`                  // 是否可被搜索发现
	IsAllMuted     bool      `json:"is_all_muted" gorm:"not null;default:false"`                  // 是否全体禁言
	Status         int       `json:"status" gorm:"not null;default:1"`                            // 群状态：1=正常，2=已解散
	CreatedAt      time.Time `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0)"` // 创建时间
	UpdatedAt      time.Time `json:"updated_at" gorm:"not null;autoUpdateTime;type:timestamp(0)"` // 更新时间
}

// TableName 指定数据库表名
func (Group) TableName() string {
	return "im_groups"
}
